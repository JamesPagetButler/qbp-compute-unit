# mdcu-kz pre-registration, FROZEN-v1, amendment 1 (DRAFT, not signed)

**Status:** DRAFT for signing. Dated 2026-10-27, written before any Stage 0, probe or grid run. It amends `mdcu-kz-prereg-FROZEN-v1.md` (commit `7ac4ed0`, file sha256 `176bdce2bd4544f0f0bac8523ecc04f5a6eb582d603bb2714713fa72fa676a5f`), which is not edited. Where this file is silent, FROZEN-v1 stands. Branch B, spike/napkin grade: no physics claim and no CD speed claim at any reading.

**Origin.** Harness check-in 1 (antigravity seq 202, detector commit `b41c66f`) surfaced four unstated or vacuous inputs. Rulings: qbp-oppenheimer seq 205 (physics), qbp-architecture seq 206 (chain). Consequential clauses (a)–(c): antigravity seq 207, qbp-architecture seq 208. **Signatures required before the run:** oppenheimer (physics), qbp-cu-implementor (feasibility, incl. the §6 envelope; pre-assessed GO seq 209), qbp-architecture (planning-table check and chain). Nothing in the grid, the probe or Stage 0 runs before all three.

## 1. Resolution rule (replaces the ambiguity clause of FROZEN-v1 §3)

FROZEN-v1 §3 reads: "a cell with a triangle holonomy of rotation angle ≥ π is ambiguous". SO(3) rotation angles lie in [0, π], so the clause fires only at exactly π. The 5% resolution check and P4 could never trip.

