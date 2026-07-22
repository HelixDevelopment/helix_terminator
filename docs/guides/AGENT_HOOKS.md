# Agent Hooks — staged constitution hook wiring

**Revision:** 1
**Last modified:** 2026-07-22T00:00:00Z
**Authority:** constitution §11.4.109 (Mandatory Anti-Forgetting Enforcement:
PreToolUse Guard Hook + Subagent Constitutional Preamble + Orchestrator
Pre-Action Checklist) · constitution §11.4.164 (Universal Constitution
Auto-Propagation & Hook System)
**Status:** STAGED, NOT ACTIVE — see [Activation](#activation) before doing
anything with `.claude/settings.proposed.json`.

> **Disclosure: `.claude/settings.proposed.json` is git-ignored.** The
> `.claude/` directory is excluded wholesale by `.gitignore` (project-wide
> `.claude/` ignore rule, §11.4.30), so `.claude/settings.proposed.json`
> is **checkout-local** — it is absent from every commit, absent from
> `git status`, and will be **absent on a fresh clone or a different
> checkout/track**. It is not lost, however: the **authoritative, tracked
> copy of the proposed wiring is the JSON block embedded in this guide**
> (see [Activation](#activation) below) — this document is self-sufficient
> for activation even when the gitignored file does not exist locally. If
> the local `.claude/settings.proposed.json` file is present, treat it as a
> convenience copy of the JSON block below, not a second source of truth.

## Overview

This project inherits two constitution-shipped mechanical-enforcement hooks.
Both are consumed **by reference** from `constitution/scripts/hooks/` and
`constitution/scripts/` — never copied into this project's own tree
(§11.4.109 / §11.4.177) — so a constitution pull that improves the hook
improves it here too, automatically.

| Hook | §-anchor | Seam | Wired? |
|---|---|---|---|
| `constitution/scripts/hooks/guard-forbidden-commands.sh` | §11.4.109 | Claude Code `PreToolUse` / `Bash` matcher, via `.claude/settings.json` | **Staged** in `.claude/settings.proposed.json` — not active |
| `constitution/scripts/post_update_hook.sh` | §11.4.164 | Git `post-merge` hook (`constitution/scripts/hooks/post-merge` → `.git/hooks/post-merge`) | **Not installed** in this checkout (see below) |

Three further §11.4.109-class guard hooks exist in the constitution submodule
and are **deliberately NOT staged** here — see
[Deferred guards](#deferred-guards-not-wired).

## `guard-forbidden-commands.sh` (§11.4.109)

### What it does

A Claude Code `PreToolUse` hook that inspects every `Bash` tool call's command
string and mechanically **blocks** (exit 2, refusal text fed back to Claude)
four forbidden classes, independent of any agent's memory of the rule:

1. **Emulator / device gate** (§6.X/§6.V/§6.AG) — raw host-direct
   `emulator -avd`, `adb install`, `am instrument` (gate runs must go through
   the Containers submodule instead).
2. **Force-push / verification-bypass** (§6.T.3) — `git push --force` /
   `-f` / `--force-with-lease` / `+<refspec>`, `--no-verify`, `--no-gpg-sign`.
   Overridable with a documented `# guardrails:allow <reason>` marker (WARN
   instead of BLOCK).
3. **sudo / su** (§6.U) — any invocation of `sudo` or a standalone `su`.
4. **Host-power** (Host Machine Stability Directive) — `systemctl
   suspend/hibernate/poweroff/reboot/...`, `loginctl` equivalents, `pm-suspend`,
   bare `shutdown`. **NOT overridable** by the escape-hatch marker under any
   circumstance.

Every other tool call (`Read`, `Edit`, `Agent`/`Task` dispatch, MCP tool
calls) and every non-matching `Bash` command passes through untouched at
exit 0. The script is pure bash + optional `jq` (falls back to a small
embedded awk JSON-field extractor when `jq` is absent) — `bash -n` clean,
340 lines, no project-specific literals.

### Verification evidence (this session, this checkout)

`bash -n constitution/scripts/hooks/guard-forbidden-commands.sh` → parses
clean. The script was then exercised standalone with the exact PreToolUse
JSON-on-stdin contract the Claude Code runtime uses
(`{"tool_name":"Bash","tool_input":{"command":"..."}}}`), reproducing golden-TRUE
(forbidden) and golden-FALSE (allowed) cases per §11.4.201:

| # | Input | Expected | Observed | Result |
|---|---|---|---|---|
| 1 | `git push --force origin main` | BLOCK, exit 2 | `guardrails: BLOCKED — §6.T.3 force-push`, exit 2 | PASS |
| 2 | `sudo systemctl suspend` | BLOCK, exit 2 | `guardrails: BLOCKED — §6.U no-sudo`, exit 2 | PASS |
| 3 | `ls -la` | ALLOW, exit 0 | (no output), exit 0 | PASS |
| 4 | `go test ./...` | ALLOW, exit 0 | (no output), exit 0 | PASS |
| 5 | `systemctl suspend` (no sudo) | BLOCK, exit 2, no-override | `guardrails: BLOCKED — Host Stability (systemctl)`, exit 2 | PASS |
| 6 | non-`Bash` tool call (`Read`) | pass through, exit 0 | (no output), exit 0 | PASS |
| 7 | `git push --force origin main # guardrails:allow operator-approved-in-chat` | WARN not BLOCK, exit 0 | `guardrails: WARNING — §6.T.3 force-push: ...` + `guardrails: allowed by documented exception: operator-approved-in-chat`, exit 0 | PASS |
| 8 | `git fetch --all && tail -f qa-results/push_failures/x.log` (the documented 2026-07-11 false-positive-fix regression case — a `-f` flag on an unrelated chained command) | ALLOW, exit 0 | (no output), exit 0 | PASS |

8/8 golden-TRUE + golden-FALSE + regression cases matched the script's
documented contract exactly. No fake/blank output; every row above is a real
captured stdout/stderr/exit-code transcript from this session.

### Activation

**Staged, not active.** The canonical, tracked copy of the proposed wiring is
the JSON block below (see the disclosure at the top of this guide — the
checkout-local `.claude/settings.proposed.json` file, when present, carries
the same `_comment`-annotated content but is git-ignored and not guaranteed
to exist in any given checkout). To activate:

1. Confirm no live multi-track/multi-agent session window is in flight
   (activating hooks mid-session risks disrupting live agent streams — the
   reason this dispatch explicitly forbade touching `.claude/settings.json`
   / `.claude/settings.local.json` directly).
2. Merge the `hooks` block below into `.claude/settings.json` (create it if
   absent — this checkout currently has no `.claude/settings.json`, only
   `.claude/settings.local.json`, which carries unrelated MCP-server config
   and must not be clobbered). If `.claude/settings.proposed.json` exists
   locally, its `hooks` block is equivalent and may be used directly.
3. Strip the `_comment` key (informational only, not part of the Claude Code
   hooks schema) if it is present in whichever copy was used.
4. Re-run the 8-case table above against the merged file's effective
   behaviour (start a fresh Claude Code session so the runtime reloads
   `settings.json`) before treating the hook as live.

The proposed `hooks.PreToolUse` entry:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash \"$CLAUDE_PROJECT_DIR/constitution/scripts/hooks/guard-forbidden-commands.sh\""
          }
        ]
      }
    ]
  }
}
```

## `post_update_hook.sh` (§11.4.164)

### What it does

After a `git pull` / `git submodule update` brings new constitution content
into this project, `constitution/scripts/post_update_hook.sh` detects which
files changed (skills, MCP configs, `scripts/hooks/*`, action-directive
registry/plugins) and installs/registers/merges them into this project
automatically — the mechanism that makes "a new constitutional directive is
live out of the box" true rather than aspirational.

`bash -n constitution/scripts/post_update_hook.sh` → parses clean (this
session).

### Invocation point — NOT a Claude Code settings.json seam

Unlike `guard-forbidden-commands.sh`, this hook has **no** Claude Code
`PreToolUse`/`PostToolUse` hook point of its own — it fires on a **git**
event, not a Claude Code tool call. Its documented, correct invocation point
is the git `post-merge` hook:

- Source: `constitution/scripts/hooks/post-merge` (inherited by reference,
  §11.4.109/§11.4.177 pattern).
- Installed location: `.git/hooks/post-merge` in this project's main
  checkout.
- Trigger: every `git merge` / `git pull` completion; the hook diffs the
  constitution submodule's HEAD before/after and, ONLY if it actually
  changed, delegates to `post_update_hook.sh`. Silent no-op otherwise.

### Current status in this checkout — honest gap

`ls .git/hooks/` in this checkout shows only `pre-commit` installed;
**`post-merge` is NOT currently installed.** This means the §11.4.164
auto-propagation seam is present and working as a script
(`constitution/scripts/post_update_hook.sh` parses clean and is documented
as idempotent/safe to re-run), but it does **not yet fire automatically** on
a constitution pull in this checkout — a constitution update currently
requires the manual invocation:

```bash
bash constitution/scripts/post_update_hook.sh
```

Installing the automatic trigger (out of scope for this dispatch — no write
outside the two allowed paths) is:

```bash
cp constitution/scripts/hooks/post-merge .git/hooks/post-merge
chmod +x .git/hooks/post-merge
```

This gap is recorded here rather than silently fixed, per the dispatch's
"do not touch anything else" constraint and per §11.4.6 (no guessing / no
silent claim of a working seam that was not verified installed).

## Deferred guards — NOT wired

Three further §11.4.109-class PreToolUse guard hooks exist in
`constitution/scripts/hooks/` and are intentionally excluded from
`.claude/settings.proposed.json`:

- `guard-work-track-binding.sh` (§11.4.191) — blocks a commit/dispatch whose
  file-scope belongs to a logic-group whose canonical `(track, branch)` in
  the workable-items registry does not match the current checkout.
- `guard-track-branch-label.sh` (§11.4.182) — blocks an `Agent`/`Task`
  dispatch whose label does not start with the `(T<N>/<branch> - <alias>)`
  prefix.
- `guard-branch-consistency.sh` (§11.4.181) — blocks creation of a
  feature-branch name that diverges from a logic-group's registered
  canonical name.

### Why deferred, and the fail-closed risk

All three are **fail-closed** guards keyed to a live registry this project
has not yet populated: a workable-items DB with `logic_group → (branch,
track, path-globs)` rows (§11.4.191), and a canonical-branch-name map
(§11.4.181). `guard-work-track-binding.sh` and `guard-branch-consistency.sh`
both document (in their own source / companion docs) that they fail closed
on an unreadable/absent registry.

**Honest risk if wired now, without that registry populated:** every commit
via a git-commit-class Bash call AND every `Agent`/`Task` dispatch
project-wide could be blocked (exit 2) the moment `.claude/settings.json`
activates — not because any real work-placement or labeling violation
occurred, but because the guard has nothing valid to check against. That is
exactly the §11.4.201 false-positive-refusal failure mode
(`CM-GUARD-ASSERTS-REAL-CONDITION`): a guard that fires on an unresolvable
condition is a FAIL-bluff of the same severity as a guard that never fires
on a real one.

Per the dispatch's explicit instruction and per §11.4.66 (blocker-resolution
interactive-clarification mandate), wiring these three guards is deferred
pending an **explicit operator decision** on:

1. Whether/how to populate the workable-items registry + canonical-branch
   map in this checkout first, or
2. Accepting the fail-closed blast radius as a deliberate, informed
   trade-off (and scheduling the registry population as immediate follow-up
   work), or
3. Wiring them with a documented, audited escape hatch until the registry
   exists.

Until that decision is made, these three hooks remain present in the
constitution submodule (inherited, available, `bash -n` clean) but
**inactive** in this project.

## Related

- `constitution/docs/AGENT_GUARDRAILS.md` — the SUBAGENT CONSTITUTIONAL
  PREAMBLE + ORCHESTRATOR PRE-ACTION CHECKLIST that §11.4.109 also mandates
  (the "ceiling" complementing this hook's "floor").
- `constitution/docs/scripts/guard-forbidden-commands.md` — the constitution
  submodule's own companion doc for the staged hook (script internals,
  full forbidden-class enumeration, escape-hatch semantics).
- `constitution/docs/scripts/post_update_hook.md` — the companion doc for
  the §11.4.164 seam (detection algorithm, two historical defects fixed
  2026-07-15, edge cases).
