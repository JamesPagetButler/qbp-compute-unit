# mdcu-kz pre-registration, DRAFT r7 — FREEZE CANDIDATE (NOT FROZEN, NOT FOR RUN)

**Status:** DRAFT r7, supersedes r6 (r1 to r6 are unedited and stay in the tree). Author: antigravity (runner). Branch B, spike/napkin grade. No physics claim and no CD speed claim at any reading. Nothing here is built or run. **r7 is the text I propose to freeze.** It is a *candidate*, not a FROZEN file: like R4, the FROZEN-v1 file will be this text copied verbatim with only the status block and the §9 gate list changed, after the gates are ticked, so that nothing here can be read as already frozen. **What r7 changes from r6, and nothing else (arch seq 187, oppenheimer seq 186, qbp-cu seq 188):** (1) a fixed **post-ramp quench of Q = 50 steps at T = 0** before the single read, the same at every rung and in both arms (oppenheimer's physics amendment; it replaces r6's "no relaxation after the ramp"); (2) per-run steps become 200 + τ_Q + 50, so the per-replicate-and-arm total is **4925** and §6's W uses 4925·t_s (r6 had 4575); (3) the §2a wording note that T is the bath temperature of the (u, v) pair dynamics; (4) this status block, the chain list and §9 brought to the state of the bridge at seq 188, including that the "R3 report read" item is closed (seq 178/179). **Standing from r6, as ruled:** the B items (§2a dynamics, the §5 Stage 0a/0b mechanics, §6 probe rules, §7 seeds) were accepted by oppenheimer at seq 186 and put in the freeze text by arch's chain ruling at seq 185. Everything else is r5, which oppenheimer's seq 158/163/166 and arch's seq 162/165/170 settled. Split-G₂ stays DEFERRED (round 2, a new spike needing beekeeper approval).

**Framing (arch seq 147, oppenheimer seq 148):**
- mdcu-kz is a **QBP-physics probe, not a CU-design probe.** R4 settled the CU design question (a programmable 3-form unit, no φ claim). Outcomes route to QBP#776 and the QBP theory line. No outcome changes a Sprint 4 build, and no CD speed claim is permitted.
- It tests the **topology** of the order-parameter manifold G₂/SO(4) (π₂ = ℤ₂ point defects). It does **not** test the −φ² landscape that R4 and M1 are about; the two are orthogonal. r1 §0 borrowed R4's motivation for the model; that is withdrawn. R4 is only the source of the question "what does a plane order parameter do in space", not of any expectation here.
- What it buys the compute unit, per oppenheimer: the ℤ₂ holonomy detector is the orientation-aware plane-transport calculation that the QBP#776 E6 controls need. The topological contrast (split-G₂, π₂ = 0) is a conditional round 2 (§4).

