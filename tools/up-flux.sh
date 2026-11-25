#!/usr/bin/env bash
# Déploiement de l'application sur un cluster prêt (US-007 AC3, AC4, AC6 ; D07 MD2 ; D10 P2 et filet P1), appelé par
# up.sh : empreintes et commit source de env/demo, signatures vérifiées, namespaces et refus par défaut, base amorcée,
# Secrets par service (repris s'ils existent), puis Flux épinglé sur les commits vérifiés ou helm en direct (--sans-flux),
# réconciliation attendue et tests de fumée.
#   ./up-flux.sh --contexte CONTEXTE --valeurs aws|k3d [--sans-flux] [--rotation]
#   ./up-flux.sh --contexte k3d-NOM --valeurs k3d --sans-flux --empreintes FICHIER [--images-locales]
# --empreintes : fichier au format de env/demo, avec --sans-flux seulement (Flux ne lit que env/demo).
# --images-locales : k3d seulement, images construites sur le poste et non signées ; vérification sautée, annoncée.
# --rotation : nouveaux mots de passe des rôles de base, puis redémarrage des services.

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

usage() {
  sed -n '6,10p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

CONTEXTE=""
CIBLE=""
SANS_FLUX=0
EMPREINTES=""
IMAGES_LOCALES=0
ROTATION=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --contexte) CONTEXTE="${2:-}"; shift 2 ;;
    --valeurs) CIBLE="${2:-}"; shift 2 ;;
    --sans-flux) SANS_FLUX=1; shift ;;
    --empreintes) EMPREINTES="${2:-}"; shift 2 ;;
    --images-locales) IMAGES_LOCALES=1; shift ;;
    --rotation) ROTATION=1; shift ;;
    *) usage ;;
  esac
done
[ -n "$CONTEXTE" ] || usage
case "$CIBLE" in aws | k3d) ;; *) usage ;; esac
contexte_autorise "$CONTEXTE"
[ -z "$EMPREINTES" ] || [ "$SANS_FLUX" -eq 1 ] ||
  echec "--empreintes exige --sans-flux : Flux ne déploie que la révision vérifiée de env/demo"
if [ "$IMAGES_LOCALES" -eq 1 ]; then
  [ "$CIBLE" = k3d ] && [[ "$CONTEXTE" == k3d-* ]] || echec "--images-locales réservé à k3d"
fi

DEPOT_GIT="https://github.com/QVL-Collector/collector"
IDENTITE_SIGNATURE="$DEPOT_GIT/.github/workflows/ci.yml@refs/heads/main"
EMETTEUR_SIGNATURE="https://token.actions.githubusercontent.com"
CHART="$DEPOT/infra/application"
VALEURS="$CHART/valeurs-$CIBLE.yaml"
DOSSIER_FLUX="$DEPOT/infra/flux"
LISTE_IMAGES="$DEPOT/tools/ci/images.txt"
POSTGRES_IMAGE="postgres:18.3@sha256:7e32e9833a6fb1c92c32552794cb6ed569d51b445a54907d35fc112ef39684db"
URL_AUTORITES_RDS="https://truststore.pki.rds.amazonaws.com/us-east-1/us-east-1-bundle.pem"
BASES=(marche interactions transactions)
NS_BASES=(place-de-marche interactions transactions)
SESSION_K8S="$DOSSIER_SESSION/k8s-$CONTEXTE"
COMMIT_ENV=""
COMMIT_SOURCE=""
VALEURS_PORTS=()

kc() { kubectl --context "$CONTEXTE" --request-timeout=30s "$@"; }
hm() { helm --kube-context "$CONTEXTE" "$@"; }
mdp() { openssl rand -hex 24; }
b64() { printf '%s' "$1" | base64 | tr -d '\n\r'; }

secret_depuis_entree() {
  local ns="$1" nom="$2" cle valeur
  {
    printf 'apiVersion: v1\nkind: Secret\ntype: Opaque\nmetadata:\n  name: %s\n  namespace: %s\n  labels:\n    collector/session: "true"\ndata:\n' "$nom" "$ns"
    while IFS='=' read -r cle valeur; do printf '  %s: %s\n' "$cle" "$(b64 "$valeur")"; done
  } | kc apply -f - >/dev/null
}

