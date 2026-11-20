locals {
  nom = "collector"

  etiquettes = {
    projet  = "collector"
    couche  = "ephemere"
    session = var.session
    outil   = "terraform"
  }

  lab_role_arn = "arn:${data.aws_partition.courante.partition}:iam::${data.aws_caller_identity.courante.account_id}:role/LabRole"

  cidr_publics = [for i in range(2) : cidrsubnet(var.cidr_vpc, 8, i)]
  cidr_prives  = [for i in range(2) : cidrsubnet(var.cidr_vpc, 4, i + 1)]

  source_alb   = var.alb_ouvert ? "0.0.0.0/0" : var.ip_admin
  hsts         = "max-age=31536000; includeSubDomains"
  nom_journaux = "/aws/eks/${local.nom}/cluster"
}
