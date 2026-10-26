#!/usr/bin/env bash
# Porte govulncheck (D10, ES-05, US-003 AC2) : bloque toute vulnérabilité Go atteignable depuis le code du module.
#
# Usage, depuis le dossier du module : tools/ci/govulncheck.sh <nom>
# govulncheck est installé à la version GOVULNCHECK_VERSION par le proxy Go (somme de contrôle vérifiée).
# Journal public : seul le nombre de vulnérabilités atteignables est affiché (MQ5) ; le détail part en SARIF.

set -euo pipefail

NOM="${1:?nom du module attendu}"
VERSION="${GOVULNCHECK_VERSION:?GOVULNCHECK_VERSION requis}"
RACINE="$(git rev-parse --show-toplevel)"
RAPPORTS="$RACINE/rapports"
mkdir -p "$RAPPORTS"

GOFLAGS='' go install "golang.org/x/vuln/cmd/govulncheck@$VERSION"
GOVULNCHECK="$(go env GOPATH)/bin/govulncheck"

"$GOVULNCHECK" -format json ./... > "$RAPPORTS/govulncheck-$NOM.json"
"$GOVULNCHECK" -format sarif ./... > "$RAPPORTS/govulncheck-$NOM.sarif"

jq -s --arg outil "govulncheck-$NOM" '
  [.[] | .finding // empty] as $f
  | ([$f[] | select(.trace[0].function != null) | .osv] | unique) as $a
  | {
      outil: $outil,
      atteignables: ($a | length),
      non_atteignables: (([$f[].osv] | unique) - $a | length)
    }
  | .bloquants = .atteignables
' < "$RAPPORTS/govulncheck-$NOM.json" > "$RAPPORTS/i6-govulncheck-$NOM.json"

cat "$RAPPORTS/i6-govulncheck-$NOM.json"
BLOQUANTS="$(jq '.bloquants' < "$RAPPORTS/i6-govulncheck-$NOM.json")"
rm -f "$RAPPORTS/govulncheck-$NOM.json"

if (( BLOQUANTS > 0 )); then
  echo "::error::govulncheck ($NOM) : $BLOQUANTS vulnérabilité(s) atteignable(s) ; détail dans l'onglet sécurité du dépôt"
  exit 1
fi
echo "govulncheck ($NOM) : aucune vulnérabilité atteignable."
