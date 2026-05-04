// Package quantum implements the QBP-Quantum algebraic error correction.
package quantum

import (
	"fmt"
)

// Pauli represents an operator in the 16-qubit symplectic space Z2^32.
type Pauli uint32

func (p Pauli) Weight() int {
	w := 0
	x := uint32(p) & 0xFFFF
	z := (uint32(p) >> 16) & 0xFFFF
	combined := x | z
	for combined > 0 {
		if combined&1 == 1 {
			w++
		}
		combined >>= 1
	}
	return w
}

func (p Pauli) Commutes(q Pauli) bool {
	x1 := uint32(p) & 0xFFFF
	z1 := (uint32(p) >> 16) & 0xFFFF
	x2 := uint32(q) & 0xFFFF
	z2 := (uint32(q) >> 16) & 0xFFFF
	inner := (x1 & z2) ^ (x2 & z1)
	pop := 0
	for inner > 0 {
		if inner&1 == 1 {
			pop++
		}
		inner >>= 1
	}
	return pop%2 == 0
}

type Code struct {
	Stabilizers []Pauli
	Logicals    []Pauli
}

// ConstructHessianCode builds a [[16, 4, 2]] code using a CSS block construction.
func ConstructHessianCode() *Code {
	var stabs []Pauli
	var logs []Pauli

	// 12 stabilizers for k = 16 - 12 = 4 logical qubits.
	
	// 8 local stabilizers (2 per block)
	for i := 0; i < 4; i++ {
		qO := uint(4 * i)
		sX := Pauli(1<<qO | 1<<(qO+1) | 1<<(qO+2) | 1<<(qO+3))
		sZ := Pauli(1<<(16+qO) | 1<<(16+qO+1) | 1<<(16+qO+2) | 1<<(16+qO+3))
		stabs = append(stabs, sX, sZ)
	}
	
	// 4 cross-block stabilizers to reduce k from 8 to 4
	for i := 0; i < 4; i++ {
		qO1 := uint(4 * i)
		qO2 := uint(4 * ((i + 1) % 4))
		// Cross-block X-stabilizer
		sX := Pauli(1<<qO1 | 1<<(qO1+1) | 1<<qO2 | 1<<(qO2+1))
		stabs = append(stabs, sX)
	}

	// 4 logical pairs (8 operators)
	for i := 0; i < 4; i++ {
		qO := uint(4 * i)
		// These logicals commute with all 12 stabilizers.
		// lX = X_qO | X_qO+2
		lX := Pauli(1<<qO | 1<<(qO+2))
		// lZ = Z_qO | Z_qO+1
		lZ := Pauli(1<<(16+qO) | 1<<(16+qO+1))
		
		logs = append(logs, lX, lZ)
	}

	return &Code{
		Stabilizers: stabs,
		Logicals:    logs,
	}
}

func (c *Code) Verify() error {
	for i, s1 := range c.Stabilizers {
		for j := i + 1; j < len(c.Stabilizers); j++ {
			if !s1.Commutes(c.Stabilizers[j]) {
				return fmt.Errorf("stabilizers %d and %d do not commute", i, j)
			}
		}
	}
	for i, l := range c.Logicals {
		for j, s := range c.Stabilizers {
			if !l.Commutes(s) {
				return fmt.Errorf("logical %d (w=%d) does not commute with stabilizer %d (w=%d)", i, l.Weight(), j, s.Weight())
			}
		}
	}
	for i := 0; i < len(c.Logicals); i += 2 {
		if c.Logicals[i].Commutes(c.Logicals[i+1]) {
			return fmt.Errorf("logical pair %d, %d commutes (should anti-commute)", i, i+1)
		}
	}
	return nil
}

func (c *Code) Distance() int {
	minW := 16
	stabGroup := generateGroup(c.Stabilizers)
	for _, l := range generateGroup(c.Logicals) {
		if l == 0 {
			continue
		}
		for _, s := range stabGroup {
			w := (l ^ s).Weight()
			if w < minW {
				minW = w
			}
		}
	}
	return minW
}

func generateGroup(generators []Pauli) []Pauli {
	group := []Pauli{0}
	m := make(map[Pauli]bool)
	m[0] = true
	for _, g := range generators {
		n := len(group)
		for i := 0; i < n; i++ {
			combined := group[i] ^ g
			if !m[combined] {
				m[combined] = true
				group = append(group, combined)
			}
		}
	}
	return group
}
