# XP00 — Sonde du compte Learner Lab

| Champ | Valeur |
|---|---|
| Décision testée | Prérequis de D03, D05, D06, D08 (`Candidate`) ; garde-fous de D11 (G1, G6) ; incertitudes de D09 |
| Date prévue | 2026-10-27, session de configuration AWS |
| Date d'exécution | 2026-10-26, de 13 h 30 à 14 h 00 UTC ; S8b hors session ensuite |
| Exécutée par | Martin |
| Scripts | `sandbox/XP00-sonde-compte/` (reprise des commandes ci-dessous, écrite après la session) |
| Référence | [D14](../decisions/D14-experimentations.md) § « Sonde du compte » (S1 à S13) |

## Objectif et critères de succès

Objectif : établir, sur le compte réel de Martin, ce que `LabRole` et la session autorisent, avant toute expérimentation cloud. Chaque réponse oriente une décision ou un repli déjà écrit.

Hypothèse : le compte est un Learner Lab « complet » en us-east-1 ; EKS, ECS Fargate, Route 53, ACM et EventBridge Scheduler sont utilisables ; `GetServiceQuota` est refusé ; une SCP bloque la lecture d'Object Lock sur S3.

Critères fixés avant l'exécution. « Bloquant » = empêche les XP cloud qui en dépendent.

