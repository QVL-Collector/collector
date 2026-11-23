#!/usr/bin/env bash
# Garde-fous de coût sans AWS (US-005) : cibles et champs des planifications G1 (AC1, AC2), journal et cumul des coûts G5
# (AC5, AC8), seuil d'arrêt à 70 % (AC6), rappel du poste G7 (AC7) avec PowerShell et at simulés, armement et piège de sortie de up.sh (AC1) avec Scheduler et Terraform simulés.
#   bash tools/tests/garde-fous-cout.sh
# shellcheck disable=SC2034

set -euo pipefail

# shellcheck source=../lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib.sh"
set +e
set -uo pipefail

TEMPORAIRE="$(mktemp -d)"
trap 'rm -rf "$TEMPORAIRE"' EXIT
ECHECS=0

verifier() {
  local nom="$1" obtenu="$2" attendu="$3"
  if [ "$obtenu" = "$attendu" ]; then
    printf 'ok    %s\n' "$nom"
  else
    printf 'ÉCHEC %s : « %s », attendu « %s »\n' "$nom" "$obtenu" "$attendu"
    ECHECS=$((ECHECS + 1))
  fi
}

reussit() { if "$@" 2>/dev/null; then printf oui; else printf non; fi; }

ressource() { printf '{"mode":"managed","address":"%s","type":"%s","values":%s}' "$1" "${1%%.*}" "$2"; }

etat() {
  local IFS=,
  printf '{"values":{"root_module":{"resources":[%s]}}}' "$*"
}

FIN=2000000000
CLUSTER="$(ressource aws_eks_cluster.principal '{"name":"collector","arn":"arn:aws:eks:us-east-1:111122223333:cluster/collector"}')"
NODE_GROUP="$(ressource aws_eks_node_group.principal '{"node_group_name":"collector-noeuds","id":"collector:collector-noeuds"}')"
RDS="$(ressource aws_db_instance.principale '{"identifier":"collector","id":"db-ABC"}')"
ALB="$(ressource aws_lb.public '{"arn":"arn:aws:elasticloadbalancing:us-east-1:111122223333:loadbalancer/app/collector/1"}')"
NAT="$(ressource aws_nat_gateway.unique '{"id":"nat-0123"}')"
EIP="$(ressource aws_eip.nat '{"allocation_id":"eipalloc-0123","public_ip":"203.0.113.7"}')"
VPC="$(ressource aws_vpc.principal '{"id":"vpc-0123"}')"
KMS_DATA='{"mode":"data","address":"data.aws_kms_alias.secrets_eks","type":"aws_kms_alias","values":{"target_key_arn":"arn:aws:kms:us-east-1:111122223333:key/1"}}'
KMS="$(ressource aws_kms_key.secrets_eks '{"arn":"arn:aws:kms:us-east-1:111122223333:key/1","id":"1"}')"
BUCKET="$(ressource aws_s3_bucket.photos '{"id":"collector-photos-abc","arn":"arn:aws:s3:::collector-photos-abc"}')"

printf 'G1 : cibles et champs des planifications (AC1, AC2)\n'
PLANIFS="$(etat "$CLUSTER" "$NODE_GROUP" "$RDS" "$ALB" "$NAT" "$EIP" "$VPC" "$KMS_DATA" | ids_garde_fou | planifications_garde_fou "$FIN")"
verifier "six planifications pour un état complet" "$(grep -c . <<<"$PLANIFS")" 6
verifier "noms des planifications" "$(jq -r '.nom' <<<"$PLANIFS" | sort | paste -sd' ')" \
  "collector-t10-cluster-eks collector-t10-eip collector-t30-alb collector-t30-nat collector-t30-node-group collector-t30-rds"
verifier "node group, RDS, ALB et NAT à T-30" \
  "$(jq -r --argjson e $((FIN - 1800)) 'select(.echeance == $e) | .cible' <<<"$PLANIFS" | sort | paste -sd' ')" \
  "ec2:deleteNatGateway eks:deleteNodegroup elasticloadbalancingv2:deleteLoadBalancer rds:deleteDBInstance"
