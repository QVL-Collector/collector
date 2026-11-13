# Service Interactions

Interactions minimal du POC ([D02](../../docs/decisions/D02-decoupage-applicatif.md), option P2) : reçoit la décision de publication diffusée par la Place de marché et en tire une notification pour le vendeur, listée dans son espace. Ni email, ni préférences, ni messagerie. Go, bibliothèque standard, `pgx`, `goose`, `libs/platform` ([D04](../../docs/decisions/D04-stack.md)). Contrats : [`contracts/api/interactions/v1.openapi.yaml`](../../contracts/api/interactions/v1.openapi.yaml), message [`publication.decidee` v1](../../contracts/README.md).

## Sous-commandes et ports

| Commande | Rôle | Compte de base attendu |
|---|---|---|
| `interactions serve` (par défaut) | API sur `ADRESSE_API`, santé sur `ADRESSE_GESTION`, consommateur des décisions de publication | `interactions_execution` |
| `interactions migrate` | Applique les migrations embarquées, puis s'arrête (Job Helm `pre-install`, `pre-upgrade`) | `interactions_migration` |

| Port | Contenu | Exposition |
|---|---|---|
| 8080 | `/interactions/v1/...` | Routé par l'ALB |
| 9090 | `GET /sante` (vivant), `GET /disponibilite` (base joignable ; ni Keycloak ni le broker interrogés) | Jamais routé ; sondes Kubernetes |

## Notifications des décisions de publication (US-028)

1. Consommateur de la file `interactions.publication-decidee` (liée à `pdm.evenements`, clé `publication.decidee.v1` ; acquittement manuel, préchargement 10). L'identité `interactions` n'a que la lecture de cette file : aucune écriture sur le canal.
2. Écarté sans remise en file ni nouvel essai, vers `interactions.publication-decidee.ecartes` : `user-id` différent de `CANAL_EMETTEUR_ATTENDU`, message non conforme au contrat (taille, JSON illisible, clé dupliquée, champ manquant ou inconnu, type ou version inconnus, issue ou motif hors liste fermée, motif présent pour `publie` ou absent sinon). Journaux Warn `notifications.decision_ecartee` (`message_id`, `raison`) puis `canal.message_ecarte`. Les messages suivants sont traités normalement ; un message relivré sans fin est mis à l'écart par le broker après 5 livraisons.
3. Une décision valide crée une notification : `INSERT … ON CONFLICT (message_id) DO NOTHING`. L'`id` de l'enveloppe identifie la décision, conservé au rejeu de l'outbox : une décision livrée deux fois ne crée qu'une notification (journal Info `notifications.doublon_ignore`, sinon `notifications.notification_creee`).
4. Une erreur transitoire (base indisponible) ne rejette jamais le message : nouvel essai avec une attente croissante, sans acquitter. Interactions arrêté, les décisions attendent dans la file durable (24 h au plus) et sont notifiées au retour.

Le service ne déclare aucune file ni aucun échange : la topologie est posée par l'IaC ([infra/canal](../../infra/canal/README.md)). Une file absente ou un droit refusé arrête le service en erreur.

### `GET /interactions/v1/notifications?page&taille`

Jeton Keycloak exigé (audience `interactions`, rôle de realm `vendeur`) : 401 sans jeton ou jeton invalide, 403 sans le rôle. Seules les notifications dont le vendeur est le sujet du jeton sont lues, de la plus récente décision à la plus ancienne. `page` (1 à 10 000, défaut 1) et `taille` (1 à 50, défaut 20) : répété, vide, non entier ou hors bornes, 400. Liste lue en transaction `READ ONLY` `REPEATABLE READ`.

```json
{
  "elements": [
    {
      "id": "6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b",
      "articleId": "8a2d4f6b-1c3e-4a5b-9d7f-0e2c4a6b8d1f",
      "revision": 2,
      "issue": "refuse",
      "motif": "annonce_incomplete",
      "decideLe": "2026-11-06T09:15:02.476513Z"
    }
  ],
  "page": 1,
  "taille": 20,
  "total": 1
}
```

