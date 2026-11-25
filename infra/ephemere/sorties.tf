output "url" {
  value = "https://${var.domaine}${var.chemin_sante}"
}

output "id_vpc" {
  value = aws_vpc.principal.id
}

output "sous_reseaux_prives" {
  value = aws_subnet.prive[*].id
}

output "arn_alb" {
  value = aws_lb.public.arn
}

output "nom_alb" {
  value = aws_lb.public.name
}

output "dns_alb" {
  value = aws_lb.public.dns_name
}

output "arn_ecouteur_https" {
  value = aws_lb_listener.https.arn
}

output "arn_ecouteur_http" {
  value = aws_lb_listener.http.arn
}

output "arns_regles_https" {
  description = "Règles de la liste blanche de l'écouteur 443, seules admises par le contrôle après déploiement (XE4)."
  value = concat(
    [aws_lb_listener_rule.front.arn],
    aws_lb_listener_rule.interne_refus[*].arn,
    aws_lb_listener_rule.auth_refus[*].arn,
    [for r in values(aws_lb_listener_rule.application) : r.arn],
  )
}

output "nodeports_application" {
  value = { for nom, c in local.cibles : nom => c.nodeport }
}

output "id_groupe_securite_alb" {
  value = aws_security_group.alb.id
}

output "alb_ouvert" {
  value = var.alb_ouvert
}

output "nodeport_front" {
  value = var.nodeport_front
}

output "adresse_rds" {
  value = aws_db_instance.principale.address
}

output "point_api_eks" {
  value = aws_eks_cluster.principal.endpoint
}

output "version_ami_noeuds" {
  value = data.aws_ssm_parameter.version_ami_eks.insecure_value
}

output "arn_groupe_cible_front" {
  value = aws_lb_target_group.front.arn
}

output "id_rds" {
  value = aws_db_instance.principale.identifier
}

output "nom_node_group" {
  value = aws_eks_node_group.principal.node_group_name
}
