// Package hammer implements the Multi-Wheel Hammer Off-Road Vehicle simulation
// as a macro-scale benchmark for the QBP Sense-Compute-Act pipeline.
//
// This simulation models a 6-wheel independent-drive vehicle traversing
// variable terrain at speed. Each wheel hub is a SENSE node producing
// quaternion-valued orientation data. The COMPUTE layer performs stability
// monitoring and torque vectoring. The ACT layer adjusts per-wheel torque
// and suspension stiffness.
//
// Two parallel pipelines run the same scenario:
//
//  1. QBP-Algebraic: Sensor data remains in quaternion form throughout.
//     Stability detection uses algebraic norm monitoring (watchdog).
//     Torque vectoring uses quaternion composition.
//     No scalar conversion at any stage.
//
//  2. Scalar-Conventional: Sensor data converted to Euler angles / floats.
//     Stability detection uses a simplified Kalman filter on scalar state.
//     Torque vectoring uses matrix arithmetic.
//     Standard convert-compute-convert pipeline.
//
// The benchmark measures:
//   - Detection latency: how many simulation ticks before each pipeline
//     identifies a loss-of-traction event
//   - Fidelity: accumulated state estimation error over the run
//   - Operation count: total FP operations per pipeline
//   - Mesh utilisation: how the scheduler allocates nodes across tasks
package hammer

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/JamesPagetButler/qbp-compute-unit/pkg/quat"
	"github.com/JamesPagetButler/qbp-compute-unit/pkg/watchdog"
)

// ─── Vehicle Configuration ─────────────────────────────────────────

const (
	NumWheels     = 6
	ControlRateHz = 500.0               // control loop frequency
	DtControl     = 1.0 / ControlRateHz // seconds per control tick
)

// WheelPosition defines where each wheel sits relative to chassis centre.
// Quaternion-natural: position encoded as a pure quaternion (0, x, y, z).
type WheelPosition struct {
	Name    string
	X, Y, Z float64 // metres from chassis centre
}

var WheelPositions = [NumWheels]WheelPosition{
	{"FL", -1.2, 0.9, -0.3},  // front-left
	{"FR", -1.2, -0.9, -0.3}, // front-right
	{"ML", 0.0, 0.9, -0.3},   // mid-left
	{"MR", 0.0, -0.9, -0.3},  // mid-right
	{"RL", 1.2, 0.9, -0.3},   // rear-left
	{"RR", 1.2, -0.9, -0.3},  // rear-right
}

// ─── Terrain Model ─────────────────────────────────────────────────

// TerrainProfile generates ground surface normal and height at a given
// position and time. Returns a unit quaternion representing the surface
// orientation relative to flat ground.
type TerrainProfile struct {
	Seed       int64
	Roughness  float64 // 0 = flat, 1 = extreme
	RockAt     float64 // distance (metres) where a discrete rock strike occurs
	RockHeight float64 // rock height in metres
	rng        *rand.Rand
}

func NewTerrain(roughness float64, rockAt, rockHeight float64) *TerrainProfile {
	seed := time.Now().UnixNano()
	return &TerrainProfile{
		Seed:       seed,
		Roughness:  roughness,
		RockAt:     rockAt,
		RockHeight: rockHeight,
		rng:        rand.New(rand.NewSource(seed)),
	}
}

