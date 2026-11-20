#!/usr/bin/env bash
# Ouverture et fermeture de l'ALB à Internet (ME5, US-004 AC10).
#   ./alb.sh ouvrir   refusé sans verdict VERT du contrôle après déploiement pour cette session et cet état ;
#                     rejoue le contrôle une fois ouvert et referme s'il rougit
#   ./alb.sh fermer   ramène l'ALB à l'adresse du poste
# ME4 (X-Forwarded-For) et ME8 (traversées vers Keycloak) restent à prouver avant la démonstration (D08).

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

usage() {
  sed -n '3,5p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

[ "$#" -eq 1 ] || usage
ACTION="$1"
case "$ACTION" in
  ouvrir | fermer) ;;
  *) usage ;;
esac

appliquer() {
  local ouvert="$1" preuve="$2"
  variables_tf "$(ip_poste)"
  tf apply -auto-approve -input=false -var="alb_ouvert=$ouvert" -var="preuve_controle=$preuve" | masquer_ip
  sauvegarder_etat "$DOSSIER_TF" ephemere "alb-$ACTION"
}

exiger_outils aws terraform jq curl sed awk sha256sum
sonde_identite
charger_session
[ -n "${SESSION_TF:-}" ] || echec "aucune session en cours"
etat_vide && echec "couche éphémère absente"

if [ "$ACTION" = "fermer" ]; then
  appliquer false ""
  phase alb fermer 0 "ALB restreint au poste"
  journal "ALB restreint à l'adresse du poste (ME5)"
  exit 0
fi

VERDICT="$DOSSIER_SESSION/controle-$SESSION_TF.json"
[ -f "$VERDICT" ] || echec "ouverture refusée : aucun contrôle après déploiement pour la session $SESSION_TF (AC10)"
[ "$(jq -r '.vert' "$VERDICT")" = "true" ] ||
  echec "ouverture refusée : dernier contrôle ROUGE (docs/mesures/US-004/controle-$SESSION_TF.txt) (AC10)"
[ "$(jq -r '.serie_etat' "$VERDICT")" = "$(serie_etat)" ] ||
  echec "ouverture refusée : l'état a changé depuis le contrôle ; relancer tools/controle-apres-deploiement.sh (AC10)"

journal "contrôle VERT pour la session $SESSION_TF et l'état courant : ouverture de l'ALB à Internet"
appliquer true "$SESSION_TF"
phase alb ouvrir 0 "ALB ouvert à Internet après contrôle vert"

if ! "$RACINE_TOOLS/controle-apres-deploiement.sh"; then
  journal "contrôle ROUGE une fois l'ALB ouvert : fermeture immédiate"
  appliquer false ""
  echec "ALB refermé : corriger les écarts du contrôle avant toute nouvelle ouverture"
fi
journal "ALB ouvert et contrôle VERT ; tools/alb.sh fermer après la démonstration (un up.sh le referme aussi)"
