package mdcu

// Round-4 spontaneity secondary (frozen R4 prereg §1.5 S-A): the round-2/3 Haar-G₂
// Kolmogorov–Smirnov test, moved into the library from cmd/mdcu-r3-analyze (whose
// copy is left untouched, being part of the round-3 record). The arithmetic and the
// RNG streams are identical, so the test reproduces R3's published p-values from
// the R3 raw (TestR4SpontaneityReproducesR3).
//
// Definition: pairwise principal-plane distances of the locked runs, KS distance D
// to the pairwise distances of an equal-size Haar-G₂ reference set (associative
// 3-planes), Monte-Carlo p over `reps` null replicates (round-2 definition: 1000).

import (
	"math"
	"math/rand/v2"
	"sort"
)

func planeDist(a, b [3][7]float64) float64 {
	s := 0.0
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			d := 0.0
			for k := 0; k < 7; k++ {
				d += a[i][k] * b[j][k]
			}
			s += d * d
		}
	}
	return math.Sqrt(math.Max(0, 3-s))
}

func planePairwise(p [][3][7]float64) []float64 {
	out := make([]float64, 0, len(p)*(len(p)-1)/2)
	for i := range p {
		for j := i + 1; j < len(p); j++ {
			out = append(out, planeDist(p[i], p[j]))
		}
	}
	return out
}

func ksDist(a, b []float64) float64 {
	x := append([]float64(nil), a...)
	y := append([]float64(nil), b...)
	sort.Float64s(x)
	sort.Float64s(y)
	i, j, d := 0, 0, 0.0
	for i < len(x) && j < len(y) {
		if x[i] <= y[j] {
			i++
		} else {
			j++
		}
		d = math.Max(d, math.Abs(float64(i)/float64(len(x))-float64(j)/float64(len(y))))
	}
	return d
}

// haarG2Plane draws a uniform associative 3-plane (u, v, uv): the G₂-orbit of the
// reference plane.
func haarG2Plane(r *rand.Rand) (p [3][7]float64) {
	unit := func() []float64 {
		v := make([]float64, 7)
		n := 0.0
		for i := range v {
			v[i] = r.NormFloat64()
			n += v[i] * v[i]
		}
		n = math.Sqrt(n)
		for i := range v {
			v[i] /= n
		}
		return v
	}
	x := unit()
	y := unit()
	d := 0.0
	for i := range x {
		d += x[i] * y[i]
	}
	n := 0.0
	for i := range y {
		y[i] -= d * x[i]
		n += y[i] * y[i]
	}
	n = math.Sqrt(n)
	for i := range y {
		y[i] /= n
	}
	x8, y8 := make([]float64, 8), make([]float64, 8)
	copy(x8[1:], x)
	copy(y8[1:], y)
	xy := MulRef(x8, y8)
	for k := 0; k < 7; k++ {
		p[0][k], p[1][k], p[2][k] = x[k], y[k], xy[k+1]
	}
	return
}

func haarG2Set(r *rand.Rand, n int) [][3][7]float64 {
	out := make([][3][7]float64, n)
	for i := range out {
		out[i] = haarG2Plane(r)
	}
	return out
}

// Spontaneity returns the KS distance D of planes against Haar-G₂ and its Monte-Carlo
// p-value over reps replicates. The RNG stream is fixed (1000, 0xdeadbeef), as in R3.
func Spontaneity(planes [][3][7]float64, reps int) (dObs, p float64) {
	obs := planePairwise(planes)
	rr := rand.New(rand.NewPCG(1000, 0xdeadbeef))
	dObs = ksDist(obs, planePairwise(haarG2Set(rr, len(planes))))
	ge := 0
	for m := 0; m < reps; m++ {
		if ksDist(planePairwise(haarG2Set(rr, len(planes))), planePairwise(haarG2Set(rr, len(planes)))) >= dObs {
			ge++
		}
	}
	return dObs, float64(ge) / float64(reps)
}

// R4Spontaneity tests an arm's 3-locked runs in a cell; arms with fewer than
// R4LockedMin locked runs are reported untestable (N set, P = NaN).
func R4Spontaneity(runs []R3Run, arm R3Arm, cell string, reps int) SponResult {
	var planes [][3][7]float64
	for _, r := range runs {
		if r.Stats.Class == "3-lock" {
			planes = append(planes, r.Stats.Plane)
		}
	}
	res := SponResult{Arm: arm, Cell: cell, N: len(planes), P: math.NaN(), D: math.NaN()}
	if len(planes) < R4LockedMin {
		return res
	}
	res.D, res.P = Spontaneity(planes, reps)
	res.Testable = true
	return res
}
