# SPECIFICATION: QBP Compute Unit Architecture v1.0

**The Hardware Blueprint for Algebraic Integrity**

Version 1.0 | April 2026
Helpful Engineering — QBP Project
Co-Authored-By: James Paget Butler (Beekeeper) & Gemini CLI (Architect)

---

## 1. The Q-Pipe (Pipeline Architecture)

The QBP Compute Unit is a **Stateful Vector Pipeline** designed for zero-latency quaternion transformations.

### 1.1 The Gearbox (Precision Scaling)
The "Gearbox" is the hardware logic that handles width-switching between QW8 and QW1024.
- **Dynamic Width Transition:** The pipeline can switch widths in a single clock cycle via the `qw` CSR.
- **Alignment:** Data in memory is always 4096-bit aligned to ensure peak throughput at QW1024.

### 1.2 The Hamilton Engine (QMUL)
The Hamilton Engine consists of four parallel MAC (Multiply-Accumulate) units that compute the 16 cross-products of a Hamilton Product in parallel.
- **Throughput:** 1 QMUL per clock cycle at QW64.
- **Headroom Mode:** At QW1024, the operation is multi-cycle (16 cycles) to maintain 4096-bit precision without heating the substrate.

## 2. Quaternion Memory (Q-Mem)

QBP-native memory is addressed by **Locale (Quaternion Spatiotemporal Coordinates)** rather than linear integer offsets.

### 2.1 Locale Addressing
The memory controller performs an internal **Nearest-Neighbor Search (KNN)** when a Q-word is requested.
- **Direct Mode:** Access by linear address (for Word 12 legacy support).
- **Associative Mode:** Access by Q-vector proximity (for Contextus scouting).

## 3. The BMA/Contextus Bridge

The QBP Compute Unit provides a **Native Interrupt** for "Seam Detection."
- **CTH Watchdog:** A dedicated hardware unit that monitors the norm of quaternions in the pipeline.
- **Trigger:** If the norm of a result deviates from 1.0 by more than $10^{-30}$ (at QW1024), the hardware triggers a **Constitutional Audit interrupt** to the BMA.

---
*QBP Architecture v1.0 | April 2026*
*Traceability: QBP Spec Rev 2.0, BMA Addendum 11.0.*
