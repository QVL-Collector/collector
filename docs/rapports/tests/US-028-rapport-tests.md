# Rapport de tests — US-028 Être notifié de la décision de publication de son article

## Back

| Champ | Valeur |
|---|---|
| US | [US-028](../../../backlog/US-028-notification-vendeur.md) |
| Branche | `feat/US-028-notification-vendeur` |
| Date | 2026-11-06 |
| Fonction | Test |
| Périmètre | `services/interactions`, `services/place-de-marche` (émission), `libs/platform/message`, `contracts/messages/publication.decidee` : AC1, AC2 (côté API, moins de 10 s), AC3, AC4 (côté API et message : seulement des codes), AC5, AC6, AC7, AC8, AC9 ; décisions [D02](../../decisions/D02-decoupage-applicatif.md), [D05](../../decisions/D05-communication-interservice.md), [D07](../../decisions/D07-donnees.md) |
| Hors périmètre | AC10 et rendu FR/EN (section Front) ; déploiement (chart, identité, base en cluster) |
| Outils | `go test`, testcontainers-go 0.44.0 avec PostgreSQL 18.3 et RabbitMQ 4.3.7 (topologie et politiques de `infra/canal/broker`), binaires du Contrôle et d'Interactions construits et lancés en processus, kin-openapi (contrat `contracts/api/interactions/v1.openapi.yaml`), `rabbitmqctl` dans le conteneur |
| Verdict | **Tous les AC du périmètre back sont couverts et conformes** ; aucune anomalie |

### Méthode

Scénarios dérivés des AC, étiquetés `US028ACn`. Trois niveaux :

- **Unitaire** : contrat `publication.decidee` v1 (`libs/platform/message`), consommateur des décisions avec un enregistreur en mémoire, API de liste avec un vrai vérificateur de jetons et une liste en mémoire.
- **Intégration Interactions** : PostgreSQL (bases `interactions` et `marche` amorcées selon le SQL du README, `REVOKE CONNECT … FROM PUBLIC`), RabbitMQ, `migrate` (deux fois) puis `serve` en processus. Les décisions sont publiées par l'identité `marche` sur `pdm.evenements`.
- **Bout en bout** : PostgreSQL commun aux bases `marche` et `interactions` (migrations réelles), RabbitMQ, Place de marché (`serve`), binaire du Contrôle, binaire d'Interactions (`migrate`, puis `serve`). Le parcours : un vendeur crée et soumet un article, le Contrôle tranche ou un administrateur décide, l'outbox est relayée sur le broker, Interactions crée la notification, et la liste est lue par `GET /interactions/v1/notifications` avec un jeton d'audience `interactions`. Un abonnement de test lié à `pdm.evenements` (posé par `admin-canal`) capture les décisions réellement diffusées. La panne de l'outbox est simulée par un déclencheur qui refuse l'insertion des seules lignes `publication.decidee`.

### Livrables de test

