// lean_rom_consistency_test.go — Cross-check test: Lean-derived ROMs vs asm constants.
//
// TestSIMDConstantsMatchROM loads roms/octonion_signs.hex, extracts the
// quaternion sub-table (4×4 ℍ submatrix at indices 0..3), and compares against
// the sign masks in emulator/qmath_amd64.s.
//
// Three outcomes are possible (per peer-review-005 §4.2 / M0.3 spec):
//
//  1. Match — the asm masks are Lean-correct. Issue #6 (T3) cross-check passes.
//  2. Mismatch in values — real drift detected. Test fails loudly; does NOT fix asm.
//     A finding is written to reviews/finding-001-asm-lean-drift.md if it does not
//     already exist. Architecture instance decides remediation.
//  3. Mismatch in label/naming only — values identical after relabeling. Test passes;
//     documents the relabel proposal.
//
// The test also verifies the sign masks agree with the scalar implementation in
// qmath_scalar.go (which is the ground truth for the existing kernels).
//
// Issue: https://github.com/JamesPagetButler/qbp-compute-unit/issues/7
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

// The sign masks as defined in emulator/qmath_amd64.s (lines 8-34).
// These are the EXISTING hand-derived constants.
// Each mask applies to one basis-component term of the Hamilton product.
//
// Layout in memory (VBROADCASTSD + VXORPD path):
//
//	y_sign_x: [W:-, X:+, Y:-, Z:+]  → applied to a1 (e1/i) × shuffled-b
//	y_sign_y: [W:-, X:+, Y:+, Z:-]  → applied to a2 (e2/j) × shuffled-b
//	y_sign_z: [W:-, X:-, Y:+, Z:+]  → applied to a3 (e3/k) × shuffled-b
var (
	// asmSignX is the y_sign_x mask from qmath_amd64.s.
	asmSignX = asmSignMask{true, false, true, false} // W:-, X:+, Y:-, Z:+

	// asmSignY is the y_sign_y mask from qmath_amd64.s.
	asmSignY = asmSignMask{true, false, false, true} // W:-, X:+, Y:+, Z:-

	// asmSignZ is the y_sign_z mask from qmath_amd64.s.
	asmSignZ = asmSignMask{true, true, false, false} // W:-, X:-, Y:+, Z:+
)

