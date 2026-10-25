package mdcu

// Round-3 measurement layer (frozen §1.3). Builds on round 2's Q1Measure (kept
// untouched) and adds what round 2 lacked: the full Gram eigenvalue spectrum,
// the λ₃-share-gated 3-lock classification, and ψ₀ calibration.

import (
	"math"
	"sort"
)

const (
	r3LockS3    = 0.95
	r3LockL3    = 0.15
	r3CollapseS = 0.95
	r3CalDelta  = 0.05
)

type R3Stats struct {
	Q1Stats
	Eig       [7]float64 // Gram eigenvalues of Z[:,1:], descending
	S2        float64
	S3        float64 // identical to Q1Stats.SVD3
	L3Share   float64 // λ₃ / Σλ
	Class     string  // "3-lock" | "2-collapse" | "diffuse"
	PlanePsi0 float64 // |ψ(u1,u2,u3)| of the top-3 principal plane
	Psi0Frac  float64 // fraction of consecutive (Gram-Schmidt) triples with |ψ| ≥ 0.95
}

// r3Classify applies the frozen thresholds. 3-lock takes precedence (the two
// conditions cannot hold together in a real spectrum: λ₃/Σλ ≥ 0.15 ⇒ s₂ ≤ 0.85).
func r3Classify(s2, s3, l3 float64) string {
	switch {
	case s3 >= r3LockS3 && l3 >= r3LockL3:
		return "3-lock"
	case s2 >= r3CollapseS:
		return "2-collapse"
	default:
		return "diffuse"
	}
}

// R3Measure measures Z against the calibration form psi (psi may be Phi()).
func R3Measure(Z [][]float64, psi *Form3) (m R3Stats) {
	m.Q1Stats = Q1Measure(Z)
	var g [7][7]float64
	for _, z := range Z {
		for a := 0; a < 7; a++ {
			for b := 0; b < 7; b++ {
				g[a][b] += z[a+1] * z[b+1]
			}
		}
	}
	ev, _ := jacobiEigVec7(g)
	e := ev[:]
	sort.Sort(sort.Reverse(sort.Float64Slice(e)))
	copy(m.Eig[:], e)
	tot := 0.0
	for _, x := range m.Eig {
		tot += x
	}
	m.S2 = (m.Eig[0] + m.Eig[1]) / tot
	m.S3 = m.SVD3
	m.L3Share = m.Eig[2] / tot
	m.Class = r3Classify(m.S2, m.SVD3, m.L3Share)

	var u [3][]float64
	for t := 0; t < 3; t++ {
		u[t] = make([]float64, 8)
		for k := 0; k < 7; k++ {
			u[t][k+1] = m.Plane[t][k]
		}
	}
	m.PlanePsi0 = math.Abs(psi.Eval(u[0], u[1], u[2]))

	n := len(Z)
	cnt := 0
	for i := 0; i < n; i++ {
		a, b, c := Z[(i-1+n)%n], Z[i], Z[(i+1)%n]
		v := [3][]float64{append([]float64(nil), a...), append([]float64(nil), b...), append([]float64(nil), c...)}
		ok := true
		for t := 0; t < 3 && ok; t++ {
			for s := 0; s < t; s++ {
				d := dot(v[t], v[s])
				for k := range v[t] {
					v[t][k] -= d * v[s][k]
				}
			}
			nn := math.Sqrt(dot(v[t], v[t]))
			if nn < 1e-9 {
				ok = false
				break
			}
			for k := range v[t] {
				v[t][k] /= nn
			}
		}
		if ok && math.Abs(psi.Eval(v[0], v[1], v[2])) >= 1-r3CalDelta {
			cnt++
		}
	}
	m.Psi0Frac = float64(cnt) / float64(n)
	return
}
