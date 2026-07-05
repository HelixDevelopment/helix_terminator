# HelixTerminator — Master Development Plan
# Version: 1.0.0
# Created: 2026-07-05
# Status: KICKED OFF

## Executive Summary

This plan organizes the full implementation of the HelixTerminator platform into 7 parallel work streams, each with granular tasks, dependencies, acceptance criteria, and physical evidence requirements. Every task must produce real, verifiable results — no bluff, no stubs, no placeholders.

## Parallel Work Streams

### STREAM A: Backend Core Services (7 services first)
Priority: P0 — Blocks all other streams
Services: gateway, auth, user, vault, host, ssh-proxy, terminal

### STREAM B: Backend Supporting Services (18 services)
Priority: P1 — Can proceed in parallel with Stream A after gateway/auth are stable
Services: sftp, port-forward, snippet, keychain, workspace, collaboration, notification, audit, analytics, ai, recording, pki, org, billing, config, health, container-bridge, helixtrack-bridge

### STREAM C: Flutter Client
Priority: P0 — Parallel with backend, needs auth/gateway API first
Screens: 32 screens, 278 widgets, 8 BLoCs, 6 service clients

### STREAM D: Infrastructure & DevOps
Priority: P1 — Parallel, needs services for K8s manifests
Terraform, K8s, Helm, CI/CD, observability, security tooling

### STREAM E: Testing & QA
Priority: P1 — Parallel, tests services as they're built
17 test types, HelixQA/Challenges integration, CI pipelines

### STREAM F: Security Hardening
Priority: P0 — Continuous, applies to all streams
Zero-trust, crypto, auth, RBAC, compliance, runbooks

### STREAM G: Documentation & Design Polish
Priority: P2 — Continuous, updates as implementation progresses
OpenDesign token refinement, API docs, user guides, ADR updates

---

## STREAM A: Backend Core Services (P0)

### A.1 Gateway Service
- [ ] A.1.1 Implement HTTP router with middleware chain (logging, recovery, CORS, request ID)
- [ ] A.1.2 Implement JWT validation middleware (Ed25519)
- [ ] A.1.3 Implement rate limiting (per-user, per-IP, per-endpoint)
- [ ] A.1.4 Implement upstream routing with health checks
- [ ] A.1.5 Implement circuit breaker pattern
- [ ] A.1.6 Implement WebSocket upgrade proxy for terminal
- [ ] A.1.7 Implement OpenAPI 3.1 spec serving (/api/v1/openapi.json)
- [ ] A.1.8 Implement Swagger/ReDoc UI endpoints
- [ ] A.1.9 Implement metrics exposition (/metrics)
- [ ] A.1.10 Write integration tests (testcontainers, httptest)
- [ ] A.1.11 Write load tests (k6)
- **Acceptance:** `curl http://localhost:8080/healthz/live` returns 200, `curl http://localhost:8080/api/v1/openapi.json` returns valid OpenAPI spec

### A.2 Auth Service
- [ ] A.2.1 Implement user registration (password hashing with Argon2id)
- [ ] A.2.2 Implement login (JWT access + refresh tokens, Ed25519 signed)
- [ ] A.2.3 Implement MFA setup (TOTP with base32 secret, QR code generation)
- [ ] A.2.4 Implement MFA verification
- [ ] A.2.5 Implement FIDO2/WebAuthn registration and authentication
- [ ] A.2.6 Implement SSO/OIDC flows (Google, GitHub, Microsoft, Okta)
- [ ] A.2.7 Implement SAML 2.0 SP
- [ ] A.2.8 Implement session management (create, validate, revoke, list)
- [ ] A.2.9 Implement refresh token rotation
- [ ] A.2.10 Implement SCIM inbound sync
- [ ] A.2.11 Implement password reset flow
- [ ] A.2.12 Implement account lockout (failed attempts tracking)
- [ ] A.2.13 Write unit tests (≥90% coverage)
- [ ] A.2.14 Write integration tests (testcontainers PostgreSQL)
- [ ] A.2.15 Write security tests (brute force, JWT algorithm confusion)
- **Acceptance:** Full auth flow works: register → login → MFA → access token → refresh → logout

