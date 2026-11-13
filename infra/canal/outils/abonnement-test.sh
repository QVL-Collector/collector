#!/usr/bin/env bash
# Abonnement de test sur la diffusion publication.decidee.v1 (US-029, ES-38, M-CAN-06) : pose et retire par
# l'exploitation (admin-canal, API de gestion), jamais par un service. Identite abonne-test en lecture seule,
# mot de passe de la session dans .genere/mots-de-passe.env (MDP_ABONNE_TEST), jamais affiche.
# Usage :
#   outils/abonnement-test.sh poser [local|kubernetes]     file quorum bornee, liaison, identite, droits
#   outils/abonnement-test.sh lire [local|kubernetes]      nombre de messages et premiers corps, remis en file
#   outils/abonnement-test.sh retirer [local|kubernetes]   liaison, file, identite, mot de passe ; verifie qu'il ne reste rien
# kubernetes : KUBE_CONTEXT obligatoire (lib.sh, exiger_contexte), API par port-forward sur 127.0.0.1:15673.
# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

ACTION="${1:-}"
CIBLE="${2:-local}"
ABONNE="abonne-test"
FILE_TEST="test.publication-decidee"
ECHANGE="pdm.evenements"
CLE="publication.decidee.v1"
LIRE_MAX="${LIRE_MAX:-5}"
NS_CANAL="${NS_CANAL:-canal}"
PORT_KUBE=15673
VAR_ABONNE="$(variable_mdp "$ABONNE")"

case "$ACTION" in poser | lire | retirer) ;; *) echec "usage : $0 poser|lire|retirer [local|kubernetes]" ;; esac
case "$CIBLE" in
  local) BASE="http://127.0.0.1:15672" ;;
  kubernetes) exiger_contexte; BASE="http://127.0.0.1:$PORT_KUBE" ;;
  *) echec "usage : $0 poser|lire|retirer [local|kubernetes]" ;;
esac
for outil in curl jq; do command -v "$outil" >/dev/null 2>&1 || echec "$outil requis"; done
charger_mdp

REPONSE="$(mktemp "$GENERE_DIR/reponse.XXXXXX")"
PF=""
nettoyer() {
  rm -f "$REPONSE"
  [[ -z "$PF" ]] || kill "$PF" 2>/dev/null || true
}
trap nettoyer EXIT

api() {
  local methode="$1" chemin="$2" corps="${3:-}"
  {
    printf 'user = "admin-canal:%s"\n' "$MDP_ADMIN_CANAL"
    [[ -z "$corps" ]] || printf 'data-binary = "%s"\n' "${corps//\"/\\\"}"
  } | curl -s -K - -X "$methode" -H 'content-type: application/json' --max-time 10 \
    -o "$(chemin_hote "$REPONSE")" -w '%{http_code}' "$BASE/api/$chemin" || true
}

exiger() {
  local code="$1" quoi="$2" attendu
  shift 2
  for attendu in "$@"; do [[ "$code" == "$attendu" ]] && return 0; done
  echec "$quoi : HTTP $code ($(head -c 300 "$REPONSE" 2>/dev/null || true))"
}

ouvrir_kubernetes() {
  [[ "$(api GET overview)" == 000 ]] || echec "127.0.0.1:$PORT_KUBE repond deja (port-forward oublie ?) : fermer avant de continuer"
  kubectl --context "$KUBE_CONTEXT" -n "$NS_CANAL" port-forward svc/rabbitmq "$PORT_KUBE:15672" >/dev/null 2>&1 &
  PF=$!
  for _ in $(seq 1 20); do
    kill -0 "$PF" 2>/dev/null || echec "port-forward arrete (127.0.0.1:$PORT_KUBE occupe ?)"
    [[ "$(api GET overview)" == 200 ]] && return 0
    sleep 1
  done
  echec "API de gestion injoignable par port-forward sur 127.0.0.1:$PORT_KUBE"
}

ranger_mdp() {
  local tmp
  tmp="$(mktemp "$GENERE_DIR/mdp.XXXXXX")"
  { grep -v "^$VAR_ABONNE=" "$MDP_FILE" || true; [[ -z "${1:-}" ]] || printf '%s=%s\n' "$VAR_ABONNE" "$1"; } >"$tmp"
  mv "$tmp" "$MDP_FILE"
}

