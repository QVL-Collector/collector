# US-001 — Publier le dépôt sur GitHub sans fuite

| Champ | Valeur |
|---|---|
| Epic | E1 Chaîne de livraison |
| Statut | Livrée |
| Sprint | 1 (J4) |
| Engagement | Ferme |
| Estimation | 2 points |
| Dépend de | — (prérequis : choix de l'adresse e-mail publique des commits par Martin) |
| Fonction(s) | DevOps |
| Service | Plateforme |
| Critères grille | C4, C8 |
| Exigences | ENF-06, ENF-22 |
| Branche | `feat/US-001-publication-depot-github` |

## Récit

En tant que Lead Developer, je veux publier le dépôt local sur un dépôt GitHub public après avoir prouvé qu'il ne contient aucun secret, afin que la CI, le registre et le déploiement continu puissent démarrer sans exposer de donnée sensible.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given l'historique complet du dépôt local, When gitleaks analyse tous les commits, Then aucun résultat n'est trouvé et le rapport est archivé.
- **AC2** — Given le choix d'adresse e-mail publique fait par Martin, When l'auteur et le committer de chaque commit sont listés, Then tous portent l'adresse choisie (adresse personnelle assumée, ou adresse `noreply` GitHub après réécriture faite avant la première poussée).
- **AC3** — Given un dépôt GitHub public créé vide (sans README ni licence générés), When `main` est poussée, Then l'historique distant est identique à l'historique local retenu en AC2.
- **AC4** — Given le dépôt publié, When les réglages sont relus, Then la détection de secrets à la poussée, les alertes Dependabot et les mises à jour Dependabot des actions sont activées.

### Non fonctionnels

- **AC5 (Sécurité)** — Given le `.gitignore` du dépôt, When un fichier `*.tfstate`, `.env` ou un export de realm est créé dans le dépôt, Then `git status` ne le propose pas au suivi.
- **AC6 (Sécurité)** — Given la détection à la poussée activée, When une branche de test contenant un secret fictif au format reconnu est poussée, Then GitHub refuse la poussée ; la branche locale est supprimée après la preuve.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-06 | Historique sans secret avant publication, détection à la poussée | AC1, AC5, AC6 |
| ES-32 | Dépôt créé proprement, base de la protection de `main` (US-002) | AC3, AC4 |

Mesures : MP8. Frontière : chaîne de livraison.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Sécurité statique | gitleaks (historique complet) | AC1 | DevOps |
| Contrôle manuel tracé | `git log --format`, réglages GitHub relevés | AC2, AC3, AC4 | DevOps, Test |
| Sécurité négative | Poussée d'un secret fictif, fichier interdit | AC5, AC6 | Test |

## Hors périmètre

- Workflow CI, hook pre-commit et protection de `main` : US-002.

## Décisions liées

- [D10 — Forge git, CI/CD](../docs/decisions/D10-forge-ci-cd.md) (« Avant le premier envoi vers GitHub »)

## Definition of Done

Reprise de [PILOTAGE.md](../docs/PILOTAGE.md) § 5, cochée au fil du circuit.

- [ ] Tous les AC sont couverts par des tests verts en CI (`ci-ok`) ou, avant l'existence de la CI, par une preuve tracée.
- [ ] Revue de code faite, retours traités.
- [ ] Martin a testé et prononcé « je valide la livraison de la feature ».
- [ ] Documentation touchée à jour.
- [ ] Statut, date de validation et liens des rapports renseignés ci-dessous.
- [ ] Aucune donnée sensible ni credential ajouté au dépôt.

## Suivi

| Étape | Lien / date |
|---|---|
| Rapport de test | [US-001-rapport-tests.md](../docs/rapports/tests/US-001-rapport-tests.md) |
| Rapport de revue de code | |
| Validation Martin | 2026-10-26 |
| Livraison (PR / merge) | https://github.com/QVL-Collector/collector/pull/9 |
