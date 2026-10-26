# US-021 — Recontrôler toute modification d'un article

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme |
| Estimation | 3 points |
| Dépend de | US-013 |
| Fonction(s) | Back, Front |
| Service | Place de marché, Front |
| Critères grille | C3, C8 |
| Exigences | EF-16, EF-18 |
| Branche | `feat/US-021-revalidation-apres-modification` |

## Récit

En tant que direction de Collector, je veux qu'un article modifié après sa publication repasse par le contrôle, afin qu'un vendeur ne puisse pas glisser des coordonnées ou un prix abusif dans une annonce déjà acceptée.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un article publié, When le vendeur modifie un seul champ (y compris le seul titre), Then une nouvelle révision est créée, l'article est retiré du catalogue et passe « en contrôle », sans seuil.
- **AC2** — Given un article « en contrôle » ou « à vérifier », When le vendeur le modifie, Then la nouvelle révision est soumise au contrôle et tout résultat ou décision portant sur l'ancienne révision est sans effet.
- **AC3** — Given la nouvelle révision acceptée, When le résultat arrive, Then l'article revient au catalogue avec le contenu modifié.
- **AC4** — Given l'écran du vendeur, When il modifie un article publié, Then il est averti que l'article sera retiré du catalogue le temps du contrôle.

### Non fonctionnels

- **AC5 (Sécurité)** — Given un résultat de contrôle redélivré par le canal après la nouvelle révision, When il est reçu, Then il est sans effet (aucune publication de l'ancienne version).

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-18 | Revalidation sans seuil, décisions sur une révision | AC1, AC2, AC5 |
| ES-31 | Fermé par défaut pendant la revalidation | AC1 |

Menaces : M-ANN-05, M-CAN-02.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire back | `go test` (machine à états, révisions) | AC1, AC2 | Back |
| Intégration | testcontainers-go (redélivraison) | AC5 | Back |
| Acceptation API et E2E | Playwright | AC1, AC3, AC4 | Test |

## Hors périmètre

- Article réservé : modification refusée, US-031.

## Décisions liées

- [D02 — Découpage](../docs/decisions/D02-decoupage-applicatif.md) (revalidation sans seuil) · [D01](../docs/decisions/D01-fonctionnalite-poc.md)

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
