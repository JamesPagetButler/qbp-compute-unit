package store

import (
	"os"
	"testing"
)

func TestJSONRoundTrip(t *testing.T) {
	// Test on minimal fixture
	path := "../testdata/minimal.json"
	inv, err := LoadInventory(path)
	if err != nil {
		t.Fatalf("LoadInventory(%s) failed: %v", path, err)
	}

	// Verify basic field
	if inv.Programme != "Minimal Test" {
		t.Errorf("expected programme 'Minimal Test', got '%s'", inv.Programme)
	}

	// Save to temp file
	tmpPath := "minimal_test_tmp.json"
	defer os.Remove(tmpPath)

	if err := SaveInventory(inv, tmpPath); err != nil {
		t.Fatalf("SaveInventory failed: %v", err)
	}

	// Load back and compare
	inv2, err := LoadInventory(tmpPath)
	if err != nil {
		t.Fatalf("LoadInventory(tmp) failed: %v", err)
	}

	if inv2.Programme != inv.Programme {
		t.Errorf("round-trip failed: expected programme '%s', got '%s'", inv.Programme, inv2.Programme)
	}

	if len(inv2.Axioms) != len(inv.Axioms) {
		t.Errorf("round-trip failed: expected %d axioms, got %d", len(inv.Axioms), len(inv2.Axioms))
	}
}

func TestLoadMultiple(t *testing.T) {
	paths := []string{
		"../testdata/qbp_v3_2.json",
		"../testdata/qbp_quantum_v0_1.json",
	}

	invs, err := LoadMultiple(paths)
	if err != nil {
		t.Fatalf("LoadMultiple failed: %v", err)
	}

	if len(invs) != 2 {
		t.Errorf("expected 2 inventories, got %d", len(invs))
	}

	if invs[0].Programme != "QBP" {
		t.Errorf("expected programme[0] 'QBP', got '%s'", invs[0].Programme)
	}

	if invs[1].Programme != "QBP-Quantum" {
		t.Errorf("expected programme[1] 'QBP-Quantum', got '%s'", invs[1].Programme)
	}
}
