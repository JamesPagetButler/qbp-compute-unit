# Peer Review 005: Stream A → Stream B Migration Plan

**Date:** 2026-05-05
**Author:** Claude Opus 4.7 (architecture / red team instance)
**Subject:** Phased migration from `Xqbp*` flat ISA (Stream A v1.1) to mode-aware RV-Fano Layer 0/1/2 (Stream B), preserving working artifacts and incorporating older memory/compute research as gated promotions.
**Status:** Draft v0.1 — working architectural plan
**Companion documents:**
- [`Ref/RISC-V-Policies-and-Best-Practices.md`](../Ref/RISC-V-Policies-and-Best-Practices.md)
- [`Ref/SiFive-Documentation-Patterns.md`](../Ref/SiFive-Documentation-Patterns.md)
- [`spec/QBP-RISCV-ISA-Spec-v1.1.md`](../spec/QBP-RISCV-ISA-Spec-v1.1.md) — Stream A v1.1
- [`archive_build/docs/RV-Fano-Implementation-Refinements.md`](../archive_build/docs/RV-Fano-Implementation-Refinements.md) — Stream B authoritative
- [`Archive/QBP-Node-Spec-v0.1-Parts-0-and-1.md`](../Archive/QBP-Node-Spec-v0.1-Parts-0-and-1.md) — Phasing/deferred-decisions philosophy
- [`Archive/QBP-Node-Spec-v0.1-Part-2.md`](../Archive/QBP-Node-Spec-v0.1-Part-2.md) — Crawl-phase deliverable inventory
- [`architecture/peer-review-002-fano-mesh-isa-redteam.md`](peer-review-002-fano-mesh-isa-redteam.md) — Conventions audit
- [`architecture/peer-review-003-qbp-node-spec-crawl.md`](peer-review-003-qbp-node-spec-crawl.md) — Tensions T1/T2/T3

---

## 1. Executive Summary

Stream A (the `Xqbp*` vendor-prefix-conformant flat ISA at v1.1) and Stream B (the mode-aware RV-Fano Layer 0/1/2 ISA captured in `archive_build/docs/RV-Fano-Implementation-Refinements.md`) are **two views of the same machine in progress**, not competing alternatives.

Stream A is the surface form a RISC-V toolchain needs to see. Stream B is the underlying machine model the QBP physics actually requires. The migration question is not "which wins" but "how do they merge while preserving working artifacts."

**Verdict:** A four-phase migration (M0 → M3) that:

- **Preserves** Gemini's QW64 / QW128 AVX-FMA kernels unchanged across all phases
- **Adds** Stream B's mode-awareness (AMODE, BSEL, PSEL), zero-divisor detection (ZDCHK / ZDCHK.SYM), and Lean-as-authority sign tables
- **Re-enters** older memory/compute research (CAM-style QNEAR, table-driven QPERM, compositional Layer 0/1/2) only via promotion gates with positive evidence
- **Defers** Run-phase silicon decisions (dual-domain vs unified, mask-burned vs field-loadable ROM) to Walk-β analysis

Net cost: M0 ≈ 3 weeks parallel-safe work. M1 ≈ 2 months. M2 ≈ 4–6 months (overlaps Walk-α). M3 is a Walk-β documentation task.

Net benefit: Stream B silicon spec is ready when Run-α opens. No flag day. Authority chain prevents recurrence of the duplicate-symbol class of bug.

---

## 2. Why this migration is needed

### 2.1 What Stream A v1.1 has

- Vendor-prefix-conformant mnemonics (`qbp.qmul.w`, `qbp.qrot.w`, etc.)
- Three extension families declared (`Xqbpquat`, `Xqbpqec`, `Xqbpmem`)
- Trap-behavior section, revision history, mcause discipline
- **Working AVX-FMA kernels** at QW64 and QW128 (verified 2026-05-05: all `TestDispatch_Equivalence` and `TestDispatch_Equivalence128` passing on FX-8350; benchmarks at 523–815 ns/op with zero allocations)
- An upstream-toolchain integration path

### 2.2 What Stream A v1.1 lacks

