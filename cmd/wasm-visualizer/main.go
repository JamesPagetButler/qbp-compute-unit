package main

import (
	"fmt"
	"math"
	"syscall/js"
	"time"
	"unsafe"
	"github.com/JamesPagetButler/qbp-emulator"
)

var cpu *emulator.CPU

func main() {
	fmt.Println("QBP WASM Visualizer: BASELINE AUDIT BOOT...")
	cpu = emulator.NewCPU()
	cpu.SetWidth(emulator.W128)

	js.Global().Set("qbp_populate", js.FuncOf(qbpPopulate))
	js.Global().Set("qbp_step", js.FuncOf(qbpStep))
	js.Global().Set("qbp_get_node_count", js.FuncOf(qbpGetNodeCount))
	js.Global().Set("qbp_get_nodes_buffer", js.FuncOf(qbpGetNodesBuffer))
	js.Global().Set("qbp_get_ri_signal", js.FuncOf(qbpGetRISignal))
	js.Global().Set("qbp_get_latency", js.FuncOf(qbpGetLatency))

	select {}
}

var lastLatency float64

func qbpGetLatency(this js.Value, args []js.Value) any {
	return lastLatency
}

func qbpGetRISignal(this js.Value, args []js.Value) any {
	totalTension := 0.0
	count := 0
	baseCount := 18 * 36
	for i := baseCount; i < len(cpu.Memory)/4; i++ {
		node := cpu.GetClimateNode(i)
		w, _ := node.Spin.W.Float64()
		if w > 0.5 {
			totalTension += w
			count++
		}
	}
	if count == 0 { return 0.0 }
	avg := totalTension / float64(count)
	return math.Min(1.0, avg / 15.0)
}

func qbpPopulate(this js.Value, args []js.Value) any {
	res := args[0].Int()
	cpu.PopulateSphere(res)
	
	phi := 22.5 * math.Pi / 180.0
	theta := -95.5 * math.Pi / 180.0
	mX := math.Cos(phi) * math.Cos(theta)
	mY := math.Cos(phi) * math.Sin(theta)
	mZ := math.Sin(phi)

	cpu.InjectDenseNodes(mX, mY, mZ, 0.03, 10000) 
	cpu.InjectDenseNodes(mX, mY, mZ, 0.15, 30000) 
	return nil
}

var miltonLat = 22.5
var miltonLon = -95.5
var miltonSpin = 0.4

func qbpStep(this js.Value, args []js.Value) any {
	start := time.Now()
	dt := 0.01
	if len(args) > 0 { dt = args[0].Float() }

	// Mode Control: 0=Milton, 1=Pole-Crossing
	mode := 0
	if len(args) > 1 { mode = args[1].Int() }

	if mode == 1 {
		// POLE-CROSSING MODE: Drive storm toward 90N
		miltonLat += 1.0 * dt
		if miltonLat > 90.0 { miltonLat = -90.0 }
		miltonLon = 0.0
	} else {
		// MILTON TRACK
		miltonLat += 0.4 * dt 
		miltonLon += 1.1 * dt 
	}
	
	miltonSpin += 0.12 * dt

	phi := miltonLat * math.Pi / 180.0
	theta := miltonLon * math.Pi / 180.0
	mX := math.Cos(phi) * math.Cos(theta)
	mY := math.Cos(phi) * math.Sin(theta)
	mZ := math.Sin(phi)

	for i := 0; i < len(cpu.Memory)/4; i++ {
		node := cpu.GetClimateNode(i)
		nX, _ := node.Pos.X.Float64()
		nY, _ := node.Pos.Y.Float64()
		nZ, _ := node.Pos.Z.Float64()
		
		dx, dy, dz := nX-mX, nY-mY, nZ-mZ
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		
		if dist < 0.2 {
			angle := math.Atan2(dy, dx)
			spiral := math.Sin(angle*4.0 - dist*30.0) 
			
			intensity := 0.0
			// Model the eye: zero intensity at dist=0, peaking at dist=0.03 (eyewall)
			if dist > 0.015 {
				// Eyewall and rainbands
				normalizedDist := dist - 0.015
				intensity = (miltonSpin * normalizedDist) / (normalizedDist*normalizedDist + 0.001)
			}
			
			// Enhance spiral effect: sharper rainbands
			rainbandMod := 1.0 + math.Max(0, spiral * 1.5)
			node.Spin.W.SetFloat64(intensity * rainbandMod)
		} else {
			node.Spin.W.SetFloat64(0)
		}
	}
	
	lastLatency = float64(time.Since(start).Microseconds())
	return nil
}

func qbpGetNodeCount(this js.Value, args []js.Value) any { return len(cpu.Memory) / 4 }

var posBuffer []float32

func qbpGetNodesBuffer(this js.Value, args []js.Value) any {
	nodeCount := len(cpu.Memory) / 4
	if len(posBuffer) != nodeCount*4 {
		posBuffer = make([]float32, nodeCount*4)
	}
	for i := 0; i < nodeCount; i++ {
		node := cpu.GetClimateNode(i)
		x, _ := node.Pos.X.Float64()
		y, _ := node.Pos.Y.Float64()
		z, _ := node.Pos.Z.Float64()
		w, _ := node.Spin.W.Float64()
		posBuffer[i*4+0] = float32(x)
		posBuffer[i*4+1] = float32(y)
		posBuffer[i*4+2] = float32(z)
		posBuffer[i*4+3] = float32(w)
	}
	
	// Convert posBuffer []float32 to []byte
	byteSlice := unsafe.Slice((*byte)(unsafe.Pointer(&posBuffer[0])), len(posBuffer)*4)
	jsArray := args[0] // Expecting a Uint8Array from JS
	js.CopyBytesToJS(jsArray, byteSlice)
	return nil
}
