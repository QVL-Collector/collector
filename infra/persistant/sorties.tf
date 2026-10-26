output "id_zone" {
  value = aws_route53_zone.collector.zone_id
}

output "serveurs_noms" {
  description = "Serveurs à poser en NS dans Cloudflare (délégation, D08 ME10)."
  value       = aws_route53_zone.collector.name_servers
}
