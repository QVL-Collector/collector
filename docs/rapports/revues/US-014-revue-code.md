# US-014 — Revue de code (partie back)

| Champ | Valeur |
|---|---|
| US | [US-014 — Journaux centralisés et corrélés d'un bout à l'autre](../../../backlog/US-014-journaux-centralises.md) |
| Branche | `feat/US-014-journaux-centralises` (HEAD `4c909e3`, égal à la base de `main` au moment de la branche) ; changements non commités revus par `git diff HEAD -- libs services` et fichiers non suivis. Le `git diff main` brut montre aussi l'infra et les outils d'US-004 : ils viennent de `main`, qui a avancé depuis, et sont hors de cette revue |
| Date | 2026-11-23 |
| Périmètre | `libs/platform/logging` (`correlation.go`, `trace.go`, `logger.go`, `mask.go`, `middleware.go`, README), `libs/platform/broker` (`consumer.go`, `publisher.go`), `libs/platform/outbox` (`event.go`, `relay.go`) ; `services/place-de-marche` : stockage (`postgres.go`), `controle/{requests,results,safeguard}.go`, `diffusion/decisions.go`, `review.go`, `api/handlers.go`, commandes `suspendre`/`retablir`/`rejouer`, migrations 00010 et 00011, README ; `services/controle/internal/canal/responder.go` ; `services/transactions/internal/commandes/marche/client.go` ; README de Contrôle, Interactions, Transactions ; tests ajoutés ([rapport de tests](../tests/US-014-rapport-tests.md)) |
| AC revus | AC1, AC4, AC5 (couche source), AC6 (côté service). AC2, AC3, seconde couche d'AC5 et collecte d'AC6 (Fluent Bit, CloudWatch) : reportés, hors revue |
| Références | [D09](../../decisions/D09-observabilite.md), [D04](../../decisions/D04-stack.md), [D05](../../decisions/D05-communication-interservice.md) ; ES-06, ES-23 ; décisions de Martin du 2026-11-20 : corrélation et `traceparent` de la soumission gardés sur l'article (migration 00011) et repris par le filet de relance ; audit de `suspendre`/`retablir` avec l'auteur constant `exploitation` |
| Verdict | **Fusion après correction** (0 bloquante, 1 majeure, 1 mineure, 3 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l .`, `go vet ./...` (`libs/platform`, `services/controle`, `services/place-de-marche`, `services/interactions`, `services/transactions`) | 0 fichier, 0 remarque |
| `golangci-lint run --config .golangci.yml ./...` (mêmes modules) | 0 issue partout |
| `go test ./... -count=1` (`libs/platform`) | `ok` dans 9 paquets, dont `logging` et `outbox` |
| `go test ./... -count=1` (`services/controle`, `services/transactions`) | `ok` |
| `go test ./cmd/place-de-marche/ -run "TestUS024FiletDelaiCroissant\|TestUS014Relance\|TestUS014Migrations" -count=1` (Docker) | `ok`, 58 s |
| Commentaires ajoutés (`//`, `/* */`) dans le Go du diff et dans les tests ajoutés | Aucun |
| Appelants des symboles partagés modifiés | `outbox.NewEvent` (ajout de `ctx`) : seuls appelants `controle/requests.go:31` et `diffusion/decisions.go:31`, à jour. `Responder.resultMessage` (ajout de `ctx`) : `responder.go:45` et `paire_test.go:70`, à jour. `broker.Message.TraceParent` et `outbox.Event.TraceParent` : champs ajoutés, aucune construction positionnelle ailleurs. Requêtes de l'outbox (`insertEventQuery`, `selectPendingQuery`) : la seule table `outbox` est celle de la Place de marché (00003), seul module qui l'utilise (`serve.go:93`, `:119`, `rejeu.go:32`) ; colonne ajoutée par 00010. `CorrelationHeader`, `WithCorrelationID`, `CorrelationID`, `CorrelationIDOrNew` : déplacés de `middleware.go` à `correlation.go`, même paquet, mêmes signatures ; appelants (`controle/responder.go:40`, `interactions/.../decisions.go:37`, `place-de-marche/.../results.go:37`, `requests.go:23`, `decisions.go:23`, `transactions/.../client.go`) compilent et passent le lint. `annonces.ResentReview` : champs ajoutés, seul producteur `scanResentReview`, seul consommateur `Safeguard.logResent`. Champ `etat` du journal `controle.resultat_applique` renommé `issue` : aucune requête, alerte ni doc ne lisait `etat` (recherche dans `*.md`, `*.yaml`, `*.json`, `*.sh`) |
| Droits des rôles sur les colonnes ajoutées | `00001` pose `ALTER DEFAULT PRIVILEGES … GRANT SELECT, INSERT, UPDATE ON TABLES TO marche_execution` : droits au niveau table, les colonnes `outbox.traceparent`, `article.demande_controle_correlation_id` et `article.demande_controle_traceparent` sont couvertes sans `GRANT` de colonne. Aucune migration du dépôt n'a de section `Down` : 00010 et 00011 suivent la convention |

## Points forts

- **Chaîne de propagation complète et sans trou** : requête HTTP (`middleware.go:14-16`, corrélation UUID reprise ou tirée, `traceparent` repris ou tiré) → `requestReview` fige la corrélation dans le contexte avant de la mémoriser et de l'encoder (`postgres.go:284`), donc l'article, l'enveloppe et la ligne d'outbox portent la même valeur → colonne `outbox.traceparent` (`event.go:30-41`) → relais → en-tête AMQP (`publisher.go:144`, `traceHeaders`) → `Consumer.process` reprend le contexte avant tout traitement, y compris pour la mise à l'écart (`consumer.go:111`, `:136-139`) → Contrôle, dont le résultat repart avec le `traceparent` du contexte (`responder.go:73-80`) → application du résultat, dont la `publication.decidee` repasse par `outbox.NewEvent(ctx, …)` (`decisions.go:31`) → Interactions. Appel sortant Transactions → Place de marché par `logging.Propagate` (`client.go:60`).
- **Aucun champ d'origine forgeable** : la corrélation reçue n'est gardée que si c'est un UUID (`correlation.go:51-57`), le `traceparent` que s'il suit strictement le format W3C en minuscules, identifiants non nuls (`trace.go:19`, `:51-57`) ; rien d'autre n'est repris des en-têtes ni des messages. `service` et `audit` ne sont posés que par le service (`logger.go:36-39`) ; le README de `logging` dit clairement qu'ils ne prouvent rien et que l'origine viendra des métadonnées Kubernetes (AC6). Les deux colonnes de 00010 et 00011 ont une contrainte `CHECK` sur le même format.
- **Filet de relance atomique, dans le bon ordre** (point demandé, vérifié) : une seule transaction `pgx.BeginFunc` (`postgres.go:378-397`) enchaîne (1) le `UPDATE … RETURNING` de `resendOverdueQuery` (`:72-90`), qui pose `demande_controle_le = now()` et `relances + 1` sous `FOR UPDATE SKIP LOCKED`, (2) pour chaque article, `resendReview` (`:399-407`) → `requestReview` → `rememberReviewContext` (`:303-310`, même `tx`) puis `outbox.Enqueue` (même `tx`). Le journal `controle.demande_relancee` n'est écrit qu'après le commit (`safeguard.go:63-68`). Il n'existe donc aucun état validé où la demande réémise est en outbox et le compteur pas encore incrémenté. L'échec observé par le testeur vient du banc, pas du produit : voir la mineure 1.
- **Masquage à la source sérieux** : correspondance par fragment de clé normalisée (`mask.go:180-185`), règles sur toute valeur texte, erreur et `fmt.Stringer` (`:187-200`), `http.Header` et `url.Values` transformés en groupe dont chaque entrée repasse dans `ReplaceAttr` (comportement de `log/slog` pour un groupe rendu par `ReplaceAttr`, ce qui masque `Authorization` par sa clé). Faux positifs vérifiés : sur toutes les clés journalisées du dépôt (`acheteur_id`, `adresse`, `cause`, `entetes`, `erreur`, `jeton`, `motif`…), seule `jeton` correspond à un fragment, et elle était déjà masquée avant. Les seuls `slog.Any` de production portent des erreurs (`erreur`, `cause`) : l'ajout du cas `fmt.Stringer` ne change le rendu d'aucun journal existant. Coût : cinq expressions régulières par valeur texte, négligeable au débit du POC.
- **Audit homogène** : `logging.Audit` (`logger.go:36-39`) remplace trois écritures à la main ; `controle.resultat_applique` n'est émis que pour un résultat effectivement appliqué (`results.go:64-85`), avec article, révision, issue, motif, version des règles, horodatage `time` (AC4) ; décision de Martin appliquée pour `comptes.suspension` (`comptes.go:20-45`, `auteur` constant, `action` dérivée des noms de commande sans doublon de chaîne).
- **Pas de code mort laissé** : `correlationIDOf` remplacé par `wellFormedOrNew`, `uuid` retiré de `rejeu.go` avec le `correlation_id` écrit à la main (désormais posé par `StartOperation`), ancienne table `sensitiveKeys` supprimée. Les README des quatre services et le nouveau README de `logging` décrivent exactement le code revu (points d'entrée, colonnes, événements d'audit, limites du masquage).

## Issues

### Bloquantes

Aucune.

### Majeures

1. `libs/platform/logging/README.md` (section « Corrélation et trace », paragraphe « Choix (2026-11-20, US-014) : pas de SDK OpenTelemetry pour ces identifiants ») et `libs/platform/logging/trace.go` — l'implémentation écrit son propre porteur W3C, sans SDK OpenTelemetry. C'est un choix défendable (aucune trace exportée, D09 option C), mais il contredit des décisions marquées « Décidée » que le diff ne met pas à jour : [D09](../../decisions/D09-observabilite.md) l. 28 et l. 110 (« D04 a retenu le SDK OpenTelemetry Go pour les traces et les métriques ; D09 ne le rouvre pas »), [architecture technique](../../architecture/architecture-technique.md) l. 423 (`libs/platform` pour … OpenTelemetry), l. 432 et l. 490 (« SDK OpenTelemetry Go dans `libs/platform` »). Le choix n'est consigné que dans le README d'un module, sans arbitrage de Martin tracé, alors que l'architecture est tranchée par lui. Conséquence concrète : US-015 (métriques) partira de la doc, introduira le SDK pour les métriques, et trouvera un second générateur de `trace_id`/`span_id` à côté. Correction : faire trancher Martin. S'il valide l'écart, l'inscrire dans D09 (section « Corrélation », avec la condition de passage à OpenTelemetry) et corriger les trois lignes de l'architecture technique ; le README de `logging` renvoie alors à D09 au lieu de porter seul la décision. S'il ne le valide pas, remplacer `trace.go` par le propagateur `propagation.TraceContext` d'OpenTelemetry (sans exportateur). Destinataires : orchestrateur (arbitrage de Martin), puis dev back pour la doc ou le code.

### Mineures

1. `services/place-de-marche/cmd/place-de-marche/filet_de_relance_test.go:168-169` et `:184-185` (test d'US-024, hors diff, signalé parce que le testeur l'a vu échouer) — `dater(…, 2*time.Minute-time.Second)` puis `exigerAucuneRelance`, qui dort `attenteSansRelance` = 1 s (`:20`). La marge avant l'échéance (1 s) est égale à la durée d'attente, alors que le filet du test balaie toutes les 50 ms (`:19`) : l'article devient en retard **avant** la fin de l'attente, d'autant plus tôt que la machine est chargée. Si la relance tombe après les relectures du sous-test « pas de relance avant 2 min », ce sous-test passe ; le sous-test suivant appelle `dater(…, 0, …)` (`:175`), qui **remet `relances` à 0** par-dessus la relance déjà faite, puis `exigerRelance` est satisfait immédiatement par le journal existant : 2 demandes en outbox, journal écrit, `relances` = 0 à la ligne 179. C'est exactement l'échec rapporté ; le produit est atomique (voir Points forts). Correction : donner une marge nettement supérieure à l'attente (par exemple `2*time.Minute-10*time.Second` et `4*time.Minute-10*time.Second`), ou attendre moins longtemps que la marge. Destinataire : testeur.

### Suggestions

1. `libs/platform/logging/middleware.go:14-16` — le middleware recompose à la main ce que fait `Resume` (`correlation.go:34-36`) : `wellFormedOrNew` + `WithCorrelationID` + `ContinueTrace`. Écrire `routed := r.WithContext(Resume(r.Context(), r.Header.Get(CorrelationHeader), r.Header.Get(TraceParentHeader)))`, puis `id, _ := CorrelationID(routed.Context())` pour l'en-tête de réponse : une seule règle de reprise pour HTTP, AMQP et le filet. Dev back.
2. `services/place-de-marche/internal/annonces/controle/safeguard.go:77` — `logResent` appelle `logging.Resume` sur le `traceparent` mémorisé, qui tire un nouveau `span_id` : la ligne `controle.demande_relancee` a le bon `trace_id`, mais un `span_id` différent de celui de la demande réémise (`resendReview`, `postgres.go:399-407`). Sans effet sur la recherche par `correlation_id` ou `trace_id` ; pour une correspondance exacte, une fonction `logging.Restore` (corrélation et trace reprises telles quelles, sans nouveau span) servirait ce cas de reprise d'un contexte déjà ouvert. Dev back.
3. `services/place-de-marche/internal/annonces/stockage/postgres.go:92-95` et `:303-310` — chaque soumission, recontrôle ou relance ajoute un second `UPDATE article` sur la ligne que la requête précédente de la même transaction vient de modifier. Poser les deux colonnes dans `reviewRequestedAssignments` (`:48`, paramètres en plus dans `submitQuery`, `reviseQuery`) et dans la CTE `relance` de `resendOverdueQuery` éviterait une version de ligne et un aller-retour par demande. Non bloquant au volume du POC ; à garder si la lisibilité de la fonction dédiée est préférée. Dev back.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 1 |
| Mineure | 1 |
| Suggestion | 3 |

**Fusion après correction de la majeure.** Le code répond aux AC du périmètre back : `service`, `correlation_id` et `trace_id` sur chaque journal lié à une soumission, dans la Place de marché, le Contrôle et Interactions, corrélation recopiée dans l'enveloppe et la propriété AMQP de chaque message, `traceparent` dans l'outbox puis l'en-tête AMQP (AC1) ; audit horodaté et complet des résultats appliqués, des décisions d'administration et des suspensions (AC4) ; masquage à la source étendu, sans faux positif sur les clés du dépôt (AC5, première couche) ; aucun champ d'origine accepté du contenu reçu (AC6, côté service). Le filet de relance est atomique : compteur, contexte mémorisé et demande réémise sont écrits dans une seule transaction, dans cet ordre, et journalisés après le commit ; l'échec vu sous charge est une course du test d'US-024, expliquée ligne à ligne. Aucune régression trouvée sur les appelants des symboles modifiés ; format, vet et lint propres, sans commentaire. La seule correction attendue est documentaire et de gouvernance : l'écart avec le SDK OpenTelemetry décidé par D04/D09 doit être arbitré par Martin et tracé. Répartition : orchestrateur puis dev back pour la majeure 1 ; testeur pour la mineure 1 ; dev back pour les suggestions. Aucune correction devops.

## Retours traités

| # | Sévérité | Traitement |
|---|---|---|
| 1 | Majeure | Décision de Martin (2026-11-25) : aligner sur D09. `trace.go` utilise l'API trace et le propagateur W3C d'OpenTelemetry (SDK v1.47.0, sans exportateur, `ParentBased(NeverSample)`), validation stricte W3C conservée, API publiques inchangées. |
| 2 | Mineure (testeur) | Marge avant échéance du test `TestUS024FiletDelaiCroissant…` corrigée dans le lot des tests instables (PR #55). |
| S1 | Suggestion | `middleware.go` passe par `Resume`. |
| S2 | Suggestion | `logging.Restore` : le journal de relance porte le même `trace_id` et `span_id` que la demande réémise. |
| S3 | Suggestion | Corrélation et `traceparent` posés dans le même `UPDATE` que la soumission et le recontrôle ; un `UPDATE` séparé ne subsiste que pour un article soumis avant la migration 00011, à sa première relance. |

Vérifications après corrections : `go mod tidy`, `gofmt`, `go vet`, `golangci-lint` sur les 5 modules ; `go test` de `libs/platform`, Contrôle et des bancs US-014 et US-024 de la Place de marché verts.
