# QBP Compute Unit — RISC-V ISA Specification v1.1

**Date:** 2026-05-04
**Target:** QBP (Quaternion-Based Physics) Compute Unit
**Status:** Draft / Stable
**Extension Family:** `Xqbp*`

---

## 1. Encoding Conventions & Overview

The QBP Compute Unit extends the standard RISC-V instruction set with native hardware support for Quaternion and Octonion floating-point operations. It relies on the RISC-V `F` (Single), `D` (Double), and `Q` (Quad) floating-point extensions as the basis for register storage.

### 1.1 Vendor Extension Naming
This specification defines the `Xqbp*` family of vendor extensions. Specifically, this document serves as the base definition for:
- `Xqbpquat`: Base Quaternion Algebra
- `Xqbpqec`: Quantum Error Correction
- `Xqbpmem`: Wide Memory Operations

All instruction mnemonics in this specification conform to the RISC-V Toolchain SIG vendor policy and mandate the `qbp.` prefix.

### 1.2 Portability Disclaimer
The instructions defined herein utilize the `custom-0` (`0x0B`), `custom-1` (`0x2B`), and `custom-2` (`0x5B`) opcode spaces. These are closed-ecosystem encodings. Code compiled using `Xqbp*` extensions is **not portable** to standard RVA22/RVA23 application processors.

### 1.3 Width Selector
The `funct3` field acts as a width selector for mathematical operations:
- `000` = QW8 (32 bits)
- `001` = QW16 (64 bits)
- `010` = QW32 (128 bits)
- `011` = QW64 (256 bits)
- `100` = QW128 (512 bits) - Primary physics target (172-day algebraic life)
- `101` = QW256 (1024 bits) - Deep compliance / Emulation
- `110` = QW512 (2048 bits) - Deep compliance / Emulation
- `111` = QW1024 (4096 bits) - Maximum theoretical bound / Emulation

---

## 2. Output A: Complete Instruction Table

### 2.1 Continuous Algebra (`Xqbpquat`, Opcode `custom-0`, `0x0B`)

All instructions use standard R-type format: `funct7 | rs2 | rs1 | funct3 | rd | opcode`.

| Mnemonic | funct7 | funct3 | rd, rs1, rs2 | Description | Cycle Count (W8/W32/W64/W128) | Epistemic Tier |
|----------|--------|--------|--------------|-------------|------------------------------|----------------|
| `qbp.qmul.w` | 0 | w | rd, rs1, rs2 | Hamilton product | 2 / 4 / 6 / 12 | T1 (Proven) |
| `qbp.qrot.w` | 1 | w | rd, rs1, rs2 | Rotation q v q* | 3 / 6 / 9 / 18 | T1 (Proven) |
| `qbp.omac.w` | 2 | w | rd, rs1, rs2 | Octonionic MAC | 4 / 8 / 16 / 32 | T1 (Proven) |
| `qbp.fano`   | 3 | - | rd, rs1, rs2 | Fano LUT lookup | 1 / 1 / 1 / 1 | T1 (Proven) |
| `qbp.qnorm.w`| 4 | w | rd, rs1 | Norm squared | 1 / 2 / 4 / 8 | T1 (Proven) |
| `qbp.qadd.w` | 5 | w | rd, rs1, rs2 | Quaternion add | 1 / 1 / 2 / 4 | T1 (Proven) |
| `qbp.qsub.w` | 6 | w | rd, rs1, rs2 | Quaternion sub | 1 / 1 / 2 / 4 | T1 (Proven) |
| `qbp.qscale.w`| 7 | w | rd, rs1, rs2 | Scalar multiply | 1 / 1 / 2 / 4 | T1 (Proven) |
| `qbp.qdot.w` | 8 | w | rd, rs1, rs2 | Dot product | 1 / 2 / 4 / 8 | T1 (Proven) |
| `qbp.qconj.w`| 9 | w | rd, rs1 | Conjugation | 1 / 1 / 1 / 1 | T1 (Proven) |
| `qbp.qinv.w` | 10 | w | rd, rs1 | Inverse | 4 / 8 / 16 / 32 | T1 (Proven) |
| `qbp.qexp.w` | 11 | w | rd, rs1 | Exponential | 10 / 20 / 40 / 80 | T2 (Measured)|
| `qbp.qlog.w` | 12 | w | rd, rs1 | Logarithm | 10 / 20 / 40 / 80 | T2 (Measured)|
| `qbp.qmac.w` | 13 | w | rd, rs1, rs2 | Quaternion MAC | 2 / 4 / 6 / 12 | T1 (Proven) |
| `qbp.oadd.w` | 14 | w | rd, rs1, rs2 | Octonion add | 1 / 2 / 4 / 8 | T1 (Proven) |
| `qbp.osub.w` | 15 | w | rd, rs1, rs2 | Octonion sub | 1 / 2 / 4 / 8 | T1 (Proven) |
| `qbp.oscale.w`| 16 | w | rd, rs1, rs2 | Octonion scale | 1 / 2 / 4 / 8 | T1 (Proven) |
| `qbp.oconj.w`| 17 | w | rd, rs1 | Octonion conj | 1 / 1 / 1 / 1 | T1 (Proven) |
| `qbp.onorm.w`| 18 | w | rd, rs1 | Octonion norm | 2 / 4 / 8 / 16 | T1 (Proven) |

