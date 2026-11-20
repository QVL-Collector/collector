#!/usr/bin/env bash
# XP03 sur k3d (protocole § B5) et sur EKS (protocole § C, par scripts/eks.sh) : PostgreSQL ou RDS,
# Keycloak, service de test, appelant, politiques reseau.
# k3d : debogage, emetteur http://localhost:18080/realms/collector (port-forward ouvert par verifier.sh).
# EKS : image optimisee poussee dans un depot ECR de session, base dans RDS (BASE=rds, defaut ; BASE=pod :
#       PostgreSQL du pod comme en k3d), Keycloak expose par le NodePort 30081 derriere l'ALB (k8s/40-eks.yaml),
#       emetteur https://auth.collector.qvl-project.com/realms/collector.
# Contexte explicite : KUBE_CONTEXTE=k3d-<nom> (k3d), ou PLATEFORME=eks et KUBE_CONTEXTE=*collector*
# (defaut collector-xp01) ; le contexte courant n'est jamais utilise.
# Usage :
#   scripts/k3d.sh up        image importee (k3d) ou poussee (ECR), secrets, realm, pile prete
#   scripts/k3d.sh k2        2 mesures : creation du pod -> Ready, base neuve
#   scripts/k3d.sh k1 [s] [etape]  K1 : kubectl top toutes les 30 s pendant s secondes (900), OOMKilled, Pending
#   scripts/k3d.sh reseau    G7-4 : ns-intrus vers Keycloak (8080 /admin/, 9000) et vers 8081 ; temoins
#   scripts/k3d.sh down      retire identite-test et les objets de XP03 des namespaces partages (EKS : depot ECR, .env)
#   puis : PLATEFORME=k3d|eks scripts/verifier.sh k3|ma12|ma
PLATEFORME="${PLATEFORME:-k3d}"
[[ "$PLATEFORME" == eks ]] || PLATEFORME=k3d
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
cd "$XP_DIR"

BASE="${BASE:-rds}"
[[ "$PLATEFORME" == eks ]] || BASE=pod
[[ "$BASE" == rds || "$BASE" == pod ]] || echec "BASE=rds|pod"
RDS_ID="${RDS_ID:-collector-xp01}"
DEPOT_ECR="${DEPOT_ECR:-collector-xp03/keycloak}"
EKS_ENV="$GENERE_DIR/eks.env"

contexte() { exiger_contexte; printf '%s' "${KUBE_CONTEXTE#k3d-}"; }

charger_eks() {
  [[ "$PLATEFORME" == eks ]] || return 0
  [[ -f "$EKS_ENV" ]] || echec "$EKS_ENV absent : lancer scripts/eks.sh up"
  # shellcheck disable=SC1090
  source "$EKS_ENV"
}

manifeste() {
  local image="$KEYCLOAK_IMAGE" url_jdbc="jdbc:postgresql://postgres:5432/keycloak"
  [[ "$PLATEFORME" != eks ]] || image="$KEYCLOAK_IMAGE_EKS"
  [[ "$BASE" != rds ]] || url_jdbc="jdbc:postgresql://$RDS_HOTE:5432/keycloak?sslmode=require"
  sed -e "s|image: KEYCLOAK_IMAGE|image: $image|" -e "s|image: POSTGRES_IMAGE|image: $POSTGRES_IMAGE|" \
      -e "s|image: ENVOY_IMAGE|image: $ENVOY_IMAGE|" -e "s|image: CURL_IMAGE|image: $CURL_IMAGE|" \
      -e "s|\"jdbc:postgresql://postgres:5432/keycloak\"|\"$url_jdbc\"|" \
      -e "s|\"URL_AUTH\"|\"$URL_AUTH\"|" "k8s/$1"
}

