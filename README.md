# Collector

Collector.shop est une marketplace d'objets de collection entre particuliers.
Ce dépôt contient le POC (proof of concept) de Collector.shop, réalisé dans le cadre du bloc CESI « Superviser et assurer le développement des applications logicielles ».

Les règles de travail du projet (rôles, circuits, statuts, Definition of Done, conventions git) sont dans [docs/PILOTAGE.md](docs/PILOTAGE.md).

## État actuel

- **Phase** : sprint 3 en cours (J5 bis). Sprints 1 et 2 clos sur le périmètre local ; les US qui demandent AWS sont reportées au sprint 3 : [sprint-1.md](backlog/sprints/sprint-1.md), [sprint-2.md](backlog/sprints/sprint-2.md), [sprint-3.md](backlog/sprints/sprint-3.md).
- **Décisions** : D01 à D14 tranchées ; D03, D05, D06 et D08 confirmées par XP01 à XP03 le 2026-11-20 (EKS, RabbitMQ, Keycloak, ALB). Suivi dans [docs/decisions/INDEX.md](docs/decisions/INDEX.md).
- **Code** : quatre services Go dans [services/](services/) (Place de marché, Contrôle, Interactions, Transactions), front React dans [web/](web/), canal et identités dans [infra/](infra/README.md), scripts d'allumage et d'extinction AWS dans [tools/](tools/README.md) (US-004, en cours).

## Démarrer / arrêter

Rien ne tourne en continu : l'environnement AWS est allumé pour un usage précis, puis éteint ([D11](docs/decisions/D11-iac-allumage-cout.md)).

```bash
bash tools/bootstrap.sh            # une fois (couche persistante), rejouable sans effet
bash tools/up.sh --fin-lab 17:05   # heure de fin lue sur le compteur du lab
bash tools/down.sh                 # à terminer avant « End Lab »
```

`bash tools/status.sh` affiche l'état et le coût estimé ; `bash tools/alb.sh ouvrir|fermer` ouvre l'ALB à Internet après un contrôle vert. IaC dans [infra/](infra/README.md), scripts dans [tools/](tools/README.md). `demo.sh` est à venir.

## Contribuer : hook pre-commit et CI

**Hook pre-commit gitleaks** (obligatoire, une fois par clone) : il refuse tout commit dont les changements indexés contiennent un secret.

```bash
git config core.hooksPath tools/git-hooks
```

Il utilise `gitleaks` s'il est installé, sinon l'image épinglée de [tools/ci/outils.yml](tools/ci/outils.yml) (Docker requis). Un faux positif se traite dans [.gitleaks.toml](.gitleaks.toml), jamais par `--no-verify`.

**CI** ([.github/workflows/ci.yml](.github/workflows/ci.yml)) : sur chaque PR vers `main` et chaque poussée sur `main`. Processus et portes : [cicd.md](docs/processus/cicd.md). `ci-ok` est la seule vérification requise ; une PR qui ne touche que la documentation la rend verte sans autre job.

