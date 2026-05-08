package emulator

import (
	"os"
	"testing"
)

func TestParseAsmConstants(t *testing.T) {
	// Create a dummy asm file with known DATA blocks
	content := `
DATA sym1+0x00(SB)/8, $0x8000000000000000 // Lane 0 neg
DATA sym1+0x08(SB)/8, $0x0000000000000000 // Lane 1 pos
DATA sym1+0x10(SB)/8, $0x8000000000000000 // Lane 2 neg
DATA sym1+0x18(SB)/8, $0x0000000000000000 // Lane 3 pos
GLOBL sym1(SB), RODATA, $32

DATA sym2+0x00(SB)/8, $0x0000000000000000
DATA sym2+0x08(SB)/8, $0x8000000000000000
DATA sym2+0x10(SB)/8, $0x8000000000000000
DATA sym2+0x18(SB)/8, $0x8000000000000000
GLOBL sym2(SB), RODATA, $32
`
	tmpFile, err := os.CreateTemp("", "test_asm_*.s")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	masks, err := parseAsmConstants(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseAsmConstants failed: %v", err)
	}

	// Verify sym1
	m1, ok := masks["sym1"]
	if !ok {
		t.Errorf("sym1 missing from parsed masks")
	}
	expected1 := asmSignMask{true, false, true, false}
	if m1 != expected1 {
		t.Errorf("sym1 mismatch: got %v, want %v", m1, expected1)
	}

	// Verify sym2
	m2, ok := masks["sym2"]
	if !ok {
		t.Errorf("sym2 missing from parsed masks")
	}
	expected2 := asmSignMask{false, true, true, true}
	if m2 != expected2 {
		t.Errorf("sym2 mismatch: got %v, want %v", m2, expected2)
	}
}
