#!/usr/bin/env bash
# install_git_hooks.sh
#
# Purpose:
#   Install this repository's canonical, TRACKED git hooks (scripts/git-hooks/*)
#   into the shared common git hooks directory. The common hooks dir is used by
#   the main checkout AND every `git worktree` (hooks live in the common gitdir,
#   not per-worktree), so a single install covers all of them.
#
#   Making the hook a tracked source + a reproducible installer (§11.4.77)
#   closes the failure mode that motivated this script: on 2026-07-07 a
#   subagent silently replaced the untracked .git/hooks/pre-commit with a
#   bypass probe, disabling the gate repo-wide with no reviewable diff.
#
# Usage:
#   bash scripts/install_git_hooks.sh
#
# Inputs:
#   scripts/git-hooks/*        canonical hook sources (tracked)
#
# Outputs:
#   <common-gitdir>/hooks/<name>   executable copies (mode 0755)
#
# Side-effects:
#   Overwrites any existing hook of the same name in the common hooks dir.
#
# Dependencies:
#   git, bash, coreutils (install, basename, mkdir)
#
# Cross-references:
#   scripts/git-hooks/pre-commit   the canonical pre-commit gate
#   scripts/pre_build_verification.sh   what the pre-commit hook execs
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." >/dev/null 2>&1 && pwd)"
SRC_DIR="$REPO_ROOT/scripts/git-hooks"

if [ ! -d "$SRC_DIR" ]; then
  echo "ERROR: canonical hook source dir not found: $SRC_DIR" >&2
  exit 1
fi

# Resolve the COMMON git dir (shared across worktrees), not a worktree-private
# gitdir. --git-common-dir may be relative; normalise to absolute.
COMMON_DIR="$(cd "$REPO_ROOT" && git rev-parse --git-common-dir)"
case "$COMMON_DIR" in
  /*) : ;;                                   # already absolute
  *)  COMMON_DIR="$REPO_ROOT/$COMMON_DIR" ;; # relative -> anchor to repo root
esac
HOOKS_DIR="$COMMON_DIR/hooks"
mkdir -p "$HOOKS_DIR"

installed=0
for src in "$SRC_DIR"/*; do
  [ -f "$src" ] || continue
  name="$(basename "$src")"
  install -m 0755 "$src" "$HOOKS_DIR/$name"
  echo "installed: $HOOKS_DIR/$name  <-  scripts/git-hooks/$name"
  installed=$((installed + 1))
done

if [ "$installed" -eq 0 ]; then
  echo "ERROR: no hook sources found in $SRC_DIR" >&2
  exit 1
fi

echo "OK: installed $installed hook(s) into $HOOKS_DIR"
