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
  description = "Private subnet IDs for MSK broker nodes"
  type        = list(string)
}

variable "vpc_id" {
  description = "VPC ID for security group"
  type        = string
}

variable "allowed_security_group_ids" {
  description = "Security group IDs allowed to access MSK"
  type        = list(string)
  default     = []
}

variable "kafka_version" {
  description = "Apache Kafka version"
  type        = string
  default     = "3.6.0"
}

variable "number_of_broker_nodes" {
  description = "Total number of broker nodes across all AZs"
  type        = number
  default     = 3
}

variable "broker_instance_type" {
  description = "MSK broker instance type"
  type        = string
  default     = "kafka.m5.large"
}

variable "broker_volume_size" {
  description = "EBS volume size per broker in GB"
  type        = number
  default     = 100
}
