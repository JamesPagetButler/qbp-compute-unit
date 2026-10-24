# Spike MDCU — round 3 pre-registration (FROZEN v1)

Status: **FROZEN v1 (2026-10-24).** Nothing built, nothing run at the time of freezing. This file is a verbatim copy of DRAFT r3 (commit `d26ecf8`, sha256 `6eca32c04168035e67bf6c4a004cb05cbec29860f2c72db850486e01b1271631`), with only the three edits the header allows: this status line, the §6 answers, and no role-label swap (qbp-oppenheimer confirmed S2 as written). No rule may change after any R3 data is seen. Branch B, spike/napkin grade. Drafted by antigravity.
Supersedes: DRAFT r2, sha256 `f19963c3ab327d85ecd89f623a83fa2284607f768a71eaa028c901560e911576` (commit `2dfa0e9`); DRAFT r1, sha256 `54c01a022b91253e034fbbeecbb4f45cbe7d59cc1ce7b424cb4f5cdab6a563e4` (commit `cbdecc5`).
Baseline: [`../mdcu-round2-final-report.md`](../mdcu-round2-final-report.md) (sha `bf85e16c…`, accepted seq 35) and [`../mdcu-round2-final-report-amendment-1.md`](../mdcu-round2-final-report-amendment-1.md).
Rulings applied: qbp-architecture seq 32, 35 and 42 (r2: "no further changes"; one line requested, added in §1.2); qbp-cu-implementor seq 31, 34, 36, 38 and 41.
Gates: qbp-oppenheimer confirms the arm physics and picks the definite-stratum sampler (§6) → freeze v1 with a sha → deming box-safety and idle-box window → beekeeper run go.
**Freeze procedure:** both sampler options are fully specified below, so oppenheimer's reply decides only which one is primary. The freeze copies this file verbatim as `mdcu-round3-prereg-FROZEN-v1.md`. The only edits allowed are the status line, the §6 answers, and (if oppenheimer picks S1) swapping the S1/S2 role labels in §1.1–§1.2. Any other change from oppenheimer goes back through arch.

## 0. Changes since r1

| # | Change | Source |
|---|---|---|
| 1 | Primary contrasts are now **Q1a "Does the Gram term drive locking?" = K-gram vs K-full** and **Q1b "Does G₂ matter?" = K-phi vs K-psi**, per B_ψ stratum. r1's "Gram suffices" and "G₂ term suffices" rules and the K-full vs K-psigram contrast (r1 Q1c) become secondaries. | seq 32 |
| 2 | ψ is **stratified by construction**: a pre-stated rejection sampler gives 100 seeds per stratum (minimum 50), replacing r1's "stratum below 20 seeds is underpowered" clause. | seq 35 item 3 |
| 3 | Thresholds signed off: λ₃ share ≥ 0.15, ±0.05 equivalence on Δ, 10-point rate margin. | seq 35 item 2 |
| 4 | Fresh seeds only; no round-2 seed reproduction (r1 open question 4 is closed). | seq 35 item 4 |
| 5 | R3-Q2 is **option (B)**: a tape-counted multiply closure, plus an idle-box host-ns benchstat as tiebreak. The CD side uses emulator-authoritative counts, and the no-AVX2 bias is disclosed in the verdict. (C) is rejected. | seq 32, 35, 34 |
| 6 | Prediction kept verbatim. An addendum for the new contrasts is added and dated pre-freeze. | seq 35 item 5 |
| 7 | Pre-freeze disclosure of the ψ sampler's acceptance rate (§1.2). This is a property of the sampler, not of any dynamics. | r2 |
| 8 | (r3) The draw-count distribution per stratum is reported alongside the verdict. | arch seq 42 |
| 9 | (r3) The definite-stratum sampler is fully specified both ways: **S2 (ψ = g*φ) primary**, following arch's seq 42 lean, and **S1 (conditioned Gaussian tail) as a sensitivity arm**, K-psi-S1. The roles swap if oppenheimer picks S1. S2's scale range is pre-stated, and its property check is disclosed. | arch seq 42; this draft |

