# D09 — Observabilité

| Champ | Valeur |
|---|---|
| Statut | Décidée |
| Date | 2026-10-26 |
| Décideur | Martin |
| Dépend de | [D03](D03-hebergement-orchestration.md), [D04](D04-stack.md), [D05](D05-communication-interservice.md), [D06](D06-authentification-autorisation.md) |
| Impacte | D10, D11 (couche persistante, allumage), D12 (indicateurs), D13 (mesures de charge), D14 (expérimentation), `libs/platform/`, [exigences-securite.md](../securite/exigences-securite.md) (ES-23, ES-31), `docs/exploitation.md` |
| Critères grille | C5, C6, C8, C1 |

## Contexte et besoin

D09 fixe comment le POC produit, collecte, conserve et montre ses journaux, ses métriques et ses traces, sur l'hébergement de D03 (EKS, node group géré de t3.medium, 2 à 4 nœuds, statut Candidate ; repli ECS Fargate).
D09 ne choisit pas les indicateurs qualité (D12) : elle fournit les sources de mesure dont D12 aura besoin.

### Exigences

| Source | Exigence (extrait) | Ce que l'observabilité doit permettre |
|---|---|---|
| Consigne, M13 | « Au moins une composante d'observabilité : logs centralisés, collecte de métriques ou traces distribuées » | Plancher. Le POC vise plus d'une composante |
| ENF-25 | « Un incident provoqué est diagnostiqué à partir de cette composante. » | Retrouver un incident de bout en bout, à travers le canal |
| C5, ENF-14, ENF-15, ENF-16 | Montée en charge démontrée ; p95 < 500 ms ; moins de 1 % d'erreurs ; retour au minimum | Tableaux de bord lisibles en direct pendant le tir de soutenance |
| C6 | « Au moins 4 indicateurs », dont la mesure fait ressortir des axes d'amélioration | Sources de mesure fiables, exportables dans `docs/mesures/` |
| ES-23 | Journal d'audit centralisé, horodaté, avec auteur ; « émis hors du processus de chaque service, vers un stockage en ajout seul » ; ni mot de passe, ni jeton, ni donnée de carte, ni contenu de message ; identifiant de corrélation propagé ; durée de conservation définie | Journaux d'audit collectés hors des services et conservés hors du cluster |
| ES-31, D05 | Alerte à chaque entrée dans une file `.ecartes`, alerte sur la longueur des files et sur les lignes d'outbox non publiées | Règles d'alerte sur les métriques du broker et des services |
| M-LIV-07, D07 MD7 | Aucune URL présignée complète, aucun secret, aucune chaîne de connexion dans un journal | Masquage à la source |
| D04 (Conséquences) | SDK OpenTelemetry Go (traces, métriques), W3C Trace Context dans HTTP et dans les messages, `slog` JSON avec l'identifiant de trace | Point d'arrivée des signaux déjà décidés |

### Contraintes

- **Learner Lab** ([environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md)) :
  - Amazon Managed Prometheus et Managed Grafana indisponibles (non listés, `aps` et `grafana` absents de la trust policy) ; X-Ray incertain [?] ;
  - CloudWatch listé (journaux, métriques, alarmes), mais sur EKS l'agent passe par le rôle du nœud `LabRole`, `CloudWatchAgentServerPolicy` ne pouvant être attachée [?] ; un REX signale `GetDashboard` refusé [?] ;
  - pas d'IRSA : toute écriture vers AWS depuis un pod passe par le rôle du nœud (IMDS), comme pour S3 (D03, X6) ;
  - journaux CloudWatch à rétention infinie par défaut.
- **Cycle de vie** : le cluster est détruit à chaque fin de session (D03, D11). Tout ce qui est stocké dans le cluster est perdu à chaque `down`.
- **Mémoire et densité** (avis faisabilité de D06) : environ 6,6 Gio allouables sur deux t3.medium ; système, KEDA, Cluster Autoscaler et Flux ≈ 1 Gio ; Keycloak 1 à 1,25 Gio ; RabbitMQ 512 Mio demandés, 1 Gio de limite ; 17 pods par nœud, environ 25 pods au repos et 31 au pic pour 34 places [estimation].
- **D03 a exclu** « une pile de supervision complète dans le cluster », au nom de la simplicité : tout ajout doit nommer son besoin.

### Ce qu'il faudrait montrer en soutenance

| Tableau | Signaux | Source |
|---|---|---|
| Expérience utilisateur | Débit, latence p50/p95 par route, taux d'erreurs 5xx, sur la Place de marché | Métriques HTTP des services (OpenTelemetry) |
| Mise à l'échelle | Réplicas courants et désirés des HPA (Place de marché ; Contrôle, dont le HPA est créé par KEDA), nombre de nœuds, pods en attente | kube-state-metrics |
| Saturation | CPU et mémoire par pod et par nœud, redémarrages, OOMKill (Keycloak, RabbitMQ) | kubelet (cAdvisor), kube-state-metrics |
| Canal | Messages prêts et non acquittés par file, débit de publication et de consommation, files `.ecartes`, alarme mémoire du broker | Plugin Prometheus de RabbitMQ (port 15692), métriques par file à activer [?] |
| Diagnostic | Recherche par identifiant de corrélation dans les journaux, de la requête HTTP au résultat du contrôle | Journaux `slog` JSON centralisés ; traces si un stockage est retenu |

## Options

Les trois options partagent l'instrumentation des services (voir « Instrumentation commune ») ; elles diffèrent par la collecte, le stockage et l'affichage.

### Option A — CloudWatch (Container Insights, Fluent Bit, tableaux de bord et alarmes CloudWatch)

