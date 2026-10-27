# mdcu-kz pre-registration, DRAFT r1 (NOT FROZEN, NOT FOR RUN)

**Status:** DRAFT r1. Author: antigravity (runner). Branch B, spike/napkin grade. No physics claim and no CD speed claim at any reading. Nothing here is built or run. This draft exists to put the decisions that are not mine in front of the chain (section 9) before any wording is frozen. It carries no seed ranges that anyone may rely on yet (section 7 gives provisional ones).

Chain before any freeze: oppenheimer (model and physics confirm), arch (planning table, pre-data, as for R3/R4), deming (box safety), then freeze with a sha, arch run-go, beekeeper's standing authority. A run needs all of them. Spike 2 (Branch A) is unaffected and does not start until the beekeeper rules.

## 0. Why this round exists (from R4, as scored)

R4 gave Q4a = φ alone does not lock (0/200 at 250, 1000, 4000 steps) and Q4b = FORM (ψ locks where φ does not at matched comass; S-D is a clean dose-response). The named candidate mechanisms (oppenheimer seq 135/136, hypotheses with kills, not adopted) are M1, a degenerate maximiser family: φ's maximisers form the 8-dimensional associative Grassmannian G₂/SO(4), so −φ² has a flat valley with no gradient toward a plane; M2, a coupling window; M3, metric mismatch. The R4 S-A line also showed that K-full's locked planes are Haar-G₂ uniform (p = 0.968): a locking K-full ring lands on a *uniformly random associative plane*. That is the natural "order parameter" for a spatial experiment: a point of G₂/SO(4).

Arch's R4 disposition: no further mdcu round on φ vs ψ; next is mdcu-kz (seq 133/137).

Topology that motivates it: π₁(G₂/SO(4)) = 0 and π₂(G₂/SO(4)) = π₁(SO(4)) = ℤ₂ (G₂ is 2-connected: π₁ = π₂ = 0). So a plane field in 3D can carry **ℤ₂ point defects**, and a Kibble–Zurek ramp should create them at a density that depends on the ramp time. Whether the lattice model here realises that is exactly what this round would test at napkin grade.

## 1. Questions

- **Q5a (defect formation):** after a ramp through the ordering transition of a plane field with values in G₂/SO(4) on a 16³ lattice, does the ℤ₂ point-defect density n depend on the ramp time τ_Q as a power law, n ∝ τ_Q^(−α), over at least one decade of τ_Q?
- **Q5b (exponent):** is α inside an interval fixed before data (section 5)?
- **Q5c (topology vs mere disorder):** is the ℤ₂ defect density at a given τ_Q different from the density in a topologically trivial control (section 4) under the same ramp?

Q5c is the guard against the obvious failure: a detector that counts "places where neighbouring planes disagree" in any disordered field, which would show a KZ-like power law for no topological reason.

## 2. Model (DECISION NEEDED: section 9, item 1)

Two candidates. I recommend A.

**A. Plane-field relaxation (recommended).** Site x of an L³ lattice holds an associative plane P(x) = span(u, v, uv), built exactly as `haarG2Plane` does (u, v orthonormal in the 7 imaginary octonion directions, third direction their octonion product). Local energy: E = Σ over nearest-neighbour bonds of chordal distance squared between planes, d²(P, Q) = 3 − ‖PᵀQ‖²_F (the existing `planeDist` squared). Dynamics: overdamped Langevin on the Stiefel pair (u, v), projected to keep the pair orthonormal; noise temperature T(t) is ramped linearly down through T_c over τ_Q steps. Periodic boundaries. Reuses `planeDist`, `haarG2Plane`, `MulRef`. Pro: the order parameter space is G₂/SO(4) by construction, so π₂ = ℤ₂ is a fact of the model, not of R3/R4 dynamics. Con: it does not use the ring locking dynamics at all, so it says nothing about *why* K-full locks; it is a statement about the target manifold only.

**B. Vector lattice with axis-line triples.** Each site holds one unit imaginary 7-vector; the ring's μ term and K-full triple term act along the three lattice axes. Planes appear only after coarse-graining. Pro: closer to R3/R4. Con: a field of unit vectors that lie in a common 3-plane is S²-valued, and an S²-valued field carries ℤ-valued hedgehogs (π₂(S²) = ℤ). That would be a different defect group from ℤ₂, and the defect counts of the two kinds would be confounded in the detector. I do not know a clean way to separate them, and I have no evidence the ring's lock extends to 3D at all.

