package mdcu

// Round-4 level layer (doc/spikes/prereg/mdcu-round4-prereg-FROZEN-v1.md, §4).
// Nothing here is part of the frozen v0.1.0-rc1 emulator API, and no round-2 or
// round-3 file is modified: R3 code is called, never edited.
//
// Level scaling (frozen §4): f_L = L·f/comass(f), φ_L = L·φ (comass(φ) = 1).
// Whole-triple energies at level κ are the level-1 energies times κ², so that
// K-gram@κ + K-phi@κ = K-full@κ exactly (Harvey–Lawson partition kept).

import (
	"math"
)

// Frozen constants (R4 prereg §2, §4).
const (
	R4Kappa     = 1.88 // constant level κ; κ² = 3.5344
	R4ComassTol = 1e-6 // two-restart-set agreement, relative
	R4StepsA    = 250
	R4StepsB    = 1000
	R4StepsC    = 4000

	R4GridLo  = 4000
	R4GridHi  = 4199
	R4SDLo    = 4200
	R4SDHi    = 4399
	R4ProbeLo = 9960
	R4ProbeHi = 9969

	R4SDPerScale = 50

	R4ComassStarts = 200
	R4ComassIters  = 500
)

// R4SDScales is the S-D deformation-scale ladder (50 seeds each).
var R4SDScales = []float64{0.005, 0.01, 0.02, 0.05}

// Round-4 arms. The "@κ" arms are written "level κ" in prose on the bus.
const (
	ArmPsi1  R3Arm = "K-psi@1"  // −4ψ², ψ rescaled to comass 1
	ArmPhiK  R3Arm = "K-phi@κ"  // −4(κφ)²
	ArmPsiK  R3Arm = "K-psi@κ"  // −4ψ², ψ rescaled to comass κ
	ArmFullK R3Arm = "K-full@κ" // κ²·K-full
	ArmGramK R3Arm = "K-gram@κ" // κ²·K-gram
)

// Stream tags for the comass restart sets. Distinct from every round-2/3 constant
// and from the post-hoc diagnostic's (postHocSeedA/B).
const (
	r4ComassSeedA uint64 = 0x3c6ef372
	r4ComassSeedB uint64 = 0xa54ff53a
)

// Tier arm lists (frozen §4), in reporting order.
var (
	R4TierAArms = []R3Arm{ArmFull, ArmGram, ArmPhi, ArmPsi, ArmPsi1, ArmPhiK, ArmPsiK, ArmFullK, ArmGramK, ArmNone}
	R4TierBArms = []R3Arm{ArmPhi, ArmPsi1, ArmPhiK, ArmPsiK}
	R4TierCArms = []R3Arm{ArmPhi, ArmPsi1}
)

// ScaleForm3 returns c·f as a new form. f is not modified.
func ScaleForm3(f *Form3, c float64) *Form3 {
	var t [7][7][7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			for k := 0; k < 7; k++ {
				t[i][j][k] = c * f.C[i][j][k]
			}
		}
	}
	return NewForm3(t)
}

type scaledEnergy struct {
	in R3Energy
	k  float64
}

// NewScaledEnergy is k·E: its gradient is k times the gradient of in.
func NewScaledEnergy(in R3Energy, k float64) R3Energy { return scaledEnergy{in, k} }

func (s scaledEnergy) Grad(x, y, z []float64, pos int) []float64 {
	g := s.in.Grad(x, y, z, pos)
	for i := range g {
		g[i] *= s.k
	}
	return g
}

// ComassRescale is the result of rescaling a form to comass L.
type ComassRescale struct {
	Comass    float64 // the comass used for the rescale (max of the two restart sets)
	ComassB   float64 // the lower of the two estimates
	RelDiff   float64 // |a−b| / max(a,b)
	Converged bool    // RelDiff <= R4ComassTol
	Form      *Form3  // L·f/Comass; f itself is not modified
}

// RescaleToComass estimates comass(f) with two independent restart sets of
// R4ComassStarts starts × R4ComassIters iterations (RNG streams seed^tagA and
// seed^tagB), takes the larger as the comass, and returns f·L/comass. A seed
// whose two sets differ by more than R4ComassTol (relative) is reported
// unconverged; the caller drops its rescaled arms and counts it.
func RescaleToComass(f *Form3, seed uint64, L float64) ComassRescale {
	a := ComassEstimate(f, newRNG(seed^r4ComassSeedA), R4ComassStarts, R4ComassIters)
	b := ComassEstimate(f, newRNG(seed^r4ComassSeedB), R4ComassStarts, R4ComassIters)
	hi, lo := math.Max(a, b), math.Min(a, b)
	rel := math.Inf(1)
	if hi > 0 {
		rel = (hi - lo) / hi
	}
	r := ComassRescale{Comass: hi, ComassB: lo, RelDiff: rel, Converged: rel <= R4ComassTol}
	if hi > 0 {
		r.Form = ScaleForm3(f, L/hi)
	}
	return r
}

