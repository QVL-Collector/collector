# Variante de repli (D03 option B), active si le verdict XP01 du 2026-11-02 retient ECS Fargate.
# Les définitions de tâches et services du POC sont ajoutées après le verdict.

resource "aws_cloudwatch_log_group" "ecs" {
  count = local.ecs ? 1 : 0

  name              = "/collector/ecs"
  retention_in_days = var.retention_journaux_jours
}

resource "aws_ecs_cluster" "principal" {
  count = local.ecs ? 1 : 0

  name = local.nom

  setting {
    name  = "containerInsights"
    value = "disabled"
  }
}

resource "aws_ecs_cluster_capacity_providers" "principal" {
  count = local.ecs ? 1 : 0

  cluster_name       = aws_ecs_cluster.principal[0].name
  capacity_providers = ["FARGATE"]

  default_capacity_provider_strategy {
    capacity_provider = "FARGATE"
    weight            = 1
  }
}

resource "aws_security_group" "front_ecs" {
  count = local.ecs ? 1 : 0

  name        = "${local.nom}-front"
  description = "Taches du front, joignables depuis l'ALB seul"
  vpc_id      = aws_vpc.principal.id

  tags = { Name = "${local.nom}-front" }
}

resource "aws_vpc_security_group_ingress_rule" "front_ecs_alb" {
  count = local.ecs ? 1 : 0

  security_group_id            = aws_security_group.front_ecs[0].id
  description                  = "Port du front depuis l'ALB seul"
  referenced_security_group_id = aws_security_group.alb.id
  ip_protocol                  = "tcp"
  from_port                    = var.port_front
  to_port                      = var.port_front
}

resource "aws_vpc_security_group_egress_rule" "front_ecs_sortie" {
  count = local.ecs ? 1 : 0

  security_group_id = aws_security_group.front_ecs[0].id
  description       = "Sortie : images par la NAT"
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
}
