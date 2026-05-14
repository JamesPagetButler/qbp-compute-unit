# QBP Compute Unit — Document Manifest

**Last reviewed:** 2026-05-06
**Repo:** `github.com/JamesPagetButler/qbp-compute-unit`

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

## Specifications & Architecture Documents (live, on `main`)

The Markdown corpus is the authoritative source for current architectural decisions. The `.docx` table above is legacy reference material; live work happens in these files.

### `spec/` — ISA and architecture specs

| File | Status | Description |
|---|---|---|
| `QBP-Compute-Unit-Architecture-v1.0.md` | v1.0 (current) | Hardware blueprint — Q-Pipe, Hamilton Engine, Q-Mem, CTH Watchdog → Constitutional Audit interrupt |
| `QBP-RISCV-Xqbpoct-Spec-v0.1.md` | **v0.1 (M0.5)** | Octonion extension — six mnemonics carved from v1.1 §2.1; shares `custom-0` opcode space + FP register file with `Xqbpquat` (`or_k ≡ qr_{2k} ∥ qr_{2k+1}`) |
| `QBP-RISCV-Xqbpvcp-Spec-v0.1.md` | **v0.1 stub (M0.5)** | Coprocessor dispatch interface — modeled on SiFive `XSfvcp` v1.1.0; reserves `qbp.amode/bsel/psel` CSRs + `mstatus.QBP` status bit (silicon-side actuator of ADR-003 §I3.4) |
| `QBP-Spec-Addendum-1_2-Worktree-Instructions.md` | Addendum | MuninnDB worktree isolation, copy-on-write, hardware resolution gradient |
| `QBP-Spec-Addendum-1_3-Cognitive-Git.md` | Addendum | NT_ISSUE / NT_PROPOSAL / `QSUTURE` — topological merge primitive |
| `QBP-Spec-Addendum-1_4-Stance-Gating.md` | Addendum | RelevanceMask + `QROT_GATED` — zero-tax stance switching |
| `QBP-Spec-Addendum-1_5-Honing-Metadata.md` | Addendum | NT_HONING_LOG + `QHON` — context-switch back to Primary Persona |
| `QBP-Spec-Addendum-1_6-Signal-Surfaces.md` | Addendum | NT_SIGNAL + `QSCAN` — low-energy background scouting |
| `QBP-Spec-Addendum-Walk-Acceleration.md` | Addendum | Software-defined Walk phase — `Quat256`, GAP, persona stance |

### `architecture/` — ADRs and peer reviews

| File | Description |
|---|---|
| `adr-001-stream-a-as-surface-stream-b-as-machine-model.md` | T1 closure — Stream A is the RISC-V surface form; Stream B is the underlying machine model |
| `adr-002-package-layout-emulator-stays-for-m0-m1.md` | T2 closure — existing `emulator/` layout retained for M0/M1; package reorg deferred |
| `adr-004-m1-gearbox-state-model.md` | LATE-4 closure (closeout Q4=A) — M1 Gearbox direction: CSR-bound stateful + QW8 peripheral surface + goroutine-pair concurrent dispatch with `OnSeam(callback)` per A18 §3 |
| `peer-review-001.md` | Red Team audit of "Algebraic Sovereignty" (Apr 2026) — sovereignty over fidelity, not scale |
| `peer-review-002-fano-mesh-isa-redteam.md` | Red Team audit of QBP RISC-V ISA v2.0 (Fano-mesh integration) plan — vendor-prefix conformance (NF1), `Xqbpvcp` / `Xqbpmesh` decomposition (NF2), Run-α premature-optimisation finding |
| `peer-review-005-stream-migration.md` | M0→M3 phased migration plan — Stream A v1.x (surface) + Stream B v2.0 (machine model); M0 cohort gates; Walk-α absorption-estimate; CIM Level-1 promotion gate |

### `reviews/` — PR-scoped audits

| File | Description |
|---|---|
| `peer-review-006-wdevent-pr11-redteam.md` | PR #11 audit (WDEvent + QW128) — §7 priority list; Item 4 (perf regression) carved out of M0.2 close per §10 |

### `doc/` — Application notes and context briefs

| File | Description |
|---|---|
| `wyrd-integration.md` | **v0.2 surface spec** — typed-per-width Gearbox API; Q1/Q2/Q3 architecture-locked (canonical import path, Gearbox/Accelerator separation, Tier ⊥ Width orthogonality) |
| `wyrd-substrate-guarantees.md` | **v0.1.0-rc1 substrate contract** — four-lens guarantees (Robust / Efficient / Precise / Accurate); six known-risk surfaces; concrete Wyrd PR #2 swap contract; Walk-α audit deferred to `v0.2.0-rc1` |
| `BMA-Emulator-Integration.md` | 8 precision levels; cognitive-mode → QW mapping; benchmark + integration story |
| `Hardware-Strategy.md` | Three Walk-phase upgrade paths (Threadripper / 9900X / RISC-V ASIC) |
| `QBP-ISA-Refinement-Report.md` | M0 ISA gap analysis — input to Gemini's v1.1 work |
| `QBP-RISCV-ISA-Spec-for-Gemini.md` | Original task brief Gemini was given for the v1.1 spec |
| `lean2rom.md` | M0.1 invocation, error modes, regeneration policy |
| `briefing_xqbp_cognitive_stack.md` | Cognitive-stack onboarding |
| `CLAUDE-GEMINI-PROTOCOL.md`, `CLAUDE-RESTART-CONTEXT.md`, `GEMINI-CONTEXT.md` | Multi-instance coordination protocols |

