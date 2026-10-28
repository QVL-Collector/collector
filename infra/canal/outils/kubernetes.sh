#!/usr/bin/env bash
# Canal sur Kubernetes (EKS, k3d) : namespaces, Secrets par l'entree standard, chart du broker.
# KUBE_CONTEXT est obligatoire et doit etre en liste blanche (lib.sh, exiger_contexte).
# Usage :
#   outils/kubernetes.sh namespaces   namespaces du broker et des services, PSS "restricted" (D10 : hors chart)
#   outils/kubernetes.sh secrets      Secret du broker, un Secret par service, Secret de supervision (MC5)
#   outils/kubernetes.sh up           namespaces, identifiants de session, Secrets, chart, guest absent
#   outils/kubernetes.sh console      console de gestion par port-forward sur 127.0.0.1:15672 (MC4, AC7)
#   outils/kubernetes.sh down         desinstalle le chart, supprime les Secrets, efface .genere/
# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
exiger_contexte

NS_CANAL="${NS_CANAL:-canal}"
RELEASE="canal"
NS_SUPERVISION_CIBLE="$(namespace_de controle)"
ETIQUETTE="app.kubernetes.io/part-of=collector-canal"

namespaces_clients() { local s; for s in "${SERVICES[@]}"; do namespace_de "$s"; printf '\n'; done; }

namespaces() {
  local ns
  for ns in "$NS_CANAL" $(namespaces_clients); do
    printf 'apiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n  labels:\n    pod-security.kubernetes.io/enforce: restricted\n    pod-security.kubernetes.io/warn: restricted\n' "$ns" |
      kc apply -f - >/dev/null
  done
  journal "Namespaces prets : $NS_CANAL $(namespaces_clients | tr '\n' ' ')"
}

manifeste_secret() {
  printf 'apiVersion: v1\nkind: Secret\ntype: Opaque\nmetadata:\n  name: %s\n  namespace: %s\n  labels:\n    %s\ndata:\n' \
    "$2" "$1" "${ETIQUETTE/=/: }"
  local cle valeur
  while read -r cle valeur; do printf '  %s: %s\n' "$cle" "$valeur"; done
}

b64() { printf '%s' "$1" | base64 -w0; }

secrets() {
  charger_mdp
  local s var url
  {
    printf '00-utilisateurs.json %s\n' "$(base64 -w0 <"$UTILISATEURS_JSON")"
    printf 'cookie-erlang %s\n' "$(base64 -w0 <"$COOKIE_FILE")"
  } | manifeste_secret "$NS_CANAL" rabbitmq-interne | kc apply -f - >/dev/null

  for s in "${SERVICES[@]}"; do
    var="$(variable_mdp "$s")"
    url="amqp://$s:${!var}@rabbitmq.$NS_CANAL.svc:5672/$VHOST"
    printf 'AMQP_URL %s\n' "$(b64 "$url")" | manifeste_secret "$(namespace_de "$s")" canal-amqp | kc apply -f - >/dev/null
  done
  url="http://keda:${MDP_KEDA}@rabbitmq.$NS_CANAL.svc:15672/$VHOST"
  printf 'host %s\n' "$(b64 "$url")" | manifeste_secret "$NS_SUPERVISION_CIBLE" canal-supervision | kc apply -f - >/dev/null
  unset url
  journal "Secrets poses : $NS_CANAL/rabbitmq-interne, canal-amqp par service, $NS_SUPERVISION_CIBLE/canal-supervision"
}

empreinte_secret_interne() {
  { kc -n "$NS_CANAL" get secret rabbitmq-interne -o jsonpath='{.data}' 2>/dev/null || true; } | sha256sum | cut -d' ' -f1
}

up() {
  namespaces
  [[ -f "$MDP_FILE" ]] || "$CANAL_DIR/outils/identifiants.sh"
  local deja=0 avant apres
  kc -n "$NS_CANAL" get statefulset rabbitmq >/dev/null 2>&1 && deja=1
  avant="$(empreinte_secret_interne)"
  secrets
  apres="$(empreinte_secret_interne)"
  hm upgrade --install "$RELEASE" "$(chemin_hote "$CANAL_DIR")" --namespace "$NS_CANAL" \
    --wait --timeout 6m >/dev/null
  if [[ $deja -eq 1 && "$avant" != "$apres" ]]; then
    journal "Identifiants changes : redemarrage du broker (files en emptyDir perdues)"
    kc -n "$NS_CANAL" rollout restart statefulset/rabbitmq >/dev/null
  fi
  kc -n "$NS_CANAL" rollout status statefulset/rabbitmq --timeout=360s >/dev/null
  if kc -n "$NS_CANAL" exec rabbitmq-0 -c rabbitmq -- rabbitmqctl -q list_users | grep -qw guest; then
    echec "utilisateur guest present"
  fi
  journal "Canal pret dans $NS_CANAL (release $RELEASE, guest absent)"
}

console() {
  journal "Console : http://127.0.0.1:15672 (identite admin-canal, mot de passe dans .genere/mots-de-passe.env) ; Ctrl+C pour fermer"
  kc -n "$NS_CANAL" port-forward svc/rabbitmq 15672:15672
}

down() {
  local ns
  hm uninstall "$RELEASE" --namespace "$NS_CANAL" --wait >/dev/null 2>&1 || true
  for ns in "$NS_CANAL" $(namespaces_clients); do
    kc -n "$ns" delete secret -l "$ETIQUETTE" --ignore-not-found >/dev/null 2>&1 || true
  done
  rm -rf "$GENERE_DIR"
  journal "Chart desinstalle, Secrets du canal supprimes, identifiants de session effaces (namespaces conserves)"
}

case "${1:-}" in
  namespaces) namespaces ;;
  secrets) secrets ;;
  up) up ;;
  console) console ;;
  down) down ;;
  *) echec "usage : $0 namespaces|secrets|up|console|down" ;;
esac
