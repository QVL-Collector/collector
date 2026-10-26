resource "aws_budgets_budget" "collector" {
  name         = "collector-budget"
  budget_type  = "COST"
  limit_amount = format("%.1f", var.budget_usd)
  limit_unit   = "USD"
  time_unit    = "ANNUALLY"

  dynamic "notification" {
    for_each = toset(var.seuils_alerte_budget)

    content {
      notification_type          = "ACTUAL"
      comparison_operator        = "GREATER_THAN"
      threshold                  = notification.value
      threshold_type             = "PERCENTAGE"
      subscriber_email_addresses = [var.courriel_alertes_budget]
    }
  }
}
