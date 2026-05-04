package emulator

import (
	"fmt"
	"math/big"
)

// Width defines the component bit-width of a quaternion word.
type Width int

const (
	W8    Width = 8
	W16   Width = 16
	W32   Width = 32
	W64   Width = 64
	W128  Width = 128
	W256  Width = 256
	W512  Width = 512
	W1024 Width = 1024
)

// QW64 is a 64-bit hardware-accelerated quaternion (4x float64) used by the emulator for W8-W64 paths.
type QW64 [4]float64

// QWord is a high-precision quaternion representation used by the emulator for W128+ paths.
type QWord struct {
	W, X, Y, Z *big.Float
}

// NewQWord creates a new quaternion with the specified precision.
func NewQWord(prec uint) QWord {
	return QWord{
		W: new(big.Float).SetPrec(prec),
		X: new(big.Float).SetPrec(prec),
		Y: new(big.Float).SetPrec(prec),
		Z: new(big.Float).SetPrec(prec),
	}
}

// SetPrec updates the precision of all components in the QWord.
func (q QWord) SetPrec(prec uint) QWord {
	q.W.SetPrec(prec)
	q.X.SetPrec(prec)
	q.Y.SetPrec(prec)
	q.Z.SetPrec(prec)
	return q
}

// String returns a mnemonic representation of the QWord.
func (q QWord) String() string {
	return fmt.Sprintf("[%v, %v, %v, %v]", q.W, q.X, q.Y, q.Z)
}

// Gearbox manages the precision context and zero-allocation scratchpads.
type Gearbox struct {
	ActiveWidth Width
	t1, t2, t3, t4 *big.Float
	rW, rX, rY, rZ *big.Float // Temp result scratchpads
	tempRot     QWord         // Scratchpad for QROT
	tempConj    QWord         // Scratchpad for QROT conjugate
}

// NewGearbox initializes the Gearbox with pre-allocated scratchpads to prevent GC thrashing.
func NewGearbox() *Gearbox {
	prec := uint(64) // default W64 precision
	return &Gearbox{
		ActiveWidth: W64,
		t1: new(big.Float).SetPrec(prec),
		t2: new(big.Float).SetPrec(prec),
		t3: new(big.Float).SetPrec(prec),
		t4: new(big.Float).SetPrec(prec),
		rW: new(big.Float).SetPrec(prec),
		rX: new(big.Float).SetPrec(prec),
		rY: new(big.Float).SetPrec(prec),
		rZ: new(big.Float).SetPrec(prec),
		tempRot: NewQWord(prec),
		tempConj: NewQWord(prec),
	}
}

// SetWidth updates the gearbox precision and re-scales the internal scratchpads.
func (g *Gearbox) SetWidth(w Width) {
	g.ActiveWidth = w
	prec := g.Precision()
	g.t1.SetPrec(prec)
	g.t2.SetPrec(prec)
	g.t3.SetPrec(prec)
	g.t4.SetPrec(prec)
	g.rW.SetPrec(prec)
	g.rX.SetPrec(prec)
	g.rY.SetPrec(prec)
	g.rZ.SetPrec(prec)
	g.tempRot.SetPrec(prec)
	g.tempConj.SetPrec(prec)
}

// Precision returns the big.Float precision required for the active width.
func (g *Gearbox) Precision() uint {
	switch g.ActiveWidth {
	case W8, W16, W32:
		return 32
	case W64:
		return 64
	case W128:
		return 128
	case W256:
		return 256
	case W512:
		return 512
	case W1024:
		return 1024
	default:
		return 64
	}
}

// Mul computes the Hamilton product in-place: dst = a * b
func (g *Gearbox) Mul(dst, a, b *QWord) {
	wA, xA, yA, zA := a.W, a.X, a.Y, a.Z
	wB, xB, yB, zB := b.W, b.X, b.Y, b.Z

	// Compute W into rW
	g.t1.Mul(wA, wB)
	g.t2.Mul(xA, xB)
	g.t3.Mul(yA, yB)
	g.t4.Mul(zA, zB)
	g.rW.Sub(g.t1, g.t2)
	g.rW.Sub(g.rW, g.t3)
	g.rW.Sub(g.rW, g.t4)

	// Compute X into rX
	g.t1.Mul(wA, xB)
	g.t2.Mul(xA, wB)
	g.t3.Mul(yA, zB)
	g.t4.Mul(zA, yB)
	g.rX.Add(g.t1, g.t2)
	g.rX.Add(g.rX, g.t3)
	g.rX.Sub(g.rX, g.t4)

	// Compute Y into rY
	g.t1.Mul(wA, yB)
	g.t2.Mul(xA, zB)
	g.t3.Mul(yA, wB)
	g.t4.Mul(zA, xB)
	g.rY.Sub(g.t1, g.t2)
	g.rY.Add(g.rY, g.t3)
	g.rY.Add(g.rY, g.t4)

	// Compute Z into rZ
	g.t1.Mul(wA, zB)
	g.t2.Mul(xA, yB)
	g.t3.Mul(yA, xB)
	g.t4.Mul(zA, wB)
	g.rZ.Add(g.t1, g.t2)
	g.rZ.Sub(g.rZ, g.t3)
	g.rZ.Add(g.rZ, g.t4)

	// Safely copy to destination (handles dst == a or b)
	dst.W.Set(g.rW)
	dst.X.Set(g.rX)
	dst.Y.Set(g.rY)
	dst.Z.Set(g.rZ)
}

// Conj computes the conjugate in-place: dst = q*
func (g *Gearbox) Conj(dst, q *QWord) {
	dst.W.Set(q.W)
	dst.X.Neg(q.X)
	dst.Y.Neg(q.Y)
	dst.Z.Neg(q.Z)
}

// Rotate applies unit quaternion q to vector v as qvq* in-place
func (g *Gearbox) Rotate(dst, q, v *QWord) {
	// 1. Compute tempRot = q * v
	g.Mul(&g.tempRot, q, v)
	
	// 2. Compute q* into tempConj
	g.Conj(&g.tempConj, q)
	
	// 3. Compute dst = tempRot * qConj
	g.Mul(dst, &g.tempRot, &g.tempConj)
}

// NormSq computes w^2 + x^2 + y^2 + z^2 into dst
func (g *Gearbox) NormSq(dst *big.Float, q *QWord) {
	dst.Mul(q.W, q.W)
	g.t1.Mul(q.X, q.X)
	dst.Add(dst, g.t1)
	g.t1.Mul(q.Y, q.Y)
	dst.Add(dst, g.t1)
	g.t1.Mul(q.Z, q.Z)
	dst.Add(dst, g.t1)
}

// FanoEntry matches the fano.Entry type.
type FanoEntry struct {
	Index int8
	Sign  int8
}

// FanoLookup returns the product of imaginary units e_i and e_j.
func (g *Gearbox) FanoLookup(i, j int) FanoEntry {
	if i == j {
		return FanoEntry{Index: 0, Sign: -1}
	}
	table := map[[2]int]FanoEntry{
		{1, 2}: {3, 1}, {2, 1}: {3, -1},
		{2, 3}: {1, 1}, {3, 2}: {1, -1},
		{3, 1}: {2, 1}, {1, 3}: {2, -1},
	}
	if res, ok := table[[2]int{i, j}]; ok {
		return res
	}
	return FanoEntry{Index: int8((i + j) % 7) + 1, Sign: 1}
}