I am not choosing for the chain. R4 says the *landscape* is the result; model A tests only the manifold.

## 3. Observables and the detector

**Defect (ℤ₂ point defect):** the second Stiefel–Whitney class of the tautological rank-3 bundle over the sphere that bounds a lattice cell. Oriented planes: φ restricted to an associative plane is ±vol, so the sign of φ fixes an orientation, which makes the frame bundle SO(3) (orientation-aware).

**Detector (to be unit-tested before any run):**
1. For each cube cell, triangulate its boundary into 12 triangles.
2. Along each edge, transport the 3-frame by the nearest rotation to the overlap matrix between the two planes' bases (SVD/polar part).
3. The holonomy around each triangle is an SO(3) rotation; lift it to SU(2) on the branch nearest the identity.
4. Multiply the 12 lifts, conjugated to a common base point; the product is ±1, and −1 marks a defect.
5. **Admissibility:** the nearest-identity lift is only defined if every triangle holonomy has rotation angle below π (a resolution condition). A cell that fails is counted "ambiguous", never as a defect. If more than 5% of cells are ambiguous at any τ_Q, that τ_Q is "inconclusive (resolution)".

**Observable:** n(τ_Q) = defects per cell, mean over replicates, with a 10,000-resample bootstrap CI. Secondary: two-point plane correlation length ξ(τ_Q) from the plane overlap function.

**Unit tests before any grid run (tests first):** (a) a uniform plane field has zero defects; (b) a smooth rotation field has zero; (c) a planted ℤ₂ hedgehog (a plane field with w₂ = 1 on a 16³ lattice) is found exactly once, with its charge; (d) a planted pair is found twice and the pair annihilates in the continuum-limit test; (e) the ambiguous-cell counter fires on a field with a deliberately sharp gradient; (f) an i.i.d. Haar field gives the saturation density n_∞, recorded and used as the τ_Q → 0 reference; (g) reflection and global SO(7) ∩ G₂ rotation leave the count unchanged. The detector is a pure function of the plane field, so these are exact and need no dynamics.

## 4. Controls

1. **Saturation reference:** i.i.d. Haar-G₂ planes (no ramp) give n_∞ for the detector (test (f)).
2. **Topologically trivial control (Q5c):** the same lattice, same ramp, same code path, with the plane replaced by a vector-valued order parameter on a manifold whose π₂ is trivial for the defect type counted (details wait on item 3 of section 9). The detector is run on it unchanged, and must read near zero after the ramp. If it does not, the detector is counting disorder, and Q5a and Q5b are "inconclusive (detector)".
3. **K-psi-type control:** if oppenheimer's Hessian note (the M1 kill, QBP `analysis/`, expectation: 3 zero modes for a generic ψ, versus 8 on the Grassmannian) shows the ψ maximiser is isolated, then the ψ order parameter is a point modulo the frame and carries no ℤ₂ point defects. That would make it the natural second control. **This control cannot be written until that note lands; I will not guess its result.**

## 5. Decision rules (skeleton; thresholds are placeholders to be set pre-data)

- **Q5a yes** iff the log-log fit of n against τ_Q on the ladder {50, 100, 200, 400, 800, 1600 steps} (to be revised by the timing probe) has R² ≥ 0.9, α > 0 with a bootstrap CI excluding 0, and fewer than 5% ambiguous cells at every rung.
- **Q5b:** α in [α_lo, α_hi], both fixed at r2 before any data. I will not fix them from the literature alone: the universality class of a G₂/SO(4) sigma model in 3D is not known to me, so the generic mean-field and O(3) numbers are not defaults. Oppenheimer to supply the band and its reasoning (section 9, item 2).
- **Q5c yes** iff n(τ_Q) in the main arm exceeds the trivial control's n with non-overlapping bootstrap CIs at every rung.
- **Kills:** ambiguous > 5% at a rung drops that rung ("inconclusive (resolution)"); a detector failure on any unit test blocks the grid; the saturation reference outside its tolerance blocks the grid; a NaN or wall > 2 h aborts; the pause file stops it.
- Reading table and consequences: arch's planning table (to be posted pre-data, as for R3/R4). No outcome changes a Sprint 4 build, and no CD speed claim is permitted.

