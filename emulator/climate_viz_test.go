package emulator

import (
	"math"
	"math/big"
	"testing"
)

// TestClimate_GlobalInit verifies the initialization of the climate sphere.
// It simulates a thermal gradient (warm equator, cold poles) and verifies
// the state across the Q-Mem manifold.
func TestClimate_GlobalInit(t *testing.T) {
	cpu := NewCPU()
	cpu.SetWidth(W64) // 64-bit is sufficient for init verification

	// 1. Populate the sphere at 10-degree resolution
	// resolution = 18 (180 deg / 10), lon = 36
	res := 18
	cpu.PopulateSphere(res)

	// 2. Apply a Thermal Gradient (Simple Sinusoidal)
	// Temp = 288.15 + 20 * cos(lat)
	for i := 0; i < res*res*2; i++ {
		node := cpu.GetClimateNode(i)

		// Get latitude from Z-component (z = sin(lat))
		// For verification, we'll just check if Z is near 1.0 (Pole) or 0.0 (Equator)
		z, _ := node.Pos.Z.Float64()

		// Very rough gradient: 300K at equator (Z=0), 250K at poles (Z=1)
		temp := 300.0 - (math.Abs(z) * 50.0)
		node.State.W.SetFloat64(temp)
	}

	// 3. Verify a few key points
	// Node 0 (South Pole, first in PopulateSphere loop)
	southPole := cpu.GetClimateNode(0)
	t.Logf("South Pole Position: %v, Temp: %v", southPole.Pos, southPole.State.W)

	// Equator Point (middle of the latitude loop)
	equatorIdx := (res / 2) * (res * 2)
	equator := cpu.GetClimateNode(equatorIdx)
	t.Logf("Equator Position:    %v, Temp: %v", equator.Pos, equator.State.W)

	// 4. Verification Check
	warm := new(big.Float).SetFloat64(295.0)
	if equator.State.W.Cmp(warm) < 0 {
		t.Errorf("FAIL: Equator temp too low: %v", equator.State.W)
	} else {
		t.Logf("CLIMATE VIZ SUCCESS: Global thermal gradient initialized on Q-Mem manifold.")
	}
}
