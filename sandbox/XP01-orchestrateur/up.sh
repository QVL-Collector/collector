#!/usr/bin/env bash
# Allumage du bac à sable XP01 : contrôles, bucket en CLI, apply horodaté, destruction programmée, HTTPS 200.
#   ./up.sh eks|ecs --debut-lab HH:MM [--libelle EKS1] [--chart DOSSIER --valeurs FICHIER]
#   ./up.sh --prolonger --debut-lab HH:MM     (réarme la destruction programmée après un nouveau « Start Lab »)

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

usage() {
  sed -n '2,4p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

PROLONGER=0
ARG_ORCH=""
ARG_LIBELLE=""
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
    --libelle) ARG_LIBELLE="${2:-}"; shift 2 ;;
    --chart) CHART="${2:-}"; shift 2 ;;
    --valeurs) VALEURS="${2:-}"; shift 2 ;;
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

planifier() {
  local nom="$1" echeance="$2" cible="$3" entree="$4" cible_json
  cible_json="$(jq -nc --arg arn "arn:aws:scheduler:::aws-sdk:$cible" --arg role "$LAB_ROLE_ARN" --arg entree "$entree"     '{Arn: $arn, RoleArn: $role, Input: $entree, RetryPolicy: {MaximumRetryAttempts: 3, MaximumEventAgeInSeconds: 900}}')"
  if ! aws scheduler create-schedule     --group-name "$GROUPE_PLANIF"     --name "$nom"     --schedule-expression "at($(utc_scheduler "$echeance"))"     --schedule-expression-timezone UTC     --flexible-time-window Mode=OFF     --action-after-completion DELETE     --target "$cible_json" >/dev/null; then
    journal "ATTENTION : planification $nom refusée ($cible)"
    return 1
  fi
  journal "planification $nom armée à $(heure_locale "$echeance") ($cible)"
}

armer_garde_fou() {
  local sorties t30 t10 essai ko=0
  local cluster_eks node_group cluster_ecs service_ecs id_rds arn_alb id_nat id_eip
  sorties="$(tf output -json 2>/dev/null || printf '{}')"
  lire() { printf '%s' "$sorties" | jq -r --arg c "$1" '.[$c].value // empty'; }
  cluster_eks="$(lire nom_cluster_eks)"
  node_group="$(lire nom_node_group)"
  cluster_ecs="$(lire nom_cluster_ecs)"
  service_ecs="$(lire nom_service_ecs)"
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
    if aws scheduler create-schedule-group --name "$GROUPE_PLANIF"       --tags Key=projet,Value=collector Key=couche,Value=ephemere Key=xp,Value=XP01 >/dev/null 2>&1; then
      break
    fi
    if [ "$essai" -eq 7 ]; then
      journal "ATTENTION : groupe de planifications $GROUPE_PLANIF non recréé"
      return 1
    fi
    sleep 10
  done

  if [ -n "$node_group" ]; then
    planifier "${NOM}-t30-node-group" "$t30" "eks:deleteNodegroup"       "$(jq -nc --arg c "$cluster_eks" --arg n "$node_group" '{clusterName: $c, nodegroupName: $n}')" || ko=1
  fi
  if [ -n "$service_ecs" ]; then
    planifier "${NOM}-t30-service-ecs" "$t30" "ecs:deleteService"       "$(jq -nc --arg c "$cluster_ecs" --arg s "$service_ecs" '{cluster: $c, service: $s, force: true}')" || ko=1
  fi
  if [ -n "$id_rds" ]; then
    planifier "${NOM}-t30-rds" "$t30" "rds:deleteDBInstance"       "$(jq -nc --arg i "$id_rds" '{DBInstanceIdentifier: $i, SkipFinalSnapshot: true, DeleteAutomatedBackups: true}')" || ko=1
  fi
  if [ -n "$arn_alb" ]; then
    planifier "${NOM}-t30-alb" "$t30" "elasticloadbalancingv2:deleteLoadBalancer"       "$(jq -nc --arg a "$arn_alb" '{LoadBalancerArn: $a}')" || ko=1
  fi
  if [ -n "$id_nat" ]; then
    planifier "${NOM}-t30-nat" "$t30" "ec2:deleteNatGateway"       "$(jq -nc --arg n "$id_nat" '{NatGatewayId: $n}')" || ko=1
  fi
  if [ -n "$cluster_eks" ]; then
    planifier "${NOM}-t10-cluster-eks" "$t10" "eks:deleteCluster"       "$(jq -nc --arg c "$cluster_eks" '{name: $c}')" || ko=1
  fi
  if [ -n "$cluster_ecs" ]; then
    planifier "${NOM}-t10-cluster-ecs" "$t10" "ecs:deleteCluster"       "$(jq -nc --arg c "$cluster_ecs" '{cluster: $c}')" || ko=1
  fi
  if [ -n "$id_eip" ]; then
    planifier "${NOM}-t10-eip" "$t10" "ec2:releaseAddress"       "$(jq -nc --arg a "$id_eip" '{AllocationId: $a}')" || ko=1
  fi
  return "$ko"
}

