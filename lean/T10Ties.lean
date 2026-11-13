import QBP.Sedenion
import T10_CDLane

/-
  T10Ties — engineering-side acceptance criteria for landing T10_CDLane.lean
  (qbp-compute-unit#82, DEP-1 confirmed by qbp-oppenheimer, live-test seq 4581).

  T10_CDLane.lean is landed byte-identical to the frozen outbox copy
  (sha256 641bb2213255f3b4ec57f2f93c3dcd7045b4b54643d86c495086330ccb17d4a7) and is
  never edited here. Its structural theorems (idx_xor_*, lane_pairing_oct,
  sign_unit_sed, imag_sq_oct) hold for any sign convention. This file supplies
  the three landing ACs oppenheimer named:

  (a) decide-tie theorems binding `bmul`'s output to this repo's own ROM source
      (Sedenion.lean's mulSignData/mulIdxData) — without these, T10 can be cited
      for the XOR-index structure but not for *this* sign convention.
  (b) the sedenion-level (k = 4) counterparts of lane_pairing_oct / imag_sq_oct,
      which the frozen file states only at the octonion level (k = 3).
  (c) the toolchain pin is lean/lean-toolchain + .github/workflows/verify-lean-roms.yml.
-/

-- ============================================================
-- AC (a): bmul ties to Sedenion.lean's own ROM tables
-- ============================================================
-- `decide` on 225/256-entry Array lookups needs more than the default 512-depth
-- recursion budget (kernel whnf reduction of Array.get!'s well-founded recursion),
-- not Mathlib-scale — matches the error's own suggested remedy.

set_option maxRecDepth 8000 in
/-- Sedenion.lean's `mulIdxData` is exactly `bmul`'s product index, for every entry
    in the table — not just "equals i XOR j" (which only restates idx_xor_sed). -/
theorem tie_idx_bmul_sed :
    ∀ i j : Fin 16, mulIdxData[i.val * 16 + j.val]! = UInt8.ofNat (bmul 4 i.val j.val).2 := by
  decide

set_option maxRecDepth 8000 in
/-- Sedenion.lean's `mulSignData` encodes exactly `bmul`'s sign (1 = bmul gives +1,
    0 = bmul gives -1), for every one of the 225 non-trivial basis-cross entries. -/
theorem tie_sign_bmul_sed :
    ∀ i j : Fin 15,
      mulSignData[i.val * 15 + j.val]! =
        (if (bmul 4 (i.val + 1) (j.val + 1)).1 = 1 then (1 : UInt8) else 0) := by
  decide

-- ============================================================
-- AC (b): sedenion-level (k = 4) lane pairing + imaginary-square
-- (the frozen file states these only for k = 3, octonions)
-- ============================================================

/-- Lane pairing at the sedenion level: lane p receives exactly the pairs (m, m XOR p). -/
theorem lane_pairing_sed : ∀ p m : Fin 16, (bmul 4 m.val (m.val ^^^ p.val)).2 = p.val := by
  decide

/-- Imaginary sedenion basis elements square to -1. -/
theorem imag_sq_sed : ∀ m : Fin 16, m.val ≠ 0 → bmul 4 m.val m.val = (-1, 0) := by
  decide

#print axioms tie_idx_bmul_sed
#print axioms tie_sign_bmul_sed
#print axioms lane_pairing_sed
#print axioms imag_sq_sed
