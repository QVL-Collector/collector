#!/usr/bin/env bash
# Local 2 (k3d) : broker, politiques reseau, KEDA et consommateur lent dans le cluster de XP01.
# Debogage de XC5 (pod), XC6, XC7 et de la partie reseau de XC9 ; sans valeur de preuve pour EKS.
# Le contexte kubectl courant doit etre un cluster k3d (k3d-<nom>) ; rien n'est fait ailleurs.
# Usage :
#   scripts/k3d.sh cluster-essai up|down   cluster k3d jetable "xp02-essai" (si celui de XP01 est absent)
#   scripts/k3d.sh up                      namespaces, broker, secrets, politiques, KEDA, XC6
#   scripts/k3d.sh sonde refus|ecartes     sonde AMQP depuis un pod de pdm-test (appelee par sonde.sh)
#   scripts/k3d.sh reseau                  XC7 + XC9 reseau (ns-intrus, pdm-test) + temoin KEDA
#   scripts/k3d.sh xc6                     vague de 300 messages, consommateurs 1 -> >=2 -> 1
#   scripts/k3d.sh down                    retire tout ce que XP02 a pose (KEDA compris)
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
PLATEFORME=k3d
K3S_IMAGE="${K3S_IMAGE:-rancher/k3s:v1.35.5-k3s1@sha256:2074403abe1bded11ef3dde09d457e13be8e0b64c218b1c4f8269b4565cfbc65}"
CLUSTER_ESSAI="xp02-essai"
KUBECONFIG_ESSAI="$GENERE_DIR/kubeconfig-$CLUSTER_ESSAI"
NS_SERVICES=(pdm-test controle-test interactions-test lecture-test)
# kubectl.exe ne lit pas les chemins /c/... : chemins relatifs au repertoire de XP02.
cd "$XP_DIR"

contexte() {
  local c
  c="$(kubectl config current-context)"
  [[ "$c" == k3d-* ]] || echec "contexte kubectl '$c' : un cluster k3d est attendu"
  printf '%s' "$c"
}
nom_cluster() { local c; c="$(contexte)"; printf '%s' "${c#k3d-}"; }

cluster_essai() {
  case "${1:-}" in
    up)
      # API sur 127.0.0.1 (host.docker.internal n'est pas joignable depuis Git Bash) ; kubeconfig
      # dedie dans .genere/ : le kubeconfig par defaut et son contexte courant ne sont pas modifies.
      mkdir -p "$GENERE_DIR"
      k3d cluster create "$CLUSTER_ESSAI" --servers 1 --agents 2 --image "$K3S_IMAGE" \
        --api-port 127.0.0.1:6550 --kubeconfig-update-default=false --kubeconfig-switch-context=false \
        --wait --timeout 300s
      k3d kubeconfig get "$CLUSTER_ESSAI" >"$KUBECONFIG_ESSAI"
      KUBECONFIG="$(chemin_hote "$KUBECONFIG_ESSAI")" kubectl wait --for=condition=Ready nodes --all --timeout=180s
      journal "Cluster pret ; pour les commandes suivantes : export KUBECONFIG=\"$(chemin_hote "$KUBECONFIG_ESSAI")\""
      ;;
    down)
      k3d cluster delete "$CLUSTER_ESSAI"
      rm -f "$KUBECONFIG_ESSAI"
      ;;
    *) echec "usage : $0 cluster-essai up|down" ;;
  esac
}

manifeste() {
  sed -e "s|image: RABBITMQ_IMAGE|image: $RABBITMQ_IMAGE|" -e "s|image: PERFTEST_IMAGE|image: $PERFTEST_IMAGE|" "$XP_DIR/k8s/$1"
}

secret_stdin() {
  # secret_stdin <namespace> <nom> : contenu "CLE=valeur" lu sur l'entree standard (MC5, MX1).
  # Le manifeste est construit par jq et passe a kubectl par l'entree standard : aucune valeur
  # en argument de commande ni sur disque.
  jq -Rn --arg ns "$1" --arg nom "$2" '
    { apiVersion: "v1", kind: "Secret", type: "Opaque",
      metadata: { name: $nom, namespace: $ns, labels: { "app.kubernetes.io/part-of": "xp02" } },
      stringData: ([inputs | select(length > 0) | capture("^(?<key>[^=]+)=(?<value>.*)$")] | from_entries) }' |
    kubectl apply -f - >/dev/null
}

