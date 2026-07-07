# Constitution Inheritance Architecture

This document details how `helix_terminator` inherits governance from the Helix Constitution submodule, how inheritance is verified, and how rules are promoted between the project and the universal constitution.

## Submodule Configuration

The Helix Constitution is mounted as a git submodule:

- **Path:** `constitution/`
- **Source:** `git@github.com:HelixDevelopment/HelixConstitution.git`
- **Pinned commit:** `e6504c273c8b352fdb180449c9f057704cf85671`
- **Branch:** `main`
- **Version tag:** None (the constitution does not use tagged releases)

### Fetching the Submodule

To clone this repository with the constitution submodule initialized:

```bash
git clone --recursive git@github.com:HelixDevelopment/helix_terminator.git
cd helix_terminator
```

Or, if already cloned without the submodule:

```bash
git submodule update --init --recursive
```

## Six Upstreams

The constitution declares six upstream repositories in `constitution/upstreams/`, enabling bidirectional synchronization:

1. **GitHub** (`upstreams/GitHub.sh`)
   - Primary upstream: HelixDevelopment organization on GitHub

2. **GitLab** (`upstreams/GitLab.sh`)
   - Mirror/alternative upstream on GitLab

3. **GitFlic** (`upstreams/GitFlic.sh`)
   - Distributed git hosting upstream

4. **GitVerse** (`upstreams/GitVerse.sh`)
   - Decentralized git protocol upstream

5. **Vasic-Digital GitHub** (`upstreams/VasicDigitalGitHub.sh`)
   - Organizational backup upstream on GitHub

6. **Vasic-Digital GitLab** (`upstreams/VasicDigitalGitLab.sh`)
   - Organizational backup upstream on GitLab

These upstreams support the constitution's resilience model: universal rules can be synchronized across multiple git hosts, reducing single-point-of-failure risk.

## Verified Anchor Strings

Constitution inheritance is proven by three forensic anchor strings embedded in the constitution's source files. These anchors are checked by the gate to prove the constitution is actually present, not merely claimed:

### 1. Constitution.md — Forensic Anchor

**Location:** `constitution/Constitution.md`

**Anchor string:** `§11.4 End-user quality guarantee — forensic anchor`

**Purpose:** Anchors the universal quality guarantee clause that requires end-to-end verification of governance inheritance.

### 2. CLAUDE.md — Anti-bluff Covenant

**Location:** `constitution/CLAUDE.md`

**Anchor string:** `MANDATORY ANTI-BLUFF COVENANT`

**Purpose:** Anchors the mandatory covenant requiring all gates to have paired mutation proofs (§1.1), ensuring no "bluff gates" that claim to verify without actually biting.

### 3. AGENTS.md — Anti-bluff Covenant

**Location:** `constitution/AGENTS.md`

**Anchor string:** `Anti-bluff covenant`

**Purpose:** Anchors the agent operating covenant requiring anti-bluff verification for agent-side governance checks.

## Inheritance Pointers in Parent Repository

Three files in the parent repository declare their inheritance from the constitution:

### 1. ./CLAUDE.md

Lines 3-11 declare inheritance:

```markdown
## INHERITED FROM constitution/CLAUDE.md

All rules in `constitution/CLAUDE.md` (and the
`constitution/Constitution.md` it references) apply unconditionally.
...

@constitution/CLAUDE.md
```

This file is checked by the gate (invariant I5).

### 2. ./AGENTS.md

Lines 3-4 declare inheritance:

```markdown
> Base agent rules: `constitution/AGENTS.md` — READ IT FIRST.
```

This file is checked by the gate (invariant I5).

### 3. ./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md

Lines 3-4 declare extension:

```markdown
This constitution **extends** the Helix Universal Constitution at
`constitution/Constitution.md`.
```

This file is checked by the gate (invariant I5).

## Verification Tooling

### Gate Script: tests/verify_constitution_inheritance.sh

**Purpose:** Verification gate that checks all inheritance invariants before build/merge.

**Location:** `tests/verify_constitution_inheritance.sh`

**Exit code:** 0 (all invariants pass), 1+ (one or more invariants fail)

**Invariants checked (I1..I5):**

- **I1:** `constitution/` directory exists
- **I2:** `constitution/Constitution.md` exists and contains the forensic anchor `§11.4 End-user quality guarantee — forensic anchor`
- **I3:** `constitution/CLAUDE.md` exists and contains the anchor `MANDATORY ANTI-BLUFF COVENANT`
- **I4:** `constitution/AGENTS.md` exists and contains the anchor `Anti-bluff covenant`
- **I5:** All three parent files (`./CLAUDE.md`, `./AGENTS.md`, `./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md`) reference the submodule

**How to run:**

```bash
bash tests/verify_constitution_inheritance.sh
```

Example output on pass:

