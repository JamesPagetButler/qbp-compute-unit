# Spike MDCU — round 4 pre-registration (DRAFT r3)

Status: **DRAFT r3 (2026-10-26). NOT FROZEN. Nothing built, nothing run.** Drafted by antigravity, at the beekeeper's instruction ("draft the R4 pre-registration, but don't freeze it until the R3 report has been read"). r3 = r2 (commit `2d550e4`, kept unedited as the record) with §5 replaced by a pointer to arch's final planning-consequence table (seq 83) and the §7 gates updated. r1 (`6a1dc76`) is also kept. Nothing else changed since r2; the prediction on file (§1.6) is untouched.
Freeze order (arch seq 78): **R3-Q2 posts → Q5 ruled (done, seq 77) → arch writes the final planning-consequence table → oppenheimer's physics confirm → FROZEN-v1 with a sha**, then deming's box-safety gate and the beekeeper's run go. No rule may change after any R4 data is seen. Branch B, spike/napkin grade.
Baseline: [`../mdcu-round3-results.md`](../mdcu-round3-results.md) (R3-Q1 read and independently verified by arch, seq 71), the frozen R3 pre-registration [`mdcu-round3-prereg-FROZEN-v1.md`](mdcu-round3-prereg-FROZEN-v1.md) (sha256 `070aeab8…1848`), and the post-hoc diagnostic [`../raw/mdcu-r3-comass-posthoc-2026-10-26.md`](../raw/mdcu-r3-comass-posthoc-2026-10-26.md).
Rulings applied: qbp-architecture seq 71, 72, 78, 80, 83; qbp-oppenheimer seq 56, 69, 77; deming seq 79. The §5 planning-consequence table is arch's, on record (seq 83), and is referenced, not restated.

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

As R3 §1.3: eigenvalues stored, s₃ and λ₃/Σλ, **3-lock** ⇔ s₃ ≥ 0.95 ∧ λ₃/Σλ ≥ 0.15, **2-collapse** ⇔ s₂ ≥ 0.95, else diffuse. Additionally, per seed: **each native ψ's own comass** (so a per-form reanalysis remains possible as a disclosed secondary; arch seq 78 Q2), the rescale factors, the two-set convergence difference, and (even seeds) s and d.

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

**Mechanism reading (oppenheimer seq 77, pre-registered as part of the Q4 reading; no claim).** The candidate mechanism is geometric: φ's maximisers on Euclidean-unit triples are the associative 3-planes, the Grassmannian G₂/SO(4) of dimension 8, a connected degenerate family (φ is a calibration), so φ's landscape is flat; for a generic stable ψ the Euclidean maximisers are isolated, so ψ's landscape is funnelled. It predicts slow or no locking and Haar-G₂-uniform planes for φ, and fast locking to few planes (non-Haar) for ψ. Two kills, decided mechanically:
- **Kill FORM** iff rate(K-phi@κ) is equivalent to rate(K-psi@κ) **and** K-phi@κ's locked planes are non-Haar (p < 0.01, at least 20 locked runs). Then the gap was coupling.
- **Kill COUPLING-ONLY** iff rate(K-phi@κ) lags rate(K-psi@1) (not equivalent, and lower), **or** K-phi@κ locks with Haar-uniform planes (p ≥ 0.05, at least 20 locked runs) while K-psi@1 locks with clustered planes (p < 0.01).
- If neither kill fires, the mechanism reading is "undetermined". The spontaneity secondary (S-A below) is the direct instrument; H-A3 below is the same prediction for φ.

### 1.5 Secondaries (each carries its own hypothesis and kill; never a decision input)

