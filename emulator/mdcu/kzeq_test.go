package mdcu

// qbp-kz-g2 round 3 (FROZEN-v1, sha256 132f6a42…1a623) tests, written before the code:
// the seed table and its guards (§2 Seeds, §5 seed guard), the frozen T table, the start
// kinds per replicate, the record count of KZEqRun, the energy-density read, the Sokal
// window and the under-sampling flag, and the KZExecuteEqOnly orchestration.

import (
	"math"
	"math/rand/v2"
	"sort"
	"strings"
	"testing"
)

// §2 T grid: a frozen literal, typed not computed.
func TestKZKzG2R3TTableLiteral(t *testing.T) {
	want := []float64{1.08, 1.10, 1.12, 1.13, 1.14, 1.15, 1.16, 1.17, 1.18, 1.20}
	if len(KZKzG2R3TTable) != len(want) {
		t.Fatalf("T table has %d entries, want %d", len(KZKzG2R3TTable), len(want))
	}
	for i := range want {
		if KZKzG2R3TTable[i] != want[i] {
			t.Errorf("T[%d] = %v, want %v", i, KZKzG2R3TTable[i], want[i])
		}
	}
	if got := KZKzG2R3LSet; len(got) != 3 || got[0] != 8 || got[1] != 12 || got[2] != 16 {
		t.Errorf("L set %v, want [8 12 16]", got)
	}
	for l, L := range []int{8, 12, 16} {
		if idx, ok := KZKzG2R3LIndex(L); !ok || idx != l {
			t.Errorf("L index of %d = %d/%v, want %d", L, idx, ok, l)
		}
	}
	if _, ok := KZKzG2R3LIndex(24); ok {
		t.Error("L = 24 must have no index")
	}
}

// §2 Seeds: 27 processes, 81 seeds, all in 6000-6118, no repeats; seed = 6000+40l+4t+j.
func TestKZKzG2R3SeedTable(t *testing.T) {
	var all []uint64
	procs := 0
	for l := 0; l < 3; l++ {
		for tt := 0; tt < 10; tt++ {
			s := KZKzG2R3Seeds(l, tt, false)
			if l == 0 && (tt == 3 || tt == 5 || tt == 7) {
				if s != nil {
					t.Errorf("L = 8, t = %d must be refused (nil seeds), got %v", tt, s)
				}
				continue
			}
			procs++
			if len(s) != 3 {
				t.Fatalf("(l=%d,t=%d): %d seeds, want 3", l, tt, len(s))
			}
			for j, v := range s {
				if want := uint64(6000 + 40*l + 4*tt + j); v != want {
					t.Errorf("(l=%d,t=%d,j=%d) seed %d, want %d", l, tt, j, v, want)
				}
				if c := KZClassifySeed(v); c != KZSeedKzG2R3 {
					t.Errorf("seed %d classified %v, want %v", v, c, KZSeedKzG2R3)
				}
			}
			all = append(all, s...)
			r := KZKzG2R3Seeds(l, tt, true)
			for j := range r {
				if r[j] != s[j]+120 {
					t.Errorf("re-run seed (l=%d,t=%d,j=%d) %d, want %d", l, tt, j, r[j], s[j]+120)
				}
				if c := KZClassifySeed(r[j]); c != KZSeedKzG2R3Rerun {
					t.Errorf("re-run seed %d classified %v, want %v", r[j], c, KZSeedKzG2R3Rerun)
				}
			}
		}
	}
	if procs != 27 {
		t.Errorf("%d processes, want 27", procs)
	}
	if len(all) != 81 {
		t.Fatalf("%d seeds, want 81", len(all))
	}
	sort.Slice(all, func(a, b int) bool { return all[a] < all[b] })
	if all[0] != 6000 || all[80] != 6118 {
		t.Errorf("seed range %d..%d, want 6000..6118", all[0], all[80])
	}
	for i := 1; i < len(all); i++ {
		if all[i] == all[i-1] {
			t.Errorf("seed %d repeats", all[i])
		}
	}
	// out-of-range indices
	for _, c := range [][2]int{{-1, 0}, {3, 0}, {0, -1}, {0, 10}} {
		if s := KZKzG2R3Seeds(c[0], c[1], false); s != nil {
			t.Errorf("(l=%d,t=%d) must give nil, got %v", c[0], c[1], s)
		}
	}
	// the registry: the two new classes, their bounds, and the neighbours still free
	for _, c := range []struct {
		seed uint64
		want KZSeedClass
	}{{5999, KZSeedFree}, {6000, KZSeedKzG2R3}, {6119, KZSeedKzG2R3}, {6120, KZSeedKzG2R3Rerun}, {6239, KZSeedKzG2R3Rerun}, {6240, KZSeedFree}, {5980, KZSeedFree}} {
		if got := KZClassifySeed(c.seed); got != c.want {
			t.Errorf("seed %d: class %v, want %v", c.seed, got, c.want)
		}
	}
	if KZSeedKzG2R3.String() != "kz-g2-r3" || KZSeedKzG2R3Rerun.String() != "kz-g2-r3-rerun" {
		t.Errorf("class names %q / %q", KZSeedKzG2R3, KZSeedKzG2R3Rerun)
	}
}

