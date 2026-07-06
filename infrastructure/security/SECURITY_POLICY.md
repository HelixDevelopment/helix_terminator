# Security Policy for HelixTerminator

## Pod Security Standards

All pods in the `helixterminator` namespace must comply with the **restricted** Pod Security Standard:

### Requirements
- Pods must run as non-root
- Containers must not allow privilege escalation
- Containers must drop all capabilities
- Read-only root filesystems are required
- Seccomp profiles must be set to RuntimeDefault

## Network Security

### Default Deny
All pods have a default-deny NetworkPolicy applied. Explicit allow policies must be created for:
- Ingress from the ingress-nginx controller
- Egress to DNS (kube-dns)
- Egress to PostgreSQL (port 5432)
- Egress to Redis (port 6379)
- Egress to Kafka (port 9092)
- Inter-service communication on port 8080

### Service Mesh
Consider implementing a service mesh (Istio/Linkerd) for:
- mTLS between services
- Traffic encryption
- Observability
- Traffic management

## Secret Management

### Kubernetes Secrets
- All secrets must be encrypted at rest (etcd encryption)
- Use external secret management (AWS Secrets Manager, HashiCorp Vault) for production
- Rotate secrets every 90 days
- Never commit secrets to Git

### Sealed Secrets
Use Bitnami Sealed Secrets for GitOps workflows:
```bash
kubeseal --controller-namespace=kube-system < secret.yaml > sealed-secret.yaml
```

## Image Security

### Container Images
- All images must be signed with Cosign
- No `:latest` tags allowed
- Use distroless or minimal base images
- Scan all images with Trivy before deployment
- Images must not contain known CVEs (CRITICAL/HIGH)

### Image Signing
```bash
cosign sign --key cosign.key $IMAGE_URI
cosign verify --key cosign.pub $IMAGE_URI
```

## Runtime Security

### Falco
Falco is deployed to detect:
- Terminal shells in containers
- Unauthorized database access
- Sensitive file access
- Privilege escalation
- Crypto mining
- Outbound connections from sensitive services

### Audit Logging
All API requests are audited:
- User ID, action, resource, timestamp
- IP address and user agent
- Success/failure status
- Retention: 1 year

## Compliance

### Data Protection
- All data at rest is encrypted (AES-256)
- All data in transit uses TLS 1.2+
- Personal data is pseudonymized where possible
- GDPR compliance for EU users

### Access Control
- RBAC for Kubernetes resources
- Least privilege principle
- Regular access reviews (quarterly)
- MFA required for all admin access

## Incident Response

### Detection
- Falco alerts → Slack/PagerDuty
- Failed auth attempts → rate limiting + alerting
- Unusual traffic patterns → anomaly detection

### Response
1. Isolate affected pods
2. Collect forensic evidence
3. Notify security team
4. Document and remediate
5. Post-incident review
