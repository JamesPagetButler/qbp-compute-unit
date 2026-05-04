# SPEC ADDENDUM: QBP Spec Addendum 1.6

**Signal Surface Monitoring & QSCAN Kernels**

Version 1.6 (Addendum) | April 2026
Helpful Engineering — QBP Project

---

## 1. `NT_SIGNAL` (The Noteworthy Node)
MuninnDB is extended with the `NT_SIGNAL` node to capture proactive persona scouting.

### 1.1 `NT_SIGNAL` Schema
- `DetectingPersonaID`: Reference to the persona who identified the signal.
- `ExternalSourceRef`: URL, DOI, or sensor stream identifier.
- `ResonanceScore`: A float64 [0-1] representing algebraic alignment with the active CTH.
- `RelevanceRationale`: A text/semantic summary of why the signal is "noteworthy."

## 2. Background Scouting Instruction: `QSCAN`
This addendum introduces the **`QSCAN`** instruction to the RISC-V "Xqbp" extension.

### 2.1 Instruction: `QSCAN rs1, rs2, mask`
- **Action:** Performs a low-precision (QW8) dot-product between an external data stream `rs1` and the active project nodes `rs2`, filtered by `mask`.
- **Low-Energy Threshold:** `QSCAN` is designed to run in background cycles (during "Sleep Consolidation") without waking the 1024-bit prestige-mode circuits.
- **Trap:** If the dot-product exceeds the `ResonanceThreshold`, the instruction emits an `NT_SIGNAL` event.

## 3. The Noteworthy Dashboard (HUD)
The Beekeeper interface shall include a **Noteworthy Dashboard** that aggregates `NT_SIGNAL` nodes.
- **Synthesis:** The Primary Persona's "Brief Synthesis Kernel" (Spec 1.5) shall produce a 3-sentence summary for each signal upon Beekeeper login.
- **Action Gate:** Every signal must provide a single-click "Fork Worktree" button to immediately investigate the resonance.

---
*QBP Spec Addendum 1.6 | April 2026*
*Traceability: BMA Theory 17.0, BMA Spec v9.14 (ScenarioWorktree).*
