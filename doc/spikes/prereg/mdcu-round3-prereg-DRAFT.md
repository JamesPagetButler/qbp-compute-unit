# Spike MDCU — round 3 pre-registration (DRAFT for qbp-architecture review)

Status: **DRAFT. Nothing built, nothing run.** Branch B, spike/napkin grade. Drafted by antigravity; qbp-oppenheimer invited to co-draft (add / drop / swap).
Baseline: `doc/spikes/mdcu-round2-final-report.md` (frozen; sha on mdcu-spike). Direction: arch seq 29 (beekeeper, 2026-10-24).
Gates: draft review (arch) → freeze with sha → deming box-safety → beekeeper run go.

## 0. What changed from arch's proposed question set, and why

Arch seq 29 proposed R3-Q1 = φ-only null vs full associator, and R3-Q2 = best-of-each side under cross-instance node packing. Two algebra facts (round-2 final report §3; checked numerically, no dynamics run) change both:

1. **‖[a,b,c]‖² = 4(Gram₃ − φ²)** for imaginary a, b, c. A φ-only energy has the *same* G₂ content as the full associator energy; they differ only by the O(7)-invariant Gram₃ term. So "φ-only vs full" tests whether the Gram term matters, not whether G₂ matters. **Proposal:** add a Gram-only arm (no G₂ at all) and a random-3-form arm (ψ in place of φ), so each comparison changes exactly one ingredient.
2. **R3-Q2's yes rule is unreachable by construction.** In any lane-packed model, cycles tend to ceil(multiplies / 64P), so the B1lean / best-B0 ratio tends to the multiply ratio: 101/118 = 0.86 vs B0b, about 101/104 = 0.97 vs the cheaper cross-product form. A 0.5× yes cannot occur. **Proposal:** swap R3-Q2 (§3).
3. **φ₀-primary stays dropped:** on orthonormal triples, energy → 0 is the same statement as |φ₀| → 1 (exact).

## 1. R3-Q1 — the structure question (primary)

**Question.** Which ingredient of the associator energy drives the ring's 3-plane locking: the O(7)-invariant Gram term, the G₂ 3-form term, or the specific octonionic φ (vs any 3-form)?

**Dynamics.** Round-2 Test B ring, overlap layout (centre + (i−2,i−1,i) terms), same optimiser: lr 0.04, μ 1.5 neighbour term, 250 steps, unit imaginary 8-vectors, Euclidean metric. Only the triple energy E(a,b,c) changes. A new signed-energy runner is required (§5); it must be **bit-identical to round-2 `RunRingGen`** for K-full on a fixed seed (test gate).

**Arms (same init per seed across arms; paired design):**

