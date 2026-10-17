package mdcu

import "math"

// ---------------------------------------------------------------------------
// Fano frames (Silicon §3.2): the 7 quaternionic subalgebras of 𝕆 are
// span{1, e_i, e_j, e_k} for the 7 lines {i,j,k} of PG(2,2), k = i⊕j.
// Frame 0 is the canonical Cayley-Dickson lo half {1,e1,e2,e3}.
// Each line is ordered so that e_i·e_j = +e_k (a genuine ℍ embedding).
// ---------------------------------------------------------------------------

// FanoLine is an oriented quaternionic triple: e_I·e_J = +e_K.
type FanoLine struct{ I, J, K int }

var fanoLines [8]FanoLine

func init() {
	n := 1
	fanoLines[0] = orient(1, 2)
	seen := map[[3]int]bool{}
	for i := 1; i < 8; i++ {
		for j := i + 1; j < 8; j++ {
			k := i ^ j
			if k < j {
				continue // each line counted once (i<j<k)
			}
			key := [3]int{i, j, k}
			if seen[key] {
				continue
			}
			seen[key] = true
			fanoLines[n] = orient(i, j)
			n++
		}
	}
	if n != 8 {
		panic("mdcu: expected 7 Fano lines")
	}
}

func orient(i, j int) FanoLine {
	if signTable[i][j] > 0 {
		return FanoLine{i, j, i ^ j}
	}
	return FanoLine{j, i, i ^ j}
}

// Frame returns the oriented line for frame f ∈ 0..7.
func Frame(f int) FanoLine { return fanoLines[f] }

// frameIdx returns (bulk indices, fiber indices) in 𝕆 for frame f.
func frameIdx(f int) (bulk, fiber [4]int) {
	l := fanoLines[f]
	bulk = [4]int{0, l.I, l.J, l.K}
	in := map[int]bool{0: true, l.I: true, l.J: true, l.K: true}
	n := 0
	for k := 0; k < 8; k++ {
		if !in[k] {
			fiber[n] = k
			n++
		}
	}
	return
}

// ---------------------------------------------------------------------------
// Boundary ledger (E4): every bit of norm that leaves the active state must
// be deposited in a named sink. Conservation: ‖state‖² + Σ sinks = ‖initial‖².
// ---------------------------------------------------------------------------

// Ledger accumulates squared norm routed to Boundary sinks.
type Ledger struct {
	Ground      float64 // ZD / norm-loss drain (𝕊→𝕆)
	Containment float64 // associator / transverse drain (𝕆→ℍ)
	Release     float64 // explicit discarded complement (lossy projection)
}

// Total returns the sum over all sinks.
func (l *Ledger) Total() float64 { return l.Ground + l.Containment + l.Release }

func norm2(x []float64) float64 {
	var s float64
	for _, v := range x {
		s += v * v
	}
	return s
}

// ---------------------------------------------------------------------------
// Tier ladder: QLIFT / QPROJ / QSPLIT / QJOIN.
// ---------------------------------------------------------------------------

// QLift embeds x (tier dim n) into tier `to` (dim ≥ n) by zero-padding the
// higher Cayley-Dickson halves. Exact.
func QLift(x []float64, to Tier) []float64 {
	z := make([]float64, to.Dim())
	copy(z, x)
	return z
}

// QSplit steps one rung down: x ∈ tier t (dim 2m) → (bulk, fiber) ∈ tier t−1.
// For 𝕆→ℍ the Fano frame f selects which quaternionic subalgebra is the bulk;
// for other rungs (and f = 0) the split is the canonical CD lo/hi halves.
// Exact (a coordinate permutation): QJoin(QSplit(x)) ≡ x bit-for-bit.
func QSplit(x []float64, f int) (bulk, fiber []float64) {
	n := len(x)
	h := n / 2
	if n == 8 && f != 0 {
		bi, fi := frameIdx(f)
		bulk, fiber = make([]float64, 4), make([]float64, 4)
		for k := 0; k < 4; k++ {
			bulk[k], fiber[k] = x[bi[k]], x[fi[k]]
		}
		return
	}
	bulk = append([]float64(nil), x[:h]...)
	fiber = append([]float64(nil), x[h:]...)
	return
}

// QJoin is the exact inverse of QSplit for the same frame.
func QJoin(bulk, fiber []float64, f int) []float64 {
	h := len(bulk)
	z := make([]float64, 2*h)
	if h == 4 && f != 0 {
		bi, fi := frameIdx(f)
		for k := 0; k < 4; k++ {
			z[bi[k]], z[fi[k]] = bulk[k], fiber[k]
		}
		return z
	}
	copy(z, bulk)
	copy(z[h:], fiber)
	return z
}

// QProj steps down keeping only the bulk; the discarded fiber norm is
// deposited in ledger.Release ("keep the complement" is QSplit; QProj is the
// explicit, accounted lossy variant — never a silent drop).
func QProj(x []float64, f int, led *Ledger) []float64 {
	b, fb := QSplit(x, f)
	led.Release += norm2(fb)
	return b
}

// QAssoc returns the associator (xy)z − x(yz) at the active tier. For tier ≤ ℍ
// it returns an exact zero vector (architectural guarantee: ℍ is associative,
// so the unit does not expose rounding residue as "curvature").
func QAssoc(t Tier, x, y, z []float64) []float64 {
	out := make([]float64, len(x))
	if t <= TierH {
		return out
	}
	l := cdMul(cdMul(x, y), z)
	r := cdMul(x, cdMul(y, z))
	for i := range out {
		out[i] = l[i] - r[i]
	}
	return out
}

