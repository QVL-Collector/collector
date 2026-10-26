# US-023 — Filtrer le catalogue par catégorie et consulter la fiche d'un article

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | À faire |
| Sprint | 2 (J5) |
| Engagement | Ferme |
| Estimation | 3 points |
| Dépend de | US-010, US-020 |
| Fonction(s) | Back, Front |
| Service | Place de marché, Front |
| Critères grille | C3 |
| Exigences | EF-02, EF-04 |
| Branche | `feat/US-023-catalogue-filtre-fiche` |

## Récit

En tant que visiteur, je veux filtrer le catalogue par catégorie et ouvrir la fiche d'un article, afin de trouver l'objet de collection qui m'intéresse et d'en connaître tous les détails.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given des articles publiés dans plusieurs catégories, When le visiteur choisit une catégorie, Then seuls les articles publiés de cette catégorie sont listés.
- **AC2** — Given un article publié, When le visiteur ouvre sa fiche, Then il voit photos, titre, description, prix, frais de port, catégorie et la mention « vendeur particulier ».
- **AC3** — Given l'adresse de la fiche d'un article non publié, When un visiteur l'ouvre, Then il voit une page « article introuvable » (404 côté API).

### Non fonctionnels

- **AC4 (Performance)** — Given le scénario de charge d'US-016 étendu à la fiche et au filtre, When le tir s'exécute, Then le p95 reste inférieur à 500 ms.
- **AC5 (Accessibilité)** — Given la fiche et le filtre, When axe-core les analyse, Then aucune violation « serious » ou « critical » ; le filtre est utilisable au clavier.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-31 | Article non publié introuvable | AC3 |
| ES-08 | Contenu affiché comme texte | AC2 |

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Acceptation API | Playwright `request` | AC1, AC3 | Test |
| E2E et accessibilité | Playwright, axe-core | AC2, AC5 | Test |
| Charge | k6 | AC4 | DevOps |

## Hors périmètre

- Note du vendeur, boutique : hors POC (données fictives).

## Décisions liées

- [D01](../docs/decisions/D01-fonctionnalite-poc.md) · [D13](../docs/decisions/D13-strategie-tests-charge.md)

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