### A.3 User Service
- [ ] A.3.1 Implement user CRUD (profile, preferences, avatar)
- [ ] A.3.2 Implement session management (list, revoke)
- [ ] A.3.3 Implement user preferences storage
- [ ] A.3.4 Implement activity tracking
- [ ] A.3.5 Write tests (≥80% coverage)
- **Acceptance:** User profile can be created, read, updated, deleted

### A.4 Vault Service
- [ ] A.4.1 Implement vault CRUD
- [ ] A.4.2 Implement vault item CRUD (password, SSH key, API key, note, environment)
- [ ] A.4.3 Implement client-side encryption helpers (AES-256-GCM key derivation)
- [ ] A.4.4 Implement vault sharing (read/write/admin access levels)
- [ ] A.4.5 Implement vault key rotation
- [ ] A.4.6 Implement Shamir's secret sharing for recovery
- [ ] A.4.7 Write tests (≥90% coverage)
- [ ] A.4.8 Write crypto fuzz tests (FuzzVaultEncryptDecrypt)
- **Acceptance:** Vault can be created, items encrypted/decrypted, shared with other users

### A.5 Host Service
- [ ] A.5.1 Implement host CRUD (name, address, port, username, auth method)
- [ ] A.5.2 Implement host groups and tags
- [ ] A.5.3 Implement host connectivity testing (TCP dial, SSH handshake)
- [ ] A.5.4 Implement host import/export (CSV, JSON, SSH config)
- [ ] A.5.5 Implement jump host chaining
- [ ] A.5.6 Write tests (≥80% coverage)
- **Acceptance:** Host can be added, tested for connectivity, connected via SSH

### A.6 SSH Proxy Service
- [ ] A.6.1 Implement SSH connection broker (password, pubkey, certificate auth)
- [ ] A.6.2 Implement SSH session management (create, maintain, close)
- [ ] A.6.3 Implement agent forwarding
- [ ] A.6.4 Implement proxy jump chains
- [ ] A.6.5 Implement container-native sessions (exec into containers)
- [ ] A.6.6 Implement session recording (asciinema format)
- [ ] A.6.7 Implement host key verification and caching
- [ ] A.6.8 Write tests (≥80% coverage)
- **Acceptance:** Can establish SSH connection to a host, execute commands, transfer files

### A.7 Terminal Service
- [ ] A.7.1 Implement WebSocket terminal I/O proxy
- [ ] A.7.2 Implement scrollback buffer (Redis-backed)
- [ ] A.7.3 Implement command-boundary detection
- [ ] A.7.4 Implement terminal resize handling
- [ ] A.7.5 Implement multi-session support
- [ ] A.7.6 Implement collaborative fan-out (CRDT-based)
- [ ] A.7.7 Write tests (≥80% coverage)
- **Acceptance:** WebSocket terminal connects, displays output, handles input, resizes

---

## STREAM B: Backend Supporting Services (P1)

### B.1 SFTP Service
- [ ] B.1.1 Implement SFTP session management
- [ ] B.1.2 Implement file operations (list, upload, download, delete, rename)
- [ ] B.1.3 Implement transfer queue with resume
- [ ] B.1.4 Implement checksum verification
- [ ] B.1.5 Implement bidirectional directory sync
- [ ] B.1.6 Write tests (≥80% coverage)

### B.2 Port Forwarding Service
- [ ] B.2.1 Implement tunnel catalog (local, remote, dynamic, reverse)
- [ ] B.2.2 Implement auto-reconnect
- [ ] B.2.3 Implement tunnel metrics
- [ ] B.2.4 Write tests (≥80% coverage)

### B.3 Snippet Service
- [ ] B.3.1 Implement snippet CRUD
- [ ] B.3.2 Implement variable substitution
- [ ] B.3.3 Implement snippet execution on remote host
- [ ] B.3.4 Write tests (≥80% coverage)

