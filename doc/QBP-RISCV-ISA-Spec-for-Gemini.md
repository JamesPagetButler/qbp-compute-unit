# QBP Compute Unit — RISC-V ISA Specification Task
**For:** Gemini (theory generator)
**From:** James Paget Butler + Claude (Red Team)
**Date:** 2026-04-11
**Output requested:** Complete RISC-V ISA specification for the QBP Compute Unit

---

## Context

The QBP (Quaternion-Based Physics) Compute Unit is a research compute architecture
that uses quaternion and octonion algebra as its native data types. The key insight:
quaternion multiplication is *algebraically* norm-preserving (`||qr|| = ||q|| ||r||`
by Hurwitz theorem), so norm drift in a computation path is a structural signal of
error, not just numerical noise. This is the basis of the algebraic watchdog.

An existing partial ISA has been designed in Go source code (`pkg/emu/emu.go`,
`pkg/qword/qword.go`). Your task is to complete it into a full specification.

---

## 1. What Already Exists (Do NOT redesign these)

### 1.1 Encoding Convention

- **Opcode space:** RISC-V custom-0 (`0x0B`)
- **Format:** R-type (`funct7 | rs2 | rs1 | funct3 | rd | opcode`)
- **funct3:** width selector (`000`=QW8, `001`=QW16, `010`=QW32, `011`=QW64, `100`=QW128)
- **funct7:** operation selector
- **Register file:** FP registers f0–f31; one QW64 quaternion occupies 4 consecutive registers

### 1.2 Defined Instructions (funct7 values 0–4)

```
QMUL.{w}  rd, rs1, rs2    ; Hamilton product (16 MUL + 12 ADD per QW64)
QROT.{w}  rd, rs1, rs2    ; Rotation qvq* (two QMUL)
OMAC.{w}  rd, rs1, rs2    ; Octonionic multiply-accumulate (8 components, Fano LUT)
FANO      rd, rs1, rs2    ; Fano plane LUT: e_i × e_j → (index, sign)
QNORM.{w} rd, rs1         ; Norm squared ||q||² = w²+x²+y²+z²
```

### 1.3 Quaternion Word Format (non-negotiable)

| Width | Components | Bits | Algebraic life @ 1 GHz |
|-------|------------|------|------------------------|
| QW8 | 4 × int8 | 32 | < 1 operation |
| QW16 | 4 × int16 | 64 | ~38 operations |
| QW32 | 4 × fp32 | 128 | ~72 ms |
| QW64 | 4 × fp64 | 256 | ~7 seconds |
| QW128 | 4 × fp128 | 512 | ~172 days |
| QW256 | 4 × fp256 | 1024 | effectively infinite |

**Physics starting point: QW128.** Scale down for throughput, up for duration.

---

## 2. What You Need to Design

### 2.1 Complete the Quaternion Instruction Set

The following operations exist in `pkg/quat/quat.go` — each Go function is
explicitly documented as mapping to "one instruction in the Run-phase RISC-V ISA."
Assign funct7 values, define operand semantics, and specify cycle counts for each width.

```
QADD.{w}   rd, rs1, rs2    ; q + r
QSUB.{w}   rd, rs1, rs2    ; q - r
QSCALE.{w} rd, rs1, rs2    ; scalar × q (rs2 holds scalar)
QDOT.{w}   rd, rs1, rs2    ; 4D dot product → scalar
QCONJ.{w}  rd, rs1         ; conjugate q* = w - xi - yj - zk
QINV.{w}   rd, rs1         ; inverse q⁻¹ = q* / ||q||²
QEXP.{w}   rd, rs1         ; exp(q) — used for time evolution
QLOG.{w}   rd, rs1         ; log(q)
QMAC.{w}   rd, rs1, rs2    ; multiply-accumulate: rd += rs1 × rs2
```

### 2.2 Complete the Octonion Instruction Set

These complete the `pkg/octonion/octonion.go` API:

```
OADD.{w}   rd, rs1, rs2    ; a + b (8 additions)
OSUB.{w}   rd, rs1, rs2    ; a - b
OSCALE.{w} rd, rs1, rs2    ; scalar × o
OCONJ.{w}  rd, rs1         ; conjugate (negate imaginary components)
ONORM.{w}  rd, rs1         ; norm squared ||o||² (8-component)
```

### 2.3 Memory Instructions

No load/store instructions exist. Define them.