```
PASS: I1: constitution/ directory exists
PASS: I2: constitution/Constitution.md exists and contains forensic-anchor literal
PASS: I3: constitution/CLAUDE.md exists and contains MANDATORY ANTI-BLUFF COVENANT
PASS: I4: constitution/AGENTS.md exists and contains Anti-bluff covenant
PASS: I5: ./CLAUDE.md, ./AGENTS.md, ./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md all reference the submodule
RESULT: ALL INVARIANTS PASSED
```

### Mutation Proof: scripts/testing/meta_test_false_positive_proof.sh

**Purpose:** §1.1 false-positive proof demonstrating the gate is not a "bluff gate."

**Location:** `scripts/testing/meta_test_false_positive_proof.sh`

**Exit code:** 0 (gate correctly rejects mutation and tree is pristine), 1+ (gate is bluff or tree corrupted)

**Steps performed:**

1. Backs up `constitution/Constitution.md`
2. Mutates the forensic anchor (replaces `§11.4 End-user quality guarantee — forensic anchor` with `MUTATED_OUT`)
3. Runs the gate; asserts it **fails** (exit != 0)
4. Restores the file
5. Runs the gate; asserts it **passes** (exit == 0)
6. Verifies the constitution/ submodule tree is pristine (no uncommitted changes)

**How to run:**

```bash
bash scripts/testing/meta_test_false_positive_proof.sh
```

Example output on pass:

```
=== Step 1: backing up ... ===
Backup created at /tmp/Constitution.md.backup.XXXXXX

=== Step 2: mutating forensic anchor in ... ===
Mutation applied successfully.

=== Step 3: running gate against MUTATED file (expect non-zero exit) ===
Gate exit code on mutated file: 1
PASS: gate correctly FAILED on mutated forensic anchor (not a bluff gate).

=== Step 4: restoring ... ===
Gate exit code on restored file: 0
PASS: gate correctly PASSED after restoring the forensic anchor.

=== Step 5: asserting constitution/ submodule tree is pristine ===
PASS: constitution/ submodule tree is pristine (git status --porcelain is empty).

RESULT: META-TEST PASSED — gate correctly rejects the false positive and tree is pristine.
```

## Anti-bluff Pairing: §1.1 Constitution Covenant

Every gate in the Helix Constitution must be paired with a mutation proof demonstrating it actually bites.

**The pairing for this project:**

- **Gate:** `tests/verify_constitution_inheritance.sh` (checks constitution is inherited)
- **Mutation proof:** `scripts/testing/meta_test_false_positive_proof.sh` (proves gate rejects a false positive)

The mutation proof explicitly mutates the forensic anchor in `constitution/Constitution.md` and verifies that:

1. The gate **rejects** the mutation (fails as expected)
2. After restoration, the gate **accepts** the file again (passes as expected)
3. The constitution/ submodule is left pristine (no side effects)

This satisfies §1.1: **no bluff gates that claim to verify without actually biting**.

## Rule Promotion Policy

Rules follow a two-tier system:

### Universal Rules (promoted to constitution/)

Universal rules apply to all HelixDevelopment projects and are stored in the constitution submodule. Once a rule is proven universal across multiple projects:

1. **Draft in parent:** Write the rule in `./CLAUDE.md`, `./AGENTS.md`, or `./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md`
2. **Prove universality:** Demonstrate the rule applies to other projects
3. **Promote via PR:** Submit a pull request to the constitution repository (HelixDevelopment/HelixConstitution.git)
4. **Never tag:** Universal rules are promoted to branch `main`, never to a tagged release
5. **Sync submodule:** Update the pinned SHA in `helix_terminator` to include the new rule

### Project-Specific Rules (stay in parent)

Rules that are specific to helix_terminator or do not meet the universal bar remain in:

- `./CLAUDE.md` (project-specific Claude Code rules)
- `./AGENTS.md` (project-specific agent rules)
- `./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md` (project-specific constitutional extensions)

These files extend the constitution but never weaken it.

## Current Status

As of the initial constitution integration:

- **Universal rules promoted:** None (the parent repository is a Go skeleton; no project-specific rules have yet proven universal across multiple projects)
- **Overrides in place:** None (see `./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md`)
- **Project-specific rules:** All configuration stays in the parent repository

When genuinely universal rules emerge, they will be contributed upstream to `HelixDevelopment/HelixConstitution.git` via pull request, per the Constitution's promotion policy.

## Public-by-Policy

The constitution is treated as **public-by-policy**:

- No secrets, credentials, or sensitive project data are stored in the constitution submodule
- No project-specific configurations that could leak implementation details are ported upstream
- The constitution remains a reusable, shareable governance framework

All sensitive or project-specific material stays in the parent repository (`helix_terminator/`).

## References

- `constitution/Constitution.md` — The universal Helix Constitution
- `constitution/CLAUDE.md` — Universal Claude Code and agent rules
- `constitution/AGENTS.md` — Universal agent operating rules
- `./CLAUDE.md` — Project-specific Claude Code rules
- `./AGENTS.md` — Project-specific agent rules
- `./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md` — Project-specific constitutional extensions
