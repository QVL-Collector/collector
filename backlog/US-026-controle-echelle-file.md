# US-026 — Absorber une vague de soumissions en ajustant le Contrôle à la file

| Champ | Valeur |
|---|---|
| Epic | E4 Exploitation, charge et disponibilité |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Coupable (ordre de coupe S2 n° 2, D01 : vague de soumissions en premier) |
| Estimation | 3 points |
| Dépend de | US-016, US-024 ; X8 validé par XP01 |
| Fonction(s) | DevOps, Test (scénario) |
| Service | Contrôle, Plateforme |
| Critères grille | C5 |
| Exigences | ENF-15, ENF-16 |
| Branche | `feat/US-026-controle-echelle-file` |

## Récit

En tant que vendeur, je veux que mon article soit contrôlé rapidement même pendant une vague de soumissions après un salon, afin de le voir publié sans attendre, sans ralentir la consultation du catalogue.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given le Contrôle à 1 consommateur, When la longueur de sa file dépasse le seuil KEDA pendant une vague de soumissions, Then au moins 2 consommateurs sont prêts en 60 secondes au plus, 3 au plus.
- **AC2** — Given la file vidée, When la période de refroidissement est écoulée, Then le Contrôle revient à 1 consommateur.
- **AC3** — Given la vague de soumissions, When elle est jouée en même temps que le tir sur le catalogue, Then le p95 de la consultation reste inférieur à 500 ms.
- **AC4** — Given `tools/demo.sh`, When le palier « vague de soumissions » est atteint, Then il utilise des jetons de vendeurs de test obtenus juste avant le tir, un vendeur par utilisateur virtuel, dans un fichier hors dépôt.

### Non fonctionnels

- **AC5 (Sécurité)** — Given l'identité de KEDA sur le broker, When ses droits sont relus, Then elle ne peut que lire la longueur des files, sans lire ni publier de message.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-38 | Identité de KEDA en lecture de métriques seule | AC5 |
| ES-06 | Jetons de tir hors dépôt | AC4 |

Mesures : MC4, MT5.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Charge | k6, Grafana | AC1 à AC4 | DevOps (tir), Test (scénario) |
| Sécurité négative | Publication avec l'identité de KEDA | AC5 | Test |

## Hors périmètre

- Troisième profil de mise à l'échelle (Interactions) : US-033.

## Décisions liées

- [D03](../docs/decisions/D03-hebergement-orchestration.md) (X8) · [D13](../docs/decisions/D13-strategie-tests-charge.md) (MT5)

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