// §5 seed guard: every other mode (smokes included) refuses 6000-6239; -run-eq refuses
// everything but the derived 3-seed sub-block of (l, t); the re-run block needs the flag.
func TestKZKzG2R3SeedGuards(t *testing.T) {
	for _, s := range []uint64{6000, 6050, 6118, 6119, 6120, 6200, 6239} {
		if err := KZCheckSmokeSeeds([]uint64{s}); err == nil {
			t.Errorf("smoke guard accepted seed %d", s)
		}
	}
	if err := KZCheckSmokeSeeds([]uint64{5998, 5999, 6000}); err == nil {
		t.Error("smoke guard accepted a block straddling into 6000")
	}
	if err := KZCheckSmokeSeeds([]uint64{6239, 6240}); err == nil {
		t.Error("smoke guard accepted a block straddling out of 6239")
	}
	if err := KZCheckSmokeSeeds([]uint64{13000, 13001}); err != nil {
		t.Errorf("free smoke seeds refused: %v", err)
	}
	ok := func(seeds []uint64, l, tt int, rerun bool) error { return KZCheckKzG2R3Seeds(seeds, l, tt, rerun) }
	if err := ok([]uint64{6100, 6101, 6102}, 2, 5, false); err != nil {
		t.Errorf("the derived sub-block of (16, t=5) refused: %v", err)
	}
	if err := ok([]uint64{6220, 6221, 6222}, 2, 5, true); err != nil {
		t.Errorf("the derived re-run sub-block of (16, t=5) refused: %v", err)
	}
	bad := []struct {
		name  string
		seeds []uint64
		l, tt int
		rerun bool
	}{
		{"wrong block", []uint64{6104, 6105, 6106}, 2, 5, false},
		{"two seeds", []uint64{6100, 6101}, 2, 5, false},
		{"four seeds (j = 3 is unused)", []uint64{6100, 6101, 6102, 6103}, 2, 5, false},
		{"out of order", []uint64{6102, 6101, 6100}, 2, 5, false},
		{"re-run block without the flag", []uint64{6220, 6221, 6222}, 2, 5, false},
		{"run block under the flag", []uint64{6100, 6101, 6102}, 2, 5, true},
		{"L = 8 at t = 3", []uint64{6012, 6013, 6014}, 0, 3, false},
		{"L = 8 at t = 5", []uint64{6020, 6021, 6022}, 0, 5, false},
		{"L = 8 at t = 7", []uint64{6028, 6029, 6030}, 0, 7, false},
		{"free seeds", []uint64{13000, 13001, 13002}, 2, 5, false},
		{"round-1 seeds", []uint64{5900, 5901, 5902}, 0, 0, false},
		{"bad l", []uint64{6000, 6001, 6002}, 3, 0, false},
		{"bad t", []uint64{6000, 6001, 6002}, 0, 10, false},
		{"empty", nil, 0, 0, false},
	}
	for _, c := range bad {
		if err := ok(c.seeds, c.l, c.tt, c.rerun); err == nil {
			t.Errorf("%s: guard accepted %v for (l=%d,t=%d,rerun=%v)", c.name, c.seeds, c.l, c.tt, c.rerun)
		}
	}
}

// §2 start rule: j = 0, 1 Haar (hot), j = 2 aligned (cold).
func TestKZKzG2R3StartKinds(t *testing.T) {
	got := KZKzG2R3Starts()
	if len(got) != 3 || got[0] != "haar" || got[1] != "haar" || got[2] != "aligned" {
		t.Errorf("starts %v, want [haar haar aligned]", got)
	}
	// the constructor dispatch
	f := kzEqStartField("aligned", 4, 13001)
	if e := f.Energy(); math.Abs(e) > 1e-12 {
		t.Errorf("aligned field energy %g, want 0 (to rounding of the frozen Energy())", e)
	}
	g := kzEqStartField("haar", 4, 13001)
	if e := g.Energy(); !(e > 0) {
		t.Errorf("Haar field energy %g, want > 0", e)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("an unknown start kind must panic")
			}
		}()
		kzEqStartField("warm", 4, 13001)
	}()
}

