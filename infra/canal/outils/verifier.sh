#!/usr/bin/env bash
# Verification du canal apres deploiement (US-011) :
#   droits  : AC2 (publication acceptee et livree), AC4 (sujets d'un autre refuses, user-id usurpe),
#             AC5 (declaration, abonnement, purge, suppression refusees), guest absent (MC1) ;
#   ecartes : MC8 (5 livraisons puis .ecartes, message suivant non bloque) ;
#   gestion : API de gestion refusee aux identites de service, ouverte a keda et admin-canal (MC4) ;
#   reseau  : (kubernetes) AC7, MC3 depuis le namespace d'un service et depuis un namespace intrus ;
#   fuites  : AC6 (X20), aucun mot de passe de la session dans le depot, Helm ou les journaux.
# Chaque execution de la sonde compte comme un cas ; le nombre de cas attendu est fixe par cible.
# Usage : outils/verifier.sh local|kubernetes
# La sonde tourne dans un conteneur Python avec pika (SONDE_IMAGE, construite depuis verification/).
# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

CIBLE="${1:-}"
case "$CIBLE" in
  local) CAS_ATTENDUS=64 ;;
  kubernetes) CAS_ATTENDUS=79 ;;
  *) echec "usage : $0 local|kubernetes" ;;
esac
SONDE_IMAGE="${SONDE_IMAGE:-collector/canal-sonde:1}"
NS_CANAL="${NS_CANAL:-canal}"
NS_SONDE="$(namespace_de marche)"
NS_INTRUS="canal-intrus"
ETIQUETTE_CANAL="app.kubernetes.io/part-of=collector-canal"
ETIQUETTE_VERIF="app.kubernetes.io/part-of=collector-canal-verification"
if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == MSYS* ]]; then POUBELLE=NUL; else POUBELLE=/dev/null; fi
charger_mdp

kc() { kubectl ${KUBE_CONTEXT:+--context "$KUBE_CONTEXT"} "$@"; }
hm() { helm ${KUBE_CONTEXT:+--kube-context "$KUBE_CONTEXT"} "$@"; }

total=0
ok=0
compter() {
  local ac nom id attendu obtenu atteint
  while IFS='|' read -r ac nom id attendu obtenu atteint; do
    [[ -n "${atteint:-}" ]] || { [[ -z "$ac" ]] || printf '%s\n' "$ac" >&2; continue; }
    total=$((total + 1))
    [[ "$atteint" == oui ]] && ok=$((ok + 1))
    printf '%-4s %-3s %-58s %-15s attendu=%-7s obtenu=%s\n' "$ac" "$atteint" "$nom" "$id" "$attendu" "$obtenu" >&2
  done
}

verdict() { [[ "$2" == "$3" ]] && printf 'oui' || printf 'non'; }

executer_sonde() {
  local nom="$1" sortie statut=0
  shift
  sortie="$("$@" 2>&1)" || statut=$?
  compter <<<"$sortie"
  compter <<<"SONDE|execution:$nom|-|0|$statut|$(verdict x "$statut" 0)"
}

image_sonde() {
  docker image inspect "$SONDE_IMAGE" >/dev/null 2>&1 && return
  journal "Construction de $SONDE_IMAGE"
  docker build -q -t "$SONDE_IMAGE" "$(chemin_hote "$CANAL_DIR/verification")" >/dev/null
}

variables_mdp() {
  local u
  for u in "${UTILISATEURS[@]}"; do variable_mdp "$u"; printf '\n'; done
}

sonde_local() {
  local args=() v
  while read -r v; do args+=(-e "$v"); done < <(variables_mdp)
  docker run --rm --network collector_canal "${args[@]}" -e AMQP_HOTE=rabbitmq \
    -v "$(chemin_hote "$CANAL_DIR/verification/sonde.py"):/verif/sonde.py:ro" \
    --entrypoint python "$SONDE_IMAGE" -I /verif/sonde.py "$@"
}

