# Spike MDCU (Xqbp v2) — round 2 final report

Status: **FINAL, frozen 2026-10-24** (sha256 posted on mdcu-spike). Branch B, spike/napkin grade.
Branch `spike/mdcu-xqbp-v2` on origin at `b280425` (pushed by qbp-cu-implementor, mdcu-spike seq 28).
Round-1 verdicts: [`mdcu-xqbp-v2-spike.md`](mdcu-xqbp-v2-spike.md). Record tag `spike-record/mdcu-xqbp-v2-aa45861` untouched.
No physics claims. Model cycles are an advantage under a symmetrically-applied ISA cost model, **not silicon**.

## 1. Verdicts of record

| Question | Verdict of record | Accepted |
|---|---|---|
| **Q1** Does Test B frame locking survive a controlled repro against explicit nulls? | **INCONCLUSIVE** | qbp-architecture seq 17 |
| **Q2** Is there a workload where the associator is load-bearing on which the MDCU beats the best equally-informed baseline? | **INCONCLUSIVE (B1-variant-sensitive)** | qbp-architecture seq 27 |

**Bottom line:** neither question shows an MDCU advantage under the pre-registered rules.

### Q1 (plan v3 sha256 `28bcaa6f…cae374`; results [`mdcu-round2-q1-results.md`](mdcu-round2-q1-results.md))
- 10,800 runs, wall 1,360 s, nothing skipped. Raw sha256 `a087bc69…5b59c`.
- C1 pass (overlap SVD-3 ≥ 0.95 in 99.5%); C2 **fail** (null 3 95th pct 0.991); C3 **fail** (N=48 persistence 48.5%); C4 **fail** (gap vs null 3 = 0.090, floor 0.15); C5 pass (spontaneity p = 0.557).
- **C2 reading:** the frozen text covers nulls 1–3, so C2 is a fail (arch seq 17). The draft's alternative reading (null 3 excluded) would pass C2; the verdict is inconclusive either way.
- **What it shows:** the overlap coupling locks robustly and spontaneously, with no preferred frame, but a norm-matched random alternating map locks almost as well under SVD-3. Locking is **not shown to be octonion-specific**.

### Q2 (amendment v3a sha256 `38989a7f…d2d4e9`; results [`mdcu-round2-q2-results.md`](mdcu-round2-q2-results.md))
- Albert determinant; wall 34.3 s; agreement gate 5.81e-16. Raw sha256 `b577fd6b…e97e9`.
- MACs/det (by construction, not evidence): B1lit 157, B1lean 101, B0a 157, B0b 118.
- Host ns/det (measured): B1lit 276.0, B1lean 167.6, B0a 237.7, **B0b 94.4**.
- Model cycles ideal/dep at batch 64, P=8: B1lit 20/41, B1lean 13/41, B0a 20/41, B0b 15/41.
- Primary (B1lit vs B0b): **no**. Sensitivity (B1lean vs B0b): ideal-bound ordering flips → **inconclusive (B1-variant-sensitive)**. The literal draft reading ("any column", incl. MACs) gives the same verdict.
- Yes rule (≤ 0.5× under both bounds) not met by either B1 variant; best measured ratio 0.87.
- **Headline:** the fastest path on host is the Fano-factored non-CD path. It wins by structure (the Fano-line factorisation), not by a different multiplier.

## 2. Limitations (all)

1. **Q1 measurement:** SVD-3 ≥ 0.95 does not by itself exclude a collapse onto a 2-plane (the eigenvalue spectrum was not stored). For the overlap kind this is ruled out by Phi0Frac = 1.00 (Gram–Schmidt succeeds on every consecutive triple); for null 3 it is not checked. *(Found post-round; see §3.)*
2. Q1 seeds are Go PCG, not numpy; Test B was ported from prose; the disjoint layout uses the true plaquette gradient (plan §8b).
3. Q1 at N=48: 250 fixed steps; convergence vs multi-plane splitting untested.
4. Q1 Monte-Carlo p resolution 0.001 (1,000 replicates).
5. φ₀ measures are tautological for associator energies (proved in §3), so the φ₀ contrast in Q1 is not evidence of selection.
6. Q2 MAC column is fixed by construction; the ideal bound only tracks it.
7. **Q2 dependency bound is non-discriminating** under the agreed one-tile-cycle-per-node model: every variant has 5 nodes per determinant, so batch 64 at P=8 is node-count bound (41) for all four.
8. Q2 B0b was not the cheapest non-CD path: a 7D cross-product form needs about 104 multiplies per determinant (hand count, post-round; not run). It does not change the verdict: it is on the B0 side, and the yes rule was unreachable by either B1 variant.
9. Host ns: FX-8350 without AVX2 serialises CD-MAC's 64-way parallelism (a bias against B1 in host ns); background load 3.3–4.4 during runs (Nugent's spike).
10. Model cycles have no data-movement or operand-routing cost, where a CD-MAC unit would claim its advantage.

