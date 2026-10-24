package mdcu

import "math"

// ---------------------------------------------------------------------------
// Round-2 Q2 (frozen plan v3 section 4): Albert-algebra determinant
//
//	det(X) = abc - a|x|^2 - b|y|^2 - c|z|^2 + 2 Re((xy)z)
//
// B1 = CD-MAC path (octonion product from the sign ROM).
// B0 = fastest non-CD exact path (3-form contraction, no CD multiplier).
//
// Every evaluator takes a *Tape. The Tape records one node per "tile batch"
// (at most TileLanes multiplies, one tile-cycle) with its dependencies, while
// the arithmetic is being done. The MAC count and the model-cycle DAG
// therefore come from the SAME code path (qbp-cu-implementor seq 4).
// A nil tape disables recording (host-ns benchmarks).
//
// Cost model (assumed, symmetric for both sides, NOT silicon): a tile has
// TileLanes=64 multiplier lanes and an adder tree, so one node (<= 64
// multiplies plus the adds that reduce them) costs one tile-cycle. This is the
// same assumption that gives OMUL its tariff of 1 in isa.go.
// ---------------------------------------------------------------------------

// TileLanes is the multiplier width of one tile (the OMUL CD-MAC is 8x8).
const TileLanes = 64

// TNode is one tile batch.
type TNode struct {
	Mults int
	Deps  []int
}

// Tape records the DAG of a computation.
type Tape struct{ Nodes []TNode }

func (t *Tape) add(mults int, deps ...int) int {
	if t == nil {
		return -1
	}
	if mults > TileLanes {
		panic("mdcu: tape node wider than a tile")
	}
	d := append([]int(nil), deps...)
	t.Nodes = append(t.Nodes, TNode{Mults: mults, Deps: d})
	return len(t.Nodes) - 1
}

// TotalMults is the multiply (MAC) count recorded on the tape.
func (t *Tape) TotalMults() (n int) {
	for _, x := range t.Nodes {
		n += x.Mults
	}
	return
}

// Q2Inst is one Albert-determinant instance.
type Q2Inst struct {
	D       [3]float64
	X, Y, Z [8]float64
}

// NewQ2Insts draws n instances with N(0,1) components (deterministic).
func NewQ2Insts(n int, seed uint64) []Q2Inst {
	r := newRNG(seed)
	out := make([]Q2Inst, n)
	for i := range out {
		for k := range out[i].D {
			out[i].D[k] = r.NormFloat64()
		}
		for k := 0; k < 8; k++ {
			out[i].X[k] = r.NormFloat64()
			out[i].Y[k] = r.NormFloat64()
			out[i].Z[k] = r.NormFloat64()
		}
	}
	return out
}

func dot8(a, b *[8]float64) (s float64) {
	for i := 0; i < 8; i++ {
		s += a[i] * b[i]
	}
	return
}

// q2common is the part shared by every variant: three norms and abc (26
// multiplies, one batch), then three d*norm products (3 multiplies, one
// batch). Returns (scalar part, tape node of the last common batch).
func q2common(in *Q2Inst, t *Tape) (float64, int) {
	nx, ny, nz := dot8(&in.X, &in.X), dot8(&in.Y, &in.Y), dot8(&in.Z, &in.Z)
	abc := in.D[0] * in.D[1] * in.D[2]
	c1 := t.add(8*3 + 2)
	v := abc - in.D[0]*nx - in.D[1]*ny - in.D[2]*nz
	c2 := t.add(3, c1)
	return v, c2
}

// finish adds the combine batch (adds only, 0 multiplies) that depends on the
// common part and on the Re((xy)z) node. Same for every variant.
func finish(t *Tape, common, re int) { t.add(0, common, re) }

// DetB1Lit: the ISA-literal CD-MAC path. Two OMUL ops (64 multiplies each);
// the scalar part of the second product is taken.
func DetB1Lit(in *Q2Inst, t *Tape) float64 {
	v, c := q2common(in, t)
	w := OMulCDMAC(&in.X, &in.Y)
	m1 := t.add(MACsOMulCDMAC)
	u := OMulCDMAC(&w, &in.Z)
	m2 := t.add(MACsOMulCDMAC, m1)
	finish(t, c, m2)
	return v + 2*u[0]
}

