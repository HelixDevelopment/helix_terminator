# `capture_operator_request.sh` — §11.4.210 operator-request auto-capture hook

**Revision:** 1
**Last modified:** 2026-07-22T00:00:00Z

## Overview

`scripts/hooks/capture_operator_request.sh` is a Claude Code `UserPromptSubmit`
hook that mechanically appends a newest-first entry to the project-local
[`docs/requests/history.md`](../requests/history.md) ledger every time the
operator submits a new prompt — the "keep-applying mechanism"
`docs/requests/history.md` describes in its own `## Keep-applying mechanism`
section (§11.4.208(D)), promoted from optional follow-up to **mandatory** by
constitution §11.4.210 (zero-loss request/prompt intake).

**Status as of this document:** the script is **built and self-tested** (see
[Testing](#testing) below) but **NOT YET ACTIVATED** — it is not referenced
from `.claude/settings.json` by the change that added it. Wiring it in is a
separate, later step (a sibling stream stages the `.claude/settings.json`
change; this hook script deliberately does not touch that file itself, per
its task's write-path restriction). Until that wiring lands, `history.md`
honestly keeps saying "NOT YET WIRED" — this document does not contradict
that; it describes a built-but-not-yet-activated mechanism.

## Prerequisites

- `bash`, `awk` (GNU awk verified; POSIX-awk fallback path also exercised),
  `date`, `mktemp`, `grep`, `basename` — all standard on any host this
  project already runs on.
- `jq` is used when present (more robust JSON parsing) but is **optional** —
  a pure-awk fallback extracts the same fields when `jq` is absent from
  `PATH` (proven by a dedicated test case, see below).
- The project's `docs/requests/history.md` ledger must exist and contain its
  `## Entries (newest first)` anchor heading exactly as currently formatted
  (the hook fails open, never open-corrupts, when it does not).

## Usage examples

### Manual / ad-hoc invocation (what Claude Code does automatically once wired)

```bash
echo '{
  "session_id": "abc123",
  "transcript_path": "/home/user/.claude/projects/.../abc123.jsonl",
  "cwd": "/mnt/track2/helix_terminator",
  "permission_mode": "default",
  "hook_event_name": "UserPromptSubmit",
  "prompt": "Add a health check endpoint"
}' | bash scripts/hooks/capture_operator_request.sh
```

This appends a `### CAP-<UTC-timestamp>.` entry directly under
`## Entries (newest first)` in `docs/requests/history.md`, above every
existing entry.

### Pointing at a different ledger (used exclusively by the test suite)

```bash
HELIX_REQUEST_HISTORY_LEDGER=/tmp/scratch_ledger.md \
  bash scripts/hooks/capture_operator_request.sh <payload.json
```

The real `docs/requests/history.md` is **never** touched unless
`HELIX_REQUEST_HISTORY_LEDGER` is unset (the production default) or
explicitly points at it.

## The `.claude/settings.json` wiring snippet (for future activation)

The constitution submodule's own installer (`constitution/scripts/install_action_prefix.sh`)
demonstrates the canonical merge shape for a `UserPromptSubmit` hook entry in
this project's settings file. The equivalent entry for this hook is:

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "bash \"$CLAUDE_PROJECT_DIR/scripts/hooks/capture_operator_request.sh\""
          }
        ]
      }
    ]
  }
}
```

Notes for whoever wires this in:

- If `.claude/settings.json` already has a `UserPromptSubmit` array (e.g. the
  constitution's own action-prefix-expansion hook, installed via
  `constitution/scripts/install_action_prefix.sh`), **append** this hook
  entry's `{ "hooks": [...] }` group to the existing array — do **not**
  replace it. Multiple `UserPromptSubmit` hook groups run independently; both
  the action-prefix expander and this capture hook can coexist.
- Use `$CLAUDE_PROJECT_DIR` (Claude Code's own project-root variable) rather
  than a hardcoded absolute path, so the hook entry stays portable across
  checkouts (§11.4.29 / §11.4.177 decoupling discipline).
- This hook **never** emits blocking stdout JSON and **never** exits 2 — it
  is always safe to add alongside any other `UserPromptSubmit` hook.

## Edge cases (all covered by the test suite)

| Input condition | Hook behaviour |
|---|---|
| Valid JSON with a non-empty `.prompt` | Entry captured + prepended, exit 0 |
| Malformed / non-JSON stdin | Honest no-op, exit 0, ledger untouched |
| `.prompt` absent or empty string | Honest no-op, exit 0, ledger untouched |
| Ledger file missing the `## Entries (newest first)` anchor | Honest no-op, exit 0, ledger untouched, reason logged to stderr |
| Ledger path does not exist at all | Honest no-op, exit 0, no file is created |
| `cwd` does not match `/mnt/track<N>/...` | `Track` recorded as the honest `?` (never guessed, §11.4.6) |
| `$CLAUDE_CONFIG_DIR` unset or not `.claude-<alias>`-shaped | `Alias` recorded as the honest `?` |
| Hook JSON has no `model` field (the current, verified schema) | `Model + effort` recorded as `UNKNOWN` with an explanatory note, never guessed |
| `jq` absent from `PATH` | Falls back to a pure-awk JSON field extractor with identical results |
| Any internal script error (unbound variable, failed `mktemp`, etc.) | An unconditional `EXIT` trap forces exit 0 regardless — the hook can never block or corrupt an operator's prompt |

