output "oidc_provider_arn" {
  description = "ARN of the EKS OIDC provider"
  value       = aws_iam_openid_connect_provider.eks.arn
}

output "oidc_provider_url" {
  description = "URL of the EKS OIDC provider"
  value       = aws_iam_openid_connect_provider.eks.url
}

output "service_role_arns" {
  description = "Map of service names to IAM role ARNs"
  value       = { for name, role in aws_iam_role.service : name => role.arn }
}

output "service_role_names" {
  description = "Map of service names to IAM role names"
  value       = { for name, role in aws_iam_role.service : name => role.name }
}
