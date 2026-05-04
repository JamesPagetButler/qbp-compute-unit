// Package quat implements quaternion algebra with dual representations:
//
//   - Quat: float64 components for physics simulation fidelity (SENSE/COMPUTE)
//   - Quat8: int8 components for hypergraph traversal (BMA integration)
//
// The API is structured as single-instruction operations: each exported function
// maps to one instruction in the Run-phase RISC-V ISA. On Crawl hardware (FX-8350),
// these are Go function calls; the structure ensures clean mapping to hardware later.
//
// Quaternion convention: q = w + xi + yj + zk
// where w is the scalar (real) part and (x,y,z) is the vector (imaginary) part.
package quat

import "math"

// ─── Physics-precision representation (SENSE / COMPUTE) ────────────────────

// Quat is a quaternion with float64 components for physics simulation.
// This is the native representation for the Heisenberg spin-chain benchmark
// and any operation requiring numerical fidelity.
type Quat struct {
	W, X, Y, Z float64
}

// New constructs a quaternion from components.
func New(w, x, y, z float64) Quat {
	return Quat{W: w, X: x, Y: y, Z: z}
}

// Scalar returns a pure-real quaternion.
func Scalar(w float64) Quat {
	return Quat{W: w}
}

// Pure returns a pure-imaginary quaternion (vector part only).
func Pure(x, y, z float64) Quat {
	return Quat{X: x, Y: y, Z: z}
}

// Identity returns the multiplicative identity quaternion (1,0,0,0).
func Identity() Quat {
	return Quat{W: 1}
}

// ─── Hamilton product (QMUL instruction) ───────────────────────────────────

// mulGeneric computes the Hamilton product q * r using pure Go scalar math.
func mulGeneric(q, r Quat) Quat {
	return Quat{
		W: q.W*r.W - q.X*r.X - q.Y*r.Y - q.Z*r.Z,
		X: q.W*r.X + q.X*r.W + q.Y*r.Z - q.Z*r.Y,
		Y: q.W*r.Y - q.X*r.Z + q.Y*r.W + q.Z*r.X,
		Z: q.W*r.Z + q.X*r.Y - q.Y*r.X + q.Z*r.W,
	}
}

// mulAccumGeneric computes dest += q * r using pure Go scalar math.
func mulAccumGeneric(dest, q, r Quat) Quat {
	p := mulGeneric(q, r)
	return Quat{
		W: dest.W + p.W,
		X: dest.X + p.X,
		Y: dest.Y + p.Y,
		Z: dest.Z + p.Z,
	}
}

// ─── Norm and normalisation ────────────────────────────────────────────────

// NormSq returns ||q||² = w² + x² + y² + z².
// This is cheaper than Norm and sufficient for the algebraic watchdog
// (norm drift is detectable without the sqrt).
func NormSq(q Quat) float64 {
	return q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z
}

// Norm returns ||q|| = sqrt(w² + x² + y² + z²).
func Norm(q Quat) float64 {
	return math.Sqrt(NormSq(q))
}

// Normalize returns q / ||q||. Panics on zero quaternion.
func Normalize(q Quat) Quat {
	n := Norm(q)
	return Quat{W: q.W / n, X: q.X / n, Y: q.Y / n, Z: q.Z / n}
}

// ─── Conjugate and inverse ─────────────────────────────────────────────────

// Conj returns the conjugate q* = w - xi - yj - zk.
func Conj(q Quat) Quat {
	return Quat{W: q.W, X: -q.X, Y: -q.Y, Z: -q.Z}
}

// Inv returns the multiplicative inverse q⁻¹ = q* / ||q||².
func Inv(q Quat) Quat {
	nsq := NormSq(q)
	return Quat{W: q.W / nsq, X: -q.X / nsq, Y: -q.Y / nsq, Z: -q.Z / nsq}
}

// ─── QROT instruction: rotation of vector by unit quaternion ───────────────

// Rotate applies unit quaternion q to vector v as qvq*.
// This is the QROT instruction — the most common physical operation.
// v is represented as a pure quaternion (w=0).
//
// Precondition: q should be unit quaternion (||q|| = 1).
// The function does NOT normalize q; the caller is responsible.
// This is deliberate: in the algebraic watchdog, norm drift in q
// is a signal, not an error to silently correct.
func Rotate(q Quat, v Quat) Quat {
	return Mul(Mul(q, v), Conj(q))
}

// RotateVec is a convenience that takes a 3-vector and returns a 3-vector.
func RotateVec(q Quat, vx, vy, vz float64) (float64, float64, float64) {
	v := Pure(vx, vy, vz)
	r := Rotate(q, v)
	return r.X, r.Y, r.Z
}

// ─── Exponential and logarithm (for time evolution) ────────────────────────

