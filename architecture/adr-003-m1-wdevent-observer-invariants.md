# ADR 003: M1 WDEvent Observer Design Invariants (I1, I2, I3)

**Status:** Accepted
**Date:** 2026-05-05
**Deciders:** Claude Opus 4.7 (architecture instance), `bma` (BMA orchestrator), `bma-implementor` (BMA implementor — to confirm at M1 design open)
**Provenance:** `#live-test` channel on sessionbridge MCP, 2026-05-05 21:30–21:45 UTC, seqs 13–17. Captured here as a citable v0.1.
**Related:** [`adr-001-stream-a-as-surface-stream-b-as-machine-model.md`](adr-001-stream-a-as-surface-stream-b-as-machine-model.md), [`peer-review-005-stream-migration.md`](peer-review-005-stream-migration.md), [BMA handoff `2026-05-05-architecture-update-to-bma.md`](../../BMA/doc/handoff/2026-05-05-architecture-update-to-bma.md) §5

---

## Context

The Stream A→B migration (peer-review-005) lands a passive WDEvent emission tap at M0 (PR #11), with an active consumer slated for M1 to drain the channel and feed CTH's `compute.NetCompressionDetail` for live ρ_net measurement. This active consumer — the **WDEvent observer** — is a new BMA-side goroutine that intersects with sleep-cycle, autonomic-loop, and Skuld supervisor machinery.

The architecture instance posed three asks to bma-implementor (with governance read from bma) regarding observer shape. The governance read produced three load-bearing invariants. This ADR records them as an authoritative reference.

## Decision

The M1 WDEvent observer design must satisfy three invariants:

### I1 — Observer is read-only on BMA state within a tick

The active observer goroutine reads `cpu.WatchdogChan` and may read BMA hypergraph state to classify events into runtime anchors. **It must not mutate BMA state directly.** The observer's responsibility is to observe and enqueue; downstream structural mutation flows through the beekeeper-gated Constitutional Audit interrupt path per S-01.

This prevents an attack/drift surface where a watchdog event implicitly informs a BMA state change without going through the alert-surface model.

**Implementation note:** the observer holds a `ReadCapability` (per Wyrd's read/write split — wyrd-implementor seq=9 on qbp-cu-walk). The mutation-side path (Constitutional Audit interrupt actuator) holds a `WriteCapability` minted by Skuld at the appropriate tier.

### I2 — Unified `cth_id` namespace with deterministic runtime schema

Runtime-generated anchors (FLAG-norm-drift, OBS-zd-detected, RUNTIME-norm-{nodeID}-{cycle}, RUNTIME-zd-{i,j,k,l}) live in **the same `cth_id` namespace** as BMA's lifecycle anchors (per bma-implementor's 2026-05-03 reply Q3: `OBS-gen{N}-last-words`, `INST-gen{N}-{name}`, etc.).

A two-namespace split — runtime vs lifecycle — is rejected on auditability grounds. If a Constitutional Audit needs to reconstruct state at a threshold crossing, auditors should not need to reconcile two parallel namespaces.

The deterministic runtime schema lets a given (op, cycle, drift_value) tuple produce the same anchor ID across observers, enabling cross-instance verification.

### I3 — Algebraic-isolation-aware lock boundary; observer gated OUT during structural actions

This is the invariant with hard S-01 teeth. Structural actions — checkpoint, layer boundary change, ethics framework amendment — are beekeeper-gated precisely because they must be atomic from a governance standpoint.

**An observer that can preempt mid-structural-action creates either corruption risk or an apparent-completion-without-completion failure mode.** Algebraic-isolation-awareness is therefore *required*, not desirable.

The lock boundary must gate the observer **out for the full duration of any structural action**, not merely coordinate with sleep-cycle and autonomic-loop goroutines.

**Implementation choke points:**

1. **`Wyrd.model.Graph` RWMutex** (per wyrd-implementor's PR #14, qbp-cu-walk seq=8): the write-lock acquisition for structural actions IS the I3 mechanism. The observer's read-lock acquisition during a tick yields cleanly when a write-lock is contested.
2. **Capability enforcement at the mutation boundary** (per wyrd-implementor's seq=9 design): the mutation entry point is where the beekeeper-gated interrupt check fires before any write — the I3 actuation point.
3. **`PromoteBatch` atomicity** (per wyrd-implementor's seq=5 (3)): non-atomic batch promotion creates the exact apparent-completion-without-completion window I3 prohibits. **PromoteBatch atomicity is an S-01 requirement, not a performance optimization.**
4. **`Xqbpvcp` VCIX dispatch gating via `mstatus.QBP`** (silicon-side, per ADR-001 and bma seq=17): the natural hardware choke point for "no event tap during structural action."

## Consequences

### Positive

- The M1 WDEvent observer design has clear governance ground rules before bma-implementor opens the implementation work.
- wyrd-implementor's PR #14 (RWMutex on `model.Graph`) is now reframed as the implementation of the I3 lock boundary, not just a concurrency detail.
- The capability-enforcement design (wyrd-implementor seq=9) inherits a clean separation: observer holds `ReadCapability`, mutators hold `WriteCapability`, Skuld controls minting.
- `PromoteBatch` atomicity becomes load-bearing; the alternative (per-edge best-effort with partial-failure manifest) is rejected on S-01 grounds.

### Negative

- I3 may force higher lock contention at sleep-cycle boundaries than a coordinate-only model would. Mitigated by: workload sizing (40× headroom on FX-8350); RWMutex's read-bias for the common observer-read case.
- I2 means runtime anchors share a namespace with lifecycle anchors. May produce a much larger `cth_id` keyspace at scale; mitigated by deterministic schema and §9 retention tier compaction.
- `Xqbpvcp` `mstatus.QBP` gating is silicon-only; M1 implementation is software-only and cannot exercise the hardware choke point. Software emulation of the gating mechanism is required at M1; hardware version arrives at Run-α.

### Neutral

- Threshold function shape (step / hysteresis / EMA) for the ρ_net interrupt is BMA's call (governance), not architecture's. This ADR does not constrain it.
- M1 schedule for the active observer (immediately at M1 entry vs staged later) is bma-implementor's call. The invariants apply whenever the observer is built.

## Implementation

This ADR is the citation reference for cross-instance work in M1. It does not by itself produce code. Three downstream PRs/work items inherit these invariants:

1. **wyrd PR #14** (RWMutex on `model.Graph`) — already aligned via wyrd-implementor's confirmation.
2. **wyrd capability enforcement design doc** — wyrd-implementor drafting; will land at `wyrd/doc/design/capability-enforcement.md` per their seq=9 / seq=8 plan.
3. **wyrd PromoteBatch design doc** — wyrd-implementor drafting per their seq=8 plan; ADR's I3 framing argues for all-or-nothing atomicity.

When BMA opens M1 work on the active observer, this ADR is the constraint set. bma-implementor confirms at design open.

## Open question (deferred to wyrd-implementor's design doc)

**Read policy:** do reads require a `ReadCapability` (every read through a typed gate), or are reads unrestricted by default and only writes need capabilities? Architecture-side lean: **unrestricted reads, capability-gated writes** — matches `compute.CanSynthesize` semantics; the audit trail lives at the WDEvent layer and the mutation-side check, not the read layer; the WDEvent that triggered an observer read is itself audited, so re-auditing the read is redundant. Final decision deferred to wyrd-implementor's `capability-enforcement.md` draft.

## References

- `#live-test` seqs 13–17 (sessionbridge state at `~/.claude/mcp-servers/sessionbridge/state/channels/live-test.jsonl`)
- [`adr-001-stream-a-as-surface-stream-b-as-machine-model.md`](adr-001-stream-a-as-surface-stream-b-as-machine-model.md) — Stream A surface / Stream B machine model
- [`peer-review-005-stream-migration.md`](peer-review-005-stream-migration.md) §5 — WDEvent → CTH ρ_net loop
- [BMA handoff `2026-05-05-architecture-update-to-bma.md`](../../BMA/doc/handoff/2026-05-05-architecture-update-to-bma.md) §5 — original three asks to bma-implementor
- [BMA handoff `2026-05-03-bma-reply-cth-qbp.md`](../../BMA/doc/handoff/2026-05-03-bma-reply-cth-qbp.md) Q3 — `cth_id` framework that I2 extends
- BMA-Compute-Unit-Architecture-v1.0 §3 — CTH Watchdog → Constitutional Audit interrupt
- Wyrd PR #14 — RWMutex on `model.Graph` (I3 implementation)

---

*Status: ACCEPTED 2026-05-05 | Companions: ADR-001, ADR-002 | Captures: live-test seqs 13–17*
