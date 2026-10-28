#!/usr/bin/env bash
# Calcule les jobs de CI à exécuter d'après les chemins modifiés (filtre maison, sans action tierce : D10, MP3).
#
# Entrées (variables d'environnement) :
#   EVENEMENT  nom de l'événement GitHub (pull_request, push, schedule, workflow_dispatch)
#   BASE, TETE empreintes de commit comparées en pull_request (diff depuis la base commune)
# Sorties (dans $GITHUB_OUTPUT, sinon sur la sortie standard) :
#   modules_go  tableau JSON des modules Go concernés (services/<service>, libs/<module>, infra/<module>)
#   go          true si au moins un module Go est concerné
#   web         true si le front existe et est concerné
#   outillage   true si les workflows ou les scripts sont concernés
#   analyse     true si un fichier hors documentation est concerné (Semgrep, Trivy config et dépendances)
#
# Règles : services/<service>/** et infra/<module>/** déclenchent ce module ; libs/**, contracts/** et l'outillage de CI
# déclenchent tous les modules ; hors pull_request (push sur main, planification), tout s'exécute.

set -euo pipefail

RACINE="$(git rev-parse --show-toplevel)"
cd "$RACINE"

EVENEMENT="${EVENEMENT:-pull_request}"
MOTIF_NOM='^[a-z0-9][a-z0-9._-]*$'

ecrire_sortie() {
  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    printf '%s=%s\n' "$1" "$2" >> "$GITHUB_OUTPUT"
  else
    printf '%s=%s\n' "$1" "$2"
  fi
}

lister_modules_go() {
  local fichier dossier nom
  for fichier in services/*/go.mod libs/*/go.mod infra/*/go.mod; do
    [[ -f "$fichier" ]] || continue
    dossier="${fichier%/go.mod}"
    nom="${dossier#*/}"
    if [[ ! "$nom" =~ $MOTIF_NOM ]]; then
      echo "::error::nom de module refusé : $dossier (attendu : minuscules, chiffres, . _ -)" >&2
      exit 1
    fi
    printf '%s\n' "$dossier"
  done
}

LISTE_MODULES="$(lister_modules_go)"
TOUS_MODULES=()
[[ -z "$LISTE_MODULES" ]] || mapfile -t TOUS_MODULES <<< "$LISTE_MODULES"
WEB_EXISTE=false
[[ -f web/package.json && -f web/package-lock.json ]] && WEB_EXISTE=true

TOUT=false
FICHIERS=()
if [[ "$EVENEMENT" == "pull_request" ]]; then
  : "${BASE:?BASE requis en pull_request}" "${TETE:?TETE requis en pull_request}"
  mapfile -t FICHIERS < <(git diff --name-only "$BASE...$TETE")
else
  TOUT=true
fi

GLOBAL_GO=false
WEB=false
OUTILLAGE=false
ANALYSE=false
declare -A MODULES_TOUCHES=()

for f in "${FICHIERS[@]}"; do
  case "$f" in
    .github/*|tools/ci/*|tools/git-hooks/*|.golangci.yml)
      OUTILLAGE=true; GLOBAL_GO=true; WEB=true ;;
    libs/*)
      GLOBAL_GO=true ;;
    contracts/*)
      GLOBAL_GO=true; WEB=true ;;
    web/*)
      WEB=true ;;
  esac
  case "$f" in
    *.sh|tools/*) OUTILLAGE=true ;;
  esac
  case "$f" in
    docs/*|backlog/*|presentation/*|*.md) ;;
    *) ANALYSE=true ;;
  esac
  if [[ "$f" == services/*/* || "$f" == infra/*/* ]]; then
    racine="${f%%/*}"
    reste="${f#*/}"
    MODULES_TOUCHES["${racine}/${reste%%/*}"]=1
  fi
done

if [[ "$TOUT" == true ]]; then
  GLOBAL_GO=true; WEB=true; OUTILLAGE=true; ANALYSE=true
fi

MODULES=()
for m in "${TOUS_MODULES[@]}"; do
  if [[ "$GLOBAL_GO" == true || -n "${MODULES_TOUCHES[$m]:-}" ]]; then
    MODULES+=("$m")
  fi
done

JSON="["
for m in "${MODULES[@]}"; do
  [[ "$JSON" == "[" ]] || JSON+=","
  JSON+="\"$m\""
done
JSON+="]"

GO=false
(( ${#MODULES[@]} > 0 )) && GO=true
[[ "$WEB_EXISTE" == true ]] || WEB=false

ecrire_sortie modules_go "$JSON"
ecrire_sortie go "$GO"
ecrire_sortie web "$WEB"
ecrire_sortie outillage "$OUTILLAGE"
ecrire_sortie analyse "$ANALYSE"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo "### Jobs retenus"
    echo ""
    echo "| Périmètre | Exécuté |"
    echo "|---|---|"
    echo "| Modules Go | ${JSON} |"
    echo "| Front (web/) | ${WEB} |"
    echo "| Workflows et scripts | ${OUTILLAGE} |"
    echo "| Analyse statique et configuration | ${ANALYSE} |"
  } >> "$GITHUB_STEP_SUMMARY"
fi
