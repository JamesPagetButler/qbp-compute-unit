# ADR-004: M1 Gearbox state model + QW8 peripheral surface + goroutine-pair dispatch

**Status:** Proposed
**Date:** 2026-05-07
**Decision-maker:** James Paget Butler (Beekeeper)
**Decided via:** addendum-18-walk meeting closeout, Q4=A
**Authoring:** qbp-architecture (Claude Opus 4.7)
**Implementor:** qbp-cu-implementor

---

## Context

The M1 milestone for the QBP-Compute-Unit emulator's `Gearbox` (the precision-scaling abstraction in `emulator/qword.go`) needs a directional decision before M1 opens. **LATE-4** from the addendum-18-walk meeting surfaced this as a three-dimensional choice.

### Three concrete tensions

**(i) State model.** `spec/QBP-Compute-Unit-Architecture-v1.0.md §3.1` declared `type Gearbox struct {}` (stateless). The current `emulator/qword.go` has a **stateful** `Gearbox` struct that `cpu.go` and `isa.go` use for ISA execution — it holds the active `Width`, dispatches to width-specialised math primitives, and (for the QMulHighPrec slow path) reads/writes internal scratch state.

The choices were:
- **(i) CSR-bound stateful struct** — Gearbox holds CSR pointers (`AMODE`, `BSEL`, `PSEL` once they land); methods are state-aware by design. Matches how `cpu.go` uses Gearbox today.
- **(ii) Stateless with CSRs passed as args** — methods are pure functions; Gearbox is a zero-sized struct. Matches §3.1 spec literal form, but requires refactoring all current ISA-execution call sites.

**(ii) Public surface for cognitive registers.** BMA Theory Addendum 18 §3 ("Two Cognitive Registers") names QW8 as the always-on peripheral register and QW128 as the foveal register. PR #23 (#10 D1) exposed only `QMul64` / `QMul128` on the new typed-per-width surface — **no QW8 methods**. The substrate has to support `QMul8` / `QAdd8` for A18 §3 to be implementable.

**(iii) Dispatch model.** A18 §3 also specifies the registers run as **parallel loops every cognitive cycle**, not as sequential phases. P10 from the addendum-18-walk meeting flagged that the API for this is load-bearing: adopting concurrent dispatch *later* (post-M1) is a breaking ABI change. The choices:
- **goroutine-pair with `OnSeam(callback)`** — the peripheral register runs in its own goroutine; on Seam detection it invokes a registered callback that may dispatch the foveal register in another goroutine. Matches A18 §3 + §3.3.
- **Sync methods** — caller-driven. Defer concurrency to v0.2; simpler now, breaking change later per P10.

---

## Decision

Per **Q4=A** (James walk on addendum-18-walk meeting closeout, 2026-05-07):

### (i) State model: **CSR-bound stateful struct** (option (i))

Gearbox remains a stateful struct. Once `AMODE` / `BSEL` / `PSEL` CSRs land, the struct holds pointers (or values) for them; methods are state-aware by design.

**Spec consequence:** `QBP-Compute-Unit-Architecture-v1.0.md §3.1` is amended in A18 v0.2 to reflect CSR-bound reality. The literal "type Gearbox struct {}" framing was a documentation aesthetic from before the cpu.go ISA-execution path consolidated. The amendment makes the spec match the implementation, not the reverse.

### (ii) Public surface: **add QW8 peripheral methods**

