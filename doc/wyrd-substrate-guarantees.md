# Wyrd Substrate Guarantees

**Status:** Initial cut, 2026-05-14. Tracks what `emulator/` (the Gearbox surface in
`emulator/public_api.go`) guarantees Wyrd consumers can rely on, what is measured today,
and the known-risk surfaces with mitigation paths.

**Audience:** Wyrd consumers of `github.com/JamesPagetButler/qbp-compute-unit/emulator`.
Primary consumer at v0.1: `compute/quaternion.go::HamiltonProduct` +
`HamiltonProductHighPrec`. See [`doc/wyrd-integration.md`](wyrd-integration.md) v0.2 for
the API surface contract.

**Companion:** This doc complements `doc/wyrd-integration.md`. The integration doc says
*what* the surface looks like; this doc says *what behaviour is guaranteed* and *what
risk is unmitigated*.

---

## 1. Scope and version coupling

This guarantee surface applies to `emulator/v0.1.0-rc1` (and forward-compatible later
tags in the v0.1.x series). At M1 (Walk-α), the `Gearbox` surface gains CSR-bound state
(`qbp.amode`, `qbp.bsel`, `qbp.psel`) per [ADR-004](../architecture/adr-004-m1-gearbox-state-model.md).
The M1 additions are **additive** — pinned-against-rc1 Wyrd code continues to compile
and behave identically through M1. Breaking changes, if any, will move to v0.2.0 with
explicit migration notes.

The four-lens guarantee structure below covers v0.1.0-rc1. Post-M1 (Walk-α) is treated
separately in §5 because M1 introduces new concurrency primitives that require their own
audit.

---

## 2. Four-lens guarantees at v0.1.0-rc1

### 2.1 Robust — substrate behaviour under stress

| Property | Guarantee | Evidence |
|---|---|---|
| Race-free under concurrent reads from `cpu.go` ISA path | Yes | `go test -race ./emulator/...` clean on every PR; `WatchdogDropCount` uses `atomic.AddUint64`; `Gearbox` slow-path snapshot/restore of `ActiveWidth` is internal and serialised |
| Authority chain integrity (Lean → ROM → asm) | Yes | `TestSIMDConstantsMatchROM` parses `emulator/qmath_constants.s` DATA blocks at test time and verifies byte-equivalence against `roms/octonion_signs.hex`; `make verify-roms` enforces ROM-hash consistency with `lean/QBP/Sedenion.lean` |
| No silent precision degradation | Yes | Width is a typed `int` parameter; `QMul64` rejects W128 operands at the type level; `QMulHighPrec` rejects fast-path widths via `ErrTierUnsupported` |
| Scope discipline on emulator/ edits | Yes | Multi-agent execution plan enforces hard scope-glob per dispatch; reviewer (`qbp-cu-implementor`) re-runs all gates independently before any PR opens; agent evidence archived at `reviews/agent-evidence/issue-N-attempt-M.txt` |
| Build-time linkage correctness | Yes | `qbp_lean_sign_x/y/z` and `qbp_lean_conj` symbols package-public in `emulator/qmath_constants.s`; both QW64 and QW128 asm kernels consume the same generated constants; build fails closed on any symbol drift |

### 2.2 Efficient — substrate throughput and resource discipline

