# QA evidence — HT-SSHCA-001 (PKI short-lived SSH certificate authority)

**Feature:** `services/pki-service/internal/sshca` — mint + verify short-lived OpenSSH
user/host certificates (SERVICE_REGISTRY.md §19). **Branch:** `feature/pki-ssh-certificates`.
**Constitution:** §11.4.43/§11.4.115 (RED→GREEN), §11.4.107(10) (self-validated oracle),
§11.4.5/§11.4.69 (captured evidence), §11.4.83 (this transcript), §11.4.50 (determinism).

## Artifacts

| File | What it proves |
|---|---|
| `01_RED_baseline.log` | RED: before the package existed, `go test` fails to build — the capability is provably absent (the gap reproduced). |
| `02_GREEN_result.log` | GREEN: `go test -race -count=1` → `ok ... coverage: 80.8%`. All 11 tests pass, incl. the golden-bad rejection suite. |
| `03_ssh_keygen_independent_oracle.log` | Independent oracle: the SYSTEM `ssh-keygen -L` (OpenSSH 9.6p1) parses our issued certs and confirms cert type (user/host), Key ID, serial, short validity window, principals (`alice`,`deploy` / `web01.helixterminator.io`), and user extensions (permit-pty …). A different implementation confirms correctness. |
| `coverage.out` | Go coverage profile (80.8% of statements). |

## Anti-bluff notes

- The golden-bad suite (`TestVerify_Rejects*`) proves the `VerifyCertificate` oracle is
  non-tautological: it MUST reject certs signed by a different CA, tampered certs, expired
  certs, wrong-principal, and wrong-cert-type. During this pass the golden-bad
  `TestVerify_RejectsCertSignedByDifferentCA` **caught a real security defect** — the first
  implementation accepted any self-consistent cert because `ssh.CertChecker.CheckCert` does
  not check the issuing authority. Fixed by an explicit `cert.SignatureKey == trusted CA`
  guard (see the comment in `sshca.go`). This is the §11.4.107(10) self-validation working
  as designed.
- Determinism (§11.4.50): 3 consecutive `go test` runs all PASS.

## Honest boundary (§11.4.6)

This validates the **crypto core**. The HTTP API + persistence layer (create/store CAs,
sign endpoints, DB-gated integration) is tracked as **HT-SSHCA-002** (needs a live Postgres
for proper TDD) and is NOT claimed done here.
