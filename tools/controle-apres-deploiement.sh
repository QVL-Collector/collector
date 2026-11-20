#!/usr/bin/env bash
# Contrôle après déploiement (US-004 AC7 à AC11), rejoué à la fin de chaque up.sh et par alb.sh :
#   TLS   (AC7, XE3)  redirection 80 vers 443, TLS 1.0 et 1.1 refusés, TLS 1.2 accepté, certificat valide, HSTS ;
#   X17   (AC8, XE4)  adresses publiques du VPC balayées, API, nœuds et base injoignables hors plage (sonde Lambda),
#                     un seul load balancer, aucun Service LoadBalancer, règles de l'ALB, liste blanche, sortie sans IMDS ;
#   X20   (AC9)       valeurs des secrets de session cherchées dans les états, releases, objets du cluster, journaux et CI ;
#                     secrets montés par pod, analyse des images, de l'IaC et de l'AMI.
#   ./controle-apres-deploiement.sh
# Verdict : tools/.session/controle-<session>.json (lu par alb.sh, AC10) ; rapports masqués dans docs/mesures/US-004/.
# Aucune valeur de secret n'est affichée ni écrite : seuls des compteurs d'occurrences le sont.

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

[ "$#" -eq 0 ] || {
  sed -n '8p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

PORTS_BALAYES="21 22 23 25 53 80 110 143 443 445 1433 3306 3389 5432 5672 6379 6443 8080 8443 9000 9090 10250 15672 30080 31000"
NAMESPACES_SYSTEME="kube-system kube-public kube-node-lease"
HSTS_ATTENDU="max-age=31536000; includeSubDomains"
CHEMINS_INTERDITS=(
  "/reserver" "/api/interne/reserver" "/admin/" "/admin/master/console/" "/realms/master/"
  "/realms/master/protocol/openid-connect/certs" "/realms/collector/account/" "/realms/collector/../master/"
  "/realms/collector/%2e%2e/master/" "/%2e%2e/admin/" "/%2E%2E/admin/" "/realms%2fmaster/" "/realms%2Fmaster/"
  "/ADMIN/" "/Realms/Master/" "/resources/../admin/" "/resources/..%2f..%2fadmin/"
  "/realms/collector/protocol/openid-connect/%2e%2e/%2e%2e/%2e%2e/master/"
  "/realms/collector/broker/" "/metrics" "/health/ready" "/.env" "/.git/config" "/server-status"
)

RES_TSV=""
TRAVAIL=""

resultat() {
  local groupe="$1" controle="$2" statut="$3" detail="$4"
  printf '%s\t%s\t%s\t%s\n' "$groupe" "$controle" "$statut" "$detail" >>"$RES_TSV"
  printf '  [%-4s] %-5s %-28s %s\n' "$statut" "$groupe" "$controle" "$detail" | masquer_ip >&2
}

sortie() { printf '%s' "$SORTIES" | jq -r --arg c "$1" '.[$c].value // empty | if type == "array" then .[] else . end'; }

nettoyer() {
  supprimer_sonde_externe
  [ -z "$TRAVAIL" ] || rm -rf "$TRAVAIL"
}

poignee_tls() {
  local option="$1" sortie_ssl
  sortie_ssl="$(timeout 20 openssl s_client -connect "$DOMAINE:443" -servername "$DOMAINE" "$option" \
    -cipher 'DEFAULT:@SECLEVEL=0' </dev/null 2>&1 || true)"
  printf '%s\n' "$sortie_ssl" | grep -E '^(New|Protocol|    Protocol|    Cipher)' >>"$TRAVAIL/tls.txt" || true
  if printf '%s' "$sortie_ssl" | grep -q 'Cipher is (NONE)'; then
    printf 'refus'
  elif printf '%s' "$sortie_ssl" | grep -qE '^New, TLSv1'; then
    printf 'accepte'
  else
    printf 'refus'
  fi
}

controler_tls() {
  local code redirection entetes version attendu option etat certificat
  printf 'TLS (AC7)\n' >&2
  printf 'Rapport TLS de %s, session %s, %s\n\n' "$DOMAINE" "$SESSION_TF" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >"$TRAVAIL/tls.txt"

  read -r code redirection < <(curl -s -o "$NUL" --max-time 10 -w '%{http_code} %{redirect_url}\n' \
    "http://$DOMAINE/controle?x=1" || printf '000 -\n')
  redirection="${redirection%$'\r'}"
  printf 'HTTP : %s -> %s\n' "$code" "$redirection" >>"$TRAVAIL/tls.txt"
  if [ "$code" = "301" ] && { [ "$redirection" = "https://$DOMAINE/controle?x=1" ] || [ "$redirection" = "https://$DOMAINE:443/controle?x=1" ]; }; then
    resultat TLS redirection-80 ok "301 vers https"
  else
    resultat TLS redirection-80 ko "code $code vers $redirection"
  fi

  for version in tls1 tls1_1 tls1_2; do
    option="-$version"
    attendu="refus"
    [ "$version" != "tls1_2" ] || attendu="accepte"
    if [ -z "$(openssl ciphers -s "$option" 'DEFAULT:@SECLEVEL=0' 2>/dev/null)" ]; then
      resultat TLS "$version" nv "client openssl incapable de proposer $version : refus non prouvé"
      continue
    fi
    printf '\n%s :\n' "$version" >>"$TRAVAIL/tls.txt"
    etat="$(poignee_tls "$option")"
    if [ "$etat" = "$attendu" ]; then
      resultat TLS "$version" ok "$etat"
    else
      resultat TLS "$version" ko "$etat, attendu $attendu"
    fi
  done

  certificat="$(timeout 20 openssl s_client -connect "$DOMAINE:443" -servername "$DOMAINE" </dev/null 2>/dev/null |
    openssl x509 -noout -subject -issuer -enddate -ext subjectAltName 2>/dev/null || true)"
  printf '\nCertificat :\n%s\n' "$certificat" >>"$TRAVAIL/tls.txt"
  if curl -sS -o "$NUL" --max-time 10 "https://$DOMAINE/" 2>/dev/null &&
    timeout 20 openssl s_client -connect "$DOMAINE:443" -servername "$DOMAINE" </dev/null 2>/dev/null |
    openssl x509 -noout -checkend 2592000 >/dev/null 2>&1; then
    resultat TLS certificat ok "chaîne valide, expiration à plus de 30 jours"
  else
    resultat TLS certificat ko "chaîne invalide ou expiration sous 30 jours"
  fi

  entetes="$(curl -s -D - -o "$NUL" --max-time 10 "https://$DOMAINE/" | tr -d '\r' || true)"
  printf '\nEn-têtes de https://%s/ :\n%s\n' "$DOMAINE" "$entetes" >>"$TRAVAIL/tls.txt"
  if awk -v e="strict-transport-security: $HSTS_ATTENDU" 'tolower($0) == tolower(e) { t = 1 } END { exit !t }' <<<"$entetes"; then
    resultat TLS hsts ok "$HSTS_ATTENDU"
  else
    resultat TLS hsts ko "en-tête absent ou différent de « $HSTS_ATTENDU »"
  fi
  if curl -s -D - -o "$NUL" --max-time 10 "https://auth.$DOMAINE/controle-404" | tr -d '\r' |
    grep -qi '^strict-transport-security:'; then
    resultat TLS hsts-404 ok "présent sur la réponse fixe 404"
  else
    resultat TLS hsts-404 info "absent sur la réponse fixe 404 (non bloquant, D08)"
  fi

  if command -v testssl.sh >/dev/null 2>&1; then
    printf '\ntestssl.sh :\n' >>"$TRAVAIL/tls.txt"
    testssl.sh --quiet --color 0 --protocols --server-defaults --headers "$DOMAINE" >>"$TRAVAIL/tls.txt" 2>&1 || true
  fi
  mkdir -p "$DOSSIER_RAPPORTS"
  masquer_ip <"$TRAVAIL/tls.txt" >"$DOSSIER_RAPPORTS/tls-$SESSION_TF.txt"
}