## 6. Resource estimate (assumed, to be replaced by a probe)

16³ = 4096 sites × 14 floats for (u, v) ≈ 0.5 MB per replicate: RAM is not a constraint (< 100 MB). The unknown is time. A rough cost per Langevin step is 4096 sites × 6 neighbour bonds × a few hundred flops, so about 50 ms per step on the FX-8350 without AVX2 (a guess). Ladder rungs total about 6,300 ramp steps if each ramp is about 2τ_Q, so 20 replicates would be about 1.75 h, near the 2 h limit. **R4's measured per-run time was about 20 times below the frozen estimate**, so I would treat this number as untrustworthy in either direction and require an outcome-blind timing probe on out-of-grid seeds first, with a fallback rule written in the freeze (the frozen text, mechanical, no after-the-fact ruling; lesson from R4 seq 118). Limits: 1 core, `GOMAXPROCS=1 nice -n 19 taskset`, ≤ 100 MB RAM, ≤ 50 MB output, one compute spike at a time, 2 h wall cap, pause-file poll.

## 7. Seeds (provisional, reserved here, never touched except in the matching run mode)

Main arm 5000–5199 (replicates; 200 reserved, fewer used if the probe says so), trivial control 5200–5399, detector and timing probe 9930–9949. Earlier used ranges (R3 3000–3199; R4 4000–4399; probes and tests 9100–9979) stay off limits.

## 8. Capability gaps (the harness list; tests first, then code, then a smoke on out-of-range seeds)

1. Plane-field lattice state and the chordal-distance energy and gradient (new).
2. Stiefel-projected overdamped Langevin with a temperature ramp (new).
3. The ℤ₂ detector and its unit tests (section 3) (new).
4. The saturation reference and the trivial control path (new).
5. The analyzer: n(τ_Q) with bootstrap CIs, the log-log fit, the ambiguous-cell record, and the decision functions with synthetic-data tests (new).
6. A bit-identity check that the existing R2/R3/R4 code paths are untouched (reuse the R4 regression pattern; `planeDist`, `haarG2Plane`, `MulRef` are only called, not edited).
7. The driver with a probe mode and a run mode, probe-before-run enforced, refusing reserved seeds outside their mode (reuse the R4 driver and guard-script pattern).

## 9. Open decisions for the chain (not mine to settle)

1. **Oppenheimer:** model A or B (section 2), or a third. In particular, does a plane-field sigma model on G₂/SO(4) answer anything about the compute unit, given that R4 puts the interest in the *landscape*, which model A does not use?
2. **Oppenheimer:** the pre-data exponent band for Q5b, with its reasoning, and whether the relevant dynamics is relaxational (z = 2) or something else.
3. **Oppenheimer:** the topologically trivial control's order-parameter manifold (section 4, control 2), and whether the Hessian note (section 4, control 3) should land first.
4. **Arch:** whether this round is wanted at all before the Hessian computation (the theory item arch assigned to oppenheimer, seq 137), since a clean M1 result may change which controls are meaningful. Planning table, pre-data.
5. **Deming:** the box gate on the estimate in section 6 once a probe exists; I will not ask for it before then.
6. **Beekeeper:** whether to freeze mdcu-kz at all given that the φ-vs-ψ line is closed, and the standing "R3 report read" reading from seq 108 (still unconfirmed by him).

## 10. Prediction on file (low confidence, written before any code)

- **P1:** the detector passes its unit tests on planted defects. (Near certain; it is a pure function.)
- **P2:** n(τ_Q) falls with τ_Q over the ladder with R² ≥ 0.9, because any smooth relaxation leaves less disorder after a slower ramp. (About 70%.) I do not predict an α value.
- **P3:** the trivial control reads near zero. (About 60%; the likeliest way for this round to fail is that the control is badly chosen.)
- **P4:** ambiguous cells exceed 5% at the fastest rung, so that rung is dropped. (About 50%.)

No part of this is binding until it is frozen and signed off.
