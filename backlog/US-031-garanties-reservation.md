# US-031 — Garantir qu'un objet unique n'est réservé qu'une fois

| Champ | Valeur |
|---|---|
| Epic | E6 Réservation |
| Statut | À faire |
| Sprint | 3 (J5 bis) |
| Engagement | Ferme |
| Estimation | 3 points |
| Dépend de | US-030 |
| Fonction(s) | Back |
| Service | Place de marché, Transactions |
| Critères grille | C3, C8, C7 |
| Exigences | EF-23 (partielle) |
| Branche | `feat/US-031-garanties-reservation` |

## Récit

En tant qu'acheteur d'un objet rare, je veux être sûr qu'il ne sera pas réservé par deux personnes et qu'un double clic ne crée pas deux commandes, afin de pouvoir me fier à ma réservation.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un article disponible, When deux acheteurs le réservent simultanément, Then une seule réservation est acceptée et l'autre reçoit 409.
- **AC2** — Given une réservation existante, When le même acheteur relance « réserver » sur le même article (délai dépassé, double clic), Then la réservation existante est renvoyée avec la même échéance, sans seconde réservation ni seconde commande.
- **AC3** — Given un acheteur qui a déjà une réservation active, When il réserve un autre article, Then la réservation est refusée.
- **AC4** — Given une réservation dont l'échéance est passée, When l'article est consulté, Then il est de nouveau disponible et la commande est présentée « expirée ».
- **AC5** — Given un article réservé, When son vendeur tente de le modifier ou de le retirer, Then la demande est refusée (409) et le prix reste figé.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-14 | Réservation atomique et idempotente (réservation seulement) | AC1, AC2 |
| ES-13 | Prix figé pendant la réservation | AC5 |
| ES-29 | Une seule réservation active par acheteur, échéance plafonnée | AC3, AC4 |

Menaces : M-RES-01, M-RES-02, M-PAY-05.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Intégration (concurrence) | testcontainers-go, requêtes parallèles, `UPDATE` conditionnel | AC1 | Back, Test |
| Intégration | testcontainers-go | AC2 à AC5 | Back |
| Acceptation API | Playwright `request` (échéance courte de démonstration) | AC2 à AC5 | Test |

## Hors périmètre

- Compensation, remboursement d'une confirmation tardive : V1.

## Décisions liées

- [D02](../docs/decisions/D02-decoupage-applicatif.md) (nouvel essai, une réservation active) · [D07](../docs/decisions/D07-donnees.md) (réservation par mise à jour conditionnelle)

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
