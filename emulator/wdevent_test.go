package emulator

import (
	"testing"
)

// Helper to construct synthetic Xqbp instructions
func buildInstForTest(funct7, rs2, rs1, funct3, rd uint32) uint32 {
	return 11 | (rd << 7) | (funct3 << 12) | (rs1 << 15) | (rs2 << 20) | (funct7 << 25)
}

func TestWatchdog_PassiveEmissionCount(t *testing.T) {
	cpu := NewCPU()

	ops := []uint32{
		buildInstForTest(Funct7QMUL, 3, 2, 3, 1),
		buildInstForTest(Funct7QADD, 3, 2, 3, 1),
		buildInstForTest(Funct7QROT, 3, 2, 3, 1),
		buildInstForTest(Funct7QCONJ, 0, 2, 3, 1),
		buildInstForTest(Funct7QNORM, 0, 2, 3, 1),
		buildInstForTest(Funct7FANO, 3, 2, 3, 1),
	}

	for _, word := range ops {
		if err := cpu.Step(word); err != nil {
			t.Fatalf("unexpected error during Step: %v", err)
		}
	}

	// Drain channel and count
	count := 0
	for {
		select {
		case <-cpu.WatchdogChan:
			count++
		default:
			goto CheckCount
		}
	}
CheckCount:
	if count != len(ops) {
		t.Fatalf("expected %d events, got %d", len(ops), count)
	}
}

func TestWatchdog_W64_AllOps(t *testing.T) {
	cpu := NewCPU()
	tests := []struct {
		name   string
		funct7 uint32
	}{
		{"QMUL", Funct7QMUL},
		{"QADD", Funct7QADD},
		{"QROT", Funct7QROT},
		{"QCONJ", Funct7QCONJ},
		{"QNORM", Funct7QNORM},
		{"FANO", Funct7FANO},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear channel
			for len(cpu.WatchdogChan) > 0 {
				<-cpu.WatchdogChan
			}

			word := buildInstForTest(tc.funct7, 3, 2, 3, 1) // W64
			if err := cpu.Step(word); err != nil {
				t.Fatalf("unexpected error during Step: %v", err)
			}

			select {
			case evt := <-cpu.WatchdogChan:
				if evt.Op != Opcode(tc.funct7) {
					t.Errorf("expected Op %v, got %v", tc.funct7, evt.Op)
				}
				if evt.Port != PortSSCI {
					t.Errorf("expected PortSSCI, got %v", evt.Port)
				}
			default:
				t.Fatalf("expected WDEvent to be emitted, but channel was empty")
			}
		})
	}
}

func TestWatchdog_W128_AllOps(t *testing.T) {
	cpu := NewCPU()
	tests := []struct {
		name   string
		funct7 uint32
	}{
		{"QMUL128", Funct7QMUL},
		{"QADD128", Funct7QADD},
		{"QROT128", Funct7QROT},
		{"QCONJ128", Funct7QCONJ},
		{"QNORM128", Funct7QNORM},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for len(cpu.WatchdogChan) > 0 {
				<-cpu.WatchdogChan
			}

			word := buildInstForTest(tc.funct7, 3, 2, 4, 1) // W128
			if err := cpu.Step(word); err != nil {
				t.Fatalf("unexpected error during Step: %v", err)
			}

			select {
			case evt := <-cpu.WatchdogChan:
				if evt.Op != Opcode(tc.funct7) {
					t.Errorf("expected Op %v, got %v", tc.funct7, evt.Op)
				}
			default:
				t.Fatalf("expected WDEvent to be emitted for W128, but channel was empty")
			}
		})
	}
}
