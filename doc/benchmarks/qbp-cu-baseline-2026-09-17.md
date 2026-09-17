# QBP-CU Performance & Correctness Baseline — 2026-09-17

**Purpose.** A durable, reproducible baseline of the QBP-CU emulator captured *before* the
between-sprint test, so future changes can be diffed against it to see improvements or
regressions, and so we have a fixed comparison point against what the host AMD x86 chip
does natively. Pairs **correctness** (functional tests) with **performance** (benchmarks) —
a fast wrong op is worse than useless.

> **Regenerate / compare:** `scripts/qbp-cu-baseline.sh [count]` (refuses to run on a busy
> box). Statistical comparison of two captures: `benchstat old.txt new.txt`
> (`go install golang.org/x/perf/cmd/benchstat@latest`). Raw, benchstat-ready capture:
> [`raw/qbp-cu-bench-c4ef642-2026-09-17.txt`](raw/qbp-cu-bench-c4ef642-2026-09-17.txt).
> Machine-readable medians: [`qbp-cu-baseline-2026-09-17.json`](qbp-cu-baseline-2026-09-17.json).

## Environment (reproducibility metadata)

| Field | Value |
|---|---|
| git | `c4ef642` (branch `feat/qbp-cu-baseline-record`) |
| Go | go1.24.x |
| CPU | AMD FX-8350 (Piledriver, 8 cores) |
| SIMD | **AVX: yes · AVX2: NO · FMA: yes** — the Gearbox has no AVX2 fast path on this host |
| GOMAXPROCS | 2 (bounded, `nice -n 10`) |
| samples | `-count=6`, medians reported |
| load @ capture | ~2 (quiet box — required for a clean baseline) |

*Why SIMD matters:* the quaternion kernel picks its path by CPU feature. On this Piledriver
host (no AVX2) the fast path can't engage, so these numbers are the **scalar/AVX-only floor**.
A host with AVX2 would benchmark differently — always compare like-for-like SIMD.

## Correctness — all instructions PASS

`go test ./emulator/... -count=1` → **ok** (full suite green). Every ISA instruction has
functional tests (QMUL/QADD/QROT/QCONJ/QNORM/FANO, plus QW128, complex, QW8, high-prec).
This baseline is therefore **correct-and-measured**, not speed alone.

## Deterministic ISA cycle cost (host-independent)

Wall-clock ns/op is host-specific; the **abstract ISA cycle count** (`emulator/isa.go`) is
host-independent and survives the M1/silicon transition — the stable cross-hardware unit.

| Instruction | ISA cycles |
|---|---|
| QMUL, QADD, QCONJ, QNORM, FANO | 1 |
| QROT | 2 (two QMULs) |

*(Composite-op cycle model is flagged for M1/v0.2 refinement — #63 §10. Class stable, values may re-tariff.)*

## Per-instruction wall-clock (median ns/op, count=6, this host)

Two layers are measured because they answer different questions:
- **Kernel (public API `*Gearbox`)** — the raw op cost a consumer (e.g. Wyrd) pays.
- **Full CPU `Step`** — decode + execute + WDEvent emission (the ISA-execution path).

| Op | Kernel ns/op | Full CPU Step ns/op | allocs | Native x86 ns/op | Kernel ÷ native |
|---|---|---|---|---|---|
| QMUL64 | 44.3 | 541.8 | 0 | 17.2 | **2.6×** |
| QADD64 | 42.2 | 538.0 | 0 | 0.50 | 84× |
| QCONJ64 | 40.5 | 559.5 | 0 | 0.50 | 81× |
| QNORM64 | 32.6 | 541.2 | 0 | 0.50 | 65× |
| QROT64 | 50.2 | 572.8 | 0 | — | — |
| FANO | — | 537.3 | 0 | — | — |
| QMUL128 | 102.8 | 586.9 | 0 | — | — |
| QADD128 | — | 564.3 | 0 | — | — |
| QCONJ128 | — | 578.5 | 0 | — | — |
| QNORM128 | — | 620.3 | 0 | — | — |
| QROT128 | — | 841.5 | 0 | — | — |
| QMUL8 | 33.3 | — | 0 | — | — |
| CMUL64 | 2.8 | — | 0 | 0.50 | 5.6× |
| CADD64 | 2.1 | — | 0 | — | — |
| CMUL128 | 42.4 | — | 0 | — | — |
| Gearbox QMul64 (AMODE0) | 43.7 | — | 0 | 17.2 | 2.5× |

## High-precision QMul (big.Float software path) — the slow path to watch

| Width | ns/op | bytes/op | allocs/op |
|---|---|---|---|
| W256 | 4042 | 1664 | 36 |
| W512 | 4046 | 1664 | 36 |
| W1024 | 3776 | 1664 | 36 |

The fast paths are **0-alloc** (a regression to >0 allocs is a first-class red flag); the
high-prec path allocates 36×/op by design. ns/op is roughly precision-independent at these
small operands — the big.Float machinery dominates over the mantissa width.

## Key findings (baseline observations)

1. **The Gearbox kernel is ~2.5× a naive Go quaternion multiply on this host** (43.7 ns vs
   17.2 ns). On the FX-8350 (no AVX2) the SIMD abstraction does not win vs the compiler's
   scalar codegen — an optimization opportunity, and a reason the Walk-phase hardware (AVX2)
   matters. Re-measure on AVX2 hardware before concluding.
2. **The CPU `Step` path costs ~500 ns of fixed overhead** (decode + WDEvent emission) on top
   of the ~40 ns kernel — the ISA-execution wrapper dominates per-instruction cost, not the math.
   If instruction throughput becomes a goal, the `Step`/emit path is where to look first.
3. **Add / conj / norm are near-free natively (~0.5 ns)** but ~40 ns via the kernel — the
   per-call overhead (mutex `RLock` + `[4]float64`↔`QW64` conversion) dominates trivial ops.
4. **Everything is 0-alloc except the high-prec big.Float path** (36 allocs/op) — matches the
   design and the existing CI zero-alloc guard.

## What "the AMD x86 should be able to do" (native floor)

The `NativeX86_*` benchmarks are plain Go on the FX-8350, no emulator: quat-mul 17 ns,
quat-add/conj/normsq ~0.5 ns, complex-mul ~0.5 ns. These are the comparison floor — the
emulator's overhead over them is the cost of the typed quaternion ISA abstraction.

## How to use this record

- **Regression/improvement check:** re-run `scripts/qbp-cu-baseline.sh`, then
  `benchstat raw/qbp-cu-bench-c4ef642-2026-09-17.txt <new>.txt` for deltas with p-values.
- **Cross-hardware:** compare the **ISA cycle counts** (host-independent), not ns/op, when the
  host changes (e.g. Walk-phase AVX2 box, or M1 silicon). Re-pin cycle values if #63 §10 re-tariffs.
- **Correctness first:** a perf comparison is only meaningful when `go test ./emulator/...` is green.
