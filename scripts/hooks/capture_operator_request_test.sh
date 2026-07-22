#!/usr/bin/env bash
# scripts/hooks/capture_operator_request_test.sh
#
# Hermetic §11.4.107(10) self-validation suite for
# scripts/hooks/capture_operator_request.sh (the §11.4.210 UserPromptSubmit
# auto-capture hook).
#
# ── Purpose ──────────────────────────────────────────────────────────────────
#   Prove the hook is genuinely load-bearing, not a bluff gate: golden-TRUE
#   (valid prompt correctly captured + prepended) + golden-FALSE/carrier
#   (malformed/empty/missing input, missing anchor, missing ledger — all
#   honest no-ops, never a crash, never exit 2) + a CONTROL NEEDLE proving the
#   insertion-point regex genuinely matches the REAL `docs/requests/history.md`
#   structure (not merely a hand-tuned synthetic fixture) + an explicit
#   exercise of the awk JSON-field-extraction fallback path (jq hidden from
#   PATH) so both extraction code paths are proven, not just present.
#
# ── Usage ────────────────────────────────────────────────────────────────────
#   bash scripts/hooks/capture_operator_request_test.sh
#   Exit 0 = every case passed. Exit 1 = one or more cases failed.
#
# ── Inputs ───────────────────────────────────────────────────────────────────
#   None from the environment. Reads (never writes) the real
#   `docs/requests/history.md` for the control-needle case — always operates
#   on a temp copy, per §11.4.6/§11.4.201: this suite NEVER mutates the real
#   ledger.
#
# ── Outputs ──────────────────────────────────────────────────────────────────
#   stdout: PASS/FAIL per case + a final summary. Exit code per Usage above.
#
# ── Side-effects ─────────────────────────────────────────────────────────────
#   Creates + removes a private temp working directory (trap-cleaned on every
#   exit path, §11.4.14). Never touches the real ledger file.
#
# ── Dependencies ─────────────────────────────────────────────────────────────
#   bash, awk, date, mktemp, grep, diff, wc, sed, ln (for the PATH-fallback
#   case), optionally jq (one case explicitly hides it to exercise the awk
#   fallback; the suite passes with or without jq installed on the host).
#
# ── Cross-references ─────────────────────────────────────────────────────────
#   §11.4.107(10) self-validated analyzer discipline, §11.4.201 control-needle
#   discipline, §11.4.210 (what this hook implements), §11.4.6 (never mutate
#   the real ledger in a test).
#
# Classification: project-local test for a project-local mechanism (§11.4.35).

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" >/dev/null 2>&1 && pwd)"
HOOK="$HERE/capture_operator_request.sh"
FIXTURE="$HERE/testdata/sample_ledger.md"
REPO_ROOT="$(cd "$HERE/../.." >/dev/null 2>&1 && pwd)"
REAL_LEDGER="$REPO_ROOT/docs/requests/history.md"

PASS=0
FAIL=0

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf -- "$WORKDIR" 2>/dev/null || true; }
trap cleanup EXIT

pass() { printf '  PASS  %s\n' "$1"; PASS=$((PASS + 1)); }
fail() { printf '  FAIL  %s -- %s\n' "$1" "$2"; FAIL=$((FAIL + 1)); }

# Snapshot the REAL ledger's content BEFORE any case in this suite runs (this
# is the "before" half of the §11.4.201/§11.4.107(10) real-ledger-immutability
# assertion used by the control-needle section below — captured this early so
# it also covers every earlier case, not just the control-needle block).
REAL_LEDGER_SNAPSHOT="$WORKDIR/.real_ledger_snapshot_at_suite_start.md"
if [ -f "$REAL_LEDGER" ]; then
  cp -- "$REAL_LEDGER" "$REAL_LEDGER_SNAPSHOT"
fi

# run_hook <ledger_path> <payload_json> [extra_env...]
# Runs the hook with HELIX_REQUEST_HISTORY_LEDGER pinned at $1 and the JSON
# payload $2 on stdin. Captures exit code into $LAST_EXIT and stderr into
# $LAST_STDERR (a file path).
run_hook() {
  local ledger="$1" payload="$2"
  shift 2
  local errfile
  errfile="$(mktemp -p "$WORKDIR")"
  # shellcheck disable=SC2086
  printf '%s' "$payload" | env "$@" HELIX_REQUEST_HISTORY_LEDGER="$ledger" bash "$HOOK" >/dev/null 2>"$errfile"
  LAST_EXIT=$?
  LAST_STDERR="$errfile"
}

