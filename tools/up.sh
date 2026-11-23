#!/usr/bin/env bash
# Allumage de la couche éphémère (D11) : contrôles, apply horodaté, destruction programmée (G1) et rappel sur le poste (G7)
# armés dès la fin de l'apply, front, HTTPS 200, contrôle après déploiement (US-004 AC11). ALB restreint au poste (ME5).
#   ./up.sh --fin-lab HH:MM|--debut-lab HH:MM [--chart DOSSIER --valeurs FICHIER] [--repetition-soutenance]
#   ./up.sh --prolonger --fin-lab HH:MM|--debut-lab HH:MM   (réarme G1 et G7 après « Start Lab »)

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

usage() {
  sed -n '4,5p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

PROLONGER=0
REPETITION=0
HEURE_LAB=""
HEURE_FIN=""
CHART=""
VALEURS=""

[ "$#" -ge 1 ] || usage
while [ "$#" -gt 0 ]; do
  case "$1" in
    --prolonger) PROLONGER=1; shift ;;
    --debut-lab) HEURE_LAB="${2:-}"; shift 2 ;;
    --fin-lab) HEURE_FIN="${2:-}"; shift 2 ;;
    --chart) CHART="${2:-}"; shift 2 ;;
    --valeurs) VALEURS="${2:-}"; shift 2 ;;
    --repetition-soutenance) REPETITION=1; shift ;;
    *) usage ;;
  esac
done

HEURE_VALIDE='^([01][0-9]|2[0-3]):[0-5][0-9]$'
if [ -n "$HEURE_FIN" ]; then
  [[ "$HEURE_FIN" =~ $HEURE_VALIDE ]] || echec "--fin-lab HH:MM (heure locale lue sur le compteur du lab)"
else
  [[ "$HEURE_LAB" =~ $HEURE_VALIDE ]] ||
    echec "--fin-lab HH:MM (compteur du lab, préféré : XP01 X14) ou --debut-lab HH:MM obligatoire"
fi

calculer_fenetre() {
  local maintenant_s
  maintenant_s="$(maintenant)"
  if [ -n "$HEURE_FIN" ]; then
    FIN_LAB="$(date -d "$(date +%Y-%m-%d) $HEURE_FIN" +%s)"
    [ "$FIN_LAB" -gt "$maintenant_s" ] || FIN_LAB=$((FIN_LAB + 24 * 3600))
    DEBUT_LAB=$((FIN_LAB - DUREE_LAB_MIN * 60))
  else
    DEBUT_LAB="$(date -d "$(date +%Y-%m-%d) $HEURE_LAB" +%s)"
    [ "$DEBUT_LAB" -le "$maintenant_s" ] || DEBUT_LAB=$((DEBUT_LAB - 24 * 3600))
    FIN_LAB=$((DEBUT_LAB + DUREE_LAB_MIN * 60))
  fi
  RESTANT_MIN=$(((FIN_LAB - maintenant_s) / 60))
  [ "$RESTANT_MIN" -gt 0 ] || echec "session du lab déjà expirée : relancer « Start Lab »"
  [ "$RESTANT_MIN" -le "$DUREE_LAB_MIN" ] ||
    echec "fin du lab dans $RESTANT_MIN min, au-delà des $DUREE_LAB_MIN min d'une session : heure saisie erronée ?"
  journal "fin de session du lab à $(heure_locale "$FIN_LAB"), reste $RESTANT_MIN min"
}

exiger_persistant() {
  [ -n "$(zone_collector)" ] || echec "couche persistante absente : zone $DOMAINE introuvable (bash tools/bootstrap.sh)"
  [ -n "$(certificat_emis)" ] || echec "couche persistante incomplète : certificat $DOMAINE non ISSUED"
  journal "couche persistante présente : zone et certificat ISSUED"
}

purger_bucket_photos() {
  aws s3api head-bucket --bucket "$BUCKET_PHOTOS" >/dev/null 2>&1 || {
    journal "bucket des photos absent (créé par bootstrap.sh) : purge sautée"
    return 0
  }
  journal "purge du bucket des photos (D07)"
  aws s3 rm "s3://$BUCKET_PHOTOS" --recursive --only-show-errors
}

