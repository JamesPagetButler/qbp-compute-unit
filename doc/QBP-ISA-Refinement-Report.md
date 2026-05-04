# QBP Compute Unit — ISA Refinement Report
**Date:** 2026-04-11
**Author:** James Paget Butler + Claude (Red Team)
**Purpose:** Context document for Gemini ISA specification task

---

## 1. State of the ISA Design

The QBP Compute Unit source already contains a substantially designed partial ISA.
This report summarises what exists, what is missing, and the epistemic standing of
each component according to CTH theory.

### 1.1 Instruction Set — Implemented (5 instructions)

Source: `pkg/emu/emu.go`, encoding complete, emulator functional.

| Mnemonic | funct7 | Description | Epistemic tier |
|----------|--------|-------------|----------------|
| `QMUL.{w}` | 0 | Hamilton product q × r | T1 — Hurwitz proof |
| `QROT.{w}` | 1 | Rotation qvq* | T1 — derived from QMUL |
| `OMAC.{w}` | 2 | Octonionic multiply-accumulate | T1 — Fano verified |
| `FANO` | 3 | Fano plane LUT lookup | T0 — axiomatic (98-byte ROM) |
| `QNORM.{w}` | 4 | Norm squared \|\|q\|\|² | T1 — derived |

**Width selector (funct3):** `000`=QW8 · `001`=QW16 · `010`=QW32 · `011`=QW64 · `100`=QW128

**Encoding:** RISC-V R-type, custom-0 opcode `0x0B`.

### 1.2 Word Format — Specified (`pkg/qword/qword.go`)

| Mnemonic | Bits | Components | Alg. life @ 1 GHz | Use case |
|----------|------|------------|-------------------|---------- |
| QW8 | 32 | 4 × int8 | < 1 op | Hypergraph edges |
| QW16 | 64 | 4 × int16 | ~38 ops | Sensor ingestion |
| QW32 | 128 | 4 × fp32 | ~72 ops | GPU-native batch |
| QW64 | 256 | 4 × fp64 | ~7 sec | Interactive compute |
| QW128 | 512 | 4 × fp128 | ~172 days | Physics default |
| QW256 | 1024 | 4 × fp256 | >> years | Extended autonomy |

Octonion words (OW8 through OW128) = 8-component equivalents.

### 1.3 Pipeline Config — Specified (`pkg/emu/emu.go`)

| Instruction | QW8 | QW16 | QW32 | QW64 | QW128 | QW256 |
|-------------|-----|------|------|------|-------|-------|
| QMUL | 1 | 1 | 1 | 1 | 2 | 4 |
| QROT | 2 | 2 | 2 | 2 | 4 | 8 |
| OMAC | 1 | 2 | 2 | 3 | 6 | — |
| FANO | 1 (width-independent) | | | | | |
| QNORM | 1 | 1 | 1 | 1 | 2 | — |

Clock target: 100 MHz (130 nm OpenMPW conservative).

### 1.4 Algebraic Lifetime — Empirically Validated (`pkg/spinchain/extwidth.go`)

| Width | DriftPerOp | 1e-6 depth | Time @ 1 GHz |
|-------|-----------|-----------|-------------|
| QW32 | ~1.4e-14 | ~7.3e7 | 73 ms |
| QW64 | ~1.36e-16 | ~7.4e9 | 7.4 sec |
| QW128 | ~2.9e-68 | ~3.5e64 | 3.5e46 sec (>>universe) |
| QW256 | ~2.0e-145 | >>1e139 | infinite for any purpose |

### 1.5 Mesh Scheduler — Specified (`pkg/mesh/mesh.go`)

Fano-topology mesh of 7 nodes. Width is dynamically allocated per task:
- Task declares: `CompositionDepth`, `DriftTolerance`
- Scheduler computes: required `qword.Width` → number of nodes per group
- Watchdog monitors: actual `CumulativeNormDrift` at runtime

### 1.6 Algebraic Watchdog — Specified (`pkg/watchdog/watchdog.go`)

Monitors `CumulativeNormDrift`, `MaxNormDrift`, `Curvature` per computation path.
Triggers re-allocation when actual drift exceeds predicted tolerance.

