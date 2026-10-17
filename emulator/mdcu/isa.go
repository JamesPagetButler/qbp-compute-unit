package mdcu

import "fmt"

// Xqbp v2 instruction set (custom-0, opcode 0x0B). Funct7 values follow the
// Multi-Dimensional Substrate doc where specified; others are spike-assigned.
type Op uint8

const (
	OpOMUL     Op = 0x06
	OpQSPLIT   Op = 0x07
	OpQJOIN    Op = 0x08
	OpQASSOC   Op = 0x09
	OpZDCHK    Op = 0x0A
	OpSMUL     Op = 0x0B // spike-assigned
	OpQSPLITLF Op = 0x0C // QSPLIT.LEAF (spike-assigned)
	OpQJOINLF  Op = 0x0D // QJOIN.LEAF  (spike-assigned)
	OpQLIFT    Op = 0x0E // spike-assigned
	OpQPROJ    Op = 0x0F // spike-assigned
	OpQWIDEN   Op = 0x10 // spike-assigned
	OpQNARROW  Op = 0x11 // spike-assigned
	OpLEAFCHK  Op = 0x12 // leaf-escape residual vs CSR-locked frame (spike-assigned)
)

// Tariff is the abstract ISA cycle cost (host-independent), following the
// existing emulator/isa.go convention (1 cycle per fully-parallel datapath op).
// Values for the CD-MAC ops assume the Silicon-doc datapath (64/256 parallel
// multipliers + log₂N adder tree pipelined to 1 issue/cycle). These are MODEL
// numbers, reported in their own column — never merged with host ns.
var Tariff = map[Op]int{
	OpOMUL: 1, OpSMUL: 1, // CD-MAC: N² multipliers in parallel
	OpQASSOC: 2, // two dependent product levels ((xy)z ∥ x(yz))
	OpZDCHK:  1,
	OpQSPLIT: 1, OpQJOIN: 1, OpQLIFT: 1, OpQPROJ: 1, // wiring + norm² for ledger
	OpQWIDEN: 1, OpQNARROW: 1,
	OpQSPLITLF: 2, OpQJOINLF: 2, // norm, dot, norm (2 reduction levels)
	OpLEAFCHK: 2,
}

// Instr is a decoded Xqbp v2 instruction. Register indices address the
// tier-grouped files (O: 0..15, S: 0..7) of RegFile.
type Instr struct {
	Op           Op
	Rd, Rs1, Rs2 int
	Rs3          int // QASSOC third operand
	Frame        int // QSPLIT/QJOIN Fano frame (0 = canonical)
}

// CPU is the spike Xqbp v2 core (sequential issue).
type CPU struct {
	CSR    CSR
	Reg    RegFile
	Led    Ledger
	Leaf   LeafFrame // leaf CSRs (locked at QSPLIT.LEAF)
	LeafQ  Leaf      // leaf state register
	Cycles uint64
	Issued map[Op]uint64
}

// NewCPU returns a CPU at tier 𝕆, QW64, D=1.
func NewCPU() *CPU {
	return &CPU{CSR: CSR{QW: W64, QTier: TierO, QDim: 1}, Issued: map[Op]uint64{}}
}

// Exec executes one instruction.
func (c *CPU) Exec(in Instr) error {
	switch in.Op {
	case OpOMUL:
		x, y := c.Reg.O(in.Rs1), c.Reg.O(in.Rs2)
		c.Reg.SetO(in.Rd, OMulCDMAC(&x, &y))
	case OpSMUL:
		x, y := c.Reg.S(in.Rs1), c.Reg.S(in.Rs2)
		z := SMulCDMAC(&x, &y)
		c.Reg.SetS(in.Rd, z)
		if c.CSR.QMode&0x80 != 0 {
			if ok, _, _ := ZDCHK(&z, 1e-12); ok {
				c.CSR.Flags |= FlagZD
			}
		}
	case OpQASSOC:
		x, y, z := c.Reg.O(in.Rs1), c.Reg.O(in.Rs2), c.Reg.O(in.Rs3)
		a := QAssoc(c.CSR.QTier, x[:], y[:], z[:])
		var o [8]float64
		copy(o[:], a)
		c.Reg.SetO(in.Rd, o)
	case OpQSPLIT: // O[rs1] → q[rd] (bulk), q[rd+1] (fiber)
		x := c.Reg.O(in.Rs1)
		b, f := QSplit(x[:], in.Frame)
		copy(c.Reg.Q[in.Rd][:], b)
		copy(c.Reg.Q[in.Rd+1][:], f)
	case OpQJOIN: // q[rs1] (bulk), q[rs2] (fiber) → O[rd]
		z := QJoin(c.Reg.Q[in.Rs1][:], c.Reg.Q[in.Rs2][:], in.Frame)
		var o [8]float64
		copy(o[:], z)
		c.Reg.SetO(in.Rd, o)
	case OpZDCHK:
		x := c.Reg.S(in.Rs1)
		if ok, _, _ := ZDCHK(&x, 1e-12); ok {
			c.CSR.Flags |= FlagZD
		} else {
			c.CSR.Flags &^= FlagZD
		}
	case OpQSPLITLF:
		s := c.Reg.S(in.Rs1)
		c.LeafQ, c.Leaf = QSplitLeaf(&s)
	case OpQJOINLF:
		c.Reg.SetS(in.Rd, QJoinLeaf(c.LeafQ, &c.Leaf))
	case OpLEAFCHK:
		s := c.Reg.S(in.Rs1)
		if LeafEscape(&s, &c.Leaf) > 1e-24 {
			c.CSR.Flags |= FlagSeam
		}
	default:
		return fmt.Errorf("mdcu: op 0x%02x not dispatchable on the register core (width/tier-vector ops are functional in the spike)", in.Op)
	}
	c.Cycles += uint64(Tariff[in.Op])
	c.Issued[in.Op]++
	return nil
}
