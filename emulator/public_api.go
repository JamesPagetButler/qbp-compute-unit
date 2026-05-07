// Package emulator provides hardware-accelerated quaternion-algebra
// kernels used by the QBP-Node compute mesh. Consumers (notably Wyrd,
// the Quaternion-native typed hypergraph) should import the Gearbox
// type for stable Crawl-phase access to the typed-per-width surface.
//
// This file (public_api.go) implements the §3.1 surface specified in
// doc/wyrd-integration.md (landed via PR #21). It complements the
// existing big.Float-based Gearbox.Mul / Conj / Rotate / NormSq methods
// (qword.go) used by the ISA execution path in cpu.go.
//
// Architectural deviation from §3.1 literal text (recorded for the
// record):
//
//   - §3.1 declares "type Gearbox struct {}" stateless. The existing
//     Gearbox in qword.go has internal state (ActiveWidth, big.Float
//     scratchpads, QWord temporaries) and is consumed by cpu.go's ISA
//     execution path. Rather than introducing a second Gearbox type,
//     the typed-per-width methods below are added to the existing
//     struct as a separate file. From the caller's perspective these
//     methods are stateless: QMul64 / QAdd64 / QRot64 / QConj64 /
//     QNorm64 / QMul128 / QAdd128 / QRot128 / QConj128 / QNorm128 /
//     CMul64 / CAdd64 / CMul128 do not read or write Gearbox fields.
//     QMulHighPrec uses the existing big.Float pool (slow path).
//     Forward-looking: M1+ may refactor to a stateless wrapper or
//     rename the internal struct.
//
//   - §3.1 specifies Width as a Go iota-based enum (W8 = 0 … W1024 = 7).
//     The existing Width type in qword.go is "type Width int" with
//     bit-count values (W8 = 8, W64 = 64, …, W1024 = 1024). The
//     existing constants are consumed by cpu.go and isa.go and cannot
//     be renumbered without breaking the ISA execution path. This file
//     reuses the existing Width / W8 / … / W1024 declarations
//     unchanged. Wyrd consumers must use the named constants rather
//     than literal integers, per the doc's intent.
package emulator

import (
	"errors"
	"fmt"
)

// ErrTierUnsupported is the sentinel error returned by Crawl-phase
// public-API methods that require a hardware extension not yet enabled.
// Specifically: OMul64 / OAdd64 (Xqbpoct extension, M1+) and
// SMul64 / SAdd64 (Xqbpqec / ZDCHK extension, M2+). Use errors.Is to
// detect this condition.
var ErrTierUnsupported = errors.New("tier not yet supported in Crawl phase")

// SedenionResult bundles a sedenion multiply result with the
// zero-divisor diagnostic flag. Sedenions admit 42 cross-copy basis-sum
// zero-divisors; SMul64 reports when an operation hits one of them so
// callers can handle 0/0 hazards rather than propagating silently.
//
// Per RV-Fano-Implementation-Refinements §2, ZDClass values are:
//
//	0 = NotZD
//	1 = CrossCopySymbolic   (caught by ZDCHK.SYM; (i,j,k,l) in ZDIndices)
//	2 = GeneralFullMultiply (caught by full ZDCHK; ZDIndices unused)
//
// Crawl phase: SMul64 always returns ErrTierUnsupported and a zero
// SedenionResult; the type is exported now so that downstream consumers
// can compile against the eventual M2+ surface without churn.
type SedenionResult struct {
	Value     [16]float64
	ZDClass   uint8
	ZDIndices [4]uint8
}

// QMul64 computes the Hamilton product a · b at QW64 precision. On
// AVX-FMA hosts, dispatches to qmul64AVX. On other hosts, dispatches
// to the scalar fallback. This is a hot path: zero-allocation.
func (g *Gearbox) QMul64(a, b [4]float64) [4]float64 {
	var dst QW64
	qa := QW64(a)
	qb := QW64(b)
	qmul64(&dst, &qa, &qb)
	return [4]float64(dst)
}

// QAdd64 computes the component-wise sum a + b at QW64 precision.
// Hot path: zero-allocation.
func (g *Gearbox) QAdd64(a, b [4]float64) [4]float64 {
	var dst QW64
	qa := QW64(a)
	qb := QW64(b)
	qadd64(&dst, &qa, &qb)
	return [4]float64(dst)
}

