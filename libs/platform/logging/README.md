# libs/platform/logging

Journaux `slog` JSON sur la sortie standard, communs à tous les services Go ([D09](../../../docs/decisions/D09-observabilite.md), US-014). Collecte par Fluent Bit vers CloudWatch Logs : hors de ce module.

## Ligne de journal

```json
{"time":"2026-11-20T10:15:02.123456789Z","level":"INFO","msg":"demande contrôlée","service":"controle","event":"controle.demande_controlee","message_id":"6f1c…","article_id":"0b7e…","revision":2,"issue":"publie","motif":"","version_regles":3,"correlation_id":"8d1f6a2e-3b4c-4d5e-9f60-718293a4b5c6","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7"}
```

- `service` : posé par `logging.New`. `correlation_id`, `trace_id`, `span_id` : lus dans le contexte, à condition de journaliser avec `…Context(ctx, …)` ou `LogAttrs(ctx, …)`. Un journal hors de toute requête ou de tout message (démarrage, boucles de fond) n'en porte pas.
- Événement d'audit (`logging.Audit`) : même ligne, avec `"audit":true` juste après `event`. Horodatage : `time`.

```json
{"time":"…","level":"INFO","msg":"résultat de contrôle appliqué","service":"place-de-marche","event":"controle.resultat_applique","audit":true,"message_id":"…","article_id":"…","revision":2,"issue":"refuse","version_regles":3,"motif":"coordonnees_detectees","correlation_id":"…","trace_id":"…","span_id":"…"}
```

## Corrélation et trace

| Point d'entrée | `correlation_id` | `trace_id` |
|---|---|---|
| Requête HTTP (`logging.Middleware`) | En-tête `X-Correlation-Id` repris s'il est un UUID, sinon tiré ; renvoyé dans la réponse | En-tête `traceparent` W3C repris s'il est bien formé (version `00`, identifiants non nuls), sinon tiré |
| Message consommé (`broker.Consumer`) | Propriété AMQP `correlation-id`, puis celle de l'enveloppe, qui fait foi | En-tête AMQP `traceparent`, sinon tiré |
| Commande ponctuelle (`logging.StartOperation`) | Tiré | Tiré |
| Reprise d'un travail mémorisé (`logging.Resume`, ex. relance d'une demande de contrôle) | Valeur mémorisée si UUID, sinon tiré | `traceparent` mémorisé, sinon tiré |
| Journal d'un travail déjà ouvert (`logging.Restore`, ex. journal de la relance) | Valeur reprise si UUID, sinon tirée | Même `trace_id` et même `span_id`, sans nouveau span |

Le middleware HTTP et le consommateur passent tous deux par `Resume` : une seule règle de reprise. Chaque point d'entrée ouvre un nouveau span (nouveau `span_id`) sous le même `trace_id`. Propagation : `outbox.NewEvent(ctx, …)` garde le `traceparent` du contexte avec l'événement (colonne `outbox.traceparent`), le relais et `broker.Publisher` le posent en en-tête AMQP ; `logging.Propagate(ctx, header)` pose `X-Correlation-Id` et `traceparent` sur un appel HTTP sortant. Le `correlation_id` voyage dans l'enveloppe ([contrat](../../../contracts/messages/enveloppe-v1.schema.json)), le `traceparent` dans les seuls en-têtes.

Implémentation alignée sur [D04](../../../docs/decisions/D04-stack.md) et [D09](../../../docs/decisions/D09-observabilite.md) (décision de Martin, 2026-11-25, US-014) : SDK OpenTelemetry Go (`go.opentelemetry.io/otel`, même version qu'US-015), propagateur W3C `propagation.TraceContext` pour lire et écrire `traceparent`, `TracerProvider` du SDK sans exportateur pour tirer `trace_id` et `span_id`. Le contexte porte donc un span OpenTelemetry ordinaire, que toute instrumentation ultérieure (`otelhttp`, traces exportées) reprend sans second générateur. Validation plus stricte que le propagateur, en amont : seule la forme `00-<32 hexa minuscules>-<16 hexa minuscules>-<2 hexa>` est lue, comme dans les contraintes `CHECK` des colonnes `traceparent` ; le propagateur écarte ensuite les identifiants nuls et les drapeaux inconnus. Échantillonnage `ParentBased(NeverSample)` : une trace tirée est non échantillonnée (`-00`), le drapeau « échantillonné » reçu est conservé (`-01`).

## Masquage à la source

Par `ReplaceAttr` sur toute clé, y compris dans les groupes et sur le message :

- clé dont le nom, en minuscules et `-` lu `_`, contient `authorization`, `cookie`, `jeton`, `token`, `password`, `passwd`, `mot_de_passe`, `secret`, `credential`, `signature`, `dsn`, `url_base_de_donnees`, `chaine_de_connexion` ou `connection_string` : valeur remplacée par `[masqué]` (couvre `X-Collector-User-Token`, `X-Amz-Signature`…) ;
- dans toute valeur texte, erreur (`*url.Error` comprise) ou valeur `fmt.Stringer` (`*url.URL`) : JWT, valeur après `Bearer `/`Basic `, valeurs des paramètres `X-Amz-Credential`, `X-Amz-Signature`, `X-Amz-Security-Token`, valeurs `password=`, `secret=`, `token=`… et mot de passe d'une URL de connexion (`://role:…@`) ;
- `http.Header` et `url.Values` journalisés en groupe, une clé par nom, chacune passée aux règles précédentes.

Hors de portée de ce module : traces de `panic` écrites sur la sortie d'erreur, contenu de structures journalisées par `slog.Any` (sérialisées telles quelles : ne pas journaliser de structure métier), journaux de Keycloak et de RabbitMQ. Le filtre Fluent Bit (seconde couche, MO3) les couvre.

## Origine des événements d'audit

`service` et `audit` sont déclarés par le service lui-même : ils ne prouvent rien. L'aiguillage vers le groupe d'audit et l'origine retenue d'un événement viennent des métadonnées Kubernetes ajoutées par Fluent Bit (namespace, pod, conteneur), jamais du contenu de la ligne (D09 MO2, AC6 d'US-014). Aucun champ d'origine « de confiance » n'est écrit dans les journaux ni dans les messages.
