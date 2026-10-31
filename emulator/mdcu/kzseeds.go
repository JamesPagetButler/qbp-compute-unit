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
//   qbp-kz-g2 round 3 (FROZEN-v1 §2, sha256 132f6a42…1a623): 6000-6119 is the run block
//   (seed = 6000 + 40·l + 4·t + j, l ∈ {0: L=8, 1: L=12, 2: L=16}, t ∈ 0..9 the T index,
//   j ∈ 0..2 the replicate; j = 3 and (l = 0, t ∈ {3,5,7}) are unused addresses); 6120-6239
//   is the reserved re-run block (same formula + 120, under a dated amendment only).

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
	KZSeedKzG2R3      // 6000-6119: qbp-kz-g2 round 3 equilibrium run block (-run-eq only)
	KZSeedKzG2R3Rerun // 6120-6239: qbp-kz-g2 round 3 reserved re-run block (amendment only)
)

func (c KZSeedClass) String() string {
	return [...]string{"free", "main", "null", "reserved", "stage0", "saturation", "probe", "used", "tilt", "tilt-calibration",
		"kz-g2-r1", "kz-g2-r1-rerun", "kz-g2-r3", "kz-g2-r3-rerun"}[c]
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
	case seed >= 6000 && seed <= 6119:
		return KZSeedKzG2R3
	case seed >= 6120 && seed <= 6239:
		return KZSeedKzG2R3Rerun
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

// ---- qbp-kz-g2 round 3 (FROZEN-v1 §2 T grid / Lattices / Seeds, §5 seed guard) --------------

// KZKzG2R3TTable is the frozen T grid, indexed t ∈ 0..9 in ascending T (units of J). A typed
// literal, never computed (FROZEN-v1 §2 "T grid").
var KZKzG2R3TTable = []float64{1.08, 1.10, 1.12, 1.13, 1.14, 1.15, 1.16, 1.17, 1.18, 1.20}

// KZKzG2R3LSet is the frozen lattice set, indexed l ∈ {0, 1, 2}.
var KZKzG2R3LSet = []int{8, 12, 16}

const (
	kzG2R3Seed0      = 6000 // run block 6000-6119
	kzG2R3RerunSeed0 = 6120 // reserved re-run block 6120-6239 (= run block + 120)
	kzG2R3StrideL    = 40   // seed addresses per L block
	kzG2R3StrideT    = 4    // seed addresses per (L, T) point (3 used, j = 3 unused)
	kzG2R3SeedsPer   = 3    // replicates j = 0, 1, 2
)

// KZKzG2R3LIndex returns the l index of L in KZKzG2R3LSet.
func KZKzG2R3LIndex(L int) (int, bool) {
	for i, v := range KZKzG2R3LSet {
		if v == L {
			return i, true
		}
	}
	return -1, false
}

// KZKzG2R3Runs reports whether the (l, t) point is in the frozen design: all ten t at L = 12
// and 16; at L = 8 only the seven coarse points t ∈ {0, 1, 2, 4, 6, 8, 9}.
func KZKzG2R3Runs(l, t int) bool {
	if l < 0 || l >= len(KZKzG2R3LSet) || t < 0 || t >= len(KZKzG2R3TTable) {
		return false
	}
	if l == 0 && (t == 3 || t == 5 || t == 7) {
		return false
	}
	return true
}

// KZKzG2R3Starts is the frozen start rule per replicate: j = 0, 1 Haar (hot), j = 2 aligned
// (cold) — FROZEN-v1 §2 "Per run", r5 start rule.
func KZKzG2R3Starts() []string { return []string{"haar", "haar", "aligned"} }

// KZKzG2R3Seeds is the 3-seed sub-block of the (l, t) point: {6000 + 40·l + 4·t + j : j = 0,
// 1, 2}, or + 120 under the re-run amendment. A point outside the design (bad l or t, or
// L = 8 at t ∈ {3, 5, 7}) returns nil.
func KZKzG2R3Seeds(l, t int, rerun bool) []uint64 {
	if !KZKzG2R3Runs(l, t) {
		return nil
	}
	base := uint64(kzG2R3Seed0)
	if rerun {
		base = kzG2R3RerunSeed0
	}
	out := make([]uint64, kzG2R3SeedsPer)
	for j := range out {
		out[j] = base + uint64(kzG2R3StrideL*l+kzG2R3StrideT*t+j)
	}
	return out
}

// KZCheckKzG2R3Seeds is the -run-eq seed guard (FROZEN-v1 §5): the seeds must be exactly the
// derived sub-block of (l, t) (the re-run sub-block iff rerun), in order, each in the
// matching class. The pair (L = 8, t ∈ {3, 5, 7}) and anything else is refused.
func KZCheckKzG2R3Seeds(seeds []uint64, l, t int, rerun bool) error {
	want := KZKzG2R3Seeds(l, t, rerun)
	if want == nil {
		if l == 0 && (t == 3 || t == 5 || t == 7) {
			return fmt.Errorf("L = 8 does not run t = %d (the 0.01 refinement points t ∈ {3, 5, 7} are L = 12/16 only)", t)
		}
		return fmt.Errorf("(l = %d, t = %d) is not a point of the frozen design (l ∈ 0..2, t ∈ 0..9)", l, t)
	}
	if len(seeds) != len(want) {
		return fmt.Errorf("%d seeds given; (l = %d, t = %d) runs exactly the %d seeds %d..%d", len(seeds), l, t, len(want), want[0], want[len(want)-1])
	}
	wantClass := KZSeedKzG2R3
	if rerun {
		wantClass = KZSeedKzG2R3Rerun
	}
	for j, s := range seeds {
		if s != want[j] {
			return fmt.Errorf("seed %d at position %d; (l = %d, t = %d) runs exactly %d..%d", s, j, l, t, want[0], want[len(want)-1])
		}
		if c := KZClassifySeed(s); c != wantClass {
			return fmt.Errorf("seed %d is in a %s range, not %s", s, c, wantClass)
		}
	}
	return nil
}
