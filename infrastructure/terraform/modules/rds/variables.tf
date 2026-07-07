variable "project_name" {
  description = "Project name"
  type        = string
  default     = "helixterminator"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for RDS"
  type        = list(string)
}

variable "rds_security_group_id" {
  description = "Security group ID for RDS"
  type        = string
}

variable "instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.r6g.large"
}

variable "allocated_storage" {
  description = "Allocated storage in GB"
  type        = number
  default     = 100
}

variable "max_allocated_storage" {
  description = "Maximum allocated storage in GB"
  type        = number
  default     = 500
}

variable "database_name" {
  description = "Database name"
  type        = string
  default     = "helixterminator"
}

variable "master_username" {
  description = "Master username"
  type        = string
  default     = "helixadmin"
}

variable "master_password" {
  description = "Master password (deprecated: use aws_secretsmanager_secret_version instead)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "master_password_secret_arn" {
  description = "ARN of AWS Secrets Manager secret containing the master password"
  type        = string
  default     = ""
}

variable "multi_az" {
  description = "Enable Multi-AZ deployment"
  type        = bool
  default     = true
}
