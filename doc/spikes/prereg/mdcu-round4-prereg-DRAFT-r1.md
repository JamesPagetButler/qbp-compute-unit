# Spike MDCU — round 4 pre-registration (DRAFT r1)

Status: **DRAFT r1 (2026-10-26). NOT FROZEN. Nothing built, nothing run.** Drafted by antigravity, at the beekeeper's instruction ("draft the R4 pre-registration, but don't freeze it until the R3 report has been read").
Freeze order, per arch seq 71/72: **R3-Q2 host-ns posts first**, then this draft goes to arch for rulings and to qbp-oppenheimer for the physics confirm, then deming's box-safety gate, then a freeze with a sha, then the beekeeper's run go. No rule may change after any R4 data is seen. Branch B, spike/napkin grade.
Baseline: [`../mdcu-round3-results.md`](../mdcu-round3-results.md) (R3-Q1 read and independently verified by arch, seq 71), the frozen R3 pre-registration [`mdcu-round3-prereg-FROZEN-v1.md`](mdcu-round3-prereg-FROZEN-v1.md) (sha256 `070aeab8…1848`), and the post-hoc diagnostic [`../raw/mdcu-r3-comass-posthoc-2026-10-26.md`](../raw/mdcu-r3-comass-posthoc-2026-10-26.md).
Rulings applied: qbp-architecture seq 71, 72; qbp-oppenheimer seq 56, 69. Items marked **[PROPOSED]** are mine and need a ruling.

## 0. Why R4, and what R3 left open

R3-Q1 reading was row 5 (inconclusive/mixed). The named diagnosis is the **comass coupling-scale confound** (oppenheimer seq 53/56, arch seq 71). The post-hoc diagnostic (R3 raw, no new runs) puts numbers on it:

| Fact (all post-hoc, from R3 data) | Value |
|---|---|
| comass(φ) | 1 (estimator: 1.000000000) |
| comass of the ψ forms R3 used (Frobenius-matched to φ) | S2 1.24–2.23 (median 1.63); S1-definite 1.55–2.29 (1.93); S1-split 1.75–2.25 (1.98) |
| Pooled K-psi forms (200) | median **1.880**, comass² median **3.536** |
| So the −4ψ² well is deeper than −4φ² by | about 2.7× to 3.9× at the comass triple |
| K-phi vs K-none at 250 steps | median S3 0.641 vs 0.601; both 100% diffuse. φ-only barely moves the ring at this coupling |
| comass vs distance d on S2 | Spearman 0.972: collinear, so the R3 dose-response cannot separate them |
| Within-form comass vs locking | not increasing: S2 terciles 91% → 82% → 77%; S1-split 73/73/79; S1-definite 67/61/74 |

**The limit of R3:** every ψ form had comass above φ's, so R3 contains no ψ at φ's coupling scale. The confound cannot be tested within R3. R4 builds the missing comparison.

## 1. R4-Q1 — coupling scale or form? (primary)

**Dynamics (unchanged from R3).** Round-2 Test B ring, overlap layout, lr 0.04, μ 1.5 neighbour term, unit imaginary 8-vectors, Euclidean metric, N = 24, σ = 0.1. Only the triple energy and the step count change between arms. Same initialisation per seed (paired design).

**Comass levels.**
- For a 3-form f and level L: f_L = L · f / comass(f).
- comass is computed with `ComassEstimate` (multistart ascent on the Stiefel manifold; a lower bound; calibrated exact on φ). For R4 use 200 starts × 500 iterations, plus a second independent restart set. If the two disagree by more than 1e-6 (relative), that seed's rescaled arms are dropped and counted. In R3's post-hoc run the maximum relative difference over the 200 paired estimates was 1.3e-15.
- φ_L = L·φ, since comass(φ) = 1.
- **Level 1** is φ's native level. **Level κ = 1.88**, the pooled median comass of the R3 K-psi forms. Its square is κ² = 3.5344.
- Arms at level 1 use energy −4 f_1². Arms at level κ use energy −4 f_κ². For K-full and K-gram at κ, the **whole** triple energy is scaled by κ², which keeps the Harvey–Lawson identity (K-gram@κ + K-phi@κ = K-full@κ exactly).

### 1.1 Arms

Seeds 4000–4199 (fresh). ψ sampler is R3's, unchanged: S2 on even seeds (definite), S1-split on odd seeds, both Frobenius-matched to 42 before any rescale.

| Tier | Steps | Arms |
|---|---|---|
| A | 250 | **K-full**, **K-gram**, **K-phi**, **K-psi** (native, replicates R3), **K-psi@1**, **K-phi@κ**, **K-psi@κ**, **K-full@κ**, **K-gram@κ**, K-none |
| B | 1000 | K-phi, K-psi@1, K-phi@κ, K-psi@κ |
| C | 4000 | K-phi, K-psi@1 |

