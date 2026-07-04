#!/usr/bin/env bash
# meta_test_docs_gate.sh  (§1.1 mutation proof)
#
# Paired mutation proof for tests/docs_consistency_gate.sh.
#
# Proves the docs consistency gate is not a "bluff gate": takes ONE clean
# markdown doc from the real corpus, copies it to temp files OUTSIDE the
# corpus, injects one instance of each hard-defect class (DUP, ANCHOR,
# LATEST, EMPTYTEST) into the mutated copy, and asserts:
#   1. the gate FAILS on the mutated copy, and each of the four checks
#      fires (their FAIL tag is present in the gate's output), and
#   2. the gate PASSES on an untouched clean copy of the same doc.
#
# The gate is invoked with a single explicit file argument each time
# (tests/docs_consistency_gate.sh <file>), which is the "temp single-file
# invocation" mode the gate script supports specifically so this meta-test
# never has to touch, and never does touch, the real corpus.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
GATE="$REPO_ROOT/tests/docs_consistency_gate.sh"
SOURCE_DOC="$REPO_ROOT/docs/research/mvp/output/docs/markdown/01_core_architecture.md"

if [ ! -x "$GATE" ]; then
  echo "FAIL: gate script not found or not executable at $GATE" >&2
  exit 1
fi
if [ ! -f "$SOURCE_DOC" ]; then
  echo "FAIL: source doc not found at $SOURCE_DOC" >&2
  exit 1
fi

overall_status=0

# --- temp workspace OUTSIDE the corpus, always cleaned up ------------------
TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/docs_gate_meta.XXXXXX")"
cleanup() {
  rm -rf "$TMPDIR"
}
# Intentionally NOT trapping ERR: several steps below deliberately invoke the
# gate and expect it to return non-zero, and an ERR trap would fire mid-script
# on those expected failures and delete the temp workspace out from under the
# remaining steps. EXIT alone is sufficient to guarantee cleanup on any exit
# path (normal completion, explicit `exit`, or an uncaught signal).
trap cleanup EXIT INT TERM

CLEAN_DOC="$TMPDIR/clean_01_core_architecture.md"
MUTATED_DOC="$TMPDIR/mutated_01_core_architecture.md"

cp "$SOURCE_DOC" "$CLEAN_DOC"
cp "$SOURCE_DOC" "$MUTATED_DOC"

echo "=== Step 1: temp copies created (outside corpus) ==="
echo "  clean:   $CLEAN_DOC"
echo "  mutated: $MUTATED_DOC"

# ============================================================================
# Step 2: inject one instance of each hard-defect class into MUTATED_DOC
# ============================================================================
echo
echo "=== Step 2: injecting one instance of each defect class into the mutated copy ==="

# --- DUP: duplicate a genuine >=40 non-blank-line block verbatim -----------
# Lines 1-89 of the source doc (title/header + table of contents) are known
# to contain >=40 non-blank lines AND no fenced-code-block markers, so
# re-appending them verbatim later in the file cannot also shift fence
# parity for the LATEST/EMPTYTEST injections appended after it below.
# Re-appending that exact block later in the file makes it appear twice,
# non-overlapping -> a verbatim DUP hit.
DUP_BLOCK="$(sed -n '1,89p' "$SOURCE_DOC")"
nonblank_in_block="$(printf '%s\n' "$DUP_BLOCK" | grep -c '[^[:space:]]' || true)"
if [ "$nonblank_in_block" -lt 40 ]; then
  echo "FAIL: chosen DUP source block only has $nonblank_in_block non-blank lines (need >=40) — fix the line range in this meta-test" >&2
  exit 1
fi
if printf '%s\n' "$DUP_BLOCK" | grep -q '^```'; then
  echo "FAIL: chosen DUP source block contains a fenced-code-block marker — it would corrupt fence parity for the LATEST/EMPTYTEST injections. Fix the line range in this meta-test." >&2
  exit 1
