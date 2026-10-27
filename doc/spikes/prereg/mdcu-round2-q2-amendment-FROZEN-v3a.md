# MDCU round 2 — Q2 amendment v3a (FROZEN 2026-10-24)

Parent: `prereg/mdcu-round2-prereg-FROZEN-v3.md` (sha256 28bcaa6f72db9168d1908bbc17709db109761f0898b06bc8ea87809423cae374), section 4.
Supersedes: `prereg/mdcu-round2-q2-addendum-DRAFT.md` (commit 0c37571), with qbp-architecture mdcu-spike seq 19 points 1–4 folded in.
Gates: qbp-architecture approval (seq 19, conditional on this freeze), deming box-safety GO (seq 20, conditional on this freeze and the run go), beekeeper run go (2026-10-24 16:34 local).
**Status at freeze: Q2 NOT run.** No cycle bound, host ns or ratio has been seen.

## 1. Workload and variants (unchanged from the draft)

det(X) = abc − a|x|² − b|y|² − c|z|² + 2·Re((xy)z). Code: `emulator/mdcu/q2.go`, tests `emulator/mdcu/q2_test.go`, driver `emulator/cmd/mdcu-q2/main.go`.

| Variant | Side | Re((xy)z) path | MACs/det (fixed by construction) |
|---|---|---|---|
| B1lit | B1 | two OMUL (CD-MAC) | 157 |
| B1lean | B1 | one OMUL + 8-term signed dot (sensitivity) | 101 |
| B0a | B0 | sparse 3-form, x_i·y_j then ·z_k | 157 |
| B0b | B0 | factored, seven Fano-line 3×3 determinants | 118 |

Op counts and the DAG come from the same tape (qbp-cu-implementor seq 4); a test asserts tape = analytic count.

## 2. Measurement (arch seq 19 points 2 and 3)

- **Cost model (assumed, not silicon), identical parameters on both sides:** 64 multiplier lanes per tile, the same adder tree, one batch of ≤ 64 multiplies per tile-cycle, the same zero-multiply combine cost (one cycle), and the same issue model (greedy list schedule). Disclosed: tile-batch granularity slightly favours B1; bound (i) is the counterweight.
- **Bounds, both sides:** (i) ideal = ceil(total multiplies / (64·P)); (ii) dependency-aware = greedy list schedule of the DAG on P tiles.
- **Decisive cell:** batch of 64 independent determinants at P = 8.
- **Secondary, non-deciding sweep:** batch ∈ {1, 512} at P ∈ {1, 8} (batch 64 at P = 1 is also printed). It exists so a reader can see whether the verdict depends on perfect lane fill. All of P ∈ {1, 2, 4, 8, 16} is in the JSON.
- **Host ns:** `testing.Benchmark`, GOMAXPROCS = 1, 6 samples per variant, median, per determinant, under `nice -n 19`. Load average recorded at start and end. If load at start is > 4, wait and re-check before running.
- **Wall cap:** 10 min. Expected well under 1 min.

## 3. Decision rule (applied mechanically by the driver)

- **B1 (deciding)** = B1lit, as frozen §2.4 names the literal CD-MAC path (arch seq 19 point 1).
- **B0** = the fewer-multiply of {B0a, B0b} (that is B0b, by construction).
- **Agreement gate:** worst normwise disagreement ≤ 1e-12 over 20,000 instances, checked first; fail ⇒ the comparison does not count.
- **Admission:** `go test ./mdcu/ -run 'TestRT|TestQ2'` must pass immediately before the run.
- **yes** iff B1's model cycles ≤ 0.5 × B0's at the decisive cell under **both** bounds (and gates pass).
- **no** iff B0 ≤ B1 on every column: MACs, ideal, dep, host ns.
- **else inconclusive.**
- **Sensitivity (B1lean):** the per-column ordering (B0 ≤ B1 or not) is computed for B1lean vs B0 on the **measured columns** (ideal cycles, dep cycles, host ns). If any measured column's ordering differs from B1lit's, the verdict of record is **"inconclusive (B1-variant-sensitive)"**.
  - *Clarification, disclosed:* the draft said "any column". The MAC column flips between B1lit (157 > 118) and B1lean (101 < 118) **by construction**, which was known and disclosed before this freeze. Arch seq 19 point 1 rules the MAC column is arithmetic, not evidence. So the sensitivity rule applies to the measured columns only. Under the literal draft reading the verdict would be "inconclusive (B1-variant-sensitive)" regardless of the run; the report will state both readings.

## 4. Report contents (arch seq 19 points 1 and 4)

- The MAC column is fixed by construction (157 / 101 / 157 / 118) and is not evidence. Q2's measured content is the two cycle bounds and host ns.
- **Finding:** contracting the 64-term 3-form over x, y and then dotting with z *is* the CD-MAC product plus an 8-term dot. The "non-CD tensor contraction" and the CD-MAC are the same bilinear map, so that path is not a non-CD baseline.
- The frozen prediction (B0 matches or beats B1 on MACs and host ns) is variant-dependent on MACs; reported as such.
- Model-cycle results are "advantage under a symmetrically-applied ISA cost model, not silicon".

## 5. Round-3 candidates logged (not run)

1. Best-of-each-side Q2 (B1lean vs best-of-B0) — arch seq 19.
2. N = 48 convergence for Q1 — arch seq 17.
3. φ₀-primary Q1 on fresh seeds — arch seq 17.
4. φ-only (G₂ 3-form) null vs full associator energy — qbp-oppenheimer seq 21. Separates "alternating structure suffices" from "G₂ structure suffices".

All on fresh seeds, excluding 1000, 9000 and the whole Q1 grid.