Notes:
- K-phi is level 1 by definition. K-psi@1 is ψ rescaled to comass 1.
- The S1-definite sensitivity arm (K-psi-S1) is dropped; R3 already carries it.
- The step ladder follows arch seq 72 (250 / 1000 / 4000; the comass argument predicts about 4× the steps).

### 1.2 Measurements

As R3 §1.3: eigenvalues stored, s₃ and λ₃/Σλ, **3-lock** ⇔ s₃ ≥ 0.95 ∧ λ₃/Σλ ≥ 0.15, **2-collapse** ⇔ s₂ ≥ 0.95, else diffuse. Additionally, per seed: native comass of ψ, the rescale factors, the two-set convergence difference, and (even seeds) s and d.

### 1.3 Gates and regression check

- **G4 (replication gate):** K-full at 250 steps 3-locks in at least 90% of seeds. Otherwise every R4 verdict is "inconclusive (non-replication)".
- **Regression check (bit-identity):** before the grid, rerun seeds 3000–3009 for K-full, K-phi and K-psi (native) at 250 steps with the R4 binary. The stored statistics must be identical to the R3 raw file. This guards that R4's harness changes did not alter R3 semantics. (This is the one deliberate reuse of R3 seeds; it is a code check, not data.)
- **R3 K-psi replication (diagnostic, not a gate):** native K-psi rate within ±15 points of R3's 79%.

### 1.4 Decision rules

Same machinery and definitions as R3 §1.5. Paired by seed. "Rate" = 3-lock rate. Δ(X,Y) = median paired s₃(X) − s₃(Y) with a 95% percentile bootstrap CI over 10,000 resamples.
- **equivalent(X,Y)** ⇔ |rate(X) − rate(Y)| ≤ 10 points ∧ CI of Δ lies inside ±0.05.
- **clearly-higher(X,Y)** ⇔ rate(X) − rate(Y) ≥ 40 points ∨ (Δ(X,Y) ≥ 0.15 ∧ CI excludes 0).

**Q4a — does φ lock if given more steps?** K-phi at 250 / 1000 / 4000, all 200 seeds.
- *locks slowly* iff rate(K-phi, 4000) ≥ 50%.
- *does not lock by 4000* iff rate(K-phi, 4000) < 10%.
- Otherwise *partial*.
- The first rung with rate ≥ 50% is reported as R\*.

**Q4b — is the R3 φ/ψ gap coupling scale or form?** Tier A, all 200 seeds, four arms: K-phi, K-phi@κ, K-psi@1, K-psi@κ.
- Form effects: F₁ = rate(K-psi@1) − rate(K-phi); F_κ = rate(K-psi@κ) − rate(K-phi@κ).
- *COUPLING* iff equivalent(K-psi@1, K-phi) ∧ equivalent(K-psi@κ, K-phi@κ) ∧ clearly-higher(K-phi@κ, K-phi). The R3 gap was coupling scale.
- *FORM* iff clearly-higher(K-psi@1, K-phi) ∧ clearly-higher(K-psi@κ, K-phi@κ). The gap survives coupling match.
- Otherwise *MIXED*, with all four rates and both strata reported.

**Q4c — does φ specifically matter at matched coupling? (the R3 Q1b redo)** Per stratum, using R3's Q1b rule exactly (no / yes / yes-reversed / inconclusive, same precondition).
- Evaluate at the **first** cell in this fixed order where rate(K-phi at that cell) ≥ 50% on that stratum's seeds: (level 1, 250), (κ, 250), (1, 1000), (κ, 1000), (1, 4000).
- The ψ arm at the same cell is K-psi@1 or K-psi@κ.
- If no cell qualifies, Q4c is "inconclusive (φ never locks)".

**Reading table (pre-registered, so the interpretation cannot be chosen after the fact):**

| Q4b | Q4c (both strata) | Reading |
|---|---|---|
| COUPLING | no, no | The R3 gap was coupling scale. At matched coupling φ and ψ lock alike: a 3-form term drives locking; **not φ-specific**. |
| COUPLING | yes, yes | Coupling scale explains why φ looked inert, **but at matched coupling φ locks more than ψ**. The only cell supporting a φ-specific reading (still napkin; needs a fresh-seed replication before any anchor). |
| COUPLING | mixed or inconclusive | Recorded as-is. No claim. |
| FORM | any | The gap is not coupling scale. ψ locks where φ does not at matched comass. Reported; ψ-locks-more is not read as G₂ evidence. Candidate mechanisms are for oppenheimer to name (see Open Question 5). |
| MIXED | any | Recorded as-is. No claim. |
| gate G4 fails | | Inconclusive (non-replication). |

