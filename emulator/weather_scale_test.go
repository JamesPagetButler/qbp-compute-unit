package emulator

import (
	"math/big"
	"testing"
)

// TestWeather_CyclonicHolon models the interaction between tornado-scale
// vorticity and tropical cyclone intensity, inspired by AMS 2020 (Paper 367581).
func TestWeather_CyclonicHolon(t *testing.T) {
	cpu := NewCPU()
	cpu.SetWidth(W256) // High precision to detect scale-interaction deltas

	// 1. Define the Macro-Vortex (Tropical Cyclone base) in q10
	// Slow, broad rotation mainly on the Z-axis.
	cpu.Q[10].W.SetFloat64(0.0)
	cpu.Q[10].X.SetFloat64(0.01)
	cpu.Q[10].Y.SetFloat64(0.01)
	cpu.Q[10].Z.SetFloat64(0.2) // Primary cyclonic spin

	// 2. Define three Micro-Vortices (Tornado-scale) in q1, q2, q3
	// These are "Small but Intense" bursts of vorticity.
	vortices := []struct {
		reg     uint8
		x, y, z float64
	}{
		{1, 0.05, 0.0, 0.1},
		{2, 0.0, 0.05, 0.1},
		{3, -0.05, -0.05, 0.1},
	}

	for _, v := range vortices {
		cpu.Q[v.reg].W.SetFloat64(0.0)
		cpu.Q[v.reg].X.SetFloat64(v.x)
		cpu.Q[v.reg].Y.SetFloat64(v.y)
		cpu.Q[v.reg].Z.SetFloat64(v.z)
	}

	// 3. Create the "Vorticity Holon" (q11) - Grouping the micro-vortices
	// q11 = sum(q1, q2, q3)
	// Instruction: QADD q11, q1, q2 -> QADD q11, q11, q3
	add1 := uint32(OpcodeCustom0 | (11 << 7) | (5 << 12) | (1 << 15) | (2 << 20) | (Funct7QADD << 25))
	add2 := uint32(OpcodeCustom0 | (11 << 7) | (5 << 12) | (11 << 15) | (3 << 20) | (Funct7QADD << 25))
	cpu.Step(add1)
	cpu.Step(add2)

	// 4. Measure Intensity BEFORE and AFTER scale interaction
	// q12 = QNORM(q10) [Base Intensity]
	// q13 = QADD(q10, q11) [Interacted Vortex]
	// q14 = QNORM(q13) [Final Intensity]

	normBase := uint32(OpcodeCustom0 | (12 << 7) | (5 << 12) | (10 << 15) | (0 << 20) | (Funct7QNORM << 25))
	cpu.Step(normBase)

	interact := uint32(OpcodeCustom0 | (13 << 7) | (5 << 12) | (10 << 15) | (11 << 20) | (Funct7QADD << 25))
	cpu.Step(interact)

	normFinal := uint32(OpcodeCustom0 | (14 << 7) | (5 << 12) | (13 << 15) | (0 << 20) | (Funct7QNORM << 25))
	cpu.Step(normFinal)

	baseInt := cpu.Q[12].W
	finalInt := cpu.Q[14].W

	t.Logf("Base Cyclone Intensity:  %v", baseInt)
	t.Logf("Interacted Intensity:    %v", finalInt)

	// 5. Calculate Intensity Delta (The AMS Finding)
	delta := new(big.Float).Sub(finalInt, baseInt)
	t.Logf("Vorticity-Driven Gain:  %v", delta)

	// AMS Result: Small-scale vorticity contributes to storm-scale intensification.
	if delta.Cmp(new(big.Float).SetFloat64(0.0)) > 0 {
		t.Logf("AMS CONSISTENT: Micro-vorticity Holon successfully intensified the Macro-vortex.")
	} else {
		t.Errorf("FAIL: Scale interaction did not show intensification.")
	}
}