// §2 row 1: e = Energy()/(3 L^3) is 0 for the aligned field and positive, <= 3, for Haar.
func TestKZEqEnergyDensity(t *testing.T) {
	L := 4
	// 0 to the rounding of the frozen Energy() (3 − ‖F Fᵀ‖² of a unit frame: ~1e-14)
	if e := KZEqEnergyDensity(NewKZPlaneFieldAligned(L, 13000)); math.Abs(e) > 1e-12 {
		t.Errorf("aligned e = %g, want 0", e)
	}
	e := KZEqEnergyDensity(NewKZPlaneFieldHaar(L, 13000))
	if !(e > 0 && e <= 3) {
		t.Errorf("Haar e = %g, want in (0, 3]", e)
	}
	// consistent with the frozen Energy() and the bond count 3N
	f := NewKZPlaneFieldHaar(L, 13002)
	if want := f.Energy() / float64(3*L*L*L); math.Abs(KZEqEnergyDensity(f)-want) > 1e-15 {
		t.Errorf("e = %g, want Energy/3N = %g", KZEqEnergyDensity(f), want)
	}
}

// KZEqRun: exactly Sampling/RecEvery records, no step-0 record; Check polled every step
// including burn-in; the frozen defaults are 5,000 / 50,000 / 10 -> 5,000 records.
func TestKZEqRunRecordCount(t *testing.T) {
	d := DefaultKZEqParams()
	if d.BurnIn != 5000 || d.Sampling != 50000 || d.RecEvery != 10 || KZEqRecords(d) != 5000 {
		t.Errorf("frozen defaults %+v (records %d), want 5000/50000/10 -> 5000", d, KZEqRecords(d))
	}
	prm := KZEqParams{BurnIn: 7, Sampling: 50, RecEvery: 10}
	L := 4
	calls := 0
	check := func() error { calls++; return nil }
	f := NewKZPlaneFieldAligned(L, 13003)
	e0 := KZEqEnergyDensity(f)
	S, E := KZEqRun(f, 13003, 0.1, 1.15, prm, check)
	if len(S) != 5 || len(E) != 5 {
		t.Fatalf("%d/%d records, want 5 (no step-0 record)", len(S), len(E))
	}
	if calls != prm.BurnIn+prm.Sampling {
		t.Errorf("Check called %d times, want %d (every step incl. burn-in)", calls, prm.BurnIn+prm.Sampling)
	}
	// no step-0 record: the first record is after 10 steps at T > 0, so e has moved off 0
	if E[0] == e0 {
		t.Errorf("first record equals the start state (step-0 record?): e = %g", E[0])
	}
	for i := range S {
		if math.IsNaN(S[i]) || math.IsNaN(E[i]) || S[i] < 0 || S[i] > 1+1e-9 || E[i] < 0 || E[i] > 3 {
			t.Errorf("record %d: S = %g e = %g out of range", i, S[i], E[i])
		}
	}
	// determinism: same seed, same series (the eq noise stream is a function of the seed only)
	S2, E2 := KZEqRun(NewKZPlaneFieldAligned(L, 13003), 13003, 0.1, 1.15, prm, nil)
	for i := range S {
		if S[i] != S2[i] || E[i] != E2[i] {
			t.Fatalf("record %d differs between identical runs", i)
		}
	}
	// the eq stream is new: a Stage-0b-stream run from the same seed and start differs
	if kzStreamEq != 100002 || kzStreamEq == kzStreamStage0b || kzStreamEq == kzStreamStage0a {
		t.Errorf("kzStreamEq = %d, want 100002 distinct from the Stage 0 streams", kzStreamEq)
	}
}