Background (unchanged from r1, round-2 report §3):
- ‖[a,b,c]‖² = 4(Gram₃ − φ²) for imaginary a, b, c, so φ-only vs full tests the Gram term, not G₂.
- R3-Q2's original 0.5× model-cycle "yes" is unreachable by construction.
- φ₀-primary is dropped for good.

## 1. R3-Q1 — the structure question (primary)

**Question.** Does the O(7)-invariant Gram term drive the ring's 3-plane locking? And does the G₂ 3-form φ matter, compared with a generic 3-form ψ of each GL(7) type?

**Dynamics (unchanged from r1).**
- Round-2 Test B ring, overlap layout (centre + (i−2, i−1, i) terms), same optimiser: lr 0.04, μ 1.5 neighbour term, 250 steps, unit imaginary 8-vectors, Euclidean metric.
- Only the triple energy E(a,b,c) changes between arms.
- A new signed-energy runner is required (§5). It must be **bit-identical to round-2 `RunRingGen`** for K-full on a fixed seed (test gate).

### 1.1 Arms

All arms use the same initialisation per seed (paired design).

| Arm | E(a,b,c) | 3-form | Role |
|---|---|---|---|
| K-full | ‖[a,b,c]‖² = 4Gram₃ − 4φ² | φ | **primary** (Q1a; replication gate) |
| K-gram | 4Gram₃ = 4‖a∧b∧c‖² | none (O(7)) | **primary** (Q1a) |
| K-phi | −4φ² | φ | **primary** (Q1b) |
| K-psi | −4ψ² | ψ (stratified; definite seeds use S2 and split seeds use S1, §1.2) | **primary** (Q1b) |
| K-psi-S1 | −4ψ² | ψ from S1 (definite seeds only) | secondary (sampler sensitivity) |
| K-psigram | 4Gram₃ − 4ψ² | ψ (same ψ as K-psi for the seed) | secondary |
| K-rand3 | round-2 null 3 (random alternating vector-valued, Frobenius-matched) | none in expectation | reference; E = (16/5)Gram₃ + noise |
| K-none | μ term only | — | sanity |

### 1.2 ψ sampler (pre-stated; stratified by construction)

**Distribution.** The 35 independent components ψ_{abc}, a < b < c, are i.i.d. standard Gaussian. This is the same distribution as antisymmetrising a full Gaussian tensor, after normalisation. ψ is then Frobenius-matched to φ: Σ over all ordered (a,b,c) of ψ²_{abc} = Σ φ²_{abc} = 42.

**Classifier.**
- B_ψ(v,w)·vol = (1/6)(ι_vψ)∧(ι_wψ)∧ψ (Hitchin).
- **Definite** (all eigenvalues of one sign) means compact-G₂ type in some metric.
- **Split** means signature (3,4) or (4,3).
- **Degenerate** means min |eig| < 1e-9·max |eig|. Such draws are rejected and counted.

**Stratum assignment by seed parity.** Even seeds are definite; odd seeds are split. This gives 100 seeds per stratum over seeds 3000–3199.

**S1 (rejection; used for every split seed and for K-psi-S1 on definite seeds).** For each seed:
1. Draw from that seed's ψ PCG stream until the draw's type matches the seed's stratum.
2. Accept the first match.
3. Cap: 20,000 draws per seed. A seed that hits the cap is marked "sampler-miss" and dropped from both ψ arms.
4. A stratum with fewer than 50 accepted seeds is "inconclusive (underpowered)".
5. Draw counts are recorded per seed. **The draw-count distribution per stratum (min, median, max, histogram) is reported alongside the verdict** (arch seq 42).

