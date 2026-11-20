# XP02 — Canal RabbitMQ dans l'orchestrateur

| Champ | Valeur |
|---|---|
| Décision testée | [D05](../decisions/D05-communication-interservice.md) (statut `Candidate (XP02)`) |
| Dates prévues | docker compose 2026-10-28 ; k3d 2026-10-29 ; EKS session 1 2026-10-30 (≈ 1 h) |
| Date d'exécution | 2026-10-28, locale : docker compose de 08 h 42 à 09 h 06 UTC, k3d (cluster jetable `xp02-essai`, 1 serveur + 2 agents) de 09 h 22 à 09 h 45 UTC ; EKS : 2026-11-20 de 11 h 02 à 11 h 34 UTC (session `EKS2`, cluster de XP01, contexte `collector-xp01`, 2 nœuds), **session partagée** avec XP03 (namespace `identite-test`) sur le même cluster |
| Exécutée par | Martin (locale) ; session EKS exécutée par un agent pour Martin |
| Scripts | `sandbox/XP02-canal-rabbitmq/` (fichier de définitions, génération des identifiants, producteur, consommateur, outbox et rejeu de test) |
| Dépend de | [XP01](XP01-orchestrateur.md) : cluster EKS de la session 1, politiques réseau actives (G7-5) |

## Objectif et critères de succès

Objectif : prouver la publication, la consommation et les droits du canal RabbitMQ déployé dans l'orchestrateur, exemple cité par la consigne ; confirmer RabbitMQ ou basculer vers NATS JetStream.

Hypothèse : le modèle de droits de RabbitMQ (`configure` vide, `write` et `read` par ressource, droits par clé de routage, `user-id` vérifié) tient ES-38 par configuration, et les politiques réseau d'EKS ferment la gestion et la distribution Erlang.

### Critères