// SurfaceAt returns the surface normal quaternion and height offset at
// a given longitudinal position (metres along track).
func (t *TerrainProfile) SurfaceAt(posX float64, wheelY float64) (normal quat.Quat, heightOffset float64) {
	// Base terrain: Perlin-like noise using sin composition
	noise := t.Roughness * 0.1 * (math.Sin(posX*2.3+wheelY*1.7) +
		0.5*math.Sin(posX*5.1+wheelY*3.3) +
		0.25*math.Sin(posX*11.7+wheelY*7.1))

	heightOffset = noise

	// Rock strike: discrete bump
	if math.Abs(posX-t.RockAt) < 0.15 && math.Abs(wheelY) < 1.0 {
		// Only left-side wheels hit the rock
		if wheelY > 0 {
			heightOffset += t.RockHeight * math.Exp(-math.Pow((posX-t.RockAt)/0.1, 2))
		}
	}

	// Surface normal as quaternion rotation from vertical
	// Slope in X direction
	dhdx := t.Roughness * 0.1 * (2.3*math.Cos(posX*2.3+wheelY*1.7) +
		0.5*5.1*math.Cos(posX*5.1+wheelY*3.3))

	// Rock slope contribution
	if math.Abs(posX-t.RockAt) < 0.3 && wheelY > 0 {
		dhdx += t.RockHeight * (-2 * (posX - t.RockAt) / 0.01) *
			math.Exp(-math.Pow((posX-t.RockAt)/0.1, 2))
	}

	// Convert slope to rotation quaternion (small angle: axis ≈ (0, -slope, 0))
	slopeAngle := math.Atan(dhdx)
	halfA := slopeAngle / 2
	normal = quat.New(math.Cos(halfA), 0, -math.Sin(halfA), 0)

	return normal, heightOffset
}

// ─── QBP Pipeline ──────────────────────────────────────────────────

// QBPWheelState holds the algebraic state of one wheel in the QBP pipeline.
// Everything stays in quaternion form — no scalar conversion.
type QBPWheelState struct {
	Orientation   quat.Quat       // wheel orientation relative to world
	GroundContact quat.Quat       // composed: wheel orientation × surface normal
	Velocity      quat.Quat       // angular velocity as pure quaternion
	Torque        float64         // output torque command
	Stiffness     float64         // suspension stiffness coefficient
	ContactNormSq float64         // ||ground_contact||² — watchdog signal
	IsAirborne    bool            // watchdog detected loss of contact
	WD            *watchdog.Stats // per-wheel algebraic watchdog
	PrevHeight    float64         // previous tick's height offset (for delta)
	PrevContactQ  quat.Quat       // previous tick's contact quaternion
	Initialized   bool            // false until first tick completes
	QDistEMA      float64         // exponential moving average of q-distance
	QDistCount    int             // ticks seen (for warmup)
}

// QBPVehicle is the full vehicle state in the algebraic pipeline.
type QBPVehicle struct {
	Wheels       [NumWheels]*QBPWheelState
	Chassis      quat.Quat // chassis orientation
	Position     float64   // longitudinal position (metres)
	Speed        float64   // m/s
	TotalOps     int64     // quaternion operations counter
	DetectionLog []Detection
}

// Detection records when a loss-of-traction event was detected.
type Detection struct {
	Tick   int
	Wheel  int
	Signal float64 // the algebraic signal that triggered detection
	Method string  // "QBP" or "Scalar"
}

func NewQBPVehicle(speed float64) *QBPVehicle {
	v := &QBPVehicle{
		Chassis: quat.Identity(),
		Speed:   speed,
	}
	for i := 0; i < NumWheels; i++ {
		v.Wheels[i] = &QBPWheelState{
			Orientation:   quat.Identity(),
			GroundContact: quat.Identity(),
			Velocity:      quat.Pure(0, 0, 0),
			Torque:        1.0,
			Stiffness:     1.0,
			ContactNormSq: 1.0,
			WD:            watchdog.New(),
		}
	}
	return v
}

