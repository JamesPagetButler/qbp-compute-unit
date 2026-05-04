# SPEC ADDENDUM: QBP Spec Addendum 1.4

**Stance-Based Gating & Relevance Metadata**

Version 1.4 (Addendum) | April 2026
Helpful Engineering — QBP Project

---

## 1. Relevance Tags in MuninnDB
To support "Reciprocal Focus," all hypergraph nodes and edges shall include a `RelevanceMask` (uint64 bitfield).

### 1.1 Category Bits
- `0x01`: Atmospheric / Climate
- `0x02`: Hydrological / Watershed
- `0x04`: Biological / Population
- `0x08`: Chemical / Contaminant
- `0x10`: Anthropogenic / Development

## 2. Hardware-Level Stance Gating
The `QROT` (Quaternion Rotation) and `QMUL` (Quaternion Multiply) instructions are extended with a **Relevance Mask** check.

### 2.1 Instruction Logic: `QROT_GATED rd, rs1, mask`
- **Execution:** 
  ```
  If (node.RelevanceMask & mask) == 0:
      rd = Identity (1, 0, 0, 0)
  Else:
      rd = Standard_QROT(node, rs1)
  ```
- **Rationale:** By returning Identity for non-relevant nodes, the hardware ensures that "Background Noise" has zero mathematical impact on the current worktree accumulation, effectively automating **Lossless Dismissal**.

## 3. Zero-Tax Context Switching
The compute unit's **Stance Controller** shall maintain the active `mask`.
- Switching from a "Loon Focus" (mask `0x06`) to a "Hurricane Focus" (mask `0x01`) is a single-cycle register update. 
- This allows a Persona to shift its "Cognitive Lens" without reloading or re-partitioning the hypergraph memory.

---
*QBP Spec Addendum 1.4 | April 2026*
*Traceability: BMA Theory 15.0, QBP Spec 1.2.*
