# Efimov Round-2 — Chained CMul64 Overhead (2026-09-27)

Chained-path companion to the [primitive baseline](qbp-cu-baseline-2026-09-17.md).
Built to @qbp-architecture's Efimov round-2 spec (live-test seq=1758): a valid
emulator-vs-native overhead factor for an O(N) dependent complex-multiply chain —
the shape a generated-code amplitude pipeline actually has, as opposed to a single op.

- **Host:** AMD FX-8350 (AVX+FMA, **NO AVX2**), 8 cores.
- **Capture:** `emulator/efimov_chain_bench_test.go`, `go test -bench=EfimovChain -benchmem -count=3`, box load ~1.5/8 (ratio-valid; both sides contend equally — not a pristine single-number capture).
- **Measurement validity:** native ns/op scales **linearly** across N (283ns → 2.7µs → 27µs → 273µs for N=100→100k). A flat native curve was the prior invalid result (constant-folding / loop-hoisting); the runtime-slice nodes + per-iteration perturbation + package-level sink defeat it. Both paths **0 B/op, 0 allocs/op**.

## Result

| N | Emulator chain (ns/op) | Native `complex128` (ns/op) | Overhead × | per-element emu / nat (ns) |
|---:|---:|---:|---:|---:|
| 100 | 1,475 | 282.9 | 5.21× | 14.75 / 2.83 |
| 1,000 | 14,741 | 2,719 | 5.42× | 14.74 / 2.72 |
| 10,000 | 147,371 | 27,045 | 5.45× | 14.74 / 2.70 |
| 100,000 | 1,475,223 | 273,214 | 5.40× | 14.75 / 2.73 |

**Headline: ~14.75 ns per emulated `CMul64` vs ~2.73 ns native → 5.40× chained overhead**, dead-flat across four orders of magnitude.

## Reading it honestly

- **This is a different number from the primitive baseline's 2.5×.** The 2.5× is the *quaternion* `QMul64` (AVX path, single op). This 5.40× is the *complex* `CMul64` in a *chain* — dominated by per-call method-abstraction cost (a Gearbox method call + `[2]float64` value passing) against a native complex multiply the compiler inlines and optimizes freely. Both are correct; they measure different things. Bank them **labeled**, not merged into one figure.
- **⚠️ NO-AVX2 CAVEAT (must travel with any banked number):** on this box the Gearbox abstraction loses to naive scalar, so overhead > 1 is a **hardware artifact, not an emulator defect**. Re-measure on AVX2 hardware before reading any factor as an architecture ceiling. Cross-hardware comparison uses host-independent **ISA cycle-counts**, not ns/op.

## Regenerate

```
cd emulator && go test -bench=EfimovChain -benchmem -count=6 -run='^$'
```
