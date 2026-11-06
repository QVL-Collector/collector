# US-024 — Rapport de tests

| Champ | Valeur |
|---|---|
| US | [US-024 — Aucune annonce publiée ni perdue quand le contrôle est indisponible](../../../backlog/US-024-controle-ferme-par-defaut.md) |
| Branche | `feat/US-024-controle-ferme-par-defaut` |
| Date | 2026-11-06 |

## Back — résilience Place de marché et Contrôle

| Champ | Valeur |
|---|---|
| Fonction | Test |
| Périmètre | `services/place-de-marche`, `services/controle` (binaire réel), `libs/platform` : US-024 AC1 à AC4 ; mesures MC6, MC8, MC9 de [D05](../../decisions/D05-communication-interservice.md) |
| Hors périmètre | Déclenchement de l'alerte d'US-015 (AC3) : seuls la mise à l'écart, le journal `canal.message_ecarte` qui la fonde et son délai sont prouvés ; test après déploiement (mise à zéro du Deployment, comptage SQL en cluster) ; alerte sur les lignes d'outbox non publiées (MC8, relève d'US-015) |
| Outils | `go test`, testcontainers-go 0.44.0 avec PostgreSQL 18.3 et RabbitMQ 4.3.7 (topologie et politiques de `infra/canal/broker`), binaire du Contrôle construit et lancé en processus, `rabbitmqctl` dans le conteneur |
| Verdict | **Tous les AC et toutes les mesures du périmètre sont couverts et conformes** |

### Méthode

Scénarios dérivés des AC et des mesures, étiquetés `US-024-ACn` ou `MCn`. Le banc de résilience lance PostgreSQL, RabbitMQ, la Place de marché (`serve`, rôle `marche_execution`) et le Contrôle réel. « Contrôle arrêté » : le processus du Contrôle est tué après avoir publié un premier article, puis relancé. Les articles sont créés, soumis et modifiés par l'API du vendeur ; le catalogue est lu sans jeton. AC4 est vérifié par la requête I3 publiée dans le [README de la Place de marché](../../../services/place-de-marche/README.md), exécutée telle quelle.

Le filet de relance fixe ses durées en constantes (relance après 2 min, balayage toutes les 30 s) : elles ne sont pas injectables. Pour un test rapide sans modifier le code de production, la date `article.demande_controle_le` d'un article est reculée de 3 min en base ; le balayage suivant (≤ 30 s, observé 20 à 22 s) le relance.

| Fichier (`services/place-de-marche/cmd/place-de-marche/`) | Contenu |
|---|---|
| `controle_ferme_test.go` | Banc de résilience : 15 scénarios (AC1, AC2, AC3, AC4, MC6, MC9) |
| `bornes_et_purge_test.go` | Bornes des files sur un broker réel (MC8, AC3 relivraisons) ; purge de l'outbox, droits et purge au démarrage (MC8) |
| `canal_test.go` | Conteneur RabbitMQ conservé dans `canalDeTest` (exécution de `rabbitmqctl`) |
| `services_test.go` | `construireBinaire` extrait de `construireControle` ; `controleDisponible` extrait de `lancerControle` (arrêt et relance du Contrôle) ; assertions existantes inchangées |

Le test de privilèges d'`integration_test.go` n'a pas eu besoin d'être modifié : il passe (`DELETE FROM outbox` refusé) ; le droit `EXECUTE` sur la fonction de purge est vérifié dans `bornes_et_purge_test.go`.

### Résultats d'exécution

| Commande | Résultat |
|---|---|
| `go test ./... -count=1` (`libs/platform`) | `ok` pour `httpjson`, `logging`, `message`, `token` ; `broker` et `outbox` sans fichier de test (couverts de bout en bout ci-dessous) ; aucun test testcontainers dans ce module |
| `go test ./... -count=1` (`services/controle`) | `ok` pour `canal`, `coordonnees`, `regles` ; aucun test testcontainers dans ce module (le binaire du Contrôle est testé par les bancs de la Place de marché) |
| `go test ./... -count=1 -v -timeout 30m` (`services/place-de-marche`) | `ok` en **221 s**, 77 tests de premier niveau verts, **0 skip** ; dont `TestUS024ControleFermeParDefautEtRepriseSansPerte` (51 s), `TestUS024MC8BornesDesFilesDuCanal` (31 s), `TestUS024MC8PurgeAuDemarrageDuService` (15 s), `TestUS024MC8PurgeDeLOutboxEtDroits` (4 s), et sans régression `TestUS013BoutEnBoutPlaceDeMarcheEtControle`, `TestUS009/022/023IntegrationPostgreSQL` |
| `golangci-lint run --config ../../.golangci.yml ./...` (v2.15.0), dans les trois modules | 0 issue partout |
| Conteneurs | Arrêtés et supprimés par testcontainers en fin de test |

### US-024 AC1 — Contrôle arrêté : article en contrôle et invisible au catalogue

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Trois articles de catégories différentes soumis, Contrôle arrêté | En contrôle, invisibles au catalogue | Soumission 202 ; demandes publiées dans `controle.demandes` (3 messages en file durable) ; 3 s plus tard : état `en_controle` (lecture vendeur et liste du vendeur), révision inchangée, absents du catalogue complet et filtré par leur catégorie, fiche publique 404, aucun résultat appliqué | Conforme |
| Article publié puis modifié pendant l'arrêt (recontrôle) | En contrôle, quitte le catalogue | Présent avant ; après modification : `en_controle`, absent du catalogue et du filtre, fiche 404 | Conforme |
| Filet de relance : article sans résultat depuis plus de 2 min | Reste en contrôle, la demande est renvoyée | Nouvelle demande `controle.demande` v1 conforme au contrat, même article et **même révision**, nouvel identifiant de message, publiée ; journal `controle.demande_relancee` ; article toujours `en_controle` et hors catalogue | Conforme |
| Témoin soumis il y a moins de 2 min | Pas de relance | Une seule demande en outbox | Conforme |

### US-024 AC2 — Contrôle relancé : issue appliquée sans perte ni doublon

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Relance du Contrôle | Chaque article soumis pendant l'arrêt contrôlé, issue appliquée | Les 5 articles en attente passent `publie` dès la disponibilité du Contrôle ; présents une seule fois au catalogue ; fiche 200 ; révisions inchangées | Conforme |
| Demande relancée par le filet (deux demandes pour la même révision) | Sans doublon | Le Contrôle publie deux résultats ; le second est acquitté sans effet (journal `controle.resultat_ignore`) ; `modifie_le` stable ; une seule ligne `resultat_controle` par article et révision, pour tous les articles du banc | Conforme |
| Fin de reprise | Sans perte | `controle.demandes` et `pdm.resultats-controle` vides ; `controle.demandes.ecartes` et `pdm.resultats-controle.ecartes` vides ; aucune ligne d'outbox non publiée | Conforme |

### US-024 AC3 — Message invalide écarté sans bloquer les suivants

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Six demandes non conformes publiées sur `controle.demandes` par l'identité `marche` (JSON illisible, clé dupliquée, `article_id` manquant, version 2, type inconnu, 70 Kio), puis une soumission valide | Écartées vers `.ecartes`, la suivante traitée | Les 6 dans `controle.demandes.ecartes` ; l'article soumis ensuite est publié en 0,4 s | Conforme |
| Mêmes six variantes sur `controle.resultats` par l'identité `controle` (champ `issue` manquant), puis une soumission valide | Écartés vers `.ecartes`, le suivant traité | Les 6 dans `pdm.resultats-controle.ecartes` ; article suivant publié en 0,5 s | Conforme |
| Nombre d'essais pour un message non conforme | Borné | Une seule livraison : en-tête `x-death` `reason=rejected`, `count=1` ; un seul journal par message, `relivre=false` | Conforme |
| Message relivré sans fin (lu sans acquittement puis connexion coupée, comme un plantage) | Écarté après un nombre borné d'essais | Écarté après **5 livraisons**, `x-death` `reason=delivery_limit`, file de travail vide | Conforme |
| Journal, base de l'alerte d'US-015 | Avant 2 min | `canal.message_ecarte` avec `file`, `file_ecartes`, `message_id`, `emetteur`, `motif`, `relivre`, émis 0,8 à 0,9 s après la publication, côté Contrôle et côté Place de marché | Conforme (alerte elle-même hors portée locale) |

### US-024 AC4 — Indicateur I3

| Scénario | Attendu (AC) | Réel | Statut |
|---|---|---|---|
| Pendant l'arrêt (contrôle de l'indicateur) | Les demandes en attente apparaissent sans résultat | `demandes_emises` 6, `resultats_appliques` 1, `demandes_sans_resultat` 5, `envois_relances` 1 | Conforme |
| Fin du test | Demandes de l'outbox = résultats appliqués | `demandes_emises` 6 = `resultats_appliques` 6 ; `demandes_sans_resultat` 0, `demandes_non_publiees` 0, `demandes_depassees` 0, `envois_relances` 1 ; identique en `marche_execution` et en rôle propriétaire ; cohérent avec `count(*)` de `resultat_controle` | Conforme |
| Après rejeu de l'outbox (MC9) | I3 inchangé | Toutes les valeurs identiques avant et après | Conforme |