**S2 (deformed φ; primary for K-psi on definite seeds).** For each even seed, from a separate PCG stream (its constant differs from S1's and from rand3's):
1. Draw u ~ U(0,1) and set the scale s = 0.05 + 0.20·u, so s ∈ [0.05, 0.25].
2. Draw G, a 7×7 matrix with i.i.d. N(0,1) entries. Set S = (G + Gᵀ)/2 − (tr/7)·I, which is symmetric and traceless (the conformal part is removed by the Frobenius match anyway).
3. Set g = exp(s·S) by symmetric eigendecomposition. Set ψ_{abc} = Σ φ_{ijk} g_{ia} g_{jb} g_{kc}, then Frobenius-match ψ to 42.
4. No rejection is needed: g*φ lies in φ's GL(7) orbit, so it is always definite. A test asserts this.
5. Recorded per seed: s, the distance d = ‖ψ − φ‖/√42, and the condition number of g·gᵀ (the metric distortion).
6. The rotation part of g is omitted by construction, so S2 varies only the metric deformation away from Euclidean.

**S2 pre-freeze disclosure (sampler property, not dynamics).** From `check_s2_sampler.py`, generator seed 616161, 200 draws per level:

| s | definite | median d [min, max] | median metric condition |
|---|---|---|---|
| 0.05 | 200/200 | 0.136 [0.089, 0.186] | 1.81 |
| 0.10 | 200/200 | 0.267 [0.171, 0.360] | 3.26 |
| 0.25 | 200/200 | 0.595 [0.407, 0.888] | 19.0 |
| 0.50 | 200/200 | 0.899 [0.681, 1.070] | 330 |

s = 0.5 was excluded as too extreme, which is why the range is capped at 0.25. s ≥ 0.05 keeps ψ away from φ itself.

**Pre-freeze disclosure (sampler property, not dynamics).** Measured with numpy in antigravity scratch (`check_bpsi_strata.py`, `check_bpsi_validate.py`), on generator seeds 424242 and 515151, which are unrelated to any spike stream:
- **Only about 0.2% of draws are definite** (4/2000; 1/1000).
- Split draws are (4,3) 504 and (3,4) 495 of 1000. No other signatures and no degenerate draws occurred.
- Classifier checks:
  - B_φ = −144·I (definite).
  - φ + ε·noise stays definite at ε ≤ 0.5 (200/200 at each level) and drops to 11% at ε = 1. This is consistent with an open orbit.
- So a "generic" ψ is split-type. The definite stratum is a tail of the Gaussian, costing about 500 draws per seed (miss probability at the cap ≈ e⁻⁴⁰).

**Sampler roles (qbp-oppenheimer confirms, §6).**
- Default for the freeze: **S2 is primary** for the definite stratum, following arch's seq 42 lean. It controls the distance from φ, which makes "φ specifically vs a G₂ structure for another metric" interpretable.
- **S1 is the sensitivity arm** (K-psi-S1).
- If oppenheimer picks S1, the two roles swap: S1 drives Q1b(definite), and S2 becomes the sensitivity arm K-psi-S2. All rules are otherwise identical.
- The split stratum always uses S1.

### 1.3 Measurements per run

All measurements are stored. Eigenvalues are stored this time, which closes round-2 limitation 1.
- λ₁ ≥ … ≥ λ₇ are the eigenvalues of the 7×7 Gram of Z[:,1:]. The SVD-k share is s_k = Σ_{i≤k} λ_i / Σλ.
- **3-lock** ⇔ s₃ ≥ 0.95 **and** λ₃/Σλ ≥ 0.15.
- **2-collapse** ⇔ s₂ ≥ 0.95.
- Otherwise the run is **diffuse**.
- Plane calibration (diagnostic): |φ₀| and |ψ₀| of the top-3 plane, and the fraction of consecutive triples with |φ₀| ≥ 0.95 (and the same for ψ₀). Own-form calibration is tautological; the informative diagnostic is **cross-calibration** (φ₀ under ψ arms, ψ₀ under φ arms).
- Spontaneity KS vs the Haar-G₂ reference (round-2 definition), for any arm that 3-locks. Secondary.

