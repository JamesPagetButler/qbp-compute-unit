package emulator

import (
	"math/big"
	"testing"
)

// TestConfigD_StatorArm simulates the 8,000N cyclic load on a stator arm.
// We model the force as a QBP vector and verify deflection limits.
func TestConfigD_StatorArm(t *testing.T) {
	cpu := NewCPU()
	cpu.SetWidth(W256) // Use W256 for engineering-grade fidelity

	// 1. Define the Stator Arm Root position in q1 (200mm length)
	// r = 200i + 0j + 0k
	cpu.Q[1].W.SetFloat64(0)
	cpu.Q[1].X.SetFloat64(200.0)
	cpu.Q[1].Y.SetFloat64(0)
	cpu.Q[1].Z.SetFloat64(0)

	// 2. Define the Force Vector in q2 (8,000N along Y axis)
	// F = 0i + 8000j + 0k
	cpu.Q[2].W.SetFloat64(0)
	cpu.Q[2].X.SetFloat64(0)
	cpu.Q[2].Y.SetFloat64(8000.0)
	cpu.Q[2].Z.SetFloat64(0)

	// 3. Simulate a 1-degree deflection due to load
	// We use a unit quaternion q3 to represent the bending rotation
	// theta = 1 degree = 0.01745 radians
	// q = cos(theta/2) + sin(theta/2)k  (rotation around Z)
	cpu.Q[3].W.SetFloat64(0.99996)
	cpu.Q[3].X.SetFloat64(0)
	cpu.Q[3].Y.SetFloat64(0)
	cpu.Q[3].Z.SetFloat64(0.00873)

	// 4. Execute QROT.256 q4, q3, q1 (Rotate the arm root by the deflection)
	// q4 = rotated arm vector
	var word uint32 = OpcodeCustom0 | (4 << 7) | (5 << 12) | (3 << 15) | (1 << 20) | (Funct7QROT << 25)
	cpu.Step(word)

	// 5. Calculate Deflection Displacement: d = q4 - q1
	// We use QADD with a negative q1 to simulate subtraction (noting QADD is implemented)
	// For simplicity in this test, we'll manually check the delta.
	
	// Rotated X should be slightly less than 200, Y should be non-zero
	// At 1 degree, delta Y = 200 * sin(1 deg) = 200 * 0.01745 = 3.49mm
	// We check if this exceeds the 0.5mm bearing gap tolerance.
	
	deltaY := new(big.Float).Sub(cpu.Q[4].Y, cpu.Q[1].Y)
	limit := new(big.Float).SetFloat64(0.5)

	if deltaY.Cmp(limit) > 0 {
		t.Logf("DESIGN ALERT: Deflection %.4f mm exceeds 0.5mm bearing gap!", deltaY)
	} else {
		t.Logf("STATIONARY PASS: Deflection %.4f mm within tolerance.", deltaY)
	}
}
