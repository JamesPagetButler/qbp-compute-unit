// fano_proof_tie_test.go — the prove-before-bake drift gate (issue #59).
//
// Closes Gap B (live-test seq=516, notary V&V finding): the octonion ROMs
// (roms/octonion_*.hex) previously derived only from lean/QBP/Sedenion.lean,
// which is *data, not proof* (def arrays + #eval, zero theorems). This gate
// ties BOTH the ROMs and the emulator fanoLUT to the KERNEL-PROVEN table
// QBP.Foundations.FanoOrientationF3.fanoTableF4 — proven forced by the
// Cayley–Dickson product via fanoTableF4_eq_cayleyDickson (kernel `decide`,
// FanoOrientationF3.lean:150).
//
// Post-#59 chain, every link drift-tested:
//
//	CD product (kernel decide) → fanoTableF4 → snapshot
//	  (QBP-CI-guarded: regenerated + diffed producer-side) →
//	  ROMs + emulator fanoLUT (THIS gate) → FANO ISA opcode.
//
// Snapshot format — pinned by producer @qbp-oppenheimer (2026-08-20), emitted
// at proofs/QBP/Foundations/fanoTableF4.snapshot in the QBP repo:
//   - '#'-prefixed comment lines (incl. a provenance header and the
//     `#print axioms fanoTableF4_eq_cayleyDickson` attestation) are skipped.
//   - Then EXACTLY 64 data lines "i j sign index", space-separated:
//     i outer 0..7, j inner 0..7, row-major; sign ∈ {-1,1}; index 0..7.
//   - LF only, trailing newline.
//
// The snapshot lives in the sibling QBP repo. CI checks it out and sets
// FANO_SNAPSHOT (see .github/workflows/gcg-fano-proof-tie.yml); that workflow
// fails-closed if the snapshot or its axiom attestation is absent. Locally the
// cross-repo gate skips unless FANO_SNAPSHOT is set — but the parser and the
// comparator are fully exercised by the synthetic-fixture tests below, which
// need no external repo.
package emulator

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// snapEntry is one (sign, index) cell of the proven fanoTableF4.
type snapEntry struct {
	sign  int8
	index int8
}

// parseFanoSnapshot reads the pinned snapshot format into an 8×8 table,
// enforcing the format invariants so a malformed producer export fails loudly
// rather than silently mis-comparing: exactly 64 data rows, full row-major
// coverage of (i,j) ∈ [0,7]², sign ∈ {-1,1}, index ∈ [0,7], no duplicates.
func parseFanoSnapshot(r io.Reader) ([8][8]snapEntry, error) {
	var table [8][8]snapEntry
	var seen [8][8]bool
	n := 0
	sc := bufio.NewScanner(r)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 4 {
			return table, fmt.Errorf("line %d: got %d fields, want 4 (i j sign index): %q", lineNo, len(fields), line)
		}
		i, e1 := strconv.Atoi(fields[0])
		j, e2 := strconv.Atoi(fields[1])
		sign, e3 := strconv.Atoi(fields[2])
		idx, e4 := strconv.Atoi(fields[3])
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
			return table, fmt.Errorf("line %d: non-integer field: %q", lineNo, line)
		}
		if i < 0 || i > 7 || j < 0 || j > 7 {
			return table, fmt.Errorf("line %d: (i,j)=(%d,%d) out of [0,7]", lineNo, i, j)
		}
		if sign != -1 && sign != 1 {
			return table, fmt.Errorf("line %d: sign=%d, want -1 or 1", lineNo, sign)
		}
		if idx < 0 || idx > 7 {
			return table, fmt.Errorf("line %d: index=%d out of [0,7]", lineNo, idx)
		}
		if seen[i][j] {
			return table, fmt.Errorf("line %d: duplicate entry (i,j)=(%d,%d)", lineNo, i, j)
		}
		seen[i][j] = true
		table[i][j] = snapEntry{sign: int8(sign), index: int8(idx)}
		n++
	}
	if err := sc.Err(); err != nil {
		return table, err
	}
	if n != 64 {
		return table, fmt.Errorf("got %d data lines, want exactly 64", n)
	}
	for i := range 8 {
		for j := range 8 {
			if !seen[i][j] {
				return table, fmt.Errorf("missing entry (i,j)=(%d,%d)", i, j)
			}
		}
	}
	return table, nil
}

