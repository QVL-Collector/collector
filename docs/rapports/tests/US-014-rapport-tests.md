# Rapport de tests — US-014 Journaux centralisés et corrélés d'un bout à l'autre

## Back

| Champ | Valeur |
|---|---|
| US | [US-014](../../../backlog/US-014-journaux-centralises.md) |
| Branche | `feat/US-014-journaux-centralises` (non commitée) |
| Date | 2026-11-20 |
| Fonction | Test |
| Périmètre | Partie back : AC1 (champs `service`, `correlation_id`, `trace_id` dans les journaux de la Place de marché et du Contrôle, corrélation recopiée dans chaque message du canal), AC4 (événement d'audit horodaté d'une décision de contrôle appliquée), AC5 côté source (masquage par `libs/platform/logging`), AC6 côté service (aucun champ d'origine écrit par le service, aucun audit ni `service` forgeable par le contenu reçu). Ajouts de pilotage du 2026-11-20 : corrélation et trace de la soumission reprises par le filet de relance (migration 00011), audit de la suspension et du rétablissement d'un compte (`comptes.suspension`). Migrations place-de-marche 00010 et 00011 |
| Hors périmètre (reporté) | **AC2** (Fluent Bit en DaemonSet, groupes CloudWatch applicatif 7 j et audit 30 j), **AC3** (incident reconstitué dans CloudWatch), **AC5 seconde couche** (filtre Fluent Bit, valeur piégée recherchée dans CloudWatch), **AC6 côté collecte** (origine tirée des métadonnées Kubernetes par Fluent Bit) : DevOps et AWS, à tester après déploiement (requêtes Logs Insights) |
| Outils | `go test` ; testcontainers-go avec PostgreSQL et RabbitMQ ; Place de marché (`serve`) en processus de test, Contrôle et Interactions construits et lancés en processus séparés, journaux de chacun capturés ; files espions sur `controle.demandes`, `controle.resultats` et `pdm.evenements` ; commande `place-de-marche suspendre`/`retablir` construite et lancée |
| Verdict | **Tous les AC du périmètre back sont couverts et conformes** ; aucune anomalie |

### Méthode

Scénarios dérivés des AC (et des deux décisions de pilotage transmises en cours de passe), étiquetés `US014…`. La doc des modules (`libs/platform/logging/README.md`, README de la Place de marché) a servi à connaître les points d'entrée (en-têtes `X-Correlation-Id` et `traceparent`, colonnes de l'outbox, commandes), jamais à dériver les attendus. Trois niveaux :

