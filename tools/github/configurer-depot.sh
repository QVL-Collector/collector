#!/usr/bin/env bash
# Applique les réglages GitHub du dépôt exigés par D10 (MP1, MP2, MP8), US-001 AC4 et US-002 AC7. Idempotent.
#
# Usage :
#   tools/github/configurer-depot.sh <proprietaire>/<depot>            applique les réglages puis les relève
#   tools/github/configurer-depot.sh <proprietaire>/<depot> --relever  relève seulement (lecture), pour la preuve
#   tools/github/configurer-depot.sh <proprietaire>/<depot> --cle-env-demo  (re)pose la clé de déploiement d'env/demo (US-007)
#
# Jeton : lu dans ~/.qvl/gh-token (ou $FICHIER_JETON) au moment de chaque appel, transmis à gh par son
# environnement seulement ; jamais affiché, jamais écrit ailleurs. Portée fine requise sur le seul dépôt :
# Administration (écriture), Contents (lecture), Secret scanning alerts (lecture), Dependabot alerts (lecture) ;
# --cle-env-demo demande aussi Environments (écriture) et Secrets (écriture).
#
# Ordre conseillé : dépôt publié (US-001), PR de US-002 ouverte et workflow exécuté une fois, puis ce script.

set -euo pipefail
export MSYS_NO_PATHCONV=1

DEPOT="${1:-}"
MODE="${2:-appliquer}"
FICHIER_JETON="${FICHIER_JETON:-$HOME/.qvl/gh-token}"
NOM_REGLES="protection-main"
NOM_REGLES_ENV="protection-env-demo"
NOM_REGLES_ECRITURE="ecriture-env-demo"
ENVIRONNEMENT="publication"
ID_GITHUB_ACTIONS=15368

if [[ ! "$DEPOT" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
  echo "usage : $0 <proprietaire>/<depot> [--relever]" >&2
  exit 2
fi
if [[ ! -r "$FICHIER_JETON" ]]; then
  echo "jeton illisible : $FICHIER_JETON" >&2
  exit 1
fi
command -v gh > /dev/null || { echo "gh (GitHub CLI) est requis" >&2; exit 1; }

api() {
  GH_TOKEN="$(< "$FICHIER_JETON")" GH_PROMPT_DISABLED=1 gh api \
    -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" "$@"
}

etape() { printf '\n== %s\n' "$1"; }

appliquer() {
  etape "Dépôt : détection de secrets à la poussée, fusion automatique, suppression des branches fusionnées"
  api --method PATCH "repos/$DEPOT" --input - --silent <<'JSON'
{
  "delete_branch_on_merge": true,
  "allow_auto_merge": true,
  "security_and_analysis": {
    "secret_scanning": { "status": "enabled" },
    "secret_scanning_push_protection": { "status": "enabled" }
  }
}
JSON

  etape "Alertes et mises à jour de sécurité Dependabot"
  api --method PUT "repos/$DEPOT/vulnerability-alerts" --silent
  api --method PUT "repos/$DEPOT/automated-security-fixes" --silent

  etape "Actions : droits par défaut en lecture, actions GitHub et liste blanche, épinglage par empreinte imposé"
  api --method PUT "repos/$DEPOT/actions/permissions/workflow" --input - --silent <<'JSON'
{ "default_workflow_permissions": "read", "can_approve_pull_request_reviews": false }
JSON
  api --method PUT "repos/$DEPOT/actions/permissions" --input - --silent <<'JSON'
{ "enabled": true, "allowed_actions": "selected", "sha_pinning_required": true }
JSON
  api --method PUT "repos/$DEPOT/actions/permissions/selected-actions" --input - --silent <<'JSON'
{
  "github_owned_allowed": true,
  "verified_allowed": false,
  "patterns_allowed": ["golangci/golangci-lint-action@*"]
}
JSON
  api --method PUT "repos/$DEPOT/actions/permissions/fork-pr-contributor-approval" --input - --silent <<'JSON'
{ "approval_policy": "all_external_contributors" }
JSON

  etape "Ensemble de règles « $NOM_REGLES » : PR obligatoire, ci-ok requise, ni poussée forcée ni suppression, aucun contournement"
  local corps id
  corps="$(cat <<JSON
{
  "name": "$NOM_REGLES",
  "target": "branch",
  "enforcement": "active",
  "bypass_actors": [],
  "conditions": { "ref_name": { "include": ["~DEFAULT_BRANCH"], "exclude": [] } },
  "rules": [
    { "type": "deletion" },
    { "type": "non_fast_forward" },
    { "type": "pull_request", "parameters": {
        "required_approving_review_count": 0,
        "dismiss_stale_reviews_on_push": false,
        "require_code_owner_review": false,
        "require_last_push_approval": false,
        "required_review_thread_resolution": false,
        "allowed_merge_methods": ["merge", "squash", "rebase"] } },
    { "type": "required_status_checks", "parameters": {
        "strict_required_status_checks_policy": false,
        "do_not_enforce_on_create": false,
        "required_status_checks": [ { "context": "ci-ok", "integration_id": $ID_GITHUB_ACTIONS } ] } }
  ]
}
JSON
)"
  id="$(api "repos/$DEPOT/rulesets" --jq ".[] | select(.name == \"$NOM_REGLES\") | .id")"
  if [[ -n "$id" ]]; then
    api --method PUT "repos/$DEPOT/rulesets/$id" --input - --silent <<< "$corps"
    echo "ensemble de règles mis à jour (id $id)"
  else
    api --method POST "repos/$DEPOT/rulesets" --input - --silent <<< "$corps"
    echo "ensemble de règles créé"
  fi

  regles_env_demo
}

ensemble_de_regles() {
  local nom="$1" corps="$2" id
  id="$(api "repos/$DEPOT/rulesets" --jq ".[] | select(.name == \"$nom\") | .id")"
  if [[ -n "$id" ]]; then
    api --method PUT "repos/$DEPOT/rulesets/$id" --input - --silent <<< "$corps"
  else
    api --method POST "repos/$DEPOT/rulesets" --input - --silent <<< "$corps"
  fi
}

regles_env_demo() {
  etape "« $NOM_REGLES_ENV » : env/demo ni forcée ni supprimée, sans aucun contournement (MP5)"
  ensemble_de_regles "$NOM_REGLES_ENV" "$(cat <<JSON
{
  "name": "$NOM_REGLES_ENV",
  "target": "branch",
  "enforcement": "active",
  "bypass_actors": [],
  "conditions": { "ref_name": { "include": ["refs/heads/env/demo"], "exclude": [] } },
  "rules": [ { "type": "deletion" }, { "type": "non_fast_forward" } ]
}
JSON
)"

  etape "« $NOM_REGLES_ECRITURE » : création et mise à jour d'env/demo par la seule clé de déploiement (MP5)"
  ensemble_de_regles "$NOM_REGLES_ECRITURE" "$(cat <<JSON
{
  "name": "$NOM_REGLES_ECRITURE",
  "target": "branch",
  "enforcement": "active",
  "bypass_actors": [ { "actor_id": null, "actor_type": "DeployKey", "bypass_mode": "always" } ],
  "conditions": { "ref_name": { "include": ["refs/heads/env/demo"], "exclude": [] } },
  "rules": [ { "type": "creation" }, { "type": "update", "parameters": { "update_allows_fetch_and_merge": false } } ]
}
JSON
)"

  etape "Environnement « $ENVIRONNEMENT » : déploiements depuis main seulement"
  api --method PUT "repos/$DEPOT/environments/$ENVIRONNEMENT" --input - --silent <<'JSON'
{ "deployment_branch_policy": { "protected_branches": false, "custom_branch_policies": true } }
JSON
  if ! api "repos/$DEPOT/environments/$ENVIRONNEMENT/deployment-branch-policies" --jq '.branch_policies[].name' | grep -qx main; then
    api --method POST "repos/$DEPOT/environments/$ENVIRONNEMENT/deployment-branch-policies" \
      -f name=main -f type=branch --silent
  fi
}

