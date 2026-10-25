package mdcu

// Tests for the R3-Q2 host-ns statistics (frozen §3.3): a Mann–Whitney U test
// equivalent to benchstat's default, and the benchstat-verdict rule.

import (
	"math"
	"testing"
)

func TestMWUCompleteSeparationExact(t *testing.T) {
	a := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	b := []float64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	p := MannWhitneyP(a, b)
	want := 2.0 / 184756.0 // 2 / C(20,10)
	if math.Abs(p-want)/want > 1e-9 {
		t.Fatalf("p=%v want %v", p, want)
	}
	if q := MannWhitneyP(b, a); math.Abs(q-p) > 1e-15 {
		t.Fatalf("not symmetric: %v vs %v", p, q)
	}
}

func TestMWUIdenticalSamplesNotSignificant(t *testing.T) {
	a := []float64{5, 6, 7, 8, 9, 10, 11, 12, 13, 14}
	if p := MannWhitneyP(a, a); p < 0.9 {
		t.Fatalf("identical samples p=%v", p)
	}
}

func TestMWUKnownSmallCase(t *testing.T) {
	// n1=n2=3, U=0 (full separation): exact two-sided p = 2/C(6,3) = 0.1
	p := MannWhitneyP([]float64{1, 2, 3}, []float64{4, 5, 6})
	if math.Abs(p-0.1) > 1e-12 {
		t.Fatalf("p=%v want 0.1", p)
	}
}

func TestMWUTiesUseApproximationAndStaySane(t *testing.T) {
	a := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	b := []float64{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	p := MannWhitneyP(a, b)
	if !(p > 0 && p < 0.001) {
		t.Fatalf("tied but fully separated should be tiny, got %v", p)
	}
	if p := MannWhitneyP(a, a); p < 0.9 {
		t.Fatalf("all-tied identical: %v", p)
	}
}

func TestBenchVerdictRules(t *testing.T) {
	fast := []float64{10, 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 10.8, 10.9}
	slow := []float64{40, 40.1, 40.2, 40.3, 40.4, 40.5, 40.6, 40.7, 40.8, 40.9}
	same := []float64{40.05, 40.15, 40.25, 40.35, 40.45, 40.55, 40.65, 40.75, 40.85, 40.95}
	mid := []float64{25, 25.1, 25.2, 25.3, 25.4, 25.5, 25.6, 25.7, 25.8, 25.9}

	// CD ≤ 0.5× non-CD with p<0.05 on a workload → yes
	if v := BenchVerdict([]BenchRow{{"W1", fast, slow}}); v != "yes" {
		t.Fatalf("want yes got %s", v)
	}
	// CD slower than non-CD everywhere → no
	if v := BenchVerdict([]BenchRow{{"W1", slow, fast}, {"W2", slow, fast}}); v != "no" {
		t.Fatalf("want no got %s", v)
	}
	// indistinguishable (~) everywhere → no
	if v := BenchVerdict([]BenchRow{{"W1", slow, same}}); v != "no" {
		t.Fatalf("want no for ~, got %s", v)
	}
	// significantly faster but not 2× → inconclusive
	if v := BenchVerdict([]BenchRow{{"W1", mid, slow}}); v != "inconclusive" {
		t.Fatalf("want inconclusive got %s", v)
	}
	// yes on one workload outranks no on another
	if v := BenchVerdict([]BenchRow{{"W1", slow, fast}, {"W2", fast, slow}}); v != "yes" {
		t.Fatalf("want yes got %s", v)
	}
}

func TestLoadCeiling(t *testing.T) {
	if !LoadOK(1.0, 1.5) || !LoadOK(0.2, 0.4) {
		t.Fatal("within ceiling must be ok")
	}
	if LoadOK(1.01, 0.5) {
		t.Fatal("start load > 1.0 must fail")
	}
	if LoadOK(0.5, 1.51) {
		t.Fatal("end load > 1.5 must fail")
	}
}

func TestParseBenchOutput(t *testing.T) {
	txt := `goos: linux
BenchmarkR3Q2W3CD            	    5000	    250000 ns/op
BenchmarkR3Q2W3CD-1          	    5000	    251000 ns/op
BenchmarkR3Q2W3NonCD         	    9000	    120000 ns/op
BenchmarkR3Q2W2CD            	    9000	     30000 ns/op
BenchmarkR3Q2W2NonCD         	    9000	     20000 ns/op
BenchmarkR3Q2W1NonCD_B0cross 	    9000	     95000 ns/op
BenchmarkR3Q2W1CD_B1lean     	    9000	     99000 ns/op
PASS
`
	m := ParseBenchOutput(txt)
	if len(m["BenchmarkR3Q2W3CD"]) != 2 || m["BenchmarkR3Q2W3NonCD"][0] != 120000 {
		t.Fatalf("parsed %v", m)
	}
	rows, err := BenchRows(m)
	if err != nil {
		t.Fatal(err)
	}
	// W1: best CD = B1lean (99000) vs best non-CD = B0cross (95000)
	var w1 *BenchRow
	for i := range rows {
		if rows[i].Workload == "W1" {
			w1 = &rows[i]
		}
	}
	if w1 == nil || Median(w1.CD) != 99000 || Median(w1.NonCD) != 95000 {
		t.Fatalf("W1 row: %+v", w1)
	}
}
