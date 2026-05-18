// Package emu implements a RISC-V instruction emulator for the QBP custom
// ISA extensions.
//
// This emulator serves three purposes:
//
//  1. ISA validation: verify that instruction encodings, operand formats,
//     and register allocation produce correct results before silicon exists.
//
//  2. Software development: the RunBackend in the HAL dispatches to this
//     emulator, allowing the full application stack (mesh scheduler, watchdog,
//     benchmarks) to run against the target ISA on Crawl/Walk hardware.
//
//  3. Performance estimation: Level 2 (pipeline) mode counts cycles per
//     instruction, models pipeline hazards, and predicts wall-clock time
//     for the target hardware. When the OpenMPW tape-out arrives, the first
//     validation test is: does the real chip match the emulator's predictions?
//
// RISC-V ENCODING:
//
// Custom QBP instructions use the RISC-V custom-0 opcode space (0x0B).
// Format: R-type (rd, rs1, rs2, funct3, funct7)
//
//	31       25 24   20 19   15 14  12 11    7 6      0
//	┌─────────┬───────┬───────┬──────┬───────┬────────┐
//	│ funct7  │  rs2  │  rs1  │ f3   │  rd   │ opcode │
//	└─────────┴───────┴───────┴──────┴───────┴────────┘
//
//	opcode = 0x0B (custom-0)
//	funct3 = width selector (000=QW8, 001=QW16, 010=QW32, 011=QW64, 100=QW128)
//	funct7 = operation (0=QMUL, 1=QROT, 2=OMAC, 3=FANO, 4=QNORM)
//
// REGISTER MAPPING:
//
// QBP operations use standard RISC-V floating-point registers (f0-f31).
// Wide quaternions span multiple registers:
//
//	QW64:  4 × f64 = registers f[rd]..f[rd+3]
//	QW128: 4 × f128 = register pairs (using Q extension)
//
// For QW8/QW16/QW32, values are packed into integer registers (x0-x31).
package emu

import (
	"fmt"
	"math"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/fano"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/qword"
)

// ─── Instruction encoding ──────────────────────────────────────────

// Opcode for QBP custom instructions (RISC-V custom-0 space).
const OpcodeCustom0 = 0x0B

// Operation codes (funct7 field).
type Op uint8

const (
	OpQMUL  Op = 0 // Quaternion Hamilton product
	OpQROT  Op = 1 // Quaternion rotation (q v q*)
	OpOMAC  Op = 2 // Octonionic multiply-accumulate
	OpFANO  Op = 3 // Fano plane lookup
	OpQNORM Op = 4 // Quaternion norm squared
)

func (op Op) String() string {
	switch op {
	case OpQMUL:
		return "QMUL"
	case OpQROT:
		return "QROT"
	case OpOMAC:
		return "OMAC"
	case OpFANO:
		return "FANO"
	case OpQNORM:
		return "QNORM"
	default:
		return fmt.Sprintf("OP(%d)", op)
	}
}

// WidthCode maps funct3 values to quaternion word widths.
type WidthCode uint8

const (
	WC8   WidthCode = 0 // funct3 = 000
	WC16  WidthCode = 1 // funct3 = 001
	WC32  WidthCode = 2 // funct3 = 010
	WC64  WidthCode = 3 // funct3 = 011
	WC128 WidthCode = 4 // funct3 = 100
)

func (wc WidthCode) ToWidth() qword.Width {
	switch wc {
	case WC8:
		return qword.W8
	case WC16:
		return qword.W16
	case WC32:
		return qword.W32
	case WC64:
		return qword.W64
	case WC128:
		return qword.W128
	default:
		return qword.W64
	}
}

func (wc WidthCode) String() string {
	return fmt.Sprintf(".%d", 8<<wc)
}

// Instruction represents a decoded QBP instruction.
type Instruction struct {
	Op    Op
	Width WidthCode
	Rd    uint8  // destination register
	Rs1   uint8  // source register 1
	Rs2   uint8  // source register 2
	Raw   uint32 // raw 32-bit encoding
}

