# US-028 — Être notifié de la décision de publication de son article

| Champ | Valeur |
|---|---|
| Epic | E5 Diffusion et notifications |
| Statut | À faire |
| Sprint | 3 (J5 bis) |
| Engagement | Ferme |
| Estimation | 5 points |
| Dépend de | US-013, US-011, US-025 |
| Fonction(s) | Back (Interactions, émission par la Place de marché), Front, DevOps (identité, base, chart) |
| Service | Interactions, Place de marché, Front |
| Critères grille | C2, C3, C8 |
| Exigences | EF-33, EF-34 (partiels), ENF-19 |
| Branche | `feat/US-028-notification-vendeur` |

## Récit

En tant que vendeur, je veux retrouver dans mon espace une notification à chaque décision sur mes articles (publié, refusé, à vérifier), afin de savoir quand agir sans surveiller mes annonces.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given une décision de publication appliquée par la Place de marché (contrôle ou admin), When elle est enregistrée, Then l'événement « décision de publication » est écrit dans l'outbox de la même transaction puis diffusé sur le canal.
- **AC2** — Given l'événement diffusé, When Interactions le reçoit, Then il crée une seule notification pour le vendeur, listée dans son espace en moins de 10 secondes.
- **AC3** — Given le même événement livré deux fois, When Interactions le reçoit, Then une seule notification existe.
- **AC4** — Given une notification, When elle est affichée, Then son texte est construit à partir de la catégorie de motif, en français ou en anglais selon la langue choisie, sans aucun contenu saisi par un utilisateur.
- **AC5** — Given un message invalide (version inconnue, issue ou motif hors liste fermée), When Interactions le reçoit, Then il est écarté vers `.ecartes` après un nombre borné d'essais, sans bloquer les suivants.

### Non fonctionnels

- **AC6 (Autorisation, BP5)** — Given les notifications du vendeur A, When le vendeur B appelle la liste, Then il ne voit que les siennes ; sans jeton ou avec un jeton invalide, la réponse est 401.
- **AC7 (Sécurité)** — Given l'identité `interactions` sur le broker, When elle tente de publier une décision de publication ou un résultat de contrôle, Then la publication est refusée ; given `controle`, When il publie une décision de publication, Then elle est refusée.
- **AC8 (Sécurité)** — Given le compte de base d'Interactions, When il tente de lire la base `marche`, Then la base refuse ; et réciproquement.
- **AC9 (Contrat)** — Given la Place de marché et Interactions, When le test de contrat de l'événement s'exécute, Then les deux respectent le schéma v1.
- **AC10 (Accessibilité)** — Given la liste des notifications, When axe-core l'analyse, Then aucune violation « serious » ou « critical ».

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-38 | Seule la Place de marché publie la décision ; Interactions en lecture ; bases cloisonnées | AC7, AC8 |
| ES-04 | Liste filtrée par l'identité du jeton | AC6 |
| ES-08 | Texte sans contenu utilisateur, contrat validé à la réception | AC4, AC5 |
| ES-31 | Message invalide écarté sans blocage | AC5 |

Mesures : MC1, MC2, MC6, MC8 ; menaces M-NOT-02, M-CAN-01, M-CAN-03.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire back | `go test` | AC3, AC4 | Back |
| Intégration | testcontainers-go (PostgreSQL, RabbitMQ) | AC1 à AC3, AC5, AC7, AC8 | Back |
| Contrat | Paire Place de marché – Interactions | AC9 | Back |
| Acceptation API, E2E, accessibilité | Playwright, axe-core | AC2, AC6, AC10 | Test |

## Hors périmètre

- Emails, préférences, suivis, messagerie : hors POC.

## Décisions liées

- [D02](../docs/decisions/D02-decoupage-applicatif.md) (option P2) · [D05](../docs/decisions/D05-communication-interservice.md)

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
