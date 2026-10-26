resource "aws_eks_cluster" "principal" {
  count = local.eks ? 1 : 0

  name                          = local.nom
  version                       = var.version_eks
  role_arn                      = local.lab_role_arn
  bootstrap_self_managed_addons = false

  access_config {
    authentication_mode                         = "API"
    bootstrap_cluster_creator_admin_permissions = true
  }

  upgrade_policy {
    support_type = "STANDARD"
  }

  vpc_config {
    subnet_ids              = aws_subnet.prive[*].id
    endpoint_private_access = true
    endpoint_public_access  = true
    public_access_cidrs     = [var.ip_admin]
  }
}

resource "aws_eks_addon" "avant_noeuds" {
  for_each = local.eks ? toset(["vpc-cni", "kube-proxy"]) : toset([])

  cluster_name                = aws_eks_cluster.principal[0].name
  addon_name                  = each.key
  addon_version               = var.versions_addons[each.key]
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"

  configuration_values = each.key == "vpc-cni" ? jsonencode({
    enableNetworkPolicy = "true"
    env = {
      ENABLE_PREFIX_DELEGATION = var.delegation_prefixes ? "true" : "false"
    }
  }) : null
}

resource "aws_launch_template" "noeuds" {
  count = local.eks ? 1 : 0

  name                   = "${local.nom}-noeuds"
  update_default_version = true

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = var.imds_limite_sauts
    instance_metadata_tags      = "disabled"
  }

  block_device_mappings {
    device_name = "/dev/xvda"

    ebs {
      volume_size           = 20
      volume_type           = "gp3"
      encrypted             = true
      delete_on_termination = true
    }
  }

  dynamic "tag_specifications" {
    for_each = toset(["instance", "volume", "network-interface"])

    content {
      resource_type = tag_specifications.key
      tags          = merge(local.etiquettes, { Name = "${local.nom}-noeud" })
    }
  }
}

resource "aws_eks_node_group" "principal" {
  count = local.eks ? 1 : 0

  cluster_name    = aws_eks_cluster.principal[0].name
  node_group_name = "${local.nom}-noeuds"
  node_role_arn   = local.lab_role_arn
  subnet_ids      = aws_subnet.prive[*].id
  version         = var.version_eks
  ami_type        = "AL2023_x86_64_STANDARD"
  release_version = data.aws_ssm_parameter.version_ami_eks[0].insecure_value
  capacity_type   = "ON_DEMAND"
  instance_types  = [var.type_noeuds]

  scaling_config {
    desired_size = var.noeuds_souhaites
    min_size     = var.noeuds_souhaites
    max_size     = var.noeuds_max
  }

  update_config {
    max_unavailable = 1
  }

  launch_template {
    id      = aws_launch_template.noeuds[0].id
    version = aws_launch_template.noeuds[0].latest_version
  }

  labels = {
    "collector/xp" = "xp01"
  }

  lifecycle {
    ignore_changes = [scaling_config[0].desired_size]
  }

  depends_on = [aws_eks_addon.avant_noeuds]
}

resource "aws_eks_addon" "apres_noeuds" {
  for_each = local.eks ? toset(["coredns", "metrics-server"]) : toset([])

  cluster_name                = aws_eks_cluster.principal[0].name
  addon_name                  = each.key
  addon_version               = var.versions_addons[each.key]
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"

  depends_on = [aws_eks_node_group.principal]
}

resource "aws_vpc_security_group_ingress_rule" "noeuds_nodeport" {
  count = local.eks ? 1 : 0

  security_group_id            = aws_eks_cluster.principal[0].vpc_config[0].cluster_security_group_id
  description                  = "NodePort du service de test depuis l'ALB seul (MS2)"
  referenced_security_group_id = aws_security_group.alb.id
  ip_protocol                  = "tcp"
  from_port                    = var.nodeport_test
  to_port                      = var.nodeport_test
}

resource "aws_autoscaling_attachment" "noeuds" {
  count = local.eks ? 1 : 0

  autoscaling_group_name = aws_eks_node_group.principal[0].resources[0].autoscaling_groups[0].name
  lb_target_group_arn    = aws_lb_target_group.test.arn
}
