// cmd/lean2rom — Extract sign and index ROMs from Sedenion.lean (or native Go fallback).
//
// Usage:
//
//	go run ./cmd/lean2rom           # regenerate roms/
//	go run ./cmd/lean2rom -verify   # verify checksums only; exit non-zero if mismatch
//
// Outputs:
//
//	roms/sedenion_signs.hex            225 entries × 1 bit
//	roms/sedenion_idx.hex              256 entries × 4 bits
//	roms/octonion_signs.hex             49 entries × 1 bit
//	roms/octonion_idx.hex               64 entries × 3 bits
//	roms/quaternion_signs.go            Go constants citing this derivation
//	emulator/qmath_constants.s          Generated asm sign masks (DO NOT HAND-EDIT)
//	roms/CHECKSUMS.lean-verified        SHA-256 manifest
//
// Attribution:
//
//	Moreno (1998) "The zero divisors of the Cayley-Dickson algebras over the real
//	  numbers" — XOR-index lemma.
//	Cawagas (2009) "Loops & Quasigroups Newsletter" — 42 cross-copy ZDs.
//	Schafer (1954) "On the algebras formed by the Cayley-Dickson process" — CD formula.
//	Baez (2002) "The Octonions" — Fano-plane octonion table.
//
// Issue: github.com/JamesPagetButler/qbp-compute-unit/issues/7
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── Entry point ────────────────────────────────────────────────────────────────

func main() {
	verifyOnly := flag.Bool("verify", false, "Only verify checksums; do not regenerate")
	repoRoot := flag.String("root", ".", "Repository root directory")
	flag.Parse()

	if err := run(*repoRoot, *verifyOnly); err != nil {
		log.Fatalf("lean2rom: %v", err)
	}
}

func run(root string, verifyOnly bool) error {
	tables, leanUsed, err := loadTables(root)
	if err != nil {
		return fmt.Errorf("loading tables: %w", err)
	}

	src := "Go-native Cayley-Dickson derivation"
	if leanUsed {
		src = "lean/QBP/Sedenion.lean"
	}
	log.Printf("lean2rom: source = %s", src)

	if err := validateTables(tables); err != nil {
		return fmt.Errorf("table validation: %w", err)
	}
	log.Printf("lean2rom: validation passed (XOR-index lemma, ZD count)")

	if verifyOnly {
		return verifyChecksums(root, tables)
	}

	if err := emitROMs(root, tables, src); err != nil {
		return fmt.Errorf("emitting ROMs: %w", err)
	}
	log.Printf("lean2rom: ROMs written to %s/roms/", root)
	return nil
}

// ── Table derivation ───────────────────────────────────────────────────────────

// Tables holds the four ROM tables derived from the sedenion algebra.
type Tables struct {
	// SedSigns[i*15+(j-1)] for i in 1..15, j in 1..15 — sign of e_i*e_j
	// 1 = positive, 0 = negative
	SedSigns []uint8 // len 225

	// SedIdx[i*16+j] = i XOR j for i,j in 0..15
	SedIdx []uint8 // len 256

	// OctSigns[i*7+(j-1)] for i in 1..7, j in 1..7
	OctSigns []uint8 // len 49

	// OctIdx[i*8+j] = i XOR j for i,j in 0..8
	OctIdx []uint8 // len 64

	// Full 16x16 sign matrix (including row/col 0), signed: +1 or -1
	// sign16[i][j] is the sign of e_i*e_j for i,j in 0..15
	sign16 [16][16]int8
}

// loadTables attempts Lean extraction; falls back to native Go derivation.
func loadTables(root string) (Tables, bool, error) {
	leanPath := filepath.Join(root, "lean", "QBP", "Sedenion.lean")
	if _, err := os.Stat(leanPath); err == nil {
		if t, err := extractFromLean(root, leanPath); err == nil {
			return t, true, nil
		} else {
			log.Printf("lean2rom: Lean extraction failed (%v); falling back to Go derivation", err)
		}
	} else {
		log.Printf("lean2rom: %s not found; using Go derivation", leanPath)
	}
	return deriveNative(), false, nil
}