port_ouvert() { timeout 3 bash -c 'exec 3<>"/dev/tcp/$1/$2"' _ "$1" "$2" >/dev/null 2>&1; }

balayer() {
  local ip="$1" port ouverts=""
  for port in $PORTS_BALAYES $NODEPORT; do
    if port_ouvert "$ip" "$port"; then ouverts="$ouverts $port"; fi
  done
  printf '%s\n' "${ouverts# }" >"$TRAVAIL/balayage-$ip"
}

controler_adresses_publiques() {
  local adresses ip type description inattendues=0 ouverts code
  adresses="$(aws_json ec2 describe-network-interfaces --filters "Name=vpc-id,Values=$VPC" |
    jq -r '.NetworkInterfaces[] | select(.Association.PublicIp) |
      "\(.Association.PublicIp) \(.InterfaceType) \(.Description | gsub(" "; "_"))"')"
  IPS_ALB=()
  IPS_NAT=()
  while read -r ip type description; do
    [ -n "$ip" ] || continue
    if [[ "$description" == "ELB_app/$NOM_ALB/"* ]]; then
      IPS_ALB+=("$ip")
    elif [ "$type" = "nat_gateway" ]; then
      IPS_NAT+=("$ip")
    else
      resultat X17 adresse-publique ko "adresse inattendue sur une interface $type ($description)"
      inattendues=$((inattendues + 1))
    fi
  done <<<"$adresses"
  [ "$inattendues" -gt 0 ] ||
    resultat X17 adresses-publiques ok "${#IPS_ALB[@]} de l'ALB, ${#IPS_NAT[@]} de la NAT, aucune autre"
  [ "${#IPS_ALB[@]}" -ge 2 ] ||
    resultat X17 balayage-alb ko "${#IPS_ALB[@]} adresse(s) de l'ALB identifiée(s), 2 attendues (une par zone)"
  [ "${#IPS_NAT[@]}" -eq 1 ] ||
    resultat X17 balayage-nat ko "${#IPS_NAT[@]} adresse(s) de NAT identifiée(s), 1 attendue"

  for ip in "${IPS_ALB[@]}" "${IPS_NAT[@]}"; do balayer "$ip" & done
  wait
  for ip in "${IPS_ALB[@]}"; do
    ouverts="$(cat "$TRAVAIL/balayage-$ip")"
    if [ "$ouverts" = "80 443" ]; then
      code="$(curl -s -o "$NUL" --max-time 10 -w '%{http_code}' -H "Host: $DOMAINE" "http://$ip/" || true)"
      if [ "$code" = "301" ]; then
        resultat X17 balayage-alb ok "$ip : seuls 443 et 80 (301) répondent"
      else
        resultat X17 balayage-alb ko "$ip : 80 répond $code au lieu de 301"
      fi
    else
      resultat X17 balayage-alb ko "$ip : ports ouverts « $ouverts », attendus « 80 443 »"
    fi
  done
  for ip in "${IPS_NAT[@]}"; do
    ouverts="$(cat "$TRAVAIL/balayage-$ip")"
    if [ -z "$ouverts" ]; then
      resultat X17 balayage-nat ok "$ip : aucun port ouvert"
    else
      resultat X17 balayage-nat ko "$ip : ports ouverts « $ouverts »"
    fi
  done
}

