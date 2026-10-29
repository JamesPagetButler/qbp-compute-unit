package mdcu

import (
	"math"
	"testing"
)

func TestKZTiltXiIsUnitAntisymmetric(t *testing.T) {
	for site := 0; site < 20; site++ {
		x := kzTiltXi(13100, site)
		fro := 0.0
		for i := 0; i < 7; i++ {
			for j := 0; j < 7; j++ {
				if math.Abs(x[i][j]+x[j][i]) > 1e-15 {
					t.Fatalf("site %d: xi not antisymmetric at %d,%d", site, i, j)
				}
				fro += x[i][j] * x[i][j]
			}
		}
		if math.Abs(fro-1) > 1e-12 {
			t.Fatalf("site %d: Frobenius norm^2 = %v, want 1", site, fro)
		}
	}
	a, b, c := kzTiltXi(13100, 3), kzTiltXi(13100, 3), kzTiltXi(13101, 3)
	if a != b {
		t.Fatal("xi must be a pure function of (seed, site)")
	}
	if a == c || kzTiltXi(13100, 4) == a {
		t.Fatal("xi must differ across seeds and sites")
	}
}

func TestKZExpSO7(t *testing.T) {
	// a single rotation generator: xi = (e0 e1^T - e1 e0^T) * s has exp = rotation by s in plane (0,1)
	var x [7][7]float64
	x[0][1], x[1][0] = -1, 1
	for _, s := range []float64{0, 0.05, 0.4, 1.3} {
		e := kzExpSO7(&x, s)
		if math.Abs(e[0][0]-math.Cos(s)) > 1e-13 || math.Abs(e[1][0]-math.Sin(s)) > 1e-13 ||
			math.Abs(e[0][1]+math.Sin(s)) > 1e-13 || math.Abs(e[2][2]-1) > 1e-13 {
			t.Fatalf("s=%v: %v", s, e)
		}
	}
	// a generic unit generator at the largest grid sigma is orthogonal to rounding
	g := kzTiltXi(13100, 9)
	e := kzExpSO7(&g, 0.4)
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			s := 0.0
			for k := 0; k < 7; k++ {
				s += e[k][i] * e[k][j]
			}
			want := 0.0
			if i == j {
				want = 1
			}
			if math.Abs(s-want) > 1e-13 {
				t.Fatalf("E^T E[%d][%d] = %v", i, j, s)
			}
		}
	}
}

func TestKZTiltedFramesAreOrthonormalThreeFrames(t *testing.T) {
	q := NewKZS3FieldHaar(4, 13110).Q
	for _, sg := range []float64{0, 0.05, 0.4} {
		f := KZTiltedFrames(4, q, sg, 13111)
		for s := range f {
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					d := 0.0
					for k := 0; k < 7; k++ {
						d += f[s][i][k] * f[s][j][k]
					}
					want := 0.0
					if i == j {
						want = 1
					}
					if math.Abs(d-want) > 1e-12 {
						t.Fatalf("sigma %v site %d: F F^T[%d][%d]=%v", sg, s, i, j, d)
					}
				}
			}
		}
	}
}

// At sigma = 0 the tilted path is the S3 adapter exactly (and its frames are [R | 0]).
func TestKZTiltSigmaZeroIsTheAdapter(t *testing.T) {
	f := NewKZS3FieldHaar(6, 13120)
	p := Amend1Z2Params()
	got := KZDetectS3Tilted(6, f.Q, 0, 13121, p)
	want := Z2DetectS3(6, f.Q, p)
	if got.NegCells != want.NegCells || got.Defects != want.Defects || got.Ambiguous != want.Ambiguous {
		t.Fatalf("%+v vs %+v", got, want)
	}
	fr := KZTiltedFrames(6, f.Q, 0, 13121)
	r := QuatToMat(f.Q[5])
	for i := 0; i < 3; i++ {
		for k := 0; k < 7; k++ {
			w := 0.0
			if k < 3 {
				w = r[i][k]
			}
			if math.Abs(fr[5][i][k]-w) > 1e-15 {
				t.Fatalf("frame [%d][%d]=%v want %v", i, k, fr[5][i][k], w)
			}
		}
	}
}

