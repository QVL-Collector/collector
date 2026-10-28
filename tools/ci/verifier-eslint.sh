#!/usr/bin/env bash
# Vérifie que la configuration ESLint du front active en erreur react/no-danger et au moins une règle jsx-a11y (D04, US-002 AC2).
# À lancer depuis web/, après npm ci. Échoue si la configuration résolue ne contient pas ces règles.

set -euo pipefail

FICHIER="$(git ls-files -- 'src/*.tsx' 'src/**/*.tsx' | awk 'NR == 1')"
if [[ -z "$FICHIER" ]]; then
  echo "::error::aucun fichier .tsx suivi sous web/src : impossible de résoudre la configuration ESLint"
  exit 1
fi

CONFIG="$(npx --no-install eslint --print-config "$FICHIER")"

if ! jq -e '
  def niveau: (if type == "array" then .[0] else . end) | tostring;
  def active: . == "2" or . == "error";
  ((.rules["react/no-danger"] // "off") | niveau | active)
  and ([.rules | to_entries[] | select(.key | startswith("jsx-a11y/")) | .value | niveau | select(active)] | length > 0)
' <<< "$CONFIG" > /dev/null; then
  echo "::error::ESLint doit activer en erreur react/no-danger et les règles jsx-a11y (fichier de référence : $FICHIER)"
  exit 1
fi

echo "Règles obligatoires actives (react/no-danger, jsx-a11y) pour $FICHIER."