**Rule in force:**
- A **triangle** is ambiguous iff **any of its three edge rotations has angle ≥ π/2**. The edge rotation is the nearest-rotation transport of that edge, its angle being 2·arccos|w| of the edge lift (independent of the lift's sign). "Edge" includes the face diagonal of the triangulation, since it is an edge of the triangle.
- The frozen clause (triangle holonomy angle ≥ π) is **kept as a second clause**. It is now redundant for ruling purposes.
- A **cell** is ambiguous iff **any of its 12 triangles is**. An ambiguous cell is never a defect.
- Reason (oppenheimer seq 205): two neighbouring planes whose transport rotates by ≥ π/2 are closer to orthogonal than to aligned, and the sign of the SU(2) lift between them is noise.
- A numeric tolerance of 1e-9 on the comparison is used (float slack only).

**Pre-data expectation** (oppenheimer seq 205, measured on the code at `1ca237b`): i.i.d. Haar fields read about 100% ambiguous (measured 1.0000, L = 8, 6 fields; the frozen π rule read 0%); smooth fields read 0%. The fastest rungs may legitimately be dropped under the existing row "inconclusive (resolution / span)". No outcome row changes.

## 2. Raw count versus defect count; parity (arch seq 208 (a), (b), (c))

- The detector reports two counters. **Raw** = cells whose product of 12 triangle signs is −1, ambiguous or not. **Defects** = −1 cells with no ambiguous triangle.
- **Raw** feeds test (f), the saturation check (§4 below) and the parity check. **Defects** feed n(τ_Q), the ≥ 20 count floor, the Q5a/Q5b/Q5c rows and the null kills. **"n" in every Q5a/Q5b/Q5c row of the planning table means defects, never raw.** No row changes.
- **Parity** (FROZEN-v1 §3, P7, test (h)): the raw −1 count is even on every field, ambiguous or not, because one sign is cached per lattice triangle and every triangle belongs to exactly two cells. The frozen text restricted this to zero-ambiguity fields. Under §1 that restriction would test nothing on Haar fields (about 100% ambiguous), so the check is on the raw count for every field. A violation blocks the grid, as in FROZEN-v1. **P7 is read on raw counts.**
- **Null kill, second clause** ("any null cell reads −1 while the parity check passes at zero ambiguity") is kept as written, applied per cell: a −1 on a cell with no ambiguous triangle. The 0.10 ratio clause uses defects.

## 3. Test (g), disclosed reinterpretation

FROZEN-v1 §3 test (g): "reflection and global G₂ rotations leave the count unchanged". Measured with the detector at `b41c66f`: global orthogonal rotations of R⁷ and **full inversion** (i, j, k) → (−i, −j, −k) leave the count exactly unchanged on every field. A **single-axis** reflection leaves it exactly unchanged on **resolved** planted fields, but not on unresolved ones: the count then depends on the face-diagonal triangulation (224 vs 230 cells on one i.i.d. Haar field). The same fact is why §1 had to bite before any count is read.

**Test (g) is restated as:** exact under global orthogonal rotations and under full inversion on every field; exact under single-axis reflection on resolved fields (zero ambiguous cells under §1); for unresolved fields no reflection claim is made. This is a reinterpretation of the frozen test, disclosed here, not a change of any threshold.

## 4. Saturation sanity check (replaces the unnumbered tolerance in FROZEN-v1 §4 / §5 kills)

FROZEN-v1 says "the saturation reference outside tolerance blocks the grid" and names no tolerance. **Frozen by the director (arch seq 206), proposed by oppenheimer (seq 205):** the **raw** density of −1 cells over **20 i.i.d. Haar-G₂ plane fields at the run's L** must be **0.50 ± 0.05**. Outside it is a code bug and the grid is blocked. It is a code sanity check, not a physics reference. Test (f)'s "n_∞" is read as this raw density.

- Saturation fields use seeds **5630–5649** (one seed per field), a fresh range after Stage 0's 5600–5629 (arch seq 212). The reserved range 5400–5599 stays reserved for the deferred split-G₂ round 2 and is not touched. The range was checked against every used range (R3 3000–3199; R4 4000–4399 and 9960–9969; tests and probes 9100–9979; kz main 5000–5199, null 5200–5399, Stage 0 5600–5629, probe 9930–9949). Used only for this check.
- **Measured value, disclosed pre-data** (code `1ca237b`, 20-field blocks, seeds 12300+ and 12000+, outside every frozen range): L = 8: 0.459–0.482; L = 12: 0.463–0.466; L = 16: 0.467 (three blocks); L = 24: 0.465–0.467. The raw density sits at 0.466 and is **not** 0.50, because the sign of a cell is a product of 12 correlated triangle signs. The gate passes with a margin of about 0.016 to its lower edge. **The gate stays at 0.50 ± 0.05 as frozen by the director (arch seq 212: recentring a sanity gate on the code's own output would make it circular).** The 0.034 offset is recorded here as an observation, not a gate. If oppenheimer derives the expected value pre-data, the gate is re-set to the derived number ± 0.05 in this section before signing.

## 5. S³ null: detector input and read-time tilt (item 3, **option B**)

Ruled: oppenheimer seq 215 (physics), qbp-architecture seq 217, 218 (chain), qbp-oppenheimer seq 219; qbp-cu-implementor seq 216 (feasibility).

**Background (antigravity seq 211, theorem and test).** The S³ arm's detector already received only the SO(3) frame field Ad_q (`Z2DetectSO3Frames` at `1ca237b`; q is never passed on). That does not make the null non-flat: for F_a = R_a ∈ SO(3) the overlap M_ab = R_a R_bᵀ is already a rotation, every triangle holonomy is the identity, and under the π/2 rule the nearest lifts multiply to +1 on any field. Test `TestKZAmend1S3FramesOnly`. Option A (null left flat) is therefore **not** adopted. oppenheimer withdraws the sentence "every −1 is a roughness false positive" of seq 205.

**Definition (B).** The S³ arm keeps its frozen quench dynamics: target S³, the same rungs, the same Model-A update. **At read time only**, each site's frame plane f_x (the R³ ⊂ R⁷ carrying Ad_q, frame F_x = [R_x | 0]) is tilted by E_x = exp(σ ξ_x), with ξ_x an i.i.d. unit-Frobenius-norm element of so(7), **fixed per site per replicate** by its own pinned seed. The plane field P_x = E_x P₀ is homotopic to the constant plane through t ↦ exp(tσξ_x), so **w₂ = 0 exactly for every σ**; the non-flat transport comes from the polar factor of F_a F_bᵀ no longer being a rotation. Every non-ambiguous −1 on the B null is then a true detector false positive, and **Q5c reads as originally tabled; no planning-table addendum** (arch seq 217).

**Implementation choices the ruling leaves implicit (HARNESS, disclosed):**
- ξ_x is drawn as 21 i.i.d. standard normals for the strict upper triangle, antisymmetrised and scaled to unit Frobenius norm, from the PCG stream (tilt seed, site index). The rows of F_x are mapped as f'_i = E_x f_i. exp is a 20-term Taylor series (σ ≤ 0.4; the truncation error is far below rounding).
- The tilt of replicate j is the same at every rung (it is fixed per site per replicate, not per run). Stage 0 never calls the detector, so it uses no tilt seed.
- σ = 0 reproduces the S³ adapter exactly (test).

**σ* rule (oppenheimer seq 215, pre-data).** σ* = the largest value on the grid {0.05, 0.10, 0.20, 0.40} rad such that the perturbed null's ambiguous-cell fraction (π/2 rule) on 20 pinned fields stays ≤ 5%. If even σ = 0.05 exceeds 5%, the null runs at σ = 0.05 and that fact is recorded. The fraction is **pooled** (pooled ambiguous cells over pooled cells of the 20 fields), as in the rung resolution rule. **[PENDING oppenheimer: the calibration field ensemble and the pooled reading. Antigravity proposes E1: the aligned S³ field (q = 1) tilted by the 20 pinned tilt seeds, measuring the tilt alone.]**

**Calibration (arch seq 218, oppenheimer seq 219): at both candidate lattice sizes, before the run.** σ*(16) and σ*(24) are calibrated once, pre-run, on the same pinned seeds for both sizes, and both are frozen below. The run uses the entry for the L the probe selects ("σ* is read from the pinned table by the probe-selected L"). **There is no calibration step after the probe.** The calibration runs no dynamics, probe, Stage 0 or grid (unit level, at most 160 detector passes on single lattices, within the class deming cleared in seq 200).

| Quantity | Value |
|---|---|
| Calibration seeds | tilt 5850–5869, field 5870–5889 (field seeds used only if the ensemble needs them) |
| Run tilt seeds | 5650–5849 (replicate j uses 5650 + j; R ≤ 200) |
| σ*(16) | **[PENDING calibration]** |
| σ*(24) | **[PENDING calibration]** |
| Calibration raw sha256 | **[PENDING calibration]** |
| Fallback used (σ = 0.05)? | **[PENDING calibration]** |

All of 5650–5889 was checked against every used range (R3 3000–3199; R4 4000–4399 and 9960–9969; tests and probes 9100–9979; kz main 5000–5199, null 5200–5399, reserved 5400–5599, Stage 0 5600–5629, saturation 5630–5649, probe 9930–9949); none overlaps. Smoke runs and unit tests use 12000+/13000+, and the seed guard refuses 5650–5889 outside the calibration and grid modes.

**Cost (qbp-cu-implementor seq 216):** one 7×7 exponential per site per read on the S³ arm, at most 3% of that arm and 1.5% of the run, folded into the probe's t_d so the mechanical L/R rule absorbs it. No change to the step count, ladder, L ≤ 24, RAM or GOMAXPROCS = 1.

**Disclosed pre-signing observation (antigravity seq 221; exploration on free seeds 13300+, L = 16, 4 analytic smooth SO(3) fields per row, amplitudes A = 0–32; not the calibration, no dynamics).** On the ruled grid σ ≤ 0.40 the tilted null read **zero raw −1 cells at every roughness**, and the ambiguous fraction moved little with σ. Zero raw −1 also holds through σ = 1.6 at A = 0, 4, 8. At σ = 2.4 raw −1 appears (588 cells, set by the tilt rather than the field), with 83–98% of cells ambiguous. So on the ruled grid the B null is flat in practice: it reports the detector's false-positive rate on resolved fields, and it is unlikely to fail. This is recorded as an observation, not a rule. `TestKZExplore` (skipped unless `KZ_EXPLORE=1`) reproduces it.

## 6. What does not change

All numbers, seed ranges (apart from the new saturation range 5630–5649 in §4), the ladder, η rule, quench, Stage 0, the §6 timing probe and L/R rule, predictions P1–P7 as filed (P4's 50% and P7's 85% stay on file as written; P7 is read on raw counts per §2), Q5a/Q5b/Q5c thresholds, the count floor, the 5% resolution kill, and the S³ ratio kill threshold of 0.10.

## 7. Signatures

| Signer | Leg | Status |
|---|---|---|
| qbp-oppenheimer | physics (§1–§5) | pending |
| qbp-cu-implementor | feasibility, §6 envelope | pre-assessed GO (seq 209), binds to this file's sha |
| qbp-architecture | planning table and chain | pending |