// QRot64 applies unit quaternion q to vector v as q · v · q* at QW64
// precision. Hot path: zero-allocation.
func (g *Gearbox) QRot64(q, v [4]float64) [4]float64 {
	var dst QW64
	qq := QW64(q)
	qv := QW64(v)
	qrot64(&dst, &qq, &qv)
	return [4]float64(dst)
}

// QConj64 computes the conjugate a* at QW64 precision. Hot path:
// zero-allocation.
func (g *Gearbox) QConj64(a [4]float64) [4]float64 {
	var dst QW64
	qa := QW64(a)
	qconj64(&dst, &qa)
	return [4]float64(dst)
}

// QNorm64 computes the norm-squared (dot product with self) of a at
// QW64 precision. Hot path: zero-allocation.
func (g *Gearbox) QNorm64(a [4]float64) float64 {
	var dst float64
	qa := QW64(a)
	qnorm64(&dst, &qa)
	return dst
}

// QMul128 computes the Hamilton product a · b at QW128 (double-double)
// precision. The [8]float64 layout is "hi×4 then lo×4": indices 0..3
// are the high components (W, X, Y, Z); indices 4..7 are the low
// components. This layout is internal to the QW128 representation and
// matches qmath_128_amd64.s. Hot path: zero-allocation.
func (g *Gearbox) QMul128(a, b [8]float64) [8]float64 {
	var dst QW128
	qa := QW128(a)
	qb := QW128(b)
	qmul128(&dst, &qa, &qb)
	return [8]float64(dst)
}

// QAdd128 computes the component-wise double-double sum a + b at QW128
// precision. Layout matches QMul128. Hot path: zero-allocation.
func (g *Gearbox) QAdd128(a, b [8]float64) [8]float64 {
	var dst QW128
	qa := QW128(a)
	qb := QW128(b)
	qadd128(&dst, &qa, &qb)
	return [8]float64(dst)
}

// QRot128 applies unit quaternion q to vector v as q · v · q* at
// QW128 precision. Hot path: zero-allocation.
func (g *Gearbox) QRot128(q, v [8]float64) [8]float64 {
	var dst QW128
	qq := QW128(q)
	qv := QW128(v)
	qrot128(&dst, &qq, &qv)
	return [8]float64(dst)
}

// QConj128 computes the conjugate a* at QW128 precision. Hot path:
// zero-allocation.
func (g *Gearbox) QConj128(a [8]float64) [8]float64 {
	var dst QW128
	qa := QW128(a)
	qconj128(&dst, &qa)
	return [8]float64(dst)
}

// QNorm128 computes the norm-squared of a at QW128 precision; the
// result is delivered in the [8]float64 layout (W component carries
// the scalar in indices 0 and 4 hi/lo; remaining components are zero).
// Hot path: zero-allocation.
func (g *Gearbox) QNorm128(a [8]float64) [8]float64 {
	var dst QW128
	qa := QW128(a)
	qnorm128(&dst, &qa)
	return [8]float64(dst)
}

// QMulHighPrec is the software fallback for W256 / W512 / W1024 widths,
// using math/big.Float internally. Inputs are 4-component float64
// approximations; the function rounds high-precision intermediates back
// to float64 outputs. Use only for verification or correctness
// baselines; performance is ~1400 ns/op or worse.
//
// Allowed widths are W256, W512, W1024. W8…W128 are rejected with a
// non-nil error so callers do not silently fall off the fast path; the
// hot-path methods (QMul64 / QMul128) remain the canonical entry for
// those widths.
func (g *Gearbox) QMulHighPrec(w Width, a, b [4]float64) ([4]float64, error) {
	var prec uint
	switch w {
	case W256:
		prec = 256
	case W512:
		prec = 512
	case W1024:
		prec = 1024
	default:
		return [4]float64{}, fmt.Errorf("QMulHighPrec: width %v not in {W256, W512, W1024}; use QMul64 / QMul128 for fast paths", w)
	}

	qa := NewQWord(prec)
	qa.W.SetFloat64(a[0])
	qa.X.SetFloat64(a[1])
	qa.Y.SetFloat64(a[2])
	qa.Z.SetFloat64(a[3])

	qb := NewQWord(prec)
	qb.W.SetFloat64(b[0])
	qb.X.SetFloat64(b[1])
	qb.Y.SetFloat64(b[2])
	qb.Z.SetFloat64(b[3])

	dst := NewQWord(prec)

	// Snapshot existing Gearbox precision so we don't disturb cpu.go's
	// in-flight ISA-execution state. SetWidth re-scales the scratchpads
	// to the requested precision; we restore on exit.
	prevWidth := g.ActiveWidth
	g.SetWidth(w)
	g.Mul(&dst, &qa, &qb)
	g.SetWidth(prevWidth)

	out := [4]float64{}
	out[0], _ = dst.W.Float64()
	out[1], _ = dst.X.Float64()
	out[2], _ = dst.Y.Float64()
	out[3], _ = dst.Z.Float64()
	return out, nil
}