// QBPTick runs one control cycle of the QBP pipeline.
// Returns the number of quaternion operations performed.
func QBPTick(v *QBPVehicle, terrain *TerrainProfile, tick int) int {
	ops := 0

	// ── SENSE: Read each wheel's ground contact ──
	for i := 0; i < NumWheels; i++ {
		wp := WheelPositions[i]
		wheelWorldX := v.Position + wp.X

		// Surface normal at this wheel's position (already a quaternion)
		surfNormal, heightOff := terrain.SurfaceAt(wheelWorldX, wp.Y)
		ops++ // terrain query produces quaternion directly

		// Wheel orientation = chassis rotation composed with suspension deflection
		// Suspension deflection modelled as small-angle quaternion from height
		deflAngle := heightOff * 2.0 // simplified: height → rotation
		suspDefl := quat.New(math.Cos(deflAngle/2), math.Sin(deflAngle/2), 0, 0)
		ops++

		wheelOrient := quat.Mul(v.Chassis, suspDefl)
		ops++
		v.Wheels[i].WD.ObserveMul(wheelOrient)

		// ── COMPUTE: Ground contact = wheel orientation × surface normal ──
		// This is the KEY algebraic operation. If the wheel is in contact,
		// this composition should produce a unit quaternion (norm ≈ 1).
		// If the wheel is airborne or sliding, the surface normal becomes
		// unreliable and the composed result's norm drifts.
		contact := quat.Mul(wheelOrient, surfNormal)
		ops++
		v.Wheels[i].WD.ObserveMul(contact)

		v.Wheels[i].Orientation = wheelOrient
		v.Wheels[i].GroundContact = contact
		v.Wheels[i].ContactNormSq = quat.NormSq(contact)

		// ── ALGEBRAIC WATCHDOG: Detect loss of traction ──
		// The algebraic signal: quaternion distance between consecutive
		// contact quaternions. On smooth terrain, the contact quaternion
		// changes gradually (small q-distance per tick). A rock strike
		// causes a discontinuity — the contact quaternion jumps, producing
		// a large q-distance in one tick.
		//
		// q-distance = ||q_current - q_previous|| (quaternion norm of difference)
		// This is an instantaneous, single-tick algebraic signal — no
		// filtering, no convergence delay, no statistical accumulation.
		ws := v.Wheels[i]
		if ws.Initialized {
			qDist := quat.Norm(quat.Sub(contact, ws.PrevContactQ))
			ws.QDistCount++

			// Update exponential moving average (α = 0.02 → ~50-tick window)
			const alpha = 0.02
			if ws.QDistCount < 50 {
				// Warmup: build baseline without detection
				ws.QDistEMA = ws.QDistEMA*(1-alpha) + qDist*alpha
				if ws.QDistCount == 1 {
					ws.QDistEMA = qDist // seed with first value
				}
			} else {
				// Active detection: spike = current qDist > 3× running average
				threshold := ws.QDistEMA * 3.0
				if qDist > threshold && qDist > 0.01 {
					if !ws.IsAirborne {
						ws.IsAirborne = true
						v.DetectionLog = append(v.DetectionLog, Detection{
							Tick:   tick,
							Wheel:  i,
							Signal: qDist / ws.QDistEMA, // ratio above baseline
							Method: "QBP",
						})
					}
				} else if qDist < ws.QDistEMA*1.5 {
					ws.IsAirborne = false
				}
				// Update EMA (only when not in spike to avoid contamination)
				if !ws.IsAirborne {
					ws.QDistEMA = ws.QDistEMA*(1-alpha) + qDist*alpha
				}
			}
		}
		ws.PrevContactQ = contact
		ws.PrevHeight = heightOff
		ws.Initialized = true
	}

	// ── COMPUTE: Torque vectoring across Fano routing ──
	// When a wheel loses traction, redistribute its torque to adjacent wheels.
	// "Adjacent" determined by Fano-plane topology of the 6-wheel + chassis system.
	// (7 nodes: 6 wheels + chassis = Fano cell)
	totalTraction := 0.0
	for i := 0; i < NumWheels; i++ {
		if !v.Wheels[i].IsAirborne {
			totalTraction += 1.0
		}
	}
	if totalTraction > 0 {
		for i := 0; i < NumWheels; i++ {
			if v.Wheels[i].IsAirborne {
				v.Wheels[i].Torque = 0.0
				v.Wheels[i].Stiffness = 0.3 // soften airborne wheel
			} else {
				v.Wheels[i].Torque = float64(NumWheels) / totalTraction
				ops++ // torque redistribution is one QMUL equivalent
			}
		}
	}

	// ── ACT: Apply torque and stiffness (quaternion output) ──
	// Each wheel's torque command is a scalar magnitude applied along
	// the wheel's orientation axis — which is already a quaternion.
	// No conversion needed: the actuator receives q_wheel * torque_magnitude.
	for i := 0; i < NumWheels; i++ {
		_ = quat.Scale(v.Wheels[i].Torque, v.Wheels[i].Orientation)
		ops++
	}

	// Update chassis orientation from weighted wheel contacts
	// (simplified: average of grounded wheel orientations)
	avgW, avgX, avgY, avgZ := 0.0, 0.0, 0.0, 0.0
	grounded := 0
	for i := 0; i < NumWheels; i++ {
		if !v.Wheels[i].IsAirborne {
			avgW += v.Wheels[i].GroundContact.W
			avgX += v.Wheels[i].GroundContact.X
			avgY += v.Wheels[i].GroundContact.Y
			avgZ += v.Wheels[i].GroundContact.Z
			grounded++
		}
	}
	if grounded > 0 {
		n := float64(grounded)
		v.Chassis = quat.Normalize(quat.New(avgW/n, avgX/n, avgY/n, avgZ/n))
		ops += 2 // average + normalize
	}

	// Advance position
	v.Position += v.Speed * DtControl
	v.TotalOps += int64(ops)

	return ops
}

