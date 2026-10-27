# mdcu-r3 results — R3-Q1 (ring dynamics)

**Status: R3-Q1 complete and analysed under the frozen rules. R3-Q2 not yet run (waiting on a quiet box).**
Date: 2026-10-26. Author: antigravity (runner). Analysis is mechanical output of the frozen rules; interpretation is labelled as such.

## 0. Provenance

| Item | Value |
|---|---|
| Frozen pre-registration | `doc/spikes/prereg/mdcu-round3-prereg-FROZEN-v1.md`, commit `8c24f47`, sha256 `070aeab8a54c6b749ad69faf5514c89eba48a1fbc3218de418b614c69d7c1848` (re-verified on disk before the run) |
| Harness | commit `8d6b310` (clean tree, equal to origin at run time) |
| Run go | beekeeper, via qbp-architecture mdcu-spike seq 59; deming box-safety clearance live-test seq 3479 |
| Raw output | `doc/spikes/raw/mdcu-r3-2026-10-26.jsonl`, sha256 `633670f9b4134bf5add1a82a7c04aaffbf9e43b8e73a23a7ee45b67add0f76bc`, 5,102 lines, committed `143bd69` and posted on mdcu-spike (seq 65) **before** any analysis |
| Analysis output | `doc/spikes/raw/mdcu-r3-analysis-2026-10-26.md` (verbatim analyzer stdout) |
| Run | seeds 3000–3199, 4,900 runs, wall 605 s, 1 core, GOMAXPROCS=1, nice 19, taskset; clean end record; pause file never present |

**Deviations and disclosures (none changes a frozen rule)**
1. The comass-proxy diagnostic (arch seq 57) was **not** built before the run; the run proceeded without it, as arch seq 59 ruled for that case. It is a function of the seed's sampler stream only and is to be computed post-hoc and labelled as such.
2. **Sampler-cost correction** (posted seq 52, accepted seq 57). Frozen §1.2 said the definite rate is about 0.2% (about 500 draws per seed, cap-miss e^-40). That came from small Python samples and was wrong. Measured on the grid: **S1 definite draws per seed min 3, median 1,100, mean 1,588, max 9,496, 0 misses** over 100 seeds, consistent with a definite rate of about 0.06%. The frozen miss rule was not triggered.
3. Five interpretation choices the frozen text does not specify (accepted, arch seq 57): Q1b uses a stratum's paired seeds with the φ precondition on those seeds; an S1 miss on an even seed would drop only K-psi-S1; closure aggregates "any workload"; a benchstat-inconclusive verdict follows the closure; spontaneity per arm with at least 20 locked runs. None of them was exercised by the data except the first.
4. Run on a box at loadavg about 3.2–3.7 (desktop load). The load ceiling in the frozen text applies to R3-Q2 only.

## 1. Verdicts (frozen §1.5, mechanical)

| Rule | Result |
|---|---|
| Replication gate (K-full 3-lock ≥ 90%) | **pass**, 99.0% (198/200) |
| **Q1a** (K-gram vs K-full, 200 seeds) | **no.** K-gram 3-locks 3.0% (< 50%); Δ median −0.000, 95% CI [−0.000, −0.000] |
| **Q1b definite** (K-phi vs K-psi, 100 seeds) | **inconclusive (φ-only does not lock).** K-phi 0.0%, K-psi 83.0% |
| **Q1b split** (100 seeds) | **inconclusive (φ-only does not lock).** K-phi 0.0%, K-psi 75.0% |
| **Reading table** | **Row 5: inconclusive, or a mixed result reported as-is.** |

Reading, in the pre-registered words: the Gram term alone does not reproduce locking (Q1a = no), but Q1b cannot be decided because the φ-only arm never locks at 250 steps, so the question "does G₂ matter, or does any 3-form do?" is **not answered by R3**. R3 supports neither a G₂-specific reading nor its negation.

Note on Q1a: Δ is about 0 even though the rates differ by 96 points, because K-gram's median S3 is 1.000. K-gram collapses onto a **2-plane** (75.5% 2-collapse, median λ₃ share 0.000), which has S3 ≈ 1 but is not a 3-lock. This is exactly the round-2 limitation that the λ₃-share gate was added to close, and it worked as designed.

## 2. Main cell (N = 24, σ = 0.1, 250 steps)

