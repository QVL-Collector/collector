import {
  to = aws_route53_zone.collector
  id = var.id_zone
}

import {
  to = aws_acm_certificate.collector
  id = var.arn_certificat
}

import {
  to = aws_route53_record.validation
  id = "${var.id_zone}_${var.nom_validation}_CNAME"
}

import {
  to = aws_budgets_budget.collector
  id = "${data.aws_caller_identity.courante.account_id}:collector-budget"
}
