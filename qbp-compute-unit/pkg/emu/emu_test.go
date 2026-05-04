package emu

import (
	"testing"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
)

func TestEncodeDecodeRoundtrip(t *testing.T) {
	inst := Instruction{
		Op:    OpQMUL,
		Width: WC128,
		Rd:    0,
		Rs1:   4,
		Rs2:   8,
	}

	encoded := inst.Encode()
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Op != inst.Op {
		t.Errorf("op mismatch: got %v want %v", decoded.Op, inst.Op)
	}
	if decoded.Width != inst.Width {
		t.Errorf("width mismatch: got %v want %v", decoded.Width, inst.Width)
	}
	if decoded.Rd != inst.Rd {
		t.Errorf("rd mismatch: got %v want %v", decoded.Rd, inst.Rd)
	}
	if decoded.Rs1 != inst.Rs1 {
		t.Errorf("rs1 mismatch: got %v want %v", decoded.Rs1, inst.Rs1)
	}
	if decoded.Rs2 != inst.Rs2 {
		t.Errorf("rs2 mismatch: got %v want %v", decoded.Rs2, inst.Rs2)
	}
}

func TestEmulatorQMUL(t *testing.T) {
	e := NewEngine(DefaultPipelineConfig())

	// Load two unit quaternions
	q1 := quat.Normalize(quat.New(1, 2, 3, 4))
	q2 := quat.Normalize(quat.New(5, -1, 2, -3))
	e.RF.LoadQuat(4, q1)
	e.RF.LoadQuat(8, q2)

	// Execute QMUL.64 f0, f4, f8
	inst := Instruction{Op: OpQMUL, Width: WC64, Rd: 0, Rs1: 4, Rs2: 8}
	cycles, err := e.Execute(inst)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if cycles != 1 {
		t.Errorf("expected 1 cycle for QW64 QMUL, got %d", cycles)
	}

	// Verify result matches software
	expected := quat.Mul(q1, q2)
	got := e.RF.ReadQuat(0)
	if got != expected {
		t.Errorf("QMUL result mismatch:\n  got:    %+v\n  expect: %+v", got, expected)
	}

	// Verify cycle accounting
	if e.Ops != 1 {
		t.Errorf("expected 1 op, got %d", e.Ops)
	}
	if e.Cycles != 1 {
		t.Errorf("expected 1 cycle, got %d", e.Cycles)
	}
}

func TestEmulatorQMUL128TwoCycles(t *testing.T) {
	e := NewEngine(DefaultPipelineConfig())

	q1 := quat.Normalize(quat.New(1, 0, 0, 0))
	q2 := quat.Normalize(quat.New(0, 1, 0, 0))
	e.RF.LoadQuat(4, q1)
	e.RF.LoadQuat(8, q2)

	inst := Instruction{Op: OpQMUL, Width: WC128, Rd: 0, Rs1: 4, Rs2: 8}
	cycles, err := e.Execute(inst)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if cycles != 2 {
		t.Errorf("expected 2 cycles for QW128 QMUL, got %d", cycles)
	}
}

func TestEmulatorFANO(t *testing.T) {
	e := NewEngine(DefaultPipelineConfig())

	// e1 × e2 should give e3 with sign +1
	e.RF.F[4] = 1 // i index
	e.RF.F[8] = 2 // j index

	inst := Instruction{Op: OpFANO, Width: WC8, Rd: 0, Rs1: 4, Rs2: 8}
	cycles, err := e.Execute(inst)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if cycles != 1 {
		t.Errorf("expected 1 cycle for FANO, got %d", cycles)
	}

	resultIdx := int(e.RF.F[0])
	resultSign := int(e.RF.F[1])
	if resultIdx != 3 || resultSign != 1 {
		t.Errorf("FANO(1,2) = (idx=%d, sign=%d), expected (3, 1)", resultIdx, resultSign)
	}
}

func TestGoAsmWord(t *testing.T) {
	inst := Instruction{Op: OpQMUL, Width: WC128, Rd: 0, Rs1: 4, Rs2: 8}
	asm := inst.GoAsmWord()
	t.Logf("Go assembly: %s", asm)
	// Just verify it doesn't panic and produces something reasonable
	if len(asm) < 10 {
		t.Errorf("GoAsmWord output too short: %s", asm)
	}
}
