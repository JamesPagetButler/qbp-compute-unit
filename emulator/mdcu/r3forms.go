package mdcu

// Round-3 3-form layer (doc/spikes/prereg/mdcu-round3-prereg-FROZEN-v1.md,
// sections 1.1-1.2). Nothing here is part of the frozen v0.1.0-rc1 emulator API.
//
// A Form3 is a scalar alternating 3-form on R^7 (the imaginary octonions),
// stored in the 0-based imaginary basis (index i <-> e_{i+1}). φ is built from
// the package's own octonion product (phi0 = <xy, z>), so the Harvey–Lawson
// identity  ||[a,b,c]||² = 4(Gram₃ − φ²)  holds for the same convention that
// assoc8 uses (checked in r3forms_test.go).

import (
	"math"
	"math/rand/v2"
	"sync"
)

// PCG stream constants for round 3. Pre-stated: they differ from round 2's
// (0xa5a5a5a5 rand3, 0x5a5a5a5a shuffle) and from each other
// (TestStreamConstantsDistinct).
const (
	r3SeedRand3 uint64 = 0x6d2b79f5
	r3SeedPsiS1 uint64 = 0x1b873593
	r3SeedPsiS2 uint64 = 0x85ebca6b
)

const (
	r3S1Cap  = 20000 // S1 rejection cap (frozen §1.2)
	r3FrobSq = 42.0  // Σ over ordered (a,b,c) of ψ²_{abc}, matched to φ
	r3SMin   = 0.05  // S2 scale floor
	r3SSpan  = 0.20  // S2 scale = r3SMin + r3SSpan·u, u ~ U(0,1)
)

type f3ent struct {
	i, j, k int
	v       float64
}

// Form3 is an alternating scalar 3-form on R^7.
type Form3 struct {
	C   [7][7][7]float64
	ent []f3ent
}

// NewForm3 wraps an alternating tensor.
func NewForm3(c [7][7][7]float64) *Form3 {
	f := &Form3{C: c}
	f.rebuild()
	return f
}

func (f *Form3) rebuild() {
	f.ent = f.ent[:0]
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			for k := 0; k < 7; k++ {
				if f.C[i][j][k] != 0 {
					f.ent = append(f.ent, f3ent{i, j, k, f.C[i][j][k]})
				}
			}
		}
	}
}

// FrobSq is Σ over all ordered (a,b,c) of C²  (42 for φ).
func (f *Form3) FrobSq() (s float64) {
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			for k := 0; k < 7; k++ {
				s += f.C[i][j][k] * f.C[i][j][k]
			}
		}
	}
	return
}

// Rescale scales the form so that FrobSq() == target.
func (f *Form3) Rescale(target float64) {
	k := math.Sqrt(target / f.FrobSq())
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			for l := 0; l < 7; l++ {
				f.C[i][j][l] *= k
			}
		}
	}
	f.rebuild()
}

// Eval is f(a,b,c) for 8-vectors with the imaginary part in components 1..7.
func (f *Form3) Eval(a, b, c []float64) (s float64) {
	for _, e := range f.ent {
		s += e.v * a[e.i+1] * b[e.j+1] * c[e.k+1]
	}
	return
}

// GradArg is the gradient of f(x,y,z) with respect to the pos-th argument
// (0, 1 or 2), as an 8-vector whose component 0 is zero.
func (f *Form3) GradArg(x, y, z []float64, pos int) (g [8]float64) {
	switch pos {
	case 0:
		for _, e := range f.ent {
			g[e.i+1] += e.v * y[e.j+1] * z[e.k+1]
		}
	case 1:
		for _, e := range f.ent {
			g[e.j+1] += e.v * x[e.i+1] * z[e.k+1]
		}
	case 2:
		for _, e := range f.ent {
			g[e.k+1] += e.v * x[e.i+1] * y[e.j+1]
		}
	default:
		panic("mdcu: GradArg position out of range")
	}
	return
}

var (
	phiOnce sync.Once
	phiForm *Form3
)

