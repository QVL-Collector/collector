# US-016 — Absorber un pic de visiteurs sur le catalogue

| Champ | Valeur |
|---|---|
| Epic | E4 Exploitation, charge et disponibilité |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme |
| Estimation | 5 points |
| Dépend de | US-010, US-015 |
| Fonction(s) | DevOps (HPA, tir, `demo.sh`), Test (scénario) |
| Service | Place de marché, Plateforme |
| Critères grille | C5, C10, C6 |
| Exigences | ENF-14, ENF-15, ENF-16 |
| Branche | `feat/US-016-montee-en-charge-catalogue` |

## Récit

En tant que direction de Collector, je veux que le catalogue reste rapide quand les visiteurs affluent après un salon ou une campagne, puis que les ressources reviennent au minimum, afin de vendre plus sans payer une capacité inutile.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given les seuils écrits dans `tests/charge/` avant le premier tir (p95 < 500 ms, erreurs < 1 %, contrôles de contenu ≥ 99 %), When le scénario k6 progressif est lancé depuis le poste à travers l'ALB sur la consultation du catalogue, Then k6 se termine avec un code de sortie nul.
- **AC2** — Given la Place de marché à 2 réplicas au repos, When la CPU dépasse le seuil du HPA pendant le tir, Then un nouveau pod est prêt en 90 secondes au plus et le nombre de réplicas monte jusqu'à 6 au plus.
- **AC3** — Given la fin du tir, When 10 minutes se sont écoulées, Then le nombre de réplicas est revenu à 2.
- **AC4** — Given des pods en attente faute de place, When le Cluster Autoscaler est retenu par XP01 (X9), Then un nœud est prêt en 5 minutes au plus, 4 nœuds au plus, et il est retiré après le tir.
- **AC5** — Given un environnement allumé et `tools/status.sh` vert, When `tools/demo.sh` est lancé, Then il affiche pods, nœuds, latence et erreurs, puis enchaîne les paliers nominal, montée et descente.
- **AC6** — Given chaque tir, When il se termine, Then le rapport HTML k6 et l'export des métriques sont archivés dans `docs/mesures/charge/<date>/`.

### Non fonctionnels

- **AC7 (Sécurité)** — Given un rapport de tir à publier, When il est relu avant commit, Then il ne contient ni jeton, ni adresse IP de Martin, ni paramètre de connexion.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-29 | Plafonds de mise à l'échelle, charge maîtrisée | AC2, AC4 |

Mesures : MS10, MT7, MT8.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Charge | k6, Prometheus, Grafana | AC1 à AC4, AC6 | DevOps (tir), Test (scénario) |
| Contrôle manuel tracé | Relecture MT7 | AC7 | Test |
| Charge (profil court, non bloquant) | k6 sur `main` | AC1 | DevOps |

## Hors périmètre

- Perte d'une instance pendant le tir : US-017.
- Vague de soumissions et KEDA : US-026.

## Décisions liées

- [D03 — Hébergement et orchestration](../docs/decisions/D03-hebergement-orchestration.md) (X7, X9) · [D13 — Stratégie de tests et de charge](../docs/decisions/D13-strategie-tests-charge.md) · [D11](../docs/decisions/D11-iac-allumage-cout.md) (`demo.sh`)

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