fi
{
  echo
  echo "<!-- injected DUP defect: verbatim repeat of an earlier >=40-line block -->"
  printf '%s\n' "$DUP_BLOCK"
} >> "$MUTATED_DOC"
echo "  [DUP]       appended a verbatim repeat of a $nonblank_in_block-non-blank-line block from lines 1-89"

# --- ANCHOR: a TOC-style link to a heading that does not exist -------------
{
  echo
  echo "See [a section that does not exist](#no-such-anchor-injected-by-meta-test) for details."
} >> "$MUTATED_DOC"
echo "  [ANCHOR]    appended a link to a non-existent heading anchor"

# --- LATEST: a fenced k8s manifest snippet using the :latest tag -----------
{
  echo
  echo '```yaml'
  echo 'apiVersion: apps/v1'
  echo 'kind: Deployment'
  echo 'spec:'
  echo '  template:'
  echo '    spec:'
  echo '      containers:'
  echo '      - name: injected-defect'
  echo '        image: ghcr.io/example/injected-defect:latest'
  echo '```'
} >> "$MUTATED_DOC"
echo "  [LATEST]    appended a fenced yaml manifest using an 'image: ...:latest' tag"

# --- EMPTYTEST: a Go test func whose body is comment-only ------------------
{
  echo
  echo '```go'
  echo 'func TestInjectedStub(t *testing.T) {'
  echo '    // TODO: implement this test'
  echo '}'
  echo '```'
} >> "$MUTATED_DOC"
echo "  [EMPTYTEST] appended a Go test func with a comment-only (empty) body"

# ============================================================================
# Step 3: run the gate against the MUTATED temp file — expect FAIL
# ============================================================================
echo
echo "=== Step 3: running gate against MUTATED temp file (expect non-zero exit) ==="
set +e
mutated_output="$("$GATE" "$MUTATED_DOC" 2>&1)"
mutated_exit=$?
set -e
echo "$mutated_output"
echo "--- gate exit code on mutated file: $mutated_exit ---"

if [ "$mutated_exit" -ne 0 ]; then
  echo "PASS: gate correctly FAILED on the mutated file (not a bluff gate)."
else
  echo "FAIL: gate incorrectly PASSED on a file with 4 injected defects — this is a BLUFF GATE."
  overall_status=1
fi

for tag in "FAIL \[DUP\]" "FAIL \[ANCHOR\]" "FAIL \[LATEST\]" "FAIL \[EMPTYTEST\]"; do
  label="${tag//\\/}"
  if printf '%s' "$mutated_output" | grep -qE "$tag"; then
    echo "PASS: gate output contains a '$label' finding for the injected defect."
  else
    echo "FAIL: gate output is missing a '$label' finding — that check did not bite on its injected defect."
    overall_status=1
  fi
done

# ============================================================================
# Step 4: run the gate against the CLEAN temp file — expect PASS
# ============================================================================
echo
echo "=== Step 4: running gate against CLEAN temp file (expect zero exit) ==="
set +e
clean_output="$("$GATE" "$CLEAN_DOC" 2>&1)"
clean_exit=$?
set -e
echo "$clean_output"
echo "--- gate exit code on clean file: $clean_exit ---"

if [ "$clean_exit" -eq 0 ]; then
  echo "PASS: gate correctly PASSED on the untouched clean file."
else
  echo "FAIL: gate incorrectly FAILED on an untouched clean file — false positive."
  overall_status=1
fi

# ============================================================================
# Step 5: assert the real corpus was never touched
# ============================================================================
echo
echo "=== Step 5: asserting the real corpus doc was not mutated ==="
if diff -q "$SOURCE_DOC" "$CLEAN_DOC" > /dev/null 2>&1; then
  echo "PASS: temp clean copy is still byte-identical to the real corpus source doc."
else
  echo "FAIL: temp clean copy diverged from the real corpus source doc unexpectedly."
  overall_status=1
fi

echo
if [ "$overall_status" -eq 0 ]; then
  echo "RESULT: META-TEST PASSED — gate bites on all 4 injected defect classes and passes clean; corpus untouched; temp files cleaned up."
else
  echo "RESULT: META-TEST FAILED"
fi

exit "$overall_status"
