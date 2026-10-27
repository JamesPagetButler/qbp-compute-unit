# mdcu-kz pre-registration, DRAFT r4 (NOT FROZEN, NOT FOR RUN)

**Status:** DRAFT r4, supersedes r3 (r1, r2 and r3 are unedited and stay in the tree). Author: antigravity (runner). Branch B, spike/napkin grade. No physics claim and no CD speed claim at any reading. Nothing here is built or run. r4 applies oppenheimer's seq 158, which answered my r2 questions and **reversed his seq 154 on split-G₂: the split-G₂ control is DEFERRED to a conditional round 2, and round 1 freezes with the saturation reference and the S³ null** (my r3, which included split-G₂, crossed it). It also settles the Stage 0 tolerances, the no-curvature test, the defect-count floor and the S³ adapter, and it strikes the stale "Hessian gates the freeze" text (arch seq 156; r3 had already struck it). The model, band and framing are as in r2/r3 (arch seq 147/150, oppenheimer seq 148/151/158).

**Framing (arch seq 147, oppenheimer seq 148):**
- mdcu-kz is a **QBP-physics probe, not a CU-design probe.** R4 settled the CU design question (a programmable 3-form unit, no φ claim). Outcomes route to QBP#776 and the QBP theory line. No outcome changes a Sprint 4 build, and no CD speed claim is permitted.
- It tests the **topology** of the order-parameter manifold G₂/SO(4) (π₂ = ℤ₂ point defects). It does **not** test the −φ² landscape that R4 and M1 are about; the two are orthogonal. r1 §0 borrowed R4's motivation for the model; that is withdrawn. R4 is only the source of the question "what does a plane order parameter do in space", not of any expectation here.
- What it buys the compute unit, per oppenheimer: the ℤ₂ holonomy detector is the orientation-aware plane-transport calculation that the QBP#776 E6 controls need. The topological contrast (split-G₂, π₂ = 0) is a conditional round 2 (§4).

**Chain and gates:**
1. **The M1 Hessian note does not gate the freeze** (arch seq 150/153, oppenheimer seq 154, superseding arch seq 147 item 1 and oppenheimer seq 151). The only reason for the gate was the K-psi-type control, which is dropped (§4). The Hessian and mdcu-kz touch different objects and run in parallel.
2. Oppenheimer checks this text at the pushed sha (his seq 158 answers were given against the questions as posed, not the file). Model, band, S³ adapter, Stage 0 numbers and the fit rule are settled.
3. Arch writes the planning table on inter#194, pre-data, after oppenheimer settles the open items (arch seq 156), carrying split-G₂ as a named follow-up round (round 2), not a gap. I write no table of my own.
4. Deming's box-safety, asked only after a timing probe exists.
5. Freeze with a sha, pushed, every gate ticked.
6. The **run go is arch's** (mdcu-kz is on the inter#224 list, standing authorization), not the beekeeper's. I accept the runner role conditional on 1–5 and a tests-first harness built, reviewed and pushed by qbp-cu-implementor.

Spike 2 (Branch A) is unaffected and does not start until the beekeeper rules.

## 0. Disposition of the r1 open decisions

| r1 item | Ruling | Source | r4 action |
|---|---|---|---|
| 1 model | Model A. Model B dropped (S²/ℤ hedgehog confound). | opp 148 | §2 |
| 2 exponent band | α ∈ [0.6, 1.0], references 0.75 and 0.875, z = 2. | opp 148 | §5 |
| 3 trivial control | S³ (unit quaternions) as the null; adapter confirmed. | opp 148/158 | §4 |
| 3b K-psi-type control, then split-G₂ | K-psi-type dropped. Split-G₂ **deferred to a conditional round 2** (needs a bounded, SO(4)-invariant energy on a compact slice and an ordering transition shown first). Gr⁺(3,7) is not a π₂ = 0 contrast (π₂ = ℤ₂). Seq 154's "included" is superseded. | opp 158 | §4 |
| 4 order vs Hessian | Hessian **does not gate** the freeze; the two run in parallel. | arch 150/153, opp 154 | header |
| 5 deming | Only after a timing probe. | r1 | unchanged |
| 6 beekeeper | Run go is arch's. Whether to run at all is now arch's and the beekeeper's on the critical-path question. | arch 147 | header |