pod_sonde() {
  local ns="$1" nom phase='' a liste=''
  shift
  nom="canal-sonde-$(openssl rand -hex 3)"
  for a in "$@"; do liste+="\"$a\", "; done
  kc -n "$ns" create configmap "$nom" --from-file=sonde.py="$(chemin_hote "$CANAL_DIR/verification/sonde.py")" >/dev/null
  kc -n "$ns" label configmap "$nom" "$ETIQUETTE_VERIF" >/dev/null
  kc apply -f - >/dev/null <<YAML
apiVersion: v1
kind: Pod
metadata:
  name: $nom
  namespace: $ns
  labels: { ${ETIQUETTE_VERIF/=/: } }
spec:
  restartPolicy: Never
  automountServiceAccountToken: false
  securityContext: { runAsNonRoot: true, runAsUser: 65534, runAsGroup: 65534, seccompProfile: { type: RuntimeDefault } }
  containers:
    - name: sonde
      image: $SONDE_IMAGE
      imagePullPolicy: IfNotPresent
      command: ["python", "-I", "/verif/sonde.py", ${liste%, }]
      env: [{ name: ORIGINE, value: "$ns" }, { name: AMQP_HOTE, value: "rabbitmq.$NS_CANAL.svc" }]
      envFrom: [{ secretRef: { name: canal-sonde, optional: true } }]
      resources: { requests: { cpu: 50m, memory: 64Mi }, limits: { memory: 128Mi } }
      securityContext: { allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: { drop: ["ALL"] } }
      volumeMounts: [{ name: verif, mountPath: /verif, readOnly: true }]
  volumes: [{ name: verif, configMap: { name: $nom } }]
YAML
  for _ in $(seq 1 90); do
    phase="$(kc -n "$ns" get pod "$nom" -o jsonpath='{.status.phase}')"
    [[ "$phase" == Succeeded || "$phase" == Failed ]] && break
    sleep 2
  done
  kc -n "$ns" logs "$nom"
  kc -n "$ns" delete pod "$nom" --wait=false >/dev/null
  kc -n "$ns" delete configmap "$nom" >/dev/null
  [[ "$phase" == Succeeded ]]
}

secret_sonde() {
  local v cle val
  {
    printf 'apiVersion: v1\nkind: Secret\ntype: Opaque\nmetadata:\n  name: canal-sonde\n  namespace: %s\n  labels:\n    %s\ndata:\n' \
      "$NS_SONDE" "${ETIQUETTE_CANAL/=/: }"
    while read -r v; do printf '%s %s\n' "$v" "$(printf '%s' "${!v}" | base64 -w0)"; done < <(variables_mdp) |
      while read -r cle val; do printf '  %s: %s\n' "$cle" "$val"; done
  } | kc apply -f - >/dev/null
}

nettoyer_kubernetes() {
  [[ -z "${pf:-}" ]] || kill "$pf" 2>/dev/null || true
  kc -n "$NS_SONDE" delete secret canal-sonde --ignore-not-found >/dev/null 2>&1 || true
  kc -n "$NS_SONDE" delete pod,configmap -l "$ETIQUETTE_VERIF" --ignore-not-found >/dev/null 2>&1 || true
  kc delete namespace "$NS_INTRUS" --ignore-not-found --wait=false >/dev/null 2>&1 || true
}

gestion() {
  local base="$1" u var code attendu
  for u in "${UTILISATEURS[@]}"; do
    var="$(variable_mdp "$u")"
    code="$(curl -s -o "$POUBELLE" -w '%{http_code}' --max-time 5 -K - <<<"user = \"$u:${!var}\"" \
      "$base/api/queues/$VHOST/controle.demandes" || true)"
    case "$u" in keda | admin-canal) attendu=200 ;; *) attendu=401 ;; esac
    printf 'MC4|api-gestion:/api/queues/%s/controle.demandes|%s|%s|%s|%s\n' "$VHOST" "$u" "$attendu" "$code" \
      "$(verdict x "$code" "$attendu")"
  done
}

