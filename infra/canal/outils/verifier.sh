#!/usr/bin/env bash
# Verification du canal apres deploiement (US-011) :
#   droits  : AC2 (publication acceptee et livree), AC4 (sujets d'un autre refuses, user-id usurpe),
#             AC5 (declaration, abonnement, purge, suppression refusees), guest absent (MC1) ;
#   ecartes : MC8 (5 livraisons puis .ecartes, message suivant non bloque) ;
#   gestion : API de gestion refusee aux identites de service, ouverte a keda et admin-canal (MC4) ;
#   reseau  : (kubernetes) AC7, MC3 depuis le namespace d'un service et depuis un namespace intrus ;
#   fuites  : AC6 (X20), aucun mot de passe de la session dans le depot, Helm ou les journaux.
# Usage : outils/verifier.sh local|kubernetes
# La sonde tourne dans un conteneur Python avec pika (SONDE_IMAGE, construite depuis verification/).
# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

CIBLE="${1:-}"
[[ "$CIBLE" == local || "$CIBLE" == kubernetes ]] || echec "usage : $0 local|kubernetes"
SONDE_IMAGE="${SONDE_IMAGE:-collector/canal-sonde:1}"
NS_CANAL="${NS_CANAL:-canal}"
NS_SONDE="$(namespace_de marche)"
NS_INTRUS="canal-intrus"
if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == MSYS* ]]; then POUBELLE=NUL; else POUBELLE=/dev/null; fi
charger_mdp

kc() { kubectl ${KUBE_CONTEXT:+--context "$KUBE_CONTEXT"} "$@"; }
hm() { helm ${KUBE_CONTEXT:+--kube-context "$KUBE_CONTEXT"} "$@"; }

total=0
ok=0
compter() {
  # Lit les lignes "ac|cas|identite|attendu|obtenu|atteint" et les affiche alignees.
  local ac nom id attendu obtenu atteint
  while IFS='|' read -r ac nom id attendu obtenu atteint; do
    [[ -n "${atteint:-}" ]] || { printf '%s\n' "$ac" >&2; continue; }
    total=$((total + 1))
    [[ "$atteint" == oui ]] && ok=$((ok + 1))
    printf '%-4s %-3s %-58s %-13s attendu=%-7s obtenu=%s\n' "$ac" "$atteint" "$nom" "$id" "$attendu" "$obtenu" >&2
  done
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
    --entrypoint python "$SONDE_IMAGE" -I /verif/sonde.py "$@" || true
}

pod_sonde() {
  # pod_sonde <namespace> <arguments de sonde.py...> : pod jetable (PSS restricted), journal puis suppression.
  local ns="$1" nom phase='' a liste=''
  shift
  nom="canal-sonde-$(openssl rand -hex 3)"
  for a in "$@"; do liste+="\"$a\", "; done
  kc -n "$ns" create configmap "$nom" --from-file=sonde.py="$(chemin_hote "$CANAL_DIR/verification/sonde.py")" >/dev/null
  kc apply -f - >/dev/null <<YAML
apiVersion: v1
kind: Pod
metadata:
  name: $nom
  namespace: $ns
  labels: { app.kubernetes.io/part-of: collector-canal-verification }
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
  kc -n "$ns" logs "$nom" || true
  kc -n "$ns" delete pod "$nom" --wait=false >/dev/null
  kc -n "$ns" delete configmap "$nom" >/dev/null
}

sonde_kubernetes() {
  # Secret temporaire (toutes les identites de la sonde), supprime des la fin de la sonde.
  local v cle val
  {
    printf 'apiVersion: v1\nkind: Secret\ntype: Opaque\nmetadata:\n  name: canal-sonde\n  namespace: %s\ndata:\n' "$NS_SONDE"
    while read -r v; do printf '%s %s\n' "$v" "$(printf '%s' "${!v}" | base64 -w0)"; done < <(variables_mdp) |
      while read -r cle val; do printf '  %s: %s\n' "$cle" "$val"; done
  } | kc apply -f - >/dev/null
  pod_sonde "$NS_SONDE" "$@"
  kc -n "$NS_SONDE" delete secret canal-sonde --ignore-not-found >/dev/null
}

