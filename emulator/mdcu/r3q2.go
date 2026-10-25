package mdcu

// Round-3 R3-Q2 (frozen §3): efficiency of the CD-MAC vs non-CD exact paths.
//
//	W1  Albert determinant scalar   CD: B1lean        non-CD: best of B0b / B0cross
//	W2  φ(a,b,c) scalar             CD: OMulCDMAC(a,b)·c   non-CD: a×b from 21 minors, dot c
//	W3  associator vector [a,b,c]   CD: 4×OMulCDMAC        non-CD: coassociative 4-form
//	                                    contraction (21 minors, 28 three-term expansions)
//
// Counting rule (frozen §3.2): data×data real multiplies only; ±1 (sign ROM)
// and ×2 are not counted. Counts come from a Tape recorded in the same code path
// that does the arithmetic. Round-2 code (q2.go) is reused, not modified.

import "math"

// V3In is one imaginary-vector triple (index 0 is exactly zero).
type V3In struct{ A, B, C [8]float64 }

// NewV3Ins draws n deterministic triples of imaginary N(0,1) vectors.
func NewV3Ins(n int, seed uint64) []V3In {
	r := newRNG(seed)
	out := make([]V3In, n)
	for i := range out {
		for k := 1; k < 8; k++ {
			out[i].A[k] = r.NormFloat64()
			out[i].B[k] = r.NormFloat64()
			out[i].C[k] = r.NormFloat64()
		}
	}
	return out
}

// ----------------------------------------------------------------- tables

// crossTab[i-1] lists, for imaginary index i, the three pairs j<k with j^k=i
// and the sign s(j,k) of e_j e_k = s e_i. imag(xy)_i = Σ s·(x_j y_k − x_k y_j).
var crossTab [7][3][3]int

// minorIdx maps an index pair j<k (1..7) to its slot in the 21-minor array.
var minorIdx [8][8]int

// assocTab: the 28 non-line triples T=(t0<t1<t2), the output index m (the
// fourth member of the coassociative quadruple), and sign ±1 such that
// [a,b,c]_m += 2·sign·det3(a,b,c over columns T). Derived at init from assoc8
// on basis vectors, so no sign is taken on trust.
type assocEntry struct {
	t       [3]int
	m, sign int
}

var assocTab []assocEntry

func init() {
	n := 0
	for j := 1; j < 8; j++ {
		for k := j + 1; k < 8; k++ {
			minorIdx[j][k] = n
			n++
		}
	}
	cnt := [8]int{}
	for j := 1; j < 8; j++ {
		for k := j + 1; k < 8; k++ {
			i := j ^ k
			crossTab[i-1][cnt[i]] = [3]int{j, k, int(signTable[j][k])}
			cnt[i]++
		}
	}
	for i := 1; i < 8; i++ {
		if cnt[i] != 3 {
			panic("mdcu: cross table construction failed")
		}
	}
	for a := 1; a < 8; a++ {
		for b := a + 1; b < 8; b++ {
			for c := b + 1; c < 8; c++ {
				if a^b == c {
					continue // Fano line: associative, zero associator
				}
				e := func(i int) []float64 { v := make([]float64, 8); v[i] = 1; return v }
				v := assoc8(e(a), e(b), e(c))
				m, hit := -1, 0
				for k := 0; k < 8; k++ {
					if math.Abs(v[k]) > 1e-9 {
						m, hit = k, hit+1
					}
				}
				if hit != 1 || math.Abs(math.Abs(v[m])-2) > 1e-12 {
					panic("mdcu: associator of basis triple is not ±2 e_m")
				}
				assocTab = append(assocTab, assocEntry{[3]int{a, b, c}, m, int(math.Copysign(1, v[m]))})
			}
		}
	}
	if len(assocTab) != 28 {
		panic("mdcu: expected 28 non-line triples")
	}
}

// minors fills mn[k] = x_j y_k' − x_k' y_j... precisely mn[minorIdx[j][k]] = x_j y_k − x_k y_j.
func minors21(x, y *[8]float64, mn *[21]float64) {
	for j := 1; j < 8; j++ {
		for k := j + 1; k < 8; k++ {
			mn[minorIdx[j][k]] = x[j]*y[k] - x[k]*y[j]
		}
	}
}

// crossImag computes imag(xy) (7 components, index 1..7) from the 21 minors.
func crossImag(mn *[21]float64) (c [8]float64) {
	for i := 1; i < 8; i++ {
		for _, p := range crossTab[i-1] {
			c[i] += float64(p[2]) * mn[minorIdx[p[0]][p[1]]]
		}
	}
	return
}

// --------------------------------------------------------------------- W2

// W2CD: φ(a,b,c) = <ab, c> with ab from the CD-MAC and a 7-term imaginary dot
// (a,b,c are imaginary, so the real component of the dot is known zero; this is
// the CD path's best case, not a handicap).
func W2CD(in *V3In, t *Tape) float64 {
	w := OMulCDMAC(&in.A, &in.B)
	m1 := t.add(MACsOMulCDMAC)
	s := 0.0
	for i := 1; i < 8; i++ {
		s += w[i] * in.C[i]
	}
	t.add(7, m1)
	return s
}

// W2NonCD: <a×b, c> from the 21 shared minors (42 multiplies) and a 7-term dot.
func W2NonCD(in *V3In, t *Tape) float64 {
	var mn [21]float64
	minors21(&in.A, &in.B, &mn)
	m1 := t.add(42)
	cr := crossImag(&mn)
	s := 0.0
	for i := 1; i < 8; i++ {
		s += cr[i] * in.C[i]
	}
	t.add(7, m1)
	return s
}

