// Command mdcu-r3q2-w2diag prints the POST-HOC W2 diagnostic from a raw `go test -bench` output.
//
//	run from emulator/:  go run ./cmd/mdcu-r3q2-w2diag -bench <raw file>
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/JamesPagetButler/qbp-compute-unit/emulator/mdcu"
)

func main() {
	bench := flag.String("bench", "", "raw benchmark output file")
	flag.Parse()
	b, err := os.ReadFile(*bench)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	r, err := mdcu.W2DiagReport(mdcu.ParseBenchOutput(string(b)))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(r)
}
