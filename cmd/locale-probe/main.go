package main

import (
	"fmt"
	"math"
)

func main() {
	probes := []struct {
		name     string
		lat, lon float64
	}{
		{"NORTH POLE", 90, 0},
		{"LONDON (0,0)", 0, 0},
		{"GULF (90W, 25N)", 25, -90},
		{"INDIA (90E, 20N)", 20, 90},
	}

	fmt.Printf("%-20s | %-20s | %-20s\n", "LOCATION", "ECEF (GO)", "THREE.JS (JS)")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, p := range probes {
		phi := p.lat * math.Pi / 180.0
		theta := p.lon * math.Pi / 180.0

		// STANDARD ECEF (Go): X=Prime, Y=90E, Z=North
		gx := math.Cos(phi) * math.Cos(theta)
		gy := math.Cos(phi) * math.Sin(theta)
		gz := math.Sin(phi)

		// MAPPING TO THREE.JS Y-UP:
		// Go X (Prime) -> JS Z+
		// Go Y (90E)   -> JS X+
		// Go Z (North) -> JS Y+
		jx := gy
		jy := gz
		jz := gx

		fmt.Printf("%-20s | [%.2f, %.2f, %.3f] | [%.2f, %.2f, %.2f]\n",
			p.name, gx, gy, gz, jx, jy, jz)
	}
}
