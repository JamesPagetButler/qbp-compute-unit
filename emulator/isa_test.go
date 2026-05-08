package emulator

import (
	"math/big"
	"testing"
)

// TestRotation120 verifies that a rotation of 120 degrees around the axis (1,1,1)
// cyclically permutes the i, j, k axes.
// q = (1 + i + j + k) / 2
// p = ai + bj + ck  => p' = ci + aj + bk
func TestRotation120(t *testing.T) {
	cpu := NewCPU()
	cpu.SetWidth(W256) // Use W256 to test the high-precision big.Float Gearbox path

	// 1. Setup the rotation quaternion q in register q1
	// q = 0.5 + 0.5i + 0.5j + 0.5k
	cpu.Q[1].W.SetFloat64(0.5)
	cpu.Q[1].X.SetFloat64(0.5)
	cpu.Q[1].Y.SetFloat64(0.5)
	cpu.Q[1].Z.SetFloat64(0.5)

	// 2. Setup the vector p in register q2
	// p = 1.0i + 2.0j + 3.0k  (a=1, b=2, c=3)
	cpu.Q[2].W.SetFloat64(0.0)
	cpu.Q[2].X.SetFloat64(1.0)
	cpu.Q[2].Y.SetFloat64(2.0)
	cpu.Q[2].Z.SetFloat64(3.0)

	// 3. Execute QROT.256 q3, q1, q2
	// p' = q * p * conj(q)
	// Expected p' = 3.0i + 1.0j + 2.0k (cyclic permutation: x=c, y=a, z=b)
	var word uint32 = OpcodeCustom0 | (3 << 7) | (5 << 12) | (1 << 15) | (2 << 20) | (Funct7QROT << 25)

	if err := cpu.Step(word); err != nil {
		t.Fatalf("QROT failed: %v", err)
	}

	// 4. Verify results
	eps := new(big.Float).SetFloat64(1e-15)

	check := func(name string, got *big.Float, want float64) {
		wantF := new(big.Float).SetFloat64(want)
		diff := new(big.Float).Sub(got, wantF)
		diff.Abs(diff)
		if diff.Cmp(eps) > 0 {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}

	check("p'.W", cpu.Q[3].W, 0.0)
	check("p'.X", cpu.Q[3].X, 3.0) // c
	check("p'.Y", cpu.Q[3].Y, 1.0) // a
	check("p'.Z", cpu.Q[3].Z, 2.0) // b
}

// TestConjugation_FastPath_W64 exercises the Q64 (float64) fast-path that
// the isa.go dispatch selects when ActiveWidth <= W64. On amd64 with
// AVX+FMA this routes through qconj64AVX; otherwise through
// qconj64Scalar.
func TestConjugation_FastPath_W64(t *testing.T) {
	cpu := NewCPU()
	cpu.SetWidth(W64)

	cpu.Q64[1] = QW64{1.0, 2.0, 3.0, 4.0}

	// QCONJ q2, q1 with funct3=3 (W64)
	var word uint32 = OpcodeCustom0 | (2 << 7) | (3 << 12) | (1 << 15) | (0 << 20) | (Funct7QCONJ << 25)
	if err := cpu.Step(word); err != nil {
		t.Fatalf("QCONJ failed: %v", err)
	}

	want := QW64{1.0, -2.0, -3.0, -4.0}
	if cpu.Q64[2] != want {
		t.Errorf("Q64 path: got %v, want %v", cpu.Q64[2], want)
	}
}

// TestConjugation_HighPrec_W256 exercises the big.Float path used when
// ActiveWidth > W128 (e.g. for sleep-cycle and constitutional verification).
func TestConjugation_HighPrec_W256(t *testing.T) {
	cpu := NewCPU()
	cpu.SetWidth(W256)

	cpu.Q[1].W.SetFloat64(1.0)
	cpu.Q[1].X.SetFloat64(2.0)
	cpu.Q[1].Y.SetFloat64(3.0)
	cpu.Q[1].Z.SetFloat64(4.0)

	// QCONJ q2, q1 with funct3=5 (W256)
	var word uint32 = OpcodeCustom0 | (2 << 7) | (5 << 12) | (1 << 15) | (0 << 20) | (Funct7QCONJ << 25)
	if err := cpu.Step(word); err != nil {
		t.Fatalf("QCONJ failed: %v", err)
	}

	if cpu.Q[2].W.Cmp(cpu.Q[1].W) != 0 ||
		cpu.Q[2].X.Cmp(new(big.Float).Neg(cpu.Q[1].X)) != 0 ||
		cpu.Q[2].Y.Cmp(new(big.Float).Neg(cpu.Q[1].Y)) != 0 ||
		cpu.Q[2].Z.Cmp(new(big.Float).Neg(cpu.Q[1].Z)) != 0 {
		t.Errorf("big.Float path: got %v", cpu.Q[2])
	}
}
