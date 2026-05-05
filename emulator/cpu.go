package emulator

import (
	"fmt"
	"sync/atomic"
)

// CPU represents the architectural state of the QBP Compute Unit.
type CPU struct {
	// Standard RISC-V state
	X [32]uint64 // Integer registers
	PC uint64    // Program Counter

	// QBP Extension state
	Q64  [32]QW64  // Hardware-accelerated fast path registers (W8-W64)
	Q128 [32]QW128 // Hardware-accelerated fast path registers (W128 DD)
	Q    [32]QWord // High-precision registers (W256+)
	
	// Q-Mem (Quaternion Memory) - Scalable storage for MuninnDB/Climate nodes
	Memory []QWord

	// Watchdog channel for passive event emission (M0)
	WatchdogChan chan WDEvent
	
	// Atomic counter for observability when the channel drops events
	WatchdogDropCount uint64

	// Gearbox for precision scaling
	GB *Gearbox

	// Stats
	Instructions uint64
	Cycles       uint64
}

// NewCPU creates a new CPU initialized at QW64 (default).
func NewCPU() *CPU {
	cpu := &CPU{
		GB:           NewGearbox(),
		WatchdogChan: make(chan WDEvent, 1024), // Buffered to prevent blocking in M0
	}
	// Initialize Q registers with current gearbox precision
	prec := cpu.GB.Precision()
	for i := range cpu.Q {
		cpu.Q[i] = NewQWord(prec)
	}
	return cpu
}

// SetWidth updates the Gearbox width and re-scales all Q registers.
// This implements the Dynamic Width Transition mentioned in the spec.
func (c *CPU) SetWidth(w Width) {
	c.GB.ActiveWidth = w
	prec := c.GB.Precision()
	for i := range c.Q {
		c.Q[i].SetPrec(prec)
	}
}

// Reset clears all registers and the PC.
func (c *CPU) Reset() {
	c.PC = 0
	c.Instructions = 0
	c.Cycles = 0
	for i := range c.X {
		c.X[i] = 0
	}
	prec := c.GB.Precision()
	for i := range c.Q {
		c.Q[i] = NewQWord(prec)
	}
}

// Step executes a single 32-bit instruction word.
// (Already defined in isa.go, adding Run here for batch execution)

// Run executes a program (slice of instruction words).
func (c *CPU) Run(program []uint32) error {
	for int(c.PC/4) < len(program) {
		word := program[c.PC/4]
		err := c.Step(word)
		if err != nil {
			return err
		}
		c.PC += 4
	}
	return nil
}

// DumpStatus returns a string representation of the current CPU state.
func (c *CPU) DumpStatus() string {
	return fmt.Sprintf("PC: 0x%X, Width: %v, Instrs: %d, Cycles: %d", 
		c.PC, c.GB.ActiveWidth, c.Instructions, c.Cycles)
}

// emitWDEvent performs a non-blocking send of a watchdog event.
func (c *CPU) emitWDEvent(evt WDEvent) {
	select {
	case c.WatchdogChan <- evt:
		// Emitted successfully
	default:
		// Channel full, drop event and increment observability counter
		atomic.AddUint64(&c.WatchdogDropCount, 1)
	}
}