## 1. Questions

- **Q5a (defect formation):** after a ramp through the ordering transition of a G₂/SO(4)-valued plane field on an L³ lattice (L = 16 provisional, §6), is the ℤ₂ point-defect density n a power law in the ramp time, n ∝ τ_Q^(−α), over at least 1.5 decades of τ_Q, with a fit CI?
- **Q5b (exponent):** is α inside [0.6, 1.0] (§5)?
- **Q5c (topology vs mere disorder):** is the defect density at a given τ_Q different from the S³ null under the same ramp? (The split-G₂ contrast is round 2, §4.)

Q5c is the guard against the obvious failure: a detector that counts disagreement between neighbouring planes in any disordered field would show a KZ-like power law for no topological reason.

## 2. Model (settled: model A)

Site x of an L³ periodic lattice holds an associative plane P(x) = span(u, v, uv), built exactly as `haarG2Plane` does. Energy: the sum over nearest-neighbour bonds of d²(P, Q) = 3 − ‖PᵀQ‖²_F = 3 − tr(Π_P Π_Q) (the existing `planeDist` squared; Π is the 7×7 projector onto the plane). Dynamics: overdamped Stiefel-projected Langevin on (u, v), noise temperature T(t) ramped linearly down through T_c over τ_Q steps. Reuses `planeDist`, `haarG2Plane`, `MulRef`, called but not edited.

**Order parameter for the equilibrium check (main arm; confirmed by oppenheimer seq 158):** M = (1/N) Σ_x Π_{P(x)} − (3/7) I, with S = ‖M‖_F, reported normalised by √(12/7) (the value at full alignment; ≈ 1.309 before normalising). It is G₂-invariant, zero for i.i.d. Haar planes in expectation, invariant under in-plane frame rotation, and does not use the detector.

**Not tested here:** the ring locking dynamics, the −φ² landscape, M1–M3.

## 3. Observables and the detector

Unchanged from r1 §3 (second Stiefel–Whitney class of the tautological rank-3 bundle over each cell boundary; 12 triangles per cell; nearest-rotation edge transport; SU(2) lift nearest the identity; product of the 12 lifts, −1 marks a defect; ambiguity rule: a cell with a triangle holonomy of rotation angle ≥ π is "ambiguous", never a defect; > 5% ambiguous at a rung makes that rung "inconclusive (resolution)").

**Added in r2:**
- **Parity invariant (checked by oppenheimer, seq 158).** On a closed periodic lattice every triangle is shared by two cells with opposite orientation, so with consistent shared-triangle lifts (the lift of the inverse is the inverse of the lift) the product over all cells is Π(g·g⁻¹) = +1 and the number of −1 cells must be **even** whenever no cell is ambiguous. It is the discrete "divergence-free" condition for the ℤ₂ flux. Unit test and runtime check. If it fails on a no-ambiguity field, either the detector or this argument is wrong: that blocks the grid, whichever it is.
- **Defect-count floor (oppenheimer seq 158).** At least **20 counted defects per rung** (Poisson relative error ≲ 22%). Below that the rung is "inconclusive (finite size)", and raising L is preferred to dropping the rung if the timing probe allows. I read "per rung" as the count pooled over the replicates of that rung, since a relative error of 1/√20 applies to a pooled count; oppenheimer to confirm. The slow end of the ladder is the risk: with α ≈ 0.75 over the 32× span n falls by about 13×, and at L = 16 the correlation length may also approach L.

**Observable:** n(τ_Q) = defects per cell, mean over replicates, 10,000-resample bootstrap CI. Secondary: the two-point plane-overlap correlation length ξ(τ_Q).

