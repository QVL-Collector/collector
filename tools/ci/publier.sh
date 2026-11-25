#!/usr/bin/env bash
# Publication sur main (D10 R1, P2 ; US-007 AC1, AC2, AC8) : pour chaque image modifiée depuis le commit source de
# env/demo, une seule construction étiquetée sha-<commit>, porte Trivy, SBOM CycloneDX, poussée GHCR, signature cosign
# sans clé, attestation du SBOM, vérification de la signature et de l'attestation avec l'identité de ce workflow ;
# puis écriture des empreintes et du commit source, seul contenu de env/demo, par la clé de déploiement de
# l'environnement « publication », sans poussée forcée.
#
# Entrées : COMMIT, GH_TOKEN (registre), ENV_DEMO_CLE (clé de déploiement), GITHUB_ACTOR, GITHUB_REPOSITORY,
# GITHUB_WORKFLOW_REF. Journal public : références d'images et empreintes seulement, jamais de jeton ni de clé.

set -euo pipefail

: "${COMMIT:?}" "${GH_TOKEN:?}" "${ENV_DEMO_CLE:?}" "${GITHUB_ACTOR:?}" "${GITHUB_REPOSITORY:?}" "${GITHUB_WORKFLOW_REF:?}"

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
CLE_HOTE_GITHUB="github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"
OUTILS=(docker compose -f tools/ci/outils.yml run --rm -T)
COSIGN=("${OUTILS[@]}" --user "$(id -u):$(id -g)" cosign)

TRAVAIL="$(mktemp -d)"
trap 'docker logout ghcr.io >/dev/null 2>&1 || true; rm -rf "$TRAVAIL"' EXIT

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

lire_publie() {
  local fichier="$TRAVAIL/env/$FICHIER_EMPREINTES"
  [[ -f "$fichier" ]] || return 0
  sed -n "s|^$1||p" "$fichier" | tr -d '\r'
}

SOURCE=""
source_connue() {
  [[ "$SOURCE" =~ ^[0-9a-f]{40}$ ]] && git cat-file -e "$SOURCE^{commit}" 2>/dev/null &&
    git merge-base --is-ancestor "$SOURCE" "$COMMIT"
}

modifiee() {
  local chemin
  source_connue || return 0
  for chemin in $1; do
    git diff --quiet "$SOURCE" "$COMMIT" -- "$chemin" || return 0
  done
  return 1
}

publier_image() {
  local nom="$1" dockerfile="$2" contexte="$3" ref empreinte sbom
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

  "${COSIGN[@]}" sign --yes "$empreinte" >/dev/null
  "${COSIGN[@]}" attest --yes --type cyclonedx --predicate "/$sbom" "$empreinte" >/dev/null
  "${COSIGN[@]}" verify --certificate-identity "$IDENTITE" --certificate-oidc-issuer "$EMETTEUR" "$empreinte" >/dev/null
  "${COSIGN[@]}" verify-attestation --type cyclonedx \
    --certificate-identity "$IDENTITE" --certificate-oidc-issuer "$EMETTEUR" "$empreinte" >/dev/null

  echo "$nom : $empreinte (signée, SBOM attesté, vérifiée)"
  printf '%s %s\n' "$nom" "$empreinte" >> "$TRAVAIL/empreintes.txt"
}

pousser_env() {
  local dossier="$TRAVAIL/env"
  (umask 077 && printf '%s\n' "$ENV_DEMO_CLE" > "$TRAVAIL/cle" && printf '%s\n' "$CLE_HOTE_GITHUB" > "$TRAVAIL/hotes")
  GIT_SSH_COMMAND="ssh -i $TRAVAIL/cle -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes -o UserKnownHostsFile=$TRAVAIL/hotes" \
    git -C "$dossier" push -q "git@github.com:$GITHUB_REPOSITORY.git" "HEAD:refs/heads/$BRANCHE_ENV"
}

ecrire_env() {
  local dossier="$TRAVAIL/env" nom cible
  find "$dossier" -mindepth 1 -maxdepth 1 ! -name .git -exec rm -rf {} +
  mkdir -p "$dossier/$(dirname "$FICHIER_EMPREINTES")"
  {
    printf 'source: %s\n' "$COMMIT"
    printf 'images:\n'
    while read -r nom cible; do printf '  %s: %s\n' "$nom" "$cible"; done < "$TRAVAIL/empreintes.txt"
  } > "$dossier/$FICHIER_EMPREINTES"
  git -C "$dossier" add -A
  if git -C "$dossier" diff --cached --quiet; then
    echo "Empreintes et source inchangées : rien à écrire sur $BRANCHE_ENV."
    return 0
  fi
  git -C "$dossier" -c user.name="github-actions[bot]" \
    -c user.email="41898282+github-actions[bot]@users.noreply.github.com" \
    commit -q -m "chore(env-demo): publier les empreintes de $ETIQUETTE"
  pousser_env
  echo "Empreintes écrites sur $BRANCHE_ENV ($ETIQUETTE)."
}

printf '%s' "$GH_TOKEN" | docker login ghcr.io --username "$GITHUB_ACTOR" --password-stdin >/dev/null

preparer_env
SOURCE="$(lire_publie 'source: ')"
if source_connue; then
  echo "Comparaison depuis le commit source de $BRANCHE_ENV : $SOURCE"
else
  echo "Commit source absent ou inconnu : toutes les images sont reconstruites."
fi
: > "$TRAVAIL/empreintes.txt"
while IFS='|' read -r -u 3 nom dockerfile contexte chemins; do
  [[ -n "$nom" ]] || continue
  precedente="$(lire_publie "  $nom: ")"
  if [[ -n "$precedente" ]] && ! modifiee "$chemins"; then
    echo "$nom : inchangée depuis la source, empreinte conservée ($precedente)"
    printf '%s %s\n' "$nom" "$precedente" >> "$TRAVAIL/empreintes.txt"
    continue
  fi
  publier_image "$nom" "$dockerfile" "$contexte"
done 3< tools/ci/images.txt
ecrire_env
