#!/usr/bin/env bash
# G6 : tir constant sur 3 réplicas, suppression forcée d'un pod au milieu de la fenêtre, délai du remplaçant Ready.
set -euo pipefail
cd "$(dirname "$0")/.."
k="kubectl --context ${CONTEXTE:-collector-xp01}"
$k patch hpa xp01-test --type merge -p '{"spec":{"minReplicas":3,"maxReplicas":3}}' >/dev/null
until [ "$($k get deploy xp01-test -o jsonpath='{.status.readyReplicas}')" = "3" ]; do sleep 2; done
$k rollout status deploy/xp01-test --timeout=120s >/dev/null
k6 run -q -e DEBIT="${DEBIT:-4}" -e MONTEE=10s -e PLATEAU=170s -e DESCENTE=1s --summary-export .session/g6-resume.json charge/g5.js > .session/g6-k6.txt 2>&1 &
K6=$!
sleep 90
avant=$($k get pods -l app=xp01-test -o jsonpath='{.items[*].metadata.name}')
cible=${avant%% *}
t0=$(date +%s)
echo "suppression forcée de $cible à $(date -u +%H:%M:%S)"
$k delete pod "$cible" --force --grace-period=0 >/dev/null 2>&1
pret_nouveau() {
  $k get pods -l app=xp01-test --no-headers -o custom-columns=N:.metadata.name,R:.status.containerStatuses[0].ready |
    awk '$2=="true"{print $1}' | grep -qvxF -f <(tr ' ' '
' <<<"$avant")
}
until pret_nouveau; do sleep 1; done
echo "remplaçant Ready en $(( $(date +%s) - t0 )) s"
wait $K6 || true
grep -E "http_req_failed|http_req_duration|iterations\b" .session/g6-k6.txt
