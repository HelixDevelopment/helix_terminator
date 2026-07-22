# helix_terminator — Gemini Project Instructions

## INHERITED FROM constitution/GEMINI.md

All rules in `constitution/GEMINI.md` (and the
`constitution/Constitution.md` it references) apply unconditionally.
Project-specific rules below extend them — they do NOT weaken any
universal clause. When this file disagrees with the constitution
submodule, the constitution wins.

@constitution/GEMINI.md

## Project

`helix_terminator` is a Go project owned by HelixDevelopment. Governance
is inherited from the Helix Constitution submodule mounted at
`constitution/`. No project-specific universal-grade rules have been
promoted yet — the parent repository currently ships no bespoke rules
that meet the universal bar (see `docs/CONSTITUTION_INHERITANCE.md`).
Project-specific configuration stays here and in
`docs/guides/HELIX_TERMINATOR_CONSTITUTION.md`; universal policy stays in
the constitution submodule.

## Inheritance verification

Run `bash tests/verify_constitution_inheritance.sh` before every build /
merge. Its paired false-positive proof is
`scripts/testing/meta_test_false_positive_proof.sh` — Constitution §1.1:
every gate ships a mutation that proves it actually bites.
