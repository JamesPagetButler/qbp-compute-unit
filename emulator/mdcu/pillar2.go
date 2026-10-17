package mdcu

import (
	"math"
	"sort"
)

// Pillar 2 ports (Test A/B/C), built from the web-Gemini prose spec. The
// original fix_test_a.py / run_fault_line_tests.py sources were never shared,
// so loop-boundary semantics are inferred. Every result is compared to the
// canonical results.json; a mismatch is reported, never tuned away.

func assoc8(x, y, z []float64) []float64 { return QAssoc(TierO, x, y, z) }

func basis(n, b int) []float64 { e := make([]float64, n); e[b] = 1; return e }

func dot(a, b []float64) (s float64) {
	for i := range a {
		s += a[i] * b[i]
	}
	return
}

func normalize(v []float64) {
	n := math.Sqrt(norm2(v))
	for i := range v {
		v[i] /= n
	}
}

// ---------------------------------------------------------------- Test A

type BounceResult struct {
	Mode             string
	Bounced, Plunged bool
	RMin, FinalR     float64
	InitA2, FinalA2  float64
	Steps            int
	QAssoc           uint64
}

// RunBounce runs one Test A regime. ispin ≤ 0 ⇒ Z frozen (g2_conserved);
// pure ⇒ ‖A‖² ≡ 0.
func RunBounce(mode string, z0 [3][]float64, pure bool, ispin float64) BounceResult {
	const (
		dt, G, alpha, kappa = 5e-4, 1.0, 0.08, 0.04
		maxSteps            = 6000
	)
	Z := [3][]float64{}
	V := [3][]float64{}
	for k := range Z {
		Z[k] = append([]float64(nil), z0[k]...)
		V[k] = make([]float64, 8)
	}
	res := BounceResult{Mode: mode}
	a2 := func() float64 {
		if pure {
			return 0
		}
		res.QAssoc++
		return norm2(assoc8(Z[0], Z[1], Z[2]))
	}
	force := func(r, A2 float64) float64 {
		return -G/(r*r) - 3*alpha/math.Pow(r, 4) + 2*kappa*A2/math.Pow(r, 5)
	}
	r0 := 1.5
	r, vr := r0, -0.6
	res.InitA2 = a2()
	res.RMin = r
	for st := 1; st <= maxSteps; st++ {
		res.Steps = st
		if r <= 0.05 {
			res.Plunged = true
			break
		}
		var A []float64
		A2 := 0.0
		if !pure {
			A = assoc8(Z[0], Z[1], Z[2])
			res.QAssoc++
			A2 = norm2(A)
		}
		F := force(r, A2)
		if !pure && ispin > 0 {
			var G3 [3][]float64
			for k := 0; k < 3; k++ { // all gradients from PRE-update Z
				g := make([]float64, 8)
				for b := 0; b < 8; b++ {
					args := [3][]float64{Z[0], Z[1], Z[2]}
					args[k] = basis(8, b)
					g[b] = dot(assoc8(args[0], args[1], args[2]), A)
					res.QAssoc++
				}
				p := dot(g, Z[k])
				for i := range g {
					g[i] -= p * Z[k][i]
				}
				G3[k] = g
			}
			pref := kappa / (math.Pow(r, 4) * ispin)
			for k := 0; k < 3; k++ {
				for i := range G3[k] {
					V[k][i] = 0.95*V[k][i] - pref*G3[k][i]*dt
					Z[k][i] += V[k][i] * dt
				}
				normalize(Z[k])
			}
		}
		vh := vr + 0.5*F*dt
		rn := r + vh*dt
		if rn <= 0.05 {
			r = math.Max(rn, 0)
			res.Plunged = true
			break
		}
		res.RMin = math.Min(res.RMin, rn)
		if vr < 0 && vh >= 0 {
			res.Bounced = true
		}
		vr = vh + 0.5*force(rn, a2())*dt
		r = rn
		if res.Bounced && r >= r0 {
			break
		}
	}
	res.FinalR = r
	res.FinalA2 = a2()
	return res
}

// ---------------------------------------------------------------- Test B

type LockMetrics struct{ Local, Global, SVD3 float64 }

func ringMetrics(Z [][]float64, triads [3][]int) LockMetrics {
	n := len(Z)
	var m LockMetrics
	for i := 0; i < n; i++ {
		m.Local += norm2(assoc8(Z[(i-1+n)%n], Z[i], Z[(i+1)%n]))
	}
	m.Local /= float64(n)
	for t := range triads[0] {
		m.Global += norm2(assoc8(Z[triads[0][t]], Z[triads[1][t]], Z[triads[2][t]]))
	}
	m.Global /= float64(len(triads[0]))
	// top-3 share of singular values² of Z[:,1:] = top-3 eigenvalues of Gram.
	var g [7][7]float64
	for _, z := range Z {
		for a := 0; a < 7; a++ {
			for b := 0; b < 7; b++ {
				g[a][b] += z[a+1] * z[b+1]
			}
		}
	}
	ev := jacobiEig7(g)
	sort.Sort(sort.Reverse(sort.Float64Slice(ev)))
	tot := 0.0
	for _, e := range ev {
		tot += e
	}
	m.SVD3 = (ev[0] + ev[1] + ev[2]) / tot
	return m
}

