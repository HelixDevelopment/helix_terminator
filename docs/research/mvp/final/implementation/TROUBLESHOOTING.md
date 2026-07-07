# HelixTerminator — Troubleshooting Guide

**Version:** 1.0.0
**Status:** Draft
**Date:** 2026-07-05
**Authority:** `CANONICAL_FACTS.md` + `SERVICE_REGISTRY.md`

---

## Quick Reference

| Symptom | Likely Cause | Quick Fix |
|---------|-------------|-----------|
| Service won't start | Missing env var, DB connection | Check logs, verify DATABASE_URL |
| 500 errors | DB connection pool exhausted | Scale DB connections, check query performance |
| High latency | Redis cache miss, DB slow query | Check cache hit rate, analyze slow queries |
| Auth failures | JWT expired, clock skew | Check token TTL, verify NTP sync |
| WebSocket disconnect | Connection timeout, proxy issue | Check nginx/ingress timeout, verify keepalive |
| Kafka lag | Consumer slow, partition imbalance | Scale consumer replicas, rebalance partitions |
| ImagePullBackOff | Registry auth, wrong tag | Verify image pull secrets, check tag exists |
| CrashLoopBackOff | App error, missing config | Check logs, fix config, verify env vars |
| Pods stuck Pending | Resource limits, node affinity | Check node capacity, scale cluster |
| Certificate errors | Expired cert, wrong CA | Check cert expiry, verify CA chain |

---

## Service Issues

### Service Won't Start

**Symptoms:**
- Container exits immediately
- `kubectl logs` shows error
- Health check fails

**Diagnosis:**
```bash
# Check logs
kubectl logs -n helixterminator deployment/<service> --previous

# Check events
kubectl get events -n helixterminator --field-selector reason=Failed

# Check config
kubectl get configmap <service>-config -n helixterminator -o yaml

# Check secrets
kubectl get secret <service>-secrets -n helixterminator -o yaml
```

**Common Fixes:**
1. Missing `DATABASE_URL` — set in environment or secret
2. Missing `JWT_SECRET` — generate and set
3. Wrong `KAFKA_BROKERS` — verify broker addresses
4. Port conflict — change `PORT` env var

### Database Connection Issues

**Symptoms:**
- 500 errors with DB timeout
- Connection pool exhausted
- pgx errors in logs

**Diagnosis:**
```bash
# Check DB connections
psql $DATABASE_URL -c "SELECT count(*) FROM pg_stat_activity;"

# Check connection pool
kubectl logs -n helixterminator deployment/<service> | grep "connection"

# Check DB health
psql $DATABASE_URL -c "SELECT pg_is_in_recovery();"
```

**Common Fixes:**
1. Increase `DB_MAX_CONNECTIONS` (default: 25)
2. Check for connection leaks (missing `defer rows.Close()`)
3. Scale DB instance if CPU/memory high
4. Check for long-running queries

### High Latency

**Symptoms:**
- p99 latency > 500ms
- Slow API responses
- Timeout errors

**Diagnosis:**
```bash
# Check Prometheus metrics
curl http://prometheus:9090/api/v1/query?query=histogram_quantile(0.99,rate(http_request_duration_seconds_bucket[5m]))

# Check Jaeger traces
# Open Jaeger UI, filter by service and operation

# Check DB slow queries
psql $DATABASE_URL -c "SELECT query, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
```

**Common Fixes:**
1. Add Redis cache for hot paths
2. Optimize slow queries (add indexes)
3. Scale service replicas
4. Check for N+1 query patterns

---

## Authentication Issues

### JWT Validation Failures

**Symptoms:**
- 401 Unauthorized
- "invalid token" errors
- Clock skew warnings

**Diagnosis:**
```bash
# Decode JWT header
jwt decode <token>

# Check JWKS endpoint
curl https://auth.helixterminator.io/.well-known/jwks.json

# Check token expiry
jwt decode <token> | grep exp

# Check server time
date -u
```

**Common Fixes:**
1. Token expired — refresh or re-login
2. Clock skew — sync NTP on all nodes
3. Wrong signing key — verify JWKS matches
4. Algorithm mismatch — ensure EdDSA (Ed25519)

### MFA Issues

**Symptoms:**
- TOTP codes rejected
- FIDO2/WebAuthn not working
- "MFA required" loop

**Diagnosis:**
```bash
# Check TOTP clock skew
date -u
# TOTP window is ±30 seconds

# Check FIDO2 origin
# Verify origin matches registered origin

# Check MFA enrollment status
curl https://auth.helixterminator.io/api/v1/auth/mfa/status \
  -H "Authorization: Bearer <token>"
```

**Common Fixes:**
1. TOTP: Sync device clock, re-enroll if needed
2. FIDO2: Verify origin, re-register if needed
3. Backup codes: Use single-use backup code

---

## Infrastructure Issues

### Kubernetes

#### Pods Stuck Pending

**Symptoms:**
- `kubectl get pods` shows Pending
- No containers created

**Diagnosis:**
```bash
# Check pod description
kubectl describe pod <pod-name> -n helixterminator

# Check node resources
kubectl top nodes

# Check events
kubectl get events -n helixterminator
```

**Common Fixes:**
1. Insufficient CPU/memory — scale node group
2. Node affinity conflict — check node labels
3. PVC not bound — check storage class
4. Image pull error — check registry access

#### CrashLoopBackOff

