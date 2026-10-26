#!/usr/bin/env bash
# Canal en local par docker compose (developpement, tests d'integration a la main).
# Usage :
#   outils/local.sh up     identifiants de session, broker pret, guest absent
#   outils/local.sh down   supprime conteneur et reseau, efface .genere/
# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
CONTENEUR="collector-canal-rabbitmq-1"

compose() { docker compose -f "$(chemin_hote "$CANAL_DIR/compose.yml")" "$@"; }

up() {
  [[ -f "$MDP_FILE" ]] || "$CANAL_DIR/outils/identifiants.sh"
  compose up -d --wait --wait-timeout 180 >/dev/null
  if docker exec --user rabbitmq "$CONTENEUR" rabbitmqctl -q list_users | grep -qw guest; then
    echec "utilisateur guest present"
  fi
  journal "Broker pret sur 127.0.0.1:5672 (vhost $VHOST) ; mots de passe dans .genere/mots-de-passe.env"
}

down() {
  compose down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$GENERE_DIR"
  journal "Canal local supprime, identifiants de session effaces"
}

case "${1:-}" in
  up) up ;;
  down) down ;;
  *) echec "usage : $0 up|down" ;;
esac
