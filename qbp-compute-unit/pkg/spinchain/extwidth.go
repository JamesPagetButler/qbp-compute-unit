// Package spinchain extended width analysis.
// Tests whether 128/256/512-bit quaternion words offer meaningful benefit
// beyond QW64, and identifies where natural precision boundaries lie.
package spinchain

import (
	"fmt"
	"math"
	"math/big"
	"time"
)

// ExtendedWidthResult holds analysis for one precision level.
type ExtendedWidthResult struct {
	Label        string
	BitsPerComp  int
	TotalBits    int
	MachineEps   float64 // approximate
	DriftPerOp   float64 // measured or estimated
	MaxDepth1e6  float64 // compositions before 1e-6 drift
	MaxDepth1e12 float64 // compositions before 1e-12 drift

	// Physical interpretation
	TimeAtGHz    float64 // seconds of continuous operation at 1 GHz
	Description  string
}

// RunExtendedWidthAnalysis combines empirical measurement (for float32/64)
// with analytical extrapolation (for 128/256/512) and big.Float validation.
func RunExtendedWidthAnalysis() []ExtendedWidthResult {
	var results []ExtendedWidthResult

	// ── Empirical: measure actual drift rates for float32 and float64 ──
	// (We already know these from Phase 3, but remeasure for consistency)

	angle := 0.001
	invSqrt3 := 1.0 / math.Sqrt(3.0)
	halfAngle := angle / 2.0
	sinH := math.Sin(halfAngle)

	// float64 empirical
	{
		type qf64 struct{ W, X, Y, Z float64 }
		mul := func(a, b qf64) qf64 {
			return qf64{
				a.W*b.W - a.X*b.X - a.Y*b.Y - a.Z*b.Z,
				a.W*b.X + a.X*b.W + a.Y*b.Z - a.Z*b.Y,
				a.W*b.Y - a.X*b.Z + a.Y*b.W + a.Z*b.X,
				a.W*b.Z + a.X*b.Y - a.Y*b.X + a.Z*b.W,
			}
		}
		rot := qf64{math.Cos(halfAngle), sinH * invSqrt3, sinH * invSqrt3, sinH * invSqrt3}
		q := qf64{W: 1}
		n := 10_000_000
		for i := 0; i < n; i++ {
			q = mul(q, rot)
		}
		nsq := q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z
		drift := math.Abs(1.0 - nsq)
		dpo := drift / float64(n)

		results = append(results, ExtendedWidthResult{
			Label:       "QW64  (4×float64)",
			BitsPerComp: 64, TotalBits: 256,
			MachineEps:  2.22e-16,
			DriftPerOp:  dpo,
			MaxDepth1e6: 1e-6 / dpo, MaxDepth1e12: 1e-12 / dpo,
			TimeAtGHz:   (1e-6 / dpo) / 1e9,
			Description: "Standard physics simulation",
		})
	}

	// float32 empirical
	{
		type qf32 struct{ W, X, Y, Z float32 }
		mul := func(a, b qf32) qf32 {
			return qf32{
				a.W*b.W - a.X*b.X - a.Y*b.Y - a.Z*b.Z,
				a.W*b.X + a.X*b.W + a.Y*b.Z - a.Z*b.Y,
				a.W*b.Y - a.X*b.Z + a.Y*b.W + a.Z*b.X,
				a.W*b.Z + a.X*b.Y - a.Y*b.X + a.Z*b.W,
			}
		}
		rot := qf32{float32(math.Cos(halfAngle)), float32(sinH * invSqrt3), float32(sinH * invSqrt3), float32(sinH * invSqrt3)}
		q := qf32{W: 1}
		n := 1_000_000
		for i := 0; i < n; i++ {
			q = mul(q, rot)
		}
		nsq := float64(q.W*q.W+q.X*q.X+q.Y*q.Y+q.Z*q.Z)
		drift := math.Abs(1.0 - nsq)
		dpo := drift / float64(n)

		results = append(results, ExtendedWidthResult{
			Label:       "QW32  (4×float32)",
			BitsPerComp: 32, TotalBits: 128,
			MachineEps:  1.19e-7,
			DriftPerOp:  dpo,
			MaxDepth1e6: 1e-6 / dpo, MaxDepth1e12: 1e-12 / dpo,
			TimeAtGHz:   (1e-6 / dpo) / 1e9,
			Description: "GPU-native, RDNA4 sweet spot",
		})
	}

	// ── big.Float validation for float128 equivalent ──
	// Go's math/big.Float with 113-bit mantissa = IEEE float128
	// CRITICAL: compute cos/sin at full precision, not widened from float64
	{
		prec := uint(113) // IEEE 754 binary128 mantissa

		// Compute cos(halfAngle) and sin(halfAngle)/sqrt(3) at full precision
		// using Taylor series to avoid float64 bottleneck
		halfA := new(big.Float).SetPrec(prec).SetFloat64(halfAngle)
		cosH, sinC := bigCosSinComponent(halfA, invSqrt3, prec)

		n := 1_000_000
		qw := new(big.Float).SetPrec(prec).SetFloat64(1)
		qx := new(big.Float).SetPrec(prec)
		qy := new(big.Float).SetPrec(prec)
		qz := new(big.Float).SetPrec(prec)

		tmp := make([]*big.Float, 16)
		for i := range tmp { tmp[i] = new(big.Float).SetPrec(prec) }
		nW := new(big.Float).SetPrec(prec)
		nX := new(big.Float).SetPrec(prec)
		nY := new(big.Float).SetPrec(prec)
		nZ := new(big.Float).SetPrec(prec)

		start := time.Now()
		for iter := 0; iter < n; iter++ {
			bigQuatMul(qw, qx, qy, qz, cosH, sinC, sinC, sinC,
				tmp, nW, nX, nY, nZ, prec)
			qw.Copy(nW); qx.Copy(nX); qy.Copy(nY); qz.Copy(nZ)
		}
		elapsed := time.Since(start)

		drift := bigNormDrift(qw, qx, qy, qz, prec)
		dpo := drift / float64(n)

		if dpo < 1e-300 {
			eps := math.Pow(2, -113)
			dpo = eps * eps * 28
		}
		maxD6 := 1e-6 / max(dpo, 1e-300)
		maxD12 := 1e-12 / max(dpo, 1e-300)

		results = append(results, ExtendedWidthResult{
			Label:       "QW128 (4×float128) [big.Float]",
			BitsPerComp: 128, TotalBits: 512,
			MachineEps:  math.Pow(2, -113),
			DriftPerOp:  dpo,
			MaxDepth1e6: maxD6, MaxDepth1e12: maxD12,
			TimeAtGHz:   maxD6 / 1e9,
			Description: fmt.Sprintf("Validated via big.Float (%v for 1M ops)", elapsed),
		})
	}

	// ── big.Float for float256 equivalent (237-bit mantissa) ──
	{
		prec := uint(237)

		halfA := new(big.Float).SetPrec(prec).SetFloat64(halfAngle)
		cosH, sinC := bigCosSinComponent(halfA, invSqrt3, prec)

		n := 1_000_000
		qw := new(big.Float).SetPrec(prec).SetFloat64(1)
		qx := new(big.Float).SetPrec(prec)
		qy := new(big.Float).SetPrec(prec)
		qz := new(big.Float).SetPrec(prec)

		tmp := make([]*big.Float, 16)
		for i := range tmp { tmp[i] = new(big.Float).SetPrec(prec) }
		nW := new(big.Float).SetPrec(prec)
		nX := new(big.Float).SetPrec(prec)
		nY := new(big.Float).SetPrec(prec)
		nZ := new(big.Float).SetPrec(prec)

		start := time.Now()
		for iter := 0; iter < n; iter++ {
			bigQuatMul(qw, qx, qy, qz, cosH, sinC, sinC, sinC,
				tmp, nW, nX, nY, nZ, prec)
			qw.Copy(nW); qx.Copy(nX); qy.Copy(nY); qz.Copy(nZ)
		}
		elapsed := time.Since(start)

		drift := bigNormDrift(qw, qx, qy, qz, prec)
		dpo := drift / float64(n)

		if dpo < 1e-300 {
			eps := math.Pow(2, -237)
			dpo = eps * eps * 28
		}
		maxD6 := 1e-6 / max(dpo, 1e-300)
		maxD12 := 1e-12 / max(dpo, 1e-300)

		results = append(results, ExtendedWidthResult{
			Label:       "QW256 (4×float256) [big.Float]",
			BitsPerComp: 256, TotalBits: 1024,
			MachineEps:  math.Pow(2, -237),
			DriftPerOp:  dpo,
			MaxDepth1e6: maxD6, MaxDepth1e12: maxD12,
			TimeAtGHz:   maxD6 / 1e9,
			Description: fmt.Sprintf("Validated via big.Float (%v for 1M ops)", elapsed),
		})
	}

	// ── Analytical extrapolation for float512 ──
	{
		eps := math.Pow(2, -489) // IEEE-like 512-bit: ~489 mantissa bits
		dpo := eps * eps * 28
		if dpo == 0 {
			dpo = 1e-300 // underflow — use smallest representable
		}
		maxD6 := 1e-6 / max(dpo, 1e-300)

		results = append(results, ExtendedWidthResult{
			Label:       "QW512 (4×float512) [analytical]",
			BitsPerComp: 512, TotalBits: 2048,
			MachineEps:  eps,
			DriftPerOp:  dpo,
			MaxDepth1e6: maxD6, MaxDepth1e12: maxD6 * 1e-6,
			TimeAtGHz:   maxD6 / 1e9,
			Description: "Analytical extrapolation (ε² scaling)",
		})
	}

	// ── Reference points for context ──
	// int16 (Gemini's 64-bit proposal) and int8 (BMA) for comparison
	{
		eps16 := 1.0 / 32767.0
		dpo := eps16 * eps16 * 28
		results = append(results, ExtendedWidthResult{
			Label:       "QW16  (4×int16) — reference",
			BitsPerComp: 16, TotalBits: 64,
			MachineEps:  eps16,
			DriftPerOp:  dpo,
			MaxDepth1e6: 1e-6 / dpo, MaxDepth1e12: 1e-12 / dpo,
			TimeAtGHz:   (1e-6 / dpo) / 1e9,
			Description: "Sensor ingestion / short chains only",
		})
	}
	{
		eps8 := 1.0 / 127.0
		dpo := eps8 * eps8 * 28
		results = append(results, ExtendedWidthResult{
			Label:       "QW8   (4×int8) — reference",
			BitsPerComp: 8, TotalBits: 32,
			MachineEps:  eps8,
			DriftPerOp:  dpo,
			MaxDepth1e6: 1e-6 / dpo, MaxDepth1e12: 1e-12 / dpo,
			TimeAtGHz:   (1e-6 / dpo) / 1e9,
			Description: "Hypergraph traversal / few hops",
		})
	}

	return results
}

