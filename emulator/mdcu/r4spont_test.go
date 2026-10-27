package mdcu

// R4 stage 4 tests: the spontaneity routine moved into the library reproduces the
// round-3 published results from the R3 raw (frozen R4 §4 "Tests first"), and
// behaves on planted planes.

import (
	"math"
	"math/rand/v2"
	"testing"
)

// R3 results (doc/spikes/mdcu-round3-results.md): K-full p = 0.545 (n = 198 locked
// planes); the ψ arms p < 0.001. Same RNG stream and arithmetic, so equality is exact.
func TestR4SpontaneityReproducesR3(t *testing.T) {
	if testing.Short() {
		t.Skip("1000-replicate Monte Carlo on ~200 planes")
	}
	d, _, err := LoadR4(r3RawPath)
	if err != nil {
		t.Skipf("R3 raw not available: %v", err)
	}
	full := R4Spontaneity(d.Get("main", ArmFull, ""), ArmFull, "main", 1000)
	if full.N != 198 || math.Abs(full.P-0.545) > 1e-12 || math.Abs(full.D-0.0080) > 5e-5 {
		t.Fatalf("K-full: n=%d D=%.4f p=%.3f, want n=198 D≈0.0080 p=0.545", full.N, full.D, full.P)
	}
	psi := R4Spontaneity(d.Get("main", ArmPsi, ""), ArmPsi, "main", 1000)
	if !psi.Testable || psi.P >= 0.001 {
		t.Fatalf("K-psi: n=%d p=%.4f, want p < 0.001", psi.N, psi.P)
	}
}

func TestR4SpontaneityPlanted(t *testing.T) {
	mkRuns := func(planes [][3][7]float64) []R3Run {
		var rs []R3Run
		for _, p := range planes {
			var r R3Run
			r.Stats.Class, r.Stats.Plane = "3-lock", p
			rs = append(rs, r)
		}
		return rs
	}
	// Haar-G₂ planes from an unrelated stream are consistent with Haar-G₂.
	haar := R4Spontaneity(mkRuns(haarG2Set(newRNG(9977), 40)), ArmFullK, "A250", 200)
	if !haar.Testable || haar.N != 40 || haar.P < 0.05 {
		t.Fatalf("Haar planes: %+v", haar)
	}
	// One plane repeated is the opposite of Haar-uniform.
	one := haarG2Plane(newRNG(9978))
	same := make([][3][7]float64, 40)
	for i := range same {
		same[i] = one
	}
	cl := R4Spontaneity(mkRuns(same), ArmPsi1, "A250", 200)
	if !cl.Testable || cl.P >= 0.01 {
		t.Fatalf("clustered planes: %+v", cl)
	}
	// Fewer than 20 locked runs: untestable, with N reported; non-locked runs ignored.
	few := mkRuns(haarG2Set(newRNG(9979), 19))
	few = append(few, R3Run{Stats: R3Stats{Class: "diffuse"}})
	u := R4Spontaneity(few, ArmPhiK, "A250", 200)
	if u.Testable || u.N != 19 || !math.IsNaN(u.P) {
		t.Fatalf("untestable: %+v", u)
	}
}

// The Haar reference really is associative: it lies in the G₂ orbit of φ's plane
// (u, v, u·v), so φ evaluates to ±1 on it.
func TestR4HaarPlaneIsAssociative(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6))
	for i := 0; i < 20; i++ {
		p := haarG2Plane(r)
		u, v, w := make([]float64, 8), make([]float64, 8), make([]float64, 8)
		for k := 0; k < 7; k++ {
			u[k+1], v[k+1], w[k+1] = p[0][k], p[1][k], p[2][k]
		}
		if x := math.Abs(Phi().Eval(u, v, w)); math.Abs(x-1) > 1e-9 {
			t.Fatalf("|φ(u,v,uv)| = %g, want 1", x)
		}
	}
}
