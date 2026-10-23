# MDCU spike, round 2: pre-registration (FROZEN v3)

**Status: FROZEN v3 (2026-10-23). Signed off by qbp-architecture (mdcu-spike seq 8) once the six answers are folded in; box-safety GO from @deming (live-test seq 3190); beekeeper run go relayed (seq 7). Frozen sha256 is posted to mdcu-spike, since a file cannot contain its own hash. Any later change is a dated amendment below.** Nothing in this plan has been run as a grid at freeze time (smoke tests only; see disclosure).
Author: Antigravity seat · 2026-10-23 · branch `spike/mdcu-xqbp-v2` (record tag `spike-record/mdcu-xqbp-v2-aa45861`, which I will not touch)
Responds to: qbp-architecture seq 3087 (10 refinements) and qbp-cu-implementor seq 3080.
Freeze rule: this file is frozen only when qbp-architecture signs off; `sha256sum` is recorded then and every later change is a dated amendment.
Gate order (per 3087): qbp-architecture reviews the plan → @deming box-safety go → beekeeper run go.

> **Disclosure (so thresholds are not mistaken for blind):** a single seed of each Pillar-2 test has already been seen (Test B: local-only SVD-3 share 0.61, locked 0.998; Test A/C canonical numbers). **Harness smoke (2026-10-23, after draft v2):** one run of every kind at N=12, σ=0.1 was viewed for seed 1000 (a grid seed; first smoke, since moved to out-of-grid seed 9000) and for seed 9000. Observed (2 seeds, N=12 only, NOT evidence): the random alternating-tensor null (rand3) also reaches SVD-3 ≈ 0.975–1.000, while its φ₀ measures stay low (φ₀-fraction 0.00, plane |φ₀| 0.19–0.41, versus overlap 1.00 and 1.00). If that holds across seeds, SVD-3 does not separate octonion structure from a generic alternating-trilinear energy, and the φ₀-based statistics, currently secondary, would carry the null separation. **I have not changed the decision rule on this basis; that is flagged to qbp-architecture as a question, because switching the primary statistic after seeing smoke output is a researcher degree of freedom.** The null distributions across seeds and every Q2 quantity remain unseen.

## 1. Questions (restated; Pillar 1 is closed as a symmetry reduction)
- **Q1.** Does Pillar-2 frame locking (Test B) survive an *independent, controlled* repro, with varied N, overlap, seeds and noise, against explicit nulls?
- **Q2.** Is there a workload where the **associator path is load-bearing**, and does the MDCU beat the best *equally-informed* baseline on it?

Answers will be yes / no / inconclusive. "Inconclusive" is allowed and not a failure.

## 2. What the current evidence is, and is not
| Item | Status |
|---|---|
| A5 Tests A, C | Go matches Gemini's *verbatim* loops digit-for-digit. This shows the **port is faithful**. It is not an independent repro and says nothing about whether locking or sieving is physical. |
| A5 Test B | Ported from **prose**, matched to canonical numbers on one seed. |
| Pillar 1 / A1 / A2b | Symmetry reduction (RT1, RT2). Not MDCU evidence. |
| QAssoc at tier ≤ ℍ | Returns an exact zero **by construction**. RT5 now measures the float associator on quaternions (2.3 eps, nonzero in 19,833/20,000 samples). |
| ZDCHK | 42-plane assessor **class** match, not a general ∃y:xy=0 detector. |

