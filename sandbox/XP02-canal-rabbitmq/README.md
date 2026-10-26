# XP02 — Canal RabbitMQ : bac à sable local

Parties locales du protocole [XP02](../../docs/experimentations/XP02-canal-rabbitmq.md) : Local 1 (docker compose) et Local 2 (k3d). La session EKS réutilise les mêmes fichiers (`broker/`, `k8s/`) et les mêmes scripts avec `PLATEFORME=eks`.

## Contenu

| Chemin | Rôle |
|---|---|
| `versions.env` | Images et chart épinglés (version et empreinte) |
| `broker/conf.d/10-collector.conf` | Configuration RabbitMQ commune (remplace le `conf.d` de l'image) |
| `broker/definitions/10-topologie.json` | Vhost, échanges, files quorum, liaisons, politiques, droits (versionné, sans utilisateur ni empreinte) |
| `docker-compose.yml` | Local 1 : broker seul, ports sur `127.0.0.1` uniquement |
| `k8s/` | Local 2 et EKS : namespaces, StatefulSet, politiques réseau (MC3), KEDA et consommateur lent (XC6) |
| `sonde/` | Sonde AMQP (Python, `pika` épinglé par empreinte) : droits refusés, `user-id`, `guest`, mise à l'écart |
| `scripts/` | Identifiants, cycle de vie, mesures (voir ci-dessous) |
| `go/` | Producteur, consommateur, outbox et rejeu de test (fourni à part) |
| `.env`, `.genere/` | Générés à chaque session, ignorés par git : mots de passe, empreintes, cookie Erlang, kubeconfig d'essai |

## Prérequis

Docker 28.5.1 (compose v2.40), k3d 5.9.0, kubectl, Helm 4.1.3, `jq`, `openssl` et `xxd` (Git Bash) dans le `PATH`. Scripts en Bash (`set -euo pipefail`), à lancer depuis Git Bash ; `MSYS_NO_PATHCONV=1` est posé par `scripts/lib.sh`.

Mesures écrites dans `docs/mesures/XP02-canal-rabbitmq.csv` et `docs/mesures/XP02-canal-rabbitmq/` ; `MESURES_DIR=<répertoire>` les redirige pour un essai à blanc. `SESSION` (défaut : date UTC) et `PLATEFORME` (`compose`, `k3d`, `eks`) renseignent les colonnes du CSV.

## Identités et droits (D05, MC1, MC2, MC4)

| Identité | Étiquette | `configure` | `write` | `read` |
|---|---|---|---|---|
| `svc-pdm` | — | vide | `controle.demandes`, `pdm.evenements` (clé `publication.decidee.v1` seule) | `pdm.resultats-controle` |
| `svc-controle` | — | vide | `controle.resultats` | `controle.demandes` |
| `svc-interactions` | — | vide | vide | `interactions.publication-decidee` |
| `svc-test` | — | vide | vide | `test.publication-decidee` |
| `keda-monitoring` | `monitoring` | vide | vide | vide |
| `admin-canal` | `administrator` | tout | tout | tout |

`guest` n'est jamais créé : les définitions sont importées au démarrage, et `loopback_users.guest = true` reste en défense. Les mots de passe sont tirés par `scripts/identifiants.sh`, hachés au format `rabbit_password_hashing_sha256` sur le poste, et n'apparaissent ni en argument de commande ni dans les sorties.

Files quorum ; politique par file de travail : `delivery-limit` 5, mise à l'écart `at-least-once` vers `collector.ecartes` puis `<file>.ecartes`, `max-length` 10 000 en `reject-publish`, `message-ttl` 24 h ; files `.ecartes` : 7 jours (MC8).

## Contrat attendu des programmes de `go/`

- Connexion : compose, `amqp://<identité>:<mot de passe>@127.0.0.1:5672/collector` (mots de passe dans `.env` : `MDP_SVC_PDM`, `MDP_SVC_CONTROLE`…), ou hôte `rabbitmq` sur le réseau docker `xp02_canal` ; k3d et EKS, variable `AMQP_URL` du Secret `canal-amqp` du namespace du service (`pdm-test`, `controle-test`, `interactions-test`, `lecture-test`).
- Aucune déclaration (droit `configure` vide) : `Consume` et `Publish` sur la topologie existante seulement.
- Propriété `user-id` égale à l'identité connectée sur chaque publication (MC2).
- Consommation par `basic.consume` (`Channel.Consume`), jamais `basic.get` : le compteur de livraisons des files quorum n'avance pas avec `basic.get` suivi d'un `nack` (constaté sur 4.3.7).
- Lancer les sondes **avant** les programmes de `go/` : la sonde lit `controle.demandes`.

## Déroulé

### A. docker compose (protocole § A)

```bash
scripts/compose.sh up                              # A1 : identifiants, broker prêt, guest absent
scripts/etat.sh avant-xc1                          # sorties avant/après chaque test
PLATEFORME=compose scripts/sonde.sh refus          # A3 (XC2) + A4 (XC9 hors réseau, API de gestion 401)
PLATEFORME=compose scripts/sonde.sh ecartes        # A6 (XC4, partie broker) : 6 livraisons puis .ecartes
# A2 (XC1), A5 (XC3), A6 (XC4 bout en bout) : programmes de go/
PLATEFORME=compose scripts/xc5-redemarrage.sh conteneur svc-pdm svc-controle   # A7 (XC5), go/ en cours
PLATEFORME=compose scripts/xc8-vague.sh 300 200    # XC8 : repos puis vague de 5 min (PerfTest)
scripts/compose.sh down                            # supprime conteneur, réseau, .env et .genere/
```

### B. k3d (protocole § B)

Dans le cluster de XP01 (contexte `k3d-*` courant), ou dans un cluster jetable :

```bash
scripts/k3d.sh cluster-essai up && export KUBECONFIG="$(cygpath -m .genere/kubeconfig-xp02-essai)"   # facultatif
scripts/k3d.sh up                                  # B1 : broker, Secrets par service, politiques, KEDA, consommateur lent
PLATEFORME=k3d scripts/sonde.sh refus              # XC2, XC9 depuis un pod de pdm-test (API par port-forward)
scripts/k3d.sh reseau                              # XC7 + XC9 réseau : ns-intrus et pdm-test, témoins 5672 et KEDA
scripts/k3d.sh xc6                                 # XC6 : 300 messages, 1 -> >= 2 -> 1 consommateur
PLATEFORME=k3d scripts/xc5-redemarrage.sh pod svc-controle        # XC5 : suppression du pod (emptyDir perdu)
PLATEFORME=k3d scripts/xc5-redemarrage.sh conteneur svc-controle  # XC5 : redémarrage du conteneur
PLATEFORME=k3d scripts/xc8-vague.sh 300 200
scripts/k3d.sh down                                # retire XP02 (KEDA compris), garde pdm-test, controle-test, ns-intrus
scripts/k3d.sh cluster-essai down                  # si le cluster jetable a été créé
```

Le consommateur lent de XC6 est un PerfTest (identité `svc-controle`, 200 ms par message) : XC6 ne dépend pas de `go/`.

## Constats de préparation (2026-10-26, poste de Martin, à blanc)

- Docker compose : broker prêt en 17 s ; sonde `refus` 43/43 conforme ; sonde `ecartes` 4/4 ; XC5 (`docker restart`) : prêt en 17 s, client PerfTest reconnecté en 4 s.
- k3d (cluster jetable 1 serveur + 2 agents) : sonde `refus` 43/43 ; réseau 10/10, avec contre-épreuve (ports ouverts depuis `ns-intrus` une fois les politiques retirées) ; XC6 : 2 consommateurs 24 s après le seuil, retour à 1 59 s après vidage ; XC5 : 8 à 9 s de reconnexion (pod et conteneur) ; mémoire 130 à 205 Mio.
- **Identité KEDA** : avec le seul tag `monitoring` et aucune entrée de droits sur `collector`, l'API de gestion répond 404 sur la file. Une entrée de droits **vides** (`configure`, `write`, `read` à `""`) suffit : 200 sur l'API, refus AMQP en lecture, écriture et purge. MC4 (« sans droit sur le vhost ») est à lire ainsi.
- **Limite de livraisons** : `delivery-limit` 5 donne 6 livraisons (1 + 5 relivraisons) avant `.ecartes`.
- **Définitions en répertoire** : les fichiers `00-utilisateurs.json` et `10-topologie.json` sont bien fusionnés au démarrage. En Kubernetes, un volume projeté monté en répertoire échoue (`eisdir` sur `..<horodatage>`) : montage fichier par fichier (`subPath`).