func jacobiEig7(a [7][7]float64) []float64 {
	for sweep := 0; sweep < 100; sweep++ {
		off := 0.0
		for p := 0; p < 7; p++ {
			for q := p + 1; q < 7; q++ {
				off += a[p][q] * a[p][q]
			}
		}
		if off < 1e-30 {
			break
		}
		for p := 0; p < 7; p++ {
			for q := p + 1; q < 7; q++ {
				if a[p][q] == 0 {
					continue
				}
				th := (a[q][q] - a[p][p]) / (2 * a[p][q])
				t := math.Copysign(1, th) / (math.Abs(th) + math.Sqrt(th*th+1))
				c := 1 / math.Sqrt(t*t+1)
				s := t * c
				for k := 0; k < 7; k++ {
					akp, akq := a[k][p], a[k][q]
					a[k][p], a[k][q] = c*akp-s*akq, s*akp+c*akq
				}
				for k := 0; k < 7; k++ {
					apk, aqk := a[p][k], a[q][k]
					a[p][k], a[q][k] = c*apk-s*aqk, s*apk+c*aqk
				}
			}
		}
	}
	ev := make([]float64, 7)
	for i := range ev {
		ev[i] = a[i][i]
	}
	return ev
}

// RunRing runs Test B regime (overlap=false: local only; true: locked).
func RunRing(z0 [][]float64, overlap bool) [][]float64 {
	const lr, mu, steps = 0.04, 1.5, 250
	n := len(z0)
	Z := make([][]float64, n)
	for i := range Z {
		Z[i] = append([]float64(nil), z0[i]...)
	}
	for st := 0; st < steps; st++ {
		grad := make([][]float64, n)
		for i := 0; i < n; i++ {
			l, r, ll := Z[(i-1+n)%n], Z[(i+1)%n], Z[(i-2+n)%n]
			g := make([]float64, 8)
			cl, cr := dot(Z[i], l), dot(Z[i], r)
			for k := range g {
				g[k] = 2 * mu * (cr*r[k] + cl*l[k])
			}
			A := assoc8(l, Z[i], r)
			var A2 []float64
			if overlap {
				A2 = assoc8(ll, l, Z[i])
			}
			for b := 1; b < 8; b++ {
				e := basis(8, b)
				g[b] += 2 * dot(A, assoc8(l, e, r))
				if overlap {
					g[b] += 2 * dot(A2, assoc8(ll, l, e))
				}
			}
			g[0] = 0
			p := dot(g, Z[i])
			for k := range g {
				g[k] -= p * Z[i][k]
			}
			grad[i] = g
		}
		for i := range Z {
			for k := range Z[i] {
				Z[i][k] -= lr * grad[i][k]
			}
			normalize(Z[i])
		}
	}
	return Z
}

// ---------------------------------------------------------------- Test C

type SieveMetrics struct{ Norm, Alt, Assoc, H, O, S float64 }

func sieveMetrics(W [][]float64) (m SieveMetrics) {
	M := len(W)
	for i := 0; i < M; i++ {
		x, y, z := W[i], W[(i-1+M)%M], W[(i-2+M)%M]
		m.Norm += math.Abs(math.Sqrt(norm2(cdMul(x, y))) - 1)
		m.Alt += math.Sqrt(norm2(QAssoc(TierS, x, x, y)))
		m.Assoc += math.Sqrt(norm2(QAssoc(TierS, x, y, z)))
		m.H += norm2(x[:4])
		m.O += norm2(x[4:8])
		m.S += norm2(x[8:])
	}
	f := float64(M)
	m.Norm, m.Alt, m.Assoc, m.H, m.O, m.S = m.Norm/f, m.Alt/f, m.Assoc/f, m.H/f, m.O/f, m.S/f
	return
}

// RunSieve runs Test C; snapshots at t=0,15,100.
func RunSieve(w0 [][]float64, ctr *Counter) (snap map[int]SieveMetrics, ground, contain float64) {
	M := len(w0)
	W := make([][]float64, M)
	for i := range W {
		W[i] = append([]float64(nil), w0[i]...)
	}
	snap = map[int]SieveMetrics{0: sieveMetrics(W)}
	clip := func(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
	for st := 1; st <= 100; st++ {
		dS := make([]float64, M)
		for i := 0; i < M; i++ {
			x, y := W[i], W[(i-1+M)%M]
			zd := clip(1-math.Sqrt(norm2(cdMul(x, y))), 0, 1)
			alt := math.Sqrt(norm2(QAssoc(TierS, x, x, y))) // verbatim: [W, W, W_prev], W_prev = np.roll(+1) = i−1
			dS[i] = clip(0.18*(zd+alt), 0, 0.5)
		}
		for i := range W {
			for m := 8; m < 16; m++ {
				b := W[i][m]
				W[i][m] *= 1 - dS[i]
				ground += b*b - W[i][m]*W[i][m]
			}
			normalize(W[i])
		}
		dO := make([]float64, M)
		for i := 0; i < M; i++ {
			a := math.Sqrt(norm2(QAssoc(TierS, W[i], W[(i-1+M)%M], W[(i-2+M)%M])))
			dO[i] = clip(0.06*a, 0, 0.3)
		}
		for i := range W {
			for m := 4; m < 8; m++ {
				b := W[i][m]
				W[i][m] *= 1 - dO[i]
				contain += b*b - W[i][m]*W[i][m]
			}
			normalize(W[i])
		}
		if st == 15 || st == 100 {
			snap[st] = sieveMetrics(W)
		}
	}
	return
}