- Algebraic mode awareness — every instruction implicitly assumes ℍ
- Zero-divisor detection — sedenion ops are dangerous without ZDCHK / ZDCHK.SYM
- Fano-line addressing as ISA primitive — BSEL / PSEL are absent
- Lean-as-authority for sign tables — masks are hand-derived
- Watchdog event semantics tied to mode — `WDEvent.AlgebraID` cannot be populated cheaply
- Layer 0 / 1 / 2 compositional structure — every new compound op needs a new opcode
- A scaling story past QW128 — the per-width register file does not scale to W256+

### 2.3 What Stream B has that addresses the gaps

Per `archive_build/docs/RV-Fano-Implementation-Refinements.md`:

- **Mode transition state machine** with five fault codes (0x10–0x14): `ILLEGAL_DECRYSTALLISATION`, `PSEL_TIMEOUT`, `BSEL_TIMEOUT`, `BUS_STATE_NONZERO`, `MALFORMED_BASIS_SUM`
- **`ZDCHK.SYM` two-stage check**: 4-bit XOR filter (1 cycle) + 4 sign-ROM lookups + 2 comparisons (3 cycles) — 7 cycles end-to-end vs 28 for full ZDCHK
- **42 cross-copy basis-sum ZDs** identified, indexed `(i,j,k,l)` with `i,k ∈ {1..7}, j,l ∈ {9..15}` — empirically verified 42/42 match
- **Lean → ROM extraction pipeline** — `mulSignData` and `mulIdxData` are the source of truth; ROMs extract byte-for-byte
- **`WDEvent.ZDClass` and `ZDIndices[4]`** — non-breaking extension for ZD-aware watchdog
- **Mode-aware cycle budgets** — `OMUL` 16 cycles, `SMUL` 28, `QMUL` 12, `ZDCHK.SYM` 7

### 2.4 What Stream B has that comes from older research

Older memory/compute unit work re-enters Stream B as candidate Layer 1 primitives, **gated by promotion evidence** (per QBP-Node Spec §0.4.1):

| Older research | Layer 1 candidate | Promotion gate |
|---|---|---|
| Content-addressable memory (1990s associative-recall designs) | `qbp.qnear` | CIM Level-1 demonstrates clean mapping |
| Table-driven Fano permutation engines | `qbp.qperm` / `qbp.qpermr` | Always promotes — surfaces existing FANO ROM |
| Canonical-form encode/decode round-trips | `qbp.qdec` / `qbp.qrec` | Tier-0 round-trip identity holds |
| Compositional Layer 0 / 1 / 2 hierarchy | M2/M3 ISA structure | Always — this *is* the philosophy |
| Watchdog tap on every operation | M0 passive emit, M1 active populate | QBP-CU-Architecture §3 contract requires |
| Holographic / Fano-redundancy storage | `qbp.bsel` / `qbp.psel` CSRs | M1 introduction |

The older research is **not preserved as-is** — it is preserved as a candidate-pool. Default position is deferral; promotion requires evidence.

---

## 3. The Bridging Principle

> **Stream A primitives are Stream B Layer 0 in disguise.**

The QW64 and QW128 AVX-FMA kernels Gemini delivered already implement Hamilton multiply, conjugate, norm, addition, and quaternion-vector rotation. Stream B's Layer 0 names these primitives differently and parameterizes them by AMODE — but the kernel asm doesn't change. Only the dispatcher and the surrounding state do.

| Stream A v1.1 mnemonic | Stream B Layer 0 role | Mode requirement |
|---|---|---|
| `qbp.qmul.w` | `MUL` | AMODE=H |
| `qbp.qrot.w` | `ROT` (composed: 2× MUL + CONJ) | AMODE=H |
| `qbp.qadd.w` | `ADD` | AMODE=H |
| `qbp.qconj.w` | `CONJ` | AMODE=H |
| `qbp.qnorm.w` | `NORM` | AMODE=H |
| `qbp.omac.w` | `MAC` | AMODE=O |
| `qbp.fano` | `LUT` | AMODE=O (FANO ROM lookup) |
| `qbp.pauli`, `qbp.synd`, `qbp.stab` | `Layer 1 QEC` | algebra-orthogonal |

This means Stream B Layer 0 introduction at M1 reuses every kernel byte Gemini just wrote. **No re-implementation.** Only new CSRs and a new dispatcher branch.

---

## 4. Four-Phase Migration with Hard Gates

### Phase M0 — Authority Chain Hardening (now → Crawl exit)

**Deliverables:**

