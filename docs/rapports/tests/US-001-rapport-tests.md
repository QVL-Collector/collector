# US-001 — Rapport de preuves

Date : 2026-10-26. Dépôt : https://github.com/QVL-Collector/collector (public).

| AC | Preuve | Résultat |
|---|---|---|
| AC1 | `gitleaks git` sur tout l'historique (28 commits), puis de nouveau après réécriture (37 commits, 3 branches) | Aucune fuite |
| AC2 | `git log --format='%an <%ae>\|%cn <%ce>'` sur `main` et les branches poussées | 37 commits sur 37 : `Martin Queval <116591403+MartinQueval@users.noreply.github.com>` en auteur et en committer |
| AC3 | Dépôt créé vide par l'API (`auto_init: false`), puis poussée de `main` | `* [new branch] main -> main`, historique identique au local réécrit |
| AC4 | Relevé de `tools/github/configurer-depot.sh --relever` | `secret_scanning` et `secret_scanning_push_protection` à `enabled`, alertes et correctifs Dependabot actifs, Dependabot des actions dans `.github/dependabot.yml` |
| AC5 | `git check-ignore` sur `*.tfstate`, `.env`, exports de realm | Fichiers ignorés ; `gitleaks dir` ne trouve des secrets que dans les `.env` et `.genere/` ignorés des bacs à sable |
| AC6 | Poussée d'une branche jetable contenant une fausse paire de clés AWS | Refusée : `GH013: Repository rule violations — GITHUB PUSH PROTECTION — Push cannot contain secrets (Amazon AWS Secret Access Key)`. Branche locale supprimée |

## Constat complémentaire

Un premier essai d'AC6 avec un faux jeton GitHub (`ghp_…`, somme de contrôle valide mais jeton inexistant) **n'a pas été bloqué** : la protection à la poussée ne bloque les jetons GitHub que s'ils sont réellement valides. La branche de test a été supprimée aussitôt. Le jeton était fictif et n'a jamais donné d'accès. Conséquence : la protection de GitHub ne suffit pas seule ; gitleaks en CI (US-003) et le hook local restent les barrières principales.

## Verdict

Conforme.