signaler_hors_etat() {
  local connus categorie ident hors=0
  connus="$(tf show -json 2>/dev/null |
    jq -r '[.. | objects | (.id?, .name?, .arn?, .identifier?) | strings] | unique[]' || true)"
  while read -r categorie ident _; do
    [ -n "$ident" ] || continue
    [ "$categorie" != "scheduler" ] || continue
    ident="${ident%.}"
    if ! grep -qxF -- "$ident" <<<"$connus"; then
      journal "ATTENTION : hors état Terraform : $categorie $ident"
      hors=$((hors + 1))
    fi
  done < <("$RACINE_TOOLS/inventaire.sh" 2>/dev/null | sed -n 's/^  //p')
  if [ "$hors" -eq 0 ]; then
    journal "reprise : toute ressource inventoriée est suivie par l'état"
  else
    journal "reprise : $hors ressource(s) hors état (nœuds et interfaces gérés par EKS attendus) ; tout orphelin se supprime ou s'importe (D11)"
  fi
}

rappel_fin() {
  local limite_down
  limite_down="$(echeance_lourde "$FIN_LAB")"
  journal "RAPPEL : lancer down.sh au plus tard à $(heure_locale "$limite_down") ; il doit finir AVANT « End Lab » ($(heure_locale "$FIN_LAB"))."
  journal "RAPPEL : lab arrêté, LabRole perd ses droits et la destruction programmée échoue (XP00 S8b) : tout ce qui reste allumé est facturé."
}

exiger_outils aws terraform jq curl sed awk sha256sum iconv base64
if [ "$PROLONGER" -eq 0 ]; then
  autoriser_allumage_g5 "$REPETITION" || echec "G5 : allumage refusé"
fi
sonde_identite
charger_session
calculer_fenetre
tf init -input=false >/dev/null

if [ "$PROLONGER" -eq 1 ]; then
  if [ -z "${SESSION_TF:-}" ] || etat_vide; then
    echec "aucune session en cours à prolonger"
  fi
  ecrire_session
  armer_garde_fou || echec "garde-fou incomplet : voir les avertissements ci-dessus"
  programmer_rappel_session
  journal "garde-fou réarmé ; fin de session à $(heure_locale "$FIN_LAB")"
  rappel_fin
  exit 0
fi

[ "${EXTINCTION_INCOMPLETE:-0}" != "1" ] ||
  echec "extinction précédente incomplète : relancer down.sh jusqu'à un inventaire vide avant tout up (AC4)"

[ "$RESTANT_MIN" -ge "$MARGE_UP_MIN" ] ||
  echec "refus d'allumer : $RESTANT_MIN min restantes, minimum $MARGE_UP_MIN (D11). Relancer « Start Lab »."

CHART="${CHART:-$DEPOT/infra/front}"
VALEURS="${VALEURS:-$DEPOT/infra/front/valeurs-aws.yaml}"
exiger_outils kubectl helm
[ -f "$CHART/Chart.yaml" ] || echec "chart du front introuvable : $CHART"
[ -f "$VALEURS" ] || echec "fichier de valeurs introuvable : $VALEURS"

statut_version="$(aws_json eks describe-cluster-versions --cluster-versions "$VERSION_EKS" | jq -r '.clusterVersions[0].versionStatus // empty')"
[ "$statut_version" = "STANDARD_SUPPORT" ] || echec "EKS $VERSION_EKS hors support standard ($statut_version)"

exiger_persistant

T0="$(maintenant)"
if etat_vide; then
  journal "état vide : G2, inventaire strict avant allumage (AC4)"
  "$RACINE_TOOLS/status.sh" --strict || echec "inventaire non vide : nettoyer ou importer les orphelins avant up"
  SESSION_TF="$(nouvelle_session)"
  DEBUT_ALLUMAGE="$T0"
else
  [ -n "${SESSION_TF:-}" ] || echec "état non vide sans fichier de session : lancer down.sh d'abord"
  journal "état non vide : reprise d'un up interrompu, inventaire comparé à l'état (D11)"
  signaler_hors_etat
  DEBUT_ALLUMAGE="${DEBUT_ALLUMAGE:-$T0}"
fi

IP_ADMIN="$(ip_poste)"
variables_tf "$IP_ADMIN"
ecrire_session
journal "session $SESSION_TF, adresse du poste relevée (non consignée)"
purger_bucket_photos
phase up controles "$(($(maintenant) - T0))" "identité, fenêtre, budget, persistant, inventaire"

if ! etat_vide; then
  journal "état existant : rafraîchissement avant apply (destruction programmée ou expiration possible)"
  tf apply -refresh-only -auto-approve -input=false "${ALB_FERME[@]}" | masquer_ip
fi

