package mdcu

// The E3 calibration ensemble (amend-1 item 3, oppenheimer seq 223): an analytic smooth
// random SO(3) field family with one roughness amplitude A, and the bisection that pins
// A_half(L), the amplitude at which the UN-tilted field's pooled ambiguous-cell fraction
// (pi/2 rule) is 2.5% (half the 5% rung kill), tolerance +-0.2% absolute, bracket [0, 32].
// sigma*(L) is then measured on the tilted fields at that amplitude.

import (
	"math"
	"math/rand/v2"
)

const (
	KZAmpBracketHi   = 32.0
	KZAmpTargetFrac  = 0.025
	KZAmpTolerance   = 0.002
	kzAmpMaxBisect   = 80
	kzSmoothStreamHi = 0x6b7a5465 // same PCG stream constant as the test RNG, so exploration numbers carry over
)

// KZSmoothField builds the smooth random SO(3) field of amplitude A for the seed: six plane
// waves k in {x,y,z,xy,yz,xz} with random per-component phases, each a rotation vector of
// size A/6 summed per site, mapped to the unit quaternion exp(rv/2). A = 0 is the identity.
func KZSmoothField(L int, A float64, seed uint64) [][4]float64 {
	rng := rand.New(rand.NewPCG(seed, kzSmoothStreamHi))
	type mode struct {
		k   [3]float64
		phi [3]float64
	}
	var modes []mode
	for _, k := range [][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 0}, {0, 1, 1}, {1, 0, 1}} {
		modes = append(modes, mode{k, [3]float64{rng.Float64() * 2 * math.Pi, rng.Float64() * 2 * math.Pi, rng.Float64() * 2 * math.Pi}})
	}
	q := make([][4]float64, L*L*L)
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				var rv [3]float64
				for _, m := range modes {
					ph := 2 * math.Pi * (m.k[0]*float64(i) + m.k[1]*float64(j) + m.k[2]*float64(k)) / float64(L)
					for c := 0; c < 3; c++ {
						rv[c] += A * math.Cos(ph+m.phi[c]) / float64(len(modes))
					}
				}
				n := math.Sqrt(rv[0]*rv[0] + rv[1]*rv[1] + rv[2]*rv[2])
				if n < 1e-15 {
					q[kzSiteIdx(L, i, j, k)] = [4]float64{1, 0, 0, 0}
					continue
				}
				h := n / 2
				q[kzSiteIdx(L, i, j, k)] = [4]float64{math.Cos(h), math.Sin(h) * rv[0] / n, math.Sin(h) * rv[1] / n, math.Sin(h) * rv[2] / n}
			}
		}
	}
	return q
}

// KZAmbigFracPooled is the un-tilted pooled ambiguous fraction (ambiguous cells / cells) of
// the smooth family at amplitude A over the field seeds.
func KZAmbigFracPooled(L int, A float64, fieldSeeds []uint64, p Z2Params) (frac float64, cells, ambiguous int) {
	for _, s := range fieldSeeds {
		r := Z2DetectS3(L, KZSmoothField(L, A, s), p)
		cells += r.Cells
		ambiguous += r.Ambiguous
	}
	return float64(ambiguous) / float64(cells), cells, ambiguous
}

// KZAmpResult is the outcome of the A_half bisection.
type KZAmpResult struct {
	A          float64
	Frac       float64 // pooled un-tilted ambiguous fraction at A
	Iterations int
	Status     string // "converged", "bracket-top" (target not reached at A = 32), "no-convergence"
}

// KZAmplitudeHalf bisects A in [0, 32] for the pooled un-tilted ambiguous fraction to equal
// KZAmpTargetFrac within KZAmpTolerance (absolute). Deterministic. If even A = 32 stays
// below the target band it returns A = 32 with Status "bracket-top"; if bisection ends
// without landing in the band (a jump in the fraction) it returns the final midpoint with
// Status "no-convergence". Both are recorded, never silently adjusted.
func KZAmplitudeHalf(L int, fieldSeeds []uint64, p Z2Params) KZAmpResult {
	fHi, _, _ := KZAmbigFracPooled(L, KZAmpBracketHi, fieldSeeds, p)
	if fHi < KZAmpTargetFrac-KZAmpTolerance {
		return KZAmpResult{A: KZAmpBracketHi, Frac: fHi, Status: "bracket-top"}
	}
	lo, hi := 0.0, KZAmpBracketHi
	var mid, f float64
	for it := 1; it <= kzAmpMaxBisect; it++ {
		mid = (lo + hi) / 2
		f, _, _ = KZAmbigFracPooled(L, mid, fieldSeeds, p)
		if math.Abs(f-KZAmpTargetFrac) <= KZAmpTolerance {
			return KZAmpResult{A: mid, Frac: f, Iterations: it, Status: "converged"}
		}
		if f < KZAmpTargetFrac {
			lo = mid
		} else {
			hi = mid
		}
	}
	return KZAmpResult{A: mid, Frac: f, Iterations: kzAmpMaxBisect, Status: "no-convergence"}
}

// KZSmoothCalibration is one lattice size's full E3 calibration: the amplitude, the sigma
// rows measured on the tilted smooth fields at that amplitude, and sigma*.
type KZSmoothCalibration struct {
	Amp      KZAmpResult
	Rows     []KZCalRow
	Sigma    float64
	Fallback bool
}

// KZCalibrateSmooth runs the ruled E3 calibration at one L: bisect A_half on the un-tilted
// field seeds, then measure the pooled ambiguous fraction at every sigma of the grid on the
// same fields tilted with the tilt seeds, and apply the sigma* rule.
func KZCalibrateSmooth(L int, fieldSeeds, tiltSeeds []uint64, p Z2Params) KZSmoothCalibration {
	amp := KZAmplitudeHalf(L, fieldSeeds, p)
	mk := func(i int) [][4]float64 { return KZSmoothField(L, amp.A, fieldSeeds[i]) }
	rows := KZCalibrateSigma(L, mk, tiltSeeds, p)
	s, fb := KZSigmaStar(rows)
	return KZSmoothCalibration{Amp: amp, Rows: rows, Sigma: s, Fallback: fb}
}