url_amqp() {
  local var; var="$(variable_mdp "$1")"
  printf 'amqp://%s:%s@rabbitmq.%s.svc:5672/collector' "$1" "${!var}" "$NS_BROKER"
}

up() {
  contexte >/dev/null
  [[ -f "$ENV_FILE" ]] || "$XP_DIR/scripts/identifiants.sh"
  charger_env
  kubectl apply -f k8s/00-namespaces.yaml >/dev/null

  kubectl -n "$NS_BROKER" create configmap rabbitmq-config \
    --from-file=10-collector.conf=broker/conf.d/10-collector.conf \
    --from-file=enabled_plugins=broker/enabled_plugins \
    --from-file=10-topologie.json=broker/definitions/10-topologie.json \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  kubectl -n "$NS_BROKER" delete secret rabbitmq-interne --ignore-not-found >/dev/null
  kubectl -n "$NS_BROKER" create secret generic rabbitmq-interne \
    --from-file=00-utilisateurs.json=.genere/00-utilisateurs.json \
    --from-file=cookie-erlang=.genere/cookie-erlang >/dev/null

  # Un Secret par service, dans son namespace (MC5).
  secret_stdin pdm-test canal-amqp <<<"AMQP_URL=$(url_amqp svc-pdm)"
  secret_stdin controle-test canal-amqp <<<"AMQP_URL=$(url_amqp svc-controle)"
  secret_stdin interactions-test canal-amqp <<<"AMQP_URL=$(url_amqp svc-interactions)"
  secret_stdin lecture-test canal-amqp <<<"AMQP_URL=$(url_amqp svc-test)"

  manifeste 10-broker.yaml | kubectl apply -f - >/dev/null
  kubectl apply -f k8s/20-politiques-reseau.yaml >/dev/null
  local debut; debut="$(ms_maintenant)"
  kubectl -n "$NS_BROKER" rollout status statefulset/rabbitmq --timeout=300s
  journal "Broker Ready en $(( ($(ms_maintenant) - debut) / 1000 )) s"
  if rmq rabbitmqctl -q list_users | grep -qw guest; then echec "utilisateur guest present (XC9)"; fi

  helm repo add kedacore https://kedacore.github.io/charts --force-update >/dev/null
  helm upgrade --install keda kedacore/keda --version "$KEDA_CHART_VERSION" \
    --namespace keda --create-namespace --wait --timeout 300s >/dev/null
  journal "KEDA $(helm -n keda list -o json | jq -r '.[0].app_version') installe (chart $KEDA_CHART_VERSION)"

  secret_stdin controle-test keda-rabbitmq <<<"host=http://keda-monitoring:${MDP_KEDA_MONITORING}@rabbitmq.$NS_BROKER.svc:15672/collector"
  manifeste 30-keda-xc6.yaml | kubectl apply -f - >/dev/null
  kubectl -n controle-test rollout status deployment/consommateur-lent --timeout=180s
  kubectl -n controle-test wait scaledobject/consommateur-lent --for=condition=Ready --timeout=120s
  journal "Pret : broker, politiques, KEDA, consommateur lent"
}

pod_jetable() {
  # pod_jetable <namespace> <image> <commande sh> [secret envFrom] : journal complet du pod.
  # (kubectl run -i perd les premieres lignes : pod cree, attendu jusqu'a sa fin, journal lu, supprime.)
  local ns="$1" image="$2" cmd="$3" secret="${4:-}" nom pull='IfNotPresent' phase='' i
  nom="xp02-sonde-$(openssl rand -hex 3)"
  [[ "$image" == "$SONDE_IMAGE" ]] && pull='Never'
  jq -n --arg ns "$ns" --arg nom "$nom" --arg image "$image" --arg pull "$pull" --arg cmd "$cmd" --arg secret "$secret" '
    { apiVersion: "v1", kind: "Pod",
      metadata: { name: $nom, namespace: $ns, labels: { "app.kubernetes.io/part-of": "xp02" } },
      spec: { restartPolicy: "Never", automountServiceAccountToken: false,
        securityContext: { runAsNonRoot: true, runAsUser: 65534, seccompProfile: { type: "RuntimeDefault" } },
        containers: [ { name: "sonde", image: $image, imagePullPolicy: $pull,
          command: ["/bin/sh", "-c"], args: [$cmd],
          securityContext: { allowPrivilegeEscalation: false, capabilities: { drop: ["ALL"] } } }
          + (if $secret == "" then {} else { envFrom: [ { secretRef: { name: $secret } } ] } end) ] } }' |
    kubectl apply -f - >/dev/null
  for i in $(seq 1 120); do
    phase="$(kubectl -n "$ns" get pod "$nom" -o jsonpath='{.status.phase}')"
    [[ "$phase" == Succeeded || "$phase" == Failed ]] && break
    sleep 2
  done
  kubectl -n "$ns" logs "$nom"
  kubectl -n "$ns" delete pod "$nom" --wait=false >/dev/null
  [[ "$phase" == Succeeded ]]
}

