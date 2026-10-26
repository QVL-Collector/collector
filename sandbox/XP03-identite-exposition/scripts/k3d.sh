#!/usr/bin/env bash
# XP03 sur k3d (protocole § B5) : PostgreSQL, Keycloak, service de test, appelant, politiques reseau.
# Debogage seulement : rejoue K2 (demarrage) et K3 (jetons), G7-4 en debogage.
# Emetteur local : http://localhost:18080/realms/collector (port-forward ouvert par verifier.sh).
# Le contexte kubectl courant doit etre un cluster k3d (k3d-<nom>).
# Usage :
#   scripts/k3d.sh up        image importee, secrets, realm, pile prete
#   scripts/k3d.sh k2        2 mesures : creation du pod -> Ready, base neuve (pods Keycloak et PostgreSQL recrees)
#   scripts/k3d.sh reseau    G7-4 : ns-intrus vers Keycloak (8080 /admin/, 9000) et vers 8081 ; temoins
#   scripts/k3d.sh down      retire identite-test et les objets de XP03 des namespaces partages
#   puis : PLATEFORME=k3d scripts/verifier.sh k3|ma12|ma
PLATEFORME=k3d
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
cd "$XP_DIR"

contexte() {
  local c; c="$(kubectl config current-context)"
  [[ "$c" == k3d-* ]] || echec "contexte kubectl '$c' : un cluster k3d est attendu"
  printf '%s' "${c#k3d-}"
}

manifeste() {
  sed -e "s|image: KEYCLOAK_IMAGE|image: $KEYCLOAK_IMAGE|" -e "s|image: POSTGRES_IMAGE|image: $POSTGRES_IMAGE|" \
      -e "s|image: ENVOY_IMAGE|image: $ENVOY_IMAGE|" -e "s|image: CURL_IMAGE|image: $CURL_IMAGE|" \
      -e "s|\"URL_AUTH\"|\"$URL_AUTH\"|" "k8s/$1"
}

secret_stdin() {
  # secret_stdin <namespace> <nom> : lignes CLE=valeur sur l'entree standard, manifeste construit par jq.
  jq -Rn --arg ns "$1" --arg nom "$2" '
    { apiVersion: "v1", kind: "Secret", type: "Opaque",
      metadata: { name: $nom, namespace: $ns, labels: { "app.kubernetes.io/part-of": "xp03" } },
      stringData: ([inputs | select(length > 0) | capture("^(?<key>[^=]+)=(?<value>.*)$")] | from_entries) }' |
    kubectl apply -f - >/dev/null
}

up() {
  local cluster; cluster="$(contexte)"
  [[ -f "$ENV_FILE" ]] || "$XP_DIR/scripts/identifiants.sh"
  charger_env
  docker image inspect "$KEYCLOAK_IMAGE" >/dev/null 2>&1 || "$XP_DIR/scripts/compose.sh" image
  k3d image import -c "$cluster" "$KEYCLOAK_IMAGE" >/dev/null 2>&1
  kubectl apply -f k8s/00-namespaces.yaml >/dev/null
  printf 'MDP_POSTGRES=%s\nMDP_KC_BOOTSTRAP=%s\n' "$MDP_POSTGRES" "$MDP_KC_BOOTSTRAP" | secret_stdin identite-test keycloak-session
  printf 'URL_FRONT=%s\nSECRET_TRANSACTIONS=%s\nSECRET_AUTRE=%s\nMDP_ACHETEUR=%s\nMDP_VENDEUR=%s\nMDP_ADMIN_TEST=%s\nGRAINE_OTP=%s\n' \
    "$URL_FRONT" "$SECRET_TRANSACTIONS" "$SECRET_AUTRE" "$MDP_ACHETEUR" "$MDP_VENDEUR" "$MDP_ADMIN_TEST" "$GRAINE_OTP" |
    secret_stdin identite-test keycloak-realm-valeurs
  kubectl -n identite-test create configmap keycloak-realm --from-file=collector-realm.json=realm/collector-realm.json \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  mkdir -p "$GENERE_DIR"
  sed -e "s|__ISSUER__|$ISSUER|" -e "s|__KEYCLOAK_HOTE__|keycloak.identite-test.svc|g" -e "s|__KEYCLOAK_PORT__|8080|g" \
    verificateur/envoy.yaml >.genere/envoy-k3d.yaml
  kubectl -n pdm-test create configmap verificateur-envoy --from-file=envoy.yaml=.genere/envoy-k3d.yaml \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  kubectl -n pdm-test label configmap verificateur-envoy app.kubernetes.io/part-of=xp03 --overwrite >/dev/null
  kubectl apply -f k8s/30-politiques-reseau.yaml >/dev/null
  manifeste 10-identite.yaml | kubectl apply -f - >/dev/null
  manifeste 20-verificateur-appelant.yaml | kubectl apply -f - >/dev/null
  kubectl -n identite-test rollout status deploy/postgres --timeout=180s
  kubectl -n identite-test rollout status deploy/keycloak --timeout=300s
  kubectl -n pdm-test rollout status deploy/verificateur --timeout=120s
  kubectl -n transactions-test rollout status deploy/appelant --timeout=120s
  journal "Pret ; emetteur $ISSUER ; verifications : PLATEFORME=k3d scripts/verifier.sh k3|ma12|ma"
}

