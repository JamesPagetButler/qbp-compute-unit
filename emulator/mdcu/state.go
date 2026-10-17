// Package mdcu is the SPIKE prototype of the multi-dimensional QBP Compute Unit
// (Xqbp v2): State Space = Tier × Width × Dimension.
//
// SPIKE STATUS: isolated from the frozen emulator v0.1.0-rc1 API (ADR-002).
// Nothing in package emulator depends on this package. Promotion into the
// Gearbox is gated on the spike verdict (doc/spikes/mdcu-xqbp-v2-spike.md).
package mdcu

import "fmt"

// Tier is the Cayley-Dickson algebraic tier (qtier CSR).
type Tier uint8

const (
	TierR Tier = iota // ℝ, 1D
	TierC             // ℂ, 2D
	TierH             // ℍ, 4D
	TierO             // 𝕆, 8D
	TierS             // 𝕊, 16D
)

// Dim returns the real dimension 2^tier of the algebra.
func (t Tier) Dim() int { return 1 << t }

func (t Tier) String() string {
	return [...]string{"R", "C", "H", "O", "S"}[t]
}

// Width is the precision width (qw CSR). Values mirror emulator.Width.
type Width int

const (
	W64  Width = 64
	W128 Width = 128 // double-double
	W256 Width = 256 // big.Float reference
)

// CSR is the Xqbp v2 configuration register set.
type CSR struct {
	QW    Width // precision width
	QTier Tier  // active algebraic tier
	QDim  int   // site / vector dimension D ∈ 1..16
	QMode uint8 // bits 0-2: Fano frame select (1..7, 0 = canonical); bit 7: ZD trap enable
	Flags uint8 // status: FlagZD, FlagSeam
}

const (
	FlagZD   uint8 = 1 << 0 // ZDCHK matched a zero-divisor plane
	FlagSeam uint8 = 1 << 1 // a narrowing / projection exceeded the drift budget
)

// RegFile is 32 physical Q-registers of 4 components (QW64 floor).
// Higher tiers are formed by Cayley-Dickson grouping (RVV LMUL-style):
//
//	O[i] = ⟨q[2i], q[2i+1]⟩            (16 octonion registers)
//	S[i] = ⟨q[4i], …, q[4i+3]⟩         (8 sedenion registers)
type RegFile struct {
	Q [32][4]float64
}

// O reads octonion register i (0..15).
func (r *RegFile) O(i int) (o [8]float64) {
	mustRange("O", i, 16)
	copy(o[:4], r.Q[2*i][:])
	copy(o[4:], r.Q[2*i+1][:])
	return
}

// SetO writes octonion register i.
func (r *RegFile) SetO(i int, o [8]float64) {
	mustRange("O", i, 16)
	copy(r.Q[2*i][:], o[:4])
	copy(r.Q[2*i+1][:], o[4:])
}

// S reads sedenion register i (0..7).
func (r *RegFile) S(i int) (s [16]float64) {
	mustRange("S", i, 8)
	for k := 0; k < 4; k++ {
		copy(s[4*k:4*k+4], r.Q[4*i+k][:])
	}
	return
}

// SetS writes sedenion register i.
func (r *RegFile) SetS(i int, s [16]float64) {
	mustRange("S", i, 8)
	for k := 0; k < 4; k++ {
		copy(r.Q[4*i+k][:], s[4*k:4*k+4])
	}
}

func mustRange(kind string, i, n int) {
	if i < 0 || i >= n {
		panic(fmt.Sprintf("mdcu: %s register %d out of range [0,%d)", kind, i, n))
	}
}