### 1.7 Quantum Error Correction Layer — Partial (`pkg/quantum/`)

**Hessian [[16,4,2]] stabilizer code** implemented:
- 16 physical qubits → 4 logical qubits, distance 2
- 12 stabilizers (8 local + 4 cross-block)
- Pauli symplectic representation: `type Pauli uint32`, Z₂³² space
- `Code.Verify()` and `Code.Distance()` implemented in Go
- **No ISA instructions yet.** The code exists in software only.

**Multi-layer protection theory** (`pkg/quantum/protection.go`):
- Layer 1: Hurwitz norm (non-unitarity detected algebraically, free)
- Layer 2: Z2 parity (sign-flip topological errors)
- Layer 3: Hessian stabilizer (Pauli errors, d≥2)

---

## 2. Gaps Identified

### 2.1 Missing Arithmetic Instructions (~15 instructions)

From `pkg/quat/quat.go` (each function maps to one RISC-V instruction):

| Operation | Source function | Missing instruction |
|-----------|-----------------|---------------------|
| q + r | `quat.Add` | `QADD.{w}` |
| q − r | `quat.Sub` | `QSUB.{w}` |
| s × q | `quat.Scale` | `QSCALE.{w}` |
| q·r (4D dot) | `quat.Dot` | `QDOT.{w}` |
| q* (conjugate) | `quat.Conj` | `QCONJ.{w}` |
| q⁻¹ (inverse) | `quat.Inv` | `QINV.{w}` |
| exp(q) | `quat.Exp` | `QEXP.{w}` ← spinchain time evolution |
| log(q) | `quat.Log` | `QLOG.{w}` |
| dest += q×r | `quat.MulAccum` | `QMAC.{w}` ← explicitly called out in source |
| QW8 Hamilton | `quat.Mul8` | `QMUL.8` variant with int8 saturation |

From `pkg/octonion/octonion.go`:

| Operation | Source function | Missing instruction |
|-----------|-----------------|---------------------|
| a + b | `octonion.Add` | `OADD.{w}` |
| a − b | `octonion.Sub` | `OSUB.{w}` |
| s × o | `octonion.Scale` | `OSCALE.{w}` |
| o* | `octonion.Conj` | `OCONJ.{w}` |
| \|\|o\|\|² | `octonion.NormSq` | `ONORM.{w}` |
| verify \|\|ab\|\|=\|\|a\|\|\|\|b\|\| | `octonion.NormMultiplicativity` | watchdog op |

### 2.2 Missing Memory Instructions

No load/store instructions defined. Every real ISA needs them:

| Instruction | Description |
|-------------|-------------|
| `QLOAD.{w} rd, imm(rs1)` | Load quaternion from memory |
| `QSTORE.{w} rs2, imm(rs1)` | Store quaternion to memory |
| `OLOAD.{w} rd, imm(rs1)` | Load octonion from memory |
| `OSTORE.{w} rs2, imm(rs1)` | Store octonion to memory |
| `QPACK.{w_from}.{w_to} rd, rs1` | Width conversion (SENSE/ACT boundary) |

### 2.3 Missing Quantum Gate Instructions (New opcode space needed)

The quantum package defines operations but no ISA instructions.
These likely need a separate opcode (custom-1 = `0x2B`) to keep the
quaternion and quantum namespaces clean.

| Instruction | Description | Source |
|-------------|-------------|--------|
| `PAULI rd, rs1, rs2` | Apply Pauli operator to state | `quantum.Pauli` type |
| `PCOMM rd, rs1, rs2` | Test Pauli commutativity | `Pauli.Commutes()` |
| `PWEIGHT rd, rs1` | Compute Pauli weight | `Pauli.Weight()` |
| `SYND rd, rs1, rs2` | Compute error syndrome | Hessian code |
| `STAB rd, rs1, rs2` | Check stabilizer | `Code.Verify()` |
| `QGATE rd, rs1, rs2` | Apply SU(2) gate to spinor | `spinchain` |
| `QERR rd, rs1` | Algebraic error signal (norm drift) | `watchdog` |

### 2.4 Missing Width Extension (QW256)

