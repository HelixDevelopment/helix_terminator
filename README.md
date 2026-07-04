# helix_terminator

A Go project owned by HelixDevelopment.

## What It Is

`helix_terminator` is a Go application developed under the governance of the Helix Constitution framework, which establishes universal rules for quality, transparency, and anti-bluff verification across all related projects.

## Governance

This project's governance is inherited from the **Helix Constitution**, mounted as a git submodule at `constitution/` (commit e6504c273c8b352fdb180449c9f057704cf85671, branch main).

All universal rules, policies, and quality standards are defined in the constitution. Project-specific rules are documented in:

- `./CLAUDE.md` — Claude Code project instructions
- `./AGENTS.md` — Agent operating rules
- `./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md` — Project-specific extensions and overrides (currently none)

When this repository's rules disagree with the constitution submodule, **the constitution wins**.

## How to Verify Inheritance

Constitution inheritance is verified by three gate scripts that prove the constitution is actually inherited (not "bluff" gates that claim to check but don't):

1. **Gate (invariant check):**
   ```bash
   bash tests/verify_constitution_inheritance.sh
   ```
   Verifies that:
   - The `constitution/` directory exists and is populated
   - The constitutional anchor strings are present in the submodule
   - The parent repository files reference the submodule (invariants I1..I5)

2. **§1.1 Anti-bluff mutation proof:**
   ```bash
   bash scripts/testing/meta_test_false_positive_proof.sh
   ```
   Proves the gate is not a bluff by:
   - Mutating the forensic anchor in the constitution
   - Asserting the gate fails
   - Restoring the file and asserting the gate passes
   - Confirming the constitution/ tree remains pristine throughout

Run the gate before every build or merge.

## Constitution Privacy Policy

The constitution is **treated as public-by-policy**. No secrets, credentials, or project-specific configurations are ever ported into the constitution submodule. All sensitive or project-specific material stays in the parent repository. The constitution remains a reusable, shareable governance framework for all HelixDevelopment projects.

## Further Reading

- `docs/CONSTITUTION_INHERITANCE.md` — Technical details on the constitution architecture, upstreams, and promotion policy
- `constitution/Constitution.md` — The universal Helix Constitution (in submodule)
- `constitution/CLAUDE.md` — Universal rules for Claude Code and agents (in submodule)
- `constitution/AGENTS.md` — Universal agent operating rules (in submodule)