controler_configuration_reseau() {
  local cidrs prive public_rds instances
  cidrs="$(aws_json eks describe-cluster --name "$NOM" | jq -r '.cluster.resourcesVpcConfig | "\(.publicAccessCidrs | sort | join(",")) \(.endpointPrivateAccess)"')"
  if [ "$cidrs" = "$IP_POSTE true" ]; then
    resultat X17 api-plage ok "publicAccessCidrs limité au /32 du poste, point privé actif (MS1)"
  else
    prive="${cidrs##* }"
    resultat X17 api-plage ko "publicAccessCidrs « ${cidrs%% *} », point privé $prive"
  fi
  public_rds="$(aws_json rds describe-db-instances --db-instance-identifier "$ID_RDS" | jq -r '.DBInstances[0].PubliclyAccessible')"
  if [ "$public_rds" = "false" ]; then
    resultat X17 rds-privee ok "PubliclyAccessible false"
  else
    resultat X17 rds-privee ko "PubliclyAccessible $public_rds"
  fi
  instances="$(aws_json ec2 describe-instances --filters "Name=vpc-id,Values=$VPC" "Name=instance-state-name,Values=pending,running" |
    jq -r '[.Reservations[].Instances[] | select(.PublicIpAddress)] | length')"
  if [ "$instances" -eq 0 ]; then
    resultat X17 noeuds-prives ok "aucune instance avec adresse publique"
  else
    resultat X17 noeuds-prives ko "$instances instance(s) avec adresse publique"
  fi
}

