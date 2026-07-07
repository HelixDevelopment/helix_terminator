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

variable "eks_cluster_name" {
  description = "EKS cluster name for OIDC provider"
  type        = string
}

variable "aws_region" {
  description = "AWS region for KMS condition keys"
  type        = string
  default     = "us-east-1"
}
