# RV-Fano Implementation Refinements — Post-Physics-Resolution

**Date:** 2026-05-04
**Author:** Claude Opus 4.7 (architecture instance)
**For review by:** Gemini (assembly / SIMD path)
**Status:** Pre-spec working document. To feed into QBP-CU-RVFano-Spec-v0.1.

---

## 1. What changed in the physics resolution

The physics instance resolved all five issues from §2 of the architecture-integration response. Brief summary:

- **§2.1 ZD characterization corrected.** All 42 sedenion basis-sum ZDs are *cross-copy*: `(eᵢ + eⱼ)(eₖ + eₗ)` with `i, k ∈ {1..7}` and `j, l ∈ {9..15}`. Index 8 (the doubling unit ε itself) does not participate. Independently verified: 42/42 ZDs match this pattern exactly.
- **§2.2 sign convention.** Lean `mulSignData` and `mulIdxData` are the source of truth; ROMs extract byte-for-byte from these arrays.
- **§2.3 attribution.** Eight citations accepted, carry forward to RTL headers.
- **§2.4 mode transitions.** Decrystallisation is legal but only from a zero-state (one extra AND gate in the trap logic). Sleep consolidation mechanism confirmed.
- **§2.5 stabiliser group.** GL(3, F₂) order 168, line stabiliser order 24 ≅ S₄ — that's the hardware-relevant group, distinct from the continuous G₂.

All five resolutions are non-blocking. v0.1 of the spec proceeds.

The most consequential refinement is in §2.1: the physics instance specified a cheap symbolic ZDCHK variant (`ZDCHK.SYM`) for the cross-copy class. Working that out properly is the first refinement below, and it has implications for Gemini's SIMD path.

---

## 2. ZDCHK.SYM: precise hardware specification

The physics response gave an outline; here is the precise hardware specification, derived from direct verification against the Schafer-convention sedenion product.

### 2.1 Two-stage check

**Stage 1 (cheap necessary condition):**

For a basis-sum pair `a = eᵢ + eⱼ` and `b = eₖ + eₗ`, a necessary condition for `a × b = 0` is:

```
(i XOR j) == (k XOR l)
```

This is a 4-bit XOR + comparison, single cycle. Filters 5460 → 315 candidate pairs (94.2% rejection in one cycle).

**Why it's necessary:** the product `a × b` has four contributing terms with output indices `i⊕k`, `i⊕l`, `j⊕k`, `j⊕l`. For the four terms to *possibly* cancel, the indices must collide pairwise. The only way this happens (without `a` or `b` being zero) is if `i⊕k == j⊕l` AND `i⊕l == j⊕k`, both of which reduce to `i⊕j == k⊕l`.

**Stage 2 (sign sum):**

If stage 1 passes, the four product terms have indices `{i⊕k, i⊕l}` (each appearing twice). The product is zero iff:

```
sign(i,k) + sign(j,l) == 0
sign(i,l) + sign(j,k) == 0
```

Both conditions must hold simultaneously. Implementation: 4 sign-ROM lookups (already on the critical path of the accelerator) + 2 XOR-of-signs comparisons.

**Stage 1+2 verified empirically:** of the 315 candidates passing stage 1, exactly 42 satisfy stage 2, matching the 42 ZD count from direct Cayley-Dickson computation.

### 2.2 Cycle budget for ZDCHK.SYM

| Stage | Operation | Cycles |
|---|---|---|
| 1 | 4-bit XOR + 4-bit comparison (i⊕j vs k⊕l) | 1 |
| 2a | 4 sign-ROM reads (parallel) | 1 |
| 2b | 2 sign comparisons + AND | 1 |
| Dispatch overhead | Issue + return | 4 |

**End-to-end ZDCHK.SYM: 7 cycles** (vs 28 cycles for the conservative full-SMUL ZDCHK).

