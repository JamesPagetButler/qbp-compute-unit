package mdcu

// mdcu-kz detector tests (frozen mdcu-kz prereg §3, tests (a)-(h)), written BEFORE
// the detector (tests first). They need no dynamics: the detector is a pure
// function of the plane field. Seeds used here are 12000+, outside every frozen
// or used kz range (5000-5629, 9930-9949, R3/R4 ranges).

import (
	"math"
	"math/rand/v2"
	"testing"
)

func kzTestRand(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, 0x6b7a5465)) }

func kzSite(L, i, j, k int) int {
	i, j, k = ((i%L)+L)%L, ((j%L)+L)%L, ((k%L)+L)%L
	return (i*L+j)*L + k
}

// hopfPlane returns the plane l(n) ⊕ R·e5 in R^7: the tautological complex line of
// the Hopf bundle over S² = CP¹ (coordinates 0-3 = C² as R⁴) plus a fixed direction.
// The bundle has w2 = c1 mod 2 = 1. The frame is deliberately gauge-arbitrary
// (phase chosen by the spinor), so the detector must be gauge-covariant.
func hopfPlane(nx, ny, nz float64) [3][7]float64 {
	r := math.Sqrt(nx*nx + ny*ny + nz*nz)
	nx, ny, nz = nx/r, ny/r, nz/r
	th := math.Acos(math.Max(-1, math.Min(1, nz)))
	ph := math.Atan2(ny, nx)
	// spinor psi = (cos(th/2), e^{i ph} sin(th/2)) in C^2 -> R^4 = (Re a, Im a, Re b, Im b)
	a, b := math.Cos(th/2), math.Sin(th/2)
	var f [3][7]float64
	f[0][0], f[0][1], f[0][2], f[0][3] = a, 0, b*math.Cos(ph), b*math.Sin(ph)
	// i*psi: (-Im a, Re a, -Im b, Re b)
	f[1][0], f[1][1], f[1][2], f[1][3] = 0, a, -b*math.Sin(ph), b*math.Cos(ph)
	f[2][4] = 1
	return f
}

// hedgehogField builds the periodic field l(n(x)) ⊕ e5 with n = V/|V| and
// V = (sin a, sin b, sin c), a = 2π(i-i0)/L etc., with the zeros of V at cell
// centres (i0 = 0.5). For L = 16 the 8 zeros are the cells {0,8}³ and carry
// alternating hedgehog charge ±1 (Poincaré–Hopf on T³: the indices sum to 0).
func hedgehogField(L int) [][3][7]float64 {
	f := make([][3][7]float64, L*L*L)
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				a := 2 * math.Pi * (float64(i) - 0.5) / float64(L)
				b := 2 * math.Pi * (float64(j) - 0.5) / float64(L)
				c := 2 * math.Pi * (float64(k) - 0.5) / float64(L)
				f[kzSite(L, i, j, k)] = hopfPlane(math.Sin(a), math.Sin(b), math.Sin(c))
			}
		}
	}
	return f
}

// fourDefectField uses V = (sin a, sin b + (1 - cos a), sin c): zeros only at a = 0,
// b in {0, pi}, c in {0, pi}: four cells (centres at the half-integer coordinates).
func fourDefectField(L int) [][3][7]float64 {
	f := make([][3][7]float64, L*L*L)
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				a := 2 * math.Pi * (float64(i) - 0.5) / float64(L)
				b := 2 * math.Pi * (float64(j) - 0.5) / float64(L)
				c := 2 * math.Pi * (float64(k) - 0.5) / float64(L)
				f[kzSite(L, i, j, k)] = hopfPlane(math.Sin(a), math.Sin(b)+(1-math.Cos(a)), math.Sin(c))
			}
		}
	}
	return f
}

func kzCellIdx(L, i, j, k int) int { return kzSite(L, i, j, k) } // cell id = its min-corner site

func sameSet(got []int, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	m := map[int]bool{}
	for _, g := range got {
		m[g] = true
	}
	for _, w := range want {
		if !m[w] {
			return false
		}
	}
	return true
}

