data "aws_caller_identity" "courante" {}

data "aws_partition" "courante" {}

data "aws_route53_zone" "collector" {
  name         = var.domaine
  private_zone = false
}

data "aws_acm_certificate" "collector" {
  domain      = var.domaine
  statuses    = ["ISSUED"]
  most_recent = true
}

data "aws_ssm_parameter" "ami_eks" {
  count = local.eks ? 1 : 0
  name  = "/aws/service/eks/optimized-ami/${var.version_eks}/amazon-linux-2023/x86_64/standard/recommended/image_id"
}

data "aws_ssm_parameter" "version_ami_eks" {
  count = local.eks ? 1 : 0
  name  = "/aws/service/eks/optimized-ami/${var.version_eks}/amazon-linux-2023/x86_64/standard/recommended/release_version"
}