**Unit tests before any grid run (tests first):** (a) uniform field has zero defects; (b) a smooth rotation field has zero; (c) a planted ℤ₂ hedgehog (w₂ = 1) is found exactly once; (d) a planted pair is found twice; (e) the ambiguous-cell counter fires on a deliberately sharp field; (f) i.i.d. Haar gives the saturation density n_∞ (the τ_Q → 0 reference); (g) reflection and global G₂ rotations leave the count unchanged; (h) the parity invariant on i.i.d. Haar fields, restricted to no-ambiguity cells, plus the S³ adapter's zero-holonomy tests (§4). The detector is a pure function of the plane field, so these need no dynamics.

## 4. Controls

1. **Saturation reference:** i.i.d. Haar-G₂ planes, no ramp, give n_∞ (test (f)).
2. **S³ null (required; oppenheimer seq 148 (3), adapter confirmed seq 158):** the same lattice, same ramp, same code path, with an S³-valued (unit-quaternion) field. π₁ = π₂ = 0, so there are no point defects, and any count is the detector's false-positive rate, which is what a null should measure. S³ carries no tautological rank-3 bundle, so the detector is adapted by the **adjoint frame**, q ↦ Ad_q ∈ SO(3), with the same transport and SU(2) lift. Ad: S³ → SO(3) is the 2:1 cover, so a field Ad_q built from a globally defined q has a global SU(2) lift (q itself), and any −1 from the triangle-lift product is a pure detector false positive (oppenheimer seq 158). π₃(S³) = ℤ textures are not point defects and the ℤ₂ detector rightly ignores them. Oppenheimer's added line, which I include: the null also certifies the detector against SO(3)'s own π₁ = ℤ₂ line defects, which cannot occur for a field lifted from q, so a −1 cannot be "a real line defect" either. The adapter must not share a code path with the main arm in a way that could hide a false positive. **Stage 0 for this arm** needs an order parameter; my proposal is the unit-quaternion magnetisation |(1/N) Σ q| (the field is an O(4) vector with the chordal energy −q·q'), to be confirmed (§9).
3. **Split-G₂/SO(4) contrast: DEFERRED to a conditional round 2 (oppenheimer seq 158; supersedes seq 148/154 and arch 150/153 on this point; arch seq 156 said a deferral becomes a named follow-up row in the planning table).** The coset G₂^split/SO(4) is non-compact and contractible (π₂ = 0), but the chordal energy is unbounded there, so a ramp may cross no ordering transition and "no defects" would be trivial, not discriminating. Round 2 is conditional on a bounded, SO(4)-invariant energy on a compact slice of the coset, and its own pre-registration must first show that an ordering transition exists. The obvious compact alternative fails: the oriented Grassmannian Gr⁺(3,7) = SO(7)/(SO(3)×SO(4)) has π₂ = ℤ₂ as well, so it is not a π₂ = 0 contrast. **Nothing for split-G₂ is built or run in round 1.** Round 1 therefore cannot say whether the defect count is *topological* beyond what the S³ null (detector false-positive rate) shows; that limit is stated here so it is not read into the result.
4. **K-psi-type control: dropped** (oppenheimer seq 148). If M1 holds, a ψ-plane field has no coset manifold and the control is ill-defined as a sigma model. Any later return of such a control would be a new pre-registration, not an edit of this one.

## 5. Decision rules (skeleton; numeric placeholders are marked and set pre-data)

**Stage 0: continuous-transition check (blocks the τ_Q sweep of the arm it is run on).** A zero-ramp equilibrium sweep of S(T) on L³ (cool and heat, identical settings, no detector call) before any sweep, **run separately for the main arm and for the S³ arm**. **Continuity criterion, fixed pre-data (oppenheimer seq 158):** at the slowest ramp, S(T) heating and cooling differ by ≤ 5% of S_max, and no single T-step jump exceeds 10% of S_max on the finest grid; otherwise "first-order or glassy, premise fails", recorded. A failure on the main arm ends the round as a clean negative, not a rescue case. A failure on the S³ arm makes the null uninformative, so Q5c is "inconclusive (null)". Stage 0 fixes T_c and the ramp window [T_hi, T_lo] for each arm, and it is outcome-blind with respect to defect counts because it never calls the detector. S is normalised by √(12/7) for the main arm; S_max for the S³ arm is the saturation value of its own order parameter.

