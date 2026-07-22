#!/usr/bin/env bash
# scripts/hooks/capture_operator_request.sh
#
# Claude Code `UserPromptSubmit` hook — the §11.4.210 zero-loss request/prompt
# CAPTURE step. Appends a newest-first entry to the project-local §11.4.208
# operator-request-history ledger (`docs/requests/history.md`) every time the
# operator submits a new prompt.
#
# STATUS AS OF THIS COMMIT: BUILT, NOT YET ACTIVATED. This script is complete
# and self-tested (see capture_operator_request_test.sh) but is NOT wired into
# `.claude/settings.json` by this change — activation is a separate, later
# step owned by the controller (a sibling stream stages the proposed settings
# entry). Until wired, `docs/requests/history.md` keeps saying "NOT YET WIRED"
# under its "Keep-applying mechanism" section, honestly (§11.4.6) — this file
# existing does not itself make that statement false.
#
# ── CONTRACT (Claude Code UserPromptSubmit hook) ─────────────────────────────
#   Verified against https://code.claude.com/docs/en/hooks (live-fetched,
#   2026-07-22 — §11.4.99 latest-source cross-reference):
#   - Receives the event JSON on stdin. Fields present on EVERY UserPromptSubmit
#     event: session_id, transcript_path, cwd, permission_mode, hook_event_name
#     (always "UserPromptSubmit"), prompt (the raw submitted prompt text).
#     `prompt_id` is explicitly ABSENT for this event per the same source.
#     There is NO `model` field in this event's schema — the model that will
#     handle the turn is not yet decided when the hook fires, so "Model +
#     effort" is honestly UNKNOWN below unless a future schema version adds
#     one (defensively probed for, never assumed present).
#   - Exit 0: stdout is parsed for JSON (hookSpecificOutput / systemMessage /
#     etc per the docs); this hook never needs to inject context, so it emits
#     NOTHING on stdout and relies on the "any exit 0 with no meaningful
#     stdout = pass-through" behaviour. All diagnostic output goes to stderr.
#   - Exit 2: BLOCKS the prompt entirely (erases it). NEVER used by this hook
#     — a capture/logging hook must never be able to block or lose an
#     operator's prompt (that would defeat the exact §11.4.210 zero-loss
#     mandate this hook exists to serve).
#   - Any other non-zero exit: shown as a non-blocking "<hook name> hook
#     error" notice but execution continues. Still avoided here — this hook
#     ALWAYS exits 0 (see the fail-open trap below) so a broken capture path
#     never produces operator-visible noise beyond one stderr line.
#
# ── Purpose ──────────────────────────────────────────────────────────────────
#   Mechanically capture every operator prompt into the project-local §11.4.208
#   request-history ledger at the moment it is submitted — the "keep-applying
#   mechanism" §11.4.208(D) describes and §11.4.210 promotes to mandatory —
#   so no request is ever lost between submission and manual/eventual
#   transcription.
#
# ── Usage ────────────────────────────────────────────────────────────────────
#   Wired as a `UserPromptSubmit` hook (see docs/scripts/capture_operator_request.md
#   for the exact `.claude/settings.json` snippet). Manual/test invocation:
#     echo '{"prompt":"hello","cwd":"/mnt/track2/helix_terminator",
#            "session_id":"s1","transcript_path":"/tmp/t.jsonl",
#            "permission_mode":"default","hook_event_name":"UserPromptSubmit"}' \
#       | bash scripts/hooks/capture_operator_request.sh
#
# ── Inputs ───────────────────────────────────────────────────────────────────
#   stdin                          : UserPromptSubmit event JSON (see contract).
#   $HELIX_REQUEST_HISTORY_LEDGER  : optional ledger path override (tests use
#                                    this to NEVER touch the real ledger file).
#   $CLAUDE_CONFIG_DIR             : (ambient env, not read from stdin) used
#                                    ONLY to derive the §11.4.182-style alias
#                                    label from its basename — never logged
#                                    verbatim, never any other env var read.
#
# ── Outputs ──────────────────────────────────────────────────────────────────
#   A new `### CAP-<UTC-timestamp>.` entry prepended to the ledger's
#   "## Entries (newest first)" section (see docs/scripts/capture_operator_request.md
#   for the id-scheme rationale — it deliberately does NOT continue the
#   existing plain `### N.` numbering, so no prior entry is ever renumbered).
#   stderr: one diagnostic line per invocation (capture succeeded / skipped /
#   failed-open). stdout: always empty (pass-through, per the hook contract).
#   Exit code: ALWAYS 0 (fail-open by construction — see the EXIT trap).
#
# ── Side-effects ─────────────────────────────────────────────────────────────
#   Writes the ledger file via write-temp-then-atomic-rename (never a partial
#   write). Reads $CLAUDE_CONFIG_DIR (path only, never a secret). No network,
#   no other file writes, no git operations (§11.4.208(D) — this hook only
#   CAPTURES; committing the ledger is a separate step, out of this hook's
#   scope, per the task that authored it).
#
# ── Dependencies ─────────────────────────────────────────────────────────────
#   bash, awk, date, mktemp, grep, basename. `jq` used when present (preferred
#   JSON field extraction), an awk fallback otherwise (mirrors the extractor
#   in constitution/scripts/hooks/action_prefix_expand.sh so this hook stays
#   dependency-light and portable to a bare-awk host).
#
# ── Cross-references ─────────────────────────────────────────────────────────
#   §11.4.208 (the ledger + its five mandatory fields + honest UNKNOWN
#   discipline), §11.4.210 (promotes this hook from optional to mandatory),
#   §11.4.182 (track/alias derivation convention), §11.4.6 (no-guessing —
#   every underivable field reads `?`/`UNKNOWN`, never invented), §11.4.201
#   (a guard/capture path must fail SAFE and say so, never silently swallow
#   or silently block), §11.4.99 (contract verified against the live official
#   docs, not memory), §11.4.10 (no secret/env value ever printed).
#
# Classification: project-local mechanism instantiating the universal §11.4.208
# / §11.4.210 rules (§11.4.35) — this SCRIPT is project data, not itself a
# constitution-submodule anchor.