// DetB1Lean: one OMUL, then the scalar part of (xy)z as an 8-term dot with
// the Re(e_i e_i)=s(i,i) signs (7 imaginary terms negative). Sensitivity
// variant; it uses a dot op the ISA also has (Albert fast uses dots).
func DetB1Lean(in *Q2Inst, t *Tape) float64 {
	v, c := q2common(in, t)
	w := OMulCDMAC(&in.X, &in.Y)
	m1 := t.add(MACsOMulCDMAC)
	re := w[0] * in.Z[0]
	for i := 1; i < 8; i++ {
		re -= w[i] * in.Z[i]
	}
	m2 := t.add(8, m1)
	finish(t, c, m2)
	return v + 2*re
}

// B0 tensor: T[i][j] = s(i,j) with k = i^j and Re(e_k e_k) folded in, so that
// Re((xy)z) = sum_{i,j} s(i,j) s(k,k) x_i y_j z_k, k = i^j. Fixed sparse
// 3-form (64 nonzeros), no CD multiplier.
var b0sign [8][8]float64

func init() {
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			k := i ^ j
			b0sign[i][j] = signTable[i][j] * signTable[k][k]
		}
	}
}

// DetB0a: full sparse 3-form contraction, two multiplies per nonzero term
// (x_i*y_j first, then *z_k), 128 multiplies in two dependent batches.
func DetB0a(in *Q2Inst, t *Tape) float64 {
	v, c := q2common(in, t)
	var p [8][8]float64
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			p[i][j] = in.X[i] * in.Y[j]
		}
	}
	s1 := t.add(64)
	var re float64
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			re += b0sign[i][j] * p[i][j] * in.Z[i^j]
		}
	}
	s2 := t.add(64, s1)
	finish(t, c, s2)
	return v + 2*re
}

// b0lines: the 7 Fano lines {a,b,c=a^b}, a<b<c, with orientation sign
// sigma = s(a,b) so that the imaginary triple product is
// sum_lines sigma * det[x;y;z restricted to the line].
var b0lines [7][4]int // a,b,c and sign encoded in [3] as +-1

func init() {
	n := 0
	for a := 1; a < 8; a++ {
		for b := a + 1; b < 8; b++ {
			c := a ^ b
			if c > b {
				b0lines[n] = [4]int{a, b, c, int(signTable[a][b])}
				n++
			}
		}
	}
	if n != 7 {
		panic("mdcu: Fano line enumeration failed")
	}
}

// DetB0b: factored scalar/vector form
//
//	Re(xyz) = x0y0z0 - x0(y.z) - y0(x.z) - z0(x.y) - (x cross y).z
//
// where the triple product is a sum of seven 3x3 determinants (one per Fano
// line, 9 multiplies each by cofactor expansion). 89 multiplies in two
// dependent batches: 64 (dots, 2x2 minors, x0y0) then 25 (cofactors, scalings).
func DetB0b(in *Q2Inst, t *Tape) float64 {
	v, c := q2common(in, t)
	var yz, xz, xy float64
	for i := 1; i < 8; i++ {
		yz += in.Y[i] * in.Z[i]
		xz += in.X[i] * in.Z[i]
		xy += in.X[i] * in.Y[i]
	}
	x0y0 := in.X[0] * in.Y[0]
	type minors struct{ m [3]float64 }
	var mn [7]minors
	for l := 0; l < 7; l++ {
		a, b, cc := b0lines[l][0], b0lines[l][1], b0lines[l][2]
		// det rows x,y,z over columns a,b,c, expanded along row x.
		mn[l].m[0] = in.Y[b]*in.Z[cc] - in.Y[cc]*in.Z[b]
		mn[l].m[1] = in.Y[a]*in.Z[cc] - in.Y[cc]*in.Z[a]
		mn[l].m[2] = in.Y[a]*in.Z[b] - in.Y[b]*in.Z[a]
	}
	s1 := t.add(21 + 42 + 1) // dots, minors, x0*y0
	var trip float64
	for l := 0; l < 7; l++ {
		a, b, cc := b0lines[l][0], b0lines[l][1], b0lines[l][2]
		d := in.X[a]*mn[l].m[0] - in.X[b]*mn[l].m[1] + in.X[cc]*mn[l].m[2]
		trip += float64(b0lines[l][3]) * d
	}
	re := x0y0*in.Z[0] - in.X[0]*yz - in.Y[0]*xz - in.Z[0]*xy - trip
	s2 := t.add(21 + 1 + 3 /* cofactors, x0y0*z0, three scalings */, s1)
	finish(t, c, s2)
	return v + 2*re
}

