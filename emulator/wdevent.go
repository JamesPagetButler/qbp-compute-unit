// Package emulator provides the cycle-accurate execution model.
//
// NOTE: The WDEvent emission tap is strictly located at the ISA execution
// boundary (in isa.go), not inside the mathematical AVX/scalar kernels.
// This is an architectural invariant to ensure that structural execution
// events are captured independently of math kernel implementations.
package emulator

// Port specifies whether the operation arrived via SSCI or VCIX.
type Port uint8

const (
	PortSSCI Port = iota
	PortVCIX
)

// ZDClass categorizes the type of zero-divisor check performed.
type ZDClass uint8

const (
	NotZD               ZDClass = iota // No ZD check performed
	CrossCopySymbolic                  // Cheap XOR/sign test for basis sums
	GeneralFullMultiply                // Full multiplication to verify ZD
)

// Opcode represents a specific execution instruction or Funct7 equivalent.
type Opcode uint8

// WDEvent is tapped at every algebraic crossing to feed the watchdog.
type WDEvent struct {
	Cycle      uint64    // Accelerator cycle of completion
	Op         Opcode    // The Funct7 opcode equivalent
	Port       Port      // Ingress port
	FanoIndex  uint8     // Relevant Fano-plane index, if applicable
	SignBit    bool      // Fano-plane sign bit, if applicable
	Associator [3]int8   // Residue of (a*b)*c - a*(b*c)
	NormDelta  int32     // Norm preservation residue (fixed-point)
	AlgebraID  uint8     // 0=H, 1=O, 2=C⊕H⊕M3(C), 3=Branch B

	// Added per RV-Fano-Implementation-Refinements.md §2.3
	ZDClass   ZDClass
	ZDIndices [4]uint8 // (i, j, k, l) for symbolic; zeros for general
}
