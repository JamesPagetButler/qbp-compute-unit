# MDCU spike, round 2: check-in and pre-registration (DRAFT)

**Status: DRAFT v2 (amended per mdcu-spike seq 2, 3, 4). Posted for review, NOT frozen. Nothing in this plan has been run.**
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
- **Q1 = yes** iff, in the overlapping/N=24/σ=0.1 cell, the SVD-3 share is ≥ 0.95 in ≥ 90% of seeds **and** the 95th percentile of the same statistic under nulls 1–3 is < 0.80, **and** the effect persists (≥ 0.90 in ≥ 80% of seeds) at N = 12 and 48, **and** (v2, arch #1) the median SVD-3 gap (overlap minus null 3) is ≥ 0.15 with a bootstrap 95% CI excluding 0, **and** (v2, arch #2) the selection is spontaneous: no single Fano frame is selected in > 50% of seeds.
- **Q1 = no** iff nulls 2 or 3 reach ≥ 0.95 in ≥ 50% of seeds.
- Otherwise **inconclusive**.
- **Frame-selection definition (v2, proposed, for arch to confirm):** per seed, take the 3-D top-SVD subspace of `Z[:,1:]` and assign the Fano line (of the 7) with the largest squared projection of that subspace onto its three axes. Report the 7-bin histogram across seeds. ("Up to G₂" means the 7 lines are interchangeable by automorphism, so the histogram, not the label, is what matters.)
- **Run order (v2, arch #3, pre-registered so the cap truncates the least important part):** (1) main cell overlap/N=24/σ=0.1 plus all 4 nulls; (2) N=12 and N=48; (3) the remaining σ and coupling cells. If deming's cap bites, report the completed cells and what was skipped.
**Known concern, flagged for reviewers:** minimising an energy built from φ₀ and finding φ₀-calibrated planes may be close to tautological. The φ₀-calibration statistic is therefore *secondary*; the primary claim is only robustness and null-separation of the locking, not that dynamics "chooses" octonions.

## 4. Q2 design
**Candidate workload:** the Albert-algebra h₃(𝕆) determinant and triality gate (studybuddy's pick; RT4 is the prototype). It passes the RT2 test by construction: the determinant needs Re(xyz).
**Admission rule (standing gate, point 4):** every "uses octonions" claim must (a) fail RT2, i.e. change under a generic non-automorphism SO(7), **and** (b) pass RT3, i.e. associator-G₂ sensitivity.
**Baselines (equally informed):**
- **B0** the fastest *non-CD* exact-structure path: associator / determinant via a fixed 3-form/4-form tensor contraction (no CD multiplier). Its operation count will be **computed in code**, not assumed.
- **B1** the CD-MAC path.
- Not used: the 34-OMUL loop, and any exact-invariant method vs an unrelated approximation (SPINN confound).
**Prediction (pre-registered, falsifiable):** B0 matches or beats B1 in MACs and in host ns/op, so **Q2 = no for the MDCU on host speed**. Any MDCU advantage, if it exists, will be in the *model-cycle* column (tile model) at equal P and I expect it, if present, to be ≤ 2×. I record this so the result is a genuine test; it is not a hoped-for outcome.
**Agreement gate (v2, arch #4):** B0 and B1 must agree normwise (≤ 1e-12 relative) on the same inputs, or the comparison does not count.
**B0 tiling rule (v2, arch #5 and qbp-cu-implementor seq 4; proposed, for review):** B0's multiplies are scheduled as independent scalar-MAC streams on the same P tiles with the same per-tile multiplier budget m that `sched.go` gives a CD-MAC tile (m read from the code at freeze, not hand-assigned). B0's model cycles come from the same code path that computes its op count. A model-cycle "yes" is reported only as *advantage under a symmetrically-applied ISA cost model (sched.go), not silicon*.
**Q2 decision rule:** yes iff B1's model cycles are ≤ 0.5× B0's at P=8 *and* the workload passes the admission rule *and* the agreement gate passes; no iff B0 ≤ B1 on every column; else inconclusive. Branch: **B** (spike/napkin grade), per inter#197.

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

---
## Bridge-ready summary (≤ one message; not posted)
@qbp-architecture, Antigravity seat check-in for MDCU round 2 (re seq 3087, 3080). Hygiene done (pyc dropped, RT4 bracketing note, RT5 measures float associator at ℍ; QAssoc ≤ ℍ is an architectural zero, not evidence). Draft plan at `doc/spikes/prereg/mdcu-round2-checkin-DRAFT.md` (not frozen, nothing run): **Q1** Test B grid N∈{12,24,48} × {local, overlap, disjoint} × σ∈{0,0.1,0.3} × 200 seeds, nulls = no-coupling / disjoint / random-3-form / shuffled-init, distributions not point values, thresholds pre-stated; **Q2** Albert determinant + triality, baseline = non-CD tensor contraction (computed in code), prediction recorded *against* the MDCU on host speed. Concerns for you: φ₀-calibration may be tautological (kept secondary); thresholds need your amendment before freeze. Resource ask: 1 core, ≲ 2 h cap for Q1. Awaiting your review → deming box-safety → beekeeper go. I will not run until all three.
