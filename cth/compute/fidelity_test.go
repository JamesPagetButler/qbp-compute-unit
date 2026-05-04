package compute

import (
	"testing"

	"github.com/helpful-engineering/cth/model"
)

func TestFidelity(t *testing.T) {
	// Test StepFidelity (§4.4)
	if got := StepFidelity("lean"); got != 1.0 {
		t.Errorf("StepFidelity(lean) = %v; want 1.0", got)
	}
	if got := StepFidelity("standard_physics"); got != 0.999 {
		t.Errorf("StepFidelity(standard_physics) = %v; want 0.999", got)
	}
	if got := StepFidelity("conjecture"); got != 0.5 {
		t.Errorf("StepFidelity(conjecture) = %v; want 0.5", got)
	}

	// Test ChainFidelity
	c := model.Chain{Fidelity: 0.95}
	if got := ChainFidelity(c); got != 0.95 {
		t.Errorf("ChainFidelity(c) = %v; want 0.95", got)
	}

	// Test ClassifyFidelityRegime (§4.4)
	tests := []struct {
		mu   float64
		want string
	}{
		{0.9995, "laminar"},
		{0.95, "low_sediment"},
		{0.8, "moderate"},
		{0.5, "heavy"},
	}

	for _, tt := range tests {
		if got := ClassifyFidelityRegime(tt.mu); got != tt.want {
			t.Errorf("ClassifyFidelityRegime(%v) = %v; want %v", tt.mu, got, tt.want)
		}
	}
}
