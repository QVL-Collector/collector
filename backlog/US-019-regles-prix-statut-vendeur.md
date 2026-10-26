# US-019 — Mettre à vérifier les prix incohérents et les vendeurs à risque

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme |
| Estimation | 3 points |
| Dépend de | US-013 |
| Fonction(s) | Back |
| Service | Contrôle, Place de marché (module Comptes minimal) |
| Critères grille | C3, C2 |
| Exigences | EF-17, EF-42 (partiel) |
| Branche | `feat/US-019-regles-prix-statut-vendeur` |

## Récit

En tant qu'administrateur, je veux que les annonces au prix incohérent avec leur catégorie ou publiées par un vendeur à risque me soient soumises au lieu d'être publiées, afin de limiter la fraude sans relire toutes les annonces.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given les plages de prix par catégorie du jeu de données fictif, When un article a un prix hors de la plage de sa catégorie, Then son issue est « à vérifier » avec le motif « prix à confirmer ».
- **AC2** — Given un vendeur suspendu dans le module Comptes minimal, When il soumet un article, Then l'issue est « refusé » avec le motif « compte non autorisé à vendre ».
- **AC3** — Given un vendeur dont l'ancienneté est inférieure au seuil configuré, When il soumet un article au prix supérieur au seuil de vigilance de la catégorie, Then l'issue est « à vérifier ».
- **AC4** — Given une soumission, When la demande est construite, Then elle porte l'identifiant opaque du compte et les seuls attributs de statut (ancienneté, suspension) lus par une interface interne du module Comptes, sans email ni nom ; le Contrôle ne lit ni base ni horloge.
- **AC5** — Given un article complet, sans coordonnées, au prix dans la plage et d'un vendeur en règle, When il est contrôlé, Then il est publié.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-18 | Règles automatiques avant publication | AC1 à AC3 |
| ES-21 | Données minimales vers le contrôle | AC4 |

Menaces : M-FRA-01 (en partie), M-FRA-02.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire back | `go test` (bornes des plages et seuils) | AC1 à AC3, AC5 | Back |
| Contrat | Test de la paire (attributs de statut) | AC4 | Back |
| Acceptation API | Playwright `request` | AC1, AC2, AC5 | Test |

## Hors périmètre

- Gestion des catégories, des boutiques et des comptes : hors POC (données fictives).

## Décisions liées

- [D01](../docs/decisions/D01-fonctionnalite-poc.md) · [D02](../docs/decisions/D02-decoupage-applicatif.md) (module Comptes minimal)

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
