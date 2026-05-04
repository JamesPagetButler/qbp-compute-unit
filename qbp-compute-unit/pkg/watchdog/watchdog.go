// Package watchdog implements the Algebraic Watchdog — a structural health
// monitor for QBP-algebraic computation.
//
// Instead of conventional error correction (detect bit-flip → fix bit),
// the watchdog monitors algebraic invariants and detects path instability
// BEFORE errors manifest as incorrect results.
//
// The primary signal is NORM DRIFT: for exact quaternion arithmetic,
// ||q₁q₂|| = ||q₁|| ||q₂|| (multiplicativity of norm). In finite-precision
// arithmetic, this drifts. The rate of drift over N operations is the
// "algebraic curvature" of the computation path.
//
// This implements Gemini's "Algebraic Curvature" concept with a concrete
// metric: cumulative norm deviation from unity after repeated unit-quaternion
// multiplications.
package watchdog

import (
	"fmt"
	"math"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

// Stats tracks algebraic health metrics over a computation.
type Stats struct {
	// Operations counted
	MulCount int64

	// Norm drift tracking
	CumulativeNormDrift float64 // Σ |1 - ||q||²| after each multiply
	MaxNormDrift        float64 // max |1 - ||q||²| observed
	LastNormSq          float64 // most recent ||q||²

	// Curvature: rate of norm drift per operation
	// Low curvature = algebraically stable path
	// High curvature = path is accumulating error, re-weight in Fano table
	DriftHistory []float64 // optional: per-operation drift values

	// Unitarity tracking (for spin-chain benchmark)
	// Tracks whether the evolution operator remains unitary
	MaxUnitarityDefect float64
}

// New creates a fresh watchdog.
func New() *Stats {
	return &Stats{
		LastNormSq: 1.0,
	}
}

// NewWithHistory creates a watchdog that records per-operation drift.
// Use for diagnostics; disable in production for memory efficiency.
func NewWithHistory(capacity int) *Stats {
	return &Stats{
		LastNormSq:   1.0,
		DriftHistory: make([]float64, 0, capacity),
	}
}

// ObserveMul records a quaternion multiplication result and updates
// algebraic health metrics.
//
// Call this after every Mul in the hot loop. The cost is one NormSq
// (4 FMAs) plus bookkeeping — negligible compared to the Mul itself.
//
// Expected usage:
//
//	q = quat.Mul(q, rotation)
//	watchdog.ObserveMul(q)
func (s *Stats) ObserveMul(result quat.Quat) {
	s.MulCount++

	nsq := quat.NormSq(result)
	drift := math.Abs(1.0 - nsq)

	s.CumulativeNormDrift += drift
	s.LastNormSq = nsq

	if drift > s.MaxNormDrift {
		s.MaxNormDrift = drift
	}

	if s.DriftHistory != nil {
		s.DriftHistory = append(s.DriftHistory, drift)
	}
}

// ObserveUnitarity records a unitarity defect for a matrix operation.
// For the spin-chain, this checks that U†U ≈ I after each time step.
// The defect is the max |δᵢⱼ - (U†U)ᵢⱼ|.
func (s *Stats) ObserveUnitarity(defect float64) {
	if defect > s.MaxUnitarityDefect {
		s.MaxUnitarityDefect = defect
	}
}

// Curvature returns the average norm drift per operation.
// This is the "algebraic curvature" metric.
//
// Interpretation:
//   - < 1e-15: machine-epsilon level, algebraically pristine
//   - 1e-15 to 1e-12: normal float64 accumulation, healthy
//   - 1e-12 to 1e-9: elevated, consider renormalisation
//   - > 1e-9: path is algebraically unstable, re-weight or renormalise
func (s *Stats) Curvature() float64 {
	if s.MulCount == 0 {
		return 0
	}
	return s.CumulativeNormDrift / float64(s.MulCount)
}

// NeedRenorm returns true if the accumulated norm drift suggests
// renormalisation is needed. The threshold is configurable but
// defaults to 1e-10 total drift.
func (s *Stats) NeedRenorm(threshold float64) bool {
	return math.Abs(1.0-s.LastNormSq) > threshold
}

// Report returns a human-readable summary of algebraic health.
func (s *Stats) Report() string {
	return fmt.Sprintf(
		"Algebraic Watchdog Report:\n"+
			"  Operations:          %d\n"+
			"  Curvature (avg):     %.3e\n"+
			"  Max norm drift:      %.3e\n"+
			"  Current ||q||²:      %.15f\n"+
			"  Cumulative drift:    %.3e\n"+
			"  Max unitarity defect:%.3e\n",
		s.MulCount,
		s.Curvature(),
		s.MaxNormDrift,
		s.LastNormSq,
		s.CumulativeNormDrift,
		s.MaxUnitarityDefect,
	)
}

// ─── Renormalisation strategies ────────────────────────────────────────────

// Renormalize returns a unit quaternion q/||q|| and records the correction.
// This is the "defensive" strategy — apply when curvature exceeds threshold.
func Renormalize(q quat.Quat) quat.Quat {
	return quat.Normalize(q)
}

// RenormEveryN creates a wrapper function that renormalises every N operations.
// Returns a function that should be called after each Mul.
// The interval N is a tuning parameter: too frequent = wasted cycles,
// too infrequent = accumulated drift.
func RenormEveryN(n int) func(quat.Quat, int64) quat.Quat {
	return func(q quat.Quat, opCount int64) quat.Quat {
		if opCount%int64(n) == 0 {
			return quat.Normalize(q)
		}
		return q
	}
}

// ─── Comparison framework for the benchmark ────────────────────────────────

// ComparisonResult holds the algebraic health comparison between
// QBP-native and float64-scalar approaches to the same simulation.
type ComparisonResult struct {
	QBPStats    *Stats
	ScalarStats *Stats
	Steps       int
	ChainLength int
}

// Summary returns a formatted comparison for the benchmark report.
func (cr *ComparisonResult) Summary() string {
	return fmt.Sprintf(
		"Algebraic Integrity Comparison (%d steps, %d-site chain):\n"+
			"  QBP-Algebraic:\n"+
			"    Curvature:       %.3e\n"+
			"    Max norm drift:  %.3e\n"+
			"    Max unitarity:   %.3e\n"+
			"  Float64-Scalar:\n"+
			"    Curvature:       %.3e\n"+
			"    Max norm drift:  %.3e\n"+
			"    Max unitarity:   %.3e\n"+
			"  Ratio (scalar/QBP curvature): %.2f×\n",
		cr.Steps, cr.ChainLength,
		cr.QBPStats.Curvature(),
		cr.QBPStats.MaxNormDrift,
		cr.QBPStats.MaxUnitarityDefect,
		cr.ScalarStats.Curvature(),
		cr.ScalarStats.MaxNormDrift,
		cr.ScalarStats.MaxUnitarityDefect,
		cr.ScalarStats.Curvature()/max(cr.QBPStats.Curvature(), 1e-300),
	)
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
