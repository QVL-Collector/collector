# infra — IaC de l'environnement AWS

Deux configurations racines Terraform, sans module interne ([D11](../docs/decisions/D11-iac-allumage-cout.md)). État local hors dépôt (`*.tfstate` ignoré), aucun secret dans l'état, Helm hors de Terraform. Scripts d'allumage et d'extinction dans [tools/](../tools/).

| Racine | Contenu | Cycle de vie |
|---|---|---|
| `persistant/` | Zone Route 53 `collector.qvl-project.com`, CAA `amazon.com` (ME7), certificat ACM `collector.qvl-project.com` + `*.collector.qvl-project.com` et son CNAME de validation, budget `collector-budget` (G6) | Appliquée une fois ; zone et certificat protégés par `prevent_destroy` (D08) |
| `ephemere/` | VPC sur deux zones, NAT unique, point de terminaison S3, ALB (TLS 1.2 minimum, HSTS, 80 → 443, 404 hors liste blanche), alias DNS, RDS PostgreSQL privée ; orchestrateur selon `orchestrateur` : `eks` (cluster, modèle de lancement IMDSv2 à 2 sauts, node group ≤ 4 nœuds) ou `ecs` (cluster Fargate) | Créée par `tools/up.sh`, détruite par `tools/down.sh` à chaque session |

Versions épinglées : Terraform 1.14.5, provider AWS 6.70.0 (`.terraform.lock.hcl` versionné dans chaque racine).

## Couche persistante : reprise des ressources de XP00

La zone, le certificat, son CNAME de validation et le budget ont été créés en CLI par XP00. Les blocs `import` de `persistant/import.tf` les reprennent au premier `apply`, sans recréation.

1. Copier `persistant/persistant.auto.tfvars.example` en `persistant.auto.tfvars` (ignoré par git) et renseigner l'ARN du certificat, le nom du CNAME de validation et l'adresse des alertes.
2. `terraform -chdir=infra/persistant init`, puis `plan` : attendu 4 imports, la création du CAA et de `aws_acm_certificate_validation`, des mises à jour d'étiquettes ; **aucune destruction ni remplacement**. Sinon, corriger la configuration avant tout `apply`.
3. `apply`, puis copier l'état hors poste, chiffré (MI2).

## Couche éphémère

- Variables fournies par `tools/up.sh` : `orchestrateur`, `session`, `ip_admin`. Surcharges dans `ephemere.auto.tfvars` (modèle `.example`).
- Garde-fous de l'IaC (G4) : version EKS ≥ 1.35, 2 à 4 nœuds, rétention des journaux ≤ 7 jours ; un `plan` hors plafond échoue avant toute création.
- `alb_ouvert` reste `false` : l'ALB n'accepte que l'adresse du poste tant que X17, X20, ME4 et ME8 ne sont pas verts (ME5, US-004 AC10).
- IMDS : limite de sauts 2 par défaut (D03, 2026-10-26) ; le filtre réseau qui refuse `169.254.169.254/32` aux pods est livré par le chart (US-006).

## Contrôles hors AWS

```bash
terraform -chdir=infra/persistant init -backend=false && terraform -chdir=infra/persistant validate
terraform -chdir=infra/ephemere init -backend=false && terraform -chdir=infra/ephemere validate
terraform fmt -check -recursive infra
```
