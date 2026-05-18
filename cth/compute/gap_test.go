package compute

import (
	"math"
	"testing"

	"github.com/helpful-engineering/cth/store"
)

func TestGapCalculations(t *testing.T) {
	// Test StepDifficulty (Definition 16)
	if got := StepDifficulty("routine_lean"); got != 0.1 {
		t.Errorf("StepDifficulty(routine_lean) = %v; want 0.1", got)
	}
	if got := StepDifficulty("irreducible"); !math.IsInf(got, 1) {
		t.Errorf("StepDifficulty(irreducible) = %v; want +Inf", got)
	}

	// Test WeightedGap on minimal.json
	inv, err := store.LoadInventory("../testdata/minimal.json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// In minimal.json:
	// MEAS-1 depends on PR-1 (proven). Path length 1.
	// IN-1 depends on nothing.
	// Wait, minimal.json chains:
	// CH-1: AX-1 -> PR-1
	// CH-2: PR-1 -> MEAS-1

	// We need a path from an input to a proven anchor.
	// Let's modify the inventory in-memory for testing if needed,
	// but let's check current state.

	gap, nearest := WeightedGap("MEAS-1", inv)
	if gap != 1.0 || nearest != "PR-1" {
		t.Errorf("WeightedGap(MEAS-1) = %v, %s; want 1.0, PR-1", gap, nearest)
	}

	// Test Proximity
	prox := EddyProximity("IN-1", inv)
	if prox != 0 {
		t.Errorf("EddyProximity(IN-1) = %v; want 0 (no path to proven)", prox)
	}

	// Test Ranking
	rankings := RankEddies(inv)
	if len(rankings) != 1 {
		t.Errorf("expected 1 input ranking, got %d", len(rankings))
	}
}