// CMul64 computes the complex product a · b at fp64 precision, where
// each operand is laid out as [real, imag]. Hot path: zero-allocation.
func (g *Gearbox) CMul64(a, b [2]float64) [2]float64 {
	return [2]float64{
		a[0]*b[0] - a[1]*b[1],
		a[0]*b[1] + a[1]*b[0],
	}
}

// CAdd64 computes the complex sum a + b at fp64 precision. Hot path:
// zero-allocation.
func (g *Gearbox) CAdd64(a, b [2]float64) [2]float64 {
	return [2]float64{a[0] + b[0], a[1] + b[1]}
}

// CMul128 computes the complex product a · b at QW128 (double-double)
// precision. Layout per operand: [real_hi, imag_hi, real_lo, imag_lo].
// Hot path: zero-allocation.
func (g *Gearbox) CMul128(a, b [4]float64) [4]float64 {
	// (a_re + a_im·i)(b_re + b_im·i) = (a_re·b_re - a_im·b_im) + (a_re·b_im + a_im·b_re)·i
	// double-double per term, then renormalize.
	pHi1, pLo1 := ddMul(a[0], a[2], b[0], b[2]) // a_re * b_re
	pHi2, pLo2 := ddMul(a[1], a[3], b[1], b[3]) // a_im * b_im
	rReHi, rReLo := ddAdd(pHi1, pLo1, -pHi2, -pLo2)

	pHi3, pLo3 := ddMul(a[0], a[2], b[1], b[3]) // a_re * b_im
	pHi4, pLo4 := ddMul(a[1], a[3], b[0], b[2]) // a_im * b_re
	rImHi, rImLo := ddAdd(pHi3, pLo3, pHi4, pLo4)

	rReHi, rReLo = twoSum(rReHi, rReLo)
	rImHi, rImLo = twoSum(rImHi, rImLo)

	return [4]float64{rReHi, rImHi, rReLo, rImLo}
}

// OMul64 computes the octonion product a · b at fp64 precision.
//
// Crawl phase: returns ErrTierUnsupported. The Xqbpoct extension lands
// in M1; until then the public surface exists only to allow Wyrd
// consumers to write tier-dispatching code that compiles today.
func (g *Gearbox) OMul64(a, b [8]float64) ([8]float64, error) {
	_, _ = a, b
	return [8]float64{}, ErrTierUnsupported
}

// OAdd64 computes the octonion sum a + b at fp64 precision.
//
// Crawl phase: returns ErrTierUnsupported. See OMul64.
func (g *Gearbox) OAdd64(a, b [8]float64) ([8]float64, error) {
	_, _ = a, b
	return [8]float64{}, ErrTierUnsupported
}

// SMul64 computes the sedenion product a · b at fp64 precision and
// reports the zero-divisor diagnostic in the returned SedenionResult.
//
// Crawl phase: returns ErrTierUnsupported. The Xqbpqec extension and
// the ZDCHK instruction land in M2; until then the public surface
// exists only to allow Wyrd consumers to write tier-dispatching code
// that compiles today.
func (g *Gearbox) SMul64(a, b [16]float64) (SedenionResult, error) {
	_, _ = a, b
	return SedenionResult{}, ErrTierUnsupported
}

// SAdd64 computes the sedenion sum a + b at fp64 precision.
//
// Crawl phase: returns ErrTierUnsupported. See SMul64.
func (g *Gearbox) SAdd64(a, b [16]float64) ([16]float64, error) {
	_, _ = a, b
	return [16]float64{}, ErrTierUnsupported
}
