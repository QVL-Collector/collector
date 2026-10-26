#!/usr/bin/env bash
# Contrôle des exceptions aux portes Trivy (.trivyignore), selon docs/processus/devsecops.md § 3 (US-003 AC6, MQ2).
#
# Chaque entrée est précédée, sans ligne vide, de ses champs en commentaire :
#   # composant: <module, image ou fichier>
#   # justification: <non atteignable, pas de correctif, faux positif démontré>
#   # mesure: <mesure compensatoire>
#   # validation: <Sécurité AAAA-MM-JJ, Martin AAAA-MM-JJ>
#   # creee: AAAA-MM-JJ
#   # renouvelee: AAAA-MM-JJ   (facultatif, un seul renouvellement, nouvelle justification)
#   <IDENTIFIANT> exp:AAAA-MM-JJ
# Échoue si un champ manque, si l'échéance dépasse 30 jours après la création (ou le renouvellement),
# ou si une exception est échue. Écrit les compteurs I6 dans rapports/i6-exceptions.json.
# AUJOURDHUI (AAAA-MM-JJ) remplace la date du jour pour les essais.

set -euo pipefail

RACINE="$(git rev-parse --show-toplevel)"
FICHIER="${1:-$RACINE/.trivyignore}"
AUJOURDHUI="${AUJOURDHUI:-$(date -u +%Y-%m-%d)}"
DUREE_MAX=30
CHAMPS=(composant justification mesure validation creee)

jour() { date -u -d "$1" +%s 2> /dev/null || return 1; }
ecart_jours() { echo $(( ( $(jour "$2") - $(jour "$1") ) / 86400 )); }

T_AUJOURDHUI="$(jour "$AUJOURDHUI")" || { echo "AUJOURDHUI invalide : $AUJOURDHUI" >&2; exit 2; }

ERREURS=0
ACTIVES=0
ECHUES=0
RENOUVELEES=0
AGE_MAX=0
declare -A BLOC=()

erreur() { echo "::error file=.trivyignore,line=$1::$2"; ERREURS=$((ERREURS + 1)); }

if [[ -f "$FICHIER" ]]; then
  N=0
  while IFS= read -r LIGNE || [[ -n "$LIGNE" ]]; do
    N=$((N + 1))
    LIGNE="${LIGNE%$'\r'}"
    if [[ -z "${LIGNE//[[:space:]]/}" ]]; then
      BLOC=()
      continue
    fi
    if [[ "$LIGNE" =~ ^#[[:space:]]*([a-z]+):[[:space:]]*(.*)$ ]]; then
      BLOC["${BASH_REMATCH[1]}"]="${BASH_REMATCH[2]}"
      continue
    fi
    [[ "$LIGNE" == \#* ]] && continue

    ID="${LIGNE%%[[:space:]]*}"
    for c in "${CHAMPS[@]}"; do
      [[ -n "${BLOC[$c]:-}" ]] || erreur "$N" "exception $ID : champ « $c » manquant"
    done
    if [[ ! "$LIGNE" =~ exp:([0-9]{4}-[0-9]{2}-[0-9]{2}) ]]; then
      erreur "$N" "exception $ID : échéance exp:AAAA-MM-JJ manquante"
      BLOC=(); continue
    fi
    EXP="${BASH_REMATCH[1]}"
    DEPART="${BLOC[renouvelee]:-${BLOC[creee]:-}}"
    if ! jour "$EXP" > /dev/null || [[ -z "$DEPART" ]] || ! jour "$DEPART" > /dev/null; then
      erreur "$N" "exception $ID : date invalide (creee, renouvelee ou exp)"
      BLOC=(); continue
    fi
    if (( $(ecart_jours "$DEPART" "$EXP") > DUREE_MAX )); then
      erreur "$N" "exception $ID : échéance à plus de $DUREE_MAX jours de $DEPART"
    fi
    if (( $(jour "$EXP") < T_AUJOURDHUI )); then
      erreur "$N" "exception $ID : échue le $EXP"
      ECHUES=$((ECHUES + 1))
    else
      ACTIVES=$((ACTIVES + 1))
    fi
    [[ -n "${BLOC[renouvelee]:-}" ]] && RENOUVELEES=$((RENOUVELEES + 1))
    if [[ -n "${BLOC[creee]:-}" ]] && jour "${BLOC[creee]}" > /dev/null; then
      AGE="$(ecart_jours "${BLOC[creee]}" "$AUJOURDHUI")"
      (( AGE > AGE_MAX )) && AGE_MAX="$AGE"
    fi
    BLOC=()
  done < "$FICHIER"
fi

mkdir -p "$RACINE/rapports"
printf '{"outil":"exceptions-trivy","actives":%d,"echues":%d,"renouvelees":%d,"age_max_jours":%d,"bloquants":%d}\n' \
  "$ACTIVES" "$ECHUES" "$RENOUVELEES" "$AGE_MAX" "$ERREURS" | tee "$RACINE/rapports/i6-exceptions.json"

if (( ERREURS > 0 )); then
  echo "::error::$ERREURS exception(s) invalide(s) ou échue(s) dans .trivyignore"
  exit 1
fi
echo "Exceptions Trivy : $ACTIVES active(s), aucune invalide ni échue."
