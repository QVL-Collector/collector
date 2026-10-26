# XP02 — Canal RabbitMQ dans l'orchestrateur

| Champ | Valeur |
|---|---|
| Décision testée | [D05](../decisions/D05-communication-interservice.md) (statut `Candidate (XP02)`) |
| Dates prévues | docker compose 2026-10-28 ; k3d 2026-10-29 ; EKS session 1 2026-10-30 (≈ 1 h) |
| Exécutée par | Martin |
| Scripts | `sandbox/XP02-canal-rabbitmq/` (fichier de définitions, génération des identifiants, producteur, consommateur, outbox et rejeu de test) |
| Dépend de | [XP01](XP01-orchestrateur.md) : cluster EKS de la session 1, politiques réseau actives (G7-5) |

## Objectif et critères de succès

Objectif : prouver la publication, la consommation et les droits du canal RabbitMQ déployé dans l'orchestrateur, exemple cité par la consigne ; confirmer RabbitMQ ou basculer vers NATS JetStream.

Hypothèse : le modèle de droits de RabbitMQ (`configure` vide, `write` et `read` par ressource, droits par clé de routage, `user-id` vérifié) tient ES-38 par configuration, et les politiques réseau d'EKS ferment la gestion et la distribution Erlang.

### Critères

| # | Mesure | Seuil | Bloquant | Si échec |
|---|---|---|---|---|
| XC1 | 1 000 messages point à point (`controle.demandes`) et 1 000 en diffusion (`pdm.evenements`, deux files abonnées), confirmations de publication | 100 % des confirmations ; 100 % des identifiants reçus par chaque abonné | **Oui** | Bascule vers NATS JetStream |
| XC2 | Avec l'identité d'un service : publier sur l'échange d'un autre ; lire la file d'un autre ; déclarer une file ou un échange ; publier avec une clé de routage non accordée | Refus dans 100 % des cas (`ACCESS_REFUSED`) | **Oui** | Bascule vers NATS, XC2 rejoué sur la liste complète de `$JS.API` |
| XC3 | Consommateur arrêté pendant une vague de 1 000 messages, relancé après 2 min | Aucun message perdu (ENF-12) | **Oui** | Bascule vers NATS |
| XC4 | Message malformé et message à clé JSON dupliquée dans un flux valide | Écarté sans bloquer les suivants ; mis à l'écart vers `.ecartes` après 5 livraisons ; rejouable | Non | À corriger avant J4 |
| XC5 | Redémarrage du conteneur du broker, puis suppression de son pod (`emptyDir` perdu) | Producteurs et consommateurs reconnectés sans intervention ≤ 60 s après le retour `Ready` ; après suppression, filet et rejeu de l'outbox de test rétablissent 100 % des événements (identifiants uniques) | **Oui** | Reconnexion instable : bascule vers NATS. Rejeu insuffisant : pilote EBS CSI réexaminé |
| XC6 | KEDA sur `controle.demandes` (scaler `rabbitmq` en mode `http`, identité `monitoring`) | 1 → au moins 2 consommateurs ≤ 60 s après le seuil (5 messages) ; retour à 1 après vidage | Non | Contrôle sur la CPU ou à nombre fixe |
| XC7 | Pod quelconque (`ns-intrus`) vers 5672 et 15672 ; flux témoins : service → 5672, KEDA → 15672 | Refus constatés ; flux témoins passants | **Oui** (reclassé par l'avis sécurité de D14) | Politique corrigée et rejouée ; politiques inopérantes : G7 de XP01, bascule de D03 |
| XC8 | Empreinte au repos et sous vague | Dans la demande (0,25 vCPU, 512 Mio) et la limite (1 Gio) ; aucun nœud ajouté au repos | Non | Revue du dimensionnement ou bascule vers NATS |
| XC9 | Identité de service : connexion à l'API de gestion, liaison d'une file, purge et suppression de la file d'un autre service ; pod applicatif vers 4369 et 25672 ; publication avec un `user-id` usurpé ; utilisateur `guest` | Refus dans 100 % des cas ; `guest` absent | **Oui** | Erreur de configuration : corrigée et rejouée avant verdict. Défaut structurel (`user-id` non vérifié, droits contournables) : bascule vers NATS |

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
| Image RabbitMQ `4.x-management` | à fixer avant exécution, par empreinte | À fixer |
| KEDA (chart) | à fixer avant exécution | À fixer |
| Docker / docker compose | 28.5.1 / à relever | À relever |
| k3d, image k3s, EKS, kubectl, Helm | voir [XP01](XP01-orchestrateur.md) | Communes |
| Producteur et consommateur de test | image du bac à sable, épinglée par empreinte | À fixer |
| Outil de vague (XC8) | candidat : PerfTest de RabbitMQ, épinglé | À fixer |

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
| XC1 | | | | |
| XC2 | | | | |
| XC3 | | | | |
| XC4 | | | | |
| XC5 | | | | |
| XC6 | — | | | |
| XC7 | — | | | |
| XC8 | | | | |
| XC9 | | | | |

## Difficultés

À compléter par Martin pendant l'exécution, sans réécriture.

## Limites

- Producteur et consommateur de test, pas le client du POC (`libs/platform/`) : la reconnexion et la validation stricte du POC sont reprises en AC du sprint 1.
- Un seul nœud RabbitMQ : la haute disponibilité du broker n'est pas testée (point unique de défaillance assumé par D03).
- NATS JetStream n'est expérimenté qu'en cas de bascule.

## Conclusion

- **Relecture architecture** :
- **Relecture sécurité** (droits du canal, politiques réseau) :
- **Verdict de Martin** :
- **Mise à jour de D05** :