lire_secret() {
  kc -n "$1" get secret "$2" -o json 2>/dev/null | jq -r --arg c "$3" '.data[$c] // empty' | base64 -d 2>/dev/null || true
}

mdp_url() { sed -nE 's|^postgres://[^:]+:([0-9a-f]+)@.*|\1|p'; }

lire_empreintes() {
  local nom valeur noms
  mkdir -p "$SESSION_K8S"
  if [ -z "$EMPREINTES" ]; then
    EMPREINTES="$SESSION_K8S/empreintes.yaml"
    git -C "$DEPOT" fetch -q "$DEPOT_GIT" env/demo || echec "branche env/demo illisible sur $DEPOT_GIT"
    COMMIT_ENV="$(git -C "$DEPOT" rev-parse FETCH_HEAD)"
    git -C "$DEPOT" show "$COMMIT_ENV:deploy/demo/empreintes.yaml" >"$EMPREINTES"
  fi
  [ -f "$EMPREINTES" ] || echec "fichier d'empreintes introuvable : $EMPREINTES"
  COMMIT_SOURCE="$(sed -n 's/^source: *//p' "$EMPREINTES" | tr -d '\r')"
  noms="$(cut -d'|' -f1 "$LISTE_IMAGES" | tr -d '\r' | grep .)"
  IMAGES_EMPREINTES=""
  while read -r nom; do
    valeur="$(sed -n "s|^  $nom: ||p" "$EMPREINTES" | tr -d '\r')"
    [[ "$valeur" =~ /$nom@sha256:[0-9a-f]{64}$ ]] ||
      echec "images.$nom absente, ou pas épinglée sur son propre dépôt, dans $EMPREINTES"
    IMAGES_EMPREINTES+="$valeur"$'\n'
  done <<<"$noms"
  [ "$(grep -c '^  [a-z-]*: ' "$EMPREINTES")" -eq "$(printf '%s\n' "$noms" | wc -l)" ] ||
    echec "images inattendues dans $EMPREINTES (liste attendue : tools/ci/images.txt)"
  if [ "$SANS_FLUX" -eq 0 ]; then
    [[ "$COMMIT_SOURCE" =~ ^[0-9a-f]{40}$ ]] || echec "commit source absent de env/demo : Flux ne peut pas être épinglé"
    journal "env/demo ${COMMIT_ENV:0:12}, chart et empreintes de main ${COMMIT_SOURCE:0:12}"
  fi
}

cosign_verifier() {
  if command -v cosign >/dev/null 2>&1; then
    cosign verify --certificate-identity "$IDENTITE_SIGNATURE" --certificate-oidc-issuer "$EMETTEUR_SIGNATURE" "$1"
  else
    docker compose -f "$DEPOT/tools/ci/outils.yml" run --rm -T -e DOCKER_CONFIG=/tmp/anonyme cosign verify \
      --certificate-identity "$IDENTITE_SIGNATURE" --certificate-oidc-issuer "$EMETTEUR_SIGNATURE" "$1"
  fi
}

verifier_signatures() {
  local image refus=0
  if [ "$IMAGES_LOCALES" -eq 1 ]; then
    journal "ATTENTION : images locales non signées, vérification des signatures sautée (k3d seulement)"
    return 0
  fi
  while read -r image; do
    [ -n "$image" ] || continue
    if cosign_verifier "$image" </dev/null >/dev/null 2>&1; then
      journal "signature vérifiée : $image"
    else
      journal "signature absente ou d'une autre identité : $image"
      refus=1
    fi
  done <<<"$IMAGES_EMPREINTES"
  [ "$refus" -eq 0 ] || echec "AC4 : déploiement refusé, au moins une image n'est pas signée par le workflow de main"
}

lire_ports() {
  local sorties
  [ "$CIBLE" = aws ] || return 0
  sorties="$(tf output -json)"
  VALEURS_PORTS=(
    "front.nodePort=$(printf '%s' "$sorties" | jq -r '.nodeport_front.value')"
    "identite.nodePort=$(printf '%s' "$sorties" | jq -r '.nodeports_application.value.keycloak')"
    "placeDeMarche.nodePort=$(printf '%s' "$sorties" | jq -r '.nodeports_application.value["place-de-marche"]')"
    "interactions.nodePort=$(printf '%s' "$sorties" | jq -r '.nodeports_application.value.interactions')"
    "transactions.nodePort=$(printf '%s' "$sorties" | jq -r '.nodeports_application.value.transactions')"
  )
  if printf '%s\n' "${VALEURS_PORTS[@]}" | grep -qv '=[0-9]\{5\}$'; then
    echec "NodePorts illisibles dans les sorties Terraform (application_active ?)"
  fi
  journal "NodePorts lus dans Terraform : ${VALEURS_PORTS[*]}"
}