// KZEqRun aborts through Check (any step, burn-in included) with the check's code.
func TestKZEqRunCheckAbortsInBurnIn(t *testing.T) {
	prm := KZEqParams{BurnIn: 5, Sampling: 20, RecEvery: 10}
	n := 0
	check := func() error {
		n++
		if n == 3 {
			return &KZAbortError{Code: 3, Reason: "pause file present"}
		}
		return nil
	}
	defer func() {
		r := recover()
		ae, ok := r.(*KZAbortError)
		if !ok || ae.Code != 3 {
			t.Fatalf("recovered %v, want *KZAbortError code 3", r)
		}
		if n != 3 {
			t.Errorf("abort after %d checks, want 3 (inside the burn-in)", n)
		}
	}()
	KZEqRun(NewKZPlaneFieldHaar(4, 13004), 13004, 0.1, 1.15, prm, check)
	t.Fatal("no abort")
}

// Sokal window: W = smallest W with W >= 6 tau(W), tau(W) = 1/2 + sum_{k<=W} rho(k), record
// units; n_eff = N/(2 tau); undersampled iff n_eff < 20.
func TestKZSokalTauInt(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	// white noise: tau ~ 0.5
	n := 4000
	x := make([]float64, n)
	for i := range x {
		x[i] = r.NormFloat64()
	}
	tau, W := KZSokalTauInt(x, 6)
	if math.Abs(tau-0.5) > 0.15 {
		t.Errorf("white noise tau = %.3f (W = %d), want ~0.5", tau, W)
	}
	if W < 1 || float64(W) < 6*tau {
		t.Errorf("window W = %d violates W >= 6 tau = %.2f", W, 6*tau)
	}
	// AR(1) with phi = 0.9: tau_int = (1+phi)/(2(1-phi)) = 9.5
	phi := 0.9
	y := make([]float64, 20000)
	for i := 1; i < len(y); i++ {
		y[i] = phi*y[i-1] + math.Sqrt(1-phi*phi)*r.NormFloat64()
	}
	tau, W = KZSokalTauInt(y, 6)
	if math.Abs(tau-9.5) > 2.5 {
		t.Errorf("AR(1) phi=0.9 tau = %.2f (W = %d), want ~9.5", tau, W)
	}
	// a constant series has no correlation to sum: tau = 0.5 by convention, no NaN
	c := make([]float64, 100)
	for i := range c {
		c[i] = 1.25
	}
	if tau, _ = KZSokalTauInt(c, 6); tau != 0.5 {
		t.Errorf("constant series tau = %v, want 0.5", tau)
	}
	// the flag arithmetic
	if ne, under := KZEqNEff(5000, 10); math.Abs(ne-250) > 1e-9 || under {
		t.Errorf("tau 10: n_eff %.1f under %v, want 250 false", ne, under)
	}
	if ne, under := KZEqNEff(5000, 125.0001); ne >= 20 || !under {
		t.Errorf("tau 125+: n_eff %.3f under %v, want < 20 true", ne, under)
	}
	if ne, under := KZEqNEff(5000, 125); math.Abs(ne-20) > 1e-9 || under {
		t.Errorf("tau 125: n_eff %.3f under %v, want 20 false (strict <)", ne, under)
	}
	if ne, _ := KZEqNEff(5000, 0.1); math.IsInf(ne, 0) || math.IsNaN(ne) || ne > 5000 {
		t.Errorf("tau below 1/2 must floor at the white-noise value: n_eff %v", ne)
	}
}

