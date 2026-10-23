package mdcu

// Round-2 Q1 harness (Test B frame locking, controlled repro with nulls).
// Pre-registration: doc/spikes/prereg/mdcu-round2-checkin-DRAFT.md.
//
// Nothing here is part of the frozen v0.1.0-rc1 emulator API (ADR-002).
// RunRingGen with LayLocal/LayOverlap and T=assoc8 is bit-identical to the
// canonical RunRing (checked in q1grid_test.go), so the grid's main cell is the
// canonical dynamics, not a re-derivation.
//
// Deviations from the draft, stated here so they are not silent:
//   - Seeds use Go's math/rand/v2 PCG, not numpy default_rng, so the 200 seeds
//     are not the numpy seeds of the canonical single draw.
//   - Null 3 is a random *alternating vector-valued trilinear map* of equal
//     Frobenius norm to the true associator tensor (the associator is
//     vector-valued, so a scalar 3-form cannot play its role in ‖A‖²).
//   - The disjoint layout uses the true gradient of the plaquette energy.
//   - "Frame" (7 Fano lines) is axis-aligned; a locked plane can be any
//     associative 3-plane, so PlanePhi0 and the stored Plane basis are also kept
//     for a frame-independent spread analysis.

import (
	"math"
	"math/rand/v2"
	"sort"
)

type Layout int

const (
	LayLocal    Layout = iota // canonical: centre term only
	LayOverlap                // canonical: centre + (i-2,i-1,i) term
	LayDisjoint               // stride-3 plaquettes, true plaquette gradient
	LayNone                   // μ term only (no associator coupling)
)

func (l Layout) String() string {
	return [...]string{"local", "overlap", "disjoint", "none"}[l]
}

// Trilinear is an (alternating) vector-valued trilinear map on imaginary 8-vectors.
type Trilinear func(x, y, z []float64) []float64