### B.4 Keychain Service
- [ ] B.4.1 Implement SSH key generation (RSA, Ed25519, ECDSA)
- [ ] B.4.2 Implement SSH key import/export
- [ ] B.4.3 Implement key fingerprinting
- [ ] B.4.4 Write tests (≥80% coverage)

### B.5 Workspace Service
- [ ] B.5.1 Implement workspace CRUD
- [ ] B.5.2 Implement session layout management
- [ ] B.5.3 Write tests (≥80% coverage)

### B.6 Collaboration Service
- [ ] B.6.1 Implement real-time session sharing
- [ ] B.6.2 Implement CRDT buffer sync
- [ ] B.6.3 Implement broadcast mode
- [ ] B.6.4 Implement chat sidebar
- [ ] B.6.5 Write tests (≥80% coverage)

### B.7 Notification Service
- [ ] B.7.1 Implement multi-channel delivery (in-app, email, push, Slack, webhook)
- [ ] B.7.2 Implement templates and deduplication
- [ ] B.7.3 Write tests (≥80% coverage)

### B.8 Audit Service
- [ ] B.8.1 Implement append-only audit log (Merkle-chained)
- [ ] B.8.2 Implement compliance query/export
- [ ] B.8.3 Implement retention enforcement
- [ ] B.8.4 Write tests (≥90% coverage)

### B.9 Analytics Service
- [ ] B.9.1 Implement usage metrics collection
- [ ] B.9.2 Implement insights generation
- [ ] B.9.3 Write tests (≥80% coverage)

### B.10 AI Service
- [ ] B.10.1 Implement command autocomplete
- [ ] B.10.2 Implement output explanation
- [ ] B.10.3 Implement anomaly detection
- [ ] B.10.4 Write tests (≥80% coverage)

### B.11 Recording Service
- [ ] B.11.1 Implement asciinema recording assembly
- [ ] B.11.2 Implement Ed25519 signing
- [ ] B.11.3 Implement playback API
- [ ] B.11.4 Implement MP4 export
- [ ] B.11.5 Write tests (≥80% coverage)

### B.12 PKI Service
- [ ] B.12.1 Implement SSH CA (user and host certificates)
- [ ] B.12.2 Implement certificate issuance
- [ ] B.12.3 Implement CA rotation
- [ ] B.12.4 Implement revocation (CRL/OCSP)
- [ ] B.12.5 Write tests (≥90% coverage)

### B.13 Organization Service
- [ ] B.13.1 Implement org CRUD
- [ ] B.13.2 Implement member management
- [ ] B.13.3 Implement RBAC role/permission model
- [ ] B.13.4 Implement SCIM provisioning
- [ ] B.13.5 Write tests (≥80% coverage)

### B.14 Billing Service
- [ ] B.14.1 Implement subscription management
- [ ] B.14.2 Implement Stripe integration
- [ ] B.14.3 Implement usage metering
- [ ] B.14.4 Write tests (≥80% coverage)

### B.15 Config Service
- [ ] B.15.1 Implement feature flags
- [ ] B.15.2 Implement settings management
- [ ] B.15.3 Write tests (≥80% coverage)

### B.16 Health Service
- [ ] B.16.1 Implement health aggregation from all services
- [ ] B.16.2 Implement SLO error-budget calculation
- [ ] B.16.3 Write tests (≥80% coverage)

### B.17 Container Bridge Service
- [ ] B.17.1 Implement container runtime integration
- [ ] B.17.2 Write tests (≥80% coverage)

### B.18 HelixTrack Bridge Service
- [ ] B.18.1 Implement HelixTrack integration
- [ ] B.18.2 Write tests (≥80% coverage)

---

## STREAM C: Flutter Client (P0)

### C.1 Project Setup & Architecture
- [ ] C.1.1 Configure BLoC pattern with flutter_bloc
- [ ] C.1.2 Configure dependency injection (get_it)
- [ ] C.1.3 Configure routing (go_router)
- [ ] C.1.4 Configure HTTP client (Dio with interceptors)
- [ ] C.1.5 Configure local database (drift)
- [ ] C.1.6 Configure state persistence
- [ ] C.1.7 Verify build for all 6 platforms (Web, macOS, Windows, Linux, iOS, Android)
- **Acceptance:** `flutter build` succeeds for all platforms

