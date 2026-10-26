#!/usr/bin/env bash
# Allumage de la couche éphémère (D11) : contrôles, apply horodaté, destruction programmée, HTTPS 200.
#   ./up.sh eks|ecs --debut-lab HH:MM [--chart DOSSIER --valeurs FICHIER] [--repetition-soutenance]
#   ./up.sh --prolonger --debut-lab HH:MM     (réarme la destruction programmée après un nouveau « Start Lab »)

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

usage() {
  sed -n '3,4p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

PROLONGER=0
REPETITION=0
ARG_ORCH=""
HEURE_LAB=""
CHART=""
VALEURS=""

[ "$#" -ge 1 ] || usage
case "$1" in
  eks | ecs) ARG_ORCH="$1" ;;
  --prolonger) PROLONGER=1 ;;
  *) usage ;;
esac
shift
while [ "$#" -gt 0 ]; do
  case "$1" in
    --debut-lab) HEURE_LAB="${2:-}"; shift 2 ;;
    --chart) CHART="${2:-}"; shift 2 ;;
    --valeurs) VALEURS="${2:-}"; shift 2 ;;
    --repetition-soutenance) REPETITION=1; shift ;;
    *) usage ;;
  esac
done

[[ "$HEURE_LAB" =~ ^[0-2][0-9]:[0-5][0-9]$ ]] || echec "--debut-lab HH:MM (heure locale du « Start Lab ») obligatoire"

calculer_fenetre() {
  local maintenant_s
  maintenant_s="$(maintenant)"
  DEBUT_LAB="$(date -d "$(date +%Y-%m-%d) $HEURE_LAB" +%s)"
  [ "$DEBUT_LAB" -le "$maintenant_s" ] || echec "heure de « Start Lab » dans le futur"
  FIN_LAB=$((DEBUT_LAB + DUREE_LAB_MIN * 60))
  RESTANT_MIN=$(((FIN_LAB - maintenant_s) / 60))
  [ "$RESTANT_MIN" -gt 0 ] || echec "session du lab déjà expirée : relancer « Start Lab »"
  journal "fin de session du lab à $(heure_locale "$FIN_LAB"), reste $RESTANT_MIN min"
}

controler_budget() {
  local cumul plafond
  cumul="$(cumul_couts)"
  plafond="$(awk -v b="$BUDGET_USD" -v p="$SEUIL_G5_PCT" 'BEGIN { printf "%.2f", b * p / 100 }')"
  journal "G5 : cumul estimé $cumul \$, seuil d'arrêt $plafond \$ ($SEUIL_G5_PCT % de $BUDGET_USD \$)"
  if awk -v c="$cumul" -v p="$plafond" 'BEGIN { exit !(c >= p) }'; then
    [ "$REPETITION" -eq 1 ] ||
      echec "G5 : seuil d'arrêt atteint, seules les répétitions de soutenance sont autorisées (--repetition-soutenance)"
    journal "G5 : seuil atteint, allumage autorisé pour une répétition de soutenance"
  fi
}

controler_persistant() {
  local zone certificat
  zone="$(aws_json route53 list-hosted-zones-by-name --dns-name "$DOMAINE" --max-items 1 |
    jq -r --arg d "$DOMAINE." '.HostedZones[] | select(.Name == $d) | .Id')"
  [ -n "$zone" ] || echec "couche persistante absente : zone $DOMAINE introuvable (appliquer infra/persistant)"
  certificat="$(aws_json acm list-certificates --certificate-statuses ISSUED |
    jq -r --arg d "$DOMAINE" '.CertificateSummaryList[] | select(.DomainName == $d) | .CertificateArn' | head -n 1)"
  [ -n "$certificat" ] || echec "couche persistante incomplète : certificat $DOMAINE non ISSUED"
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

planifier() {
  local nom="$1" echeance="$2" cible="$3" entree="$4" cible_json
  cible_json="$(jq -nc --arg arn "arn:aws:scheduler:::aws-sdk:$cible" --arg role "$LAB_ROLE_ARN" --arg entree "$entree" \
    '{Arn: $arn, RoleArn: $role, Input: $entree, RetryPolicy: {MaximumRetryAttempts: 3, MaximumEventAgeInSeconds: 900}}')"
  if ! aws scheduler create-schedule \
    --group-name "$GROUPE_PLANIF" \
    --name "$nom" \
    --schedule-expression "at($(utc_scheduler "$echeance"))" \
    --schedule-expression-timezone UTC \
    --flexible-time-window Mode=OFF \
    --action-after-completion DELETE \
    --target "$cible_json" >/dev/null; then
    journal "ATTENTION : planification $nom refusée ($cible)"
    return 1
  fi
  journal "planification $nom armée à $(heure_locale "$echeance") ($cible)"
}

