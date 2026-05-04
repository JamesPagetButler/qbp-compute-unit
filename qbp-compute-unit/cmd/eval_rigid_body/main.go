package main

import (
	"fmt"
	"math"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

// Body represents a rigid body in 3D space.
type Body struct {
	Name        string
	Orientation quat.Quat // Current orientation
	AngularVel  quat.Quat // Angular velocity (pure imaginary quaternion)
	Mass        float64
}

func NewBody(name string, mass float64) *Body {
	return &Body{
		Name:        name,
		Orientation: quat.Identity(),
		AngularVel:  quat.Pure(0, 0, 0),
		Mass:        mass,
	}
}

// Update performs a single time step integration.
// q(t+dt) = q(t) + 0.5 * omega * q(t) * dt
func (b *Body) Update(dt float64) {
	// Derivative: dq/dt = 0.5 * omega * q
	derivative := quat.Scale(0.5, quat.Mul(b.AngularVel, b.Orientation))
	
	// Euler integration
	b.Orientation = quat.MulAccum(b.Orientation, derivative, quat.Scalar(dt))
	
	// Re-normalize to prevent drift (as per QBP requirements)
	b.Orientation = quat.Normalize(b.Orientation)
}

func main() {
	fmt.Println("========================================================================")
	fmt.Println("QBP EVALUATION: 3D RIGID BODY DYNAMICS (PHASE 2)")
	fmt.Println("Testing QROT and Hamilton Product in a dynamic simulation.")
	fmt.Println("========================================================================")

	// Simulation parameters
	dt := 0.01
	steps := 1000
	
	// Create a body rotating around the Z-axis at 1 rad/s
	body := NewBody("TestGyro", 1.0)
	body.AngularVel = quat.Pure(0, 0, 1.0)

	fmt.Printf("Initial Orientation: %+v\n", body.Orientation)
	fmt.Printf("Angular Velocity:    %+v\n", body.AngularVel)
	fmt.Println("\nRunning 1000 steps (10 seconds simulation)...")

	start := time.Now()
	for i := 0; i < steps; i++ {
		body.Update(dt)
	}
	elapsed := time.Since(start)

	fmt.Printf("Final Orientation:   %+v\n", body.Orientation)
	fmt.Printf("Final Norm:          %f\n", quat.Norm(body.Orientation))
	fmt.Printf("Wall time:           %v\n", elapsed)

	// Verification: A 1 rad/s rotation for 10s should be 10 radians.
	// 10 radians = 10 * (180/pi) = 572.95 degrees.
	// 572.95 % 360 = 212.95 degrees.
	// Rotation of 212.95 degrees around Z-axis.
	// Expected w = cos(theta/2) = cos(10/2) = cos(5) = 0.28366
	// Expected z = sin(theta/2) = sin(5) = -0.95892
	
	expectedW := math.Cos(5.0)
	expectedZ := math.Sin(5.0)

	fmt.Printf("\nExpected (analytical): W: %.5f, Z: %.5f\n", expectedW, expectedZ)
	
	diffW := math.Abs(body.Orientation.W - expectedW)
	diffZ := math.Abs(body.Orientation.Z - expectedZ)

	fmt.Printf("Accuracy Delta:      W: %.5e, Z: %.5e\n", diffW, diffZ)

	if diffW < 1e-4 && diffZ < 1e-4 {
		fmt.Println("\nVERDICT: QBP Rigid Body Integration PASS")
	} else {
		fmt.Println("\nVERDICT: QBP Rigid Body Integration FAIL (Drift too high)")
	}
	fmt.Println("========================================================================")
}
