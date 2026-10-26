# D05 — Communication interservice (synchrone, asynchrone, broker)

| Champ | Valeur |
|---|---|
| Statut | Candidate (XP en D14) |
| Date | 2026-10-26 |
| Décideur | Martin |
| Dépend de | [D02](D02-decoupage-applicatif.md), [D03](D03-hebergement-orchestration.md), [D04](D04-stack.md) |
| Impacte | D06, D07, D09, D11, D13, D14, [architecture-technique.md](../architecture/architecture-technique.md) § 6, § 8, § 9 et § 10, [exigences-securite.md](../securite/exigences-securite.md) (ES-31, ES-38), `contracts/`, `libs/platform/` |
| Critères grille | C2, C1, C8, C5, C9 |

## Contexte et besoin

D05 fixe comment les quatre services de D02 (option B, POC P2) échangent : le broker du canal asynchrone, sa topologie, le format et la validation des messages, le mécanisme anti-perte, et le protocole de l'unique appel synchrone.
D05 ne choisit ni le serveur d'autorisation ni le format des jetons (D06), ni le mécanisme de la réservation atomique (D07), ni la collecte des traces (D09).

### Flux à porter

Source : D02, « Périmètre du POC » et « Conséquences » ; architecture § 6.

| Flux | Mode | Producteur → consommateur | Dans le POC |
|---|---|---|---|
| Demande de contrôle | Asynchrone, point à point | Place de marché → Contrôle | Sprint 1 (WS-2) |
| Résultat de contrôle | Asynchrone, point à point | Contrôle → Place de marché | Sprint 1 |
| « Décision de publication » | Asynchrone, diffusion | Place de marché → Interactions, abonnement de test | Sprint 3 |
| Réserver un article | Synchrone, authentifié | Transactions → Place de marché | Sprint 3 |
| Prix modifié, commande payée, alertes de fraude | Asynchrone, diffusion | Place de marché, Transactions, Contrôle (adaptateur) → abonnés | V1, documenté |

Transactions n'utilise pas le canal dans le POC. Le canal du sprint 1 doit déjà permettre la diffusion du sprint 3 (D02).

### Exigences

| Source | Exigence | Conséquence pour D05 |
|---|---|---|
| ENF-12 | « Aucun événement métier perdu […] si son destinataire est momentanément indisponible. » | Canal durable ; aucun événement perdu entre l'écriture d'un état et sa transmission |
| ES-38 | Une identité par service sur le canal ; droits par sujet, en écriture comme en lecture ; aucun service ne crée d'abonnement ; chaque message validé contre son contrat à la réception ; un événement peut suspendre, jamais déclencher un mouvement de fonds | Le canal doit authentifier lui-même chaque service et refuser la création de files ou de consommateurs |
| ES-31 | Consommateur idempotent ; message en échec répété mis à l'écart après un nombre borné d'essais, sans bloquer la file, rejouable pendant une durée bornée ; alerte | Mise à l'écart native ou à coder |
| ES-26, M-CAN-04 | Conservation bornée des messages, files d'erreurs comprises | Durée et taille maximales par file |
| ES-01 | Flux internes chiffrés ou confinés dans un réseau privé non exposé | Canal et appel synchrone joints par le seul réseau du cluster |
| M-CAN-01 à 06, M-SYN-01 | Faux résultat, rejeu, file bloquée, abonné non autorisé, usurpation de l'appel synchrone | Identités, validation stricte, garde de révision, politiques réseau |
| D04 | Client Go maintenu ; validation par JSON Schema avant décodage ; clés dupliquées à refuser, le décodeur standard les acceptant | Exigence sur le client et sur la chaîne de validation |
| Consigne (C1) | Expérimentation valide citée : « broker de messages déployé dans un orchestrateur avec tests de publication et consommation » | Le broker retenu est l'objet d'une expérimentation (D14) |
| C2 | Le POC valide « la communication interservice » | Les deux modes, synchrone et asynchrone, sont exercés dans le POC |

### Contraintes

- **Learner Lab** ([environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md)) : Amazon MQ et MSK indisponibles (`mq:CreateBroker` et MSK refusés). SQS, SNS et EventBridge sont listés, mais autorisés par les seuls rôles IAM : sous `LabRole` unique, tous les services y partagent une identité, et les droits par sujet d'ES-38 ne seraient que « documentés ».
- **D03** : broker en conteneur à une instance dans le cluster EKS, joint par un Service interne, protégé par une politique réseau ; scaler KEDA requis (Contrôle de 1 à 3 réplicas sur sa file) ; pas de pilote EBS CSI sans besoin nommé, car il suppose IRSA, indisponible. Node group de 2 à 4 t3.medium (4 Gio, 17 pods par nœud).
- **D04** : services en Go ; `libs/platform/` porte la connexion au canal ; contrats dans `contracts/` (JSON Schema, OpenAPI), sans type Go partagé.
- **Allumage par session** : la pile est détruite en fin de session ; l'état du broker n'a pas à survivre d'une session à l'autre, seulement à un incident pendant la session.
- **Calendrier** : D05 tranchée avant J4 (condition de WS-2) sur trois points : mode de retour du résultat, mécanisme anti-perte, contenu du contrat v1.

## Options

### Socle commun aux deux options

Ces choix valent quel que soit le broker ; ils ne sont pas départagés par les options.

- **Synchrone : HTTP/JSON interne**, un seul appel, Transactions → Place de marché « réserver ».
  - Port interne distinct de la Place de marché, sur un Service `ClusterIP` qu'aucune règle de l'ALB ne route ; politique réseau : seul Transactions le joint (D03, X12).
  - Contrat OpenAPI versionné dans `contracts/`, erreurs au format Problem Details (D04), délai de réponse borné côté appelant.
  - Nouvel essai sûr : « réserver » pour le même acheteur et le même article renvoie la réservation existante (D02) ; Transactions peut donc réessayer après un délai dépassé.
  - Authentification : jeton de service de Transactions (audience « Place de marché », droit « réserver », courte durée), plus le jeton de l'acheteur propagé et revérifié par la Place de marché (ES-38, M-MOD-02). Émetteur, flux d'obtention et en-têtes : **D06**. Authentification du service appelé par l'appelant (M-SYN-01) : TLS interne ou confinement réseau, **D06 et D08**.
  - Écarté : gRPC. Pour un seul appel, il ajoute une génération de code et un second style de contrat sans gain mesurable.
