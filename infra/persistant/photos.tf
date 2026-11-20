locals {
  bucket_photos = "collector-photos-${substr(sha256(data.aws_caller_identity.courante.account_id), 0, 10)}"
  arn_photos    = "arn:aws:s3:::${local.bucket_photos}"
}

resource "aws_s3_bucket_public_access_block" "photos" {
  bucket = local.bucket_photos

  block_public_acls       = true
  ignore_public_acls      = true
  block_public_policy     = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "photos" {
  bucket = local.bucket_photos

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "photos" {
  bucket = local.bucket_photos

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_policy" "photos" {
  bucket = local.bucket_photos

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "RefusHorsTLS"
      Effect    = "Deny"
      Principal = "*"
      Action    = "s3:*"
      Resource  = [local.arn_photos, "${local.arn_photos}/*"]
      Condition = { Bool = { "aws:SecureTransport" = "false" } }
    }]
  })

  depends_on = [aws_s3_bucket_public_access_block.photos]
}

resource "aws_s3_bucket_lifecycle_configuration" "photos" {
  bucket = local.bucket_photos

  rule {
    id     = "depots-inacheves"
    status = "Enabled"

    filter {}

    abort_incomplete_multipart_upload {
      days_after_initiation = 1
    }
  }
}
