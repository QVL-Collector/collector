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

output "id_rds" {
  value = aws_db_instance.test.identifier
}

output "nom_cluster_eks" {
  value = local.eks ? aws_eks_cluster.principal[0].name : ""
}

output "nom_node_group" {
  value = local.eks ? aws_eks_node_group.principal[0].node_group_name : ""
}

output "ami_noeuds" {
  value = local.eks ? data.aws_ssm_parameter.ami_eks[0].insecure_value : ""
}

output "nom_cluster_ecs" {
  value = local.ecs ? aws_ecs_cluster.principal[0].name : ""
}

output "nom_service_ecs" {
  value = local.ecs ? aws_ecs_service.pdm_test[0].name : ""
}

output "groupes_securite_ecs" {
  value = { for role, sg in aws_security_group.ecs : role => sg.id }
}
