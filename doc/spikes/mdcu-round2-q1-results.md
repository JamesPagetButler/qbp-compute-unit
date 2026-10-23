# MDCU round 2, Q1 result (frozen pre-registration v3)

Plan: `doc/spikes/prereg/mdcu-round2-prereg-FROZEN-v3.md` (commit 98eeb1e, sha256 `28bcaa6f72db9168d1908bbc17709db109761f0898b06bc8ea87809423cae374`).
Run: 2026-10-23, 1 core, `nice -n 19`, 10,800 runs (9 cells x 200 seeds x 6 kinds), all cells completed, none skipped, wall 1,360 s.
Raw: `doc/spikes/raw/mdcu-q1-2026-10-23.jsonl` (13.2 MB, sha256 `a087bc6966045859ec49ca86df58c5b1ff598ed793ddf5057bec2905b8f5b59c`). Analysis output: `doc/spikes/raw/mdcu-q1-2026-10-23-analysis.txt`.
Analysis program `emulator/cmd/mdcu-q1-analyze` was written from the frozen text before any grid output was read. Status of every number below: **measured**.

## Verdict under the frozen rules: **INCONCLUSIVE**

| Rule | Measured | Needed | Result |
|---|---|---|---|
| C1 overlap SVD-3 >= 0.95 (N=24, sigma=0.1) | 99.5% of seeds | >= 90% | pass |
| C2 95th pct SVD-3: null 1 / null 2 / null 3 | 0.649 / 0.667 / **0.991** | < 0.80 | **fail** (null 3) |
| C3 persistence, SVD-3 >= 0.90 at N=12 / N=48 | 100% / **48.5%** | >= 80% | **fail** (N=48) |
| C4 median paired gap, overlap minus null 3 | 0.090, bootstrap 95% CI [0.077, 0.098] | >= 0.15 | **fail** |
| C5 spontaneity, Monte-Carlo KS vs Haar-G2 reference | D = 0.0079, p = 0.557 | p > 0.05 | pass |
| NO-rule: nulls 2 or 3 >= 0.95 in >= 50% of seeds | disjoint 0%, rand3 26.5% | no | not triggered |

Reading of C2: the frozen text applies the 95th-percentile rule to nulls 1 to 3. arch seq 8 #6 calls nulls 1 to 2 the sanity nulls covered by that rule. If null 3 were excluded from C2, C2 would pass, but C3 and C4 fail regardless, so the verdict is inconclusive on either reading.

## What was measured (distributions, main cell N=24, sigma=0.1)
| Kind | SVD-3 q5 / q50 / q95 | SVD-3 >= 0.95 | local ‖A‖² q50 | plane \|phi0\| q50 (diagnostic) |
|---|---|---|---|---|
| overlap | 0.987 / 1.000 / 1.000 | 99% | 0.000 | 1.000 |
| local | 0.596 / 0.647 / 0.697 | 0% | 0.723 | 0.546 |
| disjoint | 0.575 / 0.622 / 0.667 | 0% | 1.753 | 0.431 |
| none | 0.568 / 0.604 / 0.649 | 0% | 2.725 | 0.350 |
| rand3 | 0.794 / 0.909 / 0.991 | 27% | 1.626 | 0.331 |
| shuffle (control) | 0.984 / 1.000 / 1.000 | 98% | 0.000 | 1.000 |

Fano-line histogram (secondary): max 16.0% (no line above the 30% flag).

## What this does and does not show
- **Measured:** the canonical overlap coupling reliably collapses the ring onto a single 3-plane at N = 12 and 24 (about 99 to 100% of seeds), and the local, disjoint and no-coupling controls do not. The selected plane is not clustered: its spread across seeds matches the Haar-G2 reference (p = 0.56), so the choice is not inherited from the initialisation or the coupling layout under this test.
- **Measured, and the decisive null:** a random alternating vector-valued trilinear energy of the same norm also produces strong SVD-3 concentration (27% of seeds >= 0.95 at N=24, and 83 to 88% at N=12). The overlap-minus-null-3 gap (0.09) is below the pre-registered 0.15 floor. So SVD-3 concentration alone is **not** specific to the octonion associator.
- **Measured, not a decision input (arch seq 8 #5):** the planes the random null selects are not associative (median |phi0| about 0.33), while the overlap planes are (median about 1.00; phi0 fraction of consecutive triples 1.00). This is a difference between the two cases that the primary statistic does not see. Because phi0 was ruled diagnostic-only before this run, it cannot change the verdict. Promoting it would be a post-hoc analysis and would have to be labelled as such.
- **Measured:** at N = 48 even the overlap case does not reach a single plane (SVD-3 median 0.89, 48% of seeds >= 0.90), though its planes remain close to associative (median |phi0| 0.997, local ‖A‖² about 0.03). **Hypothesis, not tested:** either 250 steps are not enough at N = 48, or the ring splits into several associative planes. Neither was examined.
- Not shown: anything about hardware, speed or silicon. This round tested the dynamics statistic only. No physics claim.

## Caveats
- Seeds are Go PCG, not numpy `default_rng`. Disjoint layout uses the true plaquette gradient. Null 3 is an alternating vector-valued tensor with the associator tensor's Frobenius norm. See section 8b of the plan.
- The two early smoke runs (seeds 1000 and 9000, N=12) are disclosed in the plan. Seed 1000 is also a grid seed; its grid result is unchanged by having been seen once at N=12.
- The Monte-Carlo p-value uses 1,000 replicates, so its resolution is 0.001.
- Q2 (Albert determinant plus triality, B0 DAG scheduling) is **not built and not run**.