### C.2 Authentication Flow
- [ ] C.2.1 Implement login screen (email/password)
- [ ] C.2.2 Implement MFA screen (TOTP input)
- [ ] C.2.3 Implement FIDO2/WebAuthn screen
- [ ] C.2.4 Implement SSO login screen
- [ ] C.2.5 Implement registration screen
- [ ] C.2.6 Implement password reset flow
- [ ] C.2.7 Implement biometric authentication (Touch ID/Face ID)
- [ ] C.2.8 Implement session management (token refresh, logout)
- **Acceptance:** Full auth flow works end-to-end with backend

### C.3 Host Management
- [ ] C.3.1 Implement host list screen (grid/list views)
- [ ] C.3.2 Implement host detail/edit screen
- [ ] C.3.3 Implement host creation screen
- [ ] C.3.4 Implement host groups and tags
- [ ] C.3.5 Implement quick connect dialog
- [ ] C.3.6 Implement host import (CSV, SSH config)
- **Acceptance:** Can add, edit, delete, and connect to hosts

### C.4 Terminal
- [ ] C.4.1 Implement terminal screen (single session)
- [ ] C.4.2 Implement split view (2×1, 2×2)
- [ ] C.4.3 Implement terminal tabs
- [ ] C.4.4 Implement terminal toolbar
- [ ] C.4.5 Implement terminal themes (6 built-in + custom)
- [ ] C.4.6 Implement keyboard shortcuts (130+)
- [ ] C.4.7 Implement command palette
- [ ] C.4.8 Implement search/find in terminal
- **Acceptance:** Terminal connects to SSH, displays output, handles input

### C.5 SFTP
- [ ] C.5.1 Implement SFTP browser (desktop + mobile)
- [ ] C.5.2 Implement file upload/download
- [ ] C.5.3 Implement transfer queue
- [ ] C.5.4 Implement drag-and-drop (desktop)
- **Acceptance:** Can browse remote files, upload, download

### C.6 Vault
- [ ] C.6.1 Implement vault list screen
- [ ] C.6.2 Implement vault detail screen
- [ ] C.6.3 Implement item creation (password, SSH key, API key, note)
- [ ] C.6.4 Implement item editing with encryption
- [ ] C.6.5 Implement vault sharing
- **Acceptance:** Can create encrypted vault items, share vaults

### C.7 Collaboration
- [ ] C.7.1 Implement collaboration panel
- [ ] C.7.2 Implement real-time cursor sharing
- [ ] C.7.3 Implement chat sidebar
- [ ] C.7.4 Implement session sharing (observer/co-pilot/owner)
- **Acceptance:** Multiple users can view/interact with same session

### C.8 Settings & Customization
- [ ] C.8.1 Implement settings screen (appearance, security, keyboard)
- [ ] C.8.2 Implement theme switching (dark/light/system)
- [ ] C.8.3 Implement font customization
- [ ] C.8.4 Implement keyboard shortcut customization
- [ ] C.8.5 Implement notification preferences
- **Acceptance:** Settings persist and apply immediately

### C.9 Remaining Screens
- [ ] C.9.1 Implement snippets library
- [ ] C.9.2 Implement keychain manager
- [ ] C.9.3 Implement port forwarding panel
- [ ] C.9.4 Implement session recording viewer
- [ ] C.9.5 Implement notifications panel
- [ ] C.9.6 Implement organization/team management
- [ ] C.9.7 Implement billing screen
- [ ] C.9.8 Implement AI assistant screen
- [ ] C.9.9 Implement audit log screen
- [ ] C.9.10 Implement onboarding flow
- [ ] C.9.11 Implement splash screen
- [ ] C.9.12 Implement dashboard

### C.10 Widgets & Components
- [ ] C.10.1 Implement all 50+ design system components
- [ ] C.10.2 Implement terminal-specific widgets
- [ ] C.10.3 Implement SSH-specific widgets
- [ ] C.10.4 Implement responsive layouts for all breakpoints
- [ ] C.10.5 Implement accessibility features (screen reader, high contrast)
- [ ] C.10.6 Implement golden/snapshot tests for all components
- **Acceptance:** All widgets render correctly across platforms

