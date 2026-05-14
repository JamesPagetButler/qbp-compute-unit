# QBP Compute Unit — `Xqbpvcp` Coprocessor Interface Extension v0.1

**Date:** 2026-05-06
**Target:** QBP (Quaternion-Based Physics) Compute Unit
**Status:** Stub draft (v0.1)
**Extension family:** `Xqbp*` (vendor prefix `qbp.`)
**Modeled on:** SiFive `XSfvcp` v1.1.0 (Vector Coprocessor Interface eXtension)
**Companion:** `spec/QBP-RISCV-Xqbpoct-Spec-v0.1.md`

---

## 0. Status & Maturity

This is a **v0.1 stub**: a structural placeholder that fixes the extension family, opcode allocation policy, and the core invariants the dispatch interface must honour. **It does not specify instruction encodings, vector-register operand-passing protocol, or fault model.** Those land in v0.2 under architecture-instance authority.

Modeled on SiFive's `XSfvcp` v1.1.0 (Vector Coprocessor Interface eXtension), which is the closest real-world analog: a stable, ratified scalar-to-coprocessor dispatch interface whose mnemonics package operands and an opaque function code, deferring instruction-set semantics to coprocessor-internal extensions. `Ref/SiFive-Documentation-Patterns.md` §6 documents the pattern.

**SiFive precedent:** the matching coprocessor-internal extension family (`Xsfmm*`) sits at v0.6 even though SiFive has shipped silicon. By mirroring that maturity discipline, `Xqbpvcp` v0.1 deliberately does *not* claim stability ahead of evidence; it sets the architectural shape and lets the substantive content evolve.

### 0.1 Governance status — design-doc-as-S-01-review-surface (ADR-003 §I4)

**This document is the S-01 review surface for the `Xqbpvcp` extension.** Per `architecture/adr-003-m1-wdevent-observer-invariants.md` §I4 (added 2026-05-06), structural extensions and new spec documents land first as design surface, receive explicit review from `bma` + `bma-implementor` + `qbp-architecture`, and only then do downstream implementation PRs (encoder/disassembler support, CSR plumbing, dispatch-port wiring, fault-handler hooks) open. **Implementation PRs that bypass this review surface are not skipping bureaucracy; they are bypassing the S-01 mechanism by which the beekeeper exercises oversight over structural changes.**

The §I4 invariant is particularly load-bearing for `Xqbpvcp` because §3.3 of this document defines the silicon-side actuator of ADR-003 §I3 (`mstatus.QBP` gates dispatch during structural actions). A premature implementation PR that wired the gate without prior governance review would short-circuit the very mechanism it is supposed to enforce.

v0.1 → v0.2 promotion of this document is therefore **design-gated**, not implementor-discretionary. v0.1 may evolve in-place (commit-by-commit edits during the review window are expected); v0.2 is reached only when the named reviewers explicitly sign off.

---

## 1. Purpose

`Xqbpvcp` defines the dispatch interface between the **scalar RISC-V hart** and the **QBP coprocessor cluster** — the family of accelerators (`Xqbpquat`, `Xqbpoct`, `Xqbpqec`, `Xqbpmem`, eventually `Xqbpmesh`) that provide algebraic and quantum-error-correction execution. It does **not** specify the coprocessor's internal instruction sets; those are the per-feature accelerator extensions named above.

This extension solves the same problem `XSfvcp` solves for SiFive's matrix coprocessor (`Xsfmm*`): how does a scalar core dispatch to a coprocessor without baking the coprocessor's microarchitecture into the scalar ISA?

**The split is load-bearing.** Per `architecture/peer-review-002-fano-mesh-isa-redteam.md` §S2, conflating the dispatch interface with coprocessor-internal compute is the mistake the QBP programme must avoid. `Xqbpvcp` is the conformant remedy.

---

## 2. Architectural State

### 2.1 The QBP Coprocessor Cluster

The cluster is a logical unit; physical implementation may be a single execution pipeline (Crawl/Walk emulator on FX-8350 / RX 9070 XT) or a separate die (Run-phase OpenMPW 130 nm ASIC). `Xqbpvcp` abstracts the dispatch path so the same scalar binary can drive any of these.

