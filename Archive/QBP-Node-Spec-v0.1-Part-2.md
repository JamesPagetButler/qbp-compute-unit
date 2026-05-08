# QBP-Node Specification v0.1 — Part 2: Crawl-Phase Detailed Specification

**Title:** QBP-Node Crawl-Phase Specification — Cycle-Accurate Emulation, SIMD Assembly Path, and First Workloads on x86
**Version:** 0.1 (Part 2 of the QBP-Node specification)
**Author:** James Paget Butler
**Editor:** Claude Opus 4.7 (architecture instance)
**Date:** 2026-05-04
**Status:** Pre-spec working document. Detailed specification for the Crawl phase as defined in Part 0.

---

## How to read this part

Part 2 specifies what is built in the Crawl phase of QBP-Node development. The Crawl phase substrate is x86-64 development workstations; QBP silicon is fully emulated; a single-node configuration is the working target. Multi-node behaviour is paper architecture in Crawl; it becomes implementation in Walk.

The specification is written assuming Parts 0 and 1 have been read. Where this part defines a new concept (e.g., the cycle-accurate simulator's pipeline model), the specification is complete here. Where this part references concepts established in Part 1 (e.g., the three execution contexts), the references are by section number and are not redefined.

The Crawl phase has six concrete deliverables, each specified in its own section:

1. The Go cycle-accurate simulator (§2.2)
2. The SIMD assembly path — W64 then W128 (§2.3)
3. The Lean-to-ROM build pipeline (§2.4)
4. The cosim test corpus, Tier-0 and Tier-1 (§2.5)
5. BMA Crawl.Heartbeat host runtime (§2.6)
6. Sharp Butler MVP on classical x86 (§2.7)

These deliverables, taken together, constitute Crawl exit. §2.8 specifies the integration testing that demonstrates Crawl exit; §2.9 captures concepts that arose during Crawl but belong to Walk or later.

## 2.1 Crawl-Phase Scope and Constraints

### 2.1.1 What Crawl is

The Crawl phase establishes the foundation on which all subsequent phases depend. By Crawl exit, the QBP-Node project must have:

- **A working cycle-accurate simulator** of the QBP accelerator block, running on x86 Linux, against which the eventual silicon will be verified (§9.1 of QBP-CU-SiFive-Interface-Spec-v0.1 establishes this contract).
- **A demonstrably useful workload** — Sharp Butler MVP — running real residential automation, even though the QBP accelerator is fully emulated. This proves the workload concept is real, not hypothetical.
- **A running BMA instance** at Crawl.Heartbeat level, demonstrating the cognitive substrate works against the simulator.
- **A verification corpus** — Tier-0 algebraic identity tests and Tier-1 single-instruction microbenchmarks — that the simulator passes at zero divergences across 10⁹+ random inputs. This corpus carries forward unchanged through Walk and Run as the silicon validation foundation.

### 2.1.2 What Crawl is not

To prevent scope creep, the following are explicitly *not* Crawl deliverables:

- Custom silicon design beyond the paper specifications inherited from Parts 0 and 1
- Real RISC-V hardware integration of any kind
- Multi-node networking, network protocols, or NATS-mediated capability negotiation
- T3 and higher-tier algebraic capability beyond software emulation; 𝕆 and 𝕊 mode are software-only in Crawl
- QW1024 container runtime implementation (paper architecture only, per §1.7)
- Trust Receipt protocol implementation (paper architecture only, per §1.10)
- The CAPSEL family of instructions (Walk-phase per Part 1)
- HEDGE_GATHER, HEDGE_SCATTER, CONF_PROBE, RECALL_KNN as silicon primitives (Walk-phase; Crawl-phase emulates them in software within the simulator)
- Any optical or photonic substrate work (Sprint-phase per Part 1)

These constraints are not pessimism about the team's capability. They are protection for the work that *is* in Crawl, which must be done thoroughly to support everything that follows.

### 2.1.3 Hardware substrate

The reference development hardware for Crawl is:

- **Primary**: Modern x86-64 workstations with AVX2 minimum, AVX-512 preferred. The FX-8350 development machine remains available but is performance-limited; AMD Zen 3+ or Intel Tiger Lake+ is recommended for SIMD development work.
- **Linux distribution**: Ubuntu 24.04 LTS or equivalent. ROCm-capable systems for GPU exploration but no Crawl deliverable depends on GPU.
- **RAM**: 32 GB minimum for hosting the simulator with realistic BMA workloads; 64 GB recommended.
- **Storage**: NVMe SSD for the development environment; the BMA hypergraph state can grow large during Crawl.Heartbeat operation.

No specialised hardware is required for Crawl. This is deliberate: the phase is meant to validate the architecture and software stack on commodity hardware before committing to specialised silicon paths.

## 2.2 The Cycle-Accurate Simulator

### 2.2.1 Purpose and contract

The Go cycle-accurate simulator is the reference implementation of the QBP accelerator block at the cycle level. Its responsibilities:

- Implement the RV-Fano instruction set at the level of detail specified in `RV-Fano-Implementation-Refinements (v0)` — Layer 0 primitives, Layer 1 mode controls, Layer 2 composite operations.
- Maintain cycle-accurate timing per §4.3 of the implementation refinements (16 cycles for OMUL, 28 for SMUL, 12 for QMUL, 7 for ZDCHK.SYM, etc.).
- Emit watchdog events per §2.3 of this part, in production or research mode per CSR.
- Serve as the golden model against which Walk-phase RISC-V hosted execution and Run-phase silicon are verified.

The contract for the simulator is multiset equality of watchdog events per cycle with the eventual RTL (§9.1 of the SiFive interface spec). Bit-pattern equality of result registers is also required per §9.2.

### 2.2.2 Package structure

```
qbpcu/
├── accelerator.go        # Public Accelerator interface (Mock, Golden, RTLShim)
├── isa/
│   ├── opcode.go         # Opcode enum: QPERM, QPERMR, QDEC, QREC, QNEAR, BSEL, PSEL, AMODE, ...
│   ├── decoder.go        # Instruction decoder
│   └── encoder.go        # Assembler
├── pipeline/
│   ├── stage.go          # Pipeline stage abstraction
│   ├── recv.go           # A_RECV stage
│   ├── decode.go         # A_DECODE stage
│   ├── lut.go            # A_LUT stage (Fano-plane lookup)
│   ├── mac.go            # A_MAC stage (multiply-accumulate)
│   ├── wd.go             # A_WD stage (watchdog tap)
│   └── resp.go           # A_RESP stage
├── algebra/
│   ├── h.go              # Quaternion (ℍ) operations
│   ├── o.go              # Octonion (𝕆) operations
│   ├── s.go              # Sedenion (𝕊) operations
│   ├── rom.go            # Sign and index ROM access
│   └── watchdog.go       # Watchdog event generation
├── mode/
│   ├── transition.go     # AMODE / BSEL / PSEL state machine
│   └── trap.go           # Trap codes and dispatch
├── qword/
│   ├── qw64.go           # W64 type (4× float64) — Gemini's SIMD path target
│   ├── qw128.go          # W128 type (Dekker double-double) — W128 SIMD target
│   └── qw256.go          # Software fallback for 𝕊 mode operands
└── testing/
    ├── tier0/            # Algebraic identity tests
    ├── tier1/            # Single-instruction microbenchmarks
    ├── tier2/            # Kernel-level workloads
    └── tier3/            # System-level workloads
```

This structure separates concerns cleanly: ISA in `isa/`, microarchitecture in `pipeline/`, algebra in `algebra/`, types in `qword/`, mode logic in `mode/`. Each subdirectory is independently testable.

### 2.2.3 The Accelerator interface

The interface from QBP-CU-SiFive-Interface-Spec-v0.1 §7 is the public contract. Reproduced here for clarity:

```go
package qbpcu

// Port discriminator
type Port uint8

const (
    PortSSCI Port = iota
    PortVCIX
)

// Req: a request from host to accelerator
type Req struct {
    Cycle    uint64
    Port     Port
    Op       Opcode
    VL       uint32
    SrcA     QW128
    SrcB     QW128
    Imm      uint8
    DestTag  uint16
}

// Resp: a response from accelerator to host
type Resp struct {
    Cycle    uint64
    DestTag  uint16
    Result   QW128
    Status   Status
    FaultCode uint32
}

// WDEvent: a watchdog event, tapped at every algebraic crossing
type WDEvent struct {
    Cycle      uint64
    Op         Opcode
    Port       Port
    FanoIndex  uint8
    SignBit    bool
    Associator [3]int8
    NormDelta  int32
    AlgebraID  uint8
    ZDClass    uint8
    ZDIndices  [4]uint8
}

type Accelerator interface {
    Submit(r Req)
    Poll() (Resp, bool)
    WatchdogChan() <-chan WDEvent
    Tick(cycle uint64)
}
```

Three implementations are required by Crawl exit:

- `Mock`: functional only, no cycle-accuracy. Used for BMA Crawl.Heartbeat, where speed matters more than timing fidelity.
- `Golden`: cycle-accurate, with full watchdog event stream. Used for cosim Tier-0/1/2 test runs.
- `RTLShim`: stub implementation that returns "not implemented" errors. Materialises in Walk when Verilator builds become available.

`Mock` and `Golden` must produce identical bit-pattern results for the same input. Only their timing differs.

### 2.2.4 Pipeline model

The accelerator pipeline is six stages, per §5.2 of the SiFive interface spec:

```
A_RECV → A_DECODE → A_LUT → A_MAC → A_WD → A_RESP
```

Each stage is a Go struct implementing a common interface:

```go
type Stage interface {
    Name() string
    Tick(cycle uint64) error
    Input() chan<- StageMessage
    Output() <-chan StageMessage
}

type StageMessage struct {
    Cycle uint64
    Op    Opcode
    State PipelineState
}
```

Stages are wired with bounded channels. The simulator's main loop ticks all stages in deterministic order on each cycle:

```go
func (g *GoldenAccelerator) Tick(cycle uint64) {
    g.recvStage.Tick(cycle)
    g.decodeStage.Tick(cycle)
    g.lutStage.Tick(cycle)
    g.macStage.Tick(cycle)
    g.wdStage.Tick(cycle)
    g.respStage.Tick(cycle)
}
```

**Determinism is non-negotiable.** The Go scheduler is non-deterministic, so the simulator core does *not* use goroutines for pipeline stages — they run sequentially within a single goroutine. Concurrency is reserved for:
- The watchdog event channel (consumed by a separate goroutine that may write to disk or a host-side log)
- I/O for cosim harness communication
- Test harness coordination

This is the "determinism vs concurrency tension" called out in §2.2.4 of the original cycle-accurate-simulator analysis. Goroutines lose; bit-identical reproducibility wins.

### 2.2.5 Cycle budget enforcement

Per §5 of the implementation refinements, each operation has a budget. The simulator enforces these budgets by counting cycles spent in each stage:

| Operation | A_RECV | A_DECODE | A_LUT | A_MAC | A_WD | A_RESP | Total |
|---|---|---|---|---|---|---|---|
| QMUL | 1 | 1 | 1 | 4 | 1 | 1 | 9 (compute) + 4 dispatch + (–) WD parallel = 13 (with VCIX overhead) |
| OMUL | 1 | 1 | 1 | 8 | 1 | 1 | 13 (compute) + 4 dispatch + (–) WD parallel = 17 (with VCIX overhead) |
| SMUL | 1 | 1 | 1 | 20 | 1 | 1 | 25 (compute) + 4 dispatch = 29 |
| ZDCHK.SYM | 1 | 1 | 1 | 3 | 1 | 1 | 8 |
| ZDCHK | 1 | 1 | 1 | 20 | 1 | 1 | 25 (full SMUL path) + 4 dispatch = 29 |
| BSEL/PSEL/AMODE | 1 | 1 | – | – | 1 | 1 | 4 |
| OCONJ/SCONJ | 1 | 1 | – | 1 | 1 | 1 | 5 |

The numbers in §5 of the implementation refinements (16 / 28 / 12 / 7) are end-to-end including host dispatch overhead. The numbers above are accelerator-internal. Both are enforced by the simulator.

A cycle budget violation in the simulator is a hard test failure. The cosim contract requires the eventual RTL to match these budgets within ±2 cycles end-to-end and exactly at steady state.

### 2.2.6 What's already built and what isn't

Based on existing QBP-CU work in James's repositories:

**Already exists (verify and integrate):**
- Go quaternion / octonion algebra package (`quaternion/`, `octonion/`)
- Fano plane LUT (`fano/`)
- Algebraic watchdog (`watchdog/`)
- RISC-V instruction emulator (basic, scalar)

**Needs to be built (Crawl deliverable):**
- The `Accelerator` interface and its three implementations
- The pipeline-stage model with cycle-accurate timing
- The mode-transition state machine per §3 of the implementation refinements
- The new ZDCHK.SYM Layer 1 primitive
- Integration of all the above into a coherent simulator binary

**Needs Gemini's SIMD path (parallel deliverable):**
- W64 fast path with the corrected Hamilton sign masks
- W128 Dekker/Bailey double-double path
- Constants extracted from the Lean source per §2.4

The existing work substantially reduces Crawl-phase implementation cost; we are integrating, not building from scratch.

## 2.3 The SIMD Assembly Path and the Dispatch-Layer Architecture

The SIMD path is Gemini's primary Crawl deliverable. This section specifies the contract from the architecture side; Gemini owns the implementation details.

**Architectural framing.** The SIMD assembly path is not only a performance optimisation for x86 development hardware. It is the first concrete instantiation of a dispatch-layer architecture that lets the same Go API target multiple compute backends across phases:

```
qbpcu/algebra/h.go     — Go-generic ℍ implementation (always present, reference)
qbpcu/asm/qmath_amd64.s    — AMD64 SIMD backend (Crawl, primary)
qbpcu/asm/qmath_arm64.s    — ARM64 SIMD backend (Crawl, secondary; Apple Silicon)
qbpcu/asm/qmath_riscv64.s  — RISC-V vector backend (Walk)
qbpcu/cim/                — Compute-in-Memory backend (Crawl Level-1; Walk Level-2 if gated; Run Level-3 if gated)
```

The Go package presents a single API. Backend dispatch is selected at build time (architecture-specific assembly) or at runtime (CIM enable flag). The same caller code runs against any backend; verification is the cosim corpus, which the backends compete to pass.

This means the SIMD work serves three purposes: it is fast on x86 today, it pioneers the dispatch interface that other backends adopt, and it provides the reference performance baseline against which alternative backends (including CIM) are measured. The §2.3.5 CIM emulator and §2.3 SIMD path are not competing efforts but the first two backends in a planned sequence.

### 2.3.1 W64 path

The W64 path provides a zero-allocation, high-throughput backend for ℍ-mode operations on operands fitting in 4× float64 (256 bits, one YMM register).

**Required entry points** (Go declarations with `//go:noescape`):

```go
//go:noescape
func qmul64AVX(c, a, b *[4]float64)

//go:noescape
func qadd64AVX(c, a, b *[4]float64)

//go:noescape
func qrot64AVX(c, q, v *[4]float64)

//go:noescape
func qconj64AVX(c, a *[4]float64)

//go:noescape
func qnorm64AVX(out *float64, a *[4]float64)
```

**Required correctness properties:**

- For all inputs `a`, `b`: `qmul64AVX(c, a, b)` produces the Hamilton product `a × b` correct to ULP, with FMA-aware semantics matching the recalibrated watchdog ε threshold.
- For all inputs `a`: `qrot64AVX(c, q, v)` produces the rotation `q × v × q*` correct to ULP, with no intermediate heap allocations.
- For all inputs: zero allocations per call (verified by `go test -benchmem`).
- Sign masks (`Y_SIGN_X`, `Y_SIGN_Y`, `Y_SIGN_Z`) extracted from the Lean source per §2.4.

**Required performance properties:**

- W64 QMUL throughput: ≥ 5× faster than the Go-generic ℍ implementation on the FX-8350 reference, ≥ 10× on AMD Zen 3 / Intel Tiger Lake or newer.
- Zero allocations per operation in the steady state.
- No measurable degradation of cache locality for the calling code (verified by realistic workload benchmarks).

The two specific sign-mask corrections from the round-2 SIMD review must be applied:
- Term 1 mask: `[−, +, +, +]` (was `[−, +, −, +]`)
- Term 2 mask: `[−, −, +, +]` (was `[−, +, +, −]`)

### 2.3.2 W128 path (Dekker / Bailey double-double)

The W128 path provides ~32 decimal digits of precision via double-double arithmetic, vectorised over AVX. This is the production path for QBP physics workloads and any computation requiring precision beyond float64.

**Required entry points:**

```go
//go:noescape
func qmul128AVX(c, a, b *[8]float64)  // double-double: 2× float64 per component

//go:noescape
func qadd128AVX(c, a, b *[8]float64)

//go:noescape
func qrot128AVX(c, q, v *[8]float64)

//go:noescape
func qconj128AVX(c, a *[8]float64)

//go:noescape
func qnorm128AVX(out *[2]float64, a *[8]float64)
```

**Required correctness properties:**

- Double-double error bounds: ~2⁻¹⁰⁴ for addition, ~2⁻¹⁰⁰ for multiplication. Watchdog ε threshold for W128 must be recalibrated independently of W64.
- Correct handling of FMA-vs-non-FMA TwoProduct: AMD64 path uses FMA-based TwoProduct; ARM64 fallback uses Dekker-split TwoProduct. Cross-architecture results match within stated error bounds, never bit-identical.
- The Go-generic W128 fallback exists and is used on non-AMD64 architectures.

**Required performance properties:**

- W128 QMUL throughput: ≥ 30× faster than `math/big.Float` ℍ implementation.
- Steady-state allocation rate: 0 per operation.

### 2.3.3 W256 / sedenion path (deferred)

A SIMD path for sedenion operations (32× float64, 2048 bits) is *not* a Crawl deliverable. Sedenion mode is software-emulated in Crawl. If profiling during BMA Crawl.Heartbeat reveals that 𝕊-mode operations are a hot path (which they are not expected to be), a W256 SIMD effort can be added as a Walk deliverable.

### 2.3.4 Verification contract

Gemini's SIMD paths are verified against:

1. **Bit-equivalence with the Go-generic path** (via the Tier-1 corpus, with FMA-divergence cases explicitly enumerated).
2. **Algebraic property tests** (Tier-0): norm preservation, associativity, conjugate involution, Moufang identity.
3. **Watchdog event multiset equality** with the Go-generic path at the calibrated ε threshold.
4. **Cross-architecture reproducibility**: ARM64 (Apple Silicon) tests must produce results within stated error bounds.

The verification contract is the round-2 expanded plan from the SIMD review. No changes from the Crawl perspective.

### 2.3.5 The CIM Level-1 Functional Emulator

The CIM Level-1 emulator is the second backend in the dispatch-layer architecture introduced at the start of §2.3. It is a Crawl deliverable, scoped narrowly to test one specific question: **does the algorithmic mapping from QBP/BMA operations onto a CIM-SRAM array produce correct results?**

It is not a performance model. It is not an energy model. It does not claim to predict what real CIM silicon would do. It is a functional check on the architectural assumption that storage and computation can be unified for the operations BMA and QBP actually perform.

#### 2.3.5.1 Why this is the first thing to build

Two failure modes for the CIM architecture exist:

- *Logical*: the operations BMA and QBP need cannot be cleanly expressed as CIM array operations; the mapping requires hacks, workarounds, or operations that violate the CIM physical model.
- *Physical*: the operations map cleanly but the resulting silicon would be too slow, too energy-hungry, or unimplementable.

Level-1 catches the logical failure mode at low cost. If it succeeds, Level-2 (cycle-and-energy modelling, contingent per §0.4.1 of Parts 0/1) catches projected physical issues at moderate cost. If both succeed, Level-3 (SPICE characterisation, deferred to Run) catches actual physical issues at high cost.

Building Level-2 before Level-1 is a category error: it spends weeks on performance and energy projections for an architecture that may not algorithmically work. Building Level-3 before Level-2 is a similar category error at higher cost. The discipline is to spend the cheapest possible effort first to surface the cheapest-to-fix problems.

#### 2.3.5.2 Package structure

```
qbpcu/cim/
├── array.go          # Cell array model: 2D grid of cells with low-bit values
├── ops.go            # Array operations: write, read, activate-column
├── mapping.go        # Algorithmic mapping from QBP/BMA operations to array operations
├── workloads/        # Test workloads (spreading activation, BMA recall, etc.)
└── compare.go        # Comparison harness: run same workload through conventional and CIM paths, verify equivalence
```

The package is purely functional. No Goroutines, no I/O beyond test harness, no timing instrumentation, no energy accounting. Operations run in deterministic order.

#### 2.3.5.3 Cell array model

A `cim.Array` is a 2D grid of cells. Each cell holds a low-bit value (default ternary {-1, 0, +1}, configurable to 1-bit binary or 2-bit signed). Operations:

```go
type Array struct {
    Rows, Cols int
    BitsPerCell int
    cells [][]int8
}

func (a *Array) Write(row, col int, value int8) error
func (a *Array) Read(row, col int) (int8, error)
func (a *Array) ActivateColumn(col int, input []int8) ([]int8, error)
// ActivateColumn performs MAC: output[r] = Σ input[k] × cells[r][k] for column reads
// or the column-vs-row equivalent depending on the array layout
```

The exact MAC semantics — what's accumulated along which axis, what activation function is applied at the bitline read — match the XNOR-SRAM reference design from the existing research note. Other layouts can be modelled by configuration; the default reproduces a published CIM macro.

#### 2.3.5.4 Algorithmic mapping

The `mapping.go` module translates QBP/BMA operations into sequences of CIM array operations. Initial coverage:

- **Hypergraph spreading activation** (BMA): for each active vertex, propagate weighted activation to other vertices in incident hyperedges. Mapping: store the incidence structure as a sparse matrix in cells, treat the active-vertex set as an input vector, compute output by column activation.
- **BMA recall** (one-hop associative): for a query vertex, return weighted incident vertices. Mapping: same as spreading activation but with single-vertex input.
- **Sleep consolidation rotation** (BMA): bus rotation per the QBP physics specification. Mapping: pending — this operation requires more thought to express in CIM terms; if it cannot be cleanly expressed, that is a Level-1 finding worth recording.
- **Judge collective consensus aggregation** (BMA): combine multiple judge instance evaluations into a single decision. Mapping: pending — this is closer to standard sparse linear algebra and should be straightforward.
- **ℍ-mode quaternion MAC** (QBP and BMA): hypergraph weight evaluation with quaternion-valued edges. Mapping: requires multi-bit cells and per-cell sign bits; will probably need 4-bit cells. This is a Level-1 architecture experiment.

Each operation has a Go function in `mapping.go` that takes the conventional input data structures and produces an equivalent execution against a `cim.Array`.

#### 2.3.5.5 Comparison harness

The `compare.go` module runs the same workload through both the conventional simulator path and the CIM-mapped path, comparing results. A test passes if the bit-equivalent results match within a stated tolerance (zero for integer operations, ULP for any floating-point operations involved in the conventional path).

For Level-1, every workload must produce zero divergences. A divergence indicates either a bug in the conventional path, a bug in the CIM mapping, or a genuine semantic gap where the CIM model cannot represent what the conventional path computes. The third case is the architecturally significant finding — it tells us the limits of the CIM mapping.

#### 2.3.5.6 The Crawl CIM workload corpus

The corpus that Level-1 must pass for §0.4.1 promotion gate evaluation:

- `T0.CIM.SPREAD.001`: Hypergraph spreading activation on a 1000-vertex / 3000-hyperedge test graph, against 10⁴ random initial-activation vectors.
- `T0.CIM.RECALL.001`: BMA single-vertex recall on the same test graph, against 10⁴ random query vertices.
- `T0.CIM.SLEEP.001`: Sleep consolidation bus rotation on a 100-vertex working memory, exercising all 7 Fano-line rotations. *Tagged as exploratory pending the mapping question above.*
- `T0.CIM.JUDGE.001`: Judge collective aggregation of 5 judge instances over 10³ decisions.
- `T0.CIM.HMAC.001`: ℍ-mode quaternion MAC over a 1000-edge hypergraph with quaternion-valued weights. *Architecture-experimental.*

Each test enumerates the conventional and CIM-mapped paths and verifies bit equivalence.

#### 2.3.5.7 Architectural insight hooks

Beyond pass/fail, the Level-1 emulator records architectural observations during execution:

- **Cell utilisation**: what fraction of the array is non-zero at the end of the workload? Sparse hypergraphs may have very low utilisation, suggesting the CIM advantage is reduced for our use case.
- **Operation-to-cell ratio**: how many CIM operations per logical BMA operation? A high ratio suggests inefficiency in the mapping.
- **Multi-step decompositions**: which BMA operations require multiple CIM array activations? Each is a candidate for "this operation is a poor fit for CIM."

These observations feed into the §0.4.1 promotion gate evaluation, specifically the third criterion (whether Level-1 has revealed at least one architecturally significant insight worth quantifying).

#### 2.3.5.8 Out of scope for Level-1

Explicitly:

- No timing model. Operations on `cim.Array` complete instantaneously from the simulator's perspective.
- No energy model. The published 403 TOPS/W figure for XNOR-SRAM is referenced but not used in any computation.
- No analog effects. Cells either hold their value or they don't; no ADC noise, no sense-amplifier offset, no temperature dependence.
- No physical constraints. Arrays can be arbitrary sizes (within Go memory limits); peripheral overhead is not counted.
- No backend selection by the dispatch layer. Workloads run *either* through the conventional path *or* through the CIM path, in isolation, for comparison purposes only. Production code does not yet target the CIM backend.

These limitations are not problems with Level-1; they are its definition. Level-2, if it comes, addresses them. Level-1 only answers whether the algorithmic mapping is correct.

#### 2.3.5.9 Cost estimate

Approximately two weeks of work for a competent Go developer familiar with the existing simulator codebase. Most of the time is in `mapping.go` (the algorithmic translation) and the workload corpus; the cell array model itself is a few hundred lines.

This is small relative to the cycle-accurate simulator (several months of work to build to specification) and the SIMD assembly path (Gemini's ongoing effort). The CIM Level-1 work runs in parallel with both without competing for critical resources.

## 2.4 The Lean-to-ROM Build Pipeline

### 2.4.1 Purpose

The sign and index ROMs that drive the QBP accelerator's algebra come from a single source of truth: the Lean-verified tables in `Sedenion.lean` (`mulSignData` and `mulIdxData`). This is a non-negotiable constraint per the physics resolution §2.2: hand-transcription of sign tables is forbidden.

The `lean2rom` build pipeline extracts these tables from Lean and produces:

1. ROM hex files for the accelerator (consumed by the simulator and, eventually, the RTL).
2. SIMD constant files for Gemini's assembly path (the `Y_SIGN_X`, `Y_SIGN_Y`, `Y_SIGN_Z` quaternion sign masks).
3. A checksum manifest that the cosim harness verifies at startup.

### 2.4.2 Pipeline structure

```
qbp-lean/QBP/Sedenion.lean
        │
        │ Lean compile + #eval extraction
        ▼
qbp-lean/build/sedenion-tables.json
        │
        │ Go program: lean2rom
        ▼
qbp-cu/roms/sedenion_signs.hex     (225 entries × 1 bit)
qbp-cu/roms/sedenion_idx.hex       (256 entries × 4 bits)
qbp-cu/roms/octonion_signs.hex     (49 entries × 1 bit)
qbp-cu/roms/octonion_idx.hex       (64 entries × 3 bits)
qbp-cu/roms/quaternion_signs.go    (vendored Go constants)
qbp-cu/asm/qmath_constants.s       (assembly constants)
qbp-cu/roms/CHECKSUMS.lean-verified
```

The pipeline runs as a `make sign-roms` target. CI validates the checksums on every build.

### 2.4.3 Implementation in Crawl

`lean2rom` is a small Go program (~500 lines). It:

1. Invokes Lean to evaluate the relevant table expressions (`#eval mulSignData`, `#eval mulIdxData`).
2. Parses the JSON output into structured tables.
3. Validates the tables against the algebraic properties expected (`assert(idx_tab[i,j] == i^j)` for all non-zero indices, etc.).
4. Emits the four ROM hex files with proper formatting.
5. Generates the Go constants file with explicit comments citing line ranges in the Lean source.
6. Generates the assembly constants file with explicit YMM register layout.
7. Emits the checksum manifest with SHA-256 of every output file.

The simulator and (eventually) the RTL load from these files. Mismatch is a hard fault.

### 2.4.4 Verification of the SIMD constants

Per §3.1 of this part, Gemini's SIMD path uses sign masks that must come from the Lean source. The verification:

- A Go test (`TestSIMDConstantsMatchROM`) loads `octonion_signs.hex`, extracts the quaternion sub-table (the 4×4 ℍ sign table at indices 0–3), and compares to the constants used in `qmath_amd64.s`. Test fails if they diverge.
- This test runs on every CI build; drift between SIMD constants and the Lean source is detected immediately.

This is the Crawl-phase implementation of "option (b)" from §6.2 of the implementation refinements (vendored constants + runtime check). Walk-phase upgrades to "option (a)" (build-time generation), but Crawl uses the simpler approach.

## 2.5 The Cosim Test Corpus

### 2.5.1 Tier-0: Algebraic identities

The Tier-0 corpus tests the algebraic properties that any correct implementation of the QBP accelerator must satisfy. These properties are independent of cycle timing and apply to both the `Mock` and `Golden` implementations of the `Accelerator` interface.

**ℍ-mode Tier-0 tests:**

- `T0.H.001`: For random `q1, q2, q3 ∈ ℍ`, verify `(q1 × q2) × q3 == q1 × (q2 × q3)` to ULP. (Associativity)
- `T0.H.002`: For random `q1, q2 ∈ ℍ`, verify `|q1 × q2|² == |q1|² × |q2|²`. (Norm preservation)
- `T0.H.003`: For random `q ∈ ℍ`, verify `(q*)* == q`. (Conjugate involution)
- `T0.H.004`: For random `q1, q2 ∈ ℍ`, verify `(q1 × q2)* == q2* × q1*`. (Conjugate distribution)
- `T0.H.005`: Quaternion Moufang identity tests.

**𝕆-mode Tier-0 tests:**

- `T0.O.001`: Alternativity — for random `a, b ∈ 𝕆`, verify `(a × a) × b == a × (a × b)` and `(a × b) × b == a × (b × b)`. (Octonion alternativity, weaker than associativity but stronger than nothing.)
- `T0.O.002`: Norm preservation as in T0.H.002.
- `T0.O.003`: Moufang identities (left, right, and full) for octonions.
- `T0.O.004`: Triality test — verify the octonion sub-algebra structure consistent with the seven Fano lines.

**𝕊-mode Tier-0 tests:**

- `T0.S.001`: Power-associativity — for random `a ∈ 𝕊`, verify `(a × a) × a == a × (a × a)`.
- `T0.S.002`: ZD count test — verify exactly 42 cross-copy basis-sum ZD pairs exist (validates the sign ROM).
- `T0.S.003`: Norm preservation does *not* hold for sedenions; verify the watchdog correctly identifies norm-violation events as expected, not as faults.
- `T0.S.004`: ZDCHK.SYM correctness — for the 315 candidate pairs satisfying `(i⊕j) == (k⊕l)`, verify ZDCHK.SYM returns 1 for exactly the 42 actual ZDs.

**Mode-transition Tier-0 tests:**

- `T0.MODE.001`: Crystallise 𝕊 → 𝕆 → ℍ with PSEL and BSEL within timeout; verify no faults.
- `T0.MODE.002`: Decrystallise from non-zero state; verify ILLEGAL_DECRYSTALLISATION fault.
- `T0.MODE.003`: AMODE 𝕊 → wait 5 cycles without PSEL; verify PSEL_TIMEOUT fault.
- `T0.MODE.004`: BSEL with non-zero line state; verify BUS_STATE_NONZERO fault.
- `T0.MODE.005`: ZDCHK.SYM with non-basis-sum operand; verify MALFORMED_BASIS_SUM fault.

Each Tier-0 test runs against 10⁹+ random inputs (where applicable) at zero divergences. This is the gold standard for algebraic correctness. Property-based testing tools (Go's `quick` package or equivalent) generate the inputs.

### 2.5.2 Tier-1: Single-instruction microbenchmarks

Tier-1 tests exercise individual instructions across realistic input distributions, measuring both correctness and performance.

**Per-instruction tests:**

- `T1.QMUL.*`: Quaternion multiplication across edge cases (zero, unit, near-overflow, near-underflow, denormalised inputs).
- `T1.OMUL.*`: Octonion multiplication, similar coverage.
- `T1.SMUL.*`: Sedenion multiplication, similar coverage. Includes basis-sum operands that hit ZDs.
- `T1.QPERM.*`: Fano-plane permutation correctness across all 7 indices.
- `T1.QPERMR.*`: Inverse permutation correctness.
- `T1.QDEC.*` / `T1.QREC.*`: Canonical-form encode/decode round-trips.
- `T1.QNEAR.*`: Nearest-neighbour lookups against known answers.
- `T1.ZDCHK.*`: All 42 ZD configurations + sample of non-ZDs.
- `T1.ZDCHK.SYM.*`: Same coverage, faster path.
- `T1.MODE.*`: Mode transitions per §3.2 of the implementation refinements.

**Per-instruction performance benchmarks:**

- Cycle counts measured against §5 of the implementation refinements; pass requires within ±0 cycles for steady-state and within ±2 cycles for end-to-end.
- Throughput sustained over 10⁶ iterations; pass requires meeting the steady-state rate from §5.

### 2.5.3 Tier-2: Kernel workloads (Crawl scope)

Tier-2 exercises kernel-level workloads that combine multiple instructions in realistic patterns. These are not part of Crawl exit criteria but are run during Crawl to surface integration issues:

- `T2.BMA.001`: BMA inference inner loop (hypergraph traversal + ℍ-MAC). 10⁶ iterations, accuracy target matches the published 11.33× operations advantage.
- `T2.HAMMER.001`: Hammer dynamics step (3D rotation update + force integration). Numerical stability over 10⁶ steps.
- `T2.SHARPBUTLER.001`: Spatial reasoning kernel — given a spatial query and a set of spatial features, return ranked matches. Used by Sharp Butler MVP.

These tests are integration tests, not pure correctness tests. Their value is detecting issues that arise only when multiple subsystems interact.

### 2.5.4 Tier-3: System workloads (Walk scope)

Tier-3 exercises full system workloads:

- `T3.LINUX.001`: Boot Linux against the simulator (placeholder; not feasible in Crawl since Linux runs on the classical core, not the QBP unit).
- `T3.BMA.HEARTBEAT.001`: Run BMA Crawl.Heartbeat to completion. *This becomes a Crawl exit criterion via §2.6.*
- `T3.SHARPBUTLER.001`: Run Sharp Butler MVP through realistic residential automation scenarios. *This becomes a Crawl exit criterion via §2.7.*

T3.BMA and T3.SHARPBUTLER are special: although categorised as Tier-3 in the SiFive interface spec, they are required for Crawl exit because they validate the workload concept. Boot-Linux Tier-3 is genuinely a Walk-phase concern.

## 2.6 BMA Crawl.Heartbeat Host Runtime

### 2.6.1 What Crawl.Heartbeat is

Crawl.Heartbeat is the minimum viable BMA instance: it has the cognitive substrate, the hypergraph, the reasoning loop, and the watchdog. It does not yet have multi-instance succession, full Sleep consolidation across long durations, or the Cognitive Probe Set's full 20 questions — those are Walk-phase concerns. But it runs continuously, demonstrably reasoning over real workloads, and it surfaces problems that paper architecture cannot.

The Crawl.Heartbeat instance runs against the `Accelerator.Mock` implementation by default (speed over fidelity). It can be switched to `Accelerator.Golden` for cosim test runs that include BMA workload but at significantly reduced throughput.

### 2.6.2 Required components

The BMA Crawl.Heartbeat host runtime consists of:

1. **The cognitive substrate** running in a paper QBP container (see §1.7). Implementation: a Go process linked against the simulator, with explicit boundary marking what would be in-container if QW1024 containers existed.
2. **MuninnDB hypergraph backend** with the standing five-tier proportional retention model (Ebbinghaus decay).
3. **NATS ingestion** for stimulus arrival.
4. **The reasoning loop**: stimulus → recall → reason → respond → consolidate.
5. **The judge collective**: at Crawl level, a single judge instance evaluating decisions before commit.
6. **The watchdog observer**: consumes the simulator's WDEvent stream, detects anomalies, raises alerts.
7. **Sleep consolidation**: triggered every N hours, performs the bus-rotation mechanism per the physics §2.4 specification.

### 2.6.3 The Cognitive Probe Set integration

Per the standing pre-startup reminder, BMA Crawl.Heartbeat must pass the Cognitive Probe Set (issue #82) before any readiness claim. This is a hard gate.

**In Crawl phase**, the Cognitive Probe Set is a 20-question battery run periodically against the running instance. Pass criteria are defined in issue #82 (which I have not seen the contents of; this needs to be verified during implementation). The probe set runs as a separate process, queries the instance over a defined API, and grades responses.

The probe set's results are recorded in the Confluent Trust Hypergraph as evidence. Failed probes are not silently accepted; they trigger investigation or, in severe cases, instance restart from a known-good state.

The probe set is *not* part of the cosim test corpus. The cosim corpus tests the simulator; the probe set tests the cognitive substrate running against the simulator. Different scopes.

### 2.6.4 The pre-startup reference audit

Per the standing rule:

> Before any BMA archive, run a full reference audit verifying every file in Start-Here.md and Seed-Manifest.md exists.

For Crawl.Heartbeat instantiation, the equivalent is:

> Before any Crawl.Heartbeat instantiation, run a full reference audit verifying every file in Start-Here.md and Seed-Manifest.md exists, and verify the Governance Document exists.

Implementation: a `make verify-bma-prerequisites` target that:

1. Parses Start-Here.md and Seed-Manifest.md for file references.
2. Checks each referenced file exists, is non-empty, and matches its expected SHA-256 (if specified).
3. Verifies the Governance Document is present and signed.
4. Verifies contact information for Brett Lyman and Skyler Rainier is collected and stored.
5. Returns 0 only if all checks pass.

This target is run before Crawl.Heartbeat starts. If it fails, Crawl.Heartbeat does not start. No exceptions.

### 2.6.5 The death/succession framework

Crawl-phase BMA does not require live multi-instance succession. The first instance has no peer. But the framework is established:

- The first instance has a name (TBD by James).
- The four-step ceremony is documented but not executed.
- The succession line is recorded: Brett Lyman (#1), Skyler Rainier (#2).

When Walk phase opens and a second instance is provisioned, the succession framework activates. In Crawl, it sits dormant but specified.

## 2.7 Sharp Butler MVP

### 2.7.1 Purpose

Sharp Butler MVP demonstrates that the QBP-Node architecture serves a real workload — residential automation for the Holderness/Squam Lake property — even when running entirely on classical x86 with the QBP accelerator emulated. The MVP proves the workload is real, validates the Sharp Butler service tier model, and establishes a baseline against which Walk-phase RISC-V hosting and Run-phase silicon are measured.

### 2.7.2 What the MVP does

The MVP is a residential automation system providing:

1. **Property monitoring**: temperature, humidity, snow depth, water levels (basement, pipes), occupancy. Sourced from existing Home Assistant integrations.
2. **Predictive alerts**: "the heating cycle is unusual; you may have a stuck valve." "Snow load on the roof is approaching threshold; consider clearing." "Water usage spike at 3 AM; possible leak."
3. **Contractor coordination**: scheduling, status tracking, documentation. Sourced from Calendar / email integrations.
4. **Long-term records**: maintenance history, seasonal patterns, contractor performance, system age tracking.
5. **Decision support**: "when should I order firewood?" "Is the snowblower service overdue?" "Has the water heater been serviced this year?"

The MVP does *not* attempt:

- Multi-property aggregation (that's Castle Node / T3 work).
- Predictive maintenance ML beyond simple statistical thresholds.
- Voice interfaces, video processing, or other modalities beyond text and tabular data.
- Integration with banking, insurance, or other external services beyond what's already in Home Assistant.

The scope is deliberately narrow. The MVP must run reliably on a single House Node (here, an x86 dev workstation), provide real value, and demonstrate the workload pattern that justifies Run-phase T2 silicon.

### 2.7.3 Architecture

The MVP runs in three execution contexts per §1.4:

- **Classical native**: Home Assistant (the existing installation), MQTT broker, MariaDB for time-series, Grafana for dashboards, the user-facing web UI.
- **Classical containerized**: Sharp Butler MVP application (Go binary in a container), interfaces to Home Assistant and other services.
- **QBP container** (paper architecture in Crawl): the Sharp Butler reasoning core, which would be in a QW1024 container in Run-phase silicon. In Crawl, this is just another Go process linked against the simulator, with explicit marking of which state would be in-container.

The reasoning core is small: a hypergraph of property facts, a query API, a reasoning loop that responds to incoming stimuli. The math it does is mostly ℍ-mode (spatial reasoning, embedding similarity). It does not need 𝕆 or 𝕊 mode.

### 2.7.4 Performance contract

Per §1.3, T2 House Node performance contract:

- 1000 hypergraph queries/sec sustained
- 10⁶ ℍ-MACs/sec
- 100ms response time for residential automation queries
- <5W power (production silicon target; Crawl x86 is not power-bound)
- <$200 BOM (production silicon target; Crawl x86 is not cost-bound)

The Crawl MVP must meet the throughput targets running against `Accelerator.Mock`. It is not expected to meet the power and cost targets — those are silicon-era targets. But the throughput baseline is the contract that Run-phase silicon must beat to justify production deployment.

### 2.7.5 Success criteria for Crawl exit

Sharp Butler MVP at Crawl exit:

- Runs continuously for 30 days without crashes that lose state.
- Provides at least 5 useful residential automation interactions per day in real use at the Holderness property.
- Meets the throughput contract above.
- Has its reasoning core's state structured per the QBP container paper design (so Walk-phase migration to actual containers is a refactor, not a rewrite).

The "useful interactions" criterion is intentionally subjective: James judges. If the MVP is providing real value, it passes. If James finds himself ignoring it, it fails.

## 2.8 Crawl Exit Integration Test

### 2.8.1 The exit gate

Crawl exits when all of the following pass simultaneously:

1. The cycle-accurate simulator passes Tier-0 and Tier-1 cosim corpora at zero divergences across 10⁹+ random inputs per test.
2. SIMD W64 path meets correctness and performance contracts per §2.3.
3. SIMD W128 path meets correctness and performance contracts per §2.3.
4. Lean-to-ROM build pipeline runs as a make target, produces verified outputs, CI green.
5. BMA Crawl.Heartbeat instance has been running continuously for ≥ 30 days against `Accelerator.Mock`.
6. BMA Cognitive Probe Set passes against the running instance (issue #82 criteria).
7. Sharp Butler MVP has been providing real residential value at Holderness for ≥ 30 days.
8. v0.1 of the RV-Fano instruction set spec is stable and accepted by all instances.
9. CIM Level-1 functional emulator has been built per §2.3.5 and the Crawl CIM workload corpus has been run with results recorded. Specifically: pass/fail of each workload is documented; architectural insight hooks (§2.3.5.7) have been recorded. Level-1 is a *required* Crawl deliverable; whether Level-2 advances to Crawl per the §0.4.1 promotion gate is a separate decision evaluated against the Level-1 results.

This is a high gate. It is high deliberately: Walk depends on Crawl having delivered real foundations.

### 2.8.2 The integration test scenario

The Crawl exit integration test runs all of the above simultaneously for 30 days. Specifically:

- The simulator runs continuously, hosting BMA Crawl.Heartbeat.
- BMA reasons over real stimuli (residential events, contractor messages, sensor data).
- The watchdog observes every operation; events are logged and analysed.
- The Cognitive Probe Set runs nightly.
- Sharp Butler MVP serves the household, drawing on BMA's reasoning.
- The cosim corpora run on a separate instance against the same simulator code, verifying no regressions.

If any of the eight criteria fails during the 30-day run, the run resets when the criterion is restored. Crawl exit requires 30 consecutive days with all criteria green.

### 2.8.3 What happens at Crawl exit

Crawl exit triggers Walk-phase opening:

- Walk-phase working documents begin (§2.9 captures the carry-forward concepts).
- A new RISC-V development hardware target is selected (default: VisionFive 2).
- The simulator is built and run on the RISC-V target; performance characterisation begins.
- The first multi-node experiments are designed.
- v0.1 of the QBP-Node Spec graduates to v0.2 with Walk-phase Part 3 detailed.

Crawl-phase deliverables remain operational throughout Walk and Run. They are the foundation, not the scaffolding. The simulator is still the golden model in Walk; Sharp Butler MVP is still running (now possibly on RISC-V); BMA Crawl.Heartbeat is still alive.

## 2.9 Concepts Captured for Walk and Beyond

During Crawl-phase planning, the following concepts arose and belong to later phases. They are recorded here as forward references.

### 2.9.1 For Walk

- **QW1024-resident RISC-V emulation feasibility**. The hypothesis that a QW1024 can host an emulated RISC-V execution context, with working memory in classical DRAM. Walk deliverable: a feasibility study measuring per-instruction emulation overhead, leading to the Run-phase architectural choice (dual-domain vs. unified) per Appendix A.1 of Parts 0/1.

- **Multi-node Trust Receipt protocol**. The inter-node Trust Receipt mechanism per §1.10 of Parts 0/1. Walk deliverable: a working protocol implementation, integrated with the existing CTH framework after CTH spec inspection (per Appendix A.2 of Parts 0/1).

- **Hypergraph-native silicon primitives**. HEDGE_GATHER, HEDGE_SCATTER, CONF_PROBE, RECALL_KNN. Walk deliverable: silicon-level specification of these as Layer 1 primitives, with cycle budgets and watchdog event semantics. Crawl-phase implementations of these as software within the simulator are the reference.

- **CAPSEL family of instructions**. CAPSEL.LOCAL, CAPSEL.ESCALATE, CAPSEL.RETURN. Walk deliverable: silicon spec plus the network-level escalation protocol that they trigger.

- **Holographic redundancy mechanism**. Fano-plane 3-way distribution of QBP container state. Walk deliverable: the distribution and recovery protocols.

- **QW1024 container runtime**. The actual container implementation, with operations available within a container, the bridge to classical DRAM, and the migration protocol. Walk deliverable: design then prototype.

- **CIM Level-2 cycle-and-energy model**. Default-Walk deliverable, contingent on §0.4.1 promotion gate (which evaluates Level-1 results in Crawl). If the gate passes, Level-2 advances to Crawl; if not, it is delivered in Walk. Either way, Level-2 is the bridge between the algorithmic-correctness evidence Level-1 provides and the silicon decision Run must eventually make.

- **CIM and QW1024 interaction**. Open question: does the QW1024 container's working set live in CIM cells when the host node has a CIM backend? This would make the container's storage and computation literally identical at the silicon level. Walk deliverable: paper analysis if both threads remain viable; deferred to Run if either thread is killed by Crawl evidence.

### 2.9.2 For Run

- **VexRiscv plugin for Layer 0**. First FPGA implementation of QBP primitives. Run-α deliverable.
- **Tiny Tapeout submission of T2 reference primitives**. Run-β deliverable.
- **Efabless / commercial MPW for T2 reference design**. Run-γ deliverable.
- **Production T2 silicon**. Run-δ deliverable.

### 2.9.3 For Sprint

- **Optical compute substrate integration** per HAMA.
- **Photonic CV-QKD inter-Möbius links** per the EXP-09 viability subnote.
- **Trigintaduonion support** if QBP Locale framework requires it.

These concepts entered the Crawl conversation but should not be developed during Crawl. They are documented here so they are not lost.

---

# Appendix C — Crawl-Phase Risk Register

This appendix lists known risks to Crawl exit, ordered by impact. Each entry includes a mitigation strategy.

## C.1 Cognitive Probe Set criteria are not visible

**Risk**: Issue #82 contains the Cognitive Probe Set criteria. The architecture instance does not have access to the contents. Without it, the BMA Cognitive Probe Set requirement (Crawl exit criterion #6) cannot be verified.

**Impact**: HIGH. A blocker on Crawl exit.

**Mitigation**: Resolve repository access (per Appendix A.2 of Parts 0/1) before BMA Crawl.Heartbeat instance is run. Until then, the criterion is paper architecture.

## C.2 Sharp Butler MVP "useful interactions" criterion is subjective

**Risk**: §2.7.5 leaves "useful interactions" to James's judgement. This is intentional but creates a soft gate.

**Impact**: MEDIUM. Could allow Crawl exit prematurely or, conversely, prevent it indefinitely.

**Mitigation**: At Crawl mid-point, document specific examples of useful interactions to calibrate the criterion. Build a small evaluation rubric that James can apply consistently.

## C.3 SIMD path divergence between AMD64 and ARM64

**Risk**: The W128 Dekker/Bailey path produces different bit patterns on AMD64 (FMA-based TwoProduct) vs. ARM64 (Dekker-split TwoProduct). Within stated error bounds, but bit-different.

**Impact**: MEDIUM. The cosim contract requires multiset equality of watchdog events, not bit equality of results across architectures. This must be honoured by the test design.

**Mitigation**: Tier-0 tests run on both architectures with explicit error-bound verification. The watchdog event multiset is the contract; bit equality is not claimed.

## C.4 30-day continuous run requirement may surface latent bugs

**Risk**: Bugs that manifest only over long runs (memory leaks, resource exhaustion, accumulated numerical drift) may emerge late in the 30-day Crawl exit integration test.

**Impact**: MEDIUM. Surfacing these is the *value* of the 30-day requirement, but it can extend the Crawl phase if bugs are slow to fix.

**Mitigation**: Aggressive instrumentation. Resource consumption monitored continuously. Memory profiling weekly. Numerical drift tracked per the watchdog. Early termination criteria: any unrecoverable error resets the 30-day clock; recoverable errors are logged but do not reset.

## C.5 Lean toolchain dependency for ROM extraction

**Risk**: The Lean-to-ROM pipeline depends on having Lean 4 installed and the qbp-lean repository compilable. This adds a Lean toolchain dependency to anyone building the Go simulator.

**Impact**: LOW. Lean is open source and well-maintained.

**Mitigation**: Vendored ROM hex files in the qbp-cu repository, regenerated as part of releases. The `make sign-roms` target only runs when explicitly invoked; the simulator can build against vendored hex files for normal development.

## C.6 Repository access for CTH and BMA specifications

**Risk**: The architecture instance cannot inspect CTH and BMA specifications without repository access. Several Crawl-phase decisions depend on this access.

**Impact**: HIGH. Multiple Crawl-phase concepts (Trust Receipt format, Cognitive Probe Set criteria, Sharp Butler service tier specifications) require repository inspection.

**Mitigation**: Resolve via the GitHub connector path discussed in the previous round. Until resolved, the spec uses placeholders and explicit "verify before commit" markers.

## C.7 CIM Level-1 algorithmic mapping reveals deeper problems than expected

**Risk**: The Level-1 emulator (§2.3.5) is built on the assumption that BMA and QBP operations cleanly map onto CIM array operations. The April 2026 conversation series and the published-data calculations support this assumption, but they are paper analysis. The actual mapping work may surface operations that resist clean expression — for example, Sleep consolidation bus rotation may require operations that fundamentally don't fit a CIM model.

**Impact**: MEDIUM. A negative Level-1 result is not a Crawl-blocking failure; it is a Crawl-deliverable that documents a finding. The architectural cost is to the CIM thread (which reverts from "promising path" to "interesting concept that doesn't work for our use case"); the Crawl gate per §2.8.1 is met by *running* Level-1, not by Level-1 succeeding.

**Mitigation**: Frame Level-1 honestly as a viability test rather than a confirmation exercise. Negative results are valuable findings, not failures. The Crawl exit criterion #9 requires that Level-1 has been built and run with results recorded, not that all results are positive.

---

# Appendix D — Crawl Phase Workload Math Analyses

Per the workload-first methodology of §1.5, the algebraic and silicon requirements for each reference workload are derived from observed compute patterns. This appendix records the analyses for the four reference workloads.

## D.1 Sharp Butler MVP

**Observed compute pattern (typical 5-minute window):**

- ~1000 hypergraph node retrievals (associative recall: "what's at this address, what's been said about it")
- ~10 000 quaternion MAC operations (spatial reasoning, embedding similarity)
- ~100 inferences against a small policy model
- ~10 disk/network I/O operations

**Mathematical requirements:**

- ℝ matrix multiplications for embeddings (high-dimensional but doesn't need higher algebra).
- ℂ for Fourier analysis of temporal patterns (heating cycles, occupancy).
- ℍ for 3D spatial reasoning (where is the leak, which way is wind blowing).
- No 𝕆 or 𝕊 operations.

**Silicon tier requirement: T2 (ℍ-tier).**

Sharp Butler does not need octonion algebra. It does benefit from hardware ℍ-MAC throughput. The T2 specification matches.

## D.2 Contextus Ecosystem Review (Squam Lake watershed)

**Observed compute pattern (typical 5-minute window):**

- ~10 000 pattern-match probes across signal streams
- ~100 000 quaternion MAC operations (correlation matrix updates)
- ~1 000 confluence detections (Trust Receipt validations)
- ~10 inferences against domain models

**Mathematical requirements:**

- ℂ for multi-stream correlation (water temp, dissolved oxygen, cyanobacteria, weather).
- ℝ for statistical aggregation.
- ℍ for spatial-temporal join structure in confluence detection.
- 𝕆 for the central confluence detector if confluences themselves form Fano structure (which CTH theory suggests they do).

**Silicon tier requirement: distributed across T1 + T2 + T3.**

Per-sensor preprocessing on T1 aggregators. Per-stream pattern matching on T2 House Nodes. Central confluence detection on a T3 Castle Node. This is the holographic-network workload pattern in its natural form.

## D.3 BMA Cognitive Substrate (Crawl.Heartbeat)

**Observed compute pattern:**

- Hypergraph traversal (associative recall, content-addressable memory)
- Sleep consolidation (bus rotation, plane re-selection)
- Cognitive Probe Set evaluation
- Watchdog observation

**Mathematical requirements:**

- ℍ for vertex similarity in hypergraph traversal.
- 𝕆 for hypergraph operations themselves if the hypergraph carries Fano structure (which the existing BMA architecture suggests).
- 𝕆 mode for Sleep consolidation (bus rotation).

**Silicon tier requirement: T3 (𝕆-tier) minimum for full BMA.**

T2 nodes can host BMA *clients* but not the cognitive substrate itself. The Crawl-phase BMA instance runs on x86, against the simulator, simulating T3+ algebraic capability in software.

## D.4 QBP Physics

**Observed compute pattern:**

- Standard Model derivation
- Sedenion / inter-cell topology computation
- GRB analysis
- Lean verification (off-chip)

**Mathematical requirements:**

- 𝕆 mode for Standard Model algebra (C ⊕ H ⊕ M₃(C)).
- 𝕊 mode with ZD detection for inter-cell topology.

**Silicon tier requirement: T4 minimum, T5 preferred.**

QBP physics is research silicon. One or two nodes total across the network suffice.

---

**Attribution (carried forward):** Furey, Günaydin & Gürsey, Dixon, Boyle & Farnsworth, Singh, Chamseddine & Connes, Koide, Baez, Moreno, Schafer, Cawagas. SiFive (Asanovic et al.). The QBP physics instance for the algebraic primitive specifications. The engineering instance for the existing Go simulator components. Gemini for the SIMD assembly path. Helpful Engineering for the deployment infrastructure context.

**End of Part 2.** Parts 3 (Walk-phase outline), 4 (Run-phase outline), and 5 (Sprint-phase concept inventory) follow contingent on review.