image_ecr() {
  # Depot ECR de session (MX7 : supprime par down), image referencee par empreinte ; compte jamais affiche.
  local compte registre empreinte
  compte="$(aws sts get-caller-identity --query Account --output text | tr -d '\r')"
  registre="$compte.dkr.ecr.$AWS_REGION.amazonaws.com"
  aws ecr describe-repositories --repository-names "$DEPOT_ECR" >/dev/null 2>&1 ||
    aws ecr create-repository --repository-name "$DEPOT_ECR" --encryption-configuration encryptionType=AES256 \
      --tags Key=projet,Value=collector Key=couche,Value=ephemere Key=xp,Value=XP03 >/dev/null
  aws ecr get-login-password | docker login --username AWS --password-stdin "$registre" >/dev/null 2>&1 ||
    echec "connexion au registre ECR refusee"
  docker tag "$KEYCLOAK_IMAGE" "$registre/$DEPOT_ECR:26.8.1"
  docker push -q "$registre/$DEPOT_ECR:26.8.1" >/dev/null || { docker logout "$registre" >/dev/null 2>&1; echec "push ECR en echec"; }
  docker logout "$registre" >/dev/null 2>&1 || true
  empreinte="$(docker image inspect "$KEYCLOAK_IMAGE" --format '{{range .RepoDigests}}{{println .}}{{end}}' | tr -d '\r' |
    grep "^$registre/$DEPOT_ECR@sha256:" | head -1)"
  [[ -n "$empreinte" ]] || echec "empreinte ECR introuvable apres push"
  printf 'KEYCLOAK_IMAGE_EKS=%s\n' "$empreinte" >>"$EKS_ENV"
  journal "Image poussee dans ECR : $(masquer <<<"$empreinte")"
}

base_rds() {
  # Hote RDS et compte maitre (secret gere par RDS) : lus dans AWS, poses en Secret, jamais affiches.
  local hote arn
  read -r hote arn < <(aws rds describe-db-instances --db-instance-identifier "$RDS_ID" \
    --query 'DBInstances[0].[Endpoint.Address,MasterUserSecret.SecretArn]' --output text | tr -d '\r')
  [[ -n "$hote" && "$arn" == arn:* ]] || echec "instance RDS $RDS_ID introuvable ou sans secret gere"
  aws secretsmanager get-secret-value --secret-id "$arn" --query SecretString --output text | tr -d '\r' |
    jq -r --arg h "$hote" '"PGHOST=\($h)\nPGUSER=\(.username)\nPGPASSWORD=\(.password)"' |
    secret_stdin identite-test rds-maitre
  printf 'RDS_HOTE=%s\n' "$hote" >>"$EKS_ENV"
  RDS_HOTE="$hote"
}

SQL_REINIT='SELECT '"'"'CREATE ROLE keycloak LOGIN'"'"' WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '"'"'keycloak'"'"')
\gexec
ALTER ROLE keycloak WITH LOGIN PASSWORD :'"'"'mdp'"'"';
DROP DATABASE IF EXISTS keycloak WITH (FORCE);
CREATE DATABASE keycloak;
GRANT CONNECT, TEMPORARY ON DATABASE keycloak TO keycloak;
\c keycloak
GRANT USAGE, CREATE ON SCHEMA public TO keycloak;
'