// KZExecuteEqOnly: guards before any step (exit 4), the record shapes, no pooled
// statistic in the raw records, peak Sys and ms_per_step in end.
func TestKZEqOnlyFlowAndGuards(t *testing.T) {
	prm := KZEqParams{BurnIn: 4, Sampling: 20, RecEvery: 10}
	var recs []map[string]any
	base := KZEqOnlyConfig{L: 4, LIndex: -1, TIndex: 5, T: 1.15, Eta: 0.1, Smoke: true,
		Seeds: []uint64{13010, 13011}, Starts: []string{"haar", "aligned"}, Prm: prm,
		Emit: func(r map[string]any) { recs = append(recs, r) }, Meta: map[string]any{"prereg_sha": "abc"}}
	if _, err := KZExecuteEqOnly(base); err != nil {
		t.Fatalf("smoke flow: %v", err)
	}
	kinds := []string{}
	for _, r := range recs {
		kinds = append(kinds, r["kind"].(string))
	}
	if strings.Join(kinds, ",") != "meta,eq,end" {
		t.Fatalf("records %v, want meta,eq,end", kinds)
	}
	meta, eq, end := recs[0], recs[1], recs[2]
	if meta["mode"] != "smoke-eq" || meta["prereg_sha"] != "abc" || meta["noise_stream"] != kzStreamEq {
		t.Errorf("meta %v", meta)
	}
	for _, k := range []string{"L", "l_index", "T", "t_index", "eta", "seeds", "burn_in", "sampling", "rec_every", "noise_stream", "runs"} {
		if _, ok := eq[k]; !ok {
			t.Errorf("eq record lacks %q", k)
		}
	}
	for _, k := range []string{"mean_e", "var_e", "binder_v", "histogram", "barrier", "delta_f"} {
		if _, ok := eq[k]; ok {
			t.Errorf("eq record carries the pooled statistic %q (blindness, §7.1)", k)
		}
	}
	runs := eq["runs"].([]KZEqSeedResult)
	if len(runs) != 2 || runs[0].Start != "haar" || runs[1].Start != "aligned" || runs[0].Seed != 13010 {
		t.Fatalf("runs %+v", runs)
	}
	for _, ru := range runs {
		if len(ru.S) != 2 || len(ru.E) != 2 {
			t.Errorf("seed %d: %d/%d records, want 2", ru.Seed, len(ru.S), len(ru.E))
		}
		if math.IsNaN(ru.TauIntRec) || math.IsNaN(ru.NEff) {
			t.Errorf("seed %d: tau/n_eff NaN", ru.Seed)
		}
	}
	if end["steps"] != 2*(prm.BurnIn+prm.Sampling) {
		t.Errorf("end steps %v, want %d", end["steps"], 2*(prm.BurnIn+prm.Sampling))
	}
	for _, k := range []string{"wall_s", "ms_per_step", "peak_sys_mb"} {
		if _, ok := end[k]; !ok {
			t.Errorf("end record lacks %q", k)
		}
	}
	if end["peak_sys_mb"].(float64) <= 0 {
		t.Errorf("peak_sys_mb %v, want > 0", end["peak_sys_mb"])
	}

	// guards: each refused with code 4 before any record is written
	refuse := func(name string, mut func(c *KZEqOnlyConfig)) {
		c := base
		c.Seeds = append([]uint64(nil), base.Seeds...)
		c.Starts = append([]string(nil), base.Starts...)
		recs = nil
		mut(&c)
		_, err := KZExecuteEqOnly(c)
		ae, ok := err.(*KZAbortError)
		if !ok || ae.Code != 4 {
			t.Errorf("%s: err %v, want abort code 4", name, err)
		}
		if len(recs) != 0 {
			t.Errorf("%s: %d records written before the guard", name, len(recs))
		}
	}
	refuse("smoke on a run-block seed", func(c *KZEqOnlyConfig) { c.Seeds[0] = 6000 })
	refuse("smoke on a re-run-block seed", func(c *KZEqOnlyConfig) { c.Seeds[1] = 6239 })
	refuse("smoke on a round-1 seed", func(c *KZEqOnlyConfig) { c.Seeds[0] = 5900 })
	refuse("starts/seeds length mismatch", func(c *KZEqOnlyConfig) { c.Starts = c.Starts[:1] })
	refuse("unknown start kind", func(c *KZEqOnlyConfig) { c.Starts[0] = "warm" })
	refuse("no seeds", func(c *KZEqOnlyConfig) { c.Seeds, c.Starts = nil, nil })
	refuse("bad L", func(c *KZEqOnlyConfig) { c.L = 1 })
	refuse("bad T", func(c *KZEqOnlyConfig) { c.T = 0 })
	refuse("NaN T", func(c *KZEqOnlyConfig) { c.T = math.NaN() })
	refuse("bad eta", func(c *KZEqOnlyConfig) { c.Eta = 0 })
	refuse("bad rec_every", func(c *KZEqOnlyConfig) { c.Prm.RecEvery = 0 })
	refuse("sampling below rec_every", func(c *KZEqOnlyConfig) { c.Prm.Sampling = 5 })

	// run mode (Smoke = false): the frozen table binds everything
	run := KZEqOnlyConfig{L: 16, LIndex: 2, TIndex: 5, T: 1.15, Eta: 0.1, Seeds: KZKzG2R3Seeds(2, 5, false),
		Starts: KZKzG2R3Starts(), Prm: DefaultKZEqParams(), Emit: func(r map[string]any) { recs = append(recs, r) }}
	refuseRun := func(name string, mut func(c *KZEqOnlyConfig)) {
		c := run
		c.Seeds = append([]uint64(nil), run.Seeds...)
		c.Starts = append([]string(nil), run.Starts...)
		recs = nil
		mut(&c)
		_, err := KZExecuteEqOnly(c)
		ae, ok := err.(*KZAbortError)
		if !ok || ae.Code != 4 {
			t.Errorf("run: %s: err %v, want abort code 4", name, err)
		}
		if len(recs) != 0 {
			t.Errorf("run: %s: %d records written before the guard", name, len(recs))
		}
	}
	refuseRun("L off the set", func(c *KZEqOnlyConfig) { c.L = 24 })
	refuseRun("l index does not match L", func(c *KZEqOnlyConfig) { c.LIndex = 1 })
	refuseRun("t out of range", func(c *KZEqOnlyConfig) { c.TIndex = 10 })
	refuseRun("T does not match the table at t", func(c *KZEqOnlyConfig) { c.T = 1.16 })
	refuseRun("seeds of another point", func(c *KZEqOnlyConfig) { c.Seeds = KZKzG2R3Seeds(2, 6, false) })
	refuseRun("re-run seeds without the flag", func(c *KZEqOnlyConfig) { c.Seeds = KZKzG2R3Seeds(2, 5, true) })
	refuseRun("run seeds under the re-run flag", func(c *KZEqOnlyConfig) { c.Rerun = true })
	refuseRun("free seeds", func(c *KZEqOnlyConfig) { c.Seeds = []uint64{13000, 13001, 13002} })
	refuseRun("starts not haar,haar,aligned", func(c *KZEqOnlyConfig) { c.Starts = []string{"haar", "aligned", "haar"} })
	refuseRun("shortened burn-in", func(c *KZEqOnlyConfig) { c.Prm.BurnIn = 500 })
	refuseRun("shortened sampling", func(c *KZEqOnlyConfig) { c.Prm.Sampling = 2000 })
	refuseRun("other record cadence", func(c *KZEqOnlyConfig) { c.Prm.RecEvery = 20 })
	refuseRun("eta not the frozen 0.1", func(c *KZEqOnlyConfig) { c.Eta = 0.05 })
	refuseRun("L = 8 at t = 3", func(c *KZEqOnlyConfig) {
		c.L, c.LIndex, c.TIndex, c.T, c.Seeds = 8, 0, 3, 1.13, []uint64{6012, 6013, 6014}
	})
	refuseRun("no Emit is fine but no Check and no seeds is not", func(c *KZEqOnlyConfig) { c.Seeds = nil })
}

