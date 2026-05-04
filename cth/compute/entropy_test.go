package compute

import (
	"math"
	"testing"

	"github.com/helpful-engineering/cth/model"
)

func TestEntropyCalculations(t *testing.T) {
	// Test InputEntropy (Definition 7)
	const sf = 2 // 2 sig figures
	expectedInput := 6.64
	if got := InputEntropy(sf); math.Abs(got-expectedInput) > 0.01 {
		t.Errorf("InputEntropy(%d) = %v; want %v", sf, got, expectedInput)
	}

	// Test PredictionEntropyInheritance (Definition 7)
	// Case 1: μ = 1.0 (lossless)
	if got := PredictionEntropyInheritance(10.0, 1.0); got != 10.0 {
		t.Errorf("PredictionEntropyInheritance(10.0, 1.0) = %v; want 10.0", got)
	}

	// Case 2: μ = 0.5 (max uncertainty leakage)
	// η_leak = (1-0.5) * log2(1/0.5) = 0.5 * 1.0 = 0.5
	expectedLeak := 10.5
	if got := PredictionEntropyInheritance(10.0, 0.5); got != expectedLeak {
		t.Errorf("PredictionEntropyInheritance(10.0, 0.5) = %v; want %v", got, expectedLeak)
	}

	// Test Anchor entropy (Definition 7/7a)
	axiom := model.Anchor{Tier: model.Axiom, ResidualEntropyBits: 2.0}
	if got := ResidualEntropy(axiom); got != 2.0 {
		t.Errorf("ResidualEntropy(axiom) = %v; want 2.0", got)
	}
	if got := ConfirmatoryInfo(axiom); got != 0 {
		t.Errorf("ConfirmatoryInfo(axiom) = %v; want 0", got)
	}

	meas := model.Anchor{Tier: model.Measurement, ConfirmatoryInfoBits: 3.32}
	if got := ConfirmatoryInfo(meas); got != 3.32 {
		t.Errorf("ConfirmatoryInfo(meas) = %v; want 3.32", got)
	}
}