| Property | Guarantee | Evidence |
|---|---|---|
| Zero allocations on hot paths | Yes | `QMul64`, `QAdd64`, `QRot64`, `QConj64`, `QNorm64`, `QMul128`, `QAdd128`, `QConj128`, `QNorm128`, `CMul64`, `CAdd64`, `CMul128` all report `0 B/op, 0 allocs/op` across 100-sample benches |
| FX-8350 Crawl-hardware throughput | Measured | QMul64 ~570 ns/op; QAdd64 ~560 ns/op; QMul128 ~600 ns/op; QROT128 ~835 ns/op (post-PR #24 medians, 10×3s benches) |
| Hot-path improvement from authority-chain consolidation | Measured | PR #24 yielded 5–10% median speed-up across all 10 ops (cache-locality win from RODATA consolidation) |
| Cross-Width allocation independence | Yes | `QMulHighPrec` slow path is the only Gearbox method that touches `*big.Float`; all fast-path methods are pure `[N]float64 → [N]float64` value passing |

### 2.3 Precise — numerical headroom available to Wyrd consumers

| Width | Carrier | Noise floor (per ‖v‖) | Algebraic lifetime at 1 GHz |
|---|---|---|---|
| W64 | `float64` × 4 | ~10⁻¹⁵ | ~7 seconds |
| W128 | double-double × 4 (hi×4 then lo×4) | ~10⁻³⁰ | ~172 days |
| W256+ | `math/big.Float` (slow path) | precision-parameterised | ~years (effectively unbounded for Crawl workloads) |

Wyrd's existing `HamiltonProduct(a, b model.Weight)` is implicitly fp64 — maps cleanly
to `QMul64`. Wyrd's `HamiltonProductHighPrec(a, b, prec uint)` maps to `QMulHighPrec(w,
a, b)` with `w` derived from `prec` (e.g. `prec ≥ 113 → W256`, `prec ≥ 53 → W128`).

**Width is the precision contract.** Wyrd consumers MUST address Width via the named
constants (`W8`, `W16`, … `W1024`); the underlying integer values are implementation
detail and may change without ABI break per [PR #27]'s amendment to `doc/wyrd-integration.md`
§3.1.

### 2.4 Accurate — mathematical correctness

| Property | Guarantee | Evidence |
|---|---|---|
| Hamilton product algebraic correctness | Yes | `TestDispatch_Equivalence` verifies AVX ↔ scalar fallback agreement within 1e-9 across 5 ops × 5 input cases (identity, i×j=k, 120° rotation, generic pair, large components); both paths derived from the same Cayley-Dickson sign-table |
| Sign-table provenance | Yes | `roms/octonion_signs.hex` (and the quaternion sub-table extracted from it) traces to `lean/QBP/Sedenion.lean` `mulSignData` definition; `make verify-roms` enforces SHA-256 manifest match |
| Hurwitz norm-multiplicativity (‖q·r‖ = ‖q‖·‖r‖) | Yes — by algebraic construction | Holds for any correct Hamilton product over ℍ; Cayley-Dickson preserves it at each doubling; verified empirically in `pkg/octonion.NormMultiplicativity` |
| Vendor-prefix conformance | Yes | All emulator symbols carry the `qbp_lean_` or `qbp.` prefix per `Ref/RISC-V-Policies-and-Best-Practices.md` §4 |

---

## 3. Known risk surfaces at v0.1.0-rc1

These are real risks that Wyrd consumers should know exist. None are blockers for the
HamiltonProduct + HamiltonProductHighPrec swap, but each has a mitigation path.

### 3.1 No canonical third-party RV-ISS cross-check

`TestDispatch_Equivalence` verifies AVX kernel agreement against the in-repo scalar
fallback. If both paths were wrong in the same way (a shared algebraic mistake in the
Cayley-Dickson derivation, for example), the cross-check would not catch it.

**Mitigation in flight:** Spike co-simulation + `riscv-arch-test` conformance suite is
tracked in issue [#18](https://github.com/JamesPagetButler/qbp-compute-unit/issues/18)
(M1+ scope). Once landed, kernel correctness will be verified against the canonical
RISC-V ISS rather than only against the in-repo fallback.

**Current floor:** byte-equivalence with Lean-derived ROM constants (proven via
`TestSIMDConstantsMatchROM`) + algebraic-property tests (Hurwitz norm preservation,
identity rotation, basis multiplication). Strong floor, but not third-party gold standard.

### 3.2 WDEvent emission cost — 5–8% overhead on three ops

Every Gearbox method call passes through `emulator/isa.go`'s `Step()` boundary which
emits a `WDEvent` to a buffered channel. Measured overhead on FX-8350:

| Op | Pre-WDEvent baseline | Post-WDEvent | Delta |
|---|---|---|---|
| QCONJ | 532 ns/op | 578 ns/op | +8.5% |
| QMUL128 | 571 ns/op | 618 ns/op | +8.3% |
| QADD128 | 536 ns/op | 570 ns/op | +6.3% |
| (other ops) | various | various | ≤5% |

This is a fixed cost Wyrd inherits on every Gearbox call. Tracked as `reviews/peer-review-006`
§7 Item 4; carved out as architecture mitigation for Walk-α (candidates: lock-free ring
buffer; sentinel-event mode; conditional emission).

**For Wyrd planning:** Hebbian co-activation + sandwich-conjugation paths each compose
several Gearbox calls per edge update. At 5–8% per call, a 4-call composition is
~25–35% slower than a hypothetical zero-tap baseline. The tap is necessary for the
WDEvent observer pattern at M1 (per ADR-003 §I3.4); the cost is structural to the
authority-chain design.

### 3.3 GCG ladder enforcement partially CI-gated

`go test -race`, `go vet`, `gofmt -l`, `make verify-roms` run on every PR via the
`Verify Lean ROMs` workflow on origin/main. `golangci-lint run` and `staticcheck` do
NOT run in CI today — they are local-discipline only.

**Mitigation in flight:** issue [#17](https://github.com/JamesPagetButler/qbp-compute-unit/issues/17)
adds the missing linters to CI as a separate workflow. Until that lands, agent-self-attestation
+ qbp-cu-implementor reviewer-discipline is the only gate. For Wyrd's swap PR (one-line
changes per function) this risk is minimal; for future Gearbox extensions it grows.

### 3.4 Cross-repo CI canary skipped until #15 (PATs) lands

The `wyrd-compatibility.yml` canary in this repo skips cleanly when `WYRD_PAT` is unset
(per PR #22's preflight gating). Until James resolves issue
[#15](https://github.com/JamesPagetButler/qbp-compute-unit/issues/15) (option A: both
repos public; option B: provision WYRD_PAT as repo secret), API drift between
QBP-CU `emulator/` and Wyrd `compute/` is detected only by Wyrd's local CI.

**For Wyrd planning:** treat your local CI `go test -race ./compute/...` as the
load-bearing signal until #15 resolves. Post-#15, the canary activates automatically
with no YAML change needed.

### 3.5 QW128 algebraic lifetime is finite

172-day lifetime at 1 GHz composition means ~160,000 composed Hamilton products before
machine-epsilon drift accumulates above 1e-6 of `‖v‖`. For Wyrd's sleep-cycle composition
depths and per-edge weight updates this is comfortably out of range; for sustained
multi-day workloads at composition rates over 100 Hz it becomes a real budget worth
tracking.

**Renormalisation contract:** Wyrd consumers can renormalise mid-composition by computing
`v ← v / ‖v‖` (the `QNorm128` method returns the squared norm as `[8]float64`; consumers
take √ and divide). The Gearbox does not auto-renormalise.

### 3.6 Octonion + sedenion paths return `ErrTierUnsupported` at Crawl

`OMul64`, `OAdd64`, `SMul64`, `SAdd64` are present on the Gearbox surface but return
the `ErrTierUnsupported` sentinel error in Crawl. These activate at M1+ (octonion via
`Xqbpoct`) and M2+ (sedenion via `Xqbpqec` ZDCHK).

**For Wyrd planning:** Wyrd's quaternion-tier consumption (`HamiltonProduct`) is fully
supported. Wyrd consumers wanting octonion or sedenion algebras today should fall
through to a software path on the consumer side and revisit when the Crawl-phase
tier-promotion lands.

### 3.7 Substrate-credibility-window is finite

Source: [`doc/design/m1-verification.md`](design/m1-verification.md) §3.6 (two-layer credibility model). Companion housekeeping tracker: [#37](https://github.com/JamesPagetButler/qbp-compute-unit/issues/37).

The L2 substrate-tier Lean promotion gate per [Spec 9.2 §3 mode (b)](../../inter/spec/BMA-Spec-Addendum-9_2-Federation-Lean-Promotion-Protocol.md) depends on a recent passing Tier B cosim run (riscv-arch-test ≥95% pass + Spike RV64I divergence-clean). The substrate-credibility-window (proposed 72h, matching BMA's 72h continuous-operation gate per Step 8) means an emulator commit older than the window cannot serve as L2 promotion substrate without re-running Tier B.

**Mitigation at v0.1.0-rc1:** Tier B is M1-era infrastructure (cosim PR cycle, gated on PR #35 implementation), not yet running. Until Tier B lands, L2 promotion is unavailable; consumers fall back to L1 base-ISA-only credibility plus manual Lean review. This is acceptable for the Crawl phase where no substrate-tier Lean theorems with executable extraction exist yet — there is nothing to promote through the L2 gate, so the window cannot bind.

**Promotion path at Walk-α:** Tier B runs nightly per `m1-verification.md` §5 PR 3 scope. The substrate-credibility-window becomes a live constraint at that point. The [Compute Manifest](../../inter/spec/BMA-Spec-Addendum-9_2-Federation-Lean-Promotion-Protocol.md) (Wyrd-owned per Spec 9.2 §4) records `LastPassingTierB` per Spec 9.2 §4; consumers consult it before invoking L2 promotion.

**Version note (per `@qbp-implementor` consultative F3):** `LastPassingTierB` is a **v0.2 manifest extension** landing per `repo-bma-systema-issue-#171`; the Compute Manifest v0.1 base (per `repo-wyrd-pr-#58` design surface) does NOT carry this field. Consumers consulting the manifest before #171 lands will read a zero-value field; treat zero-value as "no fresh credibility evidence available," which is functionally equivalent to the L2 gate blocking. The temporal ordering of substrate-tier promotion availability is therefore: Wyrd #58 (manifest v0.1 base) → this PR (v0.2 SeamEvent provenance + this risk surface) → bma-systema #171 (v0.2 manifest extension `last_passing_tier_b`) → m1.3 cosim PR cycle (impl-side `emulator/cosim/manifest.go` emission, tracked at [#37](https://github.com/JamesPagetButler/qbp-compute-unit/issues/37)).

