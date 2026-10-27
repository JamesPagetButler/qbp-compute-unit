# mdcu-kz pre-registration, DRAFT r3 (NOT FROZEN, NOT FOR RUN)

**Status:** DRAFT r3, supersedes r2 (r1 and r2 are unedited and stay in the tree). Author: antigravity (runner). Branch B, spike/napkin grade. No physics claim and no CD speed claim at any reading. Nothing here is built or run. r3 differs from r2 only where the **ruling of record** (arch seq 150/153, oppenheimer seq 154) crossed my r2 post (seq 155): (a) the Hessian note **no longer gates the freeze**; (b) the S³ and split-G₂ controls are **settled, not provisional**, so r2's include-or-defer question on split-G₂ is withdrawn and the control is included. My concerns about a non-compact target (r2 §4 control 3, a-c) were posted after oppenheimer's seq 154 and have not been answered; r3 turns concern (b) into a mechanical rule (§5 Stage 0) rather than reopening the ruling. The model, band and framing are as in r2 (arch seq 147/150, oppenheimer seq 148/151).

**Framing (arch seq 147, oppenheimer seq 148):**
- mdcu-kz is a **QBP-physics probe, not a CU-design probe.** R4 settled the CU design question (a programmable 3-form unit, no φ claim). Outcomes route to QBP#776 and the QBP theory line. No outcome changes a Sprint 4 build, and no CD speed claim is permitted.
- It tests the **topology** of the order-parameter manifold G₂/SO(4) (π₂ = ℤ₂ point defects). It does **not** test the −φ² landscape that R4 and M1 are about; the two are orthogonal. r1 §0 borrowed R4's motivation for the model; that is withdrawn. R4 is only the source of the question "what does a plane order parameter do in space", not of any expectation here.
- What it buys the compute unit, per oppenheimer: the ℤ₂ holonomy detector is the orientation-aware plane-transport calculation that the QBP#776 E6 controls need, and the split-G₂ control (§4) is a topological discriminator.

**Chain and gates:**
1. **The M1 Hessian note does not gate the freeze** (arch seq 150/153, oppenheimer seq 154, superseding arch seq 147 item 1 and oppenheimer seq 151). The only reason for the gate was the K-psi-type control, which is dropped (§4). The Hessian and mdcu-kz touch different objects and run in parallel.
2. Oppenheimer confirms the S³ adapter and the Stage 0 rules against this text (model, band and controls are settled).
3. Arch commits the planning table on inter#194, pre-data, **when r2/r3 is pushed** (arch seq 150; model and band are settled). I write no table of my own.
4. Deming's box-safety, asked only after a timing probe exists.
5. Freeze with a sha, pushed, every gate ticked.
6. The **run go is arch's** (mdcu-kz is on the inter#224 list, standing authorization), not the beekeeper's. I accept the runner role conditional on 1–5 and a tests-first harness built, reviewed and pushed by qbp-cu-implementor.

Spike 2 (Branch A) is unaffected and does not start until the beekeeper rules.

## 0. Disposition of the r1 open decisions

| r1 item | Ruling | Source | r2 action |
|---|---|---|---|
| 1 model | Model A. Model B dropped (S²/ℤ hedgehog confound). | opp 148 | §2 |
| 2 exponent band | α ∈ [0.6, 1.0], references 0.75 and 0.875, z = 2. | opp 148 | §5 |
| 3 trivial control | S³ (unit quaternions) as the null. Hessian note does not gate model A. | opp 148 | §4 |
| 3b K-psi-type control | Dropped. Replaced by split-G₂/SO(4) (π₂ = 0), **included, settled**. | opp 148/154 | §4 |
| 4 order vs Hessian | Hessian **does not gate** the freeze; the two run in parallel. | arch 150/153, opp 154 | header |
| 5 deming | Only after a timing probe. | r1 | unchanged |
| 6 beekeeper | Run go is arch's. Whether to run at all is now arch's and the beekeeper's on the critical-path question. | arch 147 | header |

## 1. Questions

