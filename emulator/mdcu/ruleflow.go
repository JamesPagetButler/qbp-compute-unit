package mdcu

import "math"

// ---------------------------------------------------------------------------
// #688 sedenion rule-flow: V(s) = N([a,b]), s = (a,b) ∈ 𝕊, on the sphere
// {N s = 1, s₀ = 0}. Three engines, all integrated with the reference RK4
// (doc/spikes/ref/benchmark_688_integrators.py: per-stage s₀=0 + renorm).
//
//   Basis : flowlib.gradV_exact — 34 OMUL / eval (doc convention 2,176 MACs)
//   CDMAC : adjoint form g_a = −2[c,b], g_b = +2[c,a] — 6 OMUL / eval (384)
//   Leaf  : 5D leaf field — 12 MACs / eval (+32 split, +32 join per seed)
// ---------------------------------------------------------------------------

// Counter tallies instrumented work (nil = no counting; zero overhead path).
type Counter struct {
	OMUL    uint64 // octonion products issued
	FMul    uint64 // every float multiply (incl. renorm, radial, RK4 weights)
	DocMACs uint64 // MACs under the Case-doc convention
}

func (c *Counter) omul(n uint64) {
	if c != nil {
		c.OMUL += n
		c.FMul += 64 * n
	}
}
func (c *Counter) fmul(n uint64) {
	if c != nil {
		c.FMul += n
	}
}
func (c *Counter) doc(n uint64) {
	if c != nil {
		c.DocMACs += n
	}
}

func splitAB(s *[16]float64) (a, b [8]float64) {
	copy(a[:], s[:8])
	copy(b[:], s[8:])
	return
}

func comm8(x, y *[8]float64) (z [8]float64) {
	p, q := OMulCDMAC(x, y), OMulCDMAC(y, x)
	for i := range z {
		z[i] = p[i] - q[i]
	}
	return
}

// GradBasis is the flowlib.gradV_exact port: 2 + 16 + 16 = 34 OMULs.
func GradBasis(s *[16]float64, ctr *Counter) (g [16]float64) {
	a, b := splitAB(s)
	c := comm8(&a, &b)
	ctr.omul(2)
	for i := 0; i < 8; i++ {
		var e [8]float64
		e[i] = 1
		eb := comm8(&e, &b)
		ae := comm8(&a, &e)
		ctr.omul(4)
		var d1, d2 float64
		for k := 0; k < 8; k++ {
			d1 += c[k] * eb[k]
			d2 += c[k] * ae[k]
		}
		ctr.fmul(18)
		g[i], g[8+i] = 2*d1, 2*d2
	}
	ctr.doc(34 * 64)
	return
}

// GradCDMAC is the adjoint form: c = [a,b]; g_a = −2[c,b]; g_b = +2[c,a].
// 3 commutators = 6 OMULs.
func GradCDMAC(s *[16]float64, ctr *Counter) (g [16]float64) {
	a, b := splitAB(s)
	c := comm8(&a, &b)
	cb := comm8(&c, &b)
	ca := comm8(&c, &a)
	ctr.omul(6)
	for k := 0; k < 8; k++ {
		g[k], g[8+k] = -2*cb[k], 2*ca[k]
	}
	ctr.fmul(16)
	ctr.doc(6 * 64)
	return
}

// ruleField16: F = −(g − ⟨g,s⟩s), F₀ = 0.
func ruleField16(s *[16]float64, grad func(*[16]float64, *Counter) [16]float64, ctr *Counter) (f [16]float64) {
	g := grad(s, ctr)
	var gs float64
	for k := range g {
		gs += g[k] * s[k]
	}
	for k := range f {
		f[k] = -(g[k] - gs*s[k])
	}
	f[0] = 0
	ctr.fmul(32)
	return
}

func renorm16(x *[16]float64, ctr *Counter) {
	x[0] = 0
	var n float64
	for _, v := range x {
		n += v * v
	}
	n = math.Sqrt(n)
	for k := range x {
		x[k] /= n
	}
	ctr.fmul(32)
}