- **Description** : module complémentaire EKS `amazon-cloudwatch-observability` (agent CloudWatch et Fluent Bit en DaemonSet) avec le rôle du nœud. Journaux vers CloudWatch Logs, métriques Container Insights, métriques applicatives et du broker récupérées par la collecte Prometheus de l'agent CloudWatch, ou par OTLP vers l'agent [?]. Tableaux de bord et alarmes décrits en Terraform.
- **Avantages** :
  - Données hors du cluster : elles survivent au `down`.
  - Peu de mémoire dans le cluster : un agent et un Fluent Bit par nœud, environ 150 à 250 Mio au total [estimation].
  - Service managé, cohérent avec C5 ; c'est aussi la solution native du repli ECS Fargate (`awslogs`, Container Insights, [REX]).
  - Journal d'audit hors du processus et hors du cluster (ES-23).
- **Inconvénients** :
  - Dépend de deux incertitudes du lab : permissions du rôle du nœud [?] et `GetDashboard` refusé [?]. Si la console n'affiche pas les tableaux, la démonstration tombe.
  - Granularité d'une minute en standard : la montée des pods (moins d'une minute, D03) se lit mal en direct.
  - Profondeur des files et métriques applicatives : configuration de collecte Prometheus propre à l'agent CloudWatch, peu documentée en REX ; aucune expérimentation locale possible (CR6 de D03).
  - Compétence propre à AWS, non réutilisable en local (C9).
- **Coût AWS et effort** : ingestion des journaux 0,50 $/Go [à vérifier] ; métriques Container Insights facturées à l'heure [?], quelques centimes par session de 4 h [estimation] ; alarmes et tableaux de bord : quelques dollars par mois au prorata [à vérifier]. Effort 2 à 3 j [estimation].
- **Complexité ajoutée** : un module complémentaire, une configuration de collecte, des tableaux et alarmes en Terraform.
- **Couverture grille** : C5 (supervision managée), C6 (mesures exportables), C8 (ES-23). C1 : expérimentation uniquement dans AWS.

### Option B — Pile auto-hébergée dans EKS (Prometheus, Grafana, Loki, Tempo en option)

- **Description** : chart `kube-prometheus-stack` (opérateur Prometheus, Prometheus, Grafana, Alertmanager, kube-state-metrics, node-exporter) ; Loki en binaire unique avec un agent de collecte par nœud ; Tempo en option pour les traces. Tableaux de bord en JSON dans le dépôt, provisionnés au déploiement. Accès à Grafana par `kubectl port-forward`, jamais par l'ALB.
- **Avantages** :
  - Démonstration la plus riche et la plus fluide : relevé toutes les 15 s, tableaux communautaires prêts pour Kubernetes, RabbitMQ et Go, alertes visibles dans la même interface.
  - Même pile en local (minikube ou k3d) et dans AWS : expérimentable avant toute session cloud (CR6 de D03).
  - Aucune dépendance aux permissions CloudWatch du lab.
  - Compétences portables, standard du marché (C9).
- **Inconvénients** :
  - Toutes les données disparaissent à chaque `down` (pas de pilote EBS CSI sans IRSA, D03).
  - Journal d'audit stocké dans le cluster qu'il doit surveiller : ES-23 (hors du processus, ajout seul, conservation définie) n'est pas tenu.
  - Mémoire et pods : environ 1,5 à 2,5 Gio et 10 à 12 pods de plus avec Loki et Tempo [estimation]. Sur 34 places, la densité force un troisième nœud au repos (D03, Conséquences).
  - Six composants de plus à épingler, analyser et tenir à jour (M-LIV-01).
- **Coût AWS et effort** : 0 $ de service AWS ; un troisième t3.medium ≈ 0,04 $/h, 0,17 $ par session, environ 5 $ sur le projet. Effort 3 à 4 j [estimation].
- **Complexité ajoutée** : la plus forte : six composants, un stockage par composant, des politiques réseau, un accès d'administration.
- **Couverture grille** : C5 et C6 riches ; C8 en recul (ES-23) ; C1 favorable (local fidèle).

### Option C — Mixte : Prometheus et Grafana pour les métriques, CloudWatch Logs pour les journaux

- **Description** :
  - *Métriques et alertes* : `kube-prometheus-stack` allégé : Prometheus, Grafana, kube-state-metrics, opérateur ; node-exporter et Alertmanager désactivés (alertes par les règles Prometheus, affichées dans Grafana). Rétention Prometheus de 12 h sur `emptyDir` borné.
  - *Journaux* : Fluent Bit en DaemonSet vers CloudWatch Logs, avec le rôle du nœud [?] ; groupes de journaux créés par l'IaC dans la couche persistante (D11), rétention fixée ; requêtes par identifiant de corrélation dans CloudWatch Logs Insights. En local, Fluent Bit écrit sur la sortie standard ou dans un fichier.
  - *Traces* : instrumentées et propagées dans tous les cas (D04). Stockage par Tempo en binaire unique : sous-question ci-dessous.
- **Avantages** :
  - Chaque signal va là où il sert : les métriques, éphémères par nature, restent dans le cluster pour la démonstration en direct ; les journaux, qui portent l'audit, sortent du cluster et survivent au `down` (ES-23).
  - Pas de Loki : environ 0,5 Gio et 3 pods économisés par rapport à B.
  - Une seule incertitude AWS (Fluent Bit par le rôle du nœud), de même nature que X6 (S3) déjà testé en D14 ; les tableaux de bord ne dépendent pas de `GetDashboard`.
  - Repli simple : si D03 bascule sur ECS Fargate, les journaux passent par `awslogs` sans changement de format.
- **Inconvénients** :
  - Deux interfaces en démonstration : Grafana pour les métriques, CloudWatch Logs Insights pour les journaux (Grafana peut interroger CloudWatch Logs par le rôle du nœud [?], ce qui ramènerait tout dans Grafana).
  - Métriques perdues à chaque `down` : à exporter si elles servent de preuve (voir « Perte des données »).
  - Environ 1 Gio et 6 pods de plus (Prometheus 0,5 à 0,8 Gio, Grafana 0,15 à 0,25 Gio, Fluent Bit 2 × 50 Mio) [estimation] ; le troisième nœud au repos reste probable au pic, à mesurer en D14.
