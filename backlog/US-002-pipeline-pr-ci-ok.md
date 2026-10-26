# US-002 — Pipeline de PR et vérification requise `ci-ok`

| Champ | Valeur |
|---|---|
| Epic | E1 Chaîne de livraison |
| Statut | En cours |
| Sprint | 1 (J4) |
| Engagement | Ferme |
| Estimation | 3 points |
| Dépend de | US-001 |
| Fonction(s) | DevOps |
| Service | Plateforme |
| Critères grille | C4, C7, C6 |
| Exigences | ENF-22, ENF-23 |
| Branche | `feat/US-002-pipeline-pr-ci-ok` |

## Récit

En tant que développeur, je veux que chaque PR vers `main` soit construite, vérifiée et testée automatiquement, et que `main` n'accepte que des PR vertes, afin qu'aucun changement non testé n'atteigne la branche livrable.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given une PR qui modifie un service Go, When la CI s'exécute, Then les jobs lint et types (`go vet`, golangci-lint), tests unitaires (`go test -race -cover`) et construction de l'image sans poussée s'exécutent pour ce seul service, et la couverture est publiée sans être bloquante.
- **AC2** — Given une PR qui modifie `web/`, When la CI s'exécute, Then `tsc --noEmit`, ESLint (dont `jsx-a11y` et `react/no-danger`) et Vitest s'exécutent.
- **AC3** — Given une PR qui ne modifie que `docs/`, When la CI s'exécute, Then les jobs sont ignorés et `ci-ok` est vert : la PR est fusionnable.
- **AC4** — Given un test unitaire en échec dans un service, When la CI s'exécute, Then `ci-ok` est rouge et la fusion est refusée par GitHub.
- **AC5** — Given deux poussées successives sur la même PR, When la seconde arrive, Then l'exécution précédente est annulée.
- **AC6** — Given le poste d'un développeur, When le hook pre-commit gitleaks est installé selon le README, Then un commit contenant un secret fictif est refusé localement.

### Non fonctionnels

- **AC7 (Sécurité)** — Given `main` protégée avec `ci-ok` comme vérification requise unique, When une poussée directe ou forcée sur `main` est tentée, Then elle est refusée.
- **AC8 (Sécurité)** — Given le workflow, When ses actions tierces sont relues, Then chacune est épinglée par empreinte de commit et les droits par défaut du jeton de la CI sont en lecture seule.
- **AC9 (Performance)** — Given une PR qui touche un seul service, When la CI s'exécute, Then elle se termine en moins de 15 minutes.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-32 | `main` protégée, PR obligatoire, vérification requise unique | AC4, AC7, AC8 |
| ES-06 | Hook pre-commit gitleaks | AC6 |

Mesures : MP1 à MP5 ([D10](../docs/decisions/D10-forge-ci-cd.md)).

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Contrôle de pipeline | PR de démonstration (docs seule, test rouge, double poussée) | AC1 à AC5, AC9 | Test |
| Sécurité négative | Poussée directe et forcée sur `main`, commit avec secret fictif | AC6, AC7 | Test |
| Contrôle manuel tracé | Relecture du workflow, règles de branche relevées | AC8 | Revue de code |

## Hors périmètre

- Portes de sécurité (gitleaks en CI, govulncheck, Semgrep, Trivy) : US-003.
- Publication des images, signature, branche `env/demo` : US-007.
- Seuil de couverture bloquant : sprint 2 (D12, « pas de baisse » à 0,5 point).

## Décisions liées

- [D10 — Forge git, CI/CD](../docs/decisions/D10-forge-ci-cd.md)
- [D12 — Indicateurs qualité](../docs/decisions/D12-indicateurs-qualite.md)

## Definition of Done

Reprise de [PILOTAGE.md](../docs/PILOTAGE.md) § 5, cochée au fil du circuit.

- [ ] Tous les AC sont couverts par des tests verts en CI (`ci-ok`).
- [ ] Revue de code faite, retours traités.
- [ ] Martin a testé et prononcé « je valide la livraison de la feature ».
- [ ] Documentation touchée à jour.
- [ ] Statut, date de validation et liens des rapports renseignés ci-dessous.
- [ ] Aucune donnée sensible ni credential ajouté au dépôt.

## Suivi

| Étape | Lien / date |
|---|---|
| Rapport de test | |
| Rapport de revue de code | |
| Validation Martin | AAAA-MM-JJ |
| Livraison (PR / merge) | |

### Réalisation et preuves restantes

Réalisé hors ligne le 2026-10-26, en attente de la publication du dépôt (US-001) : workflow [ci.yml](../.github/workflows/ci.yml), filtre [changements.sh](../tools/ci/changements.sh), outils épinglés [outils.yml](../tools/ci/outils.yml), [.golangci.yml](../.golangci.yml), [dependabot.yml](../.github/dependabot.yml), hook [pre-commit](../tools/git-hooks/pre-commit) et [.gitleaks.toml](../.gitleaks.toml), réglages [configurer-depot.sh](../tools/github/configurer-depot.sh).

Prouvé sur le poste : actionlint et zizmor (persona `pedantic`, hors ligne) sans constat ; shellcheck sans constat ; filtre de chemins sur un dépôt d'essai (docs seule → aucun job ; un service → ce service seul ; `libs/` → tous ; nom de module invalide → échec) ; étapes du job Go (`go mod verify`, `go vet`, golangci-lint, `go test -race -cover`) sur un module d'essai ; AC6 : commit d'un jeton fictif refusé par le hook, avec gitleaks installé et par le repli Docker.

| AC | Reste à prouver sur GitHub, après la poussée |
|---|---|
| AC1 | PR de démonstration ajoutant un service Go d'essai : seul son job s'exécute, couverture dans le résumé et en artefact |
| AC2 | Première PR du front (US-009 ou US-010) : `tsc`, contrôle des règles ESLint, ESLint et Vitest exécutés |
| AC3 | PR ne touchant que `docs/` : jobs « ignorés », `ci-ok` vert, PR fusionnable |
| AC4 | PR avec un test unitaire rouge : `ci-ok` rouge, bouton de fusion bloqué |
| AC5 | Deux poussées rapprochées sur une PR : la première exécution est annulée |
| AC7 | Après `configurer-depot.sh` : poussée directe et poussée forcée sur `main` refusées ; relevé `--relever` archivé |
| AC8 | Relecture du workflow (épinglage, `permissions`) et relevé des droits par défaut en lecture |
| AC9 | Durée de l'exécution de la PR d'AC1 relevée (< 15 min) |

