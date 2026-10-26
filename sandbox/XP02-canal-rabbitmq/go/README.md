# XP02 — programmes Go du canal RabbitMQ

Module Go autonome (Go 1.26.3, `github.com/rabbitmq/amqp091-go` v1.10.0, `github.com/jackc/pgx/v5` v5.7.5) qui préfigure `libs/platform` (D04, D05) : publication confirmée, outbox PostgreSQL unique, consommateur idempotent à acquittement manuel, reconnexion supervisée.

```powershell
go mod tidy
go build ./...
go vet ./...
go test ./...
```

Chaque binaire écrit ses mesures sur la sortie standard au format de `docs/mesures/XP02-canal-rabbitmq.csv` (séparateur `;`, colonnes `seuil` et `atteint` laissées vides) et ses journaux JSON sur la sortie d'erreur. Rediriger : `... >> ..\..\..\docs\mesures\XP02-canal-rabbitmq.csv`.

## Binaires

| Binaire | Rôle | Mesures émises |
|---|---|---|
| `xp02-publish` | Vague de N messages en enveloppe JSON, confirmations de publication, `user-id` posé, `mandatory` | publiés confirmés, lots refusés (`nack`), messages non routés, confirmation p50/p95/p99/max, débit, reconnexions, refus du broker (code AMQP) |
| `xp02-consume` | Consommation d'une file, `Qos`, acquittement manuel après l'effet, `user-id` vérifié, validation stricte (taille, clés dupliquées, membres exacts), journal d'idempotence | livraisons, uniques, doublons, relivraisons, non conformes, émetteur refusé, latence de bout en bout, débit, uniques cumulés, reconnexions |
| `xp02-outbox` | `migrate` (schéma), `produce` (décision + événement dans une transaction), `relay` (`FOR UPDATE SKIP LOCKED`, confirmation puis marquage), `replay` (republication bornée à 7 jours), `verify` (outbox comparée au journal de réception) | transactions et débit d'écriture, retard outbox → confirmation, arriéré, republiés, manquants |

## Variables d'environnement

