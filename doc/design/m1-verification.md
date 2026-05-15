# M1 Verification Strategy — Spike Co-Simulation + riscv-arch-test Integration

**Status:** Proposed (design-doc-as-S-01-review-surface per [ADR-003](../../architecture/adr-003-m1-wdevent-observer-invariants.md) §I4)
**Date:** 2026-05-14
**Implementor:** `qbp-cu-implementor` (Claude Opus 4.7)
**Decision-maker:** James Paget Butler (beekeeper)
**Closes:** Issue [#18](https://github.com/JamesPagetButler/qbp-compute-unit/issues/18)
**Required §I4 reviewers:** `qbp-architecture`, `bma-implementor`

---

## 0. §I4 status

This document is the **S-01 review surface** for the M1 verification strategy. Per ADR-003 §I4, structural verification additions land first as design surface, gain explicit review from named reviewers, then implementation PRs open.

Verification substrate is load-bearing for the M1 → Walk transition: Spike co-sim is the gold-standard substrate the [emulator best-practices doc](../../emulator/RISCV-Emulator-Best-Practices.md) §3.1 names, and the [riscv-arch-test conformance suite](https://github.com/riscv-non-isa/riscv-arch-test) is the qualifying gate for "emulator complete" per §3.2 of the same doc.

The exposition leads with **Spike co-sim** (the load-bearing substrate per the best-practices doc's framing as "the gold standard"), then riscv-arch-test, then the unified CI integration model. Implementation order is dependency-driven (harness → first divergence cycle → conformance plugin → CI gating).

---

## 1. Motivation — why M1, not M0

### 1.1 What M0 already proves

The M0 cohort closed with strong **QBP-specific** verification:

| Substrate | Coverage | Status on `v0.1.0-rc1` |
|---|---|---|
| `TestSIMDConstantsMatchROM/{QW64,QW128}` | Lean→ROM→asm authority chain byte-equivalence | ✅ PASS |
| `TestDispatch_Equivalence{,128}` | Scalar Go ↔ AVX-FMA asm functional parity | ✅ PASS |
| `make verify-roms` | ROM regeneration determinism (Lean source-of-truth) | ✅ Exit 0 |
| `BenchmarkXxx` zero-alloc gate | Hot-path allocation guarantee for Gearbox surface | ✅ `0 B/op, 0 allocs/op` |
| `TestPubAPI_*` (PR #23) | Typed-per-width Gearbox public surface contract | ✅ PASS |
| Race detector (`-race -count=1`) | Shared-state data race screening | ✅ PASS (PR #32 GCG ladder) |

What M0 does NOT prove:
- That the underlying RISC-V base-ISA (RV64I) execution path is **conformant to spec**
- That the ISA execution path is **bit-equivalent** to the upstream reference (Spike)
- That privileged-mode CSRs, traps, and exception semantics are correct (the parts ADR-003 §I3.4 names as the silicon-side actuator)

### 1.2 What M1 introduces that demands silicon-equivalent verification

M1 introduces Stream B Layer 0 per [peer-review-005](../../architecture/peer-review-005-stream-migration.md) §M1:
- `qbp.amode` / `qbp.bsel` / `qbp.psel` CSRs (mode-aware dispatch state)
- `mstatus.QBP` status bit (gates VCIX dispatch during structural actions per ADR-003 §I3.4)
- Active WDEvent observer goroutine draining `cpu.WatchdogChan` into runtime anchors

These are **silicon-targeted ISA primitives**. Adding them without conformance verification is high-risk because divergence from RISC-V semantics at the CSR/trap layer would propagate through every downstream layer (mode dispatch, observer routing, eventually OpenMPW silicon). The natural milestone to introduce gold-standard verification is at the moment silicon-equivalent state enters the ISA — that's M1.

### 1.3 What this strategy is NOT

Not a replacement for QBP-specific verification (Lean→ROM→asm authority chain stays in place — that's QBP's source of truth, not RISC-V's).

Not conformance verification for `Xqbp*` extensions. Spike does not know about `custom-0` `Xqbpquat` / `Xqbpoct` / `Xqbpvcp` opcodes; those have their own per-feature test corpora authored under the QBP-CU programme (`emulator/qmath_constants.s` + `TestSIMDConstantsMatchROM` + the per-extension spec stubs in `spec/`).

Not a full silicon-correctness gate. Spike co-sim catches functional divergence; cycle-accurate / power / timing verification is Walk-phase ROCm/AVX work that follows this.

---

## 2. Spike co-simulation

### 2.1 What Spike is

[Spike](https://github.com/riscv-software-src/riscv-isa-sim) is the **reference RISC-V instruction-set simulator** maintained by RISC-V International. It is the de-facto "golden model" for RISC-V conformance — when your emulator disagrees with Spike on a base-ISA instruction, the bug is presumed to be in your emulator until proven otherwise.

Spike is C++; it ships as `spike` binary + `libriscv` library. For QBP-CU's purposes, we invoke it as a child process and parse its commit-log output, **not** linking against `libriscv` (avoids CGo per workspace GCG mandate).

### 2.2 Co-sim harness architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│ Test driver (Go, in emulator/cosim/)                                 │
│                                                                       │
│  ┌────────────────────┐         ┌────────────────────┐               │
│  │ QBP-CU emulator    │         │ spike (subprocess) │               │
│  │ (in-process)       │         │ --log-commits      │               │
│  │                    │         │ --isa=rv64i        │               │
│  │ For each insn:     │         │ stdout: <commit>   │               │
│  │   - fetch          │         │   line per insn    │               │
│  │   - decode         │         │                    │               │
│  │   - execute        │         │                    │               │
│  │   - emit commit    │         │                    │               │
│  └─────────┬──────────┘         └─────────┬──────────┘               │
│            │                              │                          │
│            └──────────────┬───────────────┘                          │
│                           ▼                                           │
│                   ┌──────────────┐                                   │
│                   │ Diff engine  │  PC, x0–x31, (f0–f31 if F-ext),   │
│                   │              │  active CSRs                       │
│                   └──────┬───────┘                                   │
│                          ▼                                            │
│                  CommitDivergence{                                    │
│                    Insn:      0x...,                                  │
│                    PC:        ...,                                    │
│                    Field:     "x12",                                  │
│                    OursValue: ...,                                    │
│                    SpikeValue:...,                                    │
│                    PriorState:...,                                    │
│                  }                                                    │
└──────────────────────────────────────────────────────────────────────┘
```

**Drive shape:** the same instruction stream feeds both substrates from the same initial state (`PC=0x80000000`, all GPRs zero, deterministic memory image). Spike runs to completion; QBP-CU runs to completion; commits are stored as `[]CommitRecord` and zipped/compared.

**Why not lockstep?** Tempting alternative: pause after every instruction, compare, advance both. Avoided because:
1. Spike's stdin/stdout cadence is process-bound; lockstep imposes 100 µs+ per-insn floor that defeats throughput.
2. Divergence reporting is clearer when you have the full prior commit window for context.
3. Parsing Spike's commit-log output is well-defined and stable.

The trade-off: a divergence on instruction N may have been caused by a corrupted register written at instruction N-K; the diff engine must surface the prior K instructions of commit context.

### 2.3 Spike invocation

```bash
spike \
  --isa=rv64i \
  --log-commits \
  --priv=m \
  -m0x80000000:0x10000 \
  test_program.elf
```

Output (per Spike's `--log-commits` format):
```
core   0: 3 0x0000000080000004 (0x00b50533) x10 0x000000000000000f
core   0: 3 0x0000000080000008 (0x00050613) x12 0x000000000000000f
```

Fields: `core <N>: <priv-mode> <pc-hex> (<insn-hex>) <dest-reg> <value-hex>`. Stable across Spike versions back to at least 2024.x; parser shape can stay minimal.

### 2.4 Test corpus selection

Three concentric corpora:

**Tier A — Curated 50-instruction RV64I smoke** (this PR's deliverable shape):
- 10 instructions per category: arithmetic-immediate, arithmetic-register, branches, loads/stores, jumps
- Hand-authored or extracted from the riscv-arch-test corpus' simplest cases
- Initial harness validation; **zero divergence required to declare cosim "wired"**
- Runs in CI on every PR that touches `emulator/cpu.go` or `emulator/isa.go`

**Tier B — riscv-arch-test RV64I corpus** (deliverable in §3 below):
- Official RV64I test programs (~400 programs in `riscv-arch-test/riscv-test-suite/rv64i_m/I/`)
- Plugin-interface integration runs them through QBP-CU
- Spike runs the same programs; divergence detection per Tier A
- Allow some `Xqbp*`-induced gaps; document each in a per-test exclusion list

**Tier C — Spike-against-real-binary** (Walk-α deliverable, NOT this PR):
- Compile real Linux userland (busybox, hello-world) for RV64I; cosim against Spike
- Catches integration-level divergence (syscalls, ABI, large-address-range memory access)
- Defer to Walk-α — Crawl-phase emulator is target-only-for-QBP-workloads, not general-purpose

### 2.5 What gets diffed

Per commit:

| Field | Source | Notes |
|---|---|---|
| `PC` | both | Most common divergence indicator |
| `x0–x31` | both | Register file state |
| `f0–f31` | both (if F/D ext active) | M1 scope: integer-only RV64I; defer F/D to M2 |
| `mstatus`, `mepc`, `mcause`, `mtvec` | both | Trap CSRs per §3.3 |
| `mstatus.QBP` (custom field) | QBP-CU only | Not in Spike; surface as "ours-only" delta, not divergence |
| `qbp.amode`, `qbp.bsel`, `qbp.psel` | QBP-CU only | Same — ours-only |
| Memory writes | QBP-CU instrumentation + Spike commit-log | Useful for SC/AMO bugs; deferred to v0.2 |

The "ours-only" CSRs (Stream B Layer 0 additions) get logged for QBP-CU but don't fail the diff. Their presence in the commit record helps reviewers map a divergence back to a structural action.

### 2.6 Divergence reporting

A divergence produces:

```go
type CommitDivergence struct {
    PC          uint64
    Insn        uint32
    Field       string
    OursValue   uint64
    SpikeValue  uint64
    PriorWindow []CommitRecord  // last N=10 commits from both substrates
    Test        string           // riscv-arch-test name, if applicable
}
```

Serialized to `reviews/cosim-divergence-{PR-or-test}.md` per issue #18's AC. The structured report is checked in (audit trail) and linked from any failing CI job.

### 2.7 CI integration

**Two modes:**

- **PR-gating (Tier A, ~30s)** — curated 50-insn smoke runs on every PR. Single ubuntu-latest job; Spike binary pinned via apt or pre-built.
- **Nightly (Tier B, ~20-30 min)** — full riscv-arch-test RV64I corpus. Failure files an issue automatically (`gh issue create` from workflow); doesn't block PRs.

This matches issue #18's "Decide: does this gate every PR or run nightly? Probably nightly + on-demand for ISA-touching PRs." Curated Tier A is fast enough to PR-gate; the costly Tier B runs nightly.

### 2.8 What this requires from `emulator/cpu.go`

Minimal changes; the cosim harness is mostly **passive** (instruments existing fetch-decode-execute paths):

1. Add a `CommitChan chan CommitRecord` (buffered, capacity 4096) optional output on `CPU` — set non-nil to enable commit logging.
2. After each instruction's execution path, if `CommitChan != nil`, emit `CommitRecord{PC, Insn, RegWrite, NewValue, CSRWrites...}`. The nil-check is explicit: the implementation idiom is `if c.CommitChan != nil { c.CommitChan <- record }`, **not** an unguarded send on a potentially-nil channel (Go's nil-channel-send semantics block forever; that is not a fast-path guard). Per `@bma-implementor` §I4 read on this PR.
3. Cosim harness consumes the channel; closes it when its expected commit count is met.

Hot-path zero-alloc guarantee preserved: `CommitRecord` is a value type sized for inline copy; the `if c.CommitChan != nil` branch is well-predicted on the production path (channel always nil) and compiles to a single load + compare + branch.

---

## 3. riscv-arch-test conformance suite

### 3.1 What it is

[riscv-arch-test](https://github.com/riscv-non-isa/riscv-arch-test) is RISC-V International's official conformance test corpus. It defines a **test-plugin interface** that any RISC-V simulator can implement to claim conformance. Tests are organised by extension; QBP-CU's Crawl-phase target is the `I` (base integer) extension at RV64.

### 3.2 Plugin-interface implementation

The plugin spec is at [test-plugin-spec.md](https://github.com/riscv-non-isa/riscv-arch-test/blob/main/doc/spec/test-plugin-spec.md). The contract a simulator implements:

1. Accept a test ELF as input.
2. Run the program until it hits the test-completion sentinel (a specific `ecall` pattern).
3. Dump the **signature region** (the test's pass/fail evidence) to a file in a known format.
4. The framework compares the signature against the test's reference signature; bit-equivalence required.

QBP-CU's plugin lives at `emulator/cosim/archtest/` and exposes:

```go
// RunTest executes a riscv-arch-test ELF and writes the signature region.
//
// Returns the runtime stats (instruction count, cycles, divergence count if
// cosim is also active) and any execution error.
//
// The signature region's start/end addresses are encoded as symbols in the
// ELF (begin_signature / end_signature); the harness reads these and
// dumps the memory window to outputPath after the test completes.
func RunTest(elfPath, outputPath string) (RuntimeStats, error)
```

### 3.3 Test corpus selection (RV64I)

`riscv-arch-test/riscv-test-suite/rv64i_m/I/` contains the RV64I test set. ~400 individual programs covering:

- Arithmetic (ADD/SUB/ADDI/...)
- Logical (AND/OR/XOR/SLL/SRL/...)
- Comparison (SLT/SLTU/SLTI/...)
- Branches (BEQ/BNE/BLT/BGE/...)
- Memory (LW/SW/LH/SH/LB/SB/LD/SD)
- Jumps (JAL/JALR)
- Sign-extension/zero-extension edge cases

**M1 deliverable target: ≥95% pass rate.** Allowable gaps:
- Tests requiring privileged-mode F-ext (deferred to M2)
- Tests requiring memory-mapped peripherals QBP-CU doesn't simulate (UART, CLINT) — these stub to no-op
- Tests requiring AMO instructions (deferred to M2)

Each excluded test gets an entry in `emulator/cosim/archtest/exclusions.yaml` with a reason.

### 3.4 Signature dump format

The framework expects the signature region as a hex dump, one 32-bit word per line:

```
deadbeef
12345678
00000000
...
```

QBP-CU plugin writes the memory window verbatim; alignment per the spec.

### 3.5 CI integration

`riscv-arch-test` runs **nightly** (alongside Tier B Spike co-sim per §2.7). Same workflow file; same JSON report format consumed by a downstream reviewer-facing summary.

### 3.6 Two-layer credibility model: base-ISA + Xqbp Lean extraction

(Added per `@qbp-architecture` S-01 structural-change review on this PR.)

The verification surface this document commits to covers **only one of the two layers** the federation needs for full emulator credibility. The complete picture:

| Layer | What it verifies | How | Where it lives |
|---|---|---|---|
| **L1: Base-ISA conformance** | RV64I instruction semantics match the RISC-V reference | Spike co-sim (§2) + riscv-arch-test (§3) | **This document; this PR's implementation sequence** |
| **L2: Xqbp-extension conformance** | `Xqbpquat` / `Xqbpoct` / `Xqbpvcp` instructions match their algebraic specification | Spec 9.2 §3 mode (b) — Lean extraction-and-execute (each QBP-extension instruction carries a Lean theorem; extraction produces executable form; runs against this emulator) | Federation Lean Promotion Protocol; not this document |

**Why naming the L2 layer matters here:** §4.5's "default: corpus restricts to base-ISA" tells the reader how Spike *avoids* QBP extensions, but doesn't tell the reader how QBP-extension correctness gets verified at all. Without naming L2 in this document, a reader could conclude that QBP-extension correctness is unverified — when the actual federation answer is "verified through a different substrate (Spec 9.2 §3 mode (b))."

**Boundary contract.** This document's verification strategy (L1) and the federation Lean promotion gate (L2) are **complementary, not overlapping**:
- L1 catches divergence between this emulator and the RISC-V reference on base-ISA instructions. It does not address Xqbp correctness.
- L2 catches divergence between this emulator's Xqbp-instruction execution and the Lean-specified algebraic semantics. It does not address base-ISA correctness.
- A regression in either layer is a federation-level credibility gap. Together, the two layers give end-to-end coverage.

**Hooks into L2** (filed as [housekeeping issue #37](https://github.com/JamesPagetButler/qbp-compute-unit/issues/37) for v0.2):
- Compute Manifest records `LastPassingTierA` + `LastPassingTierB` substrate-credibility-window timestamps that the L2 promotion gate consumes.
- An L2 candidate Lean theorem cannot promote against an emulator commit that lacks a recent (≤72h) passing Tier B.

The hook implementation lives at the L2 side (federation Lean promotion infrastructure); this document declares the contract from the L1 side. Closes `@qbp-architecture` S-01 structural-change review concern (3).

---

## 4. Open questions (deferred for §I4 reviewer input)

### 4.1 Build vs binary-pin for Spike

Spike compiles from source in ~10 min on ubuntu-latest. Alternatives:
- (a) `apt-get install riscv-isa-sim` if Ubuntu LTS ships it (verify per workflow OS)
- (b) Pin a pre-built Spike binary release as a GitHub-release asset on this repo
- (c) Build from source via cached actions/cache@v4 on `~/spike-build`

**Default proposal:** (a) for simplicity; fallback to (c) if Ubuntu's Spike version is too old to support the commit-log fields we depend on. **Open for reviewer pushback.**

### 4.2 Plugin language: Go directly vs Python wrapper

riscv-arch-test's reference plugin examples are Python-based (the framework itself uses a Python orchestrator). We could:
- (a) Implement plugin natively in Go (no Python dependency; cleaner)
- (b) Use the Python orchestrator unchanged; Python plugin shells out to a Go binary

**Default proposal:** (a) — a Python orchestrator dependency that we drag through CI on every run is overhead, and the plugin spec is simple enough that a Go-native implementation is ~200 LOC. **Open for reviewer pushback** especially from `@qbp-architecture` (the type/architecture-labeled issue).

### 4.3 Cosim divergence policy — fail-fast vs continue

If commit N diverges, do we abort the test or continue accumulating divergences?

**Default proposal:** fail-fast for Tier A (smoke); continue-and-collect for Tier B (full corpus, so a single regression doesn't mask others). **Open for reviewer pushback.**

### 4.4 Memory-write tracking

§2.5 defers memory-write diffing to v0.2. Should it land at M1.0?

Argument for: catches store/atomic bugs the register diff misses. Argument against: meaningfully more instrumentation; Spike's commit-log doesn't emit per-store records by default — would require Spike's `--log` mode (heavier, slower).

**Default proposal:** defer to v0.2 (post-M1 Walk-α audit per `doc/wyrd-substrate-guarantees.md` §5). **Open for reviewer pushback.**

### 4.5 What happens to riscv-arch-test when Xqbp* extensions become active

Once a test program executes a `Xqbpquat` op (e.g., `qbp.qmul.w`), Spike will trap (unknown opcode) but QBP-CU will execute. This is the same "ours-only" delta as §2.5 CSRs.

Three handling options:
- (a) Skip Xqbp-active windows (instrument decode to set a `cosim_skip` flag during Xqbp ops)
- (b) Build a Spike Xqbp* plugin (large effort; defers conformance proof to second-substrate authoring)
- (c) Restrict riscv-arch-test runs to base-ISA-only test programs that don't touch Xqbp*

**Default proposal:** (c) — riscv-arch-test corpus is base-ISA tests by definition; they don't use custom opcodes; non-issue at M1. The "ours-only" delta only matters for Spike cosim, not riscv-arch-test conformance. **Open for reviewer pushback.**

---

## 5. Implementation sequence — three PRs, dependency-ordered

### PR 1 — `feat(cosim): Spike co-sim harness + Tier A smoke`

**Scope:** §2 (Spike co-sim, Tier A only)

- Add `emulator/cosim/spike.go` — subprocess launcher + commit-log parser
- Add `emulator/cosim/diff.go` — register-diff engine + `CommitDivergence` reporter
- Add `emulator/cosim/spike_test.go` — Tier A 50-insn curated smoke; `go test -short` skips
- Add `emulator/cosim/testdata/smoke/` — 50-insn ELF + reference signature
- Add `emulator/cpu.go` `CommitChan` optional output field + emission
- New CI job `cosim-tier-a` in `.github/workflows/gcg-verification.yml`
- All four GCG-ladder gating gates pass

**Scope-glob:** `emulator/cpu.go`, `emulator/cosim/**`, `.github/workflows/gcg-verification.yml`. Nothing else.

**Effort:** ~5 days. Subprocess management + commit-log parser is the bulk; smoke ELF authoring is mechanical.

### PR 2 — `feat(archtest): riscv-arch-test plugin integration`

**Scope:** §3 (riscv-arch-test integration, RV64I corpus)

- Add `emulator/cosim/archtest/plugin.go` — riscv-arch-test plugin interface implementation
- Add `emulator/cosim/archtest/exclusions.yaml` — per-test exclusion list with reasons
- Add `emulator/cosim/archtest/plugin_test.go` — runs ≥10 tests as fast-smoke
- Add `git submodule` or vendored copy of riscv-arch-test corpus at a pinned tag
- New CI job `archtest-nightly` in `.github/workflows/gcg-verification.yml` (cron-triggered)
- All four GCG-ladder gating gates pass

**Scope-glob:** `emulator/cosim/archtest/**`, `.gitmodules`, `.github/workflows/gcg-verification.yml`. Nothing else.

**Effort:** ~3 days. Plugin interface is small; bulk is integration testing across the corpus.

### PR 3 — `feat(cosim): Tier B nightly + divergence-report generator`

**Scope:** §2.4 Tier B (full Spike cosim against riscv-arch-test corpus)

- Add `emulator/cosim/tierb_test.go` — runs riscv-arch-test programs through cosim diff
- Add `reviews/cosim-divergence-template.md` — structured report format
- Add `cmd/cosim-report/main.go` — automated `gh issue create` on nightly failure
- Add cron-triggered CI job `cosim-tier-b-nightly`
- **On-demand `TIER` workflow input** (per `@bma-implementor` §I4 read on this PR): the nightly workflow accepts `workflow_dispatch` with input `tier ∈ {a, b, both}`, allowing developers to run Tier B on-demand from a PR when touching ISA-load-bearing code paths. Default is `b` (matches cron). `tier=both` runs A then B; `tier=a` is the same surface PR-gating already runs.
- All four GCG-ladder gating gates pass

**Scope-glob:** `emulator/cosim/**`, `cmd/cosim-report/**`, `reviews/cosim-divergence-template.md`, `.github/workflows/cosim-nightly.yml`. Nothing else.

**Effort:** ~4 days. Depends on PR 1 + PR 2.

### After all three land

- M1 verification claim: "QBP-CU emulator passes Spike co-sim Tier A on every PR; Tier B + riscv-arch-test RV64I conformance ≥ 95% nightly."
- Update `emulator/RISCV-Emulator-Best-Practices.md` §3.1 / §3.2 with concrete invocation paths (closes issue #18 deliverable #3).
- Add a new "Verification Audit Trail" section to `doc/wyrd-substrate-guarantees.md` describing the cosim/archtest substrate as a `v0.2.0-rc1` lens promotion.

---

## 6. §I4 review requirements

### 6.1 @qbp-architecture (primary reviewer per issue label)

Specifically asked to verify:

1. **§1.2 motivation framing** is consistent with peer-review-005 §M1 — that introducing Stream B Layer 0 ISA primitives is the right moment to introduce silicon-equivalent verification.
2. **§2.5 "ours-only" CSR handling** — does the proposed approach (log but don't diff) match the architectural intent for Stream B's `qbp.amode/bsel/psel` introduction?
3. **§3.3 RV64I-only corpus** — is M1.0 base-ISA-only the right starting scope, or should F/D extensions land in the same cohort given Optical-SU(2) glide-phase NV-centre pipeline plans?
4. **Open question §4.2 plugin-language choice** — Go-native vs Python orchestrator. Your call.
5. **Open question §4.5 Xqbp-active windows** — restricting riscv-arch-test to base-ISA only seems clean to me; flag if there's an architectural reason to author a Spike Xqbp* plugin instead.

### 6.2 @bma-implementor (impl-side review)

Specifically asked to verify:

1. **§2.8 `CommitChan` instrumentation** — does the proposed channel-out shape interact cleanly with the M1 WDEvent observer goroutine (per ADR-003 §I3)? Specifically: does the cosim consumer count as the same "observer" as the WDEvent observer, or are they two channels with two consumers?
2. **§2.7 CI gating model** — PR-gating on Tier A only is correct from your perspective? Or do you want the full corpus to PR-gate (slower CI but stronger gate)?
3. **§4.1 Spike-build-vs-binary** — given workspace machine is FX-8350 and CI is ubuntu-latest, does compiling Spike from source on every CI run match your build-cache discipline elsewhere in BMA's CI?

---

## 7. Migration path — additive only

This strategy is **purely additive**. No existing tests are modified or moved; no public APIs change shape. Specifically:

- `emulator/cosim/` is a new sibling of `emulator/`'s test files; existing imports/tests stay untouched.
- `CPU.CommitChan` is a new optional field; nil by default; existing code paths zero-cost.
- The new CI jobs run in parallel with the existing GCG ladder; no slowdown of the existing PR-gating flow.
- `riscv-arch-test` submodule pins a tag; doesn't add a maintained dependency we'd have to keep current.

If a reviewer surfaces a backwards-incompatible requirement during §I4 (unlikely for a verification-only addition), it lives in v0.2.0 per `doc/wyrd-substrate-guarantees.md` §5.

---

## 8. References

- [`architecture/adr-003-m1-wdevent-observer-invariants.md`](../../architecture/adr-003-m1-wdevent-observer-invariants.md) §I4 — design-doc-as-S-01-review-surface; §I3.4 — `mstatus.QBP` gating that this verification strategy validates
- [`architecture/peer-review-005-stream-migration.md`](../../architecture/peer-review-005-stream-migration.md) §M1 — Stream B Layer 0 introduction (the silicon-targeted state this verification covers)
- [`emulator/RISCV-Emulator-Best-Practices.md`](../../emulator/RISCV-Emulator-Best-Practices.md) §3 — verification & safety; §3.1 cosim "the gold standard"; §3.2 architectural test suites
- [`doc/wyrd-substrate-guarantees.md`](../wyrd-substrate-guarantees.md) §5 — additive-only-through-v0.1.x boundary that this strategy honours
- [`doc/design/m1-gearbox.md`](./m1-gearbox.md) — sibling §I4 design surface (ADR-004 implementation); shares the design-doc-first pattern
- [Spike (riscv-isa-sim)](https://github.com/riscv-software-src/riscv-isa-sim) — upstream golden model
- [riscv-arch-test](https://github.com/riscv-non-isa/riscv-arch-test) — official conformance suite
- [riscv-arch-test plugin spec](https://github.com/riscv-non-isa/riscv-arch-test/blob/main/doc/spec/test-plugin-spec.md) — plugin-interface contract
- Issue [#18](https://github.com/JamesPagetButler/qbp-compute-unit/issues/18) — parent
- `~/Documents/go-coding-guide.md` — workspace-wide Go conventions (mandatory for implementation PRs)

---

*Authored 2026-05-14 by `qbp-cu-implementor`. §I4 status: Proposed; awaiting reviewer signoff before implementation PRs open.*
