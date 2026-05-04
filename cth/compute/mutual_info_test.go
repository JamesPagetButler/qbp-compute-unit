package compute

import (
	"math"
	"testing"
)

func TestMutualInformation(t *testing.T) {
	// Perfect agreement → high MI (Definition 10).
	// I = 0.5 * log2(1 + 2/ε) ≈ 25 bits
	mi := PairwiseMI(1.0, 1.0, 1.0, 1.0)
	if mi < 20 {
		t.Errorf("expected high MI for perfect agreement, got %v", mi)
	}

	// Large disagreement (diff = 10, sigmas = 1.0) → low MI.
	miLow := PairwiseMI(10.0, 0.0, 1.0, 1.0)
	if miLow > 0.1 {
		t.Errorf("expected low MI for large disagreement, got %v", miLow)
	}

	// CappedMI (Definition 10a).
	caps := []float64{10.0, 5.0, 8.0}
	if got := CappedMI(25.0, caps); got != 5.0 {
		t.Errorf("CappedMI(25.0, min=5.0) = %v; want 5.0", got)
	}
	if got := CappedMI(3.0, caps); got != 3.0 {
		t.Errorf("CappedMI(3.0, min=5.0) = %v; want 3.0", got)
	}

	// StructuralMI.
	if got := StructuralMI(3, 10.0); got != 3.0 {
		t.Errorf("StructuralMI(3, 10.0) = %v; want 3.0", got)
	}
	if got := StructuralMI(10, 5.0); got != 5.0 {
		t.Errorf("StructuralMI(10, 5.0) = %v; want 5.0", got)
	}
}

func TestNaryMI(t *testing.T) {
	// N=2 should equal PairwiseMI.
	p2 := NaryMI([]float64{1.0, 1.5}, []float64{1.0, 1.0})
	pw := PairwiseMI(1.0, 1.5, 1.0, 1.0)
	if math.Abs(p2-pw) > 1e-10 {
		t.Errorf("NaryMI(N=2) = %v; want PairwiseMI = %v", p2, pw)
	}

	// Perfect 3-way agreement → total correlation is unbounded (+Inf).
	nary3Perfect := NaryMI([]float64{1.0, 1.0, 1.0}, []float64{1.0, 1.0, 1.0})
	if !math.IsInf(nary3Perfect, 1) {
		t.Errorf("NaryMI(perfect 3-way) should be +Inf, got %v", nary3Perfect)
	}

	// Partial 3-way agreement: total correlation > sum of pairwise MIs (synergy term).
	// Predictions: [1.0, 1.5, 2.0], sigmas: [1.0, 1.0, 1.0]
	// Expected: T ≈ 4.1 bits > pairwise sum ≈ 2.4 bits
	preds3 := []float64{1.0, 1.5, 2.0}
	sigs3 := []float64{1.0, 1.0, 1.0}
	nary3 := NaryMI(preds3, sigs3)
	pairSum := PairwiseMI(preds3[0], preds3[1], sigs3[0], sigs3[1]) +
		PairwiseMI(preds3[0], preds3[2], sigs3[0], sigs3[2])
	if nary3 <= pairSum {
		t.Errorf("3-way NaryMI (%v) should exceed sum of pairwise (%v)", nary3, pairSum)
	}

	// Single or empty predictions → 0.
	if got := NaryMI([]float64{1.0}, []float64{1.0}); got != 0 {
		t.Errorf("NaryMI(N=1) = %v; want 0", got)
	}
	if got := NaryMI(nil, nil); got != 0 {
		t.Errorf("NaryMI(nil) = %v; want 0", got)
	}
}
