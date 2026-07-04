#!/usr/bin/env bash
# verify_constitution_inheritance.sh
#
# Constitution-inheritance verification GATE.
#
# Checks that the `constitution/` submodule is present, populated, and that
# its forensic anchors are actually inherited by the parent repository's
# CLAUDE.md / AGENTS.md / HELIX_TERMINATOR_CONSTITUTION.md files.
#
# Exits 0 only if ALL invariants (I1..I5) pass. Exits non-zero otherwise.
set -euo pipefail

# Resolve repo root robustly so this script works from any CWD.
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

overall_status=0

pass() {
  printf 'PASS: %s\n' "$1"
}

fail() {
  printf 'FAIL: %s\n' "$1"
  overall_status=1
}

# --- I1: constitution/ directory exists ------------------------------------
if [[ -d "constitution" ]]; then
  pass "I1: constitution/ directory exists"
else
  fail "I1: constitution/ directory exists"
fi

# --- I2: constitution/Constitution.md exists AND contains forensic anchor --
if [[ -f "constitution/Constitution.md" ]] && grep -Fq '§11.4 End-user quality guarantee — forensic anchor' "constitution/Constitution.md"; then
  pass "I2: constitution/Constitution.md exists and contains forensic-anchor literal"
else
  fail "I2: constitution/Constitution.md exists and contains forensic-anchor literal"
fi

# --- I3: constitution/CLAUDE.md exists AND contains MANDATORY ANTI-BLUFF COVENANT
if [[ -f "constitution/CLAUDE.md" ]] && grep -Fq 'MANDATORY ANTI-BLUFF COVENANT' "constitution/CLAUDE.md"; then
  pass "I3: constitution/CLAUDE.md exists and contains MANDATORY ANTI-BLUFF COVENANT"
else
  fail "I3: constitution/CLAUDE.md exists and contains MANDATORY ANTI-BLUFF COVENANT"
fi

# --- I4: constitution/AGENTS.md exists AND contains Anti-bluff covenant ----
if [[ -f "constitution/AGENTS.md" ]] && grep -Fq 'Anti-bluff covenant' "constitution/AGENTS.md"; then
  pass "I4: constitution/AGENTS.md exists and contains Anti-bluff covenant"
else
  fail "I4: constitution/AGENTS.md exists and contains Anti-bluff covenant"
fi

# --- I5: parent files reference the submodule ------------------------------
i5_ok=1
if [[ -f "CLAUDE.md" ]] && grep -Fq 'constitution/CLAUDE.md' "CLAUDE.md"; then
  :
else
  i5_ok=0
fi
if [[ -f "AGENTS.md" ]] && grep -Fq 'constitution/AGENTS.md' "AGENTS.md"; then
  :
else
  i5_ok=0
fi
if [[ -f "docs/guides/HELIX_TERMINATOR_CONSTITUTION.md" ]] && grep -Fq 'constitution/Constitution.md' "docs/guides/HELIX_TERMINATOR_CONSTITUTION.md"; then
  :
else
  i5_ok=0
fi

if [[ "$i5_ok" -eq 1 ]]; then
  pass "I5: ./CLAUDE.md, ./AGENTS.md, ./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md all reference the submodule"
else
  fail "I5: ./CLAUDE.md, ./AGENTS.md, ./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md all reference the submodule"
fi

if [[ "$overall_status" -eq 0 ]]; then
  echo "RESULT: ALL INVARIANTS PASSED"
else
  echo "RESULT: ONE OR MORE INVARIANTS FAILED"
fi

exit "$overall_status"
