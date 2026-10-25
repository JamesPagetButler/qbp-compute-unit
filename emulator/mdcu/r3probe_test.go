package mdcu

import (
	"os"
	"sort"
	"testing"
	"time"
)

// TestS1CostProbe measures the S1 definite-stratum draw counts and time
// (opt-in: R3_PROBE=1). It documents the sampler cost behind the resource
// estimate in the frozen pre-registration (section 2). Out-of-grid seeds only.
func TestS1CostProbe(t *testing.T) {
	if os.Getenv("R3_PROBE") == "" {
		t.Skip("set R3_PROBE=1 to run")
	}
	var draws []int
	t0 := time.Now()
	misses := 0
	for seed := uint64(9500); seed < 9560; seed += 2 {
		d, ok := SampleS1(seed, "definite")
		if !ok {
			misses++
		}
		draws = append(draws, d.Draws)
	}
	el := time.Since(t0)
	sort.Ints(draws)
	tot := 0
	for _, d := range draws {
		tot += d
	}
	t.Logf("30 definite seeds: min %d median %d max %d mean %.0f misses %d; total %d draws in %v (%.3f ms/draw)",
		draws[0], draws[len(draws)/2], draws[len(draws)-1], float64(tot)/30, misses, tot, el, float64(el.Microseconds())/1000/float64(tot))
}
