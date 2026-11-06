# US-019 — Revue de code, lot 2 (Place de marché, module Comptes)

| Champ | Valeur |
|---|---|
| US | [US-019 — Mettre à vérifier les prix incohérents et les vendeurs à risque](../../../backlog/US-019-regles-prix-statut-vendeur.md) |
| Branche | `feat/US-019-comptes-suspension` (base `origin/main` `4079387`, commit `60facfd`) |
| Date | 2026-11-06 |
| Périmètre | `git diff origin/main...HEAD` : 13 fichiers. `services/place-de-marche` : module `internal/comptes` (type, stockage), stockage Annonces, encodeur de demande, migration 00006, sous-commandes `suspendre` / `retablir`, README, tests d'intégration et de migration, [rapport de tests lot 2](../tests/US-019-rapport-tests-lot-2.md) |
| Références | [D02](../../decisions/D02-decoupage-applicatif.md) (module Comptes minimal, interne, sans API ni jointure ni événement ; suspension par sous-commande validée par Martin), [revue lot 1](US-019-revue-code.md), `services/place-de-marche/README.md` |
| Verdict | **Prêt à fusionner** (0 bloquante, 0 majeure, 3 mineures, 2 suggestions) |

## Vérifications exécutées

| Contrôle | Résultat |
|---|---|
| `gofmt -l .` | aucun fichier |
| `go vet ./...` | 0 problème |
| golangci-lint `--config ../../.golangci.yml ./...` | 0 issue |
| `go test -count=1 ./...` (Docker, testcontainers) | vert, `cmd/place-de-marche` ok (130 s) |
| Commentaires ajoutés dans le code Go | aucun (en-têtes `-- +goose Up` seuls, imposés par goose) |
| Mentions d'outils de génération dans le diff | aucune |
| Usages de `stockage.NewStore` (signature changée) | `serve.go:50`, `integration_test.go:153`, `comptes_migration_test.go:142` : tous mis à jour |
| Restes de `memberSinceColumn`, `scanReviewRequest`, `enqueueReview` | aucun |
| Dépendances entre paquets | `internal/comptes` et `internal/comptes/stockage` n'importent rien d'Annonces ; `annonces/stockage` importe le seul type `comptes` |

## Points forts

- Sens des dépendances correct : l'interface `SellerAccounts` est déclarée côté consommateur (`internal/annonces/stockage/postgres.go:121-124`), implémentée par `internal/comptes/stockage/postgres.go`, câblée dans `serve.go:50`. Aucune jointure ni clé étrangère entre `compte` et `article` ; ni route HTTP ni événement, conforme à D02.
- Inscription idempotente dans la transaction de l'appelant : `CreateDraft` passe en `pgx.BeginFunc` (`postgres.go:136-149`) ; soumission et revalidation partagent `requestReview` → `sellerStatus` (`postgres.go:242-268`), qui inscrit puis lit dans la même transaction que l'outbox. Le vendeur dont le brouillon a été créé par un ancien réplica pendant un déploiement est donc inscrit au plus tard à la soumission.
- Factorisation : `scanReviewRequest` et `enqueueReview` remplacés par un seul `requestReview` ; `submitInTransaction` et `reviseInTransaction` reviennent à `scanArticle`, la sous-requête `memberSinceColumn` disparaît des requêtes Annonces.
- Migration 00006 : `inscrit_le = min(cree_le)` en `timestamptz` (instant exact conservé), contrainte `compte_suspension_datee` (`suspendu = (suspendu_le IS NOT NULL)`) qui rend l'état incohérent impossible ; `DELETE` refusé au rôle d'exécution par les privilèges par défaut de 00001 et prouvé par `integration_test.go:176`.
- `SetSuspension` idempotent et cohérent avec la contrainte : `coalesce(suspendu_le, now())` garde la date d'une suspension déjà posée, `CASE` sans `ELSE` remet `NULL` au rétablissement (`comptes/stockage/postgres.go:21-24`) ; compte inconnu détecté par `RowsAffected() == 0`, sans création.
- Sous-commandes : `commandOf` sépare commande et opérandes, `accountIDOf` exige exactement un UUID valide (`comptes.go:43-48`), toute erreur remonte à `run` qui journalise `arret.erreur` et sort en code 1 ; succès journalisé `event=comptes.suspension` avec `compte_id` et `suspendu`, sans autre donnée.
- `member_since` converti en UTC avant `Format(time.DateOnly)` (`controle/requests.go:59`) : la mineure laissée par la revue du lot 1 est levée, et prouvée par le vendeur à 23:30 en UTC−2 du test de migration.
- AC4 : la demande ne porte que `account_id`, `member_since`, `suspended` ; `comptes.Status` n'expose que `RegisteredAt` et `Suspended`, il n'existe aucune autre donnée de compte à divulguer.
- README à jour : sous-commandes, rôle, section « Comptes », texte de la demande.

