variable "application_active" {
  description = "Règles de l'ALB vers les API et Keycloak (US-007, chart infra/application). false : front seul (US-004)."
  type        = bool
  default     = false
}

variable "cibles_application" {
  description = "Cibles de l'ALB par NodePort (D08 § 9.3). Jamais routés : port interne 8081 de la Place de marché (réserver), gestion, métriques."
  type = map(object({
    nodeport = number
    auth     = bool
    chemins  = list(string)
    sante    = string
    codes    = string
    priorite = number
  }))
  default = {
    keycloak = {
      nodeport = 30081
      auth     = true
      chemins = [
        "/realms/collector/.well-known/openid-configuration",
        "/realms/collector/protocol/openid-connect/*",
        "/realms/collector/login-actions/*",
        "/resources/*",
      ]
      sante    = "/realms/collector/.well-known/openid-configuration"
      codes    = "200"
      priorite = 30
    }
    place-de-marche = {
      nodeport = 30082
      auth     = false
      chemins  = ["/marche/*"]
      sante    = "/marche/v1/categories"
      codes    = "200"
      priorite = 40
    }
    interactions = {
      nodeport = 30083
      auth     = false
      chemins  = ["/interactions/*"]
      sante    = "/"
      codes    = "404"
      priorite = 50
    }
    transactions = {
      nodeport = 30084
      auth     = false
      chemins  = ["/transactions/*"]
      sante    = "/"
      codes    = "404"
      priorite = 60
    }
  }

  validation {
    condition = alltrue([for c in values(var.cibles_application) :
      c.nodeport >= 30000 && c.nodeport <= 32767 && alltrue([for p in c.chemins : !strcontains(p, "interne")])
    ])
    error_message = "NodePort entre 30000 et 32767 ; aucun chemin interne (réserver) ne peut être routé."
  }
}

locals {
  cibles       = var.application_active ? var.cibles_application : {}
  hote_auth    = "auth.${var.domaine}"
  refus_chemin = ".*(\\.\\.|%2[eEfF]|%5[cC]|%25|\\\\).*"
}

resource "aws_lb_target_group" "application" {
  for_each = local.cibles

  name                 = "${local.nom}-${substr(replace(each.key, "-", ""), 0, 12)}"
  vpc_id               = aws_vpc.principal.id
  protocol             = "HTTP"
  port                 = each.value.nodeport
  target_type          = "instance"
  deregistration_delay = 10

  health_check {
    path                = each.value.sante
    matcher             = each.value.codes
    interval            = 10
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }
}

resource "aws_autoscaling_attachment" "application" {
  for_each = local.cibles

  autoscaling_group_name = aws_eks_node_group.principal.resources[0].autoscaling_groups[0].name
  lb_target_group_arn    = aws_lb_target_group.application[each.key].arn
}

resource "aws_lb_listener_rule" "interne_refus" {
  count = var.application_active ? 1 : 0

  listener_arn = aws_lb_listener.https.arn
  priority     = 10

  condition {
    host_header {
      values = [var.domaine]
    }
  }

  condition {
    path_pattern {
      values = ["*/interne/*", "/interne*"]
    }
  }

  action {
    type = "fixed-response"

    fixed_response {
      content_type = "text/plain"
      message_body = "Not Found"
      status_code  = "404"
    }
  }
}

resource "aws_lb_listener_rule" "auth_refus" {
  count = var.application_active ? 1 : 0

  listener_arn = aws_lb_listener.https.arn
  priority     = 20

  condition {
    host_header {
      values = [local.hote_auth]
    }
  }

  condition {
    path_pattern {
      regex_values = [local.refus_chemin]
    }
  }

  action {
    type = "fixed-response"

    fixed_response {
      content_type = "text/plain"
      message_body = "Not Found"
      status_code  = "404"
    }
  }
}

resource "aws_lb_listener_rule" "application" {
  for_each = local.cibles

  listener_arn = aws_lb_listener.https.arn
  priority     = each.value.priorite

  condition {
    host_header {
      values = [each.value.auth ? local.hote_auth : var.domaine]
    }
  }

  condition {
    path_pattern {
      values = each.value.chemins
    }
  }

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.application[each.key].arn
  }
}

resource "aws_vpc_security_group_egress_rule" "alb_application" {
  for_each = local.cibles

  security_group_id = aws_security_group.alb.id
  description       = "Vers le NodePort ${each.key}"
  cidr_ipv4         = var.cidr_vpc
  ip_protocol       = "tcp"
  from_port         = each.value.nodeport
  to_port           = each.value.nodeport
}

resource "aws_vpc_security_group_ingress_rule" "noeuds_application" {
  for_each = local.cibles

  security_group_id            = aws_eks_cluster.principal.vpc_config[0].cluster_security_group_id
  description                  = "NodePort ${each.key} depuis le seul ALB (MS2)"
  referenced_security_group_id = aws_security_group.alb.id
  ip_protocol                  = "tcp"
  from_port                    = each.value.nodeport
  to_port                      = each.value.nodeport
}