| Fichier | Contenu |
|---|---|
| `contracts/messages/publication.decidee/exemples/v1/invalides/` | 17 exemples invalides : `cle_dupliquee` (2), `contrat_inconnu` (3 : version 2, version absente, type inconnu), `json_invalide` (1), `schema_non_respecte` (11 : issue ou motif hors liste, motif en texte libre, publié avec motif, refusé ou à vérifier sans motif, titre de l'annonce, email comme vendeur, révision 0, vendeur ou date absents) |
| `libs/platform/message/publication_decidee_test.go` | Schéma de la décision (membres requis, vendeur opaque, issues et motifs fermés), chaque issue × motif accepté, refus hors liste ou de version inconnue, aucun champ de contenu saisi accepté, exemples valides limités aux codes et identifiants |
| `libs/platform/message/contract_test.go`, `support_test.go`, `fuzz_test.go` | `publication.decidee` ajouté à `TestUS012AC2_…` ; le fuzz connaît le 3e contrat et ses motifs d'administrateur |
| `services/interactions/internal/notifications/canal/decisions_test.go` | Consommateur : 9 tests (AC2, AC3, AC4, AC5, AC9) |
| `services/interactions/cmd/interactions/api_test.go` | API : 5 tests (AC4, AC6) |
| `services/interactions/cmd/interactions/banc_test.go`, `jetons_test.go`, `contrat_test.go` | Banc PostgreSQL et RabbitMQ, jetons de test (13 jetons invalides), conformité OpenAPI |
| `services/interactions/cmd/interactions/integration_test.go` | `TestUS028IntegrationInteractions` : 14 scénarios (AC2 à AC8, contrat OpenAPI) |
| `services/place-de-marche/cmd/place-de-marche/notification_vendeur_test.go` | `TestUS028BoutEnBoutPlaceDeMarcheInteractions` : 11 scénarios (AC1, AC2, AC3, AC4, AC6, AC8, AC9) |
| `services/interactions/go.mod`, `go.sum` | Dépendances de test : testcontainers-go et son module postgres 0.44.0, amqp091-go 1.16.0, kin-openapi 0.152.0 (mêmes versions que la Place de marché) |

#### Tests existants réparés

| Fichier | Cause | Correction |
|---|---|---|
| `integration_test.go`, `comptes_migration_test.go`, `filet_de_relance_test.go` | Nouvelle signature de `stockage.NewStore` (compilation) | Aide `depotPostgreSQL(t, pool)` qui passe `diffusion.NewDecisionEncoder(validateur)` en 3e argument. `canal_fictif_test.go` n'appelle pas `NewStore` : inchangé |
| `bout_en_bout_test.go`, `revalidation_bout_en_bout_test.go`, `comptes_bout_en_bout_test.go`, `controle_ferme_test.go` | L'outbox contient maintenant aussi les lignes `publication.decidee` : la « dernière demande » ou le nombre de demandes d'un article étaient faux (US-013, US-021) | `outboxDe` renommé `demandesEnOutboxDe` et filtré sur `controle.demande`. Nouvelle `lignesDOutbox(t, base, id, type)` |
| `controle_ferme_test.go` (US-024 MC9) | Le rejeu republie aussi les décisions, qui ne produisent aucun résultat de contrôle : on attendait autant de `controle.resultat_ignore` que de lignes rejouées | La trace attend toujours toutes les lignes ; les doublons ignorés attendus se limitent aux demandes de la fenêtre |
| `libs/platform/message/fuzz_test.go` | Déjà rouge avant mes modifications : les exemples `publication.decidee` étaient des « contrats inconnus » | Contrat et motifs d'administrateur reconnus |

### Résultats d'exécution

| Commande | Résultat | Durée |
|---|---|---|
| `go test ./... -count=1 -v` (`libs/platform`) | `ok` (`httpjson`, `httpserver`, `logging`, `message`, `ratelimit`, `token`), 71 tests de premier niveau, 0 skip, 0 échec | 5 s |
| `go test ./... -count=1 -v` (`contracts`) | Aucun fichier de test (les exemples sont joués par `libs/platform/message`) | 0 s |
| `go test ./... -count=1 -v` (`services/controle`) | `ok` (`canal`, `coordonnees`, `regles`), 0 skip | 3 s |
| `go test ./... -count=1 -v` (`services/interactions`) | `ok` (`cmd/interactions` 25 s, `canal` 0,8 s), 15 tests de premier niveau, 0 skip, dont `TestUS028IntegrationInteractions` (testcontainers réels) | 27 s |
| `go test ./... -count=1 -v -timeout 60m` (`services/place-de-marche`) | `ok`, 92 tests de premier niveau, **0 skip**, 0 échec ; dont `TestUS028BoutEnBoutPlaceDeMarcheInteractions` (40 s) et, sans régression, `TestUS013…`, `TestUS021…`, `TestUS024…` | 282 s |
| `golangci-lint run --config ../../.golangci.yml ./...` (v2.15.0), dans les 5 modules | 0 issue partout | — |
| Conteneurs | Arrêtés et supprimés par testcontainers en fin de test | — |

### AC1 — Décision écrite dans l'outbox dans la même transaction, puis diffusée

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Le Contrôle publie / refuse (coordonnées) / met à vérifier (catégorie inconnue) | Événement en outbox puis diffusé | Une seule ligne `publication.decidee` pour la révision décidée, vers `pdm.evenements` / `publication.decidee.v1`, publiée ; données = article, révision, vendeur (sujet du jeton), issue, motif, `decided_at` RFC 3339 | Conforme |
| Décision de l'administrateur : publier, refuser avec chacun des 5 motifs | Événement en outbox puis diffusé | Une ligne par décision, conforme, publiée | Conforme |
| Décision de l'admin alors que l'écriture de l'événement échoue | Rien d'appliqué (même transaction) | Réponse ≥ 500 ; article toujours `a_verifier`, même révision, même `modifieLe`, aucune ligne `decision_admin`, aucun événement ni notification de plus ; une fois l'outbox rétablie, la même décision passe et est notifiée | Conforme |
| Résultat du Contrôle alors que l'écriture de l'événement échoue | Rien d'appliqué | Article `en_controle`, aucune ligne `resultat_controle`, aucun événement ni notification ; outbox rétablie : article publié et événement écrit entre 0,2 et 3 s plus tard | Conforme |
| Résultat en double (même révision, même issue ou issue différente) | Aucun nouvel événement | Deux `controle.resultat_ignore`, aucune ligne de décision de plus | Conforme |
| Résultat périmé (révision antérieure après modification et nouveau contrôle) | Aucun nouvel événement | Ignoré ; 2 décisions en outbox (une par révision appliquée), 2 notifications « publié » | Conforme |

### AC2 — Une seule notification, listée en moins de 10 secondes

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Bout en bout, décision du Contrôle (de la soumission à la liste) | Moins de 10 s | 0,6 à 1,0 s | Conforme |
| Bout en bout, décision de l'administrateur (de la réponse HTTP à la liste) | Moins de 10 s | 0,5 s | Conforme |
| Bout en bout, une décision | Une seule notification | Total 1, une seconde plus tard encore 1 ; à vérifier puis décision de l'admin : 2 notifications pour l'article, une par décision | Conforme |
| Intégration, diffusion par `marche` | Moins de 10 s | 30 ms ; une seule notification, article, révision, issue et motif exacts | Conforme |
| Chaque issue × chaque motif (19 décisions) | Une notification chacune | 19 notifications, toutes exactes | Conforme |
| Unitaire : base indisponible | Décision non perdue | Erreur rendue, message non écarté (nouvel essai) | Conforme |

### AC3 — Même événement livré deux fois : une seule notification

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Intégration : même message publié deux fois | Une notification | Total 1, journal `notifications.doublon_ignore`, rien dans `.ecartes` | Conforme |
| Intégration : seconde instance d'Interactions, même décision relivrée 4 fois | Une notification | 4 doublons ignorés, répartis sur les deux instances ; total 1 sur les deux instances | Conforme |
| Unitaire : deux livraisons, la seconde marquée relivrée | Une notification, deux acquittements | Même décision identifiée par l'`id` de l'événement, 1 notification | Conforme |
| Unitaire : deux décisions du même article (identifiants différents) | Deux notifications | 2 | Conforme |

### AC4 — Aucun contenu saisi par un utilisateur (côté API et message)

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Bout en bout : titre, description, téléphone et email saisis par le vendeur | Absents de l'événement et de la liste | Aucune de ces valeurs dans les corps diffusés ni dans la réponse de l'API | Conforme |
| Réponse de l'API | Seulement des codes et des identifiants | Champs exactement `id`, `articleId`, `revision`, `issue`, `motif`, `decideLe` ; `motif` null pour « publié », code de la liste fermée sinon ; aucun texte libre | Conforme |
| Contrat : champ de texte (`title`, `description`, `comment`…) dans les données ou l'enveloppe | Refusé | `schema_non_respecte`, sans recopier la valeur reçue dans le refus | Conforme |
| Unitaire : décision enregistrée | Issue et motif dans les listes fermées | Conforme | Conforme |

### AC5 — Message invalide écarté vers `.ecartes` après un nombre borné d'essais, sans bloquer les suivants

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Intégration : 17 exemples invalides, 70 Kio, titre ajouté, puis une décision valide | Écartés, la suivante traitée | Les 19 dans `interactions.publication-decidee.ecartes`, chacun après **une seule livraison** (`x-death` `rejected`, `count` 1) ; aucune notification pour eux ; décision suivante notifiée ; file de travail vide | Conforme |
| Décision publiée par `admin-canal` au lieu de `marche` | Écartée | Dans `.ecartes`, aucune notification | Conforme |
| Politique effective de la file | Essais bornés | `delivery-limit` 4, `dead-letter-exchange` `collector.ecartes`, clé `interactions.publication-decidee`, `at-least-once` | Conforme |
| Unitaire : chaque exemple invalide, corps vide, tableau, 64 Kio + 1, type `controle.*`, issue altérée, description ajoutée | `Reject`, aucune notification | Conforme ; un message valide passe ensuite | Conforme |

### AC6 — Vendeur B ne voit que ses notifications ; 401 sans jeton ou jeton invalide

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Bout en bout et intégration : A et B ont chacun des notifications | Chacun ne voit que les siennes | Conforme (totaux exacts) ; les paramètres `vendeur`, `vendeurId`, `sub`, `seller_id` ne donnent jamais accès à celles de A | Conforme |
| Unitaire : liste lue pour le sujet du jeton | Sujet du jeton | Liste demandée pour A puis pour B, dans cet ordre | Conforme |
| Sans jeton, Basic, illisible, `Bearer` vide, clé inconnue, expiré, sans `exp`, audience `place-de-marche`, émetteur étranger, client non autorisé, `alg: none`, signature tronquée, sujet falsifié | 401 | 401 pour les 13 cas (unitaire et intégration), sans divulguer de notification ni lire la liste | Conforme |
| Hors AC : jeton sans le rôle `vendeur` (acheteur, admin, aucun) | — | 403, liste non lue | Conforme au README |

### AC7 — Droits de publication sur le broker

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| `interactions` publie une décision sur `pdm.evenements` | Refusé | 403 `ACCESS_REFUSED` | Conforme |
| `interactions` publie sur `controle.resultats` (et `controle.demandes`, et directement dans sa file) | Refusé | 403 pour chacun | Conforme |
| `controle` publie une décision sur `pdm.evenements` (clé `publication.decidee.v1` ou `v2`) | Refusé | 403 | Conforme |
| Après ces refus | Aucune notification | 0 | Conforme |

### AC8 — Bases cloisonnées

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Même PostgreSQL, bases `marche` et `interactions` migrées, `REVOKE CONNECT … FROM PUBLIC` : `interactions_execution` et `interactions_migration` sur `marche` | Refusé | `42501 permission denied for database "marche"` | Conforme |
| `marche_execution` et `marche_migration` sur `interactions` | Refusé | `42501 permission denied for database "interactions"` | Conforme |
| `interactions_execution` : UPDATE, DELETE, TRUNCATE, CREATE TABLE, DROP, ALTER | Refusés | 42501 pour chacun ; SELECT autorisé ; notification intacte | Conforme |

### AC9 — Contrat de la paire Place de marché – Interactions

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Toutes les décisions du banc de bout en bout (23, 10 vendeurs), lues sur l'abonnement de test | Conformes au schéma v1 | 23 décisions publiées en outbox = 23 reçues ; toutes émises par `marche`, sur `pdm.evenements` / `publication.decidee.v1` ; `Validate` et `Decode` acceptent ; enveloppe `publication.decidee` v1 | Conforme |
| Les mêmes, côté Interactions | Acceptées | Rien dans `.ecartes`, aucun `notifications.decision_ecartee` ; 23 notifications pour 23 décisions | Conforme |
| Exemples du contrat (`valides/`) donnés au consommateur | Acceptés, données restituées | Acquittés ; article, révision, vendeur, issue, motif identiques | Conforme |
| Contrat : 3 issues, 9 motifs (6 du Contrôle, 3 de l'administrateur) | Acceptés | Conforme ; ensembles fermés vérifiés | Conforme |
| Réponses de l'API d'Interactions (200, page vide, 400 `taille=51` et `page=0`, 401, 403) | Conformes à l'OpenAPI | Conforme | Conforme |

### Anomalies

Aucune : aucun comportement observé ne contredit un AC du périmètre back.

### Observations (sans anomalie)

- Un résultat de contrôle refusé faute de pouvoir écrire l'événement n'est pas perdu : il est réessayé et appliqué dès le rétablissement de l'outbox (0,2 à 3 s observés).
- `services/interactions/internal/notifications/stockage` et `api` n'ont pas de tests propres : l'insertion idempotente, la liste filtrée et la lecture `READ ONLY` sont prouvées sur PostgreSQL réel par les bancs d'intégration et de bout en bout.
- Le parcours bout en bout mesure de la soumission (ou de la décision HTTP) à la liste : c'est un majorant du délai « événement diffusé → notification listée » demandé par AC2.

## Front

| Champ | Valeur |
|---|---|
| US | [US-028](../../../backlog/US-028-notification-vendeur.md) |
| Branche | `feat/US-028-notification-vendeur` |
| Date | 2026-11-06 |
| Fonction | Test |
| Périmètre | `web/` : écran `/vendeur/notifications`. AC2 (liste, nouvelle notification en moins de 10 s, rafraîchissement suspendu onglet masqué), AC4 (texte construit à partir de l'issue et du motif, FR/EN, aucun contenu utilisateur), AC6 côté front (401, 403), AC10 (axe-core) |
| Référentiel | AC de l'US ; contrat `GET /interactions/v1/notifications?page=1&taille=20` → `{elements:[{id,articleId,revision,issue,motif,decideLe}],page,taille,total}` ; README `web/` § Notifications du vendeur et § E2E |
| Méthode | Boîte noire : application complète démarrée (Vitest/jsdom, puis build de production servi à Playwright), fournisseur d'identité et service Interactions simulés par interception réseau. Textes attendus écrits en dur |
| Verdict | **Conforme** : tous les AC du périmètre front sont couverts et conformes ; aucune anomalie |

### 1. Livrables de test

| Fichier | Contenu |
|---|---|
| `web/src/acceptance/US-028/notificationFixtures.ts` | Notification au format du contrat (identifiants, issue, motif, date) |
| `web/src/acceptance/US-028/sellerNotificationsHarness.ts` | Interactions simulé (liste, statut forcé, réponse retenue, relevé des appels et du jeton), libellés FR/EN, onglet masqué, avance des faux timers |
| `web/src/acceptance/US-028/contrat-payload-notifications.test.ts` | Lecture de la réponse : page valide (3 issues), vide, date en microsecondes ; rejet de 5 enveloppes invalides et de 11 éléments invalides (issue ou motif hors liste, motif numérique, date illisible, révision textuelle, champ obligatoire absent) — 19 tests |
| `web/src/acceptance/US-028/AC2-liste-et-rafraichissement.test.tsx` | Lien « Notifications », liste dans l'ordre reçu, annonce `status` singulier/pluriel, 20 éléments, état vide, contrat d'appel et jeton ; faux timers : nouvelle notification en moins de 10 s, relecture à 5 s, premier élément sur écran vide, aucune relecture onglet masqué, rattrapage au retour — 11 tests |
| `web/src/acceptance/US-028/AC4-texte-construit-sans-contenu-utilisateur.test.tsx` | 3 issues × FR/EN, 9 motifs × FR/EN sans code brut, aucun motif pour « publié », bascule FR → EN, titre et description de l'article absents, champs libres ajoutés à la réponse non rendus ni interprétés, issue ou motif hors liste jamais affichés — 30 tests |
| `web/src/acceptance/US-028/AC6-etats-session-expiree-et-acces-reserve.test.tsx` | 401 FR/EN (« session expirée » + « Se reconnecter »), 401 à la relecture, 403 FR/EN, comptes acheteur et administrateur, chargement, erreur FR/EN et reprise par « Réessayer », réponse 200 malformée — 12 tests |
| `web/src/acceptance/US-028/AC10-accessibilite.test.tsx` | axe-core (jsdom) sur 6 états × FR/EN, structure par rôles — 13 tests |
| `web/e2e/US-028/AC2-AC4-AC6-notifications-vendeur.spec.ts` | Playwright : parcours connexion → lien « Notifications » → 3 décisions de démonstration ; décision émise en cours de consultation visible en moins de 10 s (temps réel) ; contrat d'appel ; onglet masqué 12 s sans relecture puis rattrapage ; motifs traduits sans code brut ; aucun titre d'article du vendeur repris ; 401 ; 403 ; compte sans rôle vendeur + axe-core — 9 tests par langue |
| `web/e2e/support/interactions.ts` | Configuration `source: api` et service Interactions simulé pour Playwright |
| `web/e2e/support/ecrans.ts` | 6 écrans ajoutés à l'analyse axe-core : notifications (démonstration), trois issues (API), aucune notification, indisponibles, session expirée (401), accès réservé (403) |
| `web/e2e/support/textes.ts` | Libellés FR/EN des notifications |

### 2. Exécutions (dans `web/`)

| Commande | Résultat |
|---|---|
| `npx tsc -b` | Vert |
| `npm run lint` | Vert, 0 avertissement |
| `npx prettier --check .` | Vert |
| `npx vitest run` | Vert : 35 fichiers, 432 tests (dont 85 pour US-028) |
| `E2E_LANGUES=fr,en npx playwright test` (projets `fr`, `en`, `bascule-fr-en` ; le projet `fr` est celui des PR en CI) | Vert : 81 tests, 2,9 min |

### 3. Résultats par AC

#### AC2 — Notification listée dans l'espace vendeur en moins de 10 secondes

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Nominal : vendeur connecté, lien « Notifications » de l'en-tête | Écran des notifications, liste des décisions | `/vendeur/notifications`, titre 1 « Notifications », région « Décisions sur vos articles », 3 cartes (Vitest et Playwright FR/EN) | Conforme |
| Nominal : décision émise pendant la consultation | Visible en moins de 10 s sans action | Faux timers : visible à 9,999 s ; Playwright temps réel : visible en moins de 10 s | Conforme |
| Nominal : appel de la liste | `page=1&taille=20`, jeton du vendeur | Tous les appels conformes, `Authorization: Bearer …` | Conforme |
| Limite : relecture | Toutes les 5 s | Nouvel appel après 5,1 s | Conforme |
| Limite : aucune décision puis une première | État vide, puis liste en moins de 10 s | Conforme | Conforme |
| Limite : 20 décisions, 1 décision | 20 cartes ; annonce « 1 notification » au singulier | Conforme | Conforme |
| Onglet masqué | Aucune relecture | 0 appel en 20 s (faux timers) et en 12 s (Playwright) | Conforme |
| Retour sur l'onglet | Décision arrivée entre-temps visible en moins de 10 s | Conforme (Vitest et Playwright) | Conforme |

#### AC4 — Texte construit à partir de l'issue et du motif, FR ou EN, sans contenu utilisateur

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| 3 issues × FR/EN | Titre de carte issu de l'issue, motif traduit | Conforme | Conforme |
| 9 motifs (contrôle et refus) × FR/EN | Libellé traduit, code brut absent | Conforme | Conforme |
| Décision « publié » | Aucun motif | Conforme | Conforme |
| Bascule FR → EN sans rechargement | Texte reconstruit en anglais | Conforme | Conforme |
| Article du vendeur avec titre et description saisis | Ni titre ni description affichés | Absents (Vitest) ; titres de « Mes articles » absents de l'écran (Playwright) | Conforme |
| Réponse enrichie de champs libres (`titre`, `commentaire` avec balisage, `description`) | Rien de ces contenus rendu ni interprété | Absents, aucun `<img>` injecté, aucun script exécuté | Conforme |
| Issue ou motif hors liste fermée | Valeur reçue jamais affichée | Réponse rejetée (état d'erreur), valeur absente ; la lecture du contrat rejette ces éléments | Conforme |

#### AC6 côté front — 401 et 403

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| 401 à l'ouverture, FR/EN | État « session expirée » | « Votre session a expiré » / « Your session has expired », bouton « Se reconnecter » / « Sign in again », aucune carte | Conforme |
| 401 à une relecture | Bascule en « session expirée » | Conforme | Conforme |
| 403, FR/EN | État « accès réservé » | « Accès réservé » / « Restricted access », aucune notification | Conforme |
| Compte acheteur ou administrateur | « Accès réservé » | Titre 1 « Accès réservé », aucune notification | Conforme |
| Hors AC, demandé : chargement, indisponibilité (500/503), réponse 200 malformée | Chargement annoncé ; état d'erreur avec « Réessayer » ; reprise | Conforme | Conforme |

#### AC10 — axe-core, aucune violation « serious » ou « critical »

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Playwright (WCAG 2.1 A/AA, contraste compris), FR et EN : liste de démonstration, trois issues, vide, indisponible, 401, 403, compte sans rôle vendeur | 0 violation bloquante | 0 | Conforme |
| Vitest (jsdom, contraste exclu) : 6 états × FR/EN | 0 violation bloquante | 0 | Conforme |

### 4. Anomalies

Aucune.

### 5. Remarques

- La reprise d'erreur (500/503) n'apparaît qu'après les nouvelles tentatives automatiques, soit plusieurs secondes : les tests attendent jusqu'à 15 s. Aucun AC ne fixe ce délai.
- AC2 est prouvé côté front contre un service simulé. Le délai de bout en bout (émission par la Place de marché → Interactions → écran) relève de la partie back.
