# ADR 002: Package Layout — `emulator/` Stays for M0/M1

**Status:** Accepted
**Date:** 2026-05-05
**Deciders:** James Paget Butler (beekeeper), Claude Opus 4.7 (architecture instance)
**Closes:** [issue #5 (T2)](https://github.com/JamesPagetButler/qbp-compute-unit/issues/5)
**Related:** [`peer-review-003-qbp-node-spec-crawl.md`](peer-review-003-qbp-node-spec-crawl.md) §2 T2, [`peer-review-005-stream-migration.md`](peer-review-005-stream-migration.md)

---

## Context

QBP-Node Spec Part 2 §2.2.2 proposes a package layout that does not match the existing emulator codebase:

| Proposed (QBP-Node Part 2 §2.2.2) | Existing (working, on disk) |
|---|---|
| `qbpcu/algebra/h.go` | `emulator/qword.go` |
| `qbpcu/pipeline/` | `emulator/cpu.go`, `emulator/isa.go` |
| `qbpcu/qword/` | `emulator/qword.go` |
| `qbpcu/asm/qmath_amd64.s` | `emulator/qmath_amd64.s`, `emulator/qmath_128_amd64.s` |
| `qbpcu/cim/` | (does not exist — Crawl deliverable) |
| `qbpcu/testing/tier{0,1,2,3}/` | `emulator/*_test.go` (flat) |

Part 2 §2.2.6 says *"we are integrating, not building from scratch"* but the proposed layout would move every working file. Three resolution paths exist:

1. **Discard existing scaffolding.** Move all working code under `qbpcu/`. Requires updating every import path, every test, every benchmark. Significant blast radius across `cmd/visualizer/`, `cmd/wasm-visualizer/`, all dispatch tests, and any external consumer (Wyrd, BMA-side adapters).
2. **Adopt existing layout.** Update Part 2 §2.2.2 to match `emulator/` on disk. Spec converges on reality.
3. **Documented renaming migration.** A single PR moves files into the proposed layout, all import paths updated atomically, with full test coverage to catch regressions.

## Decision

**Path 2 for M0 and M1: keep the existing `emulator/` package layout as authoritative; update QBP-Node Spec Part 2 §2.2.2 to reflect the actual on-disk structure. Path 3 (renaming migration) is a candidate for M2 boundary and remains explicitly deferred.**

### Concretely:

1. **The `emulator/` package is the authoritative QBP-CU code location** through M0 (Crawl exit) and M1 (Walk-α). All current code stays. Wyrd, BMA, and other consumers import `github.com/JamesPagetButler/qbp-compute-unit/emulator`.

2. **QBP-Node Spec Part 2 §2.2.2 will be updated** to describe the actual `emulator/` layout. The spec converges on reality rather than the reverse. The CIM subpackage (`emulator/cim/` per the actual eventual location) is added when M0 CIM Level-1 work begins per Part 2 §2.3.5.

3. **Path 3 (renaming migration) is deferred to the M2 boundary** as a candidate. At M2, when Stream B Layer 1 ops land and the package boundary between "Layer 0 / Layer 1 / Layer 2" becomes load-bearing, the `qbpcu/` reorganization may earn its complexity. Until then it doesn't.

4. **Promotion gate for Path 3 at M2:** following the QBP-Node Spec deferred-decisions discipline (§0.4.1), Path 3 advances if and only if:
   - Stream B Layer 1 ops produce a package-boundary clarity benefit (e.g., separating algebra kernels from ISA decode from pipeline scheduling)
   - The renaming PR has been scoped and benchmarked at < 1 day of churn for downstream consumers
   - No active development is in flight that the rename would conflict with (test corpus expansion, CIM Level-2 emulator)
   - Wyrd and BMA-side adapters have absorbed the M1 transition and are stable

   If any of these is not met, Path 3 is deferred again.

5. **Default position is permanent deferral.** If Path 3 never earns its complexity through Walk and Run, the `emulator/` layout stays for the lifetime of the programme. Architectural simplicity beats reorganization-for-reorganization's-sake.

## Consequences

### Positive

- **No churn for Wyrd PR coordination.** Wyrd's import path lands as `github.com/JamesPagetButler/qbp-compute-unit/emulator` — already what the working `emulator/go.mod` declares. The Wyrd integration interface ([`doc/wyrd-integration.md`](../doc/wyrd-integration.md) §2 Q1) is consistent with this ADR.
- **No risk to PR #11.** The WDEvent + QW128 work landed against the existing layout; a layout migration during PR review would have created merge conflicts and re-test cycles.
- **`cmd/visualizer/` and `cmd/wasm-visualizer/`** continue to import from `emulator/` without rewrite.
- **Delegation-friendly.** Sonnet engineering instances and Gemini SIMD work both target a stable, known layout. No "where does X live" question slows them down.

### Negative

- **QBP-Node Spec Part 2 §2.2.2 needs a small update.** Counted: ~15 lines describing the package tree. Minor doc churn.
- **The aspirational `qbpcu/` layout is on hold.** Part 2's clean architectural separation between `algebra/`, `pipeline/`, `qword/`, `mode/`, `testing/{tier0,tier1,tier2,tier3}/` is forfeit for now. If that separation matters at silicon design time, reorganization happens at M2 boundary.
- **Cosim test corpus** stays flat (`emulator/*_test.go`) rather than tiered (`testing/tier0/`, `tier1/`, etc.). Tier organization is documented in test names (`TestDispatch_Equivalence` is Tier-1, `TestCatastrophicCancellation_*` is Tier-0, etc.) but not enforced by directory structure.

### Neutral

- This decision is reversible at M2 boundary via Path 3 with explicit promotion-gate evidence.

## Implementation

This ADR closes [issue #5 (T2)](https://github.com/JamesPagetButler/qbp-compute-unit/issues/5).

### Required follow-up:

1. Update QBP-Node Spec Part 2 §2.2.2 to describe `emulator/` actual layout. ~30 min doc edit. **Owner:** architecture instance, opportunistic (next time Part 2 is touched).
2. When CIM Level-1 work begins per Part 2 §2.3.5, the CIM emulator goes at `emulator/cim/` rather than `qbpcu/cim/`. Document the location in the M0.6 follow-on issue when that lands.
3. The Path 3 promotion-gate criteria (§4 of Decision section) become a deferred-decisions appendix entry for review at M2 opening.

### Not changing:

- All current import paths
- The `emulator/go.mod` module name (`github.com/JamesPagetButler/qbp-compute-unit/emulator`)
- The flat test layout
- The Wyrd integration import path

## References

- [`peer-review-003-qbp-node-spec-crawl.md`](peer-review-003-qbp-node-spec-crawl.md) §2 T2 — Tension that prompted this ADR
- [`peer-review-005-stream-migration.md`](peer-review-005-stream-migration.md) §6 R6 — Wyrd consumer compatibility invariant
- [`doc/wyrd-integration.md`](../doc/wyrd-integration.md) §2 Q1 — Module path decision (canonical subdir)
- [`Archive/QBP-Node-Spec-v0.1-Part-2.md`](../Archive/QBP-Node-Spec-v0.1-Part-2.md) §2.2.2 — Source of the proposed `qbpcu/` layout
- [`Archive/QBP-Node-Spec-v0.1-Parts-0-and-1.md`](../Archive/QBP-Node-Spec-v0.1-Parts-0-and-1.md) §0.3 — Deferred-decisions mechanism this ADR follows

---

*Status: ACCEPTED 2026-05-05 | Closes issue #5 | Companion: ADR-001*