verifier "cluster et adresse à T-10" \
  "$(jq -r --argjson e $((FIN - 600)) 'select(.echeance == $e) | .cible' <<<"$PLANIFS" | sort | paste -sd' ')" \
  "ec2:releaseAddress eks:deleteCluster"
verifier "champs du node group" "$(jq -c 'select(.cible == "eks:deleteNodegroup") | .entree' <<<"$PLANIFS")" \
  '{"ClusterName":"collector","NodegroupName":"collector-noeuds"}'
verifier "champs de la RDS, sans instantané final" "$(jq -c 'select(.cible == "rds:deleteDBInstance") | .entree' <<<"$PLANIFS")" \
  '{"DbInstanceIdentifier":"collector","SkipFinalSnapshot":true,"DeleteAutomatedBackups":true}'
verifier "champs de l'ALB" "$(jq -r 'select(.cible | endswith("deleteLoadBalancer")) | .entree | keys | join(",")' <<<"$PLANIFS")" \
  LoadBalancerArn
verifier "champs de la NAT" "$(jq -c 'select(.cible == "ec2:deleteNatGateway") | .entree' <<<"$PLANIFS")" '{"NatGatewayId":"nat-0123"}'
verifier "champs du cluster" "$(jq -c 'select(.cible == "eks:deleteCluster") | .entree' <<<"$PLANIFS")" '{"Name":"collector"}'
verifier "champs de l'adresse" "$(jq -c 'select(.cible == "ec2:releaseAddress") | .entree' <<<"$PLANIFS")" \
  '{"AllocationId":"eipalloc-0123"}'

CIBLE="$(head -n 1 <<<"$PLANIFS" | cible_planification "arn:aws:iam::111122223333:role/LabRole")"
verifier "cible universelle Scheduler" "$(jq -r '.Arn | startswith("arn:aws:scheduler:::aws-sdk:")' <<<"$CIBLE")" true
verifier "rôle confié à Scheduler" "$(jq -r '.RoleArn' <<<"$CIBLE")" "arn:aws:iam::111122223333:role/LabRole"
verifier "Input en chaîne JSON" "$(jq -r '.Input | type' <<<"$CIBLE")" string
verifier "Input identique à l'entrée" "$(jq -c '.Input | fromjson' <<<"$CIBLE")" "$(head -n 1 <<<"$PLANIFS" | jq -c '.entree')"
verifier "champs de la cible" "$(jq -r 'keys | join(",")' <<<"$CIBLE")" "Arn,Input,RetryPolicy,RoleArn"

PARTIEL="$(etat "$VPC" "$NAT" "$EIP" | ids_garde_fou | planifications_garde_fou "$FIN")"
verifier "apply partiel : NAT et adresse seules armées" "$(jq -r '.nom' <<<"$PARTIEL" | sort | paste -sd' ')" \
  "collector-t10-eip collector-t30-nat"
verifier "état vide : aucune planification" "$(etat | ids_garde_fou | planifications_garde_fou "$FIN")" ""
verifier "objet vide : aucune planification" "$(printf '{}' | planifications_garde_fou "$FIN")" ""
PERSISTANT="$(etat "$KMS_DATA" "$KMS" "$BUCKET" | ids_garde_fou | planifications_garde_fou "$FIN")"
verifier "clé KMS et bucket jamais ciblés" "$PERSISTANT" ""
verifier "aucune cible persistante dans un état complet" \
  "$(jq -r '.cible' <<<"$PLANIFS" | grep -Eci 'kms|s3|route53|acm|logs|secretsmanager')" 0

printf 'G5 : journal et cumul des coûts (AC5, AC8)\n'
CSV_COUTS="$TEMPORAIRE/couts.csv"
verifier "cumul sans journal" "$(cumul_couts)" 0.00
verifier "coût renvoyé, identique à la ligne écrite" "$(ajouter_cout 58)" 0.33
verifier "en-tête créé" "$(head -n 1 "$CSV_COUTS")" "$ENTETE_COUTS"
verifier "ligne date;durée;coût" "$(sed -n 2p "$CSV_COUTS")" "$(date -u +%Y-%m-%d);58;0.33"
ajouter_cout 240 >/dev/null
verifier "coût d'une session complète (4 h à 0,34 \$/h)" "$(sed -n 3p "$CSV_COUTS" | cut -d';' -f3)" 1.36
verifier "cumul de deux sessions" "$(cumul_couts)" 1.69
printf '2026-11-01;60;0.34' >"$TEMPORAIRE/sans-fin.csv"
CSV_COUTS="$TEMPORAIRE/sans-fin.csv"
ajouter_cout 60 >/dev/null
verifier "ajout après une dernière ligne sans fin de ligne" "$(wc -l <"$CSV_COUTS" | tr -d ' ')" 2
verifier "lignes conservées" "$(cumul_couts)" 0.34

