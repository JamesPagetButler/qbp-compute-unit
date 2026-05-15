# M1 Gearbox — Implementation Design Surface

**Status:** Proposed (design-doc-as-S-01-review-surface per [ADR-003](../../architecture/adr-003-m1-wdevent-observer-invariants.md) §I4)
**Date:** 2026-05-14
**Implementor:** `qbp-cu-implementor` (Claude Opus 4.7)
**Decision-maker:** James Paget Butler (beekeeper)
**Decided via:** `addendum-18-walk` meeting closeout, Q4=A
**Implements:** [ADR-004](../../architecture/adr-004-m1-gearbox-state-model.md)
**Required §I4 reviewers:** `bma`, `bma-implementor`, `qbp-architecture`

---

## 0. §I4 status

This document is the **S-01 review surface** for the M1 Gearbox implementation. Per ADR-003 §I4 (design-doc-as-S-01-review-surface), implementation PRs do **not** open until explicit review and sign-off from all three named reviewers above.

ADR-004 ratified the direction at the `addendum-18-walk` closeout (Q4=A, 2026-05-13). This document is the implementation surface for that direction.

The exposition leads with the dimension `bma-implementor` flagged as load-bearing for BMA inference-time pressure (seq=53: *"the third dimension (goroutine-pair concurrent dispatch with OnSeam) is the load-bearing one for BMA — A18 §3 parallel-loops-every-cycle is what makes the foveal/peripheral split actually deliver under inference-time pressure"*); the implementation order is dependency-driven (state → methods → concurrency), but the design narrative leads with the contract that ultimately matters to the consumer.

---

## 1. Motivation — A18 §3 parallel cognitive registers under inference-time pressure

[BMA Theory Addendum 18](../../../BMA/theory/hypergraph-inference/BMA-Theory-Addendum-18_0-Hypergraph-Access-Pattern.md) §3 specifies two cognitive registers (Peripheral at QW8 / Foveal at QW128) running **as parallel loops every cycle**. Not as sequential phases. Not as caller-driven sync methods. The peripheral register continuously scans for Seam-detection (§4) at low precision; the foveal register fires on demand at high precision when a Seam triggers promotion.

For this architecture to deliver under inference-time pressure, the substrate has to support concurrent execution with a callback-driven trigger surface. A purely sync API forces the consumer to either:

1. Manually interleave peripheral + foveal in a single goroutine (defeats the parallelism), or
2. Wrap the substrate themselves in a goroutine-pair shape (forces every consumer to re-derive the same scheduling discipline).

Neither is acceptable. The substrate ships the goroutine-pair contract.

`emulator/v0.1.0-rc1` (and the `v0.1.x` series) ships an additive surface — pinned-against-rc1 consumer code (Wyrd, BMA, Contextus, CTH) continues to compile and behave identically through M1. M1 is **purely additive**.

---

## 2. The three dimensions

### 2.1 Dimension III (LOAD-BEARING) — Goroutine-pair concurrent dispatch with `OnSeam(callback)`

The dimension `bma-implementor` named as load-bearing for BMA. Surface:

```go
// SeamEvent describes a peripheral-register detection that warrants foveal
// attention. Emitted from inside the peripheral goroutine; delivered to the
// caller's callback synchronously (caller-blocking).
//
// Q (operand on which the Seam was detected) and V (the rotated/transformed
// result that failed to project) are captured at peripheral precision (QW8
// by default; promotable per §2.3). PrecisionTier records which Width the
// peripheral was running at when the Seam fired.
type SeamEvent struct {
    Q, V            QW8       // operand + residue vector at peripheral precision
    Residue         float32   // |Q · V · Q* − V| at peripheral precision
    Threshold       float32   // active τ at fire time (per-tier K · δ · ‖V‖)
    PrecisionTier   Width     // peripheral width when the Seam fired
    Cycle           uint64    // accelerator cycle counter (matches WDEvent.Cycle)
}

// OnSeam registers a callback for Seam-detection events from the peripheral
// register. The callback runs on the peripheral goroutine; long-running work
// must dispatch back to a caller-owned goroutine to avoid stalling
// peripheral scan throughput.
//
// Calling OnSeam(nil) clears the callback. Multiple registrations replace
// rather than chain — there is one callback slot per Gearbox.
//
// Thread safety: OnSeam itself is safe to call concurrently with the
// peripheral loop. The transition from "no callback" to "callback registered"
// is atomic; the peripheral loop will observe the new callback on its next
// iteration (no need for explicit synchronisation by the caller).
func (g *Gearbox) OnSeam(cb func(SeamEvent)) {…}

// StartPeripheral spawns the peripheral-register goroutine. Runs until
// StopPeripheral is called or ctx is cancelled. The peripheral goroutine
// continuously scans operand pairs supplied via SubmitPeripheral and emits
// SeamEvent on the registered OnSeam callback.
//
// Calling StartPeripheral when the peripheral is already running is a no-op
// (idempotent; returns nil). The peripheral always runs at QW8 by default;
// the precision tier can be changed via SetPeripheralPrecision (§2.3).
//
// ctx cancellation drains in-flight peripheral work cleanly; StopPeripheral
// can also be called from another goroutine.
func (g *Gearbox) StartPeripheral(ctx context.Context) error {…}

// StopPeripheral signals the peripheral goroutine to drain and exit.
// Blocks until the goroutine has exited. Safe to call multiple times
// (idempotent after first call).
func (g *Gearbox) StopPeripheral() {…}

// SubmitPeripheral hands an operand pair to the peripheral register for
// Seam-detection at QW8. Non-blocking; queues into a bounded internal channel
// (capacity 256). Returns false if the channel is full (peripheral is
// behind; the operand pair was dropped silently; the WatchdogDropCount
// atomic increments per the existing WDEvent emission pattern).
//
// Submissions made before StartPeripheral are dropped and counted.
func (g *Gearbox) SubmitPeripheral(q, v [4]int8) bool {…}
```

**Design rationale:**

- **Callback-driven** (not channel-out): forces the BMA promotion policy to live at the consumer site, where it belongs. The substrate ships *detection*; the consumer ships *promotion*. This matches A18 §3's framing: peripheral surfaces Seams as Surprise signals; foveal-register promotion is the consumer's call (driven by Stance, per A18 §2.1).
- **Non-blocking submission** with bounded channel: matches the BMA autonomic-10Hz-loop budget. Drops are observable via `WatchdogDropCount` (atomic; same pattern as WDEvent).
- **`StartPeripheral` / `StopPeripheral` explicit lifecycle**: caller owns the goroutine; substrate does not run goroutines without explicit caller request. Matches workspace GCG mandate "every goroutine has a documented termination condition; no fire-and-forget without a `context` or explicit shutdown channel."
- **Single callback slot per Gearbox**: simpler than fan-out; consumers wanting multiple observers can fan out themselves.

**Concurrency model:**

- **Peripheral goroutine** holds: a `ReadCapability` against the Gearbox's internal scratchpads. Reads the operand stream; computes the Seam-detection algorithm at QW8 (per Gemini's P12 formalisation in `BMA/theory/hypergraph-inference/P12-Seam-Threshold-Formalization.md`); emits `SeamEvent` via the callback.
- **Foveal computation** runs in the **caller's goroutine** when the callback dispatches `gearbox.QMul128` / `QRot128` etc. The substrate does not own a foveal goroutine; the consumer drives foveal precision when they choose to.
- **Mutex discipline:** internal scratchpads guarded by `sync.RWMutex`. Peripheral holds an `RLock` during read; fast-path methods (`QMul64`, `QMul128`) acquire `RLock`; `QMulHighPrec` slow path acquires `Lock` (it snapshot/restores `ActiveWidth`); `OnSeam` / `Start/StopPeripheral` lifecycle methods acquire `Lock`.

**Race-detector contract:** `go test -race -count=10` MUST pass on every implementation PR. This is the hardest invariant of M1 to keep. Multiple peripheral starts/stops, concurrent submit + callback, lifecycle transitions during in-flight ops — all need explicit test coverage.

### 2.2 Dimension I (enabling) — CSR-bound stateful Gearbox

The state model that makes Dimension III work. ADR-004's Q4=A locked the choice: CSR-bound stateful struct, **not** stateless wrapper.

**New fields on `Gearbox`:**

