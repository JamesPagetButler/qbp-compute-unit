// Package spinchain composition stress test.
// This tests long-chain operator composition — the scenario where QBP's
// algebraic norm-preservation should visibly outperform scalar arithmetic.
package spinchain

import (
	"math"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/watchdog"
)

// CompositionResult holds results from the pure composition benchmark.
type CompositionResult struct {
	Iterations int

	// QBP
	QBPFinalNormSq    float64
	QBPFinalNormDrift float64
	QBPCurvature      float64
	QBPMaxDrift       float64
	QBPWallTime       time.Duration

	// Scalar (2x2 complex matrix chain)
	ScalarFinalNormSq    float64
	ScalarFinalNormDrift float64
	ScalarCurvature      float64
	ScalarMaxDrift       float64
	ScalarWallTime       time.Duration
}

// RunCompositionBenchmark composes N quaternion multiplications and N
// equivalent SU(2) matrix multiplications, measuring norm drift.
//
// This is the "stress test" that should reveal where QBP's algebraic
// norm-preservation diverges from float64 numerical behaviour.
//
// The rotation is small and non-trivial (not aligned with any axis)
// to exercise all components of the multiplication.
func RunCompositionBenchmark(iterations int) *CompositionResult {
	result := &CompositionResult{Iterations: iterations}

	// Small rotation: 0.001 radians around axis (1,1,1)/sqrt(3)
	angle := 0.001
	invSqrt3 := 1.0 / math.Sqrt(3.0)
	halfAngle := angle / 2.0
	sinH := math.Sin(halfAngle)

	// The rotation quaternion (unit quaternion)
	rotation := quat.New(
		math.Cos(halfAngle),
		sinH*invSqrt3,
		sinH*invSqrt3,
		sinH*invSqrt3,
	)

	// ── QBP: compose quaternion multiplications ──
	wd := watchdog.NewWithHistory(0) // no history for perf
	q := quat.Identity()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		q = quat.Mul(q, rotation)
		wd.ObserveMul(q)
	}
	result.QBPWallTime = time.Since(start)
	result.QBPFinalNormSq = quat.NormSq(q)
	result.QBPFinalNormDrift = math.Abs(1.0 - result.QBPFinalNormSq)
	result.QBPCurvature = wd.Curvature()
	result.QBPMaxDrift = wd.MaxNormDrift

	// ── Scalar: compose 2x2 complex matrix multiplications ──
	// Equivalent SU(2) matrix:
	// U = [[cos(θ/2) - i·sin(θ/2)·nz, -sin(θ/2)·(ny + i·nx)],
	//      [sin(θ/2)·(ny - i·nx), cos(θ/2) + i·sin(θ/2)·nz]]
	uA := Complex{Re: math.Cos(halfAngle), Im: -sinH * invSqrt3}   // u00
	uB := Complex{Re: -sinH * invSqrt3, Im: -sinH * invSqrt3}      // u01
	uC := Complex{Re: sinH * invSqrt3, Im: -sinH * invSqrt3}       // u10
	uD := Complex{Re: math.Cos(halfAngle), Im: sinH * invSqrt3}    // u11

	// Start with identity matrix
	mA := Complex{Re: 1}
	mB := Complex{}
	mC := Complex{}
	mD := Complex{Re: 1}

	var scalarMaxDrift float64
	var scalarCumDrift float64

	start = time.Now()
	for i := 0; i < iterations; i++ {
		// Matrix multiply: M = M * U
		newA := cadd(cmul(mA, uA), cmul(mB, uC))
		newB := cadd(cmul(mA, uB), cmul(mB, uD))
		newC := cadd(cmul(mC, uA), cmul(mD, uC))
		newD := cadd(cmul(mC, uB), cmul(mD, uD))
		mA, mB, mC, mD = newA, newB, newC, newD

		// Unitarity check: |det(M)| should = 1
		// det = AD - BC
		det := csub(cmul(mA, mD), cmul(mB, mC))
		detNorm := cnorm2(det)
		drift := math.Abs(1.0 - detNorm)
		scalarCumDrift += drift
		if drift > scalarMaxDrift {
			scalarMaxDrift = drift
		}
	}
	result.ScalarWallTime = time.Since(start)

	det := csub(cmul(mA, mD), cmul(mB, mC))
	result.ScalarFinalNormSq = cnorm2(det)
	result.ScalarFinalNormDrift = math.Abs(1.0 - result.ScalarFinalNormSq)
	if iterations > 0 {
		result.ScalarCurvature = scalarCumDrift / float64(iterations)
	}
	result.ScalarMaxDrift = scalarMaxDrift

	return result
}
