package mdcu

// Round-3 signed-energy ring runner and the arm set (frozen §1.1).
//
// RunRingEnergy is RunRingGen's overlap layout (centre + (i−2,i−1,i) terms)
// with the triple energy abstracted behind R3Energy. For the vector-valued arms
// (K-full: assoc8; K-rand3: a random alternating tensor) the energy gradient is
// computed by exactly the same floating-point operations as RunRingGen, so the
// two are bit-identical (r3ring_test.go gate). Nothing here is part of the
// frozen v0.1.0-rc1 emulator API.

// R3Arm names an energy arm.
type R3Arm string

const (
	ArmFull    R3Arm = "K-full"    // ||[a,b,c]||² = 4Gram₃ − 4φ²
	ArmGram    R3Arm = "K-gram"    // 4Gram₃ (no G₂)
	ArmPhi     R3Arm = "K-phi"     // −4φ²
	ArmPsi     R3Arm = "K-psi"     // −4ψ² (ψ stratified: S2 on definite seeds, S1 on split seeds)
	ArmPsiS1   R3Arm = "K-psi-S1"  // −4ψ² with ψ from S1 (definite seeds only; sampler sensitivity)
	ArmPsiGram R3Arm = "K-psigram" // 4Gram₃ − 4ψ²
	ArmRand3   R3Arm = "K-rand3"   // round-2 null 3 (random alternating vector-valued tensor)
	ArmNone    R3Arm = "K-none"    // μ term only
)

// R3AllArms lists every arm in reporting order.
var R3AllArms = []R3Arm{ArmFull, ArmGram, ArmPhi, ArmPsi, ArmPsiS1, ArmPsiGram, ArmRand3, ArmNone}

type vecEnergy struct{ T Trilinear }

// NewVecEnergy is E = ||T(x,y,z)||² for a vector-valued trilinear map T. Its
// gradient replicates RunRingGen's basis-probe arithmetic exactly.
func NewVecEnergy(T Trilinear) R3Energy { return vecEnergy{T} }

func (v vecEnergy) Grad(x, y, z []float64, pos int) []float64 {
	A := v.T(x, y, z)
	g := make([]float64, 8)
	for b := 1; b < 8; b++ {
		e := basis(8, b)
		var Tb []float64
		switch pos {
		case 0:
			Tb = v.T(e, y, z)
		case 1:
			Tb = v.T(x, e, z)
		case 2:
			Tb = v.T(x, y, e)
		default:
			panic("mdcu: vecEnergy position out of range")
		}
		g[b] = 2 * dot(A, Tb)
	}
	return g
}

// R3EnergyFor returns the energy of an arm. psi is required for the ψ arms,
// rand3 for K-rand3. K-none returns nil (RunRingEnergy treats nil as μ only).
func R3EnergyFor(arm R3Arm, psi *Form3, rand3 *AltTensor) R3Energy {
	switch arm {
	case ArmFull:
		return NewVecEnergy(assoc8)
	case ArmRand3:
		if rand3 == nil {
			panic("mdcu: K-rand3 needs a tensor")
		}
		return NewVecEnergy(rand3.Apply)
	case ArmGram:
		return NewScalarEnergy(true, nil)
	case ArmPhi:
		return NewScalarEnergy(false, Phi())
	case ArmPsi, ArmPsiS1:
		if psi == nil {
			panic("mdcu: " + string(arm) + " needs ψ")
		}
		return NewScalarEnergy(false, psi)
	case ArmPsiGram:
		if psi == nil {
			panic("mdcu: K-psigram needs ψ")
		}
		return NewScalarEnergy(true, psi)
	case ArmNone:
		return nil
	}
	panic("mdcu: unknown round-3 arm " + string(arm))
}

// RunRingEnergy runs the round-2 Test B ring (overlap layout; lr 0.04, μ 1.5
// neighbour term, unit imaginary 8-vectors, Euclidean metric) for the given
// number of steps with triple energy en (nil = μ term only). The input is not
// modified.
func RunRingEnergy(z0 [][]float64, en R3Energy, steps int) [][]float64 {
	const lr, mu = 0.04, 1.5
	n := len(z0)
	Z := make([][]float64, n)
	for i := range Z {
		Z[i] = append([]float64(nil), z0[i]...)
	}
	for st := 0; st < steps; st++ {
		grad := make([][]float64, n)
		for i := 0; i < n; i++ {
			l, r, ll := Z[(i-1+n)%n], Z[(i+1)%n], Z[(i-2+n)%n]
			g := make([]float64, 8)
			cl, cr := dot(Z[i], l), dot(Z[i], r)
			for k := range g {
				g[k] = 2 * mu * (cr*r[k] + cl*l[k])
			}
			if en != nil {
				g1 := en.Grad(l, Z[i], r, 1)
				g2 := en.Grad(ll, l, Z[i], 2)
				for b := 1; b < 8; b++ {
					g[b] += g1[b]
					g[b] += g2[b]
				}
			}
			g[0] = 0
			pp := dot(g, Z[i])
			for k := range g {
				g[k] -= pp * Z[i][k]
			}
			grad[i] = g
		}
		for i := range Z {
			for k := range Z[i] {
				Z[i][k] -= lr * grad[i][k]
			}
			normalize(Z[i])
		}
	}
	return Z
}
