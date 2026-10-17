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
	for _, c := range []struct {
		m    string
		pure bool
		I    float64
	}{{"pure_gravity", true, 0}, {"g2_conserved", false, 0}, {"so8_fast_slip", false, 0.005}, {"so8_high_inertia", false, 2.0}} {
		r := RunBounce(c.m, z0, c.pure, c.I)
		t.Logf("A %-17s bounced=%v plunged=%v rmin=%.5f final=%.5f A2 %.5f→%.5f steps=%d qassoc=%d",
			r.Mode, r.Bounced, r.Plunged, r.RMin, r.FinalR, r.InitA2, r.FinalA2, r.Steps, r.QAssoc)
	}
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
}