// RK4Step16 is rk4_step_16d.
func RK4Step16(s [16]float64, h float64, grad func(*[16]float64, *Counter) [16]float64, ctr *Counter) [16]float64 {
	stage := func(base *[16]float64, c float64, k *[16]float64) (o [16]float64) {
		for i := range o {
			o[i] = base[i] + c*k[i]
		}
		ctr.fmul(16)
		renorm16(&o, ctr)
		return
	}
	k1 := ruleField16(&s, grad, ctr)
	s2 := stage(&s, 0.5*h, &k1)
	k2 := ruleField16(&s2, grad, ctr)
	s3 := stage(&s, 0.5*h, &k2)
	k3 := ruleField16(&s3, grad, ctr)
	s4 := stage(&s, h, &k3)
	k4 := ruleField16(&s4, grad, ctr)
	var t [16]float64
	for i := range t {
		t[i] = s[i] + (h/6.0)*(k1[i]+2.0*k2[i]+2.0*k3[i]+k4[i])
	}
	ctr.fmul(16 * 4)
	t[0] = 0
	renorm16(&t, ctr)
	return t
}

// LeafRuleField is leaf_ruleField (12 doc-MACs).
func LeafRuleField(q *Leaf, ctr *Counter) (f Leaf) {
	a1, a2, c1, c2 := q[0], q[1], q[2], q[3]
	det := a1*c2 - a2*c1
	g := Leaf{8.0 * det * c2, -8.0 * det * c1, -8.0 * det * a2, 8.0 * det * a1, 0}
	var rad float64
	for k := range g {
		rad += g[k] * q[k]
	}
	for k := range f {
		f[k] = -(g[k] - rad*q[k])
	}
	ctr.fmul(2 + 8 + 5 + 5)
	ctr.doc(12)
	return
}

func renorm5(x *Leaf, ctr *Counter) {
	var n float64
	for _, v := range x {
		n += v * v
	}
	n = math.Sqrt(n)
	for k := range x {
		x[k] /= n
	}
	ctr.fmul(10)
}

// RK4StepLeaf is rk4_step_leaf.
func RK4StepLeaf(q Leaf, h float64, ctr *Counter) Leaf {
	stage := func(c float64, k *Leaf) (o Leaf) {
		for i := range o {
			o[i] = q[i] + c*k[i]
		}
		ctr.fmul(5)
		renorm5(&o, ctr)
		return
	}
	k1 := LeafRuleField(&q, ctr)
	q2 := stage(0.5*h, &k1)
	k2 := LeafRuleField(&q2, ctr)
	q3 := stage(0.5*h, &k2)
	k3 := LeafRuleField(&q3, ctr)
	q4 := stage(h, &k3)
	k4 := LeafRuleField(&q4, ctr)
	var qn Leaf
	for i := range qn {
		qn[i] = q[i] + (h/6.0)*(k1[i]+2.0*k2[i]+2.0*k3[i]+k4[i])
	}
	ctr.fmul(5 * 4)
	renorm5(&qn, ctr)
	return qn
}

// Potential is V(s) = N([a,b]).
func Potential(s *[16]float64) float64 {
	a, b := splitAB(s)
	c := comm8(&a, &b)
	return norm2(c[:])
}

// Engine selects a rule-flow integration path.
type Engine int

const (
	EngBasis Engine = iota
	EngCDMAC
	EngLeaf
)

func (e Engine) String() string { return [...]string{"basis", "cdmac", "leaf"}[e] }

// RunQuench integrates every seed for `steps` RK4 steps with engine e and
// returns the final 16D states.
func RunQuench(seeds [][16]float64, h float64, steps int, e Engine, ctr *Counter) [][16]float64 {
	out := make([][16]float64, len(seeds))
	for i, s := range seeds {
		switch e {
		case EngBasis, EngCDMAC:
			g := GradBasis
			if e == EngCDMAC {
				g = GradCDMAC
			}
			for t := 0; t < steps; t++ {
				s = RK4Step16(s, h, g, ctr)
			}
			out[i] = s
		case EngLeaf:
			q, fr := QSplitLeaf(&s)
			ctr.doc(32)
			for t := 0; t < steps; t++ {
				q = RK4StepLeaf(q, h, ctr)
			}
			out[i] = QJoinLeaf(q, &fr)
			ctr.doc(32)
		}
	}
	return out
}
