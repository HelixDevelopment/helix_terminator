terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# ElastiCache Subnet Group
resource "aws_elasticache_subnet_group" "main" {
  name       = "${var.project_name}-${var.environment}-redis-subnet-group"
  subnet_ids = var.private_subnet_ids

  tags = {
    Name        = "${var.project_name}-redis-subnet-group"
    Environment = var.environment
  }
}

# ElastiCache Parameter Group
resource "aws_elasticache_parameter_group" "main" {
  family = "redis7"
  name   = "${var.project_name}-${var.environment}-redis-params"

  parameter {
    name  = "activedefrag"
    value = "yes"
  }

  parameter {
    name  = "maxmemory-policy"
    value = "allkeys-lru"
  }

  parameter {
    name  = "tcp-keepalive"
    value = "60"
  }

  tags = {
    Name        = "${var.project_name}-redis-params"
    Environment = var.environment
  }
}

# ElastiCache Replication Group (Redis cluster)
resource "aws_elasticache_replication_group" "main" {
  replication_group_id = "${var.project_name}-${var.environment}-redis"
  description        = "Redis cluster for ${var.project_name} ${var.environment}"

  node_type            = var.node_type
  num_cache_clusters   = var.num_cache_clusters
  port                 = 6379

  automatic_failover_enabled = var.automatic_failover_enabled
  multi_az_enabled           = var.multi_az_enabled

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  transit_encryption_mode    = "required"

  auth_token = var.auth_token != "" ? var.auth_token : null

  subnet_group_name  = aws_elasticache_subnet_group.main.name
  security_group_ids = [aws_security_group.redis.id]
  parameter_group_name = aws_elasticache_parameter_group.main.name

  snapshot_retention_limit = 7
  snapshot_window          = "05:00-06:00"
  maintenance_window     = "sun:06:00-sun:07:00"

  apply_immediately = false

  tags = {
    Name        = "${var.project_name}-redis"
    Environment = var.environment
  }
}

# Security Group for Redis
resource "aws_security_group" "redis" {
  name_prefix = "${var.project_name}-redis-"
  vpc_id      = var.vpc_id

  ingress {
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = var.allowed_security_group_ids
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "${var.project_name}-redis-sg"
    Environment = var.environment
  }
}