LIGNE_SURE='^[0-9]{4}-[0-9]{2}-[0-9]{2};[0-9]+;[0-9]+\.[0-9]{2}$'
verifier "AC8 : ligne sans compte, adresse ni session" "$(ligne_cout 2026-11-20 58 | grep -Ec "$LIGNE_SURE")" 1
JOURNAL_REEL="$RACINE_TOOLS/../docs/mesures/couts.csv"
if [ -f "$JOURNAL_REEL" ]; then
  verifier "AC8 : en-tête du journal du dépôt" "$(head -n 1 "$JOURNAL_REEL" | tr -d '\r')" "$ENTETE_COUTS"
  verifier "AC8 : lignes du journal du dépôt au format sûr" \
    "$(sed 1d "$JOURNAL_REEL" | tr -d '\r' | grep -vEc "$LIGNE_SURE")" 0
  verifier "AC8 : ni compte, ni adresse, ni session dans le journal du dépôt" \
    "$(grep -Ec '[0-9]{12}|([0-9]{1,3}\.){3}[0-9]{1,3}|[0-9]{8}T[0-9]{4}Z|arn:|ASIA|AKIA' "$JOURNAL_REEL")" 0
fi

printf 'G5 : seuil d'"'"'arrêt à 70 %% du budget (AC6)\n'
verifier "seuil à 35 \$ pour 50 \$" "$(seuil_g5_usd)" 35.00
CSV_COUTS="$TEMPORAIRE/sous-seuil.csv"
printf '%s\r\n2026-11-01;600;10.00\r\n2026-11-02;600;20.00\r\n2026-11-03;600;4.99\r\n' "$ENTETE_COUTS" >"$CSV_COUTS"
verifier "cumul d'un journal en CRLF" "$(cumul_couts)" 34.99
verifier "34,99 \$ : allumage autorisé" "$(reussit autoriser_allumage_g5 0)" oui
printf '2026-11-04;1;0.01\r\n' >>"$CSV_COUTS"
verifier "35,00 \$ : seuil atteint" "$(reussit seuil_g5_atteint "$(cumul_couts)")" oui
verifier "35,00 \$ : allumage refusé" "$(reussit autoriser_allumage_g5 0)" non
verifier "35,00 \$ : répétition de soutenance autorisée" "$(reussit autoriser_allumage_g5 1)" oui
CSV_COUTS="$TEMPORAIRE/au-dela.csv"
printf '%s\n2026-11-01;9000;51.00\n' "$ENTETE_COUTS" >"$CSV_COUTS"
verifier "au-delà du budget : allumage refusé" "$(reussit autoriser_allumage_g5 0)" non
CSV_COUTS="$TEMPORAIRE/absent.csv"
verifier "sans journal : allumage autorisé" "$(reussit autoriser_allumage_g5 0)" oui

printf 'G7 : rappel sur le poste 45 min avant la fin (AC7)\n'
verifier "échéance du rappel" "$(echeance_rappel "$FIN")" $((FIN - 2700))
verifier "apostrophe protégée pour sh" "$(sh -c "printf '%s' $(entre_apostrophes "l'apply « fini »")")" "l'apply « fini »"

