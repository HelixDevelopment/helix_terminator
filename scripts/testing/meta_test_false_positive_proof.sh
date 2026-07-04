#!/usr/bin/env bash
# meta_test_false_positive_proof.sh
#
# §1.1 false-positive mutation proof, paired with
# tests/verify_constitution_inheritance.sh.
#
# Proves the gate is not a "bluff gate": mutates the forensic anchor in
# constitution/Constitution.md and asserts the gate FAILS, then restores the
# file and asserts the gate PASSES again, leaving the constitution/ submodule
# tree pristine no matter what happens (success, failure, or interruption).
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

GATE="$REPO_ROOT/tests/verify_constitution_inheritance.sh"
TARGET="$REPO_ROOT/constitution/Constitution.md"
ANCHOR='§11.4 End-user quality guarantee — forensic anchor'
MUTATED='MUTATED_OUT'

overall_status=0

BACKUP="$(mktemp /tmp/Constitution.md.backup.XXXXXX)"

restore() {
  if [[ -f "$BACKUP" ]]; then
    cp "$BACKUP" "$TARGET"
    rm -f "$BACKUP"
  fi
}
trap restore EXIT ERR INT

echo "=== Step 1: backing up $TARGET ==="
cp "$TARGET" "$BACKUP"
echo "Backup created at $BACKUP"

echo
echo "=== Step 2: mutating forensic anchor in $TARGET ==="
sed -i "s/${ANCHOR}/${MUTATED}/g" "$TARGET"
if grep -Fq "$MUTATED" "$TARGET"; then
  echo "Mutation applied successfully."
else
  echo "FAIL: mutation did not apply (sed found no match) — aborting meta-test."
  exit 1
fi

echo
echo "=== Step 3: running gate against MUTATED file (expect non-zero exit) ==="
set +e
"$GATE"
mutated_gate_exit=$?
set -e
echo "Gate exit code on mutated file: $mutated_gate_exit"

if [[ "$mutated_gate_exit" -ne 0 ]]; then
  echo "PASS: gate correctly FAILED on mutated forensic anchor (not a bluff gate)."
else
  echo "FAIL: gate incorrectly PASSED on mutated forensic anchor — this is a BLUFF GATE."
  overall_status=1
fi

echo
echo "=== Step 4: restoring $TARGET and re-running gate (expect zero exit) ==="
restore
trap - EXIT ERR INT
trap restore EXIT ERR INT

set +e
"$GATE"
restored_gate_exit=$?
set -e
echo "Gate exit code on restored file: $restored_gate_exit"

if [[ "$restored_gate_exit" -eq 0 ]]; then
  echo "PASS: gate correctly PASSED after restoring the forensic anchor."
else
  echo "FAIL: gate did not pass after restoring the forensic anchor."
  overall_status=1
fi

echo
echo "=== Step 5: asserting constitution/ submodule tree is pristine ==="
porcelain_output="$(git -C "$REPO_ROOT/constitution" status --porcelain)"
if [[ -z "$porcelain_output" ]]; then
  echo "PASS: constitution/ submodule tree is pristine (git status --porcelain is empty)."
else
  echo "FAIL: constitution/ submodule tree is NOT pristine:"
  echo "$porcelain_output"
  overall_status=1
fi

echo
if [[ "$overall_status" -eq 0 ]]; then
  echo "RESULT: META-TEST PASSED — gate correctly rejects the false positive and tree is pristine."
else
  echo "RESULT: META-TEST FAILED"
fi

exit "$overall_status"
