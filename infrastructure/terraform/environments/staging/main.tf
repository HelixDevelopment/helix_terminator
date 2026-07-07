terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  backend "s3" {
    bucket         = "helixterminator-terraform-state"
    key            = "environments/staging/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "helixterminator-terraform-locks"
  }
}

provider "aws" {
  region = var.aws_region
  default_tags {
    tags = {
      Project     = "helixterminator"
      Environment = "staging"
      ManagedBy   = "terraform"
    }
  }
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "staging"
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "helixterminator"
}

# VPC Module
module "vpc" {
  source = "../../modules/vpc"

  project_name       = var.project_name
  environment        = var.environment
  vpc_cidr           = "10.1.0.0/16"
  availability_zones = ["us-east-1a", "us-east-1b"]
}

# EKS Module
module "eks" {
  source = "../../modules/eks"

  project_name               = var.project_name
  environment                = var.environment
  kubernetes_version         = "1.31"
  private_subnet_ids         = module.vpc.private_subnet_ids
  eks_node_security_group_id = module.vpc.eks_node_security_group_id
  allowed_cidr_blocks        = var.allowed_cidr_blocks
  node_desired_size          = 2
  node_min_size              = 1
  node_max_size              = 5
  node_instance_types        = ["m6i.large"]
  node_disk_size             = 50
  ssh_key_name               = var.ssh_key_name
}

# RDS Module
module "rds" {
  source = "../../modules/rds"

  project_name        = var.project_name
  environment         = var.environment
  private_subnet_ids  = module.vpc.private_subnet_ids
  rds_security_group_id = module.vpc.rds_security_group_id
  instance_class        = "db.t3.large"
  allocated_storage     = 50
  max_allocated_storage = 200
  database_name         = "helixterminator"
  master_username       = "helixadmin"
  master_password       = var.rds_master_password
  multi_az              = false
}

# ElastiCache Module
module "elasticache" {
  source = "../../modules/elasticache"

  project_name              = var.project_name
  environment               = var.environment
  private_subnet_ids        = module.vpc.private_subnet_ids
  elasticache_security_group_id = module.vpc.eks_node_security_group_id
  node_type                 = "cache.t3.medium"
  num_cache_nodes           = 1
}

# MSK Module
module "msk" {
  source = "../../modules/msk"

  project_name       = var.project_name
  environment        = var.environment
  private_subnet_ids = module.vpc.private_subnet_ids
  msk_security_group_id = module.vpc.eks_node_security_group_id
  kafka_version      = "3.9.0"
  broker_instance_type = "kafka.t3.small"
  number_of_broker_nodes = 2
}

# IAM Module
module "iam" {
  source = "../../modules/iam"

  project_name = var.project_name
  environment  = var.environment
  cluster_name = module.eks.cluster_name
  oidc_provider_arn = module.eks.oidc_provider_arn
}

variable "allowed_cidr_blocks" {
  description = "Allowed CIDR blocks for EKS public access"
  type        = list(string)
  default     = []
}

variable "ssh_key_name" {
  description = "SSH key name for EKS nodes"
  type        = string
  default     = ""
}

variable "rds_master_password" {
  description = "RDS master password"
  type        = string
  sensitive   = true
}

output "vpc_id" {
  value = module.vpc.vpc_id
}

output "eks_cluster_endpoint" {
  value = module.eks.cluster_endpoint
}

output "rds_endpoint" {
  value = module.rds.db_instance_endpoint
}

output "elasticache_endpoint" {
  value = module.elasticache.cluster_endpoint
}

output "msk_brokers" {
  value = module.msk.bootstrap_brokers
}
