# Métriques des services (US-015)

SDK OpenTelemetry Go et son exportateur Prometheus, sans collecteur ([D09](../../../docs/decisions/D09-observabilite.md), D04). Chaque service expose `GET /metrics` sur `ADRESSE_METRIQUES` (`:9091`), serveur HTTP distinct de l'API et de la santé : jamais routé par l'ALB, ouvert au seul Prometheus par politique réseau (MO5). Toute autre route y répond 404.

## Branchement

- `metrics.Install(serviceName, logger)` au début de `serve` : fournisseur de mesures global, registre Prometheus propre au service, échecs de mesure journalisés en Warn `event=metriques.echec`. Retourne le gestionnaire à servir sur `ADRESSE_METRIQUES`.
- `metrics.Middleware` entre `logging.Middleware` et `httpserver.Recover` : à cette place, il voit le modèle de route posé par le `ServeMux` et compte en 500 une panique interceptée.
- Les instruments du canal (`broker`, `outbox`) passent par le fournisseur global : sans `Install` (tests, sous-commandes), ils ne mesurent rien.
- **Un seul `Install` par processus.** Les compteurs `canal_messages_*` et `outbox_relances_total` sont créés à l'initialisation du paquet, sur le fournisseur global par défaut, qui ne délègue qu'une fois, au premier `SetMeterProvider` (`delegateMeterOnce` d'OpenTelemetry) : après un second `Install`, ils restent liés au premier fournisseur, alors que l'histogramme HTTP et les jauges de l'outbox suivent le fournisseur courant. Impact nul en production (un `serve` par pod). Un banc qui mesure le canal lance le service dans un processus séparé ; ne pas ajouter de garde qui refuserait un second `Install`, car les bancs lancent `serve` plusieurs fois dans le même processus.
- **Middleware maison plutôt qu'`otelhttp`** (écart à D09, 2026-11-25) : les étiquettes restent bornées à la méthode, au modèle de route et au code de réponse, sans `server.address`, `network.protocol.*` ni `url.scheme`, et le modèle de route est lu de façon sûre après le `ServeMux`. Si `otelhttp` est posé pour propager les traces, lui passer `otelhttp.WithMeterProvider(noop.NewMeterProvider())`, faute de quoi il enregistre lui aussi `http.server.request.duration` et double les séries des tableaux de bord.

## Dette

- `statusRecorder` et la route `inconnue` existent en double, dans `metrics/middleware.go` et dans `logging/middleware.go`. Ils seront regroupés dans `httpserver` (`StatusRecorder` exporté, lecture du modèle de route) après la fusion d'US-014, qui retouche `logging/middleware.go`.
- `outbox.ObserveBacklog(pool)` : relevé de l'outbox lu en base à chaque collecte (délai 2 s), donc juste même si le relais est bloqué.

## Catalogue

| Métrique Prometheus | Type | Étiquettes | Source |
|---|---|---|---|
| `http_server_request_duration_seconds` | Histogramme (`_bucket`, `_count`, `_sum`) | `http_request_method`, `http_route`, `http_response_status_code` | API publiques et API interne de la Place de marché ; ni santé ni métriques |
| `canal_messages_publies_total` | Compteur | `echange` | Messages confirmés et routés par le broker (relais de l'outbox, résultats du Contrôle) |
| `canal_messages_consommes_total` | Compteur | `file` | Messages traités puis acquittés |
| `canal_messages_ecartes_total` | Compteur | `file`, `motif` (`emetteur_inattendu`, `refuse_par_le_traitement`) | Mises à l'écart par le consommateur ; celles du broker (`delivery-limit`) se lisent sur les files `.ecartes` (RabbitMQ) |
| `canal_messages_relances_total` | Compteur | `file` | Nouveaux essais après un échec transitoire |
| `outbox_relances_total` | Compteur | aucune | Cycles du relais différés après un lot non publié |
| `outbox_non_publies` | Jauge | aucune | Lignes en attente de publication (Place de marché) |
| `outbox_plus_ancien_non_publie_age_seconds` | Jauge | aucune | Âge de la plus ancienne ligne non publiée, 0 si aucune |

L'exportateur ajoute `otel_scope_name` et `otel_scope_version` à chaque série, et la série `target_info` (`service_name`).

## Règles

- **Cardinalité bornée** : `http_route` est le modèle de route du `ServeMux` sans sa méthode (`/marche/v1/articles/{id}`), jamais le chemin brut ; requête sans modèle : `inconnue`. Méthode hors liste HTTP : `_OTHER`. `file`, `echange` et `motif` sont des constantes du code.
- **Aucune donnée personnelle** : ni identifiant d'utilisateur, d'article, de message ou de corrélation en étiquette.
- **Latence** : limites de l'histogramme en secondes, resserrées autour de l'objectif p95 < 500 ms (ENF-14) : 0,005 ; 0,01 ; 0,025 ; 0,05 ; 0,1 ; 0,2 ; 0,3 ; 0,4 ; 0,5 ; 0,75 ; 1 ; 2,5 ; 5 ; 10.

## Requêtes types

- Débit : `sum by (http_route) (rate(http_server_request_duration_seconds_count[1m]))`
- Latence p95 : `histogram_quantile(0.95, sum by (le, http_route) (rate(http_server_request_duration_seconds_bucket[5m])))`
- Taux d'erreurs : `sum by (http_route) (rate(http_server_request_duration_seconds_count{http_response_status_code=~"5.."}[5m])) / sum by (http_route) (rate(http_server_request_duration_seconds_count[5m]))`
- Outbox non publiée : `max(outbox_plus_ancien_non_publie_age_seconds) > 60` (tous les réplicas lisent la même table : `max`, jamais `sum`), à doubler d'un `absent(outbox_plus_ancien_non_publie_age_seconds)` (base injoignable ou service arrêté).
