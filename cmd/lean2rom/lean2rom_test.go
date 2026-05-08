package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorStability(t *testing.T) {
	// Create a temp directory for output
	tmpDir, err := os.MkdirTemp("", "lean2rom-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Derive tables using native Go fallback (deterministic)
	tables := deriveNative()
	src := "test-source"

	// Emit ROMs to temp dir
	if err := emitROMs(tmpDir, tables, src); err != nil {
		t.Fatalf("emitROMs failed: %v", err)
	}

	// Verify qmath_constants.s exists and contains expected symbols
	asmPath := filepath.Join(tmpDir, "emulator", "qmath_constants.s")
	content, err := os.ReadFile(asmPath)
	if err != nil {
		t.Fatalf("failed to read generated asm: %v", err)
	}

	expectedSymbols := []string{
		"GLOBL qbp_lean_sign_x(SB)",
		"GLOBL qbp_lean_sign_y(SB)",
		"GLOBL qbp_lean_sign_z(SB)",
		"GLOBL qbp_lean_conj(SB)",
	}

	for _, sym := range expectedSymbols {
		if !strings.Contains(string(content), sym) {
			t.Errorf("generated asm missing expected symbol: %s", sym)
		}
	}

	// Verify no Plan-9 private markers remain
	if strings.Contains(string(content), "<>") {
		t.Errorf("generated asm still contains Plan-9 private marker '<>'")
	}
}