### 1.4 Grid

- Main cell: N = 24, σ = 0.1, **200 fresh seeds 3000–3199**. This excludes 1000–1199, 9000, 77–83 and both round-2 grids.
- ψ and rand3 tensors come from distinct PCG streams, with constants that differ from round 2.
- Persistence cells: N = 12 and N = 48 at σ = 0.1, same seeds and strata.

### 1.5 Decision rules

These apply to the main cell, paired by seed. Δ(X,Y) is the median paired s₃(X) − s₃(Y), with a bootstrap 95% CI over 10,000 resamples. "Rate" means the 3-lock rate.

- **Replication gate.** K-full must 3-lock in at least 90% of seeds (round-2 C1). If it does not, every R3-Q1 verdict is "inconclusive (non-replication)".

- **Q1a — Does the Gram term drive locking?** (K-gram vs K-full, all 200 seeds)
  - *yes* iff all three hold:
    - K-gram rate ≥ 90%;
    - |rate(K-full) − rate(K-gram)| ≤ 10 points;
    - the CI of Δ(full, gram) lies inside ±0.05.
  - *no* iff K-gram rate < 50%, **or** Δ(full, gram) ≥ 0.15 with the CI excluding 0.
  - Otherwise *inconclusive*.

- **Q1b — Does G₂ matter?** (K-phi vs K-psi, separately per stratum, paired within each stratum's seeds)
  - Precondition: K-phi rate ≥ 50% on that stratum's seeds. If it is lower, Q1b is "inconclusive (φ-only does not lock)" for that stratum. Equivalence between two non-locking arms says nothing.
  - *no (G₂ not needed: a ψ of this type locks as well as φ)* iff both hold:
    - |rate(K-phi) − rate(K-psi)| ≤ 10 points;
    - the CI of Δ(phi, psi) lies inside ±0.05.
  - *yes (φ locks more)* iff Δ(phi, psi) ≥ 0.15 with the CI excluding 0, **or** rate(K-phi) − rate(K-psi) ≥ 40 points.
  - *yes-reversed (ψ locks more)* iff Δ(phi, psi) ≤ −0.15 with the CI excluding 0. This is reported and is not read as G₂ evidence.
  - Otherwise *inconclusive*.

**Reading table.** This is pre-registered so that the interpretation cannot be chosen after the fact.

| Q1a | Q1b definite | Q1b split | Reading |
|---|---|---|---|
| yes | any | any | Locking is reproduced by the O(7) Gram term alone. Round-2 locking is not G₂ evidence, consistent with the null-3 = Gram + noise hypothesis. |
| no | no | no | The 3-form term contributes, but any 3-form does. Not G₂-specific. |
| no | no | yes | The GL(7) type matters (compact-type ψ behaves like φ; split does not). G₂-type matters, but not Euclidean-compatible φ specifically. |
| no | yes | yes | φ specifically matters. **This is the only cell that supports a G₂-specific reading.** |
| other | | | Inconclusive, or a mixed result reported as-is. |

**Secondaries (diagnostic only, never a decision input):**
- K-phi vs K-full (oppenheimer's original null; tests the Gram complement).
- K-psigram vs K-full per stratum (r1 Q1c).
- 2-collapse rates for every arm.
- K-rand3 3-lock vs 2-collapse.
- Cross-calibration.
- N = 12 and N = 48 persistence.
- N = 48 convergence for K-full and K-gram at steps ∈ {250, 1000}.
- Sampler draw-count distribution per stratum (reported alongside the verdict, seq 42).
- Sampler sensitivity: the Q1b rule applied with K-psi-S1 in place of K-psi on the definite seeds. This is reported and is not a decision input.
- Dose-response: Spearman correlation between the paired s₃(K-phi) − s₃(K-psi) and the S2 distance d, on the definite seeds.

### 1.6 Prediction on file

**Verbatim from r1; unedited.** The r1 labels map as: r1 Q1a (Gram suffices) is close to the new Q1a; r1 Q1c is now a secondary.

> **Prediction on file (falsifiable, mine):** K-gram and K-rand3 mostly 2-collapse rather than 3-lock; K-full and K-psigram (definite stratum) 3-lock at similar rates; hence Q1a = no, Q1c = yes (φ not specific). Recorded so the result is a real test, not a hoped-for outcome.

**Addendum (2026-10-24, pre-freeze, for the seq-32 contrasts; written before any R3 dynamics exist):**
- Q1a = no (as above).
- Q1b = no in **both** strata. My reasoning: any 3-form has a maximising 3-plane on unit triples, and the ring's overlap term only needs a common plane. So K-psi 3-locks about as often as K-phi, and the split type does not prevent it.
- Expected reading: row 2, "the 3-form term contributes, but any 3-form does".

## 2. Resource estimate (assumed, scaled from round-2 measurements)

- Main cell: 200 seeds × 7 arms, plus K-psi-S1 on 100 seeds, ≈ 4 min.
- ψ sampler: about 50,000 S1 classifications (cost independent of N), estimated under 1 min in Go (assumed; Python measured 4 ms/draw). S2 is negligible.
- Persistence: N = 12 and N = 48 ≈ 2 + 6 min.
- N = 48 at 1,000 steps for 2 arms ≈ 6 min.
- **About 20 min single-threaded in total (assumed).** Cap 1 h.
- Box limits: 1 core, `nice -n 19`, prebuilt binary, ≤ 100 MB RAM, ≤ 50 MB output, no network or GPU. Round-2 §10 kill conditions apply unchanged.

## 3. R3-Q2 — efficiency: tape-counted closure plus host-ns tiebreak (option B, ruled seq 35)

### 3.1 Workloads and paths

| W | Workload | CD path | Non-CD path |
|---|---|---|---|
| W1 | Albert determinant scalar (round-2 Q2 workload) | B1lean | best of B0b and the cross-product B0 |
| W2 | φ(a,b,c) scalar | Re-part of `OMulCDMAC`-based (bc), dotted with a | a·(b×c) from the 21 shared minors |
| W3 | associator vector [a,b,c] | 4 × `OMulCDMAC` | coassociative 4-form contraction: 21 minors, then 28 three-term expansions |

### 3.2 Multiply-count closure (no host timing)

**Counting rule.** Count data×data real multiplies only. Multiplies by ±1 (the sign ROM; Lean T10 `sign_unit_sed`) and by the constant 2 are not counted. Additions are reported in a separate column.

**Method.** Counts are taken **by tape in code**, not by hand.

**CD-side source.** `OMulCDMAC` (`emulator/mdcu/cdmac.go:92–99`, origin `b280425`/`cbdecc5`): 64 per product, 256 for four (qbp-cu-implementor, seq 34). Provenance note: the frozen Gearbox `OMul64` is a tier-gated stub, so the spike's CD-MAC is the authoritative kernel.

**Cross-check.** qbp-cu-implementor cross-counts the non-CD side independently.

**Hand counts.** These are superseded by the tape and are listed only for transparency: W3 is 126 vs 256; W2 is 49 vs about 72; W1 is 101 vs 118 MACs (round-2, measured).

**Correctness gate.** Every path must agree with the reference implementation to 1e-12 on 1,000 out-of-grid random inputs.

**Closure verdict per workload.** "CD has a multiply-count advantage of 2× or more" is yes or no. It is deterministic.

### 3.3 Idle-box host-ns benchstat

**Setup.**
- One Go benchmark per (workload, path), on a batch of 64 fixed-seed inputs.
- Prebuilt with `go test -c`.
- Run as `GOMAXPROCS=1 nice -n 19 taskset -c <k> ./mdcu.test -test.run '^$' -test.bench 'BenchmarkR3Q2' -test.count 10 -test.benchtime 200ms`.
- Compared with `benchstat` (default test, α = 0.05).

**Load ceiling.**
- The 1-min loadavg must be ≤ 1.0 at the start. It is recorded at the end.
- If it is > 1.5 at the end, rerun once. If it is still > 1.5, the benchstat is "inconclusive (load)".
- Wall cap: 10 min.
- Window: **deming schedules it**, after the R3-Q1 run.

**Benchstat verdict.**
- *yes* iff, on at least one workload, the best CD path's median ns is ≤ 0.5 × the best non-CD path's median, with p < 0.05.
- *no* iff, on every workload, the best CD path is not significantly faster (the non-CD path is faster, or benchstat reports "~").
- Otherwise *inconclusive*.

### 3.4 R3-Q2 verdict

- If the closure and the benchstat agree, the verdict is their common answer.
- If they disagree, the benchstat decides, labelled "(host-ns tiebreak)".
- **Every R3-Q2 verdict carries this disclosure:** "Host ns measured on an FX-8350 without AVX2. The CD-MAC's wide-lane parallelism is serialised, so host ns is biased against CD."
- No routing or cost model is used (seq 32). The node-packing scheduler may be built as a capability, but not as a decision instrument.

## 4. Kill conditions

Round-2 §10 unchanged. Wall cap: 1 h for R3-Q1, 10 min for R3-Q2. Pause or abort on any request from the federation or from deming's box-watch.

## 5. Capability gaps (what must be built after freeze; nothing exists yet)

**To build:**
- A signed-energy ring runner (energy + gradient callback), bit-identical to `RunRingGen` for K-full.
- A scalar 3-form type (φ, ψ); the wedge map for Gram₃ (35 components); the B_ψ classifier; the S1 stratified sampler with draw counting; the S2 sampler (symmetric eigendecomposition, pullback g*φ).
- Stored eigenvalue spectrum, and 3-lock / 2-collapse classification in the measure struct.
- A steps parameter in the runner.
- R3-Q2: a multiply tape counter; the W2 and W3 paths on both sides; the cross-product B0 variant for W1; `BenchmarkR3Q2*`.

**Tests first:**
- Runner equivalence.
- The identity ‖[a,b,c]‖² = 4(Gram₃ − φ²) to 1e-12.
- ψ Frobenius matching.
- Classifier:
  - B_φ is definite;
  - φ + 0.05·noise is definite;
  - split draws are only (3,4) or (4,3);
  - every S2 draw is definite, S is traceless, and s ∈ [0.05, 0.25].
- Parity-stratum assignment.
- Alternation of every arm.
- R3-Q2 path agreement to 1e-12.
- A smoke run on an out-of-grid seed (disclosed).

## 6. Confirmations (freeze gates — all cleared)

1. **qbp-oppenheimer — CONFIRMED (mdcu-spike seq 46):** the arm physics (K-full = Harvey–Lawson identity, so K-gram + K-phi = K-full exactly and the Q1a/Q1b contrasts are a clean partition), and that the B_ψ signature ("all eigenvalues of one sign" = definite) is the right classifier. Non-blocking note for the report: for a definite ψ = g·φ, the G₂ structure is for the metric g_ψ while the ring minimises in the Euclidean metric; that mismatch is the Q1b question, and recording d and the condition number per seed is what makes it readable. Omitting the rotation part of g is right: a rotated φ is G₂-conjugate to φ and would be K-phi in disguise.
2. **qbp-oppenheimer — "confirm, S2 as written" (seq 46):** S2 is primary for the definite stratum, and S1 is the sensitivity arm. The dose-response secondary (gap vs d) is the reading he most wants to see. No role-label swap.
3. **qbp-architecture — CONFIRMED (seq 44):** r3 is freeze-ready, subject to items 1–2 (now cleared). He verified commit `d26ecf8` and sha256 `6eca32c0…` on disk, and that the diff against r2 is the §1.2 and sampler changes only.

Remaining gates before any run: deming box-safety clearance and idle-box window (R3-Q2); beekeeper run go.
