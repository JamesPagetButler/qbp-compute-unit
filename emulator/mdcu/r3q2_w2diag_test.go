package mdcu

import (
	"strings"
	"testing"
)

func TestW2DiagReportDisappears(t *testing.T) {
	// Unrolled non-CD is as fast as CD: the frozen "CD <= 0.5x non-CD" condition fails => gap explained.
	m := map[string][]float64{
		"BenchmarkR3Q2W2CD":               {8400, 8410, 8420, 8430, 8440, 8450, 8460, 8470, 8480, 8490},
		"BenchmarkR3Q2W2NonCD":            {23000, 23010, 23020, 23030, 23040, 23050, 23060, 23070, 23080, 23090},
		"BenchmarkPostHocW2NonCDUnrolled": {8000, 8010, 8020, 8030, 8040, 8050, 8060, 8070, 8080, 8090},
	}
	r, err := W2DiagReport(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r, "gap explained") {
		t.Fatalf("expected 'gap explained', got:\n%s", r)
	}
	if !strings.Contains(r, "POST-HOC DIAGNOSTIC") || !strings.Contains(r, "no verdict change") {
		t.Fatalf("report must be labelled post-hoc, no verdict change:\n%s", r)
	}
}

func TestW2DiagReportSurvives(t *testing.T) {
	// Unrolled non-CD is still 2.7x slower than CD: the gap survives table removal.
	m := map[string][]float64{
		"BenchmarkR3Q2W2CD":               {8400, 8410, 8420, 8430, 8440, 8450, 8460, 8470, 8480, 8490},
		"BenchmarkR3Q2W2NonCD":            {23000, 23010, 23020, 23030, 23040, 23050, 23060, 23070, 23080, 23090},
		"BenchmarkPostHocW2NonCDUnrolled": {22000, 22010, 22020, 22030, 22040, 22050, 22060, 22070, 22080, 22090},
	}
	r, err := W2DiagReport(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r, "gap survives") {
		t.Fatalf("expected 'gap survives', got:\n%s", r)
	}
}

func TestW2DiagReportMissing(t *testing.T) {
	if _, err := W2DiagReport(map[string][]float64{"BenchmarkR3Q2W2CD": {1, 2}}); err == nil {
		t.Fatal("expected an error when benchmarks are missing")
	}
}