### 2.2 CSRs introduced (placeholder list)

The following CSRs are reserved for `Xqbpvcp` v0.x. Their exact addresses, layouts, and semantics are deferred to v0.2.

- **`qbp.amode`** (Algebra Mode) — selects ℍ / 𝕆 / 𝕊 / Branch-A / Branch-B execution domain. Must be readable by the WDEvent emission path so `WDEvent.AlgebraID` is populated correctly (closes the AlgebraID land mine flagged in `reviews/peer-review-006-wdevent-pr11-redteam.md` §6.4).
- **`qbp.bsel`** (Fano Line Selector) — basis-line selector for Stream B layer-1 ops; introduced per `architecture/peer-review-005-stream-migration.md` §M1.1.
- **`qbp.psel`** (Projection Selector) — projection selector, paired with `qbp.bsel`.
- **`mstatus.QBP`** (status bit) — gates VCIX dispatch during structural actions; load-bearing for ADR-003 §I3.4. See §3.3 below.

The exact CSR-address allocations are deferred. An RVI-aligned mapping for vendor-extension CSRs is preferred over collision with standard `mstatus`/`mhartid` namespace — to be pinned in v0.2.

### 2.3 Dispatch ports (SSCI / VCIX)

The coprocessor cluster accepts dispatched operations through two ports, mirroring the existing `emulator/wdevent.go` `Port` enum:

- **`PortSSCI`** — Scalar/Sync Coprocessor Interface. Operations dispatched here block the scalar hart until completion. Used by per-instruction algebraic ops (`qbp.qmul.w`, `qbp.omac.w`, etc.) where the scalar core needs the result in a subsequent instruction.
- **`PortVCIX`** — Vector Coprocessor Interface eXtension port. Asynchronous; the scalar hart may continue executing while the coprocessor processes a queued operation. Reserved for batch / streaming workloads (e.g., the future Stream B layer-2 mesh ops, `Xqbpmem` wide-load streaming).

**v0.1 stub posture:** SSCI is the fully-defined path (it is what `emulator/isa.go` already implements). VCIX is reserved; v0.x will define its handshake, queue depth, and back-pressure semantics. The `PortVCIX` enum value is wired through `emulator/wdevent.go` today but no `qbp.vc.*` instructions exist yet.

---

## 3. Invariants This Extension Enforces

These are **load-bearing properties** that any implementation of `Xqbpvcp` v0.1 must respect. They are derived from upstream architectural decisions and re-stated here so a future reviewer of the v0.2 instruction-encoding draft can check encodings against them.

### 3.1 Vendor mnemonic prefix (Toolchain SIG conformance)

All instructions defined under `Xqbpvcp` carry the **`qbp.`** prefix (e.g., `qbp.vc.x`, `qbp.vc.v` modeled on `sf.vc.x` / `sf.vc.v`). Bare-mnemonic forms are forbidden. This is the same conformance constraint that closed `architecture/peer-review-002-fano-mesh-isa-redteam.md` NF1 for the base spec.

### 3.2 No microarchitecture leak into ISA

`Xqbpvcp` instructions describe *dispatch* (scalar register sources, opaque function code, completion port). They do not describe mesh topology, lane count, FANO ROM port count, or any other coprocessor-internal microarchitectural choice. Coprocessor-internal compute lives in the per-feature extensions:

| Concern | Where it lives |
|---------|----------------|
| Scalar core dispatches operands to coprocessor | **`Xqbpvcp` (this extension)** |
| Quaternion algebra | `Xqbpquat` (v1.1) |
| Octonion algebra | `Xqbpoct` (v0.1) |
| Quantum error correction | `Xqbpqec` (v1.1 §4) |
| Wide memory ops | `Xqbpmem` (v1.1 §6) |
| Mesh allocation, width / watchdog config | `qmesh` CSR cluster + `Xqbpmesh` (future) |
| Mesh-internal compute | `Xqbpmesh` (future) |

