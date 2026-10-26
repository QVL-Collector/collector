# US-033 — Rejouer la charge avec les quatre services et ajuster Interactions à sa file

| Champ | Valeur |
|---|---|
| Epic | E4 Exploitation, charge et disponibilité |
| Statut | À faire |
| Sprint | 3 (J5 bis) |
| Engagement | Coupable (ordre de coupe S3 n° 2, D02) |
| Estimation | 2 points |
| Dépend de | US-016, US-028, US-030 |
| Fonction(s) | DevOps, Test |
| Service | Plateforme, Interactions |
| Critères grille | C5, C6 |
| Exigences | ENF-14, ENF-15, ENF-16 |
| Branche | `feat/US-033-non-regression-quatre-services` |

## Récit

En tant que Lead Developer, je veux vérifier que l'ajout d'Interactions et de Transactions n'a pas dégradé la montée en charge, et montrer un troisième profil de mise à l'échelle, afin de présenter en soutenance une plateforme complète et mesurée.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given les quatre services déployés, When le tir de consultation d'US-016 est rejoué avec les mêmes seuils, Then k6 se termine avec un code de sortie nul et les mesures sont comparées à celles du sprint 2 dans le relevé d'indicateurs.
- **AC2** — Given Interactions à 1 consommateur, When la longueur de sa file dépasse le seuil KEDA pendant une vague de décisions, Then un second consommateur est prêt en 60 secondes au plus, puis le service revient à 1 après vidage.
- **AC3** — Given une réservation pendant le tir, When le tir se termine, Then aucune double réservation n'est constatée en base.

## Sécurité

Aucune exigence ES nouvelle ; les plafonds de mise à l'échelle (ES-29) restent appliqués.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Charge | k6, Prometheus, Grafana | AC1, AC2 | DevOps (tir), Test (scénario) |
| Contrôle SQL | Requête de comptage | AC3 | Test |

## Hors périmètre

- Tir sur la réservation : chaque réservation consomme un objet unique (D02) ; la double réservation est prouvée par le test de concurrence d'US-031.

## Décisions liées

- [D02](../docs/decisions/D02-decoupage-applicatif.md) (sprint 3) · [D13](../docs/decisions/D13-strategie-tests-charge.md)

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
