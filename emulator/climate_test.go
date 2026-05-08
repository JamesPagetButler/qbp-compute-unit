package emulator

import (
	"math/big"
	"testing"
)

// TestClimate_HopfLocale verifies pole-free navigation on a spherical surface.
// We move a "Wind Vector" from the Equator to the North Pole and verify
// that the QBP math handles the coordinate singularity (the Pole) effortlessly.
func TestClimate_HopfLocale(t *testing.T) {
	cpu := NewCPU()
	cpu.SetWidth(W256) // Use high precision to eliminate "Coordinate Tax" drift

	// 1. Define Reference Position (Equator at 0 Longitude) in q1
	// In our Hopf map, the vector part represents the surface point.
	// Equator Point = 1.0i + 0.0j + 0.0k
	cpu.Q[1].W.SetFloat64(0.0)
	cpu.Q[1].X.SetFloat64(1.0)
	cpu.Q[1].Y.SetFloat64(0.0)
	cpu.Q[1].Z.SetFloat64(0.0)

	// 2. Define "Eastward Wind" at the Equator in q2
	// East is the j-axis in our reference frame.
	cpu.Q[2].W.SetFloat64(0.0)
	cpu.Q[2].X.SetFloat64(0.0)
	cpu.Q[2].Y.SetFloat64(1.0)
	cpu.Q[2].Z.SetFloat64(0.0)

	// 3. Define the "Equator-to-Pole" Rotation in q3
	// This is a 90-degree rotation around the Y-axis.
	// q = cos(45deg) + sin(45deg)j = 0.7071... + 0.7071...j
	cpu.Q[3].W.SetFloat64(0.7071067811865476)
	cpu.Q[3].X.SetFloat64(0.0)
	cpu.Q[3].Y.SetFloat64(0.7071067811865476)
	cpu.Q[3].Z.SetFloat64(0.0)

	// 4. Rotate Position: q4 = QROT(q3, q1)
	// Equator (i) rotated 90deg around Y should become North Pole (k or -k)
	var word1 uint32 = OpcodeCustom0 | (4 << 7) | (5 << 12) | (3 << 15) | (1 << 20) | (Funct7QROT << 25)
	cpu.Step(word1)

	// 5. Rotate Wind Vector: q5 = QROT(q3, q2)
	// Eastward wind (j) rotated 90deg around Y remains j (still pointing East)
	var word2 uint32 = OpcodeCustom0 | (5 << 7) | (5 << 12) | (3 << 15) | (2 << 20) | (Funct7QROT << 25)
	cpu.Step(word2)

	t.Logf("Equator Position: %v", cpu.Q[1])
	t.Logf("Equator Wind:     %v", cpu.Q[2])
	t.Logf("North Pole Pos:   %v", cpu.Q[4])
	t.Logf("North Pole Wind:  %v", cpu.Q[5])

	// 6. Verification:
	// In a traditional grid, longitude becomes undefined at the Pole (1/cos(lat) -> infinity).
	// In QBP, the North Pole Pos (q4) should be precisely [0, 0, 0, -1] or [0, 0, 0, 1].
	eps := new(big.Float).SetFloat64(1e-15)
	absZ := new(big.Float).Abs(cpu.Q[4].Z)
	one := new(big.Float).SetFloat64(1.0)

	diff := new(big.Float).Sub(absZ, one)
	if new(big.Float).Abs(diff).Cmp(eps) > 0 {
		t.Errorf("POLE SINGULARITY DETECTED: North Pole Z-coord is %v, expected 1.0", cpu.Q[4].Z)
	} else {
		t.Logf("CLIMATE SUCCESS: Hopf Locale handled the North Pole without any numerical instability.")
	}

	// Verify the wind vector is still unit length
	nsq := new(big.Float).SetPrec(cpu.GB.Precision())
	cpu.GB.NormSq(nsq, &cpu.Q[5])
	if new(big.Float).Sub(nsq, one).Abs(new(big.Float).Sub(nsq, one)).Cmp(eps) > 0 {
		t.Errorf("WIND ENERGY DECAY: Wind vector norm changed: %v", nsq)
	} else {
		t.Logf("CONSERVATION SUCCESS: Wind energy density preserved across the coordinate transform.")
	}
}