## Constats

### Bloquante

Aucune.

### Majeure

Aucune.

### Mineure

1. `services/place-de-marche/internal/migrations/00006_inscrire_les_comptes.sql:2` — les privilèges par défaut de 00001 donnent `SELECT, INSERT, UPDATE` sur `compte` à `marche_execution`. Le service n'a besoin que de `INSERT` (inscription) et `SELECT` (statut) ; avec `UPDATE`, le rôle d'exécution peut lever une suspension (`UPDATE compte SET suspendu = false, suspendu_le = NULL`), alors que le README réserve la suspension au rôle de migration et que D02 décrit un module en lecture seule pour Annonces. Le précédent existe déjà (`00005:24`, `REVOKE UPDATE ON decision_admin`). Ajouter `REVOKE UPDATE ON compte FROM marche_execution;` et `exigerRefusDeDroit(t, pool, "UPDATE compte SET suspendu = false, suspendu_le = NULL")` dans `integration_test.go` à côté de la ligne 176. → `@qvl-backend`
2. `services/place-de-marche/README.md:58-59` — la colonne « Commande » vaut toujours « les deux » pour `BASE_DE_DONNEES_URL` et `BASE_DE_DONNEES_TLS_EXIGE`, alors que quatre commandes les lisent (`config.LoadDatabase` est aussi appelé par `comptes.go:24`). Écrire « toutes » (ou lister `serve`, `migrate`, `suspendre`, `retablir`) ; idem ligne 72 pour `JOURNAL_NIVEAU`, lu par `run` pour toute commande. → `@qvl-backend`
3. `services/place-de-marche/internal/annonces/stockage/postgres.go:121-124` et `README.md` § Comptes — l'interface lue par Annonces contient une écriture (`Enroll`, `INSERT … ON CONFLICT DO NOTHING`), alors que D02 § 3 décrit un statut « en lecture seule, consulté par le module Annonces ». Le choix se défend (l'ancienneté réelle exige une date d'inscription, sans API ni jointure), mais il élargit D02 sans trace : le noter dans D02 (ou dans le Suivi de l'US) comme précision validée par Martin, pour que l'écart ne ressorte pas à l'audit de fin de sprint. → orchestrateur (décision)

### Suggestion

1. `services/place-de-marche/cmd/place-de-marche/comptes.go:28` — la sous-commande Comptes ouvre sa connexion par `annonces/stockage.Connect`. La fonction n'a rien de propre à Annonces ; la déplacer dans un paquet commun (`internal/base` ou `internal/config`) éviterait que le chemin d'exploitation Comptes dépende du paquet de stockage Annonces.
2. `services/place-de-marche/internal/comptes/stockage/postgres.go:42-55` — `Status` renvoie `ErrAccountNotFound` sur `pgx.ErrNoRows`, branche inatteignable aujourd'hui puisque `sellerStatus` inscrit juste avant dans la même transaction. Garder le contrôle est raisonnable (contrat de l'interface) ; à défaut de test, il reste du code défensif non couvert.

## Verdict

Bloquante : 0 · Majeure : 0 · Mineure : 3 · Suggestion : 2. Lot 2 prêt à fusionner ; la mineure 1 (droit `UPDATE` du rôle d'exécution sur `compte`) est la seule à portée sécurité, à traiter de préférence avant la fusion car elle se corrige en deux lignes.
