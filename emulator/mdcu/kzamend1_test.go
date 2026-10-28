package mdcu

// Tests for the dated amendment mdcu-kz-prereg-FROZEN-v1-amend-1 (seq 205/206):
//   1. resolution rule: a triangle is ambiguous iff any of its three edge rotations has
//      angle >= pi/2 (the pi clause on the triangle holonomy is kept, redundant);
//   3. the S3 arm's detector receives only the SO(3) frame field Ad_q.
// FROZEN-v1's own tests (a)-(h) in kzdetect_test.go are unchanged and still run against
// DefaultZ2Params (the frozen pi rule).

import (
	"math"
	"testing"
)

func TestKZAmend1EdgeRuleTriangle(t *testing.T) {
	p := Amend1Z2Params()
	idq := [4]float64{1, 0, 0, 0}
	rot := func(angle float64) [4]float64 { return [4]float64{math.Cos(angle / 2), math.Sin(angle / 2), 0, 0} }
	// the holonomy of this triangle is the identity (e, e^-1, id) but one edge rotates by pi/2
	e := rot(math.Pi / 2)
	_, amb := TriangleZ2P(e, QuatConj(e), idq, p)
	if !amb {
		t.Fatalf("an edge rotation of exactly pi/2 must make the triangle ambiguous")
	}
	// just below pi/2 on all edges: not ambiguous
	e = rot(math.Pi/2 - 1e-3)
	if _, amb = TriangleZ2P(e, QuatConj(e), idq, p); amb {
		t.Fatalf("edge rotations below pi/2 must not be ambiguous")
	}
	// the edge lift's overall sign (w<0) must not matter: angle is read from |w|
	en := [4]float64{-e[0], -e[1], -e[2], -e[3]}
	if _, amb = TriangleZ2P(en, QuatConj(en), idq, p); amb {
		t.Fatalf("edge ambiguity must be sign-of-lift independent")
	}
	big := rot(math.Pi / 2)
	bn := [4]float64{-big[0], -big[1], -big[2], -big[3]}
	if _, amb = TriangleZ2P(bn, QuatConj(bn), idq, p); !amb {
		t.Fatalf("edge ambiguity must be sign-of-lift independent (ambiguous case)")
	}
	// the frozen rule is a different, vacuous rule: it does not fire here
	if _, amb = TriangleZ2P(rot(math.Pi/2), QuatConj(rot(math.Pi/2)), idq, DefaultZ2Params()); amb {
		t.Fatalf("frozen rule must not fire on a trivial-holonomy triangle")
	}
	// the redundant second clause still works
	ex := rot(math.Pi / 4)
	if _, amb = TriangleZ2P(ex, ex, ex, p); amb { // product = 3pi/4 rotation, edges pi/4: holonomy not ~pi
		t.Fatalf("unexpected ambiguity")
	}
	if _, amb = TriangleZ2P(rot(math.Pi/2-1e-3), rot(math.Pi/2-1e-3), idq, p); amb {
		t.Fatalf("holonomy pi-0.002 with sub-pi/2 edges must not be ambiguous")
	}
}

// Frozen rule vs amended rule on i.i.d. Haar: frozen reads ~0% ambiguous (the vacuity
// finding), amended reads ~100% (oppenheimer's pre-data expectation). Smooth: 0% both.
func TestKZAmend1HaarVsSmooth(t *testing.T) {
	L := 8
	N := L * L * L
	frozenAmb, amendAmb := 0, 0
	for s := uint64(0); s < 6; s++ {
		f := haarG2Set(kzTestRand(12100+s), N)
		rf := Z2DetectPlanes(L, f, DefaultZ2Params())
		ra := Z2DetectPlanes(L, f, Amend1Z2Params())
		frozenAmb += rf.Ambiguous
		amendAmb += ra.Ambiguous
		if rf.NegCells != ra.NegCells {
			t.Fatalf("the raw -1 count must not depend on the ambiguity rule: %d vs %d", rf.NegCells, ra.NegCells)
		}
		if !ra.ParityOK {
			t.Fatalf("raw parity must hold on every field, ambiguous or not (seed %d)", s)
		}
		if ra.Defects+ra.Ambiguous > N || ra.Defects > ra.NegCells {
			t.Fatalf("inconsistent counters %+v", ra)
		}
	}
	if frozenAmb != 0 {
		t.Fatalf("frozen rule is expected vacuous on Haar, got %d ambiguous cells", frozenAmb)
	}
	frac := float64(amendAmb) / float64(6*N)
	t.Logf("amended ambiguous fraction on i.i.d. Haar (L=8, 6 fields) = %.4f", frac)
	if frac < 0.95 {
		t.Fatalf("pre-data expectation: i.i.d. Haar ~100%% ambiguous; got %.4f", frac)
	}
}