sonde() {
  local scenario="${1:?scenario}"
  charger_env
  k3d image import -c "$(nom_cluster)" "$SONDE_IMAGE" >/dev/null 2>&1
  # Secret temporaire (identites de toutes les sondes), supprime des la fin de la sonde.
  {
    for u in "${UTILISATEURS[@]}"; do v="$(variable_mdp "$u")"; printf '%s=%s\n' "$v" "${!v}"; done
    printf 'AMQP_HOTE=rabbitmq.%s.svc\n' "$NS_BROKER"
  } | secret_stdin pdm-test xp02-sonde
  pod_jetable pdm-test "$SONDE_IMAGE" "python -I /sonde/sonde.py $scenario" xp02-sonde || true
  kubectl -n pdm-test delete secret xp02-sonde --ignore-not-found >/dev/null
}

reseau() {
  csv_init
  local sortie="$SORTIES_DIR/$(horodatage)-$PLATEFORME-reseau.txt" svc noeud ok=0 total=0
  svc="rabbitmq.$NS_BROKER.svc"
  noeud="rabbitmq-0.rabbitmq-noeuds.$NS_BROKER.svc"
  essai() { printf 'for c in %s; do h=${c%%%%:*}; p=${c##*:}; if nc -z -w 5 "$h" "$p"; then echo "$c ouvert"; else echo "$c ferme"; fi; done' "$*"; }
  {
    pod_jetable ns-intrus "$CURL_IMAGE" "$(essai "$svc:5672" "$svc:15672" "$noeud:4369" "$noeud:25672")" | sed 's/^/ns-intrus /'
    pod_jetable pdm-test "$CURL_IMAGE" "$(essai "$svc:5672" "$svc:15672" "$svc:15692" "$noeud:4369" "$noeud:25672")" | sed 's/^/pdm-test /'
  } >"$sortie"
  while read -r ns cible etat; do
    [[ -n "${etat:-}" ]] || continue
    local attendu=ferme critere=XC7
    [[ "$ns" == pdm-test && "$cible" == *:5672 ]] && { attendu=ouvert; critere=TEMOIN; }
    [[ "$cible" == *:4369 || "$cible" == *:25672 ]] && critere=XC9
    total=$((total + 1)); [[ "$etat" == "$attendu" ]] && ok=$((ok + 1))
    csv "$critere" "reseau" "$etat" "port" "$attendu" "$([[ "$etat" == "$attendu" ]] && echo oui || echo non)" "$ns -> $cible"
  done <"$sortie"
  # Temoin : KEDA joint 15672 (ScaledObject pret, aucune erreur du scaler).
  local pret
  pret="$(kubectl -n controle-test get scaledobject consommateur-lent -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}')"
  echo "keda-operator rabbitmq:15672 $([[ "$pret" == True ]] && echo ouvert || echo ferme)" >>"$sortie"
  total=$((total + 1)); [[ "$pret" == True ]] && ok=$((ok + 1))
  csv TEMOIN reseau "$pret" "Ready" True "$([[ "$pret" == True ]] && echo oui || echo non)" "KEDA -> 15672 (ScaledObject)"
  journal "Reseau : $ok / $total conformes (detail : $sortie)"
  cat "$sortie" >&2
  [[ $ok -eq $total ]]
}

