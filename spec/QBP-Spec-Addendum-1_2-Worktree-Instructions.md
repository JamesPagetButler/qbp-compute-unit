# SPEC ADDENDUM: QBP Spec Addendum 1.2

**Technical Requirements for Investigation Worktrees**

Version 1.2 (Addendum) | April 2026
Helpful Engineering — QBP Project

---

## 1. Worktree Metadata Support
The MuninnDB hypergraph substrate must support native Worktree isolation to facilitate BMA "Playground" investigations.

### 1.1 Node & Edge Tagging
- **WorktreeID:** All nodes and edges in the `pkg/gap` and `pkg/mesh` packages shall include a `uint64 WorktreeID`.
- **Core Namespace:** A `WorktreeID` of `0` is reserved for the authoritative Core CTH. 
- **Inheritance:** When forking, a new WorktreeID is generated. Reads from the worktree will recursively check the parent WorktreeID until Core (0) is reached (Copy-on-Write semantics).

## 2. Kernel-Level Copy-on-Write (CoW)
To ensure the integrity of the core models during speculation, the Hamilton Engine must implement CoW.
- **Speculative Mutation:** Any algebraic instruction (`QMUL`, `QROT`, `QINV`) targeting a Core node while a non-zero `WorktreeID` is active MUST generate a new overlay node in the Worktree namespace.
- **Memory Footprint:** Only the delta (mutated nodes) shall consume prestige-mode memory.

## 3. Hardware-Accelerated Resolution Gradient
The Q-Mem controller shall implement the **Systema Resolution Gradient** to manage memory bandwidth.
- **Focal Node Addressing:** High-precision requests (QW128/QW256) are honored for nodes within the active Worktree focal range.
- **Mean-Field Aggregation:** For out-of-worktree (background) requests, the controller shall return a low-precision `Quat8` aggregate representing the "Mean Field" of that locale, bypassing the need to fetch high-precision data for context.

## 4. Worktree Instruction Extensions
This addendum proposes the following RISC-V "Xqbp" instructions for the Walk Phase:
- `WFORK rd, rs1`: Fork current context into a new Worktree ID stored in `rd`.
- `WMERGE rs1`: Propose the merge of Worktree `rs1` to Core (triggers Governance Audit).
- `WGRAD rs1, rd`: Fetch the Resolution Gradient state of locale `rs1`.

---
*QBP Spec Addendum 1.2 | April 2026*
*Traceability: Systema v0.7.3 Worktree primitive, QBP Architecture v1.0.*