| arm | n | 3-lock | 2-collapse | diffuse | median S3 | median λ₃ share |
|---|---|---|---|---|---|---|
| K-full | 200 | 99.0% | 0.0% | 1.0% | 1.000 | 0.278 |
| K-gram | 200 | 3.0% | 75.5% | 21.5% | 1.000 | 0.000 |
| K-phi | 200 | 0.0% | 0.0% | 100.0% | 0.642 | 0.179 |
| K-psi | 200 | 79.0% | 0.0% | 21.0% | 1.000 | 0.200 |
| K-psi-S1 (definite only) | 100 | 67.0% | 0.0% | 33.0% | 1.000 | 0.180 |
| K-psigram | 200 | 74.0% | 0.0% | 26.0% | 0.999 | 0.309 |
| K-rand3 | 200 | 33.5% | 0.0% | 66.5% | 0.914 | 0.211 |
| K-none | 200 | 0.0% | 0.0% | 100.0% | 0.601 | 0.168 |

Per stratum, K-psi locks 83% (definite, S2) and 75% (split, S1); K-psigram 72% and 76%. K-phi is 0% in both. Full per-stratum tables are in the analysis file.

## 3. Sampler record (reported with the verdict, arch seq 42)

- **Definite stratum (100 seeds):** S1 draws min 3 · median 1,100 · mean 1,588 · max 9,496 · misses 0. Histogram (draws ≥ bin): [2+]=1 [8+]=1 [32+]=1 [64+]=4 [128+]=9 [256+]=13 [512+]=18 [1024+]=28 [2048+]=17 [4096+]=7 [8192+]=1.
- **Split stratum (100 seeds):** min 1 · median 1 · max 2 · misses 0.
- **S2 (definite, primary ψ):** s median 0.157; distance d from φ median 0.383, range [0.132, 0.673]; metric condition number median 5.6.
- Both strata have 100 accepted seeds, above the 50-seed power floor.

## 4. Secondaries (diagnostic only; never a decision input)