Per peer-review-002 §S2 / `Ref/SiFive-Documentation-Patterns.md` §6, this decomposition delivers everything the QBP programme needs without modifying the v1.0/v1.1 base spec. Each extension matures at its own rate.

### 3.3 `mstatus.QBP` gates dispatch during structural actions (ADR-003 §I3.4)

This is the silicon-side actuator of the I3 invariant: **algebraic-isolation-aware lock boundary; observer gated OUT during structural actions.**

When `mstatus.QBP = 0`, no `qbp.vc.*` dispatch may complete. Any dispatched op observes the gate and either:

- (a) traps with a defined fault code (preferred), or
- (b) stalls until `mstatus.QBP = 1` is restored (acceptable if (a) is unimplementable on a target).

Structural actions (checkpoint, layer-boundary change, ethics-framework amendment) clear `mstatus.QBP` for their full duration. Dispatching during a structural action is the exact apparent-completion-without-completion failure mode I3 prohibits.

**M1 implementation note:** the active WDEvent observer (BMA-side goroutine, see ADR-003) is software in the Crawl/Walk phase. Software emulation of the `mstatus.QBP` gate is required at M1; hardware enforcement arrives at Run-α. ADR-003 §I3.4 is the citation reference.

**Relationship to `Wyrd.model.Graph` RWMutex (ADR-003 §I3.1):** the RWMutex on `model.Graph` is the *complementary software-side* I3 mechanism. `mstatus.QBP` is the silicon-side hardware mechanism. Both gate the same observer, at different layers of the substrate stack.

### 3.4 Trap behaviour

`Xqbpvcp` adheres to standard RISC-V exception semantics. Specifically:

1. **Illegal Instruction Trap** (`mcause = 2`) on any dispatch with an unsupported function code, an invalid extension target, or an invalid CSR address.
2. **Structural-Action Gate Trap** (fault code TBD, in v0.2) when dispatch occurs while `mstatus.QBP = 0`. v0.1 reserves the fault code; the numeric value pins in v0.2.
3. **No silent precision change.** Dispatch never autonomously downgrades precision; the determinism critique that closed `peer-review-002-fano-mesh-isa-redteam.md` §R2 (`QYIELD` rejection) applies symmetrically here.

### 3.5 Independence from `Xqbpquat` v1.1 freeze

`Xqbpvcp` v0.1 is a **new draft extension** that does **not** modify `QBP-RISCV-ISA-Spec-v1.1.md`. The v1.1 freeze invariant (`peer-review-002-fano-mesh-isa-redteam.md` §S1) is honoured.

---

## 4. v0.1 Deliverable Scope

**What this stub commits to:**

- Extension X-name registered: `Xqbpvcp`
- Mnemonic prefix registered: `qbp.`
- Modeled on `XSfvcp` v1.1.0; structure inherited
- CSR list reserved (§2.2); semantics deferred
- Port enum aligned with `emulator/wdevent.go` (`PortSSCI`, `PortVCIX`); behaviour deferred for `PortVCIX`
- Architectural invariants from upstream (ADR-003, peer-review-002, peer-review-005) re-stated as enforcement requirements (§3)

**What this stub does NOT commit to (deferred to v0.2 under architecture-instance authority):**

- Bit-by-bit encoding of `qbp.vc.*` instructions
- CSR address allocation
- VCIX queue depth, back-pressure model, async-completion handshake
- Vector-register operand-passing protocol
- Numeric structural-action-gate fault code
- Power model / silicon process binding

**v0.2 review surface:** per ADR-003 §I4, the v0.2 promotion of this stub will land first as a design doc with explicit review from `qbp-architecture` + `bma` + `bma-implementor` before any implementation PR opens. v0.1 → v0.2 is design-gated; v0.1 → ratified silicon is gated by Walk-phase ROCm/AVX validation per `Ref/SiFive-Documentation-Patterns.md` §5.

---

## 5. Open Questions

### 5.1 SSCI vs VCIX boundary for per-instruction algebra