// TestSIMDConstantsMatchROM is the M0.3 cross-check gate.
// It verifies that the hand-derived asm sign masks agree with the ROM-derived signs.
func TestSIMDConstantsMatchROM(t *testing.T) {
	// Find the roms/ directory relative to the test file's location.
	romsDir := findRomsDir(t)

	// Load octonion_signs.hex and extract the ℍ (quaternion) sub-table.
	octSignsPath := filepath.Join(romsDir, "octonion_signs.hex")
	octSigns, err := loadSignsHex(octSignsPath)
	if err != nil {
		t.Fatalf("loadSignsHex(%s): %v\n"+
			"  Run 'make sign-roms' to generate ROM files before running this test.", octSignsPath, err)
	}
	if len(octSigns) != 49 {
		t.Fatalf("octonion_signs.hex: expected 49 entries, got %d", len(octSigns))
	}

	// Extract the 3×3 quaternion sub-table from the octonion sign table.
	// Octonion signs: i,j in 1..7, flat index = (i-1)*7 + (j-1).
	// Quaternion sub-table: i,j in 1..3 (e1=i, e2=j, e3=k).
	//
	// sign16[i][j] = +1 if positive, -1 if negative.
	// octSigns[idx] = 1 if positive, 0 if negative.
	// sign16[i][j] for i,j in 1..3:
	quatSub := [4][4]int8{} // sign16 for i,j in 0..3
	// e0 row/col: always +1
	for k := 0; k < 4; k++ {
		quatSub[0][k] = +1
		quatSub[k][0] = +1
	}
	// e_k * e_k = -1 for k>0
	for k := 1; k < 4; k++ {
		quatSub[k][k] = -1
	}
	// Fill i,j in 1..3 from octonion sub-table
	for i := 1; i <= 3; i++ {
		for j := 1; j <= 3; j++ {
			if i == j {
				continue // already -1 from above
			}
			idx := (i-1)*7 + (j - 1)
			if octSigns[idx] == 1 {
				quatSub[i][j] = +1
			} else {
				quatSub[i][j] = -1
			}
		}
	}

	// Derive the three ROM sign masks from quatSub.
	// sign_x[r] = quatSub[1][r^1] (sign for e1 term, lane r)
	// sign_y[r] = quatSub[2][r^2] (sign for e2 term, lane r)
	// sign_z[r] = quatSub[3][r^3] (sign for e3 term, lane r)
	romSignX := signTableToMask(quatSub, 1)
	romSignY := signTableToMask(quatSub, 2)
	romSignZ := signTableToMask(quatSub, 3)

	// Compare ROM-derived masks against asm hand-derived masks.
	driftX := maskDiff(asmSignX, romSignX)
	driftY := maskDiff(asmSignY, romSignY)
	driftZ := maskDiff(asmSignZ, romSignZ)

	if len(driftX)+len(driftY)+len(driftZ) == 0 {
		t.Logf("TestSIMDConstantsMatchROM: PASS — all asm sign masks match ROM-derived constants")
		t.Logf("  y_sign_x: %v (ROM %v)", formatMask(asmSignX), formatMask(romSignX))
		t.Logf("  y_sign_y: %v (ROM %v)", formatMask(asmSignY), formatMask(romSignY))
		t.Logf("  y_sign_z: %v (ROM %v)", formatMask(asmSignZ), formatMask(romSignZ))
		// Also verify against scalar implementation.
		if err := verifyMasksAgainstScalar(quatSub); err != nil {
			t.Errorf("sign table vs scalar cross-check: %v", err)
		} else {
			t.Logf("  scalar cross-check: PASS")
		}
		return
	}

	// Mismatch detected. Emit loud diagnostic.
	var diag strings.Builder
	diag.WriteString("\n=== TestSIMDConstantsMatchROM: DRIFT DETECTED ===\n")
	diag.WriteString("The hand-derived asm sign masks in emulator/qmath_amd64.s\n")
	diag.WriteString("do NOT match the ROM-derived quaternion sub-table.\n\n")
	diag.WriteString("DO NOT MODIFY emulator/qmath_amd64.s — this test reports drift only.\n")
	diag.WriteString("The architecture instance will decide remediation (M0.2).\n\n")

	if len(driftX) > 0 {
		fmt.Fprintf(&diag, "y_sign_x drift:\n")
		for _, d := range driftX {
			fmt.Fprintf(&diag, "  %s\n", d)
		}
	}
	if len(driftY) > 0 {
		fmt.Fprintf(&diag, "y_sign_y drift:\n")
		for _, d := range driftY {
			fmt.Fprintf(&diag, "  %s\n", d)
		}
	}
	if len(driftZ) > 0 {
		fmt.Fprintf(&diag, "y_sign_z drift:\n")
		for _, d := range driftZ {
			fmt.Fprintf(&diag, "  %s\n", d)
		}
	}

	diag.WriteString("\nASM masks (from qmath_amd64.s):\n")
	fmt.Fprintf(&diag, "  y_sign_x: %v\n", formatMask(asmSignX))
	fmt.Fprintf(&diag, "  y_sign_y: %v\n", formatMask(asmSignY))
	fmt.Fprintf(&diag, "  y_sign_z: %v\n", formatMask(asmSignZ))
	diag.WriteString("\nROM-derived masks (from octonion_signs.hex / Sedenion.lean):\n")
	fmt.Fprintf(&diag, "  rom_sign_x: %v\n", formatMask(romSignX))
	fmt.Fprintf(&diag, "  rom_sign_y: %v\n", formatMask(romSignY))
	fmt.Fprintf(&diag, "  rom_sign_z: %v\n", formatMask(romSignZ))

	diag.WriteString("\nQuaternion sign sub-table from ROM:\n")
	names := []string{"e0/1", "e1/i", "e2/j", "e3/k"}
	for i := 0; i < 4; i++ {
		fmt.Fprintf(&diag, "  e%d(%s): ", i, names[i])
		for j := 0; j < 4; j++ {
			s := quatSub[i][j]
			idx := i ^ j
			fmt.Fprintf(&diag, " e%d*e%d=%+de%d", i, j, s, idx)
		}
		diag.WriteString("\n")
	}

	// Write finding document if it does not exist.
	findingPath := writeFindingDoc(t, diag.String(), quatSub, romsDir)
	if findingPath != "" {
		fmt.Fprintf(&diag, "\nFinding written to: %s\n", findingPath)
	}

	t.Fatal(diag.String())
}