func randQuat(r *rand.Rand) [4]float64 {
	var q [4]float64
	n := 0.0
	for i := range q {
		q[i] = r.NormFloat64()
		n += q[i] * q[i]
	}
	n = math.Sqrt(n)
	for i := range q {
		q[i] /= n
	}
	return q
}

func matDet3(m [3][3]float64) float64 {
	return m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) - m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) + m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
}

func trRtM(rm, m [3][3]float64) float64 {
	s := 0.0
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			s += rm[i][j] * m[i][j]
		}
	}
	return s
}

// ---- building blocks -------------------------------------------------------

func TestKZNearestRotationOfARotation(t *testing.T) {
	r := kzTestRand(12000)
	for n := 0; n < 200; n++ {
		q := randQuat(r)
		got := NearestRotationQuat(QuatToMat(q))
		d := math.Abs(got[0]*q[0] + got[1]*q[1] + got[2]*q[2] + got[3]*q[3])
		if d < 1-1e-9 {
			t.Fatalf("nearest rotation of a rotation is not itself: |<q,q'>| = %v", d)
		}
	}
}

func TestKZNearestRotationIsOptimal(t *testing.T) {
	r := kzTestRand(12001)
	for n := 0; n < 50; n++ {
		var m [3][3]float64
		for i := range m {
			for j := range m[i] {
				m[i][j] = r.NormFloat64()
			}
		}
		q := NearestRotationQuat(m)
		rm := QuatToMat(q)
		if math.Abs(matDet3(rm)-1) > 1e-9 {
			t.Fatalf("not a proper rotation, det %v", matDet3(rm))
		}
		best := trRtM(rm, m)
		for s := 0; s < 400; s++ {
			if v := trRtM(QuatToMat(randQuat(r)), m); v > best+1e-9 {
				t.Fatalf("a random rotation beats the nearest one: %v > %v", v, best)
			}
		}
	}
}

func TestKZQuatAlgebra(t *testing.T) {
	r := kzTestRand(12002)
	a, b := randQuat(r), randQuat(r)
	ab := QuatMul(a, b)
	ra, rb, rab := QuatToMat(a), QuatToMat(b), QuatToMat(ab)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			s := 0.0
			for k := 0; k < 3; k++ {
				s += ra[i][k] * rb[k][j]
			}
			if math.Abs(s-rab[i][j]) > 1e-12 {
				t.Fatalf("R(ab) != R(a)R(b) at %d,%d", i, j)
			}
		}
	}
	c := QuatMul(a, QuatConj(a))
	if math.Abs(c[0]-1) > 1e-12 || math.Abs(c[1])+math.Abs(c[2])+math.Abs(c[3]) > 1e-12 {
		t.Fatalf("a·conj(a) != 1: %v", c)
	}
}

// ---- tests (a)-(h) of the frozen text, §3 -----------------------------------

// (a) uniform field has zero defects.
func TestKZ_a_UniformZero(t *testing.T) {
	L := 8
	f := make([][3][7]float64, L*L*L)
	p := haarG2Plane(kzTestRand(12010))
	for i := range f {
		f[i] = p
	}
	res := Z2DetectPlanes(L, f, DefaultZ2Params())
	if res.Defects != 0 || res.Ambiguous != 0 || res.NegCells != 0 {
		t.Fatalf("uniform field: %+v", res)
	}
}