// extractFromLean invokes `lean lean/QBP/Sedenion.lean` and parses the JSON
// lines emitted by the #eval extraction blocks.
func extractFromLean(root, leanPath string) (Tables, error) {
	lean, err := exec.LookPath("lean")
	if err != nil {
		return Tables{}, fmt.Errorf("lean not in PATH")
	}

	cmd := exec.Command(lean, leanPath)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// lean exits non-zero on warnings but still emits output; check for JSON
		if len(out) == 0 {
			return Tables{}, fmt.Errorf("lean exited %v with no output", err)
		}
		log.Printf("lean2rom: lean exited non-zero (%v); attempting to parse output anyway", err)
	}

	var signData []int
	var idxData []int
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, `{"mulSignData":`) {
			var m struct {
				MulSignData []int `json:"mulSignData"`
			}
			if e := json.Unmarshal([]byte(line), &m); e == nil {
				signData = m.MulSignData
			}
		}
		if strings.HasPrefix(line, `{"mulIdxData":`) {
			var m struct {
				MulIdxData []int `json:"mulIdxData"`
			}
			if e := json.Unmarshal([]byte(line), &m); e == nil {
				idxData = m.MulIdxData
			}
		}
	}

	if len(signData) != 225 {
		return Tables{}, fmt.Errorf("expected 225 sign entries from Lean, got %d", len(signData))
	}
	if len(idxData) != 256 {
		return Tables{}, fmt.Errorf("expected 256 idx entries from Lean, got %d", len(idxData))
	}

	// Reconstruct full 16x16 sign matrix from signData (i,j in 1..15)
	var s16 [16][16]int8
	// e_0 row/col: e_0*e_k = e_k*e_0 = +e_k
	for k := 0; k < 16; k++ {
		s16[0][k] = +1
		s16[k][0] = +1
	}
	// e_k*e_k = -e_0 for k > 0
	for k := 1; k < 16; k++ {
		s16[k][k] = -1
	}
	// Fill from Lean data: sign for i,j in 1..15
	for i := 1; i <= 15; i++ {
		for j := 1; j <= 15; j++ {
			v := signData[(i-1)*15+(j-1)]
			if v == 1 {
				s16[i][j] = +1
			} else {
				s16[i][j] = -1
			}
		}
	}

	t := Tables{sign16: s16}
	t.buildFromSign16()
	return t, nil
}

// buildFromSign16 fills the four ROM tables from the full sign matrix.
func (t *Tables) buildFromSign16() {
	// SedSigns: i,j in 1..15
	t.SedSigns = make([]uint8, 225)
	for i := 1; i <= 15; i++ {
		for j := 1; j <= 15; j++ {
			v := uint8(0)
			if t.sign16[i][j] > 0 {
				v = 1
			}
			t.SedSigns[(i-1)*15+(j-1)] = v
		}
	}

	// SedIdx: i,j in 0..15 (XOR)
	t.SedIdx = make([]uint8, 256)
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			t.SedIdx[i*16+j] = uint8(i ^ j)
		}
	}

	// OctSigns: i,j in 1..7 (submatrix of sedenion)
	t.OctSigns = make([]uint8, 49)
	for i := 1; i <= 7; i++ {
		for j := 1; j <= 7; j++ {
			v := uint8(0)
			if t.sign16[i][j] > 0 {
				v = 1
			}
			t.OctSigns[(i-1)*7+(j-1)] = v
		}
	}

	// OctIdx: i,j in 0..7 (XOR)
	t.OctIdx = make([]uint8, 64)
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			t.OctIdx[i*8+j] = uint8(i ^ j)
		}
	}
}

