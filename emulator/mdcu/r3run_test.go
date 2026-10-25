package mdcu

// Round-3 run-layer tests (frozen §1.2 sampler roles/miss rules, §1.4 arms).
// Written before r3run.go. Out-of-grid seeds only: 9900 (even/definite),
// 9901 (odd/split), 9902/9903 for miss-injection. The R3 grid is 3000–3199.

import (
	"math"
	"testing"
)

func armSet(arms []R3Arm) map[R3Arm]bool {
	m := map[R3Arm]bool{}
	for _, a := range arms {
		m[a] = true
	}
	return m
}

func TestR3InGrid(t *testing.T) {
	for _, s := range []uint64{3000, 3100, 3199} {
		if !R3InGrid(s) {
			t.Fatalf("seed %d must be in the R3 grid", s)
		}
	}
	for _, s := range []uint64{2999, 3200, 9900, 1000, 77} {
		if R3InGrid(s) {
			t.Fatalf("seed %d must not be in the R3 grid", s)
		}
	}
}

func TestR3PsiEvenSeedUsesS2(t *testing.T) {
	p := BuildR3Psi(9900)
	if p.Stratum != "definite" || p.Sampler != "S2" || p.Primary == nil {
		t.Fatalf("even seed: stratum=%q sampler=%q primary=%v", p.Stratum, p.Sampler, p.Primary)
	}
	if kind, _ := ClassifyPsi(p.Primary); kind != "definite" {
		t.Fatalf("S2 primary classified %q, want definite", kind)
	}
	want := SampleS2(9900)
	if p.S2.S != want.S || p.S2.Dist != want.Dist || p.S2.Cond != want.Cond {
		t.Fatal("recorded S2 draw differs from SampleS2")
	}
	if p.S1Miss {
		if p.S1Def != nil {
			t.Fatal("S1 miss but S1Def set")
		}
	} else {
		if p.S1Def == nil || p.S1Draws < 1 {
			t.Fatal("S1 definite form missing without a miss")
		}
		if kind, _ := ClassifyPsi(p.S1Def); kind != "definite" {
			t.Fatalf("S1Def classified %q", kind)
		}
	}
}

func TestR3PsiOddSeedUsesS1Split(t *testing.T) {
	p := BuildR3Psi(9901)
	if p.Stratum != "split" || p.Sampler != "S1" {
		t.Fatalf("odd seed: stratum=%q sampler=%q", p.Stratum, p.Sampler)
	}
	if p.S1Miss {
		t.Skip("split-type S1 miss (astronomically unlikely)")
	}
	if kind, _ := ClassifyPsi(p.Primary); kind != "split" {
		t.Fatalf("S1 primary classified %q, want split", kind)
	}
	if p.S1Def != nil || p.S2 != nil {
		t.Fatal("odd seed must not carry S1Def/S2")
	}
}

func TestR3ArmsForSeed(t *testing.T) {
	even := armSet(R3ArmsFor(BuildR3Psi(9900)))
	for _, a := range []R3Arm{ArmFull, ArmGram, ArmPhi, ArmPsi, ArmPsiGram, ArmRand3, ArmNone} {
		if !even[a] {
			t.Fatalf("even seed missing arm %s", a)
		}
	}
	if pe := BuildR3Psi(9900); even[ArmPsiS1] == pe.S1Miss {
		t.Fatal("K-psi-S1 must run on even seeds iff no S1 miss")
	}
	odd := armSet(R3ArmsFor(BuildR3Psi(9901)))
	if odd[ArmPsiS1] {
		t.Fatal("K-psi-S1 must not run on odd seeds")
	}
	if !odd[ArmPsi] || !odd[ArmPsiGram] {
		t.Fatal("odd seed with an S1 split must run K-psi and K-psigram")
	}
}