---

## 3. Output B: Register Model

Quaternions are vectors spanning 4 contiguous RISC-V floating point registers. We define the **Quaternion Register (QR)** aliases for assembly programming to remain fully compatible with the RISC-V ABI.

### 3.1 Aliases & Calling Convention

There are 8 Quaternion Registers (`qr0` to `qr7`), grouping the 32 standard FP registers.

- `qr0` (`f0-f3` / `ft0-ft3`) : Caller-saved temporary
- `qr1` (`f4-f7` / `ft4-ft7`) : Caller-saved temporary
- `qr2` (`f8-f11` / `fs0-fs3`) : Callee-saved
- `qr3` (`f12-f15` / `fa0-fa3`) : Caller-saved (Arguments)
- `qr4` (`f16-f19` / `fa4-fa7`) : Caller-saved (Arguments)
- `qr5` (`f20-f23` / `fs4-fs7`) : Callee-saved
- `qr6` (`f24-f27` / `fs8-fs11`) : Callee-saved
- `qr7` (`f28-f31` / `ft8-ft11`) : Caller-saved temporary

### 3.2 Width Modalities
- **QW8-QW64**: Fits inside `F` and `D` extension physical registers.
- **QW128**: Requires the RISC-V `Q` extension. Each FP register `f_i` is 128-bits wide, meaning `qr_k` perfectly fits a 512-bit QW128 state.
- **Hessian [[16, 4, 2]] Mapping**: 16 physical qubits encode into 4 logical qubits. This block of 4 logical qubits corresponds exactly to the 4 components of one Quaternion register (e.g., `qr0`).

---

## 4. Output C: Quantum Opcode Decision (`Xqbpqec`)

The Quantum / QEC instructions utilize the `custom-1` (`0x2B`) namespace.

**Rationale:** Quantum Pauli operators and syndrome extractions operate on symplectic $Z_2^{32}$ discrete parity representations, not continuous floating-point fields. Isolating them into `custom-1` allows routing to a dedicated bitwise/integer ALU, preserving the performance of the primary floating-point path.

| Mnemonic | funct7 | funct3 | rd, rs1, rs2 | Description |
|----------|--------|--------|--------------|-------------|
| `qbp.pauli`  | 0 | 000 | rd, rs1, rs2 | Apply Pauli rs2 to rs1 |
| `qbp.pcomm`  | 1 | 000 | rd, rs1, rs2 | Commutator test |
| `qbp.pweight`| 2 | 000 | rd, rs1 | Pauli weight of rs1 |
| `qbp.synd`   | 3 | 000 | rd, rs1, rs2 | Compute syndrome |
| `qbp.stab`   | 4 | 000 | rd, rs1, rs2 | Stabilizer check |
| `qbp.correct`| 5 | 000 | rd, rs1, rs2 | Apply correction |
| `qbp.qerr.w`   | 6 | w | rd, rs1 | Algebraic error: rd = \|1 - \|\|rs1\|\|²\| |
| `qbp.qdrift.w` | 7 | w | rd, rs1, rs2 | Accumulated drift rate |

---

## 5. Output D: Fano Orientation Recommendation

The `FANO` ROM must be fabricated with the **Conway-Smith standard orientation**:
`Lines: {1,2,4}, {2,3,5}, {3,4,6}, {4,5,7}, {5,6,1}, {6,7,2}, {7,1,3}`

