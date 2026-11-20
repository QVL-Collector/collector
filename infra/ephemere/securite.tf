resource "aws_security_group" "alb" {
  name        = "${local.nom}-alb"
  description = "ALB du POC, restreint au poste hors demonstration (ME5)"
  vpc_id      = aws_vpc.principal.id

  tags = { Name = "${local.nom}-alb" }
}

resource "aws_vpc_security_group_ingress_rule" "alb" {
  for_each = toset(["80", "443"])

  security_group_id = aws_security_group.alb.id
  description       = var.alb_ouvert ? "Port ${each.key} ouvert a Internet (demonstration)" : "Port ${each.key} depuis le poste seul"
  cidr_ipv4         = local.source_alb
  ip_protocol       = "tcp"
  from_port         = tonumber(each.key)
  to_port           = tonumber(each.key)

  lifecycle {
    precondition {
      condition     = !var.alb_ouvert || var.preuve_controle == var.session
      error_message = "Ouverture de l'ALB refusée : contrôle après déploiement non vert pour cette session (US-004 AC10). Passer par tools/alb.sh ouvrir."
    }
  }
}

resource "aws_vpc_security_group_egress_rule" "alb_cibles" {
  security_group_id = aws_security_group.alb.id
  description       = "Vers les cibles du VPC"
  cidr_ipv4         = var.cidr_vpc
  ip_protocol       = "tcp"
  from_port         = var.nodeport_front
  to_port           = var.nodeport_front
}

resource "aws_security_group" "rds" {
  name        = "${local.nom}-rds"
  description = "RDS du POC, 5432 depuis les seuls clients autorises (MI10)"
  vpc_id      = aws_vpc.principal.id

  tags = { Name = "${local.nom}-rds" }
}