consommateurs() {
  rmq rabbitmqctl -q list_queues -p collector name messages consumers 2>/dev/null | awk '$1=="controle.demandes" { print $2, $3 }'
}

xc6() {
  charger_env
  csv_init
  local t_seuil='' t_deux='' t_vide='' t_un='' debut m c
  debut="$(ms_maintenant)"
  kubectl -n pdm-test run xp02-vague-xc6 --image="$PERFTEST_IMAGE" --restart=Never \
    --labels="app.kubernetes.io/part-of=xp02" --overrides="{\"spec\":{\"automountServiceAccountToken\":false,
      \"containers\":[{\"name\":\"p\",\"image\":\"$PERFTEST_IMAGE\",
        \"args\":[\"--exchange\",\"controle.demandes\",\"--routing-key\",\"v1\",\"--queue\",\"controle.demandes\",\"--predeclared\",
          \"--skip-binding-queues\",\"--producers\",\"1\",\"--consumers\",\"0\",\"--pmessages\",\"300\",\"--rate\",\"50\",\"--confirm\",\"20\",\"--flag\",\"persistent\"],
        \"env\":[{\"name\":\"URI\",\"valueFrom\":{\"secretKeyRef\":{\"name\":\"canal-amqp\",\"key\":\"AMQP_URL\"}}}]}]}}" >/dev/null
  while (( ($(ms_maintenant) - debut) / 1000 < 600 )); do
    read -r m c < <(consommateurs)
    printf '%s messages=%s consommateurs=%s\n' "$(date -u +%H:%M:%S)" "$m" "$c" >&2
    [[ -z "$t_seuil" && $m -gt 5 ]] && t_seuil="$(ms_maintenant)"
    [[ -n "$t_seuil" && -z "$t_deux" && $c -ge 2 ]] && t_deux="$(ms_maintenant)"
    [[ -n "$t_deux" && -z "$t_vide" && $m -eq 0 ]] && t_vide="$(ms_maintenant)"
    [[ -n "$t_vide" && -z "$t_un" && $c -le 1 ]] && { t_un="$(ms_maintenant)"; break; }
    sleep 3
  done
  kubectl -n pdm-test delete pod xp02-vague-xc6 --ignore-not-found --wait=false >/dev/null
  if [[ -n "$t_deux" ]]; then
    local d=$(( (t_deux - t_seuil) / 1000 ))
    csv XC6 "montee" "$d" s "<=60" "$([[ $d -le 60 ]] && echo oui || echo non)" "seuil 5 messages -> 2 consommateurs"
    journal "XC6 montee : $d s"
  else
    csv XC6 "montee" "absente" s "<=60" non "2 consommateurs jamais observes"
  fi
  if [[ -n "$t_un" ]]; then
    csv XC6 "retour" "$(( (t_un - t_vide) / 1000 ))" s "retour a 1" oui "file vide -> 1 consommateur"
    journal "XC6 retour a 1 : $(( (t_un - t_vide) / 1000 )) s apres vidage"
  else
    csv XC6 "retour" "absent" s "retour a 1" non "pas de retour a 1 en 10 min"
  fi
}

down() {
  local c; c="$(contexte)"
  manifeste 30-keda-xc6.yaml | kubectl delete -f - --ignore-not-found >/dev/null 2>&1 || true
  helm uninstall keda -n keda >/dev/null 2>&1 || true
  kubectl delete namespace keda collector-canal interactions-test lecture-test --ignore-not-found --wait=true >/dev/null
  for ns in pdm-test controle-test ns-intrus; do
    kubectl -n "$ns" delete pod,secret -l app.kubernetes.io/part-of=xp02 --ignore-not-found >/dev/null 2>&1 || true
  done
  kubectl get crd -o name | grep 'keda.sh' | xargs -r kubectl delete >/dev/null 2>&1 || true
  journal "XP02 retire du cluster $c (namespaces partages pdm-test, controle-test, ns-intrus conserves)"
}

case "${1:-}" in
  cluster-essai) shift; cluster_essai "$@" ;;
  up) up ;;
  sonde) shift; sonde "$@" ;;
  reseau) reseau ;;
  xc6) xc6 ;;
  down) down ;;
  *) echec "usage : $0 cluster-essai up|down | up | sonde refus|ecartes | reseau | xc6 | down" ;;
esac
