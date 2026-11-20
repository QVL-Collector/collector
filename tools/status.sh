#!/usr/bin/env bash
# État de l'environnement AWS (D11, US-004 AC5) : session et coût, couche persistante, ressources, nœuds et pods,
# dernier contrôle après déploiement.
#   ./status.sh            affiche tout, code retour 0
#   ./status.sh --strict   inventaire strict hors couche persistante et contrôle de la couche persistante (AC4, MI8) :
#                          code retour 1 si une ressource facturable subsiste ou si la couche persistante a dérivé

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

STRICT=0
case "${1:-}" in
  "") ;;
  --strict) STRICT=1 ;;
  *)
    sed -n '4,6p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
    exit 2
    ;;
esac

ECARTS=0

ligne() { printf '  %-28s %s\n' "$1" "$2" | masquer; }

ecart() {
  ligne "$1" "ÉCART : $2"
  ECARTS=$((ECARTS + 1))
}

afficher_session() {
  local reste ecoule cout cumul
  printf 'Session\n'
  if [ -z "${SESSION_TF:-}" ]; then
    ligne "session" "aucune (pas de fichier de session)"
    return 0
  fi
  ligne "session" "$SESSION_TF"
  [ "${EXTINCTION_INCOMPLETE:-0}" != "1" ] || ligne "extinction" "INCOMPLÈTE : relancer down.sh, up.sh refusé d'ici là"
  if [ -n "${FIN_LAB:-}" ]; then
    reste=$(((FIN_LAB - $(maintenant)) / 60))
    ligne "fin du lab" "$(heure_locale "$FIN_LAB"), reste $reste min ; down.sh au plus tard à $(heure_locale $((FIN_LAB - 30 * 60)))"
  fi
  if [ -n "${DEBUT_ALLUMAGE:-}" ]; then
    ecoule=$((($(maintenant) - DEBUT_ALLUMAGE + 59) / 60))
    cout="$(cout_estime "$ecoule")"
    cumul="$(awk -v c="$(cumul_couts)" -v s="$cout" 'BEGIN { printf "%.2f", c + s }')"
    ligne "allumé depuis" "$ecoule min"
    ligne "coût estimé de la session" "$cout \$ ($COUT_HORAIRE_USD \$/h)"
    ligne "cumul estimé du projet" "$cumul \$ sur $BUDGET_USD \$ (arrêt à $SEUIL_G5_PCT %)"
  fi
}

controler_persistant() {
  local blocage retention groupe attendu
  printf 'Couche persistante\n'
  if [ -n "$(zone_collector)" ]; then ligne "route53:zone" "présente"; else ecart "route53:zone" "absente (bootstrap.sh)"; fi

  if [ -n "$(certificat_emis)" ]; then ligne "acm:certificat" "ISSUED"; else ecart "acm:certificat" "non ISSUED"; fi

  if aws s3api head-bucket --bucket "$BUCKET_PHOTOS" >/dev/null 2>&1; then
    blocage="$(aws_json s3api get-public-access-block --bucket "$BUCKET_PHOTOS" 2>/dev/null |
      jq -r '.PublicAccessBlockConfiguration | [.BlockPublicAcls, .IgnorePublicAcls, .BlockPublicPolicy, .RestrictPublicBuckets] | all' ||
      printf 'false')"
    if [ "$blocage" = "true" ]; then
      ligne "s3:bucket-photos" "présent, accès public bloqué"
    else
      ecart "s3:bucket-photos" "blocage d'accès public absent ou partiel (MI8)"
    fi
  else
    ecart "s3:bucket-photos" "absent (bootstrap.sh)"
  fi

  for groupe in $GROUPES_JOURNAUX_PERSISTANTS; do
    attendu="$RETENTION_JOURNAL_APP"
    [ "$groupe" != "/collector/audit" ] || attendu="$RETENTION_JOURNAL_AUDIT"
    retention="$(aws_json logs describe-log-groups --log-group-name-prefix "$groupe" |
      jq -r --arg g "$groupe" '.logGroups[] | select(.logGroupName == $g) | (.retentionInDays // "infinie")')"
    if [ -z "$retention" ]; then
      ecart "logs:$groupe" "absent (bootstrap.sh)"
    elif [ "$retention" != "$attendu" ]; then
      ecart "logs:$groupe" "rétention $retention, attendue $attendu jours (D09 MO7)"
    else
      ligne "logs:$groupe" "rétention $retention jours"
    fi
  done
}

afficher_ephemere() {
  local ressources
  printf 'Couche éphémère (état Terraform)\n'
  if [ ! -f "$DOSSIER_TF/terraform.tfstate" ] || etat_vide; then
    ligne "état" "vide"
    return 0
  fi
  ressources="$(tf state list 2>/dev/null | grep -c . || true)"
  ligne "ressources suivies" "$ressources"
  ligne "url" "$(tf output -raw url 2>/dev/null || printf '?')"
  ligne "ALB ouvert à Internet" "$(tf output -raw alb_ouvert 2>/dev/null || printf '?')"
}

afficher_cluster() {
  printf 'Cluster\n'
  if ! command -v kubectl >/dev/null 2>&1 || ! kubectl config get-contexts "$NOM" >/dev/null 2>&1; then
    ligne "contexte $NOM" "absent"
    return 0
  fi
  if ! k get namespaces >/dev/null 2>&1; then
    ligne "API du cluster" "injoignable (cluster absent ou adresse du poste changée)"
    return 0
  fi
  k get nodes -o wide | masquer_ip | sed 's/^/  /'
  printf '\n'
  k get pods -A -o wide | masquer_ip | sed 's/^/  /'
  printf '\n'
  k get hpa -A 2>/dev/null | sed 's/^/  /' || true
}

afficher_controle() {
  local verdict="$DOSSIER_SESSION/controle-${SESSION_TF:-aucune}.json"
  printf 'Contrôle après déploiement\n'
  if [ ! -f "$verdict" ]; then
    ligne "verdict" "aucun pour cette session"
    return 0
  fi
  jq -r '"  verdict                      \(if .vert then "VERT" else "ROUGE" end) (\(.horodatage), \([.resultats[] | select(.statut != "ok")] | length) écart(s))"' "$verdict"
  jq -r '.resultats[] | select(.statut != "ok") | "    \(.statut) \(.groupe)/\(.controle) : \(.detail)"' "$verdict" | masquer_ip
}

exiger_outils aws terraform jq curl sed awk sha256sum
sonde_identite
charger_session

printf 'État du %s (région %s)\n\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$AWS_REGION"

if [ "$STRICT" -eq 1 ]; then
  controler_persistant
  printf '\n'
  STATUT=0
  "$RACINE_TOOLS/inventaire.sh" --strict || STATUT=1
  [ "$ECARTS" -eq 0 ] || STATUT=1
  [ "$STATUT" -eq 0 ] || printf 'status --strict : ÉCHEC (ressource facturable restante ou couche persistante en écart)\n'
  exit "$STATUT"
fi

afficher_session
printf '\n'
controler_persistant
printf '\n'
afficher_ephemere
printf '\n'
afficher_cluster
printf '\n'
afficher_controle
printf '\nInventaire du compte hors couche persistante\n'
"$RACINE_TOOLS/inventaire.sh" | sed '1d'