set -uo pipefail

# Fail-open, unconditionally, no matter what happens below (including a
# `set -u` unbound-variable abort, which bash treats as fatal regardless of
# `errexit` — verified: an EXIT trap still fires and can force the final
# status). A broken capture hook must NEVER block or degrade the operator's
# prompt (§11.4.201 — a guard that refuses/blocks on a condition that is not
# real is itself a bluff; here the analogous failure mode is "silently eats
# the operator's turn", equally forbidden).
trap 'exit 0' EXIT

cor_warn() {
  # Never print env VALUES (§11.4.10) — only this literal diagnostic text.
  printf 'capture_operator_request: %s\n' "$1" >&2
}

# Extract a top-level JSON string field WITHOUT requiring jq (mirrors
# constitution/scripts/hooks/action_prefix_expand.sh's extractor so this hook
# ports cleanly to a bare-awk host). jq is used when present (more robust).
cor_json_field() {
  local payload="$1" key="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$payload" | jq -r --arg k "$key" '.[$k] // empty' 2>/dev/null
    return 0
  fi
  printf '%s' "$payload" | awk -v key="$key" '
    BEGIN { RS="\0" }
    {
      s = $0
      idx = index(s, "\"" key "\"")
      if (idx == 0) { exit }
      rest = substr(s, idx + length(key) + 2)
      sub(/^[ \t\r\n]*:[ \t\r\n]*/, "", rest)
      if (substr(rest, 1, 1) != "\"") { exit }
      rest = substr(rest, 2)
      out = ""; i = 1; n = length(rest)
      while (i <= n) {
        c = substr(rest, i, 1)
        if (c == "\\") {
          nx = substr(rest, i+1, 1)
          if (nx == "n") out = out "\n"
          else if (nx == "t") out = out "\t"
          else if (nx == "r") out = out "\r"
          else if (nx == "\"") out = out "\""
          else if (nx == "\\") out = out "\\"
          else if (nx == "/") out = out "/"
          else out = out nx
          i += 2; continue
        }
        if (c == "\"") break
        out = out c; i += 1
      }
      printf "%s", out
    }
  '
}