poser_namespaces() {
  kc apply -f "$DOSSIER_K8S/socle.yaml" >/dev/null
  kc apply -f "$DOSSIER_K8S/application.yaml" >/dev/null
  kc -n front delete networkpolicy sortie-sans-imds --ignore-not-found >/dev/null
  journal "namespaces applicatifs, refus par défaut et DNS posés"
}

preparer_base() {
  local ns arn
  if [ "$CIBLE" = k3d ]; then
    BASE_HOTE="postgres.base-locale.svc"
    BASE_TLS="sslmode=disable"
    kc apply -f "$DOSSIER_K8S/base-locale.yaml" >/dev/null
    if ! kc -n base-locale get secret base-maitre >/dev/null 2>&1; then
      printf 'MOT_DE_PASSE=%s\n' "$(mdp)" | secret_depuis_entree base-locale base-maitre
    fi
    MDP_MAITRE="$(lire_secret base-locale base-maitre MOT_DE_PASSE)"
    kc -n base-locale rollout status deployment/postgres --timeout=5m >/dev/null
  else
    BASE_HOTE="$(tf output -raw adresse_rds)"
    BASE_TLS="sslmode=verify-full&sslrootcert=/etc/autorites-rds/ca.pem"
    arn="$(aws_json rds describe-db-instances --db-instance-identifier "$NOM" | jq -r '.DBInstances[0].MasterUserSecret.SecretArn')"
    MDP_MAITRE="$(aws secretsmanager get-secret-value --secret-id "$arn" --query SecretString --output text | jq -r .password)"
    curl -fsS --max-time 30 -o "$SESSION_K8S/autorites-rds.pem" "$URL_AUTORITES_RDS"
    for ns in collector identite "${NS_BASES[@]}"; do
      kc -n "$ns" create configmap autorites-rds --from-file=ca.pem="$SESSION_K8S/autorites-rds.pem" \
        --dry-run=client -o yaml | kc apply -f - >/dev/null
    done
  fi
}

charger_mots_de_passe() {
  local i b ns r valeur
  declare -gA MDP_ROLES=()
  for i in "${!BASES[@]}"; do
    b="${BASES[$i]}"
    ns="${NS_BASES[$i]}"
    for r in migration execution; do
      valeur=""
      [ "$ROTATION" -eq 1 ] || valeur="$(lire_secret "$ns" "base-$r" BASE_DE_DONNEES_URL | mdp_url)"
      MDP_ROLES["${b}_$r"]="${valeur:-$(mdp)}"
    done
  done
  valeur=""
  [ "$ROTATION" -eq 1 ] || valeur="$(lire_secret identite keycloak KC_DB_PASSWORD)"
  MDP_ROLES[keycloak]="${valeur:-$(mdp)}"
  SECRET_TRANSACTIONS="$(lire_secret identite keycloak SECRET_CLIENT_TRANSACTIONS)"
  SECRET_TRANSACTIONS="${SECRET_TRANSACTIONS:-$(mdp)}"
  MDP_AMORCE="$(lire_secret identite keycloak KC_BOOTSTRAP_ADMIN_PASSWORD)"
  MDP_AMORCE="${MDP_AMORCE:-$(mdp)}"
}

nettoyer_amorcage() {
  kc -n collector delete secret/amorcage-base configmap/amorcage-base networkpolicy/amorcage-base \
    --ignore-not-found >/dev/null 2>&1 || true
}

