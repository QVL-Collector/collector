# US-015 — Revue de code (partie back, instrumentation)

| Champ | Valeur |
|---|---|
| US | [US-015 — Métriques et tableaux de bord de la plateforme](../../../backlog/US-015-metriques-tableaux-de-bord.md) |
| Branche | `feat/US-015-metriques-tableaux-de-bord` (HEAD `47ad500`) ; changements non commités revus par `git diff HEAD` et fichiers non suivis, limités à `libs/` et `services/` |
| Date | 2026-11-25 |
| Périmètre | `libs/platform` : nouveau paquet `metrics` (`exporter.go`, `instrument.go`, `middleware.go`, README, tests), `broker/metrics.go`, `outbox/metrics.go`, branchements dans `broker/consumer.go`, `broker/publisher.go`, `outbox/relay.go`, `go.mod`/`go.sum` ; `services/{place-de-marche,controle,interactions,transactions}` : `serve`, `config.go` (`ADRESSE_METRIQUES`), Dockerfile (`EXPOSE 9091`), README, `go.mod`/`go.sum`, bancs de test ([rapport de tests](../tests/US-015-rapport-tests.md)) |
| Références | [D09](../../decisions/D09-observabilite.md) (SDK OpenTelemetry Go, exportateur Prometheus sans collecteur, port de métriques distinct, route = gabarit, limites d'histogramme autour de 0,5 s), D04 ; ENF-14, ENF-15, ES-31 |
| Hors revue | AC1 « relevé par Prometheus », AC2, AC3, AC4, AC5 infra (`ClusterIP`, ALB, politique réseau du port 9091, Grafana) : DevOps |
| Verdict | **Fusion possible après correction de la majeure** (0 bloquante, 1 majeure, 3 mineures, 1 suggestion) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l .`, `go vet ./...` (5 modules) | 0 fichier, 0 remarque |
| `golangci-lint run --config .golangci.yml ./...` (5 modules) | 0 issue |
| `go mod tidy -diff` (5 modules) | modules propres |
| `go test ./metrics/ ./broker/ ./outbox/ -count=1` (`libs/platform`) | `ok` (13 tests de `metrics`) |
| `go test ./... -count=1` (`services/controle`, `services/transactions`) | `ok`, 0 échec |
| `go build ./...` (`services/place-de-marche`, `services/interactions`) | `ok` ; bancs Docker longs non rejoués (1 413 s), résultats du rapport de tests repris |
| `go list -m` des modules OTel par service | voir majeure 1 |
| Commentaires ajoutés (`//`, `/* */`) | Aucun |
| Appelants des symboles partagés modifiés | `Publisher.PublishBatch` : comportement et retour inchangés (comptage ajouté avant le retour `UnroutableError`) ; `Consumer.settle`, `discard`, `process` : seul ajout d'un comptage après `Ack`/`Nack` réussi ; `Relay.cycle` : comptage ajouté, retour inchangé ; `config.Server` des 4 services : champ `MetricsAddress` ajouté, aucun littéral de `Server` hors `LoadServer` ; tous les lancements de `serve` des bancs qui posaient `ADRESSE_GESTION` posent aussi `ADRESSE_METRIQUES` (même liste de fichiers) ; aucun manifeste de déploiement dans le dépôt ne fixe encore les ports (DevOps) |
| Index de l'outbox pour la jauge | `outbox_en_attente_idx ON outbox (cree_le, id) WHERE publie_le IS NULL` (`place-de-marche/internal/migrations/00003_soumettre_au_controle.sql:33`) couvre `count(*)` et `min(cree_le)` de `backlogQuery` |
| Délégation du fournisseur global OTel | `otel@v1.47.0/internal/global/state.go:152` : `delegateMeterOnce.Do` — les instruments créés avant le premier `SetMeterProvider` restent liés au premier fournisseur installé (explique l'observation du rapport de tests) |

## Points forts

- **Cardinalité réellement bornée** : `http.route` est le gabarit du `ServeMux` sans la méthode (`metrics/middleware.go:59-68`), `inconnue` sans gabarit, méthode hors liste → `_OTHER` (`:52-57`), code HTTP entier. `echange` vient des constantes du code écrites dans l'outbox (`outbox/event.go:31`, `controle/internal/canal/responder.go:74`), `file` de la configuration, `motif` des deux constantes `discardCause` (`broker/consumer.go:21-22`). Aucun identifiant, sujet, corrélation ni chemin brut en étiquette ; prouvé en unitaire (50 UUID → une série) et de bout en bout sur les 4 services.
- **Placement du middleware juste** : `logging.Middleware` → `metrics.Middleware` → `httpserver.Recover` (`place-de-marche/.../serve.go:119-121`, idem dans les 3 autres services) : le gabarit est lu après le `ServeMux` (aucun intergiciel intermédiaire ne copie la requête, vérifié dans `httpserver/middleware.go`), et une panique est comptée en 500. Factorisation `instrumented` pour les deux API de la Place de marché.
- **Port distinct et fermé** : serveur HTTP séparé (`httpserver.New`, mêmes délais que les autres), `GET /metrics` seul, 404 partout ailleurs (`metrics/exporter.go:43-48`), registre Prometheus propre (pas de `DefaultRegisterer`, donc ni métriques Go ni processus exposées par accident). Conforme à D09 « Port de métriques distinct ».
- **Jauge de l'outbox bien pensée** : lue en base à chaque relevé, donc juste même relais bloqué ; délai de 2 s (`outbox/metrics.go:18`), indispensable car l'exportateur appelle les rappels sans échéance ; requête servie par un index partiel existant ; jauge (et non `UpDownCounter`) parce que tous les réplicas lisent la même table, et le README impose `max`, jamais `sum`, plus `absent()` pour la base injoignable. Une erreur de rappel n'empêche pas l'exposition des autres séries (journal Warn `metriques.echec`).
- **Nommage OTel respecté** : noms en points, unités UCUM (`s`, `{message}`, `{evenement}`), pas de `_total` dans le nom de l'instrument ; l'exportateur produit `_seconds`, `_total` comme documenté. Limites d'histogramme resserrées autour de 0,5 s, conformément au point d'attention de D09.
- **Documentation à jour** : catalogue, règles et requêtes types dans `libs/platform/metrics/README.md` ; port 9091 et `ADRESSE_METRIQUES` dans les 4 README de service ; mention de `canal_messages_ecartes_total` dans la section « Mise à l'écart » de la Place de marché.

## Issues

### Bloquantes

Aucune.

### Majeures

1. `libs/platform/go.mod:15-19` — versions OpenTelemetry désalignées. L'API est en `v1.47.0` (`otel`, `otel/metric`, imposés par `github.com/pressly/goose/v3 v3.29.0`, vérifié par `go mod graph`), mais le SDK est épinglé en `v1.38.0` (`otel/sdk`, `otel/sdk/metric`) avec l'exportateur `v0.60.0`, qui est la version appariée à `v1.38`. Résultat constaté par `go list -m` : Contrôle et Transactions livrent API 1.47 + SDK 1.38 ; Place de marché et Interactions livrent API 1.47 + **SDK 1.47** (tiré par `otelhttp v0.72.0` de leurs dépendances de test) + exportateur 0.60. La bibliothèque est donc testée sur une combinaison (`libs/platform`, SDK 1.38) et livrée sur une autre dans deux services, et l'exportateur 0.60 n'est pas la version publiée avec le SDK 1.47. Les tests passent aujourd'hui ; le risque est une divergence silencieuse de comportement de l'exportateur ou du SDK selon le service. Épingler 1.38 pour s'aligner sur US-014 est par ailleurs impossible : goose impose déjà l'API 1.47 à tout module qui dépend de `libs/platform`. Correction : `go get go.opentelemetry.io/otel/exporters/prometheus@v0.69.0` dans `libs/platform` (vérifié : son `go.mod` exige `otel`, `sdk`, `sdk/metric` en `v1.47.0`, `go 1.26.0` comme le dépôt), puis `go mod tidy` dans les 4 services, pour une seule combinaison API/SDK/exportateur 1.47 partout ; signaler à US-014 de viser `v1.47.0` (MVS la retiendra de toute façon). Dev : back.

### Mineures

1. `libs/platform/broker/metrics.go:29` et `libs/platform/outbox/metrics.go:25` — l'observation du rapport de tests (second `serve` dans le même processus sans `canal_messages_*`) est confirmée et expliquée : ces instruments sont créés à l'initialisation du paquet, sur le fournisseur global par défaut, qui ne délègue qu'**une fois** au premier `SetMeterProvider` (`otel/internal/global/state.go:152`). L'histogramme HTTP et les jauges de l'outbox, créés après `Install`, suivent au contraire le fournisseur courant : le comportement diffère selon l'instrument. Impact production nul (un `serve` par pod) ; à ne pas « corriger » par un garde qui refuserait un second `Install` : les bancs existants lancent `serve` plusieurs fois en processus. Correction : documenter la contrainte dans `libs/platform/metrics/README.md` (§ Branchement, à côté de la ligne 9) : « un seul `Install` par processus ; les instruments du canal et `outbox_relances_total` restent liés au premier fournisseur installé : un banc qui mesure le canal lance le service en processus séparé ». Dev : back.
2. `libs/platform/metrics/middleware.go:19`, `:59-68`, `:70-91` — `statusRecorder` (avec `WriteHeader`, `Write`, `Unwrap`) et la constante `unknownRoute = "inconnue"` sont la copie exacte de `libs/platform/logging/middleware.go:14`, `:61-66`, `:68-89` ; seule `routeOf` diffère (gabarit sans méthode côté métriques, avec méthode côté journaux). Deux recorders empilés sur chaque requête et deux définitions de « route inconnue » qui peuvent diverger. Correction : extraire dans `httpserver` un `StatusRecorder` exporté (avec `Status()`) et `RoutePattern(r)` / la constante de route inconnue, utilisés par les deux middlewares. À faire **après la fusion d'US-014**, qui retouche `logging/middleware.go` (corrélation, `trace_id`), pour éviter un conflit ; le premier des deux à fusionner en second porte la factorisation. Dev : back.
3. `libs/platform/metrics/middleware.go:37-50` — D09 (§ Développement, ligne 279) prévoyait « `otelhttp` sur le routeur » ; le middleware maison est un choix défendable (contrôle exact des étiquettes, `otelhttp` ajoutant `server.address`, `network.protocol.*`, `url.scheme`), mais l'écart n'est tracé ni dans D09 ni dans le README de `metrics`. Point de vigilance pour US-014 : si `otelhttp` y est ajouté pour la propagation des traces, il enregistrera lui aussi `http.server.request.duration` sur le fournisseur global (portée `otelhttp`) et doublera les séries des tableaux de bord. Correction : une ligne dans `libs/platform/metrics/README.md` (« middleware maison plutôt qu'`otelhttp` : étiquettes bornées à méthode, gabarit, code ; si `otelhttp` est posé pour les traces, lui passer `otelhttp.WithMeterProvider(noop.NewMeterProvider())` ») et une note dans D09. Dev : back.

### Suggestions

1. `services/interactions/cmd/interactions/metriques_us015_test.go:62-113` duplique l'essentiel de `lancerInteractions` (`banc_test.go:441-490`, mêmes variables d'environnement, même attente de `/disponibilite`), et `metriques_releve_test.go` est copié entre Interactions et la Place de marché (174 et 190 lignes). Les paquets `main` distincts justifient la seconde copie ; la première pourrait partager la construction des variables (`variablesInteractions(bases, canal, jetons, adresses)`) entre lancement en processus et en processus séparé. Testeur.

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 1 |
| Mineure | 3 |
| Suggestion | 1 |

**Fusion possible après correction de la majeure 1.** AC1 côté services est couvert : débit, latence p95 (histogramme resserré autour de 0,5 s), erreurs par route et par code, attente de l'outbox et flux du canal sont exposés par les 4 services, sur un port distinct ; AC5 côté service : le port 9091 ne sert que `GET /metrics`, les ports publics, interne et de santé n'exposent rien. Aucune donnée personnelle ni identifiant en étiquette, cardinalité bornée par construction, coût de la jauge lu en base maîtrisé (index partiel, délai de 2 s). Aucun appelant cassé, aucun code mort, format, vet et lint propres, sans commentaire. La seule correction attendue est l'alignement des versions OpenTelemetry sur 1.47 ; les mineures sont de la documentation et une factorisation à coordonner avec US-014. Répartition : back pour la majeure et les mineures ; testeur pour la suggestion. Côté DevOps (hors revue) : politique réseau du port 9091 réservée à Prometheus et `ServiceMonitor` restent à livrer.

## Retours traités

| # | Sévérité | Traitement |
|---|---|---|
| 1 | Majeure | Versions alignées : API, SDK et SDK métriques OpenTelemetry en v1.47.0, exportateur Prometheus en v0.69.0, `go mod tidy` dans les 5 modules (`go list -m` identique partout). |
| m1 | Mineure | README de `metrics` : « un seul `Install` par processus », cause (`delegateMeterOnce`), impact nul en production. |
| m2 | Mineure | Doublon de `statusRecorder` noté en dette dans le README, regroupement dans `httpserver` après la fusion d'US-014. |
| m3 | Mineure | Écart à D09 tracé (README et D09, 2026-11-25) : middleware maison, `otelhttp` avec fournisseur de mesures noop s'il est ajouté pour les traces. |
| S1 | Suggestion | Reportée (factorisation du lancement d'Interactions dans les tests). |

Vérifications après corrections : `go mod tidy`, `go build`, `golangci-lint` sur les 5 modules, `go test` de `libs/platform/metrics` verts.