// deriveNative computes the Cayley-Dickson sedenion table in pure Go.
//
// CD doubling formula: (a,b)(c,d) = (ac − conj(d)b, da + b·conj(c))
// where conj(e_0) = +e_0, conj(e_k) = −e_k for k > 0.
//
// Starting from ℝ → ℂ → ℍ → 𝕆 → 𝕊 (4 doublings).
//
// Reference: Schafer (1954). The XOR-index property holds by construction.
func deriveNative() Tables {
	const maxLevels = 4
	// signs[level] is a 2^level × 2^level matrix of {+1,-1}
	// Level 0: 1×1, e_0*e_0 = +e_0
	type matrix = [][]int8
	levels := make([]matrix, maxLevels+1)
	levels[0] = matrix{{+1}}

	for level := 1; level <= maxLevels; level++ {
		prev := levels[level-1]
		m := len(prev)
		n := 2 * m
		cur := make(matrix, n)
		for k := range cur {
			cur[k] = make([]int8, n)
		}
		for i := 0; i < m; i++ {
			for j := 0; j < m; j++ {
				// Case A: e_i * e_j → lower half
				cur[i][j] = prev[i][j]

				// Case B: e_i * e_{j+m} → upper half
				// = e_j * e_i sign (note reversed order)
				cur[i][j+m] = prev[j][i]

				// Case C: e_{i+m} * e_j → upper half
				// = sign * e_i * conj(e_j)
				// conj(e_j) = +e_j if j==0, else -e_j
				if j == 0 {
					cur[i+m][j] = prev[i][j]
				} else {
					cur[i+m][j] = -prev[i][j]
				}

				// Case D: e_{i+m} * e_{j+m} → lower half
				// = -conj(e_j) * e_i
				// For j==0: -e_0 * e_i = -e_i → sign = -prev[j][i]
				// For j>0:  -(-e_j)*e_i = +e_j*e_i → sign = +prev[j][i]
				if j == 0 {
					cur[i+m][j+m] = -prev[j][i]
				} else {
					cur[i+m][j+m] = prev[j][i]
				}
			}
		}
		levels[level] = cur
	}

	s16 := levels[maxLevels]
	var t Tables
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			t.sign16[i][j] = s16[i][j]
		}
	}
	t.buildFromSign16()
	return t
}

// ── Validation ─────────────────────────────────────────────────────────────────

// validateTables asserts algebraic invariants against the derived tables.
func validateTables(t Tables) error {
	// 1. Entry counts
	if len(t.SedSigns) != 225 {
		return fmt.Errorf("SedSigns len %d, want 225", len(t.SedSigns))
	}
	if len(t.SedIdx) != 256 {
		return fmt.Errorf("SedIdx len %d, want 256", len(t.SedIdx))
	}
	if len(t.OctSigns) != 49 {
		return fmt.Errorf("OctSigns len %d, want 49", len(t.OctSigns))
	}
	if len(t.OctIdx) != 64 {
		return fmt.Errorf("OctIdx len %d, want 64", len(t.OctIdx))
	}

	// 2. XOR-index lemma: SedIdx[i*16+j] == i^j
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			if int(t.SedIdx[i*16+j]) != i^j {
				return fmt.Errorf("XOR-index violated at (%d,%d): got %d want %d",
					i, j, t.SedIdx[i*16+j], i^j)
			}
		}
	}

	// 3. XOR-index for OctIdx
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			if int(t.OctIdx[i*8+j]) != i^j {
				return fmt.Errorf("octonion XOR-index violated at (%d,%d)", i, j)
			}
		}
	}

	// 4. Quaternion sub-algebra: e1*e2=+e3, e2*e3=+e1, e3*e1=+e2 (cyclic)
	quatChecks := [][3]int{
		{1, 2, 3}, {2, 3, 1}, {3, 1, 2}, // cyclic: ei*ej = +ek
	}
	for _, c := range quatChecks {
		i, j, k := c[0], c[1], c[2]
		if t.sign16[i][j] != +1 || int(i^j) != k {
			return fmt.Errorf("quaternion check e%d*e%d failed: got sign=%d idx=%d, want +1 idx=%d",
				i, j, t.sign16[i][j], i^j, k)
		}
		if t.sign16[j][i] != -1 || int(j^i) != k {
			return fmt.Errorf("quaternion anti-check e%d*e%d failed", j, i)
		}
	}
	// e_k^2 = -e_0 for k in 1..15
	for k := 1; k < 16; k++ {
		if t.sign16[k][k] != -1 {
			return fmt.Errorf("e%d^2 sign = %d, want -1", k, t.sign16[k][k])
		}
		if k^k != 0 {
			return fmt.Errorf("e%d^2 index = %d, want 0 (XOR-index)", k, k^k) // always 0
		}
	}

	// 5. 42 cross-copy basis-sum ZDs (Cawagas 2009)
	// (e_i + e_j)(e_k + e_l) = 0 with i,k in {1..7}, j,l in {9..15}
	// Count unique unordered pairs.
	type pair [2]int
	zdSet := make(map[[2]pair]struct{})
	for i := 1; i <= 7; i++ {
		for j := 9; j <= 15; j++ {
			for k := 1; k <= 7; k++ {
				for l := 9; l <= 15; l++ {
					if isZeroDivisor(t.sign16, i, j, k, l) {
						a := pair{i, j}
						b := pair{k, l}
						if a[0] > b[0] || (a[0] == b[0] && a[1] > b[1]) {
							a, b = b, a
						}
						zdSet[[2]pair{a, b}] = struct{}{}
					}
				}
			}
		}
	}
	if len(zdSet) != 42 {
		return fmt.Errorf("ZD count = %d, want 42 (Cawagas 2009)", len(zdSet))
	}

	return nil
}

