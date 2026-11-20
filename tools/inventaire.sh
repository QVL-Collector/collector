#!/usr/bin/env bash
# Inventaire du compte (D11 G2, MI8), indépendant de l'état Terraform ; bloque le up suivant s'il reste quelque chose.
#   ./inventaire.sh            liste les ressources hors couche persistante
#   ./inventaire.sh --strict   idem, code retour 1 si une ressource subsiste ou si une lecture échoue
# Hors périmètre : couche persistante (zone, CAA et validation ACM, certificat, budget, bucket des photos,
# groupes de journaux persistants) et outillage du lab (Lambda, règles voc-*).

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

STRICT=0
case "${1:-}" in
  "") ;;
  --strict) STRICT=1 ;;
  *)
    sed -n '2,6p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
    exit 2
    ;;
esac

TROUVES=0

constat() {
  printf '  %-26s %s\n' "$1" "$2" | masquer
  TROUVES=$((TROUVES + 1))
}

verifier() {
  local categorie="$1" sortie ligne
  shift
  if ! sortie="$("$@" 2>/dev/null)"; then
    constat "$categorie" "lecture impossible"
    return 0
  fi
  while IFS= read -r ligne; do
    [ -z "$ligne" ] || constat "$categorie" "$ligne"
  done <<<"$sortie"
}

q_eks() { aws_json eks list-clusters | jq -r '.clusters[]'; }

q_ecs() {
  local arns
  arns="$(aws_json ecs list-clusters | jq -r '.clusterArns[]')"
  [ -n "$arns" ] || return 0
  # shellcheck disable=SC2086
  aws_json ecs describe-clusters --clusters $arns |
    jq -r '.clusters[] | select(.status != "INACTIVE") | "\(.clusterName) \(.status) services=\(.activeServicesCount) taches=\(.runningTasksCount)"'
}

q_ec2() {
  aws_json ec2 describe-instances \
    --filters "Name=instance-state-name,Values=pending,running,stopping,stopped,shutting-down" |
    jq -r '.Reservations[].Instances[] | "\(.InstanceId) \(.InstanceType) \(.State.Name)"'
}

q_nat() {
  aws_json ec2 describe-nat-gateways --filter "Name=state,Values=pending,available,deleting,failed" |
    jq -r '.NatGateways[] | "\(.NatGatewayId) \(.State)"'
}

q_eip() { aws_json ec2 describe-addresses | jq -r '.Addresses[] | "\(.AllocationId) associee=\(.AssociationId // "non")"'; }

q_elbv2() { aws_json elbv2 describe-load-balancers | jq -r '.LoadBalancers[] | "\(.LoadBalancerName) \(.Type) \(.Scheme)"'; }

q_elb() { aws_json elb describe-load-balancers | jq -r '.LoadBalancerDescriptions[] | "\(.LoadBalancerName) classique"'; }

q_tg() { aws_json elbv2 describe-target-groups | jq -r '.TargetGroups[].TargetGroupName'; }

q_rds() { aws_json rds describe-db-instances | jq -r '.DBInstances[] | "\(.DBInstanceIdentifier) \(.DBInstanceStatus)"'; }

q_rds_instantanes() { aws_json rds describe-db-snapshots | jq -r '.DBSnapshots[] | "\(.DBSnapshotIdentifier) \(.SnapshotType)"'; }

q_ebs() { aws_json ec2 describe-volumes | jq -r '.Volumes[] | "\(.VolumeId) \(.State) \(.Size)Gio"'; }

q_eni() {
  aws_json ec2 describe-network-interfaces --filters "Name=status,Values=available" |
    jq -r '.NetworkInterfaces[] | "\(.NetworkInterfaceId) \(.Description)"'
}

q_vpc() { aws_json ec2 describe-vpcs --filters "Name=is-default,Values=false" | jq -r '.Vpcs[] | "\(.VpcId) \(.CidrBlock)"'; }

q_sg() {
  aws_json ec2 describe-security-groups |
    jq -r '.SecurityGroups[] | select(.GroupName != "default") |
      "\(.GroupId) \(.GroupName)" +
      (if ([.IpPermissions[].IpRanges[]?.CidrIp] | index("0.0.0.0/0")) then " OUVERT-A-INTERNET" else "" end)'
}