sql_amorcage() {
  local b r roles
  printf '\\set ON_ERROR_STOP on\n'
  for b in "${BASES[@]}" keycloak; do
    if [ "$b" = keycloak ]; then set -- keycloak; else set -- "${b}_migration" "${b}_execution"; fi
    for r in "$@"; do
      printf '\\getenv mdp MDP_%s\n' "${r^^}"
      printf "SELECT format('CREATE ROLE %%I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE', '%s') WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '%s') \\\\gexec\n" "$r" "$r"
      printf "ALTER ROLE %s PASSWORD :'mdp';\n" "$r"
    done
    if [ "$b" != keycloak ]; then
      printf "ALTER ROLE %s_execution CONNECTION LIMIT 40;\n" "$b"
      printf "ALTER ROLE %s_execution SET statement_timeout = '5s';\n" "$b"
      printf "ALTER ROLE %s_execution SET idle_in_transaction_session_timeout = '10s';\n" "$b"
    fi
    roles="$*"
    printf 'GRANT %s TO CURRENT_USER;\n' "$1"
    printf "SELECT 'CREATE DATABASE %s OWNER %s' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '%s') \\gexec\n" "$b" "$1" "$b"
    printf 'REVOKE CONNECT ON DATABASE %s FROM PUBLIC;\nGRANT CONNECT ON DATABASE %s TO %s;\n' "$b" "$b" "${roles// /, }"
    printf '\\c %s\nSET ROLE %s;\nREVOKE CREATE ON SCHEMA public FROM PUBLIC;\nRESET ROLE;\n\\c postgres\n' "$b" "$1"
  done
}

amorcer_base() {
  local r ssl
  trap nettoyer_amorcage EXIT
  {
    printf 'PGPASSWORD=%s\n' "$MDP_MAITRE"
    for r in "${!MDP_ROLES[@]}"; do printf 'MDP_%s=%s\n' "${r^^}" "${MDP_ROLES[$r]}"; done
  } | secret_depuis_entree collector amorcage-base
  kc -n collector create configmap amorcage-base --from-literal=amorcage.sql="$(sql_amorcage)" \
    --dry-run=client -o yaml | kc apply -f - >/dev/null
  ssl="disable"
  [ "$CIBLE" = k3d ] || ssl="verify-full"
  kc -n collector delete job -l app.kubernetes.io/name=amorcage-base --ignore-not-found >/dev/null
  JOB_AMORCAGE="amorcage-base-$(date -u +%Y%m%d%H%M%S)"
  kc apply -f - >/dev/null <<YAML
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: amorcage-base
  namespace: collector
spec:
  podSelector:
    matchLabels: { app.kubernetes.io/name: amorcage-base }
  policyTypes: [Egress]
  egress:
    - to:
        - namespaceSelector:
            matchLabels: { kubernetes.io/metadata.name: kube-system }
      ports:
        - { protocol: UDP, port: 53 }
        - { protocol: TCP, port: 53 }
    - to:
        - ipBlock: { cidr: 0.0.0.0/0, except: [169.254.169.254/32] }
        - namespaceSelector:
            matchLabels: { kubernetes.io/metadata.name: base-locale }
      ports:
        - { protocol: TCP, port: 5432 }
---
apiVersion: batch/v1
kind: Job
metadata:
  name: $JOB_AMORCAGE
  labels: { app.kubernetes.io/name: amorcage-base }
  namespace: collector
spec:
  backoffLimit: 2
  activeDeadlineSeconds: 300
  template:
    metadata:
      labels: { app.kubernetes.io/name: amorcage-base }
    spec:
      restartPolicy: Never
      automountServiceAccountToken: false
      securityContext:
        runAsNonRoot: true
        runAsUser: 999
        runAsGroup: 999
        seccompProfile: { type: RuntimeDefault }
      containers:
        - name: psql
          image: $POSTGRES_IMAGE
          command: ["sh", "-c", "until pg_isready -q; do sleep 2; done; exec psql -X -q -f /sql/amorcage.sql"]
          env:
            - { name: PGHOST, value: "$BASE_HOTE" }
            - { name: PGUSER, value: collector_admin }
            - { name: PGDATABASE, value: postgres }
            - { name: PGSSLMODE, value: $ssl }
            - { name: PGSSLROOTCERT, value: /etc/autorites-rds/ca.pem }
          envFrom:
            - secretRef: { name: amorcage-base }
          resources:
            requests: { cpu: 50m, memory: 32Mi }
            limits: { memory: 128Mi }
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities: { drop: ["ALL"] }
          volumeMounts:
            - { name: sql, mountPath: /sql, readOnly: true }
            - { name: autorites, mountPath: /etc/autorites-rds, readOnly: true }
      volumes:
        - name: sql
          configMap: { name: amorcage-base }
        - name: autorites
          configMap: { name: autorites-rds, optional: true }
YAML
  if ! kc -n collector wait --for=jsonpath='{.status.succeeded}'=1 "job/$JOB_AMORCAGE" --timeout=5m >/dev/null; then
    kc -n collector logs "job/$JOB_AMORCAGE" --tail=20 | masquer >&2 || true
    echec "amorçage de la base en échec"
  fi
  kc -n collector delete job "$JOB_AMORCAGE" --wait=false >/dev/null
  nettoyer_amorcage
  trap - EXIT
  MDP_MAITRE=""
  journal "bases, rôles de migration et d'exécution amorcés ; Secret maître supprimé (MD2)"
}