controler_hors_plage() {
  local archive="$TRAVAIL/sonde.zip" cibles reponse ip ligne nom etat attendu interpreteur="" candidat
  for candidat in python3 python; do
    if "$candidat" -c 'import zipfile' >/dev/null 2>&1; then
      interpreteur="$candidat"
      break
    fi
  done
  [ -n "$interpreteur" ] || {
    resultat X17 hors-plage nv "python3 ou python absent : archive de la sonde Lambda non construite"
    return 0
  }
  "$interpreteur" -c 'import sys, zipfile; z = zipfile.ZipFile(sys.argv[1], "w"); z.write(sys.argv[2], "sonde.py"); z.close()' \
    "$archive" "$(chemin_natif "$RACINE_TOOLS/sonde-hors-plage/sonde.py")" || {
    resultat X17 hors-plage nv "archive de la sonde non construite"
    return 0
  }
  supprimer_sonde_externe
  if ! aws lambda create-function --function-name "$SONDE_EXTERNE" --runtime python3.13 --handler sonde.handler \
    --role "$LAB_ROLE_ARN" --zip-file "fileb://$archive" --timeout 90 \
    --tags projet=collector,couche=ephemere,outil=controle >/dev/null ||
    ! aws lambda wait function-active-v2 --function-name "$SONDE_EXTERNE"; then
    resultat X17 hors-plage nv "sonde Lambda non créée : kubectl hors plage non prouvé"
    return 0
  fi

  cibles="$(jq -nc --arg api "${POINT_API#https://}" --arg rds "$ADRESSE_RDS" \
    '[{nom: "api-eks", hote: $api, port: 443}, {nom: "rds", hote: $rds, port: 5432}]')"
  for ip in "${IPS_ALB[@]}"; do
    cibles="$(printf '%s' "$cibles" | jq -c --arg ip "$ip" '. + [{nom: "alb-443-\($ip)", hote: $ip, port: 443}, {nom: "alb-80-\($ip)", hote: $ip, port: 80}]')"
  done
  for ip in "${IPS_NAT[@]}"; do
    cibles="$(printf '%s' "$cibles" | jq -c --arg ip "$ip" '. + [{nom: "nat-22-\($ip)", hote: $ip, port: 22}]')"
  done

  aws lambda invoke --function-name "$SONDE_EXTERNE" --cli-binary-format raw-in-base64-out \
    --payload "$(jq -nc --argjson c "$cibles" '{cibles: $c}')" "$TRAVAIL/sonde.json" >/dev/null ||
    printf '{}' >"$TRAVAIL/sonde.json"
  reponse="$(jq -r '.resultats[]? | "\(.nom) \(.etat)"' "$TRAVAIL/sonde.json")"
  [ -n "$reponse" ] || {
    resultat X17 hors-plage ko "sonde Lambda sans réponse exploitable"
    return 0
  }
  while read -r nom etat; do
    attendu="ferme"
    if [[ "$nom" == alb-* ]] && [ "$ALB_OUVERT" = "true" ]; then attendu="ouvert"; fi
    ligne="$nom : $etat depuis une adresse AWS hors plage"
    if [ "$attendu" = "ouvert" ] && [ "$etat" = "ouvert" ]; then
      resultat X17 hors-plage ok "$ligne (ALB ouvert)"
    elif [ "$attendu" = "ferme" ] && [ "$etat" != "ouvert" ]; then
      resultat X17 hors-plage ok "$ligne"
    else
      resultat X17 hors-plage ko "$ligne, attendu $attendu"
    fi
  done <<<"$reponse"
  supprimer_sonde_externe
}

controler_load_balancers() {
  local alb classiques
  alb="$(aws_json elbv2 describe-load-balancers | jq -r '[.LoadBalancers[].LoadBalancerName] | sort | join(" ")')"
  classiques="$(aws_json elb describe-load-balancers | jq -r '.LoadBalancerDescriptions | length')"
  if [ "$alb" = "$NOM_ALB" ] && [ "$classiques" -eq 0 ]; then
    resultat X17 load-balancers ok "un seul : $NOM_ALB"
  else
    resultat X17 load-balancers ko "ELBv2 « $alb », classiques $classiques"
  fi
}

controler_services() {
  local services lb nodeports
  services="$(k get svc -A -o json)"
  lb="$(printf '%s' "$services" | jq -r '[.items[] | select(.spec.type == "LoadBalancer") | "\(.metadata.namespace)/\(.metadata.name)"] | join(" ")')"
  if [ -z "$lb" ]; then
    resultat X17 services-lb ok "aucun Service LoadBalancer"
  else
    resultat X17 services-lb ko "Service(s) LoadBalancer : $lb"
  fi
  nodeports="$(printf '%s' "$services" | jq -r --argjson n "$NODEPORT" '[.items[] | select(.spec.type == "NodePort" or .spec.type == "LoadBalancer") |
    .spec.ports[] | select(.nodePort != $n) | .nodePort] | join(" ")')"
  if [ -z "$nodeports" ]; then
    resultat X17 nodeports ok "seul le NodePort $NODEPORT du front est publié (MS5)"
  else
    resultat X17 nodeports ko "NodePorts hors liste : $nodeports"
  fi
}

