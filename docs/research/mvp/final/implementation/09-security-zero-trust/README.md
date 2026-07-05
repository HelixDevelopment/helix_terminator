# 09 — Security — Zero Trust

**Status:** `Draft`  
**Module:** A + B  
**Authority:** `CANONICAL_FACTS.md` (CD-7, CD-8, CD-10) + `SERVICE_REGISTRY.md`  

---

## Overview

HelixTerminator implements a zero-trust security model: every connection, every byte of stored credential, every API call is cryptographically authenticated. There are no unencrypted paths.

| Layer | Technology |
|-------|-----------|
| Identity | SPIFFE/SPIRE, Istio mTLS |
| Auth | EdDSA (Ed25519) JWT, FIDO2/WebAuthn, SAML 2.0, OIDC |
| Encryption | AES-256-GCM, Argon2id, Vault envelope encryption |
| PKI | Short-lived SSH certificates (TTL=8h), CA rotation |
| Audit | Merkle-chained append-only log, crypto-shred GDPR erasure |
| Compliance | SOC 2 Type II, ISO 27001, FedRAMP Moderate, GDPR |

---

## Authentication

### JWT: EdDSA (Ed25519)

Canonical per CD-7. Token structure:
```json
{
  "iss": "https://auth.helixterminator.io",
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "aud": ["helixterm:api"],
  "exp": 1751120400,
  "iat": 1751116800,
  "jti": "unique-token-id",
  "scope": "api:read api:write",
  "org_id": "org-uuid",
  "session_id": "session-uuid",
  "mfa_verified": true
}
```

| Token Type | Lifetime | Storage |
|------------|----------|---------|
| Access token | 15 minutes | Memory |
| Refresh token | 30 days (sliding) | HttpOnly Secure cookie |
| API key | No expiry (until revoked) | Hashed in DB |

### MFA Methods

- **TOTP:** RFC 6238, 6-digit, 30-second window
- **FIDO2/WebAuthn:** CTAP2-compliant hardware keys (YubiKey 5, Google Titan)
- **Biometric:** Face ID, Touch ID, fingerprint, Windows Hello
- **Backup codes:** 8 single-use codes per TOTP setup

### SSO

- SAML 2.0 + OIDC
- SCIM 2.0 inbound provisioning
- IdP OAuth tokens envelope-encrypted at rest (pgcrypto)

---

## RBAC

Canonical 6-role vocabulary per CD-8:

| Role | Scope | Permissions |
|------|-------|-------------|
| `super_admin` | Cross-org | Platform operations, break-glass |
| `org_admin` | Organization | Membership, billing, policy, settings |
| `team_admin` | Team | Team membership, host access control |
| `member` | Self | Day-to-day SSH, vault, collaboration |
| `auditor` | Read-only | Audit trails, compliance exports |
| `api_user` | Scoped | CI/CD automation, service accounts |

Custom roles via `roles` / `role_assignments` tables with JSONB permission arrays.

---

## Zero-Knowledge Vault (CD-10)

- **HARD requirement:** Client-side end-to-end encryption only
- Server stores only ciphertext
- Key derivation: Argon2id (time_cost=3, memory_cost=65536, parallelism=4)
- AES-256-GCM with per-item IV
- **Explicitly non-ZK:** SSH password-auth hosts (carved out per CD-10)

> **DEFERRED:** Server-side key generation and re-wrap must be removed from the spec's design in a future security redesign increment.

---

## SPIFFE/SPIRE + mTLS

- Every pod receives a SPIFFE ID: `spiffe://helixterminator.io/ns/<namespace>/sa/<serviceaccount>`
- Istio mTLS in STRICT mode — no plaintext service-to-service communication
- AuthorizationPolicy enforces per-service RBAC at the mesh level

---

## Audit & Compliance

### Merkle-Chained Audit Log

- Each event references SHA-256 hash of previous event
- Append-only; hash chain verification on every read
- PII columns envelope-encrypted with per-subject DEK
- GDPR erasure via crypto-shredding (DEK destruction, not row mutation)
- **DEFERRED:** External WORM anchoring (S3 Object Lock, HSM signature, notarization)

### Compliance Frameworks

| Framework | Scope | Evidence Source |
|-----------|-------|-----------------|
| SOC 2 Type II | All services | Audit events, access logs, change management |
| ISO 27001 | All services | ISMS policies, risk register, control testing |
| FedRAMP Moderate | Enterprise tier | Controls mapped to NIST 800-53 |
| GDPR | EU users | Crypto-shred erasure, data residency, DPO |
| HIPAA | Healthcare customers | BAA, encryption at rest/transit, access controls |

---

## Diagrams

| Diagram | Source |
|---------|--------|
| Vault Encryption (Draw.io) | `diagrams/drawio/03_vault_encryption.drawio` |
| Zero-Trust Network (Draw.io) | `diagrams/drawio/05_zero_trust_network.drawio` |
| SSH Sequence | `diagrams/mermaid/06_ssh_sequence.mmd` |
| Auth Sequence | `diagrams/mermaid/07_auth_sequence.mmd` |
| Vault Sequence | `diagrams/mermaid/08_vault_sequence.mmd` |
| mTLS Sequence | `diagrams/mermaid/14_mtls_sequence.mmd` |

---

## Cross-References

- [04 — API Specification](../04-api-specification/) — Auth endpoints, JWT spec
- [05 — Database Schema](../05-database-schema/) — Audit tables, RLS policies, PII encryption
- [08 — DevOps Infrastructure](../08-devops-infrastructure/) — Network policies, PSS, Istio
- [14 — Constitution Compliance](../14-constitution-compliance/) — Governance, CI gates
- [16 — References](../16-references/) — CD-7 (JWT), CD-8 (RBAC), CD-10 (ZK)

---

*Section 09 — Security — Zero Trust*  
*Consolidated from: 05_security_zero_trust.md, CANONICAL_FACTS.md (CD-7, CD-8, CD-10)*
