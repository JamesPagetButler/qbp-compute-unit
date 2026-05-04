package emulator

import (
	"testing"
)

func TestCPU_QMUL_QW1024(t *testing.T) {
	cpu := NewCPU()
	
	// Set width to QW1024
	cpu.SetWidth(W1024)
	prec := cpu.GB.Precision()

	// Load q1 with a known value (0.5 + 0.5i + 0.5j + 0.5k)
	cpu.Q[1].W.SetFloat64(0.5)
	cpu.Q[1].X.SetFloat64(0.5)
	cpu.Q[1].Y.SetFloat64(0.5)
	cpu.Q[1].Z.SetFloat64(0.5)

	// Load q2 with identity (1.0 + 0i + 0j + 0k)
	cpu.Q[2].W.SetFloat64(1.0)
	cpu.Q[2].X.SetFloat64(0.0)
	cpu.Q[2].Y.SetFloat64(0.0)
	cpu.Q[2].Z.SetFloat64(0.0)

	// Encode QMUL.1024 q3, q1, q2
	// Opcode=0x0B, Rd=3, Funct3=7 (QW1024), Rs1=1, Rs2=2, Funct7=0 (QMUL)
	var word uint32 = OpcodeCustom0 | (3 << 7) | (7 << 12) | (1 << 15) | (2 << 20) | (Funct7QMUL << 25)

	err := cpu.Step(word)
	if err != nil {
		t.Fatalf("Step failed: %v", err)
	}

	// Verify q3 == q1 (since q2 is identity)
	if cpu.Q[3].W.Cmp(cpu.Q[1].W) != 0 {
		t.Errorf("Expected W=0.5, got %v", cpu.Q[3].W)
	}

	// Check precision of the result
	if cpu.Q[3].W.Prec() != prec {
		t.Errorf("Expected precision %d, got %d", prec, cpu.Q[3].W.Prec())
	}
}

func TestCPU_GearboxShift(t *testing.T) {
	cpu := NewCPU()
	
	// Default W64
	if cpu.GB.ActiveWidth != W64 {
		t.Errorf("Expected initial width W64, got %v", cpu.GB.ActiveWidth)
	}

	// Shift to W8 via instruction (QMUL.8 q0, q0, q0)
	var word uint32 = OpcodeCustom0 | (0 << 7) | (0 << 12) | (0 << 15) | (0 << 20) | (Funct7QMUL << 25)
	cpu.Step(word)

	if cpu.GB.ActiveWidth != W8 {
		t.Errorf("Expected width W8 after step, got %v", cpu.GB.ActiveWidth)
	}
}
