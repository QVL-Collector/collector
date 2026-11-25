#!/usr/bin/env bash
# Déploiement de l'application sur un cluster prêt (US-007 AC3, AC4, AC6 ; D07 MD2 ; D10 P2 et filet P1) :
#   empreintes de env/demo, signatures vérifiées, namespaces et refus par défaut, base amorcée, Secrets par service,
#   puis Flux (source-controller et helm-controller, usurpation du seul compte deployeur) ou helm en direct (--sans-flux),
#   réconciliation attendue et tests de fumée.
#   ./up-flux.sh --contexte CONTEXTE --valeurs aws|k3d [--sans-flux] [--empreintes FICHIER] [--images-locales]
# --empreintes : fichier au format de env/demo (par défaut, lu sur la branche env/demo du dépôt public).
# --images-locales : k3d seulement, images construites sur le poste et non signées ; vérification sautée, annoncée.

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

usage() {
  sed -n '6,8p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

CONTEXTE=""
CIBLE=""
SANS_FLUX=0
EMPREINTES=""
IMAGES_LOCALES=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --contexte) CONTEXTE="${2:-}"; shift 2 ;;
    --valeurs) CIBLE="${2:-}"; shift 2 ;;
    --sans-flux) SANS_FLUX=1; shift ;;
    --empreintes) EMPREINTES="${2:-}"; shift 2 ;;
    --images-locales) IMAGES_LOCALES=1; shift ;;
    *) usage ;;
  esac
done
[ -n "$CONTEXTE" ] || usage
case "$CIBLE" in aws | k3d) ;; *) usage ;; esac
contexte_k8s "$CONTEXTE"
if [ "$IMAGES_LOCALES" -eq 1 ]; then
  [ "$CIBLE" = k3d ] && [[ "$CONTEXTE" == k3d-* ]] || echec "--images-locales réservé à k3d"
fi

DEPOT_GIT="https://github.com/QVL-Collector/collector"
IDENTITE_SIGNATURE="$DEPOT_GIT/.github/workflows/ci.yml@refs/heads/main"
EMETTEUR_SIGNATURE="https://token.actions.githubusercontent.com"
CHART="$DEPOT/infra/application"
VALEURS="$CHART/valeurs-$CIBLE.yaml"
DOSSIER_FLUX="$DEPOT/infra/flux"
FLUX_CLI="ghcr.io/fluxcd/flux-cli:v2.10.1@sha256:0253c5c82a8d98858f33dfe722afdfde83aa17dce05571f5846c26ea40d71b50"
FLUX_IMAGES=(
  "ghcr.io/fluxcd/source-controller:v1.10.2|ghcr.io/fluxcd/source-controller:v1.10.2@sha256:1405a9e460833df9aff920664243c0df271f42be05e0bf38db31d2b7e9ebbaab"
  "ghcr.io/fluxcd/helm-controller:v1.7.2|ghcr.io/fluxcd/helm-controller:v1.7.2@sha256:cc944eff709767f9a4d22de6b52dfa330f872962465868ed7787320b056b84f8"
)
POSTGRES_IMAGE="postgres:18.3@sha256:7e32e9833a6fb1c92c32552794cb6ed569d51b445a54907d35fc112ef39684db"
URL_AUTORITES_RDS="https://truststore.pki.rds.amazonaws.com/us-east-1/us-east-1-bundle.pem"
BASES=(marche interactions transactions)
NS_BASES=(place-de-marche interactions transactions)
SESSION_K8S="$DOSSIER_SESSION/k8s-$CONTEXTE"

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

lire_empreintes() {
  mkdir -p "$SESSION_K8S"
  if [ -z "$EMPREINTES" ]; then
    EMPREINTES="$SESSION_K8S/empreintes.yaml"
    git -C "$DEPOT" fetch -q "$DEPOT_GIT" env/demo || echec "branche env/demo illisible sur $DEPOT_GIT"
    git -C "$DEPOT" show FETCH_HEAD:deploy/demo/empreintes.yaml >"$EMPREINTES"
    journal "empreintes lues sur env/demo ($(git -C "$DEPOT" rev-parse --short FETCH_HEAD))"
  fi
  [ -f "$EMPREINTES" ] || echec "fichier d'empreintes introuvable : $EMPREINTES"
  IMAGES_EMPREINTES="$(sed -n 's/^  [a-z-]*: //p' "$EMPREINTES" | tr -d '\r')"
  [ "$(printf '%s\n' "$IMAGES_EMPREINTES" | grep -c '@sha256:[0-9a-f]\{64\}$')" -eq 6 ] ||
    echec "six images épinglées par empreinte attendues dans $EMPREINTES"
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
    if cosign_verifier "$image" </dev/null >/dev/null 2>&1; then
      journal "signature vérifiée : $image"
    else
      journal "signature absente ou d'une autre identité : $image"
      refus=1
    fi
  done <<<"$IMAGES_EMPREINTES"
  [ "$refus" -eq 0 ] || echec "AC4 : déploiement refusé, au moins une image n'est pas signée par le workflow de main"
}

