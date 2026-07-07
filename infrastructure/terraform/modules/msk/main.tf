terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# MSK Cluster
resource "aws_msk_cluster" "main" {
  cluster_name           = "${var.project_name}-${var.environment}-kafka"
  kafka_version          = var.kafka_version
  number_of_broker_nodes = var.number_of_broker_nodes

  broker_node_group_info {
    instance_type = var.broker_instance_type

    client_subnets  = var.private_subnet_ids
    security_groups = [aws_security_group.msk.id]

    storage_info {
      ebs_storage_info {
        volume_size = var.broker_volume_size
      }
    }
  }

  encryption_info {
    encryption_at_rest_kms_key_arn = aws_kms_key.msk.arn
    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
  }

  client_authentication {
    sasl {
      iam = true
    }
  }

  open_monitoring {
    prometheus {
      jmx_exporter {
        enabled_in_broker = true
      }
      node_exporter {
        enabled_in_broker = true
      }
    }
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.msk.name
      }
    }
  }

  tags = {
    Name        = "${var.project_name}-kafka"
    Environment = var.environment
  }
}

# KMS Key for MSK encryption at rest
resource "aws_kms_key" "msk" {
  description             = "KMS key for MSK encryption at rest"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = {
    Name        = "${var.project_name}-msk-key"
    Environment = var.environment
  }
}

resource "aws_kms_alias" "msk" {
  name          = "alias/${var.project_name}-${var.environment}-msk"
  target_key_id = aws_kms_key.msk.key_id
}

# CloudWatch Log Group for MSK broker logs
resource "aws_cloudwatch_log_group" "msk" {
  name              = "/aws/msk/${var.project_name}-${var.environment}"
  retention_in_days = 7

  tags = {
    Name        = "${var.project_name}-msk-logs"
    Environment = var.environment
  }
}

# Security Group for MSK
resource "aws_security_group" "msk" {
  name_prefix = "${var.project_name}-msk-"
  vpc_id      = var.vpc_id

  ingress {
    from_port       = 9098
    to_port         = 9098
    protocol        = "tcp"
    security_groups = var.allowed_security_group_ids
    description     = "MSK SASL/IAM port"
  }

  ingress {
    from_port       = 9094
    to_port         = 9094
    protocol        = "tcp"
    security_groups = var.allowed_security_group_ids
    description     = "MSK TLS port"
  }

  ingress {
    from_port       = 2181
    to_port         = 2181
    protocol        = "tcp"
    security_groups = var.allowed_security_group_ids
    description     = "Zookeeper port"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "${var.project_name}-msk-sg"
    Environment = var.environment
  }
}
