# CERTIFICATE_ROTATION.md

## 1. Objective

Rotate TLS certificates for helix_terminator services, load balancers, and internal mTLS with zero downtime.

## 2. Scope

| Certificate | Issuer | Rotation Method |
|-------------|--------|-----------------|
| API Gateway (public) | Let's Encrypt via cert-manager | Automated |
| Internal service mTLS | SPIRE / SPIFFE | Automated (SVID TTL) |
| PostgreSQL TLS | Internal CA (Vault PKI) | Manual |
| Kafka broker TLS | Internal CA (Vault PKI) | Manual |
| Vault itself | Internal CA (Vault PKI) | Manual |

## 3. Automated Rotation: Public API Gateway (cert-manager)

```bash
# Step 1: Verify the Certificate resource
kubectl get certificate api-gateway-tls -n production -o yaml

# Step 2: If forcing early rotation
kubectl annotate certificate api-gateway-tls -n production \
  cert-manager.io/trigger-cert-renewal=true --overwrite

# Step 3: Watch the CertificateRequest
kubectl get certificaterequest -n production -w

# Step 4: Verify the new secret
kubectl get secret api-gateway-tls -n production -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -noout -text | grep -E "Subject:|Issuer:|Not Before|Not After"

# Step 5: Confirm Gateway / Ingress has reloaded
kubectl rollout restart deployment/gateway-service -n production
kubectl rollout status deployment/gateway-service -n production

# Step 6: External verification
echo | openssl s_client -connect api.helix.internal:443 -servername api.helix.internal 2>/dev/null | \
  openssl x509 -noout -dates
```

## 4. Automated Rotation: SPIFFE mTLS (SPIRE)

SPIRE automatically rotates SVIDs before expiry. No manual action is required under normal conditions.

```bash
# Verify SVID TTL and rotation
kubectl exec -it spire-agent-xxxxx -n spire -- \
  /opt/spire/bin/spire-agent api fetch -socketPath /run/spire/sockets/agent.sock

# Check SVID expiry
kubectl exec -it spire-agent-xxxxx -n spire -- \
  openssl x509 -in /run/spire/sockets/svid.pem -noout -dates
```

If manual rotation is needed (e.g., CA compromise), see `docs/guides/runbooks/SSH_CA_INCIDENT.md`.

## 5. Manual Rotation: PostgreSQL TLS (Vault PKI)

```bash
# Step 1: Generate a new certificate from Vault PKI
vault write pki/issue/helix-postgres \
  common_name=postgres.helix.internal \
  ttl=8760h \
  format=pem

# Save the returned certificate and CA chain to files:
#   postgres-new.crt, postgres-new.key, ca-chain.pem

# Step 2: Create a Kubernetes secret with the new cert
kubectl create secret tls postgres-tls-new \
  --cert=postgres-new.crt --key=postgres-new.key \
  -n data --dry-run=client -o yaml | kubectl apply -f -

# Step 3: Rolling restart of PostgreSQL pods (or apply via Cloud SQL / RDS custom CA)
# For self-managed Postgres in Kubernetes:
kubectl rollout restart statefulset/postgres -n data
kubectl rollout status statefulset/postgres -n data

# Step 4: Update client CA bundles
kubectl create configmap postgres-ca-bundle \
  --from-file=ca.crt=ca-chain.pem \
  -n production --dry-run=client -o yaml | kubectl apply -f -
kubectl rollout restart deployment/auth-service -n production

# Step 5: Verify TLS handshake
psql "sslmode=verify-ca host=postgres.helix.internal dbname=helix" -c "SELECT 1;"
```

## 6. Manual Rotation: Kafka Broker TLS (Vault PKI)

```bash
# Step 1: Issue new broker certificates (one per broker)
for i in 0 1 2; do
  vault write pki/issue/helix-kafka \
    common_name="kafka-broker-$i.kafka.helix.internal" \
    alt_names="kafka-broker-$i.kafka.helix.internal,kafka.helix.internal" \
    ttl=8760h format=pem > "kafka-broker-$i.json"
done

# Step 2: Update the Kafka CR (Strimzi example)
# Edit the Kafka CR to reference the new secret names, or patch:
kubectl patch kafka helix-kafka -n data --type=merge -p '
{"spec":{"kafka":{"tls":{"certificate":{"secretName":"kafka-broker-tls-new"}}}}}'

# Step 3: Wait for rolling restart of brokers
kubectl rollout status statefulset/helix-kafka-kafka -n data

# Step 4: Verify broker TLS
kubectl exec -it helix-kafka-kafka-0 -n data -- \
  openssl s_client -connect localhost:9093 -CAfile /var/run/secrets/ca.crt </dev/null | \
  openssl x509 -noout -subject -dates
```

## 7. Verification Checklist

- [ ] New certificate is valid and not expired.
- [ ] Services have reloaded the certificate without restart errors.
- [ ] External TLS scan passes (e.g., SSL Labs, `openssl s_client`).
- [ ] Internal mTLS connections succeed (`spire-agent api fetch` returns valid SVIDs).
- [ ] No certificate expiry alerts in the next 30 days.

## 8. References
- `docs/guides/runbooks/KEY_ROTATION.md`
- `docs/guides/runbooks/SSH_CA_INCIDENT.md`
- `infrastructure/helm/cert-manager/` — cert-manager configuration