# Derive the §11.4.182-style track label from the hook-reported cwd.
# Deterministic, never guessed: /mnt/track<N>/... -> "T<N>"; anything else
# (including an empty/unreadable cwd) -> the honest "?" (§11.4.6).
cor_derive_track() {
  local cwd="$1"
  if [[ "$cwd" =~ ^/mnt/track([0-9]+)/ ]]; then
    printf 'T%s' "${BASH_REMATCH[1]}"
  else
    printf '?'
  fi
}

# Derive the §11.4.182-style alias label from $CLAUDE_CONFIG_DIR's basename
# (".claude-<alias>" -> "<alias>"). Only the PATH is read, never any secret
# env var, and only the basename component is ever emitted (§11.4.10).
cor_derive_alias() {
  local cfgdir base
  cfgdir="${CLAUDE_CONFIG_DIR:-}"
  if [ -z "$cfgdir" ]; then
    printf '?'
    return 0
  fi
  base="$(basename -- "$cfgdir" 2>/dev/null || printf '')"
  if [[ "$base" =~ ^\.claude-(.+)$ ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
  else
    printf '?'
  fi
}

# Render each (possibly multi-line) line of $1 as an indented markdown
# blockquote line, matching the ledger's existing verbatim-quote convention.
cor_blockquote() {
  local text="$1" line
  # Normalise CRLF -> LF so no stray \r lands in the doc.
  text="${text//$'\r'/}"
  while IFS= read -r line; do
    printf '  > %s\n' "$line"
  done <<<"$text"
}

# cor_render_entry <id> <utc-ts> <local-ts-or-empty> <track> <alias> <model> <prompt>
cor_render_entry() {
  local id="$1" utc="$2" local_ts="$3" track="$4" alias_name="$5" model="$6" prompt="$7"
  local when model_line date_only

  if [ -n "$local_ts" ]; then
    when="${utc} (UTC) = ${local_ts} (Asia/Aqtau, this project's declared default timezone)"
  else
    when="${utc} (UTC); Asia/Aqtau conversion UNAVAILABLE on this host (tzdata missing) — UTC is authoritative"
  fi

  if [ -z "$model" ]; then
    model_line='UNKNOWN (the `UserPromptSubmit` hook JSON schema — verified against the official Claude Code hooks documentation, source verified 2026-07-22 — carries `session_id`/`transcript_path`/`cwd`/`permission_mode`/`hook_event_name`/`prompt` only, no model field at submit time; honestly UNKNOWN rather than guessed, §11.4.6)'
  else
    model_line="$model"
  fi

  date_only="${utc%%T*}"

  printf '### %s. %s — Auto-captured operator prompt\n\n' "$id" "$date_only"
  printf -- '- **Request content (LIVE CAPTURE — verbatim, captured by the §11.4.210 `UserPromptSubmit` auto-capture hook at intake, before any agent processing):**\n'
  cor_blockquote "$prompt"
  printf -- '- **Accepted (when):** %s\n' "$when"
  printf -- '- **Track:** `%s`\n' "$track"
  printf -- '- **Alias:** `%s`\n' "$alias_name"
  printf -- '- **Model + effort:** %s\n' "$model_line"
  printf -- '- **Source:** live `UserPromptSubmit` auto-capture hook (`scripts/hooks/capture_operator_request.sh`) — NOT a reconstruction.\n\n'
}

# Insert $2 (a file path holding the rendered entry block) into ledger $1,
# immediately below the "## Entries (newest first)" anchor line, swallowing
# the single blank line that already follows it (the block supplies its own
# trailing blank separator). Write-temp-then-atomic-rename — never a partial
# write. Returns non-zero (never aborts the caller) if the anchor is absent
# or the ledger is missing, so the caller can fail open with a clear reason.
cor_insert_into_ledger() {
  local ledger="$1" blockfile="$2" anchor='## Entries (newest first)'

  if [ ! -f "$ledger" ]; then
    cor_warn "ledger not found at $ledger — skipping insert (no-op; the operator's prompt itself still reaches the agent normally, only this ledger entry is not recorded — the prompt text is never logged to stderr, by design, §11.4.10)"
    return 1
  fi
  if ! grep -qF "$anchor" -- "$ledger"; then
    cor_warn "anchor line '$anchor' not found in $ledger — ledger structure unrecognised, skipping insert to avoid corrupting it"
    return 1
  fi

  local tmp
  tmp="$(mktemp "${ledger}.capXXXXXX" 2>/dev/null)" || {
    cor_warn "mktemp failed for $ledger — skipping insert"
    return 1
  }

  awk -v anchor="$anchor" -v blockfile="$blockfile" '
    BEGIN { after_anchor = 0; done = 0 }
    {
      if (after_anchor == 1) {
        after_anchor = 0
        if ($0 == "") {
          # Preserve the pre-existing blank line that already separates the
          # anchor from the first entry, THEN inject the block (which itself
          # carries its own trailing blank separator before whatever comes
          # next — no swallowing, no double blank lines).
          print $0
          while ((getline line < blockfile) > 0) print line
          close(blockfile)
          next
        } else {
          # Defensive: no blank line existed after the anchor (malformed
          # source). Synthesize the missing separator before the block.
          print ""
          while ((getline line < blockfile) > 0) print line
          close(blockfile)
          print $0
          next
        }
      }
      print
      if (!done && $0 == anchor) {
        after_anchor = 1
        done = 1
      }
    }
    END {
      if (!done) {
        # Anchor line matched earlier by grep but not by awk (should not
        # happen — defensive only). Do not silently drop the block.
        print ""
        while ((getline line < blockfile) > 0) print line
        close(blockfile)
      }
    }
  ' "$ledger" >"$tmp" 2>/dev/null

  if [ ! -s "$tmp" ]; then
    cor_warn "insert produced an empty/failed result — leaving $ledger untouched"
    rm -f -- "$tmp" 2>/dev/null
    return 1
  fi

  if ! mv -f -- "$tmp" "$ledger" 2>/dev/null; then
    cor_warn "atomic rename into $ledger failed — leaving it untouched"
    rm -f -- "$tmp" 2>/dev/null
    return 1
  fi

  return 0
}

cor_main() {
  local payload prompt cwd model track alias_name
  payload="$(cat 2>/dev/null || true)"
  if [ -z "${payload:-}" ]; then
    cor_warn "empty stdin payload — nothing to capture (no-op)"
    return 0
  fi

  prompt="$(cor_json_field "$payload" prompt)"
  if [ -z "${prompt:-}" ]; then
    cor_warn "malformed JSON or empty/absent .prompt field — nothing to capture (no-op)"
    return 0
  fi

  cwd="$(cor_json_field "$payload" cwd)"
  model="$(cor_json_field "$payload" model)"
  track="$(cor_derive_track "${cwd:-}")"
  alias_name="$(cor_derive_alias)"

  local now_utc now_local entry_id
  now_utc="$(date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null)"
  if [ -z "${now_utc:-}" ]; then
    cor_warn "date(1) failed — cannot timestamp capture, aborting this capture (no-op)"
    return 0
  fi
  now_local="$(TZ='Asia/Aqtau' date '+%Y-%m-%dT%H:%M:%S%:z' 2>/dev/null || true)"
  entry_id="CAP-$(date -u '+%Y%m%dT%H%M%SZ' 2>/dev/null || printf 'UNKNOWN')"

  local ledger blockfile
  ledger="${HELIX_REQUEST_HISTORY_LEDGER:-}"
  if [ -z "$ledger" ]; then
    local here root
    here="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" >/dev/null 2>&1 && pwd)"
    root="$(cd "${here}/../.." >/dev/null 2>&1 && pwd)"
    ledger="${root}/docs/requests/history.md"
  fi

  blockfile="$(mktemp 2>/dev/null)" || {
    cor_warn "mktemp for entry block failed — nothing to capture (no-op)"
    return 0
  }
  cor_render_entry "$entry_id" "$now_utc" "$now_local" "$track" "$alias_name" "$model" "$prompt" >"$blockfile"

  if cor_insert_into_ledger "$ledger" "$blockfile"; then
    cor_warn "captured entry $entry_id (track=$track alias=$alias_name) into $ledger"
  fi
  rm -f -- "$blockfile" 2>/dev/null

  return 0
}

cor_main
