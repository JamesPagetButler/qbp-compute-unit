# QBP-CU Benchmarks & Baselines

Durable performance + correctness baselines for the QBP-CU emulator, so changes can be
diffed against a fixed point (improvement/regression) and compared to the host x86 native floor.

## Layout
- `qbp-cu-baseline-<date>.md` — human-readable record (env, per-instruction ns/op + cycles +
  native-baseline + overhead, findings, how-to-use). **Correctness is paired with perf.**
- `qbp-cu-baseline-<date>.json` — machine-readable medians (`schema: qbp-cu-baseline/v1`) for diffing.
- `raw/qbp-cu-bench-<sha>-<date>.txt` — raw `go test -bench` capture, **benchstat-ready**.

## Latest baseline
- **2026-09-17** — [`qbp-cu-baseline-2026-09-17.md`](qbp-cu-baseline-2026-09-17.md) · git `c4ef642` ·
  AMD FX-8350 (AVX+FMA, no AVX2). All instructions PASS; every instruction benchmarked.
- **2026-09-27** — [`qbp-cu-efimov-chain-2026-09-27.md`](qbp-cu-efimov-chain-2026-09-27.md) ·
  chained-path companion (Efimov round-2 spec): O(N) `CMul64` chain vs native `complex128` →
  **5.40× overhead** (~14.75 ns/op emulated vs ~2.73 ns native), stable N=100..100k. No-AVX2 caveat applies.

## Regenerate / compare
```
make baseline                 # env-stamped capture → raw/ (refuses on a busy box)
# or: scripts/qbp-cu-baseline.sh [count]     (count default 6)
go install golang.org/x/perf/cmd/benchstat@latest
benchstat raw/<old>.txt raw/<new>.txt        # deltas with p-values
```

## Comparison discipline
- **Same-host, over-time** (regression/improvement) → compare ns/op via benchstat.
- **Cross-hardware** (Walk AVX2 box, M1 silicon) → compare the **ISA cycle counts**
  (host-independent), not ns/op; record which SIMD path is live (AVX2 vs scalar) or numbers are apples-to-oranges.
- A perf comparison only counts when `go test ./emulator/...` is green (fast wrong op = worse than useless).