armer_garde_fou() {
  local sorties t30 t10 essai ko=0
  local cluster_eks node_group id_rds arn_alb id_nat id_eip
  sorties="$(tf output -json 2>/dev/null || printf '{}')"
  lire() { printf '%s' "$sorties" | jq -r --arg c "$1" '.[$c].value // empty'; }
  cluster_eks="$(lire nom_cluster_eks)"
  node_group="$(lire nom_node_group)"
  id_rds="$(lire id_rds)"
  arn_alb="$(lire arn_alb)"
  id_nat="$(lire id_nat)"
  id_eip="$(lire id_allocation_eip)"

  t30=$((FIN_LAB - 30 * 60))
  t10=$((FIN_LAB - 10 * 60))
  if [ "$t10" -le "$(($(maintenant) + 120))" ]; then
    journal "ATTENTION : échéance T-10 trop proche pour armer le garde-fou"
    return 1
  fi

  aws scheduler delete-schedule-group --name "$GROUPE_PLANIF" >/dev/null 2>&1 || true
  for essai in 1 2 3 4 5 6 7; do
    if aws scheduler create-schedule-group --name "$GROUPE_PLANIF" \
      --tags Key=projet,Value=collector Key=couche,Value=ephemere >/dev/null 2>&1; then
      break
    fi
    if [ "$essai" -eq 7 ]; then
      journal "ATTENTION : groupe de planifications $GROUPE_PLANIF non recréé"
      return 1
    fi
    sleep 10
  done

  if [ -n "$node_group" ]; then
    planifier "${NOM}-t30-node-group" "$t30" "eks:deleteNodegroup" \
      "$(jq -nc --arg c "$cluster_eks" --arg n "$node_group" '{clusterName: $c, nodegroupName: $n}')" || ko=1
  fi
  if [ -n "$id_rds" ]; then
    planifier "${NOM}-t30-rds" "$t30" "rds:deleteDBInstance" \
      "$(jq -nc --arg i "$id_rds" '{DBInstanceIdentifier: $i, SkipFinalSnapshot: true, DeleteAutomatedBackups: true}')" || ko=1
  fi
  if [ -n "$arn_alb" ]; then
    planifier "${NOM}-t30-alb" "$t30" "elasticloadbalancingv2:deleteLoadBalancer" \
      "$(jq -nc --arg a "$arn_alb" '{LoadBalancerArn: $a}')" || ko=1
  fi
  if [ -n "$id_nat" ]; then
    planifier "${NOM}-t30-nat" "$t30" "ec2:deleteNatGateway" \
      "$(jq -nc --arg n "$id_nat" '{NatGatewayId: $n}')" || ko=1
  fi
  if [ -n "$cluster_eks" ]; then
    planifier "${NOM}-t10-cluster-eks" "$t10" "eks:deleteCluster" \
      "$(jq -nc --arg c "$cluster_eks" '{name: $c}')" || ko=1
  fi
  if [ -n "$id_eip" ]; then
    planifier "${NOM}-t10-eip" "$t10" "ec2:releaseAddress" \
      "$(jq -nc --arg a "$id_eip" '{AllocationId: $a}')" || ko=1
  fi
  return "$ko"
}

rappel_fin() {
  local limite_down=$((FIN_LAB - 30 * 60))
  journal "RAPPEL : lancer down.sh au plus tard à $(heure_locale "$limite_down") ; il doit finir AVANT « End Lab » ($(heure_locale "$FIN_LAB"))."
  journal "RAPPEL : lab arrêté, LabRole perd ses droits et la destruction programmée échoue (XP00 S8b) : tout ce qui reste allumé est facturé."
}

exiger_outils aws terraform jq curl sed awk sha256sum
sonde_identite
charger_session
calculer_fenetre
tf init -input=false >/dev/null

if [ "$PROLONGER" -eq 1 ]; then
  if [ -z "$ORCHESTRATEUR" ] || etat_vide; then
    echec "aucune session en cours à prolonger"
  fi
  ecrire_session
  armer_garde_fou || echec "garde-fou incomplet : voir les avertissements ci-dessus"
  journal "garde-fou réarmé ; fin de session à $(heure_locale "$FIN_LAB")"
  rappel_fin
  exit 0
fi

[ "$RESTANT_MIN" -ge "$MARGE_UP_MIN" ] ||
  echec "refus d'allumer : $RESTANT_MIN min restantes, minimum $MARGE_UP_MIN (D11). Relancer « Start Lab »."

controler_budget
[ "$ARG_ORCH" != "eks" ] || exiger_outils kubectl
[ -z "$CHART" ] || exiger_outils helm
[ -z "$CHART" ] || [ -f "$VALEURS" ] || echec "--valeurs FICHIER obligatoire avec --chart"

if [ "$ARG_ORCH" = "eks" ]; then
  statut_version="$(aws_json eks describe-cluster-versions --cluster-versions "$VERSION_EKS" | jq -r '.clusterVersions[0].versionStatus // empty')"
  [ "$statut_version" = "STANDARD_SUPPORT" ] || echec "EKS $VERSION_EKS hors support standard ($statut_version)"