// Frozen miss rule: a miss on the primary sampler drops K-psi and K-psigram;
// a miss on the S1-definite sampler (even seeds) drops only K-psi-S1.
func TestR3MissRules(t *testing.T) {
	miss := func(seed uint64, want string) (S1Draw, bool) { return S1Draw{Draws: r3S1Cap}, false }
	even := buildR3Psi(9902, miss)
	if !even.S1Miss || even.S1Draws != r3S1Cap {
		t.Fatalf("injected miss not recorded: %+v", even)
	}
	ea := armSet(R3ArmsFor(even))
	if ea[ArmPsiS1] || !ea[ArmPsi] || !ea[ArmPsiGram] {
		t.Fatalf("even-seed S1 miss must drop only K-psi-S1: %v", ea)
	}
	odd := buildR3Psi(9903, miss)
	oa := armSet(R3ArmsFor(odd))
	if oa[ArmPsi] || oa[ArmPsiGram] || oa[ArmPsiS1] {
		t.Fatalf("odd-seed miss must drop K-psi and K-psigram: %v", oa)
	}
	for _, a := range []R3Arm{ArmFull, ArmGram, ArmPhi, ArmRand3, ArmNone} {
		if !oa[a] {
			t.Fatalf("non-ψ arm %s must survive a miss", a)
		}
	}
}

func TestRunR3SeedRecords(t *testing.T) {
	p := BuildR3Psi(9900)
	cell := R3Cell{Name: "unit", N: 12, Steps: 20, Sigma: 0.1}
	runs := RunR3Seed(cell, p)
	if len(runs) != len(R3ArmsFor(p)) {
		t.Fatalf("%d runs for %d arms", len(runs), len(R3ArmsFor(p)))
	}
	var full *R3Run
	for i := range runs {
		r := &runs[i]
		if r.Seed != 9900 || r.Cell != "unit" || r.N != 12 || r.Steps != 20 || r.Stratum != "definite" {
			t.Fatalf("bad header %+v", r)
		}
		if math.Abs(r.Stats.Eig[0]+r.Stats.Eig[1]+r.Stats.Eig[2]+r.Stats.Eig[3]+r.Stats.Eig[4]+r.Stats.Eig[5]+r.Stats.Eig[6]-12) > 1e-9 {
			t.Fatalf("%s: eigenvalue sum != N", r.Arm)
		}
		if r.Arm == ArmFull {
			full = r
		}
	}
	if full == nil {
		t.Fatal("no K-full run")
	}
	// K-full must reproduce round 2's RunRingGen/Q1Measure exactly (same init
	// stream, 250 steps).
	z0 := Q1Init(12, 0.1, newRNG(9900))
	q := Q1Measure(RunRingGen(z0, LayOverlap, assoc8))
	r250 := RunR3Seed(R3Cell{Name: "r2", N: 12, Steps: 250, Sigma: 0.1, Arms: []R3Arm{ArmFull}}, p)
	if len(r250) != 1 || math.Abs(r250[0].Stats.S3-q.SVD3) > 1e-12 {
		t.Fatalf("K-full S3 differs from RunRingGen S3 %v", q.SVD3)
	}
}

// Determinism: same cell + seed => identical records.
func TestRunR3SeedDeterministic(t *testing.T) {
	p := BuildR3Psi(9901)
	cell := R3Cell{Name: "unit", N: 12, Steps: 15, Sigma: 0.1}
	a, b := RunR3Seed(cell, p), RunR3Seed(cell, p)
	for i := range a {
		if a[i].Arm != b[i].Arm || a[i].Stats.S3 != b[i].Stats.S3 || a[i].Stats.Eig != b[i].Stats.Eig {
			t.Fatalf("non-deterministic arm %s", a[i].Arm)
		}
	}
}

// Cell.Arms restricts the arms (used by the N=48 steps cell).
func TestRunR3SeedArmFilter(t *testing.T) {
	p := BuildR3Psi(9901)
	cell := R3Cell{Name: "unit", N: 12, Steps: 10, Sigma: 0.1, Arms: []R3Arm{ArmFull, ArmGram}}
	runs := RunR3Seed(cell, p)
	if len(runs) != 2 || runs[0].Arm != ArmFull || runs[1].Arm != ArmGram {
		t.Fatalf("arm filter wrong: %d runs", len(runs))
	}
}
