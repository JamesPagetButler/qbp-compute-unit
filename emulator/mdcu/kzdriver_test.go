package mdcu

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestKZFrozenSeedsAreInTheirClasses(t *testing.T) {
	s := KZFrozenSeeds()
	chk := func(name string, seeds []uint64, c KZSeedClass, n int) {
		t.Helper()
		if len(seeds) != n {
			t.Fatalf("%s: %d seeds, want %d", name, len(seeds), n)
		}
		for _, x := range seeds {
			if KZClassifySeed(x) != c {
				t.Fatalf("%s: seed %d is class %v, want %v", name, x, KZClassifySeed(x), c)
			}
		}
	}
	chk("main", s.Main, KZSeedMain, 200)
	chk("null", s.Null, KZSeedNull, 200)
	chk("main0a", s.MainStage0a, KZSeedStage0, 3)
	chk("main0b", s.MainStage0b, KZSeedStage0, 10)
	chk("null0a", s.NullStage0a, KZSeedStage0, 3)
	chk("null0b", s.NullStage0b, KZSeedStage0, 10)
	chk("saturation", s.Saturation, KZSeedSaturation, 20)
	chk("eta", s.Eta, KZSeedProbe, 5)
	chk("step16", s.Step16, KZSeedProbe, 5)
	chk("step24", s.Step24, KZSeedProbe, 5)
	chk("det", s.Det, KZSeedProbe, 5)
	seen := map[uint64]string{}
	for name, l := range map[string][]uint64{"mainA": s.MainStage0a, "mainB": s.MainStage0b, "nullA": s.NullStage0a,
		"nullB": s.NullStage0b, "sat": s.Saturation, "eta": s.Eta, "s16": s.Step16, "s24": s.Step24, "det": s.Det} {
		for _, x := range l {
			if o, dup := seen[x]; dup {
				t.Fatalf("seed %d used by both %s and %s", x, o, name)
			}
			seen[x] = name
		}
	}
}

func TestKZFrozenRunConfig(t *testing.T) {
	cfg := KZFrozenRunConfig(16, 100, 0.2, 0.10)
	if cfg.L != 16 || cfg.R != 100 || cfg.Eta != 0.2 || cfg.MainSeed0 != 5000 || cfg.NullSeed0 != 5200 ||
		cfg.SatSeed0 != 5630 || cfg.SatN != 20 || len(cfg.Ladder) != 7 {
		t.Fatalf("%+v", cfg)
	}
	if cfg.Stage0 != DefaultKZStage0Params() {
		t.Fatal("stage 0 parameters must be the frozen ones")
	}
	if cfg.MainStage0a[0] != 5600 || cfg.NullStage0a[0] != 5605 || cfg.MainStage0b[0] != 5610 || cfg.NullStage0b[0] != 5620 {
		t.Fatal("stage 0 seeds")
	}
	// replicate j of the null reads with tilt seed 5650+j, in the tilt class; Stage 0 arms carry none
	for _, j := range []uint64{0, 7, 99} {
		n := cfg.MkNull(5200 + j).(*KZNullArm)
		if n.Sigma != 0.10 || n.TiltSeed != 5650+j || KZClassifySeed(n.TiltSeed) != KZSeedTilt {
			t.Fatalf("null replicate %d: sigma %v tilt seed %d", j, n.Sigma, n.TiltSeed)
		}
	}
	if cfg.MkNull(5620).(*KZNullArm).TiltSeed != 0 {
		t.Fatal("a Stage 0 null arm has no tilt seed")
	}
	if _, ok := KZPinnedSigma(16); ok {
		t.Fatal("no sigma* may be pinned before the calibration is run and amend-1 is signed")
	}
}

func kzGoodProbe() KZProbeReport {
	s := KZFrozenSeeds()
	rep := KZProbeReport{Kind: "probe", Eta: KZEtaLadder[0], EtaOK: true,
		EtaSeeds: s.Eta, Step16Seeds: s.Step16, Step24Seeds: s.Step24, DetSeeds: s.Det,
		TS16: []float64{1e-4, 1e-4, 1e-4, 1e-4, 1e-4}, TS24: []float64{3e-4, 3e-4, 3e-4, 3e-4, 3e-4},
		TD16: []float64{1e-3, 1e-3, 1e-3, 1e-3, 1e-3}, TD24: []float64{3e-3, 3e-3, 3e-3, 3e-3, 3e-3}}
	rep.Timing = KZProbeTiming{TS16: 1e-4, TD16: 1e-3, TS24: 3e-4, TD24: 3e-3}
	rep.Decision = KZProbeDecide(rep.Timing)
	return rep
}

func TestKZValidateProbeAccepts(t *testing.T) {
	d, err := KZValidateProbe(kzGoodProbe())
	if err != nil {
		t.Fatal(err)
	}
	if !d.Run || d.L != 24 || d.R != 200 {
		t.Fatalf("%+v", d)
	}
}