APPELS="$TEMPORAIRE/powershell.txt"
powershell.exe() { printf '%s\n' "$@" >>"$APPELS"; }
decoder() { base64 -d | iconv -f UTF-16LE -t UTF-8; }
FIN_PROCHE=$(($(maintenant) + 2 * 3600))
: >"$APPELS"
verifier "Windows : rappel programmé" "$(reussit programmer_rappel_poste "$FIN_PROCHE")" oui
verifier "Windows : ancienne tâche retirée d'abord" "$(grep -c "Unregister-ScheduledTask -TaskName '$TACHE_RAPPEL'" "$APPELS")" 1
SCRIPT_PS="$(grep -A1 -x -- -EncodedCommand "$APPELS" | tail -n 1 | decoder)"
verifier "Windows : tâche planifiée de l'utilisateur" "$(grep -c "Register-ScheduledTask -TaskName '$TACHE_RAPPEL'" <<<"$SCRIPT_PS")" 1
verifier "Windows : déclenchement à T-45" "$(grep -c "FromUnixTimeSeconds($(echeance_rappel "$FIN_PROCHE"))" <<<"$SCRIPT_PS")" 1
verifier "Windows : aucun privilège demandé" "$(grep -Ec 'RunLevel|Highest|-User |Principal' <<<"$SCRIPT_PS")" 0
AFFICHAGE="$(sed -n "s/.*-EncodedCommand \([A-Za-z0-9+/=]*\)'.*/\1/p" <<<"$SCRIPT_PS" | decoder)"
verifier "Windows : message affiché" "$(grep -c 'tools/down.sh' <<<"$AFFICHAGE")" 1
: >"$APPELS"
verifier "Windows : échéance passée refusée" "$(reussit programmer_rappel_poste $(($(maintenant) + 30 * 60)))" non
verifier "Windows : rien d'enregistré si l'échéance est passée" "$(grep -c -- -EncodedCommand "$APPELS")" 0
: >"$APPELS"
annuler_rappel_poste
verifier "Windows : annulation" "$(grep -c "Unregister-ScheduledTask" "$APPELS")" 1
unset -f powershell.exe

FAUX_OUTILS="$TEMPORAIRE/outils"
mkdir -p "$FAUX_OUTILS"
cat >"$FAUX_OUTILS/at" <<FIN_AT
#!/bin/sh
if [ "\$1" = -c ]; then
  cat "$TEMPORAIRE/at-tache-\$2.sh"
  exit 0
fi
printf '%s\n' "\$*" >"$TEMPORAIRE/at-arguments.txt"
cat >"$TEMPORAIRE/at-script.sh"
FIN_AT
cat >"$FAUX_OUTILS/atq" <<FIN_ATQ
#!/bin/sh
printf '1\tMon Nov 23 18:15:00 2026 c poste\n2\tMon Nov 23 19:00:00 2026 c poste\n'
FIN_ATQ
cat >"$FAUX_OUTILS/atrm" <<FIN_ATRM
#!/bin/sh
printf '%s\n' "\$1" >>"$TEMPORAIRE/at-supprimees.txt"
FIN_ATRM
cat >"$FAUX_OUTILS/notify-send" <<FIN_NOTIFY
#!/bin/sh
printf '%s' "\$4" >"$TEMPORAIRE/notification.txt"
FIN_NOTIFY
chmod +x "$FAUX_OUTILS/at" "$FAUX_OUTILS/atq" "$FAUX_OUTILS/atrm" "$FAUX_OUTILS/notify-send"
SANS_POWERSHELL="$FAUX_OUTILS:$(printf '%s' "$PATH" | tr ':' '\n' | grep -v -i system32 | paste -sd:)"
if PATH="$SANS_POWERSHELL" command -v powershell.exe >/dev/null 2>&1; then
  printf 'ignoré at : powershell.exe hors des dossiers système\n'
else
  verifier "Linux : rappel programmé par at" "$(PATH="$SANS_POWERSHELL" reussit programmer_rappel_poste "$FIN_PROCHE")" oui
  verifier "Linux : file dédiée et heure locale de T-45" "$(cat "$TEMPORAIRE/at-arguments.txt")" \
    "-q $FILE_AT_RAPPEL -t $(date -d "@$(echeance_rappel "$FIN_PROCHE")" +%Y%m%d%H%M)"
  PATH="$FAUX_OUTILS:$PATH" sh "$TEMPORAIRE/at-script.sh"
  verifier "Linux : notification du message" "$(cat "$TEMPORAIRE/notification.txt")" "$(message_rappel "$FIN_PROCHE")"
  cp "$TEMPORAIRE/at-script.sh" "$TEMPORAIRE/at-tache-1.sh"
  printf 'notify-send autre "tâche étrangère à Collector"\n' >"$TEMPORAIRE/at-tache-2.sh"
  : >"$TEMPORAIRE/at-supprimees.txt"
  PATH="$SANS_POWERSHELL" annuler_rappel_poste
  verifier "Linux : seule la tâche du rappel Collector est annulée" "$(paste -sd' ' "$TEMPORAIRE/at-supprimees.txt")" 1
