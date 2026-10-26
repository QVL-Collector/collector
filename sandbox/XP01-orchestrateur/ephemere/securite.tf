resource "aws_security_group" "alb" {
  name        = "${local.nom}-alb"
  description = "ALB du bac a sable, restreint au poste (MX2)"
  vpc_id      = aws_vpc.principal.id

  tags = { Name = "${local.nom}-alb" }
}

resource "aws_vpc_security_group_ingress_rule" "alb" {
  for_each = toset(["80", "443"])

  security_group_id = aws_security_group.alb.id
  description       = "Port ${each.key} depuis le poste seul"
  cidr_ipv4         = var.ip_admin
  ip_protocol       = "tcp"
  from_port         = tonumber(each.key)
  to_port           = tonumber(each.key)
}

resource "aws_vpc_security_group_egress_rule" "alb_cibles" {
  security_group_id = aws_security_group.alb.id
  description       = "Vers les cibles du VPC"
  cidr_ipv4         = var.cidr_vpc
  ip_protocol       = "tcp"
  from_port         = local.port_cible
  to_port           = local.port_cible
}

resource "aws_security_group" "rds" {
  name        = "${local.nom}-rds"
  description = "RDS du bac a sable, 5432 depuis les seuls clients autorises"
  vpc_id      = aws_vpc.principal.id

  tags = { Name = "${local.nom}-rds" }
}

resource "aws_vpc_security_group_ingress_rule" "rds_noeuds" {
  count = local.eks ? 1 : 0

  security_group_id            = aws_security_group.rds.id
  description                  = "PostgreSQL depuis les noeuds EKS, pods filtres par politique reseau"
  referenced_security_group_id = aws_eks_cluster.principal[0].vpc_config[0].cluster_security_group_id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
}

resource "aws_vpc_security_group_ingress_rule" "rds_pdm" {
  count = local.ecs ? 1 : 0

  security_group_id            = aws_security_group.rds.id
  description                  = "PostgreSQL depuis pdm-test seul"
  referenced_security_group_id = aws_security_group.ecs["pdm-test"].id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
}
