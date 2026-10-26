#!/usr/bin/env bash
# Bibliothèque commune de up.sh, down.sh et inventaire.sh (XP01). Sourcée, jamais exécutée seule.
# Aucun secret n'est lu, affiché ni journalisé ; les numéros de compte sont masqués à l'affichage.
# shellcheck disable=SC2034

set -euo pipefail

export MSYS_NO_PATHCONV=1
export AWS_PROFILE="${AWS_PROFILE:-collector-lab}"
export AWS_REGION="us-east-1"
export AWS_DEFAULT_REGION="$AWS_REGION"
export AWS_PAGER=""
export TF_IN_AUTOMATION=1

RACINE_XP="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOSSIER_TF="$RACINE_XP/ephemere"
DOSSIER_SESSION="$RACINE_XP/.session"
DOSSIER_SAUVEGARDES="$RACINE_XP/.etat-sauvegardes"
FICHIER_SESSION="$DOSSIER_SESSION/session.env"
DEPOT="$(cd "$RACINE_XP/../.." && pwd)"
CSV_MESURES="$DEPOT/docs/mesures/XP01-orchestrateur.csv"

NOM="collector-xp01"
DOMAINE="collector.qvl-project.com"
GROUPE_PLANIF="collector-xp01"
VERSION_EKS="${VERSION_EKS:-1.35}"
DUREE_LAB_MIN=240
MARGE_UP_MIN=75

LIBELLE="${LIBELLE:-}"
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

sonde_identite() {
  local arn region
  arn="$(aws_json sts get-caller-identity | jq -r '.Arn')" ||
    echec "identité AWS indisponible : profil $AWS_PROFILE à jour après « Start Lab » ?"
  case "$arn" in
    *":assumed-role/voclabs/"*) ;;
    *) echec "principal inattendu : $(printf '%s' "$arn" | masquer)" ;;
  esac
  COMPTE="$(aws_json sts get-caller-identity | jq -r '.Account')"
  region="$(aws_json ec2 describe-availability-zones | jq -r '.AvailabilityZones[0].RegionName')"
  [ "$region" = "us-east-1" ] || echec "région inattendue : $region"
  LAB_ROLE_ARN="arn:aws:iam::${COMPTE}:role/LabRole"
  BUCKET="${NOM}-$(printf '%s' "$COMPTE" | sha256sum | cut -c1-10)"
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
    printf 'LIBELLE=%q\n' "$LIBELLE"
    printf 'SESSION_TF=%q\n' "$SESSION_TF"
    printf 'DEBUT_LAB=%q\n' "$DEBUT_LAB"
    printf 'FIN_LAB=%q\n' "$FIN_LAB"
  } >"$FICHIER_SESSION"
}

etat_vide() { [ -z "$(tf state list 2>/dev/null || true)" ]; }

sauvegarder_etat() {
  local source="$DOSSIER_TF/terraform.tfstate"
  [ -f "$source" ] || return 0
  mkdir -p "$DOSSIER_SAUVEGARDES"
  cp "$source" "$DOSSIER_SAUVEGARDES/terraform.tfstate.$(date -u +%Y%m%dT%H%M%SZ).$1"
}

mesure() {
  [ "$#" -eq 7 ] || echec "mesure : 7 arguments attendus (critere etape valeur unite seuil atteint commentaire)"
  mkdir -p "$(dirname "$CSV_MESURES")"
  [ -f "$CSV_MESURES" ] ||
    printf '%s\n' 'horodatage_utc;xp;session;plateforme;critere;etape;valeur;unite;seuil;atteint;commentaire' >"$CSV_MESURES"
  printf '%s;XP01;%s;%s;%s;%s;%s;%s;%s;%s;%s\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$LIBELLE" "$ORCHESTRATEUR" "$@" >>"$CSV_MESURES"
}

variables_tf() {
  export TF_VAR_orchestrateur="$ORCHESTRATEUR"
  export TF_VAR_session="$SESSION_TF"
  export TF_VAR_ip_admin="$1"
}