q_secrets() { aws_json secretsmanager list-secrets | jq -r '.SecretList[].Name'; }

q_journaux() {
  aws_json logs describe-log-groups |
    jq -r --arg p "$GROUPES_JOURNAUX_PERSISTANTS" '($p | split(" ")) as $persistants | .logGroups[] |
      select(.logGroupName | test("^/(aws/eks|aws/ecs|ecs|collector|aws/rds)/|^/aws/lambda/collector-")) |
      select(.logGroupName as $n | $persistants | index($n) | not) |
      "\(.logGroupName) retention=\(.retentionInDays // "infinie")"'
}

q_lambda() { aws_json lambda list-functions | jq -r --arg p "$NOM-" '.Functions[] | select(.FunctionName | startswith($p)) | .FunctionName'; }

q_alarmes() { aws_json cloudwatch describe-alarms | jq -r '.MetricAlarms[] | select(.AlarmName | test("collector")) | .AlarmName'; }

q_planifications() {
  local groupe
  aws_json scheduler list-schedule-groups | jq -r '.ScheduleGroups[] | select(.Name != "default") | "groupe \(.Name)"'
  for groupe in $(aws_json scheduler list-schedule-groups | jq -r '.ScheduleGroups[].Name'); do
    aws_json scheduler list-schedules --group-name "$groupe" | jq -r '.Schedules[] | "\(.GroupName)/\(.Name) \(.State)"'
  done
}

q_dns() {
  local zone
  zone="$(aws_json route53 list-hosted-zones-by-name --dns-name "$DOMAINE" --max-items 1 |
    jq -r --arg d "$DOMAINE." '.HostedZones[] | select(.Name == $d) | .Id')"
  [ -n "$zone" ] || return 0
  aws_json route53 list-resource-record-sets --hosted-zone-id "$zone" |
    jq -r '.ResourceRecordSets[] | select(.Type != "NS" and .Type != "SOA" and .Type != "CAA") |
      select((.Type == "CNAME" and (.Name | startswith("_"))) | not) | "\(.Name) \(.Type)"'
}

q_buckets() {
  aws_json s3api list-buckets |
    jq -r --arg p "$NOM-" --arg photos "$BUCKET_PHOTOS" '.Buckets[].Name | select(startswith($p) and . != $photos)'
}

exiger_outils aws jq sed sha256sum
sonde_identite
charger_session

printf 'Inventaire du %s (région %s)\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$AWS_REGION"
if [ -n "${FIN_LAB:-}" ]; then
  printf 'Session %s : fin du lab à %s, reste %s min\n' "${SESSION_TF:-?}" "$(heure_locale "$FIN_LAB")" "$(((FIN_LAB - $(maintenant)) / 60))"
fi

verifier "eks:cluster" q_eks
verifier "ecs:cluster" q_ecs
verifier "ec2:instance" q_ec2
verifier "ec2:nat" q_nat
verifier "ec2:adresse-elastique" q_eip
verifier "elbv2:load-balancer" q_elbv2
verifier "elb:classique" q_elb
verifier "elbv2:groupe-cible" q_tg
verifier "rds:instance" q_rds
verifier "rds:instantane" q_rds_instantanes
verifier "ec2:volume" q_ebs
verifier "ec2:interface-disponible" q_eni
verifier "ec2:vpc" q_vpc
verifier "ec2:groupe-securite" q_sg
verifier "secretsmanager:secret" q_secrets
verifier "logs:groupe" q_journaux
verifier "lambda:fonction" q_lambda
verifier "cloudwatch:alarme" q_alarmes
verifier "scheduler" q_planifications
verifier "route53:enregistrement" q_dns
verifier "s3:bucket" q_buckets

if [ "$TROUVES" -eq 0 ]; then
  printf 'Inventaire vide hors couche persistante.\n'
  exit 0
fi

printf '%s ressource(s) ou lecture(s) en défaut.\n' "$TROUVES"
[ "$STRICT" -eq 0 ] || exit 1