### MC6 — Validation stricte, message non conforme écarté sans nouvel essai

| Scénario | Attendu (D05) | Réel | Statut |
|---|---|---|---|
| Taille, clés dupliquées, schéma, décodage, dans les deux sens | Écarté sans nouvel essai | Voir AC3 : toutes les variantes écartées dès la première livraison (`rejected`, `count=1`) | Conforme |
| Message de 130 Kio | Refusé par le broker en défense | Publication refusée : `406 PRECONDITION_FAILED - message size 133675 is larger than configured max size 131072` ; rien en file ni en `.ecartes` ; article suivant publié | Conforme |

### MC8 — Bornes des files et purge de l'outbox

| Scénario | Attendu (D05) | Réel | Statut |
|---|---|---|---|
| Politiques effectives (`rabbitmqctl list_queues effective_policy_definition`) des 3 files de travail | `reject-publish`, durée de vie 24 h avec mise à l'écart, 5 livraisons | `delivery-limit` 4, `overflow` `reject-publish`, `max-length` > 0, `message-ttl` 86 400 000, `dead-letter-exchange` `collector.ecartes`, clé = nom de la file, `at-least-once` | Conforme |
| Politiques effectives des 3 files `.ecartes` | 7 jours | `message-ttl` 604 800 000, `reject-publish` | Conforme |
| Durée de vie réduite à 5 s (seules les valeurs de la politique réelle changées) | Message expiré mis à l'écart, pas perdu | Message retrouvé dans `controle.demandes.ecartes`, `x-death` `reason=expired` | Conforme |
| Longueur maximale réduite à 2 | Publication refusée, aucun message supprimé | Publication refusée au producteur (`basic.nack`) après 3 acceptations ; tous les messages acceptés restent en file, puis passent en `.ecartes` à expiration | Conforme (voir observation) |
| Purge par `marche_execution` : lignes publiées il y a 8 j, 7 j + 1 min, 6 j 23 h, non publiée créée il y a 30 j | Lignes publiées depuis plus de 7 jours supprimées, jamais une ligne non publiée | `purger_outbox_publiee()` renvoie 2 ; restent la ligne de 6 j 23 h et la non publiée ; seconde purge : 0 | Conforme |
| Droits | Pas de `DELETE` pour le rôle d'exécution | `DELETE` et `TRUNCATE` sur `outbox` refusés (42501) ; fonction `SECURITY DEFINER`, `EXECUTE` pour `marche_execution`, pas pour `PUBLIC` ; un rôle tiers est refusé (42501) | Conforme |
| Démarrage du service avec deux lignes publiées il y a 8 et 10 jours et une d'un jour | Purge | Les deux anciennes supprimées dans les 10 s, celle d'un jour conservée ; journal `outbox.purge` avec `lignes` = 2 | Conforme |

