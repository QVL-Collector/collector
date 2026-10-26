variable "orchestrateur" {
  description = "Variante allumée : eks (sessions 1 et 2) ou ecs (session Fargate)."
  type        = string

  validation {
    condition     = contains(["eks", "ecs"], var.orchestrateur)
    error_message = "orchestrateur vaut eks ou ecs."
  }
}

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
  description = "Étiquette de session (date et heure du up), fixée par up.sh et stable pendant la session."
  type        = string

  validation {
    condition     = can(regex("^[0-9]{8}T[0-9]{4}Z$", var.session))
    error_message = "session au format AAAAMMJJTHHMMZ."
  }
}

variable "ip_admin" {
  description = "Adresse publique du poste en /32 : API du cluster (MS1, MX3) et ALB (MX2)."
  type        = string

  validation {
    condition     = can(regex("^([0-9]{1,3}\\.){3}[0-9]{1,3}/32$", var.ip_admin))
    error_message = "ip_admin est une adresse IPv4 en /32."
  }
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
  description = "Plage du VPC du bac à sable."
  type        = string
  default     = "10.40.0.0/16"
}

variable "domaine" {
  description = "Zone Route 53 et certificat ACM existants (couche persistante), lus en data."
  type        = string
  default     = "collector.qvl-project.com"
}

variable "politique_tls" {
  description = "Politique TLS de l'écouteur HTTPS (ME7)."
  type        = string
  default     = "ELBSecurityPolicy-TLS13-1-2-Res-2021-06"
}

variable "chemin_sante" {
  description = "Chemin de contrôle de santé du service HTTP de test."
  type        = string
  default     = "/"
}

variable "version_eks" {
  description = "Version Kubernetes, en support standard (D11 G4)."
  type        = string
  default     = "1.35"

  validation {
    condition     = can(regex("^1\\.[0-9]+$", var.version_eks)) && tonumber(split(".", var.version_eks)[1]) >= 35
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

variable "delegation_prefixes" {
  description = "ENABLE_PREFIX_DELEGATION du CNI (essai de densité, session 2)."
  type        = bool
  default     = false
}

variable "type_noeuds" {
  description = "Type d'instance des nœuds."
  type        = string
  default     = "t3.medium"
}

variable "noeuds_souhaites" {
  description = "Nœuds au démarrage."
  type        = number
  default     = 2
}

variable "noeuds_max" {
  description = "Plafond du node group (MS10, D11 G4)."
  type        = number
  default     = 4

  validation {
    condition     = var.noeuds_max >= 2 && var.noeuds_max <= 4
    error_message = "Plafond de nœuds entre 2 et 4."
  }
}

variable "imds_limite_sauts" {
  description = "Limite de sauts IMDSv2 des nœuds. 1 : IMDS fermé aux pods hors hostNetwork (X16, repli MS3). 2 : IMDS ouvert aux pods, filtré par politique réseau (MS3 principal, G4 par IMDS)."
  type        = number
  default     = 1

  validation {
    condition     = contains([1, 2], var.imds_limite_sauts)
    error_message = "Limite de sauts 1 ou 2."
  }
}

variable "nodeport_test" {
  description = "NodePort fixe du service HTTP de test dans le chart (cible de l'ALB, EKS)."
  type        = number
  default     = 30080

  validation {
    condition     = var.nodeport_test >= 30000 && var.nodeport_test <= 32767
    error_message = "NodePort entre 30000 et 32767."
  }
}

variable "port_test" {
  description = "Port du conteneur du service HTTP de test (ECS)."
  type        = number
  default     = 8080
}

variable "image_test" {
  description = "Image du service HTTP de test, épinglée par empreinte (ECS)."
  type        = string
  default     = ""

  validation {
    condition     = var.image_test == "" || can(regex("@sha256:[0-9a-f]{64}$", var.image_test))
    error_message = "image_test est épinglée par empreinte (@sha256:...)."
  }
}

variable "image_sonde" {
  description = "Image des tâches de sonde (curl et AWS CLI), épinglée par empreinte (ECS)."
  type        = string
  default     = ""

  validation {
    condition     = var.image_sonde == "" || can(regex("@sha256:[0-9a-f]{64}$", var.image_sonde))
    error_message = "image_sonde est épinglée par empreinte (@sha256:...)."
  }
}

variable "taches_min" {
  description = "Tâches minimum du service de test (G5)."
  type        = number
  default     = 2
}

variable "taches_max" {
  description = "Tâches maximum du service de test (G5, MS10)."
  type        = number
  default     = 6

  validation {
    condition     = var.taches_max <= 6
    error_message = "Plafond de 6 tâches."
  }
}

variable "cible_cpu" {
  description = "Cible de CPU du suivi de cible ECS, en pourcentage (G5)."
  type        = number
  default     = 60
}

variable "version_postgres" {
  description = "Version PostgreSQL RDS (XP00 S10)."
  type        = string
  default     = "18.3"
}

variable "retention_journaux_jours" {
  description = "Rétention des groupes de journaux du bac à sable."
  type        = number
  default     = 1
}