---

## STREAM D: Infrastructure & DevOps (P1)

### D.1 Terraform
- [ ] D.1.1 Implement EKS module (cluster, node groups, IAM)
- [ ] D.1.2 Implement RDS module (PostgreSQL Multi-AZ)
- [ ] D.1.3 Implement ElastiCache module (Redis Cluster)
- [ ] D.1.4 Implement MSK module (Kafka)
- [ ] D.1.5 Implement VPC module (networking, subnets, NAT)
- [ ] D.1.6 Implement IAM module (roles, policies)
- [ ] D.1.7 Implement S3 module (backups, assets)
- [ ] D.1.8 Implement CloudFront module (CDN)
- [ ] D.1.9 Implement Route53 module (DNS, health checks)
- [ ] D.1.10 Implement KMS module (key management)
- [ ] D.1.11 Create production environment
- [ ] D.1.12 Create staging environment
- [ ] D.1.13 Create DR region (eu-west-1)
- [ ] D.1.14 Validate with `terraform plan` and `terraform validate`
- **Acceptance:** `terraform plan` shows no errors, infrastructure can be provisioned

### D.2 Kubernetes
- [ ] D.2.1 Implement namespace definitions (prod, staging, dev, monitoring, spire)
- [ ] D.2.2 Implement default-deny NetworkPolicy
- [ ] D.2.3 Implement per-service NetworkPolicies
- [ ] D.2.4 Implement Deployment manifests for all 25 services
- [ ] D.2.5 Implement Service manifests
- [ ] D.2.6 Implement HPA manifests
- [ ] D.2.7 Implement PDB manifests
- [ ] D.2.8 Implement ConfigMap/Secret manifests
- [ ] D.2.9 Implement ServiceAccount manifests
- [ ] D.2.10 Implement Ingress manifests
- [ ] D.2.11 Implement cert-manager integration
- [ ] D.2.12 Validate with kube-score
- **Acceptance:** `kubectl apply --dry-run=client` succeeds for all manifests

### D.3 Helm
- [ ] D.3.1 Implement umbrella chart structure
- [ ] D.3.2 Implement subcharts for all 25 services
- [ ] D.3.3 Implement values.yaml (default)
- [ ] D.3.4 Implement values-production.yaml
- [ ] D.3.5 Implement values-staging.yaml
- [ ] D.3.6 Implement dependency charts (Kafka, PostgreSQL, Redis, RabbitMQ)
- [ ] D.3.7 Validate with `helm lint` and `helm template`
- **Acceptance:** `helm template helixterm .` renders valid K8s YAML

### D.4 CI/CD
- [ ] D.4.1 Implement PR pipeline (lint, unit test, integration test, SAST, build)
- [ ] D.4.2 Implement main pipeline (E2E, performance, security scan)
- [ ] D.4.3 Implement nightly pipeline (regression, mutation, chaos, fuzz)
- [ ] D.4.4 Implement release pipeline (golden tests, a11y, pen-test gate, publish)
- [ ] D.4.5 Implement canary deployment strategy
- [ ] D.4.6 Implement automated rollback
- [ ] D.4.7 Implement dependency update automation
- **Acceptance:** All workflows pass on GitHub Actions

### D.5 Observability
- [ ] D.5.1 Implement Prometheus scraping config
- [ ] D.5.2 Implement Grafana dashboards (Platform Overview, SSH Deep-Dive, Vault Latency)
- [ ] D.5.3 Implement OpenTelemetry Collector config
- [ ] D.5.4 Implement Loki log aggregation config
- [ ] D.5.5 Implement Jaeger tracing config
- [ ] D.5.6 Implement Alertmanager rules
- [ ] D.5.7 Implement SLO burn-rate alerts
- **Acceptance:** Metrics are visible in Grafana, alerts fire correctly

