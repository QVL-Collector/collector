#!/usr/bin/env bash
# Vérifie les sommes des dépendances téléchargées du module courant (go mod verify).
#
# Usage, depuis le dossier du module : tools/ci/verifier-modules.sh
# Les modules du dépôt reliés par une directive replace locale n'ont pas d'archive dans le cache :
# go mod verify les signale « missing ziphash ». Ces seules lignes sont ignorées ; toute autre erreur fait échouer.

set -euo pipefail

locaux="$(go list -m -f '{{if .Replace}}{{if not .Replace.Version}}{{.Path}}{{end}}{{end}}' all)"

if sortie="$(go mod verify 2>&1)"; then
  echo "$sortie"
  exit 0
fi

restantes="$(printf '%s\n' "$sortie" | while IFS= read -r ligne; do
  module="${ligne%% *}"
  if [[ "$ligne" == *"missing ziphash"* ]] && grep -qxF "$module" <<< "$locaux"; then
    continue
  fi
  printf '%s\n' "$ligne"
done)"

if [[ -n "$restantes" ]]; then
  printf '%s\n' "$restantes" >&2
  exit 1
fi

echo "all modules verified (modules locaux du dépôt exclus : $(tr '\n' ' ' <<< "$locaux"))"
