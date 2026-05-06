# QBP Compute Unit — `Xqbpoct` Octonion Extension v0.1

**Date:** 2026-05-06
**Target:** QBP (Quaternion-Based Physics) Compute Unit
**Status:** Draft (v0.1)
**Extension family:** `Xqbp*` (vendor prefix `qbp.`)
**Parent base spec:** QBP-RISCV-ISA-Spec-v1.1 (in preparation; cited as v1.1 below)
**Companion:** `spec/QBP-RISCV-Xqbpvcp-Spec-v0.1.md`

---

## 0. Status & Maturity

This is a **v0.1 stub draft**, modeled on SiFive's `Xsfmm*` cadence (currently at v0.6 despite shipped silicon — see `Ref/SiFive-Documentation-Patterns.md` §5). v0.1 means: experimental, subject to breaking changes, not in production silicon, expected to mature through v0.x → v1.0 only after Walk-phase ROCm/AVX validation and toolchain integration.

This document carves out the octonion subset of v1.1's monolithic Continuous Algebra section into a standalone vendor extension, per the per-feature decomposition recommended by `architecture/peer-review-002-fano-mesh-isa-redteam.md` §S2 / NF2. The split is structural; the underlying mathematics is unchanged from v1.1.

**v1.1 §2.1 amendment to remove the migrated mnemonics is owed by the architect-instance review pass and is not landed in this PR.** During the review window the same ops appear in both v1.1 §2.1 and this document; the duplication is acknowledged and resolved on amend.

---

## 1. Encoding Conventions & Overview

`Xqbpoct` provides hardware support for native Cayley-Dickson **octonion** algebra: the unique non-associative normed division algebra over ℝ (Hurwitz). The extension is a peer of `Xqbpquat` (continuous algebra over ℍ); both share a common encoding convention and execution boundary, but their algebraic semantics differ — most importantly, octonion multiplication is non-associative.

### 1.1 Vendor extension naming

Per RISC-V Toolchain SIG vendor policy (`Ref/RISC-V-Policies-and-Best-Practices.md` §3, §4), this extension uses:

- Extension X-name: **`Xqbpoct`**
- Mnemonic prefix: **`qbp.`** (lowercase, period-terminated, ≥2 chars; conformant)

### 1.2 Opcode space

`Xqbpoct` shares the **`custom-0` (`0x0B`)** opcode space with `Xqbpquat`. The two extensions partition this space by `funct7` value; the octonion ops occupy `funct7 ∈ {2, 14..18}` (see §2). No re-encoding is required — instructions retain the funct7 values they had in v1.1 §2.1.

