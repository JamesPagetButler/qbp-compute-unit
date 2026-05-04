# QBP Compute Unit — Document Manifest

**Last reviewed:** 2026-04-15
**Repo:** `github.com/JamesPagetButler/qbp-compute-unit` (proposed)

## What the QBP Compute Unit Is

A Go-native algebraic computing architecture for the Quaternion-Based Physics programme. Implements the sense-compute-act algebraic pipeline where quaternion structure is preserved from sensor to actuator. The architecture's central thesis: a processor whose native operations are quaternion-algebraic exhibits measurably lower impedance when modelling physical systems.

**Key finding (Crawl-phase confirmed):** The algebra works. 11.3× fewer operations for QBP-algebraic vs scalar spin-chain simulation. 2× better norm preservation in composition stress tests. Both are hardware-independent structural results. Wall-clock on FX-8350 is PENDING.

## Documents on Disk

| File | Status | Description |
|---|---|---|
| `QBP-Compute-Unit-Spec-Rev1.docx` | Reference | First architectural spec. Sense-compute-act pipeline, Crawl/Walk/Run/Fly phases, QMUL/QROT/OMAC/FANO ISA. Supersedes Gemini's FQCC. |
| `QBP-Compute-Unit-Spec-Rev2.docx` | **Current spec** | Adds: (1) Optical SU(2) pipeline (Ammendola et al., March 2026 Light:S&A), (2) NV-centre spin-photon interface, (3) **Glide phase** between Run and Fly. QW128 empirically validated as starting point for physics. |
| `QBP-Compute-Unit-Master-Record.docx` | **Authoritative** | Complete session record. 4,594-line Go codebase (11 packages). All empirical results. Honest CONFIRMED/PENDING/THEORETICAL labelling. The document to read for current state. |
| `QBP-Compute-Unit-Walk-Eval.docx` | Walk planning | BMA integration recommendation. **Key insight: spreading activation on typed hypergraph IS ternary matrix-vector multiply — same inner loop, same assembly kernel.** Recommended as Walk-phase BRIDGE provider. |
| `BMA-Crawl-Environment.docx` | **Duplicate** | Copy of BMA Archive document. No unique content. Can be removed. |
| `qbp-compute-unit` | Binary | Compiled Go binary. |
| `qbp-compute-unit-final.tar.gz` | Archive | Source archive. |

**Converted .txt files** from the docx files were created during review (2026-04-15) and can be deleted — the .docx are the canonical source.

| File | Purpose |
|---|---|
| `GEMINI.md` | Gemini context file — project overview for resuming Gemini sessions |
| `RESTART_INSTRUCTIONS.md` | Session restart instructions for continued development |

## Hardware Progression

| Phase | Hardware | Status |
|---|---|---|
| Crawl | AMD FX-8350, 32GB DDR3, Go software | **In progress** — operation counts confirmed, wall-clock PENDING |
| Walk | RX 9070 XT (RDNA 4, 16GB), ROCm | Planned — ROCm QMUL/OMAC/Fano kernels needed |
| Run | Custom RISC-V with QMUL/OMAC/QROT/FANO, OpenMPW 130nm | Future |
| **Glide** (new in Rev2) | Commercial SLMs + NV-centre in diamond cavity | Validates algebraic preservation across optical-spin boundary |
| Fly | Monolithic synthetic diamond | Far future |

## Key Empirical Results (Confirmed on Cloud Hardware)

| Result | Value | Type |
|---|---|---|
| Op advantage (spin-chain) | 11.3× fewer operations | CONFIRMED (hardware-independent) |
| Norm preservation | 2× better | CONFIRMED (hardware-independent) |
| QW128 composition lifetime | 172 days @ 1GHz | CONFIRMED via big.Float |
| QW64 composition lifetime | 7.3 seconds @ 1GHz | CONFIRMED |
| Wall-clock QBP vs scalar | Pending FX-8350 measurement | PENDING |
| AVX assembly kernel | Not written | PENDING — needed for Crawl completion |

## Precision Architecture

The empirical data identifies three natural regimes:
- **Below QW64** (<256 bits): Nanosecond lifetime. Suitable for hypergraph traversal (QW8), sensor ingestion (QW16), GPU-native computation (QW32).
- **QW64** (256 bits): ~7 second lifetime. Requires periodic renormalisation.
- **QW128+** (≥512 bits): 172+ day lifetime. **Recommended starting point for physics computation.**

## BMA Integration (Walk Phase)

From Walk-Eval: spreading activation on the BMA octonionic hypergraph IS ternary matrix-vector multiply. The compute unit's kernel handles both:
- Dense matrix (model weights) → inference mode
- Sparse matrix (hypergraph adjacency) → retrieval/spreading activation mode

Same assembly kernel, two modes. BMA doesn't need two separate compute systems.

**Benchmark gate (March 2026):** Go within 1.53× of C for ternary matmul (medium matrix 4096×4096). The 4.9× gap against C with SIMD is entirely closeable with Plan 9 assembly — no CGo.

## Relationships to Other Projects

| Project | Relationship |
|---|---|
| **QBP** | Primary research programme. QBP simulations are natural workloads for the compute unit. |
| **BMA** | Walk-phase BRIDGE provider. Shared algebraic kernel for inference + hypergraph traversal. Fano LUT as canonical implementation for hypergraph edge composition. |
| **Sharp Butler** | Layer 2 (Compute Mesh) — House Nodes contribute compute using the same architecture. Layer 4 (Deep Compute) — co-located with Möbius reactor. |
| **Möbius Fusion** | Glide phase requires 4K cryogenic environment for NV-centre cavity — thermionic cascade cooling from Möbius architecture. |

## Pending Actions

1. **Port benchmark to FX-8350** — get real wall-clock numbers on target Crawl hardware
2. **Write AVX assembly kernel** — closes 4.9× SIMD gap, validates Crawl-phase completion
3. **ROCm port** — QMUL/OMAC/Fano for RX 9070 XT (Walk phase)
4. **Lean 4 verification** — Fano LUT and norm preservation
5. **Integrate with BMA hypergraph** — spreading activation using compute unit kernel
6. **Clean up:** Remove `BMA-Crawl-Environment.docx` (duplicate) and `.txt` conversion files