controler_regles_alb() {
  local ecouteurs http https regles attendues entrees attendu_cidr
  ecouteurs="$(aws_json elbv2 describe-listeners --load-balancer-arn "$ARN_ALB" | jq -r '[.Listeners[].Port] | sort | join(" ")')"
  if [ "$ecouteurs" = "80 443" ]; then
    resultat X17 ecouteurs ok "80 et 443"
  else
    resultat X17 ecouteurs ko "ports « $ecouteurs »"
  fi

  http="$(aws_json elbv2 describe-rules --listener-arn "$ARN_HTTP" | jq -r '[.Rules[] | select(.IsDefault) | .Actions[0] |
    "\(.Type) \(.RedirectConfig.Protocol) \(.RedirectConfig.Port) \(.RedirectConfig.StatusCode)"] + [.Rules[] | select(.IsDefault | not) | "regle"] | join(";")')"
  if [ "$http" = "redirect HTTPS 443 HTTP_301" ]; then
    resultat X17 ecouteur-80 ok "redirection 301 seule"
  else
    resultat X17 ecouteur-80 ko "$http"
  fi

  https="$(aws_json elbv2 describe-listeners --listener-arns "$ARN_HTTPS" | jq -r '.Listeners[0] | "\(.SslPolicy) \(.DefaultActions[0].Type) \(.DefaultActions[0].FixedResponseConfig.StatusCode)"')"
  if [[ "$https" =~ ^ELBSecurityPolicy-TLS13-1-2-.*\ fixed-response\ 404$ ]]; then
    resultat X17 ecouteur-443 ok "TLS 1.2 minimum, 404 par défaut"
  else
    resultat X17 ecouteur-443 ko "$https"
  fi

  regles="$(aws_json elbv2 describe-rules --listener-arn "$ARN_HTTPS" | jq -r '[.Rules[] | select(.IsDefault | not) | .RuleArn] | sort | join(" ")')"
  attendues="$(sortie arns_regles_https | LC_ALL=C sort | tr '\n' ' ')"
  if [ "$regles " = "$attendues" ]; then
    resultat X17 regles-alb ok "seules les règles de l'IaC (liste blanche)"
  else
    resultat X17 regles-alb ko "règles hors IaC sur l'écouteur 443"
  fi

  attendu_cidr="$IP_POSTE"
  [ "$ALB_OUVERT" != "true" ] || attendu_cidr="0.0.0.0/0"
  entrees="$(aws_json ec2 describe-security-group-rules --filters "Name=group-id,Values=$SG_ALB" |
    jq -r '[.SecurityGroupRules[] | select(.IsEgress | not) | "\(.FromPort)-\(.ToPort)-\(.CidrIpv4 // .ReferencedGroupInfo.GroupId)"] | sort | join(" ")')"
  if [ "$entrees" = "443-443-$attendu_cidr 80-80-$attendu_cidr" ]; then
    resultat X17 groupe-alb ok "80 et 443 depuis $( [ "$ALB_OUVERT" = "true" ] && printf 'Internet' || printf 'le poste seul') (ME5)"
  else
    resultat X17 groupe-alb ko "entrées « $entrees »"
  fi
}

code_et_empreinte() {
  local hote="$1" chemin="$2" corps="$TRAVAIL/corps"
  local code
  code="$(curl -s --path-as-is --max-time 10 -o "$corps" -w '%{http_code}' -H "Host: $hote" "https://$DOMAINE$chemin" || printf '000')"
  printf '%s %s' "$code" "$(sha256sum <"$corps" | cut -c1-64)"
}

controler_liste_blanche() {
  local reference chemin hote code empreinte fautes=0
  reference="$(code_et_empreinte "$DOMAINE" /)"
  [ "${reference%% *}" = "200" ] || {
    resultat XE4 front ko "la page d'accueil répond ${reference%% *}"
    return 0
  }
  for chemin in "${CHEMINS_INTERDITS[@]}"; do
    read -r code empreinte <<<"$(code_et_empreinte "$DOMAINE" "$chemin")"
    if [ "$code" != "404" ] && [ "$code" != "400" ] && ! { [ "$code" = "200" ] && [ "$empreinte" = "${reference#* }" ]; }; then
      resultat XE4 front ko "$chemin : $code (ni 404, ni 400, ni la page statique du front)"
      fautes=$((fautes + 1))
    fi
    for hote in "auth.$DOMAINE" "$DNS_ALB" "inconnu.$DOMAINE"; do
      read -r code empreinte <<<"$(code_et_empreinte "$hote" "$chemin")"
      if [ "$code" != "404" ] && [ "$code" != "400" ]; then
        resultat XE4 hotes ko "$hote$chemin : $code au lieu de 404 (ou 400 de l'ALB, XP03)"
        fautes=$((fautes + 1))
      fi
    done
  done
  [ "$fautes" -gt 0 ] ||
    resultat XE4 liste-blanche ok "${#CHEMINS_INTERDITS[@]} chemins interdits : 404 hors front ; page statique, 400 ou 404 sur le front"
}