**Rationale:** Fano orientation is an arbitrary gauge freedom internally, but acts as a strict protocol when sharing hypergraph embeddings. BMA Walk-phase memory traversals track associator defects `(xy)z - x(yz)`. If this graph is ever exported to a standard scalar CPU for analysis, that CPU will calculate using the Conway-Smith mathematical standard. Hardwiring the ROM to Conway-Smith prevents catastrophic sign-inversions during cross-platform validation.

---

## 6. Output E: Memory Model (`Xqbpmem`)

Custom memory operations use the `custom-2` (`0x5B`) opcode namespace.

### 6.1 Load / Store Operations
- `qbp.qload.w rd, imm(rs1)` (I-Type format, Opcode `0x5B`, `funct3=width`)
- `qbp.qstore.w rs2, imm(rs1)` (S-Type format, Opcode `0x5B`, `funct3=width`)

**Alignment Constraints:** QW8 (4-byte), QW16 (8-byte), QW32 (16-byte), QW64 (32-byte), QW128 (64-byte).
**Cache Policy:** Quaternion loads/stores should inherently utilize non-temporal hints (streaming bypass of L1 cache directly to L2/L3) since memory traversals like the Heisenberg spin-chain simulation involve linear, non-repeated sweeps of large datasets.

### 6.2 Boundary Packing
To transition data to and from analog sensor boundaries (SENSE/ACT):
- `qbp.qpack.16.128 rd, rs1`: Packs QW128 state down to QW16 for DAC extrapolation.
- `qbp.qunpack.16.128 rd, rs1`: Interpolates QW16 ADC signals up to QW128 for physics computation.

---

## 7. Trap & Exception Behavior

The QBP architecture adheres strictly to standard RISC-V exception handling semantics.

1.  **Illegal Instruction Trap**: Any `qbp.*` instruction utilizing an unsupported `funct3` width or addressing an invalid `qr` register grouping will immediately trigger an illegal instruction exception (mcause = 2). The hardware will **not** attempt to autonomously downgrade or guess the precision.
2.  **Unaligned Address Trap**: `qbp.qload` or `qbp.qstore` instructions must satisfy their alignment constraints (e.g., 64-byte alignment for QW128). Unaligned accesses trigger a Load/Store Address Misaligned exception (mcause = 4 or 6).
3.  **Constitutional Audit Interrupt**: If the hardware CTH Watchdog is enabled and the scalar norm of a quaternion deviates from 1.0 beyond the threshold ($10^{-30}$ for QW1024), the hardware triggers an asynchronous interrupt to notify the BMA of algebraic drift. It does not silently downgrade precision.

---

## 8. Output F: Open Risk Register

1. **Hessian Code Physical Error Rate vs FP Mantissa Flips**
   - *Question:* Does the [[16,4,2]] distance-2 discrete code sufficiently protect against floating-point bit flips inside the 128-bit `Q` registers over 172 days?
   - *Resolution:* Requires running a multi-million-iteration QW128 simulation with stochastic fault injection targeting lower mantissa bits.
   - *Emulation:* Can be modeled in software before HDL freeze.

2. **Octonion Implicit Associativity**
   - *Question:* Is left-associative chaining of `qbp.omac` instructions safe for compiling BMA's topological memory paths, or does the compiler need an explicit `qbp.ogroup` instruction to demarcate associator tracking blocks?
   - *Resolution:* Track the associator `[x, y, z]` across a mock BMA topological graph in software. If the compiler arbitrarily reorders independent mathematical nodes during `-O3` optimization, the graph will corrupt.
   - *Emulation:* Can be modeled by writing a test compiler pass.

## Theory Gaps
- **Tier 3 Assumption:** The belief that Hurwitz continuous norm protection (Layer 1) and Hessian discrete parity protection (Layer 2) are fully orthogonal and don't destructively interfere is *unproven*. If the correction logic shifts the state outside the acceptable Hurwitz algebraic manifold, it could trigger a catastrophic norm cascade. This requires empirical hardware fault-testing to promote to Tier 2.

---

## Appendix A: Revision History

| Version | Date | Description |
|---------|------|-------------|
| **v1.1** | 2026-05-04 | Added `qbp.` vendor prefixes to all mnemonics. Formalized `Xqbp*` extension hierarchy. Added Trap & Exception Behavior section. Removed explicit references to unratified mesh execution models pending `Xqbpvcp` draft. |
| **v1.0** | 2026-05-04 | Initial architecture mapping of Continuous Algebra, Quantum Subsystem, and Memory Model into RISC-V `custom-0/1/2` opcode spaces. Established Conway-Smith Fano orientation and Register Model aliases. |