- **Format des messages : JSON**, enveloppe commune : identifiant unique de l'événement, type, version du contrat, date d'émission, identifiant de corrélation ; contexte de trace W3C dans les en-têtes (D04, D09).
- **Contrats** : un JSON Schema par message et par version dans `contracts/messages/`, `additionalProperties: false`, listes fermées pour les issues et catégories de motif. Contrat v1 du contrôle : propriétés de l'architecture § 8 (identifiant et révision de l'article, nombre de photos, statut du vendeur par identifiant opaque, issue, motif catégorisé, version des règles).
- **Validation à la réception**, dans `libs/platform/`, dans cet ordre : taille maximale, rejet des clés dupliquées (passe sur les jetons du décodeur, D04), validation par le schéma, puis décodage. Un message non conforme est écarté et signalé, jamais retenté (ES-38, M-CAN-03).
- **Anti-perte côté émetteur : outbox transactionnelle** dans la Place de marché. L'événement est écrit dans la même transaction que le changement d'état ; un relais dans le processus le publie (`FOR UPDATE SKIP LOCKED`, plusieurs réplicas sans coordination), attend la confirmation du broker, puis le marque publié. Les lignes publiées sont conservées pendant une durée bornée.
- **Filet de la file de contrôle** (préférence de D02) : tout article resté « en contrôle » au-delà d'un délai est renvoyé au contrôle ; la demande se reconstruit à partir de l'article.
- **Contrôle sans base** : il consomme la demande, publie le résultat, attend la confirmation du broker, puis acquitte la demande. Une panne entre les deux produit un doublon, jamais une perte.
- **Idempotence côté consommateur** : Place de marché par la garde d'état et de révision (D02), sans table de messages traités ; Interactions par une contrainte d'unicité sur l'identifiant de la décision (« une seule notification par décision »).
- **Topologie provisionnée par l'IaC** : files, échanges ou flux, consommateurs, identités et droits sont déclarés dans le chart, jamais par le code (M-CAN-06). L'abonnement de test est posé puis retiré par l'exploitation.
- **Règle d'argent** : un événement reçu peut suspendre un achat ou un reversement, jamais déclencher un mouvement de fonds (architecture § 6, règle 11). V1 : une « commande payée » reçue après l'échéance alors qu'un autre acheteur a réservé produit un événement de conflit ; Transactions suspend et un humain décide du remboursement tracé (ES-14).
- **État du broker sans volume persistant** : stockage sur `emptyDir`, qui survit au redémarrage du conteneur mais pas à la replanification du pod. Couverture : la file de contrôle par le filet ; la diffusion par le rejeu des lignes de l'outbox publiées depuis une date, commande réservée à l'exploitation, sans risque puisque les consommateurs sont idempotents.

### Option A — RabbitMQ en conteneur dans EKS

- **Description** :
  - image officielle `rabbitmq` 4.x épinglée, un pod (StatefulSet à une réplique), Service interne, politique réseau ;
  - topologie, utilisateurs, droits et politiques dans un fichier de définitions chargé au démarrage, versionné avec le chart ;
  - un vhost `collector` ; un utilisateur par service ; droit `configure` vide pour tous les services, donc aucune déclaration de file ou d'échange possible ; `write` limité à l'échange du producteur, `read` à ses files ; droits par clé de routage sur l'échange de diffusion ;
  - files quorum (accepté sur un seul nœud) : limite de livraisons, échange de mise à l'écart vers une file `.ecartes` par file, durée et longueur maximales par politique (ES-26, M-CAN-03) ;
  - client Go `amqp091-go` (maintenu par l'équipe RabbitMQ), confirmations de publication, `Qos` borné.
- **Topologie du POC** :

| Échange | Écrit par | Clé | File | Lue par |
|---|---|---|---|---|
| `controle.demandes` (direct) | Place de marché | `v1` | `controle.demandes` | Contrôle |
| `controle.resultats` (direct) | Contrôle | `v1` | `pdm.resultats-controle` | Place de marché |
| `pdm.evenements` (topic) | Place de marché | `publication.decidee.v1` | `interactions.publication-decidee` | Interactions |
| idem | idem | idem | `test.publication-decidee` (posée puis retirée) | identité de test, lecture seule |

  Mode de retour du résultat : une file durable dédiée, lue par toutes les instances de la Place de marché en consommateurs concurrents ; pas de file de réponse temporaire (elle exigerait le droit `configure`).
- **Avantages** :
  - Le modèle de droits (`configure`, `write`, `read`, droits par clé de routage) se superpose presque mot à mot à ES-38 : « aucun service ne crée d'abonnement » est un droit vide, vérifiable par un test négatif.
  - Mise à l'écart native (limite de livraisons et échange de mise à l'écart) : ES-31 sans code.
  - Point à point et diffusion sont deux usages du même modèle échange-file.
  - Topologie entière dans un fichier lisible : l'IaC est la seule source (M-CAN-06), et le schéma de soutenance en découle.
  - Scaler KEDA `rabbitmq` mûr, sur la longueur de file.
  - Console de gestion utile en démonstration (files qui montent puis se vident), accessible par redirection de port seulement.
