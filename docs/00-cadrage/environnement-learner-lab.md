# Contraintes de l'environnement AWS Academy Learner Lab

Recherche documentaire du **2026-10-09**, à confirmer par la sonde du premier jour (§ 7) : aucune sonde n'a encore été lancée sur le compte.

**Légende des niveaux de preuve**
- **[DOC]** : Readme officiel du Learner Lab, section « Service usage and other restrictions ». La version citée porte la mention « Instructions last updated: 2025-06-24 ». Elle est recopiée dans des dépôts étudiants de 2026, et trois copies indépendantes concordent. S'y ajoutent la documentation et les grilles tarifaires AWS.
- **[REX]** : retour d'expérience daté. Il s'agit de dépôts étudiants : sondes d'API, fichiers `terraform.tfstate` montrant des ressources effectivement créées sous le rôle `voclabs`, et runbooks.
- **[?]** : information contradictoire, déduite ou introuvable.

> **Avertissement structurant.** Il n'existe pas « un » Learner Lab, mais plusieurs variantes :
> - le Learner Lab « complet », dont la liste de services est décrite ci-dessous ;
> - l'ancien « Learner Lab – Foundational Services » (2021), dont la liste ne contient **ni ECS ni EKS** ;
> - les sandboxes intégrées à certains cours. Par exemple, la sandbox de *Cloud Developing* n'offre **ni ECS, ni Fargate, ni EKS, ni App Runner** (REX du 2026-09-23).
>
> Les permissions effectives varient aussi **d'un compte à l'autre et dans le temps**. Ainsi, les rôles `LabEksClusterRole` sont présents dans certains comptes et absents d'autres, et une SCP sur S3 est apparue en 2026. **Première action à mener :** lire le Readme du lab fourni par l'école, puis lancer un script de sonde dès le premier jour (voir §7).

---

## 0. Synthèse

1. **Sessions de 4 h, ressources persistantes.** En fin de session, **seules les instances EC2 sont arrêtées**, puis redémarrées automatiquement à la session suivante. Le control plane EKS, le NAT, les load balancers, les tâches Fargate et souvent RDS **continuent de facturer**. Le dépassement du budget **désactive le compte et efface tout** [DOC].
2. **IAM figé.** Il est impossible de créer des rôles, des utilisateurs, des fournisseurs OIDC ou d'attacher des policies. On ne dispose que de `LabRole`, de `LabInstanceProfile` et, selon les comptes, de `LabEksClusterRole` / `LabEksNodeRole`. Conséquences :
   - **pas de fédération OIDC GitHub Actions ou GitLab vers AWS** ;
   - **pas d'IRSA ni d'EKS Pod Identity**.

   Le CI/CD doit donc utiliser des identifiants de session de 4 h, à recopier à chaque session, ou un modèle « pull » (GitOps).
3. **Orchestrateurs.**
   - **ECS (EC2 et Fargate) et EKS sont utilisables.**
   - Sur ECS, l'autoscaling de service est prouvé et `LabRole` sert nativement de *task role*.
   - Sur EKS, tout ce qui demande une identité IAM par pod se fait par contournement : AWS Load Balancer Controller, EBS CSI, agent CloudWatch, External Secrets.
   - **Cloud Map est bloqué**, donc ECS Service Connect et la découverte de services DNS ne sont pas disponibles sur ECS.
4. **Briques managées indisponibles** (non listées, accès refusé en REX) : **Amazon MQ, MSK, CloudFront, CodeBuild, CodePipeline, Cloud Map**, ainsi que, faute de preuve, **App Runner, Amazon Managed Prometheus et Amazon Managed Grafana**.
   - Pour le broker : SQS/SNS/EventBridge (managés), ou RabbitMQ/Kafka en conteneur.
   - Pour l'OIDC : **Cognito User Pools** fonctionne en REX.
5. **HTTPS sans CloudFront.** Trois voies, de la plus « AWS » à la plus simple :
   - un domaine externe, une zone Route 53 et un certificat ACM validé par DNS, servis par un ALB ou un NLB (prouvé en REX) ;
   - API Gateway, dont l'endpoint `execute-api` est en HTTPS natif, relié par VPC Link à un NLB ou un ALB interne (REX) ;
   - Let's Encrypt sur sslip.io ou DuckDNS (REX).
6. **Coût d'une session de démo de 4 h** (stack EKS ou ECS minimale) : **environ 1 à 1,5 $**. Le même stack laissé allumé coûte **5 à 7 $ par jour**. **Piège majeur en octobre 2026 :** les versions EKS 1.31 à 1.33 sont en *extended support*, à **0,60 $/h** au lieu de 0,10 $/h.

---

## 1. Fonctionnement du Learner Lab

