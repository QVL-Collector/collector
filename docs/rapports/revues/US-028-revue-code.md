# US-028 — Revue de code back et front

| Champ | Valeur |
|---|---|
| US | [US-028 — Être notifié de la décision de publication de son article](../../../backlog/US-028-notification-vendeur.md) |
| Branche | `feat/US-028-notification-vendeur` (base `origin/main`, commits `e7a4a54`, `ae1f073`, `d97690b`) |
| Date | 2026-11-06 |
| Périmètre | Nouveau service `services/interactions` (config, migrations, domaine, canal, API, stockage, Dockerfile, README), émission par la Place de marché (`stockage/postgres.go`, `annonces/publication.go`, `annonces/diffusion`, `controle/requests.go`, `serve.go`), `libs/platform` (`logging.CorrelationIDOrNew`, `outbox.NewEvent`, tests `message`), contrats (`publication.decidee` v1 et exemples, `interactions/v1.openapi.yaml`, `contracts/README.md`), front `web/src/features/notifications`, `app/`, `config/`, `i18n/`, tests Go, Vitest et Playwright, [rapport de tests](../tests/US-028-rapport-tests.md) |
| Références | [D02](../../decisions/D02-decoupage-applicatif.md) (P2), [D05](../../decisions/D05-communication-interservice.md), [D06](../../decisions/D06-authentification-autorisation.md), [D07](../../decisions/D07-donnees.md) |
| Hors revue (connu) | Chart Helm, route ALB et Job d'amorçage de la base d'Interactions (DevOps, plus tard) ; en-tête mobile serré avec le lien Notifications |
| Verdict | **Fusion possible** (0 bloquante, 0 majeure, 4 mineures, 4 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l .` et `go vet ./...` (`services/interactions`) | 0 fichier, 0 erreur |
| `go test -count=1 ./internal/...` (`services/interactions`) | Vert |
| `gofmt -l .`, `go vet ./...`, `go build ./...` (`services/place-de-marche`) | 0 fichier, 0 erreur |
| `gofmt -l .`, `go vet ./...`, `go test -count=1 ./message/ ./outbox/ ./logging/` (`libs/platform`) | Vert |
| `npm run typecheck` (`web/`) | 0 erreur |
| `npx eslint` sur `features/notifications`, `app`, `config`, `i18n`, `e2e` | 0 erreur, 0 avertissement |
| `npx prettier --check` sur les mêmes dossiers et `acceptance/US-028` | Conforme |
| `npm run i18n:check` | 252 clés alignées fr, en |
| `npx vitest run src/acceptance/US-028` | 5 fichiers, 85 tests verts |
| Recherche de commentaires dans les lignes ajoutées (`//`, `/*`, `--`, `#`) | Aucun (hors `go:embed` et `+goose`) |
| Recherche des appelants de `stockage.NewStore` (Place de marché) | `serve.go:59`, `integration_test.go:215` (`depotPostgreSQL`, repris par `comptes_migration_test.go`, `filet_de_relance_test.go`) : tous adaptés |
| Recherche d'autres constructions manuelles d'`outbox.Event` / de `correlationOf` | Aucune : les deux encodeurs passent par `outbox.NewEvent` et `logging.CorrelationIDOrNew` |
| Messages de commit | Format Karma, aucune mention d'outil d'IA |

Les bancs testcontainers (PostgreSQL, RabbitMQ, binaires réels), `go test -race`, `golangci-lint` et Playwright sont tracés verts dans le rapport de tests ; non rejoués ici.

## Points forts

- Atomicité de l'émission (AC1) : la décision, sa trace (`resultat_controle` ou `decision_admin`) et la ligne d'outbox sont écrites dans la même transaction (`stockage/postgres.go:301-317` pour le contrôle, `:483-494` pour l'administrateur) ; un échec d'encodage ou d'insertion annule tout, prouvé par `ac1DecisionAdminAtomique` et `ac1ResultatControleAtomique`.
- Garde d'idempotence côté Place de marché conservée : `UPDATE … RETURNING vendeur_id, modifie_le` (`postgres.go:85-89`) ; zéro ligne donne `pgx.ErrNoRows`, sans trace ni événement (`postgres.go:303-305`). `decided_at` vient de la base (`modifie_le`), identique pour les deux chemins (`publication.go:15-33`).
- Idempotence côté Interactions (AC3) : `INSERT … ON CONFLICT (message_id) DO NOTHING` (`interactions/.../stockage/postgres.go:15-18`) sur l'id de l'enveloppe, conservé au rejeu de l'outbox. Choix juste : une unicité sur (article, révision) serait fausse, car `a_verifier` puis la décision de l'administrateur portent sur la même révision (`decideQuery` filtre `revision = $2`).
- Validation stricte à trois niveaux : schéma v1 fermé (`additionalProperties: false`, `if/then/else` motif selon issue, révision bornée à int32), domaine (`notification.go:120-128`) et contraintes SQL (`00001_creer_les_notifications.sql:9-17`). Contrat inattendu sur la file écarté (`canal/decisions.go:38-40`) ; erreur de base retentée sans rejet (`decisions.go:46-48`), même schéma que `controle/results.go:47`.
- Autorisation (AC6) : liste filtrée par le sujet du jeton uniquement (`api/handlers.go:68-80`), aucun identifiant de vendeur accepté en paramètre ; rôle `vendeur` exigé ; sujet non UUID rejeté en 401.
- Pagination bornée (`page` ≤ 10 000, `taille` ≤ 50), paramètre répété ou vide refusé, décompte et page lus dans un même instantané `READ ONLY REPEATABLE READ` (`stockage/postgres.go:50-68`), index `(vendeur_id, decide_le DESC, id DESC)` aligné sur l'`ORDER BY`.
- Droits minimaux : `interactions_execution` n'a que `SELECT, INSERT`, `UPDATE/DELETE/TRUNCATE` révoqués ; bases cloisonnées prouvées (AC8).
- Factorisation utile au passage : `logging.CorrelationIDOrNew` et `outbox.NewEvent` remplacent le code recopié dans `controle/requests.go` ; tests d'outbox existants adaptés sans affaiblissement (filtre par `type_message`, `demandesDansLaFenetre` pour MC9).
- Contrats et documentation cohérents avec le code : OpenAPI (codes `invalide`/`hors_limites`, 400/401/403/500), schéma, `contracts/README.md`, README d'Interactions (préchargement 10, file `.ecartes`, `delivery-limit` 4 de `infra/canal`), README de la Place de marché et de `web/`.
- Front : composants purement JSX, logique dans les hooks (`useNotificationsPage`, `useNotificationCard`, `toNotificationsView`) ; union discriminée exhaustive sur `view.status` ; texte construit à partir de l'issue et du code de motif uniquement (`useNotificationCard.ts:27-33`), aucun titre d'article ; i18n fr/en alignée ; `<time dateTime>`, `ul/li`, section `aria-labelledby`, annonce `role="status"` ; relecture 5 s suspendue onglet masqué et arrêtée sur 401/403 (`notificationsPolling.ts`).

## Issues

### Bloquantes

Aucune.

### Majeures

Aucune.

### Mineures

1. `services/interactions/internal/config/config.go:52-136`, `services/interactions/internal/migrations/migrations.go`, `services/interactions/internal/notifications/stockage/pool.go` — copie de la Place de marché : `migrations.go` et `pool.go` sont identiques à l'octet ; `LoadDatabase`, `checkDatabaseURL`, `boolOrDefault`, `loadToken`, `valueOrDefault` le sont au nom de constante près (`diff` : seuls l'audience par défaut, l'émetteur attendu et la limite de débit diffèrent). Le contrôle de sécurité `sslmode=verify-full` (`checkDatabaseURL`) n'a de test unitaire dans aucun des deux services. **Réponse à la question du dev back : oui, factoriser maintenant**, dans un commit `refactor(platform)` séparé, avant que Transactions ne crée une troisième copie : `libs/platform/database` (`LoadConfig(lookup)`, `Connect`, `ApplyMigrations(ctx, logger, url, fs.FS)`) et `token.LoadConfig(lookup, defaultAudience)` ; ne garder par service que les adresses, l'émetteur attendu et la limite de débit ; ajouter à cette occasion les tests du refus TLS et de l'URL non `postgres://`. Ne bloque pas la fusion de l'US. Dev : back.
2. `web/src/features/notifications/api/notificationPayload.ts:7` — `value ?? null` accepte un élément sans `motif`, alors que l'OpenAPI le déclare obligatoire (`required: [id, articleId, revision, issue, motif, decideLe]`) ; la cohérence issue/motif (`publie` sans motif, sinon motif présent) n'est pas vérifiée non plus. Écart avec la lecture stricte des autres champs. Remplacer par `parseNullable(value, …)` (absence rejetée) et rejeter un motif incohérent avec l'issue ; ajouter les deux cas à `contrat-payload-notifications.test.ts`. Dev : front.
3. `web/src/app/createAppServices.ts:22-23` et `:33` — le défaut `/interactions` est appliqué deux fois : dans `parseRuntimeConfig` (`config/runtimeConfig.ts:49-52`) et ici par `Partial<…>` + `?? defaultInteractionsApiUrl`. Le `Partial` n'existe que pour les trois harnais qui appellent `createAppServices` sans la clé (`acceptance/US-010/AC2-api-publique.test.tsx:54`, `US-010/AC4-titres-neutralises.test.tsx:63`, `US-023/publicMarketplaceHarness.tsx:146`). Rendre la clé obligatoire dans `AppServicesConfig`, supprimer le repli et passer `interactionsApiUrl` dans ces trois appels. Dev : front.
4. `docs/decisions/D05-communication-interservice.md:72`, `:297`, `:351` — **réponse : oui, aligner D05.** Remplacer `ON CONFLICT (decision_id) DO NOTHING, dans la même transaction que la notification` par « `INSERT` dans `notification` avec `ON CONFLICT (message_id) DO NOTHING` : l'`id` de l'enveloppe identifie la décision, conservé au rejeu de l'outbox » ; préciser aux lignes 72 et 351 que l'« identifiant de la décision » est cet `id` et non (article, révision), qui n'est pas unique (`a_verifier` puis décision de l'administrateur sur la même révision). Un seul `INSERT` rend la mention « même transaction » sans objet. Dev : back (ou architecte, propriétaire de D05).

### Suggestions

1. `web/src/features/notifications/hooks/notificationsView.ts:12-14`, `:29` — le tri client par `decidedAt` reprend l'`ORDER BY decide_le DESC, id DESC` du serveur ; le README `web/` et le test AC2 parlent de « liste dans l'ordre reçu ». Retirer `byMostRecent` (YAGNI), la vue devient `{ status: 'ready', notifications: query.data }`. Dev : front.
2. `web/src/features/notifications/api/notificationPayload.ts:10` (`parseNotification`) et `api/demoNotifications.ts:4` (`DemoNotification`) — exportés mais utilisés seulement dans leur fichier ; retirer `export`. Dev : front.
3. `services/interactions/internal/notifications/api/handlers.go:82-104` et `services/place-de-marche/internal/annonces/api/handlers.go:304-334` — `pageRequestOf`/`boundedParameter` recopiés, avec une forme déjà divergente (`singleParameter` inliné côté Interactions). À regrouper dans le même commit de factorisation que la mineure 1 (par exemple `httpjson.BoundedQueryInt(query, name, fallback, max)` rendant les `problem.FieldError`), les constantes `invalide`/`hors_limites` avec. Dev : back.
4. `web/src/features/notifications/hooks/useReconnect.ts:12-22` — même appel `session.signIn({ returnPath: \`${pathname}${search}\`, language })` que `app/routing/useAuthenticationGate.ts:45` et `app/layout/useAuthActions.ts:37`. Si un quatrième écran en a besoin, extraire un `useSignInHere()` dans `auth/`. Dev : front.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 0 |
| Mineure | 4 |
| Suggestion | 4 |

**Fusion possible.** Aucun bug constaté sur la transaction et l'outbox, l'idempotence, la validation, l'autorisation, la pagination, le SQL ou les droits. Aucune régression : tous les appelants de `NewStore` sont adaptés, et les tests d'outbox existants ne sont filtrés que par type de message, sans assertion affaiblie. Pas de code mort ni de commentaire dans le code. Les mineures 2 et 3 (front) et 4 (D05) se corrigent dans un commit de nettoyage. La mineure 1 (factorisation de la configuration dans `libs/platform`) peut passer dans un commit `refactor` séparé, avant le prochain service.
