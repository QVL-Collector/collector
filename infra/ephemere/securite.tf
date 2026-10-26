resource "aws_security_group" "alb" {
  name        = "${local.nom}-alb"
  description = "ALB du POC, restreint au poste hors demonstration (ME5)"
  vpc_id      = aws_vpc.principal.id

  tags = { Name = "${local.nom}-alb" }
}

resource "aws_vpc_security_group_ingress_rule" "alb" {
  for_each = { for paire in setproduct(["80", "443"], local.sources_alb) : "${paire[0]}-${paire[1]}" => paire }

  security_group_id = aws_security_group.alb.id
  description       = var.alb_ouvert ? "Port ${each.value[0]} ouvert a Internet (demonstration)" : "Port ${each.value[0]} depuis le poste seul"
  cidr_ipv4         = each.value[1]
  ip_protocol       = "tcp"
  from_port         = tonumber(each.value[0])
  to_port           = tonumber(each.value[0])
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
  description = "RDS du POC, 5432 depuis les seuls clients autorises (MI10)"
  vpc_id      = aws_vpc.principal.id

  tags = { Name = "${local.nom}-rds" }
}
