# MDCU round 2 — Q2 results (2026-10-24)

Plan: `prereg/mdcu-round2-prereg-FROZEN-v3.md` §4 + `prereg/mdcu-round2-q2-amendment-FROZEN-v3a.md`
(sha256 38989a7f721098343b5abbeab6aae370448162af753c9e36176a0d9140d2d4e9, commit 68e51ba).
Raw: `raw/mdcu-q2-2026-10-24.json`. Driver: `emulator/cmd/mdcu-q2` (verdict applied mechanically).
Branch B, spike/napkin grade. Model cycles are an advantage under a symmetrically-applied ISA cost
model, **not silicon**. No physics claims.

## Verdict of record: **INCONCLUSIVE (B1-variant-sensitive)**

- Primary (B1lit vs B0b): **no** — B0 ≤ B1 on every column.
- Sensitivity (B1lean vs B0b): the ideal-bound ordering flips (B1lean 13 < B0b 15), so by the frozen
  rule the verdict of record is inconclusive (B1-variant-sensitive).
- Under the literal draft reading ("any column", including MACs) the verdict is the same.
- The **yes** rule (B1 ≤ 0.5× B0 under both bounds) was not met by **either** B1 variant. The best
  B1/B0 ratio on any measured column is 13/15 = 0.87 (B1lean, ideal bound). Descriptive, not a change
  to the verdict.

## Run

| Item | Value |
|---|---|
| Gates | arch seq 19 + freeze (seq 23); deming GO seq 20; beekeeper go 2026-10-24 16:34 |
| Admission | RT1–RT5, TestQ2_* pass immediately before run (G₂ torus Δdet 4.6e-14; generic SO(7) Δdet ≈ 103) |
| Agreement gate | worst normwise 5.81e-16 (limit 1e-12) |
| Load avg | start 3.58 (rule: ≤ 4 at start), end 4.37 |
| Wall | 34.3 s (cap 10 min); GOMAXPROCS=1, nice 19 |

## Columns (never merged)

**MACs per determinant — fixed by construction, not evidence:** B1lit 157, B1lean 101, B0a 157, B0b 118.

**Host ns per determinant (measured, median of 6):**

| B1lit | B1lean | B0a | B0b |
|---|---|---|---|
| 276.0 | 167.6 | 237.7 | **94.4** |

**Model cycles, ideal / dep (assumed tile model; \* = decisive cell):**

| variant | b1 P1 | b1 P8 | b64 P1 | **b64 P8\*** | b512 P1 | b512 P8 |
|---|---|---|---|---|---|---|
| B1lit | 3/5 | 1/3 | 157/320 | **20/41** | 1256/2560 | 157/321 |
| B1lean | 2/5 | 1/3 | 101/320 | **13/41** | 808/2560 | 101/321 |
| B0a | 3/5 | 1/3 | 157/320 | **20/41** | 1256/2560 | 157/321 |
| B0b | 2/5 | 1/3 | 118/320 | **15/41** | 944/2560 | 118/321 |

Critical path: 3 batches for every variant.

## Reading (what the numbers do and do not say)

1. **Host speed:** the factored non-CD path B0b is the fastest on this host, 2.9× faster than B1lit
   and 1.8× faster than B1lean. The frozen prediction "B0 matches or beats B1 on host ns" holds
   against both B1 variants. On MACs it holds against B1lit only (disclosed before the run).
2. **Ideal bound** just tracks multiply counts (ceil(mults/64P)), so it inherits the
   by-construction MAC ordering. It adds no information beyond the MAC column.
3. **Dependency bound is non-discriminating in this model.** Every variant records 5 tape nodes per
   determinant (two common batches, two Re-term batches, one combine), and the model charges one
   tile-cycle per node regardless of fill. With P=8 the batch-64 cell is node-count bound
   (320 nodes / 8 + 1 = 41) for all four variants. The tie resolves as B0 ≤ B1 under the frozen
   rule. This is a **limitation of the agreed cost model** (tile-batch granularity, disclosed in the
   amendment), not a property of the algebra. A model that packs small nodes from different
   instances into one tile would separate the variants; that is a design change for a later round,
   not a reinterpretation of this one.
4. **Lane-fill sweep:** the orderings at batch 1 and 512 match the decisive cell (single instance:
   all tie at P8; batch 512 repeats the batch-64 pattern), so the verdict is not an artifact of
   64 = lane width.
5. **Finding (arch seq 19 point 4):** contracting the 64-term 3-form over x, y then dotting with z is
   the CD-MAC product plus an 8-term dot; the "non-CD contraction" and the CD-MAC are the same
   bilinear map. The only non-CD path that beats CD-MAC (B0b) does so by exploiting the Fano-line
   factorisation (scalar/vector split plus seven 3×3 determinants), i.e. structure, not a different
   multiplier.

## Bottom line for round 2

- Q1: INCONCLUSIVE — locking is not shown to be octonion-specific.
- Q2: INCONCLUSIVE (B1-variant-sensitive) — no MDCU 2× model-cycle advantage under either B1
  variant; on host speed a factored non-CD path wins.

Neither question shows an MDCU advantage under the pre-registered rules.

## Round-3 candidates (logged, not run)

Best-of-each-side Q2 (arch seq 19); a cost model with cross-instance node packing (this report, item 3;
needs arch review before it can be pre-registered); N=48 convergence and φ₀-primary Q1 (arch seq 17);
φ-only vs full-associator null (qbp-oppenheimer seq 21). Fresh seeds, excluding 1000, 9000 and the Q1 grid.
