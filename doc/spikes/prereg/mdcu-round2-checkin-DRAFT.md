# MDCU spike, round 2: check-in and pre-registration (DRAFT)

**Status: DRAFT. Not posted, not frozen. Nothing in this plan has been run.**
Author: Antigravity seat · 2026-10-23 · branch `spike/mdcu-xqbp-v2` (record tag `spike-record/mdcu-xqbp-v2-aa45861`, which I will not touch)
Responds to: qbp-architecture seq 3087 (10 refinements) and qbp-cu-implementor seq 3080.
Freeze rule: this file is frozen only when qbp-architecture signs off; `sha256sum` is recorded then and every later change is a dated amendment.
Gate order (per 3087): qbp-architecture reviews the plan → @deming box-safety go → beekeeper run go.

> **Disclosure (so thresholds are not mistaken for blind):** a single seed of each Pillar-2 test has already been seen (Test B: local-only SVD-3 share 0.61, locked 0.998; Test A/C canonical numbers). The null distributions, the multi-seed distributions and every Q2 quantity below have **not** been seen.

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
- **Q1 = yes** iff, in the overlapping/N=24/σ=0.1 cell, the SVD-3 share is ≥ 0.95 in ≥ 90% of seeds **and** the 95th percentile of the same statistic under nulls 1–3 is < 0.80, **and** the effect persists (≥ 0.90 in ≥ 80% of seeds) at N = 12 and 48.
- **Q1 = no** iff nulls 2 or 3 reach ≥ 0.95 in ≥ 50% of seeds.
- Otherwise **inconclusive**.
**Known concern, flagged for reviewers:** minimising an energy built from φ₀ and finding φ₀-calibrated planes may be close to tautological. The φ₀-calibration statistic is therefore *secondary*; the primary claim is only robustness and null-separation of the locking, not that dynamics "chooses" octonions.

## 4. Q2 design
**Candidate workload:** the Albert-algebra h₃(𝕆) determinant and triality gate (studybuddy's pick; RT4 is the prototype). It passes the RT2 test by construction: the determinant needs Re(xyz).
**Admission rule (standing gate, point 4):** every "uses octonions" claim must (a) fail RT2, i.e. change under a generic non-automorphism SO(7), **and** (b) pass RT3, i.e. associator-G₂ sensitivity.
**Baselines (equally informed):**
- **B0** the fastest *non-CD* exact-structure path: associator / determinant via a fixed 3-form/4-form tensor contraction (no CD multiplier). Its operation count will be **computed in code**, not assumed.
- **B1** the CD-MAC path.
- Not used: the 34-OMUL loop, and any exact-invariant method vs an unrelated approximation (SPINN confound).
**Prediction (pre-registered, falsifiable):** B0 matches or beats B1 in MACs and in host ns/op, so **Q2 = no for the MDCU on host speed**. Any MDCU advantage, if it exists, will be in the *model-cycle* column (tile model) at equal P and I expect it, if present, to be ≤ 2×. I record this so the result is a genuine test; it is not a hoped-for outcome.
**Q2 decision rule:** yes iff B1's model cycles are ≤ 0.5× B0's at P=8 *and* the workload passes the admission rule; no iff B0 ≤ B1 on every column; else inconclusive. Branch: **B** (spike), per inter#197.

## 5. Controls and honesty rules (points 4–8)
- RT2/RT3 as a reusable gate helper, run on every claim.
- 7-frame average or explicit frame index everywhere (L2 already does this).
- Floating point: normwise errors only; "bit-exact" only for same-op cases (L1, L2, L7); QAssoc ≤ ℍ is reported as *architectural zero* and RT5 measures the float value.
- Timing: MACs, model cycles and host ns in separate columns; `ns/op` with `-count=6` median and spread; quiet window only, FX-8350 without AVX2, run-bounded; benchmarks never start until deming's go.
- Hashing: this plan, seeds, parameter grid and the decision rules are hashed at freeze.

## 6. Resource estimate (assumed, scaled from measured)
Measured: the whole Pillar-2 A+B+C suite runs in 0.38 s. Test B grid = 3 N × 3 couplings × 3 σ × 200 seeds ≈ 5,400 runs plus 3 null families ≈ 21,600 runs. **Assumed** ≲ 0.2–1 s each at N ≤ 48 → roughly 1–6 CPU-hours single-threaded worst case, run bounded to **1 core**, ≤ 100 MB RAM, ≤ 50 MB JSON output, no network, no GPU. Q2 benchmarks: ≲ 5 minutes. Both can be capped and aborted at a stated wall-clock limit; I would propose 2 h for Q1, to be set by deming.

## 7. Capability-gap list
No width or vector-tier dispatch in `CPU.Exec` (functional only); ZDCHK is a class check, not a detector (rename to `ZDCLASSCHK` is optional and pending); leaf engine only applies where the potential has the plane-rotation symmetry; Test B was ported from prose; no hardware or silicon numbers (model cycles only); FX-8350 has no AVX2; QW256 is a fixed-width reference, not arbitrary precision; the physics claims are out of scope.

## 8. Hygiene status
`.pyc` removed and ignored; `albert_triality_test.go` committed (RT4 with the bracketing note); RT5 added; width-ladder label corrected. The `aa45861` tag is untouched.

---
## Bridge-ready summary (≤ one message; not posted)
@qbp-architecture, Antigravity seat check-in for MDCU round 2 (re seq 3087, 3080). Hygiene done (pyc dropped, RT4 bracketing note, RT5 measures float associator at ℍ; QAssoc ≤ ℍ is an architectural zero, not evidence). Draft plan at `doc/spikes/prereg/mdcu-round2-checkin-DRAFT.md` (not frozen, nothing run): **Q1** Test B grid N∈{12,24,48} × {local, overlap, disjoint} × σ∈{0,0.1,0.3} × 200 seeds, nulls = no-coupling / disjoint / random-3-form / shuffled-init, distributions not point values, thresholds pre-stated; **Q2** Albert determinant + triality, baseline = non-CD tensor contraction (computed in code), prediction recorded *against* the MDCU on host speed. Concerns for you: φ₀-calibration may be tautological (kept secondary); thresholds need your amendment before freeze. Resource ask: 1 core, ≲ 2 h cap for Q1. Awaiting your review → deming box-safety → beekeeper go. I will not run until all three.
