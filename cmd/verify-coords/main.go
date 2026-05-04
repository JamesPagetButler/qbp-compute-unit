package main

import (
	"fmt"
	"math"
)

func main() {
	// Milton Target: 22.5N, 95.5W
	lat := 22.5
	lon := -95.5

	phi := lat * math.Pi / 180.0
	theta := lon * math.Pi / 180.0

	// Standard ECEF
	x := math.Cos(phi) * math.Cos(theta)
	y := math.Cos(phi) * math.Sin(theta)
	z := math.Sin(phi)

	fmt.Printf("GEOGRAPHIC GROUND TRUTH:\n")
	fmt.Printf("Location:  %f, %f (Gulf of Mexico)\n", lat, lon)
	fmt.Printf("ECEF Vector: [%.4f, %.4f, %.4f]\n", x, y, z)
	fmt.Printf("\nREQUIRED MAPPING TO THREE.JS (Y-UP):\n")
	fmt.Printf("JS X (East)  = ECEF Y: %.4f\n", y)
	fmt.Printf("JS Y (North) = ECEF Z: %.4f\n", z)
	fmt.Printf("JS Z (Prime) = ECEF X: %.4f\n", x)
}
