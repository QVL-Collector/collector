resource "aws_cloudwatch_log_group" "app" {
  name              = "/collector/app"
  retention_in_days = var.retention_journaux_app
}

resource "aws_cloudwatch_log_group" "audit" {
  name              = "/collector/audit"
  retention_in_days = var.retention_journaux_audit

  lifecycle {
    prevent_destroy = true
  }
}
