# US-013 — Soumettre un article au contrôle automatique de complétude

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | À faire |
| Sprint | 1 (J4) |
| Engagement | Ferme — clôt le walking skeleton WS-2 |
| Estimation | 5 points |
| Dépend de | US-009, US-010, US-011, US-012 |
| Fonction(s) | Back (Place de marché, Contrôle), Front |
| Service | Place de marché, Contrôle, Front |
| Critères grille | C3, C2, C7 |
| Exigences | EF-16, EF-17 (complétude), EF-18, ENF-12, ENF-17 |
| Branche | `feat/US-013-soumission-controle-completude` |

## Récit

En tant que vendeur, je veux soumettre mon article et savoir s'il est publié ou refusé, et pourquoi, afin que seules des annonces complètes apparaissent au catalogue.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un brouillon complet (titre, description, catégorie, prix), When le vendeur le soumet, Then il passe « en contrôle », reste invisible au catalogue, puis devient « publié » et apparaît au catalogue en moins de 10 secondes.
- **AC2** — Given un brouillon sans description, When le vendeur le soumet, Then il devient « refusé » avec le motif « annonce incomplète », visible dans « Mes articles ».
- **AC3** — Given un article refusé, When le vendeur le corrige et le soumet de nouveau, Then il repasse « en contrôle » puis « publié ».
- **AC4** — Given une catégorie inconnue du Contrôle, When l'article est contrôlé, Then l'issue est « à vérifier », jamais « publié ».
- **AC5** — Given une soumission, When la Place de marché enregistre l'état « en contrôle », Then la demande est écrite dans l'outbox dans la même transaction, puis publiée sur le canal ; le résultat porte la version des règles v1.
- **AC6** — Given un résultat livré deux fois ou portant une révision antérieure à la révision courante, When la Place de marché le reçoit, Then il est sans effet.

### Non fonctionnels

- **AC7 (Sécurité)** — Given un résultat de contrôle publié par une autre identité que `controle` (`user-id` du message différent), When la Place de marché le reçoit, Then il est écarté et journalisé.
- **AC8 (Contrat)** — Given la Place de marché (producteur de demandes, consommateur de résultats) et le Contrôle, When le test de contrat s'exécute, Then les deux respectent le contrat v1.
- **AC9 (Autorisation)** — Given l'article du vendeur A, When le vendeur B tente de le soumettre, Then la réponse est 404.
- **AC10 (Accessibilité)** — Given le changement d'état affiché au vendeur, When il survient, Then il est annoncé aux technologies d'assistance (zone `aria-live`) et axe-core ne relève aucune violation « serious » ou « critical ».

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-31 | Fermé par défaut : invisible tant que non publié ; catégorie inconnue à vérifier | AC1, AC4 |
| ES-18 | Résultat périmé ou redélivré sans effet | AC6 |
| ES-38 | Émetteur du résultat vérifié | AC7 |
| ES-04 | Soumission réservée au propriétaire | AC9 |

Mesures : MC2, MC6, MC9. Frontière : canal (FC11).

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire back | `go test` (règle de complétude, idempotence) | AC2, AC4, AC6 | Back |
| Intégration | testcontainers-go (PostgreSQL, RabbitMQ) | AC5, AC6, AC7 | Back |
| Contrat | Test de la paire Place de marché – Contrôle | AC8 | Back |
| Acceptation API et E2E | Playwright, attente par `expect.poll` borné | AC1 à AC4, AC9, AC10 | Test |

## Hors périmètre

- Règles coordonnées, prix, statut vendeur, photo : sprint 2 (US-018 à US-020).
- Écran admin « à vérifier » : US-022.

## Décisions liées

- [D01 — Fonctionnalité du POC](../docs/decisions/D01-fonctionnalite-poc.md) · [D02 — Découpage, WS-2](../docs/decisions/D02-decoupage-applicatif.md) · [D05 — Communication interservice](../docs/decisions/D05-communication-interservice.md)

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