entry_count() {
  # Count "### " headings under the Entries section of $1.
  grep -c '^### ' -- "$1" 2>/dev/null || printf '0'
}

echo "§11.4.210 capture_operator_request.sh hermetic test suite"
echo "hook:    $HOOK"
echo "fixture: $FIXTURE"
echo

# ─── golden-TRUE: valid prompt gets captured + prepended ────────────────────
case1_ledger="$WORKDIR/case1_ledger.md"
cp -- "$FIXTURE" "$case1_ledger"
before_count="$(entry_count "$case1_ledger")"
payload1='{"session_id":"sess-1","transcript_path":"/tmp/t1.jsonl","cwd":"/mnt/track2/helix_terminator","permission_mode":"default","hook_event_name":"UserPromptSubmit","prompt":"Please build the capture hook.\nSecond line of the prompt."}'
run_hook "$case1_ledger" "$payload1" CLAUDE_CONFIG_DIR=/nonexistent/.claude-hooktest
after_count="$(entry_count "$case1_ledger")"

if [ "$LAST_EXIT" -eq 0 ]; then pass "golden-TRUE: hook exits 0 on valid prompt"; else fail "golden-TRUE: hook exits 0 on valid prompt" "got exit $LAST_EXIT"; fi
if [ "$after_count" -eq $((before_count + 1)) ]; then pass "golden-TRUE: entry count increased by exactly 1"; else fail "golden-TRUE: entry count increased by exactly 1" "before=$before_count after=$after_count"; fi

new_heading_line="$(grep -n '^### CAP-' -- "$case1_ledger" | head -1 | cut -d: -f1)"
old_heading_line="$(grep -n '^### 1\. 2026-01-01' -- "$case1_ledger" | head -1 | cut -d: -f1)"
if [ -n "$new_heading_line" ] && [ -n "$old_heading_line" ] && [ "$new_heading_line" -lt "$old_heading_line" ]; then
  pass "golden-TRUE: new CAP-* entry is ABOVE the pre-existing entry (newest-first preserved)"
else
  fail "golden-TRUE: new CAP-* entry is ABOVE the pre-existing entry (newest-first preserved)" "new_line=$new_heading_line old_line=$old_heading_line"
fi

if grep -q '^  > Please build the capture hook\.$' -- "$case1_ledger" && grep -q '^  > Second line of the prompt\.$' -- "$case1_ledger"; then
  pass "golden-TRUE: multi-line prompt captured verbatim as a blockquote"
else
  fail "golden-TRUE: multi-line prompt captured verbatim as a blockquote" "blockquote lines not found"
fi

if grep -q '^- \*\*Track:\*\* `T2`$' -- "$case1_ledger"; then
  pass "golden-TRUE: Track derived as T2 from cwd /mnt/track2/..."
else
  fail "golden-TRUE: Track derived as T2 from cwd /mnt/track2/..." "Track line not T2"
fi

if grep -q '^- \*\*Alias:\*\* `hooktest`$' -- "$case1_ledger"; then
  pass "golden-TRUE: Alias derived as hooktest from CLAUDE_CONFIG_DIR=.../.claude-hooktest"
else
  fail "golden-TRUE: Alias derived as hooktest from CLAUDE_CONFIG_DIR=.../.claude-hooktest" "Alias line not hooktest"
fi

if grep -q '^- \*\*Model + effort:\*\* UNKNOWN' -- "$case1_ledger"; then
  pass "golden-TRUE: Model + effort honestly UNKNOWN (schema carries no model field)"
else
  fail "golden-TRUE: Model + effort honestly UNKNOWN (schema carries no model field)" "not UNKNOWN"
fi

# The pre-existing entry's own text MUST survive byte-identical.
if diff -q <(sed -n '/^### 1\. 2026-01-01/,$p' "$case1_ledger") <(sed -n '/^### 1\. 2026-01-01/,$p' "$FIXTURE") >/dev/null 2>&1; then
  pass "golden-TRUE: pre-existing entry + trailing sections left byte-identical"
else
  fail "golden-TRUE: pre-existing entry + trailing sections left byte-identical" "tail of file diverged"
fi