// cdMul is the CD-MAC product on slices of length 2^k (≤16).
func cdMul(x, y []float64) []float64 {
	n := len(x)
	z := make([]float64, n)
	for m := 0; m < n; m++ {
		for k := 0; k < n; k++ {
			z[m^k] += signTable[m][k] * x[m] * y[k]
		}
	}
	return z
}

// ---------------------------------------------------------------------------
// Width ladder: QWIDEN / QNARROW (QW64 ↔ QW128 double-double).
// ---------------------------------------------------------------------------

// QWiden lifts QW64 components to QW128. Exact.
func QWiden(x []float64) []DD {
	z := make([]DD, len(x))
	for i, v := range x {
		z[i] = DD{v, 0}
	}
	return z
}

// QNarrow rounds QW128 → QW64 (round-to-nearest of hi+lo; hi is already that
// value for a normalized DD). The discarded lo parts are returned as the
// narrowing loss; if any |lo| exceeds budget·‖x‖ the seam flag is raised.
func QNarrow(x []DD, budget float64) (z []float64, loss float64, seam bool) {
	z = make([]float64, len(x))
	var n2 float64
	for i, v := range x {
		s, e := twoSum(v.Hi, v.Lo) // renormalize defensively
		z[i] = s
		loss = math.Max(loss, math.Abs(e))
		n2 += s * s
	}
	seam = loss > budget*math.Sqrt(n2)
	return
}

// MulDD is the QW128 CD-MAC product for any 2^k length.
func MulDD(x, y []DD) []DD {
	n := len(x)
	z := make([]DD, n)
	for m := 0; m < n; m++ {
		for k := 0; k < n; k++ {
			t := ddMul(x[m], y[k])
			if signTable[m][k] < 0 {
				t = DD{-t.Hi, -t.Lo}
			}
			z[m^k] = ddAdd(z[m^k], t)
		}
	}
	return z
}

// ---------------------------------------------------------------------------
// Leaf ladder: QSPLIT.LEAF / QJOIN.LEAF (𝕊 ↔ 5D leaf, #688 / Theorem 4).
// ---------------------------------------------------------------------------

// LeafFrame is the CSR-locked orthonormal imaginary 2-frame (u₁,u₂) ⊂ Im 𝕆.
type LeafFrame struct {
	U1, U2     [8]float64
	Degenerate bool // a fallback basis vector was substituted (L5b)
}

// Leaf is the 5D leaf state q = (a1, a2, c1, c2, b0).
type Leaf [5]float64

const degenTol = 1e-14

// QSplitLeaf acquires the leaf frame from s0 (port of extract_leaf_frame).
// Unlike the Python reference (which silently yields u = 0 via max(·,1e-30)),
// degenerate inputs get a valid orthonormal frame by fallback (L5b).
func QSplitLeaf(s *[16]float64) (Leaf, LeafFrame) {
	var a, c [8]float64
	copy(a[1:], s[1:8])
	copy(c[1:], s[9:16])
	b0 := s[8]
	var fr LeafFrame
	a1 := math.Sqrt(norm2(a[:]))
	if a1 > degenTol {
		for k := range a {
			fr.U1[k] = a[k] / a1
		}
	} else {
		fr.U1[1], fr.Degenerate = 1, true
	}
	var c1 float64
	for k := range c {
		c1 += c[k] * fr.U1[k]
	}
	var cp [8]float64
	for k := range c {
		cp[k] = c[k] - c1*fr.U1[k]
	}
	c2 := math.Sqrt(norm2(cp[:]))
	if c2 > degenTol {
		for k := range cp {
			fr.U2[k] = cp[k] / c2
		}
	} else {
		// Gram-Schmidt the first coordinate axis e1/e2 not parallel to u1.
		fr.Degenerate = true
		for _, ax := range []int{1, 2} {
			var e [8]float64
			e[ax] = 1
			d := fr.U1[ax]
			for k := range e {
				e[k] -= d * fr.U1[k]
			}
			if n := math.Sqrt(norm2(e[:])); n > 0.5 {
				for k := range e {
					fr.U2[k] = e[k] / n
				}
				break
			}
		}
		c2 = 0
	}
	if a1 <= degenTol {
		a1 = 0
	}
	return Leaf{a1, 0, c1, c2, b0}, fr
}

// QJoinLeaf lifts q back to 𝕊 via liftLeaf u₁ u₂ (Theorem 4).
func QJoinLeaf(q Leaf, fr *LeafFrame) (s [16]float64) {
	for k := 1; k < 8; k++ {
		s[k] = q[0]*fr.U1[k] + q[1]*fr.U2[k]
		s[8+k] = q[2]*fr.U1[k] + q[3]*fr.U2[k]
	}
	s[8] = q[4]
	return
}

// LeafEscape returns r² = s₀² + ‖a − P a‖² + ‖Im b − P Im b‖² against the
// CSR-LOCKED frame (not a fresh extraction — any (a, Im b) spans *some* plane).
func LeafEscape(s *[16]float64, fr *LeafFrame) float64 {
	r2 := s[0] * s[0]
	for _, off := range []int{0, 8} {
		var v [8]float64
		copy(v[1:], s[off+1:off+8])
		var p1, p2 float64
		for k := range v {
			p1 += v[k] * fr.U1[k]
			p2 += v[k] * fr.U2[k]
		}
		for k := range v {
			d := v[k] - p1*fr.U1[k] - p2*fr.U2[k]
			r2 += d * d
		}
	}
	return r2
}