Requirements:
- Quaternion load/store: aligned, width-parameterised
- Octonion load/store: aligned, 2× wider than quaternion equivalent
- Width conversion (QPACK): for SENSE boundary (sensor ADC → QW16)
  and ACT boundary (QW128 → DAC/actuator QW16)

### 2.4 Width Extension to QW256

funct3 currently uses `100` (QW128) as its maximum.
`101` through `111` are available. Define the encoding for QW256.
Note: QW256 uses 4 × fp256, which has no hardware support on any current
CPU — this is a software emulation target for Walk/Run phases.

### 2.5 Quantum Gate Extension (New Opcode Space)

**Decision to make:** Should quantum instructions share custom-0 with quaternion
instructions (more funct7 values) or use custom-1 (`0x2B`) as a separate
namespace? Argue your choice.

The quantum package (`pkg/quantum/`) has:

#### Pauli operators
```go
type Pauli uint32   // symplectic representation in Z2^32
                    // bits 0-15: X part, bits 16-31: Z part
func (p Pauli) Weight() int       // Hamming weight of X|Z combined
func (p Pauli) Commutes(q Pauli) bool  // symplectic inner product mod 2
```

#### Hessian [[16, 4, 2]] stabilizer code
- 16 physical qubits → 4 logical qubits
- 12 stabilizers (8 local + 4 cross-block)
- Distance 2: detects any single-qubit error
- Combined with Hurwitz norm protection (layer 1) and Z2 parity (layer 2),
  gives effective distance ≥ 3

**Design the following quantum instructions:**

```
; Pauli operations
PAULI   rd, rs1, rs2  ; Apply Pauli operator rs2 to state rs1 → rd
PCOMM   rd, rs1, rs2  ; Commutator: rd = (rs1.commutes(rs2)) ? 1 : 0
PWEIGHT rd, rs1       ; Pauli weight: rd = popcount(X_part | Z_part)

; Syndrome and correction
SYND    rd, rs1, rs2  ; Compute error syndrome: measure stabilizers rs2 against state rs1
STAB    rd, rs1, rs2  ; Stabilizer check: does state rs1 satisfy stabilizer rs2?
CORRECT rd, rs1, rs2  ; Apply correction from syndrome rs2 to state rs1

; Algebraic watchdog integration
QERR    rd, rs1       ; Algebraic error signal: rd = |1 - ||rs1||²|  (fast norm watchdog)
QDRIFT  rd, rs1, rs2  ; Drift rate: rd = accumulated drift(rs1) over rs2 ops
```

**Important constraint:** The Hessian code uses 16 physical qubits for 4 logical.
Each logical qubit is a quaternion state. Define how they map to the register file.

### 2.6 Register Allocation for Wide Quaternions

The RISC-V specification says:
- F extension: 32 × 32-bit FP registers (f0–f31)
- D extension: 32 × 64-bit FP registers (widen F registers)
- Q extension: 32 × 128-bit FP registers (widen D registers)

Current emulator uses 4 consecutive f64 registers per QW64 quaternion
(e.g., f0=W, f1=X, f2=Y, f3=Z). This is a convention, not enforced by hardware.

**Design a register allocation model that:**
1. Works for QW8 through QW256
2. Is ABI-compatible (caller/callee-saved convention for quaternion registers)
3. Specifies the register naming convention (e.g., `qr0` through `qr7` as
   quaternion register aliases for consecutive FP registers)
4. Handles the QW128 case where each component is 128 bits
   (does this require the Q extension, or can you use register pairs in D?)

### 2.7 Fano Orientation Selection

`pkg/fano/fano.go` uses this standard orientation:

```
Lines: {1,2,3}, {1,4,5}, {1,7,6}, {2,4,6}, {2,5,7}, {3,4,7}, {3,6,5}
```

This is one of 480 valid orientations. The source explicitly flags this as an
open question: "it may be a gauge freedom or a design parameter."

**Answer the following:**
1. Is the choice of Fano orientation physically meaningful for QBP's application
   (BMA memory traversal, spin-chain simulation)? Does it affect the output of
   `AssociativityDefect(a,b,c)` in any detectable way?
2. If it is a gauge freedom: recommend the canonical choice and justify it
   (e.g., the standard Conway-Smith orientation, or one with particularly
   clean binary encoding for hardware).
3. If it is a design parameter: what observable changes with different orientations,
   and is there a way to test which is "better" without a full QBP experiment?

### 2.8 Non-Associativity Handling for Octonions

