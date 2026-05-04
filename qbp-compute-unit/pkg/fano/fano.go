// Package fano implements the Fano plane multiplication table for octonion algebra.
//
// The Fano plane is a finite projective geometry with 7 points and 7 lines,
// each line through exactly 3 points. It governs the multiplication of
// imaginary octonion basis elements:
//
//	e_i × e_j = ±e_k    (determined by Fano plane)
//	e_i × e_i = -1       (like quaternions)
//
// The complete table has 49 entries (7×7 for imaginary units). Each entry
// is a (result_index, sign) pair. The entire algebra fits in <25 bytes.
//
// This is the FANO instruction: given two basis indices, return the product.
// On Run-phase hardware, this is a ROM lookup. On Crawl hardware, it's a
// Go array access.
//
// Fano orientation: We use the standard orientation with lines:
//
//	{1,2,3}, {1,4,5}, {1,7,6}, {2,4,6}, {2,5,7}, {3,4,7}, {3,6,5}
//
// This is one of 480 valid orientations. The choice of orientation is
// an open question (Section 10 of the spec) — it may be a gauge freedom
// or a design parameter.
package fano

// Entry represents one cell of the Fano multiplication table.
// Index is the resulting basis element (1-7), Sign is +1 or -1.
// For e_i * e_i, Index=0 signals the result is -1 (scalar).
type Entry struct {
	Index int8 // 0 = scalar result (-1), 1-7 = basis element
	Sign  int8 // +1 or -1
}

// table stores the 7×7 multiplication table for imaginary units e₁..e₇.
// Access as table[i-1][j-1] for e_i × e_j.
//
// Fano lines (oriented triples where e_a × e_b = +e_c):
//
//	e₁ × e₂ = +e₃     (line {1,2,3})
//	e₁ × e₄ = +e₅     (line {1,4,5})
//	e₁ × e₇ = +e₆     (line {1,7,6})
//	e₂ × e₄ = +e₆     (line {2,4,6})
//	e₂ × e₅ = +e₇     (line {2,5,7})
//	e₃ × e₄ = +e₇     (line {3,4,7})
//	e₃ × e₆ = +e₅     (line {3,6,5})
//
// Anti-cyclic products get sign -1.
// FanoLine represents an oriented triple: e_a × e_b = +e_c
type FanoLine struct {
	A, B, C int
}

// The 7 oriented lines of the Fano plane (standard orientation).
var lines = [7]FanoLine{
	{1, 2, 3},
	{1, 4, 5},
	{1, 7, 6},
	{2, 4, 6},
	{2, 5, 7},
	{3, 4, 7},
	{3, 6, 5},
}

// lut is the computed lookup table. Initialised by init().
// Access: lut[i][j] for e_{i+1} × e_{j+1}.
var lut [7][7]Entry

func init() {
	// Diagonal: e_i × e_i = -1 (scalar)
	for i := 0; i < 7; i++ {
		lut[i][i] = Entry{Index: 0, Sign: -1}
	}

	// From each oriented line {a,b,c}: e_a × e_b = +e_c
	// Cyclic:      e_a × e_b = +e_c,  e_b × e_c = +e_a,  e_c × e_a = +e_b
	// Anti-cyclic:  e_b × e_a = -e_c,  e_c × e_b = -e_a,  e_a × e_c = -e_b
	for _, l := range lines {
		a, b, c := l.A-1, l.B-1, l.C-1 // 0-indexed

		// Cyclic products (positive)
		lut[a][b] = Entry{Index: int8(l.C), Sign: 1}
		lut[b][c] = Entry{Index: int8(l.A), Sign: 1}
		lut[c][a] = Entry{Index: int8(l.B), Sign: 1}

		// Anti-cyclic products (negative)
		lut[b][a] = Entry{Index: int8(l.C), Sign: -1}
		lut[c][b] = Entry{Index: int8(l.A), Sign: -1}
		lut[a][c] = Entry{Index: int8(l.B), Sign: -1}
	}
}

// Lookup returns the result of e_i × e_j where i,j ∈ [1,7].
// Returns (result_index, sign). If result_index == 0, the result is -1 (scalar).
//
// This is the FANO instruction. On Run-phase RISC-V, it's a ROM lookup.
// On Crawl hardware, it's an array access — effectively zero cost.
func Lookup(i, j int) Entry {
	return lut[i-1][j-1]
}

// Verify checks that the multiplication table satisfies the expected
// algebraic properties. Returns nil if valid, error description if not.
// This is the Crawl-phase equivalent of the Lean 4 verification (Action 5).
func Verify() []string {
	var errs []string

	// Check 1: e_i × e_i = -1 for all i
	for i := 1; i <= 7; i++ {
		e := Lookup(i, i)
		if e.Index != 0 || e.Sign != -1 {
			errs = append(errs, "e_i*e_i != -1 for i="+string(rune('0'+i)))
		}
	}

	// Check 2: Anti-commutativity — e_i × e_j = -(e_j × e_i) for i != j
	for i := 1; i <= 7; i++ {
		for j := i + 1; j <= 7; j++ {
			eij := Lookup(i, j)
			eji := Lookup(j, i)
			if eij.Index != eji.Index || eij.Sign != -eji.Sign {
				errs = append(errs, "anti-commutativity violated for e_"+
					string(rune('0'+i))+"*e_"+string(rune('0'+j)))
			}
		}
	}

	// Check 3: Every off-diagonal product yields a valid basis element (1-7)
	for i := 1; i <= 7; i++ {
		for j := 1; j <= 7; j++ {
			if i == j {
				continue
			}
			e := Lookup(i, j)
			if e.Index < 1 || e.Index > 7 {
				errs = append(errs, "invalid product index")
			}
			if e.Sign != 1 && e.Sign != -1 {
				errs = append(errs, "invalid sign")
			}
		}
	}

	// Check 4: Each row is a permutation of {1..7} (excluding diagonal)
	for i := 1; i <= 7; i++ {
		seen := make(map[int8]bool)
		for j := 1; j <= 7; j++ {
			if i == j {
				continue
			}
			idx := Lookup(i, j).Index
			if seen[idx] {
				errs = append(errs, "row not a permutation")
				break
			}
			seen[idx] = true
		}
	}

	return errs
}

// TableSize returns the storage size of the LUT in bytes.
// Each entry is 2 bytes (index + sign). 49 entries = 98 bytes.
// In Run-phase hardware, this can be packed to <25 bytes using
// 4-bit fields (3 bits index + 1 bit sign = 4 bits × 49 = 196 bits).
func TableSize() int {
	return 7 * 7 * 2 // 98 bytes in Go representation
}