fi

controler_persistant

T0="$(maintenant)"
if etat_vide; then
  journal "état vide : G2, contrôle de l'inventaire avant allumage"
  "$RACINE_TOOLS/inventaire.sh" --strict || echec "inventaire non vide : nettoyer ou importer les orphelins avant up"
  SESSION_TF="$(date -u +%Y%m%dT%H%MZ)"
  DEBUT_ALLUMAGE="$T0"
else
  [ -n "${SESSION_TF:-}" ] || echec "état non vide sans fichier de session : lancer down.sh d'abord"
  ancien="$(tf output -raw orchestrateur 2>/dev/null || true)"
  [ -z "$ancien" ] || [ "$ancien" = "$ARG_ORCH" ] || echec "l'état contient la variante $ancien : down.sh avant de changer"
  DEBUT_ALLUMAGE="${DEBUT_ALLUMAGE:-$T0}"
fi
ORCHESTRATEUR="$ARG_ORCH"

IP_ADMIN="$(ip_poste)"
variables_tf "$IP_ADMIN"
ecrire_session
journal "session $SESSION_TF, variante $ORCHESTRATEUR, adresse du poste relevée (non consignée)"
purger_bucket_photos
phase up controles "$(($(maintenant) - T0))" "identité, fenêtre, budget, persistant, inventaire"

if ! etat_vide; then
  journal "état existant : rafraîchissement avant apply (destruction programmée ou expiration possible)"
  tf apply -refresh-only -auto-approve -input=false | masquer
fi

mkdir -p "$DOSSIER_SESSION"
JOURNAL_APPLY="$DOSSIER_SESSION/apply-$(date -u +%Y%m%dT%H%M%SZ).jsonl"
T_APPLY="$(maintenant)"
journal "lancement de terraform apply"

set +e
tf apply -auto-approve -input=false -json |
  tee "$JOURNAL_APPLY" |
  jq -r --unbuffered 'select(.type == "apply_complete" or .type == "apply_errored" or .type == "diagnostic" or .type == "change_summary") | .["@message"]' |
  masquer
STATUT_APPLY="${PIPESTATUS[0]}"
set -e

journal "G1 : armement de la destruction programmée (T-30, T-10)"
armer_garde_fou || journal "ATTENTION : garde-fou non armé, surveiller l'échéance à la main"

jq -r 'select(.type == "apply_complete") | [.["@timestamp"], .hook.resource.addr] | @tsv' "$JOURNAL_APPLY" |
  while IFS=$'\t' read -r horodatage adresse; do
    case "$adresse" in
      aws_nat_gateway.unique | aws_db_instance.principale | aws_lb_listener.https | aws_eks_cluster.principal* | \
        aws_eks_node_group.principal* | aws_eks_addon.* | aws_ecs_cluster.principal*)
        phase up "apply:$adresse" "$(($(date -d "$horodatage" +%s) - T_APPLY))" "fin de création depuis le lancement de l'apply"
        ;;
    esac
  done

[ "$STATUT_APPLY" -eq 0 ] || echec "apply en échec : corriger puis relancer up.sh (rafraîchissement automatique), ou down.sh"
phase up apply "$(($(maintenant) - T_APPLY))" "apply terminé"
sauvegarder_etat up

if [ "$ORCHESTRATEUR" = "eks" ]; then
  aws eks update-kubeconfig --name "$NOM" --alias "$NOM" --profile "$AWS_PROFILE" >/dev/null
  kubectl --context "$NOM" wait --for=condition=Ready nodes --all --timeout=10m >/dev/null
  phase up noeuds-ready "$(($(maintenant) - T0))" "kubeconfig et nœuds Ready"
  if [ -n "$CHART" ]; then
    helm --kube-context "$NOM" upgrade --install front "$CHART" -f "$VALEURS" --wait --timeout 10m >/dev/null
    phase up chart "$(($(maintenant) - T0))" "front installé par helm"
  else
    journal "installer le front (Service NodePort 30080) : l'attente du HTTPS 200 commence"
  fi
else
  journal "variante ECS : services du POC à décrire après le verdict XP01 ; l'attente du HTTPS 200 commence"
fi

URL="$(tf output -raw url)"
journal "attente du premier HTTPS 200 sur $URL (30 min au plus)"
CODE="000"
LIMITE=$(($(maintenant) + 30 * 60))
while [ "$(maintenant)" -lt "$LIMITE" ]; do
  CODE="$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$URL" || true)"
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

journal "ALB restreint à l'adresse du poste (ME5) : contrôle après déploiement X17/X20 à livrer avant toute ouverture (AC10, AC11)"
journal "session prête ; destruction programmée à $(heure_locale $((FIN_LAB - 30 * 60))) et $(heure_locale $((FIN_LAB - 10 * 60)))"
rappel_fin