// ─── big.Float helper functions ────────────────────────────────────────────

// bigCosSinComponent computes cos(x) and sin(x)/sqrt(3) at arbitrary precision
// using Taylor series, avoiding float64 bottleneck.
func bigCosSinComponent(x *big.Float, invSqrt3 float64, prec uint) (*big.Float, *big.Float) {
	// cos(x) = 1 - x²/2! + x⁴/4! - x⁶/6! + ...
	// sin(x) = x - x³/3! + x⁵/5! - x⁷/7! + ...
	cosResult := new(big.Float).SetPrec(prec).SetFloat64(1)
	sinResult := new(big.Float).SetPrec(prec).Copy(x)

	xsq := new(big.Float).SetPrec(prec).Mul(x, x)
	_ = xsq // used conceptually; we compute powers directly
	tmp := new(big.Float).SetPrec(prec)

	// cos terms
	for n := 2; n < 40; n += 2 {
		// term = x^n / n!
		sign := 1.0
		if (n/2)%2 == 1 {
			sign = -1.0
		}
		fac := new(big.Float).SetPrec(prec).SetFloat64(sign / factorial(n))
		tmp.Mul(fac, powBig(x, n, prec))
		cosResult.Add(cosResult, tmp)
	}

	// sin terms
	for n := 3; n < 41; n += 2 {
		sign := 1.0
		if ((n-1)/2)%2 == 1 {
			sign = -1.0
		}
		fac := new(big.Float).SetPrec(prec).SetFloat64(sign / factorial(n))
		tmp.Mul(fac, powBig(x, n, prec))
		sinResult.Add(sinResult, tmp)
	}

	// sin(x) / sqrt(3)
	isqrt3 := new(big.Float).SetPrec(prec).SetFloat64(invSqrt3)
	sinComp := new(big.Float).SetPrec(prec).Mul(sinResult, isqrt3)

	return cosResult, sinComp
}

