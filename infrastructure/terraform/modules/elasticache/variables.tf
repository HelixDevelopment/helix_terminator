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
  description = "Private subnet IDs for ElastiCache"
  type        = list(string)
}

variable "vpc_id" {
  description = "VPC ID for security group"
  type        = string
}

variable "allowed_security_group_ids" {
  description = "Security group IDs allowed to access Redis"
  type        = list(string)
  default     = []
}

variable "node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.r6g.large"
}

variable "num_cache_clusters" {
  description = "Number of cache clusters (primary + replicas)"
  type        = number
  default     = 2
}

variable "automatic_failover_enabled" {
  description = "Enable automatic failover"
  type        = bool
  default     = true
}

variable "multi_az_enabled" {
  description = "Enable Multi-AZ deployment"
  type        = bool
  default     = true
}

variable "auth_token" {
  description = "Redis AUTH token (password). If empty, no auth token is set."
  type        = string
  sensitive   = true
  default     = ""
}
