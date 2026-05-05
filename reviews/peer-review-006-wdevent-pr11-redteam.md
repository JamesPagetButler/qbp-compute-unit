# Peer Review 006: Red Team Audit of PR #11 (WDEvent + QW128)

**Date:** 2026-05-05
**Reviewer:** Claude Opus 4.7 (red team instance)
**Code under review:** commit `bd5746c` on branch `feat/issue-8-wdevent`
**PR:** [qbp-compute-unit#11](https://github.com/JamesPagetButler/qbp-compute-unit/pull/11)
**Compared against:**
- Issue [#8](https://github.com/JamesPagetButler/qbp-compute-unit/issues/8) acceptance criteria (M0.2 WDEvent emission)
- [`archive_build/docs/QBP-CU-SiFive-Interface-Spec-v0.1.md`](../archive_build/docs/QBP-CU-SiFive-Interface-Spec-v0.1.md) §7 — original WDEvent + Accelerator interface
- [`archive_build/docs/RV-Fano-Implementation-Refinements.md`](../archive_build/docs/RV-Fano-Implementation-Refinements.md) §2.3 — extension fields (ZDClass, ZDIndices)

**Audience:** Gemini, working against this review to bring PR #11 to honest M0.2 close.

**Verdict:** ⚠️ **Functional but under-tested.** Wiring is correct; spec field set is faithful to §7 + §2.3 extensions; three perf budgets breached; **test coverage is materially below what acceptance criteria claim to verify.**

---

## 1. Executive Summary

PR #11 delivers two pieces of work in one commit:

- **WDEvent passive emission** (issue #8 / M0.2) — struct definition, channel wiring, emission hooks at the ISA boundary
- **QW128 fast paths** — AVX-FMA double-double Hamilton product, scalar fallback, dispatcher integration

The WDEvent piece passes one happy-path test and is mergeable in spirit, but the acceptance criteria on issue #8 cannot honestly be checked off as written. **Three of the six AC items are unverified, wrong-test, or breached.**

The QW128 piece has been verified separately in peer-review-002 / peer-review-004 lineage; this review focuses on the WDEvent contribution and on perf interactions between the two.

---

## 2. Test Execution — What Is Actually Verified

```
$ cd ~/Documents/QBP-Compute-Unit/emulator
$ go test -v -run TestWatchdog ./...
=== RUN   TestWatchdog_PassiveEmission
--- PASS: TestWatchdog_PassiveEmission (0.00s)
PASS
```

**One test. One op. One width.** `wdevent_test.go:7`:

- `Funct7QMUL` only (1 of 6 implemented funct7 opcodes)
- `funct3 = 3` → W64 only (W128 emission path entirely untested)
- Single instruction, single channel read

That is the entire WDEvent verification surface in this PR.

---

## 3. Acceptance Criteria Audit (Issue #8 §"Acceptance criteria")

| # | AC item | Status | Evidence |
|---|---|---|---|
| 1 | `WDEvent` struct present, matching SiFive interface spec | ✅ | `wdevent.go:21–34` — all spec §7 fields plus the §2.3 ZDClass/ZDIndices extension |
| 2 | All five QW64 ops emit one event per call | ⚠️ **Unverified** | Only QMUL tested. QADD/QROT/QCONJ/QNORM emission paths untested |
| 3 | All five QW128 ops emit one event per call (incl. stubs) | ❌ **Unverified** | Test sets `funct3=3` (W64). Zero coverage of W128 emission path |
| 4 | `TestWDEvent_EmissionCount` verifies count for a known sequence | ❌ **Wrong test exists** | Actual test is `TestWatchdog_PassiveEmission` and tests one op. No count-of-N-ops test |
| 5 | Zero allocations on hot path | ✅ | All benchmarks report `0 B/op, 0 allocs/op` |
| 6 | Performance regression < 5% vs baseline | ⚠️ **Marginally breached** | See §4 |

**3 of 6 AC items cannot be claimed as written.** Honest status of #8: `WDEvent` struct + 1-op smoke test, not "passive emission verified across all ops."

---

## 4. Performance Regression Analysis

Benchmarks in `isa_bench_test.go` call `cpu.Step(word)` which executes the full **decode → execute → emit** pipeline, so the cost of `emitWDEvent` is included in the numbers below.

```
$ go test -bench=. -benchmem ./...
```

| Op | Baseline (pre-WDEvent) | This PR | Delta | < 5% AC? |
|---|---|---|---|---|
| QMUL    | 523.2 ns/op | 547.0 ns/op | **+4.6%** | ✅ |
| QADD    | 542.7 ns/op | 571.8 ns/op | **+5.4%** | ⚠ marginal |
| QROT    | 540.6 ns/op | 568.6 ns/op | **+5.2%** | ⚠ marginal |
| QCONJ   | 532.4 ns/op | 577.8 ns/op | **+8.5%** | ❌ **breach** |
| QNORM   | 515.9 ns/op | 542.8 ns/op | **+5.2%** | ⚠ marginal |
| QMUL128 | 570.8 ns/op | 618.3 ns/op | **+8.3%** | ❌ **breach** |
| QADD128 | 536.2 ns/op | 570.0 ns/op | **+6.3%** | ❌ **breach** |
| QROT128 | 814.7 ns/op | 846.4 ns/op | **+3.9%** | ✅ |
| QCONJ128| 518.4 ns/op | 549.2 ns/op | **+5.9%** | ⚠ marginal |
| QNORM128| 561.0 ns/op | 567.0 ns/op | **+1.1%** | ✅ |

**Three breaches** (QCONJ, QMUL128, QADD128). The cost is consistent with a non-blocking channel send (≈25–50 ns).

### 4.1 Important nuance: most events are dropped

`emitWDEvent` (`cpu.go:92`) uses non-blocking send with a `default:` drop. The channel is buffered to 1024. Benchmarks run ~2M iterations.

After iteration 1024, the channel saturates and **every subsequent emission hits the drop path**. So the perf numbers above measure mostly the **drop fast-path cost**, not the cost of an actively-consumed channel.

**When M1 wires an active consumer, the cost will go up, not down.** The current benchmarks are not predictive of M1 perf.

---

## 5. Spec Comparison

### 5.1 QBP-CU-SiFive-Interface-Spec-v0.1 §7 — original WDEvent

| Field | Spec | Implementation (`wdevent.go`) | Note |
|---|---|---|---|
| `Cycle` | `uint64` | `uint64` | ✅ |
| `Op` | `Opcode` (typed) | `uint8` "Funct7 opcode equivalent" | ⚠️ **Type-safety regression** — bare uint8 vs typed enum |
| `Port` | `Port` | `Port` | ✅ |
| `FanoIndex` | `uint8` 0..6 | `uint8` (populated for FANO op only) | ✅ |
| `SignBit` | `bool` | `bool` (populated for FANO op only) | ✅ |
| `Associator[3]` | `[3]int8` residue (a*b)*c − a*(b*c) | `[3]int8` (always `[0,0,0]`) | OK for M0; needs population at M1/M2 |
| `NormDelta` | `int32` norm preservation residue, fixed-point | `int32` (always `0`) | Currently silent. Load-bearing for CTH watchdog Architecture v1.0 §3 trigger. |
| `AlgebraID` | `uint8` (0=H, 1=O, 2=Branch A, 3=Branch B) | `uint8` (always `0` via struct zero-init; never assigned) | **M1 land mine** — see §6.4 |

### 5.2 RV-Fano-Implementation-Refinements §2.3 — extension fields

| Field | Spec | Implementation | Note |
|---|---|---|---|
| `ZDClass` | `uint8` (0=NotZD, 1=CrossCopySymbolic, 2=GeneralFullMultiply) | `ZDClass` typed enum with named constants | ✅ **Better than spec** — adds compile-time safety |
| `ZDIndices[4]` | `[4]uint8` (i,j,k,l for symbolic; zeros for general) | `[4]uint8` (always zero in M0 — no ZDCHK yet) | ✅ Matches spec; population deferred to M2 |

---

## 6. Findings (severity-tagged)

### 6.1 [HIGH] Test coverage is single-op, single-width

The test exercises one Funct7QMUL emission at W64. Six implemented opcodes (QMUL/QADD/QROT/QCONJ/QNORM/FANO) × two relevant widths (W64, W128) = **12 emission paths. Eleven untested.**

The acceptance criterion *"All five QW128 ops emit one event per call (including stub trampolines)"* cannot be claimed without exercising the W128 path.

**Specific concern: stub trampolines.** `qrot128AVX` and `qnorm128AVX` are renamed to `qrot128Stub` / `qnorm128Stub` (jump to scalar). The emission lives in `Step()` after the kernel switch, so it should fire regardless of which kernel ran — but this should be **proven by test**, not assumed.

### 6.2 [HIGH] No sequence/count test

AC: *"`TestWDEvent_EmissionCount` verifies count matches op count for a known sequence."*

Test as delivered is `TestWatchdog_PassiveEmission` with `select { case evt := <-cpu.WatchdogChan: …; default: t.Fatalf("…channel was empty") }`. That is a **boolean "was an event emitted"** test, not an emission-count test.

The AC asks specifically: run N ops, drain channel, assert exactly N events were received. Missing.

### 6.3 [HIGH] Performance regression breaches 5% AC on three ops

QCONJ (+8.5%), QMUL128 (+8.3%), QADD128 (+6.3%) all exceed the AC threshold.

The channel-send overhead is the likely cause. Possible mitigations to evaluate:

- **Lock-free ring buffer** indexed by atomic counter, no channel select — predictable cost, no drop fast-path branch
- **Sentinel-event mode** in M0: emit only every Nth op until M1 active consumer arrives
- **Conditional emit**: gate emission on `if c.WatchdogChan != nil` so it can be turned off in benchmarks for the baseline measurement (this would also let issue #8 honestly distinguish "kernel cost" from "kernel + emission cost")

### 6.4 [MEDIUM] AlgebraID land mine

`AlgebraID` is never assigned in `isa.go:148–153`; relies on struct zero-init = 0 (= ℍ). Three problems:

1. **M1 will introduce `AMODE` CSR.** If the emission code is not updated to read from CSR, every event in 𝕆 / 𝕊 mode will silently report `AlgebraID=0`. Watchdog observers will see all sedenion ops mislabeled as quaternion ops.
2. **No TODO comment** in `isa.go:148` marks this as M1 work. Easy to miss in code review.
3. **Field is dead weight** in cosim contracts as long as it's always zero.

**Fix at M0:** add `// TODO(M1): populate from c.csr.AMODE` at `isa.go:148`.
**Fix at M1:** read CSR.

### 6.5 [MEDIUM] Channel buffer drop is invisible

`emitWDEvent` (`cpu.go:92–100`) drops on full channel without counter, log, or metric. After iteration 1024 of a 2M-iteration benchmark, every subsequent event is silently dropped.

For passive M0 this is by design, but:
- The benchmark numbers in §4 mostly measure the drop path
- A drop-counter would let M1 detect channel-saturation conditions and tune buffer sizing

**Suggested:** add `c.WatchdogDropCount uint64` (atomic increment) so drops are observable. ~15 min change.

### 6.6 [MEDIUM] `Op` typed-enum regression vs spec

Spec §7 field `Op Opcode` is a typed enum. Implementation uses `uint8` with a comment.

Defining `type Opcode uint8` and changing the field would be a one-line change with no behavioral impact. Cosim contracts comparing WDEvent multisets across simulator/RTL benefit from compile-time op-type checking.

### 6.7 [MEDIUM] Emission tied to successful execution only

`isa.go:144` returns an error for unimplemented funct7 *before* the emission code (lines 147–160). So:

- Successful op → 1 emission
- Unimplemented op → 0 emissions, error returned

This may be intentional (don't emit for instructions that didn't really execute), but the spec doesn't specify. The cosim contract ("multiset equality of watchdog events per cycle") could be tripped up if RTL emits-on-failure and emulator doesn't (or vice versa).

**Suggested:** explicit policy in spec OR add a `WDEvent.FaultCode uint16` populated for failed ops to keep per-cycle multiset symmetric.

### 6.8 [LOW] AVX kernels don't emit — fine, but undocumented

Emission lives at the `Step()` ISA boundary, not inside AVX kernels. This is the right design (kernels are pure functions, ISA execution is observable), but it's implicit. **The spec should formalize that the emission tap is at the ISA boundary** so future kernel authors don't add their own emission paths.

Add a package-level doc comment in `wdevent.go` stating this invariant.

### 6.9 [LOW] Test uses raw bit-shift instruction encoding

`wdevent_test.go:12` builds an instruction word with `uint32(11 | (1 << 7) | (3 << 12) | …)`. There's a `buildInst()` helper in `isa_bench_test.go:15`. Using it would make the test more maintainable and consistent with the rest of the test corpus.

### 6.10 [LOW] QMUL128 benchmark doesn't initialize Q128 operands

`isa_bench_test.go:74` sets up the benchmark without writing to `cpu.Q128[2]` or `cpu.Q128[3]`. So QMUL128 multiplies zero × zero. Times correctly but doesn't stress the actual ddMul path. Minor.

---

## 7. Recommended Changes Before Merge

| Priority | Change | Effort | Closes |
|---|---|---|---|
| 1 | Add `TestWDEvent_EmissionCount`: run N ops (mix of QMUL/QADD/QROT/QCONJ/QNORM/FANO), drain channel, assert N events | 30 min | AC #4 |
| 2 | Add per-op tests for QADD, QROT, QCONJ, QNORM, FANO at W64 | 1 hour | AC #2 |
| 3 | Add per-op tests for all 5 ops at W128 (verify stub trampolines emit too) | 1 hour | AC #3 |
| 4 | Investigate QCONJ / QMUL128 / QADD128 perf regression — try lock-free ring or atomic-indexed slot; document if 5% target needs revision | 2–4 hours | AC #6 |
| 5 | Add `// TODO(M1): populate from c.csr.AMODE` comment at `isa.go:148` | 1 min | §6.4 |
| 6 | Add `WatchdogDropCount uint64` atomic field for passive-mode observability | 15 min | §6.5 |
| 7 | Define `type Opcode uint8` and use it for `WDEvent.Op` | 5 min | §6.6 |
| 8 | Document at `wdevent.go` package level: emission tap is at ISA boundary, not in kernels | 5 min | §6.8 |

**Items 1–3** are AC blockers for an honest M0.2 close.
**Item 4** is an AC threshold breach that needs justification or fix.
**Items 5–8** are quality / future-proofing.

Total effort to satisfy AC honestly: **~4 hours**.

---

## 8. What This PR Does Well

- Struct definition is faithful to spec §7 + §2.3 extensions
- `ZDClass` typed enum is cleaner than the spec's bare uint8
- Channel-based emission with non-blocking send is the right pattern for passive M0
- Benchmark integration via `Step()` means perf regressions surface in CI
- Zero-allocation discipline maintained on hot path (0 B/op, 0 allocs/op across all 10 benchmarks)
- The QW128 piece bundled with this PR has solid test coverage from the dispatcher equivalence corpus
- Stub-honest renaming (`qrot128Stub` / `qnorm128Stub`) prevents the stub-pretending-to-be-AVX class of bug

---

## 9. Bottom Line

**Mergeable with conditions.** The wiring is correct and the spec is honored where implemented. The work to satisfy AC honestly is small (~4 hours for items 1–3, plus item 4 investigation).

Two paths to AC close:

- **(a) Recommended:** complete §7 items 1–4. AC #2/#3/#4/#6 then become honest checkboxes.
- **(b) Acceptable:** rewrite issue #8 AC to reflect what's delivered (single-op smoke test + +5–8% perf cost) and close at the lower bar. Acknowledges the M0 scope is "wiring + minimal verification" rather than "full coverage."

Path (a) is preferred because the missing tests will be needed at M1 anyway when active consumers come online — better to have them landed now.

---

## 10. Notes for Gemini

This review is structured for actionable response. The §7 table gives priority-ordered changes with effort estimates and AC mappings. Items 1–3 are mechanical test additions (no design judgment required); items 4 and 6 may benefit from coordination with the architecture instance on the trade-off space (lock-free ring vs channel; observability metric naming).

For each item addressed:
- A test fix in items 1–3 should land as a new commit on `feat/issue-8-wdevent` (don't amend; the existing commit is already pushed and a fresh commit with the test additions is cleaner for review).
- The perf investigation in item 4 may justify a separate small commit per mitigation approach so the perf delta from each is measurable in CI.
- Items 5, 7, 8 are tiny and can be batched into a single "M0 cleanup" commit.

If item 4 surfaces a fundamental trade-off (e.g., active M1 consumer perf is incompatible with the 5% target), that is a finding worth documenting in `architecture/` rather than forcing into a fix in this PR. The architecture instance will absorb the design conversation.

---

## 11. References

- [`archive_build/docs/QBP-CU-SiFive-Interface-Spec-v0.1.md`](../archive_build/docs/QBP-CU-SiFive-Interface-Spec-v0.1.md) §7 — original WDEvent + Accelerator interface
- [`archive_build/docs/RV-Fano-Implementation-Refinements.md`](../archive_build/docs/RV-Fano-Implementation-Refinements.md) §2.3 — ZDClass / ZDIndices extension
- [Issue #8](https://github.com/JamesPagetButler/qbp-compute-unit/issues/8) — M0.2 acceptance criteria
- [PR #11](https://github.com/JamesPagetButler/qbp-compute-unit/pull/11) — code under review
- [`architecture/peer-review-005-stream-migration.md`](../architecture/peer-review-005-stream-migration.md) §4 (M0.4) — passive emission rationale
- [`architecture/peer-review-002-fano-mesh-isa-redteam.md`](../architecture/peer-review-002-fano-mesh-isa-redteam.md) — RISC-V conventions context
- `emulator/wdevent.go`, `emulator/wdevent_test.go`, `emulator/cpu.go`, `emulator/isa.go`, `emulator/isa_bench_test.go` — under review

---

*Status: RECORDED | Audience: Gemini | Cadence: respond per §7 priority order | First review in `reviews/` folder*
