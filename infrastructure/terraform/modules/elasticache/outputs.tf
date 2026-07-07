output "replication_group_id" {
  description = "ElastiCache replication group ID"
  value       = aws_elasticache_replication_group.main.id
}

output "primary_endpoint" {
  description = "Primary endpoint address"
  value       = aws_elasticache_replication_group.main.primary_endpoint_address
}

output "cluster_endpoint" {
  description = "Alias for primary_endpoint"
  value       = aws_elasticache_replication_group.main.primary_endpoint_address
}

output "reader_endpoint" {
  description = "Reader endpoint address"
  value       = aws_elasticache_replication_group.main.reader_endpoint_address
}

output "security_group_id" {
  description = "Security group ID for Redis"
  value       = aws_security_group.redis.id
}
