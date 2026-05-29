package emulator

import "math"

// QW8 is the peripheral-register quaternion at int8 precision.
// Each component is a signed 8-bit integer; arithmetic saturates at ±127.
// QW8 is used exclusively for the peripheral-register coarse scan
// (Seam detection); it is 32× cheaper than QW128 in memory footprint
// and suitable for streaming operand pairs at high throughput.
//
// Per doc/design/m1-gearbox.md §2.3 — do not compose more than one
// quaternion product at QW8 without intermediate float64 renormalization;
// int8 saturation accumulates quickly.
type QW8 [4]int8

// PackQW8 converts a float64 quaternion to QW8 by clamping each
// component to [−127, 127] and rounding. Saturates: values outside
// the range are clamped (not wrapped).
func PackQW8(a [4]float64) QW8 {
	clamp := func(v float64) int8 {
		v = math.Round(v)
		if v > 127 {
			return 127
		}
		if v < -127 {
			return -127
		}
		return int8(v)
	}
	return QW8{clamp(a[0]), clamp(a[1]), clamp(a[2]), clamp(a[3])}
}

// UnpackQW8 converts a QW8 to float64 components. No precision loss
// since int8 values fit exactly in float64.
func UnpackQW8(a QW8) [4]float64 {
	return [4]float64{float64(a[0]), float64(a[1]), float64(a[2]), float64(a[3])}
}

// clampInt8 saturates a 32-bit product back to int8 range.
func clampInt8(v int32) int8 {
	if v > 127 {
		return 127
	}
	if v < -127 {
		return -127
	}
	return int8(v)
}

// QMul8 computes the Hamilton product a · b at QW8 (int8-saturating)
// precision. Pure-Go scalar implementation; no asm path.
// Acquires mu.RLock (M1+ concurrency model, consistent with QMul64).
// Hot path target: < 20 ns/op on FX-8350.
func (g *Gearbox) QMul8(a, b QW8) QW8 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	// Hamilton product: w=a[0], x=a[1], y=a[2], z=a[3]
	// (a+bi+cj+dk)(e+fi+gj+hk)
	w := int32(a[0])*int32(b[0]) - int32(a[1])*int32(b[1]) - int32(a[2])*int32(b[2]) - int32(a[3])*int32(b[3])
	x := int32(a[0])*int32(b[1]) + int32(a[1])*int32(b[0]) + int32(a[2])*int32(b[3]) - int32(a[3])*int32(b[2])
	y := int32(a[0])*int32(b[2]) - int32(a[1])*int32(b[3]) + int32(a[2])*int32(b[0]) + int32(a[3])*int32(b[1])
	z := int32(a[0])*int32(b[3]) + int32(a[1])*int32(b[2]) - int32(a[2])*int32(b[1]) + int32(a[3])*int32(b[0])
	return QW8{clampInt8(w), clampInt8(x), clampInt8(y), clampInt8(z)}
}

// QAdd8 computes the component-wise sum a + b at QW8 (saturating)
// precision. Hot path: zero-allocation.
func (g *Gearbox) QAdd8(a, b QW8) QW8 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return QW8{
		clampInt8(int32(a[0]) + int32(b[0])),
		clampInt8(int32(a[1]) + int32(b[1])),
		clampInt8(int32(a[2]) + int32(b[2])),
		clampInt8(int32(a[3]) + int32(b[3])),
	}
}

// QRot8 applies unit quaternion q to vector v as q · v · q* at QW8
// precision. Composed of two QMul8 calls; acquires mu.RLock once.
// Note: intermediate products can saturate; unit quaternion constraint
// is not enforced at QW8 (caller responsibility).
func (g *Gearbox) QRot8(q, v QW8) QW8 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	tmp := qmul8Raw(q, v)
	qConj := QW8{q[0], -q[1], -q[2], -q[3]}
	return qmul8Raw(tmp, qConj)
}

// QConj8 computes the conjugate a* = (w, -x, -y, -z) at QW8 precision.
// Negating −127 saturates at −127 (int8 min = −128 is skipped by PackQW8
// clamp; negating −127 produces 127, safe).
func (g *Gearbox) QConj8(a QW8) QW8 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return QW8{a[0], -a[1], -a[2], -a[3]}
}

// QNorm8 computes the norm-squared w² + x² + y² + z² as int32.
// Returns int32 to avoid overflow: max value is 4 × 127² = 64516,
// which fits comfortably in int32.
func (g *Gearbox) QNorm8(a QW8) int32 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return int32(a[0])*int32(a[0]) +
		int32(a[1])*int32(a[1]) +
		int32(a[2])*int32(a[2]) +
		int32(a[3])*int32(a[3])
}

// DetectSeam8 tests whether the operand pair (q, v) produces a Seam
// at QW8 precision using the A18 §4.1 residue criterion:
//
//	residue = |q · v · q* − v|  (using QW8 arithmetic)
//	seam = residue > threshold
//
// The threshold is caller-supplied (K · δ · ‖v‖ per A18 §9; use
// K=10 at Crawl per the compile-time default). Returns the residue as
// int32 and whether a Seam was detected.
//
// Note: QW8 norm is returned as int32 (not float32); the peripheral
// goroutine (m1.3) converts to float32 for the SeamEvent.Residue field.
func (g *Gearbox) DetectSeam8(q, v QW8, threshold int32) (residue int32, seam bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// r = q · v · q* at QW8
	tmp := qmul8Raw(q, v)
	qConj := QW8{q[0], -q[1], -q[2], -q[3]}
	r := qmul8Raw(tmp, qConj)

	// residue = |r − v| (component-wise L∞ approximation; fast at QW8)
	var maxDiff int32
	for i := range 4 {
		diff := int32(r[i]) - int32(v[i])
		if diff < 0 {
			diff = -diff
		}
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff, maxDiff > threshold
}

// qmul8Raw is the lock-free inner kernel used by QRot8 and DetectSeam8,
// both of which already hold mu.RLock.
func qmul8Raw(a, b QW8) QW8 {
	w := int32(a[0])*int32(b[0]) - int32(a[1])*int32(b[1]) - int32(a[2])*int32(b[2]) - int32(a[3])*int32(b[3])
	x := int32(a[0])*int32(b[1]) + int32(a[1])*int32(b[0]) + int32(a[2])*int32(b[3]) - int32(a[3])*int32(b[2])
	y := int32(a[0])*int32(b[2]) - int32(a[1])*int32(b[3]) + int32(a[2])*int32(b[0]) + int32(a[3])*int32(b[1])
	z := int32(a[0])*int32(b[3]) + int32(a[1])*int32(b[2]) - int32(a[2])*int32(b[1]) + int32(a[3])*int32(b[0])
	return QW8{clampInt8(w), clampInt8(x), clampInt8(y), clampInt8(z)}
}