Octonion multiplication is non-associative: `(ab)c ≠ a(bc)` in general.
A standard RISC-V instruction stream has no implied grouping — the programmer
controls order via instruction sequencing.

**Question:** For the `OMAC` instruction (dest += a × b), the accumulation
order is implicit in the instruction stream. Is this sufficient, or does the
ISA need an explicit grouping hint (e.g., a `OGROUP` instruction that marks
the start of an associativity-sensitive computation block)?

In BMA memory traversal, traversal path a→b→c→d is represented as
`(((e_a · e_b) · e_c) · e_d)` — left-associative by default. Is this the
right default for all use cases?

---

## 3. Required Outputs

Provide the following as sections of your response:

### Output A: Complete Instruction Table

A table of ALL instructions (existing + new) with:
- Mnemonic
- funct7 value (for custom-0) or opcode (for custom-1 quantum)
- funct3 usage
- Operand semantics (rd, rs1, rs2)
- Cycle count at QW8, QW32, QW64, QW128
- Epistemic tier (T0=axiom / T1=proven / T2=measured / T3=predicted)

### Output B: Register Model

Complete register allocation specification:
- Register naming (QR aliases for FP register groups)
- Width-to-register-count mapping
- Calling convention (which QRs are caller-saved, callee-saved)
- How the 16-qubit Hessian state maps to registers

### Output C: Quantum Opcode Decision

Explicit recommendation: custom-0 extension OR custom-1 new namespace.
Full encoding table for quantum instructions.

### Output D: Fano Orientation Recommendation

A definitive answer to the gauge-freedom question with a justified recommendation
for the hardwired orientation.

### Output E: Memory Model

Load/store instruction definitions with:
- Alignment requirements per width
- Cache behavior recommendations (quaternion data is typically streaming)
- QPACK encoding for SENSE/ACT boundary conversions

### Output F: Open Risk Register

A list of unresolved questions that require hardware validation before the ISA
can be considered frozen. For each:
- The question
- What experiment resolves it
- Whether it can be emulated in software first

---

## 4. Constraints and Non-Goals

**Hard constraints:**
- Must be RISC-V compatible (custom opcode spaces only; no new base ISA changes)
- Must support the existing instruction encoding without breaking changes
- funct3 width selector must remain backward-compatible
- The FANO ROM is hardwired at fabrication — the orientation decision is irreversible
- QW8 must be usable for BMA hypergraph traversal (16.6× speed win is measured)
- QW128 is the physics starting point (172-day algebraic lifetime at 1 GHz)

**Non-goals for this spec:**
- SurrealDB or MuninnDB integration (that is BMA's job)
- Operating system ABI (Linux RISC-V ABI for GP registers is unchanged)
- Floating-point exception handling (rely on RISC-V F/D exception model)
- Specific silicon process or area targets

---

## 5. Theory References

The mathematical foundations, in order of epistemic confidence:

1. **Hurwitz theorem** — The only normed division algebras over ℝ are ℝ, ℂ, ℍ, 𝕆.
   This is the axiomatic foundation for the entire instruction set.
   Lean 4 proof exists for the quaternion case (PROOF-hurwitz-quat in QBP inventory).

2. **Norm multiplicativity** — `||qr|| = ||q|| ||r||` for quaternions and octonions.
   This is what makes the algebraic watchdog possible.
   Verified empirically in `pkg/octonion.NormMultiplicativity()`.

3. **Fano plane** — The 7-point projective geometry governing e_i × e_j products.
   98-byte ROM. Verified algebraically in `pkg/fano.Verify()`.

4. **Heisenberg spin-chain** — Time evolution under H = J Σ Sᵢ·Sᵢ₊₁ maps to
   quaternion rotations. QBP simulation maintains norm by algebraic structure;
   scalar simulation requires explicit renormalisation.
   Benchmarked in `pkg/spinchain.RunBenchmark()`.

5. **Hessian [[16,4,2]] code** — CSS construction with 12 stabilizers for
   4 logical qubits at distance 2. Code construction verified (`Code.Verify()`).
   Physical error rates on real hardware: **untested** (Tier 3 prediction).

---

## 6. Deliverable Format

Please structure your response as the five Output sections (A–F) listed above,
followed by a short "Theory Gaps" section listing any mathematical claims in
this spec that you assess as unproven, under-specified, or likely to require
revision after hardware validation.

If you believe any of the design choices in the existing ISA are wrong or
suboptimal, flag them explicitly. We value clean negative results over
false confirmation.

---

*End of specification document*