| Chemin modifié | Jobs exécutés |
|---|---|
| `services/<service>/**`, `infra/<module>/**` (si `go.mod`) | Go pour ce module |
| `libs/**`, `contracts/**`, `.github/**`, `tools/ci/**`, `.golangci.yml` | Go pour tous les modules (et front pour `contracts/` et l'outillage) |
| `web/**` | Front |
| `.github/**`, `tools/**`, `*.sh` | Workflows et scripts : actionlint, zizmor, shellcheck |

Ce que la CI attend d'un nouveau module :

- **Service Go** : `services/<service>/go.mod` (nom en minuscules), construit avec `GOWORK=off` ; `Dockerfile` dans le même dossier, construit depuis la racine du dépôt (`docker build -f services/<service>/Dockerfile .`). Job : `go mod verify`, `go vet`, golangci-lint ([.golangci.yml](.golangci.yml)), `go test -race -cover` (couverture publiée, non bloquante au sprint 1), construction de l'image sans poussée.
- **Module Go d'infrastructure** : `infra/<module>/go.mod` (par exemple [infra/identite/](infra/identite/README.md), tests du realm Keycloak par testcontainers), même job que les services, sans image.
- **Front** : `web/package.json` et `web/package-lock.json` ; dépendances de développement `typescript`, `eslint` avec `eslint-plugin-react` et `eslint-plugin-jsx-a11y`, `vitest` et `@vitest/coverage-v8`. `react/no-danger` et les règles `jsx-a11y` doivent être actives en erreur, sinon le job échoue.

**Portes de sécurité** (bloquantes, seuils de [devsecops.md](docs/processus/devsecops.md) § 3) : gitleaks (diff de la PR, historique complet sur `main`), govulncheck (vulnérabilité Go atteignable), Semgrep (règles du projet dans [tools/ci/semgrep/](tools/ci/semgrep/), sévérité ERROR), Trivy `fs` (verrou npm), `config` (infra, Dockerfile, manifests, charts) et `image` (critique ou élevée corrigeable, secret embarqué). La CI tourne aussi chaque jour sur `main`. Les journaux n'affichent que des compteurs ; le détail est publié en SARIF dans l'onglet sécurité, les compteurs I6 dans l'artefact `i6`.

**Exceptions** : uniquement dans [.trivyignore](.trivyignore), une entrée précédée de ses champs (composant, justification, mesure compensatoire, validation, date de création) et d'une échéance `exp:` à 30 jours au plus ; `tools/ci/verifier-exceptions.sh` refuse une entrée incomplète ou échue. Aucune exception Semgrep en ligne (`nosemgrep` est ignoré).

Mêmes outils sur le poste : `docker compose -f tools/ci/outils.yml run --rm actionlint` (ou `zizmor --offline .github`, `shellcheck <script>`) ; portes de sécurité : `bash tools/ci/semgrep.sh`, `bash tools/ci/trivy.sh config . config`, `bash tools/ci/verifier-exceptions.sh` (Docker et `jq` requis). Réglages du dépôt GitHub (protection de `main`, détection de secrets) : [tools/github/configurer-depot.sh](tools/github/configurer-depot.sh).

## Carte des documents

Les documents marqués « à venir » sont prévus mais pas encore rédigés.

### Pilotage et cadrage

| Emplacement | Contenu | État |
|---|---|---|
| [README.md](README.md) | Entrée du dépôt : objectif, état, démarrer/arrêter, carte des documents | Disponible |
| [docs/PILOTAGE.md](docs/PILOTAGE.md) | Règles du projet : rôles, circuits, statuts, Definition of Done, conventions git | Disponible |
| [docs/00-cadrage/contexte-et-grille.md](docs/00-cadrage/contexte-et-grille.md) | Contexte de l'entreprise, travail demandé, grille d'évaluation | Disponible |
| [docs/00-cadrage/exigences.md](docs/00-cadrage/exigences.md) | Exigences fonctionnelles et non fonctionnelles reformulées | Disponible |
| [docs/00-cadrage/environnement-learner-lab.md](docs/00-cadrage/environnement-learner-lab.md) | Contraintes de l'environnement AWS Academy Learner Lab (recherche documentaire du 2026-10-09, à confirmer par la sonde du premier jour) | Disponible |
| [docs/decisions/D01-fonctionnalite-poc.md](docs/decisions/D01-fonctionnalite-poc.md) | D01 : exigences reformulées et fonctionnalité métier du POC | Disponible |
| [docs/decisions/D02-decoupage-applicatif.md](docs/decisions/D02-decoupage-applicatif.md) | D02 : architecture cible V1, découpage applicatif, périmètre du POC | Disponible |
| [docs/decisions/D03-hebergement-orchestration.md](docs/decisions/D03-hebergement-orchestration.md) | D03 : hébergement et orchestration AWS (Confirmée (XP01) : EKS, repli ECS Fargate) | Disponible |
| [docs/decisions/D04-stack.md](docs/decisions/D04-stack.md) | D04 : stack back et front (Go ; React, TypeScript et MUI) | Disponible |
| [docs/decisions/](docs/decisions/) | Décisions D05 à D14 (`Dxx-*.md`) | À venir |
| [docs/decisions/INDEX.md](docs/decisions/INDEX.md) | Index et statut des décisions | Disponible |
| [docs/decisions/_GABARIT.md](docs/decisions/_GABARIT.md) | Gabarit d'une décision | Disponible |

### Architecture, processus, sécurité

| Emplacement | Contenu | État |
|---|---|---|
| [docs/architecture/architecture-technique.md](docs/architecture/architecture-technique.md) | Architecture logique de la V1 (option B, quatre services) et tranche du POC, v0.2 | Disponible |
| [docs/architecture/schemas/](docs/architecture/schemas/) | Schémas (sources et exports) | À venir |
| [docs/processus/](docs/processus/) | `qualite-indicateurs.md`, `devsecops.md`, `cicd.md`, `tests.md`, `competences-formation.md` | À venir |
| [docs/securite/exigences-securite.md](docs/securite/exigences-securite.md) | Analyse de menaces (validée, révisée pour l'option B) et exigences de sécurité V1 | Disponible |
| [docs/securite/](docs/securite/) | `audit-v1.md`, `plan-remediation.md` | À venir |
| `docs/exploitation.md` | Services, ports, variables (noms seulement), démarrer/arrêter, démo | À venir |

### Expérimentations, mesures, audits

| Emplacement | Contenu | État |
|---|---|---|
| [docs/experimentations/](docs/experimentations/) | Protocoles et résultats des expérimentations (`XPnn-*.md`) | À venir |
| [docs/experimentations/_GABARIT.md](docs/experimentations/_GABARIT.md) | Gabarit d'une expérimentation | Disponible |
| [sandbox/](sandbox/) | Scripts des expérimentations (`XPnn-*/`), jetables | À venir |
| [docs/mesures/](docs/mesures/) | `indicateurs-sprint-N.md`, `analyse-dette.md` | À venir |
| [docs/mesures/charge/](docs/mesures/charge/) | Résultats des tests de charge | À venir |
| [docs/audits/](docs/audits/) | `sprint-N-architecture.md`, `sprint-N-securite.md` | À venir |

### Backlog

| Emplacement | Contenu | État |
|---|---|---|
| [backlog/](backlog/) | User stories (`US-xxx-*.md`), `README.md` du backlog | À venir |
| [backlog/_GABARIT-US.md](backlog/_GABARIT-US.md) | Gabarit d'une user story | Disponible |
| [backlog/sprints/](backlog/sprints/) | Contenu et objectifs de chaque sprint (`sprint-N.md`) | À venir |

### Code et infrastructure

| Emplacement | Contenu | État |
|---|---|---|
| [services/](services/) | Un dossier par service déployable (selon D02 et D04) | À venir |
| [web/](web/) | Front React : socle et catalogue public (US-010) | En cours |
| [tests/acceptance/](tests/acceptance/) | Tests d'acceptation (issus des AC des US) | À venir |
| [tests/charge/](tests/charge/) | Scénarios de test de charge | À venir |
| [infra/](infra/README.md) | IaC Terraform (couches persistante et éphémère), chart du front, socle Kubernetes (D03, D08, D11) | En cours |
| [tools/](tools/README.md) | `bootstrap.sh`, `up.sh`, `down.sh`, `status.sh`, `alb.sh`, contrôle après déploiement ; `demo.sh` à venir | En cours |
| [.github/workflows/ci.yml](.github/workflows/ci.yml) | Pipeline de PR et de `main`, vérification requise `ci-ok` (D10) | Disponible |
| [tools/ci/](tools/ci/), [tools/git-hooks/](tools/git-hooks/), [tools/github/](tools/github/) | Filtre de chemins et outils de CI, hook pre-commit, réglages du dépôt GitHub | Disponible |

### Soutenance

| Emplacement | Contenu | État |
|---|---|---|
| [presentation/](presentation/) | `plan-soutenance.md`, `demo-script.md`, `faq-architecture.md`, `faq-securite.md` | À venir |
