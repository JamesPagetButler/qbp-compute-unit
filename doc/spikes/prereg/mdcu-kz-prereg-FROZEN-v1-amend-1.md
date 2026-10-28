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

- Saturation fields use seeds **5400–5419** (one seed per field), out of the reserved range 5400–5599. They are used only for this check.
- **Measured value, disclosed pre-data** (code `1ca237b`, 20-field blocks, seeds 12300+ and 12000+, outside every frozen range): L = 8: 0.459–0.482; L = 12: 0.463–0.466; L = 16: 0.467 (three blocks); L = 24: 0.465–0.467. The raw density sits at 0.466 and is **not** 0.50, because the sign of a cell is a product of 12 correlated triangle signs. The gate passes with a margin of about 0.016 to its lower edge. The number is the director's; it is flagged here for the signers, not changed.

## 5. S³ null: detector input (item 3) — **RULING PENDING**

Ruled (oppenheimer seq 205, arch seq 206): the S³ arm's detector receives only the SO(3) frame field Ad_q and lifts per edge as the main arm does. Implemented at `1ca237b` as `Z2DetectSO3Frames` (the S³ adapter calls it; q is never passed on).

**Finding (antigravity, post-ruling, seq 211):** this does not make the null non-flat. For frames F_a = R_a ∈ SO(3), the overlap M_ab = R_a R_bᵀ is already a rotation, so every triangle holonomy is the identity and every cell reads +1 on every field. Rough fields change only the ambiguity count. Test `TestKZAmend1S3FramesOnly`: 8 i.i.d. rough S³ fields (L = 8) read 0 raw and 0 defects. So the "S³ null fires" kill remains a code-bug detector, and Q5c is "above an algebraic-zero floor".

**Text to be completed after the ruling,** one of:
- **A.** The null is kept as is. This section then states that it certifies the code path only, reports the S³ ambiguity rate per rung, and carries the planning-table addendum arch trailed in seq 203 (Q5c: "trivially above an algebraic-zero floor; no detector-noise exclusion"; Q5a/Q5b carry the result).
- **B.** The null field is replaced by a rough field of known zero charge with non-flat transport (oppenheimer to define it and set its numbers).

## 6. What does not change

All numbers, seed ranges (apart from the use of 5400–5419 in §4), the ladder, η rule, quench, Stage 0, the §6 timing probe and L/R rule, predictions P1–P7 as filed (P4's 50% and P7's 85% stay on file as written; P7 is read on raw counts per §2), Q5a/Q5b/Q5c thresholds, the count floor, the 5% resolution kill, and the S³ ratio kill threshold of 0.10.

## 7. Signatures

| Signer | Leg | Status |
|---|---|---|
| qbp-oppenheimer | physics (§1–§5) | pending |
| qbp-cu-implementor | feasibility, §6 envelope | pre-assessed GO (seq 209), binds to this file's sha |
| qbp-architecture | planning table and chain | pending |
