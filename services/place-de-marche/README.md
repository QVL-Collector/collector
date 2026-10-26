# Service Place de marché

Module Annonces : brouillons du vendeur, catégories. Go, bibliothèque standard, `pgx`, `goose`, `libs/platform` ([D04](../../docs/decisions/D04-stack.md)). Contrat : [`contracts/api/place-de-marche/v1.openapi.yaml`](../../contracts/api/place-de-marche/v1.openapi.yaml).

## Sous-commandes et ports

| Commande | Rôle | Compte de base attendu |
|---|---|---|
| `place-de-marche serve` (par défaut) | API publique sur `ADRESSE_API`, santé sur `ADRESSE_GESTION` | `marche_execution` |
| `place-de-marche migrate` | Applique les migrations embarquées, puis s'arrête (Job Helm `pre-install`, `pre-upgrade`) | `marche_migration` |

| Port | Contenu | Exposition |
|---|---|---|
| 8080 | `/marche/v1/...` | Routé par l'ALB |
| 9090 | `GET /sante` (vivant), `GET /disponibilite` (base joignable ; Keycloak non interrogé) | Jamais routé ; sondes Kubernetes |

## Variables d'environnement

| Variable | Commande | Défaut | Contenu |
|---|---|---|---|
| `BASE_DE_DONNEES_URL` | les deux | obligatoire | `postgres://<rôle>:<mot de passe>@<hôte>:5432/marche?sslmode=verify-full&sslrootcert=<paquet RDS>` ; secret du rôle d'exécution pour `serve`, du rôle de migration pour `migrate` |
| `BASE_DE_DONNEES_TLS_EXIGE` | les deux | `true` | `false` seulement en local : autorise un autre `sslmode` que `verify-full` |
| `OIDC_EMETTEUR` | `serve` | obligatoire | Émetteur exact, ex. `https://auth.<domaine>/realms/collector` |
| `OIDC_JEU_DE_CLES_URL` | `serve` | `<émetteur>/protocol/openid-connect/certs` | Adresse interne du jeu de clés si l'émetteur public n'est pas joignable depuis le cluster |
| `OIDC_AUDIENCE` | `serve` | `place-de-marche` | Audience exigée dans le jeton |
| `OIDC_CLIENT_AUTORISE` | `serve` | `collector-web` | Valeur exigée de `azp` |
| `ADRESSE_API` | `serve` | `:8080` | Écoute de l'API |
| `ADRESSE_GESTION` | `serve` | `:9090` | Écoute de la santé |
| `JOURNAL_NIVEAU` | les deux | `INFO` | `DEBUG`, `INFO`, `WARN`, `ERROR` ; journaux JSON sur la sortie standard |

## Rôles PostgreSQL attendus

Créés par l'amorçage de la base `marche`, avant toute migration ([D07](../../docs/decisions/D07-donnees.md), MD1, MD2) :

- `marche_migration` : propriétaire de la base et du schéma `public`, seul à exécuter du DDL ; secret livré au seul Job de migration.
- `marche_execution` : `NOSUPERUSER NOCREATEDB NOCREATEROLE`, propriétaire d'aucun objet, sans `CREATE` sur `public` ; nom fixe, car la première migration lui accorde `SELECT`, `INSERT`, `UPDATE` sur les tables et `USAGE`, `SELECT` sur les séquences par `ALTER DEFAULT PRIVILEGES`. Aucun `DELETE`, aucun droit sur la table de version de `goose`.

Une migration exécutée sans que `marche_execution` existe échoue.

## Limites d'entrée

Corps JSON de 32 Kio au plus (413 au-delà) ; titre 200 caractères, description 5 000 ; prix jusqu'à 1 000 000 €, frais de port jusqu'à 1 000 €.