### D.6 Security Tooling
- [ ] D.6.1 Implement Falco runtime rules
- [ ] D.6.2 Implement Sealed Secrets workflow
- [ ] D.6.3 Implement Cosign image signing
- [ ] D.6.4 Implement Trivy scanning gate
- [ ] D.6.5 Implement ClusterImagePolicy admission webhook
- [ ] D.6.6 Implement Pod Security Standards (restricted)
- [ ] D.6.7 Implement RBAC manifests
- [ ] D.6.8 Implement EKS audit logging policy
- **Acceptance:** Security gates pass in CI/CD

### D.7 Docker & Local Dev
- [ ] D.7.1 Implement docker-compose.yml (full stack)
- [ ] D.7.2 Implement k3d/kind local cluster setup
- [ ] D.7.3 Implement Coder workspace template
- [ ] D.7.4 Implement Harbor registry config
- **Acceptance:** `docker-compose up` starts full stack locally

---

## STREAM E: Testing & QA (P1)

### E.1 Unit Tests
- [ ] E.1.1 Implement unit tests for all 25 services (≥80% coverage, auth/vault ≥90%)
- [ ] E.1.2 Implement unit tests for Flutter BLoCs and widgets
- [ ] E.1.3 Implement unit tests for crypto functions (100% coverage)
- **Acceptance:** `go test -cover` meets thresholds

### E.2 Integration Tests
- [ ] E.2.1 Implement testcontainers-go setup (PostgreSQL, Redis, Kafka, RabbitMQ)
- [ ] E.2.2 Implement service-to-service integration tests
- [ ] E.2.3 Implement DB migration idempotency tests
- [ ] E.2.4 Implement API contract tests (Pact)
- **Acceptance:** All integration tests pass

### E.3 E2E Tests
- [ ] E.3.1 Implement backend E2E tests (full user journey)
- [ ] E.3.2 Implement Flutter integration tests
- [ ] E.3.3 Implement device matrix testing
- **Acceptance:** Full user journey passes end-to-end

### E.4 Performance Tests
- [ ] E.4.1 Implement k6 load tests (SSH 10k concurrent, vault 100k items, API 10k RPS, WebSocket 100k)
- [ ] E.4.2 Implement Go benchmark tests
- [ ] E.4.3 Implement terminal render performance suite
- **Acceptance:** k6 tests run and report metrics

### E.5 Security Tests
- [ ] E.5.1 Implement auth brute-force tests
- [ ] E.5.2 Implement JWT algorithm confusion tests
- [ ] E.5.3 Implement session fixation tests
- [ ] E.5.4 Implement CSRF tests
- [ ] E.5.5 Implement IDOR tests
- [ ] E.5.6 Implement RBAC tests
- [ ] E.5.7 Implement crypto tests (AES-IV uniqueness, weak key rejection)
- [ ] E.5.8 Implement SSH weak algorithm rejection tests
- [ ] E.5.9 Implement MITM detection tests
- [ ] E.5.10 Implement K8s network policy tests
- [ ] E.5.11 Implement mTLS tests
- **Acceptance:** All security tests pass

### E.6 Chaos Tests
- [ ] E.6.1 Implement LitmusChaos CRDs (pod-kill SSH proxy, Kafka broker kill, Postgres failover, network partition, memory pressure, CPU saturation)
- [ ] E.6.2 Implement recovery verification scripts
- **Acceptance:** Chaos experiments run and systems recover

### E.7 Mutation Tests
- [ ] E.7.1 Implement go-mutesting config
- [ ] E.7.2 Implement mutation score enforcement (≥80% global, ≥90% auth/vault, 100% crypto)
- **Acceptance:** Mutation score meets thresholds

### E.8 Accessibility Tests
- [ ] E.8.1 Implement axe-core WCAG 2.2 AA tests (web)
- [ ] E.8.2 Implement VoiceOver tests (iOS)
- [ ] E.8.3 Implement TalkBack tests (Android)
- [ ] E.8.4 Implement NVDA/JAWS tests (desktop)
- **Acceptance:** Accessibility tests pass