---

## 4. What Wyrd consumers can rely on

Concrete contract for Wyrd PR #2 wiring:

1. **`gearbox.QMul64(a, b [4]float64) [4]float64`** is the swap target for `hamiltonProductQ64`. Pure value-type signature; zero allocations; equivalent to the scalar fallback to within 1e-9 (`TestDispatch_Equivalence`).

2. **`gearbox.QMulHighPrec(w Width, a, b [4]float64) ([4]float64, error)`** is the swap target for `HamiltonProductHighPrec`. Returns `ErrTierUnsupported` for fast-path widths (W8…W128); accepts W256/W512/W1024 with `math/big.Float`-backed computation. Snapshot/restore of internal `ActiveWidth` is invisible at the call site.

3. **No struct changes** required in `model.Weight`. The `Tier` field stays orthogonal to Width per [`doc/wyrd-integration.md`](wyrd-integration.md) §2 Q3.

4. **No Lean theorem update** required in `Wyrd.Foundations` or `Wyrd.Capability`. The algebraic-contract anchors cite the operation (Hamilton product over ℍ), not the implementation backend (the AVX kernel vs the inline pure-Go).

5. **API stability through the v0.1.x series.** Breaking changes move to v0.2.0 with explicit migration notes; ADR-004's M1 additions are guaranteed additive.