// RunRingGen generalises RunRing over layout and trilinear map.
func RunRingGen(z0 [][]float64, lay Layout, T Trilinear) [][]float64 {
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
			switch lay {
			case LayLocal, LayOverlap:
				A := T(l, Z[i], r)
				var A2 []float64
				if lay == LayOverlap {
					A2 = T(ll, l, Z[i])
				}
				for b := 1; b < 8; b++ {
					e := basis(8, b)
					g[b] += 2 * dot(A, T(l, e, r))
					if lay == LayOverlap {
						g[b] += 2 * dot(A2, T(ll, l, e))
					}
				}
			case LayDisjoint:
				s := i % 3
				p := [3][]float64{Z[i-s], Z[i-s+1], Z[i-s+2]}
				A := T(p[0], p[1], p[2])
				for b := 1; b < 8; b++ {
					q := p
					q[s] = basis(8, b)
					g[b] += 2 * dot(A, T(q[0], q[1], q[2]))
				}
			case LayNone:
			}
			g[0] = 0
			pp := dot(g, Z[i])
			for k := range g {
				g[k] -= pp * Z[i][k]
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

// ---------------------------------------------------------------- null tensor

// AltTensor is a random alternating (in a,b,c) vector-valued trilinear map.
type AltTensor struct{ R [8][8][8][8]float64 } // indices 1..7 used

func (t *AltTensor) Apply(x, y, z []float64) []float64 {
	out := make([]float64, 8)
	for a := 1; a < 8; a++ {
		if x[a] == 0 {
			continue
		}
		for b := 1; b < 8; b++ {
			xy := x[a] * y[b]
			if xy == 0 {
				continue
			}
			for c := 1; c < 8; c++ {
				w := xy * z[c]
				for d := 1; d < 8; d++ {
					out[d] += w * t.R[a][b][c][d]
				}
			}
		}
	}
	return out
}

// AssocTensorNormSq is Σ_{a,b,c∈1..7} ‖[e_a,e_b,e_c]‖², the Frobenius norm² of
// the true associator tensor.
func AssocTensorNormSq() (s float64) {
	for a := 1; a < 8; a++ {
		for b := 1; b < 8; b++ {
			for c := 1; c < 8; c++ {
				s += norm2(assoc8(basis(8, a), basis(8, b), basis(8, c)))
			}
		}
	}
	return s
}

// NewRandomAlt draws a Gaussian tensor, antisymmetrises over (a,b,c) and scales
// it to the Frobenius norm² of the true associator tensor.
func NewRandomAlt(rng *rand.Rand) *AltTensor {
	var raw [8][8][8][8]float64
	for a := 1; a < 8; a++ {
		for b := 1; b < 8; b++ {
			for c := 1; c < 8; c++ {
				for d := 1; d < 8; d++ {
					raw[a][b][c][d] = rng.NormFloat64()
				}
			}
		}
	}
	t := &AltTensor{}
	perms := [6][3]int{{0, 1, 2}, {1, 2, 0}, {2, 0, 1}, {1, 0, 2}, {0, 2, 1}, {2, 1, 0}}
	sgn := [6]float64{1, 1, 1, -1, -1, -1}
	var ss float64
	for a := 1; a < 8; a++ {
		for b := 1; b < 8; b++ {
			for c := 1; c < 8; c++ {
				idx := [3]int{a, b, c}
				for d := 1; d < 8; d++ {
					v := 0.0
					for p := 0; p < 6; p++ {
						v += sgn[p] * raw[idx[perms[p][0]]][idx[perms[p][1]]][idx[perms[p][2]]][d]
					}
					t.R[a][b][c][d] = v
					ss += v * v
				}
			}
		}
	}
	k := math.Sqrt(AssocTensorNormSq() / ss)
	for a := 1; a < 8; a++ {
		for b := 1; b < 8; b++ {
			for c := 1; c < 8; c++ {
				for d := 1; d < 8; d++ {
					t.R[a][b][c][d] *= k
				}
			}
		}
	}
	return t
}

// ---------------------------------------------------------------- statistics

type Q1Stats struct {
	SVD3       float64      // top-3 share of Z[:,1:] Gram (primary)
	Local      float64      // mean ‖[z_{i-1},z_i,z_{i+1}]‖²
	Phi0Frac   float64      // fraction of consecutive triples with |φ0| ≥ 1-δ (secondary)
	PlanePhi0  float64      // |φ0| of the top-3 principal plane (1 = associative)
	Frame      int          // Fano line (1..7) with largest share of the top-3 plane
	FrameShare float64      // that share (in [0,1]; 3-plane fully inside a line = 1)
	Plane      [3][7]float64 // top-3 principal directions
}

const phi0Delta = 0.05

// jacobiEigVec7 returns eigenvalues and eigenvectors (columns of V) of symmetric a.
func jacobiEigVec7(a [7][7]float64) (ev [7]float64, V [7][7]float64) {
	for i := 0; i < 7; i++ {
		V[i][i] = 1
	}
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
				for k := 0; k < 7; k++ {
					vkp, vkq := V[k][p], V[k][q]
					V[k][p], V[k][q] = c*vkp-s*vkq, s*vkp+c*vkq
				}
			}
		}
	}
	for i := 0; i < 7; i++ {
		ev[i] = a[i][i]
	}
	return
}

func phi0(x, y, z []float64) float64 { return dot(MulRef(x, y), z) }