url_base() { printf 'postgres://%s:%s@%s:5432/%s?%s' "$1" "${MDP_ROLES[$1]}" "$BASE_HOTE" "$2" "$BASE_TLS"; }

poser_secrets() {
  local i b ns kc_tls
  for i in "${!BASES[@]}"; do
    b="${BASES[$i]}"
    ns="${NS_BASES[$i]}"
    printf 'BASE_DE_DONNEES_URL=%s\n' "$(url_base "${b}_execution" "$b")" | secret_depuis_entree "$ns" base-execution
    printf 'BASE_DE_DONNEES_URL=%s\n' "$(url_base "${b}_migration" "$b")" | secret_depuis_entree "$ns" base-migration
  done
  printf 'OIDC_CLIENT_SECRET=%s\n' "$SECRET_TRANSACTIONS" | secret_depuis_entree transactions oidc-client
  kc_tls=""
  [ "$CIBLE" = k3d ] || kc_tls="?sslmode=verify-full&sslrootcert=/etc/autorites-rds/ca.pem"
  {
    printf 'KC_DB_URL=jdbc:postgresql://%s:5432/keycloak%s\n' "$BASE_HOTE" "$kc_tls"
    printf 'KC_DB_USERNAME=keycloak\nKC_DB_PASSWORD=%s\nKC_DB_POOL_MAX_SIZE=20\n' "${MDP_ROLES[keycloak]}"
    printf 'KC_BOOTSTRAP_ADMIN_USERNAME=amorce\nKC_BOOTSTRAP_ADMIN_PASSWORD=%s\n' "$MDP_AMORCE"
    printf 'SECRET_CLIENT_TRANSACTIONS=%s\n' "$SECRET_TRANSACTIONS"
  } | secret_depuis_entree identite keycloak
  unset SECRET_TRANSACTIONS MDP_AMORCE MDP_ROLES
  [ -f "$DEPOT/infra/canal/.genere/mots-de-passe.env" ] ||
    KUBE_CONTEXT="$CONTEXTE" "$DEPOT/infra/canal/outils/identifiants.sh" >/dev/null
  KUBE_CONTEXT="$CONTEXTE" "$DEPOT/infra/canal/outils/kubernetes.sh" secrets
  journal "Secrets posés, chacun dans le seul namespace de son service ($([ "$ROTATION" -eq 1 ] && printf 'rotation' || printf 'repris s'"'"'ils existaient'))"
}

releases_rendues() {
  local ports
  ports="$(printf '%s\n' "${VALEURS_PORTS[@]}" | awk -F'[.=]' '{ printf "    %s:\n      nodePort: %s\n", $1, $3 }')"
  sed "s|infra/application/valeurs-aws.yaml|infra/application/valeurs-$CIBLE.yaml|" "$DOSSIER_FLUX/releases.yaml" |
    awk -v ports="$ports" '{ print } /^  releaseName: application$/ && ports != "" { printf "  values:\n%s", ports }'
}