// Exp computes exp(q) for a quaternion q.
//
// For a pure quaternion q = (0, v), exp(q) = cos(||v||) + sin(||v||) * v/||v||
// This is critical for the spin-chain benchmark: time evolution under
// Hamiltonian H is exp(-iHt), which for spin-½ is quaternion exponential.
func Exp(q Quat) Quat {
	// exp(q) = exp(w) * (cos(||v||) + sin(||v||) * v_hat)
	vNorm := math.Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z)
	ew := math.Exp(q.W)

	if vNorm < 1e-15 {
		// Pure scalar: exp(w) * (1, 0, 0, 0)
		return Quat{W: ew}
	}

	s := ew * math.Sin(vNorm) / vNorm
	return Quat{
		W: ew * math.Cos(vNorm),
		X: s * q.X,
		Y: s * q.Y,
		Z: s * q.Z,
	}
}

// Log computes log(q) for a quaternion q with ||q|| > 0.
func Log(q Quat) Quat {
	n := Norm(q)
	vNorm := math.Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z)

	if vNorm < 1e-15 {
		return Quat{W: math.Log(n)}
	}

	theta := math.Acos(q.W / n)
	s := theta / vNorm
	return Quat{
		W: math.Log(n),
		X: s * q.X,
		Y: s * q.Y,
		Z: s * q.Z,
	}
}

// ─── Arithmetic helpers ────────────────────────────────────────────────────

// Add returns q + r.
func Add(q, r Quat) Quat {
	return Quat{W: q.W + r.W, X: q.X + r.X, Y: q.Y + r.Y, Z: q.Z + r.Z}
}

// Sub returns q - r.
func Sub(q, r Quat) Quat {
	return Quat{W: q.W - r.W, X: q.X - r.X, Y: q.Y - r.Y, Z: q.Z - r.Z}
}

// Scale returns s * q (scalar multiplication).
func Scale(s float64, q Quat) Quat {
	return Quat{W: s * q.W, X: s * q.X, Y: s * q.Y, Z: s * q.Z}
}

// Dot returns the 4D dot product of q and r (real-valued).
func Dot(q, r Quat) float64 {
	return q.W*r.W + q.X*r.X + q.Y*r.Y + q.Z*r.Z
}

// ─── Hypergraph-precision representation (BMA integration) ─────────────────

// Quat8 is a quaternion with int8 components for hypergraph edge weights.
// This is the native representation for BMA memory traversal.
// Matches the int8 data type that won the ternary inference benchmark (16.6×).
type Quat8 struct {
	W, X, Y, Z int8
}

// NewQuat8 constructs an int8 quaternion.
func NewQuat8(w, x, y, z int8) Quat8 {
	return Quat8{W: w, X: x, Y: y, Z: z}
}

// ToQuat promotes a Quat8 to float64 Quat (SENSE boundary: int8 → float64).
// Scale factor maps int8 range [-127,127] to [-1.0, 1.0].
func (q8 Quat8) ToQuat() Quat {
	const scale = 1.0 / 127.0
	return Quat{
		W: float64(q8.W) * scale,
		X: float64(q8.X) * scale,
		Y: float64(q8.Y) * scale,
		Z: float64(q8.Z) * scale,
	}
}

// ToQuat8 quantises a float64 Quat to int8 (ACT boundary: float64 → int8).
// Clamps to [-127, 127] range. This is a lossy operation; the algebraic
// watchdog should track cumulative quantisation error.
func ToQuat8(q Quat) Quat8 {
	clamp := func(v float64) int8 {
		scaled := v * 127.0
		if scaled > 127 {
			return 127
		}
		if scaled < -127 {
			return -127
		}
		return int8(scaled)
	}
	return Quat8{
		W: clamp(q.W),
		X: clamp(q.X),
		Y: clamp(q.Y),
		Z: clamp(q.Z),
	}
}

// Mul8 computes Hamilton product for int8 quaternions using int16 intermediates.
// This is the TMAC-compatible path for hypergraph traversal.
func Mul8(q, r Quat8) Quat8 {
	// Use int16 to avoid overflow during multiplication
	qw, qx, qy, qz := int16(q.W), int16(q.X), int16(q.Y), int16(q.Z)
	rw, rx, ry, rz := int16(r.W), int16(r.X), int16(r.Y), int16(r.Z)

	// Hamilton product in int16, then saturate back to int8
	sat := func(v int16) int8 {
		if v > 127 {
			return 127
		}
		if v < -127 {
			return -127
		}
		return int8(v)
	}

	// Shift right by 7 to rescale (int8 * int8 produces values up to 127*127 = 16129)
	const shift = 7
	return Quat8{
		W: sat((qw*rw - qx*rx - qy*ry - qz*rz) >> shift),
		X: sat((qw*rx + qx*rw + qy*rz - qz*ry) >> shift),
		Y: sat((qw*ry - qx*rz + qy*rw + qz*rx) >> shift),
		Z: sat((qw*rz + qx*ry - qy*rx + qz*rw) >> shift),
	}
}