- **Unitaire** (`libs/platform/logging`, `libs/platform/outbox`) : `logging.New` écrit dans un tampon ; chaque ligne est relue en JSON. Corrélation et trace via `Middleware`, `ContinueTrace`, `WithReceivedCorrelationID`, `StartOperation`, `Propagate` ; audit via `logging.Audit` ; masquage avec des valeurs piégées réalistes (JWT, Bearer, Basic, URL présignée S3 complète avec `X-Amz-Credential`, `X-Amz-Security-Token`, `X-Amz-Signature`, `postgres://u:mdp@`, `amqps://u:mdp@`, DSN `password=`, `http.Header`, `url.Values`, groupes, `With`, `WithGroup`, `fmt.Stringer`, `*url.Error`). Pour chaque cas : au moins une ligne émise (le masquage ne supprime pas l'événement) et aucune valeur piégée dans la sortie.
- **Bout en bout** (`TestUS014JournauxCorrelesDeLaSoumissionAuControle`, un banc pour 10 scénarios) : PostgreSQL, RabbitMQ, Place de marché, Contrôle et Interactions réels. Soumissions par HTTP avec des en-têtes choisis (valides, absents, mal formés, piégés), suivies jusqu'à la décision et la notification. Vérifications croisées : journaux des trois services, lignes `outbox` (`correlation_id`, `traceparent`), messages AMQP captés par les espions (propriété `correlation-id`, enveloppe, en-tête `traceparent`), événement d'audit.
- **Base et commandes** : migrations 00010 et 00011 sur une base montée jusqu'à la version 9 avec une ligne d'outbox en attente ; filet de relance (`controle.NewSafeguard`, balayage rapide, article daté dans le passé) ; `place-de-marche suspendre`/`retablir` lancés comme en production, sortie relue en JSON.

### Livrables de test

| Fichier | Contenu |
|---|---|
| `libs/platform/logging/us014_correlation_test.go` | 17 tests : traceparent et corrélation repris ; absents → tirés ; 12 traceparents invalides (vide, illisible, version `ff`, trace nulle, span nul, longueurs, non hexa, drapeaux, champ en trop, séparateurs, injection de ligne) ; 10 corrélations mal formées (texte, accolades, tronquée, sans tirets, `urn:uuid:`, injection JSON avec saut de ligne, 4096 caractères, espaces, caractères de contrôle) ; propagation HTTP sortante (`Propagate`) reprise par le service appelé ; message consommé (`ContinueTrace`) avec et sans trace valide ; corrélation de message mal formée ; commande ponctuelle ; `service` sur chaque ligne ; audit horodaté (article, révision, issue, motif, corrélation, trace) ; audit hors contexte ; aucun champ d'origine dans l'audit (AC6) ; observation sur l'hexadécimal majuscule |
| `libs/platform/logging/us014_masquage_test.go` | 4 tests dont `TestUS014AC5ValeursPiegeesMasqueesALaSource` (36 cas), requête HTTP piégée traversant `Middleware` (en-têtes, URL, paramètres journalisés par le traiteur), masquage qui garde corrélation, trace, `article_id` et cause d'erreur, observation `bearer` en minuscules |
| `libs/platform/outbox/us014_trace_test.go` | 3 tests : `NewEvent` garde le `traceparent` et la corrélation du contexte ; hors trace, valeur vide ou conforme W3C ; deux événements d'une même requête partagent la trace |
| `services/place-de-marche/cmd/place-de-marche/journaux_correles_test.go` | `TestUS014JournauxCorrelesDeLaSoumissionAuControle` : 10 scénarios (AC1, AC4, AC5, AC6), banc complet à trois services |
| `services/place-de-marche/cmd/place-de-marche/journaux_migration_test.go` | `TestUS014MigrationsDeTraceLigneEnAttenteConserveeEtTraceparentContraint` : 5 scénarios (00010 et 00011) |
| `services/place-de-marche/cmd/place-de-marche/journaux_relance_et_comptes_test.go` | `TestUS014RelanceDuFiletGardeLaCorrelationEtLaTraceDeLaSoumission` (3 scénarios) et `TestUS014AuditDeSuspensionEtDeRetablissementDUnCompte` (6 scénarios) |
| `services/controle/internal/canal/paire_test.go` | Réparé : `resultMessage` prend désormais un `context.Context` en premier argument (`context.Background()`) ; aucun attendu modifié |

### Résultats d'exécution

| Commande | Résultat | Durée |
|---|---|---|
| `go vet ./...` (`services/controle`) avant réparation | Échec de compilation : `paire_test.go:69:82: not enough arguments in call to responder.resultMessage` | — |
| `go test ./...` (`services/controle`) après réparation | `ok` dans 3 paquets (`canal`, `coordonnees`, `regles`) | 4 s |
| `go test ./logging ./outbox -run US014 -count=1 -v` (`libs/platform`) | `ok`, **24 tests de premier niveau, 64 sous-tests, 0 skip, 0 échec** | 3 s |
| `go test ./... -count=1` (`libs/platform`) | `ok` dans 9 paquets (dont les 6 tests existants de `logging`) | 5 s |
| `go test ./cmd/place-de-marche/ -run TestUS014 -count=1 -v`, première passe | 6 scénarios PASS, 4 FAIL : **erreurs de test** (instant de départ de la notification fixé à « maintenant − 20 s » ; file espion des décisions non vidée avant la décision de l'administrateur, qui gardait la décision « à vérifier » du même article). Corrigées sans toucher au code de production | 26 s |
| Même commande, code de test final (après découpage pour `gocognit`) | `ok`, **4 tests de premier niveau, 24 scénarios PASS, 0 skip** | 64 s |
| `go test ./... -count=1 -v -timeout 90m` (`services/place-de-marche`), lancé pendant les suites d'Interactions et le banc US-014 | `internal/config` `ok` ; `cmd/place-de-marche` : **101 tests de premier niveau PASS, 4 FAIL, 0 skip**, dont les 4 tests US-014. Échecs : `TestUS031GarantiesDeLaReservation`, `TestUS021BoutEnBoutRevalidationSurPostgreSQLEtRabbitMQ`, `TestUS032MigrationDuTitreSurDesCommandesExistantes` au **démarrage du conteneur PostgreSQL** (« wait until ready : context deadline exceeded », machine chargée : environnement, pas de test exécuté) ; `TestUS024FiletDelaiCroissantAuxValeursDeProduction`, voir Observations | 1571 s |
| Les 3 tests en échec au démarrage du conteneur, relancés seuls | `ok`, **3 PASS, 0 skip** (US-031 : 287 s) | 338 s |
| `go test ./cmd/place-de-marche/ -run "TestUS024Filet\|TestUS014Relance" -count=3 -v` | `ok`, 3 passes : les 5 tests US-024 et le test de relance US-014 PASS à chaque passe | 242 s |
| `go test ./... -count=1` (`services/interactions`) | `ok` dans 2 paquets (`cmd/interactions`, `notifications/canal`) | 52 s |
| `go test ./... -count=1` (`services/transactions`) | `ok` dans 4 paquets | 12 s |
| `gofmt -l`, `go vet ./...`, `golangci-lint run --config <racine>/.golangci.yml ./...` dans `libs/platform`, `services/place-de-marche`, `services/controle`, `services/interactions`, `services/transactions` | 0 issue (premier jet : 4 `gocognit` et 3 `gosec` G101 sur les constantes piégées, corrigés par découpage et renommage) | — |
| Conteneurs | Arrêtés et supprimés par testcontainers en fin de test | — |

### Couverture par package

| Package | Tests US-014 | AC |
|---|---|---|
| `libs/platform/logging` | 21 tests, 64 sous-tests | AC1, AC4, AC5, AC6 |
| `libs/platform/outbox` | 3 tests | AC1 |
| `services/place-de-marche/cmd/place-de-marche` | 4 tests, 24 scénarios (bout en bout, migrations, relance, comptes) | AC1, AC4, AC5, AC6, migrations 00010 et 00011 |
| `services/controle/internal/canal` | Réparation de `paire_test.go` ; le Contrôle est éprouvé en boîte noire par le banc de la Place de marché (binaire réel) | AC1 |
| `services/interactions` | Aucun test ajouté ; Interactions est lancé dans le banc de bout en bout, ses journaux y sont vérifiés | AC1 (au-delà de l'AC), AC5 |
| `services/transactions` | Aucun test ajouté (hors de la chaîne soumission → Contrôle) ; non-régression seulement | — |

### AC1 — `service`, `correlation_id` et `trace_id` sur chaque journal ; corrélation recopiée dans chaque message du canal

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Bout en bout : soumission avec `X-Correlation-Id` (UUID) et `traceparent` W3C valides, publiée | Mêmes `correlation_id` et `trace_id` de la Place de marché au Contrôle ; corrélation dans chaque message | Réponse 202 avec le même `X-Correlation-Id`. Lignes de la Place de marché (requête, application du résultat) et du Contrôle (`controle.demande_controlee`) : `service` propre à chacun, corrélation et trace reçues. Outbox : `controle.demande` et `publication.decidee` avec la corrélation et un `traceparent` de même trace. Messages AMQP demande, résultat, décision : propriété `correlation-id`, enveloppe et en-tête `traceparent` identiques. Interactions : même corrélation, même trace | Conforme |
| Bout en bout : soumission sans en-tête, refusée (coordonnées) | Identifiants tirés puis suivis | Corrélation UUID renvoyée et journalisée, trace tirée ; mêmes valeurs dans le Contrôle, l'outbox, les trois messages et Interactions | Conforme |
| Bout en bout : `X-Correlation-Id` « pas-un-uuid-… » et `traceparent` à trace nulle, résultat « à vérifier » | Valeurs mal formées remplacées, puis suivies | Corrélation et trace tirées, suivies comme ci-dessus ; la valeur mal formée n'apparaît dans aucun des trois journaux | Conforme |
| Bout en bout : ligne portant l'article, toutes sources | Chaque ligne porte les trois champs | Toute ligne de la Place de marché portant l'article porte la corrélation de la création ou celle de la soumission, avec la trace correspondante ; idem Contrôle et Interactions (soumission seulement) ; aucune ligne sans corrélation | Conforme |
| Bout en bout : toutes les lignes des trois journaux | `service` présent | Chaque ligne JSON porte `service` = `place-de-marche`, `controle` ou `interactions` ; aucune ligne non JSON (hors la ligne d'arrêt ajoutée par le banc) | Conforme |
| Message publié directement sur `controle.demandes`, sans `traceparent`, propriété `correlation-id` ≠ corrélation de l'enveloppe | Corrélation de l'enveloppe recopiée ; trace tirée | Contrôle : corrélation de l'enveloppe, trace tirée valide ; résultat : enveloppe et propriété à la corrélation de l'enveloppe, en-tête `traceparent` de la trace journalisée par le Contrôle | Conforme |
| Même chose avec un `traceparent` valide | Trace reprise jusqu'au résultat | Contrôle et résultat portent la trace reçue | Conforme |
| Ligne d'outbox insérée sans `traceparent` (forme d'avant 00010) | Relayée, traitée, journalisée avec corrélation et trace | `publie_le` renseigné ; Contrôle : corrélation de l'enveloppe, trace tirée ; résultat à la même trace | Conforme |
| Unitaire `Middleware` : `traceparent` et corrélation valides | Repris | Trace reprise, nouveau `span_id` ≠ parent, drapeaux `01` conservés, corrélation renvoyée | Conforme |
| Unitaire : 12 `traceparent` invalides | Trace tirée valide | Trace tirée (32 hexa, non nulle), corrélation reçue conservée, `TraceParent(ctx)` cohérent | Conforme |
| Unitaire : 10 corrélations mal formées | UUID tiré | UUID tiré, renvoyé dans la réponse, trace reçue conservée, aucune ligne `audit` forgée | Conforme |
| Unitaire : deux requêtes sans en-tête | Identifiants distincts | Corrélations et traces différentes | Conforme |
| Unitaire `Propagate` puis service appelé | Même corrélation, même trace, span différent | Conforme dans les deux journaux | Conforme |
| Unitaire `ContinueTrace` (message) : trace valide, absente, nulle, illisible ; corrélation vide, texte, injection | Reprise ou tirée | Reprise si valide, sinon tirée ; corrélation mal formée remplacée par un UUID | Conforme |
| Unitaire `StartOperation`, `CorrelationIDOrNew` | Identifiants tirés, distincts | Conforme | Conforme |
| Unitaire `outbox.NewEvent` | `traceparent` et corrélation du contexte gardés | Conforme ; hors trace, valeur vide ou W3C | Conforme |
| Relance du filet (décision 2026-11-20) : soumission avec en-têtes, article daté de 3 min puis 5 min | Demande réémise et journal `controle.demande_relancee` avec la corrélation et la trace de la soumission | 1re et 2e relances : outbox (3 demandes) et journaux à la corrélation et à la trace de la soumission | Conforme |
| Relance d'une soumission sans en-tête | Identifiants tirés à la soumission repris | Conforme | Conforme |
| Relance d'un article sans identifiants gardés (soumis avant 00011) | Tirés à la première relance puis gardés | UUID et trace valides à la 1re relance, mêmes valeurs à la 2e | Conforme |

### AC4 — Événement d'audit horodaté : article, révision, issue

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Bout en bout : résultat `publie` appliqué | Un événement d'audit horodaté avec article, révision, issue | Un seul `controle.resultat_applique` pour l'article : `audit=true`, `time` RFC 3339, `article_id`, `revision`, `issue`, avec la corrélation et la trace de la soumission | Conforme |
| Bout en bout : résultat `refuse` (`coordonnees_detectees`) et `a_verifier` (`categorie_inconnue`) | Idem, avec le motif | Idem, motif présent | Conforme |
| Bout en bout : décision de l'administrateur (publier) sur un article « à vérifier » | Décision appliquée auditée | Un `annonces.decision_admin` : `audit=true`, horodaté, article, révision, issue, sujet de l'administrateur, corrélation et trace de sa requête ; message `publication.decidee` à cette corrélation | Conforme |
| Unitaire `logging.Audit` avec contexte | Horodaté, champs métier, corrélation | Conforme ; une ligne ordinaire n'est pas marquée `audit` | Conforme |
| Unitaire `logging.Audit` hors contexte | Horodaté et marqué | Conforme | Conforme |
| Commande `suspendre`, deux fois (décision 2026-11-20) | Audit avec compte, auteur, action | Une ligne `comptes.suspension` par exécution : `audit=true`, `time`, `compte_id`, `auteur` = `exploitation`, `action` = `suspendre`, `suspendu` = vrai, `service`, corrélation UUID, trace | Conforme |
| Commande `retablir`, deux fois | Idem | `action` = `retablir`, `suspendu` = faux | Conforme |
| `suspendre`/`retablir` d'un compte inconnu | Arrêt en erreur, rien d'audité | Code ≠ 0, aucun événement `comptes.suspension` | Conforme |

### AC5 — Jeton, URL présignée ou secret masqué à la source

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Unitaire : 36 formes de journalisation piégées (voir Méthode) | Valeur piégée absente, événement gardé | Une ligne au moins à chaque cas, aucune des 12 valeurs piégées dans la sortie | Conforme |
| Unitaire : requête piégée (`Authorization` Bearer JWT, `X-Collector-User-Token`, `Cookie`, `Proxy-Authorization` Basic, paramètres présignés et `token=` dans l'URL) traversant `Middleware`, traiteur qui journalise en-têtes, URL et paramètres | Aucune valeur piégée | Aucune | Conforme |
| Unitaire : erreur `*url.Error` sur une URL présignée | Masquée, journal toujours exploitable | Corrélation, trace, `article_id` et cause (« délai dépassé ») conservés, aucune valeur piégée | Conforme |
| Bout en bout : soumissions et requêtes avec en-têtes piégés, URL portant `X-Amz-Signature` et `X-Amz-Security-Token`, `Authorization: Basic …`, Bearer forgé | Aucune valeur piégée dans les journaux | 32 valeurs cherchées (dont les jetons réels des vendeurs et de l'administrateur, les mots de passe des rôles PostgreSQL et des identités RabbitMQ) dans les journaux des trois services (≈ 28 Ko, 5 Ko, 5 Ko) : aucune | Conforme |
| Commande `suspendre` | Mot de passe de `marche_migration` absent | Absent ; sortie entièrement JSON | Conforme |

### AC6 — Origine de l'audit tirée des métadonnées Kubernetes, pas du contenu (côté service)

| Scénario | Attendu (AC, côté service) | Réel | Statut |
|---|---|---|---|
| Unitaire et commande : champs de l'événement d'audit | Le service n'écrit aucun champ d'origine « de confiance » | Ni `kubernetes`, `namespace`, `pod`, `container`, `origine`, `origin`, `source` | Conforme |
| Bout en bout : titre d'article et `X-Correlation-Id` contenant `{"audit":true,"service":"controle","event":"controle.resultat_applique",…}`, en-têtes `Service` et `Audit` | Le contenu reçu ne forge ni ligne d'audit ni `service` | Chaque ligne des trois journaux garde le `service` de son processus ; aucune ligne `audit` ne porte le marqueur forgé ; un seul audit pour l'article | Conforme |
| Unitaire : corrélation avec saut de ligne et JSON injecté | Aucune ligne d'audit forgée | Aucune | Conforme |
| Aiguillage vers le groupe d'audit par les métadonnées Kubernetes | — | **Reporté** (Fluent Bit, AWS) | Reporté |

### Migrations 00010 et 00011

| Scénario | Attendu | Réel | Statut |
|---|---|---|---|
| Base à la version 9, ligne d'outbox en attente, puis migrations | Ligne conservée à l'identique, `traceparent` vide | Corrélation, corps, `cree_le`, `publie_le` inchangés ; `traceparent` NULL | Conforme |
| `outbox.traceparent` W3C ou NULL | Accepté | Accepté | Conforme |
| `outbox.traceparent` vide, illisible, majuscules, trace de 31 caractères, injection de ligne | Refusé | Violation de contrainte (23514) dans les 5 cas | Conforme |
| `marche_execution` lit et écrit la colonne | Autorisé | Autorisé | Conforme |
| `article.demande_controle_traceparent` (00011) : W3C ou NULL accepté, 5 valeurs hors format refusées | — | Conforme | Conforme |

### Anomalies

Aucune : aucun comportement observé ne contredit un AC du périmètre.

### Observations (sans anomalie)

- **Majuscules dans `traceparent`** : un `traceparent` en hexadécimal majuscule est traité comme invalide (nouvelle trace tirée). C'est la lecture stricte de W3C Trace Context (minuscules exigées) ; l'AC ne dit rien de ce cas.
- **`bearer` en minuscules dans une valeur libre** : masqué aussi (jeton absent de la sortie).
- **Lignes portant l'article** : sur chaque parcours, une seule ligne de la Place de marché, une du Contrôle et une d'Interactions portent l'identifiant de l'article en valeur ; les autres lignes du parcours (requête HTTP notamment) se rattachent par `correlation_id`. L'AC exige les champs sur chaque ligne, ce qui est vérifié ; la recherche par article seul ne retrouve pas toute la requête, la recherche par corrélation si.
- **Propriété AMQP `correlation-id` contre enveloppe** : quand elles divergent, c'est l'enveloppe qui est recopiée (journaux du Contrôle et résultat), comme le décrit le README de `logging`.
- **Test existant US-024 instable sous charge (à surveiller par les devs)** : dans la suite complète, `TestUS024FiletDelaiCroissantAuxValeursDeProduction`, sous-test « relance à 2 min », a échoué une fois : journal de relance vu, **2 demandes en outbox mais `article.relances` encore à 0** à la relecture (`filet_de_relance_test.go:179`). Non reproduit sur 3 passes isolées (et 1 passe de plus interrompue par une mise en veille de la machine, non exploitable). Le scénario n'est pas un AC d'US-014 ; c'est consigné parce que le filet de relance a été modifié dans cette US (reprise de la corrélation, migration 00011) : un écart, même bref, entre l'écriture de la demande réémise et le compteur de relances est à confirmer ou écarter par le dev back. Le test US-014 de relance n'a jamais échoué (5 exécutions).
- **Banc à trois services** : la ligne « arrêt d'Interactions : exit status 1 » en fin de test vient de l'arrêt forcé par le banc, pas d'une erreur du service.

### Reporté (hors périmètre de cette passe)

| AC | Reste à prouver | Fonction |
|---|---|---|
| AC2 | Fluent Bit en DaemonSet ; groupe applicatif (7 j) et groupe d'audit (30 j) ; `audit=true` aiguillé vers le second | DevOps |
| AC3 | Contrôle arrêté pendant une soumission ; reconstitution du parcours par l'identifiant de corrélation dans Logs Insights | DevOps, Test |
| AC5 (seconde couche) | Filtre Fluent Bit ; valeur piégée recherchée dans CloudWatch | DevOps, Test |
| AC6 (collecte) | Origine de l'audit tirée des métadonnées Kubernetes, jamais du contenu | DevOps, Test |