// Phi is the associative 3-form φ(a,b,c) = <ab, c>, from the package's octonion
// product. Lazily built (it depends on tables set in init()).
func Phi() *Form3 {
	phiOnce.Do(func() {
		var c [7][7][7]float64
		for i := 1; i < 8; i++ {
			for j := 1; j < 8; j++ {
				for k := 1; k < 8; k++ {
					c[i-1][j-1][k-1] = phi0(basis(8, i), basis(8, j), basis(8, k))
				}
			}
		}
		phiForm = NewForm3(c)
	})
	return phiForm
}

// gaussianForm draws the 35 independent components a<b<c i.i.d. N(0,1), in
// lexicographic order, antisymmetrises, and Frobenius-matches to 42.
func gaussianForm(rng *rand.Rand) *Form3 {
	var c [7][7][7]float64
	for a := 0; a < 7; a++ {
		for b := a + 1; b < 7; b++ {
			for d := b + 1; d < 7; d++ {
				v := rng.NormFloat64()
				c[a][b][d], c[b][d][a], c[d][a][b] = v, v, v
				c[b][a][d], c[a][d][b], c[d][b][a] = -v, -v, -v
			}
		}
	}
	f := NewForm3(c)
	f.Rescale(r3FrobSq)
	return f
}

// ---------------------------------------------------------------- Gram₃

// Gram3 is the Gram determinant of the imaginary parts of a, b, c
// (= ||a∧b∧c||²).
func Gram3(a, b, c []float64) float64 {
	var pp, qq, rr, pq, pr, qr float64
	for k := 1; k < 8; k++ {
		pp += a[k] * a[k]
		qq += b[k] * b[k]
		rr += c[k] * c[k]
		pq += a[k] * b[k]
		pr += a[k] * c[k]
		qr += b[k] * c[k]
	}
	return pp*(qq*rr-qr*qr) - pq*(pq*rr-qr*pr) + pr*(pq*qr-qq*pr)
}

// Gram3Wedge computes ||a∧b∧c||² as the sum of squares of the 35 3×3 minors
// (Cauchy–Binet); used to cross-check Gram3.
func Gram3Wedge(a, b, c []float64) (s float64) {
	for i := 1; i < 8; i++ {
		for j := i + 1; j < 8; j++ {
			for k := j + 1; k < 8; k++ {
				d := a[i]*(b[j]*c[k]-b[k]*c[j]) - a[j]*(b[i]*c[k]-b[k]*c[i]) + a[k]*(b[i]*c[j]-b[j]*c[i])
				s += d * d
			}
		}
	}
	return
}

// gradGram3 is the gradient of Gram₃(x,y,z) with respect to the pos-th argument.
func gradGram3(x, y, z []float64, pos int) (g [8]float64) {
	var p, q, w []float64
	switch pos {
	case 0:
		p, q, w = y, z, x
	case 1:
		p, q, w = x, z, y
	case 2:
		p, q, w = x, y, z
	default:
		panic("mdcu: gradGram3 position out of range")
	}
	var pp, qq, pq, pw, qw float64
	for k := 1; k < 8; k++ {
		pp += p[k] * p[k]
		qq += q[k] * q[k]
		pq += p[k] * q[k]
		pw += p[k] * w[k]
		qw += q[k] * w[k]
	}
	cwp := pq*qw - pw*qq
	cwq := pw*pq - pp*qw
	cww := pp*qq - pq*pq
	for k := 1; k < 8; k++ {
		g[k] = 2 * (cwp*p[k] + cwq*q[k] + cww*w[k])
	}
	return
}

// ---------------------------------------------------------------- energies

// R3Energy is a triple energy E(x,y,z), known only through its gradient with
// respect to one argument. The ring runner calls Grad(l, z_i, r, 1) and
// Grad(ll, l, z_i, 2), the same slots round 2's RunRingGen differentiates.
type R3Energy interface {
	Grad(x, y, z []float64, pos int) []float64
}

type scalarEnergy struct {
	gram bool
	f    *Form3
}

// NewScalarEnergy builds E = 4·Gram₃·[gram] − 4·f² (f may be nil):
//
//	K-gram:    gram=true,  f=nil
//	K-phi:     gram=false, f=φ
//	K-psi:     gram=false, f=ψ
//	K-psigram: gram=true,  f=ψ
func NewScalarEnergy(gram bool, f *Form3) R3Energy { return scalarEnergy{gram, f} }