func Q1Measure(Z [][]float64) (m Q1Stats) {
	n := len(Z)
	var g [7][7]float64
	for _, z := range Z {
		for a := 0; a < 7; a++ {
			for b := 0; b < 7; b++ {
				g[a][b] += z[a+1] * z[b+1]
			}
		}
	}
	ev, V := jacobiEigVec7(g)
	ord := []int{0, 1, 2, 3, 4, 5, 6}
	sort.Slice(ord, func(i, j int) bool { return ev[ord[i]] > ev[ord[j]] })
	tot := 0.0
	for _, e := range ev {
		tot += e
	}
	m.SVD3 = (ev[ord[0]] + ev[ord[1]] + ev[ord[2]]) / tot
	var u [3][]float64
	for t := 0; t < 3; t++ {
		u[t] = make([]float64, 8)
		for k := 0; k < 7; k++ {
			m.Plane[t][k] = V[k][ord[t]]
			u[t][k+1] = V[k][ord[t]]
		}
	}
	m.PlanePhi0 = math.Abs(phi0(u[0], u[1], u[2]))
	best := -1.0
	for f := 1; f <= 7; f++ {
		l := Frame(f)
		sh := 0.0
		for t := 0; t < 3; t++ {
			for _, k := range [3]int{l.I, l.J, l.K} {
				sh += m.Plane[t][k-1] * m.Plane[t][k-1]
			}
		}
		sh /= 3
		if sh > best {
			best, m.Frame = sh, f
		}
	}
	m.FrameShare = best
	cnt := 0
	for i := 0; i < n; i++ {
		a, b, c := Z[(i-1+n)%n], Z[i], Z[(i+1)%n]
		m.Local += norm2(assoc8(a, b, c))
		// Gram-Schmidt the consecutive triple
		v := [3][]float64{append([]float64(nil), a...), append([]float64(nil), b...), append([]float64(nil), c...)}
		ok := true
		for t := 0; t < 3 && ok; t++ {
			for s := 0; s < t; s++ {
				d := dot(v[t], v[s])
				for k := range v[t] {
					v[t][k] -= d * v[s][k]
				}
			}
			nn := math.Sqrt(dot(v[t], v[t]))
			if nn < 1e-9 {
				ok = false
				break
			}
			for k := range v[t] {
				v[t][k] /= nn
			}
		}
		if ok && math.Abs(phi0(v[0], v[1], v[2])) >= 1-phi0Delta {
			cnt++
		}
	}
	m.Local /= float64(n)
	m.Phi0Frac = float64(cnt) / float64(n)
	return
}

// ---------------------------------------------------------------- init + runs

// Q1Init draws n unit imaginary 8-vectors: normal base, plus σ·normal noise, renormalised.
func Q1Init(n int, sigma float64, rng *rand.Rand) [][]float64 {
	Z := make([][]float64, n)
	for i := range Z {
		z := make([]float64, 8)
		for k := 1; k < 8; k++ {
			z[k] = rng.NormFloat64()
		}
		normalize(z)
		if sigma > 0 {
			for k := 1; k < 8; k++ {
				z[k] += sigma * rng.NormFloat64()
			}
			normalize(z)
		}
		Z[i] = z
	}
	return Z
}

type Q1Run struct {
	N      int
	Sigma  float64
	Seed   uint64
	Kind   string // local | overlap | disjoint | none | rand3 | shuffle
	Init   Q1Stats
	Final  Q1Stats
}

func newRNG(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)) }

// RunQ1Kind runs one (N, σ, seed, kind) with a deterministic RNG stream.
func RunQ1Kind(n int, sigma float64, seed uint64, kind string) Q1Run {
	rng := newRNG(seed)
	z0 := Q1Init(n, sigma, rng) // same init for every kind at a given (n,σ,seed)
	run := Q1Run{N: n, Sigma: sigma, Seed: seed, Kind: kind}
	var Z [][]float64
	switch kind {
	case "local":
		Z = RunRingGen(z0, LayLocal, assoc8)
	case "overlap":
		Z = RunRingGen(z0, LayOverlap, assoc8)
	case "disjoint":
		Z = RunRingGen(z0, LayDisjoint, assoc8)
	case "none":
		Z = RunRingGen(z0, LayNone, assoc8)
	case "rand3":
		t := NewRandomAlt(newRNG(seed ^ 0xa5a5a5a5))
		Z = RunRingGen(z0, LayOverlap, t.Apply)
	case "shuffle":
		perm := newRNG(seed ^ 0x5a5a5a5a).Perm(n)
		zs := make([][]float64, n)
		for i, p := range perm {
			zs[i] = z0[p]
		}
		z0 = zs
		Z = RunRingGen(z0, LayOverlap, assoc8)
	default:
		panic("mdcu: unknown Q1 kind " + kind)
	}
	run.Init, run.Final = Q1Measure(z0), Q1Measure(Z)
	return run
}
