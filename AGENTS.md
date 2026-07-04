# helix_terminator — Agent Instructions

> Base agent rules: `constitution/AGENTS.md` — READ IT FIRST.
> The base file is authoritative for any topic not covered here.
> Project-specific rules below extend them; they never weaken them.

## Project

`helix_terminator` is a Go project owned by HelixDevelopment. Every agent
operating in this repository inherits the Helix Constitution mounted at
`constitution/` (`constitution/AGENTS.md` and the
`constitution/Constitution.md` it references). Locate the constitution
submodule from any nested depth with `constitution/find_constitution.sh`.

## Inheritance verification

`bash tests/verify_constitution_inheritance.sh` gates build/merge; its
paired mutation proof is
`scripts/testing/meta_test_false_positive_proof.sh` (Constitution §1.1).