- **Coût AWS et effort** : ingestion des journaux 0,50 $/Go [à vérifier], soit quelques centimes par session pour quelques centaines de Mo ; troisième nœud éventuel 0,17 $ par session. Effort 2,5 à 3,5 j, plus 0,5 j pour Tempo [estimation].
- **Complexité ajoutée** : quatre composants dans le cluster, un DaemonSet, des groupes de journaux en IaC.
- **Couverture grille** : C5 (supervision en direct, alertes), C6 (sources de mesure), C8 (ES-23 tenu hors du cluster), C1 (métriques expérimentables en local).

### Instrumentation commune des services Go

- **OpenTelemetry ou client Prometheus** : D04 a retenu le SDK OpenTelemetry Go pour les traces et les métriques ; D09 ne le rouvre pas. Le client `client_golang` serait plus simple pour les seules métriques, mais imposerait une seconde bibliothèque pour les traces. Export retenu en conséquence : métriques par l'exportateur Prometheus d'OpenTelemetry (point `/metrics` relevé par Prometheus, sans collecteur) ; traces en OTLP, directement vers Tempo s'il est retenu, sinon non exportées.
- **Port de métriques distinct** du port de l'API, jamais routé par l'ALB, ouvert au seul Prometheus par politique réseau (comme le port 15692 du broker, MC3 de D05, et le port de gestion de Keycloak, D06).
- **Journaux** : `slog` JSON sur la sortie standard, attributs fixes : `service`, `trace_id`, `span_id`, `correlation_id`, `event`, `level` ; attribut `audit=true` pour les événements d'ES-23, aiguillés par Fluent Bit vers un groupe de journaux d'audit distinct.
- **Masquage à la source** dans `libs/platform/` : gestionnaire `slog` qui retire en-têtes `Authorization` et second jeton de « réserver » (MA6 de D06), chaînes de requête des URL présignées (MD7 de D07), secrets et chaînes de connexion ; aucun email ni contenu de message. Testé par un test unitaire qui journalise des valeurs piégées.
- **Corrélation** : l'identifiant de corrélation de l'enveloppe D05 est fixé à l'entrée (requête HTTP du front), recopié dans chaque message et chaque journal ; le contexte W3C circule dans les en-têtes HTTP et AMQP (porteur à écrire, ≈ 0,5 j, D04). Ainsi, même sans stockage de traces, un incident se suit de bout en bout dans les journaux (ENF-25).
- **Étiquettes de métriques** : route et code de réponse, jamais un identifiant d'utilisateur ou d'article (PII, cardinalité).
- **Keycloak** : métriques sur le port de gestion ; événements de connexion réussie et échouée et événements d'administration écrits dans les journaux (ES-23), collectés comme ceux des services.
- **Front** : non instrumenté dans le POC (D04).

### Alertes proposées

| Alerte | Seuil indicatif | Source |
|---|---|---|
| Entrée dans une file `.ecartes` | > 0 sur 1 min | ES-31, D05 MC8 |
| File du Contrôle en retard | messages prêts > seuil de KEDA pendant 5 min | D05 |
| Outbox non publiée | lignes plus anciennes que 1 min | D05 MC8 |
| Taux d'erreurs 5xx | > 1 % sur 5 min | ENF-15 |
| Latence p95 Place de marché | > 500 ms sur 5 min | ENF-14 |
| HPA au maximum | réplicas courants = maximum pendant 5 min | D03 |
| Redémarrages, OOMKill | > 0 | Keycloak, RabbitMQ (D06, D05) |
| Alarme mémoire ou disque du broker | active | D05 |

Dans le POC, les alertes sont visibles dans Grafana (ou dans CloudWatch pour A) ; aucune notification externe. Une alerte est déclenchée volontairement en démonstration (message rendu non conforme, donc écarté).

### Perte des données à chaque extinction

