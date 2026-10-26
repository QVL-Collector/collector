#!/usr/bin/env bash
# XP03 local (docker compose) : cycle de vie.
# Usage :
#   scripts/compose.sh image   construit l'image Keycloak optimisee (kc.sh build) et affiche son empreinte
#   scripts/compose.sh up      secrets (si absents), image, verificateur, pile prete, autorite locale exportee
#   scripts/compose.sh down    supprime conteneurs, reseaux, volume Caddy, .env et .genere/
#                              (--garder-identifiants pour conserver .env)
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
PLATEFORME=compose

dc() {
  docker compose -p "$COMPOSE_PROJET" --env-file "$(chemin_hote "$XP_DIR/versions.env")" \
    -f "$(chemin_hote "$XP_DIR/docker-compose.yml")" "$@"
}

image() {
  docker build -q --build-arg "KEYCLOAK_BASE_IMAGE=$KEYCLOAK_BASE_IMAGE" -t "$KEYCLOAK_IMAGE" \
    "$(chemin_hote "$XP_DIR/keycloak")" >/dev/null
  journal "Image $KEYCLOAK_IMAGE : $(docker image inspect "$KEYCLOAK_IMAGE" --format '{{.Id}}')"
}

rendre_verificateur() {
  # rendre_verificateur <hote keycloak> <port> <fichier de sortie>
  mkdir -p "$(dirname "$3")"
  sed -e "s|__ISSUER__|$ISSUER|" -e "s|__KEYCLOAK_HOTE__|$1|g" -e "s|__KEYCLOAK_PORT__|$2|g" \
    "$XP_DIR/verificateur/envoy.yaml" >"$3"
}

up() {
  [[ -f "$ENV_FILE" ]] || "$XP_DIR/scripts/identifiants.sh"
  charger_env
  docker image inspect "$KEYCLOAK_IMAGE" >/dev/null 2>&1 || image
  rendre_verificateur keycloak 8080 "$GENERE_DIR/envoy.yaml"
  local debut; debut="$(ms_maintenant)"
  (cd "$XP_DIR" && dc up -d --wait --wait-timeout 300)
  journal "Pile prete en $(( ($(ms_maintenant) - debut) / 1000 )) s (dont Keycloak sur base neuve)"
  # Autorite locale de Caddy, pour curl --cacert (jamais -k).
  docker cp "$(dc ps -q caddy):/data/caddy/pki/authorities/local/root.crt" "$(chemin_hote "$CA_LOCALE")" >/dev/null
  journal "Autorite locale : .genere/caddy-racine.crt ; emetteur : $ISSUER"
}

down() {
  (cd "$XP_DIR" && dc down -v --remove-orphans)
  docker volume rm xp03-trivy-cache >/dev/null 2>&1 || true
  rm -rf "$GENERE_DIR"
  if [[ "${1:-}" != "--garder-identifiants" ]]; then
    rm -f "$ENV_FILE"
    journal "Secrets de la session supprimes"
  fi
}

case "${1:-}" in
  image) image ;;
  up) up ;;
  down) shift; down "$@" ;;
  dc) shift; dc "$@" ;;
  *) echec "usage : $0 image | up | down [--garder-identifiants] | dc <arguments docker compose>" ;;
esac