// snapshotDiscrepancies checks the proven table against BOTH wired artifacts —
// the octonion ROMs and the emulator fanoLUT (the FANO ISA opcode) — plus the
// structural invariants the proof implies (XOR-index lemma, identity rows).
// It returns every divergence found; an empty slice means the wired chain
// matches the kernel-proven table.
func snapshotDiscrepancies(table [8][8]snapEntry, g *Gearbox, idxROM, signROM []uint8) []string {
	var d []string

	// XOR-index lemma over the full 8×8: the proven product index is i⊕j.
	for i := range 8 {
		for j := range 8 {
			if int(table[i][j].index) != (i ^ j) {
				d = append(d, fmt.Sprintf("snapshot index[%d][%d]=%d violates XOR lemma (i⊕j=%d)", i, j, table[i][j].index, i^j))
			}
		}
	}
	// Identity rows/cols: e0·ej = +ej and ei·e0 = +ei (sign +1).
	for k := range 8 {
		if table[0][k].sign != 1 {
			d = append(d, fmt.Sprintf("snapshot sign[0][%d]=%+d, want +1 (e0·e%d identity)", k, table[0][k].sign, k))
		}
		if table[k][0].sign != 1 {
			d = append(d, fmt.Sprintf("snapshot sign[%d][0]=%+d, want +1 (e%d·e0 identity)", k, table[k][0].sign, k))
		}
	}

	// octonion_idx.hex: full 8×8 XOR table, row-major idx[i*8+j].
	if len(idxROM) != 64 {
		d = append(d, fmt.Sprintf("octonion_idx.hex: %d entries, want 64", len(idxROM)))
	} else {
		for i := range 8 {
			for j := range 8 {
				if int8(idxROM[i*8+j]) != table[i][j].index {
					d = append(d, fmt.Sprintf("ROM idx[%d][%d]=%d ≠ proof index %d", i, j, idxROM[i*8+j], table[i][j].index))
				}
			}
		}
	}
	// octonion_signs.hex: 7×7 imaginary submatrix, row-major (i-1)*7+(j-1),
	// 0=negative, 1=positive.
	if len(signROM) != 49 {
		d = append(d, fmt.Sprintf("octonion_signs.hex: %d entries, want 49", len(signROM)))
	} else {
		for i := 1; i <= 7; i++ {
			for j := 1; j <= 7; j++ {
				romSign := int8(-1)
				if signROM[(i-1)*7+(j-1)] == 1 {
					romSign = 1
				}
				if romSign != table[i][j].sign {
					d = append(d, fmt.Sprintf("ROM sign[%d][%d]=%+d ≠ proof sign %+d", i, j, romSign, table[i][j].sign))
				}
			}
		}
	}

	// Emulator fanoLUT (the FANO ISA opcode) over the imaginary 7×7 range.
	for i := 1; i <= 7; i++ {
		for j := 1; j <= 7; j++ {
			got := g.FanoLookup(i, j)
			if got.Index != table[i][j].index || got.Sign != table[i][j].sign {
				d = append(d, fmt.Sprintf("emulator FanoLookup(%d,%d)={Index:%d,Sign:%+d} ≠ proof {index:%d,sign:%+d}",
					i, j, got.Index, got.Sign, table[i][j].index, table[i][j].sign))
			}
		}
	}
	return d
}

