locals {
  arn_compte = "arn:aws:iam::${data.aws_caller_identity.courante.account_id}"
}

data "aws_iam_policy_document" "cle_secrets_eks" {
  statement {
    sid       = "AdministrationParIam"
    actions   = ["kms:*"]
    resources = ["*"]

    principals {
      type        = "AWS"
      identifiers = ["${local.arn_compte}:root"]
    }
  }

  statement {
    sid       = "UsageParLeClusterEks"
    actions   = ["kms:Encrypt", "kms:Decrypt", "kms:DescribeKey"]
    resources = ["*"]

    principals {
      type        = "AWS"
      identifiers = ["${local.arn_compte}:role/LabRole"]
    }
  }

  statement {
    sid       = "OctroisPourEks"
    actions   = ["kms:CreateGrant"]
    resources = ["*"]

    principals {
      type        = "AWS"
      identifiers = ["${local.arn_compte}:role/LabRole"]
    }

    condition {
      test     = "Bool"
      variable = "kms:GrantIsForAWSResource"
      values   = ["true"]
    }
  }
}

resource "aws_kms_key" "secrets_eks" {
  description             = "Chiffrement d'enveloppe des Secrets du cluster EKS Collector"
  enable_key_rotation     = true
  deletion_window_in_days = 7
  policy                  = data.aws_iam_policy_document.cle_secrets_eks.json
}

resource "aws_kms_alias" "secrets_eks" {
  name          = var.alias_cle_secrets_eks
  target_key_id = aws_kms_key.secrets_eks.key_id
}