| # | Mesure | Seuil de succès | Bloquant | Si échec | Mesure obtenue | Atteint |
|---|---|---|---|---|---|---|
| S1 | Budget total, consommé, région | Budget lu (50 ou 100 $) ; région us-east-1 | Non | 50 $ : sessions en allumage à la demande (D11), réserve de rejeu réduite | Budget 50 $ (lu par Martin) ; us-east-1 | Oui |
| S2 | Identité, rôles `Lab*`, `iam:GetRole` | Principal `assumed-role/voclabs/…` ; `LabRole` présent | **Oui** (XP01 à XP03) | Sans `LabRole` : aucune XP cloud, Martin alerte l'école. `LabEks*` absents : `LabRole` pour cluster et nœuds (D03) | `assumed-role/voclabs/user…` ; `LabRole`, `LabInstanceProfile` ; pas de `LabEks*` ; `GetRole` autorisé, confiance accordée à `eks`, `eks-fargate-pods`, `ecs-tasks`, `ec2`, `scheduler` | Oui |
| S3 | Versions EKS, add-ons, AMI | 1.35 ou plus en `STANDARD_SUPPORT` | **Oui** (XP01 EKS) | Version absente : coût du support étendu intégré (+0,50 $/h), X13 revu. EKS absent : bascule vers ECS (D03) | 1.34 à 1.37 en `STANDARD_SUPPORT` (1.35 jusqu'au 2027-03-27) ; add-ons et AMI 1.35 relevés | Oui |
| S4 | ECS Fargate : cluster et tâche 0,25 vCPU | Tâche `STOPPED`, code de sortie 0 | **Oui** (comparatif) | XP01 réduite à EKS ; rejet de Fargate documentaire seulement | `STOPPED EssentialContainerExited 0` (2e essai) | Oui |
| S5 | Quotas Fargate et EC2 | Valeur lue, ou refus constaté | Non | Plafond prudent : 4 vCPU Fargate, 9 instances | Deux `AccessDenied` sur `GetServiceQuota` ; `max-instances` = 20 | Oui (refus constaté) |
| S6 | Zone Route 53 `collector.qvl-project.com`, délégation Cloudflare (XE1) | Zone créée ; `NS` résolus par 1.1.1.1 et 8.8.8.8 | Non | Repli D2 : enregistrements posés directement dans Cloudflare | Zone créée ; 4 `NS` posés dans Cloudflare par l'API ; délégation effective (certificat validé en moins de 5 min) | Oui |
| S7 | Certificat ACM `collector.qvl-project.com` et `*.collector.qvl-project.com` (XE2) | `PENDING_VALIDATION` puis `ISSUED` | Non | Chaîne 2 de D08 (NLB, certificat dans le cluster) | `ISSUED` pour les deux noms ; politique `TLSv1.2 TLSv1.3` disponible | Oui |
| S8 | EventBridge Scheduler sous `LabRole` (X15), en session et hors session | Groupe de sécurité étiqueté supprimé à l'heure dite, planification disparue | Non | G1 de D11 réduit à `down.sh` et au rappel sur le poste | S8a : groupe supprimé à l'échéance (+30 s), planification effacée. S8b : planification exécutée à l'échéance mais suppression refusée (`UnauthorizedOperation` pour `LabRole`, lab arrêté) | S8a oui ; S8b non |
| S9 | AWS Budgets : alerte à 50 % et 80 % | Budget créé, deux notifications | Non | G6 abandonnée ; G5 couvre ES-29 (D11) | Budget `collector-budget` de 50 $, notifications à 50 % et 80 % | Oui |
| S10 | RDS : versions, chiffrement, `rds.force_ssl` ; Secrets Manager | Groupe de paramètres modifiable ; secret créé et supprimé sans délai | Non | `password_wo` au lieu de `manage_master_user_password` (D11) ; création réelle testée en XP01 | PostgreSQL 18.3 (`postgres18`) par défaut ; chiffrement et Multi-AZ proposés ; `rds.force_ssl` = 1 ; secret créé puis supprimé sans délai | Oui |
| S11 | S3 sous la SCP : bucket, blocage public, refus du non-TLS, présignature | 200 en HTTPS présigné, 403 en HTTP | Non | Écart documenté (D07) | `AES256` ; Object Lock refusé par la SCP ; blocage public et politique acceptés ; HTTPS 200, HTTP 403 | Oui |
| S12 | CloudWatch Logs : groupe avec rétention, écriture, `StartQuery` | Message retrouvé par Logs Insights | Non | Écart de D09 à arbitrer avant le sprint 1 | `Complete sonde-xp00` | Oui |
| S13 | Cognito (repli de D06) | Pool créé puis supprimé | Non | Repli B de D06 marqué indisponible | Pool créé puis supprimé | Oui |

## Environnement

- **Machine** : poste de Martin, Windows 11 Home 10.0.26200, Git Bash (version à relever en étape 0.3).
- **Région AWS** : us-east-1 uniquement.
- **Domaine** : `qvl-project.com`, registrar et DNS chez Cloudflare. Sous-domaine du projet `collector.qvl-project.com`, délégué à Route 53 par des `NS` posés dans Cloudflare en « DNS only ». Aucun autre enregistrement de `qvl-project.com` n'est modifié.
- **Coût estimé** : < 0,01 $ (tâche Fargate d'une minute, requêtes) + zone Route 53 0,50 $ par mois (couche persistante). Coût constaté : à relever dans `docs/mesures/couts.csv` après 12 h.

### Versions relevées le 2026-10-26

| Outil | Version | Statut |
|---|---|---|
| AWS CLI | 2.33.24 | Présent |
| Terraform | 1.14.5 | Présent |
| kubectl (client) | 1.34.1 | Présent ; décalage d'une mineure avec EKS 1.35, supporté |
| Helm | 4.1.3 | Présent |
| Docker | 28.5.1 | Présent |
| curl | 8.17.0 | Présent |
| k3d | 5.9.0 (k3s v1.35.5-k3s1 par défaut) | Installé le 2026-10-26 (winget `k3d.k3d`) |
| k6 | 2.2.0 | Installé le 2026-10-26 (winget `GrafanaLabs.k6`), version figée pour D13 |
| jq | 1.8.2 | Installé le 2026-10-26 (winget `jqlang.jq`), confort ; `--query` de la CLI reste la référence |
| dig | — | Absent, non requis : `nslookup` le remplace |

### Noms et étiquettes de la sonde

- Préfixe : `collector-xp00` ; étiquettes `projet=collector`, `couche=sonde` (ressources supprimées en fin de sonde) ou `couche=persistant` (zone, certificat, budget).
- Seules la zone, le certificat, son enregistrement de validation et le budget survivent à la sonde.

## Technos et interactions

Poste (AWS CLI, profil nommé) → API AWS us-east-1 : STS, IAM (lecture), EKS (lecture), ECS/Fargate, Route 53, ACM, Scheduler → EC2, Budgets, RDS (paramètres), Secrets Manager, S3, CloudWatch Logs, Cognito. Cloudflare (interface web) porte la délégation `NS` de `collector.qvl-project.com`.

## Sécurité de la sonde

- **MI6** : identifiants de session dans le profil nommé `collector-lab` de `~/.aws/credentials`, jamais en argument, `.env` ni journal ; pas de `set -x`.
- **MX5** : aucune clé, aucun jeton, aucune URL présignée recopiés dans ce protocole ou le CSV ; numéro de compte tronqué aux 4 derniers chiffres ; adresse IP du poste jamais consignée.
- **MX6** : la planification S8 vise l'identifiant exact d'un groupe de sécurité vide créé et étiqueté par la sonde ; exécution unique, `ActionAfterCompletion=DELETE` ; groupe de planifications supprimé en fin de sonde.
- Le secret de S10 est une valeur aléatoire jetable, jamais affichée.
- Un refus citant `voc-cancel-cred` signale une session expirée, pas une interdiction : relancer « Start Lab », recopier les identifiants, rejouer l'étape.

## Étapes reproductibles

Chaque étape : commandes, résultat attendu, décision qui en dépend. Consigner chaque résultat dans le CSV (voir « Données collectées »). Ordre : prérequis, S1, S2, S6 (propagation DNS), puis le reste pendant l'attente de S7, S8 en dernier (il termine la session).

### 0. Prérequis côté poste (avant « Start Lab »)

**0.1 Installer k3d** (requis avant le 2026-10-29).

```bash
winget search k3d
winget install --id <identifiant affiché> --exact
k3d version
```

Attendu : version de k3d et version de k3s par défaut affichées. Consigner les deux. À défaut de paquet winget : binaire Windows de la page des versions de `k3d-io/k3d`, somme SHA-256 vérifiée.

**0.2 Installer k6** (outil de tir de D13, figé avant le 2026-10-28).

```bash
winget install k6 --source winget
k6 version
```

**0.3 Relever les versions** (Git Bash, nouveau terminal).

```bash
bash --version | head -1; aws --version; terraform version | head -1
kubectl version --client; helm version --short; docker --version; k3d version; k6 version
```

Attendu : le tableau ci-dessus confirmé, k3d et k6 ajoutés.

**0.4 Lancer la session**. Dans le Learner Lab : « Start Lab », attendre le voyant vert. Noter l'heure de début (référence du refus sous 75 min de D11).

**0.5 Profil de session** (MI6).
- « AWS Details » → « AWS CLI » → « Show » : copier le bloc.
- Ouvrir `~/.aws/credentials` dans un éditeur, coller le bloc sous l'en-tête `[collector-lab]` (remplacer `[default]`). Ne jamais coller les clés dans le terminal.

```bash
aws configure set region us-east-1 --profile collector-lab
aws configure set output json --profile collector-lab
export AWS_PROFILE=collector-lab AWS_REGION=us-east-1 AWS_PAGER="" MSYS_NO_PATHCONV=1
export SD=collector.qvl-project.com
export TAG_SESSION=$(date -u +%Y%m%dT%H%MZ)
MON_IP=$(curl -s https://checkip.amazonaws.com)   # jamais consignée
```

Attendu : aucune sortie d'erreur. `MSYS_NO_PATHCONV=1` évite la réécriture des chemins `/aws/...` par Git Bash (D11).

### S1 — Budget

Lecture dans l'interface du Learner Lab (bandeau « Used $x of $y ») et dans « AWS Details » (région).

- Attendu : budget 50 ou 100 $, consommé proche de 0, région us-east-1.
- Décision : 100 $ → plan de D11 inchangé. 50 $ → sessions en allumage à la demande (≈ 0,85 $), seuil d'arrêt G5 à 70 % décisif, réserve de rejeu de D14 comme variable d'ajustement.
- Capture : bandeau du budget (compte masqué).

### S2 — Identité et rôles

```bash
aws sts get-caller-identity --query Arn --output text
ACCT=$(aws sts get-caller-identity --query Account --output text)
aws iam list-roles --query "Roles[?starts_with(RoleName,'Lab')].RoleName" --output text
aws iam list-instance-profiles --query "InstanceProfiles[?starts_with(InstanceProfileName,'Lab')].InstanceProfileName" --output text
aws iam get-role --role-name LabRole --query 'Role.AssumeRolePolicyDocument.Statement[].Principal.Service' --output text
```

- Attendu : `arn:aws:sts::<compte>:assumed-role/voclabs/user…` ; `LabRole` (et peut-être `LabEksClusterRole`, `LabEksNodeRole`) ; `LabInstanceProfile`. `get-role` : refus probable (REX) ; s'il passe, vérifier que `eks.amazonaws.com`, `ecs-tasks.amazonaws.com`, `ec2.amazonaws.com` et `scheduler.amazonaws.com` sont approuvés.
- Décision : `LabRole` absent → arrêt, aucune XP cloud. `LabEks*` présents → candidats pour le cluster ; le rôle des nœuds reste `LabRole` si `LabEksNodeRole` n'a pas S3 (X6, tranché en XP01). `get-role` refusé → ARN construit `arn:aws:iam::$ACCT:role/LabRole`, pas de module communautaire (D11).

**Inventaire initial** (référence pour les inventaires suivants).

```bash
aws resourcegroupstaggingapi get-resources --query 'ResourceTagMappingList[].ResourceARN' --output text
aws ec2 describe-instances --query 'Reservations[].Instances[].[InstanceId,State.Name]' --output text
aws eks list-clusters --output text; aws ecs list-clusters --output text
aws elbv2 describe-load-balancers --query 'LoadBalancers[].LoadBalancerName' --output text
aws ec2 describe-nat-gateways --filter Name=state,Values=pending,available --query 'NatGateways[].NatGatewayId' --output text
aws ec2 describe-addresses --query 'Addresses[].AllocationId' --output text
aws rds describe-db-instances --query 'DBInstances[].DBInstanceIdentifier' --output text
aws route53 list-hosted-zones --query 'HostedZones[].Name' --output text
```

- Attendu : tout vide.
- Décision : une ressource préexistante est identifiée et notée avant toute création ; une ressource facturable inconnue est signalée à Martin avant de continuer.

### S6 — Zone Route 53 et délégation Cloudflare (XE1)

Voie nominale de D14 : zone et certificat créés par Terraform dans `infra/persistant/`. Si cette couche n'est pas prête le 2026-10-27, les commandes ci-dessous la remplacent, et la zone et le certificat sont importés (`terraform import`) dans `infra/persistant/` au sprint 1. Le choix est noté dans « Difficultés ».

```bash
ZONE_ID=$(aws route53 create-hosted-zone --name "$SD" --caller-reference "collector-$TAG_SESSION" \
  --hosted-zone-config Comment=collector-persistant --query 'HostedZone.Id' --output text)
aws route53 change-tags-for-resource --resource-type hostedzone --resource-id "${ZONE_ID##*/}" \
  --add-tags Key=projet,Value=collector Key=couche,Value=persistant
aws route53 get-hosted-zone --id "$ZONE_ID" --query 'DelegationSet.NameServers' --output text
```

- Attendu : identifiant `/hostedzone/Z…` ; quatre serveurs `ns-…awsdns-…` (`.com`, `.net`, `.org`, `.co.uk`). Les consigner.
- Si refus (`AccessDenied` non lié à `voc-cancel-cred`) : repli D2, les enregistrements du projet seront posés directement dans la zone Cloudflare de `qvl-project.com` ; passer à S7 en validation DNS dans Cloudflare.

**Délégation par Martin dans Cloudflare** (zone `qvl-project.com` → DNS → Records) :
- quatre enregistrements `NS`, nom `collector`, valeur = chacun des quatre serveurs relevés, TTL Auto ;
- proxy : « DNS only » (les `NS` ne sont pas proxifiables) ;
- aucun autre enregistrement modifié ; aucun `A`, `AAAA` ou `CNAME` existant sur `collector`.

**Vérification** (quelques minutes ; relancer pendant les étapes suivantes) :

```bash
nslookup -type=NS collector.qvl-project.com 1.1.1.1
nslookup -type=NS collector.qvl-project.com 8.8.8.8
nslookup -type=SOA collector.qvl-project.com 8.8.8.8
```

- Attendu : les quatre serveurs `awsdns` ; SOA de `awsdns-hostmaster.amazon.com`. Équivalent PowerShell : `Resolve-DnsName collector.qvl-project.com -Type NS -Server 1.1.1.1`.
- Décision : XE1 atteint → chaîne 1 de D08 confirmée pour le DNS. Délégation non visible après 1 h : vérifier la saisie ; au-delà d'un jour, XP03 passe en session ECS ou en réserve (risque 3 de D14).

### S3 — Versions EKS, add-ons, AMI

```bash
aws eks describe-cluster-versions \
  --query 'clusterVersions[].[clusterVersion,versionStatus,endOfStandardSupportDate]' --output table
for a in vpc-cni coredns kube-proxy metrics-server; do
  echo "$a: $(aws eks describe-addon-versions --addon-name $a --kubernetes-version 1.35 \
    --query 'addons[0].addonVersions[?compatibilities[?defaultVersion==`true`]].addonVersion' --output text)"
done
aws ssm get-parameter --name /aws/service/eks/optimized-ami/1.35/amazon-linux-2023/x86_64/standard/recommended/image_id \
  --query Parameter.Value --output text
```

- Attendu : 1.35 (ou plus) en `STANDARD_SUPPORT` ; une version par défaut pour chaque add-on ; un identifiant d'AMI `ami-…`.
- Décision : versions recopiées dans les tableaux « Versions » de XP01 à XP03 (EKS, `vpc-cni`, `coredns`, `kube-proxy`, `metrics-server`, AMI). `metrics-server` absent des add-ons → chart Helm épinglé. 1.35 absente → version en support standard la plus récente, sinon surcoût intégré à X13.

### S4 — ECS Fargate

```bash
VPC=$(aws ec2 describe-vpcs --filters Name=is-default,Values=true --query 'Vpcs[0].VpcId' --output text)
SUBNET=$(aws ec2 describe-subnets --filters Name=vpc-id,Values=$VPC Name=default-for-az,Values=true \
  --query 'Subnets[0].SubnetId' --output text)
SG_ECS=$(aws ec2 create-security-group --group-name collector-xp00-ecs --description "sonde XP00 ECS" --vpc-id $VPC \
  --tag-specifications 'ResourceType=security-group,Tags=[{Key=projet,Value=collector},{Key=couche,Value=sonde}]' \
  --query GroupId --output text)
aws ecs create-cluster --cluster-name collector-xp00 --tags key=projet,value=collector key=couche,value=sonde \
  --query cluster.status --output text
aws ecs register-task-definition --family collector-xp00 --requires-compatibilities FARGATE --network-mode awsvpc \
  --cpu 256 --memory 512 \
  --execution-role-arn arn:aws:iam::$ACCT:role/LabRole --task-role-arn arn:aws:iam::$ACCT:role/LabRole \
  --container-definitions '[{"name":"sonde","image":"public.ecr.aws/docker/library/busybox:1.37.0","command":["sh","-c","echo ok"],"essential":true}]' \
  --query taskDefinition.revision --output text
TASK=$(aws ecs run-task --cluster collector-xp00 --launch-type FARGATE --task-definition collector-xp00 \
  --network-configuration "awsvpcConfiguration={subnets=[$SUBNET],securityGroups=[$SG_ECS],assignPublicIp=ENABLED}" \
  --query 'tasks[0].taskArn' --output text)
aws ecs wait tasks-stopped --cluster collector-xp00 --tasks $TASK
aws ecs describe-tasks --cluster collector-xp00 --tasks $TASK \
  --query 'tasks[0].[lastStatus,stopCode,containers[0].exitCode]' --output text
```

- Attendu : `ACTIVE`, révision `1`, puis `STOPPED EssentialContainerExited 0`. Coût < 0,001 $.
- Pas de VPC par défaut : `run-task` reporté à la session ECS de XP01 (S4 reste ouvert, la création du cluster suffit à prouver l'API).
- Décision : succès → comparatif EKS / Fargate de XP01 maintenu. Refus → XP01 réduite à EKS (D14).

Nettoyage :

```bash
aws ecs deregister-task-definition --task-definition collector-xp00:1 --query taskDefinition.status --output text
aws ecs delete-task-definitions --task-definitions collector-xp00:1 --output text
aws ecs delete-cluster --cluster collector-xp00 --query cluster.status --output text
aws ec2 delete-security-group --group-id $SG_ECS   # réessayer après 1 à 2 min si DependencyViolation
```

### S5 — Quotas

```bash
aws service-quotas get-service-quota --service-code fargate --quota-code L-3032A538
aws service-quotas get-service-quota --service-code ec2 --quota-code L-1216C47A
aws ec2 describe-account-attributes --attribute-names max-instances \
  --query 'AccountAttributes[0].AttributeValues[0].AttributeValue' --output text
```

- Attendu : `AccessDenied` sur les deux premières (REX) ; une valeur pour `max-instances`.
- Décision : refus → plafonds de XP01 fixés à 4 vCPU Fargate (16 tâches de 0,25 vCPU au plus, 6 utilisées) et 4 nœuds EKS, sous la limite de 9 instances.

### S7 — Certificat ACM (XE2)

```bash
CERT_ARN=$(aws acm request-certificate --domain-name "$SD" --subject-alternative-names "*.$SD" \
  --validation-method DNS --idempotency-token xp00collector \
  --tags Key=projet,Value=collector Key=couche,Value=persistant --query CertificateArn --output text)
aws acm describe-certificate --certificate-arn "$CERT_ARN" --query 'Certificate.Status' --output text
read -r RR_NAME RR_TYPE RR_VALUE <<<"$(aws acm describe-certificate --certificate-arn "$CERT_ARN" \
  --query 'Certificate.DomainValidationOptions[0].ResourceRecord.[Name,Type,Value]' --output text)"
aws route53 change-resource-record-sets --hosted-zone-id "$ZONE_ID" --change-batch \
  '{"Changes":[{"Action":"UPSERT","ResourceRecordSet":{"Name":"'"$RR_NAME"'","Type":"'"$RR_TYPE"'","TTL":300,"ResourceRecords":[{"Value":"'"$RR_VALUE"'"}]}}]}'
```

- Attendu : `PENDING_VALIDATION` ; un seul enregistrement `CNAME` de validation, commun à `collector.qvl-project.com` et `*.collector.qvl-project.com`. Si `ResourceRecord` est vide, relancer `describe-certificate` après 10 s.
- Repli D2 (S6 refusé) : poser ce `CNAME` dans Cloudflare, « DNS only ».

Attente (continuer S9 à S13 pendant ce temps) :

```bash
aws acm wait certificate-validated --certificate-arn "$CERT_ARN"
aws acm describe-certificate --certificate-arn "$CERT_ARN" --query '[Certificate.Status,Certificate.SubjectAlternativeNames]' --output text
aws elbv2 describe-ssl-policies --names ELBSecurityPolicy-TLS13-1-2-Res-2021-06 --query 'SslPolicies[0].SslProtocols' --output text
```

- Attendu : `ISSUED` avec les deux noms, en quelques minutes une fois la délégation visible ; politique TLS de ME7 disponible (`TLSv1.2 TLSv1.3`).
- Décision : `ISSUED` → XE2 atteint, chaîne 1 de D08 ; l'attachement à l'écouteur est vérifié en XP03. `request-certificate` refusé → chaîne 2 de D08.

### S9 — AWS Budgets (G6)

```bash
aws budgets describe-budgets --account-id $ACCT --query 'Budgets[].[BudgetName,BudgetLimit.Amount]' --output text
aws budgets create-budget --account-id $ACCT \
  --budget '{"BudgetName":"collector-budget","BudgetLimit":{"Amount":"<budget S1>","Unit":"USD"},"TimeUnit":"ANNUALLY","BudgetType":"COST"}' \
  --notifications-with-subscribers '[
   {"Notification":{"NotificationType":"ACTUAL","ComparisonOperator":"GREATER_THAN","Threshold":50,"ThresholdType":"PERCENTAGE"},"Subscribers":[{"SubscriptionType":"EMAIL","Address":"<courriel de Martin>"}]},
   {"Notification":{"NotificationType":"ACTUAL","ComparisonOperator":"GREATER_THAN","Threshold":80,"ThresholdType":"PERCENTAGE"},"Subscribers":[{"SubscriptionType":"EMAIL","Address":"<courriel de Martin>"}]}]'
aws budgets describe-notifications-for-budget --account-id $ACCT --budget-name collector-budget --output text
```

- Attendu : budget créé, deux notifications. Gratuit (deux premiers budgets). Conservé : c'est G6, couche persistante.
- Décision : refus → G6 abandonnée, ES-29 tenue par G5 (D11).

### S10 — RDS et Secrets Manager

```bash
read -r PG_VER PG_FAM <<<"$(aws rds describe-db-engine-versions --engine postgres --default-only \
  --query 'DBEngineVersions[0].[EngineVersion,DBParameterGroupFamily]' --output text)"
echo "$PG_VER $PG_FAM"
aws rds describe-orderable-db-instance-options --engine postgres --engine-version "$PG_VER" \
  --db-instance-class db.t3.micro --query 'OrderableDBInstanceOptions[0].[SupportsStorageEncryption,MultiAZCapable]' --output text
aws rds create-db-parameter-group --db-parameter-group-name collector-xp00 --db-parameter-group-family "$PG_FAM" \
  --description "sonde XP00" --tags Key=projet,Value=collector Key=couche,Value=sonde --output text
aws rds modify-db-parameter-group --db-parameter-group-name collector-xp00 \
  --parameters ParameterName=rds.force_ssl,ParameterValue=1,ApplyMethod=pending-reboot --output text
aws rds describe-db-parameters --db-parameter-group-name collector-xp00 \
  --query "Parameters[?ParameterName=='rds.force_ssl'].ParameterValue" --output text
aws rds delete-db-parameter-group --db-parameter-group-name collector-xp00
aws secretsmanager create-secret --name collector-xp00-sonde --secret-string "$(openssl rand -hex 16)" \
  --tags Key=projet,Value=collector Key=couche,Value=sonde --query Name --output text
aws secretsmanager delete-secret --secret-id collector-xp00-sonde --force-delete-without-recovery --query Name --output text
```

- Attendu : version PostgreSQL par défaut et sa famille (à recopier dans XP01) ; chiffrement `True` ; `rds.force_ssl` = `1` ; secret créé puis supprimé sans délai.
- Décision : version PostgreSQL épinglée pour XP01. `manage_master_user_password` est éprouvé à la création réelle de RDS en XP01 (G1) ; échec → `password_wo` (D11, MI3). Suppression sans délai refusée → inventaire MX7 adapté (nom de secret suffixé par session).

### S11 — S3 sous la SCP

```bash
B=collector-xp00-$(date +%s)
aws s3api create-bucket --bucket $B --output text
aws s3api put-bucket-tagging --bucket $B --tagging 'TagSet=[{Key=projet,Value=collector},{Key=couche,Value=sonde}]'
aws s3api put-public-access-block --bucket $B --public-access-block-configuration \
  BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
aws s3api put-bucket-policy --bucket $B --policy '{"Version":"2012-10-17","Statement":[{"Sid":"RefusNonTLS","Effect":"Deny","Principal":"*","Action":"s3:*","Resource":["arn:aws:s3:::'$B'","arn:aws:s3:::'$B'/*"],"Condition":{"Bool":{"aws:SecureTransport":"false"}}}]}'
aws s3api get-bucket-encryption --bucket $B --query 'ServerSideEncryptionConfiguration.Rules[0].ApplyServerSideEncryptionByDefault.SSEAlgorithm' --output text
aws s3api get-object-lock-configuration --bucket $B
echo sonde | aws s3 cp - s3://$B/sonde.txt
URL=$(aws s3 presign s3://$B/sonde.txt --expires-in 120)   # jamais affichée ni consignée
curl -s -o /dev/null -w 'HTTPS %{http_code}\n' "$URL"
curl -s -o /dev/null -w 'HTTP %{http_code}\n' "${URL/https:/http:}"
unset URL
aws s3 rm s3://$B/sonde.txt; aws s3api delete-bucket --bucket $B
```

- Attendu : bucket créé ; `AES256` ; `get-object-lock-configuration` refusé par une SCP (REX) ou `ObjectLockConfigurationNotFoundError` ; `HTTPS 200`, `HTTP 403`.
- Décision : SCP présente → bucket des photos créé en CLI, configuré par les ressources `aws_s3_bucket_*` (D11). Blocage public ou politique refusés → écart documenté (D07).

### S12 — CloudWatch Logs

```bash
LG=/collector/xp00/sonde
aws logs create-log-group --log-group-name $LG --tags projet=collector,couche=sonde
aws logs put-retention-policy --log-group-name $LG --retention-in-days 1
aws logs create-log-stream --log-group-name $LG --log-stream-name sonde
aws logs put-log-events --log-group-name $LG --log-stream-name sonde \
  --log-events timestamp=$(($(date +%s)*1000)),message=sonde-xp00
sleep 10
Q=$(aws logs start-query --log-group-name $LG --start-time $(($(date +%s)-900)) --end-time $(($(date +%s)+60)) \
  --query-string 'fields @message | limit 5' --query queryId --output text)
sleep 5; aws logs get-query-results --query-id $Q --query '[status,results[0][0].value]' --output text
aws logs delete-log-group --log-group-name $LG
```

- Attendu : `Complete sonde-xp00`.
- Décision : succès → la source CloudWatch de Grafana (D09) reste à éprouver depuis un pod en XP01 (IMDS). Refus de `StartQuery` → Logs Insights dans la console, écart de D09 à arbitrer avant le sprint 1.

### S13 — Cognito

```bash
aws cognito-idp list-user-pools --max-results 10 --query 'UserPools[].Name' --output text
POOL=$(aws cognito-idp create-user-pool --pool-name collector-xp00 --user-pool-tags projet=collector,couche=sonde \
  --query UserPool.Id --output text)
aws cognito-idp delete-user-pool --user-pool-id $POOL
```

- Attendu : liste vide, pool créé puis supprimé (gratuit).
- Décision : refus → repli B de D06 marqué indisponible ; Keycloak sans repli managé.

### S8 — EventBridge Scheduler sous `LabRole` (X15), en dernier

**S8a, en session** : planification à +5 min supprimant un groupe de sécurité vide étiqueté.

```bash
SG_A=$(aws ec2 create-security-group --group-name collector-xp00-x15a --description "cible X15 en session" --vpc-id $VPC \
  --tag-specifications 'ResourceType=security-group,Tags=[{Key=projet,Value=collector},{Key=couche,Value=sonde},{Key=cible,Value=x15}]' \
  --query GroupId --output text)
aws scheduler create-schedule-group --name collector-xp00 --tags Key=projet,Value=collector Key=couche,Value=sonde
QUAND=$(date -u -d '+5 minutes' +%Y-%m-%dT%H:%M:%S)
aws scheduler create-schedule --name x15a --group-name collector-xp00 \
  --schedule-expression "at($QUAND)" --schedule-expression-timezone UTC \
  --flexible-time-window Mode=OFF --action-after-completion DELETE \
  --target '{"Arn":"arn:aws:scheduler:::aws-sdk:ec2:deleteSecurityGroup","RoleArn":"arn:aws:iam::'$ACCT':role/LabRole","Input":"{\"GroupId\":\"'$SG_A'\"}"}'
```

Après 6 min :

```bash
aws ec2 describe-security-groups --group-ids $SG_A --query 'SecurityGroups[0].GroupId' --output text
aws scheduler list-schedules --group-name collector-xp00 --query 'Schedules[].Name' --output text
```

- Attendu : création acceptée (`iam:PassRole` vers Scheduler autorisé) ; puis `InvalidGroup.NotFound` et liste vide.
- Décision : refus de `PassRole` → G1 réduit à `down.sh`, G5 et G7 (D11). Groupe toujours présent → G1 non fiable, même repli.

**S8b, hors session** (seulement si S8a réussit) : même commande avec `x15b`, un groupe `collector-xp00-x15b` et `+10 minutes`, puis :

1. Vérifier que tout le nettoyage des étapes précédentes est fait (inventaire final ci-dessous, hors `SG_B`).
2. Noter l'heure, cliquer « End Lab ».
3. Attendre au moins 15 min, puis « Start Lab » ; recopier les nouveaux identifiants dans le profil `collector-lab` (étape 0.5) ; réexporter `AWS_PROFILE`, `AWS_REGION`, `MSYS_NO_PATHCONV`.
4. Contrôler :

```bash
aws ec2 describe-security-groups --filters Name=tag:cible,Values=x15 --query 'SecurityGroups[].GroupId' --output text
aws scheduler list-schedules --group-name collector-xp00 --query 'Schedules[].Name' --output text
aws scheduler delete-schedule-group --name collector-xp00
```

- Attendu : aucun groupe `cible=x15`, aucune planification ; groupe de planifications supprimé.
- Décision : succès → G1 couvre aussi une échéance hors session. Échec → G1 limitée à la session (planifications T-30 et T-10 de D11 inchangées), risque d'oubli hors session documenté. Un groupe restant est supprimé à la main et noté.

### Fin — Inventaire final et report

```bash
aws ec2 describe-security-groups --filters Name=tag:projet,Values=collector --query 'SecurityGroups[].GroupId' --output text
aws ecs list-clusters --output text
aws s3api list-buckets --query "Buckets[?starts_with(Name,'collector-xp00')].Name" --output text
aws logs describe-log-groups --log-group-name-prefix /collector --query 'logGroups[].logGroupName' --output text
aws secretsmanager list-secrets --filters Key=name,Values=collector --query 'SecretList[].Name' --output text
aws rds describe-db-parameter-groups --query "DBParameterGroups[?starts_with(DBParameterGroupName,'collector')].DBParameterGroupName" --output text
aws cognito-idp list-user-pools --max-results 10 --query 'UserPools[].Name' --output text
aws scheduler list-schedule-groups --query "ScheduleGroups[?Name!='default'].Name" --output text
aws resourcegroupstaggingapi get-resources --tag-filters Key=projet,Values=collector \
  --query 'ResourceTagMappingList[].ResourceARN' --output text
```

- Attendu : toutes les listes vides, sauf la dernière : zone `collector.qvl-project.com`, certificat ACM (et le budget, non étiquetable par cette API).
- Après la session : faits durables reportés une seule fois dans [environnement-learner-lab.md](../00-cadrage/environnement-learner-lab.md) ; versions recopiées dans XP01 à XP03 ; nouvelle vérification de XE1 depuis un autre réseau (partage mobile) le lendemain.

## Durée et coût

- **Durée** : ≈ 1 h de commandes + 15 à 20 min pour S8b (attente hors session, mise à profit pour la vérification Cloudflare).
- **Coût** : < 0,01 $ immédiat ; zone Route 53 0,50 $ par mois tant qu'elle existe.

## Données collectées

- **CSV** : `docs/mesures/XP00-sonde-compte.csv`, une ligne par commande significative. Colonnes : `horodatage_utc;xp;point;commande;resultat;code_erreur;politique_citee;decision`. `resultat` ∈ `OK`, `REFUS`, `ERREUR`, `NON_JOUE`.
- **Captures** (`docs/mesures/XP00-sonde-compte/`) : budget du lab, enregistrements `NS` dans Cloudflare, certificat `ISSUED` dans la console ACM, résultat de S8b. Compte masqué sur chaque capture.
- **Versions** : tableau « Versions relevées » complété (k3d, k6, Git Bash) et versions AWS de S3 et S10.

## Résultats

Session du 2026-10-26, budget du lab 50 $. Les 13 points sont joués, aucun n'est bloquant.

**Identité et droits.** Le principal est `assumed-role/voclabs/user…`. Seuls `LabRole` et `LabInstanceProfile` existent, sans `LabEksClusterRole` ni `LabEksNodeRole` : `LabRole` sert de rôle de cluster, de nœuds et de tâches. `iam:GetRole` est autorisé, contrairement au REX. La relation de confiance de `LabRole` couvre notamment `eks`, `eks-fargate-pods`, `ecs-tasks`, `ec2`, `scheduler`, `rds` et `elasticloadbalancing`.

**Inventaire initial.** Aucune ressource facturable. Seul l'outillage du lab est présent : fonctions Lambda `ModLabRole`, `MainMonitoringFunction` et `RoleCreationFunction`, règles `voc-*`, pile CloudFormation du lab, modèle Elastic Beanstalk d'exemple.

**Versions relevées.**

| Élément | Valeur |
|---|---|
| EKS | 1.34 à 1.37 en support standard ; 1.35 jusqu'au 2027-03-27, 1.37 jusqu'au 2027-12-01 |
| `vpc-cni` (1.35) | v1.22.4-eksbuild.8 |
| `coredns` (1.35) | v1.13.2-eksbuild.59 |
| `kube-proxy` (1.35) | v1.35.3-eksbuild.59 |
| `metrics-server` (1.35) | v0.9.0-eksbuild.14 (add-on géré, pas de chart Helm) |
| `eks-pod-identity-agent` (1.35) | v1.4.1-eksbuild.3 |
| AMI EKS 1.35 AL2023 x86_64 | `ami-06c28983cd518f095` (valeur du jour, relue par `up` à chaque session) |
| PostgreSQL RDS | 18.3 par défaut (famille `postgres18`) ; 18.6 la plus récente |
| Quota d'instances EC2 | `max-instances` = 20 |

**DNS et certificat.** La zone `collector.qvl-project.com` est créée et étiquetée `couche=persistant`. Ses serveurs sont `ns-16.awsdns-02.com`, `ns-757.awsdns-30.net`, `ns-1190.awsdns-20.org` et `ns-1918.awsdns-47.co.uk`. Les quatre `NS` sont posés par l'API dans la zone Cloudflare `qvl-project.com`, avec le commentaire « delegation Route 53 projet Collector » ; aucun autre enregistrement n'est touché. Le certificat ACM de `collector.qvl-project.com` et `*.collector.qvl-project.com` passe `ISSUED` en moins de 5 min. XE1 et XE2 sont atteints : la chaîne 1 de D08 est confirmée pour le DNS et le TLS.

**Garde-fous.** Le budget `collector-budget` (50 $, alertes à 50 % et 80 % vers l'adresse école de Martin) est créé : G6 est en place. EventBridge Scheduler accepte `LabRole`, donc `iam:PassRole` est autorisé. La planification S8a a supprimé le groupe de sécurité cible 30 s après l'échéance, puis s'est effacée (`ActionAfterCompletion=DELETE`) : G1 est tenu en session. S8b (lab arrêté à 14 h 00 UTC, échéance à 14 h 09) : la planification s'exécute à l'heure et s'efface, mais CloudTrail montre `Client.UnauthorizedOperation` sur `ec2:DeleteSecurityGroup` pour `LabRole`. Lab arrêté, `LabRole` perd ses droits d'action : **G1 ne couvre que le temps de session**. Le groupe restant a été supprimé à la main à la reprise.

**Données et journaux.**
- RDS : chiffrement disponible sur `db.t3.micro`, `rds.force_ssl` modifiable.
- Secrets Manager : suppression sans délai acceptée.
- S3 : une SCP est présente (Object Lock refusé). Chiffrement `AES256` par défaut. Blocage public et politique « refus du non-TLS » acceptés. URL présignée : 200 en HTTPS, 403 en HTTP.
- Logs Insights est opérationnel.
- Cognito est disponible : le repli B de D06 est maintenu.

**Inventaire final.** Groupes de sécurité, clusters ECS, buckets, groupes de journaux, secrets, groupes de paramètres et pools Cognito : tous vides. Survivent seulement la zone, le certificat et le budget ; le groupe de planifications `collector-xp00` est supprimé après S8b. L'API d'étiquettes liste encore quelques minutes le cluster ECS et le pool supprimés ; c'est un décalage d'index, pas une ressource.

**Conséquences.**
- Budget de 50 $ : l'estimation de D11 (38 à 47 $) laisse moins de 10 % de marge. Les sessions se font en allumage à la demande, le seuil d'arrêt G5 à 70 % devient décisif, et la réserve de rejeu de D14 sert de variable d'ajustement. Le coût vient des ressources allumées à l'heure (plan de contrôle EKS, nœuds, ALB, RDS), pas du trafic.
- G1 limitée à la session : les planifications T-30 et T-10 de D11 restent la défense principale, `down.sh` doit finir **avant** « End Lab ». Une ressource oubliée continue d'être facturée lab arrêté ; G5 (seuil de coût) et l'inventaire de début de session couvrent ce risque, documenté dans D11.
- XP01 à XP03 : recopier les versions ci-dessus. EKS 1.35 est conservée, alignée sur k3s v1.35.5 de k3d ; 1.37 est notée comme option.

## Difficultés

- `jq`, installé le jour même, n'était pas dans le `PATH` du terminal déjà ouvert. La première tentative de pose des `NS` a donc échoué sans rien créer : aucun enregistrement `collector` n'existait avant la seconde tentative, c'est vérifié. Il faut rouvrir le terminal après une installation.
- S4 : le premier `run-task`, lancé juste après `create-cluster`, a renvoyé `InvalidParameterException`. Le second essai, sur un sous-réseau explicite, a réussi. Les scripts attendront l'état `ACTIVE` du cluster et fixeront le sous-réseau.
- La zone et le certificat sont créés en CLI, car la couche `infra/persistant/` n'existe pas encore. Ils seront importés dans Terraform au sprint 1.

## Limites

- La sonde prouve des droits d'API, pas des comportements sous charge : X6 (S3 depuis un pod), `manage_master_user_password` et la politique réseau du CNI sont éprouvés en XP01.
- Un droit accordé le 2026-10-27 peut changer ensuite (variantes du lab) : la sonde des scripts `up` (D11) rejoue l'identité, la région et la version EKS à chaque session.

## Conclusion

- **Relecture architecture** :
- **Relecture sécurité** (IAM, TLS) :
- **Verdict de Martin** :
- **Mise à jour des décisions** : D08 (XE1, XE2) ; prérequis de D03, D05, D06 ; G1 et G6 de D11 ; écarts de D09.