Aucun contenu saisi par un utilisateur (ni titre, ni description) : le front construit le texte, en français ou en anglais, à partir de `issue` et `motif` (`null` pour `publie`).

## Variables d'environnement

| Variable | Commande | Défaut | Contenu |
|---|---|---|---|
| `BASE_DE_DONNEES_URL` | toutes | obligatoire | `postgres://<rôle>:<mot de passe>@<hôte>:5432/interactions?sslmode=verify-full&sslrootcert=<paquet RDS>` ; rôle d'exécution pour `serve`, rôle de migration pour `migrate` |
| `BASE_DE_DONNEES_TLS_EXIGE` | toutes | `true` | `false` seulement en local : autorise un autre `sslmode` que `verify-full` |
| `OIDC_EMETTEUR` | `serve` | obligatoire | Émetteur exact, ex. `https://auth.<domaine>/realms/collector` |
| `OIDC_JEU_DE_CLES_URL` | `serve` | `<émetteur>/protocol/openid-connect/certs` | Adresse interne du jeu de clés si l'émetteur public n'est pas joignable depuis le cluster |
| `OIDC_AUDIENCE` | `serve` | `interactions` | Audience exigée dans le jeton |
| `OIDC_CLIENT_AUTORISE` | `serve` | `collector-web` | Valeur exigée de `azp` |
| `CANAL_HOTE` | `serve` | obligatoire | Hôte AMQP du broker |
| `CANAL_PORT` | `serve` | `5672` | Port AMQP |
| `CANAL_VHOST` | `serve` | `collector` | Vhost |
| `CANAL_UTILISATEUR` | `serve` | obligatoire | Identité d'Interactions sur le canal (`interactions`) |
| `CANAL_MOT_DE_PASSE` | `serve` | obligatoire | Secret de cette identité, généré à chaque `up` (MC5) |
| `CANAL_EMETTEUR_ATTENDU` | `serve` | `marche` | Seul `user-id` accepté sur la file des décisions (MC2) |
| `ADRESSE_API` | `serve` | `:8080` | Écoute de l'API |
| `ADRESSE_GESTION` | `serve` | `:9090` | Écoute de la santé |
| `JOURNAL_NIVEAU` | toutes | `INFO` | `DEBUG`, `INFO`, `WARN`, `ERROR` ; journaux JSON sur la sortie standard |

## Base et rôles PostgreSQL attendus

Base `interactions`, créée par l'amorçage avant toute migration ([D07](../../docs/decisions/D07-donnees.md), MD1, MD2), cloisonnée des bases `marche` et `transactions` :

```sql
CREATE ROLE interactions_migration LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD '…';
CREATE ROLE interactions_execution LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD '…';
CREATE DATABASE interactions OWNER interactions_migration;
REVOKE CONNECT ON DATABASE interactions FROM PUBLIC;
GRANT CONNECT ON DATABASE interactions TO interactions_migration, interactions_execution;
```

Puis, connecté en `interactions_migration` sur `interactions` : `REVOKE CREATE ON SCHEMA public FROM PUBLIC`. La base `marche` reçoit le même `REVOKE CONNECT … FROM PUBLIC` : aucun rôle d'un service ne se connecte à la base d'un autre.

- `interactions_migration` : propriétaire de la base et du schéma `public`, seul à exécuter du DDL ; secret livré au seul Job de migration.
- `interactions_execution` : propriétaire d'aucun objet, sans `CREATE` sur `public` ; nom fixe, car la première migration lui accorde `SELECT` et `INSERT` sur les tables par `ALTER DEFAULT PRIVILEGES`. Ni `UPDATE`, ni `DELETE`, ni `TRUNCATE` sur `notification` (révoqués explicitement) : une notification ne se modifie pas. Aucun droit sur la table de version de `goose`.

Une migration exécutée sans que `interactions_execution` existe échoue.

## Limites d'entrée

Messages du canal : 64 Kio au plus, validés contre leur schéma avant décodage (MC6). Les messages ne sont jamais journalisés en clair : seuls l'identifiant du message, l'article, la révision et l'issue le sont.
