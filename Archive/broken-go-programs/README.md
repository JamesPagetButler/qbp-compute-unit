# Quarantined broken Go programs

This directory holds Go programs that have been broken since the initial commit (`68cc13e`, 2026-04 era) and that referenced APIs that never existed in the corresponding packages. They are preserved here as forensic audit material per `feedback_branch_cleanup.md` (no pre-stable deletion of historical artifacts) and to honor the experimental authoring lineage.

Each program carries a `.go.disabled extension` build tag at the top of every `.go` file, which prevents the Go toolchain from compiling them while preserving syntax highlighting + readability for future forensic review. The directory itself is NOT a Go package from any module's perspective because the build-tagged files are filtered out before package discovery.

## What's here + why each is broken

### `qbp-compute-unit-exp_i_sched_emu/`

**Origin:** `qbp-compute-unit/cmd/exp_i_sched_emu/main.go` (commit `68cc13e`).

**Intent (per the file header):** scheduler → emulator integration test. Three PASS/FAIL checks for a width-aware execution pipeline:
1. Pipeline execution at `ctx.Width` with correct cycle counts + non-zero drift
2. Reallocation round-trip (`CheckReallocation` advises upgrade; `PromoteWidth` charges stall cycles)
3. Width discrimination (drift increases monotonically as width narrows)

**Why broken:** references four APIs that don't exist in the current package surface:

| Missing API | Used at | Package state |
|---|---|---|
| `(*mesh.Task).Context()` | `main.go:129` (and 5 other lines) | `mesh.Task` struct has no `Context()` method or field |
| `emu.WidthToCode(qword.Width) emu.WidthCode` | `main.go:130, :233` | `emu.WidthCode` has `ToWidth()` (the inverse direction); no `WidthToCode` function exists |
| `mesh.CheckReallocation` | (per header comment) | not present in `mesh/` |
| `(*emu.Engine).PromoteWidth` | (per header comment) | not present in `emu/` |

The program documents an intended scheduler subsystem that was never finished. The header comments are a clean specification of what the missing APIs were supposed to do; revival work could use them as a starting point.

### `qbp-compute-unit-squam_loon_test/`

**Origin:** `qbp-compute-unit/cmd/squam_loon_test/` (commit `68cc13e`).

**Intent:** four BMA cognitive-test programs in one directory, each demonstrating a different protocol:
1. `consensus_reconciliation.go` — disconnected reconciliation (Instance B sovereign re-entering cluster with Instance A)
2. `curiosity_phase.go` — proactive curiosity / autonomous scouting (QBP-Gravity-Bridge, Feynman scout persona)
3. `loading_phase.go` — loading & cross-domain interrogation
4. `main.go` — "SQUAM LAKE LOON POPULATION (1976-2026)" cognitive test

**Why broken:** all four files declare `package main` and define `func main()` in the same directory. Go requires exactly one `main()` per package, producing `main redeclared in this block` compile error.

Each file is internally consistent (would build as its own program) but the four-file layout breaks Go's one-main-per-package rule. Revival would split each file into its own `cmd/<name>/` subdirectory.

## Provenance

Quarantined to this location via PR #48 (refs `#17` v0.3 §A.2 + `#45`, 2026-05-20). Disposition decision: option (B) "quarantine to `Archive/broken-experiments/`" per beekeeper ruling on the 2026-05-20 federation-PR-sweep session.

Files retained for:
- Forensic value (experimental code from the 2026-04 era that documented intended subsystem APIs)
- Header comments specifying never-built behavior (useful starting points if anyone wants to revive the scheduler subsystem or the four cognitive-test programs)
- Audit-trail discipline per `feedback_branch_cleanup.md` (no pre-stable deletion)

## Reviving any of these

If a future contributor wants to revive one of these:

**For `qbp-compute-unit-exp_i_sched_emu/`:** start by reading the header comments which spec the four missing APIs (`mesh.Task.Context()`, `emu.WidthToCode`, `mesh.CheckReallocation`, `Engine.PromoteWidth`). Decide whether the scheduler subsystem should be revived or whether its claims should be folded into a different design. Move the file back to `qbp-compute-unit/cmd/exp_i_sched_emu/` and remove the `.go.disabled extension` tag; implement the missing APIs in `mesh/` + `emu/`.

**For `qbp-compute-unit-squam_loon_test/`:** split each file into its own `qbp-compute-unit/cmd/squam-loon-<name>/` (or similar) subdirectory. Remove the `.go.disabled extension` tag from each file. Each file would compile and run independently as a cognitive-test demonstration program.

Neither program is currently a federation work-item; both are forensic-only at this archive.
