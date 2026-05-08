// lean_rom_consistency_test.go — Cross-check test: Lean-derived ROMs vs asm constants.
//
// This test parses emulator/qmath_constants.s, extracts the Lean-derived
// sign masks, and compares them against the ground-truth roms/octonion_signs.hex.
//
// Issue: https://github.com/JamesPagetButler/qbp-compute-unit/issues/13
package emulator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// asmSignMask encodes one AVX sign mask: 4 sign bits (true=negative) for lanes W,X,Y,Z.
// Positive = 0x0000000000000000 XOR bit; negative = 0x8000000000000000 XOR bit.
type asmSignMask [4]bool // true = negative lane

// TestSIMDConstantsMatchROM is the authority-chain verification gate.
// It parses the generated assembly constants and verifies they match the ROM source-of-truth.
func TestSIMDConstantsMatchROM(t *testing.T) {
	// 1. Load ground truth from ROMs
	romsDir := findRomsDir(t)
	octSignsPath := filepath.Join(romsDir, "octonion_signs.hex")
	octSigns, err := loadSignsHex(octSignsPath)
	if err != nil {
		t.Fatalf("loadSignsHex(%s): %v\n"+
			"  Run 'make sign-roms' to generate ROM files before running this test.", octSignsPath, err)
	}

	quatSub := deriveQuatSubTable(octSigns)
	romSignX := signTableToMask(quatSub, 1)
	romSignY := signTableToMask(quatSub, 2)
	romSignZ := signTableToMask(quatSub, 3)
	romConj := asmSignMask{false, true, true, true} // [+,-,-,-]

	// 2. Parse constants from generated assembly file
	repoRoot := filepath.Dir(romsDir)
	constantsPath := filepath.Join(repoRoot, "emulator", "qmath_constants.s")
	asmMasks, err := parseAsmConstants(constantsPath)
	if err != nil {
		t.Fatalf("parseAsmConstants(%s): %v", constantsPath, err)
	}

	// 3. Verify symmetry: Check both QW64 and QW128 expectations
	// Note: y_sign_* and y128_sign_* now both point to these same symbols.
	verifySymbol(t, "qbp_lean_sign_x", asmMasks["qbp_lean_sign_x"], romSignX)
	verifySymbol(t, "qbp_lean_sign_y", asmMasks["qbp_lean_sign_y"], romSignY)
	verifySymbol(t, "qbp_lean_sign_z", asmMasks["qbp_lean_sign_z"], romSignZ)
	verifySymbol(t, "qbp_lean_conj", asmMasks["qbp_lean_conj"], romConj)

	// 4. Verify against scalar implementation
	if err := verifyMasksAgainstScalar(quatSub); err != nil {
		t.Errorf("sign table vs scalar cross-check: %v", err)
	}
}

func verifySymbol(t *testing.T, name string, got, want asmSignMask) {
	t.Helper()
	diff := maskDiff(got, want)
	if len(diff) > 0 {
		t.Errorf("drift detected in symbol %s:\n%s\nGot:  %v\nWant: %v",
			name, strings.Join(diff, "\n"), formatMask(got), formatMask(want))
	} else {
		t.Logf("symbol %s: OK (%v)", name, formatMask(got))
	}
}

// deriveQuatSubTable extracts the ℍ sub-table from octonion signs.
func deriveQuatSubTable(octSigns []uint8) [4][4]int8 {
	quatSub := [4][4]int8{}
	for k := 0; k < 4; k++ {
		quatSub[0][k] = +1
		quatSub[k][0] = +1
	}
	for k := 1; k < 4; k++ {
		quatSub[k][k] = -1
	}
	for i := 1; i <= 3; i++ {
		for j := 1; j <= 3; j++ {
			if i == j {
				continue
			}
			idx := (i-1)*7 + (j - 1)
			if octSigns[idx] == 1 {
				quatSub[i][j] = +1
			} else {
				quatSub[i][j] = -1
			}
		}
	}
	return quatSub
}

