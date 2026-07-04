#!/usr/bin/env bash
# test_constitution_inheritance.sh
#
# HOST-SIDE constitution-inheritance test.
#
# Verifies that:
#   - the `constitution` git submodule is present, populated, and carries the
#     expected forensic anchors (Constitution.md / CLAUDE.md / AGENTS.md)
#   - the parent repository's own CLAUDE.md / AGENTS.md / docs guide reference
#     the submodule (i.e. the parent actually inherits from it)
#   - .gitmodules / `git submodule status` agree that `constitution` is wired up
#     as a real submodule
#   - RECURSIVELY, any submodule owned by this repo other than `constitution`
#     itself also inherits the Helix Constitution in its own CLAUDE.md/AGENTS.md
#     (this currently is an empty set, and that is reported honestly rather
#     than faked)
#
# This script is fully self-contained: it does not source or depend on any
# other test/gate script in the repository.
#
# Exit status: 0 iff every assertion below PASSes. Non-zero otherwise.

set -euo pipefail

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

cd "$(git rev-parse --show-toplevel)"

PASS_COUNT=0
FAIL_COUNT=0
FAILED_ASSERTIONS=()

pass() {
  local id="$1" msg="$2"
  echo "PASS [$id] $msg"
  PASS_COUNT=$((PASS_COUNT + 1))
}

fail() {
  local id="$1" msg="$2"
  echo "FAIL [$id] $msg"
  FAIL_COUNT=$((FAIL_COUNT + 1))
  FAILED_ASSERTIONS+=("$id")
}

echo "=============================================================="
echo "Constitution inheritance test (host-side)"
echo "Repo root: $(pwd)"
echo "=============================================================="

# ---------------------------------------------------------------------------
# A1: constitution/ directory exists and is non-empty
# ---------------------------------------------------------------------------

if [[ -d constitution ]] && [[ -n "$(ls -A constitution 2>/dev/null)" ]]; then
  pass "A1" "constitution/ directory exists and is non-empty"
else
  fail "A1" "constitution/ directory is missing or empty"
fi

# ---------------------------------------------------------------------------
# A2: constitution/Constitution.md carries the forensic anchor
# ---------------------------------------------------------------------------

A2_ANCHOR='§11.4 End-user quality guarantee — forensic anchor'
if [[ -f constitution/Constitution.md ]] && grep -Fq "$A2_ANCHOR" constitution/Constitution.md; then
  pass "A2" "constitution/Constitution.md contains forensic anchor '$A2_ANCHOR'"
else
  fail "A2" "constitution/Constitution.md missing forensic anchor '$A2_ANCHOR'"
fi

# ---------------------------------------------------------------------------
# A3: constitution/CLAUDE.md carries the anti-bluff covenant anchor
# ---------------------------------------------------------------------------

A3_ANCHOR='MANDATORY ANTI-BLUFF COVENANT'
if [[ -f constitution/CLAUDE.md ]] && grep -Fq "$A3_ANCHOR" constitution/CLAUDE.md; then
  pass "A3" "constitution/CLAUDE.md contains anchor '$A3_ANCHOR'"
else
  fail "A3" "constitution/CLAUDE.md missing anchor '$A3_ANCHOR'"
fi

# ---------------------------------------------------------------------------
# A4: constitution/AGENTS.md carries the anti-bluff covenant anchor
# ---------------------------------------------------------------------------

A4_ANCHOR='Anti-bluff covenant'
if [[ -f constitution/AGENTS.md ]] && grep -Fq "$A4_ANCHOR" constitution/AGENTS.md; then
  pass "A4" "constitution/AGENTS.md contains anchor '$A4_ANCHOR'"
else
  fail "A4" "constitution/AGENTS.md missing anchor '$A4_ANCHOR'"
fi

# ---------------------------------------------------------------------------
# A5: parent files reference the submodule
# ---------------------------------------------------------------------------

if [[ -f CLAUDE.md ]] && grep -Fq 'constitution/CLAUDE.md' CLAUDE.md; then
  pass "A5.1" "./CLAUDE.md references constitution/CLAUDE.md"
else
  fail "A5.1" "./CLAUDE.md does not reference constitution/CLAUDE.md"
fi

if [[ -f AGENTS.md ]] && grep -Fq 'constitution/AGENTS.md' AGENTS.md; then
  pass "A5.2" "./AGENTS.md references constitution/AGENTS.md"
else
  fail "A5.2" "./AGENTS.md does not reference constitution/AGENTS.md"
