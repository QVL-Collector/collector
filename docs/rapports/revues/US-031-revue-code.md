# US-031 — Revue de code

| Champ | Valeur |
|---|---|
| US | [US-031 — Garantir qu'un objet unique n'est réservé qu'une fois](../../../backlog/US-031-garanties-reservation.md) |
| Branche | `feat/US-031-garanties-reservation` (basée sur `origin/main`, HEAD `afe167c`) ; changements non commités revus par `git diff` et fichiers non suivis |
| Date | 2026-11-16 |
| Périmètre | `services/place-de-marche` : `internal/annonces/{article.go, stockage/postgres.go, api/handlers.go}`, `internal/config/config.go`, README ; `services/transactions` : `internal/commandes/{commande.go, service.go}`, `cmd/transactions/serve.go`, README ; contrats `place-de-marche/v1`, `transactions/v1` ; front `web/src/features/orders` (statut `expiree`, `OrderDeadline`), `web/src/features/seller` (409 `article-reserve`), démo, i18n ; tests ajoutés ([rapport de tests](../tests/US-031-rapport-tests.md)) |
| Références | [D07](../../decisions/D07-donnees.md) (Q3, option R1), [D02](../../decisions/D02-decoupage-applicatif.md) ; ES-13, ES-14, ES-29 ; décisions de Martin du 2026-11-16 (pas de route « retirer » : le refus de la modification couvre le retrait ; `expiree` dérivé à la lecture avec l'horloge injectable de Transactions, écart assumé avec « horloge de la base » de D07) |
| Hors revue (connu) | Compensation, remboursement d'une confirmation tardive (V1) ; paiement (`vendu` jamais mis à `true` dans le POC) |
| Verdict | **Fusion possible** (0 bloquante, 0 majeure, 3 mineures, 2 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l .`, `go vet ./...` (`services/place-de-marche`, `services/transactions`) | 0 fichier, 0 remarque |
| `golangci-lint run ./...` (mêmes modules) | 0 issue |
| `go test ./internal/...` (`services/transactions`) | `ok` dans `commandes`, `commandes/api`, `commandes/marche`, `montants` (dont `expiration_test.go`) |
| `go test ./cmd/place-de-marche -run TestUS031Duree -count=1` | `ok` 15,5 s (8 refus au démarrage, 3 acceptations) |
| `go test ./cmd/place-de-marche -run TestUS031GarantiesDeLaReservation -count=1 -v` (Docker) | `ok` 196,7 s, 13 scénarios PASS ; courses : 11/1 par Transactions, 10/2 en direct, jamais deux succès |
| `go test ./cmd/place-de-marche -run 'TestUS021\|TestUS013\|TestUS009\|TestUS030' -count=1` (non-régression modification, soumission, réservation) | `ok` 306,2 s, aucun échec |
| `npx tsc -b`, `npm run lint`, `npx prettier --check src e2e README.md`, `npm run i18n:check` (`web`) | 0 erreur ; Prettier conforme ; 307 clés alignées fr/en |
| `npx vitest run src/acceptance/US-031 src/acceptance/US-030 src/acceptance/US-021 src/acceptance/US-013` | 13 fichiers, 146 tests verts |
| Commentaires ajoutés (`//`, `/* */`) dans le Go et le TypeScript du diff | Aucun |
| Appelants des symboles partagés modifiés | `annonces.VerifyEditable` (fonction) remplacée par `ArticleState.VerifyEditable` : seul appelant `lockEditable` (`postgres.go:218-224`), mis à jour ; `explainRefusal` renommée `explainSubmitRefusal` : seul appelant `Submit` (`postgres.go:260`) ; `ownedState` (3 colonnes) : appelée par `lockEditable` et `explainSubmitRefusal` seulement ; `articleStateQuery` (2 colonnes, décision d'admin `postgres.go:534`) non touchée ; `commandes.NewService` (horloge ajoutée) : `serve.go:40`, `service_test.go:73`, `expiration_test.go:91` ; `positiveDuration` → `reservationDuration` : plus aucune référence à l'ancien nom ; `OrderStatus` élargi : tous les `Record<OrderStatus, …>` complétés (`OrderStatusChip`, `OrderDeadline`, `OrderSummaryView`), `tsc` vert ; clés i18n supprimées (`orders.title`, `subtitle`, `documentTitle`, `loaded`, `paymentDue`, `paymentDueHint`) : plus aucune référence ; `isDemoArticleReserved` (paramètre `now` optionnel) : appelants du catalogue de démo inchangés et compatibles |

## Points forts

- **D07 Q3 appliquée en profondeur, pas seulement en surface** : la condition « réservé ou vendu » est une seule constante SQL (`postgres.go:32`), reprise par la lecture verrouillée (`lockedStateQuery`, `SELECT … FOR UPDATE`, `:139-140`) **et** par les trois `UPDATE` vendeur via `sellerWritable` (`:34`, `:40`, `:52`, `:62`). Vérifié : en `READ COMMITTED`, le `FOR UPDATE` qui a attendu une réservation relit la version validée de la ligne, et `now()` de la transaction de modification précède l'échéance tirée par la réservation (durée d'au moins 1 s) : la fenêtre de course est fermée dans les deux ordres. La course est prouvée par le banc (24 courses par passe, les deux ordres observés, jamais deux succès).
- **Ordre des refus explicite et testable** : `ArticleState.verify` (`article.go:101-112`) rend « réservé ou vendu » prioritaire sur l'état puis sur la révision, comme le contrat (`Conflit`) ; la propriété est vérifiée avant (`ownedState` filtre par `vendeur_id` → 404), donc un tiers n'apprend jamais qu'un article est réservé. Le problème `article-reserve` (`handlers.go:43-44`) ne porte ni acheteur ni échéance, contrôlé par le banc sur les corps de réponse.
- **Retrait couvert sans route nouvelle**, conformément à la décision : seuls les `UPDATE` vendeur pouvaient sortir un article `publie` du catalogue ; vérifié que `applyReviewQuery` (`:89-93`) et `decideQuery` (`:122-128`) exigent `en_controle` ou `a_verifier`, comme l'affirme le README.
- **Expiration dérivée, simple et non destructive** : `Order.AsOf` (`commande.go:109-114`) ne transforme que `en_attente_paiement`, à l'échéance incluse (`!now.Before`), sans rien écrire ; appliquée sur les deux chemins de lecture (`service.go:49`, `:58`) ; horloge injectée (`Clock`), `time.Now` en production. Les droits du rôle d'exécution (ni `UPDATE` ni `DELETE`) restent suffisants.
- **Borne de configuration utile** : `minimumReservationDuration` (`config.go:31`, `:114-115`) refuse au démarrage une durée sous 1 s, qui rendrait la réservation inobservable ; message qui nomme la variable.
- **Front propre** : statut et textes pilotés par tables indexées par `OrderStatus` (`OrderDeadline.tsx`, `OrderStatusChip.tsx`, `OrderSummaryView.tsx`, `useOrderConfirmationText.ts`) : un statut futur sera une erreur de type, pas un oubli silencieux ; `conflictKindOf` (`articleFailure.ts`) remplace une condition booléenne par une table ; `useSubmitForControl` passe en `switch` exhaustif ; sur 409 `article-reserve`, aucun rechargement, saisie conservée, message dans la zone annoncée. Démo cohérente : la réservation de démonstration expire avec la commande (`demoReservations.ts`, `demoOrdersClient.ts`).
- **Tests de concurrence de bonne facture** : requêtes préparées à l'avance, départ synchronisé sur un canal (`enParalleleDecale`), aucun `t.Fatal` hors de la goroutine du test, chaque tour vérifié en réponse **et** en base, décalage progressif du vendeur pour obtenir les deux ordres de la course ; comparaison octet pour octet de la ligne `article` (`to_jsonb`) après refus.

## Issues

### Bloquantes

Aucune.

### Majeures

Aucune.

### Mineures

1. `web/src/i18n/locales/fr.ts:177-178` et `en.ts` (`seller.reserved`) — le message affiché pour tout 409 `urn:collector:probleme:article-reserve` dit « Cet article est réservé par un acheteur : il ne peut pas être modifié tant que la réservation court. » Or le contrat (`place-de-marche/v1`, réponse `Conflit`) et le README donnent ce même type pour un article **vendu**, et le front l'affiche aussi sur la **soumission** (`useSubmitForControl.ts:38-40`). Aujourd'hui sans effet (`vendu` jamais vrai, un article soumettable n'est jamais réservé), mais le texte deviendra faux avec le paiement, sans qu'aucun test ne le signale. Correction : formuler un message exact pour les deux cas (« Cet article est réservé ou vendu : il ne peut être ni modifié ni soumis. » / EN), ou distinguer les deux motifs le jour où le back les distingue. Dev : front.
2. `services/transactions/internal/commandes/expiration_test.go:126` et `:153` — `premiere, _ := b.service.Place(…)` ignore l'erreur de la première réservation. Si elle échouait, `premiere` serait vide et les assertions suivantes échoueraient sur un message trompeur (« nouvelle commande … sur l'ancienne commande ») au lieu de la vraie cause. Correction : vérifier `err` (et `Created`) comme le fait déjà `TestUS031AC4CommandeLueExpireeApresLEcheance` (`:107-110`). Testeur.
3. `services/place-de-marche/cmd/place-de-marche/reservation_garanties_test.go:15`, `:361`, `:676` — tout le banc tourne avec une échéance de 6 s, y compris les scénarios qui exigent que la réservation soit **encore active** (AC2 : rafale, attente de 1,5 s, relance, rafale directe ; AC5 : six tentatives du vendeur puis relectures). Mesuré en revue : 2,2 s par tour d'AC2, environ 1,5 s d'étapes pour AC5, donc 3,5 à 4,5 s de marge. Sur un exécuteur de CI lent, un dépassement ferait échouer le test **comme une anomalie du produit** (AC5 : 200 au lieu de 409, article réellement modifié), sans le dire. Correction, au choix : après les étapes sensibles, vérifier `time.Now()` avant `echeance` et échouer par un message explicite (« banc trop lent, échéance dépassée avant la fin du scénario ») ; ou démarrer ces scénarios sur une échéance plus longue et ne garder 6 s que pour les scénarios d'expiration. Testeur.

### Suggestions

1. `services/place-de-marche/internal/annonces/stockage/catalogue.go:14` et `postgres.go:32` — dans le même paquet `stockage`, la règle « réservé maintenant » est écrite deux fois (`coalesce(a.reserve_jusqu_a > now(), false)` et `coalesce(reserve_jusqu_a > now(), false)` dans `reservedOrSold`), une troisième fois dans `reservations/stockage/postgres.go:22,25,33`. C'est une règle de D07 : en faire une constante unique dans `annonces/stockage` (`reservedNow`, sans alias, et `a.` ajouté par le `FROM … AS a` ou par une seconde constante) éviterait qu'une évolution (tolérance, `vendu` au catalogue) ne touche qu'un des deux usages. Dev : back.
2. `services/place-de-marche/cmd/place-de-marche/reservation_duree_config_test.go:79-92` — `TestUS031DureeDeReservationDUneSecondeAccepteeAuDemarrage` prouve seulement que l'erreur, s'il y en a une, ne nomme pas la variable, et coûte 5 s par cas en lançant `serve`. `config.LoadServer(lookup)` est une fonction pure : un test de table dans `internal/config` (1 s et 1,5 s acceptées, 999 ms refusée, `min(durée, plafond)` effectivement appliqué) serait plus fort, instantané et compté dans la couverture du paquet. Testeur ou dev back.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 0 |
| Mineure | 3 |
| Suggestion | 2 |

**Fusion possible.** Les cinq AC sont couverts et prouvés contre les vrais services, rejoués en revue : concurrence entre acheteurs (AC1), rafale du même acheteur (AC2), une réservation active par acheteur (AC3), expiration réelle à 6 s avec nouvelle réservation et ancienne commande `expiree` (AC4), modification et soumission refusées en 409 `article-reserve`, ligne de l'article inchangée, course vendeur contre acheteurs fermée dans les deux ordres (AC5). L'atomicité est réelle (verrou de ligne et condition dans chaque `UPDATE`, D07 Q3), l'ordre des refus suit le contrat et rien n'est exposé (ni acheteur, ni échéance, ni existence pour un tiers). Le remplacement de `VerifyEditable` ne laisse aucun code mort, aucun appelant oublié ; format, lint, types et i18n sont propres, sans commentaire. L'écart d'horloge (statut `expiree` lu avec l'horloge de Transactions, échéance tirée par la base) est la décision de Martin ; il est documenté dans `services/transactions/README.md` (« Expiration »), sans effet au-delà de quelques millisecondes. Répartition : front pour la mineure 1 ; testeur pour les mineures 2 et 3 et la suggestion 2 ; back pour la suggestion 1. Aucune correction devops.

## Retours traités (2026-11-16)

| Retour | Traitement |
|---|---|
| Mineure 1 — message vendeur faux pour un article vendu ou à la soumission | `seller.reserved` reformulé FR/EN : réservé ou vendu, ni modifiable ni soumis |
| Mineure 2 — erreur de `Place` ignorée | Helper `premiereCommande` : `err`, `Created` et statut vérifiés |
| Mineure 3 — échéance de 6 s pour tout le banc | Garde `exigerAvantLEcheance` après chaque réponse des scénarios où la réservation doit rester active ; message explicite si l'échéance est franchie |
| Suggestion 1 — règle « réservé maintenant » répétée | Constante `internal/disponibilite.ReservedNow`, SQL inchangé |
| Suggestion 2 — test « une seconde acceptée » lent et faible | Test de table instantané sur `config.LoadServer` (14 cas) ; 2 refus de bout en bout conservés |

Verdict après corrections : **fusion possible**.
