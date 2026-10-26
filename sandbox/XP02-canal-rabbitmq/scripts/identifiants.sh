#!/usr/bin/env bash
# Genere les identifiants du canal pour une execution (MX1, MC5) :
#   - .env (ignore par git) : un mot de passe par identite + cookie Erlang ;
#   - .genere/00-utilisateurs.json (ignore par git) : utilisateurs et empreintes SHA-256 salees,
#     charge par RabbitMQ a cote de broker/definitions/10-topologie.json.
# Aucun mot de passe n'est affiche ni passe en argument de commande.
# Usage : scripts/identifiants.sh [--force]
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

if [[ -f "$ENV_FILE" && "${1:-}" != "--force" ]]; then
  echec "$ENV_FILE existe deja (session en cours ?) ; --force pour regenerer"
fi

mdp_aleatoire() { openssl rand -base64 48 | tr -dc 'A-Za-z0-9' | head -c 32; }

empreinte_rabbitmq() {
  # base64( sel(4 octets) || sha256( sel || mot de passe ) ) : format rabbit_password_hashing_sha256.
  local sel_hex
  sel_hex="$(openssl rand -hex 4)"
  {
    printf '%s' "$sel_hex" | xxd -r -p
    { printf '%s' "$sel_hex" | xxd -r -p; printf '%s' "$1"; } | openssl dgst -sha256 -binary
  } | base64 -w0
}

etiquettes() {
  case "$1" in
    keda-monitoring) printf '["monitoring"]' ;;
    admin-canal) printf '["administrator"]' ;;
    *) printf '[]' ;;
  esac
}

mkdir -p "$GENERE_DIR"
umask 077
env_tmp="$(mktemp "$XP_DIR/.env.XXXXXX")"
json_tmp="$(mktemp "$GENERE_DIR/utilisateurs.XXXXXX")"
trap 'rm -f "$env_tmp" "$json_tmp"' EXIT

{
  printf '# Genere par scripts/identifiants.sh le %s ; ne jamais versionner.\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf 'COOKIE_ERLANG=%s\n' "$(openssl rand -hex 32 | tr 'a-f' 'A-F')"
} >"$env_tmp"

printf '{\n  "users": [\n' >"$json_tmp"
premier=1
for u in "${UTILISATEURS[@]}"; do
  mdp="$(mdp_aleatoire)"
  printf '%s=%s\n' "$(variable_mdp "$u")" "$mdp" >>"$env_tmp"
  [[ $premier -eq 1 ]] || printf ',\n' >>"$json_tmp"
  premier=0
  printf '    { "name": "%s", "password_hash": "%s", "hashing_algorithm": "rabbit_password_hashing_sha256", "tags": %s, "limits": {} }' \
    "$u" "$(empreinte_rabbitmq "$mdp")" "$(etiquettes "$u")" >>"$json_tmp"
done
printf '\n  ]\n}\n' >>"$json_tmp"
unset mdp

mv "$env_tmp" "$ENV_FILE"
mv "$json_tmp" "$GENERE_DIR/00-utilisateurs.json"
# Le cookie est aussi materialise en fichier (monte en secret docker compose).
( set -a; source "$ENV_FILE"; printf '%s' "$COOKIE_ERLANG" >"$GENERE_DIR/cookie-erlang" )
trap - EXIT

journal "Identifiants generes pour : ${UTILISATEURS[*]} (valeurs dans .env et .genere/, ignores par git)"