# ─── golden-FALSE / carrier: malformed JSON -> honest no-op ─────────────────
case2_ledger="$WORKDIR/case2_ledger.md"
cp -- "$FIXTURE" "$case2_ledger"
run_hook "$case2_ledger" 'this is not json at all {{{'
if [ "$LAST_EXIT" -eq 0 ]; then pass "carrier: malformed JSON exits 0 (never blocks the prompt)"; else fail "carrier: malformed JSON exits 0 (never blocks the prompt)" "got exit $LAST_EXIT"; fi
if diff -q "$case2_ledger" "$FIXTURE" >/dev/null 2>&1; then
  pass "carrier: malformed JSON leaves ledger byte-identical (no phantom entry)"
else
  fail "carrier: malformed JSON leaves ledger byte-identical (no phantom entry)" "ledger was modified"
fi

# ─── golden-FALSE / carrier: empty prompt -> honest no-op ───────────────────
case3_ledger="$WORKDIR/case3_ledger.md"
cp -- "$FIXTURE" "$case3_ledger"
run_hook "$case3_ledger" '{"prompt":"","cwd":"/mnt/track1/x","session_id":"s","transcript_path":"/tmp/t","permission_mode":"default","hook_event_name":"UserPromptSubmit"}'
if [ "$LAST_EXIT" -eq 0 ]; then pass "carrier: empty .prompt exits 0"; else fail "carrier: empty .prompt exits 0" "got exit $LAST_EXIT"; fi
if diff -q "$case3_ledger" "$FIXTURE" >/dev/null 2>&1; then
  pass "carrier: empty .prompt leaves ledger byte-identical"
else
  fail "carrier: empty .prompt leaves ledger byte-identical" "ledger was modified"
fi

# ─── golden-FALSE / carrier: prompt field entirely absent -> honest no-op ───
case4_ledger="$WORKDIR/case4_ledger.md"
cp -- "$FIXTURE" "$case4_ledger"
run_hook "$case4_ledger" '{"session_id":"s","transcript_path":"/tmp/t","cwd":"/mnt/track1/x","permission_mode":"default","hook_event_name":"UserPromptSubmit"}'
if [ "$LAST_EXIT" -eq 0 ]; then pass "carrier: absent .prompt field exits 0"; else fail "carrier: absent .prompt field exits 0" "got exit $LAST_EXIT"; fi
if diff -q "$case4_ledger" "$FIXTURE" >/dev/null 2>&1; then
  pass "carrier: absent .prompt field leaves ledger byte-identical"
else
  fail "carrier: absent .prompt field leaves ledger byte-identical" "ledger was modified"
fi

# ─── golden-FALSE / carrier: ledger has no recognisable anchor -> honest no-op
case5_ledger="$WORKDIR/case5_no_anchor.md"
printf '# Not a real ledger\n\nNo Entries section here at all.\n' >"$case5_ledger"
case5_before="$(cat "$case5_ledger")"
run_hook "$case5_ledger" '{"prompt":"hello","cwd":"/mnt/track1/x","session_id":"s","transcript_path":"/tmp/t","permission_mode":"default","hook_event_name":"UserPromptSubmit"}'
if [ "$LAST_EXIT" -eq 0 ]; then pass "carrier: ledger missing the anchor line exits 0"; else fail "carrier: ledger missing the anchor line exits 0" "got exit $LAST_EXIT"; fi
if [ "$(cat "$case5_ledger")" = "$case5_before" ]; then
  pass "carrier: ledger missing the anchor line is left untouched"
else
  fail "carrier: ledger missing the anchor line is left untouched" "content changed"
fi
if grep -qi 'anchor line' -- "$LAST_STDERR"; then
  pass "carrier: missing-anchor case reports its RESOLVED reason on stderr (§11.4.201)"
else
  fail "carrier: missing-anchor case reports its RESOLVED reason on stderr (§11.4.201)" "no diagnostic found"
fi

# ─── golden-FALSE / carrier: ledger path does not exist -> honest no-op ─────
case6_ledger="$WORKDIR/does_not_exist/history.md"
run_hook "$case6_ledger" '{"prompt":"hello","cwd":"/mnt/track1/x","session_id":"s","transcript_path":"/tmp/t","permission_mode":"default","hook_event_name":"UserPromptSubmit"}'
if [ "$LAST_EXIT" -eq 0 ]; then pass "carrier: nonexistent ledger path exits 0"; else fail "carrier: nonexistent ledger path exits 0" "got exit $LAST_EXIT"; fi
if [ ! -e "$case6_ledger" ]; then
  pass "carrier: nonexistent ledger path is NOT created (no accidental file materialised)"