| # | Mesure | Seuil | Bloquant | Si échec | Mesure obtenue (local, 2026-10-28) | Atteint |
|---|---|---|---|---|---|---|
| XC1 | 1 000 messages point à point (`controle.demandes`) et 1 000 en diffusion (`pdm.evenements`, deux files abonnées), confirmations de publication | 100 % des confirmations ; 100 % des identifiants reçus par chaque abonné | **Oui** | Bascule vers NATS JetStream | compose : point à point 1 000 confirmés, 0 `nack`, 1 000/1 000 uniques ; diffusion par l'outbox 1 000 relayés, 1 000/1 000 uniques sur chacune des deux files, 0 manquant | Oui (local) ; EKS : seuil tenu (2026-11-20, voir Résultats) |
| XC2 | Avec l'identité d'un service : publier sur l'échange d'un autre ; lire la file d'un autre ; déclarer une file ou un échange ; publier avec une clé de routage non accordée | Refus dans 100 % des cas (`ACCESS_REFUSED`) | **Oui** | Bascule vers NATS, XC2 rejoué sur la liste complète de `$JS.API` | compose : 19/19 refus `ACCESS_REFUSED` par la sonde + 3/3 par les programmes `go/` ; k3d : refus conformes une fois la sonde différée (voir Difficultés) | Oui (local) ; EKS : seuil tenu (2026-11-20, voir Résultats) |
| XC3 | Consommateur arrêté pendant une vague de 1 000 messages, relancé après 2 min | Aucun message perdu (ENF-12) | **Oui** | Bascule vers NATS | compose : consommateur tué à 326 messages, relancé 2 min 17 s après ; 1 000 uniques cumulés, 1 doublon absorbé | Oui (local) ; EKS : seuil tenu (2026-11-20, voir Résultats) |
| XC4 | Message malformé et message à clé JSON dupliquée dans un flux valide | Écarté sans bloquer les suivants ; mis à l'écart vers `.ecartes` après 5 livraisons ; rejouable | Non | À corriger avant J4 | Sonde (`basic.reject`) : 6 livraisons puis `.ecartes`, suivant non bloqué ; `go/` en `reject` : 2 pièges écartés, 1 000/1 000 uniques ; `go/` en `requeue` : pièges relivrés sans fin (44 183 livraisons en 3 min), jamais écartés ; rejeu par l'outbox conforme | Non (client `go/` à corriger : `Nack` ne fait pas avancer le compteur) |
| XC5 | Redémarrage du conteneur du broker, puis suppression de son pod (`emptyDir` perdu) | Producteurs et consommateurs reconnectés sans intervention ≤ 60 s après le retour `Ready` ; après suppression, filet et rejeu de l'outbox de test rétablissent 100 % des événements (identifiants uniques) | **Oui** | Reconnexion instable : bascule vers NATS. Rejeu insuffisant : pilote EBS CSI réexaminé | compose : `docker restart` sous charge, reconnexion 2 à 4 s après le retour, 10 000/10 000 uniques ; k3d : pod et conteneur, reconnexion 10 à 12 s ; suppression du pod sous charge : 24 et 26 perdus (`emptyDir`), rejeu de l'outbox, 0 manquant | Oui (local) ; EKS : seuil tenu (2026-11-20, voir Résultats) |
| XC6 | KEDA sur `controle.demandes` (scaler `rabbitmq` en mode `http`, identité `monitoring`) | 1 → au moins 2 consommateurs ≤ 60 s après le seuil (5 messages) ; retour à 1 après vidage | Non | Contrôle sur la CPU ou à nombre fixe | k3d : 1 → 2 consommateurs 28 s après le seuil (pic 4), retour à 1 58 s après vidage | Oui (débogage) ; EKS : seuil tenu (2026-11-20, voir Résultats) |
| XC7 | Pod quelconque (`ns-intrus`) vers 5672 et 15672 ; flux témoins : service → 5672, KEDA → 15672 | Refus constatés ; flux témoins passants | **Oui** (reclassé par l'avis sécurité de D14) | Politique corrigée et rejouée ; politiques inopérantes : G7 de XP01, bascule de D03 | k3d : `ns-intrus` → 5672 et 15672 fermés ; témoins `pdm-test` → 5672 et KEDA → 15672 ouverts | Oui (débogage) ; EKS : seuil tenu (2026-11-20, voir Résultats) |
| XC8 | Empreinte au repos et sous vague | Dans la demande (0,25 vCPU, 512 Mio) et la limite (1 Gio) ; aucun nœud ajouté au repos | Non | Revue du dimensionnement ou bascule vers NATS | Mémoire max 233 à 239 Mio au repos et sous vague (compose, k3d) ; CPU k3d 0,9 à 1,3 vCPU au repos, jusqu'à 2,4 vCPU sous vague, due aux sondes `exec` | Mémoire oui, CPU non ; EKS : mémoire oui, nœuds 2 → 2, CPU sous vague au-dessus de la demande (2026-11-20) |
| XC9 | Identité de service : connexion à l'API de gestion, liaison d'une file, purge et suppression de la file d'un autre service ; pod applicatif vers 4369 et 25672 ; publication avec un `user-id` usurpé ; utilisateur `guest` | Refus dans 100 % des cas ; `guest` absent | **Oui** | Erreur de configuration : corrigée et rejouée avant verdict. Défaut structurel (`user-id` non vérifié, droits contournables) : bascule vers NATS | compose : 20/20 refus (liaison, purge, suppression, `user-id` usurpé `PRECONDITION_FAILED`, `guest` refusé et absent, API de gestion 401) ; k3d réseau : 4369 et 25672 fermés (4/4) | Oui (local) ; EKS : seuil tenu (2026-11-20, voir Résultats) |

## Environnement

- **Local 1** : docker compose sur le poste, image RabbitMQ `4.x-management` épinglée, mêmes `rabbitmq.conf` et fichier de définitions qu'en cluster, identifiants générés dans un `.env` ignoré par le dépôt. Sans politiques réseau : XC1 à XC5 et la partie hors réseau de XC9.
- **Local 2** : k3d (cluster de XP01), StatefulSet à une réplique, Service interne, politiques réseau ; débogage de XC6, XC7 et de la partie réseau de XC9, sans valeur de preuve pour EKS.
- **AWS** : cluster EKS de la session 1 de XP01, us-east-1 ; la totalité des critères.

### Topologie de test (D05)

| Échange | Écrit par | Clé | File | Lue par |
|---|---|---|---|---|
| `controle.demandes` (direct) | `svc-pdm` | `v1` | `controle.demandes` | `svc-controle` |
| `controle.resultats` (direct) | `svc-controle` | `v1` | `pdm.resultats-controle` | `svc-pdm` |
| `pdm.evenements` (topic) | `svc-pdm` | `publication.decidee.v1` | `interactions.publication-decidee` | `svc-interactions` |
| idem | idem | idem | `test.publication-decidee` | `svc-test` (lecture seule) |

Vhost `collector` ; files quorum, `x-delivery-limit` 5, échange de mise à l'écart vers `<file>.ecartes`, longueur maximale en `reject-publish`, durée de vie 24 h (MC8) ; identité KEDA au seul tag `monitoring` (MC4).

### Versions à épingler

| Élément | Version | État |
|---|---|---|
| Image RabbitMQ `4.x-management` | 4.3.7 : `rabbitmq:4.3.7-management@sha256:79fb281ef7e0a15e3c5e0e12e66e944cebbf14a129493ef464369abd2c48d01a` | Fixée |
| KEDA (chart) | chart `kedacore/keda` 2.21.0 (KEDA 2.21.0) | Fixée |
| Docker / docker compose | 28.5.1 / v2.40.2 | Relevée |
| k3d, image k3s, EKS, kubectl, Helm | voir [XP01](XP01-orchestrateur.md) | Communes |
| Producteur et consommateur de test | image du bac à sable, épinglée par empreinte | À fixer |
| Outil de vague (XC8) | PerfTest 2.25.0 : `pivotalrabbitmq/perf-test:2.25.0@sha256:b803911d60fdf6885fc6313f0d56a0c2c743bfdd27ac118f5e4b320c36c7b8c9` | Fixée |

## Technos et interactions

Producteur `svc-pdm` (outbox de test : journal local des identifiants publiés) → échanges RabbitMQ → files quorum → consommateurs idempotents (comptent les identifiants uniques). KEDA lit la longueur de file par l'API de gestion (15672). Politiques réseau (MC3) : 5672 depuis les namespaces des services de test, 15672 depuis KEDA seul, 4369 et 25672 fermés, aucun NodePort ni règle d'ALB vers le broker.

## Sécurité du bac à sable

- **MX1, MC5** : identifiants du broker, de KEDA, d'administration et cookie Erlang générés à chaque exécution, injectés par l'entrée standard dans un Secret par service ; empreintes hors du dépôt.
- **MX4** : fichier de définitions versionné sans utilisateurs ni empreintes (topologie et droits seulement).
- **MX5** : aucun mot de passe dans les sorties ; console de gestion par `kubectl port-forward` seulement.
- **MX7** : aucune ressource AWS propre ; le broker disparaît avec le cluster de XP01.

## Étapes reproductibles

### A. docker compose (2026-10-28)

1. Générer les identifiants, démarrer le broker. Attendu : `rabbitmq-diagnostics check_running` OK ; `rabbitmqctl list_users` sans `guest` (XC9).
2. **XC1** : 1 000 messages point à point, 1 000 en diffusion. Attendu : comptages égaux, aucun `nack`.
3. **XC2** : quatre tentatives interdites par identité de service. Attendu : 100 % `ACCESS_REFUSED`.
4. **XC9** (hors réseau) : `curl -u svc-pdm:… http://localhost:15672/api/overview` → 401 ; liaison, purge, suppression d'une file d'un autre service → refus ; publication avec `user-id` ≠ identité connectée → canal fermé `PRECONDITION_FAILED`.
5. **XC3** : vague, arrêt du consommateur, relance. Attendu : 1 000 identifiants uniques reçus.
6. **XC4** : message malformé et clé dupliquée insérés au milieu d'un flux. Attendu : flux non bloqué, message dans `.ecartes` après 5 livraisons, rejeu par le producteur.
7. **XC5** : `docker restart` du broker. Attendu : reconnexion ≤ 60 s ; aucune perte.

### B. k3d (2026-10-29)

1. Installer le broker et KEDA dans le cluster de XP01. Attendu : pod `Ready`, `ScaledObject` actif.
2. Rejouer XC5 (suppression du pod), **XC6**, **XC7** et la partie réseau de **XC9**. Attendu : conformes ; débogage seulement.

### C. EKS, session 1 de XP01 (2026-10-30, ≈ 1 h)

1. Installation du broker, de KEDA et des services de test. Attendu : pods `Ready`.
2. **XC7** et **XC9** réseau (G7-2 de XP01) depuis `ns-intrus` : `curl --max-time 5` vers 5672, 15672, 4369, 25672. Attendu : délais expirés ; flux témoins passants.
3. **XC1, XC2, XC9** hors réseau, **XC3**.
4. **XC5** : `kubectl exec … -- rabbitmqctl stop_app` puis arrêt du processus (redémarrage du conteneur), puis `kubectl delete pod` ; rejeu de l'outbox de test par un Job lancé depuis le poste. Attendu : 100 % des identifiants.
5. **XC6** : vague de 300 messages, consommateur lent. Attendu : 2 consommateurs ≤ 60 s, retour à 1 après vidage.
6. **XC8** : `kubectl top pod` au repos et pendant une vague de 5 min.
7. Pas d'extinction propre : le broker part avec le `destroy` de XP01 (G3).

## Durée et coût

- **Préparation** : 0,5 j (définitions, génération des identifiants, producteur, consommateur, outbox de test).
- **Exécution** : 0,5 j en local, ≈ 1 h dans la session EKS 1.
- **Coût** : aucun coût propre (compris dans XP01).

## Données collectées

- **CSV** : `docs/mesures/XP02-canal-rabbitmq.csv`, colonnes `horodatage_utc;xp;session;plateforme;critere;etape;valeur;unite;seuil;atteint;commentaire` (comptes envoyés, reçus, uniques, délais de reconnexion et de mise à l'échelle, mémoire).
- **Sorties** : `rabbitmqctl list_users`, `list_permissions -p collector`, `list_queues name messages` avant et après chaque test, dans `docs/mesures/XP02-canal-rabbitmq/`.
- **Captures** : console de gestion (par `port-forward`) pendant XC6, `kubectl get pods -w` pendant XC5.

## Résultats

| # | docker compose | k3d | EKS | Atteint |
|---|---|---|---|---|
| XC1 | Point à point : 1 000 confirmés, 1 000 uniques, confirmation p50 28 ms, p99 73 ms, bout en bout p50 718 ms (rafale à 927 msg/s) ; diffusion par l'outbox : 1 000/1 000 sur chaque file, 0 manquant | — (non rejoué) | `go/` par `port-forward` : point à point 1 000 confirmés, 0 `nack`, 0 non routé, 1 000/1 000 uniques, confirmation p50 99 ms, p99 115 ms (aller-retour du `port-forward`) ; diffusion par l'outbox : 1 000 relayés, 1 000/1 000 uniques sur chaque file, 0 manquant | Oui en local ; EKS : seuils tenus |
| XC2 | 19/19 `ACCESS_REFUSED` (sonde) + 3/3 (`go/`) | 37/37 AMQP avec la sonde différée de 20 s et le consommateur lent en pause | Sonde (pod de `pdm-test`) : 21/21 `ACCESS_REFUSED` (4 identités de service + `keda-monitoring`) ; témoins 2/2 passants | Oui en local ; EKS : seuil tenu |
| XC3 | Arrêt à 326 messages, relance 2 min 17 s après : 1 000 uniques cumulés, 1 doublon absorbé | — | `go/` tué 7 s après le début d'une vague de 1 000 (50 msg/s), 678 en file à l'arrêt, relance 122 s après : 1 000 uniques cumulés, 10 relivraisons, 0 perdu | Oui en local ; EKS : seuil tenu |
| XC4 | Sonde : 6 livraisons puis `.ecartes` ; `go/` `reject` : 2 écartés, 1 000 uniques ; `go/` `requeue` : boucle sans mise à l'écart ; rejeu : 13 000 republiés, absorbés | — | Sonde (partie broker) : 6 livraisons puis `controle.demandes.ecartes`, suivant non bloqué (4/4) ; client `go/` non rejoué | **Non** (client `go/`) ; partie broker conforme sur EKS |
| XC5 | `docker restart` sous charge : prêt en 20 s, reconnexion 4 s après, 10 000/10 000 uniques, 46 doublons absorbés | Pod : reconnexion 12 s ; conteneur : 10 s ; suppression sous charge : 24 et 26 perdus, rejeu de l'outbox, 0 manquant | Conteneur : prêt 18 s après, `svc-controle` reconnecté 9 s après ; pod : prêt 18 s après (replanifié sur l'autre nœud), reconnecté 17 s après ; suppression sous charge (`go/`, relais de 5 000) : relais et consommateurs reconnectés 15 à 24 s après la coupure, `emptyDir` perdu : 4 028 et 4 036 manquants, rejeu de l'outbox : 0 manquant | Oui en local ; EKS : seuils tenus |
| XC6 | — | 1 → 2 en 28 s (pic 4), retour à 1 en 58 s | 1 → 2 consommateurs 25 s après le seuil (pic 4), retour à 1 62 s après vidage | Oui (débogage) ; EKS : seuil tenu |
| XC7 | — | `ns-intrus` → 5672 et 15672 fermés ; témoins ouverts | `ns-intrus` → 5672 et 15672 en délai expiré (5 s) ; `pdm-test` → 15672 et 15692 fermés ; témoins `pdm-test` → 5672 et KEDA → 15672 ouverts (10/10 avec XC9) ; contre-épreuve : DNS et `kube-dns:53` joignables depuis `ns-intrus` | Oui (débogage) ; EKS : seuil tenu |
| XC8 | 233 Mio repos, 236 Mio vague (200 msg/s, 5 min) ; CPU au plafond compose (1 vCPU) par pics | 225 Mio repos, 239 Mio vague ; CPU 0,9 à 2,4 vCPU | Mémoire 190 Mio repos, 201 Mio vague (200 msg/s, 5 min) ; nœuds 2 → 2 ; CPU repos 209 à 236 m (575 m juste après XC5), vague 305 à 619 m pour une demande de 250 m | Mémoire et nœuds oui ; CPU au-dessus de la demande sous vague |
| XC9 | 20/20 refus hors réseau, `guest` absent | 4369 et 25672 fermés (4/4) ; API de gestion 401 pour les services | Hors réseau : 16/16 refus (liaison, purge, suppression, `keda-monitoring`, `user-id` usurpé `PRECONDITION_FAILED` ×2, `guest`) + API de gestion 4/4 401 ; `guest` absent ; réseau : 4369 et 25672 fermés (4/4, délais expirés, G7-2) | Oui en local ; EKS : seuil tenu |

Relevés : [`docs/mesures/XP02-canal-rabbitmq.csv`](../mesures/XP02-canal-rabbitmq.csv) et sorties d'état, de sonde et de vague dans [`docs/mesures/XP02-canal-rabbitmq/`](../mesures/XP02-canal-rabbitmq/). Le CSV suit le format de celui d'XP00 (`horodatage_utc;xp;point;commande;resultat;code_erreur;politique_citee;decision`) et non les colonnes annoncées dans « Données collectées » ; les scripts du bac à sable écrivent encore l'ancien format. Session EKS (`EKS2`) : lignes de synthèse au format XP00 dans le CSV principal ; relevés bruts des scripts et des programmes `go/` (format à 11 colonnes) dans [`20261120-eks-EKS2-releves.csv`](../mesures/XP02-canal-rabbitmq/20261120-eks-EKS2-releves.csv).

## Difficultés

À compléter par Martin pendant l'exécution, sans réécriture.

- **XC4, client `go/`** : sur RabbitMQ 4.3.7, `basic.nack` avec remise en file (`multiple` faux ou vrai) ne pose pas `x-delivery-count` et ne fait pas avancer le compteur des files quorum ; `basic.reject` le fait (6 livraisons puis `.ecartes`). Le consommateur `go/` en politique `requeue` utilise `Nack(false, true)` : deux messages piégés ont tourné 44 183 fois en 3 min sans être écartés (boucle chaude, flux valide non bloqué). Correction à faire par les devs : `Reject(true)` pour la remise en file, et vérifier `libs/platform` et `infra/canal` sur le même point.
- **XC8, CPU** : au repos, une sonde de disponibilité `exec` (`check_running` puis `check_local_alarms`) coûte 17,3 s de CPU (une VM Erlang par appel, autant d'ordonnanceurs que de cœurs de l'hôte) ; au repos, le pod consomme 1,17 vCPU, alors que les ordonnanceurs du broker sont à 100 % en sommeil. Mêmes sondes dans le chart de production `infra/canal/templates/statefulset.yaml` : sur deux t3.medium (crédits CPU), risque réel. Pistes : sonde `tcpSocket` ou HTTP, période allongée, `RABBITMQ_CTL_ERL_ARGS` limitant les ordonnanceurs de la CLI, ou limite CPU ; à mesurer en session EKS.
- **k3d, sonde immédiate** : les 6 premières connexions d'un pod neuf de `pdm-test` échouent (`AMQPConnectionError`) pendant quelques secondes, le temps que les politiques réseau de k3s prennent en compte l'adresse du pod ; une attente de 20 s les fait passer. Le témoin de lecture échoue tant que le consommateur lent de XC6 (`svc-controle`) lit `controle.demandes` : pause KEDA (`autoscaling.keda.sh/paused-replicas: "0"`) pendant la sonde. Les deux sont des artefacts de banc ; le script `k3d.sh` devrait attendre et suspendre le consommateur lent.
- **XC5 compose, premier essai** : le relevé d'état avant coupure prend environ 25 s, la production de 3 000 événements était finie avant la coupure ; essai rejoué avec 10 000 événements pour avoir des messages en vol.
- **XC1, latences de bout en bout** : mesurées en rafale (publication sans limite de débit, puis relais après production pour la diffusion) ; elles traduisent la file d'attente, pas la latence du canal.
- **XC3** : arrêt du consommateur par fin de processus (pas de Ctrl+C sous Windows) : 1 message traité mais non acquitté, relivré et absorbé par l'idempotence.
- **K3d XC5 avec l'outbox** : programmes `go/` lancés depuis le poste par `kubectl port-forward` (pas d'image des programmes) ; la reconnexion passe donc aussi par la relance du port-forward.
- **Écart avec `infra/canal` (US-011)** : sondes `exec` identiques (voir XC8) ; le reste de la topologie n'a pas été comparé en détail.
- **Session EKS (2026-11-20, `EKS2`)** : cluster partagé avec XP03 (namespace `identite-test`, aucune de ses ressources ni règle d'ALB touchée ; les mesures de CPU des nœuds incluent sa charge). Aucun défaut de script bloquant, aucun script modifié. Les politiques réseau de XP01 (`sortie-sans-imds-ni-vpc`) n'étaient pas posées dans `pdm-test`, `controle-test` et `ns-intrus` pendant la session : XC7 et XC9 réseau reposent sur les seules politiques de XP02. XC1, XC3 et XC5 (outbox) passent par `kubectl port-forward` depuis le poste et un Postgres jetable local pour l'outbox et le journal de réception : latences et délais de reconnexion incluent le `port-forward` (relancé en boucle). Pendant XC1 point à point et XC3, le consommateur lent de XC6 est mis en pause (`paused-replicas: "0"`), sinon il lit `controle.demandes`.
- **XC5 EKS, pertes à la suppression du pod** : 4 028 et 4 036 événements perdus (24 et 26 sur k3d) : les consommateurs, à ≈ 100 msg/s par le `port-forward`, étaient loin derrière le relais au moment de la coupure, la file contenait des milliers de messages confirmés perdus avec l'`emptyDir`. Le rejeu de l'outbox les rétablit tous ; l'ampleur de la perte dépend du retard des consommateurs.
- **XC8 EKS, premier relevé au repos** : 575 m de CPU, pris 20 s après le retour du pod de XC5 (démarrage) ; les relevés suivants et ceux pris après la session sont entre 209 et 236 m.

## Limites

- Producteur et consommateur de test, pas le client du POC (`libs/platform/`) : la reconnexion et la validation stricte du POC sont reprises en AC du sprint 1.
- Un seul nœud RabbitMQ : la haute disponibilité du broker n'est pas testée (point unique de défaillance assumé par D03).
- NATS JetStream n'est expérimenté qu'en cas de bascule.

## Conclusion

- **Relecture architecture** :
- **Relecture sécurité** (droits du canal, politiques réseau) :
- **Verdict de Martin** :
- **Mise à jour de D05** :