reinit_base_rds() {
  # Base keycloak neuve dans RDS (compte et base dedies), par un pod psql jetable du namespace identite-test.
  local nom="xp03-base-$(openssl rand -hex 3)"
  jq -n --arg nom "$nom" --arg image "$POSTGRES_IMAGE" --arg sql "$SQL_REINIT" '
    { apiVersion: "v1", kind: "Pod",
      metadata: { name: $nom, namespace: "identite-test", labels: { app: "base-admin", "app.kubernetes.io/part-of": "xp03" } },
      spec: { restartPolicy: "Never", automountServiceAccountToken: false,
        securityContext: { runAsNonRoot: true, runAsUser: 999, runAsGroup: 999, seccompProfile: { type: "RuntimeDefault" } },
        containers: [ { name: "psql", image: $image, command: ["sh", "-c"],
          args: ["printf \"%s\" \"$SQL\" | psql -X -q -v ON_ERROR_STOP=1 -v mdp=\"$MDP_KEYCLOAK\" -d postgres -f -"],
          env: [ { name: "SQL", value: $sql }, { name: "PGSSLMODE", value: "require" }, { name: "PGCONNECT_TIMEOUT", value: "10" },
                 { name: "MDP_KEYCLOAK", valueFrom: { secretKeyRef: { name: "keycloak-session", key: "MDP_POSTGRES" } } } ],
          envFrom: [ { secretRef: { name: "rds-maitre" } } ],
          resources: { requests: { cpu: "50m", memory: "64Mi" }, limits: { memory: "128Mi" } },
          securityContext: { allowPrivilegeEscalation: false, capabilities: { drop: ["ALL"] } } } ] } }' |
    kubectl apply -f - >/dev/null
  if ! kubectl -n identite-test wait pod "$nom" --for=jsonpath='{.status.phase}'=Succeeded --timeout=180s >/dev/null; then
    kubectl -n identite-test logs "$nom" >&2 || true
    kubectl -n identite-test delete pod "$nom" --wait=false >/dev/null
    echec "reinitialisation de la base keycloak dans RDS en echec"
  fi
  kubectl -n identite-test delete pod "$nom" --wait=false >/dev/null
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
  mkdir -p "$GENERE_DIR"
  if [[ "$PLATEFORME" == eks ]]; then
    : >"$EKS_ENV"
    image_ecr
  else
    k3d image import -c "$cluster" "$KEYCLOAK_IMAGE" >/dev/null 2>&1
  fi
  kubectl apply -f k8s/00-namespaces.yaml >/dev/null
  printf 'MDP_POSTGRES=%s\nMDP_KC_BOOTSTRAP=%s\n' "$MDP_POSTGRES" "$MDP_KC_BOOTSTRAP" | secret_stdin identite-test keycloak-session
  printf 'URL_FRONT=%s\nSECRET_TRANSACTIONS=%s\nSECRET_AUTRE=%s\nMDP_ACHETEUR=%s\nMDP_VENDEUR=%s\nMDP_ADMIN_TEST=%s\nGRAINE_OTP=%s\n' \
    "$URL_FRONT" "$SECRET_TRANSACTIONS" "$SECRET_AUTRE" "$MDP_ACHETEUR" "$MDP_VENDEUR" "$MDP_ADMIN_TEST" "$GRAINE_OTP" |
    secret_stdin identite-test keycloak-realm-valeurs
  if [[ "$PLATEFORME" == eks ]]; then
    kubectl apply -f k8s/40-eks.yaml >/dev/null
    if [[ "$BASE" == rds ]]; then
      base_rds
      reinit_base_rds
    fi
    source "$EKS_ENV"
  fi
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
  if [[ "$BASE" == rds ]]; then
    kubectl -n identite-test scale deploy/postgres --replicas=0 >/dev/null
  else
    kubectl -n identite-test rollout status deploy/postgres --timeout=180s
  fi
  kubectl -n identite-test rollout status deploy/keycloak --timeout=300s
  kubectl -n pdm-test rollout status deploy/verificateur --timeout=120s
  kubectl -n transactions-test rollout status deploy/appelant --timeout=120s
  journal "Pret ; emetteur $ISSUER ; verifications : PLATEFORME=$PLATEFORME scripts/verifier.sh k3|ma12|ma"
}

duree_pret() {
  # Secondes entre la creation du pod Keycloak et sa condition Ready.
  kubectl -n identite-test get pod -l app=keycloak -o json | jq -r '
    .items[0] | (.metadata.creationTimestamp | fromdateiso8601) as $c
    | (.status.conditions[] | select(.type == "Ready" and .status == "True") | .lastTransitionTime | fromdateiso8601) - $c'
}

