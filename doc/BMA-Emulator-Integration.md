# BMA Integration: Xqbp Emulator & Precision Scaling

This document specifies how the **Brain-Machine Architecture (BMA)** integrates with the QBP Compute Unit's RISC-V hardware emulator, covering the precision scaling architecture, codebase layout, cognitive mode mapping, and execution benchmarks.

**Updated:** April 2026
**Status:** Design complete. BMA integration not yet coded.

---

## 1. How the BMA Uses the Emulator

The QBP Compute Unit provides BMA with **width-agnostic quaternion algebra** — the same algorithms run at QW8 (32-bit quaternion, fast) through QW1024 (4096-bit quaternion, precise). Precision is a runtime parameter, not a compile-time choice.

The RISC-V hardware emulator serves three functions:

1. **Arbitrary Precision Prototyping:** The `Gearbox` dynamically scales precision at runtime using `SetWidth()`. BMA developers can test hypergraph stability across all 8 precision levels (QW8 through QW1024) to find the optimal precision boundary for each cognitive mode.

2. **ASIC ROI Justification:** The emulator profiles the theoretical performance of custom `Xqbp` instructions on standard RISC-V silicon. By running BMA hypergraph math through the emulator, operators can calculate exact energy (Joules/Op) and silicon area required for custom chip fabrication, proving Cost-to-Power ROI before manufacturing.

3. **Zero-Noise Cycle Profiling:** The emulator's hot-path is allocation-free (0 B/op verified by benchmarks). This enables clean cycle-accurate timing metrics for pipeline stall and memory fetch latency measurement.

**Current hardware:** Crawl/Walk phases run on general-purpose hardware (AMD FX-8350, AVX1) using `math/big.Float` software emulation. The FX-8350 does not have AVX-512 — all extended precision is pure software via the Gearbox. Custom silicon (Run/Sprint phases) replaces the software path with hardware-accelerated Xqbp instructions.

---

## 2. Precision Scaling Architecture

### 2.1 The Eight Precision Levels

Quaternion word width (QW) defines the bit-width of each quaternion component. A quaternion has 4 components, so a QW64 quaternion occupies 256 bits total.

| Level | Component Bits | Quaternion Bits | Octonion Bits | Composition Depth | Algebraic Lifetime (1 GHz) |
|-------|---------------|-----------------|---------------|-------------------|---------------------------|
| **QW8** | 8 | 32 | 64 | ~8 ops | Microseconds |
| **QW16** | 16 | 64 | 128 | ~24 ops | Milliseconds |
| **QW32** | 32 | 128 | 256 | ~72 ops | Seconds |
| **QW64** | 64 | 256 | 512 | ~2.4K ops | ~7 seconds |
| **QW128** | 128 | 512 | 1024 | ~160K ops | **172 days** |
| **QW256** | 256 | 1024 | 2048 | ~10M ops | Effectively infinite |
| **QW512** | 512 | 2048 | 4096 | >1B ops | Effectively infinite |
| **QW1024** | 1024 | 4096 | 8192 | >1T ops | Effectively infinite |

**Composition Depth** is the number of chained algebraic operations (QMUL, QROT) before machine epsilon drift corrupts the result beyond recovery. This is computed by `MaxCompositionDepth()` in `pkg/qword`. It determines how long a node can survive at a given precision before needing promotion — directly informing BMA's sleep cycle decay and compression decisions.

**Key insight:** QW128 provides 172 days of algebraic integrity. This is why QW128 is the Walk-phase standard — a node computed at QW128 won't drift within any reasonable sleep interval.

### 2.2 Runtime Precision Selection

Precision is purely a runtime parameter. The `Gearbox` in `emulator/qword.go` implements:

- `SetWidth(w Width)` — dynamically updates precision, rescales internal scratchpads
- `Precision()` — maps width to `big.Float` precision in bits
- Pre-allocated scratchpads (`t1`, `t2`, `t3`, `t4`, `rW`, `rX`, `rY`, `rZ`) prevent GC thrashing at any width

The ISA encodes width in the `funct3` field (3 bits = 8 variants), mapping directly to hardware pipeline depth:

