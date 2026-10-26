#!/usr/bin/env bash
# Genere les identifiants du canal pour une session (MC4, MC5, AC6), independamment de l'orchestrateur :
#   .genere/mots-de-passe.env     un mot de passe par identite + cookie Erlang ;
#   .genere/00-utilisateurs.json  utilisateurs, etiquettes et empreintes SHA-256 salees, charge par
#                                 RabbitMQ a cote de broker/definitions/10-topologie.json ;
#   .genere/cookie-erlang         cookie de la session.
# Aucun mot de passe n'est affiche ni passe en argument de commande.
# Usage : outils/identifiants.sh [--force]
# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

if [[ -f "$MDP_FILE" && "${1:-}" != "--force" ]]; then
  echec "$MDP_FILE existe deja (session en cours ?) ; --force pour regenerer"
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
    keda) printf '["monitoring"]' ;;
    admin-canal) printf '["administrator"]' ;;
    *) printf '[]' ;;
  esac
}

mkdir -p "$GENERE_DIR"
umask 077
mdp_tmp="$(mktemp "$GENERE_DIR/mdp.XXXXXX")"
json_tmp="$(mktemp "$GENERE_DIR/utilisateurs.XXXXXX")"
trap 'rm -f "$mdp_tmp" "$json_tmp"' EXIT

cookie="$(openssl rand -hex 32 | tr 'a-f' 'A-F')"
{
  printf '# Genere par outils/identifiants.sh le %s ; ne jamais versionner.\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf 'COOKIE_ERLANG=%s\n' "$cookie"
} >"$mdp_tmp"

printf '{\n  "users": [\n' >"$json_tmp"
premier=1
for u in "${UTILISATEURS[@]}"; do
  mdp="$(mdp_aleatoire)"
  printf '%s=%s\n' "$(variable_mdp "$u")" "$mdp" >>"$mdp_tmp"
  [[ $premier -eq 1 ]] || printf ',\n' >>"$json_tmp"
  premier=0
  printf '    { "name": "%s", "password_hash": "%s", "hashing_algorithm": "rabbit_password_hashing_sha256", "tags": %s, "limits": {} }' \
    "$u" "$(empreinte_rabbitmq "$mdp")" "$(etiquettes "$u")" >>"$json_tmp"
done
printf '\n  ]\n}\n' >>"$json_tmp"
unset mdp

mv "$mdp_tmp" "$MDP_FILE"
mv "$json_tmp" "$UTILISATEURS_JSON"
printf '%s' "$cookie" >"$COOKIE_FILE"
unset cookie
trap - EXIT

journal "Identifiants generes pour : ${UTILISATEURS[*]} (dans .genere/, ignore par git et par Helm)"