- **Inconvénients** :
  - `amqp091-go` ne se reconnecte pas seul : reconnexion, reprise des consommateurs et confirmations à écrire une fois dans `libs/platform/` (risque n°1 de l'avis back de D04).
  - Empreinte plus lourde (Erlang) : de l'ordre de 0,25 vCPU et 512 Mio demandés, 1 Gio de limite, seuil mémoire du broker réglé sous la limite du conteneur.
  - Pour KEDA, une identité de plus (tag `monitoring`, sans droit de lecture ni d'écriture) et l'API de gestion à ouvrir au seul pod KEDA.
  - Utilisateur `guest` à supprimer, cookie Erlang à générer au déploiement.
- **Coût AWS et effort** :
  - Aucun coût AWS propre : le pod tient sur le node group de D03 (17 pods par nœud).
  - Effort back, ordre de grandeur : client avec reconnexion 1 j ; relais de l'outbox 1 j ; validation stricte 0,5 j ; consommateurs idempotents compris dans les US. Fichier de définitions et chart : 0,5 à 1 j.
- **Complexité ajoutée** : un conteneur à état, un fichier de définitions. Chaque élément sert un besoin nommé (ENF-12, ES-38, ES-31, KEDA).
- **Couverture grille** :
  - **C1** : correspond à l'exemple de la consigne ; tests de publication, de consommation et de droits refusés reproductibles en local puis sur EKS.
  - **C2** : les deux modes et les identités par service exercés dans le POC.
  - **C8** : droits par sujet démontrables par des tests négatifs, malgré le rôle d'exécution unique.
  - **C5** : second signal de mise à l'échelle de D02 natif.

### Option B — NATS JetStream en conteneur dans EKS

- **Description** :
  - image officielle `nats` épinglée, JetStream activé, un pod, stockage fichier sur `emptyDir` ;
  - comptes et utilisateurs déclarés dans la configuration du serveur ; droits de publication et d'abonnement par sujet ;
  - un flux par domaine (`CONTROLE`, `PDM`), un consommateur durable par abonné, créés par une tâche de l'IaC ;
  - client Go officiel `nats.go` et son paquet `jetstream`.
- **Avantages** :
  - Très léger : de l'ordre de 0,1 vCPU et 128 Mio ; démarrage en secondes.
  - Client Go officiel avec reconnexion automatique.
  - Fenêtre de déduplication à l'émission (en-tête `Nats-Msg-Id`), en complément de l'idempotence.
  - Scaler KEDA `nats-jetstream` disponible.
- **Inconvénients** :
  - Interdire la création de consommateurs passe par des refus sur les sujets de l'API JetStream (`$JS.API.CONSUMER.CREATE.>` et voisins), tout en autorisant ceux de la lecture (`$JS.API.CONSUMER.MSG.NEXT.<flux>.<consommateur>`) : faisable, plus délicat à rendre correct et à démontrer (D04 l'a relevé, « à prouver en J3 »).
  - Pas de file de mise à l'écart native : au-delà du nombre maximal de livraisons, JetStream émet un avis ; copier le message à l'écart et le rendre rejouable est à coder (ES-31).
  - Le scaler KEDA lit le port de supervision HTTP du serveur, sans authentification par défaut : politique réseau à poser pour le réserver à KEDA.
  - Topologie répartie entre la configuration du serveur et une tâche de création des flux et consommateurs : deux sources à tenir alignées.
- **Coût AWS et effort** :
  - Aucun coût AWS propre ; empreinte la plus faible.
  - Effort back comparable à A : la reconnexion est fournie, mais la mise à l'écart est à écrire (≈ 0,5 à 1 j) ; droits de l'API JetStream à mettre au point en expérimentation (≈ 0,5 à 1 j, risque principal).
- **Complexité ajoutée** : un conteneur à état, une configuration de serveur, une tâche de création des flux, un code de mise à l'écart.
- **Couverture grille** : C1, C2, C5 comme A. C8 comme A si les refus sur l'API JetStream sont prouvés ; à défaut, droits par sujet partiellement démontrés.

### Options écartées d'emblée

| Option | Motif |
|---|---|
| Amazon MQ (RabbitMQ ou ActiveMQ managé), MSK, MSK Serverless | Indisponibles dans le Learner Lab [REX] |
| SQS et SNS, EventBridge | Disponibles et les plus simples à exploiter, mais autorisés par les seuls rôles IAM : sous `LabRole` unique, une seule identité pour tous les services ; ES-38 tombe en « documentée » et l'avis sécurité de D02 conditionne son accord à un canal qui authentifie lui-même chaque service. Ne sert pas l'exemple « broker déployé dans un orchestrateur » de C1. En V1, sous un rôle par service, SQS et SNS redeviendraient candidats |
| Kafka (mode KRaft) en conteneur | Journal partitionné surdimensionné pour quelques messages par seconde ; JVM d'au moins 1 Gio ; pas de mise à l'écart native ; parallélisme du Contrôle plafonné par le nombre de partitions, ce qui borne KEDA ; ACL par utilisateur exigeant SASL à configurer |
| Redpanda en conteneur | Compatible Kafka, plus léger que Kafka mais recommandé autour de 1 à 2 Gio par cœur ; mêmes limites de modèle que Kafka |
| Table PostgreSQL servant de file | Le plus simple techniquement, mais une file partagée casse le cloisonnement des données par service (ES-38, M-MOD-01) et ne fournit pas de diffusion ; ne sert pas C1 |

### Synthèse

| Critère | A — RabbitMQ | B — NATS JetStream |
|---|---|---|
| Identité par service, droits par sujet (ES-38) | Natifs ; création de file interdite par un droit vide | Natifs sur les sujets ; création de consommateur interdite par refus fins sur l'API JetStream, à prouver |
| Point à point et diffusion | Échange direct et échange topic | Flux et consommateurs durables |
| Mise à l'écart, rejeu borné (ES-31, ES-26) | Native (limite de livraisons, échange de mise à l'écart, politiques) | À coder à partir des avis |
| Topologie par l'IaC (M-CAN-06) | Un fichier de définitions | Configuration du serveur et tâche de création |
| Scaler KEDA | `rabbitmq`, identité de supervision | `nats-jetstream`, port de supervision à cloisonner |
| Client Go (D04) | `amqp091-go`, reconnexion à écrire | `nats.go`, reconnexion fournie |
| Empreinte sur le node group | ≈ 0,25 vCPU, 512 Mio à 1 Gio | ≈ 0,1 vCPU, 128 Mio |
| Coût AWS propre | Nul | Nul |
| Lisibilité en soutenance (C10) | Échanges, files, clés : vocabulaire courant | Sujets, flux, consommateurs : clair, moins répandu |

## Recommandation

**Option A — RabbitMQ en conteneur dans EKS, avec le socle commun : HTTP/JSON pour l'unique appel synchrone, outbox et filet de relance côté émetteur, idempotence côté consommateur, sans volume persistant.**
**Statut visé après avis : `Candidate (XPnn)`, confirmé par l'expérimentation du canal (D14), mutualisée avec celle de D03.**

1. **C'est l'option où ES-38 est la plus simple à rendre correcte et à prouver.** « Aucun service ne crée d'abonnement » se réduit à un droit `configure` vide ; « seul le producteur publie » à un droit `write` sur un seul échange. Avec B, le même résultat dépend de refus fins sur l'API JetStream, plus faciles à mal régler.
2. **Elle couvre ES-31 sans code** : limite de livraisons, mise à l'écart, durée et longueur bornées sont des réglages. Avec B, la mise à l'écart est du code à écrire et à tester.
3. **Toute la topologie tient dans un fichier**, source unique pour l'IaC, la revue de sécurité et le schéma.
4. **Son surcoût est connu et borné** : environ 0,5 Gio de plus que B sur un node group qui en a 8, et une reconnexion à écrire une fois dans `libs/platform/`.

**Pourquoi pas B.** B est plus léger et son client Go est plus complet ; c'est le repli si l'expérimentation montre un défaut de A (reconnexion instable, empreinte qui force un troisième nœud). Il perd sur les deux points qui portent la sécurité et la fiabilité du POC : l'interdiction de créer des consommateurs, et la mise à l'écart.

**Pourquoi pas SQS et SNS, plus simples.** Ils retirent un conteneur à état, mais rendent ES-38 indémontrable dans le POC sous le rôle unique. C'est le choix naturel de la V1 une fois les rôles par service disponibles ; la frontière `libs/platform/` rend ce changement local.

