package hweval

// FX-8350 Benchmark Baseline (Crawl Phase)
// Measured at 5.44 ns/op per thread on 8 cores (4.0 GHz)
// 1 second / 5.44 ns = 183.8 Million ops/sec/thread.
// Across 8 cores = ~1.47 Billion ops/sec.
const BaselineQMULsPerSec = 1.47e9

// FX8350 constants for scaling
const BaselineClockGHz = 4.0
const BaselineCores = 8.0
const BaselineVectorWidth = 1.0 // Standard AVX 256-bit

type PerformanceReport struct {
	EstimatedQMULs float64
	SpeedupFactor  float64
}

// EstimatePerformance calculates the theoretical throughput scaling
// based on Clock Speed, Cores, and Vector Width extensions.
func EstimatePerformance(cpu Component) PerformanceReport {
	// Vector Multiplier: AVX-512 (or 512-bit RVV) operates on 2x the data per clock
	// compared to the standard 256-bit AVX baseline on the FX-8350.
	vectorMultiplier := 1.0
	if cpu.AVX512 {
		vectorMultiplier = 2.0
	}

	// Ratio scaling
	clockRatio := cpu.ClockGHz / BaselineClockGHz
	coreRatio := float64(cpu.Cores) / BaselineCores

	// Calculate estimated throughput
	estimatedThroughput := BaselineQMULsPerSec * clockRatio * coreRatio * vectorMultiplier

	speedup := estimatedThroughput / BaselineQMULsPerSec

	return PerformanceReport{
		EstimatedQMULs: estimatedThroughput,
		SpeedupFactor:  speedup,
	}
}
