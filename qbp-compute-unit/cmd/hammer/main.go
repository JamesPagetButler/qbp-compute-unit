// Command hammer runs the Multi-Wheel Hammer Off-Road Vehicle simulation.
//
// This is the macro-scale benchmark for the QBP Sense-Compute-Act pipeline.
//
// Usage:
//
//	go run cmd/hammer/main.go [-speed 15] [-duration 2] [-roughness 0.5]
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/hammer"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/mesh"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/qword"
)

func main() {
	speed := flag.Float64("speed", 15.0, "vehicle speed (m/s)")
	duration := flag.Float64("duration", 2.0, "simulation duration (seconds)")
	roughness := flag.Float64("roughness", 0.5, "terrain roughness (0-1)")
	rockPos := flag.Float64("rock", 12.0, "rock position (metres)")
	rockH := flag.Float64("rock-height", 0.15, "rock height (metres)")
	flag.Parse()

	banner := strings.Repeat("=", 60)
	fmt.Println(banner)
	fmt.Println("QBP COMPUTE UNIT — HAMMER VEHICLE SIMULATION")
	fmt.Println("Macro-Scale Sense-Compute-Act Pipeline Benchmark")
	fmt.Println(banner)
	fmt.Println()

	cfg := hammer.SimConfig{
		Speed:        *speed,
		Duration:     *duration,
		TerrainRough: *roughness,
		RockPosition: *rockPos,
		RockHeight:   *rockH,
	}

	fmt.Println("Running simulation...")
	result := hammer.RunSimulation(cfg)
	fmt.Println()
	fmt.Println(hammer.FormatResult(result))

	// ── Mesh Scheduler Demonstration ──
	fmt.Println()
	fmt.Println(banner)
	fmt.Println("MESH SCHEDULER — DYNAMIC PRECISION ALLOCATION")
	fmt.Println(banner)
	fmt.Println()

	sched := mesh.NewScheduler(1, qword.W64) // 1 Fano cell, native QW64
	fmt.Println("Mesh: 1 Fano cell (7 nodes), native QW64")
	fmt.Println()

	// Submit vehicle tasks with different precision requirements
	tasks := []*mesh.Task{
		{ID: "traction-FL", CompositionDepth: 50, DriftTolerance: 1e-3},
		{ID: "traction-FR", CompositionDepth: 50, DriftTolerance: 1e-3},
		{ID: "traction-ML", CompositionDepth: 50, DriftTolerance: 1e-3},
		{ID: "stability",   CompositionDepth: 500, DriftTolerance: 1e-6},
		{ID: "trajectory",  CompositionDepth: 5000, DriftTolerance: 1e-9},
		{ID: "chassis-int", CompositionDepth: 200, DriftTolerance: 1e-6},
		{ID: "watchdog",    CompositionDepth: 100, DriftTolerance: 1e-3},
	}

	fmt.Printf("  %-16s %8s %10s %10s %8s\n", "Task", "Depth", "Tolerance", "Width", "Nodes")
	fmt.Printf("  %-16s %8s %10s %10s %8s\n", "────────────────", "────────", "──────────", "──────────", "────────")

	for _, t := range tasks {
		err := sched.Submit(t)
		status := "OK"
		if err != nil {
			status = "FULL"
		}
		fmt.Printf("  %-16s %8d %10.0e %10s %6d  [%s]\n",
			t.ID, t.CompositionDepth, t.DriftTolerance,
			t.RequiredWidth, t.NodesNeeded, status)
	}

	fmt.Println()
	st := sched.Status()
	fmt.Println(" ", st)
	fmt.Println()

	// Show what happens during the rock strike — watchdog triggers reallocation
	fmt.Println("── Rock Strike Event: Watchdog Reallocation ──")
	fmt.Println()

	// Simulate: traction-FL detects instability, needs precision upgrade
	if t, ok := sched.Tasks["traction-FL"]; ok {
		t.ActualDepth = 30
		t.ActualDrift = 5e-4 // elevated drift from rock strike
		advice := mesh.CheckReallocation(t)
		fmt.Printf("  traction-FL: %s\n", advice)
	}

	// Simulate: trajectory task running cleanly, could downgrade
	if t, ok := sched.Tasks["trajectory"]; ok {
		t.ActualDepth = 2000
		t.ActualDrift = 1e-15 // very clean — could use lower precision
		advice := mesh.CheckReallocation(t)
		fmt.Printf("  trajectory:  %s\n", advice)
	}

	fmt.Println()
	fmt.Println("  The mesh breathes: tasks that need more precision get more nodes.")
	fmt.Println("  Tasks running cleaner than predicted release nodes back to the pool.")
	fmt.Println()

	// Estimation for different scenarios
	fmt.Println("── Precision Estimates for Common Vehicle Tasks ──")
	fmt.Println()
	scenarios := []struct{ name string; depth int64; tol float64 }{
		{"ABS pulse (single brake)", 10, 1e-3},
		{"Traction per wheel/tick", 50, 1e-3},
		{"Stability 100ms window", 500, 1e-6},
		{"Trajectory 1s lookahead", 5000, 1e-9},
		{"Terrain map 10s window", 50000, 1e-9},
		{"Mission log 1hr continuous", 1800000, 1e-12},
	}

	fmt.Printf("  %-30s %10s %10s %6s\n", "Scenario", "Depth", "Tolerance", "Width")
	fmt.Printf("  %-30s %10s %10s %6s\n", "──────────────────────────────", "──────────", "──────────", "──────")
	for _, sc := range scenarios {
		w, _, _ := mesh.EstimateAllocation(sc.depth, sc.tol, qword.W64)
		fmt.Printf("  %-30s %10d %10.0e %6s\n", sc.name, sc.depth, sc.tol, w)
	}

	fmt.Println()
	fmt.Println(banner)
}
