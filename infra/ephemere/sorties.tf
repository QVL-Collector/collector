output "orchestrateur" {
  value = var.orchestrateur
}

output "url" {
  value = "https://${var.domaine}${var.chemin_sante}"
}

output "id_vpc" {
  value = aws_vpc.principal.id
}

output "sous_reseaux_prives" {
  value = aws_subnet.prive[*].id
}

output "id_nat" {
  value = aws_nat_gateway.unique.id
}

output "id_allocation_eip" {
  value = aws_eip.nat.allocation_id
}

output "arn_alb" {
  value = aws_lb.public.arn
}

output "arn_groupe_cible_front" {
  value = aws_lb_target_group.front.arn
}

output "id_rds" {
  value = aws_db_instance.principale.identifier
}

output "nom_cluster_eks" {
  value = local.eks ? aws_eks_cluster.principal[0].name : ""
}

output "nom_node_group" {
  value = local.eks ? aws_eks_node_group.principal[0].node_group_name : ""
}

output "nom_cluster_ecs" {
  value = local.ecs ? aws_ecs_cluster.principal[0].name : ""
}

output "groupe_securite_front_ecs" {
  value = local.ecs ? aws_security_group.front_ecs[0].id : ""
}
