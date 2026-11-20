variable "region" {
  description = "Région du Learner Lab."
  type        = string
  default     = "us-east-1"

  validation {
    condition     = var.region == "us-east-1"
    error_message = "Le Learner Lab n'autorise que us-east-1."
  }
}

variable "session" {
  description = "Étiquette de session (date et heure du up), fixée par tools/up.sh et stable pendant la session (G3)."
  type        = string

  validation {
    condition     = can(regex("^[0-9]{8}T[0-9]{4}Z$", var.session))
    error_message = "session au format AAAAMMJJTHHMMZ."
  }
}

variable "ip_admin" {
  description = "Adresse publique du poste en /32 : API du cluster (MS1) et ALB hors démonstration (ME5)."
  type        = string

  validation {
    condition     = can(regex("^([0-9]{1,3}[.]){3}[0-9]{1,3}/32$", var.ip_admin))
    error_message = "ip_admin est une adresse IPv4 en /32."
  }
}

variable "alb_ouvert" {
  description = "Ouverture de l'ALB à Internet (ME5). Interdite tant que le contrôle après déploiement n'est pas vert (US-004 AC10) : tools/up.sh la force à false, seul tools/alb.sh ouvrir la pose."
  type        = bool
  default     = false
}

variable "preuve_controle" {
  description = "Session dont le contrôle après déploiement est vert, fournie par tools/alb.sh ; exigée égale à session pour ouvrir l'ALB (AC10)."
  type        = string
  default     = ""
}

variable "zones" {
  description = "Deux zones de disponibilité (us-east-1e exclue : pas de plan de contrôle EKS)."
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]

  validation {
    condition     = length(var.zones) == 2 && !contains(var.zones, "us-east-1e")
    error_message = "Deux zones, us-east-1e exclue."
  }
}

variable "cidr_vpc" {
  description = "Plage du VPC du POC (distincte de celle du bac à sable XP01)."
  type        = string
  default     = "10.30.0.0/16"
}

variable "domaine" {
  description = "Zone Route 53 et certificat ACM de la couche persistante, lus en data."
  type        = string
  default     = "collector.qvl-project.com"
}

variable "alias_cle_secrets_eks" {
  description = "Alias de la clé KMS des Secrets EKS de la couche persistante, lu en data (AWS-0039)."
  type        = string
  default     = "alias/collector-secrets-eks"
}

variable "politique_tls" {
  description = "Politique TLS de l'écouteur HTTPS (ME7) : TLS 1.2 minimum."
  type        = string
  default     = "ELBSecurityPolicy-TLS13-1-2-Res-2021-06"
}

variable "chemin_sante" {
  description = "Chemin de contrôle de santé du front nginx."
  type        = string
  default     = "/"
}

variable "version_eks" {
  description = "Version Kubernetes, en support standard (D11 G4, US-004 AC6)."
  type        = string
  default     = "1.35"

  validation {
    condition     = can(regex("^1[.][0-9]+$", var.version_eks)) && tonumber(split(".", var.version_eks)[1]) >= 35
    error_message = "Version EKS 1.35 au minimum."
  }
}

variable "versions_addons" {
  description = "Versions épinglées des modules complémentaires EKS (XP00 S3)."
  type        = map(string)
  default = {
    "vpc-cni"        = "v1.22.4-eksbuild.8"
    "kube-proxy"     = "v1.35.3-eksbuild.59"
    "coredns"        = "v1.13.2-eksbuild.59"
    "metrics-server" = "v0.9.0-eksbuild.14"
  }
}

variable "journaux_plan_controle" {
  description = "Types de journaux du plan de contrôle EKS (D09). Vide : aucun groupe créé."
  type        = list(string)
  default     = []

  validation {
    condition     = alltrue([for t in var.journaux_plan_controle : contains(["api", "audit", "authenticator", "controllerManager", "scheduler"], t)])
    error_message = "Types admis : api, audit, authenticator, controllerManager, scheduler."
  }
}

variable "type_noeuds" {
  description = "Type d'instance des nœuds."
  type        = string
  default     = "t3.medium"
}

variable "noeuds_souhaites" {
  description = "Nœuds au démarrage, plancher du node group."
  type        = number
  default     = 2

  validation {
    condition     = var.noeuds_souhaites >= 1 && var.noeuds_souhaites <= var.noeuds_max
    error_message = "Nœuds au démarrage entre 1 et noeuds_max."
  }
}

variable "noeuds_max" {
  description = "Plafond du node group (MS10, D11 G4, US-004 AC6)."
  type        = number
  default     = 4

  validation {
    condition     = var.noeuds_max >= 2 && var.noeuds_max <= 4
    error_message = "Plafond de nœuds entre 2 et 4."
  }
}

variable "imds_limite_sauts" {
  description = "Limite de sauts IMDSv2 des nœuds. 2 : décision D03 du 2026-10-26, IMDS joignable des pods et filtré par politique réseau (MS3 principal, US-006). 1 : repli, IMDS fermé aux pods hors hostNetwork."
  type        = number
  default     = 2

  validation {
    condition     = contains([1, 2], var.imds_limite_sauts)
    error_message = "Limite de sauts 1 ou 2."
  }
}

variable "nodeport_front" {
  description = "NodePort fixe du Service du front nginx (cible de l'ALB)."
  type        = number
  default     = 30080

  validation {
    condition     = var.nodeport_front >= 30000 && var.nodeport_front <= 32767
    error_message = "NodePort entre 30000 et 32767."
  }
}

variable "version_postgres" {
  description = "Version PostgreSQL RDS (XP00 S10)."
  type        = string
  default     = "18.3"
}

variable "retention_journaux_jours" {
  description = "Rétention des groupes de journaux éphémères (D09, G4)."
  type        = number
  default     = 1

  validation {
    condition     = contains([1, 3, 5, 7], var.retention_journaux_jours)
    error_message = "Rétention courte : 1, 3, 5 ou 7 jours."
  }
}

variable "delegation_prefixes" {
  description = "ENABLE_PREFIX_DELEGATION du CNI, à fixer selon la mesure de densité de XP01."
  type        = bool
  default     = false
}
