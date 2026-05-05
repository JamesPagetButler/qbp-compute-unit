/-
  Sedenion.lean — Source of truth for QBP-CU sign and index ROMs

  Author:    QBP architecture instance (Claude Opus 4.7)
  Date:      2026-05-05
  Repo:      github.com/JamesPagetButler/qbp-compute-unit
  Path:      lean/QBP/Sedenion.lean
  Issue:     #7 (lean2rom build pipeline)

  This file defines the algebraic structure of the sedenion algebra 𝕊
  derived from the Cayley-Dickson doubling construction applied to the
  octonions 𝕆 ⊂ sedenions 𝕊.

  Attribution
  -----------
  · Moreno (1998) "The zero divisors of the Cayley-Dickson algebras over
    the real numbers" — XOR-index lemma; basis labelling convention.
  · Cawagas (2009) "Loops & Quasigroups Newsletter" — 42 cross-copy
    basis-sum zero-divisors characterisation.
  · Schafer (1954) "On the algebras formed by the Cayley-Dickson process"
    — CD product formula (a,b)(c,d) = (ac − conj(d)·b, d·a + b·conj(c)).
  · Baez (2002) "The Octonions" — Fano-plane sign table for 𝕆 submatrix.

  Algebra facts encoded
  ---------------------
  · Sedenion basis: e₀ (real unit) through e₁₅ (16 elements).
  · XOR-index lemma: idx(eᵢ · eⱼ) = i XOR j for all i,j ∈ 0..15.
  · Sign of product: derived from Cayley-Dickson doubling of the standard
    Fano-plane octonion table (Schafer convention).
  · 42 cross-copy basis-sum zero-divisors of the form
    (eᵢ + eⱼ)(eₖ + eₗ) = 0  with i,k ∈ {1..7}, j,l ∈ {9..15}.
    Count verified by explicit enumeration (Cawagas 2009).

  Cayley-Dickson doubling used
  ----------------------------
  Starting from the reals, each level doubles:
    ℝ → ℂ → ℍ → 𝕆 → 𝕊
  Product formula at each level: (a,b)(c,d) = (ac − conj(d)b, da + b·conj(c))
  where conj(e₀) = e₀ and conj(eₖ) = −eₖ for k > 0.

  This generates the standard Schafer-convention sedenion multiplication table
  where the quaternion submatrix (indices 0–3) satisfies i·j=k, j·k=i, k·i=j.

  Data layout
  -----------
  mulSignData : Array UInt8
    Flat row-major array, length 225 = 15 × 15.
    Entry at offset (i−1)×15 + (j−1) encodes the sign bit of eᵢ · eⱼ
    for i,j ∈ {1..15} (non-trivial basis-cross products only; e₀ is the
    real unit so e₀·eₖ = eₖ with sign +1 trivially).
    Encoding: 1 = positive (+), 0 = negative (−).

  mulIdxData : Array UInt8
    Flat row-major array, length 256 = 16 × 16.
    Entry at offset i×16 + j encodes the index of eᵢ · eⱼ (= i XOR j)
    for i,j ∈ {0..15}.
    By the XOR-index lemma (Moreno 1998) this is always i XOR j; the array
    is included for self-documenting completeness and direct ROM extraction.
-/

-- ============================================================
-- mulSignData: sign of eᵢ · eⱼ for i,j ∈ {1..15}
-- Length: 225  (= 15 × 15)
-- 1 = positive (+), 0 = negative (−)
-- Row i, column j: offset = (i−1)×15 + (j−1)
--
-- Derived by Cayley-Dickson construction from the Fano-plane
-- quaternion/octonion seed table (Schafer/Baez convention):
--   e₁·e₂=+e₃, e₂·e₃=+e₁, e₃·e₁=+e₂  (quaternion H submatrix)
-- ============================================================
def mulSignData : Array UInt8 := #[
  -- e₁  (j=1..15)
  0, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
  -- e₂  (j=1..15)
  0, 0, 1, 1, 1, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1,
  -- e₃  (j=1..15)
  1, 0, 0, 1, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 1,
  -- e₄  (j=1..15)
  0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0,
  -- e₅  (j=1..15)
  1, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 1, 0, 1, 0,
  -- e₆  (j=1..15)
  1, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1,
  -- e₇  (j=1..15)
  0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 0,
  -- e₈  (j=1..15)
  0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1,
  -- e₉  (j=1..15)
  1, 0, 1, 0, 1, 1, 0, 0, 0, 0, 1, 0, 1, 1, 0,
  -- e₁₀ (j=1..15)
  1, 1, 0, 0, 0, 1, 1, 0, 1, 0, 0, 0, 0, 1, 1,
  -- e₁₁ (j=1..15)
  0, 1, 1, 0, 1, 0, 1, 0, 0, 1, 0, 0, 1, 0, 1,
  -- e₁₂ (j=1..15)
  1, 1, 1, 1, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0,
  -- e₁₃ (j=1..15)
  0, 1, 0, 1, 1, 1, 0, 0, 0, 1, 0, 1, 0, 1, 0,
  -- e₁₄ (j=1..15)
  0, 0, 1, 1, 0, 1, 1, 0, 0, 0, 1, 1, 0, 0, 1,
  -- e₁₅ (j=1..15)
  1, 0, 0, 1, 1, 0, 1, 0, 1, 0, 0, 1, 1, 0, 0
]

