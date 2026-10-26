# US-009 — Rapport de tests back

| Champ | Valeur |
|---|---|
| US | [US-009 — Créer et modifier un brouillon d'article](../../../backlog/US-009-brouillon-article-vendeur.md) |
| Branche | `feat/US-009-brouillon-article-vendeur` |
| Date | 2026-10-26 |
| Fonction | Test |
| Périmètre de ce lot | Back (`services/place-de-marche`, `libs/platform`) : AC1, AC2, AC3, AC4 (partie API), AC6, AC7, AC8, AC9 |
| Hors périmètre de ce lot | AC5 (validation et publication de l'OpenAPI en CI, DevOps), AC10 (Job d'amorçage, DevOps), AC11 (accessibilité, Front), affichage de l'erreur dans le formulaire (AC4, Front) |
| Outils | `go test` (Go 1.26.10), `httptest`, testcontainers-go v0.44.0 et module `postgres` v0.44.0 (image `postgres:18.3` épinglée par empreinte), kin-openapi v0.152.0, golangci-lint v2.15.0 |
| Verdict | **Tous les AC du périmètre sont couverts et conformes sur le stockage en mémoire ; la preuve sur PostgreSQL réel reste à obtenir en CI (Docker indisponible sur le poste)** |

## Méthode

Scénarios Given/When/Then dérivés des AC de l'US, du contrat `contracts/api/place-de-marche/v1.openapi.yaml` et de D06/D13, nommés `US-009-ACn`, écrits sans lecture de la logique de production. Seules les signatures publiques (interfaces du dépôt, `publicHandler`, `token.Config`) ont servi au branchement.

- **Acceptation HTTP sans Docker** : serveur `httptest` branché sur le vrai routeur du service (`publicHandler` : journalisation, reprise, en-têtes de sécurité, routes `/marche/v1`), vrai service `annonces`, vrai vérificateur `libs/platform/token`. Le dépôt est remplacé par un faux en mémoire respectant `annonces.Articles` et `annonces.Categories`. Jetons RS256 forgés en test, clé publique servie par un faux JWKS `httptest`.
- **Intégration PostgreSQL** : mêmes scénarios métier rejoués sur un PostgreSQL 18.3 réel (testcontainers), migrations goose appliquées deux fois (idempotence), vrai dépôt pgx. Saut explicite si Docker est absent (`SkipIfProviderIsNotHealthy`) ou en `-short`.
- **Contrat** : chaque échange HTTP enregistré est validé par kin-openapi (route, statut documenté, en-têtes, schéma du corps), en mémoire et sur PostgreSQL.
- **Unitaires de `libs/platform`** : vérificateur de jetons et décodeur JSON testés en boîte noire par leur API publique.

| Fichier | Contenu |
|---|---|
| `services/place-de-marche/cmd/place-de-marche/jetons_test.go` | Clés RSA, faux JWKS, fabrique de jetons (RS256, HS256, `none`, revendications modifiées) |
| `services/place-de-marche/cmd/place-de-marche/memoire_test.go` | Faux dépôt en mémoire et jeu de catégories fictif |
| `services/place-de-marche/cmd/place-de-marche/environnement_test.go` | Démarrage du serveur, client HTTP, lecture des réponses et des Problem Details |
| `services/place-de-marche/cmd/place-de-marche/scenarios_test.go` | Scénarios métier partagés mémoire / PostgreSQL (AC1 à AC4, AC7) |
| `services/place-de-marche/cmd/place-de-marche/acceptation_test.go` | AC3 (jeu fictif exact), AC6, AC8, AC9 |
| `services/place-de-marche/cmd/place-de-marche/contrat_test.go` | Conformité des réponses à l'OpenAPI `/v1` |
| `services/place-de-marche/cmd/place-de-marche/integration_test.go` | PostgreSQL réel, migrations, scénarios métier, contrat |
| `libs/platform/token/verifier_test.go` | Jeton valide, refus (exp, iss, aud, azp, typ, kid, clé, `none`), 401/403 du middleware |
| `libs/platform/httpjson/decode_test.go` | 415, 413, limite exacte, JSON mal formé, clés dupliquées, racine non objet, champ inconnu, type invalide |

## Résultats d'exécution

| Commande | Résultat |
|---|---|
| `go test ./... -count=1 -cover` (`services/place-de-marche`) | Vert : 151 cas réussis, 1 saut (`TestUS009IntegrationPostgreSQL`, Docker absent) |
| Couverture du service mesurée avec `-coverpkg=./...` | `internal/annonces` 94,0 %, `internal/annonces/api` 79,6 %, `stockage`, `migrations`, `config` 0 % sans Docker ; total 48,1 % |
| `go test ./... -count=1 -cover` (`libs/platform`) | Vert : `token` 85,9 %, `httpjson` 76,7 %, `message` 82,8 % (inchangé) |
| `golangci-lint run --config ../../.golangci.yml ./...` (les deux modules) | 0 problème |
| `go vet ./...` (les deux modules) | 0 problème |
| `go test -race` | Non exécutable sur le poste (cgo absent sous Windows) ; exécuté par la CI Linux |

## AC1 — Brouillon créé en révision 1 et listé

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Nominal : titre, description, catégorie, prix, frais de port | 201, état `brouillon`, révision 1, présent dans « Mes articles » | 201, `brouillon`, révision 1, contenu renvoyé à l'identique, `Location` vers le brouillon, relecture identique, liste de total 1 | Conforme |
| Limite : corps `{}` (complétude jugée à la soumission, contrat) | 201 brouillon vide | 201, titre vide, prix et catégorie `null` | Conforme |
| Limites acceptées : titre de 200 caractères (ASCII et accentués), description de 5000, prix 0 et 100 000 000, frais 0 et 100 000 | 201 | 201 pour chaque borne | Conforme |
| Liste paginée (`page`, `taille`), plus récent en tête | Éléments et total cohérents | 2 + 1 éléments, total 3, dernier modifié en tête | Conforme |

## AC2 — Modification et révision

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Nominal : modification en révision 1, puis 2 | Changements enregistrés, révision 2 puis 3 | 200, révision 2 puis 3, relecture conforme | Conforme |
| Remplacement complet : champs absents | Champs vidés (contrat) | Description vide, catégorie, prix, frais `null` | Conforme |
| Erreur : révision périmée (1 après passage en 2) ou future (7) | 409 Problem Details, article inchangé | 409 `application/problem+json`, article resté en révision 2 | Conforme |
| Concurrence : 8 modifications simultanées en révision 1 | Une seule gagnante | 1 × 200, 7 × 409, révision finale 2 | Conforme (mémoire) |
| Erreur : révision absente ou 0 | 400 | 400 Problem Details | Conforme |
| Erreur : saisie invalide en modification | 400, article inchangé | 400 `prix`/`negatif`, révision 1 conservée | Conforme |
| Erreur : article inexistant | 404 | 404 | Conforme |

## AC3 — Catégories du jeu de données fictif

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| `GET /categories` sans jeton et avec un jeton d'acheteur | 200, liste publique | 200 dans les deux cas | Conforme |
| Liste proposée = jeu de données fictif exact | Ni plus ni moins, triée par libellé | Identique au jeu injecté, même ordre | Conforme |
| Chaque catégorie proposée est acceptée à la création | 201, libellé renvoyé | 201, `{id, libelle}` renvoyé pour chacune | Conforme |
| Catégorie hors jeu | 400 `categorie_inconnue` | 400, erreur `categorieId`/`categorie_inconnue` | Conforme |

## AC4 — Saisie invalide refusée en 400 Problem Details (partie API)

Chaque cas : statut 400, `Content-Type: application/problem+json`, `type`, `title`, `status` renseignés, erreur rattachée au champ, aucun article créé.

| Saisie | Erreur observée | Statut |
|---|---|---|
| Prix −1 | `prix…`/`negatif` | Conforme |
| Prix 100 000 001 (au-delà du plafond) | `prix…`/`au_dela_du_plafond` | Conforme |
| Titre de 201 caractères (ASCII et accentués) | `titre`/`trop_long` | Conforme |
| Catégorie inexistante | `categorieId`/`categorie_inconnue` | Conforme |
| Catégorie mal formée (`Pas Valide`) | erreur sur `categorieId` | Conforme |
| Frais de port −5 et 100 001 | `fraisDePort…`/`negatif`, `au_dela_du_plafond` | Conforme |
| Devise `USD` | `prix…`/`devise_non_prise_en_charge` | Conforme |
| Description de 5001 caractères | `description`/`trop_long` | Conforme |
| Titre avec caractère de contrôle | `titre`/`caractere_interdit` | Conforme |
| Titre trop long et prix négatif ensemble | Les deux erreurs dans `erreurs` | Conforme |

L'affichage de l'erreur dans le formulaire relève du lot front.

## AC6 — Authentification (401) et jeton valide (200)

Matrice de 16 variantes × 4 routes vendeur (lister, créer, lire, modifier) = 64 cas : 401, Problem Details et en-tête `WWW-Authenticate` attendus.

| Variante | Réel | Statut |
|---|---|---|
| Sans jeton, schéma `Basic`, `Bearer` vide, jeton mal formé | 401 | Conforme |
| Jeton expiré, jeton sans `exp` | 401 | Conforme |
| Signature d'une autre clé (même `kid`), signature altérée, `kid` inconnu | 401 | Conforme |
| Algorithme `none`, algorithme `HS256` | 401 | Conforme |
| Autre émetteur, autre audience | 401 | Conforme |
| Autre client autorisé (`azp`), `typ` `Refresh` ou `ID` | 401 | Conforme |
| Jeton valide de vendeur (audience en tableau ou en chaîne) | 200 en lecture et liste, 201 en création, 200 en modification | Conforme |

Les mêmes refus sont prouvés au niveau de `libs/platform/token` (`Verify` renvoie `ErrInvalidToken`, middleware 401/403).

## AC7 — Article d'un autre vendeur

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Vendeur B lit le brouillon de A | 404 | 404 | Conforme |
| Vendeur B modifie le brouillon de A | 404, article inchangé | 404, révision 1 et titre conservés | Conforme |
| Sans révéler l'existence | Réponse identique à celle d'un identifiant inexistant | Problem Details strictement identiques (lecture et modification) | Conforme |
| Liste de B | Sans les articles de A | Total 1 pour B, 2 pour A | Conforme |

## AC8 — Acheteur et visiteur

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Acheteur sur les 4 routes vendeur | 403 | 403 Problem Details | Conforme |
| Jeton sans rôle, rôle `vendeur` de client hors realm, rôles voisins (`vendeurs`, `Vendeur`) | 403 | 403 | Conforme |
| Aucun article créé par ces tentatives | Liste vide | Total 0 | Conforme |
| Visiteur crée un article | 401 | 401 | Conforme |

## AC9 — Corps JSON refusés

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Clé dupliquée (racine, imbriquée, échappée `titre`, à la casse près) en création | 400 | 400 Problem Details | Conforme |
| Clé dupliquée en modification (`revision`, `titre`/`TITRE`) | 400, article inchangé | 400, révision 1 conservée | Conforme |
| Champ inconnu (racine, imbriqué), type inattendu, racine tableau, JSON mal formé, corps vide | 400 | 400 | Conforme |
| Corps d'exactement 16 Kio | Accepté | 201 | Conforme |
| 16 Kio + 1 octet, 1 Mio, modification au-delà de 16 Kio | 413 | 413 Problem Details, article inchangé | Conforme |
| `Content-Type` absent, `text/plain`, formulaire, `application/jsonp` | 415 | 415 | Conforme |
| `application/json; charset=utf-8` | Accepté | 201 | Conforme |

## Conformité au contrat OpenAPI

Toutes les réponses d'un parcours couvrant 200, 201, 400 (saisie, clé dupliquée, `taille=51`), 401, 403, 404, 409, 413 et 415 sont conformes au contrat `/v1` (statuts documentés, schémas `Article`, `PageArticles`, `ListeCategories`, `Probleme`). Le contrat lui-même est valide pour kin-openapi.

## Anomalies

Aucune anomalie : aucun comportement observé ne contredit un AC du périmètre.

Observation sans impact sur un AC : au niveau de `libs/platform/httpjson`, `{"titre":"a","TITRE":"b"}` est rejeté avec la raison `champ_inconnu` et non `cle_dupliquee` ; le corps est bien refusé en 400, comme l'exige AC9.

## Reste à prouver (Docker requis)

| Point | Moyen | Où |
|---|---|---|
| AC1, AC2, AC3, AC4, AC7 sur le vrai dépôt pgx et les migrations goose (dont la concurrence AC2 et le cloisonnement AC7 en SQL) | `TestUS009IntegrationPostgreSQL`, écrit et compilé, sauté sur le poste | CI Linux (`go test -race ./...`) |
| Idempotence des migrations (deux applications successives) | Même test | CI |
| Conformité au contrat des réponses issues de PostgreSQL | Même test | CI |
| AC6 contre le Keycloak de compose | Acceptation Playwright `request` | Lot acceptation |
| AC5, AC10, AC11 | Hors lot | DevOps, Front |

Le test d'intégration tourne sans changement du workflow : la CI exécute `go test -race ./...` sur `ubuntu-24.04`, où Docker est disponible. Il n'a pas encore été exécuté : un échec à la première exécution en CI pourrait venir du banc de test lui-même, pas forcément du service.

Recommandation pour le workflow (non modifié) : ajouter `-coverpkg=./...` à l'étape « Tests unitaires » ; sans cette option, la couverture des paquets `internal/` exercés par les tests d'acceptation de `cmd/` est publiée à 0 %.

## Dépendances ajoutées

`services/place-de-marche/go.mod` : `github.com/getkin/kin-openapi v0.152.0`, `github.com/testcontainers/testcontainers-go v0.44.0`, `github.com/testcontainers/testcontainers-go/modules/postgres v0.44.0` (tests uniquement) ; aucune dépendance existante n'a changé de version.
