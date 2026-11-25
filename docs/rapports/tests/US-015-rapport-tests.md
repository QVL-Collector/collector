# Rapport de tests — US-015 Métriques et tableaux de bord de la plateforme

## Back

| Champ | Valeur |
|---|---|
| US | [US-015](../../../backlog/US-015-metriques-tableaux-de-bord.md) |
| Branche | `feat/US-015-metriques-tableaux-de-bord` (non commitée) |
| Date | 2026-11-23 |
| Fonction | Test |
| Périmètre | Partie back : **AC1 côté services** (débit, latence, erreurs par route et longueur des files exposés par chaque service sur un port de métriques distinct) et **AC5 côté service** (le port de métriques n'est jamais le port public). `libs/platform/metrics`, `ADRESSE_METRIQUES`, instruments du canal et de l'outbox, branchement dans Place de marché, Contrôle, Interactions et Transactions |
| Hors périmètre | AC1 « When Prometheus les relève » (collecte réelle), AC2 (Grafana), AC3 (règles d'alerte), AC4 (export CSV), AC5 infra (`ClusterIP`, ALB, mot de passe Grafana) : **reportés (DevOps/AWS)** |
| Outils | `go test`, testcontainers-go (PostgreSQL, RabbitMQ, Keycloak pour le banc de Transactions), services construits et **lancés en processus** (`serve`) sur des adresses libres, relevé `GET /metrics` analysé au format d'exposition Prometheus |
| Verdict | **Tous les AC du périmètre back sont couverts et conformes** ; aucune anomalie. Une observation sur l'isolation des tests (voir plus bas) |

### Méthode

Scénarios dérivés de AC1 et AC5, étiquetés `US015…`, au format Given/When/Then dans le nom des tests. Le relevé de métriques est lu comme le ferait Prometheus : `GET /metrics` sur `ADRESSE_METRIQUES`, analyse ligne à ligne du format texte. Trois niveaux :

- **Unitaire** (`libs/platform/metrics`) : une API d'essai (`ServeMux` réel, chaîne `logging.Middleware` → `metrics.Middleware` → `httpserver.Recover`) et le gestionnaire retourné par `metrics.Install`.
- **Bout en bout en processus** : Place de marché, Interactions, Contrôle et Transactions construits (`go install`) puis lancés en `serve` avec quatre adresses distinctes (API, interne, santé, métriques). Les requêtes passent par les vrais ports ; les messages par un vrai RabbitMQ (topologie du dépôt).
- **Bancs existants** : `ADRESSE_METRIQUES` libre ajoutée à chaque lancement de `serve` (bloquant levé).

Correspondance AC → besoin mesuré :

| AC1 demande | Preuve |
|---|---|
| Débit | `http_server_request_duration_seconds_count` incrémenté de exactement 1 par requête |
| Latence p95 | histogramme `_bucket` avec limites resserrées (dont `le="0.5"` et `+Inf`), `_sum` |
| Taux d'erreurs par route | étiquettes `http_route` (modèle) et `http_response_status_code` (401, 404, 500, 503 comptés) |
| Longueur des files | `outbox_non_publies`, `outbox_plus_ancien_non_publie_age_seconds`, `canal_messages_{publies,consommes,ecartes}_total` par file |

### Livrables de test

| Fichier | Contenu |
|---|---|
| `libs/platform/metrics/metrics_us015_test.go` | 13 tests unitaires (voir AC1, AC5) |
| `services/place-de-marche/cmd/place-de-marche/metriques_releve_test.go` | Relevé et analyse de `/metrics`, assertions communes (routes paramétrées, aucune donnée identifiante, ports sans métriques, 404 hors `/metrics`) |
| `services/place-de-marche/cmd/place-de-marche/metriques_us015_test.go` | `TestUS015MetriquesDeLaPlaceDeMarche` : 12 scénarios, Place de marché lancée en processus |
| `services/place-de-marche/cmd/place-de-marche/metriques_controle_transactions_us015_test.go` | `TestUS015MetriquesDuControle` (5 scénarios), `TestUS015MetriquesDeTransactions` (3 scénarios, banc Keycloak d'US-030) |
| `services/interactions/cmd/interactions/metriques_releve_test.go` | Relevé de `/metrics` (copie du paquet Place de marché, paquets distincts) |
| `services/interactions/cmd/interactions/metriques_us015_test.go` | `TestUS015MetriquesDInteractions` : 7 scénarios, Interactions lancé en processus |
| Bancs modifiés (bloquant) | `ADRESSE_METRIQUES` libre ajoutée : `place-de-marche/…/services_test.go` (Place de marché en processus de test, Contrôle), `reservation_banc_test.go` (Transactions, Place de marché sur Keycloak), `reservation_duree_config_test.go`, `notification_vendeur_test.go` (Interactions), `interactions/…/banc_test.go` ; champs `metriques` ajoutés à `processusControle`, `processusTransactions` et `serviceInteractions` (avec `gestion`) |

### AC1 — Métriques exposées sur un port distinct

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Unitaire — 3 GET sur `/essai/articles/{id}` avec 3 UUID | +3 sous `http_route="/essai/articles/{id}"`, aucun UUID exposé | +3, aucun UUID | OK |
| Unitaire — 50 identifiants différents (cardinalité) | une seule série pour la route | une seule série | OK |
| Unitaire — méthode `BREW` | comptée sous `_OTHER`, `BREW` jamais exposé | `_OTHER`, `BREW` absent | OK |
| Unitaire — méthode `DELETE` | conservée telle quelle | `DELETE` | OK |
| Unitaire — panique interceptée | réponse 500 et comptée en `500` | 500, +1 sous `500` | OK |
| Unitaire — 2 réponses 503 (taux d'erreurs par route) | +2 sous la route et le code 503, identifiant absent | +2, identifiant absent | OK |
| Unitaire — chemin sans route `/inexistant/client-<uuid>` | 404 compté, chemin brut jamais exposé | 404 compté, chemin absent | OK |
| Unitaire — histogramme de latence | limites dont `0.5` et `+Inf` (au moins 5), `_sum` présent | limites présentes dont `0.5` et `+Inf`, `_sum` présent | OK |
| Unitaire — étiquettes HTTP | seulement méthode, route, code (+ portée OTel, `le`) | conforme | OK |
| Unitaire — compteur créé par `metrics.Counter` sur le fournisseur global | +2 exposé en `_total` | +2 | OK |
| Unitaire — identité du service | `target_info{service_name=…}` | présent | OK |
| Place de marché — 2 créations puis 2 lectures d'article | +2 GET 200 sous une route modèle (`{…}`), POST 201 comptés | +2, route paramétrée, POST comptés | OK |
| Place de marché — lecture d'un article inconnu | +1 GET 404 sous la route modèle | +1 | OK |
| Place de marché — API interne `POST /interne/v1/articles/<uuid>/reservations` sans jeton | comptée sous une route modèle `/interne/…{…}` avec son code | comptée, route paramétrée | OK |
| Place de marché — soumission d'un article | `canal_messages_publies_total{echange="controle.demandes"}` incrémenté par le relais | incrémenté | OK |
| Place de marché — outbox vidée | `outbox_non_publies` = 0 et âge = 0 | 0 et 0 | OK |
| Place de marché — résultat valide du Contrôle | `canal_messages_consommes_total{file="pdm.resultats-controle"}` +1 | +1 | OK |
| Place de marché — résultat publié par `admin-canal` | `canal_messages_ecartes_total{motif="emetteur_inattendu"}` +1 | +1 | OK |
| Place de marché — résultat tronqué | `canal_messages_ecartes_total{motif="refuse_par_le_traitement"}` +1 | +1 | OK |
| Place de marché — broker arrêté (relais bloqué), article soumis | `outbox_non_publies` ≥ 1, âge > 0 puis **croît** de ~3 s en 3 s | 1 ; âge 3,02 s à la 2e lecture (croissance ≥ 2 s) ; `outbox_relances_total` = 0 | OK |
| Contrôle — demande non conforme publiée par `marche` | `canal_messages_ecartes_total{file="controle.demandes",motif="refuse_par_le_traitement"}` +1 | +1 | OK |
| Contrôle — demande publiée par `admin-canal` | `…{motif="emetteur_inattendu"}` +1 | +1 | OK |
| Transactions — 2 `GET /commandes/<uuid>` sans jeton | +2 GET 401 sous une route modèle, identifiants absents | +2, route paramétrée | OK |
| Interactions — liste du vendeur (200) puis sans jeton (401) | +1 par code sous `/interactions/v1/notifications`, sans la requête brute (`?taille=50`) | +1 et +1, route sans requête | OK |
| Interactions — décision valide de `marche` | `canal_messages_consommes_total{file="interactions.publication-decidee"}` +1 | +1 | OK |
| Interactions — décision publiée par `admin-canal` | `…ecartes_total{motif="emetteur_inattendu"}` +1 | +1 | OK |
| Interactions — décision tronquée | `…ecartes_total{motif="refuse_par_le_traitement"}` +1 | +1 | OK |
| Place de marché, Contrôle, Transactions, Interactions — données personnelles | aucun sujet de vendeur, identifiant d'article, de commande ou de message, ni titre d'article dans le relevé ; aucun UUID en valeur d'étiquette | aucun | OK |

Non couvert en boîte noire : `canal_messages_relances_total` (échec transitoire du traitement non provocable depuis l'extérieur) et `outbox_relances_total` (resté à 0 broker arrêté : le relais ne tente pas de lot sans connexion) ; ces deux compteurs ne sont pas exigés nommément par l'AC1, la longueur des files l'est et est prouvée.

### AC5 (côté service) — Port de métriques jamais sur le port public

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Unitaire — gestionnaire de métriques sur `/`, `/sante`, `/disponibilite`, `/metrics/`, `/metrics/x`, `/metricsx`, route d'API | 404 | 404 | OK |
| Unitaire — `POST`, `PUT`, `DELETE /metrics` | jamais de métriques servies | refusés, rien servi | OK |
| Place de marché — `GET /metrics` sur les ports API, interne et santé | pas de métriques | aucune | OK |
| Place de marché — port de métriques : `/`, `/sante`, `/disponibilite`, `/marche/v1/categories`, route d'articles | 404 | 404 | OK |
| Contrôle — `GET /metrics` sur le port de santé ; port de métriques hors `/metrics` | pas de métriques ; 404 | conforme | OK |
| Transactions — `GET /metrics` sur les ports API et santé ; port de métriques hors `/metrics` | pas de métriques ; 404 | conforme | OK |
| Interactions — `GET /metrics` sur les ports API et santé ; port de métriques hors `/metrics` | pas de métriques ; 404 | conforme | OK |

### AC2, AC3, AC4, AC5 (infra)

Reportés (DevOps/AWS) : Grafana et ses tableaux de bord, règles d'alerte (`.ecartes`, outbox non publiée), export CSV, inventaire `ClusterIP`/ALB et durcissement de Grafana. Les métriques nécessaires à AC3 existent côté service (`canal_messages_ecartes_total`, `outbox_plus_ancien_non_publie_age_seconds` qui croît relais arrêté) ; l'alerte elle-même reste à prouver.

### Bloquant traité — collisions sur `:9091`

Sans `ADRESSE_METRIQUES`, chaque `serve` lancé par un banc écoutait sur `:9091` (défaut) : deux services du même banc (Place de marché et Contrôle, Place de marché et Transactions) ou deux bancs successifs se disputaient le port. Une adresse libre (`adresseLibre(t)`) est désormais posée partout où `ADRESSE_GESTION` l'était : 7 lancements dans 5 fichiers.

### Observation (hors AC, non bloquante) — `metrics.Install` appelé deux fois dans un même processus

Constat d'exécution : quand un test lance `serve` **dans le processus de test** après un autre `serve` déjà lancé dans ce même processus (cas des bancs en processus de la Place de marché et d'Interactions), le second service expose bien l'histogramme HTTP mais **aucune série `canal_messages_*`** : décision consommée et messages écartés non mesurés en 20 s (Interactions : 3 scénarios en échec ; Place de marché : 4). Lancé seul, ou en processus séparé, tout est mesuré. Les instruments du canal restent liés au premier fournisseur installé.

- Impact production : nul, un seul `serve` par processus (un pod).
- Impact tests : les bancs US-015 lancent donc les services **en processus séparé** (binaire construit), ce qui est aussi plus proche du déploiement.
- Suggestion aux devs (`@qvl-backend`) : si l'on veut des tests en processus fiables, que les instruments du canal suivent le fournisseur courant, ou documenter la contrainte « un `Install` par processus » dans le README de `metrics`.

### Résultats d'exécution

| Commande | Résultat | Durée |
|---|---|---|
| `go test ./metrics/ -count=1` (`libs/platform`) | `ok`, 13 tests PASS | 3 s |
| `go test ./cmd/place-de-marche/ -run TestUS015 -v` (première passe, Place de marché en processus de test) | Contrôle 5/5 PASS, Transactions 3/3 PASS ; Place de marché 8/12 : 4 scénarios du canal non mesurés après un `serve` antérieur dans le processus (observation ci-dessus) | 334 s |
| `go test -run 'TestUS015MetriquesDeLaPlaceDeMarche$' -v` (seul) | 12/12 PASS | 38 s |
| `go test ./... -count=1` (`services/interactions`), banc en processus de test | `TestUS015MetriquesDInteractions` 3 FAIL (même observation) ; seul : 7/7 PASS | 145 s |
| `go test ./... -count=1` (`services/interactions`), banc **en processus séparé** (code final) | `ok` dans les 2 paquets testés, 0 échec | 93 s |
| `go test ./... -count=1` (`libs/platform`) | `ok` dans tous les paquets testés, 0 échec | 15 s |
| `go test ./... -count=1` (`services/transactions`) | `ok` dans 4 paquets, 0 échec | 9 s |
| `go test ./... -count=1` (`services/controle`) | `ok` dans 3 paquets, 0 échec | 6 s |
| `go test ./... -count=1 -timeout 60m` (`services/place-de-marche`), code final | `ok` dans les 2 paquets testés (`cmd/place-de-marche`, `internal/config`), 0 échec, dont les 3 tests `TestUS015…` et tous les bancs modifiés | 1413 s |
| `gofmt -l` (5 modules) | aucun fichier | — |
| `go vet ./...` (5 modules) | 0 problème | — |
| `golangci-lint run --config <racine>/.golangci.yml ./...` (5 modules) | 0 issue (premier jet : 1 `errcheck`, 1 `gocognit`, 1 `unused`, 1 `gosec` G204, corrigés dans les tests) | — |
| Conteneurs | Arrêtés et supprimés par testcontainers | — |

### Verdict

Tous les AC du périmètre back (AC1 côté services, AC5 côté service) sont couverts et conformes. Aucune anomalie. AC2, AC3, AC4 et AC5 infra : reportés (DevOps/AWS).
