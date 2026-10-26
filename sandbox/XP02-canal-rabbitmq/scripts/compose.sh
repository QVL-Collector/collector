#!/usr/bin/env bash
# Local 1 (docker compose) : cycle de vie du broker.
# Usage :
#   scripts/compose.sh up      identifiants (si absents), image de la sonde, broker pret, guest absent
#   scripts/compose.sh down    arret, suppression du conteneur, du reseau, de .env et .genere/
#                              (--garder-identifiants pour conserver .env et .genere/)
#   scripts/compose.sh ctl <commande>   commande dans le conteneur (ex. rabbitmqctl -q list_queues -p collector)
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
PLATEFORME=compose

dc() {
  docker compose -p "$COMPOSE_PROJET" --env-file "$(chemin_hote "$XP_DIR/versions.env")" \
    -f "$(chemin_hote "$XP_DIR/docker-compose.yml")" "$@"
}

construire_sonde() {
  docker build -q --build-arg "BASE_IMAGE=$SONDE_BASE_IMAGE" -t "$SONDE_IMAGE" "$(chemin_hote "$XP_DIR/sonde")" >/dev/null
  journal "Image de la sonde : $(docker image inspect "$SONDE_IMAGE" --format '{{.Id}}')"
}

up() {
  [[ -f "$ENV_FILE" ]] || "$XP_DIR/scripts/identifiants.sh"
  [[ -f "$GENERE_DIR/00-utilisateurs.json" ]] || echec ".genere/00-utilisateurs.json absent : scripts/identifiants.sh --force"
  construire_sonde
  local debut fin
  debut="$(ms_maintenant)"
  (cd "$XP_DIR" && dc up -d --wait --wait-timeout 180)
  fin="$(ms_maintenant)"
  journal "Broker pret en $(( (fin - debut) / 1000 )) s"
  rmq rabbitmq-diagnostics -q check_running
  if rmq rabbitmqctl -q list_users | grep -qw guest; then echec "utilisateur guest present (XC9)"; fi
  journal "check_running OK ; guest absent"
}

down() {
  (cd "$XP_DIR" && dc down -v --remove-orphans)
  if [[ "${1:-}" != "--garder-identifiants" ]]; then
    rm -f "$ENV_FILE"
    rm -rf "$GENERE_DIR"
    journal "Identifiants de la session supprimes"
  fi
}

case "${1:-}" in
  up) up ;;
  down) shift; down "$@" ;;
  ctl) shift; rmq "$@" ;;
  *) echec "usage : $0 up | down [--garder-identifiants] | ctl <commande>" ;;
esac
