# Service Transactions

Transactions réduit à la réservation dans le POC ([D02](../../docs/decisions/D02-decoupage-applicatif.md), option P2) : l'acheteur réserve un article publié, Transactions obtient la réservation de la Place de marché et enregistre la commande « en attente de paiement » avec ses montants. Ni paiement, ni séquestre, ni reversement. Go, bibliothèque standard, `pgx`, `goose`, `x/oauth2` (`libs/platform/serviceclient`), `libs/platform` ([D04](../../docs/decisions/D04-stack.md)). Contrats : [`contracts/api/transactions/v1.openapi.yaml`](../../contracts/api/transactions/v1.openapi.yaml) (API publique) ; appel sortant : [`contracts/api/place-de-marche/interne-v1.openapi.yaml`](../../contracts/api/place-de-marche/interne-v1.openapi.yaml).

## Sous-commandes et ports

| Commande | Rôle | Compte de base attendu |
|---|---|---|
| `transactions serve` (par défaut) | API sur `ADRESSE_API`, santé sur `ADRESSE_GESTION` | `transactions_execution` |
| `transactions migrate` | Applique les migrations embarquées, puis s'arrête (Job Helm `pre-install`, `pre-upgrade`) | `transactions_migration` |

| Port | Contenu | Exposition |
|---|---|---|
| 8080 | `/transactions/v1/...` | Routé par l'ALB |
| 9090 | `GET /sante` (vivant), `GET /disponibilite` (base joignable ; ni Keycloak ni la Place de marché interrogés) | Jamais routé ; sondes Kubernetes |

## Réserver et commander (US-030)

Routes réservées au jeton Keycloak de l'acheteur : audience `transactions`, `azp` = `collector-web`, rôle de realm `acheteur`. 401 sans jeton valide (dont le jeton de service de Transactions, dont l'audience n'est pas `transactions` : AC9 admet 401 ou 403), 403 sans le rôle.

### `POST /transactions/v1/commandes`

Corps `{"articleId": "<uuid>"}`. Seul `articleId` est lu : tout autre champ (prix, montant, acheteur…) est ignoré, sans erreur (AC6) ; l'acheteur est le `sub` du jeton. Clé dupliquée, JSON mal formé, corps au-delà de 4 Kio ou non JSON : 400, 413, 415. `articleId` absent ou non UUID : 400 `saisie-invalide`. Limitation de débit (`libs/platform/ratelimit`, comme US-027) posée après l'authentification et le rôle : adresse et compte.

1. Jeton de service : `x/oauth2/clientcredentials`, client `transactions`, portée `reserver`, authentification `client_secret_basic`. Le jeton (5 min) est mis en cache et renouvelé 10 s avant son expiration ; aucune redirection n'est suivie avec lui.
2. Appel `POST {PLACE_DE_MARCHE_URL_INTERNE}/interne/v1/articles/{articleId}/reservations`, sans corps : `Authorization: Bearer <jeton de service>`, `X-Collector-User-Token: <jeton de l'acheteur>` (le jeton reçu, tel quel), `X-Correlation-Id` repris. Délai 5 s. Le jeton de l'acheteur n'est jamais journalisé (`token.Credential` masqué dans `fmt` et `slog`).
3. Montants calculés par le paquet pur `internal/montants` à partir du prix et des frais de port lus par la Place de marché ; échéance reprise telle quelle. `titre` lu dans la même réponse et figé dans la commande (US-032) : une modification ultérieure de l'annonce ne le change pas. Titre absent, blanc ou au-delà de 200 caractères : réponse illisible, donc 503.
4. `INSERT … ON CONFLICT (reservation_id) DO NOTHING` : une commande par réservation. Le nouvel essai reçoit la même réservation, donc la même commande, relue par `reservation_id` et `acheteur_id`.