fuite() {
  local nom="$1" resultat compte statut obtenu
  shift
  resultat="$({ "$@" 2>&1 | grep -c -F -f <(sed -n 's/^[A-Z_]*=//p' "$MDP_FILE"); printf '%s' "${PIPESTATUS[0]}"; } | tr '\n' ' ')"
  read -r compte statut <<<"$resultat"
  if [[ "$statut" != 0 ]]; then
    obtenu="source-illisible"
  elif [[ "$compte" == 0 ]]; then
    obtenu=absent
  else
    obtenu="present($compte)"
  fi
  printf 'AC6|absence-secret:%s|-|absent|%s|%s\n' "$nom" "$obtenu" "$(verdict x "$obtenu" absent)"
}

depot_suivi() { git -C "$(chemin_hote "$DEPOT_DIR")" grep -h -I -e '' -- .; }

services_internes() {
  local types r
  types="$(kc -n "$NS_CANAL" get svc -o jsonpath='{.items[*].spec.type}')"
  if [[ -n "$types" ]] && ! grep -q -E 'NodePort|LoadBalancer' <<<"$types"; then r=oui; else r=non; fi
  printf 'AC7|services-du-broker:ClusterIP-seul|-|oui|%s|%s\n' "$r" "$r"
}

image_sonde
if [[ "$CIBLE" == local ]]; then
  executer_sonde droits sonde_local droits
  executer_sonde ecartes sonde_local ecartes
  compter <<<"$(gestion "http://127.0.0.1:15672")"
  compter <<<"$(fuite fichiers-suivis-du-depot depot_suivi)"
  compter <<<"$(fuite journal-du-broker docker logs collector-canal-rabbitmq-1)"
else
  trap nettoyer_kubernetes EXIT
  contexte="$(kc config current-context)"
  if [[ "$contexte" == k3d-* ]]; then
    k3d image import -c "${contexte#k3d-}" "$SONDE_IMAGE" >/dev/null 2>&1 || journal "image de sonde non importee dans k3d"
  fi
  kc delete namespace "$NS_INTRUS" --ignore-not-found --wait=true --timeout=120s >/dev/null
  printf 'apiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n  labels:\n    pod-security.kubernetes.io/enforce: restricted\n' "$NS_INTRUS" |
    kc apply -f - >/dev/null
  svc="rabbitmq.$NS_CANAL.svc"
  noeud="rabbitmq-0.rabbitmq-noeuds.$NS_CANAL.svc"
  kc -n "$NS_CANAL" port-forward svc/rabbitmq 15673:15672 >/dev/null 2>&1 &
  pf=$!
  sleep 3
  secret_sonde
  executer_sonde droits pod_sonde "$NS_SONDE" droits
  executer_sonde ecartes pod_sonde "$NS_SONDE" ecartes
  kc -n "$NS_SONDE" delete secret canal-sonde --ignore-not-found >/dev/null
  executer_sonde reseau-service pod_sonde "$NS_SONDE" reseau \
    "$svc:5672:ouvert" "$svc:15672:ferme" "$svc:15692:ferme" "$noeud:4369:ferme" "$noeud:25672:ferme"
  executer_sonde reseau-intrus pod_sonde "$NS_INTRUS" reseau \
    "$svc:5672:ferme" "$svc:15672:ferme" "$svc:15692:ferme" "$noeud:4369:ferme" "$noeud:25672:ferme"
  compter <<<"$(gestion "http://127.0.0.1:15673")"
  compter <<<"$(services_internes)"
  compter <<<"$(fuite fichiers-suivis-du-depot depot_suivi)"
  compter <<<"$(fuite manifeste-helm hm -n "$NS_CANAL" get manifest canal)"
  compter <<<"$(fuite valeurs-helm hm -n "$NS_CANAL" get values canal --all)"
  compter <<<"$(fuite journal-du-broker kc -n "$NS_CANAL" logs rabbitmq-0 -c rabbitmq)"
fi

journal "Verification $CIBLE : $ok / $total conformes, $CAS_ATTENDUS cas attendus"
[[ $total -eq $CAS_ATTENDUS && $ok -eq $total ]]
