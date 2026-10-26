# US-009 — Créer et modifier un brouillon d'article

| Champ | Valeur |
|---|---|
| Epic | E3 Mise en vente contrôlée |
| Statut | À faire |
| Sprint | 1 (J4) |
| Engagement | Ferme |
| Estimation | 5 points |
| Dépend de | US-008 |
| Fonction(s) | Back, Front, DevOps (amorçage de la base) |
| Service | Place de marché, Front |
| Critères grille | C3, C8, C2 |
| Exigences | EF-12 (sans photo), ENF-03, ENF-19 |
| Branche | `feat/US-009-brouillon-article-vendeur` |

## Récit

En tant que vendeur, je veux créer un article (titre, description, catégorie, prix, frais de port) et le retrouver dans mon espace, afin de préparer sa mise en vente.

## Critères d'acceptation

### Fonctionnels

- **AC1** — Given un vendeur connecté, When il remplit le formulaire d'article et l'enregistre, Then l'article est créé à l'état « brouillon », révision 1, et apparaît dans la liste « Mes articles ».
- **AC2** — Given un brouillon du vendeur, When il le modifie, Then les changements sont enregistrés et la révision augmente.
- **AC3** — Given le formulaire, When le vendeur choisit une catégorie, Then seules les catégories du jeu de données fictif (source unique) sont proposées.
- **AC4** — Given une saisie invalide (prix négatif ou au-delà du plafond, titre au-delà de la longueur maximale, catégorie inexistante), When elle est envoyée à l'API, Then l'API répond 400 au format Problem Details (RFC 9457) et le formulaire affiche l'erreur.
- **AC5** — Given l'API vendeur de la Place de marché, When la CI s'exécute, Then son OpenAPI `/v1` est validée et publiée dans `contracts/api/`.

### Non fonctionnels

- **AC6 (Autorisation, BP2)** — Given l'API vendeur, When elle est appelée sans jeton, avec un jeton expiré, une signature invalide, un autre émetteur ou une autre audience, Then elle répond 401 ; avec un jeton valide de vendeur, elle répond 200.
- **AC7 (Autorisation, BP5)** — Given un brouillon du vendeur A, When le vendeur B le lit ou le modifie, Then la réponse est 404, sans révéler son existence.
- **AC8 (Autorisation)** — Given un acheteur sans rôle vendeur ou un visiteur, When il tente de créer un article, Then la réponse est 403 (acheteur) ou 401 (visiteur).
- **AC9 (Sécurité)** — Given un corps JSON avec une clé dupliquée ou au-delà de la taille maximale, When il est envoyé, Then il est refusé (400 ou 413).
- **AC10 (Sécurité, données)** — Given chaque `up`, When le Job d'amorçage s'exécute, Then la base `marche` a un rôle de migration et un rôle d'exécution ; le rôle d'exécution ne peut pas créer de table ; une connexion sans TLS est refusée ; le Job et le Secret maître sont supprimés après l'amorçage.
- **AC11 (Accessibilité)** — Given le formulaire, When axe-core l'analyse, Then aucune violation « serious » ou « critical » ; chaque champ a un libellé et ses erreurs sont annoncées.

## Sécurité

| Exigence | Comment l'US la sert | AC |
|---|---|---|
| ES-02, ES-03 | Vérification du jeton par `libs/platform`, rôle vendeur exigé | AC6, AC8 |
| ES-04 | Propriété vérifiée sur chaque objet | AC7 |
| ES-08 | Validation stricte des entrées, requêtes paramétrées | AC4, AC9 |
| ES-38 | Compte de base propre au service, sans privilège | AC10 |

Mesures : MA4, MA6, MD1, MD2, MD3. Frontières : API publique, base.

## Tests attendus

| Type | Outil | Ce qui est prouvé | Fonction |
|---|---|---|---|
| Unitaire back | `go test -race` | AC2, AC4, AC9 | Back |
| Intégration | testcontainers-go (PostgreSQL), Keycloak de compose | AC1, AC6, AC10 | Back |
| Acceptation API | Playwright `request` | AC1 à AC4, AC6 à AC9 | Test |
| E2E et accessibilité | Playwright, axe-core | AC1, AC11 | Test |
| Contrôle propre | Validation OpenAPI | AC5 | DevOps |

## Hors périmètre

- Soumission au contrôle : US-013. Photos : US-020. Revalidation d'un article publié : US-021.

## Décisions liées

- [D04 — Stack](../docs/decisions/D04-stack.md) · [D06 — Authentification](../docs/decisions/D06-authentification-autorisation.md) · [D07 — Données](../docs/decisions/D07-donnees.md)

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