- **Q5a (defect formation):** after a ramp through the ordering transition of a G₂/SO(4)-valued plane field on an L³ lattice (L = 16 provisional, §6), is the ℤ₂ point-defect density n a power law in the ramp time, n ∝ τ_Q^(−α), over at least 1.5 decades of τ_Q, with a fit CI?
- **Q5b (exponent):** is α inside [0.6, 1.0] (§5)?
- **Q5c (topology vs mere disorder):** is the defect density at a given τ_Q different from the S³ null under the same ramp, and at the S³ level in the split-G₂ control (§4)?

Q5c is the guard against the obvious failure: a detector that counts disagreement between neighbouring planes in any disordered field would show a KZ-like power law for no topological reason.

## 2. Model (settled: model A)

Site x of an L³ periodic lattice holds an associative plane P(x) = span(u, v, uv), built exactly as `haarG2Plane` does. Energy: the sum over nearest-neighbour bonds of d²(P, Q) = 3 − ‖PᵀQ‖²_F = 3 − tr(Π_P Π_Q) (the existing `planeDist` squared; Π is the 7×7 projector onto the plane). Dynamics: overdamped Stiefel-projected Langevin on (u, v), noise temperature T(t) ramped linearly down through T_c over τ_Q steps. Reuses `planeDist`, `haarG2Plane`, `MulRef`, called but not edited.

**Order parameter for the equilibrium check:** M = (1/N) Σ_x Π_{P(x)} − (3/7) I, with S = ‖M‖_F. It is G₂-invariant, zero for i.i.d. Haar planes in expectation, and does not use the detector. (Proposal; oppenheimer to confirm.)

**Not tested here:** the ring locking dynamics, the −φ² landscape, M1–M3.

## 3. Observables and the detector

Unchanged from r1 §3 (second Stiefel–Whitney class of the tautological rank-3 bundle over each cell boundary; 12 triangles per cell; nearest-rotation edge transport; SU(2) lift nearest the identity; product of the 12 lifts, −1 marks a defect; ambiguity rule: a cell with a triangle holonomy of rotation angle ≥ π is "ambiguous", never a defect; > 5% ambiguous at a rung makes that rung "inconclusive (resolution)").

**Added in r2:**
- **Parity invariant.** On a closed periodic lattice every triangle is shared by two cells with opposite orientation, so, if shared-triangle lifts are taken consistently (the lift of the inverse is the inverse of the lift), the total number of −1 cells must be **even** whenever no cell is ambiguous. This is both a unit test and a runtime check. If it fails on a no-ambiguity field, either the detector or this argument is wrong: that blocks the grid, whichever it is. (My derivation; not yet checked by anyone else.)
- **Defect-count floor.** The slow end of the ladder may hold only a handful of defects per replicate on 16³: with α ≈ 0.75 over the 32× ladder span, n falls by about 13×. A rung with a mean count below a floor fixed pre-data (placeholder: 10 defects per replicate) is "inconclusive (finite size)". The probe may propose a larger L (§6).

**Observable:** n(τ_Q) = defects per cell, mean over replicates, 10,000-resample bootstrap CI. Secondary: the two-point plane-overlap correlation length ξ(τ_Q).

**Unit tests before any grid run (tests first):** (a) uniform field has zero defects; (b) a smooth rotation field has zero; (c) a planted ℤ₂ hedgehog (w₂ = 1) is found exactly once; (d) a planted pair is found twice; (e) the ambiguous-cell counter fires on a deliberately sharp field; (f) i.i.d. Haar gives the saturation density n_∞ (the τ_Q → 0 reference); (g) reflection and global G₂ rotations leave the count unchanged; (h) the parity invariant on i.i.d. Haar fields, restricted to no-ambiguity cells, plus the S³ adapter's zero-holonomy tests (§4). The detector is a pure function of the plane field, so these need no dynamics.

## 4. Controls