// The tilt is what makes the transport non-flat: sigma = 0 gives holonomy exactly the
// identity on any field (the theorem of seq 211), sigma > 0 does not.
func TestKZTiltMakesTransportNonFlat(t *testing.T) {
	L := 4
	q := NewKZS3FieldAligned(L).Q
	maxDev := func(sg float64) float64 {
		fr := KZTiltedFrames(L, q, sg, 13130)
		dev := 0.0
		idx := func(i, j, k int) int { return kzSiteIdx(L, i, j, k) }
		for i := 0; i < L; i++ {
			for j := 0; j < L; j++ {
				for k := 0; k < L; k++ {
					a, b, c := idx(i, j, k), idx(i+1, j, k), idx(i+1, j+1, k)
					lift := func(x, y int) [4]float64 { return NearestRotationQuat(planeOverlap(&fr[x], &fr[y])) }
					l := QuatMul(QuatMul(lift(a, b), lift(b, c)), lift(c, a))
					dev = math.Max(dev, 1-math.Abs(l[0]))
				}
			}
		}
		return dev
	}
	if d := maxDev(0); d > 1e-12 {
		t.Fatalf("sigma 0 must be flat, deviation %v", d)
	}
	if d := maxDev(0.4); d < 1e-6 {
		t.Fatalf("sigma 0.4 must give non-flat transport, deviation %v", d)
	}
}

func TestKZSigmaStarRule(t *testing.T) {
	rows := func(f ...float64) []KZCalRow {
		var r []KZCalRow
		for i, s := range KZSigmaGrid {
			r = append(r, KZCalRow{Sigma: s, PooledAmbigFrac: f[i]})
		}
		return r
	}
	cases := []struct {
		fr       []float64
		want     float64
		fallback bool
	}{
		{[]float64{0, 0, 0, 0}, 0.40, false},
		{[]float64{0, 0.02, 0.05, 0.06}, 0.20, false},  // <= 5% passes, 6% does not
		{[]float64{0.01, 0.2, 0.01, 0.9}, 0.20, false}, // the largest passing, not the first failure
		{[]float64{0.06, 0.2, 0.3, 0.9}, 0.05, true},   // none passes: sigma = 0.05, recorded
	}
	for _, c := range cases {
		got, fb := KZSigmaStar(rows(c.fr...))
		if got != c.want || fb != c.fallback {
			t.Errorf("%v: got (%v,%v) want (%v,%v)", c.fr, got, fb, c.want, c.fallback)
		}
	}
}

func TestKZCalibrateSigmaOnAlignedAndHaar(t *testing.T) {
	seeds := []uint64{13140, 13141, 13142}
	p := Amend1Z2Params()
	// an aligned S3 field has no roughness: the tilt alone sets the ambiguity
	al := KZCalibrateSigma(6, func(int) [][4]float64 { return NewKZS3FieldAligned(6).Q }, seeds, p)
	if len(al) != len(KZSigmaGrid) {
		t.Fatalf("%d rows", len(al))
	}
	for _, r := range al {
		if r.Fields != 3 || r.Cells != 3*216 {
			t.Fatalf("%+v", r)
		}
	}
	// i.i.d. Haar S3 fields are rough: essentially everything is ambiguous at every sigma
	hr := KZCalibrateSigma(6, func(i int) [][4]float64 { return NewKZS3FieldHaar(6, seeds[i]+100).Q }, seeds, p)
	for _, r := range hr {
		if r.PooledAmbigFrac < 0.9 {
			t.Fatalf("Haar S3 at sigma %v: ambiguous fraction %v", r.Sigma, r.PooledAmbigFrac)
		}
	}
}

func TestKZTiltSeedClasses(t *testing.T) {
	if KZClassifySeed(5650) != KZSeedTilt || KZClassifySeed(5849) != KZSeedTilt {
		t.Fatal("5650-5849 are the run tilt seeds")
	}
	if KZClassifySeed(5850) != KZSeedTiltCal || KZClassifySeed(5889) != KZSeedTiltCal {
		t.Fatal("5850-5889 are the calibration seeds")
	}
	if KZClassifySeed(5649) != KZSeedSaturation || KZClassifySeed(5890) != KZSeedFree {
		t.Fatal("neighbours unchanged")
	}
	if KZCheckSmokeSeeds([]uint64{5700}) == nil {
		t.Fatal("a smoke run may not use a tilt seed")
	}
}
