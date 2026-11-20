# US-032 — Revue de code

| Champ | Valeur |
|---|---|
| US | [US-032 — Suivre ses réservations dans l'espace acheteur](../../../backlog/US-032-espace-acheteur.md) |
| Branche | `feat/US-032-espace-acheteur` (worktree `Collector-US-032`) ; changements non commités revus par `git diff main` et fichiers non suivis |
| Date | 2026-11-20 |
| Périmètre | `services/transactions` : `GET /transactions/v1/commandes` (`api/handlers.go`, `api/response.go`, `service.go`, `commande.go`, `stockage/postgres.go`), titre figé (`marche/client.go`), migration `00002_figer_le_titre_et_lister_les_commandes.sql` ; `services/place-de-marche` : titre renvoyé par « réserver » (`reservations/{reservation.go, stockage/postgres.go, api/handlers.go}`) ; contrats `transactions/v1`, `place-de-marche/interne-v1` ; front `web/src/features/orders` (écran « Mes réservations » sur `/acheteur`, polling, démo, titre sur la confirmation), navigation (`app/layout`, `routes.tsx`), `ui/animations/liftOnHoverSx.ts`, `ui/hooks/useNow.ts`, `i18n` ; tests ajoutés ([rapport de tests](../tests/US-032-rapport-tests.md)) |
| Références | AC1 à AC4 ; ES-04 ; [D02](../../decisions/D02-decoupage-applicatif.md) ; décision US-031 (statut `expiree` dérivé à la lecture, horloge de Transactions) |
| Hors revue (connu) | Paiement (hors POC) |
| Verdict | **Fusion après correction** (0 bloquante, 1 majeure, 2 mineures, 2 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l .`, `go vet ./...` (`services/transactions`, `services/place-de-marche`) | 0 fichier, 0 remarque (tests compilés) |
| `golangci-lint run --config ../../.golangci.yml ./...` (mêmes modules) | 0 issue |
| `go test ./... -count=1` (`services/transactions`) | `ok` dans `commandes`, `commandes/api`, `commandes/marche`, `montants` |
| `npx tsc -b`, `npm run lint`, `npx prettier --check src e2e`, `npm run i18n:check` (`web`) | 0 erreur ; Prettier conforme ; 332 clés alignées fr/en |
| `npx vitest run src/acceptance/US-032 src/acceptance/US-030 src/acceptance/US-031` | 11 fichiers, 99 tests verts |
| Bancs Go avec Docker (`TestUS032…`, `TestUS030…`, `TestUS031…`) | Non rejoués en revue : verts dans le rapport de tests (101/101 sur `place-de-marche`, passe au repos) |
| Commentaires ajoutés (`//`, `/* */`) dans le Go et le TypeScript du diff | Aucun (seule la directive `-- +goose Up` dans la migration, obligatoire) |
| Appelants des symboles partagés modifiés | `api.OrderService` (+ `OwnOrders`) : implémenté par `commandes.Service` (`service.go:62`) et le double de test, consommé par `serve.go:48` ; `commandes.Orders` (+ `ListOwned`) : `stockage.Store` (`postgres.go:85`) et `registreSimule` ; `orderColumns`/`orderRow.targets` (+ `titre`) : ordre identique dans les trois requêtes qui les utilisent (`insertOrderQuery`, `findByReservationQuery`, `findOwnedQuery`) et dans `listOwnedQuery` ; `reserveQuery` (+ `titre` dans `RETURNING`) : seul appelant `holdArticle`, `Scan` aligné ; `reservations.Reservation.Title` : seul producteur `holdArticle` ; `OrdersClient.listMyOrders` : implémenté par les clients HTTP et démo ; `Order.title` : `parseOrder` seul constructeur, simulateurs US-030 (Vitest et Playwright) mis à jour ; `parseOrder(payload, path = 'commande')` : appelants existants inchangés et compatibles ; `SignedInAccount` (+ `canOpenBuyerSpace`) : seul appelant `AccountControl.tsx`, mis à jour |
| Compatibilité croisée des versions (ordre de déploiement) | Nouvelle Place de marché + ancien Transactions : compatible (`json.Unmarshal` sans `DisallowUnknownFields`, `marche/client.go:120`, `titre` ignoré). Ancienne Place de marché + nouveau Transactions : **toute réservation répond 503** (`titleError`, `marche/client.go:148`, refuse le titre absent). Ordre obligatoire : Place de marché d'abord |

## Points forts

- **ES-04 / AC3 tenu par construction** : l'acheteur n'est jamais lu dans la requête, seulement dans le jeton (`buyerOf`, `api/handlers.go:133`) ; les deux requêtes de la liste filtrent sur `acheteur_id = $1` (`stockage/postgres.go:31-35`) ; route derrière `buyerOnly` (`handlers.go:66`). Le détail d'autrui reste un 404 identique à l'inexistant (prouvé octet pour octet par le banc).
- **Pagination cohérente avec l'existant** : réutilise `httpserver.PaginationOf` et les mêmes bornes que la Place de marché (20/50/10 000) ; contrat et implémentation alignés (`hors_limites` = `problem.CodeOutOfBounds`, `PaginationInvalide`, `elements` toujours un tableau via `make(…, 0, …)`, `response.go:53`). Le décompte et la page sont lus dans un même instantané `REPEATABLE READ` en lecture seule (`postgres.go:37`, `:85-102`), comme `ListToVerify` côté Place de marché : `total` et `elements` ne peuvent pas diverger.
- **Index exactement taillé pour la requête** : `commande_par_acheteur_idx (acheteur_id, cree_le DESC, id DESC)` suit le filtre et le tri de `listOwnedQuery` (tri stable, départage par `id` testé sur des créations à la même seconde).
- **Migration sûre** : `ADD COLUMN … NOT NULL DEFAULT 'Article sans titre'` puis `DROP DEFAULT` reprend les lignes existantes sans réécriture applicative et force ensuite chaque `INSERT` à fournir le titre ; `CHECK (char_length BETWEEN 1 AND 200)` aligné sur le contrat (`minLength 1`, `maxLength 200`) et sur `MaxTitleLength` (`commande.go:77`). Le rôle d'exécution garde `SELECT, INSERT` seulement. Testée sur des commandes réellement créées en version 1.
- **Titre figé, validé à la frontière** : `titleError` (`marche/client.go:148`) compte en runes (200 caractères accentués acceptés) et refuse un titre absent ou blanc ; cohérent avec la Place de marché, qui ne publie jamais un titre vide (règle `annonce_incomplete` du Contrôle).
- **Statut dérivé sur la liste comme sur le détail** : `OrderPage.AsOf` (`commande.go:145`) réutilise `Order.AsOf` ; aucun état écrit.
- **Front bien découpé** : logique dans les hooks (`useMyOrdersPage`, `toMyOrdersView`, `useMyOrderCard`, `useTimeLeftLabel`, `useExpiryAnnouncement`), composants de présentation fins, `switch` exhaustifs sur `MyOrdersView` ; polling arrêté sur 401/403 (`myOrdersPolling.ts`), arrêté en arrière-plan, relancé au retour sur l'onglet ; horloge locale (`useNow`) active seulement s'il reste une réservation en attente. Liste invalidée après une réservation (`useReserveArticle.ts`). Accessibilité soignée : `<ul>` étiquetée, `<time datetime>`, annonce des expirations en zone `status`, lien de détail avec nom accessible incluant le titre, cible de 2,75 rem, animation coupée sous `prefers-reduced-motion`. i18n complète FR/EN, toutes les nouvelles clés utilisées.
- **Démo cohérente** : une réservation expirée amorcée en scénario nominal, liste paginée et triée comme le back.

## Issues

### Bloquantes

Aucune.

### Majeures

1. `services/transactions/README.md` et `services/place-de-marche/README.md:35` — documentation devenue fausse ou incomplète, non mise à jour par le diff :
   - Transactions : la section « Réserver et commander » ne décrit pas `GET /transactions/v1/commandes` (tri `creeLe` puis `id` décroissants, `page`/`taille` et leurs bornes, 400 `invalide`/`hors_limites`, liste limitée au sujet du jeton) ; l'exemple JSON de commande n'a pas `titre` alors qu'il est désormais requis par le contrat ; l'étape 2-3 ne dit pas que le titre est lu et figé (ni le 503 si la Place de marché ne le fournit pas) ; la section « Base » ne cite que la migration 00001 (ni la colonne `titre`, ni sa valeur de reprise « Article sans titre », ni l'index `commande_par_acheteur_idx`).
   - Place de marché : « Réponse : `reservationId`, `articleId`, `echeance`… » omet `titre`.
   - **Ordre de déploiement non documenté** : vérifié en revue, un Transactions à jour face à une Place de marché antérieure refuse toutes les réservations (503, `titleError`), alors que l'ordre inverse est sans effet. Rien dans le dépôt ne l'indique à qui déploiera.
   Pourquoi : la DoD exige « Documentation touchée à jour » ; ces README sont la référence des contrats d'exploitation des deux services. Correction : compléter les deux README (nouvelle route, champ `titre`, migration 00002 et repli, index) et ajouter dans le README de Transactions une ligne « Déploiement : la Place de marché (champ `titre` dans « réserver ») doit être déployée avant Transactions ». Dev : back.

### Mineures

2. `web/src/ui/components/LinkCard.tsx:17-25` — le style de survol (transition, `translateY(-0.125rem)`, `boxShadow: 3`, coupure sous `prefers-reduced-motion`) est recopié à l'identique dans le nouveau `web/src/ui/animations/liftOnHoverSx.ts:3-10`. Deux sources pour une même animation du design system : une retouche n'en changerait qu'une (catalogue, notifications et espace vendeur d'un côté, « Mes réservations » de l'autre). Correction : faire consommer `liftOnHoverSx` par `LinkCard` (`sx={[liftOnHoverSx, { height: '100%' }]}`), sans changement de rendu. Dev : front.
3. `web/src/features/orders/api/orderPayload.ts:17` — `title: expectString(order.titre, …)` accepte une chaîne vide alors que le contrat impose `minLength: 1` et que `articleId` juste au-dessus passe par `expectNonEmptyString`. Un titre vide produirait un `h3` sans texte (lien vide) et « Voir la réservation « » ». Sans effet aujourd'hui (le back le refuse en amont), mais le parseur est la frontière qui protège l'écran. Correction : `expectNonEmptyString(order.titre, \`${path}.titre\`)`. Dev : front.

### Suggestions

1. `web/src/features/orders/hooks/myOrdersView.ts:19-42` — reprend presque ligne pour ligne `web/src/features/seller/hooks/myArticlesView.ts:17-36` (chargement/erreur/vide/prêt, `pageCount = max(1, ceil(total / taille))`, `isRefreshing = isPlaceholderData`) ; seule la branche d'échec d'accès diffère. Une fonction générique de vue paginée (`toPagedView(query, requestedPage)` dans `src/shared` ou `src/ui/hooks`), appelée par les deux après le contrôle d'accès, supprimerait le doublon. Dev : front.
2. `web/src/features/orders/hooks/useTimeLeftLabel.ts:14-15` et `fr.ts` (`orders.list.deadlineReached`) — à l'échéance, la carte affiche « Échéance atteinte, mise à jour de l'état en cours… », mais aucune relecture n'est déclenchée : l'état ne change qu'au prochain polling, jusqu'à 30 s plus tard (`myOrdersPolling.ts:3`). Conforme à AC2 (« quand la liste est rafraîchie »), mais le texte promet plus que le comportement. Au choix : déclencher `refetch` quand une réservation en attente franchit son échéance (le tick de `useNow` le permet), ou un libellé neutre (« Échéance atteinte »). Dev : front.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 1 |
| Mineure | 2 |
| Suggestion | 2 |

**Fusion après correction de la majeure 1.** Le code est conforme aux quatre AC et aux contrats : liste paginée cloisonnée par le jeton (ES-04, AC3), titre figé à la réservation avec migration de reprise sûre et index adapté, statut `expiree` dérivé à la lecture et visible au rafraîchissement (AC2), écran accessible (AC4) ; aucun appelant oublié, aucun code mort, aucun commentaire ; format, lint, types, i18n et tests unitaires/acceptation front rejoués verts. Seule la documentation des deux services n'a pas suivi (nouvelle route, `titre`, migration, et surtout l'ordre de déploiement Place de marché puis Transactions, vérifié comme non optionnel). Répartition : back pour la majeure 1 ; front pour les mineures 2 et 3 et les deux suggestions. Rien pour le testeur ni le devops.

## Retours traités

| # | Sévérité | Traitement |
|---|---|---|
| 1 | Majeure | README de Transactions (liste paginée, `titre`, migration 00002, ordre de déploiement) et de Place de marché (`titre` dans la réponse de réservation, ordre de déploiement) mis à jour. |
| 2 | Mineure | `LinkCard` réutilise `liftOnHoverSx` ; `liftOnHoverSx` typé par `satisfies SxProps<Theme>` pour être composable en tableau `sx`. |
| 3 | Mineure | `titre` lu par `expectNonEmptyString` (contrat `minLength: 1`). |
| S1 | Suggestion | Non retenue dans cette US : les deux vues paginées diffèrent (nom de la liste, branche d'accès refusé) ; factorisation reportée à un refactor dédié (`items`). |
| S2 | Suggestion | `useRefetchOnDeadline` relance la lecture une fois par échéance franchie (au plus 10 s après), sans boucle si le serveur répond encore « en attente ». |

Vérifications après corrections : `tsc -b`, ESLint, Prettier, i18n, Vitest US-030 à US-032 (99 tests) verts.
