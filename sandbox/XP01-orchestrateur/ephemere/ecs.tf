locals {
  roles_ecs = local.ecs ? toset(["pdm-test", "transactions-test", "controle-test", "intrus"]) : toset([])

  journaux_ecs = {
    logDriver = "awslogs"
    options = {
      awslogs-group         = local.ecs ? aws_cloudwatch_log_group.ecs[0].name : ""
      awslogs-region        = var.region
      awslogs-stream-prefix = "xp01"
    }
  }

  sonde = {
    name             = "sonde"
    image            = var.image_sonde
    essential        = true
    entryPoint       = ["sh", "-c"]
    command          = ["echo sonde prete"]
    logConfiguration = local.journaux_ecs
  }
}

resource "aws_cloudwatch_log_group" "ecs" {
  count = local.ecs ? 1 : 0

  name              = "/collector/xp01/ecs"
  retention_in_days = var.retention_journaux_jours
}

resource "aws_ecs_cluster" "principal" {
  count = local.ecs ? 1 : 0

  name = local.nom

  setting {
    name  = "containerInsights"
    value = "disabled"
  }
}

resource "aws_ecs_cluster_capacity_providers" "principal" {
  count = local.ecs ? 1 : 0

  cluster_name       = aws_ecs_cluster.principal[0].name
  capacity_providers = ["FARGATE"]

  default_capacity_provider_strategy {
    capacity_provider = "FARGATE"
    weight            = 1
  }
}

resource "aws_security_group" "ecs" {
  for_each = local.roles_ecs

  name        = "${local.nom}-${each.key}"
  description = "Taches ${each.key} du bac a sable (G7, un groupe par service)"
  vpc_id      = aws_vpc.principal.id

  tags = { Name = "${local.nom}-${each.key}" }
}

resource "aws_vpc_security_group_egress_rule" "ecs_sortie" {
  for_each = local.roles_ecs

  security_group_id = aws_security_group.ecs[each.key].id
  description       = "Sortie : images par la NAT, S3 par le point de terminaison, RDS"
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
}

resource "aws_vpc_security_group_ingress_rule" "pdm_depuis" {
  for_each = local.ecs ? {
    alb          = aws_security_group.alb.id
    transactions = aws_security_group.ecs["transactions-test"].id
  } : {}

  security_group_id            = aws_security_group.ecs["pdm-test"].id
  description                  = "Port de test depuis ${each.key}"
  referenced_security_group_id = each.value
  ip_protocol                  = "tcp"
  from_port                    = var.port_test
  to_port                      = var.port_test
}

resource "aws_ecs_task_definition" "test" {
  count = local.ecs ? 1 : 0

  family                   = "${local.nom}-test"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = local.lab_role_arn
  task_role_arn            = local.lab_role_arn

  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }

  container_definitions = jsonencode([{
    name             = "test"
    image            = var.image_test
    essential        = true
    portMappings     = [{ containerPort = var.port_test, protocol = "tcp" }]
    logConfiguration = local.journaux_ecs
  }])

  lifecycle {
    precondition {
      condition     = var.image_test != ""
      error_message = "image_test est obligatoire pour la variante ecs."
    }
  }
}

resource "aws_ecs_task_definition" "sonde" {
  for_each = local.ecs ? { sans_role = null, labrole = local.lab_role_arn } : {}

  family                   = each.key == "labrole" ? "${local.nom}-sonde-labrole" : "${local.nom}-sonde"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = local.lab_role_arn
  task_role_arn            = each.value

  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }

  container_definitions = jsonencode([local.sonde])

  lifecycle {
    precondition {
      condition     = var.image_sonde != ""
      error_message = "image_sonde est obligatoire pour la variante ecs."
    }
  }
}

resource "aws_ecs_service" "pdm_test" {
  count = local.ecs ? 1 : 0

  name                              = "pdm-test"
  cluster                           = aws_ecs_cluster.principal[0].id
  task_definition                   = aws_ecs_task_definition.test[0].arn
  desired_count                     = var.taches_min
  launch_type                       = "FARGATE"
  platform_version                  = "LATEST"
  health_check_grace_period_seconds = 30
  wait_for_steady_state             = false

  network_configuration {
    subnets          = aws_subnet.prive[*].id
    security_groups  = [aws_security_group.ecs["pdm-test"].id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.test.arn
    container_name   = "test"
    container_port   = var.port_test
  }

  lifecycle {
    ignore_changes = [desired_count]
  }

  depends_on = [aws_lb_listener_rule.test, aws_ecs_cluster_capacity_providers.principal]
}

resource "aws_appautoscaling_target" "pdm_test" {
  count = local.ecs ? 1 : 0

  service_namespace  = "ecs"
  resource_id        = "service/${aws_ecs_cluster.principal[0].name}/${aws_ecs_service.pdm_test[0].name}"
  scalable_dimension = "ecs:service:DesiredCount"
  min_capacity       = var.taches_min
  max_capacity       = var.taches_max
}

resource "aws_appautoscaling_policy" "pdm_test_cpu" {
  count = local.ecs ? 1 : 0

  name               = "${local.nom}-cpu"
  policy_type        = "TargetTrackingScaling"
  service_namespace  = aws_appautoscaling_target.pdm_test[0].service_namespace
  resource_id        = aws_appautoscaling_target.pdm_test[0].resource_id
  scalable_dimension = aws_appautoscaling_target.pdm_test[0].scalable_dimension

  target_tracking_scaling_policy_configuration {
    target_value       = var.cible_cpu
    scale_out_cooldown = 30
    scale_in_cooldown  = 120

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}
