terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# OIDC Provider for EKS IRSA
data "aws_eks_cluster" "main" {
  name = var.eks_cluster_name
}

data "tls_certificate" "eks" {
  url = data.aws_eks_cluster.main.identity[0].oidc[0].issuer
}

resource "aws_iam_openid_connect_provider" "eks" {
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.eks.certificates[0].sha1_fingerprint]
  url             = data.aws_eks_cluster.main.identity[0].oidc[0].issuer

  tags = {
    Name        = "${var.project_name}-eks-oidc"
    Environment = var.environment
  }
}

# Service account roles for each microservice
locals {
  services = [
    "ai-service",
    "analytics-service",
    "audit-service",
    "auth-service",
    "billing-service",
    "collaboration-service",
    "config-service",
    "container-bridge-service",
    "gateway-service",
    "health-service",
    "helixtrack-bridge-service",
    "host-service",
    "keychain-service",
    "notification-service",
    "org-service",
    "pki-service",
    "port-forward-service",
    "recording-service",
    "sftp-service",
    "snippet-service",
    "ssh-proxy-service",
    "terminal-service",
    "user-service",
    "vault-service",
    "workspace-service",
  ]

  service_policies = {
    "ai-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "analytics-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "audit-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "auth-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "billing-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "collaboration-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "config-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "container-bridge-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "gateway-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "health-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "helixtrack-bridge-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "host-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "keychain-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "notification-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "org-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "pki-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "port-forward-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "recording-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "sftp-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "snippet-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "ssh-proxy-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "terminal-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "user-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "vault-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
    "workspace-service" = [
      "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
    ]
  }
}

# IRSA roles for each service
resource "aws_iam_role" "service" {
  for_each = toset(local.services)

  name = "${var.project_name}-${var.environment}-${each.value}-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = aws_iam_openid_connect_provider.eks.arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${replace(data.aws_eks_cluster.main.identity[0].oidc[0].issuer, "https://", "")}:sub" = "system:serviceaccount:${var.environment}:${each.value}"
          "${replace(data.aws_eks_cluster.main.identity[0].oidc[0].issuer, "https://", "")}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = {
    Name        = "${var.project_name}-${each.value}-role"
    Environment = var.environment
    Service     = each.value
  }
}

# Attach AWS managed policies to each service role
resource "aws_iam_role_policy_attachment" "service_managed" {
  for_each = { for pair in setproduct(local.services, ["arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"]) : "${pair[0]}-s3" => {
    role       = pair[0]
    policy_arn = pair[1]
  } }

  role       = aws_iam_role.service[each.value.role].name
  policy_arn = each.value.policy_arn
}

# Custom inline policy for additional AWS service access (RDS, ElastiCache, MSK, Secrets Manager)
resource "aws_iam_role_policy" "service_custom" {
  for_each = aws_iam_role.service

  name = "${var.project_name}-${var.environment}-${each.key}-custom-policy"
  role = each.value.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "RDSAccess"
        Effect = "Allow"
        Action = [
          "rds:DescribeDBInstances",
          "rds:DescribeDBClusters",
        ]
        Resource = "*"
      },
      {
        Sid    = "ElastiCacheAccess"
        Effect = "Allow"
        Action = [
          "elasticache:DescribeCacheClusters",
          "elasticache:DescribeCacheSubnetGroups",
        ]
        Resource = "*"
      },
      {
        Sid    = "MSKAccess"
        Effect = "Allow"
        Action = [
          "kafka:DescribeCluster",
          "kafka:DescribeClusterV2",
          "kafka:GetBootstrapBrokers",
        ]
        Resource = "*"
      },
      {
        Sid    = "SecretsManagerAccess"
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret",
        ]
        Resource = "arn:aws:secretsmanager:*:*:secret:${var.project_name}/${var.environment}/*"
      },
      {
        Sid    = "CloudWatchLogs"
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams",
        ]
        Resource = "arn:aws:logs:*:*:log-group:/aws/eks/${var.project_name}-${var.environment}/*"
      },
      {
        Sid    = "S3Access"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket",
        ]
        Resource = [
          "arn:aws:s3:::${var.project_name}-${var.environment}-*",
          "arn:aws:s3:::${var.project_name}-${var.environment}-*/*",
        ]
      },
      {
        Sid    = "KMSDecrypt"
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey",
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "kms:ViaService" = [
              "secretsmanager.${var.aws_region}.amazonaws.com",
              "s3.${var.aws_region}.amazonaws.com",
            ]
          }
        }
      },
    ]
  })
}
