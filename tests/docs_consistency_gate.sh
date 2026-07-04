#!/usr/bin/env bash
# docs_consistency_gate.sh
#
# Documentation consistency + integrity GATE for the markdown spec corpus at
# docs/research/mvp/output/docs/markdown/*.md (+ any README alongside it).
#
# Hard checks (a FAIL in any of these sets a non-zero exit code):
#   DUP        - large verbatim-duplicated blocks within a single doc
#                (>=40 consecutive non-blank identical lines appearing twice)
#   ANCHOR     - internal `](#slug)` links with no matching heading anchor
#   LATEST     - `image:` / `FROM ` lines using `:latest` inside fenced code
#                blocks (forbidden in prod manifests)
#   EMPTYTEST  - Go test funcs with empty/stub bodies (anti-bluff detection)
#
# Report-only check (never fails the build):
#   DRIFT      - distinct version strings seen per infra component, so
#                version drift across docs is visible to reviewers
#
# Usage:
#   tests/docs_consistency_gate.sh                # scan the full corpus
#   tests/docs_consistency_gate.sh file1.md ...    # scan only the given files
#
# The second form is what scripts/testing/meta_test_docs_gate.sh (the §1.1
# mutation proof) uses to point the gate's check functions at a single
# temp file, without ever touching the real corpus.
set -euo pipefail

# --- resolve repo root & corpus via git ------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"
CORPUS_DIR="$REPO_ROOT/docs/research/mvp/output/docs/markdown"

DUP_WINDOW=40

declare -a FILES=()
if [ "$#" -gt 0 ]; then
  FILES=("$@")