| Sujet | Constat | Niveau |
|---|---|---|
| Durée de session | **4 h**, avec compte à rebours. Cliquer de nouveau sur « Start Lab » avant 0:00 remet le compteur à 4 h. | [DOC] Readme (« refresh the session length »). Handout UQ du 2026-02-19 (« A lab can only remain active for 4 hours »). |
| Persistance entre sessions | L'environnement est « long-lived » : *« any data and resources that you created in the AWS account will be retained »*. | [DOC] |
| Fin de session : EC2 | Les instances en cours sont **arrêtées**, puis **redémarrées automatiquement** à la session suivante, y compris celles arrêtées manuellement. La protection contre l'arrêt est retirée. L'IP publique change, sauf avec une Elastic IP. | [DOC] |
| Fin de session : autres services | RDS : *« the lab environment may not stop an RDS instance »*. Une instance RDS arrêtée redémarre seule au bout de 7 jours. Les notebooks SageMaker sont arrêtés sans redémarrage. Un cluster EMR est cassé. Le Readme range « EMR, ECS or EKS clusters » et « NAT Gateway » parmi les ressources de calcul à éteindre soi-même. | [DOC] |
| Comportement réel d'EKS et d'ECS | Le control plane EKS, les ELB, le NAT et RDS **continuent de facturer**. Les nœuds de node group arrêtés par le lab passent `NotReady` ou sont **remplacés par l'ASG**, ce qui consomme du budget en silence. | [REX] 2026-07 et 2026-09 |
| Budget | Fixé par l'école : **50 $** (UQ 2026, UCSC, KMITL 2026) ou **100 $** (Seneca, U-Michigan). Il vaut pour toute la durée du lab, 6 mois au maximum chez UCSC. Le bouton « Reset » **ne remet pas le budget à zéro**. | [DOC] et pages d'universités |
| Affichage du budget | Il provient d'AWS Budgets et n'est **mis à jour que toutes les 8 à 12 h**. | [DOC] |
| Épuisement du budget | *« your lab account will be disabled and all progress and resources will be lost »*. | [DOC] |
| Autres désactivations | 20 instances ou plus en cours d'exécution entraînent la désactivation immédiate et la suppression des ressources. Plus généralement, *« any attempt to exceed a service limit, may result in immediate deactivation »*. En REX, deux comptes ont été suspendus (code `SUSPENDED9`) après le dépassement d'un plafond de concurrence SageMaker. | [DOC] et [REX] 2026-09 |
| Régions | **us-east-1 et us-west-2 uniquement**. La clé SSH `vockey` n'existe qu'en us-east-1. | [DOC] |
| Identifiants | Identifiants STS temporaires (`ASIA…` et `aws_session_token`), affichés via « AWS Details → AWS CLI ». Ils **meurent avec la session**, au bout d'environ 4 h. Le principal est `arn:aws:sts::<acct>:assumed-role/voclabs/userNNN=Nom`. | [REX] convergents. Pas de durée officielle trouvée [?]. |
| Après la fin de session | Toute requête échoue avec *explicit deny … policy `voc-cancel-cred`*. Ce n'est pas un problème de permission : il faut redémarrer le lab et recopier les nouveaux identifiants. Les onglets de console d'une session précédente sont également révoqués. Ce message est souvent mal diagnostiqué comme « service interdit ». | [REX] 2026-07 à 2026-10, nombreux |
| Accès | Console via le lien « AWS » de l'interface Vocareum. CLI et SDK sur le poste avec les identifiants copiés. **CloudShell** avec identifiants préconfigurés. Terminal Vocareum avec la clé `labsuser.pem`. Session Manager via `LabInstanceProfile`. | [DOC] |
| Limites EC2 | Types **nano à large** uniquement : les tailles xlarge renvoient `UnauthorizedOperation` (REX 2026-07). Au plus **9 instances simultanées et 32 vCPU par région**, ce qui inclut les nœuds créés par EKS, ECS, Elastic Beanstalk et EMR. On-Demand uniquement. EBS de 100 Go maximum, sans PIOPS. EC2 Fleet non supporté. | [DOC] et [REX] |
| Réinitialisation de compte | Un REX (2026-07) signale un compte « réinitialisé périodiquement, nouvel account ID, tout effacé, sans prévenir ». La cause n'est pas identifiée. | [?] |

---

## 2. IAM et conséquences pour le CI/CD

### Ce qui est documenté [DOC]
- On ne peut créer **ni utilisateurs, ni groupes, ni rôles**, sauf des *service-linked roles*.
- `LabRole` et `LabInstanceProfile` sont pré-créés. `LabRole` a des permissions *« very similar to the permissions you have as a user in the console »*.
- **ECR :** `LabRole` n'a qu'un accès en **lecture seule**. L'écriture est réservée à l'utilisateur de la console.
- **EKS :** *« This service can assume the IAM Roles having identifier LabEksClusterRole created for Cluster and Node »*.
- **ECS :** utiliser `LabRole` comme *task role* et comme *execution role*. Pour un cluster ECS sur EC2, créer d'abord l'ASG dans la console EC2.

### Ce qui est observé [REX]

**Sonde API du 2026-09-06 et du 2026-09-07 :**
- Refusé : `iam:CreateRole`.
- Refusé : **`iam:CreateOpenIDConnectProvider`**, avec plus de 15 dépôts concordants entre 2025-12 et 2026-09.
- Refusé : `iam:AttachRolePolicy`, y compris sur les rôles du lab.
- Refusé : `servicequotas:GetServiceQuota`.
- Refusé : `iam:GetRole` **sur `voclabs`**, par un deny explicite de la policy `Pvoclabs2`. Ce refus casse le module communautaire `terraform-aws-modules/eks`, qui lit `aws_iam_session_context`. D'autres équipes l'utilisent pourtant avec `create_iam_role=false`, d'où un statut [?].

**Autres refus :**
- `iam:Update*` sur `LabRole` est refusé par un deny explicite : impossible d'ajouter un *trust* (REX 2026-01).

**Rôles EKS pré-créés (2026-09) :**
- `…-LabEksClusterRole-…` fait confiance à `eks.amazonaws.com`. Il porte les policies EKS Cluster, Networking, **Compute, BlockStorage et LoadBalancing**, c'est-à-dire celles d'EKS Auto Mode.
- `…-LabEksNodeRole-…` fait confiance à `ec2.amazonaws.com`. Il porte les policies CNI, WorkerNode et ECR en lecture.
- **Ces rôles sont absents de certains comptes.** Un REX du 2026-09-15 a dû utiliser `LabRole` pour le cluster et les nœuds, avec succès.
- Le préfixe aléatoire change d'un compte à l'autre et à chaque réinitialisation. Il ne faut donc jamais coder l'ARN en dur : utiliser `data "aws_iam_roles"` avec `name_regex`.