// (b) a smooth rotation field has zero defects: P(x) = exp(eps*s(x)*G) P0, G a fixed
// antisymmetric 7x7 generator (acts on all frame rows identically).
func TestKZ_b_SmoothRotationZero(t *testing.T) {
	L := 12
	r := kzTestRand(12011)
	var G [7][7]float64
	for i := 0; i < 7; i++ {
		for j := i + 1; j < 7; j++ {
			g := r.NormFloat64()
			G[i][j], G[j][i] = g, -g
		}
	}
	p0 := haarG2Plane(r)
	f := make([][3][7]float64, L*L*L)
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				s := 0.7 * (math.Sin(2*math.Pi*float64(i)/float64(L)) + math.Cos(2*math.Pi*float64(j)/float64(L)) + math.Sin(2*math.Pi*float64(k)/float64(L)))
				// exp(s G) via a 24-term series (|sG| is small enough)
				var E, term [7][7]float64
				for a := 0; a < 7; a++ {
					E[a][a], term[a][a] = 1, 1
				}
				for n := 1; n <= 24; n++ {
					var nt [7][7]float64
					for a := 0; a < 7; a++ {
						for b := 0; b < 7; b++ {
							sum := 0.0
							for c := 0; c < 7; c++ {
								sum += term[a][c] * G[c][b]
							}
							nt[a][b] = sum * s / float64(n)
						}
					}
					term = nt
					for a := 0; a < 7; a++ {
						for b := 0; b < 7; b++ {
							E[a][b] += term[a][b]
						}
					}
				}
				var fr [3][7]float64
				for row := 0; row < 3; row++ {
					for b := 0; b < 7; b++ {
						for c := 0; c < 7; c++ {
							fr[row][b] += E[b][c] * p0[row][c]
						}
					}
				}
				f[kzSite(L, i, j, k)] = fr
			}
		}
	}
	res := Z2DetectPlanes(L, f, DefaultZ2Params())
	if res.Defects != 0 || res.NegCells != 0 {
		t.Fatalf("smooth rotation field has defects: %+v", res)
	}
}

// (c) a planted Z2 hedgehog (w2 = 1) is found exactly once (single-cell detector on
// the 8 corners of a cell centred on the hedgehog), and a cell far from it is +1.
func TestKZ_c_PlantedHedgehogFoundOnce(t *testing.T) {
	var c [8][3][7]float64
	for id := 0; id < 8; id++ {
		x, y, z := float64(id>>2&1)-0.5, float64(id>>1&1)-0.5, float64(id&1)-0.5
		c[id] = hopfPlane(x, y, z)
	}
	sign, amb := CellZ2Planes(c, DefaultZ2Params())
	if sign != -1 || amb {
		t.Fatalf("hedgehog cell: sign %d ambiguous %v, want -1 unambiguous", sign, amb)
	}
	// the same cell with the hedgehog moved far away (all corners near one direction)
	for id := 0; id < 8; id++ {
		x, y, z := float64(id>>2&1)-0.5, float64(id>>1&1)-0.5, float64(id&1)-0.5
		c[id] = hopfPlane(5+0.1*x, 5+0.1*y, 5+0.1*z)
	}
	if sign, amb = CellZ2Planes(c, DefaultZ2Params()); sign != 1 || amb {
		t.Fatalf("non-defect cell: sign %d ambiguous %v, want +1", sign, amb)
	}
}

// (d) planted defects on the lattice are found exactly where planted, once each
// (4 cells for the four-zero field; the 8 cells {0,8}³ for the eight-zero field).
func TestKZ_d_PlantedSetsFound(t *testing.T) {
	L := 16
	res := Z2DetectPlanes(L, fourDefectField(L), DefaultZ2Params())
	var want []int
	for _, j := range []int{0, 8} {
		for _, k := range []int{0, 8} {
			want = append(want, kzCellIdx(L, 0, j, k))
		}
	}
	if res.Defects != 4 || res.Ambiguous != 0 || !sameSet(res.DefectCells, want) {
		t.Fatalf("four-defect field: %+v want cells %v", res, want)
	}
	res = Z2DetectPlanes(L, hedgehogField(L), DefaultZ2Params())
	want = want[:0]
	for _, i := range []int{0, 8} {
		for _, j := range []int{0, 8} {
			for _, k := range []int{0, 8} {
				want = append(want, kzCellIdx(L, i, j, k))
			}
		}
	}
	if res.Defects != 8 || res.Ambiguous != 0 || !sameSet(res.DefectCells, want) {
		t.Fatalf("eight-defect field: %+v want cells %v", res, want)
	}
	if res.NegCells%2 != 0 || !res.ParityOK {
		t.Fatalf("parity: %+v", res)
	}
}