// An abort at a step emits one abort record and returns the code (pause 3; NaN 6 via the
// screened order read). The start check fires before the first step.
func TestKZEqOnlyAborts(t *testing.T) {
	prm := KZEqParams{BurnIn: 4, Sampling: 20, RecEvery: 10}
	var recs []map[string]any
	n := 0
	cfg := KZEqOnlyConfig{L: 4, TIndex: 0, T: 1.08, Eta: 0.1, Smoke: true, Seeds: []uint64{13020}, Starts: []string{"haar"},
		Prm: prm, Emit: func(r map[string]any) { recs = append(recs, r) },
		Check: func() error {
			n++
			if n == 6 {
				return &KZAbortError{Code: 3, Reason: "pause file present"}
			}
			return nil
		}}
	_, err := KZExecuteEqOnly(cfg)
	ae, ok := err.(*KZAbortError)
	if !ok || ae.Code != 3 {
		t.Fatalf("err %v, want abort code 3", err)
	}
	if len(recs) != 2 || recs[0]["kind"] != "meta" || recs[1]["kind"] != "abort" || recs[1]["code"] != 3 {
		t.Errorf("records %v, want meta then abort code 3", recs)
	}
	if _, ok := recs[1]["steps"]; !ok {
		t.Errorf("abort record lacks the step count (kill vs no-run needs it)")
	}
	// pause at start: abort before step 1, steps = 0
	recs = nil
	cfg.Check = func() error { return &KZAbortError{Code: 3, Reason: "pause file present"} }
	if _, err = KZExecuteEqOnly(cfg); err == nil {
		t.Fatal("no abort at start")
	}
	if len(recs) != 2 || recs[1]["steps"] != 0 {
		t.Errorf("abort at start: %v", recs)
	}
	// a generic error becomes code 1
	recs = nil
	cfg.Check = func() error { return errStub("disk") }
	_, err = KZExecuteEqOnly(cfg)
	if ae, ok := err.(*KZAbortError); !ok || ae.Code != 1 {
		t.Errorf("generic check error -> %v, want code 1", err)
	}
}

type errStub string

func (e errStub) Error() string { return string(e) }
