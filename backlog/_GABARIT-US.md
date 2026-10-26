# US-xxx — Titre de la user story

Règles : [PILOTAGE.md](../docs/PILOTAGE.md) § 3.3 (circuit), § 5 (Definition of Done) ; tests : [tests.md](../docs/processus/tests.md) ; sécurité : [devsecops.md](../docs/processus/devsecops.md) § 2.1. Vue d'ensemble : [BACKLOG.md](BACKLOG.md).

| Champ | Valeur |
|---|---|
| Epic | E1 Chaîne de livraison · E2 Plateforme AWS · E3 Mise en vente contrôlée · E4 Exploitation, charge et disponibilité · E5 Diffusion et notifications · E6 Réservation |
| Statut | À faire / En cours / En test / En review / Validée Martin / Livrée |
| Sprint | 1 (J4) / 2 (J5) / 3 (J5 bis) |
| Engagement | Ferme / Coupable (rang dans l'ordre de coupe) |
| Estimation | n points (1, 2, 3, 5, 8 ; au-delà de 8, l'US est découpée) |
| Dépend de | US-xxx, … |
| Fonction(s) | Back / Front / DevOps |
| Service | Place de marché / Contrôle / Interactions / Transactions / Front / Plateforme |
| Critères grille | Cn, … |
| Exigences | EF-nn, ENF-nn |
| Branche | `feat/US-xxx-slug` |

## Récit

En tant que …, je veux … afin de ….

## Critères d'acceptation

Chaque AC est numéroté et porte l'étiquette de test `@US-xxx-ACn`. Un AC ambigu est renvoyé à Martin, jamais interprété.

### Fonctionnels

- **AC1** — Given … When … Then …
- **AC2** — Given … When … Then …

### Non fonctionnels

Garder seulement les familles qui s'appliquent ; les AC de sécurité sont rédigés en tests négatifs (refus attendu).

- **AC3 (Autorisation)** — Given … When … Then …
- **AC4 (Performance)** — Given … When … Then …
- **AC5 (Accessibilité)** — Given … When … Then …
- **AC6 (Internationalisation)** — Given … When … Then …

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-nn | | ACn |

Frontières de confiance touchées : FCn ([exigences-securite.md](../docs/securite/exigences-securite.md) § 3). Mesures des décisions : MSn, MCn, MAn, MDn…

## Tests attendus

| Type ([tests.md](../docs/processus/tests.md) § 2) | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire / Intégration / Contrat / Acceptation API / E2E / Sécurité négative / Après déploiement / Charge | | ACn | Back / Front / Test / DevOps |

## Hors périmètre

- Ce que l'US ne couvre pas.

## Décisions liées

- [Dxx — titre](../docs/decisions/Dxx-slug.md)

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