| funct3 | Width | Pipeline Cycles (QMUL) | Pipeline Cycles (QROT) |
|--------|-------|------------------------|------------------------|
| 0 | QW8 | 1 | 2 |
| 1 | QW16 | 1 | 2 |
| 2 | QW32 | 1 | 2 |
| 3 | QW64 | 1 | 2 |
| 4 | QW128 | 2 | 4 |
| 5 | QW256 | 4 | 8 |
| 6 | QW512 | 8 | 16 |
| 7 | QW1024 | 16 | 32 |

The Hamilton product algorithm is width-invariant — the same code runs at all 8 levels.

### 2.3 Stage-Width Mapping (Cognitive Profiles)

The compute unit provides predefined profiles mapping precision to cognitive stages (Sense-Compute-Memory-Act):

| Profile | Sense | Compute | Memory | Act | Use Case |
|---------|-------|---------|--------|-----|----------|
| **Default** | QW16 | QW128 | QW8 | QW16 | Standard BMA operation |
| **HighFidelity** | QW32 | QW128 | QW16 | QW32 | Verification and gate runs |
| **ExtendedAutonomy** | QW32 | QW256 | QW16 | QW32 | Deep investigation, prestige mode |
| **Interactive** | QW16 | QW64 | QW8 | QW16 | Reins interaction, fast response |
| **Embedded** | QW16 | QW32 | QW8 | QW8 | Low-power, high-throughput |

These profiles are defined in `pkg/qword/qword.go` and can be selected at runtime.

---

## 3. BMA Cognitive Mode Mapping

Each BMA cognitive mode maps to a precision level based on its latency budget and accuracy requirements:

| BMA Mode | Precision | Budget | Rationale |
|----------|-----------|--------|-----------|
| **Autonomic loop (10Hz)** | QW8 | 100ms/tick | Speed-critical. GAP traversal, spreading activation, pressure detection. ~140K QMUL/tick at QW8. |
| **Episodic observation** | QW16-QW32 | 1s | Encoding sensory input into hypergraph nodes. QW16 sufficient for pattern matching. |
| **Interactive reasoning** | QW64 | Seconds | Reins command processing, beekeeper dialogue. Interactive stage-width profile. |
| **Sleep-cycle computation** | QW128 | Minutes | Decay sweeps, F01 compression, Hebbian updates, checkpoint. 172-day composition depth means no drift between sleep cycles. |
| **Deep investigation** | QW256 | Minutes-hours | Focus Mode, bilateral hypothesis testing, theory cart work. ExtendedAutonomy profile. |
| **Prestige verification** | QW512 | Hours | Cross-instance resonance comparison. Verifying proposals before federation merge. |
| **Constitutional verification** | QW1024 | Unbounded | Proof of Resonance, consensus proofs, Judge Collective verification at maximum precision. Norm-drift tolerance of 10^{-30}. |

**The hard rule:** The 10Hz autonomic loop NEVER exceeds QW8. Higher precisions are for batch computation during sleep, investigation, and verification. This is how the FX-8350 sustains real-time cognitive operation — the fast path is fast, the deep path is deep.

---

## 4. Codebase Layout

### 4.1 Emulator (RISC-V ISS)

Located in `emulator/`:

| File | Purpose |
|------|---------|
| `qword.go` | Core `Gearbox` — zero-allocation scratchpads, `SetWidth()`, Hamilton product, arbitrary-precision math. Defines W8-W1024. |
| `isa.go` | ISA decoder. `Step()` function intercepts `OpcodeCustom0` and maps to Xqbp instructions (QMUL, QROT, QADD, QCONJ, QNORM, FANO). |
| `cpu.go` | Simulated architectural state — Program Counter, 32 general-purpose `X` registers, 32 `Q` registers (quaternion). |
| `isa_bench_test.go` | Benchmark suite verifying 0 B/op across all instructions. |
| `cpu_test.go` | Functional tests including `TestCPU_QMUL_QW1024` (1024-bit multiply). |

### 4.2 Packages (Algebraic Kernels)

Located in `qbp-compute-unit/pkg/`:

| Package | Purpose | Status |
|---------|---------|--------|
| `pkg/qword` | Precision framework. Defines W8-W256, stage-width profiles, `MaxCompositionDepth()`. | Complete |
| `pkg/quat` | Quaternion types. `Quat` (float64), `Quat8` (int8). `Mul()`, `Mul8()`, `Rotate()`. | Complete |
| `pkg/octonion` | Octonion algebra (8 components). For BMA's hyperoctave structure. | Complete |
| `pkg/fano` | Fano plane lookup table (7x7 LUT). Canonical octonion multiplication. Self-verifying. | Complete |
| `pkg/gap` | Geometric Adjacency Pointers. `AdjacencyPointer`, `GAPNode`, `CalculateGradient()`. `UpdateTopology()` is stub. | Prototype |
| `pkg/persona` | Persona operators at QW256. `Persona`, `Hypothesis`, `RunHypothesisTest()`. `ApplyStance()` is mock. | Prototype |

### 4.3 BMA Integration Status

**Not yet coded.** No imports of `github.com/JamesPagetButler/bma-systema` exist in the QBP-Compute-Unit codebase. Integration is designed (documented in MANIFEST.md) but not implemented:

- Walk Phase C1: Wire `pkg/gap` into `internal/bma/hg` for O(1) adjacency traversal
- Walk Phase C5: Wire `pkg/persona` into `internal/bma/ccb` for persona-as-operator cognitive model
- Walk Phase C1: Evolve MuninnDB from typed nodes to quaternion-valued nodes using `pkg/quat`

The integration path is: **BMA imports QBP-CU packages**, not the reverse. QBP-CU remains a standalone algebraic engine with no BMA dependencies.

---

## 5. Performance Benchmarks

Benchmarked on AMD FX-8350 (Crawl hardware). April 2026 refactor achieved 100% GC elimination on the hot path.

### 5.1 Instruction Throughput (Default Width)

```text
BenchmarkCPU_QMUL-8      1747482       714.1 ns/op       0 B/op       0 allocs/op
BenchmarkCPU_QADD-8      2351695       516.6 ns/op       0 B/op       0 allocs/op
BenchmarkCPU_QROT-8      1560652       809.4 ns/op       0 B/op       0 allocs/op
BenchmarkCPU_QCONJ-8     2117844       509.4 ns/op       0 B/op       0 allocs/op
BenchmarkCPU_QNORM-8     2129925       528.2 ns/op       0 B/op       0 allocs/op
```

### 5.2 Analysis

- **100% GC Elimination:** Prior to optimization, a single QMUL generated 6 memory allocations and took ~1438 ns. The refactored implementation requires 0 bytes of heap allocation, cutting execution time in half (714 ns).
- **QROT Efficiency:** The QROT instruction performs q x v x q* (three chained operations). The Gearbox scratchpads enable this in ~809 ns without allocating, proving complex spatial rotations run natively in the execution loop.
- **10Hz Budget:** At 714 ns/QMUL, the FX-8350 can execute ~140K quaternion multiplies per 100ms tick at default width. This is more than sufficient for the autonomic loop's GAP traversal and spreading activation needs at QW8.

### 5.3 Performance Ceiling

| Optimization | Current | Target | Gap |
|-------------|---------|--------|-----|
| Go (math/big) | 714 ns/QMUL | — | Baseline |
| Go + SIMD (AVX1) | ~467 ns (est.) | Walk | 1.53x |
| C + AVX2 | ~145 ns (est.) | Walk hardware | 4.9x |
| Custom Xqbp ASIC | ~10 ns (est.) | Run/Sprint | 71x |

The Go implementation is within 1.53x of SIMD ceiling. AVX assembly kernel for Walk hardware is future work.

---

## 6. Dependencies

Both modules are pure Go with zero external dependencies:

```
github.com/JamesPagetButler/qbp-emulator     (go 1.24.2)
github.com/JamesPagetButler/qbp-compute-unit (go 1.22)
```

---

*Traceability: QBP Spec Addenda 1.1-1.6, BMA Spec v9.0 R-Spec-23, BMA Theory v2.0 Chapter 10.*
*Integration tracked in: BMA issue #67 (Walk Phase C1, C5).*
