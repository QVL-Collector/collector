#!/usr/bin/env bash
# Bibliothèque commune des scripts de tools/ (D11). Sourcée, jamais exécutée seule.
# Aucun secret n'est lu, affiché ni journalisé ; les numéros de compte sont masqués à l'affichage (MI6).
# shellcheck disable=SC2034

set -euo pipefail

export MSYS_NO_PATHCONV=1
export AWS_PROFILE="${AWS_PROFILE:-collector-lab}"
export AWS_REGION="us-east-1"
export AWS_DEFAULT_REGION="$AWS_REGION"
export AWS_PAGER=""
export TF_IN_AUTOMATION=1

RACINE_TOOLS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && { pwd -W 2>/dev/null || pwd; })"
DEPOT="$(cd "$RACINE_TOOLS/.." && { pwd -W 2>/dev/null || pwd; })"
DOSSIER_TF="$DEPOT/infra/ephemere"
DOSSIER_PERSISTANT="$DEPOT/infra/persistant"
DOSSIER_SESSION="$RACINE_TOOLS/.session"
DOSSIER_SAUVEGARDES="$DEPOT/infra/.etat-sauvegardes"
FICHIER_SESSION="$DOSSIER_SESSION/session.env"
CSV_SESSIONS="$DEPOT/docs/mesures/sessions.csv"
CSV_COUTS="$DEPOT/docs/mesures/couts.csv"
DOSSIER_RAPPORTS="$DEPOT/docs/mesures/US-004"
DOSSIER_K8S="$DEPOT/infra/k8s"

NOM="collector"
DOMAINE="collector.qvl-project.com"
GROUPE_PLANIF="collector"
PREFIXE_SECRETS="collector/"
GROUPES_JOURNAUX_PERSISTANTS="/collector/app /collector/audit"
RETENTION_JOURNAL_APP=7
RETENTION_JOURNAL_AUDIT=30
SONDE_EXTERNE="collector-sonde-hors-plage"
VERSION_EKS="${VERSION_EKS:-1.35}"
DUREE_LAB_MIN=240
MARGE_UP_MIN=75
BUDGET_USD=50
SEUIL_G5_PCT=70
COUT_HORAIRE_USD=0.34
RAPPEL_AVANT_FIN_MIN=45
ECHEANCE_LOURDE_MIN=30
ECHEANCE_FINALE_MIN=10
NB_CIBLES_G1=6
TACHE_RAPPEL="Collector-rappel-fin-lab"
FILE_AT_RAPPEL="c"
ENTETE_COUTS='date;duree_allumee_min;cout_estime_usd'

masquer() { sed -E 's/[0-9]{12}/<compte>/g'; }

masquer_ip() { sed -E 's/[0-9]{12}/<compte>/g; s/([0-9]{1,3}\.){3}[0-9]{1,3}/<ip>/g'; }

jq() { command jq "$@" | tr -d '\r'; }

journal() { printf '[%s] %s\n' "$(date -u +%H:%M:%SZ)" "$*" | masquer >&2; }

echec() {
  journal "ERREUR : $*"
  exit 1
}

exiger_outils() {
  local outil
  for outil in "$@"; do
    command -v "$outil" >/dev/null 2>&1 || echec "outil absent du PATH : $outil (rouvrir le terminal après installation)"
  done
}

aws_json() { aws "$@" --output json | tr -d '\r'; }

tf() { terraform -chdir="$DOSSIER_TF" "$@"; }

tfp() { terraform -chdir="$DOSSIER_PERSISTANT" "$@"; }

chemin_natif() { if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi; }

NUL="$(if command -v cygpath >/dev/null 2>&1; then printf nul; else printf /dev/null; fi)"

maintenant() { date -u +%s; }

utc_scheduler() { date -u -d "@$1" +%Y-%m-%dT%H:%M:%S; }

heure_locale() { date -d "@$1" +%H:%M; }

nouvelle_session() { date -u +%Y%m%dT%H%MZ; }

cout_estime() { awk -v m="$1" -v h="$COUT_HORAIRE_USD" 'BEGIN { printf "%.2f", m / 60 * h }'; }