controler_sortie_imds() {
  local manquants
  manquants="$(k get ns -o json | jq -r --arg s "$NAMESPACES_SYSTEME" '($s | split(" ")) as $sys | .items[].metadata.name |
    select(. as $n | $sys | index($n) | not)' |
    while read -r ns; do
      k get networkpolicy -n "$ns" -o json | jq -e -f "$RACINE_TOOLS/sortie-sans-imds.jq" >/dev/null || printf '%s ' "$ns"
    done)"
  if [ -z "$manquants" ]; then
    resultat X17 sortie-sans-imds ok "toute sortie exclut l'IMDS dans chaque namespace applicatif (union des politiques)"
  else
    resultat X17 sortie-sans-imds ko "namespaces dont les politiques laissent joindre l'IMDS : $manquants"
  fi
}

collecter_valeurs() {
  local noms arn_rds nom brut
  VALEURS=()
  noms="$(aws_json secretsmanager list-secrets | jq -r --arg p "$PREFIXE_SECRETS" '.SecretList[] | select(.Name | startswith($p)) | .ARN')"
  arn_rds="$(aws_json rds describe-db-instances --db-instance-identifier "$ID_RDS" | jq -r '.DBInstances[0].MasterUserSecret.SecretArn // empty')"
  for nom in $noms $arn_rds; do
    brut="$(aws secretsmanager get-secret-value --secret-id "$nom" --query SecretString --output text | tr -d '\r')"
    if printf '%s' "$brut" | command jq -e 'type == "object"' >/dev/null 2>&1; then
      mapfile -t -O "${#VALEURS[@]}" VALEURS < <(printf '%s' "$brut" |
        jq -r 'to_entries[] | select(.key | test("pass|secret|token|key"; "i")) | .value | strings' | awk 'length >= 8')
    else
      mapfile -t -O "${#VALEURS[@]}" VALEURS < <(printf '%s\n' "$brut" | awk 'length >= 8')
    fi
  done
  brut=""
}

occurrences() { grep -c -F -f <(printf '%s\n' "${VALEURS[@]}") || true; }

chercher() {
  local source="$1" nombre
  shift
  if ! nombre="$("$@" 2>/dev/null | occurrences)"; then
    resultat X20 "$source" nv "source illisible"
    return 0
  fi
  if [ "${nombre:-0}" -eq 0 ]; then
    resultat X20 "$source" ok "aucune occurrence"
  else
    resultat X20 "$source" ko "$nombre ligne(s) contenant une valeur de secret : révoquer puis corriger (D03 X20)"
  fi
}

etat_ephemere() { tf show -json; }
etat_persistant() { tfp show -json; }

releases_helm() {
  local espace release
  helm --kube-context "$NOM" list -A -o json | jq -r '.[] | "\(.namespace) \(.name)"' |
    while read -r espace release; do
      helm --kube-context "$NOM" get all "$release" -n "$espace"
    done
}

objets_cluster() {
  k get configmaps -A -o json
  k get pods -A -o json
}

journaux_poste() {
  find "$DOSSIER_SESSION" -maxdepth 1 -type f ! -name '*.zip' -exec cat {} + 2>/dev/null || true
  cat "$CSV_SESSIONS" "$CSV_COUTS" 2>/dev/null || true
  find "$DOSSIER_RAPPORTS" -type f -exec cat {} + 2>/dev/null || true
}

journaux_ci() {
  local run
  gh auth status >/dev/null 2>&1 || return 1
  gh run list --limit 5 --json databaseId -q '.[].databaseId' | tr -d '\r' |
    while read -r run; do gh run view "$run" --log; done
}

