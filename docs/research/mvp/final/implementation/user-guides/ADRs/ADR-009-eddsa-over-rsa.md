# ADR-009: EdDSA (Ed25519) over RSA for JWT Signing

## Status
Accepted

## Context
helix_terminator issues JSON Web Tokens (JWTs) for user sessions, service-to-service authentication, and SPIFFE JWT-SVIDs. The signing algorithm must be secure, performant, and future-proof against emerging cryptographic threats.

## Decision
We chose **EdDSA with Ed25519** as the sole algorithm for JWT signing. RSA (RS256) is explicitly disabled in all token issuers and verifiers.

## Consequences

### Positive
- **Performance**: Ed25519 signing and verification are significantly faster than RSA-2048/4096, reducing latency on the auth service and API gateway.
- **Key size**: Ed25519 public keys are 32 bytes and private keys are 64 bytes, versus 256+ bytes for RSA-2048, simplifying key distribution and storage.
- **Security margin**: Ed25519 is designed to resist side-channel attacks and does not require constant-time big-integer arithmetic implementations.
- **Deterministic signatures**: Ed25519 does not require a random nonce during signing, eliminating a class of nonce-reuse vulnerabilities.
- **Standardization**: Ed25519 is specified in RFC 8032 and supported by `go-jose`, `jwt-go` (v5), and Vault transit.

### Negative
- **Legacy compatibility**: Some older JWT libraries and third-party integrations do not support EdDSA; adapters or upgrades are required.
- **FIPS 140-2**: Ed25519 is not FIPS-approved at the time of writing; if FIPS compliance becomes mandatory, we may need a hybrid or fallback strategy (e.g., ECDSA P-256).
- **Key ceremony**: Hardware Security Module (HSM) support for Ed25519 is less ubiquitous than RSA; verify HSM compatibility before procurement.

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| **RSA-2048 / RSA-4096 (RS256)** | Slower, larger keys and signatures, and vulnerable to implementation flaws in padding (PKCS#1 v1.5). RSA-4096 signatures are ~512 bytes, bloating HTTP headers and cookies. |
| **ECDSA (ES256, ES384)** | Faster than RSA and FIPS-approved, but requires high-quality randomness for nonce generation; nonce reuse leaks private keys (Sony PS3 incident). Ed25519 is safer by design. |
| **RSA-PSS (PS256)** | Better padding than PKCS#1 v1.5, but still carries RSA’s size and performance penalties. |
| **HMAC (HS256)** | Fast and simple, but requires symmetric key distribution, which is unsuitable for cross-service JWT verification where the verifier should not possess the signing key. Retained only for internal short-lived service tokens where symmetric keys are acceptable. |

## References
- `infrastructure/security/jwt/` — JWT signing key configuration
- `services/auth-service/` — Token issuance and validation
- `docs/guides/runbooks/KEY_ROTATION.md` — Ed25519 key rotation procedures