### E.9 External Test Suites
- [ ] E.9.1 Wire HelixQA submodule (git@github.com:HelixDevelopment/helixqa.git)
- [ ] E.9.2 Wire Challenges submodule (git@github.com:vasic-digital/challenges.git)
- [ ] E.9.3 Create .helixqa.yaml config
- [ ] E.9.4 Create test banks for all domains
- **Acceptance:** HelixQA runs autonomous QA sessions

---

## STREAM F: Security Hardening (P0 — Continuous)

### F.1 Zero-Trust Implementation
- [ ] F.1.1 Deploy SPIRE server (3-node HA)
- [ ] F.1.2 Deploy SPIRE agent (DaemonSet)
- [ ] F.1.3 Implement SVID registration entries
- [ ] F.1.4 Implement SVID rotation Go SDK
- [ ] F.1.5 Deploy Istio with STRICT mTLS
- [ ] F.1.6 Implement PeerAuthentication (mesh-wide)
- [ ] F.1.7 Implement AuthorizationPolicy per service
- [ ] F.1.8 Implement continuous verification service
- **Acceptance:** Service-to-service calls require valid SVID

### F.2 Cryptographic Hardening
- [ ] F.2.1 Implement AES-256-GCM with random IVs
- [ ] F.2.2 Implement Argon2id key derivation
- [ ] F.2.3 Implement Ed25519 JWT signing
- [ ] F.2.4 Implement Shamir's secret sharing
- [ ] F.2.5 Implement key rotation automation
- [ ] F.2.6 Implement HSM integration (AWS KMS)
- **Acceptance:** Crypto tests pass, no weak algorithms

### F.3 Auth Hardening
- [ ] F.3.1 Implement MFA enforcement for admin roles
- [ ] F.3.2 Implement session timeout policies
- [ ] F.3.3 Implement concurrent session limits
- [ ] F.3.4 Implement IP-based access controls
- [ ] F.3.5 Implement brute-force protection
- [ ] F.3.6 Implement account lockout policies
- **Acceptance:** Auth penetration tests pass

### F.4 Infrastructure Security
- [ ] F.4.1 Implement container hardening (distroless, nonroot, read-only FS)
- [ ] F.4.2 Implement Pod Security Standards (restricted)
- [ ] F.4.3 Implement NetworkPolicies (default-deny + per-service allows)
- [ ] F.4.4 Implement Falco runtime monitoring
- [ ] F.4.5 Implement image signing (Cosign)
- [ ] F.4.6 Implement vulnerability scanning (Trivy)
- [ ] F.4.7 Implement SBOM generation
- [ ] F.4.8 Implement admission controllers
- **Acceptance:** Security scans pass in CI/CD

### F.5 Compliance
- [ ] F.5.1 Implement SOC 2 Type II controls
- [ ] F.5.2 Implement GDPR data handling
- [ ] F.5.3 Implement HIPAA safeguards
- [ ] F.5.4 Implement FedRAMP controls
- [ ] F.5.5 Implement audit evidence collection
- **Acceptance:** Compliance audits pass

---

## STREAM G: Documentation & Design Polish (P2 — Continuous)

### G.1 OpenDesign Refinement
- [ ] G.1.1 Expand platform token files (currently 2–7 tokens each, target 50+)
- [ ] G.1.2 Add component token groups for all 50 components
- [ ] G.1.3 Implement visual regression test suite (Playwright)
- [ ] G.1.4 Create design system documentation site
- [ ] G.1.5 Fix WCAG contrast failures (4 known issues)
- **Acceptance:** Token count ≥750, visual regression tests pass

### G.2 API Documentation
- [ ] G.2.1 Complete OpenAPI components.schemas (all 221 endpoints)
- [ ] G.2.2 Generate API client SDKs from OpenAPI
- [ ] G.2.3 Implement interactive API explorer
- **Acceptance:** All endpoints documented with schemas

### G.3 User Documentation
- [ ] G.3.1 Create video tutorials
- [ ] G.3.2 Create interactive onboarding guide
- [ ] G.3.3 Create troubleshooting wizard
- [ ] G.3.4 Translate docs to supported languages
- **Acceptance:** User docs cover all features

