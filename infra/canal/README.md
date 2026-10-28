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
- Secret `canal-supervision`, clé `host` (`http://keda:<mot de passe>@rabbitmq.canal.svc:15672/collector`, API de gestion), dans le namespace `controle` : lu par la `TriggerAuthentication` du scaler KEDA (US-026). L'identité `keda` (tag `monitoring`, droits vides) lit l'inventaire des files ; 15672 reste fermé aux pods de `controle`, ouvert au seul `keda-operator`.
- Aucune déclaration ni liaison (refus 403) : `Publish` et `Consume` sur la topologie existante seulement. La déclaration passive reste permise (elle ne crée rien, constaté sur 4.3.7).
- Propriété `user-id` égale à l'identité connectée sur chaque publication (MC2) ; le broker refuse un `user-id` usurpé (406).
- Consommation par `basic.consume`, jamais `basic.get` : le compteur de livraisons des files quorum n'avance pas avec `basic.get`.
- Tests d'intégration (testcontainers-go) : même image, montages de `broker/conf.d/`, `broker/enabled_plugins` et `broker/definitions/10-topologie.json`, plus un `00-utilisateurs.json` produit par le test au format de `outils/identifiants.sh` (une identité citée dans les droits mais absente fait échouer le démarrage du broker : `BOOT FAILED`, constaté sur 4.3.7).

## Utilisation (Git Bash)

```bash
infra/canal/outils/local.sh up                 # local : identifiants, broker prêt, guest absent
infra/canal/outils/verifier.sh local           # droits, mise à l'écart, API de gestion, absence de secret
infra/canal/outils/local.sh down               # supprime conteneur et réseau, efface .genere/

infra/canal/outils/kubernetes.sh up            # namespaces, Secrets, chart (KUBE_CONTEXT obligatoire)
infra/canal/outils/verifier.sh kubernetes      # + réseau depuis un service et depuis un namespace intrus, Services internes
infra/canal/outils/kubernetes.sh console       # console sur 127.0.0.1:15672 (admin-canal)
infra/canal/outils/kubernetes.sh down          # chart, Secrets, .genere/ (namespaces conservés)
```

Prérequis : Docker, `openssl`, `xxd`, `curl` ; en plus pour Kubernetes : `kubectl`, Helm 4 (et `k3d` pour importer l'image de sonde dans un cluster k3d). `SONDE_IMAGE` remplace l'image de sonde (par défaut construite depuis `verification/`).

**Garde-fou de contexte** : `kubernetes.sh` et `verifier.sh kubernetes` exigent `KUBE_CONTEXT` et le refusent hors des motifs `k3d-*` (poste) et `*collector*` (cluster du projet), avant toute commande. Chaque appel passe `--context` (`--kube-context` pour Helm) : le contexte courant n'est jamais lu, le poste en porte d'autres. Un `KUBECONFIG` au format `/c/...` est converti par `cygpath -m` ; sans cela, sous `MSYS_NO_PATHCONV=1`, `kubectl` l'ignorait et lisait `~/.kube/config`. Exemple : `KUBECONFIG=<kubeconfig k3d> KUBE_CONTEXT=k3d-<nom> infra/canal/outils/kubernetes.sh up`.

Le chart ne crée ni namespace ni RBAC (D10) : `kubernetes.sh namespaces` les pose avec la session du poste ; `up.sh` (US-004) l'appellera. Les Secrets existent avant l'installation du chart ; de nouveaux identifiants imposent un redémarrage du broker, fait par `up`.

## Écarts et limites

- **AC3, stockage éphémère** : `emptyDir`, qui survit au redémarrage du conteneur mais pas à la suppression du pod (constaté sur k3d). Couverture : filet de la Place de marché (renvoi des articles « en contrôle »), rejeu de l'outbox (MC9).
- Pas de TLS interne : confinement réseau seul (D05, écart documenté pour la V1).
- Collecte des métriques (15692) fermée tant que D09 n'est pas installé (`metriques.namespaces`).
- **Sondes** : démarrage, disponibilité et vivacité en `tcpSocket` sur 5672, faites par le kubelet. Les sondes `exec` `rabbitmq-diagnostics` démarraient une VM Erlang à chaque passage (~17 s CPU, XP02) et tenaient le broker à 0,5-1,1 vCPU au repos pour une demande de 0,25. Le port AMQP n'ouvre qu'en fin de démarrage (définitions importées) et se ferme si l'application s'arrête : vivacité en échec, redémarrage après 4 × 15 s. Pas de sonde HTTP sur la gestion : il faudrait une identité dans la spécification du pod. Non couvert : une VM figée dont le port reste ouvert (le noyau accepte encore la connexion), et les alarmes mémoire ou disque, qui ne retirent plus le pod du Service (le retirer empêcherait aussi les consommateurs de vider les files) ; elles relèvent de la supervision (D09).

## En cas de verdict ECS (XP01, 2026-11-02)

Restent valables : `broker/`, `outils/identifiants.sh`, la sonde, `compose.yml`. À refaire : le chart et `kubernetes.sh` deviennent une définition de tâche ECS (Terraform) avec volume éphémère, des secrets Secrets Manager par service à la place des Secrets Kubernetes, des groupes de sécurité à la place des politiques réseau, la découverte de service (Cloud Map) à la place du DNS `rabbitmq.canal.svc`, et une console par session SSM ; le scaler KEDA est remplacé par une mise à l'échelle Application Auto Scaling sur métrique de file.

## Vérifications du 2026-10-26 (poste)

- docker compose : `verifier.sh local` 64/64 (AC2, AC4, AC5, MC1, MC4, MC8 avec 5 livraisons, AC6).
- k3d (k3s 1.35.5, un nœud, PSS `restricted`) : `kubernetes.sh up` en 53 s ; `verifier.sh kubernetes` 79/79 (plus AC7 : 5672 seul ouvert depuis `place-de-marche`, tout fermé depuis un namespace intrus, Services `ClusterIP` seuls ; AC6 : aucun secret dans le dépôt, le manifeste et les valeurs Helm, le journal du broker).
- `helm lint`, `kubeconform -strict` (9/9), `kubectl apply --dry-run=server`, shellcheck 0.11.0 sans remarque, Trivy `config` : 7 constats LOW, aucun HIGH ni CRITICAL.

## Sondes légères et garde-fou de contexte du 2026-10-28 (k3d)

- Broker au repos, 5 min après 90 s de stabilisation : sondes `exec` 506 à 1 113 mCPU (`kubectl top`, ~840 en moyenne) ; sondes `tcpSocket` 4 à 9 mCPU (cgroup : 4 mCPU en moyenne). Pod prêt 14 s après la mise à jour.
- `rabbitmqctl stop_app` (nœud Erlang vivant, que `rabbitmq-diagnostics ping` ne voyait pas) : retrait du Service en ~25 s, redémarrage par la vivacité à ~60 s, de nouveau prêt à ~78 s.
- `verifier.sh kubernetes` 79/79, `helm lint`, `kubeconform -strict` (9/9).
- Garde-fou de contexte : sans `KUBE_CONTEXT` et avec le contexte EKS du poste, `kubernetes.sh up` et `verifier.sh kubernetes` refusent sans aucun appel à `kubectl`, `helm`, `k3d`, `docker` ni `curl` (remplacés par des espions) ; avec `k3d-*` et un `KUBECONFIG` en `/c/...`, `verifier.sh kubernetes` 79/79. shellcheck sans remarque.