fi

if [[ -f docs/guides/HELIX_TERMINATOR_CONSTITUTION.md ]] && grep -Fq 'constitution/Constitution.md' docs/guides/HELIX_TERMINATOR_CONSTITUTION.md; then
  pass "A5.3" "./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md references constitution/Constitution.md"
else
  fail "A5.3" "./docs/guides/HELIX_TERMINATOR_CONSTITUTION.md does not reference constitution/Constitution.md"
fi

# ---------------------------------------------------------------------------
# A6: .gitmodules + `git submodule status` agree constitution is wired up
# ---------------------------------------------------------------------------

if [[ -f .gitmodules ]] && grep -Fq '[submodule "constitution"]' .gitmodules; then
  pass "A6.1" ".gitmodules contains a [submodule \"constitution\"] entry"
else
  fail "A6.1" ".gitmodules is missing a [submodule \"constitution\"] entry"
fi

SUBMODULE_STATUS="$(git submodule status 2>&1 || true)"
if echo "$SUBMODULE_STATUS" | awk '{print $2}' | grep -Fxq 'constitution'; then
  pass "A6.2" "git submodule status lists 'constitution'"
else
  fail "A6.2" "git submodule status does not list 'constitution' (got: $SUBMODULE_STATUS)"
fi

# ---------------------------------------------------------------------------
# A7: RECURSIVE nested-submodule inheritance
#
# Enumerate every submodule path known recursively, excluding the top-level
# `constitution` submodule itself. For each such path (i.e. a submodule owned
# by this repository below/alongside constitution), assert that its own
# CLAUDE.md and AGENTS.md reference the Helix Constitution.
#
# If no such submodules exist (currently true — `constitution` is the only
# submodule in the tree), this is reported honestly and PASSes vacuously.
# ---------------------------------------------------------------------------

RECURSIVE_STATUS="$(git submodule status --recursive 2>&1 || true)"
echo "--- git submodule status --recursive ---"
echo "$RECURSIVE_STATUS"
echo "-----------------------------------------"

mapfile -t NESTED_SUBMODULES < <(
  echo "$RECURSIVE_STATUS" | awk 'NF>=2 {print $2}' | grep -Fxv 'constitution' || true
)

if [[ "${#NESTED_SUBMODULES[@]}" -eq 0 ]]; then
  pass "A7" "no owned nested submodules — recursive-pointer invariant vacuously satisfied"
else
  echo "Found ${#NESTED_SUBMODULES[@]} owned nested submodule(s) beyond 'constitution': ${NESTED_SUBMODULES[*]}"
  A7_ALL_OK=1
  for sub in "${NESTED_SUBMODULES[@]}"; do
    sub_ok=1

    if [[ -f "$sub/CLAUDE.md" ]] && grep -Eiq 'helix[^a-z0-9]*constitution|constitution/CLAUDE\.md|constitution/AGENTS\.md' "$sub/CLAUDE.md"; then
      pass "A7[$sub].CLAUDE.md" "$sub/CLAUDE.md references the Helix Constitution"
    else
      fail "A7[$sub].CLAUDE.md" "$sub/CLAUDE.md missing or does not reference the Helix Constitution"
      sub_ok=0
    fi

    if [[ -f "$sub/AGENTS.md" ]] && grep -Eiq 'helix[^a-z0-9]*constitution|constitution/CLAUDE\.md|constitution/AGENTS\.md' "$sub/AGENTS.md"; then
      pass "A7[$sub].AGENTS.md" "$sub/AGENTS.md references the Helix Constitution"
    else
      fail "A7[$sub].AGENTS.md" "$sub/AGENTS.md missing or does not reference the Helix Constitution"
      sub_ok=0
    fi

    [[ "$sub_ok" -eq 1 ]] || A7_ALL_OK=0
  done

  if [[ "$A7_ALL_OK" -eq 1 ]]; then
    pass "A7" "all owned nested submodules inherit the Helix Constitution"
  else
    fail "A7" "one or more owned nested submodules fail to inherit the Helix Constitution"
  fi
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

echo "=============================================================="
echo "Summary: $PASS_COUNT passed, $FAIL_COUNT failed"
if [[ "$FAIL_COUNT" -gt 0 ]]; then
  echo "Failed assertions: ${FAILED_ASSERTIONS[*]}"
  echo "RESULT: FAIL"
  exit 1
else
  echo "RESULT: PASS"
  exit 0
fi
