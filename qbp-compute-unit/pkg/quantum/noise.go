// Package quantum implements the QBP-Quantum algebraic error correction.
package quantum

import "math/rand"

// NoiseParams configures the three-channel error model for Monte Carlo simulation.
// Each channel is caught by the corresponding protection layer before the next.
type NoiseParams struct {
	// PauliRate is the depolarizing probability per qubit.
	// P(X) = P(Y) = P(Z) = PauliRate/3, P(I) = 1-PauliRate.
	// Caught by Layer 3 (Hessian syndrome).
	PauliRate float64
	// NormRate is the probability per qubit of a norm-reducing error
	// (e.g., amplitude damping). Caught by Layer 1 (Hurwitz norm check).
	NormRate float64
	// SignFlipRate is the probability per qubit of a Z2 parity flip (q → −q).
	// Caught by Layer 2 (Z2 parity check).
	SignFlipRate float64
}

// MCResult holds aggregate statistics from one Monte Carlo run.
type MCResult struct {
	Trials      int
	Params      NoiseParams
	Layer1      int     // trials heralded by norm check
	Layer2      int     // trials heralded by parity check (after L1)
	Layer3      int     // trials heralded by syndrome (after L1+L2)
	Undetected  int     // logical errors that escaped all three layers
	Clean       int     // no error on any qubit
	LogicalRate float64 // Undetected / Trials
}

// Syndrome returns a uint16 bitmask: bit i is set if stabilizer i anti-commutes with e.
func (c *Code) Syndrome(e Pauli) uint16 {
	var s uint16
	for i, stab := range c.Stabilizers {
		if !stab.Commutes(e) {
			s |= 1 << uint(i)
		}
	}
	return s
}

// RunMonteCarlo simulates trials rounds of noise injection under params.
// seed initialises the PRNG for reproducibility.
//
// Layer ordering: L1 (norm) → L2 (parity) → L3 (syndrome).
// An error caught by an earlier layer is heralded before reaching the next.
func (c *Code) RunMonteCarlo(trials int, params NoiseParams, seed int64) MCResult {
	rng := rand.New(rand.NewSource(seed))

	// Precompute the stabilizer group (up to 2^12 = 4096 elements) for O(1) lookups.
	// Used to distinguish trivial stabilizer elements from undetected logical operators.
	stabGroup := make(map[Pauli]bool, 4096)
	for _, g := range generateGroup(c.Stabilizers) {
		stabGroup[g] = true
	}

	result := MCResult{Trials: trials, Params: params}

	for range trials {
		var pauliErr Pauli
		normViolation := false
		signFlip := false

		for q := 0; q < 16; q++ {
			if rng.Float64() < params.NormRate {
				normViolation = true
			}
			if rng.Float64() < params.SignFlipRate {
				signFlip = true
			}
			r := rng.Float64()
			switch {
			case r < params.PauliRate/3:
				pauliErr ^= Pauli(1 << uint(q)) // X on qubit q
			case r < 2*params.PauliRate/3:
				pauliErr ^= Pauli(1<<uint(q) | 1<<uint(q+16)) // Y = XZ
			case r < params.PauliRate:
				pauliErr ^= Pauli(1 << uint(q+16)) // Z on qubit q
			}
		}

		// Layer 1: herald norm-reducing errors before syndrome measurement.
		if normViolation {
			result.Layer1++
			continue
		}
		// Layer 2: herald Z2 sign flips before syndrome measurement.
		if signFlip {
			result.Layer2++
			continue
		}
		// Layer 3: syndrome measurement heralds detectable Pauli errors.
		if c.Syndrome(pauliErr) != 0 {
			result.Layer3++
			continue
		}
		// Zero-syndrome trial: clean (identity or stabilizer) or undetected logical error.
		// A non-identity element with zero syndrome that is not in the stabilizer group
		// commutes with all stabilizers but acts non-trivially on the logical subspace.
		if pauliErr != 0 && !stabGroup[pauliErr] {
			result.Undetected++
		} else {
			result.Clean++
		}
	}

	if trials > 0 {
		result.LogicalRate = float64(result.Undetected) / float64(trials)
	}
	return result
}

// EnumerateSingleErrors returns the detection verdict for every weight-1 error event.
// Used to verify the d≥1 property: no single error should escape all three layers.
//
// Returns (layer1Count, layer2Count, layer3Count, escaped) where escaped is the
// list of undetected single-error Pauli operators.
func (c *Code) EnumerateSingleErrors() (l1, l2, l3 int, escaped []Pauli) {
	stabGroup := make(map[Pauli]bool, 4096)
	for _, g := range generateGroup(c.Stabilizers) {
		stabGroup[g] = true
	}

	// 16 norm violations and 16 sign flips: all caught by L1/L2 by construction.
	l1 = 16
	l2 = 16

	// 48 single-qubit Pauli errors (X, Y, Z on each of 16 qubits).
	for q := 0; q < 16; q++ {
		for _, e := range []Pauli{
			Pauli(1 << uint(q)),               // X
			Pauli(1<<uint(q) | 1<<uint(q+16)), // Y
			Pauli(1 << uint(q+16)),            // Z
		} {
			if c.Syndrome(e) != 0 {
				l3++
			} else if e != 0 && !stabGroup[e] {
				escaped = append(escaped, e)
			}
		}
	}
	return l1, l2, l3, escaped
}