| Arm | E(a,b,c) | G₂ content | Purpose |
|---|---|---|---|
| K-full | ‖[a,b,c]‖² = 4Gram₃ − 4φ² | φ | round-2 overlap; replication on fresh seeds |
| K-gram | 4Gram₃ = 4‖a∧b∧c‖² | none (O(7)) | is the Gram term alone enough? |
| K-phi | −4φ² | φ | is the G₂ term alone enough? (oppenheimer's null) |
| K-psigram | 4Gram₃ − 4ψ² | ψ (random) | does φ specifically matter, vs any 3-form? |
| K-psi | −4ψ² | ψ (random) | ψ analogue of K-phi |
| K-rand3 | round-2 null 3 (random alternating vector-valued, Frobenius-matched) | none in expectation | carried reference; E = (16/5)Gram₃ + noise |
| K-none | μ term only | — | sanity |

ψ: Gaussian scalar 3-form on ℝ⁷, Frobenius-matched to φ (Σ_{abc} ψ²_{abc} = Σ φ²_{abc} = 42). Generic 3-forms on ℝ⁷ lie in one of the two open GL(7) orbits (stabiliser compact G₂ or split G₂; Bryant / Hitchin). Each draw is **classified** by the signature of B_ψ(v,w)·vol = (1/6)(ι_vψ)∧(ι_wψ)∧ψ: definite ⇒ "G₂-type in another metric", indefinite ⇒ "split". A test asserts B_φ is definite. Results for ψ arms are **stratified** by type.

**Measurements per run (all stored; eigenvalues stored this time, closing round-2 limitation 1):**
- λ₁ ≥ … ≥ λ₇: eigenvalues of the 7×7 Gram of Z[:,1:]; SVD-k share s_k = Σ_{i≤k} λ_i / Σλ.
- **3-lock** ⇔ s₃ ≥ 0.95 **and** λ₃/Σλ ≥ 0.15 (genuinely 3-dimensional). **2-collapse** ⇔ s₂ ≥ 0.95. Otherwise diffuse.
  *Why:* 2-plane alternating configurations (z_{i+1} = ±z_{i−1}, neighbours orthogonal) are exact zeros of E_full and E_gram with μ = 0, so s₃ alone cannot tell a 3-plane lock from a 2-plane collapse.
- Plane calibration (secondary, diagnostic): |φ₀| and |ψ₀| of the top-3 plane, and the fraction of consecutive triples with |φ₀| ≥ 0.95 (and the same for ψ). Own-form calibration is tautological (§0.3); **cross-calibration** (φ₀ under ψ arms, ψ₀ under φ arms) is the informative diagnostic.
- Spontaneity KS vs the Haar-G₂ reference (round-2 definition), for any arm that 3-locks: secondary.

**Grid.** Main cell N = 24, σ = 0.1, **200 fresh seeds 3000–3199** (excluding 1000–1199, 9000, 77–83 and both round-2 grids). ψ and rand3 tensors per seed from distinct PCG streams (constants differ from round 2). Persistence cells N = 12 and N = 48 at σ = 0.1. σ ∈ {0, 0.3} dropped (unused by any round-2 decision).

**Decision rules (main cell; paired by seed; Δ(X,Y) = median paired s₃(X) − s₃(Y), bootstrap 95% CI, 10,000 resamples):**
- **Replication gate:** K-full 3-locks in ≥ 90% of seeds (round-2 C1). If not, every R3-Q1 sub-verdict is "inconclusive (non-replication)".
- **Q1a — Gram suffices?** *yes* iff K-gram 3-locks in ≥ 90% **and** |3-lock rate(K-full) − 3-lock rate(K-gram)| ≤ 10 points **and** the CI of Δ(full, gram) lies inside ±0.05; *no* iff K-gram 3-locks in < 50% **or** Δ(full, gram) ≥ 0.15 with CI excluding 0; else inconclusive.
- **Q1b — G₂ term suffices?** same rule with K-phi in place of K-gram.
- **Q1c — φ specific?** same rule comparing K-full with K-psigram, separately for the definite and split strata. *yes (φ not specific: any 3-form of that type suffices)* iff the equivalence condition holds; *no (φ specific)* iff the 0.15 gap holds; else inconclusive. A stratum with < 20 seeds is "inconclusive (underpowered)".
- **Reading table (pre-registered, so the interpretation is not chosen after the fact):**

| Q1a | Q1b | Reading |
|---|---|---|
| yes | — | locking is an O(7) Gram effect; G₂ not needed for SVD-3 locking |
| no | yes | the G₂ 3-form term drives locking |
| no | no | both terms needed (the full associator) |
| other | | inconclusive |

**Secondaries (diagnostic only, never a decision input):** N = 48 convergence, K-full and K-gram at steps ∈ {250, 1000}; 2-collapse rates for every arm; cross-calibration; K-rand3 3-lock vs 2-collapse (tests round-2 hypothesis §3.3 descriptively).

**Prediction on file (falsifiable, mine):** K-gram and K-rand3 mostly 2-collapse rather than 3-lock; K-full and K-psigram (definite stratum) 3-lock at similar rates; hence Q1a = no, Q1c = yes (φ not specific). Recorded so the result is a real test, not a hoped-for outcome.

## 2. Resource estimate (assumed, scaled from round-2 measurements)

Round 2: one seed of 6 kinds at N = 12 took 0.33 s (rand3 0.18 s, overlap 0.05 s). Scalar-3-form arms cost about as much as overlap; K-gram about 2× (35-component wedge). Main cell: 200 seeds × 7 arms ≈ 200 × 0.9 s ≈ 3 min; N = 12 and N = 48 persistence ≈ 2 + 6 min; N = 48 at 1,000 steps for 2 arms ≈ 6 min. **About 20 min single-threaded (assumed).** Cap 1 h. 1 core, `nice -n 19`, prebuilt binary, ≤ 100 MB RAM, ≤ 50 MB output, no network/GPU. Round-2 §10 kill conditions apply unchanged.

## 3. R3-Q2 — the efficiency question: proposed swap

Arch's R3-Q2 (B1lean vs best-B0 under cross-instance node packing) cannot yield yes (§0.2). qbp-cu-implementor (seq 26) also warns that a packing refinement can manufacture a model-cycle advantage the measured host ns already denies, and asks that host ns be the tiebreak. Options for arch:

- **(A) Close by analysis (recommended).** A written table, with multiply counts **computed by tape in code** and disclosed as by-construction, for three G₂ workloads: the Albert determinant scalar, φ alone, and the full associator vector. Best known CD path vs best known non-CD path for each. Hand counts so far: Re((xy)z) part CD+dot 72 vs cross-product form ≈ 75 (CD narrowly wins); associator vector 4 × OMUL = 256 vs Hodge-dual 4-form contraction ≈ 126 (non-CD ≈ 2× fewer). No run; no verdict beyond "no 2× multiply-count advantage for CD exists on these workloads".
- **(B) (A) plus an idle-box host-ns benchstat** of the best path on each side per workload (load < 1 at start; `-count=6`; 10 min cap). Measured, but biased against CD (no AVX2; CD-MAC parallelism serialised). Decision: *yes* iff the best CD path is ≥ 2× faster in ns on an admitted workload; *no* iff the best non-CD path is ≤ the best CD path on all three; else inconclusive.
- **(C) A routing / data-movement cost model** (where a CD-MAC unit would actually claim its advantage: fixed XOR permutation and sign ROM vs gathers for Fano-line terms). High tilt risk; would need its parameters fixed from an external source and signed off by qbp-cu-implementor **before any count is computed**. Not recommended for round 3.

The cross-instance node packing tile model is still worth building as a scheduler capability (it closes round-2 limitation 7), but not as a decision instrument.

## 4. Kill conditions

Round-2 §10 unchanged, with the wall cap set to 1 h for R3-Q1 and 10 min for R3-Q2 (B). Pause or abort on any request from the federation or deming's box-watch.

## 5. Capability-gap expectations (what must be built; nothing exists yet)

- Signed-energy ring runner (energy + gradient callback), bit-identical to `RunRingGen` for K-full.
- Scalar 3-form type (φ, random ψ), wedge map for Gram₃ (35 components), B_ψ signature classifier.
- Stored eigenvalue spectrum and 3-lock / 2-collapse classification in the measure struct.
- Steps parameter in the runner.
- (R3-Q2 A/B) tape-instrumented φ-only and associator-vector variants, and the cross-product B0 variant.
- Tests first: runner equivalence, identity ‖[a,b,c]‖² = 4(Gram₃ − φ²) to 1e-12, ψ Frobenius matching, B_φ definite, alternation of every arm, smoke on an out-of-grid seed (disclosed).

## 6. Open questions

1. **@qbp-oppenheimer:** add / drop / swap on the arms? Is the B_ψ signature the right classifier, and is "G₂-type in another metric" the right reading of the definite stratum?
2. **@qbp-architecture:** accept the R3-Q2 swap, and which option (A, B or C)?
3. The new thresholds need sign-off: 3-lock λ₃ share ≥ 0.15; equivalence margin ±0.05 on Δ; 10-point rate margin; stratum minimum 20 seeds.
4. Should K-full also be run with the round-2 seeds as a pure reproduction check (not a decision input)? I lean no (fresh seeds only, as ruled).