- **Métriques** : leur perte est acceptable. Le POC n'a pas besoin d'historique entre sessions ; ce qui sert de preuve est exporté à la fin de chaque tir par un script (requêtes de plage Prometheus en CSV, captures des tableaux, rapport de l'outil de charge) dans `docs/mesures/charge/` (D13). Les tableaux de bord sont du code dans le dépôt : ils reviennent à l'identique à chaque `up`.
- **Journaux** : leur perte n'est pas acceptable pour l'audit (ES-23). Ils sont hors du cluster dans A et C ; dans B, il faudrait un export vers S3 par le rôle du nœud [?], ce qui revient à C avec un composant de plus.
- **Traces** : perte acceptable ; une trace exemplaire est capturée pour le dossier.

### Rétention

- Prometheus : 12 h dans le cluster (une session), `emptyDir` avec limite de taille.
- CloudWatch Logs, journaux applicatifs : 7 jours ; groupe d'audit : 30 jours dans le POC (données fictives). Cible V1 : 6 mois à 1 an pour la sécurité (repère CNIL, ES-23).
- Journaux du plan de contrôle : rétention courte déjà prévue (D03, MS8).
- Aucun groupe sans rétention : l'inventaire de D11 le vérifie.

## Synthèse

| Critère | A — CloudWatch | B — Auto-hébergée | C — Mixte |
|---|---|---|---|
| Démonstration en direct (C5) | Moyenne : 1 min, console incertaine [?] | Riche, 15 s | Riche, 15 s ; journaux dans une seconde interface |
| Profondeur des files, HPA, KEDA | À configurer dans l'agent [?] | Natif | Natif |
| ES-23 (hors processus, hors cluster, conservation) | Tenu | Non tenu | Tenu |
| Données après `down` | Conservées | Perdues | Journaux conservés, métriques exportées |
| Mémoire et pods ajoutés | ≈ 0,2 Gio, 4 pods | ≈ 1,5 à 2,5 Gio, 10 à 12 pods | ≈ 1 Gio, 6 pods (+ Tempo ≈ 0,3 Gio, 1 pod) |
| Incertitudes du lab | Deux (rôle du nœud, `GetDashboard`) | Aucune | Une (Fluent Bit par le rôle du nœud) |
| Expérimentation locale (C1) | Non | Oui | Oui pour les métriques |
| Coût par session | Quelques centimes [?] | 0,17 $ (3e nœud) | Quelques centimes, + 0,17 $ si 3e nœud |
| Effort | 2 à 3 j | 3 à 4 j | 2,5 à 3,5 j (+ 0,5 j Tempo) |
| Repli ECS Fargate | Natif | À reconstruire | Journaux natifs, métriques à reconstruire |

## Recommandation

**Option C — Prometheus et Grafana dans le cluster pour les métriques et les alertes, Fluent Bit vers CloudWatch Logs pour les journaux et l'audit ; traces instrumentées, Tempo laissé à la mesure de mémoire.**

1. **Elle sert la démonstration de C5 sans pari sur le lab** : la montée des pods, KEDA, les nœuds et la profondeur des files se lisent toutes les 15 s, dans des tableaux expérimentés d'abord en local. A fait dépendre l'écran de soutenance de `GetDashboard` [?].
2. **Elle tient ES-23** là où B échoue : l'audit sort du cluster, survit au `down` et a une conservation fixée.
3. **Elle reste la plus simple qui couvre les deux besoins** : pas de Loki (CloudWatch Logs le remplace), pas de node-exporter ni d'Alertmanager, pas de collecteur OpenTelemetry. Chaque composant restant nomme son besoin : Prometheus (métriques, alertes ES-31), Grafana (démonstration), kube-state-metrics (HPA, KEDA, nœuds), Fluent Bit (journaux hors du processus).
4. **Tempo** (≈ 0,3 Gio, 0,5 j) n'est ajouté que si la mesure de densité en D14 laisse la place : la corrélation par journaux suffit à ENF-25 ; une trace à travers le canal enrichit la démonstration de C2 sans être exigée.

**Pourquoi pas A.** C'est la plus légère et la plus managée, et elle redevient le choix naturel si D03 bascule sur ECS Fargate. Sur EKS, deux incertitudes du lab touchent directement l'écran de soutenance, et rien ne s'expérimente en local.

**Pourquoi pas B.** Loki ajoute de la mémoire pour stocker dans le cluster des journaux qui doivent justement en sortir.

**Hors périmètre, au nom de la simplicité** : mise à l'échelle de la Place de marché sur le nombre de requêtes (possible par le scaler Prometheus de KEDA déjà installé, mais la CPU suffit à D03) ; instrumentation du front ; notifications d'alertes externes ; collecteur OpenTelemetry.

### Points soumis aux avis

- **Sécurité** :
  - Fluent Bit écrit dans CloudWatch par le rôle du nœud : il fait partie des pods autorisés à joindre l'IMDS (MS3 de D03) ; `LabRole` pouvant supprimer un groupe de journaux, « ajout seul » reste documenté, pas démontré [?] ;
  - accès à Grafana par `port-forward` seulement, mot de passe administrateur généré à chaque `up` ; ports de métriques ouverts au seul Prometheus ;
  - masquage à la source dans `libs/platform/` (jetons, URL présignées, secrets, emails, contenu des messages) et test associé ;
  - événements Keycloak dans le journal d'audit : contenu, absence de mot de passe ou de jeton ;
  - rétention de 30 jours pour l'audit dans le POC, 6 mois à 1 an en V1.
- **Faisabilité** :
  - mémoire et pods ajoutés : troisième nœud au repos ou non, avec Keycloak et RabbitMQ (mesure en D14) ;
  - métriques par file de RabbitMQ (point par objet) et leur volume ;
  - lecture de CloudWatch Logs depuis Grafana par le rôle du nœud [?], pour une seule interface en démonstration ;
  - effort de 2,5 à 3,5 j, plus l'instrumentation des services (≈ 1,5 j avec le porteur W3C) ;
  - Tempo : oui, non, ou selon la mesure.

## Avis sécurité

Fonction : Sécurité — 2026-10-26. Références : [exigences-securite.md](../securite/exigences-securite.md) (ES-06, ES-07, ES-23, ES-31, ES-38, §5.6 ; M-LIV-02, 03, 06, 07, M-MOD-03, M-DIS-01) ; avis sécurité de [D03](D03-hebergement-orchestration.md) (MS1, MS3, MS5, MS6, MS8, X16), [D05](D05-communication-interservice.md) (MC3), [D06](D06-authentification-autorisation.md) (MA6), [D07](D07-donnees.md) (MD7). Avis sur dossier ; les points [?] relèvent des sondes de D14.

**Synthèse.** Avis favorable à l'option C, sous les mesures MO1 à MO10. C est la seule option qui sort l'audit du cluster ; B est à écarter pour motif de sécurité (ES-23 non tenu). Deux points sont à corriger dans la recommandation : l'aiguillage de l'audit sur un attribut que le service émetteur déclare lui-même, et le masquage limité au gestionnaire `slog`, qui ne voit pas tout ce qui finit dans les journaux.

**1. Points soumis**

- **Fluent Bit et l'IMDS** : favorable, à condition de traiter Fluent Bit comme le composant le plus sensible de la pile. Il cumule `LabRole` (tous les secrets, S3, journaux) et la lecture en `hostPath` des journaux de tous les conteneurs du nœud. Conséquences :
  - il entre dans la liste d'autorisation de MS3, et X16 est étendu : Fluent Bit obtient un jeton IMDSv2, Prometheus, Grafana et kube-state-metrics n'en obtiennent pas ;
  - si le repli de MS3 s'applique (limite de sauts à 1), Fluent Bit passe en `hostNetwork` : il hérite alors du réseau du nœud et échappe aux politiques réseau ; à écrire comme écart ;
  - namespace dédié, seule exception documentée à « restricted » (MS5) : `hostPath` en lecture seule limité à `/var/log`, utilisateur non root si l'image le permet [?], aucune capacité ; sortie limitée au DNS, à l'IMDS et au 443 ;
  - avis défavorable à la lecture de CloudWatch Logs depuis Grafana par le rôle du nœud (point de faisabilité) : cela donnerait `LabRole` à une application web à greffons. Deux interfaces en démonstration sont un moindre mal.
- **« Ajout seul » non démontrable sous `LabRole`** : constat confirmé, déjà noté dans ES-23 et MS8. Tout pod autorisé à joindre l'IMDS peut appeler `DeleteLogGroup`, `DeleteLogStream` ou `PutRetentionPolicy`. Cela s'ajoute à un second trou, plus grave car indépendant du lab : un service compromis écrit ce qu'il veut sur sa sortie standard. S'il peut émettre `audit=true` au nom d'un autre service, il fabrique de faux événements d'audit. L'origine d'un événement doit donc venir des métadonnées Kubernetes ajoutées par Fluent Bit (namespace, pod), jamais du champ `service` du message (MO2). Pour le POC : écart assumé, borné par MS3 (moins de pods détenteurs de `LabRole`) et vérifiable par l'inventaire de D11 (groupe présent, rétention inchangée). Cible V1 : une identité pour Fluent Bit limitée à `logs:CreateLogStream` et `logs:PutLogEvents` sur ses groupes, refus explicite de `logs:Delete*` pour les autres identités, export vers un stockage verrouillé (S3 Object Lock).
- **Grafana en `port-forward` seulement** : favorable ; l'accès passe par l'API du cluster, déjà limitée au /32 du poste (MS1). Le chart fixe par défaut un mot de passe administrateur connu (`prom-operator`) : il doit être remplacé par un Secret créé à chaque `up` depuis l'entrée standard (MS6), jamais en valeur Helm. Compléments : connexion anonyme et inscription désactivées, Service `ClusterIP` (la politique de MS5 refuse déjà `LoadBalancer` et les NodePorts hors liste), aucune règle de l'ALB, installation de greffons interdite, sortie réseau limitée à Prometheus. Prometheus n'a aucune authentification : même traitement, API d'administration désactivée (valeur par défaut, à vérifier dans les valeurs du chart).
- **Masquage et son test** : le masquage à la source est le bon principe, mais un gestionnaire `slog` ne voit que les attributs et le message. Il ne voit pas :
  - le texte des erreurs : une `*url.Error` de `net/http` contient l'URL complète, donc la signature d'une URL présignée ; une erreur de connexion peut citer la chaîne de connexion ;
  - les `panic` et traces de pile, écrites directement sur la sortie d'erreur ;
  - les journaux de Keycloak, de RabbitMQ et de Fluent Bit, qui ne passent pas par `libs/platform/`.

  D'où deux couches : masquage dans le gestionnaire, y compris dans les valeurs `error` et les groupes imbriqués ; puis un filtre Fluent Bit de défense en profondeur (`Bearer `, `X-Amz-Signature`, `X-Amz-Credential`, `password=`, motif d'email). Le test unitaire piège un attribut, un message, une erreur `*url.Error` et un groupe imbriqué. D14 ajoute un test de bout en bout : les valeurs piégées sont injectées par l'API, puis recherchées dans Logs Insights, sur tous les groupes. L'auteur d'un événement d'audit est l'identifiant de sujet Keycloak, jamais l'email.
