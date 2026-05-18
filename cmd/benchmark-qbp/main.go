package main

import (
	"fmt"
	"math/big"
	"time"
)

// LOCAL TYPES (To bypass import issues during baseline establishing)
type Width int

const (
	W64   Width = 64
	W128  Width = 128
	W1024 Width = 1024
)

type QWord struct{ W, X, Y, Z *big.Float }

func NewQWord(prec uint) QWord {
	return QWord{W: new(big.Float).SetPrec(prec), X: new(big.Float).SetPrec(prec), Y: new(big.Float).SetPrec(prec), Z: new(big.Float).SetPrec(prec)}
}

func Mul(a, b QWord, prec uint) QWord {
	res := NewQWord(prec)
	t1, t2, t3, t4 := new(big.Float).SetPrec(prec), new(big.Float).SetPrec(prec), new(big.Float).SetPrec(prec), new(big.Float).SetPrec(prec)
	// W = a.w*b.w - a.x*b.x - a.y*b.y - a.z*b.z
	t1.Mul(a.W, b.W)
	t2.Mul(a.X, b.X)
	t3.Mul(a.Y, b.Y)
	t4.Mul(a.Z, b.Z)
	res.W.Sub(t1, t2).Sub(res.W, t3).Sub(res.W, t4)
	return res
}

func NormSq(q QWord, prec uint) *big.Float {
	res := new(big.Float).SetPrec(prec)
	t := new(big.Float).SetPrec(prec)
	res.Mul(q.W, q.W)
	t.Mul(q.X, q.X)
	res.Add(res, t)
	t.Mul(q.Y, q.Y)
	res.Add(res, t)
	t.Mul(q.Z, q.Z)
	res.Add(res, t)
	return res
}

func main() {
	fmt.Println("QBP COMPUTE UNIT — NUMERICAL BASELINE AUDIT")
	fmt.Println("Hardware: AMD FX-8350 | Standard: float64 vs QBP")
	fmt.Println("================================================")

	iterations := 1000000

	// 1. CONTROL: float64
	fmt.Printf("\n[CONTROL] Standard float64 (IEEE 754):\n")
	start := time.Now()
	fVal := 1.0
	fFactor := 1.0 + 1e-15
	for i := 0; i < iterations; i++ {
		fVal *= fFactor
	}
	fDur := time.Since(start)
	fmt.Printf("  Throughput: %.2f M op/s\n", float64(iterations)/fDur.Seconds()/1e6)
	fmt.Printf("  Final Value: %.15f\n", fVal)

	// 2. TEST: QBP-W128
	fmt.Printf("\n[TEST] QBP-W128 (512-bit total):\n")
	prec := uint(128)
	q := NewQWord(prec)
	q.W.SetFloat64(1.0)
	qFactor := NewQWord(prec)
	f64Factor := new(big.Float).SetPrec(prec).SetFloat64(1.0 + 1e-15)
	qFactor.W.Set(f64Factor)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		q = Mul(q, qFactor, prec)
	}
	qDur := time.Since(start)
	fmt.Printf("  Throughput: %.2f K op/s\n", float64(iterations)/qDur.Seconds()/1e3)
	nsq := NormSq(q, prec)
	fmt.Printf("  NormSq (Integrity): %v\n", nsq)

	// 3. TEST: QBP-W1024
	fmt.Printf("\n[TEST] QBP-W1024 (4096-bit total):\n")
	prec = 1024
	q1024 := NewQWord(prec)
	q1024.W.SetFloat64(1.0)
	qFactor1024 := NewQWord(prec)
	f64Factor1024 := new(big.Float).SetPrec(prec).SetFloat64(1.0 + 1e-15)
	qFactor1024.W.Set(f64Factor1024)

	it1024 := 100000
	start = time.Now()
	for i := 0; i < it1024; i++ {
		q1024 = Mul(q1024, qFactor1024, prec)
	}
	qDur1024 := time.Since(start)
	fmt.Printf("  Throughput: %.2f K op/s\n", float64(it1024)/qDur1024.Seconds()/1e3)
	nsq1024 := NormSq(q1024, prec)
	fmt.Printf("  NormSq (Integrity): %v\n", nsq1024)
}