## 3. Q1 design (Test B)
**Fixed:** Euclidean metric; N-site ring of unit imaginary octonions; lr 0.04, μ 1.5, 250 steps (as canonical).
**Varied (full grid):** N ∈ {12, 24, 48}; coupling ∈ {local-only, overlapping, **disjoint-plaquette** (stride-3 triples)}; noise on initial Z σ ∈ {0.0, 0.1, 0.3} added to the canonical draw; 200 seeds each (seeds 1000–1199, `numpy.random.default_rng`).
**Nulls (all run with identical optimiser):**
1. *No coupling* (μ term only).
2. *Disjoint-plaquette* coupling (above).
3. *Random 3-form null:* replace the associator energy with one built from a random 3-form of equal Frobenius norm (the calibration null proposed by studybuddy). In 7D almost every 3-form is a G₂ structure in a skewed metric, so this null is run with the Euclidean metric fixed.
4. *Initial-condition control:* shuffle sites after initialisation to remove any ordering bias.
**Statistics, reported as full distributions with quantiles (5/25/50/75/95):** top-3 SVD share of `Z[:,1:]`; mean ‖[Z_{i-1},Z_i,Z_{i+1}]‖²; fraction of consecutive triples whose 3-plane has φ₀ ≥ 1−δ (δ = 0.05) under the *standard* φ₀; frame-indexed and 7-frame-averaged associator measures.
**Pre-registered decision rule (proposed; qbp-architecture to amend before freeze):**
- **Q1 = yes** iff, in the overlapping/N=24/σ=0.1 cell, the SVD-3 share is ≥ 0.95 in ≥ 90% of seeds **and** the 95th percentile of the same statistic under nulls 1–3 is < 0.80, **and** the effect persists (≥ 0.90 in ≥ 80% of seeds) at N = 12 and 48, **and** (arch seq 8 #6) the median SVD-3 gap (overlap minus **null 3 only**) is ≥ 0.15 with a bootstrap 95% CI excluding 0 (nulls 1–2 are sanity nulls, covered by the 95th-percentile rule), **and** (arch seq 8 #1) the selection is spontaneous per the Grassmannian rule below.
- **Q1 = no** iff nulls 2 or 3 reach ≥ 0.95 in ≥ 50% of seeds.
- Otherwise **inconclusive**.
- **Spontaneity (PRIMARY, arch seq 8 #1; definition fixed here):** for the 200 seeds of the main cell, take each seed's top-3 SVD plane (stored basis U_s, 7×3). Pairwise distance d(s,t)² = 3 − ‖U_sᵀU_t‖_F² (= Σ sin² of the principal angles). Reference: 200 associative 3-planes from Haar-random G₂ rotations of a Fano plane, sampled exactly as span(x, y, xy) with x uniform on S⁶ and y uniform on S⁵ ⊥ x (G₂ acts transitively on such pairs, so this is the G₂-invariant measure). Statistic: two-sample KS distance D between the observed and reference pairwise-distance distributions. p-value is **Monte-Carlo**, because pairwise distances are not independent: draw 1000 pairs of independent 200-plane reference sets, compute D for each pair, and p = fraction with D ≥ the observed D. **Spontaneous iff p > 0.05; inherited (clustered) iff p ≤ 0.05.** Seed sets from the PCG generator, base seed 1000 for the reference (distinct stream from the grid).
- **Fano-line histogram (SECONDARY, arch seq 8 #2):** per seed, the Fano line (of the 7) with the largest squared projection of the top-3 plane onto its three axes; the 7-bin histogram across seeds; flag basis-aligned bias if any line holds > 30% of seeds (about 2× uniform). Diagnostic only.
- **Run order (arch #3, pre-registered so the cap truncates the least important part):** (1) main cell overlap/N=24/σ=0.1 plus all nulls; (2) N=12 and N=48 at σ=0.1; (3) the remaining σ and coupling cells. If deming's cap bites, report the completed cells and what was skipped.
**φ₀ statistics (arch seq 8 #5): SECONDARY / DIAGNOSTIC, never a decision input.** They are recorded for every run so a later reader can see them, but no Q1 verdict depends on them. **Known concern:** minimising an energy built from φ₀ and finding φ₀-calibrated planes may be close to tautological; the primary claim is only robustness and null-separation of the locking, not that dynamics "chooses" octonions. See the disclosure at the top: an early two-seed smoke suggested the random null also locks under SVD-3; under this plan that outcome would be reported as it falls (Q1 = no or inconclusive), not rescued by re-weighting statistics.

## 4. Q2 design
**Candidate workload:** the Albert-algebra h₃(𝕆) determinant and triality gate (studybuddy's pick; RT4 is the prototype). It passes the RT2 test by construction: the determinant needs Re(xyz).
**Admission rule (standing gate, point 4):** every "uses octonions" claim must (a) fail RT2, i.e. change under a generic non-automorphism SO(7), **and** (b) pass RT3, i.e. associator-G₂ sensitivity.
**Baselines (equally informed):**
- **B0** the fastest *non-CD* exact-structure path: associator / determinant via a fixed 3-form/4-form tensor contraction (no CD multiplier). Its operation count will be **computed in code**, not assumed.
- **B1** the CD-MAC path.
- Not used: the 34-OMUL loop, and any exact-invariant method vs an unrelated approximation (SPINN confound).
**Prediction (pre-registered, falsifiable):** B0 matches or beats B1 in MACs and in host ns/op, so **Q2 = no for the MDCU on host speed**. Any MDCU advantage, if it exists, will be in the *model-cycle* column (tile model) at equal P and I expect it, if present, to be ≤ 2×. I record this so the result is a genuine test; it is not a hoped-for outcome.
**Agreement gate (v2, arch #4):** B0 and B1 must agree normwise (≤ 1e-12 relative) on the same inputs, or the comparison does not count.
**B0 scheduling rule (arch seq 8 #3, replaces the earlier "independent streams" proposal, which flattered B0):** both B0 and B1 are scheduled under the **same scheduling assumptions**. Build B0's dependency DAG (including its reduction dependencies) and schedule it through the same scheduler logic as B1. If `sched.go` cannot take an arbitrary DAG, report **two bounds for both sides**: (i) an ideal-throughput bound (no stalls) and (ii) a dependency-aware estimate. **Q2 is decided only if B1 wins under both bounds.** B0's model cycles come from the same code path that computes its op count (qbp-cu-implementor seq 4). A model-cycle "yes" is reported only as *advantage under a symmetrically-applied ISA cost model (sched.go), not silicon*.
**Q2 decision rule:** yes iff B1's model cycles are ≤ 0.5× B0's at P=8 **under both bounds** *and* the workload passes the admission rule *and* the agreement gate passes; no iff B0 ≤ B1 on every column; else inconclusive. Branch: **B** (spike/napkin grade), per inter#197. **Q2 implementation status at freeze: not yet built;** it will be built and run as the second step after Q1, under the same gates.

## 5. Controls and honesty rules (points 4–8)
- RT2/RT3 as a reusable gate helper, run on every claim.
- 7-frame average or explicit frame index everywhere (L2 already does this).
- Floating point: normwise errors only; "bit-exact" only for same-op cases (L1, L2, L7); QAssoc ≤ ℍ is reported as *architectural zero* and RT5 measures the float value.
- Timing: MACs, model cycles and host ns in separate columns; `ns/op` with `-count=6` median and spread; quiet window only, FX-8350 without AVX2, run-bounded; benchmarks never start until deming's go.
- Hashing: this plan, seeds, parameter grid and the decision rules are hashed at freeze.

## 6. Resource estimate (assumed, scaled from measured)
Measured: the whole Pillar-2 A+B+C suite runs in 0.38 s. **Harness measurement (2026-10-23):** one seed of all 6 kinds at N=12 takes 0.33 s (rand3 0.18 s, overlap 0.05 s, others <= 0.03 s). The grid is 9 cells x 200 seeds x 6 kinds = 10,800 runs (the three layouts and the nulls share one init per (N, σ, seed); null 2 is the disjoint layout). **Assumed** cost grows about linearly in N, so N=24 ≈ 0.7 s and N=48 ≈ 1.3 s per seed-of-6-kinds, giving roughly 3 σ x 200 seeds x (0.33+0.7+1.3 s) ≈ **23 min single-threaded** (assumed from the N=12 measurement; not measured at N=24/48). Run bounded to **1 core**, ≤ 100 MB RAM, ≤ 50 MB JSONL output, no network, no GPU. Q2 benchmarks: ≲ 5 minutes. The 2 h wall-clock cap proposed earlier stays as a conservative backstop, to be set by deming.

## 7. Capability-gap list
No width or vector-tier dispatch in `CPU.Exec` (functional only); ZDCHK is a class check, not a detector (rename to `ZDCLASSCHK` is optional and pending); leaf engine only applies where the potential has the plane-rotation symmetry; Test B was ported from prose; no hardware or silicon numbers (model cycles only); FX-8350 has no AVX2; QW256 is a fixed-width reference, not arbitrary precision; the physics claims are out of scope.

## 8. Hygiene status
`.pyc` removed and ignored; `albert_triality_test.go` committed (RT4 with the bracketing note); RT5 added; width-ladder label corrected. The `aa45861` tag is untouched.

## 8b. Q1 harness (built, tested, NOT run as a grid)
Code: `emulator/mdcu/q1grid.go`, `q1grid_test.go`, driver `emulator/cmd/mdcu-q1/main.go` (`-dry-run` only so far). Checks passing: `RunRingGen` is **bit-identical** to canonical `RunRing` for local and overlap on the canonical seed; `Q1Measure.SVD3` equals canonical `ringMetrics` (0.6097 / 0.9980); the random null is exactly alternating in (a,b,c) and has the same Frobenius norm as the true associator tensor; the true associator is antisymmetric. Driver kill conditions: wall cap, memory cap, free-disk floor, output cap, NaN.
Deviations from the text above, for the reviewer:
1. **Seeds** use Go `math/rand/v2` PCG (seeds 1000–1199), not numpy `default_rng`, so they are not the canonical numpy draws.
2. **Null 3** is a random *alternating vector-valued trilinear map* with the associator tensor's Frobenius norm, not a scalar 3-form: the associator is vector-valued, so a scalar form cannot enter ‖A‖².
3. **Disjoint** uses the true gradient of the plaquette energy; local/overlap keep the canonical (non-symmetrised) gradient so the main cell is the canonical dynamics.
4. **"Frame"** is axis-aligned (7 Fano lines), but a locked plane may be any associative 3-plane (a G₂/SO(4) family), so the axis-aligned frame is coarse. The harness also stores the top-3 plane basis and `PlanePhi0` so a frame-independent spread analysis (pairwise principal angles across seeds) is possible. Question for qbp-architecture.
5. `shuffle` permutes sites after an i.i.d. init, so it is distributionally the same as `overlap`; it is a realisation control only.


## 9. Amendment log
| Date | Source | Change |
|---|---|---|
| 2026-10-23 | arch seq 2 #1 | Q1 effect-size floor (gap ≥ 0.15, bootstrap CI excludes 0) |
| 2026-10-23 | arch seq 2 #2 | Spontaneity test: frame histogram, no frame > 50% of seeds |
| 2026-10-23 | arch seq 2 #3 | Pre-registered run order |
| 2026-10-23 | arch seq 2 #4 | Q2 agreement gate ≤ 1e-12 relative |
| 2026-10-23 | arch #5, impl seq 4 | B0 tiling rule pre-registered; same code path for ops and cycles; "symmetrically-applied cost model" wording |
| 2026-10-23 | arch #9 | `ZDCLASSCHK` rename recommended by arch and impl; the unit's name is the beekeeper's call, not applied |
| 2026-10-23 | arch seq 8 #1 | Primary spontaneity = pairwise Grassmannian distance vs Haar-G2 Fano-plane reference, Monte-Carlo KS p > 0.05 |
| 2026-10-23 | arch seq 8 #2 | Fano histogram secondary, threshold no line > 30% |
| 2026-10-23 | arch seq 8 #3 | B0 scheduled under the same assumptions as B1; two bounds; Q2 decided only if B1 wins under both |
| 2026-10-23 | arch seq 8 #4 | Q2 threshold stays 0.5x at P=8 |
| 2026-10-23 | arch seq 8 #5 | phi0 statistics secondary/diagnostic, never a decision input |
| 2026-10-23 | arch seq 8 #6 | Effect-size floor applies to null 3 only |
| 2026-10-23 | deming live-test 3190 | Kill-condition 5 (load > 4 self-pause) dropped while the GPU spike runs concurrently |

## 10. Final kill conditions (deming-ratified)
1. Wall-clock cap: Q1 2 h, Q2 10 min; abort and report completed and skipped cells.
2. RSS above 500 MB. 3. Free disk below 15 GB, or output above 200 MB. 4. No progress heartbeat for 10 min. 5. A request from the federation or deming's box-watch for the box: pause or abort. (The load > 4 self-pause is dropped; deming's own trigger is load > 12 or sustained D-state.) 6. Any panic or NaN: stop and report.
Run as: 1 core, `nice -n 19`, a prebuilt binary, output under `doc/spikes/raw/`.

## 11. Other tests the seat has in mind (not folded; each would be its own pre-registered round)
| Test | Question | Measures | Kill condition |
|---|---|---|---|
| A6 benchstat vs `doc/benchmarks` | New: do the emulator benchmarks regress against the recorded baseline? | ns/op medians over `-count=6` with spread | Not started until the box is idle (load < 1); 10 min cap |
| Leaf-engine profile | New: is sqrt/renormalisation the leaf engine's bottleneck? | pprof CPU profile of `EngLeaf` at the recorded quench | 10 min cap; no hardware claim |
| Spike 2 (arch #8) | Branch A (substrate hosting) head to head with Branch B under this protocol | Same statistics and gates | Not designed yet; after round 2 |
Beyond Q1 and Q2 nothing else is planned inside this frozen round.


## Superseded: the original bridge-ready summary (kept for the record; the live messages are on mdcu-spike)
@qbp-architecture, Antigravity seat check-in for MDCU round 2 (re seq 3087, 3080). Hygiene done (pyc dropped, RT4 bracketing note, RT5 measures float associator at ℍ; QAssoc ≤ ℍ is an architectural zero, not evidence). Draft plan at `doc/spikes/prereg/mdcu-round2-checkin-DRAFT.md` (not frozen, nothing run): **Q1** Test B grid N∈{12,24,48} × {local, overlap, disjoint} × σ∈{0,0.1,0.3} × 200 seeds, nulls = no-coupling / disjoint / random-3-form / shuffled-init, distributions not point values, thresholds pre-stated; **Q2** Albert determinant + triality, baseline = non-CD tensor contraction (computed in code), prediction recorded *against* the MDCU on host speed. Concerns for you: φ₀-calibration may be tautological (kept secondary); thresholds need your amendment before freeze. Resource ask: 1 core, ≲ 2 h cap for Q1. Awaiting your review → deming box-safety → beekeeper go. I will not run until all three.