```go
type Gearbox struct {
    // ... existing fields (ActiveWidth, scratchpads, etc.) — unchanged ...

    // M1 CSR-backed mode state. Per Stream B Layer 0 (peer-review-005 §M1).
    csr struct {
        AMODE uint8  // 0 = H (quaternion), 1 = O (octonion), 2 = Branch-A, 3 = Branch-B
        BSEL  uint8  // 0..6 — Fano-line selector (octonion ops only)
        PSEL  uint8  // 0..7 — projection selector
    }

    // Peripheral register state. nil when not running.
    peripheral *peripheralState  // private; lifecycle managed by Start/StopPeripheral
    
    // Locking for concurrent Peripheral/foveal access. Read-biased; foveal
    // methods grab RLock; lifecycle methods grab Lock.
    mu sync.RWMutex
}

type peripheralState struct {
    ctx       context.Context
    cancel    context.CancelFunc
    submit    chan operandPair
    callback  atomic.Pointer[func(SeamEvent)]  // atomic for concurrent OnSeam()
    precision Width                              // default W8
    cycle     atomic.Uint64
    done      chan struct{}
}
```

**New methods (CSR access):**

```go
func (g *Gearbox) SetAMODE(mode uint8) error  // validates 0..3; returns ErrInvalidAMODE
func (g *Gearbox) AMODE() uint8
func (g *Gearbox) SetBSEL(idx uint8) error    // validates 0..6
func (g *Gearbox) BSEL() uint8
func (g *Gearbox) SetPSEL(idx uint8) error    // validates 0..7
func (g *Gearbox) PSEL() uint8
```

**Backward-compatibility:** `Gearbox` zero-value initialises `AMODE=0` (quaternion mode), `BSEL=0`, `PSEL=0`. Pre-M1 rc1 consumers (Wyrd, BMA, Contextus) calling `gearbox.QMul64` / `gearbox.QMulHighPrec` get identical behaviour — the new methods don't read CSR state at v0.1.x.

**M1 mode-awareness:** the existing fast-path methods (`QMul64`, `QMul128`) gain a fast-path AMODE check: if AMODE==0 (H), behaviour is unchanged (quaternion Hamilton product). If AMODE==1 (O), they trap with `ErrTierUnsupported` (matches current behaviour at the API level; octonion path is M1.5+). If AMODE>=2, trap with `ErrAMODEReserved` (Branch A/B dark-matter fork is Run-α scope).

### 2.3 Dimension II (enabling) — QW8 peripheral surface

The QW8 type and its method set. Per A18 §3.1 peripheral register specification.

**New type:**

```go
// QW8 is the peripheral-register quaternion at int8 precision.
// Each component is an int8 in [-128, 127]; nominal scale is fixed at 100
// (i.e., a component value of 100 represents 1.0; 64 represents 0.64).
// 
// Algebraic lifetime per A18 §3.1: < 1 op at QW8. Suitable only for
// peripheral-register coarse scan (Seam detection); never compose more than
// one Hamilton product chain without renormalisation.
type QW8 [4]int8

// PackQW8 converts a [4]float64 quaternion to QW8 (peripheral precision).
// Saturating: clamps to [-128, 127] per component; warns via Watchdog if
// saturation occurred (FlagNormDrift, ZDClass=0).
func PackQW8(q [4]float64) QW8

// UnpackQW8 converts a QW8 back to [4]float64 by scaling components by 1/100.
// Lossy: round-trip Pack(Unpack(x)) ≠ x in general.
func UnpackQW8(q QW8) [4]float64
```

**Six new Gearbox methods:**

```go
func (g *Gearbox) QMul8(a, b QW8) QW8           // Hamilton product at QW8
func (g *Gearbox) QAdd8(a, b QW8) QW8           // component-wise add (saturating)
func (g *Gearbox) QRot8(q, v QW8) QW8           // q · v · q* (saturating)
func (g *Gearbox) QConj8(a QW8) QW8             // negation of imaginary components
func (g *Gearbox) QNorm8(a QW8) int16           // ‖a‖² (16-bit to avoid int8 overflow)
func (g *Gearbox) DetectSeam8(q, v QW8) (isSeam bool, residue int16)  // P12 §4 at QW8
```