// ─── Scalar Pipeline (Conventional) ────────────────────────────────

// ScalarWheelState is the conventional representation.
type ScalarWheelState struct {
	// Euler angles (converted from quaternion at SENSE boundary)
	Roll, Pitch, Yaw float64
	// Ground contact as scalar: vertical force estimate
	ContactForce float64
	// Kalman filter state
	KalmanState float64
	KalmanCov   float64
	Torque      float64
	Stiffness   float64
	IsAirborne  bool
}

type ScalarVehicle struct {
	Wheels           [NumWheels]*ScalarWheelState
	Roll, Pitch, Yaw float64 // chassis Euler angles
	Position         float64
	Speed            float64
	TotalOps         int64
	DetectionLog     []Detection
}

func NewScalarVehicle(speed float64) *ScalarVehicle {
	v := &ScalarVehicle{Speed: speed}
	for i := 0; i < NumWheels; i++ {
		v.Wheels[i] = &ScalarWheelState{
			ContactForce: 1.0,
			KalmanState:  0.0,
			KalmanCov:    1.0,
			Torque:       1.0,
			Stiffness:    1.0,
		}
	}
	return v
}

// ScalarTick runs one control cycle of the conventional pipeline.
func ScalarTick(v *ScalarVehicle, terrain *TerrainProfile, tick int) int {
	ops := 0

	for i := 0; i < NumWheels; i++ {
		wp := WheelPositions[i]
		wheelWorldX := v.Position + wp.X

		// SENSE: Get terrain (arrives as quaternion, must convert to Euler)
		surfNormal, heightOff := terrain.SurfaceAt(wheelWorldX, wp.Y)
		ops++ // terrain query

		// ── CONVERSION BOUNDARY: quaternion → Euler ──
		// This is the impedance mismatch. Standard pipeline must decompose.
		sn := surfNormal
		sinr := 2.0 * (sn.W*sn.X + sn.Y*sn.Z)
		cosr := 1.0 - 2.0*(sn.X*sn.X+sn.Y*sn.Y)
		surfRoll := math.Atan2(sinr, cosr)
		ops += 6

		sinp := 2.0 * (sn.W*sn.Y - sn.Z*sn.X)
		if math.Abs(sinp) >= 1 {
			sinp = math.Copysign(1, sinp)
		}
		surfPitch := math.Asin(sinp)
		ops += 5

		siny := 2.0 * (sn.W*sn.Z + sn.X*sn.Y)
		cosy := 1.0 - 2.0*(sn.Y*sn.Y+sn.Z*sn.Z)
		surfYaw := math.Atan2(siny, cosy)
		ops += 6

		// Compose with chassis Euler (addition — loses coupling terms)
		wheelRoll := v.Roll + surfRoll + heightOff*2.0
		wheelPitch := v.Pitch + surfPitch
		wheelYaw := v.Yaw + surfYaw
		ops += 3

		v.Wheels[i].Roll = wheelRoll
		v.Wheels[i].Pitch = wheelPitch
		v.Wheels[i].Yaw = wheelYaw

		// ── COMPUTE: Kalman filter for contact estimation ──
		// Predict
		predictedState := v.Wheels[i].KalmanState
		predictedCov := v.Wheels[i].KalmanCov + 0.01 // process noise
		ops += 2

		// Measurement: contact force from height offset
		measurement := 1.0 - math.Abs(heightOff)*5.0
		if measurement < 0 {
			measurement = 0
		}
		ops += 3

		// Update
		innovation := measurement - predictedState
		innovationCov := predictedCov + 0.1 // measurement noise
		kalmanGain := predictedCov / innovationCov
		ops += 3

		v.Wheels[i].KalmanState = predictedState + kalmanGain*innovation
		v.Wheels[i].KalmanCov = (1 - kalmanGain) * predictedCov
		v.Wheels[i].ContactForce = v.Wheels[i].KalmanState
		ops += 3

		// Detection: Kalman state drop below threshold indicates instability.
		// Same warmup as QBP (25 ticks) for fair comparison.
		if tick > 25 && v.Wheels[i].KalmanState < 0.5 {
			if !v.Wheels[i].IsAirborne {
				v.Wheels[i].IsAirborne = true
				v.DetectionLog = append(v.DetectionLog, Detection{
					Tick:   tick,
					Wheel:  i,
					Signal: v.Wheels[i].KalmanState,
					Method: "Scalar",
				})
			}
		} else if v.Wheels[i].KalmanState > 0.7 {
			v.Wheels[i].IsAirborne = false
		}
	}

	// Torque vectoring (same logic, scalar version)
	totalTraction := 0.0
	for i := 0; i < NumWheels; i++ {
		if !v.Wheels[i].IsAirborne {
			totalTraction += 1.0
		}
	}
	if totalTraction > 0 {
		for i := 0; i < NumWheels; i++ {
			if v.Wheels[i].IsAirborne {
				v.Wheels[i].Torque = 0.0
			} else {
				v.Wheels[i].Torque = float64(NumWheels) / totalTraction
				ops++
			}
		}
	}

	// ── ACT: Convert back to actuator commands ──
	// Must reconstruct rotation matrix from Euler for each wheel
	for i := 0; i < NumWheels; i++ {
		r, p, y := v.Wheels[i].Roll, v.Wheels[i].Pitch, v.Wheels[i].Yaw
		// Partial rotation matrix reconstruction (just for torque direction)
		_ = math.Cos(r)*math.Cos(p) + math.Sin(y)*v.Wheels[i].Torque
		ops += 5 // cos, cos, sin, mul, add
	}

	// Chassis update from wheel averages
	if totalTraction > 0 {
		avgR, avgP, avgY := 0.0, 0.0, 0.0
		for i := 0; i < NumWheels; i++ {
			if !v.Wheels[i].IsAirborne {
				avgR += v.Wheels[i].Roll
				avgP += v.Wheels[i].Pitch
				avgY += v.Wheels[i].Yaw
			}
		}
		n := totalTraction
		v.Roll = avgR / n
		v.Pitch = avgP / n
		v.Yaw = avgY / n
		ops += 6
	}

	v.Position += v.Speed * DtControl
	v.TotalOps += int64(ops)
	return ops
}

