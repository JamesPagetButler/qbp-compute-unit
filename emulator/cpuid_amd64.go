package emulator

// hasAVXAndFMA checks if the CPU and OS support AVX and FMA3.
// It runs the CPUID instruction without importing any external dependencies.
func hasAVXAndFMA() bool