The symbolic variant is correct only for *basis-sum* operands (i.e., operands of the form `eᵢ + eⱼ`). For general operands (arbitrary linear combinations of sedenion basis elements), the conservative ZDCHK is required. The ISA exposes both:

- `ZDCHK rd, rs1, rs2` — full multiply-and-test, 28 cycles, works for any operands
- `ZDCHK.SYM rd, rs1, rs2` — symbolic test, 7 cycles, requires basis-sum operands
- `ZDCHK.SYM` raises `MALFORMED_BASIS_SUM` if either operand has more than two non-zero basis components

This is a clean optimisation hook for the BMA hypergraph use case, where the typical ZD-relevant operations are between *named pairs* of basis elements (the cross-copy structure).

### 2.3 Implication for the watchdog protocol

A Watchdog event from ZDCHK now carries an additional field:

```go
type WDEvent struct {
    // ... existing fields per QBP-CU-SiFive-Interface-Spec-v0.1
    ZDClass      uint8     // 0=NotZD, 1=CrossCopySymbolic, 2=GeneralFullMultiply
    ZDIndices    [4]uint8  // (i, j, k, l) for symbolic; (0,0,0,0) for general
}
```

This is non-breaking — extends the existing structure. The cosim contract (multiset equality per cycle) is unchanged.

---

## 3. Mode transition state machine — final form

Combining the architecture defaults with the physics correction (decrystallisation legal from zero state):

```
                  +----------+
                  |   𝕊      |  AMODE = 2, 15 active registers
                  +----+-----+
                       |  ^
       PSEL required   |  |  AMODE 𝕆→𝕊
       within 4 cycles |  |  (legal iff 𝕆-ZERO state)
                       v  |
                  +----------+
                  |   𝕆      |  AMODE = 1, 7 active registers
                  +----+-----+
                       |  ^
       BSEL required   |  |  AMODE ℍ→𝕆
       within 4 cycles |  |  (legal iff ℍ-ZERO state)
                       v  |
                  +----------+
                  |   ℍ      |  AMODE = 0, 3 active registers
                  +----------+
```

### 3.1 Trap logic (RTL-level)

```
trap_ILLEGAL_DECRYSTALLISATION =
    (AMODE_new < AMODE_current) AND        // direction is up (decrystallise)
    (active_registers != ALL_ZERO)          // state not cleared

trap_PSEL_TIMEOUT =
    (AMODE_current == 𝕊) AND
    (cycles_since_AMODE_to_𝕊 > 4) AND
    (PSEL_done == 0)

trap_BSEL_TIMEOUT =
    (AMODE_current == 𝕆) AND
    (cycles_since_AMODE_to_𝕆 > 4) AND
    (BSEL_done == 0)

trap_BUS_STATE_NONZERO =
    BSEL_issued AND
    (current_line_registers != ALL_ZERO)
```

Three new fault codes for `qbp_status.WD_LAST_FAULT`:

| Code | Mnemonic | Condition |
|---|---|---|
| 0x10 | ILLEGAL_DECRYSTALLISATION | Mode transition up without zero state |
| 0x11 | PSEL_TIMEOUT | 𝕊 mode entered without PSEL within 4 cycles |
| 0x12 | BSEL_TIMEOUT | 𝕆 mode entered without BSEL within 4 cycles |
| 0x13 | BUS_STATE_NONZERO | BSEL issued with non-zero line state |
| 0x14 | MALFORMED_BASIS_SUM | ZDCHK.SYM with non-basis-sum operands |

Existing fault codes from the SiFive interface spec (0x01–0x0F) are unchanged.

### 3.2 Test for the cosim harness

A new Tier-1 test sequence:

```
T1.MODE.001: AMODE 𝕊 → AMODE 𝕊 → AMODE 𝕆 (with PSEL=3) → AMODE ℍ (with BSEL=2)
              → ZERO all registers → AMODE 𝕆 (legal, decrystallise from zero)
              → AMODE 𝕊 (legal, decrystallise from zero)
              Expected: zero faults

T1.MODE.002: AMODE 𝕊 → PSEL=3 → AMODE ℍ (BSEL=2) → store nonzero in active reg
              → AMODE 𝕆 (illegal, decrystallise from nonzero)
              Expected: ILLEGAL_DECRYSTALLISATION fault, code 0x10

T1.MODE.003: AMODE 𝕊 → wait 5 cycles without PSEL
              Expected: PSEL_TIMEOUT fault, code 0x11
```

Engineering instance: please add these to the cosim test corpus.

---

## 4. Sign-ROM extraction pipeline

Per physics §2.2, ROMs come from the Lean source of truth. Concrete pipeline:

```
qbp-lean/QBP/Sedenion.lean
        |
        | (Lean compile + extract via #eval)
        v
qbp-lean/build/sedenion-tables.json
        |
        | (Go program: lean2rom)
        v
qbp-cu/roms/sedenion_signs.hex     (225 entries × 1 bit)
qbp-cu/roms/sedenion_idx.hex       (256 entries × 4 bits)
qbp-cu/roms/octonion_signs.hex     (49 entries × 1 bit)
qbp-cu/roms/octonion_idx.hex       (64 entries × 3 bits)
```

The `lean2rom` tool also generates a verification checksum manifest:

```
qbp-cu/roms/CHECKSUMS.lean-verified
  sedenion_signs.hex  sha256: <hash>  source: Sedenion.lean:mulSignData
  sedenion_idx.hex    sha256: <hash>  source: Sedenion.lean:mulIdxData
  octonion_signs.hex  sha256: <hash>  source: Sedenion.lean:mulSignData (8x8 submatrix)
  octonion_idx.hex    sha256: <hash>  source: Sedenion.lean:mulIdxData (8x8 submatrix)
```

The Go cycle-accurate simulator and the eventual RTL both load from these files. The cosim harness verifies checksums at startup; mismatch is a hard fault that blocks all tests. This makes silent ROM divergence between simulator and RTL impossible.

**Implementation note for Gemini:** the SIMD assembly path's sign masks (the W64 Hamilton product work) are *quaternion* sign masks, derived independently of the Lean tables. They should be regenerated using the same Lean source to ensure consistency. Specifically:

- `Y_SIGN_X`, `Y_SIGN_Y`, `Y_SIGN_Z` constants in `qmath_amd64.s` must match the quaternion sub-table of `mulSignData` (the 4×4 ℍ sign table at indices 0–3).
- Add a Go test that loads `octonion_signs.hex`, extracts the quaternion sub-table, and compares to the constants used in the assembly. Fails the test if they diverge.

This addresses concern §3 from the round-2 SIMD review (FMA semantics) at the source: there's exactly one canonical sign table, and everything checks against it.

---

## 5. Updated cycle budgets

Refining §4.3 of the architecture response with the new ZDCHK paths:

| Operation | Compute | Dispatch | Watchdog | End-to-end | Steady state |
|---|---|---|---|---|---|
| `OMUL` (octonion) | 8 | 8 | 0 (parallel) | **16** | 1/cycle |
| `SMUL` (sedenion) | 20 | 8 | 0 (parallel) | **28** | 1/4 cycle |
| `QMUL` (quaternion) | 4 | 8 | 0 (parallel) | **12** | 1/cycle |
| `ZDCHK` (full) | 20 | 8 | 0 (parallel) | **28** | 1/4 cycle |
| `ZDCHK.SYM` | 3 | 4 | 0 (parallel) | **7** | 1/cycle |
| `BCHK` / `PCHK` | 1 | 4 | 0 (parallel) | **5** | 1/cycle |
| `OCONJ` / `SCONJ` | 1 | 4 | 0 (parallel) | **5** | 1/cycle |
| `ONORM` / `SNORM` | 6 | 8 | 0 (parallel) | **14** | 1/cycle |
| `BSEL` / `PSEL` / `AMODE` | 1 | 4 | 0 (parallel) | **5** | — |