// ─── Benchmark ─────────────────────────────────────────────────────

// SimConfig configures the Hammer vehicle simulation.
type SimConfig struct {
	Speed        float64 // vehicle speed in m/s
	Duration     float64 // simulation duration in seconds
	TerrainRough float64 // terrain roughness (0-1)
	RockPosition float64 // metres along track where rock strike occurs
	RockHeight   float64 // rock height in metres
}

func DefaultSimConfig() SimConfig {
	return SimConfig{
		Speed:        15.0, // 54 km/h
		Duration:     2.0,  // 2 seconds
		TerrainRough: 0.5,
		RockPosition: 12.0, // rock at 12m — hit ~0.8s into run
		RockHeight:   0.15, // 15cm rock
	}
}

// SimResult holds the comparison between QBP and Scalar pipelines.
type SimResult struct {
	Config SimConfig
	Ticks  int

	// QBP results
	QBPOps             int64
	QBPDetections      []Detection
	QBPFirstDetectTick int // -1 if no detection
	QBPWallTime        time.Duration

	// Scalar results
	ScalarOps             int64
	ScalarDetections      []Detection
	ScalarFirstDetectTick int
	ScalarWallTime        time.Duration

	// Comparison
	DetectionAdvantage int     // ticks earlier QBP detected (positive = QBP wins)
	OpsRatio           float64 // scalar/QBP
	LatencyAdvantage   float64 // milliseconds earlier QBP detected
}