gestion() {
  local base="$1" u var code attendu
  for u in "${UTILISATEURS[@]}"; do
    var="$(variable_mdp "$u")"
    code="$(curl -s -o "$POUBELLE" -w '%{http_code}' --max-time 5 -K - <<<"user = \"$u:${!var}\"" \
      "$base/api/queues/$VHOST/controle.demandes" || true)"
    case "$u" in keda | admin-canal) attendu=200 ;; *) attendu=401 ;; esac
    printf 'MC4|api-gestion:/api/queues/%s/controle.demandes|%s|%s|%s|%s\n' "$VHOST" "$u" "$attendu" "$code" \
      "$([[ "$code" == "$attendu" ]] && echo oui || echo non)"
  done
}

fuite() {
  # fuite <nom> <commande...> : la sortie de la commande ne contient aucun mot de passe ni le cookie
  # de la session (AC6, X20). Une source illisible est un echec, jamais un "absent".
  local nom="$1" motifs="$GENERE_DIR/motifs-fuite" resultat compte statut obtenu
  shift
  sed -n 's/^[A-Z_]*=//p' "$MDP_FILE" >"$motifs"
  resultat="$({ "$@" 2>&1 | grep -c -F -f "$motifs"; printf '%s' "${PIPESTATUS[0]}"; } | tr '
' ' ')"
  rm -f "$motifs"
  read -r compte statut <<<"$resultat"
  if [[ "$statut" != 0 ]]; then obtenu="source-illisible"
  elif [[ "$compte" == 0 ]]; then obtenu=absent
  else obtenu="present($compte)"; fi
  printf 'AC6|absence-secret:%s|-|absent|%s|%s
' "$nom" "$obtenu" "$([[ "$obtenu" == absent ]] && echo oui || echo non)"
}

depot_suivi() { git -C "$(chemin_hote "$DEPOT_DIR")" grep -h -I -e '' -- .; }

image_sonde
if [[ "$CIBLE" == local ]]; then
  compter < <(
    sonde_local droits
    sonde_local ecartes
    gestion "http://127.0.0.1:15672"
    fuite fichiers-suivis-du-depot depot_suivi
    fuite journal-du-broker docker logs collector-canal-rabbitmq-1
  )
else
  contexte="$(kc config current-context)"
  if [[ "$contexte" == k3d-* ]]; then
    k3d image import -c "${contexte#k3d-}" "$SONDE_IMAGE" >/dev/null 2>&1 || journal "image de sonde non importee dans k3d"
  fi
  printf 'apiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n  labels:\n    pod-security.kubernetes.io/enforce: restricted\n' "$NS_INTRUS" |
    kc apply -f - >/dev/null
  svc="rabbitmq.$NS_CANAL.svc"
  noeud="rabbitmq-0.rabbitmq-noeuds.$NS_CANAL.svc"
  kc -n "$NS_CANAL" port-forward svc/rabbitmq 15673:15672 >/dev/null 2>&1 &
  pf=$!
  trap 'kill "$pf" 2>/dev/null || true' EXIT
  sleep 3
  compter < <(
    sonde_kubernetes droits
    sonde_kubernetes ecartes
    pod_sonde "$NS_SONDE" reseau "$svc:5672:ouvert" "$svc:15672:ferme" "$svc:15692:ferme" "$noeud:4369:ferme" "$noeud:25672:ferme"
    pod_sonde "$NS_INTRUS" reseau "$svc:5672:ferme" "$svc:15672:ferme" "$svc:15692:ferme" "$noeud:4369:ferme" "$noeud:25672:ferme"
    gestion "http://127.0.0.1:15673"
    types="$(kc -n "$NS_CANAL" get svc -o jsonpath='{.items[*].spec.type}')"
    if grep -q -E 'NodePort|LoadBalancer' <<<"$types"; then r=non; else r=oui; fi
    printf 'AC7|services-du-broker:ClusterIP-seul|-|oui|%s|%s\n' "$r" "$r"
    fuite fichiers-suivis-du-depot depot_suivi
    fuite manifeste-helm hm -n "$NS_CANAL" get manifest canal
    fuite valeurs-helm hm -n "$NS_CANAL" get values canal --all
    fuite journal-du-broker kc -n "$NS_CANAL" logs rabbitmq-0 -c rabbitmq
  )
  kc delete namespace "$NS_INTRUS" --wait=false >/dev/null
fi

journal "Verification $CIBLE : $ok / $total conformes"
[[ $total -gt 0 && $ok -eq $total ]]