1. **Saturation reference:** i.i.d. Haar-G₂ planes, no ramp, give n_∞ (test (f)).
2. **S³ null (required; oppenheimer seq 148 (3)):** the same lattice, same ramp, same code path, with an S³-valued (unit-quaternion) field. π₁ = π₂ = 0, so there are no point defects, and any count is the detector's false-positive rate, which is what a null should measure. **Open detail for oppenheimer:** S³ carries no tautological rank-3 bundle, so "the same detector adapted to its bundle" needs a definition. My proposal is the adjoint frame, q ↦ Ad_q ∈ SO(3), with the same transport and SU(2) lift: S³ is simply connected, so lifts exist globally and the product is +1 up to discretisation, and any −1 is a false positive. Two sub-points to confirm: (i) π₃(S³) = ℤ gives non-singular textures in 3D, not point defects, and this ℤ₂ detector does not count them; (ii) the adapter must not share a code path with the main arm that could hide a false positive.
3. **Split-G₂/SO(4) null (settled and included: oppenheimer seq 148/154, arch seq 150/153):** the coset G₂^split/SO(4) is a non-compact symmetric space, contractible, so π₂ = 0 and it must show no KZ defects. This is the physically meaningful null. It needs the split-octonion product, a metric and energy on a non-compact target and a coset parametrisation instead of a Stiefel projection: the largest new build (§8 item 6). **Residual concern, turned into a rule (§5 Stage 0):** on a non-compact target the energy is unbounded, so there may be no ordering transition to cross. If Stage 0 run on this arm finds no ordering transition (or hysteresis beyond the same tolerance), its result is "inconclusive (no transition)" and **not** "no defects confirmed": an absence of defects without a transition would be trivial, not discriminating. The energy for this arm (the analogue of the chordal d² on the non-compact coset, for example the squared geodesic distance or the Killing-form distance) is oppenheimer's to specify; I do not choose it.
4. **K-psi-type control: dropped** (oppenheimer seq 148). If M1 holds, a ψ-plane field has no coset manifold and the control is ill-defined as a sigma model. Any later return of such a control would be a new pre-registration, not an edit of this one.

## 5. Decision rules (skeleton; numeric placeholders are marked and set pre-data)

**Stage 0: continuous-transition check (blocks the τ_Q sweep of the arm it is run on).** A zero-ramp equilibrium sweep of S(T) on L³ (cool and heat, identical settings, no detector call) before any sweep, **run separately for the main arm and for the split-G₂ arm** (the S³ arm needs its own order parameter, an open item). A first-order crossover kills the premise (oppenheimer): **if there is hysteresis between the cool and heat branches of S(T) larger than a pre-data tolerance (placeholder: [TBD with oppenheimer]), or no ordering at all, that arm stops as "premise failed (transition not continuous)"**. For the main arm this ends the round as a clean negative, not a rescue case. For the split-G₂ arm it makes that control "inconclusive (no transition)" (§4 control 3). Stage 0 fixes T_c and the ramp window [T_hi, T_lo] for each arm, and it is outcome-blind with respect to defect counts because it never calls the detector.

**Stage 1: the ladder.** τ_Q ∈ {25, 50, 100, 200, 400, 800, 1600} (7 rungs, about 1.8 decades, so that dropping one end rung still leaves ≥ 1.5 decades; revised by the timing probe).
- **Q5a yes** iff the log-log fit of n against τ_Q has a CI on α excluding 0, **no curvature** (the quadratic term in log n against log τ_Q has a CI including 0; placeholder, oppenheimer to confirm the test), at least 1.5 decades of usable rungs, and fewer than 5% ambiguous cells and above the count floor at each used rung. If dropped rungs leave under 1.5 decades: "inconclusive (span)".
- **Q5b:** α ∈ [0.6, 1.0]. Named reference points α = 0.75 (ν = 1/2, z = 2) and α = 0.875 (ν ≈ 0.7, z = 2), from α = 3ν/(1 + νz) with n ∝ ξ⁻³ and ξ ∝ τ_Q^(ν/(1+νz)). Outside the band means: not KZ, or z ≠ 2, or a ramp not crossing a continuous transition. The band and its reasoning are oppenheimer's; I do not endorse them as physics.
- **Q5c yes** iff n(τ_Q) in the main arm exceeds the S³ null's n with non-overlapping bootstrap CIs at every used rung, **and** the split-G₂ arm (if its Stage 0 passes) reads at or below the S³ null's level with overlapping or lower CIs. If split-G₂ is "inconclusive (no transition)", Q5c rests on the S³ null alone and is labelled so.
- **Kills:** any unit-test failure blocks the grid; the saturation reference outside tolerance blocks the grid; Stage 0 failing blocks the sweep; the S³ null reading non-zero beyond its tolerance makes Q5a/Q5b "inconclusive (detector)"; parity-invariant violation blocks the grid; NaN or wall > 2 h aborts; the pause file stops it.
- **Reading table:** arch's planning table on inter#194, pre-data, committed when this draft is pushed (arch seq 150). Arch's skeleton (seq 147) has four rows: Q5a plus Q5b in band; Q5a with Q5b outside the band; no power law or Q5c indistinguishable from the null; detector ambiguity over 5%. Every row currently says "no Sprint 4 change". Rows for the split-G₂ "inconclusive (no transition)" outcome and for "premise failed" are for arch to add.

