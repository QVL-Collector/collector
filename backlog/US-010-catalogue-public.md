# US-010 — Consulter le catalogue public sans compte

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | À faire |
| Sprint | 1 (J4) |
| Engagement | Ferme |
| Estimation | 3 points |
| Dépend de | US-009 |
| Fonction(s) | Back, Front, DevOps (jeu de démonstration) |
| Service | Place de marché, Front |
| Critères grille | C3, C5 |
| Exigences | EF-01, EF-16, ENF-03 |
| Branche | `feat/US-010-catalogue-public` |

## Récit

En tant que visiteur, je veux parcourir les articles publiés sans créer de compte, afin de découvrir les objets en vente sur Collector.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given des articles publiés, When un visiteur non connecté ouvre le catalogue, Then il voit la liste des articles publiés (titre, prix, catégorie), paginée.
- **AC2** — Given des articles en brouillon, en contrôle, refusés ou à vérifier, When le catalogue est affiché ou l'API publique interrogée, Then aucun d'eux n'apparaît, et la lecture directe de l'un d'eux par l'API publique répond 404.
- **AC3** — Given chaque `up`, When le script d'amorçage du jeu de démonstration s'exécute deux fois de suite, Then le catalogue contient le même jeu d'articles publiés fictifs, sans doublon.

### Non fonctionnels

- **AC4 (Sécurité, sorties)** — Given un article publié dont le titre contient du balisage (`<script>`, `<img onerror>`), When il est affiché au catalogue, Then il s'affiche comme du texte et la page ne déclenche aucune violation de la politique de sécurité du contenu.
- **AC5 (Performance)** — Given le catalogue amorcé, When l'API publique de liste est appelée à vide, Then elle répond en moins de 500 ms ; la mesure sous charge relève d'US-016.
- **AC6 (Accessibilité)** — Given la page du catalogue, When axe-core l'analyse, Then aucune violation « serious » ou « critical ».

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-31 | Seuls les articles publiés sont visibles | AC2 |
| ES-08 | Contenu neutralisé à l'affichage | AC4 |
| ES-04 | Article non publié invisible pour autrui | AC2 |

Mesures : MD8 (CSP), ME2.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Intégration | testcontainers-go | AC2, AC3 | Back |
| Acceptation API | Playwright `request` | AC1, AC2, AC5 | Test |
| E2E, CSP et accessibilité | Playwright, axe-core | AC1, AC4, AC6 | Test |

## Hors périmètre

- Filtre par catégorie et fiche article : US-023.
- Mention « réservé » : US-030.

## Décisions liées

- [D01 — Fonctionnalité du POC](../docs/decisions/D01-fonctionnalite-poc.md) · [D07 — Données](../docs/decisions/D07-donnees.md)

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
