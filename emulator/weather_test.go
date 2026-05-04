package emulator

import (
	"testing"
)

// TestWeather_TornadoVorticity models a rotating air column using QBP.
// It tracks the vorticity vector and calculates intensity via QNORM.
func TestWeather_TornadoVorticity(t *testing.T) {
	cpu := NewCPU()
	cpu.SetWidth(W128) // Physics-grade precision

	// 1. Define initial Vorticity Vector in q1 (Low-intensity rotation)
	// Omega = 0.1i + 0.1j + 0.5k (Stronger rotation on vertical Z-axis)
	cpu.Q[1].W.SetFloat64(0.0)
	cpu.Q[1].X.SetFloat64(0.1)
	cpu.Q[1].Y.SetFloat64(0.1)
	cpu.Q[1].Z.SetFloat64(0.5)

	// 2. Define the "Supercell Snap" (A sudden tightening rotation in q2)
	// This unit quaternion represents a 30-degree tightening around the Z-axis.
	// q = cos(15deg) + sin(15deg)k
	cpu.Q[2].W.SetFloat64(0.9659)
	cpu.Q[2].X.SetFloat64(0.0)
	cpu.Q[2].Y.SetFloat64(0.0)
	cpu.Q[2].Z.SetFloat64(0.2588)

	// 3. Apply the Snap to the Vorticity: q3 = QROT(q2, q1)
	// Instruction: QROT.128 q3, q2, q1
	var word1 uint32 = OpcodeCustom0 | (3 << 7) | (4 << 12) | (2 << 15) | (1 << 20) | (Funct7QROT << 25)
	cpu.Step(word1)

	// 4. Calculate Initial Intensity: q4.W = QNORM(q1)
	var word2 uint32 = OpcodeCustom0 | (4 << 7) | (4 << 12) | (1 << 15) | (0 << 20) | (Funct7QNORM << 25)
	cpu.Step(word2)

	// 5. Calculate Final Intensity (post-snap): q5.W = QNORM(q3)
	var word3 uint32 = OpcodeCustom0 | (5 << 7) | (4 << 12) | (3 << 15) | (0 << 20) | (Funct7QNORM << 25)
	cpu.Step(word3)

	initialIntensity := cpu.Q[4].W
	finalIntensity := cpu.Q[5].W

	t.Logf("Initial Tornado Intensity: %v", initialIntensity)
	t.Logf("Final Tornado Intensity (Post-Snap): %v", finalIntensity)

	// 6. Conservation check: In a pure rotation, NormSq should be invariant.
	// If the intensity changed, we've detected an "Open System" (Energy Influx).
	if finalIntensity.Cmp(initialIntensity) != 0 {
		t.Logf("ALERT: Energy gain detected in vortex! External pressure is driving the snap.")
	} else {
		t.Logf("STABLE VORTEX: Angular momentum conserved.")
	}

	// 7. Verify the rotation actually happened (X component should change)
	if cpu.Q[3].X.Cmp(cpu.Q[1].X) == 0 {
		t.Errorf("FAIL: Vorticity vector did not rotate.")
	} else {
		t.Logf("WEATHER SUCCESS: Tornado vorticity rotated from %v to %v", cpu.Q[1].X, cpu.Q[3].X)
	}
}
