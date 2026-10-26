# Bibliotheque commune des scripts XP03 (sourcee, jamais executee seule).
# shellcheck shell=bash

set -euo pipefail
export MSYS_NO_PATHCONV=1

XP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPOT_DIR="$(cd "$XP_DIR/../.." && pwd)"
GENERE_DIR="$XP_DIR/.genere"
ENV_FILE="$XP_DIR/.env"
COMPOSE_PROJET="xp03"
XP_ID="XP03"

MESURES_DIR="${MESURES_DIR:-$DEPOT_DIR/docs/mesures}"
CSV="$MESURES_DIR/XP03-identite-exposition.csv"
SORTIES_DIR="$MESURES_DIR/XP03-identite-exposition"
SESSION="${SESSION:-$(date -u +%Y%m%d)}"
PLATEFORME="${PLATEFORME:-compose}"

# shellcheck source=../versions.env
source "$XP_DIR/versions.env"

# Noms et ports locaux (protocole XP03 : collector.localhost, auth.collector.localhost).
export HOTE_FRONT="${HOTE_FRONT:-collector.localhost}"
export HOTE_AUTH="${HOTE_AUTH:-auth.collector.localhost}"
export PORT_HTTPS="${PORT_HTTPS:-443}"
export PORT_HTTP="${PORT_HTTP:-80}"
suffixe_port() { [[ "$PORT_HTTPS" == 443 ]] || printf ':%s' "$PORT_HTTPS"; }
export URL_FRONT="${URL_FRONT:-https://$HOTE_FRONT$(suffixe_port)}"
# k3d : pas de proxy, Keycloak atteint par port-forward depuis le poste (emetteur de debogage).
PORT_FORWARD_K3D="${PORT_FORWARD_K3D:-18080}"
if [[ "$PLATEFORME" == k3d ]]; then
  export URL_AUTH="${URL_AUTH:-http://localhost:$PORT_FORWARD_K3D}"
else
  export URL_AUTH="${URL_AUTH:-https://$HOTE_AUTH$(suffixe_port)}"
fi
ISSUER="$URL_AUTH/realms/collector"
CA_LOCALE="$GENERE_DIR/caddy-racine.crt"

# Sortie jetable de curl : le curl Windows n'ecrit pas dans /dev/null quand MSYS_NO_PATHCONV=1.
if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == MSYS* ]]; then POUBELLE=NUL; else POUBELLE=/dev/null; fi

journal() { printf '[%s] %s\n' "$(date -u +%H:%M:%S)" "$*" >&2; }
echec() { journal "ECHEC : $*"; exit 1; }

chemin_hote() {
  if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi
}

charger_env() {
  [[ -f "$ENV_FILE" ]] || echec "$ENV_FILE absent : lancer scripts/identifiants.sh"
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
}

csv_init() {
  mkdir -p "$MESURES_DIR" "$SORTIES_DIR"
  if [[ ! -f "$CSV" ]]; then
    printf 'horodatage_utc;xp;session;plateforme;critere;etape;valeur;unite;seuil;atteint;commentaire\n' >"$CSV"
  fi
}

csv() {
  # csv <critere> <etape> <valeur> <unite> <seuil> <atteint> <commentaire>
  csv_init
  local champs=() c
  for c in "$@"; do champs+=("${c//;/,}"); done
  printf '%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$XP_ID" "$SESSION" "$PLATEFORME" "${champs[@]:0:7}" >>"$CSV"
}

oui_non() { if [[ "$1" == "$2" ]]; then echo oui; else echo non; fi; }

horodatage() { date -u +%Y%m%dT%H%M%SZ; }
ms_maintenant() { date +%s%3N; }

charge_utile() {
  # Charge utile decodee d'un JWT (lu sur l'entree standard) ; la signature n'est jamais affichee.
  local p
  p="$(cut -d. -f2 | tr '_-' '/+')"
  while (( ${#p} % 4 )); do p="$p="; done
  printf '%s' "$p" | base64 -d 2>/dev/null
}

resume_jeton() {
  # MX5 : seuls aud, azp, scope, exp et les roles sont consignes, jamais le jeton.
  charge_utile | jq -c '{aud, azp, scope, typ, exp, roles: (.realm_access.roles // [])}'
}
