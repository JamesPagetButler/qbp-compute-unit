package main

import (
	"fmt"
	"strings"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quantum"
)

func main() {
	banner := strings.Repeat("=", 72)
	fmt.Println(banner)
	fmt.Println("QBP-QUANTUM: HESSIAN QEC CODE VERIFICATION (CRAWL PHASE)")
	fmt.Println(banner)
	fmt.Println()

	fmt.Println("Constructing code from Hessian eigenspace decomposition...")
	code := quantum.ConstructHessianCode()

	fmt.Printf("Code parameters:\n")
	fmt.Printf("  Physical qubits: 16\n")
	fmt.Printf("  Stabilizers:     %d\n", len(code.Stabilizers))
	fmt.Printf("  Logicals:        %d\n", len(code.Logicals))
	fmt.Println()

	fmt.Println("Verifying algebraic consistency...")
	if err := code.Verify(); err != nil {
		fmt.Printf("  FAIL: %v\n", err)
	} else {
		fmt.Println("  PASS: All stabilizers commute and logicals commute with stabilizers.")
	}
	fmt.Println()

	fmt.Println("Computing code distance d...")
	fmt.Println(" (This involves searching logical operator space)")
	d := code.Distance()
	fmt.Printf("  Calculated distance d = %d\n", d)

	fmt.Println()
	fmt.Println(banner)
	if d >= 3 {
		fmt.Println("VERDICT: Hessian QEC code is a VALID [[16, k, d>=3]] code.")
		fmt.Println("         It can correct single-qubit errors.")
		fmt.Println("         The algebraic protection layer is verified.")
	} else {
		fmt.Println("VERDICT: Hessian QEC code distance d < 3.")
		fmt.Println("         It cannot correct all single-qubit errors.")
		fmt.Println("         Refinement of the eigenvector mapping is required.")
	}
	fmt.Println(banner)
}