`qword.go` defines QW256 but `emu.go` only encodes funct3 up to `100` (QW128).
funct3 has 3 bits → `101`=QW256 is available.

### 2.5 Fano Orientation is Undefined

`pkg/fano/fano.go` uses one of 480 valid Fano plane orientations.
The spec explicitly calls this "an open question" and notes it "may be a
gauge freedom or a design parameter." This needs a decision before silicon:
the FANO ROM is hardwired at fabrication.

---

## 3. CTH Epistemic Analysis

Using CTH theory to classify the instruction set by epistemic tier:

| Group | Instructions | CTH tier | Confidence |
|-------|-------------|---------|------------|
| Quaternion core | QMUL, QROT, QCONJ, QNORM, QMAC | T1 | Hurwitz theorem — Lean-verified |
| Octonion core | OMAC, OADD, OCONJ, ONORM | T1 | Fano + norm-multiplicativity |
| FANO ROM | FANO | T0 | Axiomatic — algebraic definition |
| Time evolution | QEXP, QLOG | T2 | Spinchain benchmark (empirical) |
| Precision mgmt | QPACK, width selector | T2 | Composition benchmark data |
| Quantum gates | QGATE, PAULI | T3 | Prediction — awaiting experiment |
| QEC (Hessian) | SYND, STAB, CORRECT | T3 | [[16,4,2]] code unverified on hardware |

**Critical gap:** The full QBP inventory (production-scale, not the 4-anchor stub)
is needed to run `cth analyse` and get meaningful:
- ρ_net (target 0.765) — confirms the theory has more confirmatory information than deficit
- Bridge nodes — identifies which anchors span multiple instruction domains
- Sediment analysis — flags which instruction groups have low-fidelity derivation chains
- Sensitivity bracket — tells us how robust the ISA design is to changes in axiom assumptions

The stub `qbp_v3_2.json` is sufficient to test the CTH tools but not to drive ISA priorities.

---

## 4. Open Questions for Gemini

1. **Fano orientation:** Which of the 480 valid orientations should be hardwired? Is there a physical or mathematical reason to prefer one? Or is this a free parameter that affects computation in a measurable way?

2. **Quantum opcode space:** Custom-1 (`0x2B`) for quantum ops vs. extending custom-0? The concern is that custom-1 requires separate decode hardware.

3. **Register model for QW128:** Using the RISC-V Q extension (128-bit registers) is the natural fit, but Q is rarely implemented. Alternative: 4 consecutive D (64-bit FP) registers as a software convention. Trade-offs?

4. **QEXP/QLOG precision:** These use `math.Sqrt` and `math.Acos` internally. At QW128, these transcendental functions need software emulation (no hw support for fp128 trig on any current CPU). Should QEXP/QLOG be software-only ops in Crawl/Walk with hardware deferred to Run?

5. **Quantum extension completeness:** The Hessian [[16,4,2]] code gives d=2 (detects 1 error, corrects none). Is this sufficient for the intended use case (norm watchdog + z2 parity already provide layers 1 and 2)? Or should we aim for d=3?

6. **Non-associativity handling (OMAC):** Octonion multiplication is non-associative. The RISC-V architecture assumes all operations are individually correct (no implicit grouping). But OMAC results depend on application order. Does the ISA need an explicit `OGROUP` instruction or a grouping metadata field?

---

## 5. Full QBP Inventory — What It Would Add

Running `cth analyse testdata/qbp_full.json` with a production inventory would give:
- **ρ_net = 0.765** validation (confirms CTH tools work on real data)
- **Bridge nodes:** Likely PROOF-hurwitz-quat (T1, bridges lean↔lab↔math) — confirms QMUL/QROT are the most cross-domain-critical instructions
- **Sediment:** Identifies which prediction chains (Tier 3 ops) have lowest fidelity — flags the quantum extension group
- **Localisation:** For any FLAG-J-style incoherence in the theory, `cth localise` traces it to the specific derivation step — maps directly to which instruction needs more theory backing

Recommendation: the Gemini spec below can be executed without the full inventory. Populate the inventory in parallel and run `cth analyse` when available to validate priorities.

---

*Document generated: 2026-04-11*
