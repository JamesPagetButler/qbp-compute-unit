package mdcu

// mdcu-kz S³ null, option B (amend-1 item 3, oppenheimer seq 215): at READ TIME ONLY each
// site's frame plane (the R³ ⊂ R⁷ carrying Ad_q) is tilted by E_x = exp(σ ξ_x), with ξ_x an
// i.i.d. unit-Frobenius-norm element of so(7), fixed per site per replicate by a pinned
// seed. The plane field E_x·P₀ is homotopic to the constant plane through t ↦ exp(tσξ_x),
// so w₂ = 0 exactly for every σ, while the overlap F_a F_bᵀ is no longer a rotation: the
// polar transport is non-flat. The dynamics of the S³ arm are untouched.
//
// HARNESS choices the ruling leaves implicit: ξ_x is drawn as 21 i.i.d. standard normals
// for the strict upper triangle, antisymmetrised and scaled to unit Frobenius norm; each
// site has its own PCG stream (seed, site); exp is a 20-term Taylor series (σ ≤ 0.4, so
// ‖σξ‖_F ≤ 0.4 and the truncation error is far below 1e-16); the frame rows are mapped as
// f'_i = E f_i.

import (
	"math"
	"math/rand/v2"
)

// KZSigmaGrid is the ruled grid for σ* (rad).
var KZSigmaGrid = []float64{0.05, 0.10, 0.20, 0.40}

// KZSigmaAmbigMax is the ruled ceiling: the perturbed null's pooled ambiguous-cell
// fraction (π/2 rule) must stay at or below this for σ to be admissible.
const KZSigmaAmbigMax = 0.05

// kzTiltXi returns ξ_x: unit Frobenius norm, antisymmetric, a pure function of (seed, site).
func kzTiltXi(seed uint64, site int) [7][7]float64 {
	rng := rand.New(rand.NewPCG(seed, 0x74696c7400000000|uint64(site)))
	var x [7][7]float64
	fro2 := 0.0
	for i := 0; i < 7; i++ {
		for j := i + 1; j < 7; j++ {
			v := rng.NormFloat64()
			x[i][j], x[j][i] = v, -v
			fro2 += 2 * v * v
		}
	}
	n := math.Sqrt(fro2)
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			x[i][j] /= n
		}
	}
	return x
}

// kzExpSO7 returns exp(s·x) for antisymmetric x by Taylor series (valid for the small
// arguments used here; ‖s·x‖_F ≤ 0.4 on the ruled grid).
func kzExpSO7(x *[7][7]float64, s float64) [7][7]float64 {
	var a, term, sum [7][7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			a[i][j] = s * x[i][j]
		}
		term[i][i], sum[i][i] = 1, 1
	}
	for n := 1; n <= 20; n++ {
		var nt [7][7]float64
		for i := 0; i < 7; i++ {
			for j := 0; j < 7; j++ {
				v := 0.0
				for k := 0; k < 7; k++ {
					v += term[i][k] * a[k][j]
				}
				nt[i][j] = v / float64(n)
			}
		}
		term = nt
		for i := 0; i < 7; i++ {
			for j := 0; j < 7; j++ {
				sum[i][j] += term[i][j]
			}
		}
	}
	return sum
}

// KZTiltedFrames builds the read-time frame field of an S³ field: F_x = [R_x | 0], then
// each row mapped by E_x = exp(σ ξ_x(tiltSeed, x)). σ = 0 returns [R_x | 0] exactly.
func KZTiltedFrames(L int, q [][4]float64, sigma float64, tiltSeed uint64) [][3][7]float64 {
	f := make([][3][7]float64, len(q))
	for s := range q {
		r := QuatToMat(q[s])
		if sigma == 0 {
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					f[s][i][j] = r[i][j]
				}
			}
			continue
		}
		xi := kzTiltXi(tiltSeed, s)
		e := kzExpSO7(&xi, sigma)
		for i := 0; i < 3; i++ {
			for k := 0; k < 7; k++ {
				v := 0.0
				for m := 0; m < 3; m++ { // rows of [R | 0] are zero beyond column 3
					v += e[k][m] * r[i][m]
				}
				f[s][i][k] = v
			}
		}
	}
	return f
}

// KZDetectS3Tilted is the B null's detector: tilt at read time, then the same plane
// detector as the main arm. σ = 0 is exactly the S³ adapter's frames.
func KZDetectS3Tilted(L int, q [][4]float64, sigma float64, tiltSeed uint64, p Z2Params) Z2Result {
	if sigma == 0 {
		return Z2DetectS3(L, q, p)
	}
	return Z2DetectPlanes(L, KZTiltedFrames(L, q, sigma, tiltSeed), p)
}

// ---- σ* calibration ----------------------------------------------------------------------

// KZCalRow is the calibration result at one σ over a set of pinned fields.
type KZCalRow struct {
	Sigma           float64
	Fields          int
	Cells           int
	Ambiguous       int
	NegCells        int
	Defects         int
	PooledAmbigFrac float64
	MaxFieldFrac    float64
}

// KZCalibrateSigma measures, at every σ of the grid, the perturbed null's ambiguous-cell
// fraction over the fields mk(0..len(tiltSeeds)-1), tilted with the pinned tilt seeds. It
// runs no dynamics.
func KZCalibrateSigma(L int, mk func(i int) [][4]float64, tiltSeeds []uint64, p Z2Params) []KZCalRow {
	rows := make([]KZCalRow, len(KZSigmaGrid))
	for gi, sg := range KZSigmaGrid {
		r := KZCalRow{Sigma: sg}
		for i, ts := range tiltSeeds {
			res := KZDetectS3Tilted(L, mk(i), sg, ts, p)
			r.Fields++
			r.Cells += res.Cells
			r.Ambiguous += res.Ambiguous
			r.NegCells += res.NegCells
			r.Defects += res.Defects
			if f := float64(res.Ambiguous) / float64(res.Cells); f > r.MaxFieldFrac {
				r.MaxFieldFrac = f
			}
		}
		r.PooledAmbigFrac = float64(r.Ambiguous) / float64(r.Cells)
		rows[gi] = r
	}
	return rows
}

// KZSigmaStar applies the ruled rule: the largest σ on the grid whose pooled ambiguous
// fraction is at most 5%. If none qualifies it returns σ = 0.05 with fallback = true (the
// fact is recorded in the amendment).
func KZSigmaStar(rows []KZCalRow) (sigma float64, fallback bool) {
	best := -1.0
	for _, r := range rows {
		if r.PooledAmbigFrac <= KZSigmaAmbigMax && r.Sigma > best {
			best = r.Sigma
		}
	}
	if best < 0 {
		return KZSigmaGrid[0], true
	}
	return best, false
}
