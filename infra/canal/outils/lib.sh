# Bibliotheque commune des outils du canal (sourcee, jamais executee seule).
# shellcheck shell=bash
# shellcheck disable=SC2034 # variables communes, utilisees par les scripts qui sourcent ce fichier

set -euo pipefail
export MSYS_NO_PATHCONV=1
umask 077

CANAL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPOT_DIR="$(cd "$CANAL_DIR/../.." && pwd)"
# Identifiants de la session : ignores par git (.genere/) et par Helm (.helmignore), effaces par "down".
GENERE_DIR="$CANAL_DIR/.genere"
MDP_FILE="$GENERE_DIR/mots-de-passe.env"
UTILISATEURS_JSON="$GENERE_DIR/00-utilisateurs.json"
COOKIE_FILE="$GENERE_DIR/cookie-erlang"

VHOST="collector"
# Une identite par service (MC1) ; keda : supervision seule (MC4) ; admin-canal : console du poste.
SERVICES=(marche controle interactions)
UTILISATEURS=("${SERVICES[@]}" keda admin-canal)

# Image unique pour docker compose et Kubernetes : celle de values.yaml.
RABBITMQ_IMAGE="$(sed -n 's/^image: *//p' "$CANAL_DIR/values.yaml")"
[[ -n "$RABBITMQ_IMAGE" ]] || { printf 'image absente de values.yaml\n' >&2; exit 1; }
export RABBITMQ_IMAGE

# Kubernetes : contexte explicite et en liste blanche, jamais le contexte courant (le poste en porte
# d'autres). k3d-* pour le poste, *collector* pour le cluster du projet.
CONTEXTES_AUTORISES=('k3d-*' '*collector*')
# MSYS_NO_PATHCONV=1 empeche Git Bash de convertir KUBECONFIG : kubectl ignorerait un chemin /c/...
if [[ "${KUBECONFIG:-}" == /* ]] && command -v cygpath >/dev/null 2>&1; then
  KUBECONFIG="$(cygpath -m -p "$KUBECONFIG")"
  export KUBECONFIG
fi

journal() { printf '[%s] %s\n' "$(date -u +%H:%M:%S)" "$*" >&2; }
echec() { journal "ECHEC : $*"; exit 1; }

chemin_hote() {
  if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi
}

variable_mdp() {
  local u="${1^^}"
  printf 'MDP_%s' "${u//-/_}"
}

namespace_de() {
  case "$1" in
    marche) printf 'place-de-marche' ;;
    controle) printf 'controle' ;;
    interactions) printf 'interactions' ;;
    *) echec "identite sans namespace : $1" ;;
  esac
}

charger_mdp() {
  [[ -f "$MDP_FILE" ]] || echec "identifiants absents : lancer outils/identifiants.sh"
  set -a
  # shellcheck disable=SC1090
  source "$MDP_FILE"
  set +a
}

# A appeler avant toute commande kubectl ou helm : refus sans KUBE_CONTEXT ou hors liste blanche.
exiger_contexte() {
  local motif
  [[ -n "${KUBE_CONTEXT:-}" ]] ||
    echec "KUBE_CONTEXT absent : contexte explicite exige (${CONTEXTES_AUTORISES[*]}), le contexte courant n'est jamais utilise"
  for motif in "${CONTEXTES_AUTORISES[@]}"; do
    # shellcheck disable=SC2053 # motif glob voulu
    [[ "$KUBE_CONTEXT" == $motif ]] && return 0
  done
  echec "contexte '$KUBE_CONTEXT' refuse : motifs autorises ${CONTEXTES_AUTORISES[*]}"
}

kc() { kubectl --context "$KUBE_CONTEXT" "$@"; }
hm() { helm --kube-context "$KUBE_CONTEXT" "$@"; }
