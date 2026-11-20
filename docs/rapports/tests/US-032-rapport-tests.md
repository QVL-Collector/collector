# Rapport de tests — US-032 Suivre ses réservations dans l'espace acheteur

## Back

| Champ | Valeur |
|---|---|
| US | [US-032](../../../backlog/US-032-espace-acheteur.md) |
| Branche | `feat/US-032-espace-acheteur` (non commitée) |
| Date | 2026-11-20 |
| Fonction | Test |
| Périmètre | `services/transactions` : `GET /transactions/v1/commandes` (liste de l'acheteur, pagination, tri, état dérivé à la lecture), titre figé dans la commande (création, détail, liste), migration 00002 ; `services/place-de-marche` : titre renvoyé par « réserver » au port interne. AC1 à AC3 ; contrats `transactions/v1` (`listerMesCommandes`, `PageCommandes`, `titre`, `hors_limites`) et `place-de-marche/interne-v1` (`titre`) |
| Hors périmètre | Paiement (hors POC) |
| Outils | `go test`, testcontainers-go avec PostgreSQL, RabbitMQ et **vrai Keycloak** (banc d'US-030 et US-031 réutilisé), Place de marché en processus de test avec **`RESERVATION_DUREE` = `RESERVATION_DUREE_MAX` = 6 s**, Transactions construit et lancé en processus, goose (migration jusqu'à la version 1), kin-openapi ; tests unitaires de Transactions avec horloge injectable (`commandes.Clock`) |
| Verdict | **Tous les AC du périmètre back sont couverts et conformes** ; aucune anomalie |

### Méthode

Scénarios dérivés des AC et des contrats, étiquetés `US032…`. Trois niveaux :

- **Bout en bout** (`TestUS032EspaceAcheteur`, un `t.Run` par scénario sur le banc d'US-031, échéance de 6 s) : acheteurs et vendeurs créés dans Keycloak, jetons obtenus par le vrai flux du navigateur, articles publiés par le Contrôle, réservations par `POST /commandes`, listes par `GET /commandes`. L'échéance courte permet de franchir réellement l'échéance (AC1 réservation expirée dans la liste, AC2, titre figé). La garde `exigerAvantLEcheance` d'US-031 protège les lectures qui exigent une réservation encore active. Pagination et tri prouvés sur **25 commandes amorcées en base** (créations par groupes de 3 à la même seconde pour exercer le départage par `id`, une échéance sur deux passée) plus 3 commandes d'un autre acheteur.
- **Migration** (`TestUS032MigrationDuTitreSurDesCommandesExistantes`) : base `transactions` amorcée comme en production, migrée par goose **jusqu'à la version 1** avec les scripts du dépôt, deux commandes insérées par le rôle d'exécution, puis `transactions migrate` (binaire construit) applique 00002.
- **Unitaire** : `commandes` (service avec horloge réglable et registre simulé), `commandes/api` (routeur réel, vrai vérificateur de jetons sur un JWKS de test, service simulé), `commandes/marche` (client réel contre une Place de marché simulée).

### Livrables de test

| Fichier | Contenu |
|---|---|
| `services/place-de-marche/cmd/place-de-marche/reservation_espace_acheteur_test.go` | `TestUS032EspaceAcheteur` : 9 scénarios (AC1 ×2, AC2, AC3 ×2, pagination et tri, pagination hors bornes ×7, titre de « réserver », contrats) |
| `services/place-de-marche/cmd/place-de-marche/transactions_migration_titre_test.go` | `TestUS032MigrationDuTitreSurDesCommandesExistantes` : reprise des commandes existantes, contraintes 1 à 200 caractères, titre obligatoire (3 sous-tests) |
| `services/place-de-marche/cmd/place-de-marche/reservation_banc_test.go` | `Titre` ajouté à `commandeRecue` et `reservationRecue` |
| `services/transactions/internal/commandes/mes_commandes_test.go` | 8 tests : titre figé de la réservation à la commande, liste (montant, échéance, état), bascule `expiree` à l'horloge du service (1 ns avant, à l'échéance, 1 h après), seule la commande échue bascule, liste demandée pour l'acheteur, page transmise, bornes du contrat, titre hors contrat de la Place de marché → aucune commande |
| `services/transactions/internal/commandes/api/mes_commandes_test.go` | 9 tests : corps `{elements,page,taille,total}`, liste vide `[]`, acheteur du jeton seul (paramètre `acheteurId` ignoré), 9 paginations acceptées, 14 refus 400, authentification, erreur 500 sans détail, `titre` à la création et au détail, 404 du détail d'autrui identique à une commande inexistante |
| `services/transactions/internal/commandes/marche/titre_test.go` | 2 tests : titres acceptés (1 et 200 caractères, 200 caractères accentués, guillemets), titres refusés (absent, vide, 201 caractères, 201 accentués, `null`, nombre) |
| `services/transactions/internal/commandes/service_test.go` | Registre simulé : `ListOwned` (filtre par acheteur, tri `creeLe` puis `id` décroissants, pagination), `Title` recopié |
| `services/transactions/internal/commandes/api/handlers_test.go` | Service simulé : `OwnOrders` ; `Titre` dans la commande reçue |
| `services/transactions/internal/commandes/marche/client_test.go` | Réservations simulées avec `titre` (y compris les corps incomplets, pour que leur refus reste dû au champ manquant visé) |

### Résultats d'exécution

| Commande | Résultat | Durée |
|---|---|---|
| `go vet ./...` (`services/transactions`), avant réparation | Échec de compilation des tests : `registreSimule` sans `ListOwned`, `commandesSimulees` sans `OwnOrders` (attendu, changement d'interface) | — |
| `go test ./cmd/place-de-marche/ -run TestUS032 -count=1 -v` (première passe) | 8 scénarios PASS, 1 FAIL : **erreur de test** (compte « vendeur » du banc, qui porte aussi le rôle `acheteur`, supposé sans ce rôle). Puis, avec un compte au seul rôle `vendeur` : 200 également, car le realm donne `acheteur` à **tout** compte (`default-roles-collector`). Cas retiré du banc, gardé en unitaire | 75 s |
| Même commande après correction | `ok`, **9 scénarios PASS** et migration PASS, 0 skip | 78 s |
| `go test ./... -count=1 -v -timeout 60m` (`services/place-de-marche`), machine chargée (Vitest et Playwright lancés en parallèle) | 101 tests de premier niveau : 100 PASS, **1 FAIL hors US-032** : `TestUS024FiletDelaiCroissantAuxValeursDeProduction` (« 2 relances journalisées, 1 attendue »), relancé seul : **PASS** (7 s). `TestUS030…` (140 s), `TestUS031…` (113 s) et `TestUS032…` (111 s) PASS | 761 s |
| `go test ./... -count=1 -v -cover -timeout 60m` (`services/place-de-marche`), machine au repos, code final | `ok` dans les 2 paquets testés, **101 tests de premier niveau, 101 PASS, 0 skip, 0 échec**, dont `TestUS032EspaceAcheteur` (71 s, 9 scénarios PASS), `TestUS032MigrationDuTitre…` (4 s) et, sans régression, `TestUS030…` (104 s) et `TestUS031GarantiesDeLaReservation` (107 s) | 602 s |
| `go test ./... -count=1 -cover` (`services/transactions`) | `ok` dans 4 paquets, **51 tests de premier niveau, 0 skip, 0 échec** | 10 s |
| `go test ./... -count=1 -cover` (`libs/platform`) | `ok` dans tous les paquets testés, 0 échec | 15 s |
| `gofmt -l`, `go vet ./...`, `golangci-lint run --config <racine>/.golangci.yml` (`services/transactions`, `services/place-de-marche/cmd/place-de-marche`) | 0 issue (3 `gocognit` sur le premier jet, corrigés par extraction de fonctions) | — |
| Conteneurs | Arrêtés et supprimés par testcontainers en fin de test | — |

### Couverture Go par paquet

| Paquet | Couverture |
|---|---|
| `services/transactions/internal/commandes` | 84,0 % |
| `services/transactions/internal/commandes/api` | 93,8 % |
| `services/transactions/internal/commandes/marche` | 93,4 % |
| `services/transactions/internal/montants` | 100,0 % |
| `services/transactions/internal/commandes/stockage`, `migrations`, `config`, `cmd/transactions` | 0 % en unitaire : exercés par le binaire lancé dans les bancs de `place-de-marche` (liste, tri, pagination, migration), hors de la mesure `-cover` |
| `services/place-de-marche/cmd/place-de-marche` | 64,5 % |
| `services/place-de-marche/internal/config` | 84,8 % |
| `services/place-de-marche/internal/reservations`, `reservations/api`, `reservations/stockage` et autres paquets `internal` | 0 % mesuré dans le paquet : exercés de bout en bout par les bancs de `cmd/place-de-marche`, que `-cover` n'attribue qu'au paquet testé |
| `libs/platform/token` | 94,6 % |
| `libs/platform/serviceclient` | 95,0 % |
| `libs/platform/ratelimit` | 98,1 % |
| `libs/platform/message` | 82,8 % |
| `libs/platform/logging` | 82,4 % |
| `libs/platform/httpjson` | 77,9 % |
| `libs/platform/database` | 51,2 % |
| `libs/platform/httpserver` | 41,5 % |

### AC1 — L'acheteur voit, pour chaque réservation, l'article, le montant total, l'échéance et l'état

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Acheteur sans réservation | Liste vide | 200 `{"elements":[],"page":1,"taille":20,"total":0}` | Conforme |
| Une réservation active (80 € + 5 €) | Article, montant total, échéance, état | 1 élément : même `id` et `articleId`, `titre` = titre de l'article publié, `totalPaye` 8 500 centimes, `echeance` = celle de la création, `en_attente_paiement` | Conforme |
| Échéance passée, nouvelle réservation sur un autre article | Les deux réservations, chacune avec son état | Total 2, **la plus récente d'abord** ; seconde `en_attente_paiement`, première **`expiree`**, titres, montants et échéances exacts | Conforme |
| Titre figé : après l'échéance, le vendeur renomme l'annonce (200) | La réservation montre l'article réservé | Création, base (`commande.titre`), détail et liste gardent le titre d'origine | Conforme |
| « Réserver » au port interne (201 puis reprise 200) puis `POST /commandes` | Titre transmis | `titre` = titre de l'article dans les trois réponses | Conforme |
| Unitaire service : titre de la réservation | Recopié dans la commande | Placement, commande créée, détail et liste portent le titre | Conforme |
| Unitaire client : titre de 1, de 200 caractères, 200 caractères accentués (400 octets), guillemets et symboles | Accepté (contrat : 1 à 200) | Accepté, recopié tel quel | Conforme |
| Unitaire client : titre absent, vide, 201 caractères, 201 accentués, `null`, nombre | Place de marché hors contrat | `ErrMarketplaceUnavailable` ; avec le service, **aucune commande enregistrée** ; traduit en 503 par l'API (test d'US-030) | Conforme |

### AC2 — Échéance passée pendant la consultation : la liste rafraîchie montre « expirée »

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Bout en bout : liste avant l'échéance (6 s), puis rafraîchie après | `en_attente_paiement` puis `expiree` | `en_attente_paiement` puis **`expiree`**, même `id`, même échéance, mêmes montants, même titre | Conforme |
| 25 commandes amorcées, une échéance sur deux dans le passé | État dérivé à la lecture | Chaque élément `expiree` si l'échéance est passée, `en_attente_paiement` sinon | Conforme |
| Unitaire (horloge réglable) : liste relue 1 ns avant, à l'échéance exacte, 1 h après | `en_attente_paiement`, puis `expiree` | `en_attente_paiement`, **`expiree`**, `expiree` ; échéance et montant inchangés | Conforme |
| Unitaire : deux commandes, une seule échue | Seule l'échue bascule | Première `expiree`, seconde `en_attente_paiement` | Conforme |

### AC3 — B ne voit que ses réservations ; le détail d'une réservation de A répond 404

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| A réserve ; B (sans réservation) liste | B ne voit rien de A | Total 0, `elements` vide | Conforme |
| B liste avec `?acheteurId=<sujet de A>` | Idem | Total 0 (paramètre ignoré) ; en unitaire, le service est appelé pour l'acheteur du jeton | Conforme |
| B réserve à son tour ; A et B listent | Chacun les siennes | B : seulement sa commande ; A : seulement la sienne | Conforme |
| B lit le détail de la commande de A | 404 | **404**, corps identique octet pour octet à celui d'une commande inexistante ; ni l'article, ni le titre, ni le sujet de A, ni l'échéance dans le corps ; A relit la sienne (200) | Conforme |
| Liste sans jeton, jeton illisible | Refus | 401 | Conforme |
| Liste avec le jeton de service de Transactions | Refus | 401 | Conforme |
| Unitaire : jeton expiré, d'une autre audience ; jeton sans le rôle `acheteur` | 401 ; 403 | 401, 401 ; **403** ; service jamais appelé | Conforme |

### Pagination et tri (contrat `listerMesCommandes`)

| Scénario | Attendu (contrat) | Réel | Statut |
|---|---|---|---|
| 25 commandes, sans paramètre | Page 1, taille 20, total 25, `creeLe` puis `id` décroissants | Conforme, ordre identique au tri attendu, départage par `id` aux créations simultanées | Conforme |
| `?page=2`, `?page=3`, `?page=10000&taille=50` | 5 restants ; page au-delà vide avec le total | 5 éléments ; vide, total 25 ; vide, total 25 | Conforme |
| `?taille=50` ; 25 pages de `taille=1` | Même ordre | Identique au tri attendu dans les deux parcours | Conforme |
| 3 commandes d'un autre acheteur en base | Absentes | Absentes, total 25 | Conforme |
| Bornes acceptées (unitaire) : `page` 1 et 10000, `taille` 1 et 50, défauts, ordre des paramètres, paramètre inconnu | 200 et page transmise | Conforme (9 cas) | Conforme |
| Hors bornes : `page` 0, -1, 10001 ; `taille` 0, 51, -3 | 400 `saisie-invalide`, `hors_limites` | Conforme (6 cas, et 4 de bout en bout) | Conforme |
| Invalides : `page=`, `taille=`, `abc`, `1.5`, répétés | 400 `saisie-invalide`, `invalide` | Conforme (6 cas, et 3 de bout en bout) ; service jamais appelé | Conforme |
| `page` et `taille` invalides ensemble | Une erreur par champ | `[{page, hors_limites}, {taille, invalide}]` | Conforme |

### Migration 00002

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Deux commandes créées en version 1, puis `transactions migrate` | Commandes reprises, titre valide | Reprises avec le titre « Article sans titre », prix inchangé | Conforme |
| Rôle d'exécution : titre de 1 et de 200 caractères accentués | Accepté | Accepté | Conforme |
| Rôle d'exécution : titre vide, 201 caractères | Refusé | Violation de contrainte `23514` | Conforme |
| Nouvelle commande sans titre | Refusée | `23502` (NOT NULL) | Conforme |

### Contrats OpenAPI

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| 61 échanges de Transactions du banc (passe finale), dont **50 listes** (200, 400), création (201), détail (200, 404), refus (401) | Conformes à `transactions/v1` | Conformes | Conforme |
| Échanges directs de « réserver » (201, 200) avec `titre` | Conformes à `place-de-marche/interne-v1` | Conformes | Conforme |

### Anomalies

Aucune : aucun comportement observé ne contredit un AC.

### Observations (sans anomalie)

- **Rôle `acheteur` par défaut** : le realm attribue `acheteur` à tout compte (`default-roles-collector`). Le refus 403 d'un compte sans ce rôle n'est donc pas reproductible de bout en bout ; il est prouvé en unitaire (jeton signé sans le rôle).
- **`page` au-delà des entiers 64 bits** (`?page=99999999999999999999`) : 400 avec le code `invalide`. Le contrat dit « entier entre les bornes (sinon `hors_limites`) » et « non entier → `invalide` » ; un nombre non représentable peut se lire des deux façons. Le test exige seulement un 400 `saisie-invalide` sur `page`.
- **Migration** : les commandes antérieures reçoivent le titre de repli « Article sans titre » (pas le titre réel de l'article). Aucun AC ne porte sur la reprise ; relevé pour information.
- **Instabilité hors US** : `TestUS024FiletDelaiCroissantAuxValeursDeProduction` a échoué une fois dans la suite complète lancée en même temps que Vitest et Playwright, puis est passé seul. Aucun lien avec US-032.

## Front

| Champ | Valeur |
|---|---|
| US | [US-032](../../../backlog/US-032-espace-acheteur.md) |
| Branche | `feat/US-032-espace-acheteur` (non commitée) |
| Date | 2026-11-20 |
| Fonction | Test |
| Périmètre | `web/` : écran « Mes réservations » sur `/acheteur`, entrée de navigation, rafraîchissement, états vide et erreur, mode démo, titre sur la confirmation |
| Référentiel | AC de l'US ; contrat `transactions/v1` (`listerMesCommandes`, `titre`) ; libellés FR/EN écrits en dur |
| Méthode | Boîte noire : application complète (Vitest/jsdom, puis build de production servi à Playwright), Transactions simulé (liste paginée, 503 à la demande) ; échéances calculées par rapport à l'heure courante |
| Verdict | **Conforme** : tous les AC du périmètre front sont couverts et conformes ; aucune anomalie |

### 1. Livrables de test

| Fichier | Contenu |
|---|---|
| `web/src/acceptance/US-032/buyerSpaceHarness.ts` | Libellés FR/EN, réservations en attente et expirées, ouverture de l'espace, lecture des éléments de liste |
| `web/src/acceptance/US-032/AC1-mes-reservations.test.tsx` | 15 tests : article, montant total, échéance (`<time datetime>`) et état de chaque réservation, ordre reçu, jeton envoyé, lien vers le détail, état vide, erreur puis « Réessayer », entrée de navigation, compte vendeur refusé, titre sur la confirmation ; FR/EN |
| `web/src/acceptance/US-032/AC2-expiration-pendant-la-consultation.test.tsx` | 5 tests : échéance réelle franchie (4 s) puis liste rafraîchie → « Expirée » (FR/EN), rafraîchissement périodique sans action, retour sur l'onglet, réservation expirée stable |
| `web/src/acceptance/US-032/AC4-accessibilite.test.tsx` | 6 tests axe-core : liste, état vide, état d'erreur ; FR/EN |
| `web/e2e/US-032/espace-acheteur.ts` | Libellés, réservations, ouverture de l'espace, configuration du mode démo |
| `web/e2e/US-032/mes-reservations.spec.ts` | Playwright : 9 tests par langue (AC1 ×7 dont mode démo nominal, vide et erreur ; AC2 ×2 avec horloge du navigateur contrôlée et retour sur l'onglet) |
| `web/e2e/US-032/AC4-accessibilite.spec.ts` | Playwright, axe-core WCAG 2.1 A/AA : 4 tests par langue (liste, liste démo, vide, erreur) |
| `web/src/acceptance/US-030/reservationHarness.ts` | `titre` dans `OrderWire`, `GET /commandes` simulé (`listAnswers`, `listRequests`) |
| `web/e2e/support/reservation.ts` | `titre` dans `CommandeEmise`, `GET /commandes` simulé (`listeIndisponible`, `listes`) |
| `web/e2e/support/ecrans.ts` | Écran « acheteur, mes réservations » ajouté au balayage axe-core d'US-025 |

### 2. Exécutions (dans `web/`)

| Commande | Résultat |
|---|---|
| `npx vitest run src/acceptance/US-030 src/acceptance/US-031` avant réparation | **31 échecs sur 73** : commandes simulées sans `titre`, refusées par le front (attendu, contrat modifié) |
| Même commande après ajout de `titre` au banc | 8 fichiers, **73 verts** |
| `npx vitest run src/acceptance/US-032` (première passe) | 25 verts, 1 échec : **erreur de test** (`visibilitychange` émis sur `document` au lieu de `window`) ; corrigé : vert |
| `npx tsc -b` | Vert |
| `npx eslint src e2e` | Vert, 0 avertissement (1 `no-unnecessary-condition` sur le premier jet, corrigé) |
| `npx prettier --check .` | Vert |
| `npx vitest run` (suite complète) | **46 fichiers, 540 tests, 540 verts, 0 ignoré** (188 s) ; aucun des tests instables connus (US-008 AC10, US-025 AC5, US-008 AC2) n'a échoué |
| `E2E_LANGUES=fr,en npx playwright test e2e/US-032` | **26 verts, 0 ignoré** (45 s) |
| `E2E_LANGUES=fr,en npx playwright test e2e/US-025/AC4-axe-ecrans.spec.ts -g réservations` | 2 verts |
| `E2E_LANGUES=fr,en npx playwright test` (suite complète) | **135 verts, 0 ignoré** (3,8 min) |

### 3. Résultats par AC

#### AC1 — Article, montant total, échéance et état de chaque réservation

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Une réservation en attente et une expirée (FR/EN, Vitest et Playwright) | Pour chacune : article, montant total, échéance, état | Liste sémantique « Réservations » / « Reservations » de 2 éléments ; titre, total « 125,90 € » / « €125.90 », `<time datetime>` égal à l'échéance reçue ; « En attente de paiement » / « Awaiting payment » et « Expirée » / « Expired », jamais les deux sur un même élément | Conforme |
| Trois réservations reçues de la plus récente à la plus ancienne | Ordre conservé | Même ordre | Conforme |
| Requête envoyée | Liste de l'acheteur connecté | `GET /transactions/v1/commandes` avec `Authorization: Bearer …`, sans identifiant d'acheteur | Conforme |
| Lien « Voir la réservation « titre » » (FR/EN) | Mène au détail | `/acheteur/commandes/{id}` ; le détail montre le titre | Conforme |
| Entrée de navigation « Mes réservations » / « My reservations » | Mène à l'espace | `/acheteur`, titre « Mes réservations » | Conforme |
| Aucune réservation | Écran compréhensible | « Aucune réservation pour le moment » et lien « Parcourir le catalogue » vers `/` | Conforme |
| Transactions en 503, puis rétabli | Erreur annoncée, reprise | « Vos réservations sont momentanément indisponibles », « Réessayer » affiche la liste | Conforme |
| Mode démo nominal, vide, erreur (Playwright) | Écran utilisable sans back | Liste avec au moins une réservation « Expirée » ; état vide ; état d'erreur | Conforme |
| Compte au seul rôle vendeur sur `/acheteur` | Pas d'accès aux réservations | « Accès réservé », aucune réservation affichée | Conforme |
| Confirmation de commande (FR/EN) | Titre de l'article | Titre figé affiché | Conforme |

#### AC2 — Échéance passée pendant la consultation

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Vitest, horloge réelle : réservation affichée « En attente », échéance à +4 s, le serveur la renvoie `expiree` dès l'échéance (FR/EN) | Au rafraîchissement, « Expirée » | « Expirée » affichée après le rafraîchissement suivant (≈ 30 s), nouvelle requête de liste observée | Conforme |
| Vitest : échéance lointaine, serveur qui renvoie `expiree` | Rafraîchissement périodique sans action | « Expirée » en ≈ 30 s | Conforme |
| Playwright, horloge du navigateur avancée de 2 min (FR/EN) | Idem | « Expirée », plus d'« En attente de paiement », nouvelle requête de liste | Conforme |
| Retour sur l'onglet (`focus` et `visibilitychange`), Vitest et Playwright | La liste se rafraîchit | « Expirée » aussitôt (test complet en moins d'une seconde) | Conforme |
| Réservation déjà expirée, liste rafraîchie | Reste lisible | Toujours « Expirée », contenu identique | Conforme |

#### AC4 — Accessibilité de la liste

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| axe-core jsdom : liste, état vide, état d'erreur (FR/EN) | 0 violation serious ou critical | 0 | Conforme |
| axe-core Playwright WCAG 2.1 A/AA : liste, liste démo, état vide, état d'erreur (FR/EN) | 0 violation serious ou critical | 0 | Conforme |
| Balayage des écrans d'US-025 avec l'écran « acheteur, mes réservations » (FR/EN) | 0 violation serious ou critical | 0 | Conforme |

### 4. Anomalies

Aucune.

### 5. Remarques

- AC3 relève du back (cloisonnement par le jeton, 404 du détail d'autrui) ; le front n'envoie aucun identifiant d'acheteur et affiche ce que Transactions renvoie.
- AC2 côté front : quand l'échéance passe à l'écran, l'état bascule au rafraîchissement suivant (périodique, ≈ 30 s, ou retour sur l'onglet) et non à la seconde de l'échéance. L'AC ne demande que l'état « expirée » une fois la liste rafraîchie : conforme.
- La table « Tests attendus » de l'US cite Playwright `request` pour AC1 à AC3 : ces preuves sont faites par le banc Go contre les vrais services (vraie base, vrai Keycloak, échéance réelle de 6 s), plus fidèle qu'un `request` Playwright sur des services simulés. Playwright prouve le parcours de l'acheteur et l'accessibilité (AC1, AC2, AC4).