**Volontairement exclu**, au nom de la simplicité : volume persistant et pilote EBS CSI (le filet et le rejeu de l'outbox couvrent la replanification du pod) ; cluster RabbitMQ à plusieurs nœuds et opérateur RabbitMQ (une instance suffit au POC, point unique de défaillance assumé par D03) ; TLS sur AMQP dans le cluster (confinement réseau, ES-01), sauf avis sécurité contraire ; gRPC ; maillage de services ; registre de schémas (les fichiers de `contracts/` suffisent).

### Ce que l'expérimentation du canal doit prouver

Bac à sable jetable, local (minikube ou k3d) puis sur EKS, sur le même cluster de test que D03. Seuils proposés, à figer dans le protocole avant l'exécution.

| # | Critère | Seuil de succès | Si échec |
|---|---|---|---|
| XC1 | Publication et consommation point à point et en diffusion, avec confirmations | 100 % des messages reçus par chaque abonné, deux abonnés en diffusion | Bloquant |
| XC2 | Droits : un service publie sur l'échange d'un autre, lit la file d'un autre, déclare une file ou un échange | Refus dans 100 % des cas | Bascule vers B ; à défaut, ES-38 « documentée » |
| XC3 | Abonné arrêté pendant une vague, puis relancé | Aucun message perdu (ENF-12) | Bloquant |
| XC4 | Message malformé, clé JSON dupliquée | Écarté sans bloquer les suivants ; message mis à l'écart après la limite, rejouable | À corriger avant J4 |
| XC5 | Redémarrage du conteneur, puis suppression du pod du broker | Producteurs et consommateurs reconnectés sans intervention ; après suppression, filet et rejeu de l'outbox rétablissent tous les événements | Bascule vers B si la reconnexion est instable ; pilote EBS CSI à réexaminer si le rejeu ne suffit pas |
| XC6 | KEDA sur la file de contrôle (reprend X8 de D03) | De 1 à au moins 2 consommateurs ≤ 60 s après le seuil ; retour à 1 après vidage | Contrôle sur la CPU ou à nombre fixe |
| XC7 | Politique réseau : un pod non autorisé joint le port AMQP ou l'API de gestion | Refus constaté (reprend X12) | Avis sécurité |
| XC8 | Empreinte au repos et sous vague | Dans les demandes et limites fixées ; pas de nœud supplémentaire au repos | Revue du dimensionnement ou bascule vers B |

### Points soumis aux avis

- **Sécurité** :
  - TLS sur AMQP et sur l'appel synchrone dans le cluster, ou confinement réseau seul (ES-01) ; lien avec l'authentification du service appelé (M-SYN-01, D06).
  - API de gestion : redirection de port seulement, identité de supervision de KEDA sans droit de lecture ni d'écriture : suffisant ?
  - Livraison des identifiants du canal : un Secret par service, créé au déploiement (D03) ; sous le rôle unique, un service compromis peut lire les Secrets des autres (écart connu).
  - Commande de rejeu de l'outbox : qui peut la lancer, et sa trace (ES-23).
  - Durées de conservation des files, des messages mis à l'écart et des lignes publiées de l'outbox (ES-26).
- **Faisabilité** :
  - Outbox pour toutes les émissions de la Place de marché, ou filet seul pour la demande de contrôle et outbox pour la diffusion : un mécanisme pour tous est-il plus simple qu'un mécanisme par flux ?
  - `amqp091-go` avec reconnexion écrite dans `libs/platform/`, ou client AMQP 1.0 récent de RabbitMQ 4 [?] : maturité à vérifier.
  - Files quorum sur un seul nœud, ou files classiques avec limite de livraisons comptée par le consommateur.
  - Délai du filet de relance et période de rejeu, au regard de la démonstration d'ENF-12 au sprint 3.

## Avis sécurité

Fonction : Sécurité — 2026-10-26. Références : [exigences-securite.md](../securite/exigences-securite.md) (FC11, ES-01, ES-06, ES-14, ES-23, ES-26, ES-31, ES-38, M-CAN-01 à 06, M-MOD-02 et 03, M-SYN-01, M-RES-01), mesures MS1 à MS10 de l'avis sécurité de [D03](D03-hebergement-orchestration.md), [D06](D06-authentification-autorisation.md). Avis sur dossier, avant toute sonde : les points [?] sont à lever par l'expérimentation.

### 1. Position : A, B en repli
- **A est préférable pour la sécurité.** « Aucun service ne crée d'abonnement » tient à un droit `configure` vide ; lier une file à un échange exige `write` sur la file et `read` sur l'échange, qu'aucun service n'a : M-CAN-06 est fermée par construction et vérifiable par un test négatif (XC2). Sous B, la même garantie repose sur une liste de refus sur `$JS.API.>` : une omission (création de consommateur, purge ou suppression de flux) rouvre M-CAN-06 sans que rien ne le signale.
- **La mise à l'écart native d'A** (limite de livraisons, échange de mise à l'écart) supprime une source de défaut sur ES-31 et M-CAN-03 ; sous B, le code de copie à l'écart serait lui-même une surface à tester.
- **Bascule vers B acceptée** sur les critères de D05, à condition de rejouer XC2 avec la liste exhaustive des sujets `$JS.API` refusés (création, mise à jour, suppression, purge de flux et de consommateurs).
- **Limite à connaître** : les droits par clé de routage d'A ne s'appliquent qu'aux échanges `topic`. Pour les échanges `direct`, le cloisonnement vient du droit `write` sur un seul échange par producteur, ce qui suffit ici (un producteur par échange).

### 2. TLS interne ou confinement réseau
- **POC : confinement réseau seul, accepté** (ES-01 l'autorise), sans TLS sur AMQP ni sur « réserver », aux conditions suivantes, toutes déjà démontrables : politiques réseau en refus par défaut en entrée et en sortie (MS4, X12, XC7), Pod Security « restricted » interdisant `hostNetwork` et les privilèges (MS5), API du cluster limitée au poste (MS1).
- **Risque résiduel accepté** : identifiants AMQP (mécanisme PLAIN), jeton de service et jeton de l'acheteur circulent en clair entre nœuds, dans le VPC. Les lire suppose déjà la compromission d'un nœud, qui donne de toute façon accès aux Secrets montés.
- **M-SYN-01, faux service répondant à Transactions** : sans TLS, l'appelant n'authentifie pas l'appelé. Compensation : la politique de sortie de Transactions ne vise que le namespace et les labels de la Place de marché sur le port interne ; usurper la réponse exige de créer un pod ou un Service dans ce namespace, donc l'API du cluster (MS1). Écart documenté ; TLS interne (certificats émis par le cluster) en V1.

### 3. API de gestion et identité de KEDA
- **Identité de supervision, tag `monitoring`, sans droit sur le vhost : suffisante**, avec deux réserves. Ce tag montre toutes les connexions, files et métriques de tous les vhosts (noms d'utilisateurs, adresses des pods), pas le contenu des messages : fuite d'inventaire acceptée. Il ne doit porter aucun autre tag.
- **Port de gestion (15672)** ouvert par politique réseau au seul pod de l'opérateur KEDA ; la console de démonstration passe par `kubectl port-forward` depuis le poste, avec l'identité d'administration, jamais par une règle de l'ALB ni un NodePort.
- **Identités des services sans aucun tag** : elles ne peuvent pas se connecter à l'API de gestion, qui permettrait de lire ou purger des files hors protocole AMQP. À tester (XC2).
- **Plan d'administration non mentionné par D05**, à fermer : ports de distribution Erlang (4369, 25672) et de métriques (15692) fermés à tout pod, sauf la collecte de D09 pour 15692. Le cookie Erlang et l'identité d'administration donnent le contrôle complet du broker (topologie, lecture de toutes les files) : générés au déploiement, jamais dans un Secret de service.

### 4. Secrets par service
- **Un Secret par service, dans son namespace, monté par son seul pod**, avec `automountServiceAccountToken: false` (MS5) : un service compromis n'a pas de jeton pour lire les Secrets des autres par l'API du cluster.
- **Identifiants du canal éphémères** : l'état du broker ne survivant pas à la session, les mots de passe sont générés à chaque `up` sur le poste et injectés par l'entrée standard (MS6), sans être stockés dans Secrets Manager. La Place de marché, seule à garder `LabRole` (MS3), ne peut donc pas y lire les identifiants du Contrôle.
- **Fichier de définitions** : la topologie et les droits sont versionnés avec le chart, mais les utilisateurs et leurs empreintes de mot de passe sont injectés au déploiement dans un Secret, jamais dans le dépôt, une ConfigMap ou une valeur Helm.
- **Écart résiduel [?]** : les nœuds ont `LabRole` pour rôle IAM. Un pod qui obtient ce rôle par l'IMDS pourrait s'authentifier auprès de l'API du cluster comme un nœud et lire les Secrets des pods placés sur ce nœud. Si c'est le cas, le cloisonnement des Secrets par service ne vaut que si MS3 réserve effectivement l'IMDS. À vérifier pendant X16.

### 5. Rejeu de l'outbox
- **Lancé uniquement par l'exploitation, depuis le poste** : sous-commande du binaire de la Place de marché exécutée en Job Kubernetes, donc soumise à l'API du cluster (MS1). Aucune route HTTP, publique ou interne, ne la déclenche.
- **Bornée** : fenêtre « depuis une date » plafonnée à la durée de conservation des lignes publiées ; republie les lignes telles quelles, avec leur identifiant d'origine ; ne crée, ne modifie ni ne supprime aucune ligne, hors la date de publication.
- **Tracée (ES-23)** : la création du Job dans le journal du plan de contrôle (MS8), et une entrée d'audit applicative avec la fenêtre, le nombre d'événements republiés et l'identifiant de corrélation.

### 6. Durées de conservation (ES-26, M-CAN-04)
- **Files de travail** : longueur maximale avec dépassement `reject-publish`, jamais `drop-head`. Le producteur reçoit un refus de confirmation, l'événement reste dans l'outbox ou la demande n'est pas acquittée : aucune perte silencieuse (ENF-12).
- **Durée de vie des messages** : 24 h, avec renvoi vers la file `.ecartes`, jamais une expiration sans trace.
- **Mise à l'écart** : limite de livraisons fixée explicitement à 5 (la valeur par défaut des files quorum est plus haute) ; files `.ecartes` conservées 7 jours au plus, avec une alerte à chaque entrée (ES-31).
- **Outbox** : lignes publiées purgées après 7 jours, ce qui fixe aussi la fenêtre maximale de rejeu ; lignes non publiées jamais purgées, avec une alerte au-delà de 15 min.
- **Contenu** : identifiants opaques seulement, ni texte saisi, ni adresse, ni coordonnées. En POC, la destruction de la pile à chaque session borne de fait toutes ces durées.

### 7. Appel « réserver » joignable hors du cluster ?
- **Non, si quatre conditions tiennent.**
  - Service `ClusterIP`, port interne absent de tout Service `NodePort` ou `LoadBalancer` (MS5 refuse les NodePorts hors liste).
  - Port non ouvert par le groupe de sécurité des nœuds, ni aux adresses des pods dans le VPC (MS2).
  - Aucune règle de l'ALB vers ce port (MS2).
  - Politique réseau d'entrée limitée au namespace de Transactions (MS4).
- **Condition propre à D05, à ajouter** : deux serveurs HTTP distincts dans la Place de marché, un par port, la route « réserver » n'étant enregistrée que sur le serveur interne. Sinon, un chemin oublié dans les règles de l'ALB suffit à l'exposer par le port public.
- **Jetons sur cette route** : même si la route était exposée, les contrôles de D06 refusent un jeton d'utilisateur (audience, portée, `azp`).
- **Validation stricte sur l'appel synchrone aussi** : taille maximale, clés dupliquées, schéma OpenAPI avant décodage, sur la requête reçue par la Place de marché comme sur la réponse lue par Transactions. Le schéma, sensible à la casse et avec `additionalProperties: false`, neutralise aussi la correspondance insensible à la casse des noms de champs par `encoding/json`.

### 8. Mesures minimales si la recommandation est retenue
| # | Mesure | Exigences, menaces | Dans le POC |
|---|---|---|---|
| MC1 | Droits du canal : `configure` vide pour tous les services ; `write` sur le seul échange du producteur ; `read` sur les seules files du consommateur ; aucun tag ; utilisateur `guest` supprimé | ES-38, M-CAN-01, 05, 06 | Démontrée (XC2) |
| MC2 | Émetteur authentifié dans le message : propriété `user-id` posée par chaque producteur, que RabbitMQ vérifie contre l'identité connectée ; le consommateur écarte tout message dont `user-id` n'est pas le producteur attendu du sujet | ES-38, M-CAN-01, M-CAN-05 | Démontrée (XC2) |
| MC3 | Politiques réseau du broker : 5672 depuis les namespaces des quatre services ; 15672 depuis l'opérateur KEDA seul ; 15692 depuis la collecte de D09 ; 4369 et 25672 fermés ; aucun Service `NodePort` ni règle de l'ALB vers le broker | ES-01, ES-07, M-CAN-07 | Démontrée (XC7) |
| MC4 | Identité KEDA au seul tag `monitoring`, sans droit sur le vhost ; identité d'administration et cookie Erlang générés au déploiement, hors de tout Secret de service ; console par `port-forward` seulement | ES-06, ES-38, M-CAN-07 | Démontrée (XC2, XC7) |
| MC5 | Identifiants du canal générés à chaque `up`, injectés par l'entrée standard, un Secret par service ; empreintes du fichier de définitions hors du dépôt | ES-06, M-MOD-03 | Démontrée (X20) |
| MC6 | Validation stricte (taille, clés dupliquées, schéma, puis décodage) dans `libs/platform/`, appliquée aux messages et aux deux sens de « réserver » ; message non conforme écarté sans nouvel essai | ES-38, M-CAN-03 | Démontrée (XC4) |
| MC7 | « Réserver » servi par un serveur HTTP distinct, sur le port interne seul ; politique d'entrée limitée à Transactions, politique de sortie de Transactions limitée à ce port | ES-38, M-SYN-01 | Démontrée (X17, X12, tests négatifs de D06) |
| MC8 | Bornes des files : longueur maximale en `reject-publish`, durée de vie 24 h avec mise à l'écart, limite de livraisons à 5, `.ecartes` 7 jours avec alerte ; outbox publiée purgée à 7 jours, alerte sur les lignes non publiées | ES-26, ES-31, M-CAN-03, M-CAN-04, M-CAN-08 | Démontrée (XC3, XC4) |
| MC9 | Rejeu de l'outbox par Job lancé depuis le poste, fenêtre plafonnée, sans écriture hors date de publication, tracé | ES-23, M-CAN-02 | Démontrée (XC5) |
| MC10 | Règle d'argent vérifiable : aucun consommateur d'événement n'importe le client du prestataire de paiement (test d'architecture sur les paquets) ; un événement ne fait que passer à « suspendu » | ES-14, M-CAN-05, M-RES-01 | Documentée (POC : Transactions ne consomme rien) ; démontrée en V1 |

Critère ajouté à l'expérimentation : **XC9** — avec l'identité d'un service, se connecter à l'API de gestion, lier une file, purger ou supprimer une file d'un autre service ; joindre 4369 et 25672 depuis un pod applicatif ; publier un message avec un `user-id` usurpé. Refus dans 100 % des cas ; si échec, bloquant.

### 9. Menaces à ajouter au registre
- **M-CAN-07** (S, T, I, E ; FC11) : prise de contrôle du plan d'administration du broker. Voies : cookie Erlang ou identité d'administration exposés, port de gestion ou de distribution joignable depuis un pod, empreintes de mot de passe versionnées. Effets : création de liaisons, lecture et purge de toutes les files, faux résultats de contrôle. Proposition : 2 × 4 = 8, Élevé ; mesures MC3 à MC5.
- **M-CAN-08** (D ; FC11) : perte silencieuse par les bornes elles-mêmes, quand une longueur maximale en `drop-head` ou une durée de vie sans mise à l'écart suppriment des événements sans alerte (contredit ENF-12). Proposition : 2 × 2 = 4, Modéré ; mesure MC8.
- **M-MOD-03** à compléter [?] : obtention de l'identité de nœud par l'IMDS, puis lecture des Secrets des pods co-localisés.

**Verdict** : avis favorable à l'option A et au socle commun, avec confinement réseau sans TLS interne pour le POC, à condition que MC1 à MC9 soient dans le chart dès le sprint 1 et que XC2, XC7 et XC9 réussissent. B reste acceptable en repli si XC2 est rejoué sur la liste complète de l'API JetStream.

## Avis faisabilité

Fonction : Faisabilité — 2026-10-26. Références : avis back de [D04](D04-stack.md) (risques 1 et 3), avis sécurité ci-dessus (MC1 à MC10, XC9). Avis sur dossier : les points [?] sont à lever par l'expérimentation du canal.

**Position : A faisable dans l'enveloppe de D04, avec le socle commun et l'avis sécurité.** B reste un repli crédible ; il ne ferait pas gagner de temps, la mise à l'écart à coder compensant la reconnexion fournie.

### Développement

- **Une seule outbox pour toutes les émissions de la Place de marché.** Une table (identifiant, échange, clé de routage, en-têtes, corps, dates de création et de publication), un relais, une commande de rejeu : un chemin de code et un seul jeu de tests. Publier la demande de contrôle hors outbox ouvrirait une fenêtre de perte entre la validation de la transaction et la publication, que seul le filet couvrirait, avec son délai. Le filet reste, pour la replanification du pod du broker, pas comme mécanisme principal.
- **Client : `amqp091-go`, avec la reconnexion écrite dans `libs/platform/`.**
  - Un superviseur de connexion (`NotifyClose`, attente exponentielle avec gigue), un canal par usage, reprise des consommateurs après reconnexion, confirmations par `PublishWithDeferredConfirmWithContext`, prise en compte de `NotifyBlocked` : le relais se met en pause quand le broker bloque les producteurs (alarme mémoire ou disque).
  - Le code ne déclare rien (droit `configure` vide) : une file absente fait échouer `Consume`, et le pod reste non prêt.
  - Distinguer la fermeture d'un canal pour faute (code 406, par exemple un `user-id` refusé) d'une coupure réseau : la première est un défaut à signaler, pas à réessayer en boucle.
  - Client AMQP 1.0 de RabbitMQ 4 (`rabbitmq-amqp-go-client`) : reconnexion fournie, mais version 0.x, peu de retours, et la vérification du `user-id` comme les droits seraient à reprouver [?]. Écarté pour le POC ; réexaminable en V1, derrière la même frontière.
- **Idempotence consommateur, dans les US** :
  - Place de marché : mise à jour conditionnée par l'état « en contrôle » et la révision ; zéro ligne touchée signifie doublon, acquitté sans effet.
  - Interactions : insertion avec `ON CONFLICT (decision_id) DO NOTHING`, dans la même transaction que la notification.
  - Acquittement manuel après l'effet, `Qos` de l'ordre de 10.
- **Validation stricte (MC6)** : taille vérifiée avant toute lecture du corps, et `max_message_size` côté broker en défense ; passe sur `json.Decoder.Token()` avec un ensemble de clés par objet ouvert (≈ 60 lignes, sans `encoding/json/v2`, encore expérimental) ; schéma compilé au démarrage avec `santhosh-tekuri/jsonschema` v6 ; décodage seulement ensuite.
  - Piège : `go:embed` ne lit pas hors du module ; `contracts/` porterait un module minimal qui n'embarque que les fichiers de schéma, sans type Go, ce qui reste conforme à D04.
  - « Réserver » : même validateur derrière `http.MaxBytesReader`, schémas pris dans les composants de l'OpenAPI 3.1, dans les deux sens.
  - Message non conforme : `Nack` sans remise en file, donc mise à l'écart immédiate, et journal de sécurité.
- **MC2, `user-id`** : chaque producteur pose `UserId` égal à son identité ; chaque consommateur compare `Delivery.UserId` au producteur attendu pour sa file (table de configuration) et écarte sinon. Conséquence : un message rejoué depuis une file `.ecartes` par l'identité d'administration porterait un autre `user-id` et serait refusé. Le rejeu se fait donc par le producteur : rejeu de l'outbox (MC9) ou filet pour la demande de contrôle ; les files `.ecartes` servent au diagnostic. À confirmer en XC4 [?].
- **MC7** : deux `http.Server` dans le binaire de la Place de marché, la route « réserver » enregistrée sur le seul routeur interne. **MC9** : sous-commande `replay --since` exécutée en Job, avec la fenêtre plafonnée à 7 jours. **MC10** : à vérifier par un test d'architecture (rôle du testeur) ; rien à coder dans le POC.

### Exploitation

- **Files quorum sur un seul nœud : oui.** Raft à un membre fonctionne, pour un surcoût faible. Elles seules portent la limite de livraisons sur RabbitMQ 4 ; avec des files classiques, il faudrait compter les essais dans le consommateur et republier, donc ajouter un droit `write` et du code.
  - Mise à l'écart en `at-least-once` sur les files de travail, compatible avec `reject-publish` (MC8), pour ne pas perdre un message pendant le transfert vers `.ecartes`.
- **Topologie par le fichier de définitions** : topologie et droits dans une ConfigMap, utilisateurs et empreintes dans un Secret. Soit `definitions.local.path` pointe sur un répertoire qui contient les deux fichiers [?], soit un conteneur d'initialisation les fusionne dans un `emptyDir`. Absence de `guest` à vérifier en XC9.
- **Identifiants à chaque `up`** : un script du poste génère les mots de passe et calcule les empreintes SHA-256 salées attendues par RabbitMQ (petit outil Go dans `tools/`), puis crée un Secret par service, le Secret du broker et celui de KEDA. Le cookie Erlang est généré de la même façon.
- **Ressources du broker** : seuil mémoire `vm_memory_high_watermark.absolute` à environ 600 Mio sous la limite de 1 Gio ; `emptyDir` avec `sizeLimit`, et `disk_free_limit` réglé en conséquence, sans quoi l'alarme disque bloque tous les producteurs.
- **Scaler KEDA `rabbitmq`** : mode `http` obligatoire, car le mode `amqp` fait une déclaration passive qu'interdit une identité sans droit sur le vhost (MC4). Longueur cible d'environ 5 messages, interrogation toutes les 5 à 10 s, retour au minimum après 60 s : compatible avec XC6.
- **Délais** :
  - Filet : article « en contrôle » depuis plus de 2 min, balayage toutes les 30 s ; le contrôle répond en quelques secondes.
  - Rejeu : fenêtre par défaut « depuis le début de la session », plafonnée à 7 jours.
  - Démonstration d'ENF-12 au sprint 3 : Interactions arrêté, les messages s'accumulent dans la file durable, sans rejeu ; le rejeu n'est montré que pour la suppression du pod du broker.
  - Point dur : une remise en file immédiate épuise les 5 livraisons en quelques millisecondes. Une erreur transitoire (base indisponible) ne doit donc pas faire `Nack` : le consommateur se met en pause avec attente croissante. Sinon, des messages valides sont mis à l'écart.
- **Local, docker compose** : image `rabbitmq:4.x-management` à la même version épinglée, mêmes `rabbitmq.conf` et fichier de définitions, identifiants générés par le même script dans un `.env` ignoré par le dépôt. Sans politiques réseau : XC7 et la partie réseau de XC9 se prouvent sur k3d ou EKS seulement.

**Effort, hors expérimentation** : environ 6 à 7 jours.
- Développement, environ 4,5 j : client, reconnexion, confirmations et cadre consommateur 1,5 j ; outbox, relais, purge et rejeu 1,5 j ; validation stricte et module de contrats 1 j ; `user-id` et double serveur 0,5 j.
- Exploitation, environ 2,25 j : définitions, configuration, chart et politiques 1 j ; génération des identifiants 0,5 j ; KEDA 0,5 j ; docker compose 0,25 j.
- L'écart avec les 3 à 3,5 j de l'option A tient à MC2, MC9, KEDA et la génération des identifiants, absents de cette estimation. Le code de l'expérimentation est réutilisé, pas jeté.

**Risques, par gravité** :
1. Erreurs transitoires qui épuisent la limite de livraisons et écartent des messages valides.
2. Reconnexion d'`amqp091-go`, et confusion entre faute de canal et coupure réseau (XC5).
3. Rejeu depuis `.ecartes` incompatible avec MC2 : le rejeu doit passer par le producteur.
4. Séparation définitions/Secret et absence de `guest`, à prouver (XC9).
5. Alarme disque sur `emptyDir`, qui bloque les producteurs.

**Verdict** : avis favorable à l'option A, avec une outbox unique, `amqp091-go` et des files quorum, sous réserve que XC4, XC5 et XC9 couvrent les risques 1 à 4 avant J4.

## Décision de Martin

> **« RabbitMQ »** (option A) : broker en conteneur dans EKS, avec les mesures de sécurité MC1 à MC10 (confinement réseau sans TLS interne dans le POC, écart documenté pour la V1). Statut **Candidate** jusqu'à l'expérimentation du canal (critères XC1 à XC9), mutualisée avec celle de D03 (D14). Repli : option B (NATS JetStream) si les critères de bascule échouent.
>
> Socle commun retenu : outbox unique côté Place de marché, idempotence côté consommateur, validation JSON Schema avec rejet des clés dupliquées, topologie posée par l'IaC, règle « un événement suspend, ne déplace jamais d'argent ». Appel « réserver » en HTTP/JSON sur un port interne non routé par l'ALB, jeton de service selon D06.

Date : 2026-10-26

> **Complément du 2026-10-26 (validation d'US-012), texte saisi dans les messages** : le message `controle.demande` v1 porte le titre et la description de l'annonce, avec une longueur bornée (200 et 5 000 caractères). Sans eux, le Contrôle ne peut pas détecter les coordonnées personnelles exigées par D01 et US-018. C'est un écart assumé à l'avis sécurité (« ni texte saisi »). Il est compensé par trois règles : aucune donnée de contact du vendeur dans le message, aucun refus qui recopie une valeur reçue, et des messages jamais journalisés en clair.

## Conséquences

La recommandation est retenue ; ses conséquences :

- **Statut et expérimentation** : D05 passe en `Candidate (XPnn)` ; l'expérimentation du canal (XC1 à XC8) est mutualisée avec celle de l'orchestrateur et précède J4 (condition de WS-2).
- **D06** : jeton de service de Transactions et revérification du jeton de l'acheteur sur « réserver » ; authentification du service appelé ; en-têtes de transport des deux jetons. À trancher avant le sprint 3.
- **D07** : table d'outbox dans l'espace de la Place de marché ; contrainte d'unicité sur l'identifiant de la décision dans l'espace d'Interactions ; aucun espace pour le Contrôle.
- **D09** : identifiant de corrélation et contexte de trace W3C dans les en-têtes des messages ; alertes sur la longueur des files et sur toute mise à l'écart (ES-31) ; console de gestion hors du point d'entrée.
- **D11** : le chart porte le broker, son fichier de définitions et ses politiques ; identifiants générés au déploiement, utilisateur `guest` supprimé ; aucun volume persistant ; ressources du broker comptées dans le dimensionnement du node group.
- **D13** : un test de contrat par paire producteur-consommateur sur les schémas de `contracts/` ; tests négatifs des droits du canal (XC2) ; test de perte d'un abonné (ENF-12) ; test de clé dupliquée et de message malformé ; tests négatifs de l'appel synchrone (ES-38).
- **D14** : protocole de l'expérimentation du canal, avec les critères XC1 à XC8.
- **Code** : `libs/platform/` gagne la connexion au canal avec reconnexion, le relais d'outbox et la validation stricte ; `contracts/messages/` les schémas v1 (demande et résultat de contrôle, décision de publication) ; `contracts/api/` l'OpenAPI interne de « réserver ».
- **Architecture** : § 9 d'[architecture-technique.md](../architecture/architecture-technique.md) complété (canal) ; § 10 : points « mode de diffusion, identités, anti-perte, retour du résultat » et « confirmation tardive » fermés ou reportés en V1 ; réception des alertes de l'outil de fraude par l'adaptateur, en V1, en interrogeant l'outil plutôt qu'en ouvrant un point d'entrée public.
- **Sécurité** : [exigences-securite.md](../securite/exigences-securite.md) : ES-38 passe « intégrée » pour les droits par sujet sur le canal, sous réserve de XC2 ; ES-31 « intégrée » pour la mise à l'écart.
- **Compétences (C9)** : RabbitMQ, outbox transactionnelle et idempotence entrent dans la cartographie et l'action de formation « échanges asynchrones fiables ».
- **Index** : D05 à mettre à jour dans [INDEX.md](INDEX.md).

## À savoir défendre en soutenance

1. **« Pourquoi un broker en conteneur plutôt que SQS ou Amazon MQ ? »**
   - Amazon MQ et MSK sont indisponibles dans l'environnement de formation.
   - SQS et SNS sont autorisés par les rôles IAM ; avec un rôle unique imposé, tous les services y auraient la même identité. Un broker qui authentifie lui-même chaque service rend démontrables les droits par sujet (ES-38).
   - En V1, avec un rôle par service, SQS et SNS redeviendraient candidats ; le changement est confiné à `libs/platform/`.
2. **« Comment garantissez-vous qu'aucun événement n'est perdu ? »**
   - Côté émetteur : outbox écrite dans la même transaction que l'état, publiée avec confirmation du broker.
   - Côté consommateur : acquittement après l'effet, idempotence par la révision de l'article ou par l'unicité de la décision.
   - Si le pod du broker est replanifié : relance des articles restés « en contrôle » et rejeu de l'outbox. Démonstration : Interactions arrêté puis relancé, aucune notification perdue.
3. **« Pourquoi RabbitMQ plutôt que NATS, plus léger ? »**
   - Les deux ont été comparés sur des critères fixés avant l'expérimentation.
   - RabbitMQ interdit la création de files par un simple droit vide et met les messages en échec à l'écart sans code ; NATS demande des refus fins sur son API et du code de mise à l'écart.
   - Le surcoût, environ 0,5 Gio de mémoire, tient sur les nœuds existants. NATS reste le repli.
4. **« Un faux événement peut-il déplacer de l'argent ? »**
   - Non : seul le producteur d'un sujet peut y publier, chaque message est validé contre son schéma, et un événement ne peut que suspendre un achat ou un reversement, jamais déclencher un mouvement de fonds.
5. **« Pourquoi un appel synchrone, et un seul ? »**
   - La réservation d'un objet unique demande une réponse immédiate et atomique : l'acheteur doit savoir tout de suite s'il a l'article.
   - Tout autre échange passe par le canal, pour ne pas coupler la disponibilité des services. L'appel est authentifié par un jeton de service limité à « réserver », absent du point d'entrée public, et réessayable sans double réservation.

## Sources

Documentation officielle, à revérifier au moment de l'expérimentation.

- **Learner Lab** : [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md) (Amazon MQ et MSK refusés, absence d'IRSA, EBS CSI) et ses sources.
- **RabbitMQ** :
  - Contrôle d'accès (droits `configure`, `write`, `read`, droits par clé de routage) : https://www.rabbitmq.com/docs/access-control
  - Fichier de définitions : https://www.rabbitmq.com/docs/definitions
  - Files quorum, limite de livraisons : https://www.rabbitmq.com/docs/quorum-queues
  - Mise à l'écart : https://www.rabbitmq.com/docs/dlx ; durée et longueur des files : https://www.rabbitmq.com/docs/ttl et https://www.rabbitmq.com/docs/maxlength
  - Confirmations de publication et acquittements : https://www.rabbitmq.com/docs/confirms
  - Client Go : https://github.com/rabbitmq/amqp091-go
- **NATS** :
  - Autorisations : https://docs.nats.io/running-a-nats-service/configuration/securing_nats/authorization
  - API JetStream : https://docs.nats.io/reference/reference-protocols/nats_api_reference
  - Client Go : https://github.com/nats-io/nats.go
- **KEDA** : scaler RabbitMQ : https://keda.sh/docs/latest/scalers/rabbitmq-queue/ ; scaler NATS JetStream : https://keda.sh/docs/latest/scalers/nats-jetstream/
- **Kubernetes** : volumes `emptyDir` : https://kubernetes.io/docs/concepts/storage/volumes/#emptydir
- **Motif outbox** : https://microservices.io/patterns/data/transactional-outbox.html
