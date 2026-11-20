resource "aws_route53_zone" "collector" {
  name          = var.domaine
  comment       = "collector-persistant"
  force_destroy = false

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_route53_record" "caa" {
  zone_id = aws_route53_zone.collector.zone_id
  name    = var.domaine
  type    = "CAA"
  ttl     = 3600
  records = [
    "0 issue \"amazon.com\"",
    "0 issuewild \"amazon.com\"",
  ]
}