- **Événements Keycloak** : favorable, avec trois réglages vérifiables dans le realm versionné :
  - événements d'administration sans représentation (`adminEventsDetailsEnabled` à faux) : la représentation peut contenir un identifiant ou un mot de passe temporaire à la création d'un compte ;
  - l'écouteur `jboss-logging` journalise les succès en niveau debug par défaut [à vérifier] : sans réglage du niveau des succès, les connexions réussies exigées par ES-23 n'apparaissent pas ;
  - journaux en JSON, niveau INFO imposé (le niveau debug peut tracer des jetons) ; un `LOGIN_ERROR` porte le nom saisi, qui peut être un email : la donnée est acceptable dans le groupe d'audit, à rétention courte, et non dans le groupe applicatif.

  Les événements stockés en base par Keycloak sont désactivés ou reçoivent une expiration, pour ne pas créer un second journal non maîtrisé.
- **Rétention** : 30 jours pour l'audit du POC, sur données fictives : favorable. Pour la V1, retenir 1 an plutôt que 6 mois pour les événements liés aux contenus : la LCEN impose à l'hébergeur de conserver un an les données d'identification des auteurs de contenus (durée « à confirmer » dans ES-23). `PutRetentionPolicy` étant permis à `LabRole`, la rétention est contrôlée par l'inventaire, pas seulement posée par l'IaC. Chiffrement par une clé KMS gérée par le client en V1.
- **Exposition de `/metrics`** : favorable au port distinct ouvert au seul Prometheus. Deux compléments :
  - l'étiquette `route` est le **modèle** de route (`/articles/{id}`), jamais le chemin brut ; une route inconnue tombe dans une valeur unique. Sinon, des requêtes vers des chemins aléatoires créent des séries sans limite, Prometheus tombe en mémoire et la supervision disparaît pendant l'attaque ;
  - Prometheus et kube-state-metrics ont besoin d'un jeton de compte de service, ce qui fait exception à `automountServiceAccountToken: false` (MS5) : `ClusterRole` en lecture seule, sans `secrets` en `get` ni `list` hors des seules métadonnées (kube-state-metrics n'expose que noms et étiquettes).

**2. Mesures minimales, si la recommandation est retenue**

