#!/usr/bin/env bash
# Genere les secrets d'une session XP03 dans .env (ignore par git) : base, compte d'administration
# initial (temporaire), comptes de test, graine OTP de l'admin de test, secrets des clients
# confidentiels (MX1, MA9, MA10). Rien n'est affiche.
# Usage : scripts/identifiants.sh [--force]
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

if [[ -f "$ENV_FILE" && "${1:-}" != "--force" ]]; then
  echec "$ENV_FILE existe deja (session en cours ?) ; --force pour regenerer"
fi

aleatoire() { openssl rand -base64 64 | tr -dc 'A-Za-z0-9' | head -c "$1"; }

umask 077
tmp="$(mktemp "$XP_DIR/.env.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
{
  printf '# Genere par scripts/identifiants.sh le %s ; ne jamais versionner.\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf 'MDP_POSTGRES=%s\n' "$(aleatoire 32)"
  printf 'MDP_KC_BOOTSTRAP=%s\n' "$(aleatoire 32)"
  printf 'MDP_ACHETEUR=%s\n' "$(aleatoire 24)"
  printf 'MDP_VENDEUR=%s\n' "$(aleatoire 24)"
  printf 'MDP_ADMIN_TEST=%s\n' "$(aleatoire 24)"
  printf 'GRAINE_OTP=%s\n' "$(aleatoire 20)"
  printf 'SECRET_TRANSACTIONS=%s\n' "$(aleatoire 40)"
  printf 'SECRET_AUTRE=%s\n' "$(aleatoire 40)"
} >"$tmp"
mv "$tmp" "$ENV_FILE"
trap - EXIT
journal "Secrets de session generes dans .env (ignore par git)"
