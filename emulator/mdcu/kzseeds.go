package mdcu

// mdcu-kz seed classes and the guard that keeps smoke runs and unit tests off every
// frozen or already-used seed (frozen mdcu-kz prereg §6 seed ranges, plus amend-1 §4).
//
//   main 5000-5199, null 5200-5399, reserved 5400-5599 (split-G2 round 2, untouched),
//   Stage 0 5600-5629, saturation 5630-5649 (amend-1), probe 9930-9949.
//   Used by earlier rounds: R3 3000-3199, R4 4000-4399, tests/probes 9100-9929 and
//   9950-9979 (R4 9960-9969 inside it).
//   qbp-kz-g2 round 1 (FROZEN-v1 §2, sha256 785e0ee2…f51e28): 5900-5939 is the run block
//   (τ index i ∈ 0..3 ↔ τ_Q ∈ {1600, 3200, 6400, 12800} runs seeds 5900+10i … 5900+10i+9);
//   5940-5979 is the reserved re-run block (same τ index, under a dated amendment only).

import "fmt"

type KZSeedClass int

const (
	KZSeedFree KZSeedClass = iota
	KZSeedMain
	KZSeedNull
	KZSeedReserved
	KZSeedStage0
	KZSeedSaturation
	KZSeedProbe
	KZSeedUsed
	KZSeedTilt        // 5650-5849: B null tilt seeds, one per replicate (amend-1 item 3)
	KZSeedTiltCal     // 5850-5889: sigma* calibration seeds (20 tilt + 20 field)
	KZSeedKzG2R1      // 5900-5939: qbp-kz-g2 round 1 Stage-0b-only run block (-run-0b only)
	KZSeedKzG2R1Rerun // 5940-5979: qbp-kz-g2 round 1 reserved re-run block (amendment only)
)

func (c KZSeedClass) String() string {
	return [...]string{"free", "main", "null", "reserved", "stage0", "saturation", "probe", "used", "tilt", "tilt-calibration",
		"kz-g2-r1", "kz-g2-r1-rerun"}[c]
}

// KZClassifySeed returns the class of a seed.
func KZClassifySeed(seed uint64) KZSeedClass {
	switch {
	case seed >= 5000 && seed <= 5199:
		return KZSeedMain
	case seed >= 5200 && seed <= 5399:
		return KZSeedNull
	case seed >= 5400 && seed <= 5599:
		return KZSeedReserved
	case seed >= 5600 && seed <= 5629:
		return KZSeedStage0
	case seed >= 5630 && seed <= 5649:
		return KZSeedSaturation
	case seed >= 5650 && seed <= 5849:
		return KZSeedTilt
	case seed >= 5850 && seed <= 5889:
		return KZSeedTiltCal
	case seed >= 5900 && seed <= 5939:
		return KZSeedKzG2R1
	case seed >= 5940 && seed <= 5979:
		return KZSeedKzG2R1Rerun
	case seed >= 9930 && seed <= 9949:
		return KZSeedProbe
	case seed >= 3000 && seed <= 3199, seed >= 4000 && seed <= 4399,
		seed >= 9100 && seed <= 9929, seed >= 9950 && seed <= 9979:
		return KZSeedUsed
	}
	return KZSeedFree
}

// KZCheckSmokeSeeds refuses any seed that is not free: a smoke run or unit test must
// not touch a frozen kz seed or a seed used by an earlier round.
func KZCheckSmokeSeeds(seeds []uint64) error {
	for _, s := range seeds {
		if c := KZClassifySeed(s); c != KZSeedFree {
			return fmt.Errorf("seed %d is in a %s range and may not be used outside its mode", s, c)
		}
	}
	return nil
}

// ---- qbp-kz-g2 round 1 (FROZEN-v1 §2 Processes / Seeds) ----------------------------------

// KZKzG2R1TauSet is the frozen ramp set; its index is the τ index i of the Processes
// paragraph. The seed sub-block of a process is derived from i, never typed.
var KZKzG2R1TauSet = []int{1600, 3200, 6400, 12800}

const (
	kzG2R1Seed0      = 5900 // run block 5900-5939
	kzG2R1RerunSeed0 = 5940 // reserved re-run block 5940-5979
	kzG2R1SeedsPer   = 10   // seeds per τ_Q
)

// KZKzG2R1TauIndex returns the τ index of tau in KZKzG2R1TauSet.
func KZKzG2R1TauIndex(tau int) (int, bool) {
	for i, t := range KZKzG2R1TauSet {
		if t == tau {
			return i, true
		}
	}
	return -1, false
}

// KZKzG2R1Seeds is the seed sub-block of τ index i: 5900+10i … 5900+10i+9, or under the
// re-run amendment 5940+10i … 5940+10i+9. An index outside 0..3 returns nil.
func KZKzG2R1Seeds(tauIdx int, rerun bool) []uint64 {
	if tauIdx < 0 || tauIdx >= len(KZKzG2R1TauSet) {
		return nil
	}
	base := uint64(kzG2R1Seed0)
	if rerun {
		base = kzG2R1RerunSeed0
	}
	out := make([]uint64, kzG2R1SeedsPer)
	for j := range out {
		out[j] = base + uint64(kzG2R1SeedsPer*tauIdx+j)
	}
	return out
}

// KZCheckKzG2R1Seeds is the -run-0b seed guard: the seeds must be exactly the sub-block of
// τ index tauIdx (the re-run sub-block iff rerun), in order. Anything else is refused.
func KZCheckKzG2R1Seeds(seeds []uint64, tauIdx int, rerun bool) error {
	want := KZKzG2R1Seeds(tauIdx, rerun)
	if want == nil {
		return fmt.Errorf("tau index %d is not in 0..%d", tauIdx, len(KZKzG2R1TauSet)-1)
	}
	if len(seeds) != len(want) {
		return fmt.Errorf("%d seeds given; tau index %d runs exactly the %d seeds %d..%d", len(seeds), tauIdx, len(want), want[0], want[len(want)-1])
	}
	wantClass := KZSeedKzG2R1
	if rerun {
		wantClass = KZSeedKzG2R1Rerun
	}
	for j, s := range seeds {
		if s != want[j] {
			return fmt.Errorf("seed %d at position %d; tau index %d runs exactly %d..%d", s, j, tauIdx, want[0], want[len(want)-1])
		}
		if c := KZClassifySeed(s); c != wantClass {
			return fmt.Errorf("seed %d is in a %s range, not %s", s, c, wantClass)
		}
	}
	return nil
}
