# SPEC ADDENDUM: QBP Spec Addendum 1.5

**Honing Metadata & Prompt Synthesis Kernels**

Version 1.5 (Addendum) | April 2026
Helpful Engineering — QBP Project

---

## 1. Honing Metadata in MuninnDB
To support the "Cognitive Honing Protocol," we introduce the `NT_HONING_LOG` node.

### 1.1 `NT_HONING_LOG` Schema
- `ParentIssueID`: Reference to the original `NT_ISSUE`.
- `DialogueChain`: A linked list of beekeeper-primary exchanges.
- `RefinedManifold`: A list of locale IDs that have been "promoted" during triangulation.
- `SynthesisSignature`: A 1024-bit bitstring verifying that the honed question resonates with the core CTH.

## 2. The "Honing Trap" Instruction
This addendum introduces the **`QHON`** (Cognitive Hone) signal to the RISC-V "Xqbp" extension.

### 2.1 Instruction: `QHON rs1, rd`
- **Action:** Triggers a context-switch back to the Primary Persona for task refinement.
- **Trigger Condition:** Used by specialized personas when the "Algebraic Impedance" of a task is too high (i.e., the question is too vague to resolve).
- **Result:** Suspends the current Worktree and emits an `SE_HONING_REQUIRED` event to the Beekeeper interface.

## 3. Automated Brief Generation
The **ContextPreparer** is extended with a **Brief Synthesis Kernel**.
- This kernel uses the `NT_HONING_LOG` to automatically prepend a **Topological Brief** to any prompt sent to a Persona.
- **Alignment Masking:** The brief is filtered by the target persona's `RelevanceMask` (Spec 1.4) to ensure no scale-bogging occurs.

---
*QBP Spec Addendum 1.5 | April 2026*
*Traceability: BMA Theory 16.0, QBP Spec 1.4, Systema v0.7.3.*