k2() {
  csv_init
  charger_eks
  local n d
  for n in 1 2; do
    if [[ "$BASE" == rds ]]; then
      # Keycloak arrete avant la suppression de la base, puis recree : la mesure part de la creation du pod.
      kubectl -n identite-test scale deploy/keycloak --replicas=0 >/dev/null
      kubectl -n identite-test wait pod -l app=keycloak --for=delete --timeout=180s >/dev/null 2>&1 || true
      reinit_base_rds
      kubectl -n identite-test scale deploy/keycloak --replicas=1 >/dev/null
    else
      kubectl -n identite-test delete pod -l 'app in (keycloak,postgres)' --wait=true >/dev/null
    fi
    kubectl -n identite-test rollout status deploy/keycloak --timeout=300s >/dev/null
    d="$(duree_pret)"
    csv K2 "mesure $n" "$d" s "<=180" "$([[ $d -le 180 ]] && echo oui || echo non)" "creation du pod -> Ready (/health/ready 9000), base neuve ($BASE), import du realm ($PLATEFORME)"
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
  local sortie kc ok=0 total=0 code noeuds=()
  sortie="$SORTIES_DIR/$(horodatage)-$PLATEFORME-g7-4.txt"
  kc="keycloak.identite-test.svc"
  # EKS : le NodePort de Keycloak (30081) n'est ouvert qu'aux sous-reseaux de l'ALB ; un pod quelconque est refuse.
  [[ "$PLATEFORME" != eks ]] || mapfile -t noeuds < <(kubectl get nodes -o json | tr -d '\r' |
    jq -r '.items[].status.addresses[] | select(.type == "InternalIP") | .address + ":30081"' | tr -d '\r')
  {
    essai_port ns-intrus "$kc:8080" "$kc:9000" "verificateur.pdm-test.svc:8081" "postgres.identite-test.svc:5432" "${noeuds[@]}"
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
  if [[ "$PLATEFORME" == eks ]]; then
    # Temoin du protocole : ALB -> page de connexion, depuis le poste.
    code="$(curl -sS --ssl-revoke-best-effort -o "$POUBELLE" -w '%{http_code}' \
      "$URL_AUTH/realms/collector/protocol/openid-connect/auth?client_id=collector-web&response_type=code&scope=openid&redirect_uri=$(jq -rn --arg v "$URL_FRONT/rappel" '$v|@uri')&code_challenge=$(openssl rand -hex 22)&code_challenge_method=S256" || true)"
    echo "poste-via-alb page-de-connexion $code" >>"$sortie"
    total=$((total + 1)); [[ "$code" == 200 ]] && ok=$((ok + 1))
    csv TEMOIN reseau "$code" http 200 "$(oui_non "$code" 200)" "poste -> ALB -> page de connexion"
  fi
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
  if [[ "$PLATEFORME" == eks ]]; then
    # MX7 : depot ECR, secrets de session et image locale etiquetee pour ECR supprimes sans delai.
    # Le compte et la base keycloak de RDS disparaissent avec l'instance (down.sh de XP01).
    aws ecr delete-repository --repository-name "$DEPOT_ECR" --force >/dev/null 2>&1 || true
    aws ecr describe-repositories --repository-names "$DEPOT_ECR" >/dev/null 2>&1 && echec "depot ECR $DEPOT_ECR toujours present"
    docker image ls --format '{{.Repository}}:{{.Tag}}' | tr -d '\r' | grep "/$DEPOT_ECR:" | xargs -r docker rmi >/dev/null 2>&1 || true
    rm -f "$EKS_ENV" "$ENV_FILE"
    journal "Depot ECR, .env et .genere/eks.env supprimes"
  fi
  journal "XP03 retire (namespaces partages conserves)"
}

k1() {
  # K1 : memoire de Keycloak et des noeuds toutes les 30 s ; OOMKilled et Pending sur tout le cluster.
  local duree="${1:-900}" etape="${2:-repos}" fin sortie m max=0 attente pending_max=0 oom
  csv_init
  sortie="$SORTIES_DIR/$(horodatage)-$PLATEFORME-k1-$etape.txt"
  fin=$(( $(date +%s) + duree ))
  while :; do
    { echo "== $(date -u +%H:%M:%SZ)"; kubectl top node --no-headers 2>&1; kubectl top pod -A --no-headers 2>&1; } | tr -d '\r' >>"$sortie"
    m="$(kubectl -n identite-test top pod -l app=keycloak --no-headers 2>/dev/null | tr -d '\r' | awk '{sub(/Mi$/, "", $3); print $3 + 0; exit}')"
    (( ${m:-0} > max )) && max="$m"
    attente="$(kubectl get pods -A --field-selector=status.phase=Pending --no-headers 2>/dev/null | grep -c . || true)"
    (( attente > pending_max )) && pending_max="$attente"
    (( $(date +%s) < fin )) || break
    sleep 30
  done
  oom="$(kubectl get pods -A -o json | tr -d '\r' | jq '[.items[].status.containerStatuses[]? |
    select(.lastState.terminated.reason == "OOMKilled" or .state.terminated.reason == "OOMKilled")] | length')"
  csv K1 "$etape : memoire max de Keycloak" "$max" Mio "<=1280" "$([[ $max -le 1280 ]] && echo oui || echo non)" "kubectl top toutes les 30 s pendant $duree s"
  csv K1 "$etape : conteneurs OOMKilled" "$oom" conteneurs 0 "$(oui_non "$oom" 0)" "tous namespaces"
  csv K1 "$etape : pods Pending (max)" "$pending_max" pods 0 "$(oui_non "$pending_max" 0)" "tous namespaces"
  journal "K1 $etape : Keycloak max $max Mio, OOMKilled $oom, Pending max $pending_max (detail : $sortie)"
  [[ "$oom" -eq 0 && "$pending_max" -eq 0 ]]
}

case "${1:-}" in
  up) up ;;
  k2) k2 ;;
  k1) shift; k1 "$@" ;;
  reseau) reseau ;;
  down) down ;;
  *) echec "usage : $0 up | k2 | k1 [secondes] [etape] | reseau | down" ;;
esac