// Q2Variant names an evaluator.
type Q2Variant struct {
	Name string
	Side string // "B1" or "B0"
	Fn   func(*Q2Inst, *Tape) float64
}

// Q2Variants lists every variant, in reporting order.
var Q2Variants = []Q2Variant{
	{"B1lit", "B1", DetB1Lit},
	{"B1lean", "B1", DetB1Lean},
	{"B0a", "B0", DetB0a},
	{"B0b", "B0", DetB0b},
}

// ---------------------------------------------------------------------------
// Scheduler bounds for an arbitrary tape (sched.go only takes level lists).
//
// (i)  ideal throughput: ceil(total multiplies / (TileLanes*P)), perfect lane
//      packing, no dependency stalls.
// (ii) dependency-aware: list scheduling on P tiles, one batch per tile-cycle,
//      nodes issued in tape order among the ready ones.
// ---------------------------------------------------------------------------

// IdealCycles is bound (i).
func (t *Tape) IdealCycles(P int) int {
	m := t.TotalMults()
	d := TileLanes * P
	return (m + d - 1) / d
}

// DepCycles is bound (ii) (makespan of the greedy list schedule).
func (t *Tape) DepCycles(P int) int {
	n := len(t.Nodes)
	done := make([]int, n) // cycle at which node n finishes (0 = not scheduled)
	sched := make([]bool, n)
	left, cyc := n, 0
	for left > 0 {
		cyc++
		used := 0
		for i := 0; i < n && used < P; i++ {
			if sched[i] {
				continue
			}
			ok := true
			for _, d := range t.Nodes[i].Deps {
				if !sched[d] || done[d] >= cyc {
					ok = false
					break
				}
			}
			if ok {
				sched[i], done[i] = true, cyc
				used++
				left--
			}
		}
	}
	return cyc
}

// CritPath is the dependency depth in batches (lower bound for any P).
func (t *Tape) CritPath() (best int) {
	depth := make([]int, len(t.Nodes))
	for i, n := range t.Nodes {
		d := 0
		for _, p := range n.Deps {
			if depth[p] > d {
				d = depth[p]
			}
		}
		depth[i] = d + 1
		if depth[i] > best {
			best = depth[i]
		}
	}
	return
}

// Q2Agree returns the worst normwise relative disagreement between every
// pair of variants over the instances (agreement gate, threshold 1e-12).
// Scale is |abc|+sum|d||n|+2*|x||y||z| so that cancellation is not hidden.
func Q2Agree(ins []Q2Inst) float64 {
	worst := 0.0
	for k := range ins {
		in := &ins[k]
		vals := make([]float64, len(Q2Variants))
		for i, v := range Q2Variants {
			vals[i] = v.Fn(in, nil)
		}
		nx, ny, nz := math.Sqrt(dot8(&in.X, &in.X)), math.Sqrt(dot8(&in.Y, &in.Y)), math.Sqrt(dot8(&in.Z, &in.Z))
		scale := math.Abs(in.D[0]*in.D[1]*in.D[2]) + math.Abs(in.D[0])*nx*nx + math.Abs(in.D[1])*ny*ny +
			math.Abs(in.D[2])*nz*nz + 2*nx*ny*nz
		for i := 1; i < len(vals); i++ {
			worst = math.Max(worst, math.Abs(vals[i]-vals[0])/scale)
		}
	}
	return worst
}