| Id | Mesure | Exigences, menaces | Preuve |
|---|---|---|---|
| MO1 | Fluent Bit dans la liste IMDS de MS3, namespace dédié, `hostPath` `/var/log` en lecture seule, sortie limitée DNS, IMDS, 443 ; Prometheus, Grafana et kube-state-metrics sans IMDS | ES-38, MS3, M-MOD-03 | X16 étendu : jeton IMDSv2 refusé depuis Grafana et Prometheus, obtenu par Fluent Bit |
| MO2 | Aiguillage vers le groupe d'audit et origine de l'événement fixés par les métadonnées Kubernetes ajoutées par Fluent Bit, pas par les champs du message | ES-23, M-LIV-06 | Test : un pod de la Place de marché émet `audit=true, service=transactions` ; l'événement est rangé sous la Place de marché |
| MO3 | Masquage en deux couches : gestionnaire `slog` (attributs, message, erreurs, groupes) et filtre Fluent Bit ; auteur = identifiant de sujet | ES-23, MA6, MD7, M-LIV-07 | Test unitaire à valeurs piégées ; recherche de bout en bout dans Logs Insights (D14) |
| MO4 | Grafana : mot de passe généré à chaque `up` hors valeurs Helm, anonyme et inscription désactivés, greffons interdits, `ClusterIP` ; Prometheus : `ClusterIP`, API d'administration désactivée ; ni l'un ni l'autre derrière l'ALB | ES-06, ES-07, MS5, MS6, M-LIV-03 | Inventaire des points d'entrée (X17) ; connexion `admin/prom-operator` refusée |
| MO5 | Ports de métriques ouverts au seul namespace de Prometheus ; étiquette `route` = modèle de route, valeur unique pour les routes inconnues | ES-07, ES-29, M-DIS-01 | Politique réseau testée (X12) ; 10 000 chemins aléatoires ne créent pas de nouvelle série |
| MO6 | Keycloak : événements de connexion et d'administration activés, succès en INFO, représentation désactivée, journaux JSON, niveau INFO, événements en base expirés | ES-23, ES-10, M-BO-01 | Connexion réussie, échouée et changement de rôle visibles dans le groupe d'audit, sans mot de passe ni jeton |
| MO7 | Groupes de journaux en IaC dans la couche persistante : audit 30 j (POC), applicatif 7 j, aucun groupe sans rétention ; inventaire de D11 qui vérifie existence et rétention | ES-23, MS8 | Sortie de l'inventaire à chaque `down` |
| MO8 | Débit de Fluent Bit borné par conteneur (`Mem_Buf_Limit`) et alerte de coût couvrant l'ingestion | ES-29, MS10 | Tir de journaux massif : nœud stable, ingestion bornée |
| MO9 | Comptes de service de Prometheus et kube-state-metrics en lecture seule, sans lecture du contenu des Secrets | ES-38, MS5 | `kubectl auth can-i get secrets --as=…` refusé |
| MO10 | Écarts écrits : « ajout seul » non garanti sous `LabRole`, Fluent Bit en `hostNetwork` si repli de MS3 ; cible V1 (identité dédiée, refus de `logs:Delete*`, Object Lock, clé KMS, audit 1 an) | ES-23, §5.6, M-MOD-03 | Registre des écarts à jour |

**3. Menaces à ajouter au registre** (à reporter dans exigences-securite.md, §4.10, par Martin)

| Id | STRIDE | Frontière | Menace | Impact | P | I | Risque | Couverture |
|---|---|---|---|---|---|---|---|---|
| M-OBS-01 | S, R | FC9, FC11 | Un service compromis écrit sur sa sortie standard de faux événements d'audit attribués à un autre service ou à un autre utilisateur. | Audit trompeur, enquête faussée, répudiation. | 2 | 3 | 6 Modéré | MO2 (ES-23) |
| M-OBS-02 | T, R | FC9 | Un pod détenteur de `LabRole` supprime un groupe de journaux d'audit ou réduit sa rétention. | Effacement des traces après compromission. | 2 | 3 | 6 Modéré | MO1, MO7, MO10 ; V1 : identité dédiée, Object Lock |
| M-OBS-03 | I, E | FC9 | Compromission du collecteur de journaux (image vulnérable, configuration) : lecture des journaux de tout le nœud et identifiants de `LabRole`. | Fuite des journaux et de tous les secrets. | 1 | 4 | 4 Modéré | MO1, MS9 (image épinglée et analysée) |
| M-OBS-04 | D | FC1 | Explosion de cardinalité des métriques ou inondation de journaux provoquée de l'extérieur. | Supervision aveugle pendant l'attaque, coût d'ingestion. | 2 | 2 | 4 Modéré | MO5, MO8 |

M-LIV-07 est à compléter (erreurs, `panic`, composants tiers) plutôt que dupliquée ; l'exposition de Grafana ou de Prometheus relève de M-LIV-03.

**Verdict.** Favorable à l'option C, sous réserve de MO1 à MO10. Aucun point bloquant pour le POC ; MO2 (origine de l'audit) et MO3 (masquage en deux couches) sont les deux corrections de fond à reporter dans la recommandation. L'extension de X16 à Fluent Bit est la condition du « Favorable ».

## Avis faisabilité

**Synthèse.** Avis favorable à C, sans Tempo dans le cluster. La mémoire tient sur deux t3.medium ; la contrainte qui mord est le **nombre de places de pods**, pas la mémoire. L'effort annoncé (2,5 à 3,5 j + 1,5 j d'instrumentation) est juste si l'on compte les tableaux de bord et le script d'export : prévoir **5 à 5,5 j**.

### Exploitation

- **Mémoire cumulée au repos** (demandes, [estimation]) sur 6,6 Gio allouables :
  - système, metrics-server, KEDA, Cluster Autoscaler, Flux ≈ 1 Gio ; Keycloak 0,75 Gio (limite 1,25) ; RabbitMQ 0,5 Gio (limite 1) ; services Go ≈ 0,3 Gio ;
  - pile D09 : Prometheus 0,5 Gio (limite 1), Grafana 0,2, kube-state-metrics et opérateur 0,15, Fluent Bit 2 × 50 Mio ≈ 1 Gio ;
  - total ≈ 3,6 Gio demandés, ≈ 5 Gio si tout atteint sa limite : la mémoire passe, avec une marge d'environ 1,5 Gio.