exiger_outils aws terraform jq curl sed sha256sum
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
  exit 0
fi

[ "$RESTANT_MIN" -ge "$MARGE_UP_MIN" ] ||
  echec "refus d'allumer : $RESTANT_MIN min restantes, minimum $MARGE_UP_MIN (D11). Relancer « Start Lab »."

[ "$ARG_ORCH" != "eks" ] || exiger_outils kubectl
[ -z "$CHART" ] || exiger_outils helm
[ -z "$CHART" ] || [ -f "$VALEURS" ] || echec "--valeurs FICHIER obligatoire avec --chart"

if [ "$ARG_ORCH" = "eks" ]; then
  statut_version="$(aws_json eks describe-cluster-versions --cluster-versions "$VERSION_EKS" | jq -r '.clusterVersions[0].versionStatus // empty')"
  [ "$statut_version" = "STANDARD_SUPPORT" ] || echec "EKS $VERSION_EKS hors support standard ($statut_version)"
fi

if etat_vide; then
  journal "état vide : contrôle de l'inventaire avant allumage"
  "$RACINE_XP/inventaire.sh" --strict || echec "inventaire non vide : nettoyer ou importer les orphelins avant up"
  SESSION_TF="$(date -u +%Y%m%dT%H%MZ)"
  LIBELLE="${ARG_LIBELLE:-${ARG_ORCH^^}-$(date +%Y%m%d)}"
else
  [ -n "${SESSION_TF:-}" ] || echec "état non vide sans fichier de session : lancer down.sh d'abord"
  ancien="$(tf output -raw orchestrateur 2>/dev/null || true)"
  [ -z "$ancien" ] || [ "$ancien" = "$ARG_ORCH" ] || echec "l'état contient la variante $ancien : down.sh avant de changer"
  LIBELLE="${ARG_LIBELLE:-$LIBELLE}"
fi
ORCHESTRATEUR="$ARG_ORCH"

IP_ADMIN="$(ip_poste)"
variables_tf "$IP_ADMIN"
ecrire_session
journal "session $LIBELLE, variante $ORCHESTRATEUR, adresse du poste relevée (non consignée)"

if ! etat_vide; then
  journal "état existant : rafraîchissement avant apply (destruction programmée ou expiration possible)"
  tf apply -refresh-only -auto-approve -input=false | masquer
fi

