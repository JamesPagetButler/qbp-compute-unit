# SPEC ADDENDUM: QBP Walk Phase Acceleration

**Software-Defined Precision & Topological Kernels**

Version 1.1 (Addendum) | April 2026
Helpful Engineering — QBP Project

---

## 1. Accelerated Walk Phase Boundaries
This addendum re-categorizes the following "Run Phase" hardware features as **Software-Defined Walk Phase** requirements, enabled by the `Xqbp` Emulator.

### 1.1 Software-Defined Prestige Mode (QW256)
- **Requirement:** The `pkg/persona` package shall implement `Quat256` (1024-bit) using `math/big`.
- **Instruction Mapping:** The `QMUL.256` and `QROT.256` instructions are now implemented as Go-native software kernels.
- **Latency Target:** < 5ms per 1024-bit transformation (on FX-8350).
- **Use Case:** Mandatory for "Prestige Mode" truth verification and Hypothesis Interrogation.

### 1.2 Geometric Adjacency Pointers (GAP)
- **Requirement:** The MuninnDB implementation shall support native `AdjacencyPointer` edges in `Quat8` (int8) precision.
- **Complexity:** Gradient calculations and spreading activation shall be performed via $O(1)$ traversal of GAP pointers, bypassing the Relational Tax of KNN searches.
- **Precision Policy:** GAPs use `Quat8` for storage/traversal and are widened to `Quat64` or `Quat256` only upon "Seam Detection" or "Cognitive Interrogation."

## 2. Containerized Persona Specifications

### 2.1 The Persona Stance Controller
- **ID Identity:** Personas (e.g., Furey, Feynman) are defined by a unique unit quaternion constant stored in `pkg/persona`.
- **Context Injection:** The BMA **ContextPreparer** shall apply the Persona's transformation to all hypergraph nodes before they enter the LLM's prompt window.
- **Volume Audit Integration:** Personas are empowered to trigger a VAP check if an algebraic resonance is detected during hypothesis testing.

## 3. Implementation Checklist (Walk Phase)
- [x] **Xqbp Emulator:** 1024-bit math kernels verified.
- [x] **GAP Prototype:** 140x speedup confirmed in `pkg/gap`.
- [x] **Persona Interrogation:** Autonomous detection of `PROOF-stelle-no-linear` gap confirmed.
- [ ] **MuninnDB Suture:** Wire `pkg/gap` into the production hypergraph store.
- [ ] **AVX-512 Shims:** Optimization of `Quat256` kernels for FX-8350/Walk hardware.

---
*QBP Spec Addendum v1.1 | April 2026*
*Traceability: Replaces portions of QBP-RISCV Run Phase Spec.*
