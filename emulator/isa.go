package emulator

import (
	"fmt"
)

// Opcode constants
const (
	OpcodeCustom0 = 0x0B
)

// Funct7 constants
const (
	Funct7QMUL  = 0
	Funct7QADD  = 1
	Funct7QROT  = 2
	Funct7FANO  = 3
	Funct7QCONJ = 4
	Funct7QNORM = 5
)

// Instruction represents a decoded RISC-V instruction.
type Instruction struct {
	Opcode uint8
	Rd     uint8
	Funct3 uint8
	Rs1    uint8
	Rs2    uint8
	Funct7 uint8
}

// Decode extracts fields from a 32-bit RISC-V instruction word.
func Decode(word uint32) Instruction {
	return Instruction{
		Opcode: uint8(word & 0x7F),
		Rd:     uint8((word >> 7) & 0x1F),
		Funct3: uint8((word >> 12) & 0x07),
		Rs1:    uint8((word >> 15) & 0x1F),
		Rs2:    uint8((word >> 20) & 0x1F),
		Funct7: uint8((word >> 25) & 0x7F),
	}
}

// Step executes a single 32-bit instruction word.
func (c *CPU) Step(word uint32) error {
	inst := Decode(word)
	c.Instructions++

	if inst.Opcode != OpcodeCustom0 {
		return fmt.Errorf("unsupported opcode: 0x%X", inst.Opcode)
	}

	// Update width based on Funct3 (Gearbox shift)
	switch inst.Funct3 {
	case 0:
		c.SetWidth(W8)
	case 1:
		c.SetWidth(W16)
	case 2:
		c.SetWidth(W32)
	case 3:
		c.SetWidth(W64)
	case 4:
		c.SetWidth(W128)
	case 5:
		c.SetWidth(W256)
	case 6:
		c.SetWidth(W512)
	case 7:
		c.SetWidth(W1024)
	}

	switch inst.Funct7 {
	case Funct7QMUL:
		if c.GB.ActiveWidth <= W64 {
			qmul64(&c.Q64[inst.Rd], &c.Q64[inst.Rs1], &c.Q64[inst.Rs2])
		} else if c.GB.ActiveWidth == W128 {
			qmul128(&c.Q128[inst.Rd], &c.Q128[inst.Rs1], &c.Q128[inst.Rs2])
		} else {
			// q[rd] = q[rs1] * q[rs2]
			c.GB.Mul(&c.Q[inst.Rd], &c.Q[inst.Rs1], &c.Q[inst.Rs2])
		}
		c.Cycles += 1

	case Funct7QADD:
		if c.GB.ActiveWidth <= W64 {
			qadd64(&c.Q64[inst.Rd], &c.Q64[inst.Rs1], &c.Q64[inst.Rs2])
		} else if c.GB.ActiveWidth == W128 {
			qadd128(&c.Q128[inst.Rd], &c.Q128[inst.Rs1], &c.Q128[inst.Rs2])
		} else {
			// Simple component-wise addition
			prec := c.GB.Precision()
			c.Q[inst.Rd].W.Add(c.Q[inst.Rs1].W, c.Q[inst.Rs2].W).SetPrec(prec)
			c.Q[inst.Rd].X.Add(c.Q[inst.Rs1].X, c.Q[inst.Rs2].X).SetPrec(prec)
			c.Q[inst.Rd].Y.Add(c.Q[inst.Rs1].Y, c.Q[inst.Rs2].Y).SetPrec(prec)
			c.Q[inst.Rd].Z.Add(c.Q[inst.Rs1].Z, c.Q[inst.Rs2].Z).SetPrec(prec)
		}
		c.Cycles += 1

	case Funct7QROT:
		if c.GB.ActiveWidth <= W64 {
			qrot64(&c.Q64[inst.Rd], &c.Q64[inst.Rs1], &c.Q64[inst.Rs2])
		} else if c.GB.ActiveWidth == W128 {
			qrot128(&c.Q128[inst.Rd], &c.Q128[inst.Rs1], &c.Q128[inst.Rs2])
		} else {
			// q[rd] = q[rs1] * q[rs2] * conj(q[rs1])
			c.GB.Rotate(&c.Q[inst.Rd], &c.Q[inst.Rs1], &c.Q[inst.Rs2])
		}
		c.Cycles += 2 // Two QMULs

	case Funct7QCONJ:
		if c.GB.ActiveWidth <= W64 {
			qconj64(&c.Q64[inst.Rd], &c.Q64[inst.Rs1])
		} else if c.GB.ActiveWidth == W128 {
			qconj128(&c.Q128[inst.Rd], &c.Q128[inst.Rs1])
		} else {
			// q[rd] = conj(q[rs1])
			c.GB.Conj(&c.Q[inst.Rd], &c.Q[inst.Rs1])
		}
		c.Cycles += 1

	case Funct7QNORM:
		if c.GB.ActiveWidth <= W64 {
			var normSq float64
			qnorm64(&normSq, &c.Q64[inst.Rs1])
			c.Q64[inst.Rd][0] = normSq
			c.Q64[inst.Rd][1] = 0
			c.Q64[inst.Rd][2] = 0
			c.Q64[inst.Rd][3] = 0
		} else if c.GB.ActiveWidth == W128 {
			qnorm128(&c.Q128[inst.Rd], &c.Q128[inst.Rs1])
		} else {
			// q[rd].W = normsq(q[rs1])
			c.GB.NormSq(c.Q[inst.Rd].W, &c.Q[inst.Rs1])
			c.Q[inst.Rd].X.SetFloat64(0)
			c.Q[inst.Rd].Y.SetFloat64(0)
			c.Q[inst.Rd].Z.SetFloat64(0)
		}
		c.Cycles += 1

	case Funct7FANO:
		// Basis indices are stored in X registers for FANO
		i := int(c.X[inst.Rs1])
		j := int(c.X[inst.Rs2])
		entry := c.GB.FanoLookup(i, j)
		c.X[inst.Rd] = uint64(entry.Index)
		// Sign is stored in the next register or a status bit
		c.X[(inst.Rd+1)%32] = uint64(int64(entry.Sign))
		c.Cycles += 1

	default:
		return fmt.Errorf("unimplemented funct7: %d", inst.Funct7)
	}

	// Passive emission for M0: capture basic event data and emit
	evt := WDEvent{
		Cycle:     c.Cycles,
		Op:        Opcode(inst.Funct7),
		Port:      PortSSCI,
		ZDClass:   NotZD,
		AlgebraID: 0, // TODO(M1): populate from c.csr.AMODE
	}

	if inst.Funct7 == Funct7FANO {
		evt.FanoIndex = uint8(c.X[inst.Rd])
		evt.SignBit = int64(c.X[(inst.Rd+1)%32]) == -1
	}

	c.emitWDEvent(evt)

	return nil
}
