package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/hweval"
)

func generateReport(ownedGPUs int) {
	modes := []hweval.BuildMode{
		hweval.ModeBruteForce,
		hweval.ModeEfficiency,
		hweval.ModeRiscvBeast,
		hweval.ModeRiscvOptimized,
	}

	filename := "walk_options_report.md"
	out, err := os.Create(filename)
	if err != nil {
		fmt.Println("Failed to create report:", err)
		return
	}
	defer out.Close()

	fmt.Fprintf(out, "# Walk Phase Hardware Options Comparison\n\n")
	fmt.Fprintf(out, "This report compares the four evaluated hardware paths for the QBP Compute Unit, factoring in your existing inventory (%d owned RX 9070 XT).\n\n", ownedGPUs)
	
	fmt.Fprintf(out, "| Mode | Target CPU | Est. Cost | System TDP | Est. Throughput (QMULs/sec) | Energy Efficiency (Joules/QMUL) | Cost Efficiency (USD per Billion QMULs/s) |\n")
	fmt.Fprintf(out, "| :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")

	for _, mode := range modes {
		report := hweval.Evaluate(mode, ownedGPUs)
		if report.CPU.ID == "" {
			continue
		}
		perf := hweval.EstimatePerformance(report.CPU)

		joulesPerQmul := report.TotalTDP / perf.EstimatedQMULs
		costPerBillion := report.TotalCost / (perf.EstimatedQMULs / 1e9)

		fmt.Fprintf(out, "| **%s** | %s | $%.2f | %.0fW | %.2f Billion | %.2e J/Op | $%.2f |\n",
			mode, report.CPU.Name, report.TotalCost, report.TotalTDP, perf.EstimatedQMULs/1e9, joulesPerQmul, costPerBillion)
	}

	fmt.Fprintf(out, "\n## Key Takeaways\n")
	fmt.Fprintf(out, "- **Energy Efficiency (Joules/QMUL):** Lower is better. This measures how much power is required for a single mathematical operation.\n")
	fmt.Fprintf(out, "- **Cost Efficiency:** Lower is better. This measures how much you pay (in hardware CapEx) for every Billion QMULs per second of capacity.\n")

	fmt.Printf("Report successfully written to %s\n", filename)
}

func main() {
	modeStr := flag.String("mode", "efficiency", "Hardware build mode: bruteforce, efficiency, riscv-beast, riscv-optimized, server")
	ownedGPUs := flag.Int("own-gpu", 0, "Number of RX 9070 XT GPUs already owned")
	reportFlag := flag.Bool("report", false, "Generate a markdown report comparing all paths")

	flag.Parse()

	if *reportFlag {
		generateReport(*ownedGPUs)
		return
	}

	var mode hweval.BuildMode
	switch *modeStr {
	case "bruteforce":
		mode = hweval.ModeBruteForce
	case "efficiency":
		mode = hweval.ModeEfficiency
	case "riscv-beast":
		mode = hweval.ModeRiscvBeast
	case "riscv-optimized":
		mode = hweval.ModeRiscvOptimized
	case "server":
		mode = hweval.ModeServer
	default:
		fmt.Printf("Unknown mode: %s\n", *modeStr)
		os.Exit(1)
	}

	report := hweval.Evaluate(mode, *ownedGPUs)

	fmt.Printf("=========================================\n")
	fmt.Printf("QBP Hardware Evaluator\n")
	fmt.Printf("Mode: %s\n", report.Mode)
	fmt.Printf("Owned GPUs: %d\n", *ownedGPUs)
	fmt.Printf("=========================================\n\n")

	fmt.Printf("Part List:\n")
	fmt.Printf("%-40s | %-5s | %-10s | %-10s\n", "Component", "Qty", "Cost", "TDP")
	fmt.Printf("-----------------------------------------+-------+------------+------------\n")

	for _, p := range report.Parts {
		costStr := fmt.Sprintf("$%.2f", p.Cost)
		if p.Cost == 0 && p.Component.Cost > 0 {
			costStr = "OWNED"
		}
		fmt.Printf("%-40s | %-5d | %-10s | %.0fW\n", p.Component.Name, p.Quantity, costStr, p.Component.TDPWatts*float64(p.Quantity))
	}

	fmt.Printf("\n=========================================\n")
	fmt.Printf("TOTAL ESTIMATED COST: $%.2f\n", report.TotalCost)
	fmt.Printf("TOTAL MAXIMUM TDP:    %.0fW\n", report.TotalTDP)
	fmt.Printf("=========================================\n\n")

	// Performance Estimation
	if report.CPU.ID != "" {
		perf := hweval.EstimatePerformance(report.CPU)
		fmt.Printf("=========================================\n")
		fmt.Printf("Performance Estimate vs FX-8350 Baseline \n")
		fmt.Printf("=========================================\n")
		fmt.Printf("Target CPU: %s\n", report.CPU.Name)
		fmt.Printf("Estimated QMULs/sec: %.2f Billion\n", perf.EstimatedQMULs/1e9)
		fmt.Printf("Speedup Factor:      %.2fx Faster\n", perf.SpeedupFactor)
		fmt.Printf("=========================================\n")
	}
}