poser_namespaces() {
  kc apply -f "$DOSSIER_K8S/socle.yaml" >/dev/null
  kc apply -f "$DOSSIER_K8S/application.yaml" >/dev/null
  journal "namespaces applicatifs, refus par défaut et DNS posés"
}

preparer_base() {
  local ns
  if [ "$CIBLE" = k3d ]; then
    BASE_HOTE="postgres.base-locale.svc"
    BASE_TLS="sslmode=disable"
    kc apply -f "$DOSSIER_K8S/base-locale.yaml" >/dev/null
    if ! kc -n base-locale get secret base-maitre >/dev/null 2>&1; then
      printf 'MOT_DE_PASSE=%s\n' "$(mdp)" | secret_depuis_entree base-locale base-maitre
    fi
    MDP_MAITRE="$(kc -n base-locale get secret base-maitre -o jsonpath='{.data.MOT_DE_PASSE}' | base64 -d)"
    kc -n base-locale rollout status deployment/postgres --timeout=5m >/dev/null
  else
    local arn
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
  local b r ssl
  declare -gA MDP_ROLES=()
  for b in "${BASES[@]}"; do
    for r in migration execution; do MDP_ROLES["${b}_$r"]="$(mdp)"; done
  done
  MDP_ROLES[keycloak]="$(mdp)"
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
  kc -n collector delete secret/amorcage-base configmap/amorcage-base networkpolicy/amorcage-base --ignore-not-found >/dev/null
  MDP_MAITRE=""
  journal "bases, rôles de migration et d'exécution amorcés ; Secret maître supprimé (MD2)"
}

url_base() { printf 'postgres://%s:%s@%s:5432/%s?%s' "$1" "${MDP_ROLES[$1]}" "$BASE_HOTE" "$2" "$BASE_TLS"; }

poser_secrets() {
  local i b ns kc_tls secret_transactions
  for i in "${!BASES[@]}"; do
    b="${BASES[$i]}"
    ns="${NS_BASES[$i]}"
    printf 'BASE_DE_DONNEES_URL=%s\n' "$(url_base "${b}_execution" "$b")" | secret_depuis_entree "$ns" base-execution
    printf 'BASE_DE_DONNEES_URL=%s\n' "$(url_base "${b}_migration" "$b")" | secret_depuis_entree "$ns" base-migration
  done
  secret_transactions="$(mdp)"
  printf 'OIDC_CLIENT_SECRET=%s\n' "$secret_transactions" | secret_depuis_entree transactions oidc-client
  kc_tls=""
  [ "$CIBLE" = k3d ] || kc_tls="?sslmode=verify-full&sslrootcert=/etc/autorites-rds/ca.pem"
  {
    printf 'KC_DB_URL=jdbc:postgresql://%s:5432/keycloak%s\n' "$BASE_HOTE" "$kc_tls"
    printf 'KC_DB_USERNAME=keycloak\nKC_DB_PASSWORD=%s\nKC_DB_POOL_MAX_SIZE=20\n' "${MDP_ROLES[keycloak]}"
    printf 'KC_BOOTSTRAP_ADMIN_USERNAME=amorce\nKC_BOOTSTRAP_ADMIN_PASSWORD=%s\n' "$(mdp)"
    printf 'SECRET_CLIENT_TRANSACTIONS=%s\n' "$secret_transactions"
  } | secret_depuis_entree identite keycloak
  unset secret_transactions MDP_ROLES
  [ -f "$DEPOT/infra/canal/.genere/mots-de-passe.env" ] ||
    KUBE_CONTEXT="$CONTEXTE" "$DEPOT/infra/canal/outils/identifiants.sh" >/dev/null
  KUBE_CONTEXT="$CONTEXTE" "$DEPOT/infra/canal/outils/kubernetes.sh" secrets
  journal "Secrets posés, chacun dans le seul namespace de son service"
}