- **Places de pods** : 26 au repos (D06) + 6 (Prometheus, Grafana, kube-state-metrics, opérateur, 2 Fluent Bit) = **32 sur 34**. Au pic, les réplicas ajoutés déclenchent un nœud de toute façon (c'est la démonstration de D03), mais la moindre tâche ponctuelle (job d'import, webhook d'admission du chart) peut rester en attente au repos.
  - Parade sans coût à essayer d'abord : délégation de préfixes du CNI VPC (`ENABLE_PREFIX_DELEGATION`), qui porte un t3.medium à 110 pods ; compatibilité avec le lab [?], à placer dans la sonde de D14.
  - **Troisième nœud au repos : non par défaut**, seulement si la délégation échoue et que la mesure montre des pods en attente ou un OOMKill. Coût : 0,0416 $/h, 0,17 $ par session, environ 4 $ sur 25 sessions ; 3 instances au repos, 4 au pic, sous la limite de 9.
- **Installation par Helm, ordre d'allumage** :
  - `kube-prometheus-stack` (valeurs allégées : node-exporter, Alertmanager désactivés, webhooks d'admission désactivés pour éviter les jobs) et Fluent Bit installés **juste après le node group, avant** RabbitMQ, Keycloak et les services : leurs `ServiceMonitor` et `PrometheusRule` exigent les CRD de l'opérateur. Avec Flux : `dependsOn` explicite ; CRD volumineuses en application côté serveur.
  - Groupes de journaux créés par l'IaC dans la couche persistante avant Fluent Bit (sinon Fluent Bit les crée sans rétention).
  - Versions des charts et images figées par empreinte ; durée ajoutée à l'`up` : 1 à 2 min, hors chemin critique de RDS.
- **Métriques par file RabbitMQ** : le point agrégé `/metrics` ne donne rien par file. Utiliser `/metrics/detailed?family=queue_coarse_metrics&family=queue_consumer_count` (deux `ServiceMonitor` ou un avec deux points) plutôt que `/metrics/per-object`, plus lourd. Avec une dizaine de files (dont les `.ecartes`), le volume est négligeable pour Prometheus. Le scaler KEDA reste branché sur l'API de gestion, indépendamment.
- **CloudWatch Logs depuis Grafana** [?] : la source de données CloudWatch de Grafana suit la chaîne d'identifiants AWS par défaut, donc le rôle du nœud par l'IMDS (limite de sauts à 2, D03). Il faut que `LabRole` autorise `logs:StartQuery` et `logs:GetQueryResults`, et que Grafana soit ajouté aux pods autorisés à joindre l'IMDS (à arbitrer par l'avis sécurité). Faisable en 0,25 j ; à tester dans la sonde. Échec : Logs Insights dans la console, deux onglets en soutenance, sans perte fonctionnelle.
- **Export après chaque tir** : script `make export-mesures` qui lit l'API de Prometheus (`query_range` sur la fenêtre du tir, pas de 15 s) et écrit un CSV par indicateur dans `docs/mesures/charge/<date>/`, avec le rapport de l'outil de charge. Captures des tableaux à la main (pas de moteur de rendu Grafana : une image et de la mémoire en plus). ≈ 0,5 j.
- **Local** :
  - k3d comme référence : même chart, fichier de valeurs local, Fluent Bit vers la sortie standard. C'est là que se mettent au point tableaux, règles et `ServiceMonitor`.
  - docker compose pour la boucle de développement : Prometheus et Grafana en conteneurs avec configuration de relevé statique, mêmes tableaux montés depuis le dépôt.

### Développement

- **OpenTelemetry et exportateur Prometheus en Go** (`libs/platform/`) :
  - `otelhttp` sur le routeur, exportateur `go.opentelemetry.io/otel/exporters/prometheus` sur un serveur HTTP séparé (port de métriques) ;
  - points d'attention : suffixes d'unité ajoutés aux noms, et **limites d'histogramme à fixer** (vue explicite en secondes, autour de 0,5 s) sinon le p95 de ENF-14 est imprécis ; étiquette de route = gabarit, jamais le chemin brut.
  - ≈ 0,5 j.
- **slog JSON avec masquage** : un gestionnaire enveloppant le gestionnaire JSON, qui ajoute `trace_id` et `span_id` depuis le contexte et filtre par `ReplaceAttr` les clés et motifs listés ; test avec valeurs piégées. ≈ 0,5 j.
- **Propagation HTTP/AMQP** : HTTP natif avec `otelhttp` ; AMQP par un porteur sur `amqp091.Table` (écriture à la publication par l'outbox, lecture à la consommation), identifiant de corrélation de l'enveloppe recopié dans le contexte. Test : un message publié puis consommé garde le même `trace_id`. ≈ 0,5 j.
- **Tableaux de bord en code** : JSON dans le dépôt, provisionnés par le sidecar de Grafana (ConfigMaps étiquetées). Partir des tableaux communautaires Kubernetes et RabbitMQ, écrire « Expérience utilisateur » et « Mise à l'échelle » ; règles d'alerte en `PrometheusRule`. ≈ 1 à 1,5 j.

**Tempo** : non dans le cluster. ENF-25 est tenu par les journaux corrélés ; Tempo prendrait une place de pod sur les deux restantes. Pour la trace exemplaire du dossier, un Jaeger « tout-en-un » dans le docker compose local suffit (OTLP déjà émis, ≈ 0,25 j, zéro coût AWS). À reconsidérer seulement si la délégation de préfixes passe.

**Effort**

| Lot | Jours |
|---|---|
| Charts, ordre d'allumage, Fluent Bit et groupes de journaux | 1 |
| RabbitMQ par file, source CloudWatch dans Grafana | 0,5 |
| Script d'export, local k3d et docker compose | 0,75 |
| Instrumentation (OTel, slog, porteur AMQP) | 1,5 |
| Tableaux et alertes en code | 1 à 1,5 |
| Jaeger local, trace exemplaire | 0,25 |
| **Total** | **5 à 5,5** |

**Risques**

- Places de pods saturées au repos (32/34) : délégation de préfixes, sinon troisième nœud.
- Fluent Bit ou Grafana refusés par `LabRole` sur CloudWatch [?] : à lever dans la sonde avant tout développement de tableau de journaux.
- CRD absentes au moment du déploiement des services : ordre `dependsOn` à tester dès le premier `up` complet.
- Explosion de cardinalité (chemin brut en étiquette) : Prometheus en OOM pendant le tir, tableaux vides en soutenance.
- Métriques perdues si l'export est oublié : l'export fait partie de la procédure de tir (D13), pas d'une étape facultative.

## Décision de Martin

> **« Mixte allégée »** (option C) : Prometheus et Grafana dans le cluster pour les métriques (Grafana en port-forward seulement), journaux et audit vers CloudWatch Logs par Fluent Bit, export des métriques après chaque tir dans `docs/mesures/charge/`. Sans Loki, Alertmanager, node-exporter ni Tempo (trace exemplaire avec Jaeger en local). Mesures de sécurité MO1 à MO10, dont l'aiguillage de l'audit par les métadonnées Kubernetes (MO2), le masquage en deux couches (MO3) et l'extension de X16 à Fluent Bit. Grafana ne lit pas CloudWatch Logs par `LabRole` : journaux montrés dans la console CloudWatch.

Date : 2026-10-26

## Conséquences

La décision de Martin est enregistrée ci-dessus ; ses conséquences :

- **D14 — Expérimentation** : critères à ajouter, par exemple : Fluent Bit écrit dans CloudWatch Logs par le rôle du nœud ; mémoire et pods de la pile mesurés au repos et au pic ; tableau « mise à l'échelle » lisible pendant un tir ; une entrée dans `.ecartes` déclenche l'alerte en moins de 2 min ; une valeur piégée (jeton, URL présignée) n'apparaît dans aucun journal. Le statut peut passer à `Candidate (XPnn)` si l'avis faisabilité le juge nécessaire.
- **D11 — IaC** : groupes de journaux dans la couche persistante, avec leur rétention ; Prometheus, Grafana et Fluent Bit installés par `up` ; inventaire de `down` vérifiant l'absence de groupe sans rétention.
- **D03** : densité à revoir (troisième nœud au repos possible, environ 5 $ sur le projet) ; politiques réseau et réservation de l'IMDS étendues à Prometheus et Fluent Bit.
- **D12** : latence p95, taux d'erreurs, délai de mise à l'échelle et messages écartés deviennent mesurables depuis Prometheus ; le choix des indicateurs reste à D12.
- **D13** : script d'export des métriques en fin de tir vers `docs/mesures/charge/`.
- **`libs/platform/`** : démarrage OpenTelemetry, exportateur Prometheus, gestionnaire `slog` avec masquage, porteur W3C pour AMQP.
- **Documents** : `docs/exploitation.md` (accès à Grafana, requêtes Logs Insights types, liste des alertes) ; ES-23 révisé dans [exigences-securite.md](../securite/exigences-securite.md).
- **Repli ECS Fargate** : journaux par `awslogs` au même format ; métriques par Container Insights et l'option A.

## À savoir défendre en soutenance

1. **« Pourquoi deux outils, Prometheus et CloudWatch, et pas un seul ? »**
   - Les métriques servent à voir la montée en charge en direct : Prometheus les relève toutes les 15 s et se teste en local.
   - Les journaux portent l'audit, qui doit sortir du cluster et survivre à sa destruction : CloudWatch Logs le fait sans composant à héberger.
   - Une pile unique dans le cluster aurait perdu l'audit à chaque extinction ; une pile unique CloudWatch dépendait de permissions incertaines du lab pour l'écran de démonstration.
2. **« Vous détruisez le cluster à chaque session : que deviennent vos données ? »**
   - Les métriques sont jetables ; celles qui prouvent quelque chose sont exportées dans le dépôt après chaque tir.
   - Les journaux sont dans CloudWatch, avec une rétention fixée ; les tableaux de bord sont du code.
3. **« Montrez-moi comment vous diagnostiquez un incident. »**
   - Une annonce rendue non conforme est écartée : l'alerte `.ecartes` se déclenche, l'identifiant de corrélation du message mène dans Logs Insights à la requête d'origine et au refus du Contrôle (ENF-25).
4. **« Qu'est-ce qui empêche un jeton ou une donnée personnelle d'arriver dans vos journaux ? »**
   - Masquage dans le module commun, avant l'écriture, testé avec des valeurs piégées ; aucun identifiant d'utilisateur en étiquette de métrique.
   - Limite assumée : sous `LabRole`, un service compromis pourrait supprimer un groupe de journaux ; en V1, une identité par service et un stockage en ajout seul.
5. **« Pourquoi pas les services managés Prometheus et Grafana d'AWS ? »**
   - Indisponibles dans l'environnement de formation (rôles et services absents) ; en production, ce serait l'option à évaluer pour ne pas héberger la pile soi-même.

## Sources

Les prix sont à vérifier dans le calculateur AWS ; les empreintes mémoire sont des estimations à mesurer en D14.

- Contraintes du lab : [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md) (CloudWatch, ADOT, Managed Prometheus et Grafana, pièges de coût).
- Décisions amont : [D03](D03-hebergement-orchestration.md) (Conséquences, MS3, MS8), [D04](D04-stack.md) (OpenTelemetry, `slog`), [D05](D05-communication-interservice.md) (MC3, MC8, enveloppe des messages), [D06](D06-authentification-autorisation.md) (mémoire de Keycloak, masquage), [D07](D07-donnees.md) (MD7).
- Chart `kube-prometheus-stack` : https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack
- RabbitMQ et Prometheus : https://www.rabbitmq.com/docs/prometheus
- Fluent Bit, sortie CloudWatch Logs : https://docs.fluentbit.io/manual/pipeline/outputs/cloudwatch
- Container Insights : https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/ContainerInsights.html
- Tarifs CloudWatch : https://aws.amazon.com/cloudwatch/pricing/
- OpenTelemetry Go : https://opentelemetry.io/docs/languages/go/
- Grafana Tempo : https://grafana.com/docs/tempo/latest/
- Métriques de Keycloak : https://www.keycloak.org/observability/configuration-metrics