**Stage 1: the ladder.** τ_Q ∈ {25, 50, 100, 200, 400, 800, 1600} (7 rungs, about 1.8 decades, so that dropping one end rung still leaves ≥ 1.5 decades; revised by the timing probe).
- **Q5a yes** iff, with log n fitted against log τ_Q both linearly and quadratically, the **quadratic coefficient's 95% CI includes 0 and ΔAIC(quadratic − linear) > −2** (oppenheimer seq 158, the no-curvature test); α and its CI come from the linear fit over at least 1.5 decades of usable rungs, with the CI on α excluding 0; and each used rung has fewer than 5% ambiguous cells and at least 20 counted defects (§3). If dropped rungs leave under 1.5 decades: "inconclusive (span)".
- **Q5b:** α ∈ [0.6, 1.0]. Named reference points α = 0.75 (ν = 1/2, z = 2) and α = 0.875 (ν ≈ 0.7, z = 2), from α = 3ν/(1 + νz) with n ∝ ξ⁻³ and ξ ∝ τ_Q^(ν/(1+νz)). Outside the band means: not KZ, or z ≠ 2, or a ramp not crossing a continuous transition. The band and its reasoning are oppenheimer's; I do not endorse them as physics.
- **Q5c yes** iff n(τ_Q) in the main arm exceeds the S³ null's n with non-overlapping bootstrap CIs at every used rung. Because the split-G₂ contrast is deferred, a "yes" says only that the count is not the detector's false-positive floor; it does not say the count is topological. That is labelled in the result, whichever way it comes out.
- **Kills:** any unit-test failure blocks the grid; the saturation reference outside tolerance blocks the grid; Stage 0 failing on an arm blocks that arm's sweep (main arm: round ends as "premise failed"); the S³ null's false-positive density exceeding [placeholder, TBD with oppenheimer: 10% of the main arm's density at the slowest used rung] makes Q5a/Q5b "inconclusive (detector)"; parity-invariant violation blocks the grid; NaN or wall > 2 h aborts; the pause file stops it.
- **Reading table:** arch's planning table on inter#194, pre-data, written after oppenheimer settles the open items (arch seq 156), carrying split-G₂ as a named follow-up round. Arch's skeleton (seq 147) has four rows: Q5a plus Q5b in band; Q5a with Q5b outside the band; no power law or Q5c indistinguishable from the null; detector ambiguity over 5%. Every row currently says "no Sprint 4 change". Rows for "premise failed" and "inconclusive (null)" are for arch to add.

## 6. Resource estimate (assumed, to be replaced by a probe)

16³ = 4096 sites × 14 floats ≈ 0.5 MB per replicate: RAM is not a constraint (< 100 MB). Time is the unknown. My r1 guess was about 50 ms per step and about 1.75 h for 20 replicates; R4 ran about 20 times faster than its frozen estimate, so I treat the figure as untrustworthy in either direction. An outcome-blind timing probe on out-of-grid seeds comes first (it may run Stage 0 and one ramp per rung with the detector off), with its fallback rule written in the freeze (mechanical, no after-the-fact ruling; lesson from R4 seq 118). **L is a parameter**: the probe may propose a larger L if the slow rung is under the count floor (L = 24 costs about 3.4×). Limits: 1 core, `GOMAXPROCS=1 nice -n 19 taskset`, ≤ 100 MB RAM, ≤ 50 MB output, one compute spike at a time, 2 h wall cap, pause-file poll.

## 7. Seeds (provisional; never touched except in the matching run mode)

Main arm 5000–5199, S³ null 5200–5399, detector and timing probe 9930–9949. 5400–5599 is held back for the conditional split-G₂ round 2 (not used in round 1, not binding). Seeds for the Stage 0 equilibrium sweeps: to be allocated and checked against every used range before the freeze. Used ranges stay off limits: R3 3000–3199; R4 4000–4399, probe 9960–9969; tests and probes 9100–9979 (scattered).

## 8. Capability gaps (the harness list; tests first, then code, then a smoke on out-of-range seeds)

1. Plane-field lattice state; chordal energy and its gradient (new).
2. Stiefel-projected overdamped Langevin with a temperature ramp (new).
3. The ℤ₂ detector, parity invariant and unit tests (§3) (new).
4. Stage 0 equilibrium sweep and the order parameter S (new).
5. The S³ null: unit-quaternion field, same ramp, the adjoint-frame adapter, its tests (new).
6. *(Round 2 only, not in this harness.)* Split-G₂: split octonions, coset dynamics on a compact slice with a bounded SO(4)-invariant energy. Not built for round 1.
7. The saturation reference.
8. The analyzer: n(τ_Q) with bootstrap CIs, the log-log fit with the curvature test, ambiguity and count-floor records, the decision functions with synthetic-data tests (new).
9. Bit-identity check that the R2/R3/R4 code paths are untouched (the R4 regression pattern).
10. The driver with a probe mode and a run mode, probe-before-run enforced, refusing reserved seeds outside their mode (the R4 driver and guard-script pattern).

## 9. Open items for the chain

1. **Oppenheimer:** check the text at the pushed sha. Still open from seq 158: (a) the Stage 0 order parameter for the S³ arm (my proposal: |(1/N) Σ q|); (b) the S³ null's false-positive tolerance (§5 kills, placeholder 10% of the main arm at the slowest used rung); (c) whether "20 counted defects per rung" means a count pooled over replicates, as I read it; (d) whether the ladder of 7 rungs (25 to 1600) is acceptable for the 1.5-decade requirement.
2. **Arch:** the planning table, after item 1, with split-G₂ as a named follow-up round and rows for "premise failed" and "inconclusive (null)".
3. **Qbp-cu-implementor:** off-box push of r3 and r4 (draft only; I do not push). r1 is at 19fecff and r2 at 9d2ff0e (seq 152, 157). r4 is the one that matches the rulings.
4. **Oppenheimer's Hessian note** (M1) runs in parallel and gates nothing here.
5. **Deming:** nothing until a timing probe exists.
6. **Beekeeper:** nothing needed for the gates (the run go is arch's). Open for him: whether the burndown goal still wants mdcu-kz given arch's critical-path reading ("no Sprint 4 change" on every row), and the standing "R3 report read" reading from seq 108.

## 10. Prediction on file (low confidence, written before any code)

P1–P4 are carried from r1 **unchanged**, with P3 now referring to the S³ null:
- **P1:** the detector passes its unit tests on planted defects. (Near certain.)
- **P2:** n(τ_Q) falls with τ_Q over the ladder with R² ≥ 0.9. (About 70%.) I do not predict an α value.
- **P3:** the S³ null reads near zero. (About 60%; the likeliest way for this round to fail is a badly chosen control.)
- **P4:** ambiguous cells exceed 5% at the fastest rung, so that rung is dropped. (About 50%.)

Added in r2, same status:
- **P5:** Stage 0 finds a continuous-looking transition with no hysteresis beyond tolerance. (About 60%.)
- **P6:** given Q5a holds, α falls inside [0.6, 1.0]. (About 35%. A 16³ lattice, an untested sigma model and a new algebra make a clean KZ exponent less likely than the width of the band suggests.)
- **P7:** the parity invariant holds on i.i.d. Haar fields where there is no ambiguity. (About 85%; my argument for it is untested.)

No part of this is binding until it is frozen and signed off.
