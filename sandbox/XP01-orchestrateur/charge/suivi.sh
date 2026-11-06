#!/usr/bin/env bash
# Relève toutes les 5 s l'HPA et les pods du service de test (G5, G6) : horodatage;cpu%;réplicas voulus;pods Ready;pods total
set -u
k="kubectl --context ${CONTEXTE:-collector-xp01} -n ${NS:-default}"
while true; do
  h=$($k get hpa xp01-test -o jsonpath='{.status.currentMetrics[0].resource.current.averageUtilization};{.status.desiredReplicas}' 2>/dev/null)
  p=$($k get pods -l app=xp01-test -o jsonpath='{range .items[*]}{.status.containerStatuses[0].ready}{" "}{end}' 2>/dev/null)
  echo "$(date -u +%H:%M:%S);$h;$(grep -o true <<<"$p" | wc -l);$(wc -w <<<"$p")"
  sleep 5
done