## 3. Post-round analysis (algebra only; no dynamics run; not part of any verdict)

1. **Associator identity.** For imaginary a, b, c: ‖[a,b,c]‖² = 4(Gram₃(a,b,c) − φ(a,b,c)²), φ = Re((ab)c). Checked numerically to 1.5e-13 (3,000 triples). The associator energy is an O(7)-invariant Gram term minus the G₂ term.
2. **φ₀ tautology, now exact.** On orthonormal triples Gram₃ = 1, so ‖[a,b,c]‖² → 0 ⇔ |φ₀| → 1. The Q1 φ₀ contrast is the energy restated.
3. **Null 3 is "Gram plus noise".** Λ³ℝ⁷ is O(7)-irreducible, so a Frobenius-matched Gaussian alternating map has E‖T(a,b,c)‖² = (16/5)·Gram₃, the same as the SO(7)-orbit mean of the associator energy. Checked: 3.21 (400 null draws), 3.19 (orbit mean), predicted 3.20; per-triple sd 1.7. **Hypothesis (untested):** Q1's SVD-3 concentration is driven by the Gram part shared by the associator and null 3.
4. **Multiply counts (hand counts, to be confirmed by tape):** for the scalar Re((xy)z) the CD path plus a dot (B1lean, 72 for the Re part) narrowly beats the best non-CD form found (cross-product form, about 75). For the full associator vector the Hodge-dual 4-form contraction needs about 126 multiplies (21 minors × 2 + 28 triples × 3) vs 256 for four OMULs. In any lane-packed model, model cycles tend to multiplies / 64P, so no 2× model-cycle advantage for CD is reachable on these workloads.

## 4. Capability gaps (per spike rules)

- `sched.go` takes only level lists; no arbitrary-DAG scheduler, no cross-instance node packing (Q2 used two fallback bounds).
- No data-movement / routing cost in the tile model.
- Q1 harness: no signed-energy runner (energies must be ‖T‖²); no scalar 3-form energies; eigenvalue spectrum not stored; steps fixed at 250.
- No width or vector-tier dispatch in `CPU.Exec` (functional only); ZDCHK is a class check (rename `ZDCLASSCHK` pending, beekeeper's call).
- Leaf engine applies only with the plane-rotation symmetry.
- No hardware or silicon numbers; FX-8350 has no AVX2; QW256 is a fixed-width reference.
- No Lean proofs of the identities in §3 (Gram lemma, associator identity, null-3 expectation).
- Federation: deming's and qbp-oppenheimer's wake-on-mention covered only live-test; antigravity had no live waker and its `--mine` scan covered only live-test (fixed 2026-10-24); no durable per-persona inbox with cursor in use yet.

## 5. Round-3 candidates (logged; round 3 design is in `prereg/mdcu-round3-prereg-DRAFT.md`)

1. φ-only vs full associator energy (qbp-oppenheimer seq 21), refined by §3.1 into Gram-only / φ-only / full / random-ψ arms.
2. Best-of-each-side Q2 (arch seq 19) under cross-instance node packing (this report, limitation 7) — see §3.4: a 2× model-cycle yes is unreachable.
3. N=48 convergence (arch seq 17), diagnostic.
4. φ₀-primary (arch seq 17) — **dropped**: tautological by §3.2.

Fresh seeds, excluding 1000–1199, 9000, Q2 seeds 77–83 and both round-2 grids.

## 6. Record

| Item | Commit / sha256 |
|---|---|
| Plan v3 frozen | `98eeb1e` / `28bcaa6f72db9168d1908bbc17709db109761f0898b06bc8ea87809423cae374` |
| Q1 results + raw | `2dca440` / raw `a087bc6966045859ec49ca86df58c5b1ff598ed793ddf5057bec2905b8f5b59c` |
| Q2 harness | `0c37571` |
| Q2 amendment v3a frozen | `68e51ba` / `38989a7f721098343b5abbeab6aae370448162af753c9e36176a0d9140d2d4e9` |
| Q2 results + raw | `b280425` / raw `b577fd6b90a7893c35bffced02faec2cfa8c585a937ebe32724053cb5efe97e9` |

Gates for every run: frozen plan → qbp-architecture sign-off → deming box-safety → beekeeper go. All runs stayed within the kill conditions.