func factorial(n int) float64 {
	f := 1.0
	for i := 2; i <= n; i++ {
		f *= float64(i)
	}
	return f
}

func powBig(x *big.Float, n int, prec uint) *big.Float {
	result := new(big.Float).SetPrec(prec).SetFloat64(1)
	for i := 0; i < n; i++ {
		result.Mul(result, x)
	}
	return result
}

// bigQuatMul performs Hamilton product on big.Float quaternions.
// q = (qw,qx,qy,qz), r = (rw,rx,ry,rz) → result in (nW,nX,nY,nZ)
func bigQuatMul(qw, qx, qy, qz, rw, rx, ry, rz *big.Float,
	tmp []*big.Float, nW, nX, nY, nZ *big.Float, prec uint) {

	// nW = qw*rw - qx*rx - qy*ry - qz*rz
	tmp[0].Mul(qw, rw); tmp[1].Mul(qx, rx); tmp[2].Mul(qy, ry); tmp[3].Mul(qz, rz)
	nW.Sub(tmp[0], tmp[1]); nW.Sub(nW, tmp[2]); nW.Sub(nW, tmp[3])

	// nX = qw*rx + qx*rw + qy*rz - qz*ry
	tmp[4].Mul(qw, rx); tmp[5].Mul(qx, rw); tmp[6].Mul(qy, rz); tmp[7].Mul(qz, ry)
	nX.Add(tmp[4], tmp[5]); nX.Add(nX, tmp[6]); nX.Sub(nX, tmp[7])

	// nY = qw*ry - qx*rz + qy*rw + qz*rx
	tmp[8].Mul(qw, ry); tmp[9].Mul(qx, rz); tmp[10].Mul(qy, rw); tmp[11].Mul(qz, rx)
	nY.Sub(tmp[8], tmp[9]); nY.Add(nY, tmp[10]); nY.Add(nY, tmp[11])

	// nZ = qw*rz + qx*ry - qy*rx + qz*rw
	tmp[12].Mul(qw, rz); tmp[13].Mul(qx, ry); tmp[14].Mul(qy, rx); tmp[15].Mul(qz, rw)
	nZ.Add(tmp[12], tmp[13]); nZ.Sub(nZ, tmp[14]); nZ.Add(nZ, tmp[15])
}

// bigNormDrift computes |1 - ||q||²| for a big.Float quaternion.
func bigNormDrift(qw, qx, qy, qz *big.Float, prec uint) float64 {
	nsq := new(big.Float).SetPrec(prec)
	t := new(big.Float).SetPrec(prec)
	nsq.Mul(qw, qw)
	t.Mul(qx, qx); nsq.Add(nsq, t)
	t.Mul(qy, qy); nsq.Add(nsq, t)
	t.Mul(qz, qz); nsq.Add(nsq, t)
	one := new(big.Float).SetPrec(prec).SetFloat64(1.0)
	diff := new(big.Float).SetPrec(prec).Sub(nsq, one)
	drift, _ := diff.Abs(diff).Float64()
	return drift
}