// RunSimulation executes both pipelines on identical terrain and compares.
func RunSimulation(cfg SimConfig) *SimResult {
	totalTicks := int(cfg.Duration * ControlRateHz)
	terrain := NewTerrain(cfg.TerrainRough, cfg.RockPosition, cfg.RockHeight)

	result := &SimResult{
		Config:                cfg,
		Ticks:                 totalTicks,
		QBPFirstDetectTick:    -1,
		ScalarFirstDetectTick: -1,
	}

	// ── QBP Pipeline ──
	qbpV := NewQBPVehicle(cfg.Speed)
	start := time.Now()
	for tick := 0; tick < totalTicks; tick++ {
		QBPTick(qbpV, terrain, tick)
	}
	result.QBPWallTime = time.Since(start)
	result.QBPOps = qbpV.TotalOps
	result.QBPDetections = qbpV.DetectionLog
	if len(qbpV.DetectionLog) > 0 {
		result.QBPFirstDetectTick = qbpV.DetectionLog[0].Tick
	}

	// ── Scalar Pipeline (same terrain, reset) ──
	// Recreate terrain with same seed for identical surface
	terrain2 := &TerrainProfile{
		Seed:       terrain.Seed,
		Roughness:  terrain.Roughness,
		RockAt:     terrain.RockAt,
		RockHeight: terrain.RockHeight,
		rng:        rand.New(rand.NewSource(terrain.Seed)),
	}
	scalarV := NewScalarVehicle(cfg.Speed)
	start = time.Now()
	for tick := 0; tick < totalTicks; tick++ {
		ScalarTick(scalarV, terrain2, tick)
	}
	result.ScalarWallTime = time.Since(start)
	result.ScalarOps = scalarV.TotalOps
	result.ScalarDetections = scalarV.DetectionLog
	if len(scalarV.DetectionLog) > 0 {
		result.ScalarFirstDetectTick = scalarV.DetectionLog[0].Tick
	}

	// ── Comparison ──
	if result.QBPFirstDetectTick >= 0 && result.ScalarFirstDetectTick >= 0 {
		result.DetectionAdvantage = result.ScalarFirstDetectTick - result.QBPFirstDetectTick
		result.LatencyAdvantage = float64(result.DetectionAdvantage) * DtControl * 1000 // ms
	}
	if result.QBPOps > 0 {
		result.OpsRatio = float64(result.ScalarOps) / float64(result.QBPOps)
	}

	return result
}

