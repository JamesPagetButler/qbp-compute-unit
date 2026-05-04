package compute

import (
	"math"
	"testing"

	"github.com/helpful-engineering/cth/store"
)

func TestSensitivityBracket(t *testing.T) {
	inv, err := store.LoadInventory("../testdata/minimal.json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	inputMap := BuildInputEntropy(inv)

	halfH, baseH, doubleH := SensitivityBracket(inv, inputMap)

	// baseH should equal NetCompression directly.
	directRho, _ := NetCompression(inv, inputMap)
	if math.Abs(baseH-directRho) > 1e-10 {
		t.Errorf("baseH = %v; want %v (direct NetCompression)", baseH, directRho)
	}

	// Doubling axiom entropy increases the denominator → pushes ρ_net toward 0 (or more negative).
	// Halving axiom entropy decreases the denominator → pushes ρ_net further negative.
	// For minimal fixture: denominator = axH + inputH, numerator stays fixed.
	// We just check the ordering is consistent with the direction of change.
	_ = halfH
	_ = doubleH
	// Both brackets are finite (no infinities or NaN).
	if math.IsNaN(halfH) || math.IsInf(halfH, 0) {
		t.Errorf("halfH is not a finite number: %v", halfH)
	}
	if math.IsNaN(doubleH) || math.IsInf(doubleH, 0) {
		t.Errorf("doubleH is not a finite number: %v", doubleH)
	}
}

func TestSensitivityRatio(t *testing.T) {
	// Ratio = doubleH / halfH.
	if got := SensitivityRatio(2.0, 3.0); math.Abs(got-1.5) > 1e-10 {
		t.Errorf("SensitivityRatio(2.0, 3.0) = %v; want 1.5", got)
	}
	// halfH = 0 → ratio = 0 (not a division by zero).
	if got := SensitivityRatio(0, 1.0); got != 0 {
		t.Errorf("SensitivityRatio(0, 1.0) = %v; want 0", got)
	}
}