// R4EnergyFor returns the triple energy of an arm. psiNative is the seed's R3
// Primary ψ (K-psi), psi1 and psiK its comass-1 and comass-κ rescales (K-psi@1,
// K-psi@κ). K-none returns nil (μ term only). Level-1 arms are R3's, unchanged.
func R4EnergyFor(arm R3Arm, psiNative, psi1, psiK *Form3) R3Energy {
	need := func(f *Form3) *Form3 {
		if f == nil {
			panic("mdcu: " + string(arm) + " needs its ψ")
		}
		return f
	}
	k2 := R4Kappa * R4Kappa
	switch arm {
	case ArmFull, ArmGram, ArmPhi:
		return R3EnergyFor(arm, nil, nil)
	case ArmPsi:
		return R3EnergyFor(arm, need(psiNative), nil)
	case ArmPsi1:
		return NewScalarEnergy(false, need(psi1))
	case ArmPsiK:
		return NewScalarEnergy(false, need(psiK))
	case ArmPhiK:
		return NewScalarEnergy(false, ScaleForm3(Phi(), R4Kappa))
	case ArmFullK:
		return NewScaledEnergy(R3EnergyFor(ArmFull, nil, nil), k2)
	case ArmGramK:
		return NewScaledEnergy(R3EnergyFor(ArmGram, nil, nil), k2)
	case ArmNone:
		return nil
	}
	panic("mdcu: unknown round-4 arm " + string(arm))
}

// SampleS2At is SampleS2 with the deformation scale fixed to s (S-D ladder). It
// replays SampleS2's RNG order, including the scale draw, so the 7×7 Gaussian is
// the seed's own; SampleS2At(seed, SampleS2(seed).S) equals SampleS2(seed) bit
// for bit (TestR4SampleS2At). The frozen SampleS2 is not modified.
func SampleS2At(seed uint64, s float64) S2Draw {
	rng := newRNG(seed ^ r3SeedPsiS2)
	_ = rng.Float64() // the scale draw, discarded
	var G, S [7][7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			G[i][j] = rng.NormFloat64()
		}
	}
	tr := 0.0
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			S[i][j] = 0.5 * (G[i][j] + G[j][i])
		}
		tr += S[i][i]
	}
	for i := 0; i < 7; i++ {
		S[i][i] -= tr / 7
	}
	trAfter := 0.0
	for i := 0; i < 7; i++ {
		trAfter += S[i][i]
	}
	ev, V := jacobiEigVec7(S)
	var g [7][7]float64
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			for p := 0; p < 7; p++ {
				g[i][j] += V[i][p] * math.Exp(s*ev[p]) * V[j][p]
			}
		}
	}
	phi := Phi()
	var t1, t2, t3 [7][7][7]float64
	for a := 0; a < 7; a++ {
		for j := 0; j < 7; j++ {
			for k := 0; k < 7; k++ {
				for i := 0; i < 7; i++ {
					t1[a][j][k] += g[i][a] * phi.C[i][j][k]
				}
			}
		}
	}
	for a := 0; a < 7; a++ {
		for b := 0; b < 7; b++ {
			for k := 0; k < 7; k++ {
				for j := 0; j < 7; j++ {
					t2[a][b][k] += g[j][b] * t1[a][j][k]
				}
			}
		}
	}
	for a := 0; a < 7; a++ {
		for b := 0; b < 7; b++ {
			for c := 0; c < 7; c++ {
				for k := 0; k < 7; k++ {
					t3[a][b][c] += g[k][c] * t2[a][b][k]
				}
			}
		}
	}
	f := NewForm3(t3)
	f.Rescale(r3FrobSq)
	var d2 float64
	for a := 0; a < 7; a++ {
		for b := 0; b < 7; b++ {
			for c := 0; c < 7; c++ {
				x := f.C[a][b][c] - phi.C[a][b][c]
				d2 += x * x
			}
		}
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, e := range ev {
		lo, hi = math.Min(lo, e), math.Max(hi, e)
	}
	return S2Draw{Form: f, S: s, TraceS: trAfter, Dist: math.Sqrt(d2 / r3FrobSq), Cond: math.Exp(2 * s * (hi - lo))}
}
