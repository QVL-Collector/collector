# infra/canal — Canal asynchrone RabbitMQ (US-011)

Broker RabbitMQ 4.3.7 du canal entre services ([D05](../../docs/decisions/D05-communication-interservice.md)), promu depuis le bac à sable [XP02](../../sandbox/XP02-canal-rabbitmq/README.md). La topologie, les identités et les droits sont posés par le fichier de définitions, jamais par les services (M-CAN-06). Mesures servies : MC1, MC3, MC4, MC5, MC8.

## Contenu

| Chemin | Rôle | Dépend de l'orchestrateur |
|---|---|---|
| `broker/conf.d/10-collector.conf` | Configuration du broker (remplace le `conf.d` de l'image) | Non |
| `broker/enabled_plugins` | Gestion et Prometheus | Non |
| `broker/definitions/10-topologie.json` | Vhost, échanges, files quorum, liaisons, politiques, droits (sans utilisateur ni empreinte) | Non |
| `outils/identifiants.sh` | Mots de passe, empreintes SHA-256 salées et cookie Erlang de la session, dans `.genere/` | Non |
| `compose.yml`, `outils/local.sh` | Broker seul en local, ports sur `127.0.0.1` | Non (Docker) |
| `verification/` | Sonde AMQP (Python, `pika` épinglé par empreinte) et son image | Non |
| `Chart.yaml`, `values.yaml`, `templates/` | Chart Helm : StatefulSet, Services internes, ConfigMap, politiques réseau | **Kubernetes** |
| `outils/kubernetes.sh` | Namespaces, Secrets par l'entrée standard, installation du chart, console par `port-forward` | **Kubernetes** |
| `outils/verifier.sh` | Vérification après déploiement (`local` ou `kubernetes`) | Mode `kubernetes` seulement |
| `.genere/` | Identifiants de la session : ignoré par git et par Helm (`.helmignore`), effacé par `down` | — |

L'image est épinglée une seule fois, dans `values.yaml` ; `outils/lib.sh` la relit pour docker compose.

## Topologie

| Échange | Écrit par | Clé | File | Lue par |
|---|---|---|---|---|
| `controle.demandes` (direct) | `marche` | `v1` | `controle.demandes` | `controle` |
| `controle.resultats` (direct) | `controle` | `v1` | `pdm.resultats-controle` | `marche` |
| `pdm.evenements` (topic) | `marche` | `publication.decidee.v1` | `interactions.publication-decidee` | `interactions` |
| `collector.ecartes` (direct, interne) | broker | nom de la file | `<file>.ecartes` | `admin-canal` (diagnostic) |

Files quorum. Politique de chaque file de travail : `delivery-limit` **4**, soit **5 livraisons** au total avant mise à l'écart (constaté en XP02 : la limite compte les relivraisons, 5 donne 6 livraisons), mise à l'écart `at-least-once`, `max-length` 10 000 en `reject-publish`, durée de vie 24 h. Files `.ecartes` : 7 jours, 10 000 messages (MC8).

## Identités et droits (MC1, MC4)

| Identité | Étiquette | `configure` | `write` | `read` |
|---|---|---|---|---|
| `marche` | — | vide | `controle.demandes`, `pdm.evenements` (clé `publication.decidee.v1` seule) | `pdm.resultats-controle` |
| `controle` | — | vide | `controle.resultats` | `controle.demandes` |
| `interactions` | — | vide | vide | `interactions.publication-decidee` |
| `keda` | `monitoring` | vide | vide | vide (entrée de droits vide : sans elle, l'API répond 404 sur la file) |
| `admin-canal` | `administrator` | tout | tout | tout — console du poste, hors de tout Secret de service |

`guest` n'est jamais créé ; `loopback_users.guest` reste en défense. Résiduel : `queue.purge` dépend du droit `read`, donc un service peut vider **sa propre** file (pas celle d'un autre).

## Contrat pour les services

- Secret `canal-amqp`, clé `AMQP_URL` (`amqp://<identité>:<mot de passe>@rabbitmq.canal.svc:5672/collector`), dans le namespace du service : `place-de-marche`, `controle`, `interactions`.
- Aucune déclaration ni liaison (refus 403) : `Publish` et `Consume` sur la topologie existante seulement. La déclaration passive reste permise (elle ne crée rien, constaté sur 4.3.7).
- Propriété `user-id` égale à l'identité connectée sur chaque publication (MC2) ; le broker refuse un `user-id` usurpé (406).
- Consommation par `basic.consume`, jamais `basic.get` : le compteur de livraisons des files quorum n'avance pas avec `basic.get`.
- Tests d'intégration (testcontainers-go) : même image, montages de `broker/conf.d/`, `broker/enabled_plugins` et `broker/definitions/10-topologie.json`, plus un `00-utilisateurs.json` produit par le test au format de `outils/identifiants.sh` (une identité citée dans les droits mais absente fait échouer le démarrage du broker : `BOOT FAILED`, constaté sur 4.3.7).

## Utilisation (Git Bash)

```bash
infra/canal/outils/local.sh up                 # local : identifiants, broker prêt, guest absent
infra/canal/outils/verifier.sh local           # droits, mise à l'écart, API de gestion, absence de secret
infra/canal/outils/local.sh down               # supprime conteneur et réseau, efface .genere/

infra/canal/outils/kubernetes.sh up            # namespaces, Secrets, chart (KUBE_CONTEXT, sinon contexte courant)
infra/canal/outils/verifier.sh kubernetes      # + réseau depuis un service et depuis un namespace intrus, Services internes
infra/canal/outils/kubernetes.sh console       # console sur 127.0.0.1:15672 (admin-canal)
infra/canal/outils/kubernetes.sh down          # chart, Secrets, .genere/ (namespaces conservés)
```

Prérequis : Docker, `openssl`, `xxd`, `curl` ; en plus pour Kubernetes : `kubectl`, Helm 4 (et `k3d` pour importer l'image de sonde dans un cluster k3d). `SONDE_IMAGE` remplace l'image de sonde (par défaut construite depuis `verification/`).

Le chart ne crée ni namespace ni RBAC (D10) : `kubernetes.sh namespaces` les pose avec la session du poste ; `up.sh` (US-004) l'appellera. Les Secrets existent avant l'installation du chart ; de nouveaux identifiants imposent un redémarrage du broker, fait par `up`.

## Écarts et limites

- **AC3, stockage éphémère** : `emptyDir`, qui survit au redémarrage du conteneur mais pas à la suppression du pod (constaté sur k3d). Couverture : filet de la Place de marché (renvoi des articles « en contrôle »), rejeu de l'outbox (MC9).
- Pas de TLS interne : confinement réseau seul (D05, écart documenté pour la V1).
- Collecte des métriques (15692) fermée tant que D09 n'est pas installé (`metriques.namespaces`).

## En cas de verdict ECS (XP01, 2026-11-02)

Restent valables : `broker/`, `outils/identifiants.sh`, la sonde, `compose.yml`. À refaire : le chart et `kubernetes.sh` deviennent une définition de tâche ECS (Terraform) avec volume éphémère, des secrets Secrets Manager par service à la place des Secrets Kubernetes, des groupes de sécurité à la place des politiques réseau, la découverte de service (Cloud Map) à la place du DNS `rabbitmq.canal.svc`, et une console par session SSM ; le scaler KEDA est remplacé par une mise à l'échelle Application Auto Scaling sur métrique de file.

## Vérifications du 2026-10-26 (poste)

- docker compose : `verifier.sh local` 62/62 (AC2, AC4, AC5, MC1, MC4, MC8 avec 5 livraisons, AC6).
- k3d (k3s 1.35.5, un nœud, PSS `restricted`) : `kubernetes.sh up` en 53 s ; `verifier.sh kubernetes` 75/75 (plus AC7 : 5672 seul ouvert depuis `place-de-marche`, tout fermé depuis un namespace intrus, Services `ClusterIP` seuls ; AC6 : aucun secret dans le dépôt, le manifeste et les valeurs Helm, le journal du broker).
- `helm lint`, `kubeconform -strict` (9/9), `kubectl apply --dry-run=server`, shellcheck 0.11.0 sans remarque, Trivy `config` : 7 constats LOW, aucun HIGH ni CRITICAL.
