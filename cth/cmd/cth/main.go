package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/helpful-engineering/cth/compute"
	"github.com/helpful-engineering/cth/report"
	"github.com/helpful-engineering/cth/store"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "analyse":
		analyseCmd(os.Args[2:])
	case "merge":
		mergeCmd(os.Args[2:])
	case "health":
		healthCmd(os.Args[2:])
	case "compare":
		compareCmd(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("CTH: Confluent Trust Hypergraph Engine")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cth analyse <inventory.json>                    Full analysis report")
	fmt.Println("  cth merge   <inv_a.json> <inv_b.json>           Merge two programmes")
	fmt.Println("  cth health  <inventory.json>                    Epistemic health dashboard")
	fmt.Println("  cth compare <inv_old.json> <inv_new.json>       Compression velocity")
	fmt.Println()
	fmt.Println("Flags (all commands):")
	fmt.Println("  -o <file>   Write output to file instead of stdout")
}

// ── analyse ─────────────────────────────────────────────────────────────────

func analyseCmd(args []string) {
	fs := flag.NewFlagSet("analyse", flag.ExitOnError)
	out := fs.String("o", "", "output file (default: stdout)")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: cth analyse <inventory.json> [-o output.md]")
		os.Exit(1)
	}

	inv, err := store.LoadInventory(fs.Arg(0))
	if err != nil {
		fatalf("load: %v", err)
	}

	inputMap := compute.BuildInputEntropy(inv)
	analysis := report.Analyse(inv, inputMap)
	text := report.MarkdownReport(inv, analysis)

	writeOutput(*out, text)
}

// ── merge ────────────────────────────────────────────────────────────────────

func mergeCmd(args []string) {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	out := fs.String("o", "", "save merged inventory to file")
	_ = fs.Parse(args)

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: cth merge <inv_a.json> <inv_b.json> [-o merged.json]")
		os.Exit(1)
	}

	invA, err := store.LoadInventory(fs.Arg(0))
	if err != nil {
		fatalf("load A: %v", err)
	}
	invB, err := store.LoadInventory(fs.Arg(1))
	if err != nil {
		fatalf("load B: %v", err)
	}

	merged, r := compute.MergeProgrammes(invA, invB)

	fmt.Printf("Merge: %s\n", merged.Programme)
	fmt.Printf("Shared anchors:      %d\n", len(r.SharedAnchorIDs))
	fmt.Printf("Theoretical deficit: %.2f bits\n", r.TheoreticalDeficit)
	fmt.Printf("Engineering deficit: %.2f bits\n", r.EngineeringDeficit)
	fmt.Printf("Zero theoretical:    %v\n", r.ZeroTheoretical)
	fmt.Printf("Lossless Tier 1:     %v\n", r.Lossless)

	if *out != "" {
		if err := store.SaveInventory(merged, *out); err != nil {
			fatalf("save: %v", err)
		}
		fmt.Printf("Merged inventory written to %s\n", *out)
	}
}

// ── health ───────────────────────────────────────────────────────────────────

func healthCmd(args []string) {
	fs := flag.NewFlagSet("health", flag.ExitOnError)
	out := fs.String("o", "", "output file (default: stdout)")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: cth health <inventory.json>")
		os.Exit(1)
	}

	inv, err := store.LoadInventory(fs.Arg(0))
	if err != nil {
		fatalf("load: %v", err)
	}

	inputMap := compute.BuildInputEntropy(inv)
	analysis := report.Analyse(inv, inputMap)
	text := report.Dashboard(inv, analysis)

	writeOutput(*out, text)
}

// ── compare ──────────────────────────────────────────────────────────────────

func compareCmd(args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	_ = fs.Parse(args)

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: cth compare <inv_old.json> <inv_new.json>")
		os.Exit(1)
	}

	invOld, err := store.LoadInventory(fs.Arg(0))
	if err != nil {
		fatalf("load old: %v", err)
	}
	invNew, err := store.LoadInventory(fs.Arg(1))
	if err != nil {
		fatalf("load new: %v", err)
	}

	rhoOld, _ := compute.NetCompression(invOld, compute.BuildInputEntropy(invOld))
	rhoNew, _ := compute.NetCompression(invNew, compute.BuildInputEntropy(invNew))

	nOld := len(invOld.Axioms) + len(invOld.DerivedPrinciples) + len(invOld.Anchors) + len(invOld.Inputs)
	nNew := len(invNew.Axioms) + len(invNew.DerivedPrinciples) + len(invNew.Anchors) + len(invNew.Inputs)

	velocity := compute.CompressionVelocity(
		compute.VersionSnapshot{Rho: rhoOld, NAnchor: nOld},
		compute.VersionSnapshot{Rho: rhoNew, NAnchor: nNew},
	)

	fmt.Printf("Compression velocity: Δρ/Δn\n")
	fmt.Printf("  Old: %s  ρ=%.4f  n=%d\n", invOld.Programme, rhoOld, nOld)
	fmt.Printf("  New: %s  ρ=%.4f  n=%d\n", invNew.Programme, rhoNew, nNew)
	fmt.Printf("  Δρ=%.4f  Δn=%d  v=%.6f bits/anchor\n",
		rhoNew-rhoOld, nNew-nOld, velocity)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func writeOutput(path, text string) {
	if path == "" {
		fmt.Print(text)
		return
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		fatalf("write output: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cth: "+format+"\n", args...)
	os.Exit(1)
}
