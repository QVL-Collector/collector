#!/usr/bin/env bash
# XC8 : empreinte du broker au repos puis sous vague (PerfTest, image epinglee).
# Producteur svc-pdm -> controle.demandes (confirmations, persistant) ; consommateur svc-controle.
# Les deux utilisent la topologie existante (--predeclared --skip-binding-queues) : aucun droit
# configure n'est requis. Releve toutes les 10 s : memoire et CPU du conteneur du broker.
# Usage : PLATEFORME=compose|k3d scripts/xc8-vague.sh [duree_s=300] [debit_msg_s=200]
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
charger_env

duree="${1:-300}"
debit="${2:-200}"
releves="$SORTIES_DIR/$(horodatage)-$PLATEFORME-xc8.txt"
csv_init

mesurer() {
  # Affiche "memoire_mio cpu" du broker.
  case "$PLATEFORME" in
    compose)
      docker stats --no-stream --format '{{.MemUsage}} {{.CPUPerc}}' "$CONTENEUR_COMPOSE" |
        awk '{ v=$1; u=v; gsub(/[0-9.]/,"",u); gsub(/[A-Za-z]/,"",v);
               f=(u=="GiB")?1024:(u=="KiB")?1/1024:1; printf "%d %s\n", v*f, $NF }'
      ;;
    k3d|eks)
      kubectl top pod -n "$NS_BROKER" "$POD_BROKER" --containers --no-headers |
        awk '$2=="rabbitmq" { m=$4; gsub(/Mi/,"",m); print m, $3 }'
      ;;
  esac
}

perftest() {
  # perftest <nom> <identite> <arguments PerfTest...> ; l'URI (avec le mot de passe) passe par l'environnement.
  local nom="$1" identite="$2" var; shift 2
  var="$(variable_mdp "$identite")"
  case "$PLATEFORME" in
    compose)
      URI="amqp://$identite:${!var}@rabbitmq:5672/collector" \
        docker run -d --rm --name "xp02-$nom" --network xp02_canal -e URI "$PERFTEST_IMAGE" "$@" >/dev/null
      ;;
    k3d|eks)
      # L'identite vient du Secret canal-amqp du namespace du service (pose par k3d.sh up).
      local ns
      case "$identite" in svc-pdm) ns=pdm-test ;; svc-controle) ns=controle-test ;; esac
      printf '%s\n' "$@" | jq -R . | jq -s --arg ns "$ns" --arg nom "xp02-$nom" --arg image "$PERFTEST_IMAGE" '
        { apiVersion: "v1", kind: "Pod",
          metadata: { name: $nom, namespace: $ns, labels: { "app.kubernetes.io/part-of": "xp02" } },
          spec: { restartPolicy: "Never", automountServiceAccountToken: false,
            securityContext: { runAsNonRoot: true, runAsUser: 2000, runAsGroup: 2000, seccompProfile: { type: "RuntimeDefault" } },
            containers: [ { name: "perftest", image: $image, args: .,
              env: [ { name: "URI", valueFrom: { secretKeyRef: { name: "canal-amqp", key: "AMQP_URL" } } } ],
              securityContext: { allowPrivilegeEscalation: false, capabilities: { drop: ["ALL"] } } } ] } }' |
        kubectl apply -f - >/dev/null
      ;;
  esac
}

arreter() {
  case "$PLATEFORME" in
    compose) docker rm -f xp02-vague-prod xp02-vague-cons >/dev/null 2>&1 || true ;;
    k3d|eks)
      kubectl -n pdm-test delete pod xp02-vague-prod --ignore-not-found --wait=false >/dev/null 2>&1 || true
      kubectl -n controle-test delete pod xp02-vague-cons --ignore-not-found --wait=false >/dev/null 2>&1 || true
      ;;
  esac
}
trap arreter EXIT

journal "Repos : 3 releves"
max_repos=0
for _ in 1 2 3; do
  read -r m c < <(mesurer); echo "repos $(date -u +%H:%M:%S) ${m}Mio ${c}" >>"$releves"
  (( m > max_repos )) && max_repos=$m; sleep 10
done

journal "Vague : $debit msg/s pendant $duree s"
communs=(--exchange controle.demandes --queue controle.demandes --predeclared --skip-binding-queues --time "$duree")
perftest vague-cons svc-controle "${communs[@]}" --producers 0 --consumers 1
perftest vague-prod svc-pdm "${communs[@]}" --routing-key v1 --producers 1 --consumers 0 \
  --rate "$debit" --confirm 50 --flag persistent --size 1024
max_vague=0
fin=$(( $(date +%s) + duree ))
while (( $(date +%s) < fin )); do
  read -r m c < <(mesurer); echo "vague $(date -u +%H:%M:%S) ${m}Mio ${c}" >>"$releves"
  (( m > max_vague )) && max_vague=$m; sleep 10
done

csv XC8 repos "$max_repos" Mio "<=512 demande, <=1024 limite" "$([[ $max_repos -le 512 ]] && echo oui || echo non)" "max de 3 releves"
csv XC8 vague "$max_vague" Mio "<=1024 limite" "$([[ $max_vague -le 1024 ]] && echo oui || echo non)" "$debit msg/s pendant $duree s, PerfTest"
journal "Memoire max : repos ${max_repos} Mio, vague ${max_vague} Mio (detail : $releves)"
