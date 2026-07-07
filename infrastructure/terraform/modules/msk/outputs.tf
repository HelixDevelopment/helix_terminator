output "cluster_arn" {
  description = "MSK cluster ARN"
  value       = aws_msk_cluster.main.arn
}

output "cluster_name" {
  description = "MSK cluster name"
  value       = aws_msk_cluster.main.cluster_name
}

output "bootstrap_brokers_sasl_iam" {
  description = "Bootstrap brokers for SASL/IAM authentication"
  value       = aws_msk_cluster.main.bootstrap_brokers_sasl_iam
}

output "bootstrap_brokers" {
  description = "Alias for bootstrap_brokers_sasl_iam"
  value       = aws_msk_cluster.main.bootstrap_brokers_sasl_iam
}

output "zookeeper_connect_string" {
  description = "Zookeeper connection string"
  value       = aws_msk_cluster.main.zookeeper_connect_string
}

output "security_group_id" {
  description = "Security group ID for MSK"
  value       = aws_security_group.msk.id
}

output "kms_key_arn" {
  description = "KMS key ARN used for MSK encryption at rest"
  value       = aws_kms_key.msk.arn
}
