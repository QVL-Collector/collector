#!/usr/bin/env bash
# Extinction de la couche éphémère (D11) : Flux, releases Helm, photos, secrets de session, destroy avec reprise,
# planifications, inventaire strict (G2), ligne de coût (G5). À terminer AVANT « End Lab » (XP00 S8b).
#   ./down.sh

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

[ "$#" -eq 0 ] || {
  sed -n '4p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

exiger_outils aws terraform jq curl sed awk sha256sum
sonde_identite
charger_session
tf init -input=false >/dev/null

T0="$(maintenant)"
journal "lancement de l'extinction"

if ! etat_vide; then
  ORCHESTRATEUR="$(tf output -raw orchestrateur 2>/dev/null || printf '%s' "$ORCHESTRATEUR")"
fi

retirer_charges_cluster() {
  command -v kubectl >/dev/null 2>&1 || return 0
  aws eks describe-cluster --name "$NOM" >/dev/null 2>&1 || return 0
  aws eks update-kubeconfig --name "$NOM" --alias "$NOM" --profile "$AWS_PROFILE" >/dev/null 2>&1 || return 0
  if ! kubectl --context "$NOM" --request-timeout=20s get namespaces >/dev/null 2>&1; then
    journal "ATTENTION : API du cluster injoignable (adresse du poste changée ?) ; releases non retirées, l'inventaire tranchera"
    return 0
  fi
  if command -v flux >/dev/null 2>&1 && kubectl --context "$NOM" get namespace flux-system >/dev/null 2>&1; then
    journal "retrait de Flux en premier (sinon il recrée les releases)"
    flux --context "$NOM" uninstall --silent >/dev/null
  fi
  if command -v helm >/dev/null 2>&1; then
    helm --kube-context "$NOM" list -A -o json | tr -d '\r' | jq -r '.[] | "\(.namespace) \(.name)"' |
      while read -r espace release; do
        journal "retrait de la release $release ($espace)"
        helm --kube-context "$NOM" uninstall "$release" -n "$espace" --wait --timeout 5m >/dev/null
      done
  fi
  local lb pvc
  lb="$(kubectl --context "$NOM" get svc -A -o json | tr -d '\r' | jq '[.items[] | select(.spec.type == "LoadBalancer")] | length')"
  pvc="$(kubectl --context "$NOM" get pvc -A -o json | tr -d '\r' | jq '.items | length')"
  [ "$lb" -eq 0 ] || echec "$lb Service LoadBalancer restant : le supprimer avant down (load balancer hors IaC)"
  [ "$pvc" -eq 0 ] || echec "$pvc volume persistant restant : le supprimer avant down (EBS hors IaC)"
}

vider_bucket_photos() {
  aws s3api head-bucket --bucket "$BUCKET_PHOTOS" >/dev/null 2>&1 || return 0
  journal "vidage du bucket des photos (le bucket reste, couche persistante)"
  aws s3 rm "s3://$BUCKET_PHOTOS" --recursive --only-show-errors
}

supprimer_secrets_session() {
  aws_json secretsmanager list-secrets |
    jq -r --arg p "$PREFIXE_SECRETS" '.SecretList[] | select(.Name | startswith($p)) | .Name' |
    while read -r secret; do
      journal "suppression sans délai de récupération du secret de session $secret"
      aws secretsmanager delete-secret --secret-id "$secret" --force-delete-without-recovery >/dev/null
    done
}

supprimer_eni_disponibles() {
  local vpc
  vpc="$(tf output -raw id_vpc 2>/dev/null || true)"
  [ -n "$vpc" ] || return 0
  aws_json ec2 describe-network-interfaces --filters "Name=vpc-id,Values=$vpc" "Name=status,Values=available" |
    jq -r '.NetworkInterfaces[].NetworkInterfaceId' |
    while read -r eni; do
      journal "suppression de l'interface disponible $eni"
      aws ec2 delete-network-interface --network-interface-id "$eni" || true
    done
}

detruire() {
  local essai
  SESSION_TF="${SESSION_TF:-$(date -u +%Y%m%dT%H%MZ)}"
  variables_tf "$(ip_poste 2>/dev/null || printf '192.0.2.1/32')"
  for essai in 1 2 3; do
    journal "terraform destroy, essai $essai"
    if tf destroy -auto-approve -input=false | masquer; then
      return 0
    fi
    [ "$essai" -lt 3 ] || return 1
    supprimer_eni_disponibles
    sleep 60
  done
}

noter_cout() {
  local debut="${DEBUT_ALLUMAGE:-}" duree_min cout cumul
  [ -n "$debut" ] || {
    journal "G5 : heure d'allumage inconnue, aucune ligne de coût ajoutée"
    return 0
  }
  duree_min=$((($(maintenant) - debut + 59) / 60))
  cout="$(awk -v m="$duree_min" -v h="$(cout_horaire "$ORCHESTRATEUR")" 'BEGIN { printf "%.2f", m / 60 * h }')"
  mkdir -p "$(dirname "$CSV_COUTS")"
  [ -f "$CSV_COUTS" ] || printf '%s\n' 'date;orchestrateur;duree_allumee_min;cout_estime_usd' >"$CSV_COUTS"
  printf '%s;%s;%s;%s\n' "$(date -u +%Y-%m-%d)" "${ORCHESTRATEUR:-?}" "$duree_min" "$cout" >>"$CSV_COUTS"
  DEBUT_ALLUMAGE="$(maintenant)"
  [ ! -f "$FICHIER_SESSION" ] || ecrire_session
  cumul="$(cumul_couts)"
  journal "G5 : session $duree_min min ≈ $cout \$ ; cumul estimé du projet $cumul \$ sur $BUDGET_USD \$ (arrêt à $SEUIL_G5_PCT %)"
}

[ "$ORCHESTRATEUR" != "eks" ] || retirer_charges_cluster
vider_bucket_photos
supprimer_secrets_session

STATUT_DESTROY=0
if etat_vide; then
  journal "état vide : rien à détruire par Terraform"
else
  detruire || STATUT_DESTROY=1
fi
phase down destroy "$(($(maintenant) - T0))" "fin du destroy (statut $STATUT_DESTROY)"
sauvegarder_etat down

aws scheduler delete-schedule-group --name "$GROUPE_PLANIF" >/dev/null 2>&1 || true
kubectl config delete-context "$NOM" >/dev/null 2>&1 || true

journal "G2 : inventaire strict"
STATUT_INVENTAIRE=0
"$RACINE_TOOLS/inventaire.sh" --strict || STATUT_INVENTAIRE=1

DUREE_MIN=$((($(maintenant) - T0 + 59) / 60))
phase down inventaire "$(($(maintenant) - T0))" "X5 : $DUREE_MIN min (cible 20), destroy $STATUT_DESTROY, inventaire $STATUT_INVENTAIRE"
journal "X5 : $DUREE_MIN min (cible 20), destroy $STATUT_DESTROY, inventaire $STATUT_INVENTAIRE"
noter_cout

[ "$STATUT_DESTROY" -eq 0 ] || echec "destroy en échec : relancer down.sh avant « End Lab » ; session suivante bloquée tant que l'inventaire n'est pas vide"
[ "$STATUT_INVENTAIRE" -eq 0 ] || echec "inventaire non vide : nettoyer les ressources listées avant « End Lab »"
rm -f "$FICHIER_SESSION"
journal "extinction terminée, inventaire vide hors couche persistante : « End Lab » possible"