poser() {
  local mdp
  exiger "$(api PUT "queues/$VHOST/$FILE_TEST" \
    '{"durable":true,"auto_delete":false,"arguments":{"x-queue-type":"quorum","x-max-length":10000,"x-overflow":"drop-head","x-message-ttl":86400000}}')" \
    "file $FILE_TEST" 201 204
  exiger "$(api POST "bindings/$VHOST/e/$ECHANGE/q/$FILE_TEST" "{\"routing_key\":\"$CLE\",\"arguments\":{}}")" \
    "liaison $ECHANGE -> $FILE_TEST" 201
  mdp="$(mdp_aleatoire)"
  ranger_mdp "$mdp"
  exiger "$(api PUT "users/$ABONNE" "{\"password\":\"$mdp\",\"tags\":\"\"}")" "identite $ABONNE" 201 204
  unset mdp
  exiger "$(api PUT "permissions/$VHOST/$ABONNE" '{"configure":"","write":"","read":"^test[.]publication-decidee$"}')" \
    "droits de $ABONNE" 201 204
  exiger "$(api GET "permissions/$VHOST/$ABONNE")" "relecture des droits" 200
  journal "Abonnement de test pose ($CIBLE) : $ECHANGE --$CLE--> $FILE_TEST ; $ABONNE $(jq -r '"configure=\"\(.configure)\" write=\"\(.write)\" read=\"\(.read)\""' <"$REPONSE")"
}

lire() {
  exiger "$(api GET "queues/$VHOST/$FILE_TEST")" "file $FILE_TEST" 200
  jq -r '"\(.name) : \(.messages // 0) message(s), \(.messages_ready // 0) prets, \(.messages_unacknowledged // 0) non acquittes (statistiques de gestion)"' <"$REPONSE"
  exiger "$(api POST "queues/$VHOST/$FILE_TEST/get" \
    "{\"count\":$LIRE_MAX,\"ackmode\":\"ack_requeue_true\",\"encoding\":\"auto\",\"truncate\":50000}")" "lecture de $FILE_TEST" 200
  jq -r 'length as $n | "\($n) message(s) lu(s) (au plus '"$LIRE_MAX"', remis en file)",
    (.[] | "- cle=\(.routing_key) user_id=\(.properties.user_id // "-") message_id=\(.properties.message_id // "-")",
      "  \(.payload | (fromjson? | tojson) // .)")' <"$REPONSE"
}

retirer() {
  local restes=()
  exiger "$(api DELETE "bindings/$VHOST/e/$ECHANGE/q/$FILE_TEST/$CLE")" "liaison $ECHANGE -> $FILE_TEST" 204 404
  exiger "$(api DELETE "queues/$VHOST/$FILE_TEST")" "file $FILE_TEST" 204 404
  exiger "$(api DELETE "users/$ABONNE")" "identite $ABONNE" 204 404
  ranger_mdp ""

  [[ "$(api GET "queues/$VHOST/$FILE_TEST")" == 404 ]] || restes+=("file")
  [[ "$(api GET "users/$ABONNE")" == 404 ]] || restes+=("identite")
  [[ "$(api GET "permissions/$VHOST/$ABONNE")" == 404 ]] || restes+=("droits")
  exiger "$(api GET "exchanges/$VHOST/$ECHANGE/bindings/source")" "liaisons de $ECHANGE" 200
  [[ "$(jq --arg f "$FILE_TEST" '[.[] | select(.destination == $f)] | length' <"$REPONSE")" == 0 ]] || restes+=("liaison")
  ! grep -q "^$VAR_ABONNE=" "$MDP_FILE" || restes+=("mot de passe")
  [[ ${#restes[@]} -eq 0 ]] || echec "retrait incomplet ($CIBLE) : ${restes[*]}"
  journal "Abonnement de test retire ($CIBLE) : liaison, file, identite, droits et mot de passe absents"
}

[[ "$CIBLE" != kubernetes ]] || ouvrir_kubernetes
"$ACTION"
