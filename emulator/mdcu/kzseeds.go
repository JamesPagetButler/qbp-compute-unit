package mdcu

// mdcu-kz seed classes and the guard that keeps smoke runs and unit tests off every
// frozen or already-used seed (frozen mdcu-kz prereg §6 seed ranges, plus amend-1 §4).
//
//   main 5000-5199, null 5200-5399, reserved 5400-5599 (split-G2 round 2, untouched),
//   Stage 0 5600-5629, saturation 5630-5649 (amend-1), probe 9930-9949.
//   Used by earlier rounds: R3 3000-3199, R4 4000-4399, tests/probes 9100-9929 and
//   9950-9979 (R4 9960-9969 inside it).

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
	KZSeedTilt    // 5650-5849: B null tilt seeds, one per replicate (amend-1 item 3)
	KZSeedTiltCal // 5850-5889: sigma* calibration seeds (20 tilt + 20 field)
)

func (c KZSeedClass) String() string {
	return [...]string{"free", "main", "null", "reserved", "stage0", "saturation", "probe", "used", "tilt", "tilt-calibration"}[c]
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