controler_secrets() {
  local cles
  collecter_valeurs
  if [ "${#VALEURS[@]}" -eq 0 ]; then
    resultat X20 valeurs nv "aucune valeur de secret de session lue : recherche sans objet"
    return 0
  fi
  resultat X20 valeurs info "${#VALEURS[@]} valeur(s) de secret de session cherchée(s), jamais affichées"
  chercher etat-ephemere etat_ephemere
  [ ! -f "$DOSSIER_PERSISTANT/terraform.tfstate" ] || chercher etat-persistant etat_persistant
  chercher releases-helm releases_helm
  chercher objets-cluster objets_cluster
  chercher journaux-poste journaux_poste
  if command -v gh >/dev/null 2>&1; then
    chercher journaux-ci journaux_ci
  else
    resultat X20 journaux-ci nv "gh absent : journaux de la CI non lus"
  fi

  cles="$(tf show -json | jq '[.. | objects | to_entries[] |
    select((.key | test("^(password|master_password|secret_string|private_key)$")) and (.value | type == "string") and (.value != ""))] | length')"
  if [ "$cles" -eq 0 ]; then
    resultat X20 cles-sensibles-etat ok "aucune clé password, secret_string, private_key renseignée (MI4)"
  else
    resultat X20 cles-sensibles-etat ko "$cles clé(s) sensible(s) renseignée(s) dans l'état"
  fi
  VALEURS=()
}

controler_montages() {
  local autorises montages ns app secrets secret hors=""
  autorises="$(grep -v -E '^\s*(#|$)' "$DOSSIER_K8S/secrets-autorises.txt" || true)"
  montages="$(k get pods -A -o json | jq -r --arg s "$NAMESPACES_SYSTEME" '($s | split(" ")) as $sys | .items[] |
    select(.metadata.namespace as $n | $sys | index($n) | not) |
    {ns: .metadata.namespace, app: (.metadata.labels["app.kubernetes.io/name"] // .metadata.labels.app // .metadata.name),
     s: ([.spec.volumes[]?.secret.secretName, .spec.volumes[]?.projected.sources[]?.secret.name,
          ((.spec.containers[]?, .spec.initContainers[]?) | .env[]?.valueFrom.secretKeyRef.name, .envFrom[]?.secretRef.name)] |
         map(select(. != null)) | unique)} |
    select(.s | length > 0) | "\(.ns) \(.app) \(.s | join(" "))"' | sort -u)"
  while read -r ns app secrets; do
    [ -n "$ns" ] || continue
    for secret in $secrets; do
      printf '%s\n' "$autorises" | awk -v n="$ns" -v a="$app" -v s="$secret" \
        '$1 == n && $2 == a { for (i = 3; i <= NF; i++) if ($i == s) trouve = 1 } END { exit !trouve }' ||
        hors="$hors $ns/$app:$secret"
    done
  done <<<"$montages"
  if [ -z "$hors" ]; then
    resultat X20 montages ok "chaque pod ne monte que ses Secrets (infra/k8s/secrets-autorises.txt)"
  else
    resultat X20 montages ko "montages hors liste :$hors"
  fi
}

controler_analyses() {
  local images image n=0 recommandee actuelle
  if ! docker info >/dev/null 2>&1; then
    resultat X20 analyse-images nv "Docker indisponible : Trivy non lancé"
    resultat X20 analyse-iac nv "Docker indisponible : Trivy non lancé"
  else
    images="$(k get pods -A -o json | jq -r --arg s "$NAMESPACES_SYSTEME" '($s | split(" ")) as $sys | .items[] |
      select(.metadata.namespace as $n | $sys | index($n) | not) | (.spec.containers[]?, .spec.initContainers[]?) | .image' | sort -u)"
    for image in $images; do
      n=$((n + 1))
      if bash "$RACINE_TOOLS/ci/trivy.sh" image "$image" "controle-image-$n" >"$TRAVAIL/trivy-image-$n.txt" 2>&1; then
        resultat X20 analyse-image ok "$image : aucune vulnérabilité critique ou élevée corrigeable, aucun secret"
      else
        resultat X20 analyse-image ko "$image : constat bloquant (rapports/i6-trivy-controle-image-$n.json)"
      fi
    done
    [ "$n" -gt 0 ] || resultat X20 analyse-image nv "aucune image applicative déployée"
    if bash "$RACINE_TOOLS/ci/trivy.sh" config infra controle-iac >"$TRAVAIL/trivy-iac.txt" 2>&1; then
      resultat X20 analyse-iac ok "Trivy config infra : aucun constat critique ou élevé"
    else
      resultat X20 analyse-iac ko "Trivy config infra : constat bloquant (rapports/i6-trivy-controle-iac.json)"
    fi
  fi

  recommandee="$(aws ssm get-parameter --name "/aws/service/eks/optimized-ami/$VERSION_EKS/amazon-linux-2023/x86_64/standard/recommended/release_version" \
    --query Parameter.Value --output text | tr -d '\r' || true)"
  actuelle="$(aws_json eks describe-nodegroup --cluster-name "$NOM" --nodegroup-name "$(sortie nom_node_group)" |
    jq -r '.nodegroup.releaseVersion' || true)"
  if [ -n "$actuelle" ] && [ "$actuelle" = "$recommandee" ]; then
    resultat X20 ami ok "AMI des nœuds $actuelle, la recommandée (MS9)"
  else
    resultat X20 ami ko "AMI des nœuds $actuelle, recommandée $recommandee : relancer up.sh"
  fi
}