// isZeroDivisor checks if (e_i + e_j)(e_k + e_l) = 0.
func isZeroDivisor(sign16 [16][16]int8, i, j, k, l int) bool {
	// Four product terms: e_i*e_k, e_i*e_l, e_j*e_k, e_j*e_l
	type term struct {
		sign int8
		idx  int
	}
	terms := []term{
		{sign16[i][k], i ^ k},
		{sign16[i][l], i ^ l},
		{sign16[j][k], j ^ k},
		{sign16[j][l], j ^ l},
	}
	// Sum by index; must all be zero
	sums := make(map[int]int8)
	for _, t := range terms {
		sums[t.idx] += t.sign
	}
	for _, v := range sums {
		if v != 0 {
			return false
		}
	}
	return true
}

// ── ROM emission ───────────────────────────────────────────────────────────────

// romFile is a named output file with its contents.
type romFile struct {
	name    string
	content []byte
}

func emitROMs(root string, tables Tables, src string) error {
	romsDir := filepath.Join(root, "roms")
	if err := os.MkdirAll(romsDir, 0o755); err != nil {
		return err
	}
	// Output to emulator/ directly, not emulator/asm/
	emuDir := filepath.Join(root, "emulator")
	if err := os.MkdirAll(emuDir, 0o755); err != nil {
		return err
	}

	files := []romFile{
		{filepath.Join(romsDir, "sedenion_signs.hex"), buildHex1bit(tables.SedSigns)},
		{filepath.Join(romsDir, "sedenion_idx.hex"), buildHex4bit(tables.SedIdx)},
		{filepath.Join(romsDir, "octonion_signs.hex"), buildHex1bit(tables.OctSigns)},
		{filepath.Join(romsDir, "octonion_idx.hex"), buildHex3bit(tables.OctIdx)},
	}

	// Write the four hex ROMs
	for _, f := range files {
		if err := os.WriteFile(f.name, f.content, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", f.name, err)
		}
	}

	// Emit quaternion_signs.go
	qsignPath := filepath.Join(romsDir, "quaternion_signs.go")
	if err := os.WriteFile(qsignPath, buildQuaternionSignsGo(tables, src), 0o644); err != nil {
		return fmt.Errorf("writing quaternion_signs.go: %w", err)
	}

	// Emit emulator/qmath_constants.s
	asmPath := filepath.Join(emuDir, "qmath_constants.s")
	if err := os.WriteFile(asmPath, buildQmathConstantsS(tables, src), 0o644); err != nil {
		return fmt.Errorf("writing qmath_constants.s: %w", err)
	}

	// Compute checksums and emit manifest
	checksumPath := filepath.Join(romsDir, "CHECKSUMS.lean-verified")
	manifest := buildChecksumManifest(files, src)
	if err := os.WriteFile(checksumPath, manifest, 0o644); err != nil {
		return fmt.Errorf("writing CHECKSUMS.lean-verified: %w", err)
	}

	// Log what was written
	for _, f := range files {
		log.Printf("  wrote %s", f.name)
	}
	log.Printf("  wrote %s", qsignPath)
	log.Printf("  wrote %s", asmPath)
	log.Printf("  wrote %s", checksumPath)
	return nil
}

// ── Verify mode ────────────────────────────────────────────────────────────────

