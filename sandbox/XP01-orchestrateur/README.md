# XP01 — bac à sable EKS / ECS Fargate

IaC et scripts de [XP01](../../docs/experimentations/XP01-orchestrateur.md), écrits à la forme de la future couche `infra/ephemere/` ([D11](../../docs/decisions/D11-iac-allumage-cout.md), [D14](../../docs/decisions/D14-experimentations.md)). Jetable après J3 ; le POC n'en dépend pas.

| Chemin | Rôle |
|---|---|
| `ephemere/` | Racine Terraform unique : réseau, ALB, DNS, RDS communs ; `eks.tf` ou `ecs.tf` selon `orchestrateur` |
| `up.sh` | Contrôles, bucket en CLI, `apply` horodaté (G2), destruction programmée T-30/T-10, attente du HTTPS 200 |
| `down.sh` | Releases Helm, bucket, `destroy` avec reprise sur les interfaces, planifications, inventaire strict (G3) |
| `inventaire.sh` | Inventaire MX7 du compte, indépendant de l'état ; `--strict` échoue s'il reste quoi que ce soit |
| `lib.sh` | Fonctions communes (profil, masquage du compte, CSV de mesures) |

**Une racine et une variable plutôt que deux racines** : le socle (VPC, NAT, ALB, DNS, RDS) est identique dans les deux variantes et n'est écrit qu'une fois ; seuls `eks.tf` et `ecs.tf` diffèrent, ce qui sert aussi le comptage G10. Un seul état, une seule commande `down`. La variante est figée par l'état : passer de `eks` à `ecs` exige un `down` complet.

## Prérequis

- Terraform 1.14.5, provider AWS 6.70.0 (`ephemere/.terraform.lock.hcl` versionné), AWS CLI 2.33.24, `jq`, `curl` ; kubectl 1.34+ et Helm 4.1.3 pour EKS.
- Git Bash : `MSYS_NO_PATHCONV=1` est posé par `lib.sh`. Lancer les scripts par `bash ./up.sh …`.
- Profil `collector-lab` à jour après « Start Lab » (`~/.aws/credentials`), jamais d'identifiant en argument ni en fichier du dépôt.
- Couche persistante présente : zone `collector.qvl-project.com` et certificat ACM `ISSUED`, lus en `data`.
- Variante ECS : `ephemere/xp01.auto.tfvars` créé à partir de l'exemple, avec `image_test` et `image_sonde` épinglées par empreinte.

## Commandes

```bash
bash ./inventaire.sh                                   # doit être vide avant la session 1
bash ./up.sh eks --debut-lab 13:05 --libelle EKS1      # heure locale du « Start Lab »
bash ./up.sh eks --debut-lab 13:05 --chart ../chart-test --valeurs ../chart-test/valeurs-eks.yaml
bash ./up.sh --prolonger --debut-lab 15:40             # après un nouveau « Start Lab » : réarme T-30/T-10
bash ./down.sh                                         # en fin de travail, puis vérifier « Inventaire vide »
bash ./up.sh ecs --debut-lab 09:02 --libelle ECS
```

- `up.sh` refuse d'allumer sous 75 min restantes, relève l'adresse du poste (jamais consignée) pour l'ALB et l'API du cluster, et contrôle l'inventaire si l'état est vide. Sinon, il rafraîchit l'état avant l'`apply`. Les phases sont horodatées dans `docs/mesures/XP01-orchestrateur.csv`.
- Sans `--chart`, `up.sh` attend 30 min le HTTPS 200 pendant que le chart de test (NodePort 30080) est installé à la main.
- Tâches de sonde ECS (G4, G7, G8), sous-réseau explicite (constat XP00) :

```bash
SN=$(terraform -chdir=ephemere output -json sous_reseaux_prives | jq -r '.[0]')
SG=$(terraform -chdir=ephemere output -json groupes_securite_ecs | jq -r '.intrus')
aws ecs run-task --cluster collector-xp01 --launch-type FARGATE --task-definition collector-xp01-sonde \
  --network-configuration "awsvpcConfiguration={subnets=[$SN],securityGroups=[$SG],assignPublicIp=DISABLED}" \
  --overrides '{"containerOverrides":[{"name":"sonde","command":["curl -s -m 5 http://169.254.170.2$AWS_CONTAINER_CREDENTIALS_RELATIVE_URI; echo code=$?"]}]}'
```

Résultat dans le groupe de journaux `/collector/xp01/ecs`. `collector-xp01-sonde-labrole` porte `LabRole` (G4).

## Durée et coût attendus

| Variante | `up` | `down` + inventaire | Coût horaire | Session |
|---|---|---|---|---|
| EKS (2 × t3.medium) | 30 à 45 min (cluster puis node group) | 12 à 20 min | ≈ 0,34 $/h (D11) | 4 h ≈ 1,35 $ |
| ECS Fargate (2 à 6 tâches 0,25 vCPU) | 15 à 20 min (RDS sur le chemin critique) | 8 à 12 min | ≈ 0,14 $/h | 2 h ≈ 0,30 $ |

## Garde-fous et nettoyage

- Étiquettes `projet=collector`, `couche=ephemere`, `xp=XP01`, `session` sur toutes les ressources (nœuds, disques et interfaces par le modèle de lancement).
- Destruction programmée (D11 G1, MI5) : groupe `collector-xp01`, cibles exactes, `ActionAfterCompletion=DELETE`. Node group ou service ECS, RDS, ALB et NAT à T-30 ; cluster et adresse à T-10. Le groupe est purgé par `up` et supprimé par `down`.
- Aucun secret dans l'état : mot de passe maître géré par RDS (`manage_master_user_password`), supprimé avec l'instance. Aucune sortie ni ressource aléatoire secrète ; pas de fichier de plan.
- **Repli si `manage_master_user_password` échoue** : retirer cet argument, puis ajouter une variable `mdp_maitre` (`ephemeral = true`, `sensitive = true`) et `password_wo = var.mdp_maitre`, `password_wo_version = 1` sur `aws_db_instance.test`. La valeur est fournie au seul processus (`TF_VAR_mdp_maitre="$(openssl rand -base64 24)" bash ./up.sh …`), jamais écrite.
- État local `ephemere/terraform.tfstate`, copies horodatées dans `.etat-sauvegardes/`, journaux d'`apply` dans `.session/` : tous ignorés par git.
- Nettoyage manuel si `down.sh` échoue : relancer `down.sh` ; sinon supprimer chaque ressource listée par `inventaire.sh`, puis `bash ./inventaire.sh --strict` jusqu'au code retour 0.