## Internal behaviour

1. Reads the `UserPromptSubmit` event JSON from stdin.
2. Extracts `.prompt` (required) and `.cwd` (used for `Track` derivation) via
   `jq` if present, else a self-contained awk parser.
3. Derives `Track` from `cwd` (`/mnt/track<N>/...` → `T<N>`, else `?`) and
   `Alias` from `$CLAUDE_CONFIG_DIR`'s basename (`.claude-<alias>` → `<alias>`,
   else `?`) — the same §11.4.182 derivation convention `docs/requests/history.md`
   already documents for reconstructed entries.
4. Renders a `### CAP-<UTC-compact-timestamp>.` entry block (see
   [ID-scheme rationale](#id-scheme-rationale-why-cap-and-not-a-continuation-of--n)
   below) with all five §11.4.208 mandatory fields.
5. Inserts the block into the ledger immediately below the
   `## Entries (newest first)` anchor line via a write-temp-then-atomic-rename
   (never a partial write), preserving every byte of every pre-existing entry.
6. Emits one diagnostic line to stderr (never stdout — stdout is reserved for
   the hook's optional JSON context-injection output, which this hook never
   uses) and always exits 0.

## ID-scheme rationale: why `CAP-<timestamp>` and not a continuation of `### N.`

`docs/requests/history.md`'s existing entries use a plain sequential
`### N. <date> — <title>` heading, numbered `1` (newest) through `11`
(oldest) — i.e. **ascending by age**, because the ledger is newest-first and
those eleven entries were all written in one reconstruction pass, in
newest-to-oldest order, at once.

A **live-capture** hook cannot use that scheme without renumbering every
prior entry on every single new prompt: a new, even-newer entry would need to
become `### 1`, pushing the old `### 1` to `### 2`, and so on down the whole
ledger — an O(n) rewrite of already-landed history on every prompt, which
directly violates the append-only-in-spirit intent §11.4.208 describes for
this document and risks a partial/corrupted rewrite on every single capture.

Instead, this hook mints a **new, disjoint, self-sorting identifier space**:

```
### CAP-YYYYMMDDTHHMMSSZ. <date> — Auto-captured operator prompt
```

- `CAP-` never collides with the plain numeric scheme, so the two families
  coexist in the same document without ambiguity.
- The UTC timestamp component is inherently monotonic — entries inserted
  later always sort correctly when read top-to-bottom, and the id itself is
  greppable (`grep '^### CAP-'`) without needing a separate counter file or
  state store (contrast the `ATM-NNN` allocator's need for
  `.atm_ticket_state.json` — deliberately not reused here since these are
  informational capture entries, not tracked workable items).
- No existing entry, numbered or `CAP-`-prefixed, is ever renumbered,
  renamed, or rewritten by a later capture — every insertion is a pure
  prepend immediately below the section anchor.

Future controller/operator process may, at its own discretion, later
transcribe/consolidate some `CAP-*` entries into the numbered narrative style
(as happened for the pre-mandate reconstruction) — that is an editorial
decision outside this hook's scope; the hook's own contract is only ever to
append, never to touch.

## Testing

```bash
bash scripts/hooks/capture_operator_request_test.sh
```

27 cases, all self-contained and hermetic:

- **golden-TRUE** (8 cases): a valid, multi-line prompt is captured
  verbatim, prepended above the pre-existing entry, with correctly-derived
  `Track`/`Alias`/honest-`UNKNOWN` `Model + effort`, and the pre-existing
  entry survives byte-identical.
- **golden-FALSE / carrier** (11 cases): malformed JSON, empty prompt,
  missing prompt field, a ledger missing its anchor, and a nonexistent
  ledger path — every case is an honest no-op (exit 0, ledger untouched, no
  file ever created out of nothing).
- **control needle** (5 cases, §11.4.201 discipline): the suite copies the
  **real, checked-in** `docs/requests/history.md` (never the real file
  itself — always a temp copy) and proves the insertion mechanism genuinely
  matches its actual structure — not merely a hand-tuned synthetic fixture
  that could quietly diverge from the real document's format. Every
  pre-existing real entry is proven byte-identical after insertion.
- **awk-fallback exercise** (3 cases): `jq` is hidden from `PATH` (a minimal
  symlink-only `PATH` is constructed) to force the pure-awk extractor branch
  and prove it — not merely the `jq` branch — genuinely parses the payload
  correctly.

Real output at authoring time: **27/27 PASS**, exit 0. The real
`docs/requests/history.md` is confirmed untouched by the suite (`git status`
shows no diff) after every run.

## Related scripts

- `docs/requests/history.md` — the ledger this hook writes to (§11.4.208).
- `constitution/scripts/hooks/action_prefix_expand.sh` — the sibling
  `UserPromptSubmit` hook this project already inherits (§11.4.140); its
  documented contract-verification methodology against the live official
  Claude Code hooks docs is the source this hook's own contract section was
  cross-checked against (§11.4.99).
- `constitution/scripts/install_action_prefix.sh` — demonstrates the
  `.claude/settings.json` `UserPromptSubmit` array merge shape reused above.

## Last verified date

2026-07-22 — contract verified live against
`https://code.claude.com/docs/en/hooks` (§11.4.99), script and test suite run
on this date with the results recorded under [Testing](#testing).