func (s scalarEnergy) Grad(x, y, z []float64, pos int) []float64 {
	g := make([]float64, 8)
	if s.gram {
		gg := gradGram3(x, y, z, pos)
		for k := 1; k < 8; k++ {
			g[k] += 4 * gg[k]
		}
	}
	if s.f != nil {
		fv := s.f.Eval(x, y, z)
		gf := s.f.GradArg(x, y, z, pos)
		for k := 1; k < 8; k++ {
			g[k] += -8 * fv * gf[k]
		}
	}
	return g
}

// ---------------------------------------------------------------- B_ψ

var (
	permsOnce sync.Once
	perms7    [5040][7]uint8
	permSign  [5040]float64
)

func buildPerms() {
	n := 0
	var a [7]uint8
	for i := range a {
		a[i] = uint8(i)
	}
	var rec func(k int)
	rec = func(k int) {
		if k == 7 {
			perms7[n] = a
			// sign by counting inversions
			inv := 0
			for i := 0; i < 7; i++ {
				for j := i + 1; j < 7; j++ {
					if a[i] > a[j] {
						inv++
					}
				}
			}
			if inv%2 == 0 {
				permSign[n] = 1
			} else {
				permSign[n] = -1
			}
			n++
			return
		}
		for i := k; i < 7; i++ {
			a[k], a[i] = a[i], a[k]
			rec(k + 1)
			a[k], a[i] = a[i], a[k]
		}
	}
	rec(0)
	if n != 5040 {
		panic("mdcu: permutation table")
	}
}

// BPsi is Hitchin's symmetric form (up to the positive normalisation that is
// irrelevant for its signature):
//
//	B_ij = Σ_σ sgn(σ) ψ[i][σ0][σ1] ψ[j][σ2][σ3] ψ[σ4][σ5][σ6]
//
// (B_φ = ±144·I).
func BPsi(f *Form3) (B [7][7]float64) {
	permsOnce.Do(buildPerms)
	var u, w [7]float64
	for n := 0; n < 5040; n++ {
		p := &perms7[n]
		c := permSign[n] * f.C[p[4]][p[5]][p[6]]
		if c == 0 {
			continue
		}
		for i := 0; i < 7; i++ {
			u[i] = f.C[i][p[0]][p[1]] * c
			w[i] = f.C[i][p[2]][p[3]]
		}
		for i := 0; i < 7; i++ {
			if u[i] == 0 {
				continue
			}
			for j := 0; j < 7; j++ {
				B[i][j] += u[i] * w[j]
			}
		}
	}
	for i := 0; i < 7; i++ {
		for j := i + 1; j < 7; j++ {
			m := 0.5 * (B[i][j] + B[j][i])
			B[i][j], B[j][i] = m, m
		}
	}
	return
}

// ClassifyPsi sorts a 3-form by the signature of B_ψ:
//
//	"definite"   all eigenvalues of one sign (compact-G₂ type in some metric)
//	"split"      signature (3,4) or (4,3)
//	"degenerate" min|eig| < 1e-9·max|eig| (rejected by the samplers)
func ClassifyPsi(f *Form3) (string, [7]float64) {
	ev, _ := jacobiEigVec7(BPsi(f))
	maxAbs, minAbs := 0.0, math.Inf(1)
	pos, neg := 0, 0
	for _, e := range ev {
		a := math.Abs(e)
		maxAbs = math.Max(maxAbs, a)
		minAbs = math.Min(minAbs, a)
		if e > 0 {
			pos++
		} else if e < 0 {
			neg++
		}
	}
	switch {
	case minAbs < 1e-9*maxAbs:
		return "degenerate", ev
	case pos == 7 || neg == 7:
		return "definite", ev
	default:
		return "split", ev
	}
}

// Signature returns the numbers of positive and negative eigenvalues of B_ψ.
func Signature(f *Form3) (pos, neg int) {
	ev, _ := jacobiEigVec7(BPsi(f))
	for _, e := range ev {
		if e > 0 {
			pos++
		} else if e < 0 {
			neg++
		}
	}
	return
}

// ---------------------------------------------------------------- samplers