| # | Item | Owner | Estimate |
|---|------|-------|----------|
| M0.1 | `lean2rom` build pipeline per `RV-Fano-Implementation-Refinements.md` §4 | Engineering instance | ~2 weeks |
| M0.2 | All sign masks in `emulator/qmath_amd64.s` and `qmath_128_amd64.s` regenerated from `Sedenion.lean` | Engineering instance + Gemini | ~3 days |
| M0.3 | CI test `TestSIMDConstantsMatchROM` enforcing parity | Engineering instance | ~1 day |
| M0.4 | `WDEvent` emission added to existing AVX/scalar kernels (passive — events go to a channel, not yet consumed) | Gemini | ~1 week |
| M0.5 | Stub `Xqbpoct` v0.1 and `Xqbpvcp` v0.1 spec docs (placeholder) | Architecture | ~1 day |

**Why this is M0:** the `qmath_128_amd64.s` duplicate-symbol bug taught the lesson — hand-derived implementation choices propagate. Move authority to Lean **before** adding new vocabulary.

**Gate to M1:**
- All ROMs regenerated from Lean, zero hand-derived constants in asm
- WDEvent stream consumed by at least one downstream observer (initially the test harness counting events)
- T1 (ISA-fork tension from peer-review-003) resolved: Stream A v1.x is the surface form; Stream B opens at v2.0

**What stays the same:** All Gemini-delivered kernels. Tests still pass. Performance unchanged.

**What changes:** Authority chain. Where masks come from.

---

### Phase M1 — Stream B Layer 0 Wraps Stream A (Crawl exit → Walk-α)

**Deliverables:**

| # | Item | Owner | Estimate |
|---|------|-------|----------|
| M1.1 | Three new CSRs: `qbp.amode` (algebra mode), `qbp.bsel` (Fano line), `qbp.psel` (projection) | Architecture + Engineering | ~2 weeks |
| M1.2 | Defaults: `AMODE=H, BSEL=0, PSEL=0` — Stream A code unchanged | Architecture | spec only |
| M1.3 | New v2.0 ISA spec: `Xqbpmode` extension — defines AMODE state machine, transitions, illegal-mode trap codes (0x10–0x14) | Architecture | ~1 week |
| M1.4 | Existing `qbp.qmul.w` etc. become **mode-aware**: dispatch on AMODE | Engineering | ~3 weeks |
| M1.5 | Rich WDEvent populated from CSR state | Gemini | ~1 week |
| M1.6 | Cosim Tier-1 corpus updated with `T1.MODE.001`–`T1.MODE.005` | Engineering | ~2 weeks |

**Why this works:** existing Stream A code still compiles, links, runs identically. New code that sets `qbp.amode` before invoking the kernels gets Stream B semantics. **No flag day.**

**Gate to M2:**
- All Tier-0 algebraic-identity tests pass under both Stream-A invocation pattern (no AMODE write) and Stream-B invocation pattern (explicit AMODE write)
- WDEvent multiset equivalence holds in both modes
- Mode-transition state machine has formally-verified safety invariants (Lean lemmas: "BSEL while bus state nonzero ⇒ trap")
- Context-switch correctness demonstrated: kernel preempts during AMODE 𝕊, restores cleanly

---

### Phase M2 — Stream B Layer 1 (Walk-α → Walk-β)

**Deliverables:**

| # | Item | Owner | Estimate |
|---|------|-------|----------|
| M2.1 | `Xqbpqec` Layer 1 instructions: `qbp.zdchk` (full, 28 cyc), `qbp.zdchk.sym` (basis-sum, 7 cyc), plus Pauli family | Engineering + Architecture | ~6 weeks |
| M2.2 | `Xqbpmem` Layer 1 ops modernized from older research: `qbp.qnear`, `qbp.qperm`, `qbp.qpermr`, `qbp.qdec`, `qbp.qrec` | Engineering | ~4 weeks |
| M2.3 | Hypergraph-native Layer 1 ops: `qbp.hedge.gather`, `qbp.hedge.scatter`, `qbp.conf.probe`, `qbp.recall.knn` | Engineering | ~6 weeks |
| M2.4 | Explicit `Xqbpoct` extension finalized (octonion ops as separate extension per peer-review-002 NF2) | Architecture | ~1 week |

**The older-research re-entry test:** QNEAR ships **only if** the CIM Level-1 emulator (per QBP-Node Part 2 §2.3.5) demonstrates clean algorithmic mapping for at least one BMA workload. Default position: ship without QNEAR if Level-1 reveals it doesn't fit.

