#!/usr/bin/env bash
# Cas positifs et négatifs du filtre tools/sortie-sans-imds.jq (contrôle X17 sortie-sans-imds, US-004 AC8).
#   bash tools/tests/sortie-sans-imds.sh

set -euo pipefail

FILTRE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/sortie-sans-imds.jq"
command -v cygpath >/dev/null 2>&1 && FILTRE="$(cygpath -m "$FILTRE")"
ECHECS=0

politique() {
  local selecteur="$1" sortie="$2"
  printf '{"spec":{"podSelector":%s,"policyTypes":["Egress"],"egress":%s}}' "$selecteur" "$sortie"
}

SOCLE="$(politique '{}' '[{"to":[{"ipBlock":{"cidr":"0.0.0.0/0","except":["169.254.169.254/32"]}}]}]')"

cas() {
  local attendu="$1" nom="$2" politiques="$3" obtenu
  obtenu="$(printf '{"items":[%s]}' "$politiques" | jq -f "$FILTRE" | tr -d '\r')"
  if [ "$obtenu" = "$attendu" ]; then
    printf 'ok    %s\n' "$nom"
  else
    printf 'ÉCHEC %s : %s, attendu %s\n' "$nom" "$obtenu" "$attendu"
    ECHECS=$((ECHECS + 1))
  fi
}

cas true "socle : 0.0.0.0/0 sauf l'IMDS" "$SOCLE"
cas true "plages privées seules" "$(politique '{}' '[{"to":[{"ipBlock":{"cidr":"10.0.0.0/8"}},{"ipBlock":{"cidr":"172.16.0.0/12"}}]}]')"
cas true "exception plus large que l'IMDS" "$(politique '{}' '[{"to":[{"ipBlock":{"cidr":"0.0.0.0/0","except":["169.254.0.0/16"]}}]}]')"
cas true "sélecteurs de pods et de namespaces" "$(politique '{}' '[{"to":[{"namespaceSelector":{}}],"ports":[{"port":53,"protocol":"UDP"}]}]')"
cas true "socle et politique complémentaire conforme" "$SOCLE,$(politique '{"matchLabels":{"app":"a"}}' '[{"to":[{"ipBlock":{"cidr":"10.0.0.0/8"}}]}]')"
cas true "IPv6 ignoré pour l'IMDS IPv4" "$(politique '{}' '[{"to":[{"ipBlock":{"cidr":"::/0"}}]}]')"
cas true "socle et politique d'entrée seule" "$SOCLE,"'{"spec":{"podSelector":{},"policyTypes":["Ingress"],"ingress":[{}]}}'

cas false "aucune politique" ""
cas false "egress: [{}], tout autoriser" "$(politique '{}' '[{}]')"
cas false "to vide" "$(politique '{}' '[{"to":[]}]')"
cas false "0.0.0.0/0 sans exception" "$(politique '{}' '[{"to":[{"ipBlock":{"cidr":"0.0.0.0/0"}}]}]')"
cas false "exception puis 169.254.0.0/16 autorisé" "$(politique '{}' '[{"to":[{"ipBlock":{"cidr":"0.0.0.0/0","except":["169.254.169.254/32"]}},{"ipBlock":{"cidr":"169.254.0.0/16"}}]}]')"
cas false "128.0.0.0/1 couvre l'IMDS" "$(politique '{}' '[{"to":[{"ipBlock":{"cidr":"128.0.0.0/1"}}]}]')"
cas false "IMDS en /32 autorisé" "$(politique '{}' '[{"to":[{"ipBlock":{"cidr":"169.254.169.254/32"}}]}]')"
cas false "seconde politique qui ouvre toute la sortie" "$SOCLE,$(politique '{"matchLabels":{"app":"a"}}' '[{}]')"
cas false "politique limitée à certains pods" "$(politique '{"matchLabels":{"app":"a"}}' '[{"to":[{"ipBlock":{"cidr":"10.0.0.0/8"}}]}]')"
cas false "sortie implicite par egress sans policyTypes" "$SOCLE,"'{"spec":{"podSelector":{},"egress":[{}]}}'


aws() {
  local ns="$1" selecteur="$2" sortie="$3"
  printf '{"metadata":{"namespace":"%s"},"spec":{"podSelector":%s,"policyTypes":["Egress"],"egress":%s}}' "$ns" "$selecteur" "$sortie"
}
ACCES_AWS='{"matchLabels":{"app.kubernetes.io/name":"place-de-marche","collector/acces-aws":"true"}}'
REGLES_AWS='[{"to":[{"ipBlock":{"cidr":"169.254.169.254/32"}}],"ports":[{"port":80,"protocol":"TCP"}]},{"to":[{"ipBlock":{"cidr":"0.0.0.0/0","except":["10.0.0.0/8","169.254.0.0/16"]}}],"ports":[{"port":443,"protocol":"TCP"}]}]'
REFUS='{"metadata":{"namespace":"place-de-marche"},"spec":{"podSelector":{},"policyTypes":["Ingress","Egress"]}}'
FLUX='{"metadata":{"namespace":"collector"},"spec":{"podSelector":{},"policyTypes":["Ingress","Egress"],"ingress":[{"from":[{"podSelector":{}}]}],"egress":[{}]}}'

cas true "Place de marché : exception accesAws du chart" "$REFUS,$(aws place-de-marche "$ACCES_AWS" "$REGLES_AWS")"
cas false "Place de marché : même règle sans l'étiquette accesAws" "$REFUS,$(aws place-de-marche '{"matchLabels":{"app.kubernetes.io/name":"place-de-marche"}}' "$REGLES_AWS")"
cas false "étiquette accesAws hors du namespace place-de-marche" "$REFUS,$(aws interactions "$ACCES_AWS" "$REGLES_AWS")"
cas false "exception accesAws qui ouvre aussi toute la sortie" "$REFUS,$(aws place-de-marche "$ACCES_AWS" '[{"to":[{"ipBlock":{"cidr":"169.254.169.254/32"}}]},{}]')"
cas false "exception accesAws vers 169.254.0.0/16" "$REFUS,$(aws place-de-marche "$ACCES_AWS" '[{"to":[{"ipBlock":{"cidr":"169.254.0.0/16"}}]}]')"
cas false "politique allow-egress de flux install" "$FLUX"
[ "$ECHECS" -eq 0 ] || {
  printf '%s cas en échec\n' "$ECHECS"
  exit 1
}
printf 'Tous les cas passent.\n'
