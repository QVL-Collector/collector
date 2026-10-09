# Collector

Collector.shop est une marketplace d'objets de collection entre particuliers.
Ce dépôt contient le POC (proof of concept) de Collector.shop, réalisé dans le cadre du bloc CESI « Superviser et assurer le développement des applications logicielles ».

Les règles de travail du projet (rôles, circuits, statuts, Definition of Done, conventions git) sont dans [docs/PILOTAGE.md](docs/PILOTAGE.md).

## État actuel

- **Phase** : besoin et découpage en cours (jalon J1).
- **Décisions** : Session S1 close : [D01](docs/decisions/D01-fonctionnalite-poc.md) décidée (mise en vente d'un article avec contrôle automatisé de publication). Prochaine étape : D02 (découpage applicatif). Suivi dans [docs/decisions/INDEX.md](docs/decisions/INDEX.md).
- **Code** : aucun service, front ou infrastructure pour l'instant.

## Démarrer / arrêter

> À compléter après la décision D11 (IaC, allumage/extinction, garde-fou coût).

Principe déjà acté : rien ne tourne en continu. L'environnement est allumé pour un usage précis, puis éteint.
Scripts prévus dans [tools/](tools/) : `up.sh`, `down.sh`, `status.sh`, `demo.sh`.

## Carte des documents

Les documents marqués « à venir » sont prévus mais pas encore rédigés.

### Pilotage et cadrage

| Emplacement | Contenu | État |
|---|---|---|
| [README.md](README.md) | Entrée du dépôt : objectif, état, démarrer/arrêter, carte des documents | Disponible |
| [docs/PILOTAGE.md](docs/PILOTAGE.md) | Règles du projet : rôles, circuits, statuts, Definition of Done, conventions git | Disponible |
| [docs/00-cadrage/contexte-et-grille.md](docs/00-cadrage/contexte-et-grille.md) | Contexte de l'entreprise, travail demandé, grille d'évaluation | Disponible |
| [docs/00-cadrage/exigences.md](docs/00-cadrage/exigences.md) | Exigences fonctionnelles et non fonctionnelles reformulées | Disponible |
| [docs/decisions/D01-fonctionnalite-poc.md](docs/decisions/D01-fonctionnalite-poc.md) | D01 : exigences reformulées et fonctionnalité métier du POC | Disponible |
| [docs/decisions/](docs/decisions/) | Décisions D02 à D14 (`Dxx-*.md`) | À venir |
| [docs/decisions/INDEX.md](docs/decisions/INDEX.md) | Index et statut des décisions | Disponible |
| [docs/decisions/_GABARIT.md](docs/decisions/_GABARIT.md) | Gabarit d'une décision | Disponible |

### Architecture, processus, sécurité

| Emplacement | Contenu | État |
|---|---|---|
| [docs/architecture/](docs/architecture/) | `architecture-technique.md` | À venir |
| [docs/architecture/schemas/](docs/architecture/schemas/) | Schémas (sources et exports) | À venir |
| [docs/processus/](docs/processus/) | `qualite-indicateurs.md`, `devsecops.md`, `cicd.md`, `tests.md`, `competences-formation.md` | À venir |
| [docs/securite/exigences-securite.md](docs/securite/exigences-securite.md) | Analyse de menaces du besoin et exigences de sécurité V1 | Disponible |
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
| [web/](web/) | Front | À venir |
| [tests/acceptance/](tests/acceptance/) | Tests d'acceptation (issus des AC des US) | À venir |
| [tests/charge/](tests/charge/) | Scénarios de test de charge | À venir |
| [infra/](infra/) | IaC et déploiement (selon D03 et D11) | À venir |
| [tools/](tools/) | `up.sh`, `down.sh`, `status.sh`, `demo.sh` | À venir |
| Pipeline CI/CD | Emplacement selon la forge retenue (D10) | À venir |

### Soutenance

| Emplacement | Contenu | État |
|---|---|---|
| [presentation/](presentation/) | `plan-soutenance.md`, `demo-script.md`, `faq-architecture.md`, `faq-securite.md` | À venir |
