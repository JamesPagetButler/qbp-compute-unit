package emulator

import (
	"testing"
)

func TestWatchdog_PassiveEmission(t *testing.T) {
	cpu := NewCPU()
	
	// Create a synthetic QMUL instruction (W64)
	// opcode = 0x0B, rd = 1, funct3 = 3 (W64), rs1 = 2, rs2 = 3, funct7 = Funct7QMUL (0)
	word := uint32(11 | (1 << 7) | (3 << 12) | (2 << 15) | (3 << 20) | (Funct7QMUL << 25))

	err := cpu.Step(word)
	if err != nil {
		t.Fatalf("unexpected error during Step: %v", err)
	}

	// Verify an event was emitted to the WatchdogChan
	select {
	case evt := <-cpu.WatchdogChan:
		if evt.Op != Funct7QMUL {
			t.Errorf("expected Op %v, got %v", Funct7QMUL, evt.Op)
		}
		if evt.Port != PortSSCI {
			t.Errorf("expected PortSSCI, got %v", evt.Port)
		}
		if evt.Cycle != cpu.Cycles {
			t.Errorf("expected Cycle %v, got %v", cpu.Cycles, evt.Cycle)
		}
		if evt.ZDClass != NotZD {
			t.Errorf("expected ZDClass NotZD, got %v", evt.ZDClass)
		}
	default:
		t.Fatalf("expected WDEvent to be emitted, but channel was empty")
	}
}