if ! aws s3api head-bucket --bucket "$BUCKET" >/dev/null 2>&1; then
  journal "création du bucket de test en CLI"
  aws s3api create-bucket --bucket "$BUCKET" >/dev/null
  aws s3api put-public-access-block --bucket "$BUCKET" \
    --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
  aws s3api put-bucket-encryption --bucket "$BUCKET" \
    --server-side-encryption-configuration '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'
  aws s3api put-bucket-policy --bucket "$BUCKET" --policy "$(jq -nc --arg b "arn:aws:s3:::$BUCKET" \
    '{Version: "2012-10-17", Statement: [{Sid: "RefusHorsTLS", Effect: "Deny", Principal: "*", Action: "s3:*", Resource: [$b, ($b + "/*")], Condition: {Bool: {"aws:SecureTransport": "false"}}}]}')"
  aws s3api put-bucket-tagging --bucket "$BUCKET" \
    --tagging 'TagSet=[{Key=projet,Value=collector},{Key=couche,Value=ephemere},{Key=xp,Value=XP01}]'
fi

mkdir -p "$DOSSIER_SESSION"
JOURNAL_APPLY="$DOSSIER_SESSION/apply-$(date -u +%Y%m%dT%H%M%SZ).jsonl"
T0="$(maintenant)"
journal "G2 : lancement de terraform apply"

set +e
tf apply -auto-approve -input=false -json |
  tee "$JOURNAL_APPLY" |
  jq -r --unbuffered 'select(.type == "apply_complete" or .type == "apply_errored" or .type == "diagnostic" or .type == "change_summary") | .["@message"]' |
  masquer
STATUT_APPLY="${PIPESTATUS[0]}"
set -e

journal "armement de la destruction programmée (T-30, T-10)"
armer_garde_fou || journal "ATTENTION : garde-fou non armé, surveiller l'échéance à la main"

jq -r 'select(.type == "apply_complete") | [.["@timestamp"], .hook.resource.addr] | @tsv' "$JOURNAL_APPLY" |
  while IFS=$'\t' read -r horodatage adresse; do
    case "$adresse" in
      aws_nat_gateway.unique | aws_db_instance.test | aws_lb_listener.https | aws_eks_cluster.principal* | \
        aws_eks_node_group.principal* | aws_eks_addon.* | aws_ecs_cluster.principal* | aws_ecs_service.pdm_test*)
        mesure G2 "apply:$adresse" "$(($(date -d "$horodatage" +%s) - T0))" s "" "" "fin de création depuis le lancement de l'apply"
        ;;
    esac
  done

[ "$STATUT_APPLY" -eq 0 ] || echec "apply en échec : corriger puis relancer up.sh (rafraîchissement automatique), ou down.sh"
mesure G2 apply "$(($(maintenant) - T0))" s "" "" "apply terminé"
sauvegarder_etat up

if [ "$ORCHESTRATEUR" = "eks" ]; then
  aws eks update-kubeconfig --name "$NOM" --alias "$NOM" --profile "$AWS_PROFILE" >/dev/null
  kubectl wait --for=condition=Ready nodes --all --timeout=10m >/dev/null
  mesure G2 noeuds-ready "$(($(maintenant) - T0))" s "" "" "kubeconfig et nœuds Ready"
  journal "nœuds prêts ; politiques réseau du CNI à constater en G7-5"
  if [ -n "$CHART" ]; then
    helm upgrade --install xp01-test "$CHART" -f "$VALEURS" --wait --timeout 10m >/dev/null
    mesure G2 chart "$(($(maintenant) - T0))" s "" "" "chart de test installé"
  else
    journal "installer le chart de test (NodePort 30080) : l'attente du HTTPS 200 commence"
  fi
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

DUREE_S=$(($(maintenant) - T0))
DUREE_MIN=$(((DUREE_S + 59) / 60))
if [ "$CODE" = "200" ]; then
  ATTEINT="non"
  [ "$DUREE_MIN" -gt 45 ] || ATTEINT="oui"
  mesure G2 https-200 "$DUREE_MIN" min 45 "$ATTEINT" "premier HTTPS 200 par l'ALB"
  journal "G2 : HTTPS 200 en $DUREE_MIN min (seuil 45, alerte 60)"
else
  mesure G2 https-200 "$DUREE_MIN" min 45 non "pas de HTTPS 200 (dernier code $CODE)"
  echec "pas de HTTPS 200 après $DUREE_MIN min (dernier code $CODE)"
fi

journal "session prête ; destruction programmée à $(heure_locale $((FIN_LAB - 30 * 60))) et $(heure_locale $((FIN_LAB - 10 * 60))) ; down.sh en fin de travail"