else
  shopt -s nullglob
  FILES=("$CORPUS_DIR"/*.md)
  if [ -f "$CORPUS_DIR/README.md" ]; then
    FILES+=("$CORPUS_DIR/README.md")
  fi
  shopt -u nullglob
fi

if [ "${#FILES[@]}" -eq 0 ]; then
  echo "docs_consistency_gate: no markdown files found to scan (corpus dir: $CORPUS_DIR)" >&2
  exit 1
fi

# ============================================================================
# DUP — verbatim-duplicated blocks within a single doc
# ============================================================================
# Strategy (awk/sort-style hashing without external hashing tools): for each
# file, take only the non-blank lines (so re-flowed blank-line spacing can't
# hide/create false runs), slide a window of DUP_WINDOW consecutive lines
# across them, and use an awk associative array as the "sort/uniq" bucket:
# the first time a window's exact text is seen it is recorded; if the exact
# same DUP_WINDOW-line text is seen again at a position at least DUP_WINDOW
# lines later (i.e. a genuinely separate, non-overlapping occurrence), that
# is a verbatim-duplicated block.
check_dup() {
  local failed=0
  local f out anchor_line repeat_line
  echo "--- DUP check (>= ${DUP_WINDOW} consecutive non-blank identical lines, repeated in the same file) ---"
  for f in "$@"; do
    out=$(awk -v W="$DUP_WINDOW" '
      NF { idx++; content[idx] = $0; origline[idx] = FNR }
      END {
        last_end = 0
        for (i = 1; i <= idx - W + 1; i++) {
          key = ""
          for (k = 0; k < W; k++) key = key content[i + k] SUBSEP
          if (key in seen) {
            j = seen[key]
            if (i - j >= W && j > last_end) {
              print origline[j] "\t" origline[i]
              last_end = i + W - 1
            }
          } else {
            seen[key] = i
          }
        }
      }
    ' "$f")
    if [ -n "$out" ]; then
      failed=1
      while IFS=$'\t' read -r anchor_line repeat_line; do
        echo "FAIL [DUP] $f:$anchor_line: duplicated block of >=${DUP_WINDOW} identical non-blank lines (verbatim repeat starts again at line $repeat_line)"
      done <<< "$out"
    fi
  done
  if [ "$failed" -eq 0 ]; then
    echo "PASS [DUP] no verbatim duplicate blocks of >= ${DUP_WINDOW} lines found in any scanned doc"
    return 0
  fi
  return 1
}

# ============================================================================
# ANCHOR — broken internal TOC/anchor links
# ============================================================================
# Builds the GitHub-style slug for every heading in the file, then extracts
# every `](#...)` link and flags any whose target slug is not in that set.
check_anchor() {
  local failed=0
  local f slug lineno anchor
  echo "--- ANCHOR check (internal ](#slug) links must resolve to a heading) ---"
  for f in "$@"; do
    declare -A slugset=()
    while IFS= read -r slug; do
      [ -n "$slug" ] && slugset["$slug"]=1
    done < <(
      awk 'match($0, /^#{1,6}[ \t]+/) { print substr($0, RLENGTH + 1) }' "$f" \
        | tr '[:upper:]' '[:lower:]' \
        | sed -E 's/[^a-z0-9 _-]//g' \
        | tr ' ' '-'
    )

    while IFS=: read -r lineno anchor; do
      [ -z "$lineno" ] && continue
      if [ -z "${slugset[$anchor]:-}" ]; then
        echo "FAIL [ANCHOR] $f:$lineno: broken anchor '#$anchor' has no matching heading"
        failed=1
      fi
    done < <(
      awk '
        {
          line = $0
          pos = 1
          while (match(substr(line, pos), /\]\(#[^)]+\)/)) {
            s = pos + RSTART - 1
            full = substr(line, s, RLENGTH)
            anchor = substr(full, 4, length(full) - 4)
            print FNR ":" anchor
            pos = s + RLENGTH
          }
        }
      ' "$f"
    )
    unset slugset
  done
  if [ "$failed" -eq 0 ]; then
    echo "PASS [ANCHOR] all internal anchors resolve to a heading"
    return 0
  fi
  return 1
}

# ============================================================================
# LATEST — forbidden `:latest` image tags inside fenced code blocks
# ============================================================================
check_latest() {
  local failed=0
  local f out ln rest
  echo "--- LATEST check (no ':latest' image/FROM tags inside fenced code blocks) ---"
  for f in "$@"; do
    out=$(awk '
      /^```/ { infence = !infence; next }
      infence && ($0 ~ /image:[ \t]*[^ \t]*:latest([^0-9A-Za-z_.-]|$)/ ||
                  $0 ~ /^[ \t]*FROM[ \t]+[^ \t]*:latest([^0-9A-Za-z_.-]|$)/) {
        print FNR ": " $0
      }
    ' "$f")
    if [ -n "$out" ]; then
      failed=1
      while IFS= read -r line; do
        ln="${line%%:*}"
        rest="${line#*: }"
        echo "FAIL [LATEST] $f:$ln: forbidden ':latest' tag in manifest -> ${rest}"
      done <<< "$out"
    fi
  done
  if [ "$failed" -eq 0 ]; then
    echo "PASS [LATEST] no ':latest' image/FROM tags found in fenced code blocks"
    return 0
  fi
  return 1
}

# ============================================================================
# EMPTYTEST — Go test funcs with empty/stub bodies (anti-bluff detection)
# ============================================================================
check_emptytest() {
  local failed=0
  local f out ln
  echo "--- EMPTYTEST check (Go test funcs must not have empty/comment-only bodies) ---"
  for f in "$@"; do
    out=$(awk '
      /^func Test[A-Za-z0-9_]*\(t \*testing\.T\)[ \t]*\{[ \t]*$/ {
        startline = FNR
        capturing = 1
        bodyempty = 1
        next
      }
      capturing {
        trimmed = $0
        gsub(/^[ \t]+|[ \t]+$/, "", trimmed)
        if (trimmed == "}") {
          if (bodyempty) print startline
          capturing = 0
          next
        }
        if (trimmed == "" || trimmed ~ /^\/\//) {
          next
        }
        bodyempty = 0
        capturing = 0
      }
    ' "$f")
    if [ -n "$out" ]; then
      failed=1
      while IFS= read -r ln; do
        echo "FAIL [EMPTYTEST] $f:$ln: test function has an empty/stub body (only whitespace or comments before the closing brace)"
      done <<< "$out"
    fi
  done
  if [ "$failed" -eq 0 ]; then
    echo "PASS [EMPTYTEST] no empty/stub Go test function bodies found"
    return 0
  fi
  return 1
}

# ============================================================================
# DRIFT — report-only version-string inventory (never fails the gate)
# ============================================================================
_drift_report() {
  local label="$1" pattern="$2"
  shift 2
  local versions count
  versions=$(grep -ohE "$pattern" "$@" 2>/dev/null | sort -u || true)
  if [ -z "$versions" ]; then
    echo "INFO [DRIFT] $label: no version strings found in scanned docs"
    return 0
  fi
  count=$(printf '%s\n' "$versions" | grep -c . || true)
  if [ "$count" -le 1 ]; then
    echo "INFO [DRIFT] $label: single consistent version string: $versions"
  else
    echo "WARN [DRIFT] $label: $count distinct version strings found across the corpus (drift):"
    printf '%s\n' "$versions" | sed 's/^/    - /'
  fi
}

check_drift() {
  echo "--- DRIFT check (report-only; distinct version strings per component; never fails the gate) ---"
  _drift_report "postgres/postgresql" '[Pp]ostgres(ql)?[:@ /_-]+v?[0-9]+\.[0-9]+(\.[0-9]+)?' "$@"
  _drift_report "golang/go"           '\b([Gg]olang|[Gg]o)[:@ /_-]*v?1\.[0-9]+(\.[0-9]+)?\b' "$@"
  _drift_report "kafka"               '[Kk]afka[:@ /_-]+v?[0-9]+\.[0-9]+(\.[0-9]+)?' "$@"
  _drift_report "redis"               '[Rr]edis[:@ /_-]+v?[0-9]+\.[0-9]+(\.[0-9]+)?' "$@"
  _drift_report "kubernetes/k8s"      '([Kk]ubernetes|[Kk]8s)[:@ /_-]*v?[0-9]+\.[0-9]+(\.[0-9]+)?' "$@"
  return 0
}

# ============================================================================
# main
# ============================================================================
main() {
  local overall=0

  if ! check_dup "${FILES[@]}"; then overall=1; fi
  echo
  if ! check_anchor "${FILES[@]}"; then overall=1; fi
  echo
  if ! check_latest "${FILES[@]}"; then overall=1; fi
  echo
  if ! check_emptytest "${FILES[@]}"; then overall=1; fi
  echo
  check_drift "${FILES[@]}"
  echo

  echo "==================== SUMMARY ===================="
  echo "Files scanned: ${#FILES[@]}"
  for f in "${FILES[@]}"; do
    echo "  - $f"
  done
  if [ "$overall" -eq 0 ]; then
    echo "RESULT: PASS (DUP, ANCHOR, LATEST, EMPTYTEST all clean; see DRIFT warnings above, if any)"
  else
    echo "RESULT: FAIL (one or more hard checks failed; see FAIL lines above)"
  fi

  exit "$overall"
}

main