else
  fail "carrier: nonexistent ledger path is NOT created (no accidental file materialised)" "file was created"
fi

# ─── CONTROL NEEDLE: the insertion regex genuinely matches the REAL file ────
# §11.4.201(7): a null/no-op on a synthetic fixture proves nothing about the
# REAL file's structure. Run against a COPY of the actual, checked-in
# docs/requests/history.md and prove insertion succeeds there too, and that
# every byte of the original from its first pre-existing entry onward is
# preserved untouched. NEVER touches the real ledger file itself.
if [ -f "$REAL_LEDGER" ]; then
  needle_ledger="$WORKDIR/real_ledger_copy.md"
  cp -- "$REAL_LEDGER" "$needle_ledger"
  real_before_count="$(entry_count "$needle_ledger")"
  first_real_heading="$(grep -m1 '^### ' -- "$needle_ledger")"
  run_hook "$needle_ledger" '{"session_id":"needle","transcript_path":"/tmp/needle.jsonl","cwd":"/mnt/track3/helix_terminator","permission_mode":"default","hook_event_name":"UserPromptSubmit","prompt":"CONTROL NEEDLE: this exact string must appear verbatim in the real ledger copy after insertion."}'
  real_after_count="$(entry_count "$needle_ledger")"

  if [ "$LAST_EXIT" -eq 0 ]; then
    pass "control needle: hook exits 0 against a copy of the REAL history.md"
  else
    fail "control needle: hook exits 0 against a copy of the REAL history.md" "got exit $LAST_EXIT"
  fi
  if [ "$real_after_count" -eq $((real_before_count + 1)) ]; then
    pass "control needle: entry count in the REAL ledger's structure increased by exactly 1"
  else
    fail "control needle: entry count in the REAL ledger's structure increased by exactly 1" "before=$real_before_count after=$real_after_count"
  fi
  if grep -qF 'CONTROL NEEDLE: this exact string must appear verbatim in the real ledger copy after insertion.' -- "$needle_ledger"; then
    pass "control needle: the needle prompt text is present verbatim in the real ledger copy (proves the anchor/insert path is not blind on the real file)"
  else
    fail "control needle: the needle prompt text is present verbatim in the real ledger copy" "needle text not found"
  fi
  # Everything from the first PRE-EXISTING heading onward must be byte-identical
  # to the original real ledger (proves no existing entry was mangled/renumbered).
  if diff -q \
      <(awk -v h="$first_real_heading" 'BEGIN{f=0} $0==h{f=1} f{print}' "$needle_ledger") \
      <(awk -v h="$first_real_heading" 'BEGIN{f=0} $0==h{f=1} f{print}' "$REAL_LEDGER") \
      >/dev/null 2>&1; then
    pass "control needle: every pre-existing real entry survives byte-identical (no renumbering, no mutation)"
  else
    fail "control needle: every pre-existing real entry survives byte-identical (no renumbering, no mutation)" "tail diverged from the real file"
  fi
  # The real, tracked ledger file itself must be untouched by this whole run.
  # §11.4.201/§11.4.107(10) — this was previously an unconditional `pass` that
  # asserted nothing (IMPORTANT-2, R-HOOKSBATCH review); it is now a genuine
  # two-part check, capable of FAILing: (1) the real file's live content is
  # byte-diffed against the snapshot taken at suite start, BEFORE any case
  # ran; (2) `git status --porcelain` on the real file's path independently
  # confirms git itself sees no change. Either signal alone is sufficient to
  # FAIL this case. Proven load-bearing during authoring: temporarily pointed
  # `HELIX_REQUEST_HISTORY_LEDGER` at the real `$REAL_LEDGER` for one hook
  # invocation instead of a temp copy — this assertion FAILed on both signals
  # as expected, then the change was reverted and the ledger restored from
  # git (`git checkout -- docs/requests/history.md`) before landing this file.
  real_ledger_mutated=0
  real_ledger_mutated_reason=""
  if [ ! -f "$REAL_LEDGER_SNAPSHOT" ]; then
    real_ledger_mutated=1
    real_ledger_mutated_reason="no pre-suite snapshot available (REAL_LEDGER missing at suite start)"
  else
    if ! diff -q "$REAL_LEDGER" "$REAL_LEDGER_SNAPSHOT" >/dev/null 2>&1; then
      real_ledger_mutated=1
      real_ledger_mutated_reason="content diff vs pre-suite snapshot is non-empty"
    fi
    if command -v git >/dev/null 2>&1 && git -C "$REPO_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
      git_porcelain="$(git -C "$REPO_ROOT" status --porcelain -- docs/requests/history.md 2>/dev/null || true)"
      if [ -n "$git_porcelain" ]; then
        real_ledger_mutated=1
        real_ledger_mutated_reason="${real_ledger_mutated_reason:+$real_ledger_mutated_reason; }git status --porcelain reports: $git_porcelain"
      fi
    fi
  fi
  if [ "$real_ledger_mutated" -eq 0 ]; then
    pass "control needle: the real docs/requests/history.md was only ever COPIED, never opened for writing by this suite (content diff + git status both clean)"
  else
    fail "control needle: the real docs/requests/history.md was only ever COPIED, never opened for writing by this suite (content diff + git status both clean)" "$real_ledger_mutated_reason"
  fi
