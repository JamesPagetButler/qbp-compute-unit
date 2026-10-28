package mdcu

// mdcu-kz lattice dynamics (frozen mdcu-kz prereg §2, §2a, §5 Stage 0).
//
// Main arm: a plane field on an L^3 periodic lattice. Site s holds an orthonormal pair
// (u, v) of imaginary octonions (R^7); the plane is span(u, v, uv), the frame
// F_s = (u, v, w = uv). Bond energy d^2 = 3 - ||F_a F_b^T||_F^2 with J = 1, summed over
// the 3N bonds. S^3 arm: unit quaternions, bond energy 1 - q.q'.
//
// Step: synchronous Euler-Maruyama. All sites use the gradient of the state at the start
// of the step; each component then gets -eta*grad + sqrt(2*eta*T)*xi, followed by
// Gram-Schmidt (u normalised, v orthogonalised against u and normalised) on the main
// arm and renormalisation on the S^3 arm. At T = 0 no random numbers are drawn.
//
// Noise stream (harness choice, documented): KZNoiseRNG(seed, stream) is a PCG with
// state (seed, kzNoiseBase+stream); the rung's tau_Q is used as the stream, so the two
// arms and every rung of a replicate see independent noise from the same seed. Per
// step, the main arm draws 7 normals for u then 7 for v at each site in site order; the
// S^3 arm draws 4 per site.

import (
	"math"
	"math/rand/v2"
)

const kzNoiseBase = 0x6b7a4e6f

// KZNoiseRNG returns the deterministic noise generator for (seed, stream).
func KZNoiseRNG(seed, stream uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, kzNoiseBase+stream))
}

// KZEtaLadder is the frozen step-size ladder, largest first.
var KZEtaLadder = []float64{0.1, 0.05, 0.02, 0.01}

// ---- imaginary octonion product ---------------------------------------------

// kzC[i][j] = (k, sign) with e_i e_j = sign e_k (i != j, imaginary units); for i == j the
// imaginary part of the product is 0.
var kzC [7][7]struct {
	k    int
	sign float64
}

func init() {
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			if i == j {
				continue
			}
			x, y := make([]float64, 8), make([]float64, 8)
			x[i+1], y[j+1] = 1, 1
			p := MulRef(x, y)
			for k := 1; k < 8; k++ {
				if p[k] != 0 {
					kzC[i][j].k = k - 1
					kzC[i][j].sign = p[k]
				}
			}
		}
	}
}

// kzOctIm is the imaginary part of the product of two imaginary octonions.
func kzOctIm(u, v [7]float64) (w [7]float64) {
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			if i == j {
				continue
			}
			c := kzC[i][j]
			w[c.k] += c.sign * u[i] * v[j]
		}
	}
	return
}

// ---- main arm -----------------------------------------------------------------

// KZPlaneField is the main-arm field: U[s], V[s] orthonormal imaginary octonions.
type KZPlaneField struct {
	L    int
	U, V [][7]float64
	nbr  [][6]int
}

func kzNeighbours(L int) [][6]int {
	n := make([][6]int, L*L*L)
	at := func(i, j, k int) int { return (((i+L)%L)*L+((j+L)%L))*L + (k+L)%L }
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				s := at(i, j, k)
				n[s] = [6]int{at(i+1, j, k), at(i-1, j, k), at(i, j+1, k), at(i, j-1, k), at(i, j, k+1), at(i, j, k-1)}
			}
		}
	}
	return n
}

// NewKZPlaneFieldHaar draws an i.i.d. Haar-G2 plane at every site; the seed fixes the
// field and nothing else.
func NewKZPlaneFieldHaar(L int, seed uint64) *KZPlaneField {
	f := &KZPlaneField{L: L, U: make([][7]float64, L*L*L), V: make([][7]float64, L*L*L), nbr: kzNeighbours(L)}
	r := kzHaarRand(seed)
	for s := range f.U {
		p := haarG2Plane(r)
		f.U[s], f.V[s] = p[0], p[1]
	}
	return f
}

// NewKZPlaneFieldAligned puts the same Haar-G2 plane (drawn from seed) at every site.
func NewKZPlaneFieldAligned(L int, seed uint64) *KZPlaneField {
	f := &KZPlaneField{L: L, U: make([][7]float64, L*L*L), V: make([][7]float64, L*L*L), nbr: kzNeighbours(L)}
	p := haarG2Plane(kzHaarRand(seed))
	for s := range f.U {
		f.U[s], f.V[s] = p[0], p[1]
	}
	return f
}