## 6. Resource estimate (assumed, to be replaced by a probe)

16³ = 4096 sites × 14 floats ≈ 0.5 MB per replicate: RAM is not a constraint (< 100 MB). Time is the unknown. My r1 guess was about 50 ms per step and about 1.75 h for 20 replicates; R4 ran about 20 times faster than its frozen estimate, so I treat the figure as untrustworthy in either direction. An outcome-blind timing probe on out-of-grid seeds comes first (it may run Stage 0 and one ramp per rung with the detector off), with its fallback rule written in the freeze (mechanical, no after-the-fact ruling; lesson from R4 seq 118). **L is a parameter**: the probe may propose a larger L if the slow rung is under the count floor (L = 24 costs about 3.4×). Limits: 1 core, `GOMAXPROCS=1 nice -n 19 taskset`, ≤ 100 MB RAM, ≤ 50 MB output, one compute spike at a time, 2 h wall cap, pause-file poll.

## 7. Seeds (provisional; never touched except in the matching run mode)

Main arm 5000–5199, S³ null 5200–5399, split-G₂ null 5400–5599, detector and timing probe 9930–9949. Seeds for the Stage 0 equilibrium sweeps: to be allocated and checked against every used range before the freeze. Used ranges stay off limits: R3 3000–3199; R4 4000–4399, probe 9960–9969; tests and probes 9100–9979 (scattered).

## 8. Capability gaps (the harness list; tests first, then code, then a smoke on out-of-range seeds)

1. Plane-field lattice state; chordal energy and its gradient (new).
2. Stiefel-projected overdamped Langevin with a temperature ramp (new).
3. The ℤ₂ detector, parity invariant and unit tests (§3) (new).
4. Stage 0 equilibrium sweep and the order parameter S (new).
5. The S³ null: unit-quaternion field, same ramp, the adjoint-frame adapter, its tests (new).
6. The split-G₂ null: split octonions, coset dynamics, the energy oppenheimer specifies, tests (new; the largest single item).
7. The saturation reference.
8. The analyzer: n(τ_Q) with bootstrap CIs, the log-log fit with the curvature test, ambiguity and count-floor records, the decision functions with synthetic-data tests (new).
9. Bit-identity check that the R2/R3/R4 code paths are untouched (the R4 regression pattern).
10. The driver with a probe mode and a run mode, probe-before-run enforced, refusing reserved seeds outside their mode (the R4 driver and guard-script pattern).

## 9. Open items for the chain

1. **Oppenheimer:** (a) the S³ adapter definition (§4 control 2) and an order parameter for the S³ arm's Stage 0; (b) the split-G₂ energy and its Stage 0 order parameter, and whether the rule "no transition means inconclusive, not confirmed" (§4 control 3, §5) is acceptable; (c) the hysteresis tolerance; (d) the curvature test; (e) whether S is a fair order parameter for the main arm.
2. **Arch:** the planning table, committed when this draft is pushed (seq 150), with rows added for the split-G₂ "inconclusive (no transition)" and "premise failed" outcomes.
3. **Qbp-cu-implementor:** off-box push of r2 and r3 (draft only; I do not push). r1 is already pushed at 19fecff (seq 152).
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
