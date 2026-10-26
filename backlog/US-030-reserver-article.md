# US-030 — Réserver un article publié

| Champ | Valeur |
|---|---|
| Epic | E6 Réservation |
| Statut | À faire |
| Sprint | 3 (J5 bis) |
| Engagement | Ferme |
| Estimation | 5 points |
| Dépend de | US-010, US-006, US-008 |
| Fonction(s) | Back (Transactions, Place de marché), Front, DevOps (client Keycloak de service, port interne, politique réseau, base) |
| Service | Transactions, Place de marché, Front |
| Critères grille | C2, C3, C8 |
| Exigences | EF-23 (partielle), EF-24, ENF-01 |
| Branche | `feat/US-030-reserver-article` |

## Récit

En tant qu'acheteur, je veux réserver un objet publié, afin qu'il me soit garanti pendant le temps du paiement et que personne d'autre ne puisse l'acheter.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un acheteur connecté et un article publié disponible, When il clique « Réserver », Then Transactions obtient un jeton de service (audience Place de marché, droit `reserver`, durée courte), appelle « réserver » en propageant le jeton de l'acheteur, puis enregistre une commande « en attente de paiement ».
- **AC2** — Given la commande créée, When l'acheteur la consulte à l'écran de confirmation, Then il voit le prix de référence et les frais de port lus par la Place de marché dans ses propres données, la commission de 5 % sur le total, les montants au centime et l'échéance fixée par la Place de marché, plafonnée.
- **AC3** — Given l'article réservé, When un visiteur consulte le catalogue ou la fiche, Then l'article porte la mention « réservé » et n'est plus réservable.

### Non fonctionnels

- **AC4 (Autorisation)** — Given la route « réserver » de la Place de marché, When elle est appelée sans jeton de service, avec le jeton d'un utilisateur, d'un autre service, un jeton expiré ou destiné à une autre audience, Then la réponse est 401 ou 403.
- **AC5 (Autorisation, adjoint confus)** — Given un jeton de service valide, When l'identité d'acheteur propagée est absente, forgée ou ne correspond pas au jeton de l'acheteur, ou que l'article n'est pas publié, Then la réservation est refusée.
- **AC6 (Sécurité)** — Given un prix ou un montant envoyé dans la requête de l'acheteur, When la commande est créée, Then il est ignoré : les montants sont recalculés côté serveur.
- **AC7 (Sécurité, exposition)** — Given l'environnement allumé, When « réserver » est appelée depuis Internet, Then elle échoue faute de route (absente de l'ALB) ; depuis un pod autre que Transactions, Then la politique réseau la refuse.
- **AC8 (Sécurité, données)** — Given le compte de base de Transactions, When il tente de lire la base `marche`, Then la base refuse ; et réciproquement.
- **AC9 (Autorisation)** — Given le jeton de service de Transactions, When il est présenté sur une route d'acheteur, Then la réponse est 403.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-38 | Jeton de service à audience et droit limités, identité propagée et revérifiée, bases cloisonnées | AC1, AC4, AC5, AC8 |
| ES-13 | Montants recalculés côté serveur, prix lu chez le propriétaire | AC2, AC6 |
| ES-03 | Rôle acheteur ; jeton de service refusé sur les routes d'acheteur | AC9 |
| ES-07 | « Réserver » jamais routée par l'ALB | AC7 |

Mesures : MC7, MA3, MA5, MA6, MD1 ; menaces M-MOD-02, M-SYN-01. Frontière : appel synchrone entre services (FC11).

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire back | `go test` (montants au centime, commission) | AC2, AC6 | Back |
| Intégration | testcontainers-go, Keycloak de compose | AC1, AC4, AC5, AC8, AC9 | Back |
| Contrat | OpenAPI interne de « réserver » | AC1 | Back |
| Acceptation API et E2E | Playwright | AC1 à AC3, AC9 | Test |
| Après déploiement | Script curl depuis Internet, pod de test | AC7 | DevOps |

## Hors périmètre

- Concurrence, nouvel essai, échéance, article figé : US-031. Espace acheteur : US-032.
- Paiement, confirmation, séquestre : hors POC, documentés.

## Décisions liées

- [D02](../docs/decisions/D02-decoupage-applicatif.md) (option P2) · [D05](../docs/decisions/D05-communication-interservice.md) · [D06](../docs/decisions/D06-authentification-autorisation.md) · [D07](../docs/decisions/D07-donnees.md)

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