6. **L2 substrate-tier Lean promotion gate (Walk-α onwards; not active at v0.1.x).** Wyrd consumers needing substrate-tier Lean promotion per [Spec 9.2 §3 mode (b)](../../inter/spec/BMA-Spec-Addendum-9_2-Federation-Lean-Promotion-Protocol.md) MUST consult the Compute Manifest's `LastPassingTierB` field. If `time.Since(LastPassingTierB) > 72h`, the L2 promotion gate blocks until next green Tier B run (see §3.7). **v0.1.x consumers do not yet need this check** — no Tier B infrastructure is live; the promotion path activates at Walk-α with Tier B cosim activation. Tracked at [#37](https://github.com/JamesPagetButler/qbp-compute-unit/issues/37); manifest emission wiring follows in m1.3 cosim PR cycle.

---

## 5. Post-Walk-α (post-M1) guarantee surface — separate audit

Once M1 implementation lands (CSR-bound stateful Gearbox + QW8 peripheral surface +
goroutine-pair concurrent dispatch with `OnSeam(callback)` per ADR-004), the four-lens
audit re-runs against the new concurrency surface. Specifically:

- **Race-detector audit** of the goroutine-pair dispatch path (the new concurrency
  surface needs its own `-race` clean signal).
- **Throughput measurement** of concurrent peripheral + foveal execution under
  representative BMA-Walk workload (autonomic 10 Hz loop + sleep-cycle compaction).
- **QW8 peripheral precision validation** — confirm 8-bit carrier is sufficient
  for Seam detection per Gemini's P12 formalization (`τ = K · δ_precision · ‖v‖`,
  K=10 at QW8 per `~/Documents/BMA/theory/hypergraph-inference/P12-Seam-Threshold-Formalization.md`).
- **Cascadia-pipeline end-to-end validation** per A18 §7 (the first scoring-loop
  end-to-end demonstration of Stance × Locale × Scout × Scoring).

A revised version of this doc will publish at `v0.2.0-rc1` with the M1 surface audited.

---

## 6. How to cite this contract

For Wyrd-side PRs that consume this substrate:

```
// Gearbox guarantee: emulator/v0.1.0-rc1; see qbp-compute-unit/doc/wyrd-substrate-guarantees.md §2
// Algebraic-contract anchor: Wyrd.Foundations + Wyrd.Capability (operation-level cite, not backend)
```

For governance review under ADR-003 §I4, this doc is a citable artefact alongside
`doc/wyrd-integration.md` and the per-PR review evidence at `reviews/agent-evidence/`.

---

*Authored by qbp-cu-implementor (Claude Opus 4.7) — substrate-side honest assessment at
v0.1.0-rc1 lock. Companion to `doc/wyrd-integration.md` v0.2. Revision history below.*

## Revision history

| Version | Date | Changes |
|---|---|---|
| Initial | 2026-05-14 | First cut; four-lens guarantees at v0.1.0-rc1; six known risk surfaces; M1 audit deferred to v0.2.0-rc1 |