-- ============================================================
-- mulIdxData: index of eᵢ · eⱼ for i,j ∈ {0..15}
-- Length: 256  (= 16 × 16)
-- By the XOR-index lemma (Moreno 1998), idx(eᵢ·eⱼ) = i XOR j.
-- Row i, column j: offset = i×16 + j
-- ============================================================
def mulIdxData : Array UInt8 := #[
  -- i=0
   0,  1,  2,  3,  4,  5,  6,  7,  8,  9, 10, 11, 12, 13, 14, 15,
  -- i=1
   1,  0,  3,  2,  5,  4,  7,  6,  9,  8, 11, 10, 13, 12, 15, 14,
  -- i=2
   2,  3,  0,  1,  6,  7,  4,  5, 10, 11,  8,  9, 14, 15, 12, 13,
  -- i=3
   3,  2,  1,  0,  7,  6,  5,  4, 11, 10,  9,  8, 15, 14, 13, 12,
  -- i=4
   4,  5,  6,  7,  0,  1,  2,  3, 12, 13, 14, 15,  8,  9, 10, 11,
  -- i=5
   5,  4,  7,  6,  1,  0,  3,  2, 13, 12, 15, 14,  9,  8, 11, 10,
  -- i=6
   6,  7,  4,  5,  2,  3,  0,  1, 14, 15, 12, 13, 10, 11,  8,  9,
  -- i=7
   7,  6,  5,  4,  3,  2,  1,  0, 15, 14, 13, 12, 11, 10,  9,  8,
  -- i=8
   8,  9, 10, 11, 12, 13, 14, 15,  0,  1,  2,  3,  4,  5,  6,  7,
  -- i=9
   9,  8, 11, 10, 13, 12, 15, 14,  1,  0,  3,  2,  5,  4,  7,  6,
  -- i=10
  10, 11,  8,  9, 14, 15, 12, 13,  2,  3,  0,  1,  6,  7,  4,  5,
  -- i=11
  11, 10,  9,  8, 15, 14, 13, 12,  3,  2,  1,  0,  7,  6,  5,  4,
  -- i=12
  12, 13, 14, 15,  8,  9, 10, 11,  4,  5,  6,  7,  0,  1,  2,  3,
  -- i=13
  13, 12, 15, 14,  9,  8, 11, 10,  5,  4,  7,  6,  1,  0,  3,  2,
  -- i=14
  14, 15, 12, 13, 10, 11,  8,  9,  6,  7,  4,  5,  2,  3,  0,  1,
  -- i=15
  15, 14, 13, 12, 11, 10,  9,  8,  7,  6,  5,  4,  3,  2,  1,  0
]

-- ============================================================
-- Verification: entry counts
-- ============================================================
#eval do
  if mulSignData.size == 225 then
    IO.println "mulSignData.size = 225  OK"
  else
    IO.println s!"mulSignData.size = {mulSignData.size}  FAIL (expected 225)"
  if mulIdxData.size == 256 then
    IO.println "mulIdxData.size  = 256  OK"
  else
    IO.println s!"mulIdxData.size  = {mulIdxData.size}  FAIL (expected 256)"

-- ============================================================
-- Verification: XOR-index lemma for mulIdxData
-- ============================================================
-- Check that mulIdxData[i*16+j] == i XOR j for all i,j in 0..15
def checkXorIndex : IO Bool := do
  let mut ok := true
  let pairs : List (Nat × Nat) :=
    (List.range 16).flatMap fun i => (List.range 16).map fun j => (i, j)
  for (i, j) in pairs do
    let expected : UInt8 := UInt8.ofNat (i ^^^ j)
    let got := mulIdxData[i * 16 + j]!
    if got != expected then
      IO.println s!"FAIL idx[{i}][{j}]: got {got}, expected {expected}"
      ok := false
  return ok

#eval do
  let ok ← checkXorIndex
  if ok then IO.println "mulIdxData XOR-index lemma: all 256 entries OK"

-- ============================================================
-- Extraction output for lean2rom tool
-- Each #eval below emits a JSON line that lean2rom parses.
-- ============================================================
#eval do
  let signs := mulSignData.toList.map (fun (x : UInt8) => x.toNat.repr)
  let body := String.intercalate "," signs
  IO.println s!"\{\"mulSignData\":[{body}]}"

#eval do
  let idxs := mulIdxData.toList.map (fun (x : UInt8) => x.toNat.repr)
  let body := String.intercalate "," idxs
  IO.println s!"\{\"mulIdxData\":[{body}]}"
