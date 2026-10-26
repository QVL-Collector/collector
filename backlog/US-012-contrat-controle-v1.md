# US-012 — Contrat v1 du contrôle de publication

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | Livrée |
| Sprint | 1 (J4) |
| Engagement | Ferme |
| Estimation | 2 points |
| Dépend de | US-002 |
| Fonction(s) | Back |
| Service | Place de marché, Contrôle (`contracts/messages/`, `libs/platform`) |
| Critères grille | C2, C7 |
| Exigences | ENF-17, ENF-19, EF-42 (partiel) |
| Branche | `feat/US-012-contrat-controle-v1` |

## Récit

En tant que direction de Collector, je veux que le contrôle des annonces repose sur un contrat d'échange documenté et versionné, afin de pouvoir brancher plus tard un outil de détection des fraudes, interne ou acheté, sans réécrire la Place de marché.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given `contracts/messages/`, When le contrat v1 est publié, Then il contient les JSON Schema de la demande et du résultat avec l'enveloppe commune : version du contrat, identifiant de corrélation, identifiant de l'article, numéro de révision, identifiant opaque du compte et attributs de statut utiles, nombre de photos, version des règles, issue (publié, refusé, à vérifier) et catégorie de motif.
- **AC2** — Given les exemples valides et invalides versionnés avec le contrat, When la validation de `libs/platform` les lit, Then chaque exemple valide est accepté et chaque exemple invalide est refusé avec la raison.
- **AC3** — Given un message dont l'issue ou la catégorie de motif est hors liste fermée, ou dont la version est inconnue, When il est validé, Then il est refusé.

### Non fonctionnels

- **AC4 (Sécurité)** — Given un message contenant une clé dupliquée, un champ inconnu ou un email de vendeur, When il est validé, Then il est refusé (données minimales, validation stricte).
- **AC5 (Sécurité)** — Given un résultat, When son motif est lu, Then il donne une catégorie et jamais la règle exacte ni le seuil appliqué.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-21 | Contrat versionné, données minimales, point de branchement de l'outil de fraude | AC1, AC4 |
| ES-08 | Validation stricte, clés dupliquées refusées | AC3, AC4 |

Mesures : MC6.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Contrat | `jsonschema` dans `go test`, exemples de `contracts/` | AC1 à AC5 | Back ; Test (exemples) |
| Unitaire back | Fuzzing natif du validateur (`-fuzztime=30s`) | AC3, AC4 | Back |

## Hors périmètre

- Test de contrat de la paire producteur-consommateur : US-013.
- Adaptateur de l'outil de fraude : hors POC.

## Décisions liées

- [D01 — Fonctionnalité du POC](../docs/decisions/D01-fonctionnalite-poc.md) · [D05 — Communication interservice](../docs/decisions/D05-communication-interservice.md) · [architecture-technique.md](../docs/architecture/architecture-technique.md) § 8

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
| Validation Martin | 2026-10-26 |
| Livraison (PR / merge) | https://github.com/QVL-Collector/collector/pull/4 |