func (f *KZPlaneField) frame(s int) (fr [3][7]float64) {
	fr[0], fr[1] = f.U[s], f.V[s]
	fr[2] = kzOctIm(f.U[s], f.V[s])
	return
}

// Frames returns the (u, v, uv) frame at every site.
func (f *KZPlaneField) Frames() [][3][7]float64 {
	out := make([][3][7]float64, len(f.U))
	for s := range out {
		out[s] = f.frame(s)
	}
	return out
}

func kzOverlap(a, b *[3][7]float64) (m [3][3]float64) {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			s := 0.0
			for k := 0; k < 7; k++ {
				s += a[i][k] * b[j][k]
			}
			m[i][j] = s
		}
	}
	return
}

// Energy is the sum over the 3N bonds of 3 - ||F_a F_b^T||_F^2.
func (f *KZPlaneField) Energy() float64 {
	fr := f.Frames()
	e := 0.0
	for s := range fr {
		for d := 0; d < 3; d++ { // forward neighbours: +x, +y, +z are nbr[s][0,2,4]
			b := f.nbr[s][2*d]
			m := kzOverlap(&fr[s], &fr[b])
			n2 := 0.0
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					n2 += m[i][j] * m[i][j]
				}
			}
			e += 3 - n2
		}
	}
	return e
}

// Gradient returns dE/du and dE/dv at every site (raw Euclidean gradient with respect to
// the stored u, v, the third frame row w = uv being differentiated through).
func (f *KZPlaneField) Gradient() (gu, gv [][7]float64) {
	fr := f.Frames()
	N := len(fr)
	gu, gv = make([][7]float64, N), make([][7]float64, N)
	for s := 0; s < N; s++ {
		var g [3][7]float64 // dE/dF_s rows
		for _, b := range f.nbr[s] {
			m := kzOverlap(&fr[s], &fr[b])
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					for k := 0; k < 7; k++ {
						g[i][k] -= 2 * m[i][j] * fr[b][j][k]
					}
				}
			}
		}
		gu[s], gv[s] = g[0], g[1]
		// chain through w = uv
		u, v := f.U[s], f.V[s]
		for i := 0; i < 7; i++ {
			for j := 0; j < 7; j++ {
				if i == j {
					continue
				}
				c := kzC[i][j]
				gw := g[2][c.k] * c.sign
				gu[s][i] += v[j] * gw
				gv[s][j] += u[i] * gw
			}
		}
	}
	return
}

// Step advances the field by one synchronous Euler-Maruyama step at temperature T.
func (f *KZPlaneField) Step(eta, T float64, rng *rand.Rand) {
	gu, gv := f.Gradient()
	sigma := math.Sqrt(2 * eta * T)
	for s := range f.U {
		var u, v [7]float64
		for k := 0; k < 7; k++ {
			u[k] = f.U[s][k] - eta*gu[s][k]
			if T > 0 {
				u[k] += sigma * rng.NormFloat64()
			}
		}
		for k := 0; k < 7; k++ {
			v[k] = f.V[s][k] - eta*gv[s][k]
			if T > 0 {
				v[k] += sigma * rng.NormFloat64()
			}
		}
		f.U[s], f.V[s] = kzGramSchmidt(u, v)
	}
}

func kzGramSchmidt(u, v [7]float64) ([7]float64, [7]float64) {
	n := 0.0
	for k := range u {
		n += u[k] * u[k]
	}
	n = math.Sqrt(n)
	for k := range u {
		u[k] /= n
	}
	d := 0.0
	for k := range u {
		d += u[k] * v[k]
	}
	n = 0.0
	for k := range v {
		v[k] -= d * u[k]
		n += v[k] * v[k]
	}
	n = math.Sqrt(n)
	for k := range v {
		v[k] /= n
	}
	return u, v
}

// OrderS is ||mean(Pi) - (3/7) I||_F / sqrt(12/7), Pi = F^T F the projector onto the
// plane: 0 for a Haar field in the large-L limit, 1 for an aligned field.
func (f *KZPlaneField) OrderS() float64 {
	var mean [7][7]float64
	fr := f.Frames()
	for s := range fr {
		for i := 0; i < 7; i++ {
			for j := 0; j < 7; j++ {
				for r := 0; r < 3; r++ {
					mean[i][j] += fr[s][r][i] * fr[s][r][j]
				}
			}
		}
	}
	n := float64(len(fr))
	ss := 0.0
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			x := mean[i][j] / n
			if i == j {
				x -= 3.0 / 7.0
			}
			ss += x * x
		}
	}
	return math.Sqrt(ss) / math.Sqrt(12.0/7.0)
}