fi

printf 'G1 : armement, purge et piège de sortie de up.sh, Scheduler et Terraform simulés (AC1)\n'
APPELS_AWS="$TEMPORAIRE/aws.txt"
CREATIONS="$TEMPORAIRE/creations.txt"
GROUPE="$TEMPORAIRE/groupe"
JOURNAL="$TEMPORAIRE/journal.txt"
DOSSIER_SESSION="$TEMPORAIRE/session"
LAB_ROLE_ARN="arn:aws:iam::111122223333:role/LabRole"
ETAT_TF="$(etat "$CLUSTER" "$NODE_GROUP" "$RDS" "$ALB" "$NAT" "$EIP" "$VPC")"
aws() {
  printf '%s\n' "$2" >>"$APPELS_AWS"
  case "$2" in
    get-schedule-group) [ -f "$GROUPE" ] ;;
    delete-schedule-group) rm -f "$GROUPE" ;;
    create-schedule-group)
      if [ -f "$TEMPORAIRE/interrompre" ]; then
        rm -f "$TEMPORAIRE/interrompre"
        exit 130
      fi
      : >"$GROUPE"
      ;;
    create-schedule)
      printf '%s\n' "$*" >>"$CREATIONS"
      [ ! -f "$TEMPORAIRE/refuser" ]
      ;;
  esac
}
tf() { printf '%s' "$ETAT_TF"; }
sleep() { :; }
programmer_rappel_poste() { printf 'rappel %s\n' "$1" >>"$APPELS_AWS"; }
compter() { grep -cx -- "$1" "$APPELS_AWS"; }
reinitialiser() {
  : >"$APPELS_AWS"
  : >"$CREATIONS"
  : >"$GROUPE"
  : >"$JOURNAL"
  rm -rf "$DOSSIER_SESSION" "$TEMPORAIRE/refuser" "$TEMPORAIRE/interrompre"
  FIN_LAB=$(($(maintenant) + 3 * 3600))
}
sortie_apres_apply() {
  (
    APPLY_LANCE=1
    GARDE_FOU_TENTE="$2"
    set -euo pipefail
    trap armer_en_sortie EXIT
    exit "$1"
  )
}

reinitialiser
armer_garde_fou 2>"$JOURNAL"
verifier "état complet : armement réussi" "$?" 0
verifier "état complet : six planifications créées" "$(compter create-schedule)" 6
verifier "purge du groupe avant sa recréation" \
  "$(grep -E '^(delete|create)-schedule-group$' "$APPELS_AWS" | paste -sd' ')" "delete-schedule-group create-schedule-group"
verifier "planification lue en un seul appel : nom et échéance transmis" \
  "$(grep -c -- "--name collector-t30-nat --schedule-expression at($(utc_scheduler "$(echeance_lourde "$FIN_LAB")"))" "$CREATIONS")" 1
verifier "état complet : 6 sur 6 annoncées" "$(grep -c '6 planification(s) armée(s) sur 6 attendue(s)' "$JOURNAL")" 1

reinitialiser
ETAT_TF="$(etat "$VPC" "$NAT" "$EIP")"
armer_garde_fou 2>"$JOURNAL"
verifier "apply partiel : deux planifications créées" "$(compter create-schedule)" 2
verifier "apply partiel : 2 sur 2 annoncées, écart signalé" \
  "$(grep -c -e '2 planification(s) armée(s) sur 2 attendue(s)' -e '4 cible(s) absente(s)' "$JOURNAL")" 2
ETAT_TF="$(etat "$CLUSTER" "$NODE_GROUP" "$RDS" "$ALB" "$NAT" "$EIP" "$VPC")"