else
  fail "control needle: real ledger present at docs/requests/history.md" "file not found — cannot run control needle"
fi

# ─── awk-fallback exercise: force jq absent, prove the pure-awk path works ──
# Build a minimal PATH containing everything the hook needs EXCEPT jq, so
# `command -v jq` genuinely fails and cor_json_field falls through to its
# awk branch — proving that branch is load-bearing, not merely present.
fallback_bin="$WORKDIR/fallback_bin"
mkdir -p "$fallback_bin"
fallback_ok=1
for tool in bash awk date mktemp grep basename cat mv rm sed dirname cut env printf; do
  real_path="$(command -v "$tool" 2>/dev/null || true)"
  if [ -n "$real_path" ]; then
    ln -sf -- "$real_path" "$fallback_bin/$tool" 2>/dev/null || fallback_ok=0
  else
    fallback_ok=0
  fi
done

if [ "$fallback_ok" -eq 1 ]; then
  case7_ledger="$WORKDIR/case7_ledger.md"
  cp -- "$FIXTURE" "$case7_ledger"
  errfile7="$(mktemp -p "$WORKDIR")"
  printf '%s' '{"session_id":"s","transcript_path":"/tmp/t","cwd":"/mnt/track4/helix_terminator","permission_mode":"default","hook_event_name":"UserPromptSubmit","prompt":"awk-fallback path exercise"}' \
    | env -i PATH="$fallback_bin" CLAUDE_CONFIG_DIR=/nonexistent/.claude-fallbacktest HELIX_REQUEST_HISTORY_LEDGER="$case7_ledger" HOME="$HOME" bash "$HOOK" >/dev/null 2>"$errfile7"
  fb_exit=$?

  if [ "$fb_exit" -eq 0 ]; then
    pass "awk-fallback: hook exits 0 with jq absent from PATH"
  else
    fail "awk-fallback: hook exits 0 with jq absent from PATH" "got exit $fb_exit ($(cat "$errfile7" 2>/dev/null))"
  fi
  if grep -q '^  > awk-fallback path exercise$' -- "$case7_ledger" 2>/dev/null; then
    pass "awk-fallback: prompt correctly extracted + captured via the pure-awk JSON parser"
  else
    fail "awk-fallback: prompt correctly extracted + captured via the pure-awk JSON parser" "captured entry not found or prompt wrong"
  fi
  if grep -q '^- \*\*Track:\*\* `T4`$' -- "$case7_ledger" 2>/dev/null; then
    pass "awk-fallback: cwd field also correctly extracted via the awk path (Track=T4)"
  else
    fail "awk-fallback: cwd field also correctly extracted via the awk path (Track=T4)" "Track not T4"
  fi
else
  echo "  SKIP  awk-fallback exercise (could not build a minimal PATH on this host)"
fi

echo
echo "  total: PASS=$PASS FAIL=$FAIL"
if [ "$FAIL" -gt 0 ]; then
  echo "  RESULT: FAIL"
  exit 1
fi
echo "  RESULT: PASS (all $PASS cases)"
exit 0
