#!/usr/bin/env bash
# Extinction de la couche éphémère (D11) : Flux, releases Helm, photos, secrets de session, destroy avec reprise,
# planifications (G1), inventaire strict (G2), ligne de coût (G5), rappel du poste annulé (G7). À terminer AVANT « End Lab » (XP00 S8b).
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
SESSION_TF="${SESSION_TF:-$(nouvelle_session)}"
export EXTINCTION_INCOMPLETE=1
ecrire_session

retirer_flux() {
  local manifeste="$DOSSIER_SESSION/k8s-$NOM/flux-install.yaml"
  k get crd helmreleases.helm.toolkit.fluxcd.io >/dev/null 2>&1 || return 0
  journal "retrait de Flux en premier (sinon il recrée les releases)"
  if ! k -n "$NS_FLUX" delete helmrelease --all --timeout=5m >/dev/null; then
    journal "ATTENTION : HelmRelease non désinstallées par Flux ; finaliseurs levés, les releases sont retirées par Helm"
    k -n "$NS_FLUX" get helmrelease -o name | while read -r hr; do
      k -n "$NS_FLUX" patch "$hr" --type=merge -p '{"metadata":{"finalizers":null}}' >/dev/null || true
    done
  fi
  k -n "$NS_FLUX" delete gitrepository,ocirepository,helmchart --all --timeout=2m >/dev/null 2>&1 || true
  [ -f "$manifeste" ] || { mkdir -p "$(dirname "$manifeste")" && manifestes_flux >"$manifeste"; } ||
    { journal "ATTENTION : manifeste Flux indisponible, contrôleurs non retirés ; l'inventaire tranchera"; return 0; }
  k delete -f "$manifeste" --ignore-not-found --timeout=3m >/dev/null || journal "ATTENTION : Flux non retiré, l'inventaire tranchera"
}

retirer_charges_cluster() {
  command -v kubectl >/dev/null 2>&1 || return 0
  aws eks describe-cluster --name "$NOM" >/dev/null 2>&1 || return 0
  aws eks update-kubeconfig --name "$NOM" --alias "$NOM" --profile "$AWS_PROFILE" >/dev/null 2>&1 || return 0
  if ! kubectl --context "$NOM" --request-timeout=20s get namespaces >/dev/null 2>&1; then
    journal "ATTENTION : API du cluster injoignable (adresse du poste changée ?) ; releases non retirées, l'inventaire tranchera"
    return 0
  fi
  retirer_flux
  if command -v helm >/dev/null 2>&1; then
    { helm --kube-context "$NOM" list -A -o json || printf '[]'; } | tr -d '\r' | jq -r '.[] | "\(.namespace) \(.name)"' |
      while read -r espace release; do
        journal "retrait de la release $release ($espace)"
        helm --kube-context "$NOM" uninstall "$release" -n "$espace" --wait --timeout 5m >/dev/null ||
          journal "ATTENTION : release $release ($espace) non retirée"
      done || journal "ATTENTION : releases Helm non listées"
  fi
  local lb pvc
  if ! lb="$(k get svc -A -o json | jq '[.items[] | select(.spec.type == "LoadBalancer")] | length')" ||
    ! pvc="$(k get pvc -A -o json | jq '.items | length')"; then
    journal "ATTENTION : Services ou volumes non lus ; l'inventaire tranchera"
    return 0
  fi
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
  variables_tf "$(ip_poste 2>/dev/null || printf '192.0.2.1/32')"
  for essai in 1 2 3; do
    journal "terraform destroy, essai $essai"
    if tf destroy -auto-approve -input=false "${ALB_FERME[@]}" | masquer_ip; then
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
  cout="$(ajouter_cout "$duree_min")"
  DEBUT_ALLUMAGE="$(maintenant)"
  ecrire_session
  cumul="$(cumul_couts)"
  journal "G5 : session $duree_min min ≈ $cout \$ ; cumul estimé du projet $cumul \$ sur $BUDGET_USD \$ (arrêt à $SEUIL_G5_PCT %)"
}

retirer_charges_cluster
vider_bucket_photos
supprimer_secrets_session
supprimer_sonde_externe

STATUT_DESTROY=0
if etat_vide; then
  journal "état vide : rien à détruire par Terraform"
else
  detruire || STATUT_DESTROY=1
fi
phase down destroy "$(($(maintenant) - T0))" "fin du destroy (statut $STATUT_DESTROY)"
sauvegarder_etat "$DOSSIER_TF" ephemere down

journal "G1 : suppression des planifications restantes (AC4)"
supprimer_planifications || journal "ATTENTION : planifications non supprimées, l'inventaire tranchera"
kubectl config delete-context "$NOM" >/dev/null 2>&1 || true

journal "G2 : inventaire strict (status.sh --strict, AC4)"
STATUT_INVENTAIRE=0
"$RACINE_TOOLS/status.sh" --strict || STATUT_INVENTAIRE=1

DUREE_MIN=$((($(maintenant) - T0 + 59) / 60))
phase down inventaire "$(($(maintenant) - T0))" "X5 : $DUREE_MIN min (cible 20), destroy $STATUT_DESTROY, inventaire $STATUT_INVENTAIRE"
journal "X5 : $DUREE_MIN min (cible 20), destroy $STATUT_DESTROY, inventaire $STATUT_INVENTAIRE"
noter_cout

[ "$STATUT_DESTROY" -eq 0 ] || echec "destroy en échec : relancer down.sh avant « End Lab » ; up.sh refusé tant qu'un down.sh n'a pas abouti"
[ "$STATUT_INVENTAIRE" -eq 0 ] || echec "inventaire non vide : nettoyer les ressources listées puis relancer down.sh avant « End Lab » ; up.sh refusé d'ici là"
rm -f "$FICHIER_SESSION"
rm -rf "$DEPOT/infra/canal/.genere"
annuler_rappel_poste
journal "extinction terminée, inventaire vide hors couche persistante : « End Lab » possible"