reinitialiser
: >"$TEMPORAIRE/refuser"
verifier "planifications refusées : armement en échec" "$(reussit armer_garde_fou)" non
verifier "planifications refusées : écart signalé" "$(armer_garde_fou 2>&1 | grep -c '6 planification(s) non armée(s)')" 1

reinitialiser
FIN_LAB=$(($(maintenant) + ECHEANCE_LOURDE_MIN * 60 + 60))
verifier "T-$ECHEANCE_LOURDE_MIN trop proche : armement refusé" "$(reussit armer_garde_fou)" non
verifier "T-$ECHEANCE_LOURDE_MIN trop proche : aucun appel Scheduler" "$(grep -c . "$APPELS_AWS")" 0

reinitialiser
ETAT_TF="$(etat "$VPC")"
armer_garde_fous 2>"$JOURNAL"
verifier "état sans ressource facturable : aucun appel Scheduler" "$(grep -vc '^rappel' "$APPELS_AWS")" 0
verifier "état sans ressource facturable : aucun rappel G7" "$(grep -c '^rappel' "$APPELS_AWS")" 0
ETAT_TF="$(etat "$CLUSTER" "$NODE_GROUP" "$RDS" "$ALB" "$NAT" "$EIP" "$VPC")"

reinitialiser
verifier "armement complet : tentative notée à la fin" "$(GARDE_FOU_TENTE=0; armer_garde_fous 2>/dev/null; printf '%s' "$GARDE_FOU_TENTE")" 1
verifier "armement complet : rappel G7 programmé une fois" "$(compter "rappel $FIN_LAB")" 1

reinitialiser
sortie_apres_apply 130 0 2>"$JOURNAL"
verifier "Ctrl-C après l'apply : code 130 conservé" "$?" 130
verifier "Ctrl-C après l'apply : armement par le piège" "$(compter create-schedule)" 6
verifier "Ctrl-C après l'apply : rappel G7 programmé" "$(compter "rappel $FIN_LAB")" 1

reinitialiser
sortie_apres_apply 1 1 2>/dev/null
verifier "sortie après un armement réussi : code conservé" "$?" 1
verifier "sortie après un armement réussi : pas de double armement" "$(grep -c . "$APPELS_AWS")" 0

reinitialiser
(
  APPLY_LANCE=0
  GARDE_FOU_TENTE=0
  trap armer_en_sortie EXIT
  exit 2
) 2>/dev/null
verifier "sortie avant l'apply : rien armé" "$(grep -c . "$APPELS_AWS")" 0

reinitialiser
(
  APPLY_LANCE=1
  GARDE_FOU_TENTE=0
  set -euo pipefail
  trap armer_en_sortie EXIT
  : >"$TEMPORAIRE/interrompre"
  armer_garde_fous
  exit 0
) 2>/dev/null
verifier "armement interrompu : code 130" "$?" 130
verifier "armement interrompu : rejoué par le piège" "$(compter create-schedule)" 6
verifier "armement interrompu : groupe recréé au second essai" "$(compter create-schedule-group)" 2

reinitialiser
sortie_apres_apply 129 0
verifier "terminal fermé : code 129 conservé" "$?" 129
verifier "terminal fermé : armement par le piège" "$(compter create-schedule)" 6
verifier "terminal fermé : trace écrite dans tools/.session" \
  "$(grep -c 'sortie de up.sh (code 129)' "$DOSSIER_SESSION/armement-sortie.log" 2>/dev/null)" 1

reinitialiser
sortie_apres_apply 130 0 2>&-
verifier "stderr fermé : code 130 conservé" "$?" 130
verifier "stderr fermé : armement par le piège" "$(compter create-schedule)" 6

if [ -w /dev/full ]; then
  reinitialiser
  sortie_apres_apply 130 0 2>/dev/full
  verifier "stderr en erreur d'écriture : code 130 conservé" "$?" 130
  verifier "stderr en erreur d'écriture : armement par le piège" "$(compter create-schedule)" 6
else
  printf 'ignoré /dev/full : absent sur ce poste, cas couvert par stderr fermé\n'
fi

[ "$ECHECS" -eq 0 ] || {
  printf '%s cas en échec\n' "$ECHECS"
  exit 1
}
printf 'Tous les cas passent.\n'
