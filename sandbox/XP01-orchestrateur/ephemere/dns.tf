resource "aws_route53_record" "alb" {
  for_each = toset(["", "auth."])

  zone_id = data.aws_route53_zone.collector.zone_id
  name    = "${each.key}${var.domaine}"
  type    = "A"

  alias {
    name                   = aws_lb.public.dns_name
    zone_id                = aws_lb.public.zone_id
    evaluate_target_health = false
  }
}