// ---- S^3 arm --------------------------------------------------------------------

// KZS3Field is the S^3 control: a unit quaternion at every site.
type KZS3Field struct {
	L   int
	Q   [][4]float64
	nbr [][6]int
}

func NewKZS3FieldHaar(L int, seed uint64) *KZS3Field {
	f := &KZS3Field{L: L, Q: make([][4]float64, L*L*L), nbr: kzNeighbours(L)}
	r := kzHaarRand(seed)
	for s := range f.Q {
		var q [4]float64
		n := 0.0
		for c := range q {
			q[c] = r.NormFloat64()
			n += q[c] * q[c]
		}
		n = math.Sqrt(n)
		for c := range q {
			q[c] /= n
		}
		f.Q[s] = q
	}
	return f
}

func NewKZS3FieldAligned(L int) *KZS3Field {
	f := &KZS3Field{L: L, Q: make([][4]float64, L*L*L), nbr: kzNeighbours(L)}
	for s := range f.Q {
		f.Q[s] = [4]float64{1, 0, 0, 0}
	}
	return f
}

// Energy is the sum over the 3N bonds of 1 - q.q'.
func (f *KZS3Field) Energy() float64 {
	e := 0.0
	for s := range f.Q {
		for d := 0; d < 3; d++ {
			b := f.nbr[s][2*d]
			dot := 0.0
			for c := 0; c < 4; c++ {
				dot += f.Q[s][c] * f.Q[b][c]
			}
			e += 1 - dot
		}
	}
	return e
}

// Step is the same synchronous step on S^3: gradient -sum of the 6 neighbours.
func (f *KZS3Field) Step(eta, T float64, rng *rand.Rand) {
	sigma := math.Sqrt(2 * eta * T)
	next := make([][4]float64, len(f.Q))
	for s := range f.Q {
		var q [4]float64
		for c := 0; c < 4; c++ {
			sum := 0.0
			for _, b := range f.nbr[s] {
				sum += f.Q[b][c]
			}
			q[c] = f.Q[s][c] + eta*sum // -eta * grad, grad = -sum of neighbours
			if T > 0 {
				q[c] += sigma * rng.NormFloat64()
			}
		}
		n := math.Sqrt(q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3])
		for c := range q {
			q[c] /= n
		}
		next[s] = q
	}
	f.Q = next
}

// OrderM is ||<q>||: 0 for Haar (large L), 1 for aligned; q and -q are not identified.
func (f *KZS3Field) OrderM() float64 {
	var m [4]float64
	for s := range f.Q {
		for c := 0; c < 4; c++ {
			m[c] += f.Q[s][c]
		}
	}
	n := float64(len(f.Q))
	return math.Sqrt(m[0]*m[0]+m[1]*m[1]+m[2]*m[2]+m[3]*m[3]) / n
}

// ---- eta rule ---------------------------------------------------------------------

// kzDescentOK is the frozen descent test for one arm: 200 steps at T = 0 from the Haar
// start of the given seed, with the energy non-increasing at every step (relative
// tolerance 1e-9 for rounding).
func kzDescentOK(L int, seed uint64, eta float64, mainArm bool) bool {
	rng := KZNoiseRNG(seed, 0)
	if mainArm {
		f := NewKZPlaneFieldHaar(L, seed)
		prev := f.Energy()
		for n := 0; n < 200; n++ {
			f.Step(eta, 0, rng)
			e := f.Energy()
			if e > prev+1e-9*math.Abs(prev) || math.IsNaN(e) {
				return false
			}
			prev = e
		}
		return true
	}
	f := NewKZS3FieldHaar(L, seed)
	prev := f.Energy()
	for n := 0; n < 200; n++ {
		f.Step(eta, 0, rng)
		e := f.Energy()
		if e > prev+1e-9*math.Abs(prev) || math.IsNaN(e) {
			return false
		}
		prev = e
	}
	return true
}

// KZChooseEta returns the largest eta of KZEtaLadder whose T = 0 energy is non-
// increasing over 200 steps on every seed, for both arms (one eta: the smaller of the
// two arms' largest). ok is false if none passes.
func KZChooseEta(L int, seeds []uint64) (eta float64, ok bool) {
	for _, e := range KZEtaLadder {
		pass := true
		for _, sd := range seeds {
			if !kzDescentOK(L, sd, e, true) || !kzDescentOK(L, sd, e, false) {
				pass = false
				break
			}
		}
		if pass {
			return e, true
		}
	}
	return 0, false
}
