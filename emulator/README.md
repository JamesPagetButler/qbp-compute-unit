# QBP RISC-V Emulator

**Purpose:** This directory contains the emulation path for the QBP Compute Unit. It is specifically designed to run on brute-force hardware (e.g., AMD FX-8350 x86) to simulate, test, and mathematically justify the transition to the **Run Phase** (custom RISC-V silicon).

## Strategic Context

The QBP project maintains two distinct engineering efforts:
1. **Localized Optimization (Crawl/Walk):** High-throughput inference and hypergraph traversal running natively on standard x86/GPU hardware (e.g., the AVX SIMD kernels).
2. **RISC-V Emulation (Run Prep):** *This directory.* A sandbox to evaluate if fabricating a custom ASIC provides enough compute capacity and energy savings to justify the cost.

## Goals of the Emulator

This emulator is not meant to execute production BMA hypergraph inferences. Instead, its primary goals are:

1. **Instruction Set Verification (Xqbp):** Model the proposed custom RISC-V ISA extensions for Quaternion-Based Physics (`QMUL`, `OMAC`, `FANO`).
2. **Precision Testing:** Safely test massive arbitrary-precision physics (QW128 up to QW1024) using Go's `math/big` before burning algorithms into hardware.
3. **Cycle-Accurate Profiling (Planned Upgrade):** Transition from functional execution to a cycle-accurate model that tracks pipeline stalls, fetch times, and register read delays.
4. **Physical Cost-Benefit Analysis:** Output synthetic energy and silicon area profiles (e.g., Joules-per-instruction at 130nm or 28nm) to definitively prove that a native hardware Fano-plane LUT and 512-bit hardware multiplier offer a massive ROI compared to standard CPUs.

## Best Practices

If you are developing or modifying the emulator, please strictly adhere to the guidelines laid out in `RISCV-Emulator-Best-Practices.md`.
