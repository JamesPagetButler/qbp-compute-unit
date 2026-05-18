// Package spinchain multi-width composition test.
// Demonstrates that the same algebra at different precisions produces
// vastly different composition lifetimes — proving that fixed-width
// is the wrong architectural choice.
package spinchain

import (
	"math"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

// WidthResult holds composition benchmark results at one precision.
type WidthResult struct {
	Label          string
	Bits           int
	Iterations     int
	FinalNormDrift float64
	DriftPerOp     float64
	WallTime       time.Duration

	// Estimated max compositions before 1e-6 tolerance exceeded
	EstMaxDepth int64
}

// RunMultiWidthBenchmark runs the same quaternion composition at multiple
// precisions to demonstrate why scalable-by-design is architecturally necessary.
func RunMultiWidthBenchmark(iterations int) []WidthResult {
	var results []WidthResult

	// Small rotation (same as composition benchmark)
	angle := 0.001
	invSqrt3 := 1.0 / math.Sqrt(3.0)
	halfAngle := angle / 2.0
	sinH := math.Sin(halfAngle)

	// ── QW64: float64 (256-bit quaternion word) ──
	{
		rotation := quat.New(
			math.Cos(halfAngle),
			sinH*invSqrt3,
			sinH*invSqrt3,
			sinH*invSqrt3,
		)
		q := quat.Identity()
		start := time.Now()
		for i := 0; i < iterations; i++ {
			q = quat.Mul(q, rotation)
		}
		elapsed := time.Since(start)
		drift := math.Abs(1.0 - quat.NormSq(q))
		dpo := drift / float64(iterations)
		results = append(results, WidthResult{
			Label:          "QW64 (4×float64)",
			Bits:           256,
			Iterations:     iterations,
			FinalNormDrift: drift,
			DriftPerOp:     dpo,
			WallTime:       elapsed,
			EstMaxDepth:    int64(1e-6 / max(dpo, 1e-300)),
		})
	}

	// ── QW32: float32 (128-bit quaternion word) ──
	{
		cosH := float32(math.Cos(halfAngle))
		sinComp := float32(sinH * invSqrt3)

		type q32 struct{ W, X, Y, Z float32 }
		mul32 := func(a, b q32) q32 {
			return q32{
				W: a.W*b.W - a.X*b.X - a.Y*b.Y - a.Z*b.Z,
				X: a.W*b.X + a.X*b.W + a.Y*b.Z - a.Z*b.Y,
				Y: a.W*b.Y - a.X*b.Z + a.Y*b.W + a.Z*b.X,
				Z: a.W*b.Z + a.X*b.Y - a.Y*b.X + a.Z*b.W,
			}
		}

		rot := q32{W: cosH, X: sinComp, Y: sinComp, Z: sinComp}
		q := q32{W: 1}
		start := time.Now()
		for i := 0; i < iterations; i++ {
			q = mul32(q, rot)
		}
		elapsed := time.Since(start)
		nsq := float64(q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z)
		drift := math.Abs(1.0 - nsq)
		dpo := drift / float64(iterations)
		results = append(results, WidthResult{
			Label:          "QW32 (4×float32)",
			Bits:           128,
			Iterations:     iterations,
			FinalNormDrift: drift,
			DriftPerOp:     dpo,
			WallTime:       elapsed,
			EstMaxDepth:    int64(1e-6 / max(dpo, 1e-300)),
		})
	}

	// ── QW16: int16 fixed-point (64-bit quaternion word — Gemini's proposal) ──
	{
		// Scale factor: map [-1,1] to [-32767, 32767]
		const scale int32 = 32767
		const shift = 15 // right-shift after multiply to rescale

		type q16 struct{ W, X, Y, Z int16 }
		toI16 := func(f float64) int16 {
			v := f * float64(scale)
			if v > 32767 {
				return 32767
			}
			if v < -32767 {
				return -32767
			}
			return int16(v)
		}

		mul16 := func(a, b q16) q16 {
			aw, ax, ay, az := int32(a.W), int32(a.X), int32(a.Y), int32(a.Z)
			bw, bx, by, bz := int32(b.W), int32(b.X), int32(b.Y), int32(b.Z)
			sat := func(v int32) int16 {
				if v > 32767 {
					return 32767
				}
				if v < -32767 {
					return -32767
				}
				return int16(v)
			}
			return q16{
				W: sat((aw*bw - ax*bx - ay*by - az*bz) >> shift),
				X: sat((aw*bx + ax*bw + ay*bz - az*by) >> shift),
				Y: sat((aw*by - ax*bz + ay*bw + az*bx) >> shift),
				Z: sat((aw*bz + ax*by - ay*bx + az*bw) >> shift),
			}
		}

		rot := q16{
			W: toI16(math.Cos(halfAngle)),
			X: toI16(sinH * invSqrt3),
			Y: toI16(sinH * invSqrt3),
			Z: toI16(sinH * invSqrt3),
		}
		q := q16{W: int16(scale)}
		start := time.Now()
		for i := 0; i < iterations; i++ {
			q = mul16(q, rot)
		}
		elapsed := time.Since(start)
		// Compute norm in float64 for comparison
		nsq := (float64(q.W)*float64(q.W) + float64(q.X)*float64(q.X) +
			float64(q.Y)*float64(q.Y) + float64(q.Z)*float64(q.Z)) / (float64(scale) * float64(scale))
		drift := math.Abs(1.0 - nsq)
		dpo := drift / float64(iterations)
		results = append(results, WidthResult{
			Label:          "QW16 (4×int16) — Gemini 64-bit proposal",
			Bits:           64,
			Iterations:     iterations,
			FinalNormDrift: drift,
			DriftPerOp:     dpo,
			WallTime:       elapsed,
			EstMaxDepth:    int64(1e-6 / max(dpo, 1e-300)),
		})
	}

	// ── QW8: int8 fixed-point (32-bit quaternion word — BMA hypergraph) ──
	{
		const scale8 int16 = 127
		const shift8 = 7

		rot8 := quat.Quat8{
			W: func() int8 {
				v := math.Cos(halfAngle) * 127
				return int8(v)
			}(),
			X: func() int8 {
				v := sinH * invSqrt3 * 127
				return int8(v)
			}(),
			Y: func() int8 {
				v := sinH * invSqrt3 * 127
				return int8(v)
			}(),
			Z: func() int8 {
				v := sinH * invSqrt3 * 127
				return int8(v)
			}(),
		}
		q := quat.Quat8{W: 127}
		start := time.Now()
		for i := 0; i < iterations; i++ {
			q = quat.Mul8(q, rot8)
		}
		elapsed := time.Since(start)
		nsq := (float64(q.W)*float64(q.W) + float64(q.X)*float64(q.X) +
			float64(q.Y)*float64(q.Y) + float64(q.Z)*float64(q.Z)) / (127.0 * 127.0)
		drift := math.Abs(1.0 - nsq)
		dpo := drift / float64(iterations)
		results = append(results, WidthResult{
			Label:          "QW8  (4×int8) — BMA hypergraph",
			Bits:           32,
			Iterations:     iterations,
			FinalNormDrift: drift,
			DriftPerOp:     dpo,
			WallTime:       elapsed,
			EstMaxDepth:    int64(1e-6 / max(dpo, 1e-300)),
		})
	}

	return results
}
