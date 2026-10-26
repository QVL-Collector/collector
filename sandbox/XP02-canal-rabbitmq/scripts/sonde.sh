#!/usr/bin/env bash
# Sonde des droits et de la mise a l'ecart, cote broker, independante des programmes de go/.
#   refus   : XC2 et XC9 hors reseau (tentatives interdites par identite, user-id usurpe, guest,
#             API de gestion) + temoins autorises.
#   ecartes : partie broker de XC4 (5 livraisons puis .ecartes, message suivant non bloque).
# Resultats : CSV + detail dans docs/mesures/XP02-canal-rabbitmq/.
# Usage : PLATEFORME=compose|k3d scripts/sonde.sh refus|ecartes
# A lancer AVANT le producteur et le consommateur de go/ : la sonde lit controle.demandes.
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
charger_env

scenario="${1:-}"
[[ "$scenario" == refus || "$scenario" == ecartes ]] || echec "usage : $0 refus|ecartes"

csv_init
sortie="$SORTIES_DIR/$(horodatage)-$PLATEFORME-$scenario.txt"
VARIABLES_MDP=(MDP_SVC_PDM MDP_SVC_CONTROLE MDP_SVC_INTERACTIONS MDP_SVC_TEST MDP_KEDA_MONITORING MDP_ADMIN_CANAL)

lancer_sonde() {
  case "$PLATEFORME" in
    compose)
      local args=() v
      for v in "${VARIABLES_MDP[@]}"; do args+=(-e "$v"); done
      docker run --rm --network xp02_canal "${args[@]}" -e AMQP_HOTE=rabbitmq "$SONDE_IMAGE" "$scenario"
      ;;
    k3d)
      "$XP_DIR/scripts/k3d.sh" sonde "$scenario"
      ;;
    *) echec "PLATEFORME inconnue : $PLATEFORME" ;;
  esac
}

api_gestion() {
  # Identites de service refusees (aucune etiquette) ; supervision (KEDA) et administration acceptees.
  local base="$1" u var code attendu critere
  for u in "${SERVICES[@]}" keda-monitoring admin-canal; do
    var="$(variable_mdp "$u")"
    code="$(curl -s -o "$POUBELLE" -w '%{http_code}' --max-time 5 -K - <<<"user = \"$u:${!var}\"" \
      "$base/api/queues/collector/controle.demandes" || true)"
    case "$u" in svc-*) attendu=401; critere=XC9 ;; *) attendu=200; critere=TEMOIN ;; esac
    printf '%s|api-gestion:/api/queues/collector/controle.demandes|%s|%s|%s|%s\n' \
      "$critere" "$u" "$attendu" "$code" "$([[ "$code" == "$attendu" ]] && echo oui || echo non)"
  done
}

base_gestion="http://127.0.0.1:15672"
if [[ "$PLATEFORME" == k3d && "$scenario" == refus ]]; then
  # API de gestion par port-forward seulement (MC4), depuis le poste.
  kubectl port-forward -n collector-canal svc/rabbitmq 15673:15672 >/dev/null 2>&1 &
  pf=$!
  trap 'kill "$pf" 2>/dev/null || true' EXIT
  sleep 3
  base_gestion="http://127.0.0.1:15673"
fi

{
  lancer_sonde || true
  [[ "$scenario" == refus ]] && api_gestion "$base_gestion"
  true
} >"$sortie"

total=0; ok=0
while IFS='|' read -r critere cas identite attendu obtenu atteint; do
  [[ -n "${atteint:-}" ]] || continue
  total=$((total + 1)); [[ "$atteint" == oui ]] && ok=$((ok + 1))
  csv "$critere" "sonde-$scenario" "$obtenu" "resultat" "$attendu" "$atteint" "$identite $cas"
done <"$sortie"

journal "Sonde $scenario : $ok / $total conformes (detail : $sortie)"
column -t -s '|' "$sortie" >&2 || cat "$sortie" >&2
[[ $total -gt 0 && $ok -eq $total ]]