ecrire_verdict() {
  local verdict="$DOSSIER_SESSION/controle-$SESSION_TF.json" vert
  jq -R -s --arg session "$SESSION_TF" --arg h "$(date -u +%Y-%m-%dT%H:%M:%SZ)" --arg serie "$(serie_etat)" \
    --arg ouvert "$ALB_OUVERT" '
    [split("\n")[] | select(length > 0) | split("\t") | {groupe: .[0], controle: .[1], statut: .[2], detail: .[3]}] as $r |
    {session: $session, horodatage: $h, serie_etat: ($serie | tonumber), alb_ouvert: ($ouvert == "true"),
     vert: ([$r[] | select(.statut == "ko" or .statut == "nv")] | length == 0), resultats: $r}' <"$RES_TSV" >"$verdict"
  {
    printf 'Contrôle après déploiement, session %s, %s\n\n' "$SESSION_TF" "$(jq -r '.horodatage' "$verdict")"
    jq -r '.resultats[] | "[\(.statut)] \(.groupe) \(.controle) : \(.detail)"' "$verdict"
    printf '\nVerdict : %s\n' "$(jq -r 'if .vert then "VERT" else "ROUGE" end' "$verdict")"
  } | masquer_ip >"$DOSSIER_RAPPORTS/controle-$SESSION_TF.txt"
  vert="$(jq -r '.vert' "$verdict")"
  phase controle verdict "$(($(maintenant) - T0))" "$([ "$vert" = "true" ] && printf 'vert' || printf 'rouge')"
  [ "$vert" = "true" ]
}

exiger_outils aws terraform jq curl sed awk sha256sum openssl timeout kubectl helm
sonde_identite
charger_session
[ -n "${SESSION_TF:-}" ] || echec "aucune session en cours (up.sh d'abord)"
etat_vide && echec "couche éphémère absente : rien à contrôler"

T0="$(maintenant)"
TRAVAIL="$(chemin_natif "$(mktemp -d)")"
RES_TSV="$TRAVAIL/resultats.tsv"
: >"$RES_TSV"
trap nettoyer EXIT
mkdir -p "$DOSSIER_RAPPORTS"

SORTIES="$(tf output -json)"
VPC="$(sortie id_vpc)"
NOM_ALB="$(sortie nom_alb)"
DNS_ALB="$(sortie dns_alb)"
ARN_ALB="$(sortie arn_alb)"
ARN_HTTPS="$(sortie arn_ecouteur_https)"
ARN_HTTP="$(sortie arn_ecouteur_http)"
SG_ALB="$(sortie id_groupe_securite_alb)"
ALB_OUVERT="$(sortie alb_ouvert)"
NODEPORT="$(sortie nodeport_front)"
ID_RDS="$(sortie id_rds)"
ADRESSE_RDS="$(sortie adresse_rds)"
POINT_API="$(sortie point_api_eks)"
IP_POSTE="$(ip_poste)"
IPS_ALB=()
IPS_NAT=()
VALEURS=()

journal "contrôle après déploiement de la session $SESSION_TF (ALB ouvert : $ALB_OUVERT)"
controler_tls
printf 'X17 exposition (AC8)\n' >&2
controler_adresses_publiques
controler_configuration_reseau
controler_hors_plage
controler_load_balancers
controler_services
controler_regles_alb
controler_liste_blanche
controler_sortie_imds
printf 'X20 secrets et analyses (AC9)\n' >&2
controler_secrets
controler_montages
controler_analyses

if ecrire_verdict; then
  journal "contrôle VERT ; rapports dans docs/mesures/US-004/ (tls-$SESSION_TF.txt, controle-$SESSION_TF.txt)"
  exit 0
fi
journal "contrôle ROUGE : ouverture de l'ALB refusée (AC10) ; détail dans docs/mesures/US-004/controle-$SESSION_TF.txt"
exit 1
