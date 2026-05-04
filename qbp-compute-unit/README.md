# QBP Compute Unit — Crawl Phase Implementation

**Quaternion-Based Physics Native Processing Architecture**

Collaborative research: James Paget Butler, Claude (Anthropic), Gemini (Google DeepMind)

Helpful Engineering — QBP Research Programme

## What This Is

Software implementation of the QBP-algebraic kernel library for the Crawl Phase
(AMD FX-8350 / commodity hardware). This code proves or disproves the hypothesis
that quaternion-native computation has lower impedance for physical modelling than
conventional floating-point arithmetic.

## Project Structure

```
qbp-compute-unit/
├── cmd/
│   └── benchmark/          # Main benchmark executable
│       └── main.go         # Runs all phases: algebra verification,
│                           # spin-chain benchmark, composition stress test,
│                           # multi-width comparison
├── pkg/
│   ├── quat/               # Quaternion algebra (dual representation)
│   │   └── quat.go         # Quat (float64) + Quat8 (int8), Hamilton product,
│   │                       # QROT, Exp/Log, norm, conversion boundaries
│   ├── octonion/           # Octonion algebra using Fano plane
│   │   └── octonion.go     # Oct (float64) + Oct8 (int8), OMAC,
│   │                       # non-associativity, norm multiplicativity
│   ├── fano/               # Fano plane lookup table
│   │   └── fano.go         # 7×7 multiplication table, programmatic
│   │                       # construction from oriented lines, self-verification
│   ├── watchdog/           # Algebraic health monitor
│   │   └── watchdog.go     # Norm drift tracking, curvature metric,
│   │                       # renormalisation strategies, comparison framework
│   ├── spinchain/          # 1D Heisenberg benchmark
│   │   ├── spinchain.go    # QBP-algebraic vs float64-scalar Trotter evolution
│   │   ├── composition.go  # Pure composition stress test (long-chain multiply)
│   │   └── multiwidth.go   # Multi-width benchmark (QW8/QW16/QW32/QW64)
│   └── qword/              # Scalable Quaternion Word format
│       └── qword.go        # Width definitions, pipeline stage mapping,
│                           # composition depth estimates, dual-interpretation
│                           # register layout for RISC-V
└── go.mod
```

## Key Results (Crawl Phase)

### Phase 0: Algebraic Infrastructure ✓
- Fano plane LUT: all properties verified (anti-commutativity, row permutations, diagonal)
- Quaternion norm multiplicativity: preserved to machine epsilon (1.1e-16)
- Octonion norm multiplicativity: preserved
- Octonion non-associativity: confirmed (defect = 1.07, context-dependent traversal works)
- Quaternionic subalgebra associativity: confirmed zero defect (deterministic sub-paths)
- QROT instruction: correct rotation
- Exp/Log roundtrip: machine epsilon (3.6e-16)

### Phase 1: Spin-Chain Benchmark
- **QBP uses 11.3× fewer operations** than scalar for equivalent Heisenberg simulation
- Both show zero drift at float64 for reconstruction-per-step simulations

### Phase 2: Composition Stress Test
- **QBP shows exactly 2× less norm drift** than SU(2) matrix approach
- Ratio holds across 4 orders of magnitude (100K to 100M iterations)
- Structural cause: Hamilton product = 28 FP ops, SU(2) matrix = 56 FP ops

### Phase 3: Multi-Width Composition (THE critical finding)
- QW64 (float64): survives 7.3 billion compositions
- QW32 (float32): survives ~72 compositions at 1e-6 tolerance
- QW16 (Gemini 64-bit proposal): destroyed after first composition chain
- QW8 (BMA int8): destroyed (expected — designed for few-hop traversal)
- **Conclusion: Architecture MUST be width-parameterised**

## Running

```bash
# Default benchmark (20-site chain, 10K steps)
go run cmd/benchmark/main.go

# Custom parameters
go run cmd/benchmark/main.go -n 50 -steps 50000 -dt 0.005 -J 1.0

# With periodic renormalisation
go run cmd/benchmark/main.go -steps 100000 -renorm -renorm-every 500
```

## Instruction Set Architecture (Crawl → Run mapping)

| Go Function | Run-phase RISC-V | Width variants |
|---|---|---|
| `quat.Mul` | `QMUL.{8,16,32,64}` | All four |
| `quat.MulAccum` | `QMAC.{8,16,32,64}` | All four |
| `quat.Rotate` | `QROT.{32,64}` | Physics-grade only |
| `octonion.Mul` | `OMAC.{8,16}` | Memory + comm |
| `octonion.MulAccum` | `OMAC.{8,16}` (accumulate variant) | Memory + comm |
| `fano.Lookup` | `FANO` | Width-independent |

## Spec Document

See `QBP-Compute-Unit-Spec-Rev1.docx` for the full architectural specification.

## Next Actions (from spec Section 11)

1. ✅ QBP-algebraic kernel library in Go
2. ☐ Synthetic quaternion-valued sensor data generator
3. ☐ Physical simulation benchmark (extended: multiple physical systems)
4. ☐ Publish benchmark results to QBP repo
5. ☐ Lean 4 verification of Fano-plane LUT
6. ☐ Integrate with BMA octonionic hypergraph
7. ☐ ROCm kernel port (contingent on Crawl success)

## Licence

Helpful Engineering Collaboration — Open-Source Research