func TestKZValidateProbeRefuses(t *testing.T) {
	cases := map[string]func(r *KZProbeReport){
		"wrong kind":         func(r *KZProbeReport) { r.Kind = "grid" },
		"eta seeds":          func(r *KZProbeReport) { r.EtaSeeds = []uint64{1, 2, 3, 4, 5} },
		"step seeds":         func(r *KZProbeReport) { r.Step16Seeds[0] = 5000 },
		"det seeds":          func(r *KZProbeReport) { r.DetSeeds = r.DetSeeds[:4] },
		"eta not on ladder":  func(r *KZProbeReport) { r.Eta = 0.123456 },
		"eta not ok":         func(r *KZProbeReport) { r.EtaOK = false },
		"timing not medians": func(r *KZProbeReport) { r.Timing.TS16 *= 2 },
		"short arrays":       func(r *KZProbeReport) { r.TS24 = r.TS24[:3] },
		"decision tampered": func(r *KZProbeReport) {
			r.Decision.R = 25
		},
		"round does not run": func(r *KZProbeReport) {
			for i := range r.TS16 {
				r.TS16[i], r.TS24[i] = 1, 1
			}
			r.Timing = KZProbeTiming{TS16: 1, TD16: 1e-3, TS24: 1, TD24: 3e-3}
			r.Decision = KZProbeDecide(r.Timing)
		},
	}
	for name, mut := range cases {
		r := kzGoodProbe()
		r.Step16Seeds = append([]uint64{}, r.Step16Seeds...)
		mut(&r)
		if _, err := KZValidateProbe(r); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestKZRunProbeShape(t *testing.T) {
	spec := KZProbeSpec{L1: 4, L2: 6, Steps: 3,
		Eta: []uint64{13500, 13501}, Step16: []uint64{13510, 13511}, Step24: []uint64{13520, 13521}, Det: []uint64{13530, 13531}}
	rep, err := KZRunProbe(spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.TS16) != 2 || len(rep.TS24) != 2 || len(rep.TD16) != 2 || len(rep.TD24) != 2 {
		t.Fatalf("%+v", rep)
	}
	for _, v := range append(append(append(append([]float64{}, rep.TS16...), rep.TS24...), rep.TD16...), rep.TD24...) {
		if !(v > 0) {
			t.Fatalf("non-positive timing %v", v)
		}
	}
	if rep.Timing.TS16 != KZMedian(rep.TS16) || rep.Timing.TD24 != KZMedian(rep.TD24) {
		t.Fatal("timings are medians over seeds")
	}
	if rep.Decision != KZProbeDecide(rep.Timing) {
		t.Fatal("decision follows from the timings")
	}
	if !rep.EtaOK || rep.Eta <= 0 {
		t.Fatalf("eta %v ok %v", rep.Eta, rep.EtaOK)
	}
}

func TestKZRunProbeCheckAborts(t *testing.T) {
	spec := KZProbeSpec{L1: 4, L2: 6, Steps: 3,
		Eta: []uint64{13500}, Step16: []uint64{13510}, Step24: []uint64{13520}, Det: []uint64{13530}}
	_, err := KZRunProbe(spec, func() error { return &KZAbortError{Code: 3, Reason: "pause"} })
	if err == nil {
		t.Fatal("check error must abort the probe")
	}
}

// ---- reading the run records back ---------------------------------------------------------

func kzJSONL(recs []map[string]any) *bytes.Reader {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	for _, r := range recs {
		_ = enc.Encode(r)
	}
	return bytes.NewReader(b.Bytes())
}

func kzSynthRun(ladder []int, R int, drop func(arm string, tau, rep int) bool) []map[string]any {
	recs := []map[string]any{{"kind": "saturation", "density": 0.466}}
	for j := 0; j < R; j++ {
		for _, arm := range []string{"main", "null"} {
			for _, tau := range ladder {
				if drop != nil && drop(arm, tau, j) {
					continue
				}
				d := 0
				if arm == "main" {
					d = 10 + j
				}
				recs = append(recs, map[string]any{"kind": "run", "arm": arm, "tau": tau, "rep": j, "cells": 512,
					"defects": d, "ambiguous": 3, "neg": d + 2, "parity": true})
			}
		}
	}
	return append(recs, map[string]any{"kind": "end", "verdict": "complete", "runs": 0})
}

func TestKZLoadRunRecords(t *testing.T) {
	ladder := []int{25, 50}
	recs := kzSynthRun(ladder, 4, nil)
	main, null, err := KZLoadRunRecords(kzJSONL(recs), ladder, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(main) != 2 || len(null) != 2 || main[0].Tau != 25 || main[1].Tau != 50 {
		t.Fatalf("%+v", main)
	}
	if len(main[0].Defects) != 4 || main[0].Defects[3] != 13 || main[0].Cells != 512 || main[0].Neg[0] != 12 ||
		main[0].Ambiguous[0] != 3 || null[1].Defects[0] != 0 {
		t.Fatalf("%+v", main[0])
	}
}

func TestKZLoadRunRecordsRefusesPartialData(t *testing.T) {
	ladder := []int{25, 50}
	cases := map[string][]map[string]any{
		"missing replicate": kzSynthRun(ladder, 4, func(a string, tau, j int) bool { return a == "main" && tau == 50 && j == 2 }),
		"no end record": func() []map[string]any {
			r := kzSynthRun(ladder, 4, nil)
			return r[:len(r)-1]
		}(),
		"aborted": func() []map[string]any {
			r := kzSynthRun(ladder, 4, nil)
			r[len(r)-1] = map[string]any{"kind": "abort", "code": 3, "reason": "pause"}
			return r
		}(),
		"premise failed": func() []map[string]any {
			return []map[string]any{{"kind": "end", "verdict": "premise failed", "runs": 0}}
		}(),
		"duplicate": func() []map[string]any {
			r := kzSynthRun(ladder, 4, nil)
			return append(r[:2], r[1:]...)
		}(),
	}
	for name, recs := range cases {
		if _, _, err := KZLoadRunRecords(kzJSONL(recs), ladder, 4); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, _, err := KZLoadRunRecords(strings.NewReader("not json\n"), ladder, 4); err == nil {
		t.Error("garbage accepted")
	}
}