// R3Stratum assigns a seed to a B_ψ stratum by parity (frozen §1.2): even
// seeds are definite, odd seeds are split (100 each over 3000–3199).
func R3Stratum(seed uint64) string {
	if seed%2 == 0 {
		return "definite"
	}
	return "split"
}

// S1Draw is the result of the S1 rejection sampler.
type S1Draw struct {
	Form       *Form3
	Draws      int // attempts including the accepted one (cap on a miss)
	Degenerate int // degenerate draws rejected on the way
}

// SampleS1 draws Frobenius-matched Gaussian 3-forms from the seed's S1 stream
// until one has the wanted type ("definite" or "split"), up to r3S1Cap draws.
// ok=false is a "sampler-miss".
func SampleS1(seed uint64, want string) (S1Draw, bool) {
	rng := newRNG(seed ^ r3SeedPsiS1)
	out := S1Draw{}
	for n := 1; n <= r3S1Cap; n++ {
		f := gaussianForm(rng)
		kind, _ := ClassifyPsi(f)
		out.Draws = n
		if kind == "degenerate" {
			out.Degenerate++
			continue
		}
		if kind == want {
			out.Form = f
			return out, true
		}
	}
	return out, false
}

// S2Draw is a deformed φ: ψ = g*φ with g = exp(s·S), S symmetric traceless.
type S2Draw struct {
	Form   *Form3
	S      float64 // scale, in [0.05, 0.25)
	TraceS float64 // trace of the symmetric matrix S (should be ~0)
	Dist   float64 // ||ψ − φ|| / sqrt(42), after the Frobenius match
	Cond   float64 // condition number of g·gᵀ (metric distortion)
}

// SampleS2 builds the S2 form for a seed from its own PCG stream. The rotation
// part of g is omitted by construction (a rotated φ is G₂-conjugate to φ).
// No rejection is needed: g*φ lies in φ's GL(7) orbit, so it is always definite.
func SampleS2(seed uint64) S2Draw {
	rng := newRNG(seed ^ r3SeedPsiS2)
	u := rng.Float64()
	s := r3SMin + r3SSpan*u
	var G, S [7][7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			G[i][j] = rng.NormFloat64()
		}
	}
	tr := 0.0
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			S[i][j] = 0.5 * (G[i][j] + G[j][i])
		}
		tr += S[i][i]
	}
	for i := 0; i < 7; i++ {
		S[i][i] -= tr / 7
	}
	trAfter := 0.0
	for i := 0; i < 7; i++ {
		trAfter += S[i][i]
	}
	ev, V := jacobiEigVec7(S)
	var g [7][7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			for p := 0; p < 7; p++ {
				g[i][j] += V[i][p] * math.Exp(s*ev[p]) * V[j][p]
			}
		}
	}
	phi := Phi()
	var t1, t2, t3 [7][7][7]float64
	for a := 0; a < 7; a++ {
		for j := 0; j < 7; j++ {
			for k := 0; k < 7; k++ {
				for i := 0; i < 7; i++ {
					t1[a][j][k] += g[i][a] * phi.C[i][j][k]
				}
			}
		}
	}
	for a := 0; a < 7; a++ {
		for b := 0; b < 7; b++ {
			for k := 0; k < 7; k++ {
				for j := 0; j < 7; j++ {
					t2[a][b][k] += g[j][b] * t1[a][j][k]
				}
			}
		}
	}
	for a := 0; a < 7; a++ {
		for b := 0; b < 7; b++ {
			for c := 0; c < 7; c++ {
				for k := 0; k < 7; k++ {
					t3[a][b][c] += g[k][c] * t2[a][b][k]
				}
			}
		}
	}
	f := NewForm3(t3)
	f.Rescale(r3FrobSq)
	var d2 float64
	for a := 0; a < 7; a++ {
		for b := 0; b < 7; b++ {
			for c := 0; c < 7; c++ {
				x := f.C[a][b][c] - phi.C[a][b][c]
				d2 += x * x
			}
		}
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, e := range ev {
		lo, hi = math.Min(lo, e), math.Max(hi, e)
	}
	return S2Draw{Form: f, S: s, TraceS: trAfter, Dist: math.Sqrt(d2 / r3FrobSq), Cond: math.Exp(2 * s * (hi - lo))}
}
