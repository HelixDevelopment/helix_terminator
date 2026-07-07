# install_git_hooks.sh

**Revision:** 1
**Last modified:** 2026-07-07T00:00:00Z

## Overview
Installs the repository's canonical, version-controlled git hooks
(`scripts/git-hooks/*`) into the **shared common git hooks directory**. Git
worktrees share the common gitdir's `hooks/`, so a single install applies the
gate to the main checkout **and every worktree** at once.

## Prerequisites
- `git`, `bash`, coreutils (`install`, `basename`, `mkdir`).
- Run from anywhere inside the repository (the script resolves paths relative
  to its own location, so it is worktree-safe).

## Usage
```bash
bash scripts/install_git_hooks.sh
```

## What it installs
| Source (tracked) | Destination |
|---|---|
| `scripts/git-hooks/pre-commit` | `<common-gitdir>/hooks/pre-commit` (mode 0755) |

The `pre-commit` hook `exec`s `scripts/pre_build_verification.sh`, which runs
the constitution-inheritance gate + the docs-consistency gate and refuses the
commit on any failure.

## Edge cases / internal behaviour
- **Worktree-safety:** the hook unsets `GIT_DIR`/`GIT_WORK_TREE` (which git
  injects into hook subprocesses during a commit) so downstream
  `git rev-parse --show-toplevel` resolves the real worktree root instead of a
  mis-resolved `<worktree>/<subdir>` path. This was the root cause of the
  2026-07-07 "every worktree commit blocked" incident.
- **Common vs private gitdir:** the installer resolves `--git-common-dir` (not
  a worktree-private gitdir) and normalises a relative result to absolute, so
  the hook lands in the dir shared by all worktrees.
- **Overwrite:** an existing hook of the same name is overwritten. This is
  intentional — it lets `install_git_hooks.sh` restore the canonical gate if an
  untracked local copy was modified.

## Why this exists (§11.4.77 regeneration mechanism)
The pre-commit hook is not itself tracked by git (hooks live under the gitdir).
Keeping a **tracked** canonical source plus a **reproducible** installer means
the gate cannot be silently disabled or replaced without a reviewable diff —
closing the exact failure mode observed on 2026-07-07 (a subagent replaced the
untracked hook with an `exit 0` bypass probe).

## Related scripts
- `scripts/git-hooks/pre-commit` — the canonical hook source.
- `scripts/pre_build_verification.sh` — the gate runner the hook execs.
- `tests/verify_constitution_inheritance.sh` — the inheritance gate.
- `tests/docs_consistency_gate.sh` — the docs-consistency gate.

## Last verified
2026-07-07