### MC9 — Rejeu de l'outbox

Joué par la sous-commande `rejouer` (`dispatch`), en rôle `marche_migration`, sur le banc de résilience après AC4.

| Scénario | Attendu (D05) | Réel | Statut |
|---|---|---|---|
| Sans argument, `--depuis` sans valeur ou absent, date hors RFC3339 ou sans heure, `--jusqu-a` hors RFC3339 | Refus | Erreur d'usage ; aucune trace `outbox.rejeu` ; outbox identique (toutes colonnes) | Conforme |
| Début égal à la fin, après la fin, dans le futur ; fin dans le futur | Fenêtre refusée | « le début doit précéder la fin » / « la fin est dans le futur » ; aucune écriture | Conforme |
| Début au-delà de 7 jours, avec ou sans fin | Fenêtre plafonnée | « le début dépasse la conservation de 168h0m0s » ; aucune écriture | Conforme |
| Début à 7 j − 1 min, fenêtre d'une minute | Acceptée | Succès, trace `lignes` 0, aucune écriture | Conforme |
| Fenêtre qui exclut la première demande et couvre les 6 suivantes | Seule la date de publication réécrite, tracé, sans double effet | Trace `outbox.rejeu` (`correlation_id`, `depuis`, `jusqu_a`, `lignes` 6) ; les 6 lignes republiées (nouvelle `publie_le`), la ligne hors fenêtre intacte ; toutes les autres colonnes identiques, aucune ligne ajoutée ; 6 résultats en double ignorés (`controle.resultat_ignore`) ; nombre de résultats appliqués, état, révision et `modifie_le` des articles inchangés ; I3 inchangé | Conforme |

### Anomalies

Aucune anomalie : aucun comportement observé ne contredit un AC de l'US ou une mesure MC6, MC8, MC9.

### Observations (sans anomalie)

- **Durées du filet non injectables** : `overdueAfter` (2 min) et `sweepInterval` (30 s) sont des constantes de `internal/annonces/controle/safeguard.go`. Le test contourne en reculant la date d'envoi en base ; un paramètre de `NewSafeguard` permettrait de tester aussi la borne des 2 min sans manipulation de données.
- **Longueur maximale des files quorum** : avec `max-length` 2, RabbitMQ a accepté 3 publications avant de refuser la 4e. Le dépassement d'un message est propre aux files quorum en `reject-publish` ; aucun message n'est supprimé et le refus parvient au producteur, ce que demande MC8. À garder en tête pour le seuil de 10 000.
- **`libs/platform/broker` et `libs/platform/outbox`** n'ont pas de tests unitaires ; leur comportement (mise à l'écart, relais, rejeu, purge) est prouvé de bout en bout par les bancs de la Place de marché.
- Le rejeu est joué dans le processus de test (fonction `dispatch`, comme `serve`) et non par le binaire, pour garder un lint sans exception (gosec G204 sur un sous-processus à arguments variables).
