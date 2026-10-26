#!/usr/bin/env bash
# Porte Trivy (D10, ES-05, US-003 AC3 et AC5) : analyse, rapport SARIF, compteurs I6, décision bloquante.
#
# Usage : tools/ci/trivy.sh <fs|config|image> <cible> <nom>
#   fs      dépendances d'un verrou (web/) ; bloque une vulnérabilité critique ou élevée corrigeable
#   config  configuration en code (infra/, Dockerfile, manifests, charts) ; bloque une mauvaise configuration critique ou élevée
#   image   image construite ; bloque une vulnérabilité critique ou élevée corrigeable et tout secret embarqué
# Exceptions : .trivyignore (validé par tools/ci/verifier-exceptions.sh).
# Journal public : seuls des compteurs sont affichés (MQ5) ; le détail part dans l'onglet sécurité (SARIF).

set -euo pipefail
export MSYS_NO_PATHCONV=1

MODE="${1:?mode attendu : fs, config ou image}"
CIBLE="${2:?cible attendue}"
NOM="${3:?nom de rapport attendu}"

RACINE="$(git rev-parse --show-toplevel)"
cd "$RACINE"
mkdir -p rapports .cache/trivy
chmod 0777 rapports

case "$MODE" in
  fs) ARGS=(fs --scanners vuln) ;;
  config) ARGS=(config --skip-dirs sandbox --skip-dirs rapports --skip-dirs .cache --skip-dirs node_modules) ;;
  image) ARGS=(image --scanners "vuln,secret") ;;
  *) echo "mode inconnu : $MODE" >&2; exit 2 ;;
esac

TRIVY=(docker compose -f tools/ci/outils.yml run --rm -T trivy)
JSON="rapports/trivy-$NOM.json"

"${TRIVY[@]}" "${ARGS[@]}" --quiet --format json --output "/$JSON" "$CIBLE"
"${TRIVY[@]}" convert --quiet --format sarif --output "/rapports/trivy-$NOM.sarif" "/$JSON"

jq --arg outil "trivy-$NOM" '
  def par_severite: group_by(.Severity) | map({key: .[0].Severity, value: length}) | from_entries;
  def grave: .Severity == "CRITICAL" or .Severity == "HIGH";
  [.Results[]?.Vulnerabilities[]?] as $v
  | [.Results[]?.Misconfigurations[]? | select(.Status == "FAIL")] as $m
  | [.Results[]?.Secrets[]?] as $s
  | {
      outil: $outil,
      vulnerabilites: ($v | par_severite),
      vulnerabilites_bloquantes: ([$v[] | select(grave and ((.FixedVersion // "") != ""))] | length),
      mauvaises_configurations: ($m | par_severite),
      mauvaises_configurations_bloquantes: ([$m[] | select(grave)] | length),
      secrets: ($s | length)
    }
  | .bloquants = .vulnerabilites_bloquantes + .mauvaises_configurations_bloquantes + .secrets
' < "$JSON" > "rapports/i6-trivy-$NOM.json"

cat "rapports/i6-trivy-$NOM.json"
BLOQUANTS="$(jq '.bloquants' < "rapports/i6-trivy-$NOM.json")"
rm -f "$JSON"

if (( BLOQUANTS > 0 )); then
  echo "::error::Trivy ($MODE, $NOM) : $BLOQUANTS constat(s) bloquant(s) ; détail dans l'onglet sécurité du dépôt"
  exit 1
fi
echo "Trivy ($MODE, $NOM) : aucun constat bloquant."