cle_env_demo() {
  local travail
  command -v ssh-keygen > /dev/null || { echo "ssh-keygen est requis" >&2; exit 1; }
  travail="$(mktemp -d)"
  trap 'rm -rf "$travail"' RETURN
  etape "Clé de déploiement d'env/demo : nouvelle paire, publique sur le dépôt, privée dans l'environnement $ENVIRONNEMENT"
  ssh-keygen -q -t ed25519 -N '' -C "env-demo-$(date -u +%Y%m%d)" -f "$travail/cle"
  api "repos/$DEPOT/keys" --jq '.[] | select(.title | startswith("env-demo")) | .id' |
    while read -r id; do api --method DELETE "repos/$DEPOT/keys/$id" --silent; done
  api --method POST "repos/$DEPOT/keys" -f title="env-demo-$(date -u +%Y%m%d)" \
    -f key="$(cat "$travail/cle.pub")" -F read_only=false --silent
  GH_TOKEN="$(< "$FICHIER_JETON")" GH_PROMPT_DISABLED=1 gh secret set ENV_DEMO_CLE \
    --repo "$DEPOT" --env "$ENVIRONNEMENT" < "$travail/cle"
  echo "clé posée ; aucune copie conservée sur le poste"
}

relever() {
  etape "Relevé des réglages de $DEPOT ($(date -u +%Y-%m-%dT%H:%M:%SZ))"
  api "repos/$DEPOT" --jq '{visibility, default_branch, delete_branch_on_merge, allow_auto_merge,
    secret_scanning: .security_and_analysis.secret_scanning.status,
    push_protection: .security_and_analysis.secret_scanning_push_protection.status}'
  api "repos/$DEPOT/actions/permissions/workflow"
  api "repos/$DEPOT/actions/permissions"
  api "repos/$DEPOT/actions/permissions/selected-actions"
  api "repos/$DEPOT/actions/permissions/fork-pr-contributor-approval"
  if api "repos/$DEPOT/vulnerability-alerts" --silent 2> /dev/null; then
    echo '{"dependabot_alerts":"enabled"}'
  else
    echo '{"dependabot_alerts":"disabled"}'
  fi
  api "repos/$DEPOT/automated-security-fixes"
  local id
  id="$(api "repos/$DEPOT/rulesets" --jq ".[] | select(.name == \"$NOM_REGLES\") | .id")"
  if [[ -n "$id" ]]; then
    api "repos/$DEPOT/rulesets/$id" --jq '{name, enforcement, bypass_actors, conditions, rules}'
  else
    echo "ensemble de règles « $NOM_REGLES » absent"
  fi
  api "repos/$DEPOT/rulesets" --jq ".[] | select(.name == \"$NOM_REGLES_ENV\" or .name == \"$NOM_REGLES_ECRITURE\") | .id" |
    while read -r id; do api "repos/$DEPOT/rulesets/$id" --jq '{name, enforcement, bypass_actors, conditions, rules}'; done
  api "repos/$DEPOT/environments/$ENVIRONNEMENT" --jq '{name, deployment_branch_policy}' || echo "environnement $ENVIRONNEMENT absent"
}

case "$MODE" in
  appliquer) appliquer; relever ;;
  --relever) relever ;;
  --cle-env-demo) cle_env_demo ;;
  *) echo "mode inconnu : $MODE" >&2; exit 2 ;;
esac