| Réponse de la Place de marché | Réponse de Transactions |
|---|---|
| 201 (réservation créée) | 201, `Location`, commande créée ; journal Info `commandes.commande_creee` |
| 200 (même réservation) | 200, commande existante (créée si l'essai précédent n'avait pas abouti) ; journal `commandes.commande_existante` |
| 422 `article-non-reservable` (article inexistant, non publié, ou mis en vente par l'acheteur lui-même) | 422 `urn:collector:probleme:article-non-reservable` |
| 409 `article-deja-reserve` | 409 `urn:collector:probleme:article-deja-reserve` |
| 409 `reservation-deja-active` | 409 `urn:collector:probleme:reservation-deja-active` |
| Injoignable, délai dépassé, jeton de service refusé, 401, 403, 404, 5xx, réponse illisible | 503 ; journal Warn `commandes.place_de_marche_indisponible` ; aucune commande |

```json
{
  "id": "6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b",
  "articleId": "8a2d4f6b-1c3e-4a5b-9d7f-0e2c4a6b8d1f",
  "titre": "Montre Lip Nautic-Ski 1970",
  "statut": "en_attente_paiement",
  "prix": { "montantCentimes": 12000, "devise": "EUR" },
  "fraisDePort": { "montantCentimes": 590, "devise": "EUR" },
  "totalPaye": { "montantCentimes": 12590, "devise": "EUR" },
  "commission": { "montantCentimes": 630, "devise": "EUR" },
  "montantVendeur": { "montantCentimes": 11960, "devise": "EUR" },
  "echeance": "2026-11-16T10:25:00.123456Z",
  "creeLe": "2026-11-16T10:15:00.234567Z"
}
```

### `GET /transactions/v1/commandes` (US-032)

Commandes dont l'acheteur est le sujet du jeton, jamais lu dans la requête, de la plus récente à la plus ancienne (`creeLe` puis `id` décroissants). `page` (défaut 1, au plus 10 000) et `taille` (défaut 20, au plus 50), bornes de la Place de marché ; paramètre répété, vide ou non entier : 400 `invalide`, hors bornes : 400 `hors_limites` (`urn:collector:probleme:saisie-invalide`). Réponse `{elements, page, taille, total}`, `elements` toujours un tableau, vide au-delà du total. Décompte et page lus dans une même transaction `READ ONLY` `REPEATABLE READ`. Statut dérivé à la lecture, comme le détail. Pas de limitation de débit.

### `GET /transactions/v1/commandes/{id}`

Commande dont l'acheteur est le sujet du jeton ; sinon 404, comme une commande inexistante ou un identifiant mal formé.

### Déploiement (US-032)

La Place de marché (champ `titre` dans la réponse de « réserver ») doit être déployée **avant** Transactions. Un Transactions à jour face à une Place de marché antérieure répond 503 à toute réservation (titre absent) ; l'ordre inverse est sans effet (champ inconnu ignoré).

### Expiration (US-031)

`statut` vaut `expiree` dès que `echeance <= maintenant`, `en_attente_paiement` avant. Rien n'est stocké ni planifié : la colonne `statut` garde `en_attente_paiement` et le statut est dérivé à chaque lecture (`Order.AsOf`), sur `GET /commandes/{id}` comme sur la réponse de `POST /commandes`. « Maintenant » est l'horloge du service, injectée dans `commandes.NewService` (`time.Now` en production, horloge fixe en test) ; l'échéance vient de l'horloge de la base `marche`, un écart d'horloge de quelques millisecondes entre les deux est sans effet visible. Après l'échéance, un nouvel essai sur le même article reçoit de la Place de marché un nouveau `reservationId` (si l'article est toujours libre), donc une nouvelle commande (201) ; l'ancienne reste lisible, `expiree`.

## Montants et arrondi (EF-24, ENF-01)

Règle unique, dans `internal/montants` (`Compute`), en centimes entiers (`int64`, jamais de flottant) :

- `totalPaye = prix + fraisDePort` ;
- `commission = 5 % de totalPaye`, arrondie au centime le plus proche, **demi-centime arrondi au centime supérieur** : `(totalPaye × 500 + 5 000) div 10 000` ;
- `montantVendeur = totalPaye − commission`, donc `commission + montantVendeur = totalPaye` au centime près.

Exemples : 10 c → 1 c (0,5 arrondi vers le haut) ; 9 c → 0 c (0,45) ; 12 590 c → 630 c (629,5). Montant négatif, devises différentes ou total hors de `int64` : refus. Les mêmes égalités sont des contraintes `CHECK` de la table `commande`.

## Variables d'environnement

| Variable | Commande | Défaut | Contenu |
|---|---|---|---|
| `BASE_DE_DONNEES_URL` | toutes | obligatoire | `postgres://<rôle>:<mot de passe>@<hôte>:5432/transactions?sslmode=verify-full&sslrootcert=<paquet RDS>` ; rôle d'exécution pour `serve`, rôle de migration pour `migrate` |
| `BASE_DE_DONNEES_TLS_EXIGE` | toutes | `true` | `false` seulement en local |
| `OIDC_EMETTEUR` | `serve` | obligatoire | Émetteur exact, ex. `https://auth.<domaine>/realms/collector` |
| `OIDC_JEU_DE_CLES_URL` | `serve` | `<émetteur>/protocol/openid-connect/certs` | Adresse interne du jeu de clés |
| `OIDC_AUDIENCE` | `serve` | `transactions` | Audience exigée dans le jeton de l'acheteur |
| `OIDC_CLIENT_AUTORISE` | `serve` | `collector-web` | Valeur exigée de `azp` |
| `OIDC_JETON_URL` | `serve` | `<émetteur>/protocol/openid-connect/token` | Point de jeton pour le jeton de service, adresse interne si besoin |
| `OIDC_CLIENT_ID` | `serve` | `transactions` | Client confidentiel de Transactions |
| `OIDC_CLIENT_SECRET` | `serve` | obligatoire | Secret de ce client, tiré à chaque session (le même que `SECRET_CLIENT_TRANSACTIONS` à l'import du realm) ; monté dans le seul pod Transactions, jamais journalisé |
| `PLACE_DE_MARCHE_URL_INTERNE` | `serve` | obligatoire | Adresse du port interne de la Place de marché, ex. `http://place-de-marche-interne:8081` |
| `LIMITE_DEBIT_PAR_MINUTE`, `LIMITE_DEBIT_RAFALE`, `LIMITE_DEBIT_TIR_*` | `serve` | `30`, `10`, voir la Place de marché | Limitation de débit de `POST /commandes` |
| `ADRESSE_API` | `serve` | `:8080` | Écoute de l'API |
| `ADRESSE_GESTION` | `serve` | `:9090` | Écoute de la santé |
| `JOURNAL_NIVEAU` | toutes | `INFO` | `DEBUG`, `INFO`, `WARN`, `ERROR` |

## Base et rôles PostgreSQL attendus

Base `transactions`, créée par l'amorçage avant toute migration ([D07](../../docs/decisions/D07-donnees.md), MD1, MD2), cloisonnée des bases `marche` et `interactions` (AC8) :

```sql
CREATE ROLE transactions_migration LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD '…';
CREATE ROLE transactions_execution LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD '…';
CREATE DATABASE transactions OWNER transactions_migration;
REVOKE CONNECT ON DATABASE transactions FROM PUBLIC;
GRANT CONNECT ON DATABASE transactions TO transactions_migration, transactions_execution;
```

Puis, connecté en `transactions_migration` sur `transactions` : `REVOKE CREATE ON SCHEMA public FROM PUBLIC`. Les bases `marche` et `interactions` portent le même `REVOKE CONNECT … FROM PUBLIC` : `transactions_execution` ne s'y connecte pas, et réciproquement.

- `transactions_migration` : propriétaire de la base et du schéma, seul à exécuter du DDL ; secret livré au seul Job de migration.
- `transactions_execution` : propriétaire d'aucun objet ; nom fixe, la première migration lui accorde `SELECT` et `INSERT` sur les tables (`ALTER DEFAULT PRIVILEGES`). Ni `UPDATE`, ni `DELETE` : une commande ne change pas d'état en base dans le POC (« expirée » se lit sur l'échéance, voir « Expiration »).

Table `commande` (migration 00001) : `reservation_id` unique (`commande_une_par_reservation`), montants en centimes `bigint`, devise `EUR`, `CHECK` sur `total_paye = prix + frais_port` et `commission + montant_vendeur = total_paye`. Migration 00002 (US-032) : colonne `titre` (`CHECK` 1 à 200 caractères), commandes existantes reprises avec « Article sans titre », puis défaut retiré (tout `INSERT` fournit le titre) ; index `commande_par_acheteur_idx (acheteur_id, cree_le DESC, id DESC)` pour la liste.

## Exposition et cloisonnement (AC7, session AWS)

À poser avec le chart Helm, hors de ce lot :

- Service `ClusterIP` de la Place de marché pour le port interne 8081, absent de toute règle de l'ALB (seul `/transactions/v1/` de Transactions y est routé).
- Politique réseau : entrée du port 8081 de la Place de marché autorisée depuis les seuls pods Transactions ; sortie de Transactions limitée à la Place de marché (8081), à Keycloak (point de jeton et jeu de clés), à la base et au DNS (M-SYN-01, faute de TLS interne).
- Secret `OIDC_CLIENT_SECRET` dans le seul pod Transactions.
- Preuve : appel de « réserver » depuis Internet (aucune route) et depuis un pod de test (refus de la politique réseau).