// TestFanoProofTie_MatchesKernelProvenTable is the cross-repo prove-before-bake
// gate. It requires the QBP-produced fanoTableF4.snapshot, whose path is passed
// via FANO_SNAPSHOT. Run in CI by .github/workflows/gcg-fano-proof-tie.yml,
// which checks out QBP and fails-closed if the snapshot is absent — so a
// missing snapshot can never present as a silent pass.
func TestFanoProofTie_MatchesKernelProvenTable(t *testing.T) {
	snapPath := os.Getenv("FANO_SNAPSHOT")
	if snapPath == "" {
		// Skip whenever FANO_SNAPSHOT is unset — INCLUDING under CI. The
		// cross-repo drift gate runs ONLY under
		// .github/workflows/gcg-fano-proof-tie.yml, which checks out QBP,
		// fail-closes at the workflow level if the snapshot is absent, and sets
		// FANO_SNAPSHOT before invoking this test — that workflow is the real
		// guard against a silent skip. Every OTHER emulator CI job
		// (verify-lean-roms, the GCG ladder) runs `go test ./...` WITHOUT that
		// env, and must skip this cross-repo gate, not fail on it. (A prior
		// `CI=true`→Fatal guard here was too broad: it broke those unrelated
		// jobs on every emulator-touching PR.) Parser + comparator are covered
		// by the synthetic-fixture tests in this file, which always run.
		t.Skip("FANO_SNAPSHOT unset — the cross-repo drift gate runs only under gcg-fano-proof-tie.yml (which checks out QBP and sets it); skipped here.")
	}

	f, err := os.Open(snapPath)
	if err != nil {
		t.Fatalf("open snapshot %s: %v", snapPath, err)
	}
	defer f.Close()
	table, err := parseFanoSnapshot(f)
	if err != nil {
		t.Fatalf("parse snapshot %s: %v", snapPath, err)
	}

	romsDir := findRomsDir(t)
	idxROM, err := loadSignsHex(filepath.Join(romsDir, "octonion_idx.hex"))
	if err != nil {
		t.Fatalf("load octonion_idx.hex: %v", err)
	}
	signROM, err := loadSignsHex(filepath.Join(romsDir, "octonion_signs.hex"))
	if err != nil {
		t.Fatalf("load octonion_signs.hex: %v", err)
	}

	if d := snapshotDiscrepancies(table, NewGearbox(), idxROM, signROM); len(d) > 0 {
		for _, m := range d {
			t.Errorf("prove-before-bake drift: %s", m)
		}
		t.Fatalf("%d divergence(s) between kernel-proven fanoTableF4 and the wired ROMs/emulator — the chain is NOT proof-tied", len(d))
	}
}

// buildProvenFixture renders the 64-line snapshot implied by the currently
// wired artifacts (emulator fanoLUT + the XOR-index lemma). It exists ONLY to
// exercise the parser and comparator mechanism offline — it is NOT a proof
// source (that is QBP's kernel-proven fanoTableF4, consumed via FANO_SNAPSHOT).
func buildProvenFixture(t *testing.T) string {
	t.Helper()
	g := NewGearbox()
	var b strings.Builder
	b.WriteString("# synthetic fixture for mechanism testing — NOT the proof source\n")
	b.WriteString("# axioms: [propext, Classical.choice, Quot.sound]\n")
	for i := range 8 {
		for j := range 8 {
			idx := i ^ j
			var sign int8 = 1 // identity rows/cols and cyclic positives
			switch {
			case i == 0 || j == 0:
				sign = 1
			case i == j:
				sign = -1 // e_i·e_i = -e0
			default:
				sign = g.FanoLookup(i, j).Sign
			}
			fmt.Fprintf(&b, "%d %d %d %d\n", i, j, sign, idx)
		}
	}
	return b.String()
}

