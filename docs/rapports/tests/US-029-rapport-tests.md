# Rapport de tests — US-029 Aucune notification perdue quand Interactions est arrêté

## Back

| Champ | Valeur |
|---|---|
| US | [US-029](../../../backlog/US-029-reprise-interactions.md) |
| Branche | `feat/US-029-reprise-interactions` (empilée sur US-028) |
| Date | 2026-11-13 |
| Fonction | Test |
| Périmètre | `services/place-de-marche` (publication, catalogue, outbox), `services/interactions` (reprise de la file), broker du canal et abonnement de test posé par l'exploitation : AC1, AC2, AC3, AC4 ; exigences ES-31, ES-38, mesure M-CAN-06 ; décisions [D02](../../decisions/D02-decoupage-applicatif.md), [D05](../../decisions/D05-communication-interservice.md) |
| Hors périmètre | Test « après déploiement » (mise à zéro du Deployment d'Interactions, DevOps) ; mise à l'échelle d'Interactions (US-033) |
| Outils | `go test`, testcontainers-go 0.44.0 avec PostgreSQL 18.3 et RabbitMQ 4.3.7 (topologie et politiques de `infra/canal/broker`, API de gestion 15672 exposée au test), Place de marché (`serve`) en processus de test, binaires du Contrôle et d'Interactions construits et lancés en processus (arrêt brutal par `Kill`) |
| Verdict | **Tous les AC sont couverts et conformes** ; aucune anomalie |

### Méthode

Scénarios dérivés des AC, dans un seul test de bout en bout `TestUS029RepriseDInteractionsApresArret`, un `t.Run` par scénario, joués dans l'ordre sur le même banc :

- **Banc** : le banc de bout en bout d'US-028 (PostgreSQL commun aux bases `marche` et `interactions`, RabbitMQ, Place de marché, Contrôle), mais **sans lancer Interactions** au démarrage. Interactions (`migrate` puis `serve`) est lancé, arrêté brutalement (`Kill`, aucun arrêt propre) et relancé par le test.
- **Décisions** : AC1 et AC3 passent par les vraies routes de la Place de marché (création, soumission, décision du Contrôle ou de l'administrateur). Pour l'arrêt brutal (AC2), 2 000 décisions conformes au contrat `publication.decidee` v1 sont diffusées par l'identité `marche` sur `pdm.evenements` / `publication.decidee.v1`, pour que l'arrêt tombe à coup sûr pendant la consommation.
- **Comptages** : longueur des files par `admin-canal` (`queue.declare` passif), notifications lues par `GET /interactions/v1/notifications` (jeton vendeur, audience `interactions`) et comptées dans la table `notification` de la base `interactions`.
- **Abonnement de test (AC3, AC4)** : contrat de l'exploitation, identique à `infra/canal/outils/abonnement-test.sh` (relu après coup, mêmes noms et mêmes arguments), joué par `admin-canal` sur l'API de gestion : identité `abonne-test` sans étiquette ; droits sur `collector` configure `""`, write `""`, read `^test[.]publication-decidee$` ; file quorum `test.publication-decidee` (`x-max-length` 10 000, `x-overflow` `drop-head`, `x-message-ttl` 86 400 000) ; liaison `pdm.evenements` → file, clé `publication.decidee.v1`. Retrait : liaison, file, identité.

### Livrables de test

| Fichier | Contenu |
|---|---|
| `services/place-de-marche/cmd/place-de-marche/reprise_interactions_test.go` | `TestUS029RepriseDInteractionsApresArret` : 6 scénarios (AC1, AC2 ×2, AC3 ×2, AC4) ; client de l'API de gestion du broker, pose, vérification et retrait de l'abonnement de test, diffusion en rafale avec confirmations, arrêt brutal à la première notification |
| `services/place-de-marche/cmd/place-de-marche/notification_vendeur_test.go` | Banc d'US-028 découpé sans changer ses scénarios : `demarrerBancSansInteractions` (banc + lanceur d'Interactions), `lancerProcessusInteractions`, `attendreDisponibilite`, `arreterBrutalement`, `exigerEnVie` ; `demarrerBancNotification` les compose ; `lancerInteractions`, devenu inutilisé, supprimé |
| `services/place-de-marche/cmd/place-de-marche/comptes_migration_test.go` | Inchangé ; son aide `instant` est réutilisée pour comparer `decideLe` et `decided_at` |
| `services/place-de-marche/cmd/place-de-marche/canal_test.go` | Port de gestion `15672/tcp` exposé par le conteneur RabbitMQ du banc (`portDeGestion`) |

### Résultats d'exécution

| Commande | Résultat | Durée |
|---|---|---|
| `go test ./... -run TestUS029 -count=1 -v` (`services/place-de-marche`) | `ok`, 1 test, 6 sous-tests PASS, 0 skip ; conteneurs PostgreSQL et RabbitMQ réellement démarrés (temps d'exécution de 46 à 49 s selon le passage) | 59 s |
| `go test ./cmd/place-de-marche/ -run 'TestUS029\|TestUS028' -count=1 -v` | `ok` : `TestUS028BoutEnBoutPlaceDeMarcheInteractions` (45 s, sans régression après le découpage du banc) et `TestUS029…` (46 s) | 91 s |
| Après les retours de revue : `go test ./... -run 'TestUS029\|TestUS028' -count=1 -v` | `ok` : `TestUS028…` (46,2 s, 11 sous-tests PASS), `TestUS029…` (48,6 s, 6 sous-tests PASS), 0 skip ; arrêt brutal après 1 notification, 0 relivraison | 95 s |
| Après les retours de revue : `go vet ./...`, `golangci-lint run` | 0 remarque, 0 issue | — |
| `go test ./... -count=1 -v -timeout 40m` (`services/place-de-marche`, suite complète) | `ok`, 93 tests de premier niveau, **0 skip**, 0 échec ; dont `TestUS029…` (46 s) et `TestUS028…` (41 s) | 310 s |
| `go vet ./...` (`services/place-de-marche`) | 0 remarque | — |
| `golangci-lint run --config .golangci.yml ./...` (`services/place-de-marche`) | 0 issue | — |
| Conteneurs | Arrêtés et supprimés par testcontainers en fin de test | — |

### AC1 — Interactions arrêté : publications et catalogue non affectés

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Interactions jamais lancé ; 3 articles publiés par le Contrôle (2 vendeurs) | Publication non affectée | Articles `publie` et décision diffusée (outbox publiée vers `pdm.evenements` / `publication.decidee.v1`) 0,7 à 1,0 s après la soumission | Conforme |
| 2 articles refusés par le Contrôle (coordonnées) | Décision non affectée | `refuse`, motif `coordonnees_detectees`, décision diffusée en 1,0 s | Conforme |
| Décisions de l'administrateur (à vérifier → publier ; à vérifier → refuser `article_interdit`) | Décision non affectée | 200, état et motif attendus, deux décisions diffusées par article | Conforme |
| Catalogue public | Articles publiés servis, refusés absents | Les 4 publiés présents sans doublon, fiche publique conforme (id, titre) ; les 3 refusés absents de la liste et 404 sur leur fiche | Conforme |
| File `interactions.publication-decidee` | Décisions conservées pendant l'arrêt | Exactement 9 messages (une par décision), stables 2 s plus tard ; rien dans `.ecartes` ; aucune notification en base | Conforme |

### AC2 — Interactions relancé : notifications créées, sans perte ni doublon

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Relance après AC1 (9 décisions en attente) ; borne : 20 s | Toutes les notifications créées | 9 notifications 0,22 s après la relance ; file vide, stable 2 s | Conforme |
| Une notification par décision | Sans doublon | Par vendeur : total = nombre de ses décisions ; pour chaque (article, révision, issue), exactement une notification, motif exact ; 9 en base ; 9 `notifications.notification_creee` ; rien dans `.ecartes` | Conforme |
| Arrêt brutal en cours de consommation : 2 000 décisions en file (40 vendeurs × 50), Interactions lancé puis tué (`Kill`) dès la première notification | Aucune perte | Tué après 1 ou 2 notifications sur 2 000 selon le passage ; 1 999 décisions rendues à la file, dont 0 ou 1 relivraison (décision déjà notifiée mais non acquittée au `Kill`) | Conforme |
| Relance après l'arrêt brutal ; borne : 90 s | Aucune perte ni doublon | 2 000 notifications en base 1,5 à 2,3 s après la relance ; file vide, stable 2 s ; exactement 2 000 de plus en base ; chaque vendeur : total 50, 50 articles distincts ; `notification_creee` + `notifications.doublon_ignore` = 1 999 décisions consommées à la relance (1 998 + 1 avec une relivraison, 1 999 + 0 sans) ; rien dans `.ecartes` | Conforme |

### AC3 — Abonnement de test posé par l'exploitation : même événement, puis retrait

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Pose par `admin-canal` sur l'API de gestion | Identité en lecture seule, abonnement posé | 201/204 ; relecture : étiquettes `[]`, droits `""` / `""` / `^test[.]publication-decidee$`, file `quorum` durable avec les trois bornes (`x-max-length` 10 000, `x-overflow` `drop-head`, `x-message-ttl` 86 400 000), une liaison `publication.decidee.v1` | Conforme |
| Interactions arrêté ; un article publié par le Contrôle | L'abonnement de test reçoit l'événement | `test.publication-decidee` : 1 message ; `interactions.publication-decidee` : 1 message | Conforme |
| Lecture par `abonne-test` (AMQP) | Même événement | Corps identique octet pour octet à la ligne de l'outbox, `user-id` `marche`, `pdm.evenements` / `publication.decidee.v1`, données égales à la décision diffusée ; un seul message | Conforme |
| Relance d'Interactions | Interactions reçoit le même événement | Notification listée 0,24 s après la relance ; `decideLe` = `decided_at` de l'événement de l'abonnement de test ; une notification dont `message_id` = `id` de l'enveloppe reçue par l'abonnement de test | Conforme |
| Retrait par `admin-canal` (liaison, file, identité) | Retiré après le test | 204 pour chacun ; ensuite 404 sur l'identité, ses droits et la file ; aucune liaison vers la file, aucune file ni identité restante dans les listes ; connexion AMQP de `abonne-test` refusée (403 `username or password not allowed`) | Conforme |
| Diffusion après le retrait | Non affectée | Nouvel article publié, décision diffusée, une notification en 0,84 s | Conforme |

### AC4 — L'identité de l'abonnement de test ne peut pas publier (M-CAN-06)

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| `abonne-test` publie une décision conforme sur `pdm.evenements` (`publication.decidee.v1`, puis `v2`) | Refusé | 403 `ACCESS_REFUSED - write access to exchange 'pdm.evenements'`, canal fermé | Conforme |
| Publication sur `controle.demandes`, `controle.resultats`, échange par défaut vers sa file et vers `interactions.publication-decidee` | Refusé | 403 pour chacun (`amq.default` pour l'échange par défaut) | Conforme |
| Déclarer une file, déclarer un échange, supprimer sa file | Refusé (aucun abonnement créé par un service) | 403 `configure access … refused` pour chacun ; ni `test.intrus` file ni échange créés | Conforme |
| Lier sa file à `pdm.evenements` (`#`) ou à `controle.resultats` | Refusé | 403 `write access to queue 'test.publication-decidee' refused` ; la seule liaison reste `publication.decidee.v1` | Conforme |
| API de gestion avec `abonne-test` (`whoami`, déclaration de file, liaison, publication) | Refusé | 401 pour chacun | Conforme |
| Après ces tentatives | Aucun message parvenu | Files `interactions.publication-decidee` et `test.publication-decidee` inchangées, aucune notification de plus | Conforme |

### Retours de revue traités ([revue](../revues/US-029-revue-code.md))

| Retour | Traitement |
|---|---|
| Contrat de l'abonnement de test : `x-overflow` `drop-head` au lieu de `reject-publish` | Pose et relecture d'AC3 alignées (constante `argumentsAbonnementDeTest` et vérification de la file) ; identique à `abonnement-test.sh` mis à jour |
| `instantDe` dupliquait `instant` | Supprimée ; `instant` réutilisée pour `decideLe` et `decided_at` |
| Relivraison non systématique | Sous-test renommé « AC2 arrêt brutal en cours de consommation puis relance : aucune perte ni doublon » ; rapport : 0 ou 1 relivraison |
| Retrait d'AC3 : lecture des liaisons et trois boucles d'absence | `clesDesLiaisonsDeLAbonnementDeTest` (aussi utilisée par la pose et AC4) et `exigerAbsentDeLaListe` |

### Anomalies

Aucune : aucun comportement observé ne contredit un AC.

### Observations (sans anomalie)

- Quand elle a lieu, la relivraison après l'arrêt brutal se lit dans le journal de la relance (`notifications.doublon_ignore` pour la décision notifiée mais non acquittée au moment du `Kill`). Elle n'est pas systématique : 0 ou 1 relivraison selon l'instant de l'arrêt (1 sur les trois premiers passages, 0 sur le passage après les retours de revue) ; le test exige l'exactitude du compte final, pas un nombre de relivraisons.
- Le contrat de l'abonnement de test joué par le test est celui de `infra/canal/outils/abonnement-test.sh` (mêmes noms, droits, arguments et appels de l'API de gestion) ; le script lui-même n'est pas exécuté par ce banc (il vise le broker docker compose ou Kubernetes).
