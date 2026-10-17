package mdcu

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// A0: CD-MAC gradient ≡ basis gradient.
func TestA0_GradEquivalence(t *testing.T) {
	r := rng()
	worst := 0.0
	var cb, cc Counter
	for n := 0; n < 20000; n++ {
		var s [16]float64
		copy(s[:], randVec(r, 16))
		s[0] = 0
		renorm16(&s, nil)
		a, b := GradBasis(&s, &cb), GradCDMAC(&s, &cc)
		worst = math.Max(worst, norm(sub(a[:], b[:])))
	}
	t.Logf("max ‖g_basis − g_cdmac‖ = %.3g; OMUL/eval basis %d, cdmac %d", worst, cb.OMUL/20000, cc.OMUL/20000)
	if worst > 1e-14 {
		t.Fatal("A0 gate failed")
	}
}

// A2: three engines on the #688 seeds.
func TestA2_ThreeEngines(t *testing.T) {
	seeds := rf688(t)
	var c [3]Counter
	out := make([][][16]float64, 3)
	for e := EngBasis; e <= EngLeaf; e++ {
		out[e] = RunQuench(seeds, 0.02, 400, e, &c[e])
	}
	maxd, mean := 0.0, 0.0
	for i := range seeds {
		for k := 0; k < 16; k++ {
			maxd = math.Max(maxd, math.Abs(out[0][i][k]-out[1][i][k]))
			maxd = math.Max(maxd, math.Abs(out[0][i][k]-out[2][i][k]))
		}
		mean += out[2][i][8] * out[2][i][8]
	}
	mean /= float64(len(seeds))
	// closed form at long time: b0²(∞) = b0²/(b0² + √((1−b0²)² − V0·k)), k∈{1,¼}
	long := RunQuench(seeds, 0.02, 1000, EngLeaf, nil)
	var errK [2]float64
	for i, s := range seeds {
		b2 := s[8] * s[8]
		for j, k := range []float64{1, 0.25} {
			pred := b2 / (b2 + math.Sqrt((1-b2)*(1-b2)-Potential(&s)*k))
			errK[j] = math.Max(errK[j], math.Abs(pred-long[i][8]*long[i][8]))
		}
	}
	t.Logf("engine agreement max %.3g; ⟨b0²⟩(t=8) = %.8f (canon 0.13365313)", maxd, mean)
	t.Logf("closed form @t=20 max err: V0=Potential %.3g, V0=Potential/4 %.3g", errK[0], errK[1])
	for e := range c {
		t.Logf("%-5v OMUL=%d fmul=%d docMACs=%d", Engine(e), c[e].OMUL, c[e].FMul, c[e].DocMACs)
	}
	if maxd > 1e-13 || math.Abs(mean-0.13365313) > 5e-9 {
		t.Fatal("A2 gate failed")
	}
}

// Pillar 2 vs canonical results.json (report-only per plan: PARTIAL on mismatch).
func TestPillar2(t *testing.T) {
	za := loadSeeds(t, "testA_Z")
	var z0 [3][]float64
	for k := range z0 {
		z0[k] = za[k]
	}
	round := func(v float64, d int) float64 { p := math.Pow(10, float64(d)); return math.Round(v*p) / p }
	canonA := map[string][3]float64{ // rmin, final r, final ‖A‖² (5 dp), steps
		"pure_gravity": {0.05222, 0.03016, 0}, "g2_conserved": {0.38075, 1.50017, 3.76226},
		"so8_fast_slip": {0.05968, 0.04288, 0}, "so8_high_inertia": {0.38065, 1.50003, 3.75193},
	}
	canonSteps := map[string]int{"pure_gravity": 2195, "g2_conserved": 4719, "so8_fast_slip": 2374, "so8_high_inertia": 4734}
	var qa uint64
	for _, c := range []struct {
		m    string
		pure bool
		I    float64
	}{{"pure_gravity", true, 0}, {"g2_conserved", false, 0}, {"so8_fast_slip", false, 0.005}, {"so8_high_inertia", false, 2.0}} {
		r := RunBounce(c.m, z0, c.pure, c.I)
		qa += r.QAssoc
		t.Logf("A %-17s bounced=%v plunged=%v rmin=%.5f final=%.5f A2 %.5f→%.5f steps=%d qassoc=%d",
			r.Mode, r.Bounced, r.Plunged, r.RMin, r.FinalR, r.InitA2, r.FinalA2, r.Steps, r.QAssoc)
		w := canonA[c.m]
		if round(r.RMin, 5) != w[0] || round(r.FinalR, 5) != w[1] || round(r.FinalA2, 5) != w[2] || r.Steps != canonSteps[c.m] {
			t.Errorf("A %s mismatch vs canon %v", c.m, w)
		}
	}
	t.Logf("A total QASSOC %d (canon 194248; includes per-run diagnostic ‖A‖² reads)", qa)
	zb := loadSeeds(t, "testB_Z")
	raw, _ := os.ReadFile("testdata/numpy_seeds.json")
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	var tri [][][]int
	if err := json.Unmarshal(m["testB_triads"], &tri); err != nil {
		t.Fatal(err)
	}
	t3 := func(c int) [3][]int { return [3][]int{tri[c][0], tri[c][1], tri[c][2]} }
	t.Logf("B initial %+v (canon 2.0570/1.7575/0.5824)", ringMetrics(zb, t3(0)))
	t.Logf("B local   %+v (canon 0.69837/1.66143/0.6097)", ringMetrics(RunRing(zb, false), t3(1)))
	t.Logf("B locked  %+v (canon 0.0019834/0.0164209/0.998)", ringMetrics(RunRing(zb, true), t3(2)))
	snap, g, ct := RunSieve(loadSeeds(t, "testC_W"), nil)
	for _, k := range []int{0, 15, 100} {
		t.Logf("C t%-3d %+v", k, snap[k])
	}
	t.Logf("C ledgers ground %.4f (canon 31.5934) containment %.4f (canon 19.3425)", g, ct)
	c100 := snap[100]
	if round(c100.Norm, 6) != 0.000932 || round(c100.Alt, 6) != 0.048362 || round(c100.Assoc, 6) != 0.195015 ||
		round(snap[15].Assoc, 6) != 0.875411 || round(g, 4) != 31.5934 || round(ct, 4) != 19.3425 {
		t.Error("C mismatch vs canon")
	}
}