Currently every `Xqbpquat` and `Xqbpoct` op dispatches via SSCI (the scalar hart blocks until completion). This is correct for sub-cycle ops where the result is consumed immediately. At what cycle-count or workload-pattern threshold does it become advantageous to route an op through VCIX (async)?

**Resolution path:** measure the BMA spreading-activation benchmark (autonomic 10 Hz loop, ~140K QMUL/tick at QW8) under both dispatch paths once VCIX is implementable. Pin the threshold based on data, not theory. Tracked in the M1 → M2 extension queue.

### 5.2 `qbp.amode` interaction with context switches

If a process is preempted mid-`qbp.omac.w` chain (left-associative octonion multiply, where ordering is significant per `Xqbpoct` §5.1), the preemptor must save the AMODE CSR, and the resumed code must restore it before the next dispatch. RVV's `vtype`/`vl` save-restore is the precedent (`Ref/RISC-V-Policies-and-Best-Practices.md` §6.4). v0.2 must specify the equivalent for `qbp.amode`, `qbp.bsel`, `qbp.psel`.

### 5.3 Multi-hart dispatch arbitration

If an SoC has multiple harts each capable of dispatching to `Xqbp*` extensions, do they share a single coprocessor cluster (with arbitration) or one cluster per hart? `XSfvcp`'s vector state is per-hart by spec. `Xqbpvcp` v0.2 must adopt the same model or justify divergence — v0.1 leaves this open.

### 5.4 Branch-A / Branch-B dark-matter fork (`AMODE` codes 2 / 3)

`WDEvent.AlgebraID` reserves codes 2 (Branch A: C ⊕ ℍ ⊕ M3(C)) and 3 (Branch B). These reflect the dark-matter-fork architectural choice flagged in `Archive/QBP-Node-Spec-v0.1-Parts-0-and-1.md` Appendix A.4. v0.1 reserves the codes; the substantive question of whether dual-domain or unified silicon ships at Run-α is deferred per peer-review-005 §M3.

---

## 6. References

- `spec/QBP-RISCV-ISA-Spec-v1.1.md` (in preparation) — base extension `Xqbpquat`, current home of all algebraic ops
- `spec/QBP-RISCV-Xqbpoct-Spec-v0.1.md` — companion octonion-extension stub
- `spec/QBP-Compute-Unit-Architecture-v1.0.md` §3 — CTH Watchdog → Constitutional Audit interrupt path
- `architecture/adr-003-m1-wdevent-observer-invariants.md` §I3.4, §I4 — mstatus.QBP gating; design-doc-as-S-01-review-surface
- `architecture/peer-review-002-fano-mesh-isa-redteam.md` §S2 / NF1–NF3 — extension-decomposition rationale
- `architecture/peer-review-005-stream-migration.md` §M1 — `qbp.amode`, `qbp.bsel`, `qbp.psel` introduction
- `reviews/peer-review-006-wdevent-pr11-redteam.md` §6.4 — AlgebraID land mine the AMODE CSR closes
- `Ref/SiFive-Documentation-Patterns.md` §6 — VCIX coprocessor-interface model (architectural template)
- `Ref/RISC-V-Policies-and-Best-Practices.md` §3, §4, §6.4 — vendor naming, prefix conventions, RVV configuration template (context-switch precedent)
- `emulator/wdevent.go` — `Port` enum (SSCI / VCIX) currently in use

---

## Appendix A: Revision History

| Version | Date | Description |
|---------|------|-------------|
| **v0.1** | 2026-05-06 | Initial stub. Extension X-name registered (`Xqbpvcp`); modeled on SiFive `XSfvcp` v1.1.0. Architectural invariants from ADR-003, peer-review-002, peer-review-005 re-stated as enforcement requirements. CSRs (`qbp.amode`, `qbp.bsel`, `qbp.psel`, `mstatus.QBP`) listed as placeholders; semantics deferred. SSCI / VCIX port enum aligned with `emulator/wdevent.go`. v0.1 → v0.2 promotion is design-gated per ADR-003 §I4. |

---

*Status: STUB v0.1 | Owner: qbp-cu-implementor | Review: qbp-architecture (pending) | Audit trail: qbp-cu-walk seq=15–18*
