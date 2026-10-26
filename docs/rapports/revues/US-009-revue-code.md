# US-009 — Revue de code back

| Champ | Valeur |
|---|---|
| US | [US-009 — Créer et modifier un brouillon d'article](../../../backlog/US-009-brouillon-article-vendeur.md) |
| Branche | `feat/US-009-brouillon-article-vendeur` (base `origin/main`, commits `a1ab8f0`, `b9cb21e`) |
| Date | 2026-10-26 |
| Périmètre | `services/place-de-marche` (service, Dockerfile, migrations), `libs/platform` (`httpjson`, `httpserver`, `logging`, `problem`, `token`, `internal/jsonscan`, refactor de `message`), `contracts/api/place-de-marche/v1.openapi.yaml`, [rapport de tests](../tests/US-009-rapport-tests.md) et fichiers de test non suivis de l'arbre de travail |
| Références | [D04](../../decisions/D04-stack.md), [D06](../../decisions/D06-authentification-autorisation.md), [D07](../../decisions/D07-donnees.md), [D09](../../decisions/D09-observabilite.md), [architecture technique](../../architecture/architecture-technique.md) |
| Verdict | **Fusion impossible en l'état : 1 bloquante, 2 majeures** (4 mineures, 3 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `go vet ./...` (`libs/platform`, `services/place-de-marche`) | 0 problème |
| `golangci-lint run --config ../../.golangci.yml ./...` (v2.15.0, deux modules) | 0 problème |
| `gofmt -l .` (service) | Aucun fichier |
| `go test ./... -count=1` (`libs/platform`) | Vert : `httpjson`, `message` (non-régression du refactor), `token` |
| `go test ./... -count=1` (service) | Vert ; intégration PostgreSQL sautée (Docker absent sur le poste) |
| Recherche de commentaires (`//`, `/* */`, `--`, `#`) dans le Go, le SQL et le Dockerfile | Aucun ; seules les directives `//go:embed` et `-- +goose Up` |
| Recherche de mentions d'outils d'IA dans le diff et les messages de commit | Aucune |
| `git status` | 9 fichiers de test non suivis, `go.mod`/`go.sum` du service modifiés et non commités (voir M2) |

## Points forts

- Architecture simple et dans le bon sens : domaine `annonces` sans dépendance technique hormis `uuid` (`article.go`, `service.go`, `validation.go`) ; `api` et `stockage` dépendent du domaine, jamais l'inverse ; câblage limité à `cmd/place-de-marche/serve.go:18-44`. L'interface `api.ArticleService` est déclarée côté consommateur (`handlers.go:33`).
- `libs/platform` reste strictement technique et réutilisable : décodage JSON strict, Problem Details, serveur HTTP, journaux, jetons. Le refactor `message/duplicates.go` → `internal/jsonscan` garde la sémantique exacte (`jsonscan.ExactKeys`, mêmes détails d'erreur) ; tests de `message` verts ; aucun autre appelant de l'ancien code.
- Jetons (`token/verifier.go`) : RS256 imposé avant analyse et signature unique (`:105-115`), `iss`, `aud`, `exp` et `sub` exigés, `typ` = `Bearer`, `azp` = client attendu (piège Keycloak de D06 couvert), décalage d'horloge borné, jeu de clés en cache avec délai de récupération ; aucune option `Skip*`. Rôle lu dans `realm_access.roles` (D06 l. 147), refus par défaut.
- Propriété (BP5) : toutes les requêtes filtrent sur `vendeur_id` = `sub` (`postgres.go:33`, `:42`, `:47`, `:53`) ; un article d'un autre vendeur et un identifiant invalide répondent 404 (`handlers.go:101-104`, `:128-132`). Aucun identifiant de vendeur lu dans le corps.
- Révision optimiste conforme à D07 : `UPDATE … WHERE id, vendeur_id, revision, etat` (`postgres.go:28-37`), puis distinction 404 / 409 périmée / 409 non modifiable (`:95-111`).
- Requêtes entièrement paramétrées ; pagination bornée (`service.go:8-12`, `handlers.go:193-209`).
- Entrées : `Content-Type` exigé, 16 Kio, UTF-8 valide, objet racine, clés dupliquées (casse ignorée) et inconnues refusées avant décodage, puis décodage strict (`httpjson/decode.go:65-77`) ; contrôles de domaine doublés de contraintes `CHECK` en base (`00001_…sql`).
- Aucune fuite : `problem.Internal()` sans détail, erreur de configuration de base masquée (`pool.go:14`, `migrations.go:21`), masquage des jetons et mots de passe de chaîne de connexion dans les attributs, messages et erreurs (`logging/mask.go`).
- Serveur durci : délais de lecture, d'écriture, d'inactivité et taille d'en-têtes (`httpserver/server.go:12-31`), arrêt propre, reprise sur panique, en-têtes `no-store`, CSP, `nosniff`, `no-referrer` ; port de gestion séparé.
- TLS base exigé par défaut (`sslmode=verify-full`, `config.go:91-100`), y compris pour `migrate`.
- Dockerfile : images épinglées par empreinte, `distroless/static:nonroot`, `USER 65532`, `CGO_ENABLED=0`, `go mod verify`, `-trimpath`.
- Contrat OpenAPI cohérent avec le code (routes, codes, `erreurs[].champ/code`, types d'URN, 16 Kio) et avec le front (`elements`, `page`, `taille`, `total`, `prix.montantCentimes/devise`, `categorie.id/libelle`, identifiants des catégories de démonstration identiques à `00002_amorcer_categories.sql`).

## Issues

### Bloquante

**B1 — L'intégration PostgreSQL échouera en CI : le rôle `marche_execution` n'existe pas dans le conteneur de test.**
- Où : `services/place-de-marche/internal/migrations/00001_creer_categories_et_articles.sql:26-27` (`GRANT … TO marche_execution`) et `services/place-de-marche/cmd/place-de-marche/integration_test.go:26-31` (conteneur créé avec le seul utilisateur `marche_test`).
- Pourquoi : PostgreSQL refuse `GRANT … TO` un rôle inexistant ; goose annule la migration, `migrations.Apply` échoue et `TestUS009IntegrationPostgreSQL` s'arrête sur `t.Fatalf`. Le poste n'a pas Docker (test sauté), mais le runner `ubuntu-24.04` de la CI en a un : le job Go du service sera rouge et la preuve sur PostgreSQL réel annoncée par le rapport de tests n'existe pas.
- Correction : dans le harnais, créer avant `Apply` les rôles de l'amorçage (`marche_execution` au minimum, idéalement un rôle de migration propriétaire de la base), puis connecter le dépôt avec `marche_execution` pour éprouver les droits réels (SELECT/INSERT/UPDATE suffisants, pas de CREATE). Lancer une fois le test avec Docker avant de livrer.
- Dev : **testeur** (harnais) ; **devops** : l'amorçage (AC10) doit créer exactement le rôle `marche_execution`.

### Majeures

**M1 — Masquage des journaux sans test unitaire, exigé par D09 (MO3).**
- Où : `libs/platform/logging/mask.go` (paquet `logging` : « no test files »).
- Pourquoi : D09 l. 113 et MO3 exigent un test unitaire à valeurs piégées (attribut, message, erreur `*url.Error`, groupe imbriqué). C'est le seul rempart contre la fuite de jetons et de mots de passe ; une régression des expressions `signedTokenPattern` / `connectionPasswordFormat` passerait inaperçue.
- Correction : test boîte noire via `logging.New` sur un `bytes.Buffer` : jeton `eyJ…` dans un attribut, dans le message, dans une erreur ; DSN `postgres://u:motdepasse@h/db` dans un `*url.Error` ; clés `Authorization`, `jeton`, `secret` ; attribut dans un groupe ; vérifier qu'aucune valeur piégée n'apparaît et que `correlation_id` est bien ajouté.
- Dev : **testeur**.

**M2 — Les tests ne sont pas dans la branche.**
- Où : `git status` : `libs/platform/httpjson/decode_test.go`, `libs/platform/token/verifier_test.go`, 7 fichiers `services/place-de-marche/cmd/place-de-marche/*_test.go` non suivis ; `services/place-de-marche/go.mod`/`go.sum` (testcontainers, kin-openapi) modifiés non commités ; rapport de tests non suivi.
- Pourquoi : `HEAD` livre ~2 900 lignes sans aucun test ; la DoD (« AC couverts par des tests verts en CI ») ne peut pas être évaluée.
- Correction : commiter les tests, le rapport et les `go.mod`/`go.sum` du service (après B1), puis relancer la CI.
- Dev : **devops** (commit), **testeur** (contenu).

### Mineures

**m1 — Droits du rôle d'exécution posés autrement que D07, nom de rôle non documenté.**
- Où : `00001_creer_categories_et_articles.sql:26-27`.
- Pourquoi : D07 l. 252 et l. 313 prévoient `ALTER DEFAULT PRIVILEGES` (tables et séquences) dans la première migration. Le `GRANT` par table est plus restrictif (pas de `DELETE`), donc défendable, mais l'écart n'est pas tracé et chaque future table devra penser à son `GRANT`. Le nom `marche_execution` n'apparaît que dans ce fichier.
- Correction : soit aligner sur D07, soit consigner le choix (GRANT explicite par table) dans D07 ; documenter le nom du rôle pour l'amorçage.
- Dev : **back**.

**m2 — Certificat d'autorité RDS absent de l'image.**
- Où : `services/place-de-marche/Dockerfile:17-18`.
- Pourquoi : D07 l. 266 prévoit le paquet d'autorités RDS embarqué dans l'image ; `distroless/static` ne contient que les autorités publiques, donc `verify-full` échouera contre RDS sans `sslrootcert`. Sans effet en local.
- Correction : embarquer `global-bundle.pem` et le référencer par `sslrootcert`, ou acter un montage par Secret/ConfigMap dans D07. À traiter avant le déploiement sur EKS.
- Dev : **devops**.

**m3 — Une description conforme au contrat peut être refusée en 413.**
- Où : `internal/annonces/api/handlers.go:22` (`maxBodyBytes = 16 << 10`) et `internal/annonces/validation.go:12` (`MaxDescriptionLength = 5000` caractères).
- Pourquoi : 5 000 caractères de 4 octets (emoji) ou de 3 octets (`€`, CJK) dépassent 16 384 octets ; le front recevra 413 au lieu d'un enregistrement ou d'un 400 rattaché au champ.
- Correction : porter la limite à 32 Kio (couvre 5 000 × 4 octets + titre + structure), ou documenter la limite en octets dans le contrat et la faire respecter côté formulaire.
- Dev : **back**.

**m4 — Variables d'environnement du service non documentées.**
- Où : `internal/config/config.go:14-21`, `cmd/place-de-marche/main.go:19`.
- Pourquoi : `ADRESSE_API`, `ADRESSE_GESTION`, `BASE_DE_DONNEES_URL`, `BASE_DE_DONNEES_TLS_EXIGE`, `OIDC_*`, `JOURNAL_NIVEAU`, sous-commandes `serve`/`migrate` et ports 8080/9090 ne figurent dans aucune doc ; le chart et l'amorçage devront les deviner.
- Correction : `services/place-de-marche/README.md` court (variables, valeurs par défaut, sous-commandes, ports, rôles de base attendus).
- Dev : **back**.

### Suggestions

- **S1** — `stockage/postgres.go:126-129` : le total et la page sont lus par deux requêtes hors transaction ; `count(*) OVER ()` dans `listOwnedQuery` (ou une transaction `REPEATABLE READ`) rend la page et son total cohérents en une seule lecture. Dev : back.
- **S2** — `libs/platform/httpjson/decode.go:140` : détection du champ inconnu par préfixe du message d'erreur de `encoding/json`, fragile entre versions de Go ; quand la cible est une structure, `jsonscan` a déjà refusé la clé, la branche ne sert qu'aux cibles avec `map`/`interface`. Ajouter un test de ce cas ou le documenter par le nom d'une fonction dédiée. Dev : back.
- **S3** — `token/middleware.go:34-37` : l'échec de vérification est journalisé en `Info` sans distinction de cause ; un champ `motif` fixe (expiré, émetteur, audience, signature) faciliterait la détection (D09) sans exposer le jeton. Dev : back.

## Hors périmètre de ce lot (rappel, non bloquant pour cette revue)

- AC5 : aucune étape de validation et publication de l'OpenAPI dans `.github/workflows/ci.yml` (DevOps).
- AC10 : Job d'amorçage, rôles, refus sans TLS, suppression du Secret maître (DevOps).
- AC4 (affichage dans le formulaire) et AC11 : front non livré dans cette branche.

## Corrections par dev

| Dev | Issues |
|---|---|
| Testeur | B1, M1, M2 (contenu) |
| DevOps | B1 (nom du rôle à l'amorçage), M2 (commit), m2 |
| Back | m1, m3, m4, S1 à S3 |
