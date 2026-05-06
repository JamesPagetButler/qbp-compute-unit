package emulator

import (
	"testing"
)

// Helper to construct synthetic Xqbp instructions
func buildInst(funct7, rs2, rs1, funct3, rd uint32) uint32 {
	// opcode = OpcodeCustom0 (0x0B = 11)
	return 11 | (rd << 7) | (funct3 << 12) | (rs1 << 15) | (rs2 << 20) | (funct7 << 25)
}

func BenchmarkCPU_QMUL(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7QMUL, 3, 2, 3, 1) // W64 QMUL

	cpu.Q64[2][0] = 1.0 // W
	cpu.Q64[3][0] = 2.0 // W

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}

func BenchmarkCPU_QADD(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7QADD, 3, 2, 3, 1) // W64 QADD

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}

func BenchmarkCPU_QROT(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7QROT, 3, 2, 3, 1) // W64 QROT

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}

func BenchmarkCPU_QCONJ(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7QCONJ, 0, 2, 3, 1) // W64 QCONJ (rs2 is ignored)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}

func BenchmarkCPU_QNORM(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7QNORM, 0, 2, 3, 1) // W64 QNORM (rs2 is ignored)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}

func BenchmarkCPU_QMUL128(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7QMUL, 3, 2, 4, 1) // W128 QMUL

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}

func BenchmarkCPU_QADD128(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7QADD, 3, 2, 4, 1) // W128 QADD

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}

func BenchmarkCPU_QROT128(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7QROT, 3, 2, 4, 1) // W128 QROT

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}

func BenchmarkCPU_QCONJ128(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7QCONJ, 0, 2, 4, 1) // W128 QCONJ

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}

func BenchmarkCPU_QNORM128(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7QNORM, 0, 2, 4, 1) // W128 QNORM

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}
