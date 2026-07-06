# Trivy Security Scanning

## Configuration

### Scan All Services
```bash
# Scan filesystem
trivy fs --severity HIGH,CRITICAL .

# Scan Docker images
for service in services/*/; do
  trivy image "helixterminator/$(basename $service):latest"
done

# Scan Kubernetes manifests
trivy config infrastructure/kubernetes/

# Scan Terraform
trivy config infrastructure/terraform/
```

### CI/CD Integration
```yaml
- name: Trivy vulnerability scan
  uses: aquasecurity/trivy-action@master
  with:
    scan-type: 'fs'
    format: 'sarif'
    output: 'trivy-results.sarif'
    severity: 'CRITICAL,HIGH'
    exit-code: '1'
```

### Ignore File
Create `.trivyignore` for known acceptable vulnerabilities:
```
# CVE-2023-XXXX: Accepted risk - only affects debug endpoint
CVE-2023-XXXX
```
