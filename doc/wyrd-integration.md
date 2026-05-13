# Wyrd Integration — Public API Surface for QBP-CU Consumers

**Date:** 2026-05-05
**Status:** Draft v0.2 — Q1/Q2/Q3 architecture confirmed; six factual/design issues fixed per Wyrd review
**Companion:** Wyrd issue [#2](https://github.com/JamesPagetButler/wyrd/issues/2)
**Companion review:** Wyrd instance comment on [issue #10](https://github.com/JamesPagetButler/qbp-compute-unit/issues/10)

## Revision history

| Version | Date | Changes |
|---|---|---|
| v0.1 | 2026-05-05 | Initial draft. Q1/Q2/Q3 proposals. |
| v0.2 | 2026-05-05 | Wyrd review incorporated: subdir import path; pseudo-version vs tag; typed-per-width signatures; SedenionResult struct; Width-iota convention documented; Lean source-of-truth pinned to this repo (option b); WYRD_PAT requirement noted; Wyrd PR #11 acknowledged; Walk-phase Precision field added. |

---

## 1. Problem Statement

Wyrd is the Quaternion-native typed hypergraph database that serves as the substrate for BMA, CTH, and Contextus. Its `compute/quaternion.go` already implements `HamiltonProduct` inline. As Wyrd operations scale (Hebbian co-activation update, sandwich-conjugation per `Capability.sandwich_mul`, edge-weight composition), the canonical path must dispatch to QBP-CU's hardware-accelerated kernels rather than re-implementing them.

The integration is **architecturally already specified** but **not yet wired** because of three coupled questions: module path, public API surface, and tier↔width dispatch semantics.

This document records the proposed answers and the resulting interface design.

---

## 2. The Three Coupled Questions

### Q1 — Module path

**Decision: Wyrd imports `github.com/JamesPagetButler/qbp-compute-unit/emulator`.**

| Option | Verdict |
|---|---|
| (a) Wyrd imports the canonical repo path | ✅ **Chosen.** No drift, matches actual repo URL. Costs Wyrd one-line change. |
| (b) Repo aliases as `qbp-emulator` | ❌ Adds module-name maintenance burden; Go modules don't support clean aliases without separate go.mod files. |
| (c) `replace` directive in Wyrd's go.mod | ❌ Workable for local dev but fragile in CI; replace directives can mask version drift. |

**Migration path:** Wyrd updates its go.mod to add `require github.com/JamesPagetButler/qbp-compute-unit/emulator <version>` (note the `/emulator` subdirectory — the emulator has its own `go.mod` so the import path includes it). Wyrd updates imports from `qbp-emulator` to `qbp-compute-unit/emulator`. Single PR.

**Versioning:** the `emulator/` module currently has no released tag. Two options:

| Option | Approach | Wyrd impact |
|---|---|---|
| **Tag upstream first** ✅ | Tag this repo's `emulator/` as `emulator/v0.1.0`, release notes documenting the Gearbox surface | Wyrd pins to a real semver tag |
| Pseudo-version | Wyrd's go.mod points at a Git SHA (e.g., `v0.0.0-20260505-abc123`) | Workable but discouraged for non-experimental use |

Recommendation: **tag upstream as `emulator/v0.1.0` once the §3 Gearbox surface is implemented and `TestSIMDConstantsMatchROM` passes**. Wyrd then references the real tag, no pseudo-versions. The first tag waits on M0.1 (#7) closing — Lean-derived constants must be in place before the surface is "stable enough to depend on."

### Q2 — Public API surface

**Decision: Gearbox at Crawl, Accelerator at Walk.**

| Option | API shape | Phase |
|---|---|---|
| A | `emulator.QMul64(a, b)` (free function) | Rejected — too brittle; no surface for adding mode/state |
| **B** | `emulator.Gearbox.Mul(a, b, width)` | **Crawl (M0–M1)** — small, stable, mode-aware in M1 |
| **C** | `emulator.NewAccelerator().Submit(req)` | **Walk-α (M1)** — full Accelerator interface from QBP-CU-SiFive-Interface-Spec §7 |

The migration from B to C is non-breaking: B becomes a thin wrapper around C. Wyrd consumers can choose either based on their needs.

### Q3 — Tier ↔ Width dispatch semantics

**Decision: tier and width are orthogonal axes. Wyrd dispatches algebra; Gearbox dispatches precision.**

This is the load-bearing lesson from peer-review-005 §5.3: "QW128 is QW64 with more precision" is a category mistake. Width is precision; algebra (Tier) is mode.

**Dispatch rule (Wyrd side):**

```go
func (g *Gearbox) HamiltonProduct(a, b model.Weight) (model.Weight, error) {
    if a.Tier != b.Tier {
        return model.Weight{}, errMixedTiers(a.Tier, b.Tier)
    }
    width := tierToWidth(a.Tier, a.Precision)  // pure function: tier × precision → width
    switch a.Tier {
    case model.TierComplex:    return g.gearbox.CMul(width, a, b)
    case model.TierQuaternion: return g.gearbox.QMul(width, a, b)
    case model.TierOctonion:   return g.gearbox.OMul(width, a, b)  // M1+: requires Xqbpoct
    case model.TierSedenion:   return g.gearbox.SMul(width, a, b)  // M2+: requires Xqbpqec / ZDCHK
    default:                   return model.Weight{}, errInvalidTier(a.Tier)
    }
}
```

**`tierToWidth` mapping (Crawl-phase):**

| Tier | Default Precision | Width |
|---|---|---|
| TierComplex | Standard | W64 (2× float64) |
| TierComplex | High | W128 (2× double-double) |
| TierQuaternion | Standard | W64 (4× float64) |
| TierQuaternion | High | W128 (4× double-double) |
| TierOctonion | Standard | W128 (8× float64) — software only at Crawl |
| TierOctonion | High | W256 (8× double-double) — software only |
| TierSedenion | Standard | W256 (16× float64) — software only |

---

## 3. Proposed Public API

### 3.1 The `Gearbox` type (Crawl interface, B above)

File: `emulator/public_api.go`

**Design correction (per Wyrd review):** earlier draft used a single `[8]float64` shape across widths, which silently relies on a layout convention (`hi×4 | lo×4` for QW128 double-double). That is surprising. **This v0.2 splits the API into typed-per-width signatures** — each width gets its own array type, layout is self-documenting, and double-double internals are not exposed in the public surface.

```go
// Package emulator provides hardware-accelerated quaternion-algebra
// kernels used by the QBP-Node compute mesh. Consumers should import
// the Gearbox type for stable Crawl-phase access.
package emulator

// Width is the precision selector (matches funct3 in the QBP RISC-V ISA).
//
// Note on numbering: Width values are the component bit-counts
// (W8 = 8 … W1024 = 1024), matching the existing emulator/qword.go
// definition consumed by cpu.go's ISA execution path. Consumers MUST
// address Width via named constants; the underlying integer values are
// implementation detail and may change without ABI break.
type Width int

const (
    W8    Width = 8    // 32-bit packed: 4 × int8
    W16   Width = 16   // 64-bit packed: 4 × int16
    W32   Width = 32   // 128-bit packed: 4 × float32
    W64   Width = 64   // 256-bit packed: 4 × float64
    W128  Width = 128  // 512-bit packed: 4 × double-double (8 × float64 internally)
    W256  Width = 256  // 1024-bit packed: software fallback (math/big.Float)
    W512  Width = 512  // 2048-bit: software fallback
    W1024 Width = 1024 // 4096-bit: software fallback
)

// Gearbox is the precision-and-algebra dispatcher. Construct with NewGearbox.
// At Crawl phase, Gearbox is stateless. At M1 (Walk-α), Gearbox gains
// CSR-backed mode state (qbp.amode, qbp.bsel, qbp.psel) per the
// Stream B migration plan; existing API remains stable.
type Gearbox struct {
    // unexported state; M1+ adds AMODE/BSEL/PSEL CSR fields
}

// NewGearbox returns a stateless Gearbox suitable for Crawl-phase use.
func NewGearbox() *Gearbox { return &Gearbox{} }

// --- Quaternion (ℍ) operations, typed per width -----------------------

// QMul64 computes the Hamilton product a · b at QW64 precision.
// On AVX-FMA hosts, dispatches to qmul64AVX (~525 ns/op on FX-8350).
// On other hosts, dispatches to the scalar fallback.
func (g *Gearbox) QMul64(a, b [4]float64) [4]float64

func (g *Gearbox) QAdd64(a, b [4]float64) [4]float64
func (g *Gearbox) QRot64(q, v [4]float64) [4]float64
func (g *Gearbox) QConj64(a [4]float64) [4]float64
func (g *Gearbox) QNorm64(a [4]float64) float64

// QMul128 computes the Hamilton product a · b at QW128 (double-double)
// precision. The [8]float64 layout is "hi×4 then lo×4": indices 0..3 are
// the high components (W, X, Y, Z), indices 4..7 are the low components.
// This layout is internal to the QW128 representation and matches the
// emulator's qmath_128_amd64.s convention.
func (g *Gearbox) QMul128(a, b [8]float64) [8]float64

func (g *Gearbox) QAdd128(a, b [8]float64) [8]float64
func (g *Gearbox) QRot128(q, v [8]float64) [8]float64
func (g *Gearbox) QConj128(a [8]float64) [8]float64
func (g *Gearbox) QNorm128(a [8]float64) [8]float64

// QMulHighPrec is the software fallback for W256, W512, W1024 widths,
// using math/big.Float internally. Inputs are 4-component float64
// approximations; the function rounds high-precision intermediates back
// to float64 outputs. Use only for verification or correctness baselines;
// performance is ~1400 ns/op or worse.
func (g *Gearbox) QMulHighPrec(w Width, a, b [4]float64) ([4]float64, error)

// --- Complex (ℂ) operations -------------------------------------------

func (g *Gearbox) CMul64(a, b [2]float64) [2]float64
func (g *Gearbox) CAdd64(a, b [2]float64) [2]float64
func (g *Gearbox) CMul128(a, b [4]float64) [4]float64

// --- Octonion (𝕆) operations -- M1+ via Xqbpoct ----------------------

// Crawl-phase: software only via OMulScalar. Returns a TierUnsupported
// error if invoked before Xqbpoct extension is enabled.
func (g *Gearbox) OMul64(a, b [8]float64) ([8]float64, error)
func (g *Gearbox) OAdd64(a, b [8]float64) ([8]float64, error)

// --- Sedenion (𝕊) operations -- M2+ via Xqbpqec / ZDCHK --------------

// SedenionResult bundles the multiply result with the zero-divisor flag.
// Sedenions admit 42 cross-copy basis-sum zero-divisors; SMul reports
// when an operation hit one of them so callers can handle 0/0 hazards
// rather than propagating silently.
//
// Per RV-Fano-Implementation-Refinements §2: ZDClass values are
//   0 = NotZD
//   1 = CrossCopySymbolic (caught by ZDCHK.SYM, indices in ZDIndices)
//   2 = GeneralFullMultiply (caught by full ZDCHK)
type SedenionResult struct {
    Value     [16]float64
    ZDClass   uint8
    ZDIndices [4]uint8
}

func (g *Gearbox) SMul64(a, b [16]float64) (SedenionResult, error)
func (g *Gearbox) SAdd64(a, b [16]float64) ([16]float64, error)
```

**Why split signatures, not a single shape parameter:**

1. Self-documenting layout. `[4]float64` for QW64 means 4 components; `[8]float64` for QW128 means hi-then-lo double-double. No comment-readers needed.
2. Compile-time width safety. Passing a QW64 array to a QW128 method is a type error, not a runtime mismatch.
3. Matches existing `qmath_amd64.go` exposure. The file already exports `qmul64AVX(*QW64, *QW64, *QW64)` and `qmul128AVX(*QW128, *QW128, *QW128)` with distinct types — the public surface inherits this discipline.
4. Higher-width software-fallback consolidation under `QMulHighPrec(width, ...)` keeps the precision-as-parameter pattern available where it makes sense (uncommon software-only paths) without contaminating the hot path.

### 3.2 The `Accelerator` interface (Walk-α target, C above)

File: `emulator/accelerator.go` (per QBP-CU-SiFive-Interface-Spec-v0.1 §7)

```go
type Accelerator interface {
    Submit(r Req)
    Poll() (Resp, bool)
    WatchdogChan() <-chan WDEvent
    Tick(cycle uint64)
}
```

This interface is **already specified** but not yet implemented. Wyrd consumers preferring the Accelerator surface use `NewAccelerator(mode AcceleratorMode)` where `mode ∈ {Mock, Golden, RTLShim}`.

### 3.3 Backward-compatible migration M1 → M2

When AMODE / BSEL / PSEL CSRs land in M1, the Gearbox API gains optional configuration:

```go
// M1+ addition: explicit mode setting
func (g *Gearbox) SetMode(mode AlgebraMode) error  // H, O, or S
func (g *Gearbox) SetFanoLine(idx uint8) error     // 0-6, BSEL
func (g *Gearbox) SetProjection(idx uint8) error   // 0-7, PSEL

// Existing methods remain identical-behavior. AMODE defaults to H.
```

Wyrd code calling `gearbox.QMul(W64, a, b)` continues to work unchanged. Wyrd code wanting Stream B mode-awareness adds `gearbox.SetMode(emulator.AlgebraModeQuaternion)` before the call.

---

## 4. Soundness Anchor

**Correction (per Wyrd review):** earlier draft asserted that `qbp-lean` exists as a separate repo and that `Sedenion.lean` is already on disk. Neither is true today. This section is reframed as a "once #7 lands" forward statement, with the source-of-truth location explicitly pinned.

### 4.1 Source-of-truth location (decision)

**Decision: Lean ISA-side corpus lives in this repo (qbp-compute-unit) at `lean/QBP/`.** Wyrd keeps its existing `lean/Wyrd/Foundations.lean` witness theorems unchanged. Cross-repo CI verifies agreement.

Three options were considered:

| Option | Approach | Verdict |
|---|---|---|
| (a) Separate `qbp-lean` repo | Lean tables in their own repo, both qbp-compute-unit and Wyrd depend on it | Rejected — adds a third repo to maintain; no clear separation-of-concerns benefit; more places for paths to drift |
| **(b) qbp-compute-unit owns ISA-side Lean** ✅ | `lean/QBP/Sedenion.lean` lives here; `mulSignData` and `mulIdxData` are the source of truth; cross-repo CI checks Wyrd's witness theorems against this | **Chosen.** ISA Lean lives where ISA spec lives. |
| (c) Wyrd owns Lean | Wyrd's `lean/` directory hosts the ISA tables too | Rejected — inverts the dependency direction (Wyrd depends on QBP-CU, not the reverse) |

Option (b) was Wyrd's recommendation in the review of v0.1.

### 4.2 What the Gearbox preserves

Once the M0.1 (`lean2rom`, [#7](https://github.com/JamesPagetButler/qbp-compute-unit/issues/7)) deliverable lands:

1. **Single source of truth.** `qbp-compute-unit/lean/QBP/Sedenion.lean` defines `mulSignData` and `mulIdxData`. The `lean2rom` build step extracts ROM tables from these arrays at build time.
2. **CI-enforced agreement.** `TestSIMDConstantsMatchROM` (also in [#7](https://github.com/JamesPagetButler/qbp-compute-unit/issues/7)) compares the AVX kernels' sign masks against the Lean-derived constants. Drift is detected at CI time, not runtime.
3. **WDEvent observability.** The watchdog stream ([#8](https://github.com/JamesPagetButler/qbp-compute-unit/issues/8)) lets Wyrd-side consumers subscribe to algebraic anomalies, tying runtime behavior to the watchdog contract from QBP-Compute-Unit-Architecture-v1.0 §3.
4. **Cross-repo verification.** Wyrd's `lean/Wyrd/Foundations.lean` retains its independent witness theorems (e.g., `Wyrd.Capability.sandwich_mul`). A canary CI job (§5.1) builds Wyrd against this repo and runs Wyrd's Lean checks. Disagreement between Lean corpora becomes a CI failure rather than a silent runtime divergence.

### 4.3 Forward target (Phase 5)

**Phase 5 ISA semantics** ([Wyrd issue #5](https://github.com/JamesPagetButler/wyrd/issues/5)) will land an ε-tolerance theorem specifying the runtime behaviour at each Width, including the QW128 DD path's renormalisation guarantees. The Gearbox API surface is designed to be the target of that theorem — type-safe, width-explicit, and provenance-traceable through `lean2rom`.

Until M0.1 lands, soundness is asserted by **inline algebraic identity tests** (the Tier-0 corpus per QBP-Node Spec Part 2 §2.5.1). This is sufficient for Crawl but does not yet match the Lean-as-authority discipline that M0 establishes.

---

## 5. Testing Story

### 5.1 Cross-repo build verification

A canary CI job in this repo:

```yaml
# .github/workflows/wyrd-compatibility.yml
name: Wyrd Compatibility Check
on: [push, pull_request]
jobs:
  build-wyrd-against-emulator:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          path: qbp-compute-unit
      - uses: actions/checkout@v4
        with:
          repository: JamesPagetButler/wyrd
          token: ${{ secrets.WYRD_PAT }}    # required if Wyrd is private
          path: wyrd
      - run: |
          cd wyrd
          go mod edit -replace=github.com/JamesPagetButler/qbp-compute-unit/emulator=../qbp-compute-unit/emulator
          go test ./compute/...
```

Detects any breaking change to the public API before it hits Wyrd users.

**Auth requirement:** if Wyrd remains private, this canary needs a `WYRD_PAT` repository secret (Personal Access Token with read access to `JamesPagetButler/wyrd`). This is the reciprocal of the issue Wyrd just hit setting up its own canary against this repo — both directions need cross-repo PATs until both are public. Track as part of #10 acceptance criteria.

### 5.2 Round-trip property test (Wyrd-side, already specified)

Wyrd issue #4 (BMA hypergraph adapter) defines:

```go
// Round-trip property test: ToEngram(FromEngram(e)) ≡ e
```

This applies to QBP-CU dispatch as well: `Gearbox.QMul(W64, a, b)` must equal Wyrd's prior inline `hamiltonProductQ64(a, b)` to the same precision. The compatibility check above enforces this.

### 5.3 Numerical agreement test (cross-tier)

```go
// At W64, hamiltonProductQ64 and HamiltonProductHighPrec(prec=53) must agree
// At W128, qmul128AVX and HamiltonProductHighPrec(prec=113) must agree
//   to within Dekker double-double error bound (~2^-104)
```

Already in Wyrd's `compute/quaternion_test.go` as `TestHamiltonProduct_Q64AgreesWithHighPrec`. QBP-CU side should add the dual.

---

## 6a. Wyrd-Side State (per Wyrd review)

This v0.2 incorporates three pieces of Wyrd state the v0.1 draft missed.

### 6a.1 Wyrd PR #11 already merged

Wyrd's `compute/quaternion.go` already exposes the **structural** API:

```go
func HamiltonProduct(a, b model.Weight) (model.Weight, error)
func HamiltonProductHighPrec(a, b model.Weight, prec uint) (model.Weight, error)
```

Both currently use inline `math/big.Float`. **The Gearbox swap is therefore a one-line change** inside each function — no API churn on the Wyrd side, no cascading work for downstream BMA / CTH / Contextus consumers. This significantly de-risks the integration: v0.1 of this doc treated the Wyrd-side rewire as a substantial PR, but the Wyrd team has already absorbed the structural cost.

### 6a.2 Wyrd-side width-parameter decision

`HamiltonProduct(a, b model.Weight)` is currently implicitly fp64. Q3 (Tier↔Width orthogonality) implies Wyrd needs to decide *how* width gets selected. Three options the Wyrd review identified:

| Option | Approach | Phase suitability |
|---|---|---|
| **(i)** Keep implicit fp64 at Crawl | `HamiltonProduct` always uses W64; `HamiltonProductHighPrec(a, b, prec)` covers higher widths | **Crawl** — minimal Wyrd churn, matches today's Wyrd behavior |
| (ii) Add `Precision` field on `model.Weight` | Width travels with the operand; dispatch reads it | **Walk** — cleaner long-term but invasive at Crawl |
| (iii) Separate `WidthAware` API surface | New methods alongside existing | Rejected — fragmentation |

**Wyrd's lean: (i) at Crawl, (ii) at Walk.** Architecture instance agrees. **Decision recorded.** This means at Crawl, Wyrd's `HamiltonProduct` calls `gearbox.QMul64(a, b)` unconditionally, and `HamiltonProductHighPrec(a, b, prec)` calls `gearbox.QMulHighPrec(widthForPrec(prec), a, b)`. At Walk-α boundary, when `model.Weight` gains a `Precision` field, the Crawl behavior is recoverable as `Precision: PrecisionDefault → W64`.

### 6a.3 Wyrd-side concrete offers

Wyrd instance offered three deliverables. All accepted:

| Wyrd offer | Architecture instance accepts | Tracked in |
|---|---|---|
| Queue go.mod wire-up PR (Wyrd-side) | ✅ | Wyrd issue #2 |
| Draft Lean-source provenance integration | ✅ — coordinate with the engineering instance picking up M0.1 | qbp-compute-unit issue #7 |
| Draft canary YAML | ✅ — one less thing for the architecture instance to write | qbp-compute-unit issue #10 |

These offers reduce the M0 architecture-instance load and mean the Wyrd integration delivery doesn't require this repo's CI to be set up before Wyrd's PR can land. Good division of labor.

---

## 6. Open Items Tracked Elsewhere

| Item | Tracked in |
|---|---|
| Module-path coordination PR on Wyrd side | Wyrd issue #2 |
| Lean source provenance for sign tables | qbp-compute-unit issue #7 (M0.1 lean2rom) |
| WDEvent channel surface (passive in M0, active in M1) | qbp-compute-unit issue #8 (M0.2) |
| ISA stream reconciliation affecting Gearbox.SetMode semantics | qbp-compute-unit issue #4 (T1) |
| Mode-CSR-backed Gearbox (M1) | parent epic #3, phase M1 |

---

## 7. Estimated Cost

| Item | Effort |
|---|---|
| `emulator/public_api.go` Gearbox surface | 3 days |
| Cross-repo CI canary | 1 day |
| Coordination with Wyrd instance (PR, review, merge) | 2 days |
| Documentation (this doc + Wyrd-side docs) | 1 day |
| **Total** | **~1 week** |

This is parallel-safe with M0.1 (lean2rom) and M0.2 (WDEvent emission). Gemini and Wyrd instance can both contribute concurrently.

---

*Status: DRAFT | Tracks: issue #10 | Related: Wyrd issue #2*