installer_flux() {
  local fichier="$SESSION_K8S/flux-install.yaml"
  manifestes_flux >"$fichier"
  grep -q "image: ghcr.io/fluxcd/helm-controller:.*@sha256:" "$fichier" || echec "image du contrôleur Helm non épinglée"
  grep -q -- "--default-service-account=deployeur" "$fichier" || echec "compte deployeur non imposé au contrôleur Helm"
  ! grep -q '^kind: ClusterRoleBinding' "$fichier" || echec "liaison de cluster restée dans le manifeste Flux"
  kc apply -f "$DOSSIER_FLUX/controleurs.yaml" -f "$DOSSIER_FLUX/deployeur.yaml" >/dev/null
  kc apply --server-side --force-conflicts -f "$fichier" >/dev/null
  kc -n "$NS_FLUX" rollout status deployment/source-controller --timeout=5m >/dev/null
  kc -n "$NS_FLUX" rollout status deployment/helm-controller --timeout=5m >/dev/null
  journal "Flux installé dans $NS_FLUX : deux contrôleurs, droits du seul namespace, usurpation du seul deployeur, sortie sans IMDS (MP4)"
  sed -e "s|^    branch: env/demo$|    commit: $COMMIT_ENV|" -e "s|^    branch: main$|    commit: $COMMIT_SOURCE|" \
    "$DOSSIER_FLUX/sources.yaml" | kc apply -f - >/dev/null
  releases_rendues | kc apply -f - >/dev/null
  journal "sources épinglées (env/demo ${COMMIT_ENV:0:12}, main ${COMMIT_SOURCE:0:12}) ; attente de la réconciliation"
  kc -n collector wait --for=condition=Ready helmrelease/canal --timeout=10m >/dev/null ||
    echec "HelmRelease canal non prête (kubectl -n collector describe helmrelease canal)"
  kc -n collector wait --for=condition=Ready helmrelease/application --timeout=15m >/dev/null ||
    echec "HelmRelease application non prête (kubectl -n collector describe helmrelease application)"
}

installer_sans_flux() {
  local options=() valeur
  journal "filet P1 : helm upgrade --install, même chart, mêmes valeurs, mêmes empreintes"
  hm upgrade --install canal "$(chemin_natif "$DEPOT/infra/canal")" -n canal --wait --timeout 6m >/dev/null
  [ "$IMAGES_LOCALES" -eq 0 ] ||
    options+=(--set "registreAttendu=$(printf '%s\n' "$IMAGES_EMPREINTES" | head -n 1 | cut -d/ -f1)/")
  for valeur in "${VALEURS_PORTS[@]}"; do options+=(--set "$valeur"); done
  hm upgrade --install application "$(chemin_natif "$CHART")" -n collector \
    -f "$(chemin_natif "$VALEURS")" -f "$(chemin_natif "$EMPREINTES")" "${options[@]}" \
    --wait --timeout 15m >/dev/null || echec "installation du chart en échec"
}

redemarrer_apres_rotation() {
  local ns
  [ "$ROTATION" -eq 1 ] || return 0
  for ns in "${NS_BASES[@]}"; do kc -n "$ns" rollout restart "deployment/$ns" >/dev/null; done
  kc -n identite rollout restart deployment/keycloak >/dev/null
  journal "rotation : services et Keycloak redémarrés sur les nouveaux mots de passe"
}

tests_de_fumee() {
  local d pid code="000" limite ko=0
  for d in front/front place-de-marche/place-de-marche controle/controle interactions/interactions \
    transactions/transactions identite/keycloak; do
    if kc -n "${d%%/*}" rollout status "deployment/${d#*/}" --timeout=5m >/dev/null; then
      journal "fumée : ${d#*/} disponible"
    else
      journal "fumée : ${d#*/} indisponible"
      ko=1
    fi
  done
  kc -n place-de-marche port-forward svc/place-de-marche 18082:8080 >/dev/null 2>&1 &
  pid=$!
  limite=$(($(maintenant) + 30))
  while [ "$(maintenant)" -lt "$limite" ]; do
    code="$(curl -s -o "$NUL" -w '%{http_code}' --max-time 5 http://127.0.0.1:18082/marche/v1/categories || true)"
    [ "$code" = 200 ] && break
    sleep 1
  done
  kill "$pid" 2>/dev/null || true
  if [ "$code" = 200 ]; then journal "fumée : catalogue des catégories 200"; else journal "fumée : catégories $code"; ko=1; fi
  [ "$ko" -eq 0 ] || echec "tests de fumée en échec"
}

exiger_outils kubectl helm jq openssl git curl docker base64
T0="$(maintenant)"
lire_empreintes
verifier_signatures
lire_ports
poser_namespaces
preparer_base
charger_mots_de_passe
amorcer_base
poser_secrets
if [ "$SANS_FLUX" -eq 1 ]; then installer_sans_flux; else installer_flux; fi
redemarrer_apres_rotation
tests_de_fumee
journal "application déployée en $((($(maintenant) - T0 + 59) / 60)) min ($([ "$SANS_FLUX" -eq 1 ] && printf 'helm, filet P1' || printf 'Flux'))"