mkdir -p "$DOSSIER_SESSION"
JOURNAL_APPLY="$DOSSIER_SESSION/apply-$(date -u +%Y%m%dT%H%M%SZ).jsonl"
T_APPLY="$(maintenant)"
journal "lancement de terraform apply (ALB restreint au poste, ME5)"

# shellcheck disable=SC2034
GARDE_FOU_TENTE=0
# shellcheck disable=SC2034
APPLY_LANCE=1
trap armer_en_sortie EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP
set +e
tf apply -auto-approve -input=false -json "${ALB_FERME[@]}" |
  tee "$JOURNAL_APPLY" |
  command jq -r --unbuffered 'select(.type == "apply_complete" or .type == "apply_errored" or .type == "diagnostic" or .type == "change_summary") | .["@message"]' |
  masquer_ip
STATUT_APPLY="${PIPESTATUS[0]}"
set -e

armer_garde_fous

jq -r 'select(.type == "apply_complete") | [.["@timestamp"], .hook.resource.addr] | @tsv' "$JOURNAL_APPLY" |
  while IFS=$'\t' read -r horodatage adresse; do
    case "$adresse" in
      aws_nat_gateway.unique | aws_db_instance.principale | aws_lb_listener.https | aws_eks_cluster.principal* | \
        aws_eks_node_group.principal* | aws_eks_addon.*)
        phase up "apply:$adresse" "$(($(date -d "$horodatage" +%s) - T_APPLY))" "fin de création depuis le lancement de l'apply"
        ;;
    esac
  done

[ "$STATUT_APPLY" -eq 0 ] || echec "apply en échec : corriger puis relancer up.sh (rafraîchissement automatique), ou down.sh"
phase up apply "$(($(maintenant) - T_APPLY))" "apply terminé"
sauvegarder_etat "$DOSSIER_TF" ephemere up

aws eks update-kubeconfig --name "$NOM" --alias "$NOM" --profile "$AWS_PROFILE" >/dev/null
kubectl --context "$NOM" wait --for=condition=Ready nodes --all --timeout=10m >/dev/null
phase up noeuds-ready "$(($(maintenant) - T0))" "kubeconfig et nœuds Ready"
journal "socle : namespaces applicatifs et sortie sans IMDS par namespace (XP01 session 1)"
k apply -f "$DOSSIER_K8S/socle.yaml" >/dev/null
helm --kube-context "$NOM" upgrade --install front "$CHART" -n front -f "$VALEURS" --set "service.nodePort=$(tf output -raw nodeport_front)" --wait --timeout 10m >/dev/null
phase up chart "$(($(maintenant) - T0))" "front installé par helm"

URL="$(tf output -raw url)"
journal "attente du premier HTTPS 200 sur $URL (30 min au plus)"
CODE="000"
LIMITE=$(($(maintenant) + 30 * 60))
while [ "$(maintenant)" -lt "$LIMITE" ]; do
  CODE="$(curl -s -o "$NUL" -w '%{http_code}' --max-time 10 "$URL" || true)"
  [ "$CODE" = "200" ] && break
  sleep 10
done

DUREE_MIN=$((($(maintenant) - T0 + 59) / 60))
if [ "$CODE" = "200" ]; then
  phase up https-200 "$(($(maintenant) - T0))" "premier HTTPS 200 en $DUREE_MIN min (cible 45, alerte 60)"
  journal "X4 : HTTPS 200 en $DUREE_MIN min (cible 45, alerte 60)"
else
  phase up https-200 "$(($(maintenant) - T0))" "pas de HTTPS 200 (dernier code $CODE)"
  rappel_fin
  echec "pas de HTTPS 200 après $DUREE_MIN min (dernier code $CODE)"
fi

journal "contrôle après déploiement : TLS, X17, XE4, X20 (AC7 à AC9, AC11)"
STATUT_CONTROLE=0
"$RACINE_TOOLS/controle-apres-deploiement.sh" || STATUT_CONTROLE=1
journal "session prête ; destruction programmée à $(heure_locale "$(echeance_lourde "$FIN_LAB")") et $(heure_locale "$(echeance_finale "$FIN_LAB")")"
rappel_fin
[ "$STATUT_CONTROLE" -eq 0 ] ||
  echec "contrôle après déploiement rouge : ALB restreint au poste, ouverture refusée (AC10) ; corriger puis relancer le contrôle"
journal "contrôle vert : ouverture possible par tools/alb.sh ouvrir (ME4 et ME8 restent à prouver avant la démonstration)"
