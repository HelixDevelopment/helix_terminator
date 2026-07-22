# Operator Request History — sample-fixture

**Revision:** 1
**Last modified:** 2026-01-01T00:00:00Z

Constitution §11.4.208 operator-request-history ledger. Append-only, **newest-first**.
Every operator request/prompt is recorded with exactly five fields: **Request
content**, **Accepted (when)**, **Track**, **Alias**, **Model + effort**. This
document is project-local (§11.4.208(F)) — the *rule* mandating it lives in the
constitution submodule; this file and its contents are this project's own data.

## Project-declared default timezone

**Asia/Aqtau (UTC+5)** — synthetic fixture value, mirrors the real ledger's
declared default for test purposes only.

---

## Entries (newest first)

### 1. 2026-01-01 — Pre-existing sample entry (fixture baseline)

- **Request content (SUMMARY, not verbatim):** a pre-existing entry used as a
  fixture baseline so the insertion tests can prove a new entry is prepended
  ABOVE this one, and that this one is left byte-identical afterward.
- **Accepted (when):** 2026-01-01T00:00:00Z (UTC).
- **Track:** UNKNOWN.
- **Alias:** UNKNOWN.
- **Model + effort:** UNKNOWN.
- **Source:** synthetic test fixture, not a real session.

---

## Keep-applying mechanism (§11.4.208(D) — honest status)

**NOT YET WIRED** in this fixture (fixture text only, mirrors the real
document's trailing section so the insertion tests exercise the same overall
file shape, including content AFTER the Entries section).
