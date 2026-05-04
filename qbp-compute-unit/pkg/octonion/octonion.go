// Package octonion implements the octonion algebra 𝕆 using the Fano plane LUT.
//
// An octonion has 8 components: one real (a₀) and seven imaginary (a₁..a₇).
// Multiplication of imaginary units is governed by the Fano plane.
//
// For BMA integration:
//   - The real component a₀ encodes edge salience (weight/strength)
//   - The seven imaginary components encode edge type (relationship kind)
//   - Composition of edges is octonion multiplication
//
// The key property: octonions are NON-ASSOCIATIVE. (ab)c ≠ a(bc) in general.
// This means traversal order through the hypergraph matters — the grouping
// IS the context. This is a feature for memory systems, not a deficiency.
package octonion

import (
	"math"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/fano"
)

// ─── Physics-precision representation ──────────────────────────────────────

// Oct is an octonion with float64 components.
// Components[0] is the real part (salience); Components[1..7] are imaginary (edge types).
type Oct struct {
	C [8]float64 // C[0] = real, C[1..7] = e₁..e₇
}

// New constructs an octonion from 8 components.
func New(c0, c1, c2, c3, c4, c5, c6, c7 float64) Oct {
	return Oct{C: [8]float64{c0, c1, c2, c3, c4, c5, c6, c7}}
}

// Real returns a pure-real octonion (scalar).
func Real(v float64) Oct {
	return Oct{C: [8]float64{v}}
}

// Basis returns the i-th basis element (i=0 for real, 1-7 for imaginary).
func Basis(i int) Oct {
	var o Oct
	o.C[i] = 1.0
	return o
}

// ─── OMAC: Octonionic Multiply-Accumulate ──────────────────────────────────

// Mul computes the octonion product a * b using the Fano plane LUT.
//
// This is the OMAC instruction (without accumulate). The product decomposes:
//   - real × real → real
//   - real × imaginary → imaginary (scaling)
//   - imaginary × imaginary → Fano plane lookup
//
// On Run-phase RISC-V, this is a single-cycle instruction operating on
// two 64-bit packed octonion words. On Crawl hardware, it's ~120 FMAs.
func Mul(a, b Oct) Oct {
	var result Oct

	// real × real
	result.C[0] = a.C[0] * b.C[0]

	// real × imaginary and imaginary × real
	for i := 1; i <= 7; i++ {
		result.C[i] += a.C[0] * b.C[i]
		result.C[i] += a.C[i] * b.C[0]
	}

	// imaginary × imaginary: use Fano plane
	for i := 1; i <= 7; i++ {
		if a.C[i] == 0 {
			continue
		}
		for j := 1; j <= 7; j++ {
			if b.C[j] == 0 {
				continue
			}
			prod := a.C[i] * b.C[j]

			if i == j {
				// e_i × e_i = -1
				result.C[0] -= prod
			} else {
				entry := fano.Lookup(i, j)
				result.C[entry.Index] += float64(entry.Sign) * prod
			}
		}
	}

	return result
}

// MulAccum computes dest += a * b (octonionic multiply-accumulate).
// This is the full OMAC instruction.
func MulAccum(dest, a, b Oct) Oct {
	p := Mul(a, b)
	for i := 0; i < 8; i++ {
		dest.C[i] += p.C[i]
	}
	return dest
}

// ─── Norm ──────────────────────────────────────────────────────────────────

// NormSq returns ||o||² = Σ cᵢ².
func NormSq(o Oct) float64 {
	var s float64
	for i := 0; i < 8; i++ {
		s += o.C[i] * o.C[i]
	}
	return s
}

// Norm returns ||o||.
func Norm(o Oct) float64 {
	return math.Sqrt(NormSq(o))
}

// Normalize returns o / ||o||.
func Normalize(o Oct) Oct {
	n := Norm(o)
	var r Oct
	for i := 0; i < 8; i++ {
		r.C[i] = o.C[i] / n
	}
	return r
}

// ─── Conjugate ─────────────────────────────────────────────────────────────

// Conj returns the conjugate: negate all imaginary components.
func Conj(o Oct) Oct {
	var r Oct
	r.C[0] = o.C[0]
	for i := 1; i < 8; i++ {
		r.C[i] = -o.C[i]
	}
	return r
}

// ─── Arithmetic ────────────────────────────────────────────────────────────

// Add returns a + b.
func Add(a, b Oct) Oct {
	var r Oct
	for i := 0; i < 8; i++ {
		r.C[i] = a.C[i] + b.C[i]
	}
	return r
}

// Sub returns a - b.
func Sub(a, b Oct) Oct {
	var r Oct
	for i := 0; i < 8; i++ {
		r.C[i] = a.C[i] - b.C[i]
	}
	return r
}

// Scale returns s * o.
func Scale(s float64, o Oct) Oct {
	var r Oct
	for i := 0; i < 8; i++ {
		r.C[i] = s * o.C[i]
	}
	return r
}

// ─── Algebraic property tests ──────────────────────────────────────────────

// NormMultiplicativity checks that ||ab|| ≈ ||a|| × ||b|| within tolerance.
// This is the fundamental property that makes octonions a normed division algebra.
// If this fails under quantisation, the composition property is broken.
func NormMultiplicativity(a, b Oct, tol float64) bool {
	nab := Norm(Mul(a, b))
	nanb := Norm(a) * Norm(b)
	return math.Abs(nab-nanb) < tol
}

// AssociativityDefect computes ||(ab)c - a(bc)|| for three octonions.
// For quaternionic subalgebras (restricting to any 3 imaginary units),
// this should be zero. For general octonions, it's nonzero — that's the
// context-dependence feature.
func AssociativityDefect(a, b, c Oct) float64 {
	lhs := Mul(Mul(a, b), c) // (ab)c
	rhs := Mul(a, Mul(b, c)) // a(bc)
	return Norm(Sub(lhs, rhs))
}

// ─── Hypergraph-precision representation (BMA integration) ─────────────────

// Oct8 is an octonion with int8 components for hypergraph edge storage.
// 8 bytes per edge — compact enough for CIM-SRAM.
type Oct8 struct {
	C [8]int8 // C[0] = salience, C[1..7] = edge type components
}

// NewOct8 constructs an int8 octonion from components.
func NewOct8(c [8]int8) Oct8 {
	return Oct8{C: c}
}

// ToOct promotes an Oct8 to float64 Oct.
func (o8 Oct8) ToOct() Oct {
	var o Oct
	const scale = 1.0 / 127.0
	for i := 0; i < 8; i++ {
		o.C[i] = float64(o8.C[i]) * scale
	}
	return o
}

// ToOct8 quantises a float64 Oct to int8.
func ToOct8(o Oct) Oct8 {
	var o8 Oct8
	for i := 0; i < 8; i++ {
		scaled := o.C[i] * 127.0
		if scaled > 127 {
			o8.C[i] = 127
		} else if scaled < -127 {
			o8.C[i] = -127
		} else {
			o8.C[i] = int8(scaled)
		}
	}
	return o8
}
