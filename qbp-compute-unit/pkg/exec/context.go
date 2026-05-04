// Package exec defines shared execution context types for the QBP Compute Unit.
//
// ExecutionContext is the bridge between the mesh scheduler's policy decisions
// and the HAL/emulator execution layer. It carries the required quaternion word
// width — computed from task depth and drift tolerance — down to the point where
// arithmetic actually happens.
//
// This package imports only pkg/qword. Both pkg/mesh and pkg/hal import it,
// keeping the dependency graph acyclic:
//
//	qword ← exec ← mesh
//	qword ← exec ← hal ← backends
package exec

import "github.com/JamesPagetButler/qbp-compute-unit/pkg/qword"

// ExecutionContext carries per-task execution parameters from the scheduler
// to the HAL and emulator. It is an allocation-free value type — pass by value.
type ExecutionContext struct {
	// TaskID identifies the mesh-scheduled task that owns this operation.
	// Empty string means no owning task; operations use DefaultContext.
	TaskID string

	// Width is the quaternion word width required for this task, as computed
	// by mesh.RequiredWidthFor(depth, tolerance).
	//
	// The HAL passes this to the emulator, which quantizes operands to the
	// declared width before executing arithmetic. This makes the precision
	// loss observable in norm-drift measurements, validating the scheduler's
	// width selection against empirical drift data.
	Width qword.Width
}

// DefaultContext returns an ExecutionContext for operations that have not been
// submitted to the mesh scheduler. Uses W64 (native float64 register precision).
//
// Use this at call sites that exist prior to scheduler integration, or in tests
// that only care about functional correctness rather than precision modelling.
func DefaultContext() ExecutionContext {
	return ExecutionContext{Width: qword.W64}
}