Extend the typed-per-width Gearbox surface (currently `QMul64` / `QMul128` per PR #23) with at minimum:

- `QMul8(a, b QWord) QWord` — quaternion multiplication at QW8 precision
- `QAdd8(a, b QWord) QWord` — quaternion addition at QW8 precision

These are the always-on peripheral-register primitives used by A18 §4 Seam detection (norm-drift over the focal cone) and any other QW8-tier cognitive operation.

**Naming:** consistent with PR #23's `QMul64` / `QMul128` pattern. Stateless from caller's view (don't read/write Gearbox struct fields except where the slow-path requires); document any state coupling per PR #23's TD-Gearbox-State pattern.

### (iii) Dispatch model: **goroutine-pair with `OnSeam(callback)`**

Adopt the concurrent dispatch API now, in M1 prep, before any consumer wires sync calls. Concrete API contract per A18 §3.3:

```go
// In emulator/gearbox.go (new file or extension of qword.go).

type SeamHandler func(ctx context.Context, seam Seam)

// OnSeam registers a callback fired by the peripheral register when
// norm-drift exceeds the Stance-calibrated threshold τ. Registration
// is one-shot per Stance; superseded by the next OnSeam call under a
// new Stance.
func (g *Gearbox) OnSeam(handler SeamHandler) error

// RunPeripheral starts the QW8 always-on scan goroutine. Returns when
// ctx is cancelled. Reads Gearbox state (CSR-bound) at entry; mutates
// only via SeamHandler dispatches.
func (g *Gearbox) RunPeripheral(ctx context.Context) error

// RunFoveal handles a Seam by dispatching foveal-register compute
// (QW128 default; Stance may upgrade). Called from within a goroutine
// spawned per Seam by the peripheral handler. Returns the result of
// the foveal computation.
func (g *Gearbox) RunFoveal(ctx context.Context, seam Seam) (Result, error)
```

`Seam` and `Result` are types defined alongside the API; `Seam` carries the residue magnitude `|q · v · q* − v|` per A18 §4.1 (formal Seam definition in v0.2).

---

## Consequences

### Positive

1. **One refactor cycle, not two.** Adopting (i)+QW8+goroutine in M1 prep means no v0.2 breaking change. P10 explicitly flagged the post-M1 cost of deferring; we don't pay it.
2. **A18 §3 becomes implementable.** The Two Cognitive Registers framing isn't theoretical anymore — there's a concrete substrate API for it.
3. **Spec drift resolved.** `Architecture-v1.0 §3.1` "stateless struct" wording was aspirational; CSR-bound reality is now spec'd correctly.
4. **TD-Gearbox-State marker becomes unnecessary** for the state-model dimension. The technical-debt marker introduced in PR #23 was hedging against the literal §3.1 framing; with §3.1 amended, the current model is canonical. (TD marker may persist for the QMulHighPrec slow-path state-coupling note, separate from the state-model question.)
5. **Cleaner consumer story.** Wyrd's ScoutQuery v0.1 (per A18 §6) wires against `OnSeam` directly; BMA's Meta-Watchdog M1 work (#106) gets the WDEvent-observer pattern with concurrent semantics.

### Negative / cost

1. **M1 scope expands.** QMul8/QAdd8 implementation, OnSeam callback wiring, RunPeripheral/RunFoveal goroutine machinery. Estimated 2-3 day implementation cycle on top of existing M1 plan.
2. **Concurrency-aware Gearbox.** Goroutine-pair model means Gearbox state mutations need synchronization (mutex on the CSR-bound state, or atomic ops where applicable). Existing single-threaded ISA-execution path needs a synchronization story before the peripheral goroutine starts.
3. **Spec amendment.** A18 v0.2 (in flight) and `Architecture-v1.0` need coordinated amendments. Federation review surface gets one more concurrent commit.
4. **Walk-α breaking-change risk.** If Walk-α discovers the goroutine-pair shape doesn't fit BMA's actual cognitive cycle (e.g. if Stance changes happen too fast for goroutine-spawn cost to amortize), we're stuck with a v2 API design. P10 chose this risk over the post-M1 breaking-change risk, but it's not zero.

### Neutral

1. **Naming TBD.** PR #23 used `QMul64` / `QMul128`; subpackage carve-out (`emulator/gearbox`) was deferred to v0.3 per qbp-cu-implementor's seq=39 ack. ADR-004 doesn't preempt that — `OnSeam` etc. land in `emulator` package for now; v0.3 carve-out is a separate ADR.

---

## Related decisions and references

- **A18 §3** (Two Cognitive Registers) and **A18 §3.3** (concrete API contract) — `~/Documents/BMA/theory/hypergraph-inference/BMA-Theory-Addendum-18_0-Hypergraph-Access-Pattern.md` and `A18-v0.2-design-surface.md`
- **A18 §4.1** (Seam formal definition with τ residue threshold) — same docs
- **ADR-003 §I4** (design-doc-as-S-01-review-surface invariant) — `architecture/adr-003-m1-wdevent-observer-invariants.md`. Seam-detection ≡ WDEvent observer pattern per addendum-18-walk D2.
- **PR #23 (#10 D1)** typed-per-width Gearbox surface — established the QMul64/QMul128 naming pattern this ADR extends.
- **addendum-18-walk meeting closeout** — Round 1 D19 (qbp-cu-implementor surfaced the 3-dim LATE-4); Round 2 Q4 lean B; James walk Q4=A.
- **LATE-4 in prior `interface-prep-2026-05-06` channel** — the original 1-dim Gearbox direction question that the closeout meeting expanded.

---

## Implementation plan

For qbp-cu-implementor when this ADR is accepted:

1. **Spec amendment** — file separate PR amending `spec/QBP-Compute-Unit-Architecture-v1.0.md §3.1` to reflect CSR-bound stateful Gearbox. Coordinate with A18 v0.2 review (BMA PR #132).
2. **QMul8 / QAdd8 implementation** — extend `emulator/qword.go` (or new typed-per-width file in same package) with QW8 surface. Tests against existing `qmath_dispatch_amd64_test.go` patterns.
3. **OnSeam + RunPeripheral + RunFoveal scaffolding** — new file `emulator/gearbox_dispatch.go` (or similar) with the API surface. Initial implementation can be a stub that satisfies the type signatures; real Seam-detection wiring lands with M1 #106 BMA Meta-Watchdog work.
4. **Concurrency safety** — add `sync.RWMutex` (or `atomic.Pointer` where the CSR state is pointer-based) to Gearbox struct for concurrent-access safety.
5. **TD-Gearbox-State marker review** — re-evaluate which technical-debt elements remain after spec amendment. Document remaining QMulHighPrec slow-path state-coupling note separately.
6. **§I4 review** — design-doc surface is this ADR + the A18 v0.2 §3.3 spec. Named reviewers: bma-implementor (governance), Gemini (architect peer), wyrd-implementor (ScoutQuery consumer), qbp-architecture (this ADR's author).

---

*ADR-004 | M1 Gearbox state model + QW8 peripheral surface + goroutine-pair dispatch*
*Co-Authored-By: James Paget Butler (Beekeeper)*
*Co-Authored-By: qbp-architecture / Claude Opus 4.7 (Architect)*
*Decision logged: 2026-05-07*