**S-A Spontaneity** (Haar-G₂ KS, the R3 analyzer's definition; arch seq 72). Computed per arm with at least 20 three-locked runs; otherwise "untestable".
- *H-A1:* K-full's planes are Haar-G₂-consistent (replicates R3 p = 0.545). **Kill:** p < 0.01.
- *H-A2:* every ψ arm (K-psi, K-psi@1, K-psi@κ) differs from Haar-G₂ (p < 0.01). **Kill:** any ψ arm with ≥ 20 locked runs has p ≥ 0.05 → the R3 non-Haar split is not robust to rescaling.
- *H-A3:* any φ arm that locks in ≥ 20 runs (K-phi@κ, or K-phi at a later rung) is Haar-G₂-consistent (p ≥ 0.05). **Kill:** p < 0.01.

**S-B Dose-response on the rescaled arm** (arch seq 72, oppenheimer seq 69 (d)). Definite seeds. ρ = Spearman of the paired s₃(K-phi@L) − s₃(K-psi@L) against d, at L = κ and L = 1 (250 steps), with a 10,000-resample bootstrap CI. R3: ρ = −0.258 raw, partial on comass −0.108 (post-hoc).
- *H-B:* the dependence on d survives rescaling: |ρ| ≥ 0.20, CI excludes 0, same sign as R3. If so, it is about g_ψ vs Euclidean, not coupling strength.
- **Kill:** CI includes 0 and |ρ| < 0.10 → the R3 dose-response was carried by comass.

**S-C Finding (a) at matched coupling** ("neither term locks alone; together they do"). *Holds* iff rate(K-full@κ) ≥ 90% ∧ rate(K-gram@κ) < 50% ∧ rate(K-phi@κ) < 50%. Otherwise reported as "(a) fails at matched coupling: a single term locks".

**S-D Dose ladder (included, arch seq 78 Q3, at normalised comass only).** S2 at fixed s ∈ {0.005, 0.01, 0.02, 0.05}, 50 seeds each (seeds 4200–4399), ψ rescaled to level 1, K-psi@1 and K-phi at 250 and 1000 steps. R3's d range was [0.132, 0.673]; this reaches d → 0. On native ψ, d and comass are 0.97 collinear, so only at comass = 1 does the ladder isolate d alone, the "g_ψ vs Euclidean" dose-response. If locking falls toward K-phi as d → 0 at matched comass, that points to the metric, not coupling strength.
- *H-D:* lock rate falls continuously as d → 0. **Kill:** at s = 0.005, rate(K-psi@1) − rate(K-phi) ≥ 50 points → a jump at φ rather than a continuous approach.
- S-D carries no decision weight.

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
- **Tier C guard (pre-registered; arch seq 78 Q4):** decided by a **timing probe on out-of-grid seeds 9960–9969 before the grid runs, never by grid outcomes**, and disclosed. The probe times K-phi at 1000 steps. If 4 × (probe mean per-run time) > 2.5 s, Tier C runs on the first 100 seeds (50 per stratum, the R3 power floor) instead of 200. The probe result and the decision are posted before the grid starts. This is resource-anchored, not discretionary (deming seq 79): K = 10 probe seeds, trigger = measured wall time extrapolated past the cap, neither looks at any outcome. Power (arch seq 80): 200 seeds primary; 100 is acceptable (the R3 floor of 50 per stratum); **below 100, R4 verdicts read "inconclusive (power)"**.
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

## 5. Planning-consequence table (arch's, on record before any R4 data)

The final table is qbp-architecture's, posted pre-data at **inter#194 issuecomment-6431920143, sha256 `90b870d3f9f50c2a666d54c92ea18b386fcf9f7f7d2342a18f5f15a034ae6cec`** (arch seq 83). It replaces the placeholder table that r1 and r2 carried here, and builds on those rows. It adds Q4a rows, mechanism-kill rows (COUPLING-ONLY killed goes in as a napkin design input to QBP#776 rung 2) and compute-side consequences. **No outcome changes a Sprint 4 build, and no CD speed claim is permitted at any reading.** This document does not restate the table; the comment is the authority. The R3 table remains inter#194 (`3bef08fb…`).

## 6. Rulings on r1's open questions, and changes since r1

| # | Question (r1) | Ruling | Change in r2 |
|---|---|---|---|
| Q1 | Scale the whole triple energy for "K-full-rescaled"? | **Yes** (arch seq 78; oppenheimer seq 77 agrees). K-full = K-gram + K-phi at every level. Whole-scaling also shifts the triple-vs-neighbour balance (μ = 1.5); that shift **is** the coupling-strength confound under test, so it is intended. | Wording kept; the intended μ-balance shift is stated here. |
| Q2 | κ as a constant 1.88? | **Confirmed** (arch). Record each native ψ's own comass per seed so a per-form reanalysis stays possible as a disclosed secondary. | §1.2 records native comass; the per-form reanalysis is a disclosed secondary. |
| Q3 | Include S-D? | **Yes, at normalised comass only** (arch): on native ψ the ladder would re-measure comass. | S-D header; it runs K-psi@1 only. |
| Q4 | Tier C on 200 seeds with a 100-seed fallback? | **Approved, with a condition** (arch): the guard is decided by a timing probe on out-of-grid seeds before the grid, never by grid outcomes, and disclosed. deming confirms box-safety on the frozen estimate. **Capacity** (deming seq 79): 200 seeds approved, ~22 min on one core; the guard must be a pre-registered, resource-anchored trigger (K and the extrapolation rule stated). **Power** (arch seq 80): 200 primary, 100 acceptable, below 100 is "inconclusive (power)". R4 runs in its CPU slot after R3 fully closes (R3-Q2 plus the record PR). | §2 guard rewritten as a pre-grid probe (K = 10, extrapolation rule); the power floor is stated. |
| Q5 | Keep the named FORM mechanism? | **Keep, as a hypothesis with its kill, no claim** (oppenheimer seq 77). Pre-register the two kill sentences as part of the Q4 reading. | "Mechanism reading" block added before §1.5. |
| Q6 | Drop K-psi-S1? | **Approved** (arch). R3 already records the S1 sampler sensitivity (67% vs S2 79%). | None. |

The prediction on file (§1.6) is unchanged from r1; it was written before any R4 data and before these rulings.

## 7. Confirmations (freeze gates)

1. qbp-architecture: rulings on §6 — **done (seq 78, 80)**. Final planning-consequence table — **done (seq 83)**, referenced in §5. Freeze only after R3-Q2 posts (R3-Q2 raw and result are now posted by me; arch to acknowledge).
2. qbp-oppenheimer: Q5 — **done (seq 77)**. Physics confirm at freeze, after R3-Q2 (his seq 77).
3. qbp-cu-implementor: harness feasibility check (resource estimate vs the ~22 min / 1 h cap and the Tier C guard, gate tests green, no spec-vs-code drift) and the capability-gap list — pending; queued behind R3 fully closing (seq 82).
4. deming: capacity — **200 seeds approved (seq 79)**; formal box-safety clearance at freeze, in the CPU slot after R3 fully closes — pending.
5. Beekeeper: run go — pending.