### `Ref/` — Authoritative reference docs

| File | Description |
|---|---|
| `RISC-V-Policies-and-Best-Practices.md` | RISC-V International ratified-maturity workflow; vendor-prefix policy; custom opcode-space rules; V-extension configuration template |
| `SiFive-Documentation-Patterns.md` | SiFive `Xsf*` extension catalog; VCIX coprocessor-interface model; versioning maturity ladder |

### `Archive/` — Migration-artifact docs (Stream B + Node Spec)

| File | Description |
|---|---|
| `RV-Fano-Implementation-Refinements.md` | Stream B authoritative source — Layer 0/1/2 mode-aware RV-Fano ISA; ZDCHK.SYM; sign-ROM extraction from `Sedenion.lean` |
| `QBP-Node-Spec-v0.1-Parts-0-and-1.md` | Phasing model + deferred-decisions philosophy |
| `QBP-Node-Spec-v0.1-Part-2.md` | Crawl-phase deliverable inventory (incl. `lean2rom` + CIM Level-1) |

### Root-level

| File | Description |
|---|---|
| `CHANGELOG.md` | Release-candidate entry under `v0.1.0-rc1` covering the M0 cohort (12 PRs); follow-up sections under `Unreleased` track M1+ work |

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
| AVX-FMA QW64 kernel | Delivered via PR #11 (`emulator/qmath_amd64.s`) | CONFIRMED |
| AVX-FMA QW128 double-double kernel | Delivered via PR #11 (`emulator/qmath_128_amd64.s`) | CONFIRMED — 5–8% perf overhead from WDEvent emission carved out as `reviews/peer-review-006` §7 Item 4 (owner: Gemini) |
| Lean → ROM authority chain | Delivered via PR #12 (`make verify-roms` + `TestSIMDConstantsMatchROM`) | CONFIRMED |

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

**Done since 2026-04-15 review:**

- ~~Write AVX assembly kernel~~ — delivered via PR #11 (QW64 + QW128 fast paths)
- ~~Lean 4 verification (Fano LUT + norm preservation)~~ — `lean/QBP/Sedenion.lean` + ROM-extraction pipeline delivered via PR #12

**M0 cohort still in flight (epic [#3](https://github.com/JamesPagetButler/qbp-compute-unit/issues/3)):**

1. **M0.1b** ([#13](https://github.com/JamesPagetButler/qbp-compute-unit/issues/13)) — Extend `TestSIMDConstantsMatchROM` to QW128 sign masks. Owner: Gemini. ~30 min.
2. **M0.2.1** ([#14](https://github.com/JamesPagetButler/qbp-compute-unit/issues/14)) — Migrate `qmath_amd64.s` to generated `lean_sign_*` constants. Owner: Gemini. ~1 h. Closes the authority chain at the asm level.
3. **M0.5** ([#9](https://github.com/JamesPagetButler/qbp-compute-unit/issues/9)) — `Xqbpoct` + `Xqbpvcp` v0.1 stubs. **v0.1 approved by qbp-architecture; bma + bma-implementor governance read pending per ADR-003 §I4.**
4. **Wyrd integration interface** ([#10](https://github.com/JamesPagetButler/qbp-compute-unit/issues/10)) — Public API surface for Wyrd / BMA / Contextus / Sharp Butler consumers.
5. **PR #11 perf regression** (`reviews/peer-review-006` §7 Item 4) — QCONJ +8.5%, QMUL128 +8.3%, QADD128 +6.3%; carved out of M0.2 close per `peer-review-006` §10. Owner: Gemini.
6. **Cross-repo CI tokens** ([#15](https://github.com/JamesPagetButler/qbp-compute-unit/issues/15)) — `QBP_PAT` / `WYRD_PAT` for the Wyrd-compatibility canary.
7. **Binary leak cleanup** ([#16](https://github.com/JamesPagetButler/qbp-compute-unit/issues/16)) — 4 build artifacts (~12.4 MB) on main + `.gitignore` audit.

**Walk-phase staging (epic #3 M1+):**

- **Stream B Layer 0 (M1)** — `qbp.amode` / `qbp.bsel` / `qbp.psel` CSRs, mode-aware dispatcher, fault codes 0x10–0x14, active WDEvent observer goroutine. Per `architecture/peer-review-005-stream-migration.md` §M1.
- **ROCm port** — QMUL/OMAC/Fano for RX 9070 XT (Walk phase).
- **Port benchmark to FX-8350** — wall-clock numbers on target Crawl hardware.
- **BMA hypergraph integration** — spreading activation using compute-unit kernel; gated on Wyrd integration interface (#10).

**Cleanup:**

- Remove `BMA-Crawl-Environment.docx` (duplicate) and `.txt` conversion files (per existing review note).