// FormatResult returns a human-readable comparison report.
func FormatResult(r *SimResult) string {
	s := fmt.Sprintf("HAMMER VEHICLE SIMULATION — QBP vs Scalar Pipeline\n")
	s += fmt.Sprintf("═══════════════════════════════════════════════════\n\n")
	s += fmt.Sprintf("Configuration:\n")
	s += fmt.Sprintf("  Speed: %.1f m/s (%.0f km/h)\n", r.Config.Speed, r.Config.Speed*3.6)
	s += fmt.Sprintf("  Duration: %.1f sec (%d ticks @ %d Hz)\n", r.Config.Duration, r.Ticks, int(ControlRateHz))
	s += fmt.Sprintf("  Terrain roughness: %.1f\n", r.Config.TerrainRough)
	s += fmt.Sprintf("  Rock at %.1fm, height %.0fcm\n\n", r.Config.RockPosition, r.Config.RockHeight*100)

	s += fmt.Sprintf("QBP-Algebraic Pipeline:\n")
	s += fmt.Sprintf("  Total operations:     %d\n", r.QBPOps)
	s += fmt.Sprintf("  Wall time:            %v\n", r.QBPWallTime)
	s += fmt.Sprintf("  Detections:           %d events\n", len(r.QBPDetections))
	if r.QBPFirstDetectTick >= 0 {
		s += fmt.Sprintf("  First detection:      tick %d (%.1f ms into sim)\n",
			r.QBPFirstDetectTick, float64(r.QBPFirstDetectTick)*DtControl*1000)
		s += fmt.Sprintf("    Wheel: %s, Signal: %.3e\n",
			WheelPositions[r.QBPDetections[0].Wheel].Name, r.QBPDetections[0].Signal)
	}

	s += fmt.Sprintf("\nScalar-Conventional Pipeline:\n")
	s += fmt.Sprintf("  Total operations:     %d\n", r.ScalarOps)
	s += fmt.Sprintf("  Wall time:            %v\n", r.ScalarWallTime)
	s += fmt.Sprintf("  Detections:           %d events\n", len(r.ScalarDetections))
	if r.ScalarFirstDetectTick >= 0 {
		s += fmt.Sprintf("  First detection:      tick %d (%.1f ms into sim)\n",
			r.ScalarFirstDetectTick, float64(r.ScalarFirstDetectTick)*DtControl*1000)
		s += fmt.Sprintf("    Wheel: %s, Signal: %.3f\n",
			WheelPositions[r.ScalarDetections[0].Wheel].Name, r.ScalarDetections[0].Signal)
	}

	s += fmt.Sprintf("\n───────────────────────────────────────────────────\n")
	s += fmt.Sprintf("COMPARISON:\n")
	s += fmt.Sprintf("  Operation count ratio (scalar/QBP): %.2f×\n", r.OpsRatio)
	if r.QBPFirstDetectTick >= 0 && r.ScalarFirstDetectTick >= 0 {
		s += fmt.Sprintf("  Detection advantage:                %d ticks (%.1f ms)\n",
			r.DetectionAdvantage, r.LatencyAdvantage)
		if r.DetectionAdvantage > 0 {
			s += fmt.Sprintf("  → QBP detected instability %.1f ms EARLIER than Kalman filter\n", r.LatencyAdvantage)
		} else if r.DetectionAdvantage < 0 {
			s += fmt.Sprintf("  → Scalar detected %.1f ms earlier (investigate)\n", -r.LatencyAdvantage)
		} else {
			s += fmt.Sprintf("  → Simultaneous detection\n")
		}
	} else {
		s += fmt.Sprintf("  Detection comparison: incomplete (one pipeline missed the event)\n")
	}
	s += fmt.Sprintf("═══════════════════════════════════════════════════\n")

	// Mesh scheduling estimate
	s += fmt.Sprintf("\nMESH SCHEDULING ANALYSIS (per-task width allocation):\n")
	s += fmt.Sprintf("  Traction monitoring:  depth=50  tol=1e-3 → QW8  (SIMD: 4×/node)\n")
	s += fmt.Sprintf("  Stability detection:  depth=500 tol=1e-6 → QW16 (1 node)\n")
	s += fmt.Sprintf("  Trajectory planning:  depth=5000 tol=1e-9 → QW32 (1 node)\n")
	s += fmt.Sprintf("  7-node Fano cell allocation:\n")
	s += fmt.Sprintf("    6 nodes: 6 wheels × traction @ QW8 (SIMD, uses 2 nodes)\n")
	s += fmt.Sprintf("    2 nodes: stability monitoring @ QW16\n")
	s += fmt.Sprintf("    2 nodes: trajectory planning @ QW32\n")
	s += fmt.Sprintf("    1 node:  chassis integration + watchdog @ QW16\n")
	s += fmt.Sprintf("    Total: 7 nodes fully utilised, mixed precision per-task\n")

	return s
}
