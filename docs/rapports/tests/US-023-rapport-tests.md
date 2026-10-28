# US-023 — Rapport de tests

| Champ | Valeur |
|---|---|
| US | [US-023 — Filtrer le catalogue par catégorie et consulter la fiche d'un article](../../../backlog/US-023-catalogue-filtre-fiche.md) |
| Branche | `feat/US-023-catalogue-filtre-fiche` |
| Date | 2026-10-28 |

## Back — API publique du catalogue

| Champ | Valeur |
|---|---|
| Fonction | Test |
| Périmètre | `services/place-de-marche` : US-023 AC1 (filtre), AC3 (404), partie API d'AC2 (contenu de la fiche) ; US-010 AC2 (seuls les publiés, sans compte, aucune donnée du vendeur) |
| Hors périmètre | Photos et mention « vendeur particulier » d'AC2 (affichage front, E2E), AC4 (charge k6), AC5 (axe-core), front `web/` |
| Outils | `go test`, `httptest`, kin-openapi 0.152.0 (validation des réponses contre `contracts/api/place-de-marche/v1.openapi.yaml`), testcontainers-go 0.44.0 avec PostgreSQL 18.3 |
| Verdict | **Tous les AC du périmètre sont couverts et conformes** |

### Méthode

Scénarios dérivés des AC, étiquetés `US-023-ACn` ou `US-010-AC2`, joués sans jeton sur `GET /marche/v1/articles` et `GET /marche/v1/articles/{id}`. Les articles sont amenés dans chaque état par le parcours réel : création et soumission par l'API du vendeur, puis issue de contrôle (`publie`, `refuse`, `a_verifier`) livrée au consommateur de résultats sous forme de message `controle.resultat` v1 signé et validé. Les catégories utilisées sont lues dans `GET /categories` ; les assertions sont exprimées en écart par rapport à l'état initial, ce qui permet de jouer le même jeu sur une base partagée.

| Fichier | Contenu |
|---|---|
| `cmd/place-de-marche/catalogue_test.go` | 14 scénarios ; banc en mémoire (un banc neuf par scénario) et validation de toutes les réponses du catalogue contre le contrat OpenAPI |
| `cmd/place-de-marche/catalogue_integration_test.go` | Mêmes scénarios sur PostgreSQL réel, rôle `marche_execution`, contrat, index partiels, rejeu des migrations, droits du rôle |
| `cmd/place-de-marche/memoire_test.go` | `ListPublished` et `FindPublished` ajoutés au stockage en mémoire (les tests ne compilaient plus) |
| `cmd/place-de-marche/canal_fictif_test.go`, `integration_test.go` | Extraction de `monterBanc` et `servicePostgreSQL`, sans changement des assertions existantes |

### Résultats d'exécution (depuis `services/place-de-marche`)

| Commande | Résultat |
|---|---|
| `go test ./... -count=1 -cover` | `ok` (≈ 72 s), tous les tests verts, dont US-009, US-013 et l'intégration RabbitMQ existants |
| `go test ./cmd/... -count=1 -coverpkg=./...` | Couverture du service 83,9 % ; `internal/annonces/stockage/catalogue.go` : `ListPublished` 81,2 %, `FindPublished` 87,5 % ; `api/catalogue.go` 85,7 à 100 % |
| `golangci-lint run --config ../../.golangci.yml ./...` (v2.15.0) | 0 issue |
| Conteneurs | Arrêtés et supprimés par testcontainers en fin de test ; aucun conteneur PostgreSQL ou RabbitMQ résiduel |

### US-023 AC1 — Filtre par catégorie

| Scénario | Attendu (AC) | Réel (mémoire et PostgreSQL) | Statut |
|---|---|---|---|
| Nominal : 2 publiés en catégorie A, 1 en B, un article de A dans chaque état non publié | Seuls les publiés de A listés | Filtre A : +2 exactement, tous en catégorie A, ni l'article de B ni les non publiés ; filtre B : +1, sans les articles de A | Conforme |
| Catégorie existante sans article publié (seulement des non publiés) | Liste vide | 200, `total` 0, `elements` vide, page 1 | Conforme |
| Tri : deux articles publiés dans l'ordre inverse de leur création, puis un troisième | Du plus récemment publié au plus ancien | Le dernier publié en tête ; l'ordre suit la publication et non la création, avec et sans filtre | Conforme |
| Pagination : taille 1 sur tout le filtre | Pages cohérentes avec le total | Chaque page : 1 élément, `page` et `taille` renvoyés, `total` constant ; concaténation identique à la liste complète, sans doublon | Conforme |
| Page `total + 1` et page 10000 | Page vide | 200, `elements` vide, `total` inchangé | Conforme |
| Sans paramètre | Page 1, taille 20 | `page` 1, `taille` 20, au plus 20 éléments | Conforme |
| Bornes : `page=1&taille=1`, `page=10000`, `taille=50`, filtre + `taille=50` | Acceptées | 200 | Conforme |
| `categorie=inconnue` | 400 `categorie_inconnue` | 400, `erreurs[{champ: categorie, code: categorie_inconnue}]` | Conforme |
| `categorie=` vide, `categorie` répétée (deux valeurs ou la même deux fois) | 400 `invalide` | 400 `invalide` | Conforme |
| `page=0`, `-1`, `10001`, `taille=0`, `51` | 400 `hors_limites` | 400 `hors_limites` | Conforme |
| `page=`, `taille=`, `page=abc`, `taille=1.5`, `page` ou `taille` répétés | 400 `invalide` | 400 `invalide` | Conforme |
| `categorie=FIGURINES` (hors format) | 400 | 400 | Conforme |

### US-023 AC2 — Fiche (partie API)

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Article publié avec frais de port | Titre, description, prix, frais de port, catégorie | Fiche identique au contenu saisi (id, titre, description multiligne, prix, frais, catégorie id et libellé) | Conforme |
| Article publié sans frais de port | Fiche lisible | `fraisDePort: null` | Conforme |

Photos et mention « vendeur particulier » : aucune donnée correspondante dans le contrat de l'API ; leur affichage relève du front et de l'E2E.

### US-023 AC3 — Article non publié introuvable

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Fiche d'un article brouillon, en contrôle, refusé, à vérifier | 404 | 404 `application/problem+json`, `type`, `title`, `detail` identiques à un article inexistant | Conforme |
| Identifiant inexistant, mal formé, presque UUID | 404 identique | 404 identique | Conforme |
| Témoin : article publié | 200 | 200 | Conforme |

### US-010 AC2 — Seuls les articles publiés, sans compte, aucune donnée du vendeur

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Un article dans chaque état non publié | Aucun n'apparaît | Absents de la liste complète et du filtre ; `total` inchangé | Conforme |
| Lecture directe de chacun | 404 | 404 | Conforme |
| Article refusé puis publié | Absent tant que refusé, visible une fois publié | 404 et absent, puis 200 et présent | Conforme |
| Sans compte | Lecture sans jeton | Toutes les requêtes du catalogue sans en-tête `Authorization` : 200 | Conforme |
| Données du vendeur | Aucune | Ni l'identifiant du vendeur, ni `vendeur`, `seller`, `etat`, `revision`, `motif` dans la liste ou la fiche ; clés limitées à `id, titre, prix, categorie` (entrée) et `+ description, fraisDePort` (fiche) | Conforme |

### Contrat, base de données et migrations

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Toutes les réponses du catalogue (200, 400, 404), mémoire et PostgreSQL | Conformes au contrat OpenAPI v1 | Aucune non-conformité relevée par kin-openapi | Conforme |
| Lectures avec le rôle `marche_execution` | Suffisant pour le catalogue | Tous les scénarios verts sur une connexion en `marche_execution` | Conforme |
| Droits du rôle d'exécution | DML limité à ses tables | DDL, `DELETE`, `TRUNCATE` refusés (42501) ; aucun objet possédé | Conforme |
| Migrations appliquées deux fois au démarrage, puis rejouées sur une base peuplée | Sans erreur ni effet | Aucune erreur ; catalogue identique avant et après le rejeu | Conforme |
| Migration 00004 | Index partiels sur les publiés | `article_catalogue_idx (modifie_le DESC, id DESC)` et `article_catalogue_categorie_idx (categorie_id, modifie_le DESC, id DESC)`, tous deux `WHERE etat = 'publie'` | Conforme |

### Anomalies

Aucune anomalie : aucun comportement observé ne contredit un AC du périmètre Back.

### Observations (sans anomalie)

- Les catégories amorcées en base (migration 00002 : `bandes-dessinees`, `cartes-a-collectionner`, `figurines`…) diffèrent du jeu fictif du banc en mémoire (`cartes-postales`, `figurines`, `timbres`) ; les scénarios lisent donc les catégories par `GET /categories`.
- Le tri « du plus récemment publié » s'appuie sur la date de dernière modification (`modifie_le`) ; c'est équivalent tant qu'un article publié n'est plus modifiable, ce qu'impose US-013.
