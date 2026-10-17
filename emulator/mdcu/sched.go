package mdcu

// P-tile scheduler model (L9 / A4 / frame average). A kernel is a list of
// dependency levels; each level holds independent ops of a given tariff.
// With P tiles, a level costs ceil(n/P)·tariff cycles (no overlap between
// levels). These are MODEL cycles — never mixed with host ns or MAC counts.

type Level struct {
	N      int // independent ops
	Tariff int // cycles per op
}

type Kernel struct {
	Name   string
	Levels []Level
}

func (k Kernel) Cycles(P int) (c int) {
	for _, l := range k.Levels {
		c += (l.N + P - 1) / P * l.Tariff
	}
	return
}

// Ops is the total op count (sequential issue ≡ P=1 with unit tariff).
func (k Kernel) Ops() (n int) {
	for _, l := range k.Levels {
		n += l.N
	}
	return
}

var Tiles = []int{1, 2, 4, 8, 16}

// Kernels modelled in the spike.
var Kernels = []Kernel{
	// Albert fast path: 6 OMUL ∥ 3 dot-pairs (6 dots) ∥ 6 scalings, then
	// one accumulate level (3 diag + 3 off-diag sums).
	{"albert_fast", []Level{{6 + 6 + 6, 1}, {6, 1}}},
	// Albert brute: 54 OMUL, then 9 entry accumulations.
	{"albert_brute", []Level{{54, 1}, {9, 1}}},
	// 7-frame average: 7 QSPLIT+norm² in parallel, one reduce.
	{"frame_avg7", []Level{{7, 1}, {1, 1}}},
	// Rule-flow gradient: basis 34 OMUL vs CD-MAC 6 OMUL (+1 combine).
	{"grad_basis", []Level{{34, 1}, {1, 1}}},
	{"grad_cdmac", []Level{{6, 1}, {1, 1}}},
	// Leaf round-trip: QSPLIT.LEAF → QJOIN.LEAF.
	{"leaf_roundtrip", []Level{{1, Tariff[OpQSPLITLF]}, {1, Tariff[OpQJOINLF]}}},
	// QASSOC (two dependent product levels).
	{"qassoc", []Level{{1, Tariff[OpQASSOC]}}},
}
