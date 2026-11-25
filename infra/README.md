# infra — IaC de l'environnement AWS

Deux configurations racines Terraform, sans module interne ([D11](../docs/decisions/D11-iac-allumage-cout.md)). État local hors dépôt (`*.tfstate` ignoré), aucun secret dans l'état, Helm hors de Terraform. Scripts d'allumage et d'extinction dans [tools/](../tools/).

| Racine | Contenu | Cycle de vie |
|---|---|---|
| `persistant/` | Zone Route 53 `collector.qvl-project.com`, CAA `amazon.com` (ME7), certificat ACM `collector.qvl-project.com` + `*.collector.qvl-project.com` et son CNAME de validation, budget `collector-budget` (G6), configuration du bucket des photos (accès public bloqué, SSE-S3, refus hors TLS, D07 MS7), groupes de journaux `/collector/app` (7 j) et `/collector/audit` (30 j, D09 MO7), clé KMS `alias/collector-secrets-eks` des Secrets EKS (rotation, suppression différée 7 j, AWS-0039) | Appliquée par `tools/bootstrap.sh`, rejouable sans effet ; zone, certificat et audit protégés par `prevent_destroy` |
| `ephemere/` | VPC sur deux zones, NAT unique, point de terminaison S3, ALB (TLS 1.2 minimum, HSTS, 80 → 443, 404 hors liste blanche), alias DNS, RDS PostgreSQL privée ; cluster EKS (D03, verdict XP01 du 2026-11-20) aux Secrets chiffrés par la clé KMS persistante, modèle de lancement IMDSv2 à 2 sauts, node group ≤ 4 nœuds, cible de l'ALB par NodePort | Créée par `tools/up.sh`, détruite par `tools/down.sh` à chaque session |

Versions épinglées : Terraform 1.14.5, provider AWS 6.70.0 (`.terraform.lock.hcl` versionné dans chaque racine).

## Couche persistante : `tools/bootstrap.sh`

La zone, le certificat, son CNAME de validation et le budget ont été créés en CLI par XP00. Les blocs `import` de `persistant/import.tf` les reprennent au premier `apply`, sans recréation. Le bucket des photos est créé en CLI par `bootstrap.sh` (SCP du lab : `aws_s3_bucket` échoue, D07), puis configuré par les ressources `aws_s3_bucket_*` ; son nom `collector-photos-<empreinte du compte>` est calculé à l'identique par Terraform et `tools/lib.sh`, sans numéro de compte dans le dépôt. Les groupes de journaux `/collector/app` et `/collector/audit` sont créés par Terraform avant Fluent Bit : sinon Fluent Bit les crée lui-même, sans rétention. La clé KMS des Secrets EKS (Trivy AWS-0039, décision de Martin du 2026-11-20) est persistante, car une clé créée à chaque session laisserait derrière elle une clé en suppression différée, facturée ; `infra/ephemere` la lit par son alias.

1. Copier `persistant/persistant.auto.tfvars.example` en `persistant.auto.tfvars` (ignoré par git) et renseigner l'ARN du certificat, le nom du CNAME de validation et l'adresse des alertes.
2. `bash tools/bootstrap.sh` : bucket créé s'il manque, puis `plan` ; **tout plan qui détruit ou remplace une ressource est refusé**. Premier lancement attendu : 4 imports, CAA, `aws_acm_certificate_validation`, configuration du bucket, 2 groupes de journaux, clé KMS et alias des Secrets EKS, mises à jour d'étiquettes. Second lancement : « aucun changement » (AC1).
3. Copier l'état hors poste, chiffré (MI2).

## Couche éphémère

- Variables fournies par `tools/up.sh` : `session`, `ip_admin`. Surcharges dans `ephemere.auto.tfvars` (modèle `.example`).
- Garde-fous de l'IaC (G4) : version EKS ≥ 1.35, 2 à 4 nœuds, rétention des journaux ≤ 7 jours ; un `plan` hors plafond échoue avant toute création.
- `alb_ouvert` reste `false` : `up.sh` le force par `-var` (au-dessus de tout `*.auto.tfvars`) ; une précondition refuse `true` sans `preuve_controle` égale à la session, posée par `tools/alb.sh ouvrir` après un contrôle vert (ME5, US-004 AC10). ME4 et ME8 restent à prouver avant la démonstration.
- IMDS : limite de sauts 2 par défaut (D03, 2026-10-26) ; politique de sortie sans `169.254.169.254/32` dans chaque namespace applicatif, `default` compris (`k8s/socle.yaml`, constat XP01 session 1), vérifiée par le contrôle après déploiement sur l'union des politiques de chaque namespace (`tools/sortie-sans-imds.jq`) ; durcissement complet en US-006.
- Descriptions des règles de groupes de sécurité en ASCII sans apostrophe : EC2 refuse l'apostrophe (XP01 session 1).

## Cluster

| Chemin | Contenu |
|---|---|
| `k8s/socle.yaml` | Politique `sortie-sans-imds` du namespace `default` ; appliqué par `up-flux.sh` avant tout chart |
| `k8s/secrets-autorises.txt` | Secrets que chaque application peut monter (X20) ; tout autre montage rend le contrôle rouge |
| `k8s/application.yaml` | Namespaces de l'application (un par service, `front` compris ; `collector` pour Flux, le compte `deployeur` et les versions Helm), Pod Security `restricted`, refus par défaut en entrée et en sortie, DNS, sortie 5432 des pods de migration (le premier crochet précède les politiques du chart ; `base-locale` couvre k3d, où un `ipBlock` ne couvre pas les adresses des pods) ; appliqué par `tools/up-flux.sh` avant les Secrets (US-006 AC1, US-007) |
| `k8s/base-locale.yaml` | k3d seulement : PostgreSQL de session à la place de RDS, mot de passe maître dans le Secret `base-maitre` posé par `up-flux.sh` |
| `application/` | Chart de l'application : quatre services, front, Keycloak, migrations, flux réseau autorisés ([README](application/README.md)) |
| `flux/` | `controleurs.yaml` : `Role` des contrôleurs Flux limitée à `collector` (à la place de `crd-controller` et de `cluster-admin`) et sortie DNS, 443, 6443 sans IMDS ; `deployeur.yaml` : compte `deployeur` des `HelmRelease`, `ClusterRole` liée par `RoleBinding` dans chaque namespace applicatif, Secrets dans `collector` seul, usurpation réservée au contrôleur Helm ; `sources.yaml` : `main` (chart) avec `env/demo` (empreintes) incluse, épinglées par `up-flux.sh` sur les commits vérifiés ; `releases.yaml` : canal puis application, retour arrière automatique (US-007) |
| `ephemere/application.tf` | Règles de l'ALB vers les API (`/marche/*`, `/interactions/*`, `/transactions/*`) et `auth.` (liste blanche et refus des traversées de XP03), `application_active` vrai par défaut ; `/interne` en 404, aucun chemin `interne` admis par validation ; NodePorts en sortie `nodeports_application`, lus par `up-flux.sh` |

## Contrôles hors AWS

```bash
terraform -chdir=infra/persistant init -backend=false && terraform -chdir=infra/persistant validate
terraform -chdir=infra/ephemere init -backend=false && terraform -chdir=infra/ephemere validate
terraform fmt -check -recursive infra
helm lint infra/application -f infra/application/valeurs-aws.yaml -f <empreintes>
bash tools/ci/trivy.sh config infra infra
```
