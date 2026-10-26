#!/usr/bin/env bash
# Porte gitleaks (D10, ES-06, US-003 AC1) : diff de la PR en pull_request, historique complet du commit analysé sinon
# (HEAD seul : par défaut gitleaks lit toutes les références, une branche de démonstration rougirait main).
# Tout résultat est bloquant. Valeurs masquées (--redact) ; détail en SARIF dans l'onglet sécurité.
#
# Entrées : EVENEMENT ; BASE et TETE en pull_request.

set -euo pipefail
export MSYS_NO_PATHCONV=1

RACINE="$(git rev-parse --show-toplevel)"
cd "$RACINE"
mkdir -p rapports
chmod 0777 rapports

EVENEMENT="${EVENEMENT:-pull_request}"
ARGS=(git --redact --no-banner --exit-code 1 --config /depot/.gitleaks.toml
      --report-format sarif --report-path /rapports/gitleaks.sarif)
if [[ "$EVENEMENT" == "pull_request" ]]; then
  : "${BASE:?BASE requis en pull_request}" "${TETE:?TETE requis en pull_request}"
  ARGS+=(--log-opts "$BASE..$TETE")
  PORTEE="diff de la PR"
else
  ARGS+=(--log-opts "HEAD")
  PORTEE="historique complet"
fi

STATUT=0
docker compose -f tools/ci/outils.yml run --rm -T gitleaks "${ARGS[@]}" /depot || STATUT=$?

jq --arg portee "$PORTEE" '{outil: "gitleaks", portee: $portee, secrets: ([.runs[]?.results[]?] | length)} | .bloquants = .secrets' \
  < rapports/gitleaks.sarif > rapports/i6-secrets.json
cat rapports/i6-secrets.json

if (( STATUT != 0 )); then
  echo "::error::gitleaks ($PORTEE) : secret détecté ; appliquer la procédure de fuite (PILOTAGE § 7)"
  exit 1
fi
echo "gitleaks ($PORTEE) : aucun secret."
