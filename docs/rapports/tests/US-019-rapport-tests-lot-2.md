# US-019 — Rapport de tests, lot 2 (Place de marché, module Comptes)

| Champ | Valeur |
|---|---|
| US | [US-019 — Mettre à vérifier les prix incohérents et les vendeurs à risque](../../../backlog/US-019-regles-prix-statut-vendeur.md) |
| Branche | `feat/US-019-comptes-suspension` |
| Date | 2026-11-06 |
| Fonction | Test |
| Périmètre | Place de marché : module Comptes minimal (inscription, suspension par l'exploitation, reprise par la migration 00006), demande `controle.demande` v1 (soumission et revalidation US-021), bout en bout avec le binaire Contrôle réel sur PostgreSQL et RabbitMQ (testcontainers-go) |
| Hors périmètre | AC1 et AC5 côté Contrôle, couverts par le [lot 1](US-019-rapport-tests.md) ; acceptation API Playwright `request` |
| Référentiel | AC2, AC3, AC4 de l'US ; contrat d'exploitation `place-de-marche suspendre <compte-id>` / `retablir <compte-id>` (README du service, § Comptes) ; D02 (module Comptes sans API) |
| Outils | Go 1.26, `go test`, golangci-lint 2.15.0 |
| Verdict | **Tous les AC sont couverts et conformes** ; aucune anomalie. Les deux points à clarifier du lot 1 (AC2 de bout en bout, module Comptes d'AC4) sont levés |

## Méthode

Scénarios Given/When/Then dérivés des AC, étiquetés `US019Lot2`. La sous-commande d'exploitation est jouée en boîte noire : le binaire `place-de-marche` est construit par le test et lancé comme un processus, avec le rôle `marche_migration` ; seuls le code de sortie et la sortie standard sont observés. Le vendeur porte dans son jeton un email, un nom, un prénom, un nom de famille et un identifiant usuel, pour prouver qu'aucun n'atteint la demande. Aucun commentaire dans le code de test.

| Fichier | Contenu | Tests |
|---|---|---|
| `services/place-de-marche/cmd/place-de-marche/comptes_bout_en_bout_test.go` | `TestUS019Lot2ComptesSurPostgreSQLRabbitMQEtControle` : Place de marché sur PostgreSQL et RabbitMQ réels, binaire Contrôle réel, binaire `place-de-marche` pour `suspendre` et `retablir` | 9 scénarios ordonnés, 10 cas d'erreur de commande |
| `services/place-de-marche/cmd/place-de-marche/comptes_migration_test.go` | `TestUS019Lot2MigrationInscritLesVendeursExistantsALaDateDeLeurPremierArticle` : base migrée par goose jusqu'à la version 4, articles historiques insérés par le rôle de migration, puis migrations complètes (deux fois) | 2 scénarios |

## Résultats par AC

| AC | Scénario | Attendu | Réel | Statut |
|---|---|---|---|---|
| AC4 | Vendeur neuf soumet une figurine à 45 € | Demande conforme v1 ; `seller` = exactement `account_id` (sujet du jeton), `member_since` (date UTC du jour), `suspended=false` ; ni email ni nom dans la demande ; article publié | Conforme | ✅ |
| AC2 | Brouillon créé, puis `suspendre <id>`, puis soumission | Commande en code 0, journal `comptes.suspension` avec l'id ; demande `suspended=true` ; article `refuse` / `compte_non_autorise`, aussi dans « mes articles » | Conforme (le compte existe dès la création du brouillon) | ✅ |
| AC2 | Vendeur suspendu modifie son article publié (US-021) | Nouvelle révision en contrôle, demande `suspended=true`, `refuse` / `compte_non_autorise`, hors catalogue | Conforme | ✅ |
| AC2 | Seconde `suspendre` sur un compte suspendu | Code 0, date de suspension inchangée, nouvelle soumission toujours refusée `compte_non_autorise` | Conforme | ✅ |
| AC2 | `retablir` deux fois, puis soumission, puis correction et resoumission de l'article refusé | Codes 0 ; demandes `suspended=false`, même `member_since` ; publiés | Conforme | ✅ |
| AC2 | `retablir` sur un compte jamais suspendu | Code 0 ; soumission publiée | Conforme | ✅ |
| AC2 | `suspendre` / `retablir` : UUID inconnu, `pas-un-identifiant`, `42`, vide, UUID tronqué, sans argument (×2), deux identifiants, compte existant suivi d'un intrus | Code non nul, aucun événement `comptes.suspension` ; aucun compte créé pour l'UUID inconnu ; le compte existant reste en règle ; le vendeur de l'UUID refusé, inscrit ensuite, n'est pas suspendu | Conforme dans les 10 cas | ✅ |
| AC3 | Vendeur inscrit le jour même, figurine à 300,01 € | Demande `member_since` du jour, `suspended=false` ; `a_verifier` / `vendeur_a_verifier` | Conforme | ✅ |
| AC3 | Compte inscrit il y a 40 jours (date posée en base), figurine à 300,01 € | Demande `member_since` = date d'inscription du compte ; publié | Conforme | ✅ |
| AC4 | Migration 00006 : vendeur à 3 articles (min 2025-03-14T10:00Z), vendeur à 1 article créé le 2025-07-15T23:30-02:00, vendeur à 2 articles à 23:59:59.999999Z / minuit | Un compte par vendeur, `inscrit_le` = min(`cree_le`) à l'instant près, non suspendu, sans date de suspension ; migrations rejouables | Conforme | ✅ |
| AC4 | Vendeurs repris créent et soumettent un nouvel article | `member_since` = date UTC du premier article (`2025-03-14`, `2025-07-16`, `2026-01-31`), pas la date du jour ; aucun compte supplémentaire | Conforme | ✅ |

« Le Contrôle ne lit ni base ni horloge » (AC4) reste prouvé par le lot 1.

## Non-régression

Toute la suite `services/place-de-marche` passe, dont les bouts en bout US-013/US-018/US-019 lot 1 et US-021.

## Exécution

| Commande | Résultat |
|---|---|
| `go test -count=1 ./...` dans `services/place-de-marche` (Docker) | ✅ `cmd/place-de-marche` ok (126 s) |
| `gofmt -l .` | ✅ aucun fichier |
| `go vet ./...` | ✅ |
| `golangci-lint run --config ../../.golangci.yml ./...` (2.15.0) | ✅ 0 issue |
| `go test -race` | Non exécuté (cgo indisponible localement) |

Conteneurs de test supprimés en fin de campagne par testcontainers-go.

## Points à clarifier

Aucun. Remarque sans écart : la reprise par migration conserve l'instant exact du premier article (`timestamptz`) ; la conversion en date UTC se fait à la construction de la demande, ce que prouve le vendeur créé à 23:30 en UTC−2 (`2025-07-16`).

Aucune anomalie : aucun comportement observé ne contredit un AC.