**Rationale:** quaternion and octonion ops are continuous floating-point algebra dispatched to the same FP execution path. Co-locating them in `custom-0` matches the SiFive convention for related accelerator extensions sharing an opcode space (e.g., `XSfvqmaccdod` and `XSfvqmaccqoq` both occupy SiFive's vector custom space).

### 1.3 Width selector

The `funct3` field acts as a width selector identical to `Xqbpquat`'s (v1.1 §1.3). Octonion words at width W occupy 2× the bit-width of the corresponding quaternion word at the same width, since an octonion has 8 components vs a quaternion's 4.

| funct3 | Width | Octonion bits | Algebraic life @ 1 GHz |
|--------|-------|---------------|------------------------|
| `000` | OW8 (per-component 8 bits) | 64 | < 1 op |
| `001` | OW16 | 128 | tens of ops |
| `010` | OW32 | 256 | ~ms |
| `011` | OW64 | 512 | ~s |
| `100` | OW128 | 1024 | days–months (172d at QW128 quaternion-equivalent) |
| `101` | OW256 | 2048 | Deep compliance / Emulation |
| `110` | OW512 | 4096 | Deep compliance / Emulation |
| `111` | OW1024 | 8192 | Maximum theoretical bound / Emulation |

Algebraic-life figures inherit from the quaternion analysis; refinement specific to the octonion drift envelope is a v0.x open question (§5).

### 1.4 Portability disclaimer

`Xqbpoct` uses the closed-ecosystem `custom-0` opcode space. Code compiled against `Xqbpoct` is **not portable** to standard RVA22/RVA23 application processors. This matches the disclaimer in `Xqbpquat` v1.1 §1.2.

---

## 2. Instructions

All instructions use standard RISC-V R-type format: `funct7 | rs2 | rs1 | funct3 | rd | opcode`.

| Mnemonic | funct7 | funct3 | rd, rs1, rs2 | Description | Cycle Count (W8 / W32 / W64 / W128) | Epistemic Tier |
|----------|--------|--------|--------------|-------------|--------------------------------------|----------------|
| `qbp.omac.w`   |  2 | w | rd, rs1, rs2 | Octonionic multiply-accumulate (uses Fano LUT) | 4 / 8 / 16 / 32 | T1 (Proven) |
| `qbp.oadd.w`   | 14 | w | rd, rs1, rs2 | Octonion add | 1 / 2 / 4 / 8 | T1 (Proven) |
| `qbp.osub.w`   | 15 | w | rd, rs1, rs2 | Octonion sub | 1 / 2 / 4 / 8 | T1 (Proven) |
| `qbp.oscale.w` | 16 | w | rd, rs1, rs2 | Scalar × octonion (rs2 = scalar) | 1 / 2 / 4 / 8 | T1 (Proven) |
| `qbp.oconj.w`  | 17 | w | rd, rs1      | Octonion conjugate (negate imaginary components) | 1 / 1 / 1 / 1 | T1 (Proven) |
| `qbp.onorm.w`  | 18 | w | rd, rs1      | Octonion norm squared `‖o‖²` (8-component) | 2 / 4 / 8 / 16 | T1 (Proven) |

Cycle counts mirror v1.1 §2.1; they are reproduced verbatim during the carve-out and have not been re-validated against measurement.

### 2.1 Dependence on `qbp.fano`

`qbp.omac.w` requires the Fano-plane lookup primitive `qbp.fano` (defined in `Xqbpquat` v1.1 §2.1, funct7=3). `qbp.fano` is a **shared primitive** that remains in the base extension and is invoked by `Xqbpoct`'s multiplication paths. The Fano ROM orientation is fixed at the **Conway-Smith standard** per v1.1 §5.

This shared-primitive arrangement is a stub-time choice; if the FANO ROM access pattern is later shown to differ between quaternion and octonion contexts, a follow-on extension (`Xqbpoct.fano`) may be carved out with its own lookup. v0.1 keeps the primitive shared.

---

## 3. Register Model

Octonions are 8-component vectors and require 8 contiguous RISC-V FP registers (vs 4 for quaternions). This document defines **Octonion Register (OR)** aliases, paralleling `Xqbpquat`'s `qr0`–`qr7`.

### 3.1 Aliases

There are 4 Octonion Registers (`or0`–`or3`), grouping the 32 standard FP registers in pairs of 8.

- `or0` (`f0–f7` / `ft0–ft3, ft4–ft7`) : Caller-saved temporary
- `or1` (`f8–f15` / `fs0–fs3, fa0–fa3`) : Mixed callee/caller-saved (fa0–fa3 are arguments)
- `or2` (`f16–f23` / `fa4–fa7, fs4–fs7`) : Mixed callee/caller-saved
- `or3` (`f24–f31` / `fs8–fs11, ft8–ft11`) : Mixed callee/caller-saved

Calling convention: octonion arguments are passed in `or1` (overlapping `fa0–fa3` plus `fs0–fs3`); a callee that uses `fs0–fs3` must spill them per RISC-V ABI. v0.1 does not strictly bind callee/caller saves at the OR granularity — that is a v0.2 open question pending toolchain ABI alignment.

### 3.2 Width modalities

- **OW8–OW64**: Fits inside `F` and `D` extension physical registers (8 × FP-register-width per OR).
- **OW128**: Requires the RISC-V `Q` extension. Each FP register `f_i` is 128 bits wide, so an OR of 8 contiguous `Q`-extension registers holds a 1024-bit OW128 octonion.
- **OW256+**: Software emulation only (no RISC-V FP-register width supports 256+ bits natively).

### 3.3 Relationship to `qr_k`

An `or_k` overlaps two consecutive `qr_{2k}` and `qr_{2k+1}`. This means the same physical FP register file is shared between quaternion and octonion compute:

- `or0` ≡ `qr0` ∥ `qr1` (concatenation)
- `or1` ≡ `qr2` ∥ `qr3`
- `or2` ≡ `qr4` ∥ `qr5`
- `or3` ≡ `qr6` ∥ `qr7`

This sharing is **stub-time** behaviour — the alternative (separate OR file) is a v0.x design choice deferred to architecture-instance review.

---

## 4. Trap & Exception Behaviour

`Xqbpoct` adheres to the standard RISC-V exception handling semantics inherited from `Xqbpquat` v1.1 §7:

1. **Illegal Instruction Trap.** Any `qbp.o*` instruction with an unsupported `funct3` width or an invalid `or` register grouping triggers an illegal instruction exception (`mcause = 2`). Hardware does **not** silently downgrade precision.
2. **Constitutional Audit Interrupt.** If the hardware CTH Watchdog is enabled and the scalar norm of an octonion deviates from 1.0 beyond the threshold (~10⁻³⁰ at OW128), the hardware triggers an asynchronous interrupt to BMA, **not** a silent precision change. This matches the `Xqbpquat` semantics; the threshold scaling for octonion ops at OW128 is a v0.x open question pending empirical drift measurement.
3. **Unaligned Address Trap.** N/A in `Xqbpoct` — no load/store ops are defined here. Octonion memory operations live in `Xqbpmem` (see v1.1 §6 / future `Xqbpmem` carve-out).

---

## 5. Open Questions

### 5.1 Associativity tracking

Octonion multiplication is non-associative: `(ab)c ≠ a(bc)` in general. v1.1 §F2 left this open: should `qbp.omac.w` chains be left-associative by default (as the RISC-V instruction stream implicitly orders), or is an explicit grouping primitive (`qbp.ogroup`) required?

`architecture/peer-review-002-fano-mesh-isa-redteam.md` §3 and §6 (M7) flag this as a **compiler-pass open question**: aggressive `-O3` reordering of independent `qbp.omac.w` instructions could corrupt BMA topological-memory paths if the compiler reorders without honouring associator boundaries. Resolution requires a software test compiler pass (per v1.1 §F2 emulation note).

**v0.1 stance:** left-associative by default; `qbp.ogroup` is a v0.x candidate primitive, not yet specified. Compilers targeting `Xqbpoct` should treat `qbp.omac.w` chains as ordering-sensitive until `qbp.ogroup` semantics are pinned.

### 5.2 Octonion drift envelope

The cycle counts and algebraic-life figures in §2 / §1.3 are inherited from the quaternion analysis. The composition-depth lifetime of octonion arithmetic at OW128 has not been independently measured; it may differ from the QW128 quaternion case because octonion multiplication is non-associative and norm-multiplicativity composition behaves differently under reorderings.

**Resolution path:** run the pkg/octonion `NormMultiplicativity` benchmark across composition depths 10² to 10⁹ at OW128 and report the empirical drift envelope; promote `T1 (Proven)` to `T1 + measured` once data is in. Tracked in the M0 → M0.x extension queue.

### 5.3 Watchdog integration

Per `architecture/adr-003-m1-wdevent-observer-invariants.md` §I2 and `architecture/peer-review-005-stream-migration.md` §2.3, the M1 active observer reads `cpu.WatchdogChan` events with `AlgebraID ∈ {0=H, 1=O, 2=Branch A, 3=Branch B}`. `Xqbpoct` ops emit `WDEvent` with `AlgebraID = 1` ("O").

**Implementation reminder:** v1.1 §7 / `emulator/isa.go:153` carries a `// TODO(M1): populate from c.csr.AMODE` comment for the `AlgebraID` field; once `Xqbpvcp` v0.x defines AMODE-as-CSR (see `spec/QBP-RISCV-Xqbpvcp-Spec-v0.1.md` §2), the emission path must be updated to read AMODE from CSR rather than relying on struct zero-init. `Xqbpoct` ops that execute while AMODE is still 0 will silently mis-tag as quaternion events — the **AlgebraID land mine** flagged in `reviews/peer-review-006-wdevent-pr11-redteam.md` §6.4. Closure of this open question is a hard prerequisite for promoting `Xqbpoct` past v0.1.

### 5.4 Independence of opcode space from `Xqbpquat`

§1.2 keeps `Xqbpoct` in `custom-0` shared with `Xqbpquat`. This is the conservative choice and matches v1.1's monolithic placement. The alternative — moving `Xqbpoct` to a fresh `custom-3` opcode space (`0x7B`) — was considered and rejected for v0.1 on grounds of (a) no encoding collision exists in current funct7 assignments, (b) shared dispatch path simplifies the `Xqbpvcp` interface contract (§3 of the companion stub), and (c) custom-3 reservation is preferable to keep available for `Xqbpmesh` per peer-review-002 §S2.

If a future Run-phase ASIC implements `Xqbpquat` and `Xqbpoct` as physically separate execution units (rather than a shared FP datapath), splitting opcode spaces becomes attractive. v0.x decision deferred.

---

## 6. References

- `spec/QBP-RISCV-ISA-Spec-v1.1.md` (in preparation) — base extension `Xqbpquat`, source for the carve-out
- `spec/QBP-RISCV-Xqbpvcp-Spec-v0.1.md` — companion stub for the coprocessor dispatch interface
- `spec/QBP-Compute-Unit-Architecture-v1.0.md` §3 — CTH Watchdog → Constitutional Audit interrupt path
- `architecture/adr-003-m1-wdevent-observer-invariants.md` §I2, §I3.4 — observer namespace + silicon-side gating
- `architecture/peer-review-002-fano-mesh-isa-redteam.md` §S2 / NF2 — extension-decomposition rationale
- `architecture/peer-review-005-stream-migration.md` §2.3 — Stream B `WDEvent.AlgebraID` schema
- `reviews/peer-review-006-wdevent-pr11-redteam.md` §6.4 — AlgebraID land mine
- `Ref/SiFive-Documentation-Patterns.md` §4–§6 — vendor extension catalog and maturity ladder
- `Ref/RISC-V-Policies-and-Best-Practices.md` §3, §4 — vendor naming and prefix conventions

---

## Appendix A: Revision History

| Version | Date | Description |
|---------|------|-------------|
| **v0.1** | 2026-05-06 | Initial carve-out from QBP-RISCV-ISA-Spec-v1.1 §2.1. Extension family declared. Six octonion mnemonics migrated. Shared opcode space with `Xqbpquat` retained; shared FP register file via `or_k ↔ qr_{2k} ∥ qr_{2k+1}`. Open questions documented for associativity tracking, drift envelope, watchdog integration, opcode-space independence. |

---

*Status: DRAFT v0.1 | Owner: qbp-cu-implementor | Review: qbp-architecture (pending) | Audit trail: qbp-cu-walk seq=15–18*
