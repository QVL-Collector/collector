#!/usr/bin/env bash
# Publication sur main (D10 R1, P2 ; US-007 AC1, AC2, AC8) : pour chaque image modifiée, une seule construction
# étiquetée sha-<commit>, porte Trivy, SBOM CycloneDX, poussée GHCR, signature cosign sans clé, attestation du SBOM,
# vérification de la signature et de l'attestation avec l'identité de ce workflow ; puis écriture des empreintes,
# seul contenu de la branche env/demo, sans poussée forcée.
#
# Entrées : COMMIT, AVANT (github.event.before), GH_TOKEN (jeton du job), GITHUB_ACTOR, GITHUB_REPOSITORY,
# GITHUB_WORKFLOW_REF. Journal public : références d'images et empreintes seulement, jamais de jeton.

set -euo pipefail

: "${COMMIT:?}" "${GH_TOKEN:?}" "${GITHUB_ACTOR:?}" "${GITHUB_REPOSITORY:?}" "${GITHUB_WORKFLOW_REF:?}"
AVANT="${AVANT:-}"

RACINE="$(git rev-parse --show-toplevel)"
cd "$RACINE"
mkdir -p rapports
chmod 0777 rapports

REGISTRE="ghcr.io/${GITHUB_REPOSITORY_OWNER:-${GITHUB_REPOSITORY%%/*}}"
REGISTRE="${REGISTRE,,}"
ETIQUETTE="sha-$COMMIT"
IDENTITE="https://github.com/${GITHUB_WORKFLOW_REF%@*}@refs/heads/main"
EMETTEUR="https://token.actions.githubusercontent.com"
BRANCHE_ENV="env/demo"
FICHIER_EMPREINTES="deploy/demo/empreintes.yaml"
OUTILS=(docker compose -f tools/ci/outils.yml run --rm -T)
COSIGN=("${OUTILS[@]}" --user "$(id -u):$(id -g)" cosign)

# nom|Dockerfile|contexte|chemins qui la modifient
IMAGES=(
  "place-de-marche|services/place-de-marche/Dockerfile|.|services/place-de-marche/ libs/ contracts/"
  "controle|services/controle/Dockerfile|.|services/controle/ libs/ contracts/"
  "interactions|services/interactions/Dockerfile|.|services/interactions/ libs/ contracts/"
  "transactions|services/transactions/Dockerfile|.|services/transactions/ libs/"
  "front|web/Dockerfile|web|web/"
  "keycloak|infra/identite/Dockerfile|.|infra/identite/Dockerfile infra/identite/realm-collector.json"
)

TRAVAIL="$(mktemp -d)"
trap 'rm -rf "$TRAVAIL"' EXIT

entete_auth() { printf 'AUTHORIZATION: basic %s' "$(printf 'x-access-token:%s' "$GH_TOKEN" | base64 -w0)"; }

preparer_env() {
  git init -q "$TRAVAIL/env"
  git -C "$TRAVAIL/env" remote add origin "https://github.com/$GITHUB_REPOSITORY.git"
  if git -C "$TRAVAIL/env" fetch -q --depth 1 origin "$BRANCHE_ENV" 2>/dev/null; then
    git -C "$TRAVAIL/env" checkout -q -B "$BRANCHE_ENV" FETCH_HEAD
  else
    echo "Branche $BRANCHE_ENV absente : création (première publication)."
    git -C "$TRAVAIL/env" checkout -q --orphan "$BRANCHE_ENV"
  fi
}

empreinte_publiee() {
  local fichier="$TRAVAIL/env/$FICHIER_EMPREINTES"
  [[ -f "$fichier" ]] || return 0
  sed -n "s|^  $1: ||p" "$fichier"
}

modifiee() {
  local chemins="$1" chemin
  if [[ -z "$AVANT" || "$AVANT" =~ ^0+$ ]] || ! git cat-file -e "$AVANT^{commit}" 2>/dev/null; then
    return 0
  fi
  for chemin in $chemins; do
    git diff --quiet "$AVANT" "$COMMIT" -- "$chemin" || return 0
  done
  return 1
}

