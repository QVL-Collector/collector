# US-029 — Aucune notification perdue quand Interactions est arrêté

| Champ | Valeur |
|---|---|
| Epic | E5 Diffusion et notifications |
| Statut | À faire |
| Sprint | 3 (J5 bis) |
| Engagement | Ferme |
| Estimation | 2 points |
| Dépend de | US-028 |
| Fonction(s) | Back, DevOps (abonnement de test) |
| Service | Interactions, Place de marché |
| Critères grille | C2, C5 |
| Exigences | ENF-12 |
| Branche | `feat/US-029-reprise-interactions` |

## Récit

En tant que vendeur, je veux recevoir toutes mes notifications même si le service de notification a été indisponible, afin de ne manquer aucune décision sur mes articles.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given Interactions arrêté, When des articles sont publiés, Then les publications et le catalogue ne sont pas affectés.
- **AC2** — Given Interactions relancé, When il reprend sa file, Then les notifications des décisions survenues pendant l'arrêt sont créées, sans perte ni doublon.
- **AC3** — Given un abonnement de test posé par l'exploitation avec une identité en lecture seule, When une décision est diffusée alors qu'Interactions est arrêté, Then l'abonnement de test reçoit le même événement (preuve d'un second abonné) ; il est retiré après le test.

### Non fonctionnels

- **AC4 (Sécurité)** — Given l'identité de l'abonnement de test, When elle tente de publier, Then la publication est refusée.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-31 | Notifications créées au retour, sans blocage des publications | AC1, AC2 |
| ES-38 | Abonnement posé par l'exploitation, identité en lecture seule | AC3, AC4 |

Mesure : M-CAN-06 (aucun service ne crée d'abonnement).

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Résilience | testcontainers-go (arrêt et relance) à chaque PR | AC1, AC2 | Back |
| Intégration | Abonnement de test | AC3, AC4 | DevOps, Back |
| Après déploiement | Mise à zéro du Deployment d'Interactions | AC1, AC2 | DevOps |

## Hors périmètre

- Mise à l'échelle d'Interactions : US-033.

## Décisions liées

- [D02](../docs/decisions/D02-decoupage-applicatif.md) (second abonné) · [D05](../docs/decisions/D05-communication-interservice.md)

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
