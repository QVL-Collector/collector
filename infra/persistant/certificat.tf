locals {
  validation = one(distinct([
    for option in aws_acm_certificate.collector.domain_validation_options : {
      nom    = trimsuffix(option.resource_record_name, ".")
      type   = option.resource_record_type
      valeur = option.resource_record_value
    }
  ]))
}

resource "aws_acm_certificate" "collector" {
  domain_name               = var.domaine
  subject_alternative_names = ["*.${var.domaine}"]
  validation_method         = "DNS"
  key_algorithm             = "RSA_2048"

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_route53_record" "validation" {
  zone_id = aws_route53_zone.collector.zone_id
  name    = local.validation.nom
  type    = local.validation.type
  ttl     = 300
  records = [local.validation.valeur]

  lifecycle {
    precondition {
      condition     = local.validation.nom == var.nom_validation
      error_message = "nom_validation ne correspond pas à l'enregistrement demandé par le certificat."
    }
  }
}

resource "aws_acm_certificate_validation" "collector" {
  certificate_arn         = aws_acm_certificate.collector.arn
  validation_record_fqdns = [aws_route53_record.validation.fqdn]
}
