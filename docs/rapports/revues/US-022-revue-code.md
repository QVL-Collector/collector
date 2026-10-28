# US-022 — Revue de code

| Champ | Valeur |
|---|---|
| US | [US-022 — Trancher les articles « à vérifier »](../../../backlog/US-022-ecran-admin-a-verifier.md) |
| Branche | `feat/US-022-ecran-admin-a-verifier` (commit `36e1b21`), empilée sur `feat/US-021-revalidation-apres-modification` (`63a50c8`, PR #27, revue à part) |
| Date | 2026-10-28 |
| Périmètre | `git diff feat/US-021-revalidation-apres-modification...HEAD` : 77 fichiers, 4 commits. Back `services/place-de-marche` (domaine `decision.go`, handlers et payload, stockage `postgres.go`, migration 00005, tests en mémoire et PostgreSQL), contrat `v1.openapi.yaml` (tag Administration), module `infra/identite` (realm Keycloak et tests testcontainers), CI (`tools/ci/changements.sh`, `.golangci.yml`), front `web/src/features/admin`, en-tête (`SpaceLink`), motifs partagés, i18n, tests d'acceptation US-022, [rapport de tests](../tests/US-022-rapport-tests.md) |
| Références | [D06](../../decisions/D06-authentification-autorisation.md) (realm en code, MFA admin, ES-20), [D07](../../decisions/D07-donnees.md) (décision avec révision), `services/place-de-marche/README.md`, `web/README.md`, `infra/identite/README.md`, README racine (CI) |
| Verdict | **Fusion après correction** (0 bloquante, 1 majeure, 3 mineures, 3 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `services/place-de-marche` : `gofmt -l .` | Aucun fichier |
| `services/place-de-marche` : `go vet ./...` | 0 constat |
| `services/place-de-marche` : `golangci-lint run --config ../../.golangci.yml ./...` | 0 issues |
| `services/place-de-marche` : `go test -count=1 ./...` (Docker disponible) | Vert, 96 s (dont `TestUS022…` en mémoire et `TestUS022IntegrationPostgreSQL`) |
| `infra/identite` : `gofmt -l .`, `go vet`, `golangci-lint --config ../../.golangci.yml` | 0 constat, 0 issues |
| `infra/identite` : `GOWORK=off go test -count=1 -v ./...` | Vert, 45 s : `TestSecondFacteurDuRealmCollector` et ses 5 sous-tests (vendeur sans OTP, admin configure l'OTP, admin saisit l'OTP, PKCE S256, redirection hors liste) |
| `web` : `npm run typecheck`, `npm run lint`, `npx prettier --check .` | 0 erreur, 0 avertissement, conforme |
| `web` : `npm run i18n:check` | `Traductions alignées : 226 clés, langues fr, en.` |
| `web` : `npx vitest run --pool=forks --poolOptions.forks.maxForks=2` | Vert : 30 fichiers, 346 tests (dont les cas US-022), aucun dépassement de délai sur ce passage |
| Recherche de commentaires dans les lignes ajoutées (Go, TS, SQL) | Aucun (seuls des commentaires d'en-tête de fichiers de configuration YAML et shell déjà commentés) |
| Recherche de mentions d'outils d'IA dans le diff et les messages de commit | Aucune |

## Points forts

- Autorisation (AC5) : `restrictedTo(role)` (`services/place-de-marche/internal/annonces/api/handlers.go:67-73`) factorise `sellerOnly` et `adminOnly` sur le même `Authenticate` + `token.RequireRole` ; les deux routes admin (l. 82-83) passent par `adminOnly`. `subjectOf[ID]` (l. 255) généralise `sellerOf` sans changer son comportement (401 si sujet absent ou illisible). Prouvé : vendeur et acheteur 403 sur les deux routes, sans jeton ou jeton invalide 401, admin 403 sur les routes vendeur (`administration_test.go:534-595`).
- Décision et audit dans la même transaction (AC3, AC4) : `Store.Decide` (`stockage/postgres.go:349`) fait, dans `pgx.BeginFunc`, un `UPDATE … WHERE id = $1 AND revision = $2 AND etat = 'a_verifier' RETURNING *` (`decideQuery`, l. 97) puis l'`INSERT INTO decision_admin` ; l'échec de l'un annule l'autre. Course vérifiée par raisonnement et par test : deux décisions concurrentes sur la même révision, la seconde attend le verrou de ligne, réévalue le `WHERE` (état changé) et ne trouve rien → 409 `article-non-a-verifier`, une seule ligne `decision_admin` (`administration_integration_test.go:178`, `ac3DecisionsConcurrentesSurPostgreSQL`). La contrainte `UNIQUE (article_id, revision)` ferme le cas en défense en profondeur.
- Ordre des conflits : identifiant de chemin illisible → 404 avant lecture du corps ; corps invalide → 400/422 ; puis `explainDecisionConflict` (l. 384) : article absent → 404, même révision mais plus `a_verifier` → 409 `article-non-a-verifier`, sinon 409 `revision-perimee` (`annonces.DecisionConflict`, `decision.go:90`). Logique partagée entre PostgreSQL et le faux stockage (`memoire_test.go`), sans duplication.
- Journal d'audit après validation : `h.logDecision` (`handlers.go:233`) n'est appelé qu'après le retour sans erreur de `service.Decide`, donc après `COMMIT` ; `event=annonces.decision_admin`, `admin_id`, `article_id`, `revision`, `decision`, `motif` (si refus). Vérifié sur le journal JSON réel (`exigerEvenementDeDecision`, `administration_test.go:169`).
- Migration 00005 : `decision_admin` avec contraintes nommées (décision connue, motif dans la liste fermée de l'admin, `motif` obligatoire si et seulement si `refuser`, révision ≥ 1, unicité par révision), `REVOKE UPDATE` (le rôle d'exécution n'a jamais eu `DELETE`) : table en ajout seul, vérifiée par `verifierDroitsMinimaux` (`integration_test.go:173-174`). Index partiel `article_a_verifier_idx (modifie_le, id) WHERE etat = 'a_verifier'` aligné sur `ORDER BY a.modifie_le ASC, a.id ASC` de `listToVerifyQuery`. Contrainte `article_motif_connu` étendue aux trois nouveaux motifs.
- Séparation des motifs : le consommateur `controle.resultat` v1 reste sur `ParseReviewReason` et ses six motifs (`review.go:40`, `controle/results.go:111`) ; les motifs de refus de l'admin passent par `ParseRefusalReason` (`decision.go:32`). Le contrat le dit (`MotifDuControle` / `MotifDeRefus`), le schéma de message n'est pas touché.
- Liste en `READ ONLY` `REPEATABLE READ` (`readOnlySnapshot`, total et page cohérents), même schéma que le catalogue. Contrat OpenAPI complet (403, 404, 409 `ConflitDecision`, 422 `MotifInvalide`) et vérifié par `TestUS022ReponsesAdministrationConformesAuContratOpenAPI`.
- Realm (`infra/identite/realm-collector.json`) : client public, code d'autorisation seul (`implicitFlowEnabled`, `directAccessGrantsEnabled` à `false` : pas de contournement de l'OTP par le grant « mot de passe »), `pkce.code.challenge.method = S256`, URI de redirection exactes, `fullScopeAllowed: false` avec `scopeMappings` explicites, trois mappers d'audience, jeton d'accès 300 s, rafraîchissement à usage unique, force brute active. Le sous-flux `CONDITIONAL` « Condition - user role » `admin` + « OTP Form » `REQUIRED` est le schéma Keycloak attendu ; prouvé de bout en bout par le test (y compris OTP faux refusé et absence de code avant configuration). Aucun utilisateur ni secret versionné ; mot de passe d'amorce aléatoire dans le test.
- CI : `infra/*/go.mod` ajouté à `lister_modules_go` et filtré par `TOUS_MODULES` (un changement dans `infra/canal`, sans `go.mod`, ne crée pas de job) ; la construction d'image est déjà conditionnée à la présence d'un `Dockerfile`. Exclusion gosec G505 limitée à `infra/identite/totp_test.go` et justifiée (TOTP RFC 6238 en HMAC-SHA1). README racine et `infra/identite/README.md` à jour.
- Front :
  - séparation stricte hooks / présentation : composants sans logique (`ReviewArticleCard`, `ReviewDecisionActions`, `RefusalDialog`, `DecisionFailureMessage`, `ReviewQueueContent`), fonctions pures de vue (`toReviewQueueView`, `toDecisionFailure`) testables isolément. Le découpage en petits hooks n'est pas de la sur-découpe : il reprend la granularité de `features/seller/hooks` (28 fichiers, même schéma `…View.ts` + `use…Announcement`), et les hooks partagés ont chacun deux appelants (`useReviewArticleTitle` : carte et annonce ; `useReloadReviewQueue` : décision et retrait) ;
  - AC7 : aucun `dangerouslySetInnerHTML`, titres interpolés par React et i18next en texte ; jeu de démonstration avec `<script>` et `<b>` ;
  - a11y : `Dialog` relié par `aria-labelledby`/`aria-describedby`, `fieldset`/`legend` pour les motifs, erreur « motif manquant » en `role="alert"` reliée par `aria-describedby`, boutons au nom accessible incluant le titre (« Publier « … » »), annonces en `role="status"`, focus ramené sur le titre de la liste après décision, cibles de 2,75 rem ;
  - double soumission impossible (`canDecide` faux pendant la requête), 409/404 → invitation à recharger et boutons désactivés (AC3) ;
  - refacto `SpaceLink` (`web/src/app/layout/SpaceLink.tsx`) : `SellerSpaceLink` et `AdminSpaceLink` partagent le rendu compact/étendu, comportement de `SellerSpaceLink` inchangé ; `parseDate` et `parseUuid` remontés de `sellerArticlePayload.ts` vers `shared/marketplacePayload.ts` (DRY), seul appelant précédent mis à jour ;
  - régression vérifiée : le vendeur lit désormais les motifs de l'admin (`sellerArticlePayload.ts` passe à `articleReasons`, `SellerArticle.reason` à `ArticleReason`) ; sans ce changement, un article refusé par l'admin aurait cassé l'espace vendeur (AC2). Libellés fr/en des trois nouveaux motifs présents.

## Issues

### Bloquantes

Aucune.

### Majeures

1. `web/src/features/admin/hooks/reviewQueueView.ts:35` — `if (totalItems === 0 || articles.length === 0) return { status: 'empty' }` : quand l'admin tranche le dernier article d'une page au-delà de la première (plus de 20 articles à vérifier), `useRemoveReviewedArticle` vide la page en cache, puis le rechargement de la même page renvoie `elements: []` avec `total > 0`. La vue passe à `empty` : l'écran affiche « Aucun article à vérifier — Toutes les annonces ont été tranchées » alors que la page 1 en contient encore, et la pagination est masquée (`ReviewQueueResults` n'est plus rendu), sans moyen d'y revenir hors rechargement de la route. Information fausse sur l'écran de décision. `toMyArticlesView` (`features/seller/hooks/myArticlesView.ts:25`) ne teste que `totalItems === 0`. Fix : tester `totalItems === 0` seul pour l'état vide et, si `articles.length === 0` avec `totalItems > 0`, ramener la page demandée à la dernière page existante (`useAdminReviewPage` : `setPage(Math.min(page, pageCount))`, ou reculer d'une page après retrait du dernier élément) ; ajouter un test d'acceptation à 21 articles. Dev : front.

### Mineures

1. `services/place-de-marche/README.md:3, 17-31` — le README du service n'est pas étendu à US-022 : rien sur les routes `GET /marche/v1/admin/articles` et `POST /marche/v1/admin/articles/{id}/decision` (rôle `admin`, 403), la décision sur révision (409 `revision-perimee` / `article-non-a-verifier`, 422 motif), la table `decision_admin` en ajout seul (migration 00005, `REVOKE UPDATE`) ni l'événement `annonces.decision_admin`. La ligne 3 décrit toujours le module comme « brouillons du vendeur, catégories, soumission… ». Le `web/README.md` et `infra/identite/README.md` sont, eux, à jour. DoD « Documentation touchée à jour ». Ajouter une section « Décision d'administration (US-022) ». Dev : back.
2. `web/src/features/admin/domain/adminProblems.ts:2, 11` — code mort : `adminProblemTypes.invalidInput` et `adminHttpStatuses.unprocessableContent` ne sont référencés nulle part (recherché dans `web/src`, tests d'acceptation compris ; `decisionFailure.ts` ne traite que 403, 404 et 409). Sans effet à l'exécution, d'où mineure. Les supprimer. Dev : front.
3. `web/src/i18n/useControlReasonLabel.ts:5` (et clés `control.reasons.*` dans `fr.ts`/`en.ts`) — le hook traduit désormais `ArticleReason` (motifs du contrôle et motifs de refus de l'admin, utilisé par `useRefusalDialog` pour la liste fermée de l'admin), mais garde un nom et un espace de clés propres au contrôle automatique. Le back a fait la séparation inverse (`ParseReviewReason` / `ParseRefusalReason`). Renommer en `useArticleReasonLabel` et, au besoin, `control.reasons` en `article.reasons` (renommage mécanique, `i18n:check` le vérifie). Dev : front.

### Suggestions

1. `web/src/app/createAppServices.ts:14` — `adminClient?` est optionnel et `AppProviders` passe `services.adminClient ?? null`, alors que `sellerClient` est obligatoire et que le harnais US-010 fournit un `neutralSellerClient`. La seule raison est de ne pas toucher `catalogueHarness.tsx:143` ; une route `/admin` rendue sans client lève « AdminClientProvider is missing » au lieu d'une erreur de type. Rendre `adminClient` obligatoire et ajouter un `neutralAdminClient` au harnais, pour la symétrie. Dev : front.
2. `web/src/features/admin/domain/reviewArticle.ts:5` — `ReviewCategory` (`{ id, label }`) redéclare la forme rendue par `parseCategory` (`CategoryPayload`, `shared/marketplacePayload.ts`) ; réutiliser ce type. Dev : front.
3. `services/place-de-marche/internal/annonces/api/payload.go:276` — `reasonText` renvoie `""` pour un motif nul, valeur hors de l'énumération `MotifDuControle` que le contrat déclare obligatoire. Branche inatteignable (contrainte `article_motif_selon_etat` : un `a_verifier` a toujours un motif), donc sans effet ; si elle était atteinte, la réponse violerait le contrat en silence. Préférer une erreur explicite côté stockage ou documenter l'invariant par le type (motif non pointeur dans un type « article à vérifier »). Dev : back.

### Observations (hors issues)

- Validation de la décision : `revision` et `decision` invalides répondent 400, `motif` invalide 422 (`invalidReason`, `payload.go:160`). Choix cohérent (le motif dépend du verdict) et documenté dans le contrat (`MotifInvalide`) ; aucune action.
- `explainDecisionConflict` relit l'état hors transaction après l'échec de l'`UPDATE` : l'explication peut décrire un état postérieur (409 l'un pour l'autre), jamais un faux succès ; même schéma qu'`explainRefusal` validé en US-013 et US-021.
- Les rôles `moderation` et `administration` du realm ne sont lus par aucun service : ils servent le découpage ES-20 de D06 ; le back n'autorise que `admin`, qui est aussi la condition de l'OTP, donc aucun chemin de décision sans second facteur.
- Écart V1 assumé par l'US (écran servi par le front public, ES-20) : constaté, conforme au backlog.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 1 |
| Mineure | 3 |
| Suggestion | 3 |

**Fusion après correction de la majeure 1.** Le back, la migration, le realm et la CI sont prêts : autorisation, transaction décision + audit, ordre des conflits et second facteur admin sont corrects et prouvés sur PostgreSQL et Keycloak réels. Les mineures 1 à 3 sont à solder de préférence dans la même branche (un paragraphe de README, deux constantes à supprimer, un renommage).