**Contenu de `LabRole` :**
- *Trust policy* (relevée dans un tfstate de 2026-07) : apigateway, application-autoscaling, autoscaling, cloudformation, codedeploy, cognito-idp, ec2, ecs, **ecs-tasks**, **eks**, **eks-fargate-pods**, elasticloadbalancing, events, lambda, logs, pipes, rds, s3, scheduler, secretsmanager, sns, sqs, ssm, states, entre autres.
- **Principaux absents :** `pods.eks.amazonaws.com` (donc pas d'EKS Pod Identity), `codebuild`, `codepipeline`, `apprunner`, `xray`, `grafana` et `aps`.
- Policies attachées (2026-09) : `VocLabPolicy1/2/3`, `AmazonSSMManagedInstanceCore`, `AmazonEKSClusterPolicy`, `AmazonEKSWorkerNodePolicy` et `AmazonEC2ContainerRegistryReadOnly`.

### Conséquences pour le CI/CD
- **Fédération OIDC GitHub Actions ou GitLab CI vers AWS : impossible**, puisqu'il faut créer un fournisseur d'identité et un rôle. [REX], très convergent.
- **Contournement le plus répandu.** Stocker `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` et `AWS_SESSION_TOKEN` en secrets du dépôt, puis les **rafraîchir à chaque session**, par exemple avec un script `gh secret set` au démarrage du lab. Au-delà de 4 h, les jobs échouent avec `ExpiredToken` ou `voc-cancel-cred`. [REX] 2026-09, plusieurs dépôts.
- **Variante « CI builds only ».** La CI construit et pousse les images, et le déploiement est lancé localement (ou manuellement) avec des identifiants frais. Certaines équipes poussent vers **GHCR** pour ne pas dépendre des identifiants AWS dans la CI. [REX]
- **Modèle « pull ».** ArgoCD ou Flux tourne dans le cluster, sur EKS ou k3s, et la CI ne touche jamais AWS. Ce montage est documenté en REX sur k3s dans le Learner Lab. Sur EKS, l'extraction depuis ECR passe par le rôle du nœud, en lecture seule. [REX]
- **Runner auto-hébergé sur EC2 avec `LabInstanceProfile`.** Les identifiants seraient renouvelés automatiquement par IMDS, mais `LabRole` est **en lecture seule sur ECR** : il faudrait pousser vers GHCR. Le runner s'arrête aussi en fin de session. Aucun REX trouvé. [?]
- **EKS : pas d'IRSA.** Les pods qui appellent des API AWS ont deux options :
  - l'**identité du nœud via IMDS** : si le rôle de nœud est `LabRole`, les pods obtiennent ses permissions larges. Si c'est `LabEksNodeRole`, ils n'ont que CNI et ECR, sans S3 ni Secrets Manager ;
  - des **identifiants de session injectés dans des Secrets Kubernetes**, qui expirent au bout de 4 h.
- **ECS : `LabRole` sert nativement de *task role*.** Les présignatures S3 pour les photos, SQS et Secrets Manager fonctionnent sans bricolage. C'est un avantage structurel d'ECS dans ce lab.

---

## 3. Disponibilité des services

Statuts : ✅ disponible, ⚠️ disponible avec limites ou contournements, ❌ indisponible, ❓ incertain.

| Service | Statut | Détails | Source |
|---|---|---|---|
| **EKS** | ⚠️ | Listé, avec instances nano à large. Création réussie (2026-07 à 2026-09) du cluster, du managed node group (AL2023, t3.small ou t3.medium) et des add-ons vpc-cni, kube-proxy, coredns et metrics-server. Versions proposées : 1.31 à 1.36 (2026-09). Accès via access entries, avec `bootstrap_cluster_creator_admin_permissions`. Détails et limites ci-dessous. | [DOC] et [REX] |
| ↳ eksctl | ⚠️ | Fonctionne avec un `ClusterConfig` qui fixe `iam.withOIDC: false`, `serviceRoleARN` (LabRole ou LabEksClusterRole) et `instanceRoleARN` sur le node group. En mode par défaut, il échoue sur `CreateRole`. | [REX] 2026-07 |
| ↳ OIDC / IRSA / Pod Identity | ❌ | Impossible de créer l'OIDC provider. Pas de rôle faisant confiance à `pods.eks.amazonaws.com`. | [REX] |
| ↳ AWS Load Balancer Controller | ⚠️❓ | Pas d'IRSA. Les REX contournent de deux façons : un Secret contenant les identifiants de session (expiration 4 h), ou le rôle de nœud via IMDS (hop limit 2). L'alternative adoptée par plusieurs équipes : **NLB ou ALB géré par Terraform**, combiné à ingress-nginx en NodePort. Elle évite aussi les load balancers orphelins au `destroy`. | [REX] |
| ↳ EBS CSI | ⚠️ | Désactivé dans les REX faute d'IRSA. Recommandation : pas de volumes persistants, et un état applicatif stocké dans RDS et S3. | [REX] |
| ↳ Profils Fargate | ❓ | `LabRole` fait confiance à `eks-fargate-pods`, ce qui rend le montage plausible, mais **aucun REX ne montre un succès**. Fargate exige aussi le LB Controller avec des cibles IP, qui demande un IRSA indisponible. Difficilement exploitable. | Déduit |
| ↳ EKS Auto Mode | ❓ | `LabEksClusterRole` porte les policies nécessaires, et un REX (2026-07) l'a utilisé via la console. Risque : Karpenter peut choisir des instances de taille supérieure à large, qui seront refusées ; il faut contraindre le NodePool. Surcoût d'environ 12 % sur l'EC2. Une équipe l'a écarté pour garder des coûts prévisibles. | [REX] isolé |
| ↳ Pièges EKS | ⚠️ | us-east-1e ne peut pas héberger le control plane : épingler 1a, 1b et 1c. Densité de pods sans *prefix delegation* : **t3.small = 11 pods**, **t3.medium = 17 pods**. Durées observées : cluster en 15 à 20 min, node group en 8 à 12 min, destruction en environ 10 min. | [REX] |
| **ECS** | ✅ | Listé, sur EC2 (créer l'ASG d'abord) et sur Fargate. Task role et execution role = `LabRole`. **Autoscaling de service par target tracking prouvé**, via le rôle lié `AWSServiceRoleForApplicationAutoScaling_ECSService`. Container Insights fonctionne. | [DOC] et [REX] tfstate 2026-04 |
| ↳ Cloud Map / Service Connect | ❌ | `servicediscovery:*` refusé. Les équipes passent par un ALB ou NLB interne, ou par l'API ECS. | [REX] ×5, 2025-12 à 2026-09 |
| **Fargate** | ✅ | Listé ; peut assumer `LabRole`. | [DOC] |
| **ECR** | ✅ | `LabRole` en lecture seule ; écriture avec les identifiants utilisateur. | [DOC] |
| **App Runner** | ❓/❌ | Non listé. `LabRole` ne fait pas confiance aux principaux App Runner. Un support de cours (date inconnue) l'utilise. Sandbox *Cloud Developing* : absent. | [DOC] et [?] |
| **Lambda** | ⚠️ | Listé. **10 environnements d'exécution concurrents au maximum.** | [DOC] |
| **API Gateway** | ✅ | Listé. HTTP API avec VPC Link vers un NLB utilisé en REX (2026-09). L'endpoint par défaut est en HTTPS. | [DOC] et [REX] |
| **ALB / NLB** | ✅ | Listés. Création réussie par Terraform. Un REX affirmant « Classic ELB seulement » confond probablement avec le `Service type=LoadBalancer` *in-tree*. | [DOC] et [REX] |
| **ACM** | ⚠️ | Listé. **`RequestCertificate` validé par DNS jusqu'à l'état `ISSUED`** dans des tfstates (2026-04). `ImportCertificate` : refusé dans un compte (2026-02), réussi dans un autre (2026-09). | [DOC] et [REX] |
| **Route 53** | ⚠️ | *« You cannot register a domain »* [DOC]. **Création de zones hébergées réussie** dans plusieurs tfstates (2026-04) ; un REX affirme le contraire. Un domaine externe est nécessaire (registrar ou sous-domaine délégué de l'école). | [DOC] et [REX] |
| **CloudFront** | ❌ | Présent dans la liste de 2021, **absent de celle de 2025**. `cloudfront:*` refusé (REX 2026-06 et 2026-09). | [REX] |
| **RDS PostgreSQL / Aurora** | ⚠️ | Classes nano à medium, burstable. gp2 jusqu'à 100 Go (gp3 accepté en REX). On-Demand. Pas d'Enhanced Monitoring. Aurora en mode *provisioned* uniquement. **Multi-AZ :** « not supported » en 2021, non mentionné en 2025, création Multi-AZ refusée dans un REX (2026-02) : considérer comme indisponible [?]. **Read replica réussie** (REX 2026-06). db.t3.micro PostgreSQL réussi (REX 2026-04 et 2026-09). Création en environ 10 à 16 min. | [DOC] et [REX] |
| **DynamoDB** | ✅ | Listé. | [DOC] |
| **S3** | ⚠️ | Listé. **Une SCP de l'organisation refuse `s3:GetBucketObjectLockConfiguration`.** Le provider Terraform `aws_s3_bucket` et Pulumi échouent donc au `plan` ou au `refresh`. Contournement : créer le bucket en CLI (`terraform_data` + `local-exec`), le lire en `data`, puis le configurer avec les ressources `aws_s3_bucket_*`. | [REX], plus de 10 dépôts, 2026-02 à 2026-10 |
| **SQS / SNS** | ✅ | Listés. | [DOC] |
| **EventBridge** | ✅ | Listé, Pipes via `LabRole`. | [DOC] |
| **Amazon MQ** | ❌ | Non listé. `mq:CreateBroker` refusé. | [REX] 2026-04 |
| **MSK** | ❌ | Non listé. MSK, MSK Serverless et MSK Connect refusés. *« Not available in the Learner Lab »* selon le cours UQ. | [REX] 2026-06 et 2026-07 |
| **ElastiCache** | ✅ | Listé. Redis cache.t3.micro réussi. | [DOC] et [REX] |
| **Cognito** | ⚠️ | **Absent du Readme 2025**, présent dans la liste Foundation de 2021. **User Pool, client et domaine créés** (tfstate 2026-04, plusieurs dépôts). Identity Pools inutilisables faute de rôles. | [REX] |
| **Secrets Manager** | ✅ | Listé. | [DOC] et [REX] |
| **SSM Parameter Store** | ✅ | SSM listé. SecureString utilisé. | [DOC] et [REX] |
| **CloudWatch** | ⚠️ | Listé : logs, métriques et alarmes. Container Insights fonctionne sur ECS (simple réglage du cluster). Sur EKS, l'agent a besoin de permissions sur le nœud ; l'attachement de `CloudWatchAgentServerPolicy` est refusé, ce qui suppose d'utiliser `LabRole` comme rôle de nœud [?]. Un REX signale `GetDashboard` refusé [?]. | [DOC] et [REX] |
| **X-Ray** | ❓ | Absent du Readme 2025. Les REX se contredisent. | [?] |
| **ADOT** | ❓ | Le collecteur est un simple logiciel. L'export vers CloudWatch ou X-Ray dépend des permissions. L'add-on EKS demande une identité IAM, a priori sans IRSA. | [?] |
| **Managed Prometheus / Grafana** | ❌❓ | Non listés, aucun REX. Amazon Managed Grafana exige IAM Identity Center et des rôles. Les REX hébergent eux-mêmes Prometheus, Grafana, Loki et Tempo, sur ECS ou dans le cluster. | [?] |
| **CloudFormation** | ✅ | Listé ; assume `LabRole`. Le Readme le recommande pour créer et détruire l'infrastructure par stack. | [DOC] |
| **CDK** | ⚠️ | `cdk bootstrap` standard échoue (`CreateRole`). Contournements : `CliCredentialsStackSynthesizer` ou `LegacyStackSynthesizer` sans bootstrap, ou template de bootstrap expurgé de l'IAM. Le bucket `cdk-hnb659fds-*` a été bloqué dans un REX : créer son propre bucket d'assets. | [REX] 2026-07 |
| **Terraform** | ⚠️ | Fonctionne avec les identifiants de session. Pièges : SCP S3 (voir ci-dessus) ; module EKS communautaire et `GetRole` sur `voclabs` ; expiration en plein `apply` qui laisse le state verrouillé (ne pas lancer d'`apply` EKS après la 3e heure) ; ARN de rôles codés en dur. Verrouillage du state : `use_lockfile` sur S3. | [REX] |
| **CodePipeline / CodeBuild** | ❌ | Non listés. `LabRole` n'est pas assumable par ces services. REX : bloqués, avec un deny explicite sur CodeBuild. CodeDeploy et CodeCommit sont listés. | [DOC] et [REX] |
| **WAF** | ❓ | Listé, mais `wafv2:CreateWebACL` refusé dans un REX (2026-02). | [DOC] et [REX] |
| **NAT Gateway** | ✅ | Créé en REX. C'est un coût récurrent. Alternative REX : instance NAT, ou sous-réseaux publics avec security groups. | [REX] |
| **EC2 / ASG** | ⚠️ | Voir §1 : nano à large, 9 instances et 32 vCPU au maximum par région. | [DOC] |

---

## 4. Coûts indicatifs en us-east-1

Prix horaires On-Demand. Sources : grille AWS (*price list* publiée le 2026-09-11 pour MQ et MSK, le 2026-09-28 pour EKS) et pages de tarification consultées le 2026-10-09.

| Ressource | Prix | 4 h de démo | Laissé allumé 24 h |
|---|---|---|---|
| Control plane EKS, *standard support* | 0,10 $/h | 0,40 $ | 2,40 $ |
| Control plane EKS, *extended support* (**1.31 à 1.33 en oct. 2026**) | 0,60 $/h | 2,40 $ | 14,40 $ |
| Nœud t3.medium (2 vCPU, 4 Gio) | 0,0416 $/h | 0,17 $ par nœud | 1,00 $ par nœud |
| Nœud t3.small | environ 0,0208 $/h | 0,08 $ par nœud | 0,50 $ par nœud |
| Fargate Linux/x86 | 0,04048 $/vCPU-h et 0,004445 $/Go-h | tâche 0,5 vCPU / 1 Go ≈ 0,025 $/h, soit 0,10 $ | 0,59 $ par tâche |
| Fargate Linux/ARM | 0,03238 $/vCPU-h et 0,00356 $/Go-h | environ 20 % moins cher | – |
| NAT Gateway (par passerelle) | 0,045 $/h et 0,045 $/Go traité | 0,18 $ | 1,08 $ |
| ALB | 0,0225 $/h et 0,008 $/LCU-h | environ 0,12 $ | environ 0,75 $ |
| NLB | 0,0225 $/h et 0,006 $/NLCU-h | environ 0,10 $ | environ 0,65 $ |
| RDS db.t3.micro PostgreSQL, Single-AZ | environ 0,018 $/h, plus le stockage | 0,07 $ | 0,43 $ |
| Amazon MQ mq.t3.micro (RabbitMQ ou ActiveMQ, instance unique) | 0,02704 $/h | 0,11 $ | 0,65 $ (**service indisponible**) |
| Broker MSK kafka.t3.small | 0,0456 $/h par broker, au moins 2 brokers | 0,36 $ | 2,19 $ (**indisponible**) |
| MSK Serverless | 0,75 $/h par cluster et 0,0015 $/h par partition | 3,00 $ | 18 $ (**indisponible**) |
| IPv4 publique, utilisée ou inactive | 0,005 $/h par adresse | 0,02 $ | 0,12 $ par adresse |

Le prix RDS (0,018 $/h) et le prix t3.small proviennent d'agrégateurs (vantage.sh) : à vérifier dans le calculateur AWS.

**Ordres de grandeur d'une session de 4 h :**
- **Stack EKS minimale** : cluster, 2 × t3.medium, ALB ou NLB, db.t3.micro et un NAT. Environ 0,30 $/h, soit **environ 1,2 $ par session**. Laissé allumé : environ **7 $ par jour**. Un REX du 2026-09 donne environ 5 $ par jour pour EKS, NAT, LB et RDS.
- **Stack ECS Fargate** : environ 6 tâches de 0,5 vCPU / 1 Go, ALB, db.t3.micro et NAT. Environ 0,25 $/h, soit **environ 1 $ par session**. Pendant un pic d'autoscaling, compter environ 0,025 $/h par tâche supplémentaire.
- **k3s sur EC2** : 1 à 3 instances t3.medium avec Elastic IP, sans frais de control plane. Environ 0,05 à 0,15 $/h. Ce n'est pas un environnement managé.
- **Coûts fixes** s'il reste des ressources entre les sessions : stockage RDS (environ 0,115 $/Go-mois en gp2), ECR (0,10 $/Go-mois), Secrets Manager (0,40 $ par secret et par mois), zone Route 53 (0,50 $ par mois). Ce sont des tarifs publics usuels, non revérifiés ici.

**Pièges de coût fréquents :**
1. **Croire que la fin de session arrête tout.** Seules les instances EC2 sont arrêtées [DOC]. EKS, NAT, ELB, Fargate et parfois RDS continuent.
2. **Redémarrage automatique des instances EC2** à chaque nouvelle session [DOC]. Des nœuds oubliés se remettent à consommer. **L'ASG d'un node group remplace les nœuds arrêtés par le lab** [REX].
3. **Versions EKS en *extended support***, soit six fois le prix du control plane. Créer en 1.35 ou plus ; la 1.34 sort du *standard support* le 2026-12-02.
4. **Load balancers créés par Kubernetes** (`Service type=LoadBalancer`, Ingress) : ils sont absents du state Terraform, donc le `destroy` échoue ou les laisse en place, et ils continuent de facturer [REX].
5. **NAT Gateway par AZ** pour la haute disponibilité : le coût est multiplié par le nombre d'AZ. Elastic IP orphelines après un `destroy` interrompu [REX].
6. **Budget affiché en retard de 8 à 12 h** [DOC] : on peut dépasser sans le voir. Créer une alerte AWS Budgets reste à vérifier dans le lab [?].
7. **RDS arrêtée** : redémarrage automatique au bout de 7 jours [DOC].
8. **Logs CloudWatch** à rétention infinie par défaut, et métriques Container Insights.
9. **Volumes EBS** des nœuds et snapshots RDS finaux : utiliser `skip_final_snapshot`.

---

## 5. Retours d'expérience sur Kubernetes et ECS dans le Learner Lab

### EKS (nombreux dépôts de 2026)
- **Ça fonctionne** pour un cluster, un managed node group t3.small ou t3.medium, et le HPA avec metrics-server. Des démonstrations de montée en charge par HPA sont rapportées.
- **Rôles.** Retrouver dynamiquement `LabEksClusterRole` ou `LabEksNodeRole`, et se rabattre sur `LabRole` s'ils sont absents. Activer `authentication_mode = API_AND_CONFIG_MAP` avec des access entries.
- **Pas d'IRSA**, ce qui entraîne en cascade : pas d'External Secrets Operator (`InvalidProviderConfig`), pas d'EBS CSI, et un LB Controller ou un agent CloudWatch qui doivent passer par le nœud ou des identifiants statiques. Solutions observées :
  - un script de déploiement local qui lit Secrets Manager avec les identifiants de session et crée les Secrets Kubernetes ;
  - un NLB géré par Terraform avec ingress-nginx en NodePort.
- **Module Terraform EKS communautaire** : il échoue à cause d'`iam:GetRole` sur `voclabs` (REX 2026-09). Solution : écrire directement `aws_eks_cluster` et `aws_eks_node_group`.
- **Durées et session de 4 h** : il faut 40 min pour tout monter depuis zéro (réseau, EKS, RDS, Lambda, API Gateway) (REX 2026-09). Planifier le `apply` en début de session et prévoir un `teardown` vérifié.
- **Nœuds `NotReady` après le renouvellement de session**, par expiration ou arrêt par le lab. Solution : redimensionner ou recréer le node group.
- **Discipline recommandée par plusieurs équipes** : détruire le cluster à chaque session et le recréer par IaC. Les ressources peu coûteuses (ECR, buckets, secrets, zone DNS et certificat) vont dans un *layer* persistant.

### k3s sur EC2
- Choix fréquent pour économiser les 0,10 $/h du control plane EKS (ADR étudiants, 2026). Une ou plusieurs instances EC2, Elastic IP, `LabInstanceProfile`, souvent accompagnées d'ArgoCD en GitOps.
- Les instances sont arrêtées en fin de session et redémarrées à la suivante : le cluster survit sans recréation. Les pods doivent supporter le redémarrage.
- Limites : **ce n'est pas un environnement managé**, et il n'y a pas d'autoscaling de nœuds natif (HPA uniquement, ou ASG avec jonction k3s à construire). k3s embarque Traefik : le désactiver pour retrouver le même Ingress qu'en local.

### ECS
- **Fargate, ALB, autoscaling en target tracking, ElastiCache, DynamoDB et RDS** déployés par Terraform ou CDK (REX 2026-04 à 2026-07).
- `LabRole` en *task role* : l'accès aux services AWS depuis le code fonctionne sans contournement.
- Limites : **pas de Cloud Map ni de Service Connect**, donc la découverte de services passe par un ALB ou NLB interne avec des variables d'environnement. FireLens est déclaré indisponible par un REX [?]. « Event history » de la console ECS exige Container Insights.

---

## 6. Correspondance avec les exigences du POC (factuel, non prescriptif)

| Exigence | EKS | ECS Fargate | k3s sur EC2 |
|---|---|---|---|
| Environnement managé | Oui (control plane managé) | Oui (serverless) | Non |
| Autoscaling | HPA OK. Nœuds : manuel ou ASG ; Cluster Autoscaler via le rôle du nœud [?] ; Auto Mode [?]. | Autoscaling de service prouvé | HPA uniquement |
| Disponibilité | Multi-AZ possible pour les nœuds. RDS sans Multi-AZ [?], read replica OK. | Tâches Multi-AZ, RDS idem | Faible |
| HTTPS | ACM et Route 53 avec domaine externe sur NLB ou ALB, ou API Gateway, ou cert-manager et LE (limite de LE : 5 certificats identiques par semaine, problématique avec des recréations quotidiennes) | ACM sur ALB, ou API Gateway | LE avec Caddy, sslip.io ou DuckDNS |
| CI/CD depuis une forge | Identifiants de session en secrets (rafraîchis toutes les 4 h), ou GitOps | Identifiants de session en secrets | GitOps, ou SSH et identifiants |
| Accès des services à S3, SQS et Secrets | Contournement (IMDS ou identifiants statiques) | Natif (`LabRole` en task role) | IMDS (`LabInstanceProfile`) |
| Découverte de services | CoreDNS natif | Cloud Map ❌, passer par ALB ou NLB interne | CoreDNS natif |

---

## 7. Inconnues à lever par une sonde au premier jour

Toutes ces vérifications se font sur le compte réel, car les écarts entre comptes sont documentés.

```bash
aws sts get-caller-identity
aws iam list-roles --query "Roles[?contains(RoleName,'Lab')].RoleName"   # LabEks* présents ?
aws eks describe-cluster-versions --query "clusterVersions[].[clusterVersion,status]"
aws route53 create-hosted-zone --name probe-$RANDOM.example.com --caller-reference $RANDOM   # puis supprimer
aws acm request-certificate --domain-name probe.example.com --validation-method DNS          # puis supprimer
aws cognito-idp create-user-pool --pool-name probe                                           # puis supprimer
aws servicediscovery list-namespaces ; aws xray get-sampling-rules ; aws amp list-workspaces
aws mq list-brokers ; aws kafka list-clusters-v2 ; aws cloudfront list-distributions
aws wafv2 list-web-acls --scope REGIONAL ; aws apprunner list-services ; aws codebuild list-projects
aws rds create-db-instance ... --multi-az     # Multi-AZ autorisé ?
aws s3api get-object-lock-configuration --bucket <bucket>   # SCP présente ?
```

Chaque réponse `AccessDenied` doit mentionner une **policy d'identité ou une SCP**. Si elle cite **`voc-cancel-cred`**, la session est expirée : il ne s'agit pas d'une interdiction.

### Constats sur le compte du projet (sonde XP00 du 2026-10-26)

Détail et commandes : [XP00](../experimentations/XP00-sonde-compte.md).

| Point | Constat |
|---|---|
| Budget | 50 $, région us-east-1 |
| IAM | `LabRole` et `LabInstanceProfile` seulement, sans `LabEks*`. `iam:GetRole` est autorisé. La confiance de `LabRole` couvre `eks`, `eks-fargate-pods`, `ecs-tasks`, `ec2`, `scheduler`, `rds` et `elasticloadbalancing` |
| EKS | 1.34 à 1.37 en support standard ; add-ons gérés, dont `metrics-server` |
| ECS Fargate | Tâche lancée et terminée avec le code 0 sous `LabRole` |
| Quotas | `GetServiceQuota` refusé ; `max-instances` = 20 |
| Route 53, ACM | Zone publique créée ; certificat validé par DNS en moins de 5 min |
| EventBridge Scheduler | `iam:PassRole` de `LabRole` autorisé ; suppression programmée exécutée en session |
| AWS Budgets | Création de budget et d'alertes autorisée |
| RDS, Secrets Manager | PostgreSQL 18.3 par défaut, chiffrement et Multi-AZ proposés ; suppression de secret sans délai autorisée |
| S3 | SCP présente (Object Lock refusé) ; blocage public et politique de bucket autorisés |
| CloudWatch Logs, Cognito | Logs Insights et création de pool autorisés |

---

## Sources (datées)

**Documentation officielle et universités**
- Readme Learner Lab (« Instructions last updated: 2025-06-24 »), copie fidèle dans un dépôt étudiant (commit 2026-08-17) : https://github.com/BobTheBot988/cloud_project/blob/HEAD/Plans/Learner%20Lab%20instruction.md. Autres copies concordantes : https://github.com/Akawatmor/KPS-Enterprise/blob/HEAD/document/phase1/learnerlab-limit.txt (2026-04-27) et https://github.com/arnold6444/WaferGuard/blob/HEAD/docs/AWS_Academy_Guide.md (2026-05-30).
- « AWS Academy Learner Lab – Foundation Services » (© 2021, liste sans ECS ni EKS) : https://cyberlab.pacific.edu/courses/comp175/resources/aws-academy/AWS_Academy_Learner_Lab-Foundational_Services.pdf
- Handout de l'Université du Queensland, 2026-02-19 (4 h, 50 $, budget non affiché en temps réel) : https://csse6400.uqcloud.net/handouts/learner-lab.pdf
- UCSC (50 $, 6 mois au maximum) : https://its.ucsc.edu/services/teaching-and-learning/learning-management/aws-academy

**Calendrier et tarifs AWS** (consultés le 2026-10-09)
- Calendrier des versions EKS : https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html
- Tarifs EKS : https://aws.amazon.com/eks/pricing/
- Tarifs Fargate : https://aws.amazon.com/fargate/pricing/
- Tarifs ELB : https://aws.amazon.com/elasticloadbalancing/pricing/
- Tarifs VPC (NAT et IPv4) : https://aws.amazon.com/vpc/pricing/
- Price List API : `https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/{AmazonMQ,AmazonMSK,AmazonEKS}/current/us-east-1/index.json`
- Agrégateur vantage.sh : t3.medium (https://instances.vantage.sh/aws/ec2/t3.medium) et db.t3.micro (https://instances.vantage.sh/aws/rds/db.t3.micro)

**Retours d'expérience : IAM, EKS et coûts**
- Sondes d'API du Learner Lab, 2026-09-06 et 2026-09-07 (IAM, EKS, SCP S3, CloudFront, rôles LabEks*) : https://github.com/Vnntnn/Cloud_Computing2026-Project/blob/HEAD/docs/lab-probe-2026-09-06.txt, https://github.com/Vnntnn/Cloud_Computing2026-Project/blob/HEAD/docs/lab-probe-2026-09-07.txt et https://github.com/Vnntnn/Cloud_Computing2026-Project/blob/HEAD/docs/PROJECT-KNOWLEDGE-BASE.md
- `LabEksClusterRole` absent, `LabRole` utilisé pour EKS ; t3.small limité à 11 pods ; us-east-1e (2026-09-15) : https://github.com/gabrielMauad/oficina-mecanica-infra-k8s/blob/HEAD/variables.tf
- *Trust policy* de `LabRole` relevée dans un tfstate EKS (2026-07-14) ; README avec nœuds `NotReady` : https://github.com/Orco-morado/proyecto_DevOps_examen
- Chaîne IRSA bloquée, deny `iam:Update*` sur `LabRole` (2026-01-15) : https://github.com/brianmonteiro54/togglemaster-microservices/blob/HEAD/kubernet/README.MD
- eksctl avec `LabRole` (2026-07-13) : https://github.com/francoabl/EV2DevOPS/blob/HEAD/eks/cluster-config.yaml
- Container Insights, EBS CSI et IRSA sur EKS (2026-07-08) : https://github.com/JONAHBRUZZI/EF/blob/HEAD/tf-aws-eks-main/README.md
- LB Controller avec identifiants en Secret ; compte « réinitialisé » (2026-07-27) : https://github.com/13SOAT-andromeda/iac-video-processor-infra/blob/HEAD/docs/RUNBOOK.md
- EKS Auto Mode dans le Learner Lab (2026-07-23) : https://github.com/piymundada/intelligent-autoscaling-k8s/blob/HEAD/k8s/eks/EKS_SETUP.md
- Environ 5 $/jour, 40 min de montage depuis zéro, `voc-cancel-cred` (2026-09-15) : https://github.com/tech-challenge-grupo-160/tech-challenge-oficina-mecanica/blob/HEAD/docs/CICLO-DE-VIDA.md

**Retours d'expérience : services managés, réseau et HTTPS**
- tfstate avec Route 53, ACM `ISSUED`, Cognito et RDS PostgreSQL sous `voclabs` (2026-04-27) : https://github.com/BialySztorm/PWR-WIT-Inf-2ndDegree/blob/HEAD/PiWSwC/Project1/infra/terraform.tfstate
- Refus de `CreateDBInstance` (Multi-AZ), `ImportCertificate`, `CreateWebACL` et de la SCP S3 (2026-02-04) : https://github.com/issye/CCS6344-Assignment2-Group01/blob/HEAD/Limitations.md
- Let's Encrypt via DuckDNS importé dans ACM, ALB et ASG (2026-09-21) : https://github.com/stephy0410/aws-high-availability-lab
- tfstate avec ECS Fargate, autoscaling et ElastiCache (2026-04-18) : https://github.com/vanessazlh/Distributed_Real_Time_Bidding_Platform/blob/HEAD/infra/terraform.tfstate
- ADR « Learner Lab profile » : refus de `cloudfront:*` et `servicediscovery:*`, ElastiCache et read replica OK (2026-06-04) : https://github.com/jordanraji/TicketTailor/blob/HEAD/model/adrs/0017-learner-lab-deployment-profile.md
- Cloud Map bloqué : https://github.com/yufei-ilariahuang/Distributed-Caching-Optimization/blob/HEAD/aws-terraform/LEARNER_LAB_NOTES.md (2025-12) et https://github.com/Cloud-Native-Convivo/terraform-cloud/blob/HEAD/nlb.tf (2026-09)
- `mq:CreateBroker` refusé (2026-04-22) : https://github.com/Rui091/iot-rabbitmq/blob/HEAD/Proyecto_Explicacion_Completa.md
- MSK refusé, rien au-delà de la taille large (2026-07-29) : https://github.com/helvetios16/SimulateFlink/blob/HEAD/docs/REPRODUCIR.md
- CodePipeline (2026-07-06) : https://github.com/aloysiowinter/aws_coffeecode/blob/HEAD/README.md
- CodeBuild avec deny explicite (2025-06-23) : https://github.com/Grambot-ops/REACT-SEC-OPS/blob/HEAD/README.md
- Cognito Identity Pool inutilisable (2026-06-04) : https://github.com/janechiangccy/ergoguard/blob/HEAD/ergoguard-ui/README.md
- X-Ray indisponible selon un REX (2025-12-02) : https://github.com/edoturb/cnn-chile-infra/blob/HEAD/DEVELOPER-GUIDE.md
- Sandbox *Cloud Developing* sans ECS, EKS, Fargate ni App Runner (2026-09-23) : https://github.com/Fabiobazacas/aie-cloud/blob/HEAD/aulas/03-serverless-containers-aws/lab/guia-lab.md

**Retours d'expérience : outillage IaC et CI/CD**
- CDK avec `CliCredentialsStackSynthesizer`, ECS, Grafana et Prometheus auto-hébergés, pas de Cloud Map (2026-07-15) : https://github.com/FIAP-13SOAT/media-processor-cdk
- Hack CDK (2022) : https://github.com/wongcyrus/aws-cdk-hack-for-aws-academy-learner-lab
- SCP S3 et Terraform (2026-09-30) : https://github.com/CaiqueSouzaa/prova-primeiro-bimestre-devops/blob/HEAD/infra/README.md
- `voc-cancel-cred`, OIDC refusé, script de rafraîchissement des secrets GitHub (2026-09-23) : https://github.com/rayenmabrouk/cloudpulse/blob/HEAD/docs/troubleshooting.md
- `voc-cancel-cred` et secrets GitHub Actions (2026-09-11) : https://github.com/GrupoPosTech-FIAP/oficina-app-api/blob/HEAD/docs/cicd-github-actions.md
- Suspension `SUSPENDED9` (2026-09) : https://github.com/Julian-Rincon/nlp-pipeline-api/blob/HEAD/STATUS.md
