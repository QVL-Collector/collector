#!/usr/bin/env bash
# Bibliothèque commune de up.sh, down.sh et inventaire.sh (D11). Sourcée, jamais exécutée seule.
# Aucun secret n'est lu, affiché ni journalisé ; les numéros de compte sont masqués à l'affichage (MI6).
# shellcheck disable=SC2034

set -euo pipefail

export MSYS_NO_PATHCONV=1
export AWS_PROFILE="${AWS_PROFILE:-collector-lab}"
export AWS_REGION="us-east-1"
export AWS_DEFAULT_REGION="$AWS_REGION"
export AWS_PAGER=""
export TF_IN_AUTOMATION=1

RACINE_TOOLS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPOT="$(cd "$RACINE_TOOLS/.." && pwd)"
DOSSIER_TF="$DEPOT/infra/ephemere"
DOSSIER_PERSISTANT="$DEPOT/infra/persistant"
DOSSIER_SESSION="$RACINE_TOOLS/.session"
DOSSIER_SAUVEGARDES="$DEPOT/infra/.etat-sauvegardes"
FICHIER_SESSION="$DOSSIER_SESSION/session.env"
CSV_SESSIONS="$DEPOT/docs/mesures/sessions.csv"
CSV_COUTS="$DEPOT/docs/mesures/couts.csv"

NOM="collector"
DOMAINE="collector.qvl-project.com"
GROUPE_PLANIF="collector"
PREFIXE_SECRETS="collector/"
GROUPES_JOURNAUX_PERSISTANTS="/collector/app /collector/audit"
VERSION_EKS="${VERSION_EKS:-1.35}"
DUREE_LAB_MIN=240
MARGE_UP_MIN=75
BUDGET_USD=50
SEUIL_G5_PCT=70

ORCHESTRATEUR="${ORCHESTRATEUR:-}"

masquer() { sed -E 's/[0-9]{12}/<compte>/g'; }

journal() { printf '[%s] %s\n' "$(date -u +%H:%M:%SZ)" "$*" | masquer >&2; }

echec() {
  journal "ERREUR : $*"
  exit 1
}

exiger_outils() {
  local outil
  for outil in "$@"; do
    command -v "$outil" >/dev/null 2>&1 || echec "outil absent du PATH : $outil (rouvrir le terminal après installation)"
  done
}

aws_json() { aws "$@" --output json | tr -d '\r'; }

tf() { terraform -chdir="$DOSSIER_TF" "$@"; }

maintenant() { date -u +%s; }

utc_scheduler() { date -u -d "@$1" +%Y-%m-%dT%H:%M:%S; }

heure_locale() { date -d "@$1" +%H:%M; }

cout_horaire() {
  case "$1" in
    eks) printf '0.34' ;;
    ecs) printf '0.14' ;;
    *) printf '0.34' ;;
  esac
}

sonde_identite() {
  local identite arn region
  identite="$(aws_json sts get-caller-identity)" ||
    echec "identité AWS indisponible : profil $AWS_PROFILE à jour après « Start Lab » ?"
  arn="$(printf '%s' "$identite" | jq -r '.Arn')"
  case "$arn" in
    *":assumed-role/voclabs/"*) ;;
    *) echec "principal inattendu : $(printf '%s' "$arn" | masquer)" ;;
  esac
  COMPTE="$(printf '%s' "$identite" | jq -r '.Account')"
  region="$(aws_json ec2 describe-availability-zones | jq -r '.AvailabilityZones[0].RegionName')"
  [ "$region" = "us-east-1" ] || echec "région inattendue : $region"
  LAB_ROLE_ARN="arn:aws:iam::${COMPTE}:role/LabRole"
  BUCKET_PHOTOS="${NOM}-photos-$(printf '%s' "$COMPTE" | sha256sum | cut -c1-10)"
  journal "identité : rôle voclabs, compte masqué, région $region, profil $AWS_PROFILE"
}

ip_poste() {
  local ip
  ip="$(curl -fsS --max-time 10 https://checkip.amazonaws.com | tr -d '\r\n ')" || echec "adresse publique du poste introuvable"
  [[ "$ip" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] || echec "adresse publique du poste illisible"
  printf '%s/32' "$ip"
}

charger_session() {
  if [ -f "$FICHIER_SESSION" ]; then
    # shellcheck source=/dev/null
    . "$FICHIER_SESSION"
  fi
}

ecrire_session() {
  mkdir -p "$DOSSIER_SESSION"
  {
    printf 'ORCHESTRATEUR=%q\n' "$ORCHESTRATEUR"
    printf 'SESSION_TF=%q\n' "$SESSION_TF"
    printf 'DEBUT_LAB=%q\n' "$DEBUT_LAB"
    printf 'FIN_LAB=%q\n' "$FIN_LAB"
    printf 'DEBUT_ALLUMAGE=%q\n' "$DEBUT_ALLUMAGE"
  } >"$FICHIER_SESSION"
}

etat_vide() { [ -z "$(tf state list 2>/dev/null || true)" ]; }

sauvegarder_etat() {
  local source="$DOSSIER_TF/terraform.tfstate"
  [ -f "$source" ] || return 0
  mkdir -p "$DOSSIER_SAUVEGARDES"
  cp "$source" "$DOSSIER_SAUVEGARDES/ephemere.tfstate.$(date -u +%Y%m%dT%H%M%SZ).$1"
}

phase() {
  [ "$#" -eq 4 ] || echec "phase : 4 arguments attendus (script etape duree_s commentaire)"
  mkdir -p "$(dirname "$CSV_SESSIONS")"
  [ -f "$CSV_SESSIONS" ] ||
    printf '%s\n' 'horodatage_utc;session;orchestrateur;script;etape;duree_s;commentaire' >"$CSV_SESSIONS"
  printf '%s;%s;%s;%s;%s;%s;%s\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "${SESSION_TF:-}" "${ORCHESTRATEUR:-}" "$@" >>"$CSV_SESSIONS"
}

cumul_couts() {
  [ -f "$CSV_COUTS" ] || {
    printf '0'
    return 0
  }
  awk -F';' 'NR > 1 { s += $4 } END { printf "%.2f", s + 0 }' "$CSV_COUTS"
}

variables_tf() {
  export TF_VAR_orchestrateur="$ORCHESTRATEUR"
  export TF_VAR_session="$SESSION_TF"
  export TF_VAR_ip_admin="$1"
}
