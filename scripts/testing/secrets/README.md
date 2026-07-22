# scripts/testing/secrets/

Local, operator-owned credential files for the feature tiers this project
already implements (push, payments, email — see
[`docs/credentials/FEATURE_TIER_SETUP.md`](../../../docs/credentials/FEATURE_TIER_SETUP.md)),
per the Constitution §11.4.10 credentials-handling mandate ("Per-service
file separation limits blast radius").

This directory is **gitignored** except for this `README.md`, a
`.gitkeep` (so the empty directory itself stays tracked), and any
`*.example` placeholder files — real credentials placed here are
**never** committed. See the root `.gitignore` `scripts/testing/secrets/*`
block.

## What goes here

Any per-service credential an operator prefers to keep as a local file
rather than a shell-exported environment variable — for example, a
scratch copy of a Stripe **test-mode** secret key used by a local
scripting session, or SMTP relay credentials for a local test harness.
None of the services in this repository read a file from this directory
directly — every real credential is consumed via the environment
variables documented in
[`docs/credentials/FEATURE_TIER_SETUP.md`](../../../docs/credentials/FEATURE_TIER_SETUP.md).
This directory exists as a safe, pre-gitignored place to *stage* those
values locally (e.g. `source scripts/testing/secrets/stripe.env` before
running a manual test) instead of leaving them in shell history or an
untracked-but-un-ignored stray file.

The Firebase/FCM service-account JSON has its **own** dedicated,
already-gitignored location — `scripts/firebase/secrets/` (populated by
`scripts/firebase/firebase_setup.sh`) — do not duplicate it here.

## Rules (Constitution §11.4.10 / §11.4.30)

- `chmod 600` every credential file you place here; this directory
  itself should be `chmod 700`.
- **Never** commit a real secret. Review `git status` / `git diff
  --staged` before every commit (§11.4.30 pre-commit attention) — a
  tracked file matching a secret pattern despite the `.gitignore` line
  is a violation of equal severity to a missing ignore rule.
- Before storing any operator-supplied credential here (or anywhere),
  audit for prior accidental leaks per Constitution §11.4.10.A
  (`git ls-files | xargs grep -l <value>` and
  `git log -S<value> --all --source --remotes`) **before** persisting
  it.
- Rotate immediately on any suspected leak.

## See also

- [`docs/credentials/FEATURE_TIER_SETUP.md`](../../../docs/credentials/FEATURE_TIER_SETUP.md)
  — the per-tier operator setup checklist (which env var, which file,
  how to confirm it activated).
- [`infrastructure/docker/compose/.env.example`](../../../infrastructure/docker/compose/.env.example)
  — the tracked placeholder template for compose-based local bring-up.
- `scripts/firebase/firebase_setup.sh` — the FCM/APNs service-account
  provisioning script (writes to `scripts/firebase/secrets/`, not here).