**Symptoms:**
- Pod restarts repeatedly
- `kubectl logs` shows error then exits

**Diagnosis:**
```bash
# Check logs
kubectl logs -n helixterminator <pod-name> --previous

# Check resource limits
kubectl describe pod <pod-name> -n helixterminator | grep -A 5 Limits

# Check liveness probe
kubectl describe pod <pod-name> -n helixterminator | grep -A 5 Liveness
```

**Common Fixes:**
1. Application error — fix code, redeploy
2. OOMKilled — increase memory limit
3. Liveness probe too aggressive — adjust timing
4. Missing dependency — verify all services ready

### Docker

#### Image Build Failures

**Symptoms:**
- `docker build` fails
- Multi-stage build error

**Diagnosis:**
```bash
# Build with verbose output
docker build --progress=plain -t <image> .

# Check base image availability
docker pull <base-image>

# Check Dockerfile syntax
docker build --no-cache -t <image> .
```

**Common Fixes:**
1. Network issue — retry, check proxy
2. Missing file — verify COPY paths
3. Base image not found — check tag, registry auth
4. Go module error — run `go mod tidy`

---

## Network Issues

### WebSocket Disconnects

**Symptoms:**
- Terminal connection drops
- Real-time features stop working
- "Connection closed" errors

**Diagnosis:**
```bash
# Check nginx/ingress timeout
kubectl get ingress -n helixterminator -o yaml | grep timeout

# Check WebSocket headers
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  -H "Sec-WebSocket-Version: 13" \
  https://api.helixterminator.io/api/v1/terminal/stream

# Check proxy logs
kubectl logs -n helixterminator deployment/gateway
```

**Common Fixes:**
1. Increase proxy timeout (nginx: `proxy_read_timeout 3600s`)
2. Enable keepalive
3. Check for network policies blocking traffic
4. Verify WebSocket upgrade headers

### mTLS Issues

**Symptoms:**
- Service-to-service calls fail
- "certificate verify failed" errors
- SPIFFE ID mismatch

**Diagnosis:**
```bash
# Check Istio mTLS status
istioctl authn tls-check <pod-name>.<namespace>

# Check SPIFFE ID
kubectl exec -n helixterminator <pod-name> -- cat /var/run/secrets/spire/svid.pem | openssl x509 -text | grep URI

# Check certificate expiry
kubectl exec -n helixterminator <pod-name> -- cat /var/run/secrets/spire/svid.pem | openssl x509 -noout -dates
```

**Common Fixes:**
1. SPIRE agent not running — check DaemonSet
2. Certificate expired — trigger rotation
3. Wrong SPIFFE ID — check registration entries
4. Istio policy mismatch — verify PeerAuthentication

---

## Performance Issues

### Redis Cache

**Symptoms:**
- High cache miss rate
- Slow response times
- Redis memory high

**Diagnosis:**
```bash
# Check cache hit rate
redis-cli info stats | grep keyspace

# Check memory usage
redis-cli info memory | grep used_memory

# Check eviction policy
redis-cli info memory | grep maxmemory_policy
```

**Common Fixes:**
1. Increase cache size
2. Optimize cache keys (shorter, consistent)
3. Add TTL to prevent stale data
4. Use Redis Cluster for horizontal scaling

### Kafka

**Symptoms:**
- High consumer lag
- Message backlog
- Slow event processing

**Diagnosis:**
```bash
# Check consumer lag
kafka-consumer-groups.sh --bootstrap-server $KAFKA_BROKERS --describe --group <group>

# Check partition distribution
kafka-topics.sh --bootstrap-server $KAFKA_BROKERS --describe --topic <topic>

# Check broker health
kafka-broker-api-versions.sh --bootstrap-server $KAFKA_BROKERS
```

**Common Fixes:**
1. Scale consumer replicas
2. Rebalance partitions
3. Increase batch size
4. Check for poison messages

---

## Flutter Client Issues

### Build Failures

**Symptoms:**
- `flutter build` fails
- Compilation errors

**Diagnosis:**
```bash
# Clean build
flutter clean
flutter pub get

# Check Flutter version
flutter doctor

# Verbose build
flutter build apk --verbose
```

**Common Fixes:**
1. Dependency conflict — update `pubspec.yaml`
2. Dart SDK mismatch — use correct Flutter version
3. Missing platform tools — run `flutter doctor` and fix
4. Code generation — run `build_runner`

### Connection Issues

**Symptoms:**
- "Network error" in app
- API calls fail
- WebSocket not connecting

**Diagnosis:**
```bash
# Check API health
curl https://api.helixterminator.io/healthz

# Check certificate
openssl s_client -connect api.helixterminator.io:443 -servername api.helixterminator.io

# Check DNS
nslookup api.helixterminator.io
```

**Common Fixes:**
1. Wrong API URL — check `baseUrl` in config
2. Certificate issue — verify cert chain
3. CORS blocked — check gateway CORS config
4. Network policy — verify allow rules

---

## Getting Help

| Issue Type | Contact | Response Time |
|-----------|---------|--------------|
| Development questions | #dev-helixterminator Slack | 4 hours |
| Production incidents | #incidents PagerDuty | 15 minutes |
| Security concerns | security@helixdevelopment.io | 1 hour |
| Infrastructure issues | #sre-ops Slack | 2 hours |

---

*HelixTerminator Troubleshooting Guide*
*Consolidated from: DEVELOPMENT_KICKOFF.md, 08-devops-infrastructure/README.md*