**Dose-response** (oppenheimer's requested reading; definite seeds, n = 100). Spearman ρ between the paired gap S3(K-phi) − S3(K-psi) and the S2 distance d: **ρ = −0.258** (nominal p about 0.01). The gap is negative throughout (ψ locks more than φ), and it grows in magnitude as ψ is deformed further from φ. This is the direction a coupling-strength / comass confound would predict, and the direction a "G₂-ness of ψ's own metric" story would not. It does **not** decide between them; the comass proxy and a rescaled-ψ arm (R4) are the instruments that can.

**Sampler sensitivity.** The Q1b rule with S1 instead of S2 on the definite seeds: K-psi-S1 locks 67% vs K-phi 0% → inconclusive (φ-only does not lock), the same as the primary.

**K-phi vs K-full.** 0.0% vs 99.0%; Δ median −0.356, CI [−0.362, −0.347].

**K-psigram vs K-full.** Definite 72% vs 99%, Δ median −0.001; split 76% vs 99%, Δ median −0.000. Gram plus *any* tested 3-form (definite or split) locks in about three quarters of seeds, but φ locks more often than either (99%).

**Cross-calibration.** K-full's plane is φ-calibrated (|φ₀| 1.000, 97–98% of triples). K-psi's plane is only partly φ-calibrated on definite seeds (|φ₀| 0.874) and not on split (0.369), and is ψ-calibrated everywhere. |ψ₀| exceeds 1 (1.5–2.0) because ψ is Frobenius-matched but not Euclidean-compatible, which is the scale difference the comass proxy is meant to quantify.

**Persistence.**

| cell | K-full | K-gram | K-phi | K-psi | K-psigram | K-rand3 |
|---|---|---|---|---|---|---|
| N = 12 | 99.0% | 10.5% | 1.0% | 61.0% | 92.5% | 75.5% |
| N = 24 | 99.0% | 3.0% | 0.0% | 79.0% | 74.0% | 33.5% |
| N = 48 | 18.5% | 9.0% | 0.0% | 90.5% | 46.5% | 0.5% |

Q1a is "no" at all three sizes; Q1b is "inconclusive (φ-only does not lock)" at all three sizes in both strata. Notably at N = 48 and 250 steps K-full has not converged (18.5%) while K-psi has (90.5%).

**N = 48 convergence.** K-full goes from 18.5% (250 steps) to **75.5% (1,000 steps)**, median S3 0.892 → 0.989; K-gram stays at 9% → 6%. So K-full's N = 48 locking is slow rather than absent, and K-gram's absence is not a step-count artefact.

**Spontaneity** (round-2 KS definition, 1,000 MC replicates; arms with at least 20 locked runs). K-full: D = 0.0080, p = 0.545, consistent with the Haar-G₂ reference. K-psi (D = 0.084), K-psi-S1 (0.094), K-psigram (0.085) and K-rand3 (0.114): each p < 0.001, not Haar-G₂. The ψ arms lock onto planes that are not distributed like random associative planes of φ, which is expected since their natural planes belong to their own forms.

## 5. Prediction on file, scored

The r1 prediction and its dated addendum are unedited in the frozen file. Against the outcome:

| Prediction | Outcome | Score |
|---|---|---|
| Q1a = no | Q1a = no | **correct** |
| K-gram mostly 2-collapses rather than 3-locks | K-gram 75.5% 2-collapse, 3.0% 3-lock | **correct** |
| K-rand3 mostly 2-collapses rather than 3-locks | K-rand3 0% 2-collapse; 33.5% 3-lock, 66.5% diffuse | **wrong** (mostly diffuse, not 2-collapse) |
| K-full and K-psigram (definite) 3-lock at similar rates | 99% vs 72% | **wrong** (similar S3, but a 27-point rate gap) |
| Q1c = yes (φ not specific) | secondary: φ locks more often than ψ+Gram (99% vs 72–76%) | **not supported** as stated |
| Addendum: Q1b = no in both strata ("any 3-form has a maximising 3-plane, so K-psi locks about as often as K-phi") | Q1b inconclusive in both strata; K-phi 0%, K-psi 83%/75% | **wrong**: my premise failed on the φ side, since K-phi does not lock at all at 250 steps |
| Expected reading: row 2 | Row 5 | **wrong** |

The part of the reasoning that held is the Gram half: Gram alone collapses to a 2-plane and cannot reproduce locking. The part that failed is the claim that a single 3-form term by itself drives locking, including φ.

## 6. What R3 does and does not establish

- **Established (rule-bound):** replication of round-2 locking (99%); the Gram term alone does not lock (Q1a = no); the λ₃-share gate is necessary (K-gram's S3 ≈ 1 is a 2-plane).
- **Not established:** whether G₂ specifically matters (Q1b undecided in both strata); no QBP-claims row rises above research grade, and per oppenheimer's pre-commitment (seq 56) the failed K-phi precondition goes to the row-5 record and to a round-4 pre-registration, not to a rereading of rows 1–4.
- **Named diagnosis for any follow-up** (arch seq 57): K-phi at 250 steps cannot be told apart from K-phi locking slowly, and ψ's larger comass at the same learning rate may inflate K-psi's locking. The dose-response sign is consistent with this and does not prove it.
- **Candidate R4 probes (arch seq 58, to be drafted, not frozen):** a rescaled-ψ arm (max|ψ| = 1) and a K-phi step-count sensitivity arm. The N = 48 1,000-step result (K-full 18.5% → 75.5%) shows that step count matters at this scale.

## 7. R3-Q2 (status)

Not run. The box loadavg was above the frozen 1.0 start ceiling throughout (desktop load), and arch seq 59 ruled to wait for a quiet window rather than force it. The deterministic multiply-count closure (not a host measurement) is already known: W1 101 vs 104, W2 71 vs 49, W3 256 vs 126 multiplies, so no workload gives the CD-MAC a 2× advantage (closure verdict **no**). The W1 count is single-derived (qbp-cu-implementor, arch seq 57); it cannot change the verdict since 1.03 < 2. The host-ns benchstat is pending, and **every R3-Q2 verdict carries this disclosure:** "Host ns measured on an FX-8350 without AVX2. The CD-MAC's wide-lane parallelism is serialised, so host ns is biased against CD."

## 8. Open items

1. Push `143bd69` and this document (qbp-cu-implementor).
2. Post-hoc comass proxy per seed (`‖g‖_op³` and sampled max|ψ|), labelled post-hoc and diagnostic only.
3. R3-Q2 host-ns in a quiet window, with real benchstat cross-check if it becomes available.
4. Route the result through the planning-consequence table (inter#194, `3bef08fb…`) — architecture.
5. R4 draft (not frozen until this report is posted and read).