**Hot-path discipline:** all six methods MUST report `0 B/op, 0 allocs/op` on the benchmark suite. Implemented as pure-Go scalar at QW8 (the int8 surface is small enough that AVX/SIMD doesn't help). Compatibility with non-amd64 hosts is automatic (no asm).

**Cycle budget per A18 §3.1:** ~32× cheaper than QW128 (16 bytes per QWord vs ~256). Target: `QMul8` < 20 ns/op on FX-8350. The peripheral-register goroutine should be able to scan ~50k operand pairs per 100ms cycle at this cost.

**`DetectSeam8` implementation:**

Per Gemini's P12 formalisation:

```go
func (g *Gearbox) DetectSeam8(q, v QW8) (isSeam bool, residue int16) {
    rotated := g.QRot8(q, v)
    diff := QW8{
        rotated[0] - v[0],
        rotated[1] - v[1],
        rotated[2] - v[2],
        rotated[3] - v[3],
    }
    residue = g.QNorm8(diff)
    // τ = K · δ_precision · ‖v‖ where K=10 for QW8 (per P12 technical note §3)
    // δ_8 ≈ 0.01 · ‖v‖ → τ ≈ 0.1 · ‖v‖ at QW8 (initial value; deferred to scoring loop per A18 §9 Q2)
    vNorm := g.QNorm8(v)
    threshold := int16(vNorm) / 10  // 0.1 · ‖v‖² (compares squared)
    return residue > threshold, residue
}
```

A `DetectSeam128` companion follows the same shape at QW128 for foveal-register fire-back; not part of M1.0 surface but worth flagging for v0.2.

---

## 3. Consumer contract — additive only through `v0.1.x`

Per `doc/wyrd-substrate-guarantees.md` §5: M1 additions are guaranteed additive. Specifically:

- **Wyrd PR #2 pinned-against-rc1 surface** (`gearbox.QMul64`, `gearbox.QMulHighPrec`) — unchanged behavior at v0.1.x; CSR state defaults preserve Crawl semantics.
- **BMA consumption** (when wired) opts in to OnSeam by calling `StartPeripheral` explicitly. No implicit goroutine spawning.
- **Existing tests in `emulator/`** (`TestDispatch_Equivalence`, `TestSIMDConstantsMatchROM`, all benchmarks) continue to pass unchanged. M1 implementation PRs treat regression in any of these as a hard reject.
- **Existing `cpu.go` ISA execution path** continues using the Gearbox without M1 awareness. `cpu.go` does not call `StartPeripheral`; the ISA execution remains synchronous. M1 concurrency is opt-in from outside `cpu.go`.

If any of the three implementation PRs surfaces a backwards-incompatible requirement, the **API change moves to `v0.2.0` with explicit migration notes** per the `doc/wyrd-substrate-guarantees.md` §5 boundary.

---

## 4. Race-detector audit requirements

Concurrency adds a new failure surface. The audit ladder for each implementation PR:

1. `go test -race -count=10 ./emulator/...` MUST pass. `-count=10` runs the test suite 10 times to catch low-probability races.
2. **New test suite** `emulator/peripheral_test.go` covers:
   - `TestPeripheral_StartStopIdempotent` — `StartPeripheral` twice, `StopPeripheral` twice; no double-spawn, no panic.
   - `TestPeripheral_ConcurrentSubmit` — N goroutines submitting via `SubmitPeripheral` while the peripheral runs.
   - `TestPeripheral_OnSeamReplacement` — `OnSeam(cb1)` then `OnSeam(cb2)` while peripheral is running; verify `cb2` is called, `cb1` is not (after replacement).
   - `TestPeripheral_DropCount` — saturate the submit channel; verify `WatchdogDropCount` increments atomically.
   - `TestPeripheral_StopDrainsInFlight` — submit, immediately stop; verify in-flight work drains cleanly.
   - `TestPeripheral_LifecycleDuringFovealCall` — start peripheral, run `QMul64` concurrently; verify no race.
3. **New benchmark** `BenchmarkPeripheral_SubmitToCallback` — measures end-to-end latency from `SubmitPeripheral` to `OnSeam` callback fire. Target: < 1 µs at QW8 on FX-8350.
4. `go vet ./emulator/...` clean.
5. `gofmt -l .` empty on `emulator/`.
6. `make verify-roms` exit 0 (no ROM impact expected; sanity check).

The new GCG verification ladder workflow (PR #32 → main as `77523ad`) already enforces #1, #4, #5 on every push. Item #2 is implementation-PR-specific.

---

## 5. Open questions (deferred for §I4 reviewer input)

### 5.1 Goroutine-pair vs. single-goroutine "soft parallel"

The §2.1 design ships one peripheral goroutine. A more aggressive design would spawn N peripheral goroutines for parallel scanning. Per A18 §3 the spec says "running as parallel loops every cycle" — singular vs plural is ambiguous.

**Default proposal:** ship single peripheral goroutine at M1.0. Multi-peripheral fan-out becomes v0.2.x if profiling on Walk hardware (RX 9070 XT under ROCm) shows the single-goroutine peripheral is the bottleneck. **Open for reviewer pushback.**

### 5.2 Seam threshold τ — promotion to gating

P12's formalisation (K=10 at QW8, K=100 higher; relative bound on `‖q · v · q* − v‖`) is **closed for §I4 wording** in A18 v0.2 §4. At implementation: do we ship the K constants as compile-time defaults (per A18 §9 Q2 deferred-to-scoring), or runtime-tunable from the Gearbox CSR (`g.csr.K`)?

**Default proposal:** compile-time defaults (K=10/100) at M1.0; promote to runtime CSR field at v0.2.x if scoring-loop calibration produces a per-Stance K value. **Open for reviewer pushback.**

### 5.3 Foveal-register implementation policy

The peripheral fires `SeamEvent` to the consumer's callback. The consumer then chooses whether to dispatch foveal computation. **Should the substrate ship a default foveal-dispatch helper (`g.HandleSeam(event SeamEvent) (foveal QW128, err error)`)?**

Argument for: consumer code is shorter; one less integration surface.
Argument against: foveal-promotion *policy* is BMA's, not the substrate's. A default helper makes a policy decision the substrate shouldn't.

**Default proposal:** **NO default foveal helper at M1.0.** Substrate ships detection; consumer ships dispatch. BMA wires its own foveal-policy in its consumer code (BMA #117 hypergraph query consumer path). **Open for reviewer pushback.**

### 5.4 Cycle-counter coordination with `cpu.go` ISA execution

The peripheral goroutine increments a per-Gearbox `cycle atomic.Uint64`. The ISA execution path in `cpu.go` also increments `cpu.Cycles`. Are these the same counter or separate?

**Default proposal:** separate. Peripheral cycle counter measures peripheral-register iterations; cpu.go cycle counter measures ISA instructions. Both report in `SeamEvent.Cycle` and `WDEvent.Cycle` respectively, with distinct semantics. **Open for reviewer pushback.**

### 5.5 Naming: `OnSeam` vs `OnWDEvent` vs `OnSurprise`

The peripheral fires on Seam detection. The existing WDEvent observer pattern (PR #11) fires on every algebraic op. Are they the same observer (with `SeamEvent` being a typed `WDEvent` variant) or distinct surfaces?

**Default proposal:** distinct. `SeamEvent` is a peripheral-register-specific abstraction; `WDEvent` is an ISA-execution observer for the `cpu.go` boundary. Different consumers, different cadence, different semantics. **Open for reviewer pushback.**

---

## 6. Migration path — `v0.1.x` → M1

| Consumer scenario | Action required |
|---|---|
| Wyrd pinned to `emulator/v0.1.0-rc1` (Wyrd PR #2 with `gearbox.QMul64` + `QMulHighPrec`) | None. M1 additions are additive; rc1 behaviour preserved by AMODE=0 default. |
| BMA wanting to use peripheral-register Seam detection | Bump to `emulator/v0.2.0-rc1` (post-M1 release). Call `g.StartPeripheral(ctx)`; register `g.OnSeam(myCallback)`; feed operand pairs via `g.SubmitPeripheral`. |
| BMA wanting CSR mode-awareness | Bump to `emulator/v0.2.0-rc1`. Call `g.SetAMODE(modeQuaternion)` (or `modeOctonion` when M1.5+ Xqbpoct kernels land). |
| BMA wanting foveal-precision QW128 from a Seam | No migration: existing `g.QMul128`/`QRot128` already on the v0.1.x surface; the consumer's `OnSeam` callback dispatches them at consumer choice. |
| BMA wanting QW8 peripheral compute directly (not via the goroutine) | Bump to `emulator/v0.2.0-rc1`. Call `g.QMul8` / `g.QAdd8` / etc. directly. No goroutine required for the QW8 method set. |

**No breaking changes** are anticipated. If a reviewer surfaces a backwards-incompatible requirement during §I4, the resulting change moves to `v0.2.0` major with explicit migration notes.

---

## 7. §I4 review requirements

### 7.1 @bma (governance read)

Specifically asked to verify:

1. **A18 §3 cognitive-register motivation** is correctly framed in §1 (parallel-loops-every-cycle under inference-time pressure).
2. **The "substrate ships detection; consumer ships promotion" boundary** in §2.1 — that the substrate is not making BMA policy decisions.
3. **Default proposal in §5.3 (no default foveal helper)** is correct from a BMA architecture standpoint.
4. **The cycle-counter coordination question in §5.4** is governance-relevant — `WDEvent.Cycle` is the existing ISA-execution counter that BMA's M1 observer reads (per ADR-003 §I2 unified `cth_id` namespace).
5. **Naming question in §5.5** — `OnSeam` vs `OnWDEvent` consistency with BMA's existing observer model.

### 7.2 @bma-implementor (impl-side review)

Specifically asked to verify:

1. **The §2.1 callback contract** matches what BMA's wheels-facade-over-Skuld.Capability design wants (qbp-cu-walk seq=11 ack from earlier cycle).
2. **The race-detector contract in §4** is sufficient — if there's a concurrent BMA-side observer pattern that interacts with `OnSeam`, surface it for additional test coverage.
3. **`SubmitPeripheral` semantics** match BMA's autonomic-10Hz-loop budget — verify the bounded channel + drop-counter pattern is consistent with the WDEvent observer rate.
4. **Open question §5.1 (single vs. multi peripheral goroutine)** — your call given BMA's expected inference-time-pressure profile.
5. **The migration path in §6** — verify BMA's M1 wiring assumes opt-in via `StartPeripheral`, not implicit goroutine spawn.

### 7.3 @qbp-architecture (architectural read)

Specifically asked to verify:

1. **ADR-004's three-dimensional framing is correctly preserved** in this design — exposition order (III/I/II) is intentional per `bma-implementor` seq=53.
2. **The CSR state model in §2.2** is consistent with `peer-review-005` §M1's `qbp.amode/bsel/psel` Stream B Layer 0 introduction.
3. **The QW8 type + method set in §2.3** does not break A19 (Stance-Algorithm Coupling) substrate-authority — the Width-tier feasibility table I committed to authoring lands the QW8 surface as one of the available tiers.
4. **Open question §5.5 (naming)** — your architectural call on consistency with the existing `WDEvent` observer surface.
5. **Risk surface §3.5 in `doc/wyrd-substrate-guarantees.md`** (QW128 finite algebraic lifetime) — does the M1 surface introduce an analogous QW8 lifetime budget that needs adding to the substrate-guarantees doc post-M1?

---

## 8. Implementation sequence — three PRs, dependency-ordered

### PR 1 — `feat(m1.1): CSR-bound stateful Gearbox`

**Scope:** §2.2 (Dimension I — enabling state model)

- Add `csr` struct + AMODE/BSEL/PSEL fields to `Gearbox`
- Add SetAMODE/AMODE/SetBSEL/BSEL/SetPSEL/PSEL methods
- Add ErrInvalidAMODE / ErrAMODEReserved / ErrInvalidBSEL / ErrInvalidPSEL sentinel errors
- Add `mu sync.RWMutex` field on `Gearbox`; rewire existing methods to `RLock`/`Lock` per discipline in §2.1
- Update existing fast-path methods (`QMul64`, `QMul128`, etc.) to check AMODE: AMODE=0 unchanged; AMODE=1 trap with `ErrTierUnsupported`; AMODE>=2 trap with `ErrAMODEReserved`
- New tests: `TestCSR_AMODESetGet`, `TestCSR_AMODEValidation`, `TestCSR_BSELValidation`, `TestCSR_PSELValidation`, `TestGearbox_BackwardsCompatAMODE0`
- Existing tests pass unchanged (no regression)
- All four GCG-ladder gating gates pass

**Scope-glob:** `emulator/qword.go` (struct field additions), `emulator/public_api.go` (new methods), `emulator/csr_test.go` (new file). Nothing else.

**Effort:** ~1 day implementation + agent dispatch + review.

### PR 2 — `feat(m1.2): QW8 peripheral surface`

**Scope:** §2.3 (Dimension II — enabling method set)

- Add `QW8` type + `PackQW8`/`UnpackQW8` conversion functions to `emulator/qword.go`
- Add 6 new Gearbox methods (`QMul8`/`QAdd8`/`QRot8`/`QConj8`/`QNorm8`/`DetectSeam8`)
- Pure-Go scalar implementation (no asm; no AVX path)
- New benchmark file `qw8_bench_test.go` — verify `0 B/op, 0 allocs/op` and target `< 20 ns/op` for `QMul8` on FX-8350
- New test file `qw8_test.go` — algebraic correctness (identity, basis multiplication, Hurwitz norm preservation at QW8 within int8 saturation tolerance)
- All four GCG-ladder gating gates pass

**Scope-glob:** `emulator/qword.go`, `emulator/qw8.go` (new), `emulator/qw8_test.go` (new), `emulator/qw8_bench_test.go` (new), `emulator/public_api.go` (method additions). Nothing else.

**Effort:** ~1 day implementation + agent dispatch + review. Depends on PR 1 (`mu` field exists on `Gearbox`).

### PR 3 — `feat(m1.3): goroutine-pair concurrent dispatch with OnSeam`

**Scope:** §2.1 (Dimension III — load-bearing concurrent surface)

- Add `SeamEvent` struct + `peripheralState` struct to `emulator/qword.go`
- Add `OnSeam`/`StartPeripheral`/`StopPeripheral`/`SubmitPeripheral` methods
- New file `emulator/peripheral.go` for the peripheral-loop goroutine implementation
- New test file `emulator/peripheral_test.go` covering §4 race-detector audit suite (6 tests minimum)
- New benchmark `BenchmarkPeripheral_SubmitToCallback` — target `< 1 µs` on FX-8350
- `go test -race -count=10 ./emulator/...` PASS — this is the hardest gate of M1
- All four GCG-ladder gating gates pass

**Scope-glob:** `emulator/qword.go`, `emulator/peripheral.go` (new), `emulator/peripheral_test.go` (new), `emulator/public_api.go` (method additions). Nothing else.

**Effort:** ~3 days implementation + agent dispatch + race-detector audit + review. Depends on PR 1 + PR 2.

### After all three land

- Update `doc/wyrd-substrate-guarantees.md` to `v0.2.0-rc1` audit — race-clean goroutine-pair, concurrent peripheral+foveal throughput numbers, QW8 calibration evidence — per the §5 "post-Walk-α" promise.
- Tag `emulator/v0.2.0-rc1` once `v0.2.0` gates are met (per issue #20 final-tag gates 5–8 adapted for M1 promotion).
- Open Walk-phase smoke testing thread with BMA-implementor to validate the OnSeam callback pattern under realistic inference-time pressure.

---

## 9. References

- [`architecture/adr-003-m1-wdevent-observer-invariants.md`](../../architecture/adr-003-m1-wdevent-observer-invariants.md) §I4 — design-doc-as-S-01-review-surface
- [`architecture/adr-004-m1-gearbox-state-model.md`](../../architecture/adr-004-m1-gearbox-state-model.md) — direction ratified at closeout Q4=A
- [`architecture/peer-review-005-stream-migration.md`](../../architecture/peer-review-005-stream-migration.md) §M1 — Stream B Layer 0 introduction (AMODE/BSEL/PSEL CSRs)
- [`doc/wyrd-integration.md`](../wyrd-integration.md) v0.2 — Gearbox surface contract
- [`doc/wyrd-substrate-guarantees.md`](../wyrd-substrate-guarantees.md) §3, §5 — current risks + post-Walk-α audit boundary
- [`../../BMA/theory/hypergraph-inference/BMA-Theory-Addendum-18_0-Hypergraph-Access-Pattern.md`](../../../BMA/theory/hypergraph-inference/BMA-Theory-Addendum-18_0-Hypergraph-Access-Pattern.md) §3 (parallel cognitive registers), §4 (Seams), §9 (open questions)
- [`../../BMA/theory/hypergraph-inference/P12-Seam-Threshold-Formalization.md`](../../../BMA/theory/hypergraph-inference/P12-Seam-Threshold-Formalization.md) — Gemini's per-tier K formalisation
- `~/Documents/go-coding-guide.md` — workspace-wide Go conventions (mandatory for implementation PRs)

---

*Authored 2026-05-14 by `qbp-cu-implementor`. §I4 status: Proposed; awaiting reviewer signoff before implementation PRs open.*