// Encode produces the 32-bit RISC-V instruction word.
func (inst Instruction) Encode() uint32 {
	var w uint32
	w |= uint32(OpcodeCustom0)    // bits 6:0
	w |= uint32(inst.Rd) << 7     // bits 11:7
	w |= uint32(inst.Width) << 12 // bits 14:12 (funct3)
	w |= uint32(inst.Rs1) << 15   // bits 19:15
	w |= uint32(inst.Rs2) << 20   // bits 24:20
	w |= uint32(inst.Op) << 25    // bits 31:25 (funct7)
	return w
}

// Decode extracts a QBP instruction from a 32-bit word.
// Returns an error if the opcode doesn't match custom-0.
func Decode(word uint32) (Instruction, error) {
	opcode := word & 0x7F
	if opcode != OpcodeCustom0 {
		return Instruction{}, fmt.Errorf("not a QBP instruction: opcode=0x%02X", opcode)
	}
	return Instruction{
		Rd:    uint8((word >> 7) & 0x1F),
		Width: WidthCode((word >> 12) & 0x07),
		Rs1:   uint8((word >> 15) & 0x1F),
		Rs2:   uint8((word >> 20) & 0x1F),
		Op:    Op((word >> 25) & 0x7F),
		Raw:   word,
	}, nil
}

// Mnemonic returns the assembly-language representation.
func (inst Instruction) Mnemonic() string {
	return fmt.Sprintf("%s%s f%d, f%d, f%d",
		inst.Op, inst.Width, inst.Rd, inst.Rs1, inst.Rs2)
}

// ─── Register file ─────────────────────────────────────────────────

// RegisterFile holds the emulated floating-point register state.
// For QW64, each quaternion occupies 4 consecutive registers.
// For QW128, each component occupies 2 registers (128 bits each).
type RegisterFile struct {
	F [32]float64 // 32 × 64-bit FP registers (rv64 F/D extensions)

	// For QW128 emulation, we use a parallel high-bits array.
	// In real hardware, this would be the Q extension's 128-bit registers.
	// In emulation, we store each component as float64 (good enough for
	// semantic validation; pipeline simulation uses big.Float).
	FHi [32]float64 // high 64 bits of each 128-bit register (Q extension)
}

// LoadQuat loads a Quat into 4 consecutive registers starting at base.
func (rf *RegisterFile) LoadQuat(base uint8, q quat.Quat) {
	rf.F[base] = q.W
	rf.F[base+1] = q.X
	rf.F[base+2] = q.Y
	rf.F[base+3] = q.Z
}

// ReadQuat reads a Quat from 4 consecutive registers starting at base.
func (rf *RegisterFile) ReadQuat(base uint8) quat.Quat {
	return quat.Quat{
		W: rf.F[base],
		X: rf.F[base+1],
		Y: rf.F[base+2],
		Z: rf.F[base+3],
	}
}

// ─── Execution engine ──────────────────────────────────────────────

// Engine is the QBP instruction emulator.
type Engine struct {
	RF     RegisterFile
	Cycles int64 // total cycles consumed (Level 2)
	Ops    int64 // total instructions executed

	// Pipeline configuration (Level 2)
	Config PipelineConfig
}

// PipelineConfig defines the cycle costs for each instruction.
// These are design targets for the Run-phase RISC-V.
type PipelineConfig struct {
	QMULCycles  map[qword.Width]int // cycles per QMUL at each width
	QROTCycles  map[qword.Width]int // cycles per QROT
	OMACCycles  map[qword.Width]int // cycles per OMAC
	FANOCycles  int                 // FANO is width-independent
	QNORMCycles map[qword.Width]int // cycles per QNORM
	ClockMHz    float64             // target clock frequency
}

// DefaultPipelineConfig returns design-target cycle counts.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		QMULCycles: map[qword.Width]int{
			qword.W8:   1,
			qword.W16:  1,
			qword.W32:  1,
			qword.W64:  1,
			qword.W128: 2, // 128-bit needs two-cycle pipeline
			qword.W256: 4, // 256-bit needs four-cycle pipeline
		},
		QROTCycles: map[qword.Width]int{
			qword.W8: 2, qword.W16: 2, qword.W32: 2,
			qword.W64: 2, qword.W128: 4, qword.W256: 8,
		},
		OMACCycles: map[qword.Width]int{
			qword.W8: 1, qword.W16: 2, qword.W32: 2,
			qword.W64: 3, qword.W128: 6,
		},
		FANOCycles: 1, // ROM lookup, always 1 cycle
		QNORMCycles: map[qword.Width]int{
			qword.W8: 1, qword.W16: 1, qword.W32: 1,
			qword.W64: 1, qword.W128: 2,
		},
		ClockMHz: 100, // conservative for 130nm OpenMPW
	}
}