func TestFanoSnapshotParser_AcceptsWellFormed(t *testing.T) {
	table, err := parseFanoSnapshot(strings.NewReader(buildProvenFixture(t)))
	if err != nil {
		t.Fatalf("well-formed fixture rejected: %v", err)
	}
	if got := table[0][3]; got != (snapEntry{sign: 1, index: 3}) {
		t.Errorf("table[0][3]=%+v, want {sign:1 index:3} (e0·e3 identity)", got)
	}
	if got := table[2][2]; got != (snapEntry{sign: -1, index: 0}) {
		t.Errorf("table[2][2]=%+v, want {sign:-1 index:0} (e2·e2 = -e0)", got)
	}
}

func TestFanoSnapshotParser_RejectsMalformed(t *testing.T) {
	cases := []struct{ name, in, wantSub string }{
		{"wrong field count", "0 1 1", "want 4"},
		{"non-integer field", "0 1 x 1", "non-integer"},
		{"i out of range", "8 1 1 1", "out of [0,7]"},
		{"bad sign", "0 1 0 1", "want -1 or 1"},
		{"index out of range", "0 1 1 9", "out of [0,7]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := parseFanoSnapshot(strings.NewReader(c.in + "\n")); err == nil || !strings.Contains(err.Error(), c.wantSub) {
				t.Errorf("in=%q: err=%v, want substring %q", c.in, err, c.wantSub)
			}
		})
	}
}

func TestFanoSnapshotParser_RejectsWrongCount(t *testing.T) {
	base := strings.TrimRight(buildProvenFixture(t), "\n")
	lines := strings.Split(base, "\n")
	short := strings.Join(lines[:len(lines)-1], "\n") + "\n" // 63 data lines
	if _, err := parseFanoSnapshot(strings.NewReader(short)); err == nil {
		t.Fatal("short snapshot accepted; want count/missing-entry error")
	} else if !strings.Contains(err.Error(), "64") && !strings.Contains(err.Error(), "missing") {
		t.Errorf("short snapshot: err=%v, want count/missing error", err)
	}
}

func TestFanoSnapshotParser_RejectsDuplicate(t *testing.T) {
	dup := buildProvenFixture(t) + "0 0 1 0\n" // 65 lines: (0,0) repeated
	if _, err := parseFanoSnapshot(strings.NewReader(dup)); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("duplicate snapshot: err=%v, want duplicate error", err)
	}
}

// TestFanoProofTie_DetectsDrift proves the comparator both (a) passes a table
// consistent with the wired artifacts and (b) catches a corrupted one — so the
// gate cannot let real drift through. Uses synthetic fixtures, no QBP checkout.
func TestFanoProofTie_DetectsDrift(t *testing.T) {
	romsDir := findRomsDir(t)
	idxROM, err := loadSignsHex(filepath.Join(romsDir, "octonion_idx.hex"))
	if err != nil {
		t.Fatalf("load octonion_idx.hex: %v", err)
	}
	signROM, err := loadSignsHex(filepath.Join(romsDir, "octonion_signs.hex"))
	if err != nil {
		t.Fatalf("load octonion_signs.hex: %v", err)
	}
	g := NewGearbox()

	good, err := parseFanoSnapshot(strings.NewReader(buildProvenFixture(t)))
	if err != nil {
		t.Fatalf("fixture parse: %v", err)
	}
	if d := snapshotDiscrepancies(good, g, idxROM, signROM); len(d) > 0 {
		t.Fatalf("consistent fixture flagged %d false discrepancies: %v", len(d), d)
	}

	// Flip one proven sign ⇒ the comparator MUST catch it.
	badSign := good
	badSign[3][5].sign = -badSign[3][5].sign
	if len(snapshotDiscrepancies(badSign, g, idxROM, signROM)) == 0 {
		t.Fatal("sign-flip at [3][5] not detected — the gate would pass drift")
	}

	// Perturb one proven index ⇒ MUST be caught (XOR lemma + ROM + emulator).
	badIdx := good
	badIdx[3][5].index = (badIdx[3][5].index + 1) % 8
	if len(snapshotDiscrepancies(badIdx, g, idxROM, signROM)) == 0 {
		t.Fatal("index perturbation at [3][5] not detected — the gate would pass drift")
	}
}