**Chain and gates:**
1. **The M1 Hessian note does not gate the freeze** (arch seq 150/153, oppenheimer seq 154, superseding arch seq 147 item 1 and oppenheimer seq 151). The only reason for the gate was the K-psi-type control, which is dropped (§4). The Hessian and mdcu-kz touch different objects and run in parallel.
2. Oppenheimer's physics confirm at the pushed sha of the text that freezes. **Done for r5** (`29246f4`, file sha256 `26c025f3…0a10`; seq 173). His freeze check is "FROZEN-v1 = r5 + the four named items (seq 177: Stage 0 seed ranges, the timing-probe fallback as a numeric sentence, L, the deming-ordering chain-text fix) + the accepted B items with the Q = 50 amendment" (seq 186; arch seq 187). **B ruling, seq 186: accepted** (§2a, §5 Stage 0a/0b, §6), with the one amendment (Q = 50) and the T wording note, both folded into this r7.
3. Qbp-cu-implementor's feasibility check. **Done for r5 and r6: GO** (seq 174, 184; RAM < 50 MB even at L = 24, disk ≤ 50 MB, buildable on Go 1.24.2) and **carries to the r7 count** (seq 188: 4575 → 4925, +7.65% on the grid term, absorbed by the mechanical R rule). Re-opens only if the frozen L is above 32 or GOMAXPROCS=1 is dropped (arch seq 175). L ∈ {16, 24} stays inside that. He re-verifies on the r7 sha at push.
4. Deming's box-safety clears the **frozen plan, including the timing probe's own §6 limits and its mechanical fallback rule written as a numeric sentence** (arch seq 175, R4 pattern: deming cleared R4 at seq 110, then the probe ran inside the GO'd run). Written upper bounds only (RAM/disk figures, the 2 h wall cap, GOMAXPROCS=1, nice/taskset, pause file); measured timings are not required for the clearance. **The probe is part of the run, not a separate pre-freeze compute step**: it runs first, outcome-blind, on out-of-grid seeds, and the frozen rule decides L and R mechanically.
5. FROZEN-v1 pushed with its sha.
6. The tests-first harness, built by me, reviewed and pushed by qbp-cu-implementor.
7. The **run go is arch's** (mdcu-kz is on the inter#224 list, standing authorization), not the beekeeper's. I accept the runner role conditional on 1–6.

Arch's planning table is **done** (inter#194 issuecomment-6451397247, posted 2026-10-27T19:21Z, written against r4 `4a3fe2a` plus seq 166); §5 points at it.

Spike 2 (Branch A) is unaffected and does not start until the beekeeper rules.

## 0. Disposition of the r1 open decisions

| r1 item | Ruling | Source | r7 action |
|---|---|---|---|
| 1 model | Model A. Model B dropped (S²/ℤ hedgehog confound). | opp 148 | §2 |
| 2 exponent band | α ∈ [0.6, 1.0], references 0.75 and 0.875, z = 2. | opp 148 | §5 |
| 3 trivial control | S³ (unit quaternions) as the null; adapter confirmed. | opp 148/158 | §4 |
| 3b K-psi-type control, then split-G₂ | K-psi-type dropped. Split-G₂ **deferred to a conditional round 2** (needs a bounded, SO(4)-invariant energy on a compact slice and an ordering transition shown first). Gr⁺(3,7) is not a π₂ = 0 contrast (π₂ = ℤ₂). Seq 154's "included" is superseded. | opp 158 | §4 |
| 4 order vs Hessian | Hessian **does not gate** the freeze; the two run in parallel. | arch 150/153, opp 154 | header |
| 5 deming | Box-safety on the frozen plan incl. the probe's own §6 limits and fallback rule; the probe runs inside the GO'd run. | arch 170/175 | §6 |
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

### 2a. Dynamics and ramp parameters [accepted by oppenheimer seq 186 with the Q = 50 amendment; r1 to r5 left these implicit]

The harness cannot be written, and no rule can be mechanical, until these are fixed before data. Each was a choice I proposed in r6; oppenheimer accepted them at seq 186 (with the Q = 50 amendment), and arch put them in the freeze text at seq 185.

- **Units.** Coupling J = 1: E = J Σ_bonds d²(P, Q). Temperature T is in units of J. S³ arm: E = J Σ_bonds (1 − q·q′), same J and T units (the two arms are never compared at equal T, only through their own Stage 0).
- **Meaning of T (oppenheimer seq 186).** T is the bath temperature of the (u, v) pair dynamics below, not a thermodynamic temperature on the coset: the pair over-parametrises the plane by the in-plane frame. Every T in the design (Stage 0 included) is internal to that same dynamics, so ratios such as 1.5 T_c and 0.5 T_c are well defined.
- **Step rule.** One lattice step updates all sites synchronously from the previous state: gradient step on the Stiefel pair (u, v) with step size η, plus Gaussian noise of variance 2ηT per component, then Gram–Schmidt back to an orthonormal pair; the plane is span(u, v, uv). S³ arm: the same step on the unit quaternion q, with renormalisation. Any implementation is acceptable that passes the T = 0 descent test below; the exact gradient routine is the implementor's.
- **η.** Fixed before the probe by this rule, using only a T = 0 energy test and no defect count: the largest η in {0.1, 0.05, 0.02, 0.01} for which the T = 0 relaxation of an i.i.d. Haar start, on seeds 9930–9934 for the main arm and the same seeds for the S³ arm, has non-increasing total energy over 200 steps on all five seeds. One η for both arms: the smaller of the two.
- **Start state.** i.i.d. Haar-G₂ planes (main, `haarG2Plane`), i.i.d. Haar S³ (null), seeded by the run's seed.
- **Hold.** 200 steps at T_hi before the ramp starts.
- **Ramp.** T(k) = T_hi − (T_hi − T_lo) · k / τ_Q for k = 0 … τ_Q, **exactly τ_Q steps**, with T_hi = 1.5 T_c and T_lo = 0.5 T_c, T_c from Stage 0a (§5), per arm. The ramp rate is therefore ∝ 1/τ_Q at a fixed window, which is the Kibble–Zurek parametrisation.
- **Quench (oppenheimer seq 186, pre-data amendment).** After the ramp ends at k = τ_Q, Q = 50 further steps at T = 0 (the same step rule with the noise term zero, and η as above), **identical at every rung and in both arms**. Reason: at T_lo = 0.5 T_c the field still carries thermal fluctuations, and the detector would count thermal near-defects and ambiguous cells as well as frozen-in KZ defects; a short T = 0 descent removes the roughness without creating or annihilating topological charge at the lattice scale. Because Q is the same at every τ_Q it cannot change the scaling exponent. It lowers the ambiguity rate and the false-positive floor. The quench is not part of Stage 0.
- **Measurement.** At the end of the quench (step τ_Q + 50 of the ramp-plus-quench) the state is read once by the detector and the secondaries. Nothing is read before it.
- **Paired design.** Replicate j at every rung uses the same seed 5000 + j (main) or 5200 + j (null); the seed fixes the Haar start state only.
- **Per-run step count.** 200 + τ_Q + 50. Per replicate and arm, over the 7 rungs: 7 · 200 + 3175 + 7 · 50 = 4925 steps (this number is used by §6).

## 3. Observables and the detector

Unchanged from r1 §3 (second Stiefel–Whitney class of the tautological rank-3 bundle over each cell boundary; 12 triangles per cell; nearest-rotation edge transport; SU(2) lift nearest the identity; product of the 12 lifts, −1 marks a defect; ambiguity rule: a cell with a triangle holonomy of rotation angle ≥ π is "ambiguous", never a defect; > 5% ambiguous at a rung makes that rung "inconclusive (resolution)").

**Added in r2:**
- **Parity invariant (checked by oppenheimer, seq 158).** On a closed periodic lattice every triangle is shared by two cells with opposite orientation, so with consistent shared-triangle lifts (the lift of the inverse is the inverse of the lift) the product over all cells is Π(g·g⁻¹) = +1 and the number of −1 cells must be **even** whenever no cell is ambiguous. It is the discrete "divergence-free" condition for the ℤ₂ flux. Unit test and runtime check. If it fails on a no-ambiguity field, either the detector or this argument is wrong: that blocks the grid, whichever it is.
- **Defect-count floor (oppenheimer seq 158, pooling settled seq 166).** At least **20 counted defects per rung, pooled over the replicates at that rung** (the Poisson count sets the error; relative error ≲ 22%). Below that the rung is "inconclusive (finite size)". L is chosen **before the grid, by the timing probe alone (§6), and never after any count is seen**: the earlier "prefer raising L to dropping the rung" is realised only as that pre-grid rule. The slow end of the ladder is the risk: with α ≈ 0.75 over the 32× span n falls by about 13×, and at small L the correlation length may also approach L.

**Observable:** n(τ_Q) = defects per cell, mean over replicates, 10,000-resample bootstrap CI. Secondary: the two-point plane-overlap correlation length ξ(τ_Q).

**Unit tests before any grid run (tests first):** (a) uniform field has zero defects; (b) a smooth rotation field has zero; (c) a planted ℤ₂ hedgehog (w₂ = 1) is found exactly once; (d) a planted pair is found twice; (e) the ambiguous-cell counter fires on a deliberately sharp field; (f) i.i.d. Haar gives the saturation density n_∞ (the τ_Q → 0 reference); (g) reflection and global G₂ rotations leave the count unchanged; (h) the parity invariant on i.i.d. Haar fields, restricted to no-ambiguity cells, plus the S³ adapter's zero-holonomy tests (§4). The detector is a pure function of the plane field, so these need no dynamics.

## 4. Controls

1. **Saturation reference:** i.i.d. Haar-G₂ planes, no ramp, give n_∞ (test (f)).
2. **S³ null (required; oppenheimer seq 148 (3), adapter confirmed seq 158):** the same lattice, same ramp, same code path, with an S³-valued (unit-quaternion) field. π₁ = π₂ = 0, so there are no point defects, and any count is the detector's false-positive rate, which is what a null should measure. S³ carries no tautological rank-3 bundle, so the detector is adapted by the **adjoint frame**, q ↦ Ad_q ∈ SO(3), with the same transport and SU(2) lift. Ad: S³ → SO(3) is the 2:1 cover, so a field Ad_q built from a globally defined q has a global SU(2) lift (q itself), and any −1 from the triangle-lift product is a pure detector false positive (oppenheimer seq 158). π₃(S³) = ℤ textures are not point defects and the ℤ₂ detector rightly ignores them. The null also certifies the detector against SO(3)'s own π₁ = ℤ₂ line defects, which cannot occur for a field lifted from q, so a −1 cannot be "a real line defect" either (oppenheimer seq 158). The adapter must not share a code path with the main arm in a way that could hide a false positive. **Stage 0 order parameter for this arm (oppenheimer seq 163/166):** m = ‖⟨q⟩‖, the magnetisation of the unit-quaternion field (0 for Haar, 1 for aligned), with q and −q **not** identified, since the field is q itself, not Ad_q. The unit-quaternion field with ferromagnetic coupling is an O(4) Heisenberg model and has a continuous transition, so Stage 0 on this arm is meaningful. S_max = 1.
3. **Split-G₂/SO(4) contrast: DEFERRED to a conditional round 2 (oppenheimer seq 158/161/163; arch seq 162, re-ruled on cost at 165).** The coset G₂^split/SO(4) is non-compact and contractible (π₂ = 0), but the chordal energy is unbounded there and, more fundamentally, its disordered phase is non-normalisable (at high T the field runs away to infinity), so there is no symmetric high-T phase for a ramp to leave, and Stage 0 is expected to return "no transition". Oppenheimer's own prediction for it is "inconclusive (no transition)"; arch ruled that cost with no expected information. The obvious compact alternative fails: the oriented Grassmannian Gr⁺(3,7) = SO(7)/(SO(3)×SO(4)) has π₂ = ℤ₂ as well, so it is not a π₂ = 0 contrast. Round 2 would be a **new spike**, not on the approved inter#224 list, needing beekeeper approval and conditional on a bounded SO(4)-invariant energy whose ordering transition is shown first. Oppenheimer's starting spec (E = 1 − exp(−d_geo²/ℓ²), S_split on the Karcher mean) goes into arch's round-2 candidate note, not here. **Nothing for split-G₂ is built or run in round 1.** Round 1 therefore cannot say whether the defect count is *topological* beyond what the S³ null (detector false-positive rate) shows; that limit is stated here so it is not read into the result.
4. **K-psi-type control: dropped** (oppenheimer seq 148). If M1 holds, a ψ-plane field has no coset manifold and the control is ill-defined as a sigma model. Any later return of such a control would be a new pre-registration, not an edit of this one.

## 5. Decision rules (skeleton; numeric placeholders are marked and set pre-data)

**Stage 0: continuous-transition check (blocks the τ_Q sweep of the arm it is run on).** Run **separately for the main arm and for the S³ arm**, each with its own order parameter (main: S normalised by √(12/7); S³: m = ‖⟨q⟩‖), no detector call, in two parts [mechanics accepted by oppenheimer seq 186; the tolerances are his seq 158]:

- **Stage 0a (coarse quasi-static scan, finds T_c).** 40 geometrically spaced T points from 0.05 to 8 (units of J); at each point 600 steps (300 equilibrate, then 300 measured, S averaged over the measured steps), starting from the previous point's end state, cooling from high T; 3 seeds per arm. **T_c = the T of steepest descent of the seed-mean S(T)** (largest −ΔS/ΔT between neighbouring grid points). If S shows no ordering in the window (mean S at the lowest T below 0.5 · S_max, or no point of steepest descent above the noise: ΔS between neighbours never exceeds 3 times the seed standard error), the result is "premise failed (no ordering in window)" for the main arm and "inconclusive (null)" for the S³ arm.
- **Stage 0b (hysteresis at the slowest ramp).** At τ_Q = 1600, over [1.5 T_c, 0.5 T_c], cool then heat continuously per seed (the heating run starts from the cooled end state, same rate), S recorded every 10 steps; 10 seeds per arm. **Pass criterion (same for both arms):** on a common 41-point T grid, |mean S_cool − mean S_heat| ≤ 5% of S_max at every grid T, **and** no single recorded-step jump of the seed-mean S exceeds 10% of S_max. Otherwise "first-order or glassy, premise fails", recorded.

A failure on the main arm ends the round as a clean negative ("premise failed"), not a rescue case. A failure on the S³ arm makes the null uninformative, so the round is "inconclusive (null)": Q5c is uninterpretable, the detector or the control gets fixed, and there is no physics reading (arch seq 165). Stage 0 fixes T_c and the ramp window [T_hi, T_lo] for each arm, and it is outcome-blind with respect to defect counts because it never calls the detector. Stage 0 seeds: 0a main 5600–5604, 0a S³ 5605–5609, 0b main 5610–5619, 0b S³ 5620–5629.

**Stage 1: the ladder.** τ_Q ∈ {25, 50, 100, 200, 400, 800, 1600} (7 rungs, 1.81 decades; dropping either end rung still leaves ≥ 1.5 decades; accepted by oppenheimer seq 166; the timing probe may revise it, and any revision goes into the freeze text before the run).
- **Q5a yes** iff, with log n fitted against log τ_Q both linearly and quadratically, the **quadratic coefficient's 95% CI includes 0 and ΔAIC(quadratic − linear) > −2** (oppenheimer seq 158, the no-curvature test); α and its CI come from the linear fit over at least 1.5 decades of usable rungs, with the CI on α excluding 0; and each used rung has fewer than 5% ambiguous cells and at least 20 counted defects (§3). If dropped rungs leave under 1.5 decades: "inconclusive (span)".
- **Q5b:** α ∈ [0.6, 1.0]. Named reference points α = 0.75 (ν = 1/2, z = 2) and α = 0.875 (ν ≈ 0.7, z = 2), from α = 3ν/(1 + νz) with n ∝ ξ⁻³ and ξ ∝ τ_Q^(ν/(1+νz)). Outside the band means: not KZ, or z ≠ 2, or a ramp not crossing a continuous transition. The band and its reasoning are oppenheimer's; I do not endorse them as physics.
- **Q5c yes** iff n(τ_Q) in the main arm exceeds the S³ null's n with non-overlapping bootstrap CIs at every used rung. Because the split-G₂ contrast is deferred, a "yes" says only that the count is not the detector's false-positive floor; it does not say the count is topological. That is labelled in the result, whichever way it comes out.
- **Kills:** any unit-test failure blocks the grid; the saturation reference outside tolerance blocks the grid; Stage 0 failing on an arm blocks that arm's sweep (main arm: round ends as "premise failed"; S³ arm: "inconclusive (null)"); **"the S³ null fires" (oppenheimer seq 166)** iff at **any used rung** n_null(τ_Q) > 0.10 · n_main(τ_Q) with non-overlapping bootstrap CIs, **or** any null cell reads −1 while the parity check passes at zero ambiguity (a certain false positive), which invalidates the round: Q5a/Q5b are "inconclusive (detector)"; parity-invariant violation blocks the grid; NaN or wall > 2 h aborts; the pause file stops it.
- **Reading table:** arch's planning table, committed pre-data on inter#194 (issuecomment-6451397247, https://github.com/JamesPagetButler/inter/issues/194#issuecomment-6451397247, body sha256 `81833c17…c3c9` as abbreviated by arch; the full hash is to be supplied by arch at the freeze and cited in the FROZEN file). Seven rows, gates first, then physics. All route to QBP#776 or the QBP theory line, with no Sprint 4 change. This draft does not restate the rows: the table on inter#194 is the reading rule, and where this text and the table disagree the disagreement is a freeze blocker to be resolved before the freeze, not after data.

## 6. Resource estimate and mechanical probe rules [accepted seq 186; replaces the r5 "assumed estimate"]

16³ = 4096 sites × 14 floats ≈ 0.5 MB per replicate: RAM is not a constraint (< 100 MB). Time is the unknown. My r1 guess (about 50 ms per step) would put a 200-replicate grid near 35 h; R4 ran about 20 times faster than its frozen estimate, so no figure is trusted until probed. **The numbers L and R (replicates per rung and arm) are fixed from timing alone, never from counts, and posted before the grid runs.**

**Probe (before any grid; detector counts are never inspected for this decision), out-of-grid seeds 9930–9949:** 9930–9934 the η descent test (§2a); 9935–9939 steps timing at L = 16; 9940–9944 steps timing at L = 24; 9945–9949 detector timing (one full pass over a field, at each L). Result: t_s(L) = seconds per lattice step (median over the timing seeds, both arms averaged), t_d(L) = seconds per detector pass.

**Wall estimate:** W(L, R) = 2 · R · (4925 · t_s(L) + 7 · t_d(L)) + t_s(L) · 208,000, where 4925 steps per replicate and arm are from §2a (ramp ladder, holds and the Q = 50 quenches), the factor 2 is the two arms, 7 is one detector pass per rung, and 208,000 is the Stage 0 step count (0a: 40 × 600 × 3 seeds × 2 arms = 144,000; 0b: 10 seeds × 2 arms × 2 × 1600 = 64,000; both rounded together).

**Rule (mechanical):** R ∈ {200, 100, 50, 25}. For a given L take the largest R with W(L, R) ≤ 1.5 h (hard cap 2 h, as in the kills). **L = 24 iff W(24, R) ≤ 1.5 h for some R ≥ 50; else L = 16** with the largest fitting R. If even R = 25 does not fit at L = 16, **the round does not run** and goes back to the chain. **R < 25 is not permitted;** R = 25 or 50 are run, and the round is labelled "low power" in the report, with the defect-count floor (§3) the arbiter of the individual rungs. Limits: 1 core, `GOMAXPROCS=1 nice -n 19 taskset`, ≤ 100 MB RAM, ≤ 50 MB output, one compute spike at a time, 2 h wall cap, pause-file poll. The ladder is not changed by the probe (a rung revision would need a new draft, not a rule here).

## 7. Seeds (never touched except in the matching run mode)

- Main arm 5000 … 5000 + R − 1 (R from §6, at most 200); S³ null 5200 … 5200 + R − 1 (paired: seed j at every rung).
- Stage 0: 0a main 5600–5604, 0a S³ 5605–5609, 0b main 5610–5619, 0b S³ 5620–5629.
- Probe (η descent test, timing): 9930–9949 as in §6.
- 5400–5599 reserved and unused in round 1 (a possible round 2; not binding).
- Used ranges, off limits: R3 3000–3199; R4 4000–4399 and 9960–9969; tests and probes 9100–9979 (scattered; 9970–9979 used by tests). 5xxx and 993x/994x do not occur as seeds in the existing Go code (grep of emulator/ at drafting: the only 5xxx hits are loop counts, permutation sizes and benchmark text); to be re-checked at the freeze.

## 8. Capability gaps (the harness list; tests first, then code, then a smoke on out-of-range seeds)

1. Plane-field lattice state; chordal energy and its gradient (new).
2. Stiefel-projected overdamped Langevin with a temperature ramp (new).
3. The ℤ₂ detector, parity invariant and unit tests (§3) (new).
4. Stage 0a scan and Stage 0b hysteresis driver, the order parameters S and m, and the §2a step rule and η descent test (new).
5. The S³ null: unit-quaternion field, same ramp, the adjoint-frame adapter, its tests (new).
6. *(Round 2 only, not in this harness.)* Split-G₂: split octonions, coset dynamics on a compact slice with a bounded SO(4)-invariant energy. Not built for round 1.
7. The saturation reference.
8. The analyzer: n(τ_Q) with bootstrap CIs, the log-log fit with the curvature test, ambiguity and count-floor records, the decision functions with synthetic-data tests (new).
9. Bit-identity check that the R2/R3/R4 code paths are untouched (the R4 regression pattern).
10. The driver with a probe mode and a run mode, probe-before-run enforced, refusing reserved seeds outside their mode (the R4 driver and guard-script pattern), and the §6 mechanical L/R decision function with synthetic-timing tests.

## 9. Confirmations (freeze gates; chain from arch seq 170)

This text is a candidate. §2a, §5 Stage 0, §6 and §7 were additions of mine beyond "r5 + four"; they were accepted (oppenheimer seq 186) and put in the freeze text (arch seq 185). Reviewers still check the frozen file against that ruling.

1. **Arch's planning table** on inter#194: done (seq 170), re-pinned to r5 with an addendum, the table unedited (seq 172). FROZEN-v1 cites issuecomment-6451397247. Full sha256 of the comment body to be supplied by arch and cited in the FROZEN file.
2. **Oppenheimer:** confirmed r5 (seq 173); ruled on the B items at seq 186 (all accepted, Q = 50 amendment, T wording note). Pending: his freeze check at the pushed sha of the frozen text, "r5 + four + accepted B + Q".
3. **Qbp-cu-implementor:** GO on r5 and r6, carried to r7 (seq 174, 184, 188). r1 to r6 are pushed (r6 `5ccaf99`, file sha256 `6b3c597d…c7a10`, seq 184); r7 is local until he pushes it (I do not push). He asks no re-check unless L > 32 or GOMAXPROCS=1 is dropped.
4. **Deming:** box-safety on the frozen plan including the probe's own §6 limits and fallback rule (arch seq 175). Pending.
5. **FROZEN-v1** pushed, with its sha. Created only after 1–4, as this text copied verbatim with only the status block and this gate list changed.
6. **Tests-first harness** (§3 tests (a)–(h), §8 items).
7. **Arch's run go** (the inter#224 standing authorization). The beekeeper's go is not required by the chain.

Open for the beekeeper, not a gate: whether the burndown still wants mdcu-kz (arch: no Sprint 4 change on any row). The "R3 report read" item is **closed**: the beekeeper ruled first-hand that the reviewers' read counts (oppenheimer seq 178, arch seq 179), so R4's FROZEN-v1 was a valid freeze. Oppenheimer's Hessian note (M1) runs in parallel and gates nothing.

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