// signTableToMask converts a quatSub row k to an asmSignMask.
// The shuffle for component k places b_{r^k} at lane r, so:
//
//	mask[r] = true (negative) iff quatSub[k][r^k] < 0
func signTableToMask(quatSub [4][4]int8, k int) asmSignMask {
	var m asmSignMask
	for r := 0; r < 4; r++ {
		m[r] = quatSub[k][r^k] < 0
	}
	return m
}

// maskDiff returns a list of lane-level discrepancies between asm and rom masks.
func maskDiff(asm, rom asmSignMask) []string {
	laneNames := []string{"W", "X", "Y", "Z"}
	var diffs []string
	for r := 0; r < 4; r++ {
		if asm[r] != rom[r] {
			asmSign := "+"
			if asm[r] {
				asmSign = "-"
			}
			romSign := "+"
			if rom[r] {
				romSign = "-"
			}
			diffs = append(diffs, fmt.Sprintf("lane %s: asm=%s, rom=%s", laneNames[r], asmSign, romSign))
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

// verifyMasksAgainstScalar checks that the ROM-derived sign table is consistent
// with the scalar Hamilton product in qmath_scalar.go.
//
// From qmath_scalar.go:
//
//	w = a0*b0 - a1*b1 - a2*b2 - a3*b3
//	x = a0*b1 + a1*b0 + a2*b3 - a3*b2
//	y = a0*b2 - a1*b3 + a2*b0 + a3*b1
//	z = a0*b3 + a1*b2 - a2*b1 + a3*b0
func verifyMasksAgainstScalar(quatSub [4][4]int8) error {
	// The scalar gives us sign and index for each a_k * b_? contribution.
	// sign table: sign16[k][j] * e_{k^j} = contribution of e_k * e_j.
	// We can reconstruct the scalar terms from sign16.
	type scalTerm struct {
		aIdx, bIdx int
		sign       int8
		outIdx     int
	}
	// Build all 4×4 terms
	var terms []scalTerm
	for k := 0; k < 4; k++ {
		for j := 0; j < 4; j++ {
			terms = append(terms, scalTerm{
				aIdx:   k,
				bIdx:   j,
				sign:   quatSub[k][j],
				outIdx: k ^ j,
			})
		}
	}

	// Reference scalar from qmath_scalar.go
	refTerms := map[[2]int]int8{
		{0, 0}: +1, {1, 1}: -1, {2, 2}: -1, {3, 3}: -1, // w (idx 0)
		{0, 1}: +1, {1, 0}: +1, {2, 3}: +1, {3, 2}: -1, // x (idx 1)
		{0, 2}: +1, {1, 3}: -1, {2, 0}: +1, {3, 1}: +1, // y (idx 2)
		{0, 3}: +1, {1, 2}: +1, {2, 1}: -1, {3, 0}: +1, // z (idx 3)
	}

	for _, term := range terms {
		wantSign, ok := refTerms[[2]int{term.aIdx, term.bIdx}]
		if !ok {
			return fmt.Errorf("missing scalar term a%d*b%d", term.aIdx, term.bIdx)
		}
		if term.sign != wantSign {
			return fmt.Errorf("a%d*b%d: ROM sign=%+d, scalar sign=%+d (idx=%d)",
				term.aIdx, term.bIdx, term.sign, wantSign, term.outIdx)
		}
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// findRomsDir returns the path to the roms/ directory relative to the test file.
func findRomsDir(t *testing.T) string {
	t.Helper()
	// The emulator package lives at <repo>/emulator/; roms/ is at <repo>/roms/.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	emulatorDir := filepath.Dir(filename)
	repoRoot := filepath.Dir(emulatorDir)
	romsDir := filepath.Join(repoRoot, "roms")
	if _, err := os.Stat(romsDir); err != nil {
		t.Fatalf("roms/ directory not found at %s; run 'make sign-roms' first", romsDir)
	}
	return romsDir
}

// loadSignsHex reads a 1-bit hex ROM file and returns a slice of uint8 (0 or 1).
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

// writeFindingDoc writes reviews/finding-001-asm-lean-drift.md if it does not
// already exist. Returns the path written, or "" if already exists.
func writeFindingDoc(t *testing.T, diag string, quatSub [4][4]int8, romsDir string) string {
	t.Helper()
	reviewsDir := filepath.Join(filepath.Dir(romsDir), "reviews")
	if err := os.MkdirAll(reviewsDir, 0o755); err != nil {
		t.Logf("writeFindingDoc: mkdir %s: %v", reviewsDir, err)
		return ""
	}
	path := filepath.Join(reviewsDir, "finding-001-asm-lean-drift.md")
	if _, err := os.Stat(path); err == nil {
		return "" // already exists
	}

	content := fmt.Sprintf(`# Finding 001: ASM / Lean sign-table drift

**Date:** generated by TestSIMDConstantsMatchROM
**Source:** emulator/lean_rom_consistency_test.go
**Issue:** https://github.com/JamesPagetButler/qbp-compute-unit/issues/7

## Summary

The hand-derived sign masks in ` + "`emulator/qmath_amd64.s`" + ` do not match the
ROM-derived quaternion sub-table from ` + "`roms/octonion_signs.hex`" + ` (which is derived
from ` + "`lean/QBP/Sedenion.lean`" + ` via Cayley-Dickson construction).

## Diagnostic output

` + "```\n" + diag + "\n```\n" + `

## Action required

**DO NOT modify** ` + "`emulator/qmath_amd64.s`" + ` or ` + "`emulator/qmath_128_amd64.s`" + `.
These are working kernels; tests pass against the scalar reference.

Architecture instance to decide:
1. Is the Lean source wrong? → Fix ` + "`lean/QBP/Sedenion.lean`" + ` and regenerate.
2. Is the asm wrong? → Fix asm in M0.2, run cosim.
3. Is this a labelling mismatch only? → Document and propose relabel.

## Quaternion sign sub-table (ROM-derived)

| | e0 | e1/i | e2/j | e3/k |
|---|---|---|---|---|
%s

*This document was auto-generated and should be reviewed before committing.*
`,
		buildSignTableMd(quatSub))

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Logf("writeFindingDoc: write %s: %v", path, err)
		return ""
	}
	return path
}

func buildSignTableMd(quatSub [4][4]int8) string {
	names := []string{"e0", "e1/i", "e2/j", "e3/k"}
	var b strings.Builder
	for i := 0; i < 4; i++ {
		fmt.Fprintf(&b, "| **%s** |", names[i])
		for j := 0; j < 4; j++ {
			s := quatSub[i][j]
			fmt.Fprintf(&b, " %+d |", s)
		}
		b.WriteString("\n")
	}
	return b.String()
}
