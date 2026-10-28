# US-023 — Revue de code (back et front)

| Champ | Valeur |
|---|---|
| US | [US-023 — Filtrer le catalogue par catégorie et consulter la fiche d'un article](../../../backlog/US-023-catalogue-filtre-fiche.md), back d'[US-010](../../../backlog/US-010-catalogue-public.md) AC2 |
| Branche | `feat/US-023-catalogue-filtre-fiche` (base `origin/main`, commits `36174b2`, `1adf105`, `b5806a6`) |
| Date | 2026-10-28 |
| Périmètre | `services/place-de-marche` (API publique, stockage, migration 00004, tests), contrat `v1.openapi.yaml`, `web/` (filtre, fiche, refactors partagés), README, rapports de tests US-010 et US-023 ; 74 fichiers |
| Références | `contracts/api/place-de-marche/v1.openapi.yaml`, `services/place-de-marche/README.md`, `web/README.md`, [rapport de tests](../tests/US-023-rapport-tests.md) |
| Verdict | **Fusion possible** (0 bloquante, 0 majeure, 2 mineures, 3 suggestions) ; photos d'AC2 à trancher hors code |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l .`, `go vet ./...` (place-de-marche) | Aucun écart |
| `golangci-lint run ./...` (v2.15.0) | 0 issue |
| `go test ./...` (dont intégration PostgreSQL testcontainers) | Vert |
| `npm run lint`, `npm run typecheck`, `npx prettier --check src` | 0 erreur |
| `npm run i18n:check` | 175 clés alignées, fr et en |
| `npx vitest run` | 28 fichiers, 303 tests verts (dont US-008, US-010, US-025 inchangés) |
| Recherche de commentaires dans le diff | Aucun (seule l'annotation `-- +goose Up`, requise par goose) |
| Recherche de mentions d'outils d'IA dans le diff et les commits | Aucune |

## Points forts

- Aucune fuite : `stockage/catalogue.go:16-18` filtre `etat = 'publie' AND prix_centimes IS NOT NULL` dans un fragment commun aux trois requêtes ; les colonnes lues (`catalogueEntryColumns`, l. 14, plus description et frais de port) excluent `vendeur_id`, `revision`, `motif`, dates. Requêtes paramétrées (`$1..$3`), aucune concaténation de saisie.
- 404 identique : identifiant mal formé (`api/catalogue.go:57-59`), inexistant ou non publié (`ErrArticleNotFound` via `writeError`, `handlers.go:260`) produisent le même `problem.NotFound()` ; test `us023Ac3NonPublieIntrouvable`.
- Paramètres : `singleParameter` (`handlers.go:244`) refuse répétition et valeur vide, réutilisé par `boundedParameter` sans changement de comportement pour la liste vendeur (`?page=` déjà invalide auparavant) ; catégorie hors motif ou inexistante en 400 `categorie_inconnue` (`catalogue.go:45-58`), `wellFormed` factorisé avec la validation du brouillon (`validation.go:99`). Cas couverts (`catalogue_test.go:369-383`).
- Cohérence total/page : décompte et liste dans une même transaction `REPEATABLE READ READ ONLY` (`stockage/catalogue.go:35,56`).
- Migration 00004 : index partiels `WHERE etat = 'publie'` alignés sur l'ordre `modifie_le DESC, id DESC` et sur le filtre catégorie ; test de présence et de rejeu sur base peuplée.
- Contrat OpenAPI mis à jour (routes publiques, `additionalProperties: false`, 404 commun) et vérifié par test de conformité en mémoire et sur PostgreSQL.
- Front : réponses validées champ par champ (`parsePublishedArticle`, `cataloguePayload.ts:45`) ; description en texte React brut avec `pre-line` (`ArticleDescription.tsx:29-35`), aucun `dangerouslySetInnerHTML` ; `encodeURIComponent` sur l'identifiant (`httpCatalogueClient.ts:18`, `cataloguePaths.ts:8`).
- URL et historique : filtre dans `?categorie=`, retour à la page 1, entrée d'historique, identifiant hors motif ignoré (`useCatalogueCategoryParam.ts`) ; retour au catalogue via un `state` traité comme `unknown` et validé par motif (`catalogueReturn.ts:14-20`), pas de redirection ouverte.
- Accessibilité : `preserveFocusState` garde le focus sur le filtre natif, focus sur le `h1` sinon (`usePageIdentity.ts:15`), annonce unique `role="status"`, `prefers-reduced-motion` respecté dans `LinkCard`.
- Refactors partagés sans régression : `parseMoney`/`parseCategory`/`parseNullable` déplacés à l'identique dans `shared/marketplacePayload.ts` ; `LinkCard` et `CardTitle` reprennent exactement le rendu de `MyArticleCard` (tests US-011/US-012 verts).
- Séparation hooks/présentation nette (`useArticlePage`, `useArticlePageText`, `usePublishedArticleSummary`, `useCategoryFilter`), unions discriminées exhaustives.

## Issues

### Bloquantes

Aucune.

### Majeures

Aucune.

### Mineures

1. `web/src/features/catalogue/api/cataloguePayload.ts:38` — `parseCatalogueCategories` est la copie exacte de `parseSellerCategories` (`web/src/features/seller/api/sellerArticlePayload.ts:74`). La factorisation vers `shared/marketplacePayload.ts` s'est arrêtée à mi-chemin : y ajouter `parseCategoryList(payload)` et l'utiliser des deux côtés. Dev : front.
2. `web/src/features/catalogue/domain/catalogueProblems.ts:5` — `notFoundStatuses = [400, 404]` : le contrat de `GET /articles/{id}` ne prévoit pas de 400 (identifiant mal formé déjà en 404 côté back), et un 400 inattendu serait présenté comme « Article introuvable » au lieu de l'état d'erreur. Comportement documenté dans `web/README.md`, mais il masque un écart de contrat ; restreindre à 404. Dev : front.

### Suggestions

1. `services/place-de-marche/internal/annonces/stockage/catalogue.go:79` — `FindPublished` ouvre une transaction `REPEATABLE READ READ ONLY` pour une seule requête : une requête isolée est déjà cohérente, et `BEGIN`/`COMMIT` ajoutent deux allers-retours sur la route la plus appelée (p95 d'AC4). `s.pool.QueryRow` suffit, sauf si le `READ ONLY` est voulu comme défense en profondeur (le préciser alors dans le README). Dev : back.
2. `services/place-de-marche/internal/annonces/service.go:35-36` — `Articles` mêle désormais écriture vendeur et lecture publique ; un port dédié (`PublishedCatalogue` avec `ListPublished`/`FindPublished`) limiterait ce que doivent implémenter les doubles de test et isolerait le catalogue. Non bloquant, l'implémentation unique `Store` reste simple. Dev : back.
3. `web/src/features/seller/components/ArticleSummary.tsx:19-34` — même structure `dl/dt/dd` que `FactList`/`FactItem` ; réutiliser ces composants (en autorisant `pre-line` sur la valeur) éviterait deux rendus divergents de la même liste de faits. Dev : front.

## Point à trancher (hors code)

- AC2 exige des photos ; US-023 dépend d'US-020, encore « À faire », et le contrat v1 n'a pas de champ photo. Écart signalé par le rapport de tests (clarification) et le README ; Martin décide si l'AC2 est livré sans photos ou si la fusion attend US-020.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 0 |
| Mineure | 2 |
| Suggestion | 3 |

**Fusion possible.** Les mineures peuvent être traitées dans ce lot ou le suivant ; la question des photos d'AC2 relève de Martin.