manifestes_flux() {
  local paire
  if command -v flux >/dev/null 2>&1; then
    flux install --components=source-controller,helm-controller --export
  else
    docker run --rm "$FLUX_CLI" install --components=source-controller,helm-controller --export
  fi | tr -d '\r' | {
    local flux_sed=()
    for paire in "${FLUX_IMAGES[@]}"; do flux_sed+=(-e "s|image: ${paire%%|*}\$|image: ${paire#*|}|"); done
    sed "${flux_sed[@]}"
  }
}

installer_flux() {
  local fichier="$SESSION_K8S/flux-install.yaml" valeurs_flux
  manifestes_flux >"$fichier"
  grep -q "image: ghcr.io/fluxcd/helm-controller:.*@sha256:" "$fichier" || echec "image du contrôleur Helm non épinglée"
  kc apply --server-side --force-conflicts -f "$fichier" >/dev/null
  kc delete clusterrolebinding cluster-reconciler-flux-system --ignore-not-found >/dev/null
  kc apply -f "$DOSSIER_FLUX/deployeur.yaml" >/dev/null
  kc -n flux-system patch deployment helm-controller --type=json -p '[
    {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--default-service-account=deployeur"},
    {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--no-cross-namespace-refs=true"}]' >/dev/null
  kc -n flux-system rollout status deployment/source-controller --timeout=5m >/dev/null
  kc -n flux-system rollout status deployment/helm-controller --timeout=5m >/dev/null
  journal "Flux installé : source-controller et helm-controller, cluster-admin retiré, compte deployeur imposé (MP4)"
  kc apply -f "$DOSSIER_FLUX/sources.yaml" >/dev/null
  valeurs_flux="infra/application/valeurs-$CIBLE.yaml"
  sed "s|infra/application/valeurs-aws.yaml|$valeurs_flux|" "$DOSSIER_FLUX/releases.yaml" | kc apply -f - >/dev/null
  journal "HelmRelease appliquées depuis env/demo ; attente de la réconciliation (15 min au plus)"
  kc -n collector wait --for=condition=Ready helmrelease/canal --timeout=10m >/dev/null ||
    echec "HelmRelease canal non prête (kubectl -n collector describe helmrelease canal)"
  kc -n collector wait --for=condition=Ready helmrelease/application --timeout=15m >/dev/null ||
    echec "HelmRelease application non prête (kubectl -n collector describe helmrelease application)"
}

installer_sans_flux() {
  local registre=()
  journal "filet P1 : helm upgrade --install, même chart, mêmes valeurs, mêmes empreintes"
  hm upgrade --install canal "$(chemin_natif "$DEPOT/infra/canal")" -n canal --wait --timeout 6m >/dev/null
  [ "$IMAGES_LOCALES" -eq 0 ] ||
    registre=(--set "registreAttendu=$(printf '%s\n' "$IMAGES_EMPREINTES" | head -n 1 | cut -d/ -f1)/")
  hm upgrade --install application "$(chemin_natif "$CHART")" -n collector \
    -f "$(chemin_natif "$VALEURS")" -f "$(chemin_natif "$EMPREINTES")" "${registre[@]}" \
    --wait --timeout 15m >/dev/null || echec "installation du chart en échec"
}

chemin_natif() { if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi; }

tests_de_fumee() {
  local d pid code ko=0
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
  sleep 3
  code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 http://127.0.0.1:18082/marche/v1/categories || true)"
  kill "$pid" 2>/dev/null || true
  if [ "$code" = 200 ]; then journal "fumée : catalogue des catégories 200"; else journal "fumée : catégories $code"; ko=1; fi
  [ "$ko" -eq 0 ] || echec "tests de fumée en échec"
}

exiger_outils kubectl helm jq openssl git curl docker base64
T0="$(maintenant)"
lire_empreintes
verifier_signatures
poser_namespaces
preparer_base
amorcer_base
poser_secrets
if [ "$SANS_FLUX" -eq 1 ]; then installer_sans_flux; else installer_flux; fi
tests_de_fumee
journal "application déployée en $((($(maintenant) - T0 + 59) / 60)) min ($([ "$SANS_FLUX" -eq 1 ] && printf 'helm, filet P1' || printf 'Flux'))"