**Gate to M3:**
- CIM Level-1 emulator results in: which Layer 1 memory primitives map cleanly, which don't
- Cosim Tier-1 corpus passes for every shipped Layer 1 op
- BMA inference inner loop benchmarks demonstrate Layer 1 ops outperform Stream A composition at ≥3×

---

### Phase M3 — Stream B Becomes Authoritative (Walk-β → Run-α)

**Deliverables:**

| # | Item | Owner | Estimate |
|---|------|-------|----------|
| M3.1 | v3.0 spec: Stream B is authoritative; Stream A is "v1/v2 surface compatibility form" | Architecture | ~2 weeks |
| M3.2 | Layer 2 composite operations defined as documented compositions (no new opcodes) | Architecture | ~1 week |
| M3.3 | `Xqbpvcp` extension finalized (coprocessor interface modeled on SiFive VCIX) | Architecture | ~2 weeks |
| M3.4 | Run-phase silicon target spec written against Stream B; Stream A backwards-compat documented | Architecture | ~3 weeks |

**Gate to Run-α opening:**
- Stream B silicon spec validated against the cycle-accurate simulator
- Branch B (dark-matter fork) headroom decision per QBP-Node App A.4
- Mask-vs-field-loadable sign ROM decision per App A.5
- Production T2 silicon BOM and power targets validated against measurements

---

## 5. Transition Invariants (must hold throughout M0–M3)

| Invariant | Why it matters | Gate that enforces |
|---|---|---|
| **Existing tests never regress.** Each phase advance includes prior phase's Tier-0/Tier-1 corpora green. | Otherwise migration breaks known-good behavior. | CI required-pass on every PR |
| **WDEvent multiset equivalence.** Same workload at any M-phase produces equivalent watchdog event streams. | Cross-phase equivalence is how cosim survives the migration. | Cosim harness requirement |
| **Authority chain only tightens.** Once Lean-derived, never hand-derived again. | Duplicate-symbol bug came from hand-derivation. Don't reintroduce. | M0.3 CI gate |
| **Stream A code path keeps working.** Until M3 explicitly retires it, every Stream A invocation produces identical results. | Walk-phase deployments running Stream A can't break. | Default-AMODE-H mechanism |
| **Per-phase deferred decisions are documented.** Each M-phase opening lists what it depends on and what it cannot decide. | Matches QBP-Node Spec philosophy (Parts 0/1 §0.3). | Spec review |
| **Old research re-enters via promotion gates.** QNEAR, QPERM, etc. ship only if evidence supports. | Older research is a *source of candidates*, not a commitment. | M2 promotion gates |

---

## 6. Risks and Mitigations

### R1 — Stream A users see ABI breakage at M1 if AMODE default isn't explicit

**Mitigation:** spec mandates `qbp.amode` resets to `H` on every mode-transition exit and on context restore. Stream A code that never writes `qbp.amode` sees `H` always — identical to v1.1 behavior. Document in M1 release notes.

### R2 — CIM Level-1 fails or partially fails

**Mitigation:** Stream B Layer 1 is **not** dependent on CIM. The QPERM/QDEC/QREC subset always ships (they exist in software regardless). QNEAR ships only if CIM passes. Layer 1 is therefore a partial-promotion: ship what passes, defer what doesn't.

### R3 — M3 deprecation of Stream A surface invalidates field deployments

**Mitigation:** M3 does not delete Stream A. It re-frames Stream A as "v1/v2 compatibility surface." Stream A code continues to assemble and run on Stream B silicon via the AMODE-defaults mechanism. Sharp Butler MVPs running Stream A keep working.

### R4 — `lean2rom` produces sign masks disagreeing with current asm

**Mitigation:** This is the most-likely failure mode of M0. If `lean2rom` produces sign masks that don't match `emulator/qmath_amd64.s` constants, two possibilities:
1. The Lean source is wrong — fix Lean, regenerate.
2. The current asm is wrong — fix asm, run cosim.

Either way the resolution is mechanical. Allocate Crawl time for it (~2 weeks).

### R5 — Older research idioms aren't viable under modern silicon constraints

**Mitigation:** Promotion-gate pattern handles this. QNEAR doesn't ship just because it's in older research; it ships if Level-1 evidence supports it. Default-deferral protects the baseline.

