# D11 — IaC, allumage et extinction, garde-fou coût

| Champ | Valeur |
|---|---|
| Statut | Décidée |
| Date | 2026-10-26 |
| Décideur | Martin |
| Dépend de | [D03](D03-hebergement-orchestration.md), D10 (à instruire) ; reprend les conséquences de [D05](D05-communication-interservice.md), [D06](D06-authentification-autorisation.md), [D07](D07-donnees.md) |
| Impacte | D08 (zone et certificat), D09 (rétention des journaux), D10 (analyse de l'IaC en CI), D13 (`demo.sh`), D14 (X4, X5, X13, X14, X15), `infra/`, `tools/`, `docs/exploitation.md`, section « Démarrer / arrêter » du README |
| Critères grille | C5, C10, C8 |

## Contexte et besoin

- **Exigences** :
  - ENF-28 : « l'environnement complet se crée depuis zéro et s'éteint par commande », vérifié par une « création puis destruction complètes, sans action manuelle » ([exigences.md](../00-cadrage/exigences.md)).
  - ES-29 : mise à l'échelle « avec un minimum, un maximum et une alerte de coût ».
  - ES-05 : analyse « de la configuration décrite en code » dans la chaîne de livraison ; ES-06 : aucun secret dans le dépôt, les journaux ou les sorties.
  - Grille C5 : « allumage et extinction maîtrisés » ([contexte-et-grille.md](../00-cadrage/contexte-et-grille.md)).
- **Acquis de D03** (non rediscutés ici) : EKS en node group géré, socle commun (VPC sur deux zones, une seule NAT, ALB créé par l'IaC, RDS Single-AZ, bucket créé en CLI), deux couches d'IaC persistante et éphémère, `up` en début de session, aucun `apply` du cluster après la 3e heure, `down` suivi d'un inventaire, version EKS ≥ 1.35 épinglée, plafonds HPA 6, KEDA 3, nœuds 4, destruction programmée à prouver par X15, enveloppe d'environ 38 $.
- **Acquis de D05, D06, D07** :
  - RDS recréée vide à chaque session (`skip_final_snapshot`), `rds.force_ssl`, chiffrée ; amorçage en Job `psql` éphémère dans le cluster, avant `helm install` ; jeu de démonstration rejoué ;
  - bucket des photos créé une fois en CLI, configuré par les ressources `aws_s3_bucket_*`, sans versionnage, vidé par `down` et purgé par `up` ;
  - mots de passe et secrets régénérés à chaque `up`, rangés dans Secrets Manager, livrés en Secrets Kubernetes depuis l'entrée standard (MS6, MD2) ;
  - topologie RabbitMQ (fichier de définitions) et realm Keycloak décrits dans le chart, pas dans l'IaC AWS.
- **Contraintes du Learner Lab** ([environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md)) :
  - sessions de 4 h ; identifiants de session d'environ 4 h, dont l'expiration en plein `apply` laisse l'état verrouillé [REX] ;
  - en fin de session, seules les instances EC2 s'arrêtent ; plan de contrôle EKS, NAT, ALB et RDS continuent de facturer [DOC] ;
  - IAM figé : aucun rôle créable, `LabRole` seul ; `iam:GetRole` refusé sur `voclabs` [REX] ;
  - SCP S3 : la ressource `aws_s3_bucket` échoue au `plan` (refus de `s3:GetBucketObjectLockConfiguration`) [REX] ;
  - budget de 50 $ ou 100 $ selon l'école, pour tout le projet ; affichage en retard de 8 à 12 h ; épuisement = compte désactivé et tout perdu [DOC] ;
  - au plus 9 instances EC2 et 32 vCPU par région [DOC].
- **Question ouverte du cadrage** : où stocker l'état IaC, puisque Terraform ne peut pas créer lui-même le bucket.

## Options

D11 regroupe six sous-questions. Chacune a ses options ; la synthèse les rassemble.

### Q1 — Outil d'IaC

#### Option T1 — Terraform

- **Description** : Terraform, version épinglée (`required_version`), provider AWS épinglé, fichier de verrouillage des providers versionné ; ressources `aws_eks_cluster` et `aws_eks_node_group` écrites directement (le module communautaire échoue sur `iam:GetRole` [REX]).
- **Avantages** : tous les REX du Learner Lab sont en Terraform, pièges compris (SCP S3, `GetRole`, expiration) ; documentation et exemples les plus abondants ; outil le plus cité sur le marché (C9).
- **Inconvénients** : licence BSL depuis 2023, sans effet pour un usage interne ; pas de chiffrement de l'état côté client.
- **Coût AWS et effort** : 0 $ ; effort compris dans les 2 à 3 j d'IaC de D03.
- **Complexité ajoutée** : aucune au-delà du besoin.
- **Couverture grille** : C5 (ENF-28), C9.

#### Option T2 — OpenTofu

- **Description** : la même configuration, exécutée par OpenTofu (bifurcation libre de Terraform, mêmes providers, même langage).
- **Avantages** : licence libre ; **chiffrement natif de l'état** côté client, utile si l'état contient un secret et qu'il est stocké dans un lieu lisible par `LabRole`.
- **Inconvénients** : aucun REX dans le Learner Lab, mais le provider AWS est le même, donc les mêmes pièges ; communauté plus petite ; perdre la phrase de passe, c'est perdre l'état.
- **Coût AWS et effort** : 0 $ ; passage de T1 à T2 en moins d'une heure.
- **Complexité ajoutée** : une phrase de passe à gérer si le chiffrement est activé.
- **Couverture grille** : identique à T1.

#### Options écartées

- **CDK** : l'amorçage crée des rôles IAM (refusés) ; sa construction EKS crée rôles et fonctions Lambda (D03).
- **Pulumi** : échoue lui aussi sur la SCP S3 [REX] ; un langage généraliste donne plus de liberté, donc plus de complexité, sans besoin nommé.
- **CloudFormation seul** : l'annulation côté AWS survit à l'expiration des identifiants [?], mais la partie Helm reste sur le poste, le modèle est plus verbeux et aucun REX EKS n'existe dans le lab.

### Q2 — Emplacement de l'état

#### Option E1 — État local sur le poste, hors dépôt

- **Description** : un état par couche, dans un répertoire du poste exclu du dépôt ; copie horodatée par `up.sh` et `down.sh` dans un dossier de sauvegarde ; **aucune valeur secrète dans l'état** (voir Q3).
- **Avantages** :
  - rien à amorcer ;
  - **robuste à l'expiration des identifiants** : l'écriture locale réussit toujours, sans verrou distant bloqué ni `errored.tfstate` à repousser (piège décrit par l'avis faisabilité de D03) ;
  - hors de portée de `LabRole`, donc d'un pod qui l'obtiendrait par IMDS (écart résiduel de MS3).
- **Inconvénients** : perte du poste = perte de l'état ; inutilisable depuis une CI ; pratique moins « standard » à défendre.
- **Mitigations** : la couche éphémère est vide entre deux sessions ; la couche persistante (une dizaine de ressources) se réimporte par blocs `import` ; l'inventaire ne dépend pas de l'état (Q5).
- **Coût AWS et effort** : 0 $ ; 0 j.

#### Option E2 — Bucket S3 créé en CLI, verrou natif S3

- **Description** : `bootstrap.sh` crée en CLI un bucket d'état (versionné, chiffré, Block Public Access), puis le backend S3 l'utilise avec `use_lockfile = true` (verrou natif par objet, sans DynamoDB). Le backend n'appelle pas la lecture d'Object Lock : le contournement est documenté [REX].
- **Avantages** : pratique standard ; versionnage de l'état ; utilisable depuis une CI si D10 en a besoin ; survit à la perte du poste.
- **Inconvénients** :
  - une expiration en plein `apply` laisse le verrou posé et un `errored.tfstate` local : reprise par `state push` et `force-unlock` ;
  - l'état devient lisible par `LabRole`, donc par tout pod qui obtient le rôle par IMDS ; s'il contient un secret, il faut le chiffrer (T2) ;
  - un bucket et un script d'amorçage de plus.
- **Coût AWS et effort** : < 0,01 $ par mois ; 0,25 j.

#### Variantes écartées

- **Verrou DynamoDB** : remplacé par le verrou natif S3, une table de plus sans gain.
- **État géré par la forge** (backend HTTP, si D10 retient une forge qui le propose) : hors d'AWS et hors de portée de `LabRole`, mais dépend de D10 et d'un jeton de plus sur le poste. À reconsidérer si D10 fait appliquer l'IaC par la CI.

### Q3 — Découpage en couches, réseau, secrets dans l'état

- **Deux configurations racines, sans module interne** : un seul environnement existe, un module réutilisable ne servirait à rien (YAGNI). Fichiers séparés par domaine.

| Couche | Contenu | Cycle de vie |
|---|---|---|
| `infra/persistant/` | Dépôts ECR ; zone DNS et certificat ACM (D08, la validation DNS n'est faite qu'une fois) ; configuration du bucket des photos (le bucket est créé par `bootstrap.sh`) | Appliquée une fois, détruite en fin de projet |
| `infra/ephemere/` | VPC, sous-réseaux, passerelle Internet, NAT et son adresse, point de terminaison S3 passerelle, groupes de sécurité ; ALB, écouteurs, groupes cibles, règles en liste blanche ; enregistrements DNS vers l'ALB ; cluster EKS, modèle de lancement (IMDSv2, étiquettes), node group, modules complémentaires (CNI avec politiques réseau, CoreDNS, kube-proxy, metrics-server) ; groupe de journaux à rétention courte ; RDS, groupe de sous-réseaux, groupe de paramètres | Créée par `up`, détruite par `down` |
| Chart Helm (D10) | KEDA, Cluster Autoscaler, services, RabbitMQ et ses définitions (D05), Keycloak et son realm (D06), Job de migration (D07) | Installé par `up` après l'IaC |

- **Pas de provider Kubernetes ni Helm dans Terraform** : ils dépendent d'un cluster créé dans le même `apply` et recopieraient dans l'état les valeurs des Secrets. Le script enchaîne Terraform puis Helm.
- **Liens entre couches** : la couche éphémère lit la persistante par sources de données (nom de la zone, du bucket, des dépôts), pas par lecture de son état. L'ARN de `LabRole` est construit à partir de l'identifiant du compte, sans `GetRole`.
- **Aucun secret dans l'état** : le mot de passe maître RDS est soit géré par RDS dans Secrets Manager (`manage_master_user_password`, à sonder [?]), soit passé en argument en écriture seule (`password_wo`), jamais conservé. Les autres secrets sont créés par `up.sh` en CLI. Cela retire à T2 son principal avantage et rend E1 sûr.
- **Réseau sans NAT payante ?** La NAT est actée par D03 ; son coût est chiffré ici pour mémoire :

| Variante | Coût horaire | Sur 25 sessions | Verdict |
|---|---|---|---|
| Une NAT gérée et son adresse (D03) | ≈ 0,050 $ + 0,045 $/Go traité | ≈ 5 $ | Retenue par D03 |
| Instance NAT t4g.nano ou t3.nano | ≈ 0,010 $ | ≈ 1 $ | Gain ≈ 4 $, mais une instance de plus sous la limite de 9, une image à maintenir, un composant non managé |
| Nœuds en sous-réseaux publics, sans NAT | 0 $ | 0 $ | Contraire à MS2 (nœuds privés), écartée par l'avis sécurité de D03 |
| Points de terminaison d'interface (ECR ×2, STS, EC2, journaux) | ≥ 0,10 $ sur deux zones | ≥ 10 $ | Plus cher que la NAT, et les images publiques (Keycloak, RabbitMQ) resteraient injoignables |

### Q4 — Scripts et ordre d'allumage

Quatre scripts dans `tools/`, une bibliothèque commune, plus `bootstrap.sh` lancé une fois. Bash, exécuté sous WSL ou Git Bash sur le poste (à valider en X4). Chaque phase est horodatée dans un journal de session (X4, X5, X13).

**`up.sh`** — durée cible ≤ 45 min (X4) :

| # | Étape | Durée attendue |
|---|---|---|
| 0 | Contrôles : identité, région, couche persistante présente, version EKS en support standard (`describe-cluster-versions`), heure du « Start Lab » saisie et **refus sous 75 min restantes** ; adresse publique du poste relevée pour MS1 ; purge du bucket des photos (D07) | 1 min |
| 1 | `apply` éphémère : réseau et NAT (≈ 3 min), puis en parallèle RDS (10 à 16 min) et cluster (15 à 20 min), puis node group (8 à 12 min) ; ALB et DNS en parallèle | 30 à 38 min |
| 2 | Armement du garde-fou (Q5) dès que l'`apply` réussit, même si la suite échoue | < 1 min |
| 3 | kubeconfig, KEDA et Cluster Autoscaler par Helm | 2 à 3 min |
| 4 | Mots de passe générés, rangés dans Secrets Manager, Secrets Kubernetes créés depuis l'entrée standard | 1 min |
| 5 | Job d'amorçage `psql` (bases, rôles, extensions), puis suppression du Job et du Secret maître (MD2) | 1 min |
| 6 | `helm upgrade --install` : crochet `pre-install` de migration, services, RabbitMQ, Keycloak et import du realm en parallèle | 3 à 5 min |
| 7 | Amorçage du jeu de démonstration et des photos de test (D07) | 1 à 2 min |
| 8 | Test HTTPS 200 sur le domaine ; inventaire des points d'entrée (un seul load balancer, aucun Service `LoadBalancer`) | 1 min |

Total attendu : 40 à 50 min ; chemin critique = cluster puis node group.

**`down.sh`** — durée cible ≤ 20 min (X5) :
1. retrait du Cluster Autoscaler, puis des releases Helm ; arrêt s'il reste un Service `LoadBalancer` ou un volume persistant ;
2. vidage du bucket des photos (`aws s3 rm --recursive`) ;
3. suppression des secrets de session **sans délai de récupération** (`--force-delete-without-recovery`) : sinon le nom reste réservé pendant au moins 7 jours et le `up` suivant échoue ;
4. `destroy` éphémère : node group puis cluster ; RDS ; ALB, NAT, adresse ; VPC en dernier, avec reprise sur `DependencyViolation` ;
5. suppression des planifications du garde-fou ;
6. inventaire strict (`status.sh --strict`) : code retour non nul si une ressource facturable subsiste, ce qui bloque la session suivante.

**`status.sh`** : ressources éphémères présentes, nœuds et pods, temps de session écoulé et restant, coût estimé de la session (heures × tarifs) et cumul du projet tiré du journal, orphelins.

**`demo.sh`** : part d'un `status.sh` vert, HPA et KEDA au minimum, 2 nœuds ; affiche pods, nœuds, latence et erreurs ; enchaîne tir progressif, suppression d'un pod de la Place de marché, vague de soumissions pour KEDA (D03). L'outil de charge et les seuils relèvent de D13 ; D11 ne fixe que le cadre.

### Q5 — Garde-fous coût

| # | Garde-fou | Ce qu'il arrête | Limite |
|---|---|---|---|
| G1 | **Destruction programmée** par EventBridge Scheduler sous `LabRole`, créée par `up.sh` en CLI (planifications ponctuelles, supprimées après exécution) : node group, RDS (`SkipFinalSnapshot`), ALB et NAT 30 min avant la fin de session ; cluster et adresse 10 min avant. `up.sh --prolonger` les repousse après un nouveau « Start Lab » | L'oubli d'extinction (7 à 8 $ par 24 h) | À prouver par X15 [?] : droit de confier `LabRole` à Scheduler, comportement hors session. Ce qui reste (VPC, groupes de sécurité, journaux) ne facture pas ou presque |
| G2 | **`down.sh` et inventaire strict**, qui interroge chaque service (EKS, EC2 y compris instances arrêtées, NAT, adresses, ELBv2, RDS et instantanés, volumes et interfaces disponibles, planifications, groupes de journaux sans rétention) en plus de l'API d'étiquetage | Les orphelins, dont ceux qu'EKS crée sans les étiquettes du projet | Bloquant par code retour |
| G3 | **Étiquettes** par `default_tags` : `projet=collector`, `couche`, `session` (date et heure du `up`), `outil` ; nœuds et disques étiquetés par le modèle de lancement | Rend l'inventaire et la lecture des coûts possibles | Ne couvre pas ce qu'EKS crée lui-même |
| G4 | **Plafonds validés par l'IaC** : variables avec règle de validation (nœuds ≤ 4, version EKS ≥ 1.35) ; HPA 6 et KEDA 3 dans le chart ; rétention des journaux courte | Dépassement de la limite de 9 instances, support étendu (+ 0,50 $/h), journaux à rétention infinie | Erreur de saisie rattrapée avant l'`apply` |
| G5 | **Journal des coûts** : chaque session ajoute une ligne (durée allumée, coût estimé) dans `docs/mesures/couts.csv` ; rapprochement hebdomadaire avec le budget affiché par le lab ; **seuil d'arrêt** à 70 % du budget réel, au-delà seules les répétitions de soutenance sont autorisées | La dérive lente sur plusieurs semaines | Estimation, pas facture ; le budget affiché a 8 à 12 h de retard |
| G6 | **Alerte AWS Budgets** par courriel à 50 % et 80 % du budget réel, si la sonde montre qu'elle est créable [?] | Filet tardif | 8 à 12 h de retard [DOC] : n'arrête pas un oubli d'une nuit. Exigence ES-29 « alerte de coût » couverte par G5 si G6 est refusée |
| G7 | **Rappel sur le poste** 45 min avant l'échéance (D03) | L'oubli humain | Dépend du poste allumé |

### Q6 — Analyse de l'IaC en CI (lien avec D10)

#### Option A1 — Trivy en mode configuration

- **Description** : `trivy config` sur `infra/` et sur le chart Helm, bloquant en critique et élevée ; exceptions dans un fichier versionné, chacune justifiée et datée (ES-05), par exemple l'ALB public voulu ou l'API du cluster publique restreinte à une adresse (MS1). En plus : `terraform fmt -check` et `terraform validate` (`init -backend=false`, sans identifiant AWS).
- **Avantages** : un seul outil pour les images (D04, D06 MA14), l'IaC, le chart et les secrets ; s'exécute sans AWS, donc reste vert hors session.
- **Inconvénients** : moins de règles que Checkov sur certains services AWS.
- **Coût et effort** : 0 $ ; 0,25 j.

#### Option A2 — Checkov

- **Description** : `checkov -d infra/` et sur le chart, avec ses règles de graphe entre ressources.
- **Avantages** : catalogue de règles plus large, contrôles croisés (par exemple un groupe de sécurité relié à une base publique).
- **Inconvénients** : un second outil d'analyse dans la CI, à épingler et maintenir, alors que Trivy est déjà retenu pour les images.
- **Coût et effort** : 0 $ ; 0,25 j de plus.

- **Pas de `plan` en CI** : il exige des identifiants de session, valables 4 h et à recopier (D03) ; le `plan` reste sur le poste, dans `up.sh`.

### Coût détaillé

Tarifs us-east-1 du cadrage, à confirmer par la mesure (X13).

| Poste de la couche éphémère | Base | $/h |
|---|---|---|
| Plan de contrôle EKS, support standard | 0,10 $/h | 0,100 |
| 2 nœuds t3.medium au repos | 2 × 0,0416 $/h | 0,083 |
| Nœuds de pointe en moyenne, disques des nœuds | estimation | 0,010 |
| NAT gérée et données traitées (≈ 1 Go d'images par session) | 0,045 $/h + 0,045 $/Go | 0,056 |
| ALB et unités de capacité | 0,0225 $/h + ≈ 0,008 | 0,031 |
| RDS db.t3.micro Single-AZ et 20 Go | 0,018 $/h + stockage | 0,021 |
| IPv4 publiques : NAT 1, ALB 2 | 3 × 0,005 $/h | 0,015 |
| Journaux CloudWatch, rétention courte | estimation | 0,020 |
| Secrets Manager, une dizaine au prorata | 0,40 $ par mois et par secret | 0,005 |
| **Total** | | **≈ 0,34** |

| Indicateur | Valeur |
|---|---|
| Session de 4 h allumée en entier | ≈ 1,35 $ |
| Session de 2 h 30 allumée (allumage à la demande, extinction dès la fin du travail cloud) | ≈ 0,85 $ |
| Couche persistante : zone DNS 0,50 $, ECR ≈ 0,30 $ pour 3 Go, buckets et certificat ≈ 0 | ≈ 0,8 $ par mois, ≈ 4 $ sur le projet |
| **Projet : 25 sessions de 4 h et couche persistante** | **≈ 38 $** |
| Expérimentation comparative ECS (D03, D14) | ≈ 1,25 $ |
| Marge d'incident : un oubli d'une nuit | ≈ 8 $ |
| **Enveloppe avec marge** | **≈ 47 $** |
| Oubli de 24 h / version en support étendu | ≈ 7 à 8 $ / + 2 $ par session |

**Lecture** : avec un budget de 100 $, la marge est confortable. Avec 50 $, elle ne couvre qu'un incident : les sessions passent en allumage à la demande (≈ 0,85 $) et le seuil d'arrêt de G5 devient décisif. Le budget réel est lu le premier jour.

### Synthèse

| Sous-question | Options | Ce qui départage |
|---|---|---|
| Q1 Outil | T1 Terraform / T2 OpenTofu | Sans secret dans l'état, le chiffrement de T2 n'apporte plus rien ; T1 a tous les REX du lab |
| Q2 État | E1 local hors dépôt / E2 S3 en CLI + verrou natif | E1 résiste à l'expiration des identifiants et reste hors de portée de `LabRole` ; E2 ne sert que si une CI applique l'IaC, ce que MS1 exclut |
| Q3 Couches | Deux racines sans module interne ; Helm hors Terraform ; NAT unique (D03) | Un seul environnement ; pas de secret dans l'état |
| Q4 Scripts | `up`, `down`, `status`, `demo` et `bootstrap` | Cibles ≤ 45 min et ≤ 20 min (X4, X5) |
| Q5 Garde-fous | G1 à G7 | G1 et G2 arrêtent l'oubli ; G5 la dérive ; G6 est un filet tardif |
| Q6 Analyse | A1 Trivy / A2 Checkov | Trivy est déjà dans la chaîne pour les images |

## Recommandation

**Terraform (T1), état local hors dépôt (E1) sans aucun secret dans l'état, deux configurations racines sans module, scripts `up`/`down`/`status`/`demo`, garde-fous G1 à G5 (G6 si la sonde le permet), analyse par Trivy (A1).**

1. **C'est la combinaison la plus simple qui couvre ENF-28 et le risque dominant du lab**, l'expiration des identifiants : un état local ne laisse ni verrou distant ni écriture finale en échec.
2. **La sécurité de l'état est réglée à la source** : aucun secret n'y entre, et il n'est pas lisible par `LabRole`. Le chiffrement d'OpenTofu et le bucket d'état deviennent inutiles.
3. **Terraform est l'outil des REX** : chaque piège du lab y est déjà documenté. OpenTofu reste un remplacement immédiat si Martin préfère la licence libre.
4. **Le garde-fou ne repose pas sur la mémoire** : destruction programmée (G1), inventaire bloquant (G2), journal et seuil d'arrêt (G5). L'alerte AWS Budgets, en retard de 8 à 12 h, n'est qu'un filet.
5. **Un seul analyseur** : Trivy couvre images, IaC et chart ; ES-05 tenue sans outil de plus.

**Condition de bascule vers E2** : si D10 fait exécuter l'IaC par la CI, ou si Martin veut montrer un état distant au jury. Coût : 0,25 j et la procédure de reprise après expiration.

**Volontairement exclu** : modules internes, Terragrunt, providers Kubernetes et Helm dans Terraform, verrou DynamoDB, `plan` en CI, instance NAT, Checkov en plus de Trivy, outil de suivi de coût tiers. Chaque ajout devra nommer le besoin qu'il sert.

### Points soumis aux avis

- **Sécurité** :
  - état local sur le poste (E1) contre état S3 lisible par `LabRole` (E2) ; protection et sauvegarde de l'état local ;
  - absence de secret dans l'état : `manage_master_user_password` [?] ou `password_wo` ; vérification par X20 ;
  - G1 confie à Scheduler un pouvoir de suppression sous `LabRole` : bornage des planifications au compte et au projet ;
  - suppression des secrets sans délai de récupération : aucun retour arrière, acceptable pour des secrets régénérés à chaque session ;
  - liste des exceptions Trivy acceptées d'emblée.
- **Faisabilité** :
  - allumage ≤ 45 min et extinction ≤ 20 min tenables (X4, X5) ; reprise après expiration en plein `up` ;
  - G1 réellement exécutable sous `LabRole` (X15), et ce qu'elle laisse ;
  - scripts Bash sur un poste Windows (WSL ou Git Bash) ;
  - fiabilité de l'inventaire face à ce qu'EKS crée hors étiquettes ;
  - instance NAT pour gagner ≈ 4 $ si le budget réel est de 50 $.

## Avis sécurité

Fonction : Sécurité — 2026-10-26. Références : [exigences-securite.md](../securite/exigences-securite.md) (FC8, FC9, ES-05, ES-06, ES-07, ES-29, ES-30 ; M-LIV-02, M-LIV-03, M-MOD-03) ; avis sécurité de [D03](D03-hebergement-orchestration.md) (MS1, MS2, MS3, MS6, X16, X20) ; [PILOTAGE.md](../PILOTAGE.md) § 7. Avis sur dossier : aucun code d'IaC n'existe encore ; les points [?] relèvent des sondes de D14.

**Synthèse.** Avis favorable à T1 + E1 + A1 et aux garde-fous G1 à G5, sous les mesures MI1 à MI10. Aucun point bloquant. Un écart documentaire à corriger : MS6 (D03) exige un « état Terraform chiffré et privé », alors qu'E1 est un fichier en clair sur le poste. Soit MS6 devient « état sans secret, privé », soit le disque du poste est chiffré (MI2) ; les deux sont recommandés.

**1. État local (E1) contre S3 (E2)**

- **Confidentialité** : favorable à E1. Le risque dominant reste la chaîne « pod compromis → IMDS → `LabRole` » (M-MOD-03) : un état dans S3 lui serait lisible, un état local non. Même sans secret, l'état est une carte complète de l'infrastructure (ARN, adresses, règles des groupes de sécurité) : confidentialité « interne », pas « critique ».
- **Perte** : impact faible. La couche éphémère est vide entre deux sessions, la persistante se réimporte. Le vrai risque est ailleurs : un `apply` sans état recrée en double et laisse des ressources non gérées, invisibles au `destroy` (M-INF-05). L'inventaire indépendant de l'état (G2) couvre ce cas.
- **Sauvegarde** : les copies horodatées sur le même disque protègent d'une corruption, pas de la perte du poste. Ajouter une copie hors poste, chiffrée (archive chiffrée ou `age`), jamais dans le dépôt ni dans un dossier synchronisé en clair. ES-30 ne s'applique pas : l'état ne contient aucune donnée métier.
- **Poste sous Windows 11 Famille** : BitLocker complet n'y existe pas ; vérifier que le « chiffrement de l'appareil » est actif. C'est la seule protection de l'état et des identifiants en cas de vol du poste.

**2. Aucun secret dans l'état : vrai seulement si la configuration l'impose**

- `manage_master_user_password` : l'état ne garde que l'ARN du secret créé par RDS, pas sa valeur ; à préférer si la sonde le confirme. `password_wo` : valable seulement si les versions épinglées de Terraform et du provider prennent en charge les arguments en écriture seule.
- **Pièges qui réintroduisent un secret sans qu'on le voie** : ressource `random_password` (valeur dans l'état ; générer en CLI ou utiliser la variante éphémère) ; `aws_secretsmanager_secret_version` avec `secret_string` ; `output` d'une valeur secrète, même marqué `sensitive` (le marquage masque l'affichage, pas l'état) ; fichiers de plan (`-out`), qui contiennent les valeurs en clair.
- **Vérification** (dans X20 et `status.sh`) : après `up`, `terraform show -json` sur chaque couche, recherche des valeurs générées pour la session (lues dans Secrets Manager, jamais affichées) et des clés `password`, `secret_string`, `private_key` ; tout attribut de `sensitive_attributes` est justifié. Attendu : aucune occurrence.
- **`.gitignore`** : `*.tfstate`, `*.tfstate.*`, `errored.tfstate`, `*.tfplan`, `.terraform/`, `crash.log`, dossier des sauvegardes. Le fichier de verrouillage des providers reste versionné.

**3. G1 : pouvoir de suppression confié à EventBridge Scheduler sous `LabRole`**

- Ce pouvoir n'est pas nouveau : `LabRole` le détient déjà, et un pod qui l'obtient peut créer ses propres planifications (M-MOD-03). Le risque propre à G1 est l'erreur de ciblage : supprimer une ressource persistante (zone DNS, certificat, ECR), ou une planification restée armée qui détruit la session suivante.
- Bornage (MI5) : groupe de planifications dédié ; chaque cible désignée par son identifiant exact, sans joker ni enchaînement `List*` puis suppression ; actions limitées aux suppressions de la couche éphémère ; `ActionAfterCompletion=DELETE` ; `up.sh` purge le groupe avant d'armer, `down.sh` le vide, l'inventaire le contrôle ; aucun secret dans les charges utiles.
- `SkipFinalSnapshot` et suppression des secrets sans délai de récupération : acceptables (données fictives, secrets régénérés à chaque `up`, aucun retour arrière attendu).

**4. Identifiants de session sur le poste**

- Clés temporaires (environ 4 h) mais complètes pour `LabRole`. Les ranger dans un profil nommé du fichier d'identifiants AWS de l'utilisateur ; jamais dans le dépôt, un `.env`, un fichier de démarrage du shell, ni en argument de commande (historique).
- Les scripts ne tracent ni l'environnement ni les commandes (pas de `set -x` dans les phases qui manipulent des secrets) ; le journal de session ne contient que des identifiants de ressources ; `down.sh` efface le profil.
- Aucun identifiant AWS en CI pour l'IaC (déjà acté) : c'est la meilleure mesure contre M-LIV-02 sur l'hébergement, à conserver.

**5. Exceptions Trivy acceptées d'emblée**

- Acceptables, chacune justifiée, datée et posée **par ressource**, jamais par règle globale : ALB public (ES-07) ; port 80 de l'ALB pour la seule redirection ; API du cluster publique restreinte au /32 du poste (MS1) ; sortie des nœuds vers Internet via la NAT (images publiques) ; journaux de flux du VPC absents (coût) ; RDS sans protection contre la suppression, sans rétention de sauvegarde ni authentification IAM (base recréée vide, données fictives ; ES-30 reste exigée en V1) ; Secrets EKS sans clé KMS du client, si la sonde la montre non créable.
- **Jamais en exception** : base ou bucket publics, stockage non chiffré, entrée depuis `0.0.0.0/0` hors ALB, IMDSv1, nœuds avec IP publique, dépôt ECR sans analyse à la poussée.

**6. Groupes de sécurité, sous-réseaux, chiffrement par défaut**

- Réseau : RDS dans des sous-réseaux privés, `publicly_accessible = false`, 5432 ouvert au seul groupe des nœuds ; nœuds sans IP publique (MS2) ; groupe de sécurité par défaut du VPC repris par l'IaC et vidé.
- Chiffrement : `storage_encrypted` de RDS vaut `false` par défaut dans Terraform, à poser explicitement ; disques des nœuds chiffrés dans le modèle de lancement, sans dépendre du réglage du compte ; bucket des photos avec chiffrement, blocage de l'accès public et propriété imposée déclarés ; ECR, journaux et Secrets Manager sont chiffrés par défaut.

**7. L'inventaire est aussi un contrôle de sécurité**

- Un orphelin n'est pas qu'un coût : load balancer né d'un Service `LoadBalancer`, groupe ouvert, adresse publique, instantané RDS ou planification destructive restée armée sont des expositions hors IaC, donc hors de l'analyse Trivy. `status.sh --strict` doit échouer aussi sur ces critères (MI8).

**Mesures**

| # | Mesure | Menaces | Preuve |
|---|---|---|---|
| MI1 | État hors dépôt, `.gitignore` complet (§ 2), fichiers de plan jamais conservés | M-LIV-02, M-INF-01 | Revue du `.gitignore`, détection de secrets en CI |
| MI2 | Chiffrement du disque du poste actif ; copie de l'état hors poste chiffrée ; MS6 reformulée | M-INF-01, M-INF-05 | Capture du réglage, procédure dans `docs/exploitation.md` |
| MI3 | Mot de passe RDS par `manage_master_user_password` ou `password_wo` ; ni `random_password`, ni `secret_string`, ni `output` secret | M-INF-01 | Revue de code, X20 |
| MI4 | Recherche des secrets de session dans l'état après chaque `up` | M-INF-01 | X20, sortie de `status.sh` |
| MI5 | Planifications G1 bornées (groupe dédié, cibles exactes, couche éphémère seule, suppression après exécution, purge par `up` et `down`) | M-INF-03 | X15, inventaire |
| MI6 | Identifiants de session en profil nommé, jamais en argument, `.env` ni journal ; profil effacé par `down.sh` ; pas de `set -x` | M-INF-02 | Revue des scripts, détection de secrets sur le journal de session |
| MI7 | Exceptions Trivy par ressource, justifiées et datées, liste fermée du § 5 ; toute nouvelle exception critique ou élevée validée par Martin | M-LIV-03 | Fichier d'exceptions versionné, rapport CI |
| MI8 | Inventaire bloquant aussi sur : load balancer hors IaC, groupe ouvert à `0.0.0.0/0` hors ALB, RDS publique, bucket sans blocage d'accès public, adresse publique, planification restante, instantané RDS | M-INF-04 | `status.sh --strict`, X5 |
| MI9 | Chiffrement explicite : RDS, disques du modèle de lancement, bucket | M-LIV-03 | `trivy config`, inventaire |
| MI10 | RDS privée filtrée sur le groupe des nœuds, groupe par défaut du VPC vidé, nœuds sans IP publique | M-LIV-03 | `trivy config`, X16 |

**Menaces à ajouter au registre** (§ 4.10, risque brut, échelle du § 1.3 ; non reportées ici dans le registre)

| # | STRIDE | Frontière | Menace | Impact | V | I | Risque |
|---|---|---|---|---|---|---|---|
| M-INF-01 | I | FC9 | Un secret entre dans l'état ou un plan (ressource aléatoire, version de secret, sortie), puis fuit par le poste, une sauvegarde ou un commit | Accès à la base ou aux intégrations | 2 | 3 | 6 Modéré |
| M-INF-02 | S, E | FC9 | Vol des identifiants de session sur le poste (fichier, historique, journal) | Contrôle du compte du lab pendant la session | 2 | 3 | 6 Modéré |
| M-INF-03 | D | FC9 | Planification de destruction mal ciblée ou restée armée | Ressource persistante perdue, session suivante détruite | 2 | 2 | 4 Modéré |
| M-INF-04 | I | FC1, FC8 | Ressource orpheline hors IaC restée exposée (load balancer, groupe ouvert, instantané) | Exposition non analysée, coût | 2 | 3 | 6 Modéré |
| M-INF-05 | D, T | FC9 | Perte ou désynchronisation de l'état : ressources en double ou non gérées | Orphelins exposés et facturés | 2 | 2 | 4 Modéré |

## Avis faisabilité

Fonction : Faisabilité — 2026-10-26. Périmètre : Q1 à Q6, points soumis aux avis. Référence : D03 (socle, X4, X5, X15), D06, D07, D08, D09 et D10 (avis faisabilité, ordre d'allumage), [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md).

**Synthèse.** Avis favorable à T1, E1, deux racines, A1 et G1 à G5. Rien ne bloque, mais la recommandation doit être recalée sur les décisions voisines : Q4 et Q3 ignorent Flux, GHCR (D10) et la pile d'observabilité (D09). Une fois recalé, l'allumage atteint 45 min en médiane, pas au pire ; l'extinction ≤ 20 min est tenable. Effort : **5 à 6 j**, au lieu de 3,75 à 5,25 j.

### Écarts à corriger dans la recommandation

- **ECR dans la couche persistante** : si D10 retient GHCR (R1), retirer ECR et son coût (≈ 0,30 $ par mois). Ajouter le trafic de tirage par la NAT, de l'ordre de 1 à 2 Go par session.
- **Groupes de journaux** : Q3 les range dans la couche éphémère, D09 (MO7) dans la persistante. Seule la couche persistante tient l'audit (30 j) d'une session à l'autre : y placer les groupes applicatif et d'audit. Le groupe `/aws/eks/<cluster>/cluster` reste éphémère, mais il doit être **déclaré par Terraform avant le cluster** : sinon EKS le crée lui-même, sans rétention, et l'`apply` suivant échoue sur « already exists ».
- **Q4 sans Flux ni observabilité** : les étapes 3 et 6 doivent suivre l'ordre de D09 et D10, décrit ci-dessous.

### Allumage chronométré (X4)

| # | Étape | Durée réaliste | Cumul |
|---|---|---|---|
| 0 | Contrôles, refus sous 75 min restantes, purge du bucket | 1 min | 1 |
| 1 | Réseau, NAT, groupes de sécurité | 3 à 4 min | 5 |
| 2 | Cluster EKS (chemin critique) ; en parallèle : RDS (10 à 16 min), ALB, écouteurs, alias DNS | 12 à 20 min | 17 à 25 |
| 3 | Modules complémentaires, puis node group (2 nœuds) | 5 à 12 min | 22 à 37 |
| 4 | Armement de G1, kubeconfig | 1 min | 23 à 38 |
| 5 | `flux install` ; `kube-prometheus-stack` allégé et ses CRD en application côté serveur ; Fluent Bit ; KEDA ; Cluster Autoscaler | 3 à 5 min | 26 à 43 |
| 6 | Secrets générés (Secrets Manager, puis Secrets Kubernetes), Job `psql` d'amorçage (RDS déjà prête) | 1 à 2 min | 27 à 45 |
| 7 | Réconciliation Flux : RabbitMQ, Keycloak et import du realm, migration, services. Tirage des images GHCR par la NAT | 4 à 6 min | 31 à 51 |
| 8 | Jeu de démonstration, test HTTPS 200, inventaire des points d'entrée | 2 min | 33 à 53 |

- **Lecture** : 40 min en médiane, 53 min au pire. Le REX du lab donne 40 min depuis zéro. Garder 45 min comme cible et 60 min comme seuil d'alerte dans le journal. Sur 4 h, il reste au moins 2 h 30 de travail utile : le refus sous 75 min est cohérent.
- **Gains possibles** : lancer le node group dès la création du cluster, sans attendre une autre ressource (`depends_on` minimal) ; installer KEDA et CA pendant que Flux tire les images ; et surtout ne jamais réécrire la couche persistante pendant l'`up`.
- **Reprise après expiration en plein `apply`** : avec E1, l'état est écrit, mais une ressource en cours de création peut exister sans être suivie. Avant tout nouvel `apply`, `up.sh` doit relancer l'inventaire ; tout orphelin se supprime ou s'importe. Le même cas se produit après G1 : un `terraform apply -refresh-only` doit précéder l'`apply`.

### Extinction et inventaire (X5)

- **Durées** : releases Helm, puis Flux (1 à 2 min) ; node group (4 à 8 min), puis cluster (4 à 8 min) ; RDS (5 à 10 min) en parallèle ; NAT (1 à 2 min) ; VPC. Total : **12 à 20 min**. Retirer Flux **avant** les releases, sinon il les recrée.
- **Ce que Terraform rate**, et que l'inventaire doit interroger service par service :
  - **ENI** `available` laissées par le CNI VPC (`aws-K8S-*`) et ENI de l'ALB en cours de libération : cause principale de `DependencyViolation` sur les sous-réseaux et les groupes de sécurité. Prévoir une boucle de reprise (3 essais, 60 s) et une suppression des ENI `available` étiquetées du cluster ;
  - **groupe de sécurité du cluster** `eks-cluster-sg-*`, créé par EKS : en principe détruit avec le cluster, mais orphelin si la destruction est interrompue ;
  - **volumes EBS** : aucun attendu (ni pilote CSI ni PVC) ; seuls les disques racine des nœuds partent avec eux. L'inventaire vérifie les volumes `available` ;
  - **groupes de journaux** : celui du plan de contrôle s'il a été créé par EKS ; tout groupe sans rétention ;
  - **secrets** : ceux de session, supprimés sans délai de récupération, et le secret maître de RDS s'il est géré par RDS (supprimé avec l'instance, à vérifier) ;
  - **adresses Elastic IP** non associées ; planifications Scheduler restantes ; instantanés RDS (aucun attendu).
- L'API d'étiquetage ne voit pas les ressources non étiquetées : l'inventaire filtre par VPC et par préfixe de nom, pas seulement par étiquette.

### Destruction programmée G1 (X15)

- **Faisable [?], probabilité élevée** : `scheduler.amazonaws.com` figure dans la politique d'approbation de `LabRole` (relevée dans un état de 2026-07). Le seul point ouvert est `iam:PassRole` de `LabRole` vers Scheduler. Les cibles universelles couvrent `eks:DeleteNodegroup`, `eks:DeleteCluster`, `rds:DeleteDBInstance`, `ec2:DeleteNatGateway`, `elasticloadbalancing:DeleteLoadBalancer` et `ec2:ReleaseAddress`.
- **Comportement hors session** : Scheduler assume le rôle lui-même, indépendamment des identifiants de la session. Il doit donc fonctionner après la fin de session, mais c'est à prouver par X15 : planification à +10 min, puis observation après un « End Lab ».
- **Contraintes d'enchaînement** : le cluster ne se supprime qu'une fois le node group parti (5 à 8 min) ; l'adresse ne se libère qu'une fois la NAT supprimée. Les planifications à T-30 et T-10 de G1 sont donc justes ; une marge de 5 min de plus serait prudente.
- **Ce qui reste** : VPC, sous-réseaux, groupes de sécurité, groupes de cibles et groupe de journaux du plan de contrôle. Tout cela est gratuit ou presque, et le `refresh` du `up` suivant le nettoie.
- **Repli** si `PassRole` est refusé : G2, G5 et G7 seuls. Le risque d'oubli d'une nuit (≈ 8 $) est alors couvert par la seule discipline.

### Scripts Bash sous Windows

- **Git Bash suffit**, car `aws`, `terraform`, `kubectl`, `helm` et `flux` existent en binaires Windows natifs. Trois pièges :
  - conversion de chemins MSYS : `/aws/eks/...` devient `C:/Program Files/Git/aws/...`. Exporter `MSYS_NO_PATHCONV=1` dans la bibliothèque commune ;
  - fins de ligne : `.gitattributes` avec `*.sh eol=lf` ;
  - outils à installer : `jq`, et `openssl` pour générer les mots de passe.
- **WSL** est plus fidèle (même environnement que la CI), mais impose de recopier les identifiants de session dans un second `~/.aws` à chaque « Start Lab ». À garder en repli.
- `shellcheck` en CI sur `tools/` (0,1 j).

### NAT gérée ou instance NAT

- **Garder la NAT gérée.** L'instance NAT (image `fck-nat`, t4g.nano) fait gagner ≈ 4 $, mais coûte 0,5 j. Elle occupe aussi une instance sur 9, devient un point de panne non managé (aucun redémarrage automatique) et ajoute une cible à G1 et à l'inventaire. Ne la reconsidérer qu'avec un budget réel de 50 $ **et** un G5 au-delà de 60 % avant la mi-projet.

### Effort et risques

| Poste | Effort |
|---|---|
| IaC des deux couches, dont la reprise et le `refresh` | 2,5 à 3 j |
| `up`, `down`, `status`, bibliothèque, ordre Flux et observabilité | 1,5 à 2 j |
| Inventaire strict (ENI, groupes de sécurité, journaux, secrets) et G1 avec X15 | 0,75 j |
| `fmt`, `validate`, Trivy, `shellcheck` en CI | 0,25 j |
| **Total** | **5 à 6 j** |

- **Risques** :
  - expiration des identifiants en plein `apply` : ressources non suivies, mitigées par l'inventaire avant l'`apply` ;
  - `DependencyViolation` sur le VPC à cause des ENI : `down` au-delà de 20 min ;
  - CRD absentes au déploiement : `dependsOn` à tester dès le premier `up` ;
  - `PassRole` refusé : G1 inexistante ;
  - compte réinitialisé : zone et re-délégation DNS à refaire (D08).

## Décision de Martin

> **« Terraform, état local »** : Terraform en version épinglée, état local hors dépôt sur un poste dont le chiffrement de l'appareil est activé (reformulation de MS6), deux configurations racines persistante et éphémère, Helm hors de Terraform, aucun secret dans l'état (vérifié par `terraform show -json`). Scripts `bootstrap`, `up` (médiane ~40 min, alerte à 60 min), `down` (12 à 20 min, Flux retiré en premier) suivi d'un inventaire bloquant au-delà des étiquettes, `status` et `demo`. Garde-fous G1 à G5 dont la destruction programmée (X15) et l'arrêt à 70 % du budget ; NAT gérée conservée ; Trivy pour l'IaC. Mesures de sécurité MI1 à MI10.
>
> Ajustements issus de D10 : ECR retiré de la couche persistante (GHCR) ; groupes de journaux applicatif et d'audit dans la couche persistante (D09).

Date : 2026-10-26

## Conséquences

La décision de Martin est enregistrée ci-dessus ; ses conséquences :

- **Arborescence** : `infra/persistant/`, `infra/ephemere/` ; `tools/bootstrap.sh`, `up.sh`, `down.sh`, `status.sh`, `demo.sh` et une bibliothèque commune ; état local dans un répertoire exclu par `.gitignore`.
- **D08** : zone et certificat dans la couche persistante ; enregistrements repointés vers l'ALB à chaque `up`, ce qui garde l'émetteur Keycloak stable (D06).
- **D09** : groupes de journaux créés par l'IaC avec une rétention courte ; volume compté dans X13.
- **D10** : jobs `terraform fmt`, `validate` et `trivy config` sur `infra/` et le chart, bloquants en critique et élevée ; aucun identifiant AWS dans la CI pour l'IaC ; le déploiement reste lancé depuis le poste (MS1).
- **D13** : `demo.sh` appelle l'outil de charge retenu ; seuils fixés par D13.
- **D14** : X4, X5, X13, X14 et X15 mesurés avec ces scripts ; ajout à la sonde du premier jour : `manage_master_user_password`, création d'un budget AWS, `iam:PassRole` de `LabRole` vers Scheduler.
- **Sécurité** : [exigences-securite.md](../securite/exigences-securite.md) : ES-29 précisée (alerte de coût par G5, G6 si disponible) ; ES-05 pour l'IaC.
- **Exploitation** : `docs/exploitation.md` (procédures, reprise après expiration, jour J) et section « Démarrer / arrêter » du README ; `docs/mesures/couts.csv` alimenté à chaque session.
- **Coût** : ≈ 0,34 $/h, ≈ 1,35 $ par session, ≈ 38 $ sur le projet, ≈ 47 $ avec marge. Budget réel relevé par XP00 le 2026-10-26 : **50 $**. La marge tombe sous 10 % : sessions en allumage à la demande, seuil G5 à 70 % décisif, réserve de rejeu de D14 comme variable d'ajustement, coût réel suivi dans `docs/mesures/couts.csv` à chaque session. XP00 S8b : une planification EventBridge exécutée lab arrêté est refusée (`LabRole` sans droits hors session) ; G1 ne protège que pendant la session, `down.sh` doit donc se terminer avant « End Lab ».
- **Effort** : IaC 2 à 3 j (D03) ; scripts, inventaire, garde-fou et journal 1,5 à 2 j ; analyse en CI 0,25 j.
- **Index** : D11 à mettre à jour dans [INDEX.md](INDEX.md).

## À savoir défendre en soutenance

1. **« Pourquoi un état Terraform local, alors que la bonne pratique est un état distant ? »**
   - L'état distant sert le travail à plusieurs et l'exécution en CI. Ici, une seule personne applique l'IaC, depuis son poste (MS1).
   - Dans ce lab, les identifiants expirent au bout de 4 h : un état distant laisse alors un verrou posé. L'état local, lui, s'écrit toujours.
   - Il ne contient aucun secret et `LabRole`, partagé par les pods, ne peut pas le lire. La couche éphémère est vide entre deux sessions et la couche persistante se réimporte.
   - En V1 : état distant chiffré et verrouillé, appliqué par la CI avec une identité fédérée.
2. **« Comment êtes-vous sûr de ne rien oublier d'allumé ? »**
   - Trois barrières : `down.sh` suivi d'un inventaire bloquant qui interroge chaque service, une destruction programmée avant la fin de session (X15), un journal des coûts avec un seuil d'arrêt.
   - L'alerte de budget arrive 8 à 12 h trop tard : c'est un filet, pas la protection.
3. **« Combien coûte une session, et le projet ? »**
   - ≈ 0,34 $/h, détaillé poste par poste ; ≈ 1,35 $ par session, ≈ 38 $ sur le projet, ≈ 47 $ avec marge. Chiffres mesurés par X13 à reporter ici.
   - Les deux pièges les plus chers sont neutralisés par l'IaC : version EKS en support étendu (validation de variable) et oubli d'une nuit (G1, G2).
4. **« Pourquoi pas de NAT, ou pourquoi une NAT payante ? »**
   - Une NAT gérée : ≈ 5 $ sur le projet, nœuds privés (ES-07). Les points de terminaison coûteraient le double ; les nœuds publics ou une instance NAT économisent quelques dollars au prix de l'exposition ou d'un composant non managé.
5. **« Comment l'environnement se recrée-t-il depuis zéro ? »**
   - `up.sh` : contrôles, `apply`, garde-fou armé, composants du cluster, secrets régénérés, amorçage de la base, `helm install`, jeu de démonstration, test HTTPS. Durée mesurée par X4, journal horodaté à l'appui (ENF-28).

## Sources

Références tirées du cadrage et des décisions précédentes ; les pages externes ci-dessous sont à vérifier avant la décision.

- **Projet** : [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md) (contraintes, tarifs, sonde) ; [D03](D03-hebergement-orchestration.md) (socle, avis faisabilité, X4, X5, X13 à X15, MS1 à MS10) ; [D05](D05-communication-interservice.md), [D06](D06-authentification-autorisation.md), [D07](D07-donnees.md) (§ Conséquences et Exploitation).
- **Terraform** : backend S3 et `use_lockfile` : https://developer.hashicorp.com/terraform/language/backend/s3 ; arguments en écriture seule : documentation Terraform « write-only arguments ».
- **OpenTofu** : chiffrement de l'état : https://opentofu.org/docs/language/state/encryption/
- **AWS** :
  - EventBridge Scheduler, cibles universelles : https://docs.aws.amazon.com/scheduler/latest/UserGuide/managing-targets-universal.html
  - Suppression d'un secret sans délai : https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html
  - Tarifs : https://aws.amazon.com/eks/pricing/ ; https://aws.amazon.com/vpc/pricing/ ; https://aws.amazon.com/elasticloadbalancing/pricing/ ; https://aws.amazon.com/route53/pricing/ ; https://aws.amazon.com/ecr/pricing/
- **Analyse de l'IaC** : Trivy : https://trivy.dev/ ; Checkov : https://www.checkov.io/