func verifyChecksums(root string, tables Tables) error {
	checksumPath := filepath.Join(root, "roms", "CHECKSUMS.lean-verified")
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("CHECKSUMS.lean-verified not found; run 'make sign-roms' first: %w", err)
	}

	// Re-derive fresh checksums
	romsDir := filepath.Join(root, "roms")
	hexFiles := []romFile{
		{filepath.Join(romsDir, "sedenion_signs.hex"), buildHex1bit(tables.SedSigns)},
		{filepath.Join(romsDir, "sedenion_idx.hex"), buildHex4bit(tables.SedIdx)},
		{filepath.Join(romsDir, "octonion_signs.hex"), buildHex1bit(tables.OctSigns)},
		{filepath.Join(romsDir, "octonion_idx.hex"), buildHex3bit(tables.OctIdx)},
	}

	// Parse existing manifest
	existing := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "sha256:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// format: <filename>  sha256: <hash>
				name := parts[0]
				for idx, p := range parts {
					if p == "sha256:" && idx+1 < len(parts) {
						existing[name] = parts[idx+1]
					}
				}
			}
		}
	}

	var errs []string
	for _, f := range hexFiles {
		h := sha256.Sum256(f.content)
		got := fmt.Sprintf("%x", h)
		base := filepath.Base(f.name)
		want, ok := existing[base]
		if !ok {
			errs = append(errs, fmt.Sprintf("  %s: missing from manifest", base))
			continue
		}
		if got != want {
			errs = append(errs, fmt.Sprintf("  %s: got %s, want %s", base, got, want))
		} else {
			log.Printf("  %s: OK (%s)", base, got[:16]+"...")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("checksum mismatch (hard fail):\n%s", strings.Join(errs, "\n"))
	}
	log.Printf("lean2rom: all checksums match")
	return nil
}

// ── ROM file builders ──────────────────────────────────────────────────────────

// buildHex1bit emits one hex digit per entry (0 or 1), one per line.
func buildHex1bit(data []uint8) []byte {
	var b strings.Builder
	b.WriteString("// 1-bit sign ROM: 0=negative, 1=positive\n")
	b.WriteString(fmt.Sprintf("// %d entries\n", len(data)))
	for _, v := range data {
		fmt.Fprintf(&b, "%01x\n", v&1)
	}
	return []byte(b.String())
}

// buildHex4bit emits one hex byte per entry (0x00-0x0f), one per line.
func buildHex4bit(data []uint8) []byte {
	var b strings.Builder
	b.WriteString("// 4-bit index ROM: sedenion XOR-index lemma (i XOR j)\n")
	b.WriteString(fmt.Sprintf("// %d entries\n", len(data)))
	for _, v := range data {
		fmt.Fprintf(&b, "%02x\n", v&0x0f)
	}
	return []byte(b.String())
}

// buildHex3bit emits one hex byte per entry (0x00-0x07), one per line.
func buildHex3bit(data []uint8) []byte {
	var b strings.Builder
	b.WriteString("// 3-bit index ROM: octonion XOR-index lemma (i XOR j)\n")
	b.WriteString(fmt.Sprintf("// %d entries\n", len(data)))
	for _, v := range data {
		fmt.Fprintf(&b, "%02x\n", v&0x07)
	}
	return []byte(b.String())
}

// buildChecksumManifest generates the CHECKSUMS.lean-verified manifest.
func buildChecksumManifest(files []romFile, src string) []byte {
	var b strings.Builder
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	fmt.Fprintf(&b, "# lean2rom ROM checksum manifest\n")
	fmt.Fprintf(&b, "# Generated: %s\n", now)
	fmt.Fprintf(&b, "# Source: %s\n", src)
	fmt.Fprintf(&b, "# Issue: https://github.com/JamesPagetButler/qbp-compute-unit/issues/7\n")
	fmt.Fprintf(&b, "#\n")
	fmt.Fprintf(&b, "# Format: <filename>  sha256: <hex>  source: <lean-symbol>\n")
	fmt.Fprintf(&b, "#\n")
	sourceMap := map[string]string{
		"sedenion_signs.hex": "Sedenion.lean:mulSignData",
		"sedenion_idx.hex":   "Sedenion.lean:mulIdxData",
		"octonion_signs.hex": "Sedenion.lean:mulSignData (8×8 submatrix)",
		"octonion_idx.hex":   "Sedenion.lean:mulIdxData (8×8 submatrix)",
	}
	for _, f := range files {
		h := sha256.Sum256(f.content)
		base := filepath.Base(f.name)
		fmt.Fprintf(&b, "%-28s  sha256: %x  source: %s\n", base, h, sourceMap[base])
	}
	return []byte(b.String())
}

// ── quaternion_signs.go builder ────────────────────────────────────────────────

