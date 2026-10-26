#!/usr/bin/env bash
# Extinction du bac à sable XP01 : releases Helm, bucket, destroy avec reprise, planifications, inventaire strict.
#   ./down.sh

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

[ "$#" -eq 0 ] || {
  sed -n '2,3p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

exiger_outils aws terraform jq curl sed sha256sum
sonde_identite
charger_session
tf init -input=false >/dev/null

T0="$(maintenant)"
journal "G3 : lancement de l'extinction"

if ! etat_vide; then
  ORCHESTRATEUR="$(tf output -raw orchestrateur 2>/dev/null || printf '%s' "$ORCHESTRATEUR")"
fi
LIBELLE="${LIBELLE:-${ORCHESTRATEUR^^}-$(date +%Y%m%d)}"

retirer_releases() {
  command -v kubectl >/dev/null 2>&1 || return 0
  aws eks describe-cluster --name "$NOM" >/dev/null 2>&1 || return 0
  aws eks update-kubeconfig --name "$NOM" --alias "$NOM" --profile "$AWS_PROFILE" >/dev/null 2>&1 || return 0
  if ! kubectl --context "$NOM" --request-timeout=20s get namespaces >/dev/null 2>&1; then
    journal "ATTENTION : API du cluster injoignable (adresse du poste changée ?) ; releases non retirées, l'inventaire tranchera"
    return 0
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

vider_bucket() {
  aws s3api head-bucket --bucket "$BUCKET" >/dev/null 2>&1 || return 0
  journal "vidage et suppression du bucket de test"
  aws s3 rm "s3://$BUCKET" --recursive --only-show-errors
  aws s3api delete-bucket --bucket "$BUCKET"
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

[ "$ORCHESTRATEUR" != "eks" ] || retirer_releases
vider_bucket

STATUT_DESTROY=0
if etat_vide; then
  journal "état vide : rien à détruire par Terraform"
else
  detruire || STATUT_DESTROY=1
fi
mesure G3 destroy "$(($(maintenant) - T0))" s "" "" "fin du destroy (statut $STATUT_DESTROY)"
sauvegarder_etat down

aws scheduler delete-schedule-group --name "$GROUPE_PLANIF" >/dev/null 2>&1 || true
kubectl config delete-context "$NOM" >/dev/null 2>&1 || true

journal "inventaire strict (MX7)"
STATUT_INVENTAIRE=0
"$RACINE_XP/inventaire.sh" --strict || STATUT_INVENTAIRE=1

DUREE_MIN=$((($(maintenant) - T0 + 59) / 60))
ATTEINT="non"
if [ "$STATUT_DESTROY" -eq 0 ] && [ "$STATUT_INVENTAIRE" -eq 0 ] && [ "$DUREE_MIN" -le 20 ]; then
  ATTEINT="oui"
fi
mesure G3 inventaire-vide "$DUREE_MIN" min 20 "$ATTEINT" "destroy $STATUT_DESTROY, inventaire $STATUT_INVENTAIRE"
journal "G3 : $DUREE_MIN min (seuil 20), destroy $STATUT_DESTROY, inventaire $STATUT_INVENTAIRE"

[ "$STATUT_DESTROY" -eq 0 ] || echec "destroy en échec : relancer down.sh ; session suivante bloquée tant que l'inventaire n'est pas vide"
[ "$STATUT_INVENTAIRE" -eq 0 ] || echec "inventaire non vide : nettoyer les ressources listées avant toute autre session"
rm -f "$FICHIER_SESSION"
journal "extinction terminée, inventaire vide hors couche persistante"