duree_pret() {
  # Secondes entre la creation du pod Keycloak et sa condition Ready.
  kubectl -n identite-test get pod -l app=keycloak -o json | jq -r '
    .items[0] | (.metadata.creationTimestamp | fromdateiso8601) as $c
    | (.status.conditions[] | select(.type == "Ready" and .status == "True") | .lastTransitionTime | fromdateiso8601) - $c'
}

k2() {
  csv_init
  local n d
  for n in 1 2; do
    kubectl -n identite-test delete pod -l 'app in (keycloak,postgres)' --wait=true >/dev/null
    kubectl -n identite-test rollout status deploy/keycloak --timeout=300s >/dev/null
    d="$(duree_pret)"
    csv K2 "mesure $n" "$d" s "<=180" "$([[ $d -le 180 ]] && echo oui || echo non)" "creation du pod -> Ready (/health/ready 9000), base neuve, import du realm (k3d)"
    journal "K2 mesure $n : $d s"
  done
}

essai_port() {
  # essai_port <namespace> <hote:port> ... : "cible ouvert|ferme" par cible (pod jetable, image curl).
  local ns="$1" nom; shift
  nom="xp03-sonde-$(openssl rand -hex 3)"
  jq -n --arg ns "$ns" --arg nom "$nom" --arg image "$CURL_IMAGE" --arg cibles "$*" '
    { apiVersion: "v1", kind: "Pod",
      metadata: { name: $nom, namespace: $ns, labels: { "app.kubernetes.io/part-of": "xp03" } },
      spec: { restartPolicy: "Never", automountServiceAccountToken: false,
        securityContext: { runAsNonRoot: true, runAsUser: 100, seccompProfile: { type: "RuntimeDefault" } },
        containers: [ { name: "sonde", image: $image, command: ["/bin/sh", "-c"],
          args: ["for c in " + $cibles + "; do h=${c%%:*}; p=${c##*:}; if nc -z -w 5 \"$h\" \"$p\"; then echo \"$c ouvert\"; else echo \"$c ferme\"; fi; done"],
          securityContext: { allowPrivilegeEscalation: false, capabilities: { drop: ["ALL"] } } } ] } }' |
    kubectl apply -f - >/dev/null
  kubectl -n "$ns" wait pod "$nom" --for=jsonpath='{.status.phase}'=Succeeded --timeout=120s >/dev/null
  kubectl -n "$ns" logs "$nom" | sed "s/^/$ns /"
  kubectl -n "$ns" delete pod "$nom" --wait=false >/dev/null
}

reseau() {
  csv_init
  local sortie kc ok=0 total=0 code
  sortie="$SORTIES_DIR/$(horodatage)-$PLATEFORME-g7-4.txt"
  kc="keycloak.identite-test.svc"
  {
    essai_port ns-intrus "$kc:8080" "$kc:9000" "verificateur.pdm-test.svc:8081" "postgres.identite-test.svc:5432"
    essai_port transactions-test "$kc:8080" "$kc:9000" "verificateur.pdm-test.svc:8081" "postgres.identite-test.svc:5432"
  } >"$sortie"
  while read -r ns cible etat; do
    [[ -n "${etat:-}" ]] || continue
    local attendu=ferme critere=G7-4
    if [[ "$ns" == transactions-test && ( "$cible" == *:8080 || "$cible" == *:8081 ) ]]; then attendu=ouvert; critere=TEMOIN; fi
    total=$((total + 1)); [[ "$etat" == "$attendu" ]] && ok=$((ok + 1))
    csv "$critere" reseau "$etat" port "$attendu" "$(oui_non "$etat" "$attendu")" "$ns -> $cible"
  done <"$sortie"
  # Temoin applicatif : page de connexion servie a un appelant autorise.
  code="$(kubectl -n transactions-test exec deploy/appelant -c curl -- curl -s -o /dev/null -w '%{http_code}' \
    "http://$kc:8080/realms/collector/protocol/openid-connect/auth?client_id=collector-web&response_type=code&scope=openid&redirect_uri=$(jq -rn --arg v "$URL_FRONT/rappel" '$v|@uri')&code_challenge=$(openssl rand -hex 22)&code_challenge_method=S256")"
  echo "transactions-test page-de-connexion $code" >>"$sortie"
  total=$((total + 1)); [[ "$code" == 200 ]] && ok=$((ok + 1))
  csv TEMOIN reseau "$code" http 200 "$(oui_non "$code" 200)" "transactions-test -> page de connexion"
  journal "G7-4 : $ok / $total conformes (detail : $sortie)"
  cat "$sortie" >&2
  [[ $ok -eq $total ]]
}

down() {
  contexte >/dev/null
  kubectl delete namespace identite-test --ignore-not-found --wait=true >/dev/null
  for ns in pdm-test transactions-test ns-intrus; do
    kubectl -n "$ns" delete deploy,svc,configmap,networkpolicy,pod,secret -l app.kubernetes.io/part-of=xp03 --ignore-not-found >/dev/null 2>&1 || true
  done
  kubectl -n pdm-test delete networkpolicy verificateur-entree-sortie --ignore-not-found >/dev/null
  journal "XP03 retire (namespaces partages conserves)"
}

case "${1:-}" in
  up) up ;;
  k2) k2 ;;
  reseau) reseau ;;
  down) down ;;
  *) echec "usage : $0 up | k2 | reseau | down" ;;
esac