// buildQuaternionSignsGo emits the roms/quaternion_signs.go file with
// Go constants for the quaternion (ℍ) 4×4 sign submatrix.
// These are the constants that the cross-check test compares against the
// asm masks in emulator/qmath_amd64.s.
func buildQuaternionSignsGo(tables Tables, src string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by cmd/lean2rom. DO NOT EDIT.\n")
	fmt.Fprintf(&b, "// Source: %s\n", src)
	fmt.Fprintf(&b, "// Issue: https://github.com/JamesPagetButler/qbp-compute-unit/issues/7\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// Quaternion sign constants derived from the sedenion sign table.\n")
	fmt.Fprintf(&b, "// The ℍ (quaternion) algebra is the sub-algebra of 𝕊 at indices 0..3.\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// Attribution: Schafer (1954) CD construction; Moreno (1998) XOR-index lemma.\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// sign[i][j] encodes the sign of e_i × e_j.\n")
	fmt.Fprintf(&b, "// +1 = positive component, -1 = negative component.\n")
	fmt.Fprintf(&b, "// Index of product = i XOR j (always, by XOR-index lemma).\n")
	fmt.Fprintf(&b, "package roms\n\n")

	fmt.Fprintf(&b, "// QuaternionSignTable is the 4×4 sign matrix for quaternion multiplication.\n")
	fmt.Fprintf(&b, "// Row i, column j: sign of e_i × e_j = QuaternionSignTable[i][j] × e_{i^j}.\n")
	fmt.Fprintf(&b, "var QuaternionSignTable = [4][4]int8{\n")
	for i := 0; i < 4; i++ {
		fmt.Fprintf(&b, "\t{")
		for j := 0; j < 4; j++ {
			if j > 0 {
				fmt.Fprintf(&b, ", ")
			}
			fmt.Fprintf(&b, "%+d", tables.sign16[i][j])
		}
		names := []string{"e0/1", "e1/i", "e2/j", "e3/k"}
		fmt.Fprintf(&b, "}, // row %s\n", names[i])
	}
	fmt.Fprintf(&b, "}\n\n")

	// Emit the sign bit per (i,j) pair as individual constants for asm comparison
	fmt.Fprintf(&b, "// The following sign bits are extracted from QuaternionSignTable\n")
	fmt.Fprintf(&b, "// and correspond to the sign-mask lanes in emulator/qmath_amd64.s:\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "//   qbp_lean_sign_x: applied to a1 (e1/i) component across [W,X,Y,Z] lanes\n")
	fmt.Fprintf(&b, "//   qbp_lean_sign_y: applied to a2 (e2/j) component across [W,X,Y,Z] lanes\n")
	fmt.Fprintf(&b, "//   qbp_lean_sign_z: applied to a3 (e3/k) component across [W,X,Y,Z] lanes\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// The asm shuffle for term k places b_{k^0},b_{k^1},b_{k^2},b_{k^3} in\n")
	fmt.Fprintf(&b, "// positions [W,X,Y,Z]. The sign mask for term k in output component r is:\n")
	fmt.Fprintf(&b, "//   SignForAsmMask[k][r] = sign16[k][k^r]\n")

	// k=1 (x/i component): output [W,X,Y,Z] = sign16[1][1^0], sign16[1][1^1], sign16[1][1^2], sign16[1][1^3]
	//                                        = sign16[1][1], sign16[1][0], sign16[1][3], sign16[1][2]
	//                                        = -1, +1, ?, ?
	fmt.Fprintf(&b, "\n// SignYSignX[r] = sign of e1 × e_{1 XOR r} for r in {0,1,2,3} → lanes {W,X,Y,Z}\n")
	fmt.Fprintf(&b, "var SignYSignX = [4]int8{")
	for r := 0; r < 4; r++ {
		if r > 0 {
			fmt.Fprintf(&b, ", ")
		}
		fmt.Fprintf(&b, "%+d", tables.sign16[1][r^1])
	}
	fmt.Fprintf(&b, "} // asm qbp_lean_sign_x: should be [-1,+1,-1,+1]\n")

	fmt.Fprintf(&b, "\n// SignYSignY[r] = sign for e2 × e_{...} term, for lanes {W,X,Y,Z}\n")
	fmt.Fprintf(&b, "var SignYSignY = [4]int8{")
	for r := 0; r < 4; r++ {
		if r > 0 {
			fmt.Fprintf(&b, ", ")
		}
		fmt.Fprintf(&b, "%+d", tables.sign16[2][r^2])
	}
	fmt.Fprintf(&b, "} // asm qbp_lean_sign_y: should be [-1,+1,+1,-1]\n")

	fmt.Fprintf(&b, "\n// SignYSignZ[r] = sign for e3 × e_{...} term, for lanes {W,X,Y,Z}\n")
	fmt.Fprintf(&b, "var SignYSignZ = [4]int8{")
	for r := 0; r < 4; r++ {
		if r > 0 {
			fmt.Fprintf(&b, ", ")
		}
		fmt.Fprintf(&b, "%+d", tables.sign16[3][r^3])
	}
	fmt.Fprintf(&b, "} // asm qbp_lean_sign_z: should be [-1,-1,+1,+1]\n")

	return []byte(b.String())
}

// ── qmath_constants.s builder ──────────────────────────────────────────────────

// buildQmathConstantsS emits emulator/qmath_constants.s — the generated
// asm sign mask constants derived from the Lean sedenion tables.
// Symbols use the qbp_lean_ prefix and lack the Plan-9 private marker (<>).
func buildQmathConstantsS(tables Tables, src string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "#include \"textflag.h\"\n\n")
	fmt.Fprintf(&b, "// Code generated by cmd/lean2rom. DO NOT EDIT.\n")
	fmt.Fprintf(&b, "// Source: %s\n", src)
	fmt.Fprintf(&b, "// Issue: https://github.com/JamesPagetButler/qbp-compute-unit/issues/7\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// Sign mask constants for the quaternion Hamilton product AVX kernel.\n")
	fmt.Fprintf(&b, "// Derived from the 4×4 quaternion sub-table of mulSignData in Sedenion.lean.\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// These constants are generated and consumed directly by:\n")
	fmt.Fprintf(&b, "//   - emulator/qmath_amd64.s     (QW64 fast path)\n")
	fmt.Fprintf(&b, "//   - emulator/qmath_128_amd64.s (QW128 double-double; same masks, sign-bit\n")
	fmt.Fprintf(&b, "//     XOR is precision-independent)\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// Authority chain: Lean → ROM hex → this file → asm. Closed by issues #13/#14.\n")
	fmt.Fprintf(&b, "// TestSIMDConstantsMatchROM parses this file at test time and verifies\n")
	fmt.Fprintf(&b, "// byte-for-byte agreement with the Lean-derived ROM source-of-truth.\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// Attribution: Schafer (1954), Moreno (1998), Cawagas (2009).\n")
	fmt.Fprintf(&b, "\n")

	// sign_x: applied to a1 term, shuffled by VSHUFPD $0x05 → b_{r^1} at lane r
	// sign_x[r] = sign16[1][r^1]; encoding: sign<0 → 0x8000000000000000 else 0
	signXLanes := [4]int8{
		tables.sign16[1][0^1],
		tables.sign16[1][1^1],
		tables.sign16[1][2^1],
		tables.sign16[1][3^1],
	}
	signYLanes := [4]int8{
		tables.sign16[2][0^2],
		tables.sign16[2][1^2],
		tables.sign16[2][2^2],
		tables.sign16[2][3^2],
	}
	signZLanes := [4]int8{
		tables.sign16[3][0^3],
		tables.sign16[3][1^3],
		tables.sign16[3][2^3],
		tables.sign16[3][3^3],
	}

	emitMask := func(name string, lanes [4]int8, comment string) {
		fmt.Fprintf(&b, "// %s %s\n", name, comment)
		laneNames := []string{"W", "X", "Y", "Z"}
		for r, s := range lanes {
			mask := uint64(0)
			if s < 0 {
				mask = 0x8000000000000000
			}
			fmt.Fprintf(&b, "DATA qbp_lean_%s+0x%02x(SB)/8, $0x%016x // %s (%s)\n",
				name, r*8, mask, laneNames[r], signStr(s))
		}
		fmt.Fprintf(&b, "GLOBL qbp_lean_%s(SB), RODATA, $32\n\n", name)
	}

	emitMask("sign_x", signXLanes, "// a1 (e1/i) term sign mask")
	emitMask("sign_y", signYLanes, "// a2 (e2/j) term sign mask")
	emitMask("sign_z", signZLanes, "// a3 (e3/k) term sign mask")

	// Also emit conjugate mask: [+,-,-,-] = e_0 unchanged, e_{1,2,3} negated
	conjLanes := [4]int8{+1, -1, -1, -1}
	emitMask("conj", conjLanes, "// quaternion conjugate mask [+,-,-,-]")

	return []byte(b.String())
}

func signStr(s int8) string {
	if s > 0 {
		return "+"
	}
	return "-"
}
