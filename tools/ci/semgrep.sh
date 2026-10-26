#!/usr/bin/env bash
# Porte Semgrep (D10, ES-05, US-003 AC4) : règles versionnées du projet (tools/ci/semgrep/regles.yml), bloquant sur « ERROR ».
# Les règles sont d'abord testées sur leurs cas d'essai (tools/ci/semgrep). Les exceptions en ligne (nosemgrep) sont
# désactivées : aucune exception sans entrée datée (MQ3). Journal public : compteur seul ; détail en SARIF.

set -euo pipefail
export MSYS_NO_PATHCONV=1

RACINE="$(git rev-parse --show-toplevel)"
cd "$RACINE"
mkdir -p rapports
chmod 0777 rapports

SEMGREP=(docker compose -f tools/ci/outils.yml run --rm -T semgrep semgrep)

"${SEMGREP[@]}" --test --metrics=off tools/ci/semgrep

STATUT=0
"${SEMGREP[@]}" scan --config tools/ci/semgrep/regles.yml --severity ERROR --error --disable-nosem --metrics=off --quiet \
  --sarif-output=/rapports/semgrep.sarif --json-output=/rapports/semgrep.json . > /dev/null || STATUT=$?

jq '{outil: "semgrep", constats: (.results | length), erreurs_outil: (.errors | length)} | .bloquants = .constats' \
  < rapports/semgrep.json > rapports/i6-semgrep.json
cat rapports/i6-semgrep.json
rm -f rapports/semgrep.json

if (( STATUT != 0 )); then
  echo "::error::Semgrep : constat(s) de sévérité ERROR ou erreur d'analyse ; détail dans l'onglet sécurité du dépôt"
  exit 1
fi
echo "Semgrep : aucun constat bloquant."