### R6 — Wyrd / BMA / Contextus consumers depend on Stream A surface and don't follow the migration

**Mitigation:** Wyrd's `compute/quaternion.go` already imports through a stable `model.Weight` surface. Stream B addition is transparent to Wyrd as long as `HamiltonProduct` semantics don't change. Coordinate with Wyrd via the integration interface (separate doc, see issue tracker).

---

## 7. The Single Most Important Decision This Plan Forces

**At M3 (Walk-β → Run-α), the spec changes its center of gravity.** Up to M2, Stream A is the surface and Stream B is the underlying machine model. At M3, Stream B becomes the authoritative ISA description and Stream A becomes a compatibility layer.

This decision is **not deferrable past Walk-β**. Run-phase silicon must be designed against Stream B (otherwise it inherits Stream A's per-width-register-file scaling problems and lacks first-class ZD/AMODE support).

The QBP-Node Spec App A.1 already captures the parallel Run-phase architectural choice (dual-domain vs unified). The Stream A→B authority transition is the *companion* decision. Both should be made in the same Walk-phase analysis cycle, with the same evidence base.

---

## 8. Concrete Next Actions (M0 immediate)

These four items are independent and parallel-safe:

1. **Build `lean2rom`** per QBP-Node Part 2 §2.4 — already a named Crawl deliverable. Start with octonion sign tables (49 entries × 1 bit each).
2. **Add WDEvent emission** to the AVX kernels Gemini just delivered. Passive at first — events go to a channel that nothing consumes yet.
3. **Open the `Xqbpoct` and `Xqbpvcp` draft extension stubs** from peer-review-002's recommendations. v0.1 placeholders.
4. **Document this migration plan** as `architecture/peer-review-005-stream-migration.md` (this document). Citable from spec, not chat-only.

These four steps cost approximately **three weeks** total. They unblock Phase M1 without committing to it.

---

## 9. References

- [`archive_build/docs/RV-Fano-Implementation-Refinements.md`](../archive_build/docs/RV-Fano-Implementation-Refinements.md) — Stream B authoritative source
- [`spec/QBP-RISCV-ISA-Spec-v1.1.md`](../spec/QBP-RISCV-ISA-Spec-v1.1.md) — Stream A authoritative source
- [`Archive/QBP-Node-Spec-v0.1-Parts-0-and-1.md`](../Archive/QBP-Node-Spec-v0.1-Parts-0-and-1.md) — Phasing model + deferred-decisions
- [`Archive/QBP-Node-Spec-v0.1-Part-2.md`](../Archive/QBP-Node-Spec-v0.1-Part-2.md) — Crawl deliverables incl. lean2rom and CIM Level-1
- [`Ref/RISC-V-Policies-and-Best-Practices.md`](../Ref/RISC-V-Policies-and-Best-Practices.md) §3, §4, §6 — Vendor extension policy + V-extension config template
- [`Ref/SiFive-Documentation-Patterns.md`](../Ref/SiFive-Documentation-Patterns.md) §6 — VCIX coprocessor-interface model
- [`architecture/peer-review-002-fano-mesh-isa-redteam.md`](peer-review-002-fano-mesh-isa-redteam.md) — Conventions audit; NF1/NF2/NF3 findings
- [`architecture/peer-review-003-qbp-node-spec-crawl.md`](peer-review-003-qbp-node-spec-crawl.md) §2 — Tensions T1, T2, T3
- `emulator/qmath_amd64.s` and `emulator/qmath_128_amd64.s` — Stream A working kernels (verified 2026-05-05)
- Wyrd repo `compute/quaternion.go` — Downstream consumer; integration discussion in companion doc

**Attribution carrying forward.** Furey, Günaydin & Gürsey, Dixon, Boyle & Farnsworth, Singh, Chamseddine & Connes, Koide, Baez, Moreno, Schafer, Cawagas. SiFive (Asanovic et al.) for VCIX. The QBP physics instance for the algebraic primitive specifications and the cross-copy ZD characterization. The engineering instance for the Go cycle-accurate simulator and Lean toolchain. Gemini for the SIMD assembly path (QW64 + QW128).

---

*Status: RECORDED | Companion: peer-review-001 (April 2026), peer-review-002, peer-review-003 | Reference: PROGRESS_LOG.md*
