# US-032 — Suivre ses réservations dans l'espace acheteur

| Champ | Valeur |
|---|---|
| Epic | E6 Réservation |
| Statut | À faire |
| Sprint | 3 (J5 bis) |
| Engagement | Coupable (ordre de coupe S3 n° 1, D02 : l'écran de confirmation d'US-030 suffit à la démonstration) |
| Estimation | 2 points |
| Dépend de | US-031 |
| Fonction(s) | Back, Front |
| Service | Transactions, Front |
| Critères grille | C3 |
| Exigences | EF-23 (partielle) |
| Branche | `feat/US-032-espace-acheteur` |

## Récit

En tant qu'acheteur, je veux retrouver mes réservations avec leur montant et leur échéance, afin de savoir combien de temps il me reste pour payer.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un acheteur avec des réservations, When il ouvre son espace, Then il voit, pour chacune, l'article, le montant total, l'échéance et l'état (en attente de paiement, expirée).
- **AC2** — Given une réservation dont l'échéance passe pendant la consultation, When la liste est rafraîchie, Then l'état devient « expirée ».

### Non fonctionnels

- **AC3 (Autorisation, BP5)** — Given les réservations de l'acheteur A, When l'acheteur B appelle la liste ou le détail, Then il ne voit que les siennes ; le détail d'une réservation de A répond 404.
- **AC4 (Accessibilité)** — Given la liste, When axe-core l'analyse, Then aucune violation « serious » ou « critical ».

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-04 | Contrôle d'accès par propriété sur les commandes | AC3 |

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Acceptation API | Playwright `request` | AC1 à AC3 | Test |
| E2E et accessibilité | Playwright, axe-core | AC1, AC4 | Test |

## Hors périmètre

- Paiement : hors POC.

## Décisions liées

- [D02](../docs/decisions/D02-decoupage-applicatif.md) (ordre de coupe du sprint 3)

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
