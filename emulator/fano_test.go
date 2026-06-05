// fano_test.go — full 7×7 verification of FanoLookup against the
// Lean-derived ROM ground truth + algebraic property tests.
//
// Issue: https://github.com/JamesPagetButler/qbp-compute-unit/issues/57
// Prior to #57, FanoLookup hardcoded only the quaternion pairs and
// returned a wrong (index, sign) for all octonion-range products; CI
// never caught it because no test asserted product correctness.
package emulator

import (
	"path/filepath"
	"testing"
)

// TestFanoLookup_MatchesROMs is the drift gate: every FanoLookup(i,j)
// over the full 7×7 imaginary-unit range must match the Lean-derived
// ROMs (roms/octonion_idx.hex XOR-index lemma + roms/octonion_signs.hex
// orientation signs). This extends ROM-drift coverage from the ℍ
// sign-mask submatrix (TestSIMDConstantsMatchROM) to the full octonion
// table consumed by the FANO ISA opcode.
func TestFanoLookup_MatchesROMs(t *testing.T) {
	romsDir := findRomsDir(t)

	signs, err := loadSignsHex(filepath.Join(romsDir, "octonion_signs.hex"))
	if err != nil {
		t.Fatalf("loadSignsHex: %v", err)
	}
	if len(signs) != 49 {
		t.Fatalf("octonion_signs.hex: got %d entries, want 49", len(signs))
	}

	idx, err := loadSignsHex(filepath.Join(romsDir, "octonion_idx.hex"))
	if err != nil {
		t.Fatalf("load octonion_idx.hex: %v", err)
	}
	if len(idx) != 64 {
		t.Fatalf("octonion_idx.hex: got %d entries, want 64 (8×8 XOR table)", len(idx))
	}

	g := NewGearbox()
	for i := 1; i <= 7; i++ {
		for j := 1; j <= 7; j++ {
			got := g.FanoLookup(i, j)

			// Index: ROM is the XOR-index lemma over the 8×8 table.
			wantIdx := int8(idx[i*8+j])
			if got.Index != wantIdx {
				t.Errorf("FanoLookup(%d,%d).Index = %d, want %d (ROM XOR lemma)",
					i, j, got.Index, wantIdx)
			}

			// Sign: ROM is 7×7 row-major, 0=negative 1=positive.
			wantSign := int8(-1)
			if signs[(i-1)*7+(j-1)] == 1 {
				wantSign = 1
			}
			if got.Sign != wantSign {
				t.Errorf("FanoLookup(%d,%d).Sign = %d, want %d (ROM orientation)",
					i, j, got.Sign, wantSign)
			}
		}
	}
}

// TestFanoLookup_AlgebraicProperties verifies the structural octonion
// axioms independently of the ROMs (defense in depth: a corrupted ROM
// pair could still satisfy a drift test).
func TestFanoLookup_AlgebraicProperties(t *testing.T) {
	g := NewGearbox()

	// Diagonal: e_i × e_i = -1 (scalar).
	for i := 1; i <= 7; i++ {
		e := g.FanoLookup(i, i)
		if e.Index != 0 || e.Sign != -1 {
			t.Errorf("FanoLookup(%d,%d) = %+v, want {Index:0 Sign:-1}", i, i, e)
		}
	}

	// Anti-commutativity: e_i × e_j = -(e_j × e_i) for i ≠ j.
	for i := 1; i <= 7; i++ {
		for j := 1; j <= 7; j++ {
			if i == j {
				continue
			}
			eij := g.FanoLookup(i, j)
			eji := g.FanoLookup(j, i)
			if eij.Index != eji.Index || eij.Sign != -eji.Sign {
				t.Errorf("anti-commutativity violated: e%d×e%d=%+v, e%d×e%d=%+v",
					i, j, eij, j, i, eji)
			}
		}
	}

	// Off-diagonal products yield a valid basis element with valid sign.
	for i := 1; i <= 7; i++ {
		for j := 1; j <= 7; j++ {
			if i == j {
				continue
			}
			e := g.FanoLookup(i, j)
			if e.Index < 1 || e.Index > 7 {
				t.Errorf("FanoLookup(%d,%d).Index = %d out of [1,7]", i, j, e.Index)
			}
			if e.Sign != 1 && e.Sign != -1 {
				t.Errorf("FanoLookup(%d,%d).Sign = %d invalid", i, j, e.Sign)
			}
		}
	}

	// Each row is a permutation of {1..7}\{?}: no repeated product index.
	for i := 1; i <= 7; i++ {
		seen := map[int8]bool{}
		for j := 1; j <= 7; j++ {
			if i == j {
				continue
			}
			k := g.FanoLookup(i, j).Index
			if seen[k] {
				t.Errorf("row %d: product index %d repeated", i, k)
			}
			seen[k] = true
		}
	}
}

// TestFanoLookup_OctonionRegressions pins the octonion-range products
// the pre-#57 fallback got wrong. Each case documents the value the
// broken formula ((i+j)%7+1, always +1) would have returned.
func TestFanoLookup_OctonionRegressions(t *testing.T) {
	g := NewGearbox()
	cases := []struct {
		i, j int
		want FanoEntry
		// brokenIdx is what the pre-#57 fallback returned (always Sign +1).
		brokenIdx int8
	}{
		{4, 5, FanoEntry{Index: 1, Sign: 1}, 3},  // line {1,4,5} cyclic
		{5, 4, FanoEntry{Index: 1, Sign: -1}, 3}, // anti-cyclic
		{2, 4, FanoEntry{Index: 6, Sign: 1}, 7},  // line {2,4,6} cyclic
		{6, 7, FanoEntry{Index: 1, Sign: -1}, 7}, // line {1,7,6}: e6×e7 anti-cyclic of e7×e6=+e1
		{7, 6, FanoEntry{Index: 1, Sign: 1}, 7},  // e7×e6 = +e1 cyclic
		{3, 6, FanoEntry{Index: 5, Sign: 1}, 3},  // line {3,6,5} cyclic
		{5, 7, FanoEntry{Index: 2, Sign: 1}, 6},  // line {2,5,7}: e5×e7 cyclic (b,c)→a
		{4, 7, FanoEntry{Index: 3, Sign: 1}, 5},  // line {3,4,7}: e4×e7 cyclic (b,c)→a
		{7, 5, FanoEntry{Index: 2, Sign: -1}, 6}, // anti-cyclic
		{7, 4, FanoEntry{Index: 3, Sign: -1}, 5}, // anti-cyclic
	}
	for _, c := range cases {
		got := g.FanoLookup(c.i, c.j)
		if got != c.want {
			t.Errorf("FanoLookup(%d,%d) = %+v, want %+v (pre-#57 fallback returned {Index:%d Sign:1})",
				c.i, c.j, got, c.want, c.brokenIdx)
		}
	}
}

// TestFanoLookup_OutOfRange documents the bounds contract: indices
// outside [1,7] return the zero FanoEntry.
func TestFanoLookup_OutOfRange(t *testing.T) {
	g := NewGearbox()
	for _, pair := range [][2]int{{0, 1}, {1, 0}, {8, 1}, {1, 8}, {0, 0}, {-1, 3}} {
		if got := g.FanoLookup(pair[0], pair[1]); got != (FanoEntry{}) {
			t.Errorf("FanoLookup(%d,%d) = %+v, want zero FanoEntry", pair[0], pair[1], got)
		}
	}
}