// NewEngine creates an emulator with the given pipeline configuration.
func NewEngine(cfg PipelineConfig) *Engine {
	return &Engine{Config: cfg}
}

// Execute runs one instruction and updates state.
// Returns the number of cycles consumed.
func (e *Engine) Execute(inst Instruction) (int, error) {
	w := inst.Width.ToWidth()
	var cycles int

	switch inst.Op {
	case OpQMUL:
		a := e.RF.ReadQuat(inst.Rs1)
		b := e.RF.ReadQuat(inst.Rs2)
		result := quat.Mul(a, b)
		e.RF.LoadQuat(inst.Rd, result)
		cycles = e.Config.QMULCycles[w]

	case OpQROT:
		q := e.RF.ReadQuat(inst.Rs1)
		v := e.RF.ReadQuat(inst.Rs2)
		result := quat.Rotate(q, v)
		e.RF.LoadQuat(inst.Rd, result)
		cycles = e.Config.QROTCycles[w]

	case OpFANO:
		// Rs1 and Rs2 hold basis indices in integer registers
		// For emulation, we use the float register's integer representation
		i := int(e.RF.F[inst.Rs1])
		j := int(e.RF.F[inst.Rs2])
		entry := fano.Lookup(i, j)
		e.RF.F[inst.Rd] = float64(entry.Index)
		e.RF.F[inst.Rd+1] = float64(entry.Sign)
		cycles = e.Config.FANOCycles

	case OpQNORM:
		q := e.RF.ReadQuat(inst.Rs1)
		nsq := quat.NormSq(q)
		e.RF.F[inst.Rd] = nsq
		cycles = e.Config.QNORMCycles[w]

	default:
		return 0, fmt.Errorf("unimplemented operation: %s", inst.Op)
	}

	if cycles == 0 {
		cycles = 1 // minimum 1 cycle
	}
	e.Cycles += int64(cycles)
	e.Ops++

	return cycles, nil
}

// ─── Performance estimation ────────────────────────────────────────

// EstimatedTime returns the predicted wall-clock time for the
// instructions executed so far, based on the pipeline config's clock rate.
func (e *Engine) EstimatedTime() float64 {
	if e.Config.ClockMHz == 0 {
		return 0
	}
	return float64(e.Cycles) / (e.Config.ClockMHz * 1e6)
}

// CPI returns the average cycles per instruction.
func (e *Engine) CPI() float64 {
	if e.Ops == 0 {
		return 0
	}
	return float64(e.Cycles) / float64(e.Ops)
}

// Report returns a summary of emulation statistics.
func (e *Engine) Report() string {
	return fmt.Sprintf(
		"QBP RISC-V Emulator Report:\n"+
			"  Instructions:    %d\n"+
			"  Total cycles:    %d\n"+
			"  Avg CPI:         %.2f\n"+
			"  Clock:           %.0f MHz\n"+
			"  Est. wall time:  %.6f sec\n"+
			"  Est. throughput: %.2f M instructions/sec\n",
		e.Ops, e.Cycles, e.CPI(),
		e.Config.ClockMHz,
		e.EstimatedTime(),
		float64(e.Ops)/math.Max(e.EstimatedTime(), 1e-15)/1e6,
	)
}

// ─── Go assembly directive helper ──────────────────────────────────

// GoAsmWord returns the Go assembly .word directive that would emit
// this instruction in a Go assembly file targeting RISC-V.
//
// Usage in a .s file:
//
//	TEXT ·qmul128(SB), NOSPLIT, $0
//	    WORD $0x0000040B  // QMUL.128 f0, f1, f2
//	    RET
func (inst Instruction) GoAsmWord() string {
	encoded := inst.Encode()
	return fmt.Sprintf("WORD $0x%08X  // %s", encoded, inst.Mnemonic())
}
