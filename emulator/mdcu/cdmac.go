package mdcu

import (
	"math"
	"math/big"
)

// ---------------------------------------------------------------------------
// Cayley-Dickson convention (Schafer 1954), identical to
//   - cmd/lean2rom (source of roms/*.hex, Lean-verified)
//   - QBP proofs/QBP/Algebra/CDAlg.lean (mulCoeff)
//   - QBP analysis/688-in-flight/flowlib.py
//
//	⟨a,b⟩⟨c,d⟩ = ⟨ac − d̄b, da + bc̄⟩
//
// Basis products are sign-twisted XOR:  e_m · e_k = s(m,k) · e_{m⊕k}.
// ---------------------------------------------------------------------------

// conjSlice returns the CD conjugate (negate all but component 0).
func conjSlice(x []float64) []float64 {
	y := make([]float64, len(x))
	y[0] = x[0]
	for i := 1; i < len(x); i++ {
		y[i] = -x[i]
	}
	return y
}

// MulRef is the generic recursive Cayley-Dickson product on slices of length
// 2^k. It is a line-for-line port of flowlib.mul and is the reference
// ("basis loop") kernel. It allocates; it is NOT a hot path.
func MulRef(x, y []float64) []float64 {
	n := len(x)
	if n == 1 {
		return []float64{x[0] * y[0]}
	}
	h := n / 2
	a, b := x[:h], x[h:]
	c, d := y[:h], y[h:]
	lo1 := MulRef(a, c)
	lo2 := MulRef(conjSlice(d), b)
	hi1 := MulRef(d, a)
	hi2 := MulRef(b, conjSlice(c))
	z := make([]float64, n)
	for i := 0; i < h; i++ {
		z[i] = lo1[i] - lo2[i]
		z[h+i] = hi1[i] + hi2[i]
	}
	return z
}

// signTable[m][k] = s(m,k) ∈ {±1} for 16×16 (sedenion). Octonion and
// quaternion tables are the leading 8×8 / 4×4 sub-blocks (CD tower nesting).
var signTable [16][16]float64

func init() {
	for m := 0; m < 16; m++ {
		for k := 0; k < 16; k++ {
			em := make([]float64, 16)
			ek := make([]float64, 16)
			em[m], ek[k] = 1, 1
			p := MulRef(em, ek)
			idx := m ^ k
			if math.Abs(math.Abs(p[idx])-1) != 0 {
				panic("mdcu: XOR-index property violated")
			}
			signTable[m][k] = p[idx]
		}
	}
}

// SignTable returns s(m,k) for the 16×16 sedenion basis.
func SignTable() [16][16]float64 { return signTable }

// ---------------------------------------------------------------------------
// CD-MAC: fixed XOR permutation + sign ROM + N² multiply-accumulate.
//   z_{m⊕k} += s(m,k) · x_m · y_k
// One MAC per (m,k) pair: 64 MACs for 𝕆, 256 for 𝕊.
// ---------------------------------------------------------------------------

// MACs per product for each kernel (analytic; used for the MAC column).
const (
	MACsOMulCDMAC = 64
	MACsSMulCDMAC = 256
	// Recursive CD on top of 16-MAC quaternion products:
	// 𝕆 = 4 ℍ-products = 64 ; 𝕊 = 4 𝕆-products = 256 (same count, different order).
	MACsOMulRec = 64
	MACsSMulRec = 256
)

// OMulCDMAC multiplies two octonions with the CD-MAC datapath.
func OMulCDMAC(x, y *[8]float64) (z [8]float64) {
	for m := 0; m < 8; m++ {
		xm := x[m]
		for k := 0; k < 8; k++ {
			z[m^k] += signTable[m][k] * xm * y[k]
		}
	}
	return
}

// SMulCDMAC multiplies two sedenions with the CD-MAC datapath.
func SMulCDMAC(x, y *[16]float64) (z [16]float64) {
	for m := 0; m < 16; m++ {
		xm := x[m]
		for k := 0; k < 16; k++ {
			z[m^k] += signTable[m][k] * xm * y[k]
		}
	}
	return
}

// ---------------------------------------------------------------------------
// Recursive CD on fixed arrays (no allocation): ℍ → 𝕆 → 𝕊.
// ---------------------------------------------------------------------------

// cmul is CD over ℝ: (a,b)(c,d) = (ac − d·b, d·a + b·c)  (conj is identity on ℝ).
func cmul(a0, a1, c0, c1 float64) (float64, float64) {
	return a0*c0 - c1*a1, c1*a0 + a1*c0
}

