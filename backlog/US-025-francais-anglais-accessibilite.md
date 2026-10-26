# US-025 — Utiliser Collector en anglais, avec des écrans accessibles

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme |
| Estimation | 3 points |
| Dépend de | US-022, US-023 |
| Fonction(s) | Front, DevOps (portes CI) |
| Service | Front |
| Critères grille | C3, C7 |
| Exigences | ENF-26, ENF-30 |
| Branche | `feat/US-025-francais-anglais-accessibilite` |

## Récit

En tant que collectionneur non francophone, je veux utiliser Collector en anglais ; et en tant qu'utilisateur de technologies d'assistance, je veux des écrans accessibles, afin que la plateforme puisse s'ouvrir à l'international et à tous.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given n'importe quel écran du POC, When l'utilisateur choisit l'anglais, Then tous les textes, y compris les motifs de refus et les messages d'erreur, s'affichent en anglais sans rechargement ni modification de code, et le choix est conservé pendant la session.
- **AC2** — Given les fichiers de traduction, When la CI s'exécute, Then le script de parité bloque toute clé manquante dans une langue et le lint bloque tout texte en dur.
- **AC3** — Given une PR, When l'acceptation s'exécute, Then le parcours est joué en français ; sur `main`, en français et en anglais.

### Non fonctionnels

- **AC4 (Accessibilité)** — Given chaque écran du POC (catalogue, fiche, connexion, espace vendeur, écran admin), When axe-core l'analyse en CI, Then aucune violation « serious » ou « critical » ; la porte est bloquante.
- **AC5 (Accessibilité)** — Given le parcours de mise en vente, When il est joué au clavier seul, Then il est réalisable de bout en bout, avec un focus visible ; la vérification manuelle est tracée en fin de sprint.

## Sécurité

Aucune exigence ES directement servie.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| i18n | Script de parité, parcours Playwright `fr` et `en` | AC1 à AC3 | Front, Test |
| Accessibilité | `@axe-core/playwright`, `jsx-a11y` | AC4 | Front, Test |
| Contrôle manuel tracé | Parcours au clavier | AC5 | Test |

## Hors périmètre

- Conformité RGAA complète : V1.

## Décisions liées

- [D01](../docs/decisions/D01-fonctionnalite-poc.md) (i18n et accessibilité, version minimale) · [D13](../docs/decisions/D13-strategie-tests-charge.md)

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
