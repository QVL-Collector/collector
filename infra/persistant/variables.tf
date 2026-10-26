variable "region" {
  description = "Région du Learner Lab."
  type        = string
  default     = "us-east-1"

  validation {
    condition     = var.region == "us-east-1"
    error_message = "Le Learner Lab n'autorise que us-east-1."
  }
}

variable "domaine" {
  description = "Sous-domaine du projet, délégué depuis Cloudflare à la zone Route 53 (D08)."
  type        = string
  default     = "collector.qvl-project.com"
}

variable "id_zone" {
  description = "Identifiant de la zone Route 53 créée par XP00 S6, reprise par le bloc import."
  type        = string
  default     = "Z09494721JNIE0RFRPOI0"
}

variable "arn_certificat" {
  description = "ARN du certificat ACM créé par XP00 S7, fourni par persistant.auto.tfvars non versionné."
  type        = string

  validation {
    condition     = can(regex("^arn:aws:acm:us-east-1:[0-9]{12}:certificate/[0-9a-f-]{36}$", var.arn_certificat))
    error_message = "arn_certificat est un ARN de certificat ACM de us-east-1."
  }
}

variable "nom_validation" {
  description = "Nom du CNAME de validation ACM, sans point final, fourni par persistant.auto.tfvars non versionné."
  type        = string

  validation {
    condition     = can(regex("^_[0-9a-f]+[.]collector[.]qvl-project[.]com$", var.nom_validation))
    error_message = "nom_validation est de la forme _<empreinte>.collector.qvl-project.com, sans point final."
  }
}

variable "budget_usd" {
  description = "Budget réel du Learner Lab, lu par XP00 S1 (G6)."
  type        = number
  default     = 50
}

variable "seuils_alerte_budget" {
  description = "Seuils des alertes AWS Budgets, en pourcentage du budget (G6)."
  type        = list(number)
  default     = [50, 80]
}

variable "courriel_alertes_budget" {
  description = "Destinataire des alertes AWS Budgets, fourni par persistant.auto.tfvars non versionné."
  type        = string
  sensitive   = true

  validation {
    condition     = can(regex("^[^@ ]+@[^@ ]+[.][a-z]{2,}$", var.courriel_alertes_budget))
    error_message = "courriel_alertes_budget est une adresse de courriel."
  }
}
