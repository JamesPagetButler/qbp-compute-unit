# R3-Q2 result (frozen R3 prereg §3, sha256 `070aeab8…1848`)

Run of record: the **rerun**, `mdcu-r3q2-hostns-rerun-2026-10-26.txt`, sha256 `72046c10f73f5dc6e2b325d0b56034a65864044f6c8cf0c06e7ee7d13a4c782d`, commit `36b2c38`.
Load: start 0.64, end 1.17 (ceiling: start ≤ 1.0, end ≤ 1.5; both met). Pause file absent. FX-8350, go1.24.2, GOMAXPROCS=1, nice 19, taskset core 7, 10 × 200 ms.

First run (kept in the record, not the run of record): `mdcu-r3q2-hostns-2026-10-26.txt`, sha256 `9550eb4ab198697cb7981a264983a4b4c9a52640dc07899606c11ab438a70d67`, commit `bce1358`. Load start 1.17 > 1.0, so the §3.3 start condition was not met; the analyzer labelled it "inconclusive (load)". My first bridge post (seq 85) said no rerun was needed; that was wrong, and was corrected in seq 86.

## Multiply-count closure (tape-counted, deterministic)

| workload | CD mults | best non-CD mults | non-CD/CD | ≥ 2× for CD? |
|---|---|---|---|---|
| W1 Albert determinant | 101 | 104 | 1.030 | no |
| W2 φ scalar | 71 | 49 | 0.690 | no |
| W3 associator vector | 256 | 126 | 0.492 | no |

Closure verdict: **no**.

## Host-ns (rerun, Mann–Whitney U, α = 0.05)

| workload | CD median ns | non-CD median ns | CD/non-CD | p |
|---|---|---|---|---|
| W1 | 10159 | 5958 | 1.705 | 0.00018 |
| W2 | 8412 | 23023 | 0.365 | 1.1e-05 |
| W3 | 35256 | 33951 | 1.038 | 1.1e-05 |

First run, same direction: W1 1.688, W2 0.366, W3 1.040.

Benchstat verdict (§3.3: *yes* iff on at least one workload CD median ≤ 0.5 × non-CD median with p < 0.05): **yes**, on W2 only (0.365).

## Verdict (§3.4)

Closure *no*, benchstat *yes*: they disagree, so the benchstat decides, labelled:

**R3-Q2: YES (host-ns tiebreak).**

> Host ns measured on an FX-8350 without AVX2. The CD-MAC's wide-lane parallelism is serialised, so host ns is biased against CD.

Applied exactly as frozen; no rule was changed after data.

## Reading it (not a rule change)

- The "yes" rests on **W2 alone**. W1 is 1.7× *slower* for CD; W3 is a statistical tie at 1.04×.
- W2 is the workload where CD does **more** multiplies (71 vs 49) yet runs 2.7× faster. Arithmetic cost cannot explain that.
- Likely cause (from reading `emulator/mdcu/r3q2.go` L100–147, not profiled): `W2NonCD` goes through table lookups (`minorIdx[j][k]` 2-D index, `crossTab` with int→float sign conversion) on every call, while the CD-MAC is straight-line code. That would make W2 a comparison of **implementation shape**, not of architecture. This is a hypothesis; I have not tested it.
- So the tiebreak verdict is correct under the frozen rule, but it should not be read as "CD is more efficient". The multiply closure (no workload with ≥ 2× CD advantage) is the architecture-level statement, and it is unchanged.

## Possible post-hoc diagnostic (not run; needs a go-ahead)

An unrolled, table-free W2 non-CD kernel (same 49 multiplies, straight-line), benchmarked the same way and labelled post-hoc, to test whether the W2 gap is table overhead. It would not change the R3-Q2 verdict.
