terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# RDS Subnet Group
resource "aws_db_subnet_group" "main" {
  name       = "${var.project_name}-db-subnet-group"
  subnet_ids = var.private_subnet_ids

  tags = {
    Name = "${var.project_name}-db-subnet-group"
  }
}

# RDS Parameter Group
resource "aws_db_parameter_group" "main" {
  name   = "${var.project_name}-postgres-params"
  family = "postgres17"

  parameter {
    name  = "log_connections"
    value = "1"
  }

  parameter {
    name  = "log_disconnections"
    value = "1"
  }

  parameter {
    name  = "log_checkpoints"
    value = "1"
  }

  parameter {
    name  = "ssl"
    value = "1"
  }
}

# RDS Instance
resource "aws_db_instance" "main" {
  identifier = "${var.project_name}-${var.environment}"

  engine         = "postgres"
  engine_version = "17.2"
  instance_class = var.instance_class

  allocated_storage     = var.allocated_storage
  max_allocated_storage   = var.max_allocated_storage
  storage_type           = "gp3"
  storage_encrypted      = true

  db_name  = var.database_name
  username = var.master_username
  password = var.master_password_secret_arn != "" ? null : var.master_password

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [var.rds_security_group_id]
  parameter_group_name   = aws_db_parameter_group.main.name

  backup_retention_period = 30
  backup_window          = "03:00-04:00"
  maintenance_window     = "Mon:04:00-Mon:05:00"

  multi_az               = var.multi_az
  publicly_accessible    = false
  deletion_protection    = true
  skip_final_snapshot    = false
  final_snapshot_identifier = "${var.project_name}-${var.environment}-final-snapshot"

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]

  performance_insights_enabled = true
  performance_insights_retention_period = 7

  tags = {
    Name        = "${var.project_name}-postgres"
    Environment = var.environment
  }
}

# AWS Secrets Manager Secret for RDS password
resource "aws_secretsmanager_secret" "rds_password" {
  count = var.master_password_secret_arn == "" ? 1 : 0

  name        = "${var.project_name}/${var.environment}/rds-master-password"
  description = "Master password for RDS PostgreSQL instance"

  tags = {
    Name        = "${var.project_name}-rds-password"
    Environment = var.environment
  }
}

resource "aws_secretsmanager_secret_version" "rds_password" {
  count = var.master_password_secret_arn == "" ? 1 : 0

  secret_id     = aws_secretsmanager_secret.rds_password[0].id
  secret_string = jsonencode({
    password = var.master_password != "" ? var.master_password : random_password.rds_password[0].result
  })
}

resource "random_password" "rds_password" {
  count = var.master_password_secret_arn == "" && var.master_password == "" ? 1 : 0

  length           = 32
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

