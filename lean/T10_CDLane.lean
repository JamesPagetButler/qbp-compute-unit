/-
  T10 — CD-MAC lane formulation, finite core (core Lean 4, no Mathlib, kernel `decide` only).
  Basis product of a 2^k Cayley–Dickson algebra: e_m * e_n = s(m,n) * e_{idx(m,n)}, defined by the
  doubling recursion <a,b><c,d> = <ac - conj(d) b, d a + b conj(c)> (same convention as the Python checks).
  The CD-MAC datapath rests on one finite fact: idx(m,n) = m XOR n. Then output lane p collects exactly the
  terms x_m * y_{m XOR p}, i.e. lane p = signed dot product of x with the XOR-by-p permutation of y.
-/
def cj (i : Nat) : Int := if i = 0 then 1 else -1

def bmul : Nat → Nat → Nat → Int × Nat
  | 0, _, _ => (1, 0)
  | k+1, m, n =>
    let h := 2 ^ k
    if m < h then
      if n < h then bmul k m n
      else let r := bmul k (n - h) m; (r.1, r.2 + h)
    else
      if n < h then let r := bmul k (m - h) n; (r.1 * cj n, r.2 + h)
      else let r := bmul k (n - h) (m - h); (-(cj (n - h)) * r.1, r.2)

/-- Octonions (k = 3): the product index is XOR. -/
theorem idx_xor_oct : ∀ m n : Fin 8, (bmul 3 m.val n.val).2 = m.val ^^^ n.val := by decide

/-- Sedenions (k = 4): the product index is XOR. -/
theorem idx_xor_sed : ∀ m n : Fin 16, (bmul 4 m.val n.val).2 = m.val ^^^ n.val := by decide

/-- Lane pairing: lane p receives exactly the pairs (m, m XOR p). -/
theorem lane_pairing_oct : ∀ p m : Fin 8, (bmul 3 m.val (m.val ^^^ p.val)).2 = p.val := by decide

/-- Every sign is ±1 (so the sign ROM is a 1-bit-per-entry table). -/
theorem sign_unit_sed : ∀ m n : Fin 16, (bmul 4 m.val n.val).1 = 1 ∨ (bmul 4 m.val n.val).1 = -1 := by decide

/-- Imaginary basis elements square to -1 (sanity: matches the algebra). -/
theorem imag_sq_oct : ∀ m : Fin 8, m.val ≠ 0 → bmul 3 m.val m.val = (-1, 0) := by decide

#print axioms idx_xor_sed
#print axioms lane_pairing_oct
