# QBP Compute Unit - Gemini Context

This directory contains the **Crawl Phase Implementation** of the Quaternion-Based Physics (QBP) Native Processing Architecture. This is a research project aimed at implementing a quaternion-native algebraic kernel to evaluate its efficiency for physical modeling compared to conventional floating-point arithmetic.

## Project Overview

- **Core Technology:** Go (Golang) implementation.
- **Algebraic Kernel:** Support for Quaternions (`pkg/quat`), Octonions (`pkg/octonion`), and Fano plane logic (`pkg/fano`).
- **Research Goal:** Benchmarking QBP-algebraic performance vs. scalar processing in physics simulations (e.g., Heisenberg spin-chains).
- **Key Discovery:** Architecture must be width-parameterized (QW8 to QW64) to maintain numerical fidelity across different scales of computation.

## Directory Structure

The primary source code is contained within the `qbp-compute-unit-final.tar.gz` archive.

- `qbp-compute-unit-final.tar.gz`: Complete Go source code and benchmarks.
- `QBP-Compute-Unit-Spec-Rev1.docx`: Architectural specification (Section 11 contains next actions).
- `QBP-Compute-Unit-Spec-Rev2.docx`: Updated specification.
- `QBP-Compute-Unit-Master-Record.docx`: Project master record.

### Internal Source Structure (within tarball)
- `cmd/benchmark/`: Main entry point for running verification and benchmarks.
- `pkg/quat/`: Physics-precision (float64) and low-precision (int8) quaternion algebra.
- `pkg/octonion/`: Octonion algebra and Fano plane integration.
- `pkg/spinchain/`: 1D Heisenberg benchmark and composition stress tests.
- `pkg/watchdog/`: Health monitoring for algebraic drift and renormalization.

## Building and Running

To work with the source code, first extract the archive:

```bash
tar -zxvf qbp-compute-unit-final.tar.gz
cd qbp-compute-unit
```

### Key Commands

- **Run Benchmarks:**
  ```bash
  go run cmd/benchmark/main.go
  ```
- **Custom Simulation:**
  ```bash
  go run cmd/benchmark/main.go -n 50 -steps 50000 -dt 0.005 -J 1.0
  ```
- **With Renormalization:**
  ```bash
  go run cmd/benchmark/main.go -steps 100000 -renorm -renorm-every 500
  ```

## Development Conventions

- **Instruction Mapping:** Go functions in `pkg/` are designed to map 1:1 to proposed RISC-V instructions (e.g., `quat.Mul` -> `QMUL`).
- **Numerical Fidelity:** Always use `float64` (`Quat` type) for physics simulations; `int8` (`Quat8`) is reserved for hypergraph traversal and low-precision memory operations.
- **Health Checks:** Use the `watchdog` package to monitor norm drift in long-chain compositions.

### RISC-V Emulator Implementation
- **Authoritative Reference:** [emulator/RISCV-Emulator-Best-Practices.md](emulator/RISCV-Emulator-Best-Practices.md)
- **MANDATORY REFRESH:** Before starting any emulator-related task, the agent MUST search for the latest RISC-V specifications and review the Best Practices document to ensure up-to-date knowledge of instruction decoding and verification strategies.

## Ternary & BMA Integration

The project includes specific work related to **Ternary Inference** and the **BMA Hypergraph**:
- **Quat8/Oct8 Representations:** Use `int8` components specifically optimized for hypergraph traversal and BMA memory integration.
- **Ternary Benchmark:** The `int8` data type used in `Quat8` was selected because it outperformed others in a ternary inference benchmark (achieving a 16.6× speedup).
- **Future Integration:** A key "Next Action" is the integration with the **BMA octonionic hypergraph** for few-hop traversal and memory-efficient communication.

1.  **Context:** Refer to the `README.md` (once extracted) for the latest benchmark results and "Crawl Phase" milestones.
2.  **Specifications:** Treat the `.docx` files as the authoritative architectural source (Rev 2 is the most recent).
3.  **Algebra:** Ensure any new physics-related logic uses the `pkg/quat` or `pkg/octonion` kernels rather than standard scalar math where QBP principles apply.