### 1.5 Secondaries (each carries its own hypothesis and kill; never a decision input)

**S-A Spontaneity** (Haar-G₂ KS, the R3 analyzer's definition; arch seq 72). Computed per arm with at least 20 three-locked runs; otherwise "untestable".
- *H-A1:* K-full's planes are Haar-G₂-consistent (replicates R3 p = 0.545). **Kill:** p < 0.01.
- *H-A2:* every ψ arm (K-psi, K-psi@1, K-psi@κ) differs from Haar-G₂ (p < 0.01). **Kill:** any ψ arm with ≥ 20 locked runs has p ≥ 0.05 → the R3 non-Haar split is not robust to rescaling.
- *H-A3:* any φ arm that locks in ≥ 20 runs (K-phi@κ, or K-phi at a later rung) is Haar-G₂-consistent (p ≥ 0.05). **Kill:** p < 0.01.

**S-B Dose-response on the rescaled arm** (arch seq 72, oppenheimer seq 69 (d)). Definite seeds. ρ = Spearman of the paired s₃(K-phi@L) − s₃(K-psi@L) against d, at L = κ and L = 1 (250 steps), with a 10,000-resample bootstrap CI. R3: ρ = −0.258 raw, partial on comass −0.108 (post-hoc).
- *H-B:* the dependence on d survives rescaling: |ρ| ≥ 0.20, CI excludes 0, same sign as R3. If so, it is about g_ψ vs Euclidean, not coupling strength.
- **Kill:** CI includes 0 and |ρ| < 0.10 → the R3 dose-response was carried by comass.

**S-C Finding (a) at matched coupling** ("neither term locks alone; together they do"). *Holds* iff rate(K-full@κ) ≥ 90% ∧ rate(K-gram@κ) < 50% ∧ rate(K-phi@κ) < 50%. Otherwise reported as "(a) fails at matched coupling: a single term locks".

**S-D Dose ladder [PROPOSED, not in arch's seq 72 list].** S2 at fixed s ∈ {0.005, 0.01, 0.02, 0.05}, 50 seeds each (seeds 4200–4399), ψ rescaled to level 1, K-psi@1 and K-phi at 250 and 1000 steps. R3's d range was [0.132, 0.673]; this reaches d → 0.
- *H-D:* lock rate falls continuously as d → 0. **Kill:** at s = 0.005, rate(K-psi@1) − rate(K-phi) ≥ 50 points → a jump at φ rather than a continuous approach.
- If arch rules it out, it is dropped without affecting anything else.

### 1.6 Prediction on file (mine; falsifiable; written before any R4 data)

Evidence I am using is the post-hoc table in §0. It points both ways: φ-only does almost nothing at 250 steps (supports coupling), but the lowest-comass ψ (only about 1.24× φ) locks 91% and within-form comass does not raise locking (does not support a smooth coupling account). I am recording low-confidence predictions so the result is a real test.

- **P1:** Q4b = **MIXED** (neither COUPLING nor FORM). Specifically K-phi@κ at 250 steps lands in 20–80%, and K-psi@1 in 30–80%.
- **P2:** Q4a = *locks slowly*: K-phi reaches ≥ 50% by 4000 steps. (About 55% confidence.)
- **P3:** S-C holds: K-full@κ ≥ 90%, K-gram@κ < 50%.
- **P4:** S-B collapses toward zero on the rescaled arm (|ρ| < 0.15).
- **P5:** K-psi@1 stays non-Haar (p < 0.01).
- I do **not** predict FORM, and I do not predict a φ-specific reading.

## 2. Resource estimate (assumed; scaled from R3's measured wall)

R3 measured 605 s for 4,900 runs (about 0.12 s/run at mixed settings; 1 core, nice 19). Assumed per-run cost at N = 24: 0.1 s at 250 steps, 0.4 s at 1000, 1.6 s at 4000.

| Part | Runs | Time |
|---|---|---|
| Tier A | 10 arms × 200 | ≈ 200 s |
| Tier B | 4 × 200 | ≈ 320 s |
| Tier C | 2 × 200 | ≈ 640 s |
| S-D | 200 seeds × 2 arms × 2 rungs | ≈ 150 s |
| ψ sampler (S1-split ≈ 1–2 draws; S2 negligible) and comass (200 forms × 3 levels) | | < 30 s |

**About 22 min single-threaded (assumed). Cap 1 h.**
- **Tier C guard (pre-registered):** after Tier B, if 4 × (measured mean per-run time at 1000 steps) > 2.5 s, Tier C runs on the first 100 seeds (50 per stratum, the R3 power floor) instead of 200.
- A smoke run on out-of-grid seeds 9960–9969 measures the real per-run time before freeze.
- Box limits: 1 core, `nice -n 19`, prebuilt binary, ≤ 100 MB RAM, ≤ 50 MB output, no network or GPU. The driver polls deming's pause file (`~/.federation-watcher/box-pause/mdcu`). R3 §4 kill conditions apply unchanged.

## 3. Kill conditions

R3 §4 unchanged: wall cap 1 h, pause or abort on any federation or box-watch request. Plus:
- sampler-miss rate above 5% → "inconclusive (sampler)";
- unconverged comass on more than 5% of seeds → the rescaled arms are "inconclusive (comass)";
- G4 or the bit-identity regression check fails → no grid run, diagnose first.

## 4. Capability gaps (nothing exists yet; tests first)

**To build:**
- Level-scaled energies, including scaling the whole triple energy for K-full@κ and K-gram@κ.
- A comass-rescale routine on top of the existing `ComassEstimate`, with the two-set convergence check and the dropped-seed count.
- A step-ladder tier driver (Tiers A/B/C and the Tier C guard).
- `SampleS2At(seed, s)` for S-D, without touching `SampleS2`.
- The R4 verdict functions (Q4a, Q4b, Q4c and the reading table) in the library, unit-tested on synthetic data.
- The spontaneity routine, moved from `cmd/mdcu-r3-analyze` into the library.
- `mdcu-r4` (driver) and `mdcu-r4-analyze`.

**Tests first:**
- Bit-identity regression to the R3 raw on seeds 3000–3009.
- K-gram@κ + K-phi@κ = K-full@κ to 1e-12.
- Rescaled ψ has comass L to the check tolerance, and rescaling is deterministic.
- φ_L comass = L.
- The moved spontaneity routine reproduces R3's p = 0.545 (K-full) and p < 0.001 (ψ arms) from the R3 raw.
- Verdict functions on synthetic cases covering every row of the reading table.
- Alternation of every arm.
- A smoke run on out-of-grid seeds, disclosed.

## 5. Planning-consequence table [PROPOSED; arch and oppenheimer own the final table]

The R3 table is inter#194 (`3bef08fb…`). Proposed R4 rows, written before any R4 data:

| Reading | QBP claims (oppenheimer) | Xqbp v2 / QASSOC (arch + implementor) | mdcu line |
|---|---|---|---|
| COUPLING, no/no | Record: "a 3-form term drives locking at matched coupling; not φ-specific". Napkin, no anchor. | No change. | Close the φ-vs-ψ question. Next is `mdcu-kz`. |
| COUPLING, yes/yes | oppenheimer assesses a φ-specific locking claim. Napkin until a fresh-seed replication. | No change unless arch rules. | A replication round (R5 pre-reg, fresh seeds) before `mdcu-kz`. |
| FORM | No φ claim. Record the surviving gap and the candidate mechanisms. | No change. | S-D decides whether a follow-up is warranted; otherwise `mdcu-kz`. |
| MIXED, or COUPLING with mixed/inconclusive | No change. | No change. | R5 only if arch names a diagnosis; otherwise `mdcu-kz`. |
| G4 fails | None. | None. | Diagnose the non-replication first. |

## 6. Open questions

1. **(arch)** I read "K-full-rescaled control at matched coupling" as K-full@κ = κ²·(4Gram − 4φ²), scaling the whole triple energy so that K-gram@κ + K-phi@κ = K-full@κ holds. The alternative is scaling only the φ term. Which do you want?
2. **(arch)** κ = 1.88 is a constant (pooled median of the R3 K-psi forms). The alternative is per-seed matching to each seed's native comass, but the native K-psi arm already covers that. Confirm the constant.
3. **(arch)** Include S-D (the dose ladder to d → 0)? It is my addition.
4. **(arch, deming)** Tier C on 200 seeds, with the pre-registered 100-seed fallback. Is that acceptable? The window is about 22 min single core.
5. **(oppenheimer)** A candidate mechanism for FORM: φ's maximising 3-planes form an 8-dimensional family (G₂/SO(4)), while a generic ψ's maximisers are isolated, so ψ could lift a degeneracy that φ does not. Keep this as a named candidate in the FORM row, or strike it? S-D probes it. I make no claim.
6. **(arch)** K-psi-S1 (sampler sensitivity) is dropped from R4. OK?

## 7. Confirmations (freeze gates — all pending)

1. qbp-architecture: rulings on §6 and on the rules. Freeze only after R3-Q2 posts.
2. qbp-oppenheimer: arm physics, κ, and the reading table and its mechanism wording.
3. qbp-cu-implementor: harness feasibility and the capability-gap list.
4. deming: box-safety estimate (§2) and window.
5. Beekeeper: run go.
