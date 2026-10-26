# US-027 — Limiter le débit sur les routes sensibles

| Champ | Valeur |
|---|---|
| Epic | E4 Exploitation, charge et disponibilité |
| Statut | À faire |
| Sprint | 2 (J5), ou 3 si reportée |
| Engagement | Coupable (ordre de coupe S2 n° 1 : premier élément repoussé au sprint 3) |
| Estimation | 2 points |
| Dépend de | US-016 |
| Fonction(s) | Back (`libs/platform`), DevOps (adresse de tir) |
| Service | Place de marché, Interactions, Transactions |
| Critères grille | C8, C5 |
| Exigences | ENF-15 |
| Branche | `feat/US-027-limitation-debit` |

## Récit

En tant que plateforme, je veux freiner une même origine qui abuse des routes de soumission, de dépôt de photo et de réservation, afin qu'un script ne puisse ni saturer le contrôle ni monopoliser les objets rares.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given les routes de soumission, de dépôt de photo et de réservation, When une même adresse ou un même compte dépasse la limite configurée, Then la réponse est 429 avec l'en-tête `Retry-After`.
- **AC2** — Given l'adresse du client, When elle est lue, Then c'est la dernière valeur de `X-Forwarded-For` ajoutée par l'ALB (mode `append`).
- **AC3** — Given un tir de charge, When `up.sh` reçoit l'adresse de Martin pour la session, Then la limite n'est relevée que pour cette adresse, injectée à l'exécution et jamais versionnée ; aucune route de contournement n'existe.

### Non fonctionnels

- **AC4 (Sécurité)** — Given un `X-Forwarded-For` forgé (valeur unique ou liste), When la limite est atteinte, Then la réponse reste 429 (MT4, XE5).
- **AC5 (Sécurité)** — Given la fin de chaque tir, When le test de limitation est rejoué, Then il obtient 429.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-29 | Limitation par origine et par compte | AC1, AC2, AC4, AC5 |
| ES-09 | Freinage des abus (la connexion reste protégée par Keycloak) | AC1 |

Écart V1 assumé : limite par réplica, non partagée (architecture § 9.8). Mesures : ME4, MT4, MT6.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire back | `go test` (seau de jetons) | AC1, AC2 | Back |
| Sécurité négative | Script curl après déploiement | AC4, AC5 | Test |
| Contrôle manuel tracé | Relecture des variables de session | AC3 | Revue de code |

## Hors périmètre

- Règle WAF, limite partagée : V1.

## Décisions liées

- [D08](../docs/decisions/D08-exposition-tls-domaine.md) (ME4) · [D13](../docs/decisions/D13-strategie-tests-charge.md) (MT4, MT6)

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