// parseAsmConstants parses a Plan-9 assembly file and extracts DATA-block sign masks.
func parseAsmConstants(path string) (map[string]asmSignMask, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	masks := make(map[string]asmSignMask)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "DATA") {
			continue
		}

		// Example: DATA qbp_lean_sign_x+0x00(SB)/8, $0x8000000000000000
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		// Parse symbol and offset
		symPart := strings.TrimSpace(strings.TrimPrefix(parts[0], "DATA"))
		symName := ""
		offset := 0
		if idx := strings.Index(symPart, "+0x"); idx != -1 {
			symName = symPart[:idx]
			offsetStr := symPart[idx+3:]
			if parenIdx := strings.Index(offsetStr, "("); parenIdx != -1 {
				offsetStr = offsetStr[:parenIdx]
			}
			fmt.Sscanf(offsetStr, "%x", &offset)
		}

		// Parse value
		valPart := strings.TrimSpace(parts[1])
		if idx := strings.Index(valPart, "//"); idx != -1 {
			valPart = valPart[:idx]
		}
		valPart = strings.TrimSpace(valPart)
		if !strings.HasPrefix(valPart, "$0x") {
			continue
		}
		val, err := strconv.ParseUint(valPart[3:], 16, 64)
		if err != nil {
			continue
		}

		m := masks[symName]
		laneIdx := offset / 8
		if laneIdx < 4 {
			m[laneIdx] = (val == 0x8000000000000000)
			masks[symName] = m
		}
	}
	return masks, scanner.Err()
}

// signTableToMask converts a quatSub row k to an asmSignMask.
func signTableToMask(quatSub [4][4]int8, k int) asmSignMask {
	var m asmSignMask
	for r := 0; r < 4; r++ {
		m[r] = quatSub[k][r^k] < 0
	}
	return m
}

// maskDiff returns a list of lane-level discrepancies between got and want masks.
func maskDiff(got, want asmSignMask) []string {
	laneNames := []string{"W", "X", "Y", "Z"}
	var diffs []string
	for r := 0; r < 4; r++ {
		if got[r] != want[r] {
			gotSign := "+"
			if got[r] {
				gotSign = "-"
			}
			wantSign := "+"
			if want[r] {
				wantSign = "-"
			}
			diffs = append(diffs, fmt.Sprintf("lane %s: got=%s, want=%s", laneNames[r], gotSign, wantSign))
		}
	}
	return diffs
}

// formatMask formats an asmSignMask as [W:±, X:±, Y:±, Z:±].
func formatMask(m asmSignMask) string {
	laneNames := []string{"W", "X", "Y", "Z"}
	parts := make([]string, 4)
	for r, neg := range m {
		sign := "+"
		if neg {
			sign = "-"
		}
		parts[r] = laneNames[r] + ":" + sign
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// verifyMasksAgainstScalar checks consistency with qmath_scalar.go logic.
func verifyMasksAgainstScalar(quatSub [4][4]int8) error {
	refTerms := map[[2]int]int8{
		{0, 0}: +1, {1, 1}: -1, {2, 2}: -1, {3, 3}: -1,
		{0, 1}: +1, {1, 0}: +1, {2, 3}: +1, {3, 2}: -1,
		{0, 2}: +1, {1, 3}: -1, {2, 0}: +1, {3, 1}: +1,
		{0, 3}: +1, {1, 2}: +1, {2, 1}: -1, {3, 0}: +1,
	}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if quatSub[i][j] != refTerms[[2]int{i, j}] {
				return fmt.Errorf("a%d*b%d: ROM sign=%+d, scalar sign=%+d",
					i, j, quatSub[i][j], refTerms[[2]int{i, j}])
			}
		}
	}
	return nil
}

// findRomsDir returns the path to the roms/ directory.
func findRomsDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	emulatorDir := filepath.Dir(filename)
	repoRoot := filepath.Dir(emulatorDir)
	romsDir := filepath.Join(repoRoot, "roms")
	if _, err := os.Stat(romsDir); err != nil {
		t.Fatalf("roms/ directory not found at %s", romsDir)
	}
	return romsDir
}

// loadSignsHex reads a 1-bit hex ROM file.
func loadSignsHex(path string) ([]uint8, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var signs []uint8
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		v, err := strconv.ParseUint(line, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", line, err)
		}
		signs = append(signs, uint8(v))
	}
	return signs, scanner.Err()
}
