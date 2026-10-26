resource "aws_db_subnet_group" "prive" {
  name       = local.nom
  subnet_ids = aws_subnet.prive[*].id
}

resource "aws_db_parameter_group" "postgres" {
  name   = local.nom
  family = "postgres18"

  parameter {
    name  = "rds.force_ssl"
    value = "1"
  }
}

resource "aws_db_instance" "test" {
  identifier     = local.nom
  engine         = "postgres"
  engine_version = var.version_postgres
  instance_class = "db.t3.micro"

  allocated_storage = 20
  storage_type      = "gp3"
  storage_encrypted = true

  db_name                     = "xp01"
  username                    = "xp01_admin"
  manage_master_user_password = true

  db_subnet_group_name   = aws_db_subnet_group.prive.name
  parameter_group_name   = aws_db_parameter_group.postgres.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false
  multi_az               = false

  backup_retention_period      = 0
  skip_final_snapshot          = true
  delete_automated_backups     = true
  deletion_protection          = false
  auto_minor_version_upgrade   = false
  performance_insights_enabled = false
  monitoring_interval          = 0
  apply_immediately            = true
}
