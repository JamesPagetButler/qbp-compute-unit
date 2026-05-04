# RISC-V Emulator Best Practices

**Guidelines for Building Reliable and Performant Instruction Set Simulators (ISS)**

*Last Updated: April 18, 2026*

---

## 1. Architectural Design

### 1.1 Decoupling of Components
A modular design is essential for maintenance and adding custom extensions (like `Xqbp`).
- **CPU State:** Encapsulate registers (x0-x31, f0-f31, PC) and privilege modes (M, S, U) in a single structure.
- **System Bus:** Implement a generic "Bus" that routes memory accesses based on address ranges (RAM vs. MMIO).
- **Memory-Mapped I/O (MMIO):** Use a unified interface for peripheral devices (UART, CLINT, PLIC).

### 1.2 Instruction Lifecycle
Follow a strict **Fetch -> Decode -> Execute** loop to ensure architectural correctness.
- **Fetch:** Atomic 32-bit (or 16-bit for 'C' extension) read from memory at PC.
- **Decode:** Use a table-driven or switch-based approach to extract fields (opcode, rd, rs1, rs2, functs, imm).
- **Execute:** State transitions must be atomic; do not commit partial results if an exception occurs during execution.

---

## 2. Implementation Strategies (Go Context)

### 2.1 Performance Optimization
- **Minimize Allocations:** The hot-path loop must have **zero allocations**. Use fixed-size arrays for registers and memory.
- **Jump Tables:** For the decoder, use an array of function pointers/closures indexed by the opcode/funct3/funct7 combinations rather than a giant `switch` statement to reduce branch mispredictions.
- **Unsafe Access (Optional):** Use `unsafe` for memory access if bounds checks become a significant bottleneck, but only after profiling.

### 2.2 Functional vs. Cycle-Accurate
- **Functional (ISS):** Executes one instruction at a time. Best for software development and logic verification. (Recommended for QBP initially).
- **Cycle-Accurate:** Models pipeline stages and stalls. Use Goroutines and Channels in Go to simulate parallel hardware units.

---

## 3. Verification & Safety

### 3.1 Co-Simulation ("The Gold Standard")
Always run the emulator against a **Golden Model** (e.g., [Spike](https://github.com/riscv-software-src/riscv-isa-sim)).
- Log register state (PC, registers) after every instruction.
- Automated diff between emulator output and Spike output.

### 3.2 Architectural Test Suites
Integrate the [Official RISC-V Architectural Tests](https://github.com/riscv-non-isa/riscv-arch-test). No emulator is "complete" without passing these tests for the base ISA (e.g., RV64I).

### 3.3 Exception Handling
Correctly implement Traps, Faults, and CSRs (`mstatus`, `mepc`, `mtvec`). An instruction is not executed if it fails a permission or alignment check.

---

## 4. QBP-Specific Extensions (Xqbp)

- **Instruction Encodings:** Use the `custom-0` through `custom-3` opcode space.
- **State Extension:** Q-registers (`q0`-`q31`) should be handled similarly to the Vector ('V') extension state.
- **Precision Gearbox:** Ensure the emulator supports dynamic width switching (QW8 to QW1024) within the `Execute` phase.

---

## 5. Pre-Task Checklist (Refresh Mandatory)

**IMPORTANT: Before starting work on the emulator, the agent must:**
1. Search for the latest [RISC-V Unprivileged Spec](https://riscv.org/technical/specifications/) updates.
2. Review the `pkg/qword` logic to ensure parity between Go math and ISA emulation.
3. Verify that the host system has `riscv64-unknown-elf-gcc` for compiling test binaries.

---
*Reference: [1] Spike (Official ISS), [2] TinyEMU (Fabrice Bellard), [3] RISC-V Arch-Test Suite.*
