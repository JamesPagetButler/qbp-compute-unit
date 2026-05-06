# ADR 001: Stream A as Surface Form, Stream B as Machine Model

**Status:** Accepted
**Date:** 2026-05-05
**Deciders:** James Paget Butler (beekeeper), Claude Opus 4.7 (architecture instance)
**Closes:** [issue #4 (T1)](https://github.com/JamesPagetButler/qbp-compute-unit/issues/4)
**Related:** [`peer-review-005-stream-migration.md`](peer-review-005-stream-migration.md), [`peer-review-002-fano-mesh-isa-redteam.md`](peer-review-002-fano-mesh-isa-redteam.md), [`peer-review-003-qbp-node-spec-crawl.md`](peer-review-003-qbp-node-spec-crawl.md) §2 T1

---

## Context

Two parallel ISA documents exist for the QBP Compute Unit, with incompatible mnemonic conventions and structural models:

| Stream | Mnemonic style | Source | Structure |
|---|---|---|---|
| **A** | `qbp.qmul.w`, `qbp.qrot.w`, `qbp.fano` | [`spec/QBP-RISCV-ISA-Spec-v1.1.md`](../spec/QBP-RISCV-ISA-Spec-v1.1.md) | Flat per-instruction, width via funct3 |
| **B** | `QPERM`, `AMODE`, `BSEL`, `PSEL`, `ZDCHK.SYM`, `OMUL`, `SMUL` | [`archive_build/docs/RV-Fano-Implementation-Refinements.md`](../archive_build/docs/RV-Fano-Implementation-Refinements.md) | Layer 0/1/2 compositional, mode-stateful |

These represent **different machine models**, not different views of the same ISA. Stream A is RISC-V vendor-conformant (vendor prefix, custom-0/1/2 opcode allocation, X-extension naming); Stream B is physics-faithful (algebraic mode awareness, sedenion zero-divisor detection, Fano-line structural addressing, Lean-as-authority sign tables).

Until this tension is resolved, every contributor (Gemini, engineering instances, Wyrd consumers) has to guess which ISA they target. Part 2 of the QBP-Node Spec uses Stream B vocabulary throughout (§2.2.3, §2.2.5, §2.5.1) but v1.1 (released the same week) uses Stream A. Either is workable; both cannot be authoritative.

## Decision

**Stream A v1.x is the surface form a RISC-V toolchain sees. Stream B v2.0+ is the underlying machine model the QBP physics requires. They merge through phased migration M0–M3 per [`peer-review-005`](peer-review-005-stream-migration.md).**

### Concretely:

1. **Stream A v1.1 is authoritative for the v1.x line.** All current code (Wyrd, BMA, Sharp Butler, the emulator) targets the Stream A surface. No flag day.

2. **Stream B opens as v2.0 in a separate spec document** introducing the CSRs (`qbp.amode`, `qbp.bsel`, `qbp.psel`), the Layer 0/1/2 vocabulary, and the mode-transition state machine with fault codes 0x10–0x14.

3. **Stream A primitives become Stream B Layer 0 in disguise** at M1. The same kernel asm (Gemini's QW64 / QW128 paths) backs both surface forms; only the dispatcher and surrounding state differ. Default `qbp.amode = H` means existing Stream A code runs identically to v1.1 behavior.

4. **Maintenance policy for Stream A v1.x:** cleanup-only. The mandatory items from peer-review-002 (mnemonic prefix conformance, trap behavior, revision history) ship at v1.1 and v1.2. New ops do not land at v1.x — they land at v2.0+.

5. **Stream B v2.0** is the entrypoint for new functionality. ZDCHK.SYM, hypergraph-native Layer 1 ops (HEDGE_GATHER, QPERM, QNEAR if CIM Level-1 promotes), and the `Xqbpvcp` coprocessor interface all live in v2.x.

6. **At M3 (Walk-β → Run-α)** Stream B becomes authoritative. v3.0 declares Stream A the "v1/v2 compatibility surface" — it continues to assemble and run on Stream B silicon via the AMODE-defaults mechanism, but new development targets Stream B directly.

### Mapping table (Stream A ↔ Stream B Layer 0 + AMODE requirement)

| Stream A v1.1 mnemonic | Stream B Layer 0 role | Mode requirement |
|---|---|---|
| `qbp.qmul.w` | `MUL` | AMODE=H |
| `qbp.qrot.w` | composed: 2× `MUL` + `CONJ` | AMODE=H |
| `qbp.qadd.w` | `ADD` | AMODE=H |
| `qbp.qconj.w` | `CONJ` | AMODE=H |
| `qbp.qnorm.w` | `NORM` | AMODE=H |
| `qbp.omac.w` | `MAC` | AMODE=O |
| `qbp.fano` | `LUT` | AMODE=O (FANO ROM lookup) |
| `qbp.pauli`/`qbp.synd`/`qbp.stab` | Layer 1 QEC | algebra-orthogonal |

Future Stream B ops (`qbp.zdchk`, `qbp.zdchk.sym`, `qbp.qperm`, `qbp.qnear`, `qbp.amode`, `qbp.bsel`, `qbp.psel`) have no Stream A equivalent — they are net-new at v2.0.

## Consequences

### Positive

- **No flag day.** Existing Stream A code continues to compile, link, and run identically through M1, M2, and into Run-α.
- **Authority chain clarified.** When physics work or older memory/compute research surfaces a new primitive, it lands in Stream B without disturbing v1.x.
- **Toolchain integration unblocked.** Stream A's vendor-prefix conformance lets `qbp.*` mnemonics enter LLVM/GCC at v1.1; Stream B's experimental vocabulary lands behind `-menable-experimental-extensions` at v2.0.
- **Wyrd consumer stable.** `Gearbox.QMul64([4]float64) [4]float64` and the typed-per-width API stay valid surface; mode-aware dispatch hidden inside Gearbox at M1.

### Negative

- **Two parallel specs to maintain** through the M0–M3 window. Mitigated by Stream A v1.x being maintenance-only.
- **Mode-state CSRs must be designed carefully** at M1 to not break Stream A defaults (default `AMODE=H` is the load-bearing invariant).
- **Some duplication of cycle budgets** across docs. Mitigated by Stream B v2.0 referencing Stream A v1.1 cycle counts where they coincide.

### Neutral

- The QBP-Node Spec Part 2 references Stream B vocabulary; this ADR clarifies that the references are forward-looking (toward v2.0+ state) and not contradicted by v1.1 today.

## Implementation

This ADR closes [issue #4 (T1)](https://github.com/JamesPagetButler/qbp-compute-unit/issues/4).

The migration that operationalizes this decision is tracked at the parent epic [#3](https://github.com/JamesPagetButler/qbp-compute-unit/issues/3). M0 work items (#7 lean2rom, #8 WDEvent emission, #9 Xqbpoct/Xqbpvcp stubs, #10 Wyrd integration) all proceed under this ADR's ground rules.

The Stream B v2.0 spec drafting is a Walk-α deliverable per peer-review-005 §4 (M1).

## References

- [`peer-review-005-stream-migration.md`](peer-review-005-stream-migration.md) — Full migration plan with M0–M3 phases
- [`peer-review-002-fano-mesh-isa-redteam.md`](peer-review-002-fano-mesh-isa-redteam.md) — RISC-V conventions audit (NF1/NF2/NF3)
- [`peer-review-003-qbp-node-spec-crawl.md`](peer-review-003-qbp-node-spec-crawl.md) §2 T1 — Tension that prompted this ADR
- [`spec/QBP-RISCV-ISA-Spec-v1.1.md`](../spec/QBP-RISCV-ISA-Spec-v1.1.md) — Stream A authoritative source
- [`archive_build/docs/RV-Fano-Implementation-Refinements.md`](../archive_build/docs/RV-Fano-Implementation-Refinements.md) — Stream B authoritative source
- [`Ref/RISC-V-Policies-and-Best-Practices.md`](../Ref/RISC-V-Policies-and-Best-Practices.md) §1 — Maturity workflow that drives the v1.x cleanup-only policy

---

*Status: ACCEPTED 2026-05-05 | Closes issue #4 | Companion: ADR-002*