publier_image() {
  local nom="$1" dockerfile="$2" contexte="$3" ref empreinte cible sbom
  ref="$REGISTRE/$nom:$ETIQUETTE"
  sbom="rapports/sbom-$nom.cdx.json"

  docker build --file "$dockerfile" --tag "$ref" \
    --label "org.opencontainers.image.source=https://github.com/$GITHUB_REPOSITORY" \
    --label "org.opencontainers.image.revision=$COMMIT" "$contexte"

  bash tools/ci/trivy.sh image "$ref" "publication-$nom"
  "${OUTILS[@]}" trivy image --quiet --format cyclonedx --output "/$sbom" "$ref"

  docker push --quiet "$ref" >/dev/null
  empreinte="$(docker image inspect "$ref" --format '{{range .RepoDigests}}{{println .}}{{end}}' | grep -m1 "^$REGISTRE/$nom@sha256:")"
  [[ "$empreinte" =~ @sha256:[0-9a-f]{64}$ ]] || { echo "::error::empreinte introuvable pour $ref"; exit 1; }
  cible="$empreinte"

  "${COSIGN[@]}" sign --yes "$cible" >/dev/null
  "${COSIGN[@]}" attest --yes --type cyclonedx --predicate "/$sbom" "$cible" >/dev/null
  "${COSIGN[@]}" verify --certificate-identity "$IDENTITE" --certificate-oidc-issuer "$EMETTEUR" "$cible" >/dev/null
  "${COSIGN[@]}" verify-attestation --type cyclonedx \
    --certificate-identity "$IDENTITE" --certificate-oidc-issuer "$EMETTEUR" "$cible" >/dev/null

  echo "$nom : $cible (signée, SBOM attesté, vérifiée)"
  printf '%s %s\n' "$nom" "$cible" >> "$TRAVAIL/empreintes.txt"
}

ecrire_env() {
  local dossier="$TRAVAIL/env" nom
  find "$dossier" -mindepth 1 -maxdepth 1 ! -name .git -exec rm -rf {} +
  mkdir -p "$dossier/$(dirname "$FICHIER_EMPREINTES")"
  {
    printf 'images:\n'
    while read -r nom cible; do printf '  %s: %s\n' "$nom" "$cible"; done < "$TRAVAIL/empreintes.txt"
  } > "$dossier/$FICHIER_EMPREINTES"
  git -C "$dossier" add -A
  if git -C "$dossier" diff --cached --quiet; then
    echo "Empreintes inchangées : rien à écrire sur $BRANCHE_ENV."
    return 0
  fi
  git -C "$dossier" -c user.name="github-actions[bot]" \
    -c user.email="41898282+github-actions[bot]@users.noreply.github.com" \
    commit -q -m "chore(env-demo): publier les empreintes de $ETIQUETTE"
  git -C "$dossier" -c "http.https://github.com/.extraheader=$(entete_auth)" push -q origin "HEAD:refs/heads/$BRANCHE_ENV"
  echo "Empreintes écrites sur $BRANCHE_ENV ($ETIQUETTE)."
}

printf '%s' "$GH_TOKEN" | docker login ghcr.io --username "$GITHUB_ACTOR" --password-stdin >/dev/null
trap 'docker logout ghcr.io >/dev/null 2>&1 || true; rm -rf "$TRAVAIL"' EXIT

preparer_env
: > "$TRAVAIL/empreintes.txt"
for entree in "${IMAGES[@]}"; do
  IFS='|' read -r nom dockerfile contexte chemins <<< "$entree"
  precedente="$(empreinte_publiee "$nom")"
  if [[ -n "$precedente" ]] && ! modifiee "$chemins"; then
    echo "$nom : inchangée, empreinte conservée ($precedente)"
    printf '%s %s\n' "$nom" "$precedente" >> "$TRAVAIL/empreintes.txt"
    continue
  fi
  publier_image "$nom" "$dockerfile" "$contexte"
done
ecrire_env
