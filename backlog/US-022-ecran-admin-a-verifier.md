# US-022 — Trancher les articles « à vérifier »

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme (ordre de coupe S2 n° 3 : écran minimal, une liste et deux actions) |
| Estimation | 5 points |
| Dépend de | US-019, US-021 |
| Fonction(s) | Back, Front, DevOps (second facteur dans le realm) |
| Service | Place de marché, Front, serveur d'autorisation |
| Critères grille | C3, C8 |
| Exigences | EF-19, ENF-03 |
| Branche | `feat/US-022-ecran-admin-a-verifier` |

## Récit

En tant qu'administrateur, je veux voir les articles « à vérifier » et les publier ou les refuser avec un motif, afin de trancher les cas que le contrôle automatique ne peut pas décider.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given des articles « à vérifier », When l'admin ouvre l'écran, Then il voit la liste avec, pour chacun, le contenu de la révision examinée et le motif du contrôle.
- **AC2** — Given un article de la liste, When l'admin le publie, Then il apparaît au catalogue ; When il le refuse avec un motif choisi dans la liste fermée, Then le vendeur voit l'état « refusé » et le motif.
- **AC3** — Given une décision portant sur la révision N, When l'article est passé à la révision N+1 entre-temps, Then la décision est refusée avec 409 et l'écran invite à recharger.
- **AC4** — Given une décision appliquée, When elle est journalisée, Then un événement d'audit indique l'admin, l'article, la révision, la décision et le motif.

### Non fonctionnels

- **AC5 (Autorisation, BP5)** — Given un vendeur ou un acheteur, When il appelle la liste ou la décision admin, Then la réponse est 403.
- **AC6 (Sécurité)** — Given le compte admin, When il se connecte, Then un second facteur (OTP) lui est demandé ; il n'est pas demandé au vendeur.
- **AC7 (Sécurité, sorties)** — Given un article à vérifier dont le titre contient du balisage, When l'écran admin l'affiche, Then il s'affiche comme du texte, sans violation de la politique de sécurité du contenu.
- **AC8 (Accessibilité)** — Given l'écran admin, When axe-core l'analyse, Then aucune violation « serious » ou « critical ».

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-03 | Seul l'admin tranche | AC5 |
| ES-10 | Second facteur pour l'admin | AC6 |
| ES-08 | Neutralisation des sorties sur l'écran admin | AC7 |
| ES-18 | Décision sur une révision, 409 si périmée | AC3 |
| ES-23 | Décision d'administration journalisée | AC4 |

Écart V1 assumé : écran servi par le front public, et non sur une origine distincte (ES-20). Menaces : M-ANN-03, M-BO-01, M-ANN-05.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Intégration | testcontainers-go | AC3, AC4 | Back |
| Acceptation API | Playwright `request` | AC2, AC3, AC5 | Test |
| E2E, CSP, accessibilité | Playwright, axe-core | AC1, AC6, AC7, AC8 | Test |

## Hors périmètre

- Back-office sur une origine distincte, nouvelle authentification avant action sensible : V1.

## Décisions liées

- [D02](../docs/decisions/D02-decoupage-applicatif.md) · [D06](../docs/decisions/D06-authentification-autorisation.md) (MFA admin) · [D07](../docs/decisions/D07-donnees.md) (décision avec révision)

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
