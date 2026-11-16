# US-030 — Revue de code

| Champ | Valeur |
|---|---|
| US | [US-030 — Réserver un article publié](../../../backlog/US-030-reserver-article.md) |
| Branche | `feat/US-030-reserver-article` (basée sur `origin/main`, HEAD `46738fa`) ; changements non commités revus par `git diff` et fichiers non suivis |
| Date | 2026-11-16 |
| Périmètre | `services/transactions` (nouveau) ; `services/place-de-marche` : `internal/reservations`, migration 00009, port interne (`serve.go`, `config.go`, `Dockerfile`), champ `reserve` du catalogue ; `libs/platform` : `token` (`verifier.go`, `middleware.go`, `delegation.go`, `credential.go`), `serviceclient`, `httpjson/decode.go` ; `infra/identite` (realm, constantes, tests) ; contrats `transactions/v1`, `place-de-marche/interne-v1`, `place-de-marche/v1` ; front `web/src/features/orders`, catalogue, `auth/useReconnect.ts`, configuration ; tests ajoutés ([rapport de tests](../tests/US-030-rapport-tests.md)) |
| Références | [D06](../../decisions/D06-authentification-autorisation.md) (avis sécurité § 3 à 5, MA3 à MA6), [D07](../../decisions/D07-donnees.md) (Q3, option R1), [D12](../../decisions/D12-indicateurs-qualite.md) (I4), ENF-01, ES-13, ES-38 ; décisions de Martin du 2026-11-16 (vendeur sur son propre article : 422 identique ; commission 5 % du total, demi-centime vers le haut) |
| Hors revue (connu) | AC7 (route absente de l'ALB, politique réseau) reporté à la session AWS ; concurrence, échéance, article figé (US-031) ; espace acheteur (US-032) |
| Verdict | **Fusion après correction** (0 bloquante, 2 majeures, 2 mineures, 2 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l .`, `go vet ./...` (`libs/platform`, `services/transactions`, `services/place-de-marche`, `infra/identite`) | 0 fichier, 0 remarque |
| `golangci-lint run --config ../../.golangci.yml ./...` (mêmes modules) | 0 issue partout |
| `go test ./cmd/place-de-marche/ -run TestUS030 -count=1 -v` (Docker 28.5.1) | `ok` 172 s, 13 sous-tests PASS (AC1 à AC6, AC8, AC9, exposition des ports, jeton jamais journalisé, contrats OpenAPI) |
| `go test ./...` (`infra/identite`) | `ok` 132 s (`TestUS030JetonDeServiceDeTransactions`, `TestSecondFacteurDuRealmCollector`) |
| `go test ./cmd/place-de-marche/ -run 'Catalogue\|TestUS010\|TestUS023\|Config\|Services'` | `ok` (catalogue public avec `reserve`, configuration, sans régression) |
| `go test -coverpkg=./... ./...` (`libs/platform`) : branche contre `origin/main` (extrait par `git archive`, contrats compris) | **53,4 % → 50,2 %** ; 0 % sur `requireScope`, `ForService`, `Delegation.*`, `Credential.*`, `DecodeIgnoringUnknownFields`, `serviceclient.*` (voir majeure 1) |
| `go test -coverpkg=./... ./...` (`services/transactions`) | `ok`, couverture totale **4,1 %** (seul `internal/montants` a des tests) (voir majeure 2) |
| `tools/ci/changements.sh` (lignes 86-90, 98-101) et job `go` de `.github/workflows/ci.yml` (ligne 234) | Une PR qui ne touche que `services/transactions/**` ne lance que le module Transactions ; l'étape de couverture est « non bloquante », aucun fichier de référence `couverture-reference.json` |
| `npx tsc -b`, `npx eslint . --max-warnings 0`, `npx prettier --check .`, `npm run i18n:check` (`web`) | 0 erreur ; Prettier conforme ; 299 clés alignées fr/en |
| `npx vitest run` (`web`) | 490/492 ; les 2 échecs sont des délais dépassés de US-008 AC10 et US-025 AC5 (déconnexion du vendeur, fichiers non touchés), rejoués seuls : 19/19 PASS. Instabilité préexistante, déjà tracée dans le rapport de tests |
| `gitleaks dir --config .gitleaks.toml` (realm, services, libs, contrats, front `orders`) | Aucune fuite ; le realm ne contient que l'emplacement `${SECRET_CLIENT_TRANSACTIONS}` |
| Commentaires ajoutés (`//`, `/* */`) dans le Go et le TypeScript du diff | Aucun (hors `go:embed`) |
| Appelants des symboles partagés modifiés | `ArticleSummaryCard` (prop `action` obligatoire) : seul appelant `ArticleDetails.tsx:26`, mis à jour ; `useReconnect` déplacé dans `src/auth/` : les 3 imports pointent sur le nouveau chemin, ancien fichier supprimé ; `token.Authenticate` (crédit ajouté au contexte) : comportement inchangé pour Place de marché et Interactions ; `httpjson.Decode` : chemin strict inchangé, tests existants verts ; catalogue (`reserve` obligatoire au contrat) : `catalogue_test.go` mis à jour, front tolérant à l'absence |

## Points forts

- **D06 respectée à la lettre sur « réserver »** : `token.Delegation` (`libs/platform/token/delegation.go:22-35`) contrôle d'abord `Authorization` (401) puis `X-Collector-User-Token` (403), une seule occurrence exigée, rôle `acheteur`, acheteur lu dans le seul `sub` ; le vérificateur de service est dérivé du vérificateur d'utilisateur par `ForService` (`verifier.go:91-103`), qui partage le jeu de clés et impose `azp` et la portée (`requireScope`, `scope` découpé par espaces). Aucun corps lu côté Place de marché.
- **Piège Keycloak des rôles par défaut neutralisé et prouvé** : compte `service-account-transactions` déclaré avec `realmRoles: []`, client `transactions` sans portée `roles`, `fullScopeAllowed: false`, portée `reserver` seule porteuse de l'audience `place-de-marche` ; `default-roles-collector` redéclaré dans `roles.realm` (constat XP03). Secret hors JSON, import refusé si la variable est vide, test qui relit le secret importé.
- **Jeton de l'acheteur jamais exposé** : type `token.Credential` masqué pour `fmt` (`%v`, `%#v`) et `slog` (`credential.go`) ; `serviceclient` refuse toute redirection (le jeton de service ne suit pas) ; scénario de test dédié au journal.
- **SQL de D07 Q3 repris fidèlement** (`reservations/stockage/postgres.go`) : verrou consultatif de transaction par acheteur, refus si réservation active ailleurs, `UPDATE` conditionnel qui conserve `reservation_id` et `reserve_jusqu_a` pour le même acheteur, horloge de la base seule ; durée plafonnée par `Term.Duration()` ; journal `reservation` en `ON CONFLICT DO NOTHING` sans `UPDATE` accordé. Côté Transactions, `UNIQUE (reservation_id)` puis relecture filtrée sur l'acheteur : idempotence propre.
- **Montants exacts (ENF-01)** : `montants.Compute` en `int64`, borne `maxCommissionableCents` calculée pour que `total × 500 + 5 000` ne déborde jamais, arrondi demi-centime vers le haut conforme à la décision de Martin ; contraintes `CHECK` en base (`total = prix + frais`, `commission + vendeur = total`, positivité) ; tests exhaustifs de 0 à 20 000 c et tirages jusqu'à 10¹² c.
- **Port interne isolé** : second serveur HTTP sur `ADRESSE_INTERNE`, mux distinct, route absente du port public et routes publiques absentes du port interne (prouvé par test).
- **Traduction des erreurs Place de marché → Transactions** complète et sûre : 422 et les deux 409 typés propagés, tout le reste (401, 403, 404, 5xx, corps illisible, type inconnu, devise autre qu'EUR, échéance absente, article différent) en 503 ou 500 sans commande créée ; réponse lue avec `LimitReader`.
- **Champs inconnus ignorés uniquement là où l'AC6 le veut** : `DecodeIgnoringUnknownFields` n'est appelé que par `POST /transactions/v1/commandes` ; doublons de clés, taille, type de média restent contrôlés ; `Decode` strict inchangé partout ailleurs.
- **Front propre** : présentation pure, logique dans les hooks (`useArticleReservation`, `useReserveArticle`, `useOrderConfirmationPage`), parse strict de la commande (`orderPayload.ts`), focus déplacé sur l'alerte d'échec, icônes `aria-hidden`, mouvements réduits respectés, `Stack` partout, `useReconnect` remonté dans `src/auth/` au lieu d'être recopié, `createAppServices` factorisé (`httpFor`).
- **Documentation à jour** : READMEs de Transactions, de la Place de marché et d'`infra/identite`, `web/README.md`, contrats OpenAPI interne et public.

## Issues

### Bloquantes

Aucune.

### Majeures

1. `libs/platform/token/verifier.go:91-136`, `libs/platform/token/delegation.go`, `libs/platform/token/credential.go`, `libs/platform/serviceclient/client.go`, `libs/platform/httpjson/decode.go:76` — code de sécurité partagé sans test unitaire dans son module. Mesuré : 0 % sur `requireScope`, `ForService`, `Delegation.Authenticate`/`callerAuthenticated`/`propagatedUser`/`userTokenOf`, `Credential.String`/`GoString`/`LogValue`, `DecodeIgnoringUnknownFields`, `LoadCredentials`, `New`, `refuseRedirects`. La couverture de `libs/platform` passe de 53,4 % à 50,2 % (−3,2 points), au-delà de la tolérance de 0,5 point et du seuil d'alerte de 1 point de D12 (I4). **Effet sur la CI** : rien n'échoue aujourd'hui, car l'étape de couverture est encore « non bloquante » et le fichier de référence n'existe pas. L'indicateur I4 se dégrade quand même au relevé. Surtout, D06 (§ mise en œuvre, `jwx`) exige des « tests unitaires sans Keycloak, jetons forgés pour chaque refus (signature, émetteur, audience, `azp`, portée, expiration, `none`) » : le refus pour portée n'est prouvé qu'à travers le banc de la Place de marché. Correction, dans `token/verifier_test.go` (le banc `nouveauBanc`/`jeton`/`remplacer` existe déjà) : `ForService` refuse une portée vide et un `azp` vide ; jeton de service accepté avec `scope = "basic reserver"`, refusé sans `scope`, avec `scope` non textuel ou `"reserverx"`, avec un autre `azp`. Dans un nouveau `token/delegation_test.go` (`httptest`) : 401 sans `Authorization`, avec un jeton d'utilisateur dans `Authorization` ; 403 sans second en-tête, avec l'en-tête répété, avec un jeton de service dans le second en-tête, sans le rôle ; 200 avec le `sub` de l'acheteur dans le contexte. Pour `Credential` : `fmt.Sprintf("%v %+v %#v")` et un `slog` JSON ne contiennent jamais la valeur. Pour `serviceclient` : `LoadCredentials` (secret absent, URL invalide, valeurs par défaut) et un `httptest` qui redirige et dont la cible ne reçoit jamais `Authorization`. Pour `httpjson` : champ inconnu ignoré, clé dupliquée toujours refusée. Dev : back.

2. `services/transactions/internal/commandes/marche/client.go`, `internal/commandes/api/handlers.go`, `internal/commandes/stockage/postgres.go` — Transactions n'a de tests propres que sur `internal/montants` (couverture du module : 4,1 %). Son API, son client de la Place de marché et son stockage ne sont prouvés que par `TestUS030BoutEnBoutTransactionsPlaceDeMarche`, qui vit dans le module `services/place-de-marche` et lance le binaire. Deux conséquences constatées : (a) la couverture de Transactions (I4) ne compte rien de ce code, et la première référence « pas de baisse » serait figée à 4 % ; (b) d'après `tools/ci/changements.sh:86-90` et `:98-101`, une PR qui ne modifie que `services/transactions/**` ne lance que le job `services/transactions` : une régression de la traduction des erreurs, du décodage de la réservation ou de l'idempotence passerait en vert. Correction : `marche/client_test.go` avec `httptest.Server` : en-têtes envoyés (`X-Collector-User-Token` brut, `Accept`, corrélation, aucun corps) ; 201/200 décodés ; 422, 409 de chacun des deux types, 409 de type inconnu, 401, 403, 404, 500, corps illisible, devise autre qu'EUR, montant absent, échéance absente, tous traduits comme le tableau du README. `api/handlers_test.go` avec un faux `OrderService` et un faux authentificateur : 401 sans revendications, 400 `articleId` absent ou non UUID, champs `prix`/`acheteurId` ignorés, 201 avec `Location`, 200 au nouvel essai, 404 au `GET` d'une commande d'un autre acheteur, chaque erreur métier vers son statut. `commandes/service_test.go` : article de la réservation différent de l'article demandé, donc erreur sans enregistrement. Le stockage peut rester prouvé par le banc. Côté CI, déclarer que `services/transactions/**` déclenche aussi `services/place-de-marche` tant que le banc de bout en bout y vit. Dev : back (tests), devops (`changements.sh`).

### Mineures

1. `services/place-de-marche/internal/reservations/stockage/postgres.go:41-45` — `reservableQuery`, qui explique l'échec de l'`UPDATE`, n'a pas la condition `NOT vendu` de `reserveQuery` (`:32`). Un article vendu serait annoncé « déjà réservé » (409) au lieu de « non réservable » (422). La branche est inatteignable aujourd'hui (`vendu` n'est jamais mis à `true`), mais elle le deviendra avec le paiement. Ajouter `AND NOT vendu` à `reservableQuery`, ou factoriser les conditions communes dans une constante, comme `heldByBuyer`. Dev : back.
2. `web/src/features/catalogue/api/catalogueQueryKeys.ts:1` — `catalogueQueryRoot` est introduit pour l'invalidation après réservation (`useReserveArticle.ts:35`), mais les clés du catalogue écrivent toujours le littéral `'catalogue'` (`hooks/useCatalogueCategories.ts:7`, `hooks/usePublishedArticle.ts:5`, `hooks/usePublishedArticles.ts:6`). Si l'une change, l'invalidation ne rafraîchit plus la mention « réservé », sans qu'aucune erreur ne le signale. Construire les trois clés à partir de `catalogueQueryRoot` (`[...catalogueQueryRoot, 'published-article', id]`). Dev : front.

### Suggestions

1. `web/src/features/catalogue/components/ArticleDetails.tsx:2` et `web/src/features/catalogue/api/demoReservations.ts` — dépendance circulaire entre fonctionnalités : `catalogue` importe `orders/ArticleReservation`, et `orders` importe `catalogue` (`demoReservations`, `catalogueQueryKeys`, `cataloguePaths`, `demoArticles`). En plus, l'état de démonstration des réservations est muté par `orders/api/demoOrdersClient.ts` mais rangé dans `catalogue/api`. `ArticleSummaryCard` accepte déjà une prop `action` : faire remonter ce point d'insertion jusqu'à `ArticleDetails` et `ArticlePage`, puis composer `ArticleReservation` au niveau de la route (`app/`). Le catalogue ne dépend alors plus des commandes. Dev : front.
2. `services/transactions/internal/commandes/api/handlers.go:25-32` et `internal/commandes/marche/client.go:22-23` — les URN `article-deja-reserve` et `reservation-deja-active` sont écrites deux fois dans le même module, et `problem.Typed` est recopié à l'identique dans `place-de-marche/internal/reservations/api/handlers.go:20-27`. Ces URN font partie du contrat. Exposer des constantes `ProblemTypeAlreadyReserved` et `ProblemTypeActiveElsewhere` dans `commandes`, utilisées par `api` et `marche`, pour qu'une faute de frappe d'un côté ne transforme pas silencieusement un 409 en 503. Dev : back.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 2 |
| Mineure | 2 |
| Suggestion | 2 |

**Fusion après correction des majeures 1 et 2.** Le comportement est juste et réellement prouvé : les AC1 à AC6, AC8 et AC9 sont couverts par un banc de bout en bout vert, rejoué en revue (Keycloak, deux bases PostgreSQL, deux services réels). Le code suit D06 (ordre des contrôles, deux en-têtes, `azp`, portée, compte de service sans rôle, secret hors JSON, jeton masqué) et D07 Q3 (SQL atomique, verrou consultatif, droits minimaux). Les montants sont exacts et sans débordement, et le port interne est isolé. Lint, format et types sont propres, sans commentaire ni code mort, et aucune régression n'a été trouvée sur les appelants modifiés. Les majeures portent sur la preuve dans le bon module : du code de sécurité partagé et un service entier sans tests unitaires propres. La CI n'échoue pas aujourd'hui, mais I4 baisse (−3,2 points sur `libs/platform`, 4,1 % sur Transactions), D06 demande ces tests, et une PR limitée à Transactions ne lancerait aucun test de son API. Répartition : back pour les majeures 1 et 2, la mineure 1 et la suggestion 2 ; devops pour le filtre de la CI (majeure 2) ; front pour la mineure 2 et la suggestion 1. Aucune correction pour le testeur : ses tests d'acceptation sont complets ; il manque les tests unitaires du code, qui reviennent au dev.

Hors périmètre, à porter au pilotage : la porte de couverture « pas de baisse » prévue par D12 dès le sprint 2 n'est pas encore en place dans la CI (étape « non bloquante », pas de `couverture-reference.json`).

## Retours traités (2026-11-16)

| Retour | Traitement |
|---|---|
| Majeure 1 — code de sécurité partagé sans tests unitaires | Tests de `requireScope`, `ForService`, `Delegation` (22 refus, 1 succès), `Credential`, `DecodeIgnoringUnknownFields`, `serviceclient` ; couverture `libs/platform` 50,2 % → 57,2 % (main : 53,4 %) |
| Majeure 2 — Transactions testé à 4,1 % dans son module | Tests du client `marche`, des handlers et du service : 62,2 % ; filtre CI : une modification de Transactions, Interactions, Contrôle, `infra/identite` ou `infra/canal` relance aussi la Place de marché (`tools/ci/changements.sh`) |
| Mineure — article vendu annoncé 409 | `NOT vendu` dans `reservableQuery` : 422, prouvé par le banc |
| Mineure — clé `'catalogue'` en dur | `catalogueQueryRoot` réutilisé par les trois hooks |
| Suggestion — dépendance circulaire catalogue ↔ orders | `ArticleReservation` composé par `ReservableArticlePage` ; état de démo dans `web/src/api` |
| Suggestion — URN en double | Constantes `services/transactions/internal/problemes` |
| Trouvés par les tests unitaires | Adresse de jeton relative refusée au démarrage ; montant négatif de la Place de marché → 503 |

Verdict après corrections : **fusion possible**. Hors US : la porte « pas de baisse de couverture » de D12 n'est pas encore active en CI (dette de pilotage).