func TestKZAmend1SmoothFieldsUnambiguous(t *testing.T) {
	L := 12
	N := L * L * L
	// a slowly rotating plane field about one axis: all edge rotations tiny
	f := make([][3][7]float64, N)
	p0 := haarG2Plane(kzTestRand(12110))
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				a := 0.15 * math.Sin(2*math.Pi*float64(i)/float64(L))
				c, s := math.Cos(a), math.Sin(a)
				fr := p0
				for row := 0; row < 3; row++ { // rotate coordinates 0,1 by angle a
					fr[row][0] = c*p0[row][0] - s*p0[row][1]
					fr[row][1] = s*p0[row][0] + c*p0[row][1]
				}
				f[kzSite(L, i, j, k)] = fr
			}
		}
	}
	res := Z2DetectPlanes(L, f, Amend1Z2Params())
	if res.Ambiguous != 0 || res.NegCells != 0 {
		t.Fatalf("smooth field must have 0 ambiguity and 0 defects: %+v", res)
	}
}

// Amendment item 3: the S3 detector gets only Ad_q. The SO(3)-frame entry point and
// the q adapter must agree exactly, and q's global sign must not matter.
func TestKZAmend1S3FramesOnly(t *testing.T) {
	L := 8
	N := L * L * L
	for s := uint64(0); s < 8; s++ {
		r := kzTestRand(12120 + s)
		q := make([][4]float64, N)
		rm := make([][3][3]float64, N)
		for i := range q {
			q[i] = randQuat(r)
			if r.IntN(2) == 0 {
				for c := range q[i] {
					q[i][c] = -q[i][c]
				}
			}
			rm[i] = QuatToMat(q[i])
		}
		a := Z2DetectS3(L, q, Amend1Z2Params())
		b := Z2DetectSO3Frames(L, rm, Amend1Z2Params())
		if a.NegCells != b.NegCells || a.Ambiguous != b.Ambiguous || a.Defects != b.Defects {
			t.Fatalf("seed %d: adapter and frame entry disagree: %+v vs %+v", s, a, b)
		}
		if !b.ParityOK {
			t.Fatalf("seed %d: parity", s)
		}
		// i.i.d. rough S3: many ambiguous cells under the edge rule
		if float64(b.Ambiguous)/float64(N) < 0.5 {
			t.Fatalf("seed %d: rough S3 field expected mostly ambiguous, got %d/%d", s, b.Ambiguous, N)
		}
		// Finding recorded for the amendment: the transport of an exact SO(3) frame field
		// is R_a R_b^T, a rotation, so every triangle holonomy is the identity and every
		// cell reads +1 whatever the field. No detector input can make this arm read -1.
		if b.NegCells != 0 || b.Defects != 0 {
			t.Fatalf("seed %d: S3 frame field read -1 cells: %+v (algebra says impossible)", s, b)
		}
	}
	// smooth S3 field: nothing ambiguous
	q := make([][4]float64, N)
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				a := 0.1 * math.Sin(2*math.Pi*float64(j)/float64(L))
				q[kzSite(L, i, j, k)] = [4]float64{math.Cos(a), math.Sin(a), 0, 0}
			}
		}
	}
	if res := Z2DetectS3(L, q, Amend1Z2Params()); res.Ambiguous != 0 || res.NegCells != 0 {
		t.Fatalf("smooth S3 field: %+v", res)
	}
}

// Saturation sanity (amendment item 4): the raw -1 density over 20 i.i.d. Haar fields
// at the run's L. Logged here at L=8 and L=12; the gate (0.50 +- 0.05) is a runtime
// check in the driver. This test only checks the helper is deterministic and in (0,1).
func TestKZAmend1SaturationRawDensity(t *testing.T) {
	d1 := KZSaturationRawDensity(8, 20, 12200)
	d2 := KZSaturationRawDensity(8, 20, 12200)
	if d1 != d2 {
		t.Fatalf("not deterministic: %v vs %v", d1, d2)
	}
	t.Logf("raw -1 density, 20 i.i.d. Haar fields, L=8: %.4f", d1)
	if d1 <= 0 || d1 >= 1 {
		t.Fatalf("implausible %v", d1)
	}
	if !KZSaturationWithinTol(0.50) || !KZSaturationWithinTol(0.55) || !KZSaturationWithinTol(0.45) {
		t.Fatalf("0.45..0.55 must be inside the tolerance")
	}
	if KZSaturationWithinTol(0.4499) || KZSaturationWithinTol(0.5501) {
		t.Fatalf("outside 0.50 +- 0.05 must be rejected")
	}
}
