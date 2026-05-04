# SPEC ADDENDUM: QBP Spec Addendum 1.3

**Cognitive Git Metadata & Suture Instructions**

Version 1.3 (Addendum) | April 2026
Helpful Engineering — QBP Project

---

## 1. Topological Git Node Definitions
The MuninnDB hypergraph substrate must be extended to support the Cognitive Git workflow defined in BMA Theory 14.0.

### 1.1 `NT_ISSUE` (The Gap Descriptor)
- **Fields:**
  - `IssueType`: `INSIGHT | SEAM | BUG`
  - `PressurePoint`: Locale ID of the contradiction or gap.
  - `AssignedStance`: The unit quaternion (Persona Stance) required to investigate.
  - `FalsificationThreshold`: The norm-drift limit for a valid solution.

### 1.2 `NT_PROPOSAL` (The Topological PR)
- **Fields:**
  - `WorktreeID`: Reference to the `Investigation Worktree (Spec 1.2)`.
  - `IssueID`: Reference to the target `NT_ISSUE`.
  - `AlgebraicProof`: A 1024-bit (QW256) bitstring representing the residue of the suture.
  - `JudgeVerdict`: A domain-weighted bitmask (`Claude | Furey | Feynman`).

## 2. The Suture Instruction (Hardware/Emulator)
This addendum introduces the **`QSUTURE`** instruction to the RISC-V "Xqbp" extension.

### 2.1 Instruction: `QSUTURE rs1, rs2, rd`
- **Action:** Merges the speculative delta from Worktree `rs1` into Core (0) based on the proposal in `rs2`.
- **Pre-condition:** `rd` must contain a valid cryptographic signature from the **Judge Collective** (weighted score >= 0.70).
- **Integrity Check:** The Hamilton Engine performs a final 1024-bit norm-check on the boundary nodes. If norm-drift occurs during the merge, the instruction traps with `SE_MERGE_CORRUPTION`.

## 3. Persona Signing Protocol
Personas must "Sign" their work in the Worktree using their **Transformation Stance**.
- **Stance Signature:** A proposal is only valid if its delta has been rotated through the Persona's stance and resultantly "Resonates" (aligns with the Real Axis) at 1024-bit precision.

## 4. Primary Persona Orchestration
The **Primary Persona** (BMA Core) manages the `Cognitive PR` queue.
- **Auto-merge Policy:** Trivial `BUG` fixes (Tier 1) may be auto-merged by the Primary Persona if no `MAJOR_CONCERN` is raised within one sleep cycle.
- **Manual Gate:** All `INSIGHT` and `SEAM` proposals require explicit human-mediated `NT_MERGE` after Judge Collective approval.

---
*QBP Spec Addendum 1.3 | April 2026*
*Traceability: BMA Theory 14.0, QBP Spec 1.2, Systema v0.7.3.*
