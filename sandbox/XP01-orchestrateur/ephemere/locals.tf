locals {
  nom = "collector-xp01"
  eks = var.orchestrateur == "eks"
  ecs = var.orchestrateur == "ecs"

  etiquettes = {
    projet  = "collector"
    couche  = "ephemere"
    xp      = "XP01"
    session = var.session
    outil   = "terraform"
  }

  lab_role_arn = "arn:${data.aws_partition.courante.partition}:iam::${data.aws_caller_identity.courante.account_id}:role/LabRole"

  cidr_publics = [for i in range(2) : cidrsubnet(var.cidr_vpc, 8, i)]
  cidr_prives  = [for i in range(2) : cidrsubnet(var.cidr_vpc, 4, i + 1)]

  port_cible = local.eks ? var.nodeport_test : var.port_test
}