// (e) the ambiguous-cell counter fires on a deliberately sharp field, and a flagged
// cell is never counted as a defect. The frozen rule (a triangle holonomy angle >= pi)
// fires only at exactly pi, so the unit test drives it with (1) the triangle rule at
// exactly pi, (2) a lowered AmbigAngle parameter on a rough lattice field.
func TestKZ_e_AmbiguityCounter(t *testing.T) {
	half := math.Sqrt2 / 2
	ex := [4]float64{half, half, 0, 0} // rotation by pi/2 about x
	idq := [4]float64{1, 0, 0, 0}
	neg, amb := TriangleZ2(ex, ex, idq, math.Pi) // product = rotation by pi about x: w = 0
	if !amb {
		t.Fatalf("holonomy of exactly pi must be ambiguous (neg=%v)", neg)
	}
	if neg, amb = TriangleZ2(ex, idq, idq, math.Pi); amb {
		t.Fatalf("holonomy of pi/2 must not be ambiguous (neg=%v)", neg)
	}
	L := 8
	r := kzTestRand(12012)
	f := haarG2Set(r, L*L*L)
	lo := DefaultZ2Params()
	lo.AmbigAngle = 0.5
	res := Z2DetectPlanes(L, f, lo)
	if res.Ambiguous == 0 {
		t.Fatalf("a rough field must be flagged ambiguous at a 0.5 rad threshold: %+v", res)
	}
	for _, c := range res.DefectCells {
		for _, a := range res.AmbigCells {
			if c == a {
				t.Fatalf("cell %d is both a defect and ambiguous", c)
			}
		}
	}
}

// (f) i.i.d. Haar planes give the saturation density: reproducible, strictly inside
// (0, 1), and the same on repeat. (The value is recorded in the report; the tolerance
// that blocks the grid is not set numerically in the frozen text and is flagged.)
func TestKZ_f_HaarSaturationDensity(t *testing.T) {
	L := 8
	n1 := make([]float64, 0, 6)
	for s := uint64(0); s < 6; s++ {
		f := haarG2Set(kzTestRand(12020+s), L*L*L)
		res := Z2DetectPlanes(L, f, DefaultZ2Params())
		res2 := Z2DetectPlanes(L, f, DefaultZ2Params())
		if res.Defects != res2.Defects || res.Ambiguous != res2.Ambiguous {
			t.Fatalf("detector is not deterministic")
		}
		n1 = append(n1, float64(res.Defects)/float64(L*L*L))
	}
	m := 0.0
	for _, v := range n1 {
		m += v
	}
	m /= float64(len(n1))
	t.Logf("Haar saturation density n_inf (L=8, 6 fields) = %.4f", m)
	if m <= 0.02 || m >= 0.98 {
		t.Fatalf("implausible saturation density %v", m)
	}
}

