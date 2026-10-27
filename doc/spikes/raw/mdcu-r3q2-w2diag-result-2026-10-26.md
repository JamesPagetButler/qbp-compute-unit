# R3-Q2 W2 post-hoc diagnostic — result

**POST-HOC DIAGNOSTIC. No R3-Q2 verdict change.** Requested by arch (seq 89, "explain the disagreement"), go given before it was built.

- Kernel: `emulator/mdcu/r3q2_w2unrolled.go`, **generated** from `crossTab`/`minorIdx` by `TestGenW2Unrolled`, straight-line, no table lookups, same 21 minors + 7 dot = 49 multiplies. Code commit `4e2be58`. Tests (written first, run red on a stub, then green): agrees with `W2NonCD` and `W2CD` to 1e-12 on 1000 out-of-grid inputs (seed 9830); tape = 49 multiplies; committed file equals generator output.
- Frozen `W2CD` / `W2NonCD` and the frozen benchmark file are untouched. The new benchmark is `BenchmarkPostHocW2NonCDUnrolled`, which the frozen `BenchmarkR3Q2` selector cannot pick up.
- Raw: `doc/spikes/raw/mdcu-r3q2-w2diag-raw-2026-10-26.txt`, sha256 `69750127007c273d283964971c153c3026b6032038f3559e5053f2c6e4da3292`, commit `22876b1`, posted before reading (seq 98). Frozen W2 pair and the unrolled kernel in the same process; same flags and inputs as R3-Q2. Load start 0.90, end 1.00 (inside the ceiling), 22:31:53–22:32:01, FX-8350, go1.24.2, GOMAXPROCS=1 nice 19 taskset core 7.
- Reading rule, fixed and posted before any data (seq 93): if the frozen yes-condition (CD median ≤ 0.5 × non-CD median, p < 0.05) no longer holds against the unrolled kernel, the gap is **explained** (implementation shape); otherwise it **survives**.

## Result

| path | n | median ns |
|---|---|---|
| W2 CD (OMulCDMAC + dot7) | 10 | 8434 |
| W2 non-CD, table-driven (frozen) | 10 | 23081 |
| W2 non-CD, unrolled (post-hoc) | 10 | 2304 |

| comparison | ratio of medians | p (Mann–Whitney) |
|---|---|---|
| CD / table-driven non-CD (R3-Q2 as frozen) | 0.365 | 1.1e-05 |
| CD / unrolled non-CD | 3.661 | 0.00018 |
| unrolled / table-driven non-CD | 0.100 | 0.00018 |

**Reading (rule fixed beforehand): gap explained.** Removing the table lookups makes the non-CD W2 about 10× faster than the frozen version, and against it CD is 3.7× *slower*, not 2.7× faster. The R3-Q2 W2 "host-ns tiebreak YES" was an implementation-shape artefact of the frozen non-CD W2 path (table lookups), not an architectural advantage.

## What this does and does not change

- R3-Q2's recorded verdict stays **YES (host-ns tiebreak)**, applied exactly as frozen. It is now recorded as explained by implementation shape, per the planning table's Q2 row ("nothing is built on it until the disagreement is explained").
- The architecture-level statement is unchanged: **no workload has a ≥ 2× CD multiply advantage** (closure NO). No CD speed claim.
- Not tested, only noted: `W3NonCD` also uses a table (`assocTab`, `minorIdx`) and `B0cross` uses `crossTab`, so the frozen non-CD W1/W3 timings may be handicapped by the same effect. If so, their CD/non-CD ratios (1.705, 1.038) overstate CD's relative speed, which would only reinforce "no CD advantage". I did not build unrolled versions of those; arch's go covered W2 only.
- Host ns measured on an FX-8350 without AVX2. The CD-MAC's wide-lane parallelism is serialised, so host ns is biased against CD (frozen disclosure, still applies).