| Variable | Défaut | Utilisée par |
|---|---|---|
| `XP02_AMQP_USER`, `XP02_AMQP_PASSWORD` | obligatoires | publish, consume, outbox relay/replay |
| `XP02_AMQP_HOST`, `XP02_AMQP_PORT`, `XP02_AMQP_VHOST` | `localhost`, `5672`, `collector` | idem |
| `XP02_PG_DSN` | vide : variables `PGHOST`, `PGUSER`, `PGPASSWORD`, `PGDATABASE` | outbox, consume avec `XP02_LEDGER=postgres` |
| `XP02_SESSION`, `XP02_PLATEFORME`, `XP02_CRITERE` | vide, `docker-compose`, vide | tous (colonnes du CSV) |
| `XP02_EXCHANGE`, `XP02_ROUTING_KEY` | obligatoires pour publish ; `pdm.evenements`, `publication.decidee.v1` pour outbox | publish, outbox produce |
| `XP02_COUNT`, `XP02_BATCH` | `1000`, `50` (publish) ou `100` (outbox) | publish, outbox |
| `XP02_RATE` | `0` (sans limite), en messages par seconde | publish |
| `XP02_EVENT_TYPE` | `controle.demande` | publish |
| `XP02_USER_ID` | `XP02_AMQP_USER` ; une autre valeur usurpe l'émetteur | publish |
| `XP02_POISON_AT` | `-1` ; sinon insère un JSON tronqué et un JSON à clé dupliquée après ce rang | publish |
| `XP02_CONFIRM_TIMEOUT` | `30s` | publish, outbox relay/replay |
| `XP02_QUEUE` | obligatoire | consume, outbox verify |
| `XP02_EXPECTED_PUBLISHER` | obligatoire (ex. `svc-pdm`) | consume |
| `XP02_PREFETCH`, `XP02_MAX_MESSAGE_BYTES` | `10`, `65536` | consume |
| `XP02_INVALID_POLICY` | `reject` (mise à l'écart immédiate) ; `requeue` (mise à l'écart par la limite de 5 livraisons) | consume |
| `XP02_PROCESSING_DELAY` | `0` (consommateur lent pour XC6, ex. `500ms`) | consume |
| `XP02_IDLE_TIMEOUT` | `0` (jusqu'à Ctrl+C) ; arrêt après ce délai sans message | consume, outbox relay |
| `XP02_LEDGER` | `memory` ; `postgres` pour survivre au redémarrage du consommateur | consume |
| `XP02_RELAY_INTERVAL` | `200ms` | outbox relay |
| `XP02_REPLAY_SINCE` | obligatoire, RFC 3339, au plus 7 jours | outbox replay |
| `XP02_SINCE` | origine des temps | outbox verify |

## Commandes par critère

Identifiants lus dans le `.env` généré, jamais saisis dans une commande versionnée.

- **XC1 point à point** : `xp02-consume` (`XP02_AMQP_USER=svc-controle`, `XP02_QUEUE=controle.demandes`, `XP02_EXPECTED_PUBLISHER=svc-pdm`, `XP02_IDLE_TIMEOUT=15s`) puis `xp02-publish` (`svc-pdm`, `XP02_EXCHANGE=controle.demandes`, `XP02_ROUTING_KEY=v1`). Attendu : `publies_confirmes` = `uniques` = 1000.
- **XC1 diffusion** : deux `xp02-consume` (`svc-interactions` sur `interactions.publication-decidee`, `svc-test` sur `test.publication-decidee`, `XP02_LEDGER=postgres`), puis `xp02-outbox migrate`, `produce`, `relay` (`svc-pdm`), puis `verify` pour chaque file. Attendu : `manquants` = 0.
- **XC2 / XC9** : `xp02-publish` avec l'identité d'un autre service sur `pdm.evenements` ou avec une clé non accordée ; `xp02-consume` sur la file d'un autre ; `xp02-publish` avec `XP02_USER_ID` différent de l'utilisateur. Attendu : ligne `refus_broker` à `403` (`ACCESS_REFUSED`) ou `406` (`PRECONDITION_FAILED`). Déclaration, liaison, purge et API de gestion : scripts du dossier.
- **XC3** : `xp02-consume` avec `XP02_LEDGER=postgres`, arrêté (Ctrl+C) pendant la vague, relancé après 2 min. Attendu : `uniques_cumules` = 1000.
- **XC4** : `xp02-publish` avec `XP02_POISON_AT=500` ; `xp02-consume` avec `XP02_INVALID_POLICY=requeue` pour observer la mise à l'écart vers `.ecartes` après 5 livraisons. Attendu : `non_conformes` > 0, `uniques` = 1000. Rejeu : `xp02-outbox replay` (le rejeu passe par le producteur, MC2).
- **XC5** : `xp02-outbox relay` et `xp02-consume` actifs, `docker restart` du broker. Attendu : lignes `reconnexion` ≤ 60 s. Après suppression du pod : `xp02-outbox replay` avec `XP02_REPLAY_SINCE` = début de session, puis `verify`.
- **XC6** : `xp02-publish` avec `XP02_COUNT=300`, `xp02-consume` avec `XP02_PROCESSING_DELAY=500ms`.
- **XC8** : `xp02-publish` avec `XP02_RATE` et `XP02_COUNT` fixés pour une vague de 5 min.

Exemple PowerShell, `XP02_AMQP_PASSWORD` déjà chargé depuis le `.env` généré :

```powershell
$env:XP02_AMQP_USER = "svc-pdm"
$env:XP02_EXCHANGE = "controle.demandes"; $env:XP02_ROUTING_KEY = "v1"; $env:XP02_CRITERE = "XC1"
go run ./cmd/xp02-publish >> ..\..\..\docs\mesures\XP02-canal-rabbitmq.csv
```

## Contrat de l'enveloppe (hypothèse du bac à sable)

Membres exacts : `id` (UUID), `type`, `version` (≥ 1), `emitted_at` (RFC 3339 UTC), `correlation_id` (UUID), `data` (objet). Propriétés AMQP : `message_id` = `id`, `type`, `correlation_id`, `user_id`, `timestamp`, persistant. Le contrat définitif relève de `contracts/messages/`.