// qmul is CD over ℂ, with the same evaluation order as flowlib.mul so the
// rounding is bit-identical to the reference recursion.
func qmul(x, y [4]float64) [4]float64 {
	// x = ⟨a,b⟩, y = ⟨c,d⟩ with a,b,c,d ∈ ℂ
	l10, l11 := cmul(x[0], x[1], y[0], y[1])  // a·c
	l20, l21 := cmul(y[2], -y[3], x[2], x[3]) // d̄·b
	h10, h11 := cmul(y[2], y[3], x[0], x[1])  // d·a
	h20, h21 := cmul(x[2], x[3], y[0], -y[1]) // b·c̄
	return [4]float64{l10 - l20, l11 - l21, h10 + h20, h11 + h21}
}

func qconj(a [4]float64) [4]float64 { return [4]float64{a[0], -a[1], -a[2], -a[3]} }

// OMulRec multiplies octonions via ⟨a,b⟩⟨c,d⟩ = ⟨ac − d̄b, da + bc̄⟩ over ℍ.
func OMulRec(x, y *[8]float64) (z [8]float64) {
	a := [4]float64{x[0], x[1], x[2], x[3]}
	b := [4]float64{x[4], x[5], x[6], x[7]}
	c := [4]float64{y[0], y[1], y[2], y[3]}
	d := [4]float64{y[4], y[5], y[6], y[7]}
	l1, l2 := qmul(a, c), qmul(qconj(d), b)
	h1, h2 := qmul(d, a), qmul(b, qconj(c))
	for i := 0; i < 4; i++ {
		z[i] = l1[i] - l2[i]
		z[4+i] = h1[i] + h2[i]
	}
	return
}

func oconj(a [8]float64) [8]float64 {
	for i := 1; i < 8; i++ {
		a[i] = -a[i]
	}
	return a
}

// SMulRec multiplies sedenions via the CD formula over 𝕆.
func SMulRec(x, y *[16]float64) (z [16]float64) {
	var a, b, c, d [8]float64
	copy(a[:], x[:8])
	copy(b[:], x[8:])
	copy(c[:], y[:8])
	copy(d[:], y[8:])
	dc, cc := oconj(d), oconj(c)
	l1, l2 := OMulRec(&a, &c), OMulRec(&dc, &b)
	h1, h2 := OMulRec(&d, &a), OMulRec(&b, &cc)
	for i := 0; i < 8; i++ {
		z[i] = l1[i] - l2[i]
		z[8+i] = h1[i] + h2[i]
	}
	return
}

// ---------------------------------------------------------------------------
// QW128: double-double CD-MAC (error-free TwoProd via FMA + TwoSum).
// ---------------------------------------------------------------------------

// DD is a double-double number hi+lo with |lo| ≤ ulp(hi)/2.
type DD struct{ Hi, Lo float64 }

func twoSum(a, b float64) (s, e float64) {
	s = a + b
	bb := s - a
	e = (a - (s - bb)) + (b - bb)
	return
}

func twoProd(a, b float64) (p, e float64) {
	p = a * b
	e = math.FMA(a, b, -p)
	return
}

func ddAdd(x, y DD) DD {
	s, e := twoSum(x.Hi, y.Hi)
	t, f := twoSum(x.Lo, y.Lo)
	e += t
	s, e = twoSum(s, e)
	e += f
	s, e = twoSum(s, e)
	return DD{s, e}
}

func ddMul(x, y DD) DD {
	p, e := twoProd(x.Hi, y.Hi)
	e += x.Hi*y.Lo + x.Lo*y.Hi
	p, e = twoSum(p, e)
	return DD{p, e}
}

// SMulCDMAC128 is the QW128 CD-MAC sedenion product.
func SMulCDMAC128(x, y *[16]DD) (z [16]DD) {
	for m := 0; m < 16; m++ {
		for k := 0; k < 16; k++ {
			t := ddMul(x[m], y[k])
			if signTable[m][k] < 0 {
				t = DD{-t.Hi, -t.Lo}
			}
			z[m^k] = ddAdd(z[m^k], t)
		}
	}
	return
}

// ---------------------------------------------------------------------------
// QW256: big.Float reference CD-MAC (exact enough to grade QW64/QW128).
// ---------------------------------------------------------------------------

// BigPrec is the mantissa precision of the QW256 reference.
const BigPrec = 256

// MulBig is the CD-MAC product at BigPrec for any 2^k length.
func MulBig(x, y []*big.Float) []*big.Float {
	n := len(x)
	z := make([]*big.Float, n)
	for i := range z {
		z[i] = new(big.Float).SetPrec(BigPrec)
	}
	t := new(big.Float).SetPrec(BigPrec)
	for m := 0; m < n; m++ {
		for k := 0; k < n; k++ {
			t.Mul(x[m], y[k])
			if signTable[m][k] < 0 {
				z[m^k].Sub(z[m^k], t)
			} else {
				z[m^k].Add(z[m^k], t)
			}
		}
	}
	return z
}

// ToBig lifts float64 components into BigPrec floats.
func ToBig(x []float64) []*big.Float {
	z := make([]*big.Float, len(x))
	for i, v := range x {
		z[i] = new(big.Float).SetPrec(BigPrec).SetFloat64(v)
	}
	return z
}
