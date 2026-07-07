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

variable "kubernetes_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.31"
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for EKS"
  type        = list(string)
}

variable "eks_node_security_group_id" {
  description = "Security group ID for EKS nodes"
  type        = string
}

variable "allowed_cidr_blocks" {
  description = "Allowed CIDR blocks for public access"
  type        = list(string)
}

variable "node_desired_size" {
  description = "Desired number of worker nodes"
  type        = number
  default     = 3
}

variable "node_min_size" {
  description = "Minimum number of worker nodes"
  type        = number
  default     = 2
}

variable "node_max_size" {
  description = "Maximum number of worker nodes"
  type        = number
  default     = 10
}

variable "node_instance_types" {
  description = "EC2 instance types for worker nodes"
  type        = list(string)
  default     = ["m6i.large"]
}

variable "node_disk_size" {
  description = "Disk size for worker nodes in GB"
  type        = number
  default     = 50
}

variable "ssh_key_name" {
  description = "SSH key name for node access"
  type        = string
  default     = ""
}