// (g) a global orthogonal rotation of R^7 and a lattice reflection leave the count unchanged.
func TestKZ_g_Invariances(t *testing.T) {
	L := 8
	r := kzTestRand(12030)
	f := haarG2Set(r, L*L*L)
	base := Z2DetectPlanes(L, f, DefaultZ2Params())
	// random orthogonal Q via Gram–Schmidt
	var Q [7][7]float64
	for i := 0; i < 7; i++ {
		for k := 0; k < 7; k++ {
			Q[i][k] = r.NormFloat64()
		}
		for j := 0; j < i; j++ {
			d := 0.0
			for k := 0; k < 7; k++ {
				d += Q[i][k] * Q[j][k]
			}
			for k := 0; k < 7; k++ {
				Q[i][k] -= d * Q[j][k]
			}
		}
		n := 0.0
		for k := 0; k < 7; k++ {
			n += Q[i][k] * Q[i][k]
		}
		n = math.Sqrt(n)
		for k := 0; k < 7; k++ {
			Q[i][k] /= n
		}
	}
	g := make([][3][7]float64, len(f))
	for s := range f {
		for row := 0; row < 3; row++ {
			for a := 0; a < 7; a++ {
				for b := 0; b < 7; b++ {
					g[s][row][a] += Q[a][b] * f[s][row][b]
				}
			}
		}
	}
	if res := Z2DetectPlanes(L, g, DefaultZ2Params()); res.Defects != base.Defects || res.Ambiguous != base.Ambiguous {
		t.Fatalf("O(7) rotation changed the count: %+v vs %+v", res, base)
	}
	// Full inversion (i,j,k) -> (-i,-j,-k) is an improper reflection that maps every face
	// diagonal A->C of the frozen triangulation onto itself, so the count is exactly
	// invariant even on an unresolved (i.i.d.) field.
	h := make([][3][7]float64, len(f))
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				h[kzSite(L, -i, -j, -k)] = f[kzSite(L, i, j, k)]
			}
		}
	}
	if res := Z2DetectPlanes(L, h, DefaultZ2Params()); res.Defects != base.Defects || res.Ambiguous != base.Ambiguous {
		t.Fatalf("inversion changed the count: %d/%d vs %d/%d", res.Defects, res.Ambiguous, base.Defects, base.Ambiguous)
	}
	// A single-axis reflection x -> -x swaps the face diagonals of the y/z-normal faces, so on
	// an unresolved field the triangulation-dependent count may change (a property of the
	// frozen rule, reported, not hidden). On a resolved field it must be invariant.
	for _, fld := range [][][3][7]float64{fourDefectField(16), hedgehogField(16)} {
		a := Z2DetectPlanes(16, fld, DefaultZ2Params())
		m := make([][3][7]float64, len(fld))
		for i := 0; i < 16; i++ {
			for j := 0; j < 16; j++ {
				for k := 0; k < 16; k++ {
					m[kzSite(16, -i, j, k)] = fld[kzSite(16, i, j, k)]
				}
			}
		}
		if b := Z2DetectPlanes(16, m, DefaultZ2Params()); b.Defects != a.Defects || b.Ambiguous != a.Ambiguous {
			t.Fatalf("x-reflection changed the count on a resolved field: %d vs %d", b.Defects, a.Defects)
		}
	}

	// a gauge change of every frame (in-plane rotation) leaves the count unchanged
	gg := make([][3][7]float64, len(f))
	for s := range f {
		q := randQuat(r)
		R := QuatToMat(q)
		for row := 0; row < 3; row++ {
			for b := 0; b < 7; b++ {
				for c := 0; c < 3; c++ {
					gg[s][row][b] += R[row][c] * f[s][c][b]
				}
			}
		}
	}
	if res := Z2DetectPlanes(L, gg, DefaultZ2Params()); res.Defects != base.Defects || res.Ambiguous != base.Ambiguous {
		t.Fatalf("an in-plane frame gauge change altered the count: %+v vs %+v", res, base)
	}
}

// (h) the parity invariant on i.i.d. Haar fields, and the S³ adapter's zero-holonomy tests.
func TestKZ_h_ParityAndS3Adapter(t *testing.T) {
	L := 8
	for s := uint64(0); s < 12; s++ {
		f := haarG2Set(kzTestRand(12040+s), L*L*L)
		res := Z2DetectPlanes(L, f, DefaultZ2Params())
		if res.NegCells%2 != 0 {
			t.Fatalf("seed %d: odd number of -1 cells (%d)", s, res.NegCells)
		}
		if res.Ambiguous == 0 && !res.ParityOK {
			t.Fatalf("seed %d: parity check failed at zero ambiguity: %+v", s, res)
		}
	}
	// S3 null adapter: q -> Ad_q, global SU(2) lift: zero defects, zero ambiguity, parity ok.
	for s := uint64(0); s < 12; s++ {
		r := kzTestRand(12060 + s)
		q := make([][4]float64, L*L*L)
		for i := range q {
			q[i] = randQuat(r)
			if r.IntN(2) == 0 { // per-site sign flips do not change Ad_q
				for c := range q[i] {
					q[i][c] = -q[i][c]
				}
			}
		}
		res := Z2DetectS3(L, q, DefaultZ2Params())
		if res.Defects != 0 || res.NegCells != 0 || res.Ambiguous != 0 || !res.ParityOK {
			t.Fatalf("S3 adapter seed %d: %+v", s, res)
		}
	}
}