### G.4 ADR Updates
- [ ] G.4.1 Update ADRs as implementation reveals new constraints
- [ ] G.4.2 Create new ADRs for implementation decisions
- [ ] G.4.3 Archive superseded ADRs
- **Acceptance:** ADRs reflect actual implementation

---

## Dependency Graph

```
STREAM A (Core Backend)
  ├── A.2 Auth (blocks C.2, F.3)
  ├── A.4 Vault (blocks C.6)
  ├── A.6 SSH Proxy (blocks C.4, C.5)
  └── A.7 Terminal (blocks C.4, B.6)

STREAM B (Supporting Backend)
  ├── Depends on: A.1 Gateway, A.2 Auth
  └── B.6 Collaboration (depends on A.7 Terminal)

STREAM C (Flutter Client)
  ├── C.1 Setup (independent)
  ├── C.2 Auth (depends on A.2)
  ├── C.3 Host (depends on A.5)
  ├── C.4 Terminal (depends on A.6, A.7)
  ├── C.5 SFTP (depends on B.1)
  ├── C.6 Vault (depends on A.4)
  └── C.7 Collaboration (depends on B.6)

STREAM D (Infrastructure)
  ├── D.1 Terraform (independent)
  ├── D.2 K8s (depends on services having Dockerfiles)
  ├── D.3 Helm (depends on D.2)
  └── D.4 CI/CD (depends on all services having tests)

STREAM E (Testing)
  ├── Depends on services being implemented
  └── E.9 External (depends on submodules)

STREAM F (Security)
  ├── Continuous, applies to all streams
  └── F.1 Zero-Trust (depends on D.2 K8s)

STREAM G (Docs)
  └── Continuous, updates as implementation progresses
```

## Sprint Organization

### Sprint 1 (Week 1–2): Foundation
- A.1 Gateway (complete)
- A.2 Auth (complete)
- C.1 Project Setup (complete)
- D.1 Terraform (complete)
- F.1 Zero-Trust (start)

### Sprint 2 (Week 3–4): Core Services
- A.3 User (complete)
- A.4 Vault (complete)
- A.5 Host (complete)
- C.2 Auth Flow (complete)
- C.3 Host Management (complete)
- E.1 Unit Tests (start)

### Sprint 3 (Week 5–6): SSH & Terminal
- A.6 SSH Proxy (complete)
- A.7 Terminal (complete)
- C.4 Terminal (complete)
- C.5 SFTP (complete)
- D.2 K8s (complete)
- F.2 Crypto (complete)

### Sprint 4 (Week 7–8): Supporting Services
- B.1–B.18 (all supporting services)
- C.6 Vault (complete)
- C.7 Collaboration (complete)
- D.3 Helm (complete)
- E.2 Integration Tests (complete)

### Sprint 5 (Week 9–10): Client Completion
- C.8–C.12 (remaining screens and widgets)
- D.4 CI/CD (complete)
- E.3 E2E Tests (complete)
- E.4 Performance Tests (complete)

### Sprint 6 (Week 11–12): Hardening & Polish
- F.3–F.5 (security hardening)
- E.5–E.8 (security, chaos, mutation, a11y tests)
- G.1–G.4 (design and docs polish)
- Final integration and verification

## Physical Evidence Requirements

Every task MUST produce:
1. **Code changes** (real files, not stubs)
2. **Test results** (passing tests with coverage reports)
3. **Build artifacts** (compiled binaries, Docker images)
4. **Documentation updates** (if applicable)
5. **Verification commands** (exact commands to reproduce the result)

## Anti-Bluff Verification

- Every test must have a paired mutation test (§1.1)
- Every gate must have a meta-test proving it can fail
- Every claim must have physical evidence (screenshots, logs, artifacts)
- No task is "done" until verification commands are provided and run

## Progress Tracking

- Daily standup: What was done yesterday, what will be done today, blockers
- Weekly review: Sprint progress, demo working features
- Bi-weekly retrospective: What went well, what didn't, improvements
- Continuous: All work committed and pushed to upstreams daily
