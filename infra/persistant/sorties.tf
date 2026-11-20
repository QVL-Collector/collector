output "id_zone" {
  value = aws_route53_zone.collector.zone_id
}

output "serveurs_noms" {
  description = "Serveurs à poser en NS dans Cloudflare (délégation, D08 ME10)."
  value       = aws_route53_zone.collector.name_servers
}

output "bucket_photos" {
  value = local.bucket_photos
}

output "groupes_journaux" {
  value = {
    app   = aws_cloudwatch_log_group.app.name
    audit = aws_cloudwatch_log_group.audit.name
  }
}

output "cle_secrets_eks" {
  description = "Clé KMS des Secrets EKS, lue par infra/ephemere via son alias (AWS-0039)."
  value = {
    alias = aws_kms_alias.secrets_eks.name
    arn   = aws_kms_key.secrets_eks.arn
  }
}
