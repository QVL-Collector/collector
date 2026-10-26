# Bibliotheque commune des scripts XP02 (sourcee, jamais executee seule).
# shellcheck shell=bash

set -euo pipefail
export MSYS_NO_PATHCONV=1

XP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPOT_DIR="$(cd "$XP_DIR/../.." && pwd)"
GENERE_DIR="$XP_DIR/.genere"
ENV_FILE="$XP_DIR/.env"
COMPOSE_PROJET="xp02"
XP_ID="XP02"

# Sorties de mesure : docs/mesures par defaut, surchargeable pour un essai a blanc.
MESURES_DIR="${MESURES_DIR:-$DEPOT_DIR/docs/mesures}"
CSV="$MESURES_DIR/XP02-canal-rabbitmq.csv"
SORTIES_DIR="$MESURES_DIR/XP02-canal-rabbitmq"
SESSION="${SESSION:-$(date -u +%Y%m%d)}"
PLATEFORME="${PLATEFORME:-compose}"

# shellcheck source=../versions.env
source "$XP_DIR/versions.env"

SERVICES=(svc-pdm svc-controle svc-interactions svc-test)
UTILISATEURS=("${SERVICES[@]}" keda-monitoring admin-canal)

# Sortie jetable de curl : le curl Windows n'ecrit pas dans /dev/null quand MSYS_NO_PATHCONV=1.
if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == MSYS* ]]; then POUBELLE=NUL; else POUBELLE=/dev/null; fi

journal() { printf '[%s] %s\n' "$(date -u +%H:%M:%S)" "$*" >&2; }
echec() { journal "ECHEC : $*"; exit 1; }

chemin_hote() {
  # Chemin comprehensible par Docker Desktop (C:/...) depuis Git Bash.
  if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi
}

variable_mdp() {
  # svc-pdm -> MDP_SVC_PDM
  local u="${1^^}"
  printf 'MDP_%s' "${u//-/_}"
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

horodatage() { date -u +%Y%m%dT%H%M%SZ; }

NS_BROKER="collector-canal"
POD_BROKER="rabbitmq-0"
CONTENEUR_COMPOSE="xp02-rabbitmq-1"

rmq() {
  # Commande rabbitmqctl / rabbitmq-diagnostics dans le broker de la plateforme courante.
  case "$PLATEFORME" in
    compose) docker exec -i --user rabbitmq "$CONTENEUR_COMPOSE" "$@" ;;
    k3d|eks) kubectl exec -i -n "$NS_BROKER" "$POD_BROKER" -c rabbitmq -- "$@" ;;
    *) echec "PLATEFORME inconnue : $PLATEFORME" ;;
  esac
}

broker_pret() { rmq rabbitmq-diagnostics -q check_running >/dev/null 2>&1; }

ms_maintenant() { date +%s%3N; }