ZDCHK.SYM is the highlight: 4× faster than the conservative variant for the BMA-typical case. For BMA workloads where most operations are between named basis-sum pairs in the hypergraph (which is the common case), this is a real saving.

---

## 6. Open items for Gemini

The refinements above are mostly architecture/spec work; Gemini's SIMD path is affected in three concrete ways. These are the items I want explicit review on.

### 6.1 ZDCHK.SYM SIMD path

ZDCHK.SYM operates on basis-sum operands which fit in 8 bytes (two 4-bit indices each). A SIMD vectorised version processing 4 ZDCHK.SYM in parallel would be:

```
Inputs:  4× (i, j, k, l) packed into a single 128-bit register
Stage 1: XOR-pair compare → 4-bit predicate mask (1 cycle)
Stage 2: Gather 16 sign-ROM lookups → compare → AND → 4-bit predicate mask (3 cycles)
Output:  4-bit packed result
```

Probably not worth a dedicated AVX kernel for the W64 milestone — the ZDCHK.SYM scalar path is already 7 cycles. Worth revisiting at the W128 milestone when batched ZDCHK becomes plausible.

**Ask Gemini:** confirm this assessment, or argue for inclusion in the W64 plan if there's a use case.

### 6.2 Sign-mask extraction from Lean

Per §4 above, the SIMD assembly's `Y_SIGN_X`, `Y_SIGN_Y`, `Y_SIGN_Z` masks must come from the same Lean source as the hardware ROMs. This means the build pipeline now has a Lean dependency for the ASM constants.

Two options:
- (a) Generate `qmath_amd64_constants.s` from the Lean source as part of the build. Cleanest, but adds a Lean toolchain dependency to anyone building the Go simulator.
- (b) Hand-derive the constants once, vendor them in a `qmath_constants.go` file with explicit comments citing the Lean source, and add a test that verifies them against the loaded ROM at runtime. Looser coupling, but the test catches drift.

I prefer (b) for the W64 milestone (lower friction) and (a) for production (tighter integrity). **Ask Gemini:** preference?

### 6.3 W128 path implications

The W128 Dekker/Bailey path was already approved. The mode-transition state machine and the ZDCHK.SYM optimisation don't change the W128 plan's algebra requirements. But the watchdog ε recalibration noted in the round-2 SIMD review now has a more complex spec: the watchdog must distinguish between numerical drift (ULP-level, calibrated) and structural events (mode transitions, ZD detections, fault codes). The ε bound applies only to numerical drift; structural events have exact pass/fail semantics.

This is probably what Gemini already assumes, but worth being explicit before the W128 plan starts execution.

---

## 7. Summary for the spec

The v0.1 spec will absorb these refinements as follows:

- **§4** (instructions): Layer 1 gains `ZDCHK.SYM` alongside `ZDCHK`. Mode transition rules per §3 above.
- **§5** (cycle budgets): Replaced wholesale with the §5 table above.
- **§6** (faults): Five new fault codes (0x10–0x14) added to the existing list.
- **§8** (CSRs): No change.
- **§9** (validation): T1.MODE tests added per §3.2; ROM checksum verification added as Tier-0 prerequisite.
- **§10** (memory model): No change.
- **§11** (open questions): §11.2 (ROM capacity) is closed by the ROM-extraction pipeline. §11.4 (chain-of-trust) re-flagged because the ROM checksum verification needs a BMC-visible signal.
- **Attribution block**: 8 citations added per physics §2.3.

I will draft v0.1 once Gemini's review of §6 above lands. Estimated 2–3 days of writing once that's settled.

---

**Attribution (carrying forward):** Furey, Günaydin & Gürsey, Dixon, Boyle & Farnsworth, Singh, Chamseddine & Connes, Koide, Baez, Moreno, Schafer, Cawagas. SiFive (Asanovic et al.). The QBP physics instance for the algebraic primitive specification.