sonde_identite() {
  local identite arn region
  identite="$(aws_json sts get-caller-identity)" ||
    echec "identité AWS indisponible : profil $AWS_PROFILE à jour après « Start Lab » ?"
  arn="$(printf '%s' "$identite" | jq -r '.Arn')"
  case "$arn" in
    *":assumed-role/voclabs/"*) ;;
    *) echec "principal inattendu : $(printf '%s' "$arn" | masquer)" ;;
  esac
  COMPTE="$(printf '%s' "$identite" | jq -r '.Account')"
  region="$(aws_json ec2 describe-availability-zones | jq -r '.AvailabilityZones[0].RegionName')"
  [ "$region" = "us-east-1" ] || echec "région inattendue : $region"
  LAB_ROLE_ARN="arn:aws:iam::${COMPTE}:role/LabRole"
  BUCKET_PHOTOS="${NOM}-photos-$(printf '%s' "$COMPTE" | sha256sum | cut -c1-10)"
  journal "identité : rôle voclabs, compte masqué, région $region, profil $AWS_PROFILE"
}

ip_poste() {
  local ip
  ip="$(curl -fsS --max-time 10 https://checkip.amazonaws.com | tr -d '\r\n ')" || echec "adresse publique du poste introuvable"
  [[ "$ip" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] || echec "adresse publique du poste illisible"
  printf '%s/32' "$ip"
}

zone_collector() {
  aws_json route53 list-hosted-zones-by-name --dns-name "$DOMAINE" --max-items 1 |
    jq -r --arg d "$DOMAINE." '.HostedZones[] | select(.Name == $d) | .Id'
}

certificat_emis() {
  aws_json acm list-certificates --certificate-statuses ISSUED |
    jq -r --arg d "$DOMAINE" '.CertificateSummaryList[] | select(.DomainName == $d) | .CertificateArn' | head -n 1
}

supprimer_sonde_externe() {
  aws lambda delete-function --function-name "$SONDE_EXTERNE" >/dev/null 2>&1 || true
  aws logs delete-log-group --log-group-name "/aws/lambda/$SONDE_EXTERNE" >/dev/null 2>&1 || true
}

charger_session() {
  if [ -f "$FICHIER_SESSION" ]; then
    # shellcheck source=/dev/null
    . "$FICHIER_SESSION"
  fi
}

ecrire_session() {
  mkdir -p "$DOSSIER_SESSION"
  {
    printf 'SESSION_TF=%q\n' "$SESSION_TF"
    printf 'DEBUT_LAB=%q\n' "${DEBUT_LAB:-}"
    printf 'FIN_LAB=%q\n' "${FIN_LAB:-}"
    printf 'DEBUT_ALLUMAGE=%q\n' "${DEBUT_ALLUMAGE:-}"
    printf 'EXTINCTION_INCOMPLETE=%q\n' "${EXTINCTION_INCOMPLETE:-0}"
  } >"$FICHIER_SESSION"
}

etat_vide() { [ -z "$(tf state list 2>/dev/null || true)" ]; }

sauvegarder_etat() {
  local source="$1/terraform.tfstate"
  [ -f "$source" ] || return 0
  mkdir -p "$DOSSIER_SAUVEGARDES"
  cp "$source" "$DOSSIER_SAUVEGARDES/$2.tfstate.$(date -u +%Y%m%dT%H%M%SZ).$3"
}

phase() {
  [ "$#" -eq 4 ] || echec "phase : 4 arguments attendus (script etape duree_s commentaire)"
  mkdir -p "$(dirname "$CSV_SESSIONS")"
  [ -f "$CSV_SESSIONS" ] ||
    printf '%s\n' 'horodatage_utc;session;script;etape;duree_s;commentaire' >"$CSV_SESSIONS"
  printf '%s;%s;%s;%s;%s;%s\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "${SESSION_TF:-}" "$@" >>"$CSV_SESSIONS"
}

cumul_couts() {
  [ -f "$CSV_COUTS" ] || {
    printf '0.00'
    return 0
  }
  awk -F';' 'NR > 1 { s += $3 } END { printf "%.2f", s + 0 }' "$CSV_COUTS"
}

seuil_g5_usd() { awk -v b="$BUDGET_USD" -v p="$SEUIL_G5_PCT" 'BEGIN { printf "%.2f", b * p / 100 }'; }

seuil_g5_atteint() { awk -v c="$1" -v p="$(seuil_g5_usd)" 'BEGIN { exit !(c >= p) }'; }

autoriser_allumage_g5() {
  local cumul plafond
  cumul="$(cumul_couts)"
  plafond="$(seuil_g5_usd)"
  journal "G5 : cumul estimé $cumul \$, seuil d'arrêt $plafond \$ ($SEUIL_G5_PCT % de $BUDGET_USD \$)"
  seuil_g5_atteint "$cumul" || return 0
  if [ "${1:-0}" -eq 1 ]; then
    journal "G5 : seuil atteint, allumage autorisé pour une répétition de soutenance"
    return 0
  fi
  journal "G5 : seuil d'arrêt atteint, seules les répétitions de soutenance sont autorisées (--repetition-soutenance)"
  return 1
}

ligne_cout() { printf '%s;%s;%s\n' "$1" "$2" "$(cout_estime "$2")"; }

ajouter_cout() {
  local ligne
  mkdir -p "$(dirname "$CSV_COUTS")"
  [ -s "$CSV_COUTS" ] || printf '%s\n' "$ENTETE_COUTS" >"$CSV_COUTS"
  [ -z "$(tail -c 1 "$CSV_COUTS")" ] || printf '\n' >>"$CSV_COUTS"
  ligne="$(ligne_cout "$(date -u +%Y-%m-%d)" "$1")"
  printf '%s\n' "$ligne" >>"$CSV_COUTS"
  printf '%s' "${ligne##*;}"
}

echeance_lourde() { printf '%s' $(($1 - ECHEANCE_LOURDE_MIN * 60)); }

echeance_finale() { printf '%s' $(($1 - ECHEANCE_FINALE_MIN * 60)); }

ids_garde_fou() {
  jq -c '
    def valeur($adresse; $champ):
      [.values.root_module.resources[]? | select(.mode == "managed" and .address == $adresse) | .values[$champ] | strings] |
      first // "";
    {
      cluster: valeur("aws_eks_cluster.principal"; "name"),
      node_group: valeur("aws_eks_node_group.principal"; "node_group_name"),
      rds: valeur("aws_db_instance.principale"; "identifier"),
      alb: valeur("aws_lb.public"; "arn"),
      nat: valeur("aws_nat_gateway.unique"; "id"),
      eip: valeur("aws_eip.nat"; "allocation_id")
    }'
}

planifications_garde_fou() {
  jq -c --arg nom "$NOM" --argjson t30 "$(echeance_lourde "$1")" --argjson t10 "$(echeance_finale "$1")" '
    def p($suffixe; $echeance; $cible; $entree):
      {nom: "\($nom)-\($suffixe)", echeance: $echeance, cible: $cible, entree: $entree};
    (.cluster // "") as $cluster | (.node_group // "") as $node_group | (.rds // "") as $rds |
    (.alb // "") as $alb | (.nat // "") as $nat | (.eip // "") as $eip |
    (if $cluster != "" and $node_group != "" then
      p("t30-node-group"; $t30; "eks:deleteNodegroup"; {ClusterName: $cluster, NodegroupName: $node_group}) else empty end),
    (if $rds != "" then
      p("t30-rds"; $t30; "rds:deleteDBInstance"; {DbInstanceIdentifier: $rds, SkipFinalSnapshot: true, DeleteAutomatedBackups: true})
    else empty end),
    (if $alb != "" then p("t30-alb"; $t30; "elasticloadbalancingv2:deleteLoadBalancer"; {LoadBalancerArn: $alb}) else empty end),
    (if $nat != "" then p("t30-nat"; $t30; "ec2:deleteNatGateway"; {NatGatewayId: $nat}) else empty end),
    (if $cluster != "" then p("t10-cluster-eks"; $t10; "eks:deleteCluster"; {Name: $cluster}) else empty end),
    (if $eip != "" then p("t10-eip"; $t10; "ec2:releaseAddress"; {AllocationId: $eip}) else empty end)'
}

cible_planification() {
  jq -c --arg role "$1" '{
    Arn: "arn:aws:scheduler:::aws-sdk:\(.cible)",
    RoleArn: $role,
    Input: (.entree | tojson),
    RetryPolicy: {MaximumRetryAttempts: 3, MaximumEventAgeInSeconds: 900}
  }'
}

supprimer_planifications() {
  local essai
  aws scheduler get-schedule-group --name "$GROUPE_PLANIF" >/dev/null 2>&1 || return 0
  aws scheduler delete-schedule-group --name "$GROUPE_PLANIF" >/dev/null || return 1
  for essai in 1 2 3 4 5 6 7 8 9 10 11 12; do
    aws scheduler get-schedule-group --name "$GROUPE_PLANIF" >/dev/null 2>&1 || return 0
    [ "$essai" -eq 12 ] || sleep 5
  done
  return 1
}

planifier() {
  local planif="$1" nom echeance cible
  IFS=$'\t' read -r nom echeance cible < <(printf '%s' "$planif" | jq -r '[.nom, .echeance, .cible] | @tsv')
  if ! aws scheduler create-schedule \
    --group-name "$GROUPE_PLANIF" \
    --name "$nom" \
    --schedule-expression "at($(utc_scheduler "$echeance"))" \
    --schedule-expression-timezone UTC \
    --flexible-time-window Mode=OFF \
    --action-after-completion DELETE \
    --target "$(printf '%s' "$planif" | cible_planification "$LAB_ROLE_ARN")" >/dev/null; then
    journal "ATTENTION : planification $nom refusée ($cible)"
    return 1
  fi
  journal "planification $nom armée à $(heure_locale "$echeance") ($cible)"
}

armer_garde_fou() {
  local ids planifs planif essai armees=0
  G1_A_ARMER=0
  ids="$(tf show -json 2>/dev/null | ids_garde_fou || true)"
  [ -n "$ids" ] || ids='{}'
  planifs="$(printf '%s' "$ids" | planifications_garde_fou "$FIN_LAB")"
  if [ -z "$planifs" ]; then
    journal "G1 : aucune ressource facturable dans l'état, rien à armer"
    return 0
  fi
  G1_A_ARMER="$(grep -c . <<<"$planifs")"
  if [ "$(echeance_lourde "$FIN_LAB")" -le "$(($(maintenant) + 120))" ]; then
    journal "ATTENTION : échéance T-$ECHEANCE_LOURDE_MIN trop proche pour armer le garde-fou : lancer down.sh maintenant"
    return 1
  fi

  supprimer_planifications || journal "ATTENTION : groupe de planifications $GROUPE_PLANIF toujours en suppression"
  for essai in 1 2 3 4 5 6; do
    if aws scheduler create-schedule-group --name "$GROUPE_PLANIF" \
      --tags Key=projet,Value=collector Key=couche,Value=ephemere >/dev/null 2>&1; then
      break
    fi
    if [ "$essai" -eq 6 ]; then
      journal "ATTENTION : groupe de planifications $GROUPE_PLANIF non recréé"
      return 1
    fi
    sleep 10
  done

  while IFS= read -r planif; do
    if planifier "$planif"; then armees=$((armees + 1)); fi
  done <<<"$planifs"
  journal "G1 : $armees planification(s) armée(s) sur $G1_A_ARMER attendue(s) d'après l'état ($NB_CIBLES_G1 cibles facturables au plus : node group, RDS, ALB, NAT, cluster, adresse)"
  [ "$G1_A_ARMER" -eq "$NB_CIBLES_G1" ] ||
    journal "G1 : $((NB_CIBLES_G1 - G1_A_ARMER)) cible(s) absente(s) de l'état (apply partiel), rien à détruire pour elle(s)"
  [ "$armees" -eq "$G1_A_ARMER" ] || {
    journal "ATTENTION : $((G1_A_ARMER - armees)) planification(s) non armée(s)"
    return 1
  }
}

programmer_rappel_session() {
  if [ "${G1_A_ARMER:-0}" -eq 0 ]; then
    journal "G7 : aucune ressource facturable dans l'état, aucun rappel programmé sur le poste"
    return 0
  fi
  programmer_rappel_poste "$FIN_LAB" || journal "ATTENTION : rappel G7 non programmé sur le poste"
}

armer_garde_fous() {
  journal "G1 : armement de la destruction programmée (T-$ECHEANCE_LOURDE_MIN, T-$ECHEANCE_FINALE_MIN)"
  armer_garde_fou || journal "ATTENTION : garde-fou incomplet, surveiller l'échéance à la main"
  programmer_rappel_session
  GARDE_FOU_TENTE=1
}

armer_en_sortie() {
  local statut=$?
  trap - EXIT
  set +e
  if [ "${APPLY_LANCE:-0}" -eq 1 ] && [ "${GARDE_FOU_TENTE:-0}" -eq 0 ]; then
    if [ "$statut" -eq 129 ] && mkdir -p "$DOSSIER_SESSION" 2>/dev/null; then
      exec 2>>"$DOSSIER_SESSION/armement-sortie.log"
    fi
    journal "G1 : sortie de up.sh (code $statut) avant l'armement, armement sur ce qui a été créé" || true
    armer_garde_fous || true
  fi
  exit "$statut"
}

echeance_rappel() { printf '%s' $(($1 - RAPPEL_AVANT_FIN_MIN * 60)); }

message_rappel() {
  printf 'Collector : lancer bash tools/down.sh maintenant. Destruction programmée à %s, fin du lab à %s : down.sh doit finir avant « End Lab ».' \
    "$(heure_locale "$(echeance_lourde "$1")")" "$(heure_locale "$1")"
}

encoder_powershell() { iconv -f UTF-8 -t UTF-16LE | base64 | tr -d '\r\n'; }

entre_apostrophes() { printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"; }

script_rappel_windows() {
  local q="'" message affichage
  message="${2//$q/$q$q}"
  affichage="$(printf '%s' "Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show('$message', 'Collector', 'OK', 'Warning', 'OK', 'ServiceNotification') | Out-Null" |
    encoder_powershell)"
  printf '%s\n' \
    "\$ErrorActionPreference = 'Stop'" \
    "\$action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument '-NoProfile -NonInteractive -WindowStyle Hidden -EncodedCommand $affichage'" \
    "\$declencheur = New-ScheduledTaskTrigger -Once -At ([DateTimeOffset]::FromUnixTimeSeconds($1).LocalDateTime)" \
    "\$reglages = New-ScheduledTaskSettingsSet -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries" \
    "Register-ScheduledTask -TaskName '$TACHE_RAPPEL' -Action \$action -Trigger \$declencheur -Settings \$reglages -Force | Out-Null"
}

script_rappel_at() {
  printf ': %s\n' "$TACHE_RAPPEL"
  printf 'm=%s\n' "$(entre_apostrophes "$1")"
  # shellcheck disable=SC2016
  printf '%s\n' 'notify-send -u critical Collector "$m" 2>/dev/null || printf "%s\n" "$m" | wall 2>/dev/null'
}

annuler_rappel_poste() {
  local tache
  if command -v powershell.exe >/dev/null 2>&1; then
    powershell.exe -NoProfile -NonInteractive -Command \
      "Unregister-ScheduledTask -TaskName '$TACHE_RAPPEL' -Confirm:\$false -ErrorAction SilentlyContinue" >/dev/null 2>&1 || true
  elif command -v atq >/dev/null 2>&1 && command -v atrm >/dev/null 2>&1; then
    atq -q "$FILE_AT_RAPPEL" 2>/dev/null | awk '{ print $1 }' | while read -r tache; do
      if at -c "$tache" 2>/dev/null | grep -qxF ": $TACHE_RAPPEL"; then atrm "$tache" 2>/dev/null || true; fi
    done
  fi
  return 0
}

programmer_rappel_poste() {
  local fin="$1" echeance message
  echeance="$(echeance_rappel "$fin")"
  annuler_rappel_poste
  if [ "$echeance" -le "$(($(maintenant) + 60))" ]; then
    journal "G7 : échéance du rappel ($(heure_locale "$echeance")) déjà passée : lancer down.sh maintenant"
    return 1
  fi
  message="$(message_rappel "$fin")"
  if command -v powershell.exe >/dev/null 2>&1; then
    powershell.exe -NoProfile -NonInteractive -EncodedCommand "$(script_rappel_windows "$echeance" "$message" | encoder_powershell)" >/dev/null ||
      return 1
  elif command -v at >/dev/null 2>&1; then
    script_rappel_at "$message" | at -q "$FILE_AT_RAPPEL" -t "$(date -d "@$echeance" +%Y%m%d%H%M)" >/dev/null 2>&1 || return 1
  else
    journal "G7 : ni PowerShell ni at sur le poste, rappel non programmé"
    return 1
  fi
  journal "G7 : rappel programmé sur le poste à $(heure_locale "$echeance") ($RAPPEL_AVANT_FIN_MIN min avant la fin du lab)"
}

variables_tf() {
  export TF_VAR_session="$SESSION_TF"
  export TF_VAR_ip_admin="$1"
}

ALB_FERME=(-var=alb_ouvert=false -var=preuve_controle=)

k() {
  kubectl --context "$NOM" --request-timeout=20s "$@" | tr -d '\r'
}

serie_etat() {
  local source="$DOSSIER_TF/terraform.tfstate"
  [ -f "$source" ] || {
    printf '0'
    return 0
  }
  jq -r '.serial' "$source"
}