// --------------------------------------------------------------------- W3

// W3CD: [a,b,c] = (ab)c − a(bc): four CD-MAC products, two independent pairs.
func W3CD(in *V3In, t *Tape) (r [8]float64) {
	ab := OMulCDMAC(&in.A, &in.B)
	n1 := t.add(MACsOMulCDMAC)
	bc := OMulCDMAC(&in.B, &in.C)
	n2 := t.add(MACsOMulCDMAC)
	l := OMulCDMAC(&ab, &in.C)
	t.add(MACsOMulCDMAC, n1)
	rr := OMulCDMAC(&in.A, &bc)
	t.add(MACsOMulCDMAC, n2)
	for i := range r {
		r[i] = l[i] - rr[i]
	}
	return
}

// W3NonCD: coassociative 4-form contraction. 21 minors of (b,c) (42 multiplies),
// then 28 three-term expansions of a over each non-line triple (84 multiplies).
func W3NonCD(in *V3In, t *Tape) (r [8]float64) {
	var mn [21]float64
	minors21(&in.B, &in.C, &mn)
	m1 := t.add(42)
	for _, e := range assocTab {
		a0, a1, a2 := e.t[0], e.t[1], e.t[2]
		d := in.A[a0]*mn[minorIdx[a1][a2]] - in.A[a1]*mn[minorIdx[a0][a2]] + in.A[a2]*mn[minorIdx[a0][a1]]
		r[e.m] += 2 * float64(e.sign) * d
	}
	// 84 multiplies exceed one tile: record as 64 + 20 (same dependency).
	m2 := t.add(64, m1)
	t.add(20, m1, m2)
	return
}

// -------------------------------------------------------------------- W1

// DetB0Cross is the cross-product non-CD path for the Albert determinant:
//
//	Re(xyz) = x0y0z0 − x0(y·z) − y0(x·z) − z0(x·y) − x·(y×z)
//
// with y×z from the 21 minors (42) and the triple product as a 7-term dot
// (7): 3 dots (21) + 42 + 7 + x0y0z0 (2) + 3 scalings = 75 multiplies for the
// Re part, 104 with the 29 common multiplies.
func DetB0Cross(in *Q2Inst, t *Tape) float64 {
	v, c := q2common(in, t)
	var yz, xz, xy float64
	for i := 1; i < 8; i++ {
		yz += in.Y[i] * in.Z[i]
		xz += in.X[i] * in.Z[i]
		xy += in.X[i] * in.Y[i]
	}
	var mn [21]float64
	minors21(&in.Y, &in.Z, &mn)
	cr := crossImag(&mn)
	s1 := t.add(21 + 42) // dots + minors
	trip := 0.0
	for i := 1; i < 8; i++ {
		trip += in.X[i] * cr[i]
	}
	re := in.X[0]*in.Y[0]*in.Z[0] - in.X[0]*yz - in.Y[0]*xz - in.Z[0]*xy - trip
	s2 := t.add(7+2+3, s1)
	finish(t, c, s2)
	return v + 2*re
}

// R3W1Paths: every W1 evaluator the closure considers.
var R3W1Paths = []Q2Variant{
	{"B1lean", "B1", DetB1Lean},
	{"B0b", "B0", DetB0b},
	{"B0cross", "B0", DetB0Cross},
}

// ---------------------------------------------------------------- closure

// R3Q2Row is one workload's deterministic multiply-count closure.
type R3Q2Row struct {
	Workload    string
	CDPath      string
	NonCDPath   string
	CDMults     int
	NonCDMults  int
	CDAdvantage float64 // best non-CD mults / best CD mults
	CDYes       bool    // "CD has a multiply-count advantage of 2× or more"
	// Analytic add counts (hand-derived, NOT tape-verified; no verdict uses them).
	CDAdds, NonCDAdds int
}

// R3Q2Closure computes the three rows from the tapes. Fixed out-of-grid inputs.
func R3Q2Closure() []R3Q2Row {
	in3 := NewV3Ins(1, 9810)[0]
	in1 := NewQ2Insts(1, 9811)[0]
	count := func(f func(*Tape)) int { t := &Tape{}; f(t); return t.TotalMults() }

	row := func(w, cdp, ncp string, cd, nc, cda, nca int) R3Q2Row {
		adv := float64(nc) / float64(cd)
		return R3Q2Row{w, cdp, ncp, cd, nc, adv, adv >= 2, cda, nca}
	}
	b0b := count(func(t *Tape) { DetB0b(&in1, t) })
	b0x := count(func(t *Tape) { DetB0Cross(&in1, t) })
	ncp, nc1, nca1 := "B0b", b0b, 0
	if b0x < b0b {
		ncp, nc1, nca1 = "B0cross", b0x, 88
	}
	return []R3Q2Row{
		row("W1 Albert determinant", "B1lean", ncp,
			count(func(t *Tape) { DetB1Lean(&in1, t) }), nc1, 88, nca1),
		row("W2 φ scalar", "OMulCDMAC+dot7", "minors21+dot7",
			count(func(t *Tape) { W2CD(&in3, t) }), count(func(t *Tape) { W2NonCD(&in3, t) }), 62, 41),
		row("W3 associator vector", "4×OMulCDMAC", "4-form contraction",
			count(func(t *Tape) { W3CD(&in3, t) }), count(func(t *Tape) { W3NonCD(&in3, t) }), 232, 98),
	}
}
