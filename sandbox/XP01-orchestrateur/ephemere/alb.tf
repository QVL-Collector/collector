resource "aws_lb" "public" {
  name                       = local.nom
  load_balancer_type         = "application"
  internal                   = false
  subnets                    = aws_subnet.public[*].id
  security_groups            = [aws_security_group.alb.id]
  drop_invalid_header_fields = true
  xff_header_processing_mode = "append"
  enable_deletion_protection = false

  tags = { Name = local.nom }
}

resource "aws_lb_target_group" "test" {
  name                 = "${local.nom}-test"
  vpc_id               = aws_vpc.principal.id
  protocol             = "HTTP"
  port                 = local.port_cible
  target_type          = local.eks ? "instance" : "ip"
  deregistration_delay = 10

  health_check {
    path                = var.chemin_sante
    matcher             = "200"
    interval            = 10
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.public.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = var.politique_tls
  certificate_arn   = data.aws_acm_certificate.collector.arn

  default_action {
    type = "fixed-response"

    fixed_response {
      content_type = "text/plain"
      message_body = "Not Found"
      status_code  = "404"
    }
  }
}

resource "aws_lb_listener_rule" "test" {
  listener_arn = aws_lb_listener.https.arn
  priority     = 10

  condition {
    host_header {
      values = [var.domaine]
    }
  }

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.test.arn
  }
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.public.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = "redirect"

    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}
