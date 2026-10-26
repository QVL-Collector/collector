#!/usr/bin/env bash
# XC5 : coupure du broker, puis delai de reconnexion des producteurs et consommateurs de go/.
#   compose : docker restart (etat conserve, equivalent du redemarrage du conteneur)
#   k3d|eks : mode "conteneur" = arret du processus (redemarrage du conteneur, emptyDir conserve)
#             mode "pod"       = kubectl delete pod (emptyDir perdu : filet et rejeu de l'outbox ensuite)
# Mesure : t0 = coupure ; t1 = broker pret ; t2 = toutes les identites attendues reconnectees.
# Seuil : t2 - t1 <= 60 s. Les identites attendues sont celles des programmes de go/ en cours.
# Usage : PLATEFORME=compose scripts/xc5-redemarrage.sh [conteneur|pod] [identite...]
#         (par defaut : conteneur svc-pdm svc-controle)
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

mode="${1:-conteneur}"; shift || true
attendues=("$@"); [[ ${#attendues[@]} -gt 0 ]] || attendues=(svc-pdm svc-controle)
DELAI_MAX_S="${DELAI_MAX_S:-180}"

connectees() { rmq rabbitmqctl -q list_connections user 2>/dev/null | tail -n +2 | sort -u || true; }

toutes_connectees() {
  local liste u
  liste="$(connectees)"
  for u in "${attendues[@]}"; do grep -qx "$u" <<<"$liste" || return 1; done
}

"$XP_DIR/scripts/etat.sh" "avant-xc5-$mode" || true
toutes_connectees || journal "ATTENTION : identites attendues non connectees avant la coupure : ${attendues[*]}"

t0="$(ms_maintenant)"
case "$PLATEFORME:$mode" in
  compose:conteneur) docker restart "$CONTENEUR_COMPOSE" >/dev/null ;;
  k3d:conteneur|eks:conteneur)
    rmq rabbitmqctl stop_app >/dev/null
    # Arret du processus principal : kubelet redemarre le conteneur, l'emptyDir est conserve.
    kubectl exec -n "$NS_BROKER" "$POD_BROKER" -c rabbitmq -- rabbitmqctl shutdown >/dev/null 2>&1 || true
    ;;
  k3d:pod|eks:pod) kubectl delete pod -n "$NS_BROKER" "$POD_BROKER" --wait=false >/dev/null ;;
  *) echec "mode $mode non supporte sur $PLATEFORME" ;;
esac
sleep 3

until broker_pret; do
  (( ($(ms_maintenant) - t0) / 1000 < DELAI_MAX_S )) || echec "broker non pret apres $DELAI_MAX_S s"
  sleep 1
done
t1="$(ms_maintenant)"
journal "Broker pret $(( (t1 - t0) / 1000 )) s apres la coupure"

t2=""
while (( ($(ms_maintenant) - t1) / 1000 < DELAI_MAX_S )); do
  if toutes_connectees; then t2="$(ms_maintenant)"; break; fi
  sleep 1
done

csv XC5 "$mode:pret" "$(( (t1 - t0) / 1000 ))" s "-" "-" "delai coupure -> broker pret"
if [[ -n "$t2" ]]; then
  d=$(( (t2 - t1) / 1000 ))
  atteint=$([[ $d -le 60 ]] && echo oui || echo non)
  csv XC5 "$mode:reconnexion" "$d" s "<=60" "$atteint" "identites : ${attendues[*]}"
  journal "Reconnexion de ${attendues[*]} en $d s apres le retour du broker (seuil 60 s) : $atteint"
else
  csv XC5 "$mode:reconnexion" "absente" s "<=60" non "identites : ${attendues[*]} ; connectees : $(connectees | tr '\n' ' ')"
  journal "Reconnexion non observee en $DELAI_MAX_S s ; connectees : $(connectees | tr '\n' ' ')"
fi
"$XP_DIR/scripts/etat.sh" "apres-xc5-$mode" || true
[[ -n "$t2" ]]
