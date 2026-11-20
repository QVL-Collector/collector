#!/usr/bin/env bash
# Couche persistante (D11, US-004 AC1), lancée une fois et rejouable sans effet : bucket des photos créé en CLI
# (SCP du lab, D07), puis infra/persistant (zone, CAA, certificat et validation, budget, configuration du bucket,
# groupes de journaux applicatif et d'audit). Refuse tout plan qui détruit ou remplace une ressource.
#   ./bootstrap.sh

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

[ "$#" -eq 0 ] || {
  sed -n '5p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

creer_bucket_photos() {
  if aws s3api head-bucket --bucket "$BUCKET_PHOTOS" >/dev/null 2>&1; then
    journal "bucket des photos présent"
    return 0
  fi
  journal "création du bucket des photos en CLI (configuration par Terraform)"
  aws s3api create-bucket --bucket "$BUCKET_PHOTOS" >/dev/null
  aws s3api wait bucket-exists --bucket "$BUCKET_PHOTOS"
  aws s3api put-public-access-block --bucket "$BUCKET_PHOTOS" \
    --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
  aws s3api put-bucket-tagging --bucket "$BUCKET_PHOTOS" \
    --tagging 'TagSet=[{Key=projet,Value=collector},{Key=couche,Value=persistant},{Key=outil,Value=bootstrap}]'
}

appliquer_persistant() {
  local plan="$DOSSIER_SESSION/persistant.tfplan" statut destructions
  mkdir -p "$DOSSIER_SESSION"
  tfp init -input=false >/dev/null

  set +e
  tfp plan -input=false -detailed-exitcode -out="$plan" | masquer
  statut="${PIPESTATUS[0]}"
  set -e

  case "$statut" in
    0)
      journal "couche persistante à jour : aucun changement (second lancement sans effet)"
      rm -f "$plan"
      return 0
      ;;
    2) ;;
    *) echec "plan de la couche persistante en échec" ;;
  esac

  destructions="$(tfp show -json "$plan" |
    jq -r '[.resource_changes[] | select(.change.actions | index("delete")) | .address] | join(" ")')"
  if [ -n "$destructions" ]; then
    rm -f "$plan"
    echec "plan refusé : destruction ou remplacement de $destructions (corriger la configuration ou persistant.auto.tfvars)"
  fi

  sauvegarder_etat "$DOSSIER_PERSISTANT" persistant avant-bootstrap
  journal "application de la couche persistante (imports, créations, mises à jour, aucune destruction)"
  tfp apply -input=false "$plan" | masquer
  rm -f "$plan"
  sauvegarder_etat "$DOSSIER_PERSISTANT" persistant bootstrap
}

verifier_delegation() {
  local attendus publics
  attendus="$(tfp output -json serveurs_noms | jq -r '.[]' | sed 's/\.$//' | LC_ALL=C sort | tr '\n' ' ' | sed 's/ $//')"
  publics="$(curl -fsS --max-time 10 "https://dns.google/resolve?name=$DOMAINE&type=NS" |
    jq -r '[.Answer[]? | select(.type == 2) | .data | sub("\\.$"; "")] | sort | join(" ")' || true)"
  journal "serveurs de noms de la zone : $attendus"
  if [ "$publics" = "$attendus" ]; then
    journal "délégation Cloudflare vers Route 53 constatée"
  else
    journal "ATTENTION : délégation non constatée ; poser ces NS dans Cloudflare pour $DOMAINE (D08 ME10)"
  fi
}

verifier_certificat() {
  [ -n "$(certificat_emis)" ] || echec "certificat $DOMAINE non ISSUED : vérifier la délégation et le CNAME de validation"
  journal "certificat $DOMAINE et *.$DOMAINE ISSUED"
}

exiger_outils aws terraform jq curl sed awk sha256sum
[ -f "$DOSSIER_PERSISTANT/persistant.auto.tfvars" ] ||
  echec "infra/persistant/persistant.auto.tfvars absent : copier le modèle .example et le renseigner (infra/README.md)"
sonde_identite

T0="$(maintenant)"
creer_bucket_photos
appliquer_persistant
verifier_delegation
verifier_certificat

[ "$(tfp output -raw bucket_photos)" = "$BUCKET_PHOTOS" ] ||
  echec "nom du bucket des photos divergent entre tools/lib.sh et infra/persistant"
"$RACINE_TOOLS/status.sh" --strict >/dev/null 2>&1 ||
  journal "ATTENTION : status.sh --strict signale un écart ; le lancer pour le détail"
phase bootstrap persistant "$(($(maintenant) - T0))" "couche persistante appliquée ou inchangée"
journal "bootstrap terminé ; copier l'état de infra/persistant hors poste, chiffré (MI2)"
