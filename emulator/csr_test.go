package emulator

import (
	"errors"
	"testing"
)

func TestCSR_AMODESetGet(t *testing.T) {
	g := NewGearbox()
	if g.AMODE() != 0 {
		t.Fatalf("initial AMODE: got %d, want 0", g.AMODE())
	}
	if err := g.SetAMODE(0); err != nil {
		t.Fatalf("SetAMODE(0) unexpected error: %v", err)
	}
	if g.AMODE() != 0 {
		t.Fatalf("after SetAMODE(0): got %d, want 0", g.AMODE())
	}
}

func TestCSR_AMODEValidation(t *testing.T) {
	g := NewGearbox()

	err := g.SetAMODE(1)
	if err == nil {
		t.Fatal("SetAMODE(1) expected error for unsupported mode, got nil")
	}
	if !errors.Is(err, ErrInvalidAMODE) {
		t.Fatalf("SetAMODE(1): want ErrInvalidAMODE, got %v", err)
	}

	err = g.SetAMODE(2)
	if err == nil {
		t.Fatal("SetAMODE(2) expected error for reserved mode, got nil")
	}
	if !errors.Is(err, ErrAMODEReserved) {
		t.Fatalf("SetAMODE(2): want ErrAMODEReserved, got %v", err)
	}

	err = g.SetAMODE(255)
	if !errors.Is(err, ErrAMODEReserved) {
		t.Fatalf("SetAMODE(255): want ErrAMODEReserved, got %v", err)
	}
}

func TestCSR_BSELValidation(t *testing.T) {
	g := NewGearbox()

	if err := g.SetBSEL(0); err != nil {
		t.Fatalf("SetBSEL(0) unexpected error: %v", err)
	}
	if g.BSEL() != 0 {
		t.Fatalf("after SetBSEL(0): got %d, want 0", g.BSEL())
	}

	err := g.SetBSEL(1)
	if err == nil {
		t.Fatal("SetBSEL(1) expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidBSEL) {
		t.Fatalf("SetBSEL(1): want ErrInvalidBSEL, got %v", err)
	}
}

func TestCSR_PSELValidation(t *testing.T) {
	g := NewGearbox()

	if err := g.SetPSEL(0); err != nil {
		t.Fatalf("SetPSEL(0) unexpected error: %v", err)
	}
	if g.PSEL() != 0 {
		t.Fatalf("after SetPSEL(0): got %d, want 0", g.PSEL())
	}

	err := g.SetPSEL(1)
	if err == nil {
		t.Fatal("SetPSEL(1) expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidPSEL) {
		t.Fatalf("SetPSEL(1): want ErrInvalidPSEL, got %v", err)
	}
}

func TestGearbox_BackwardsCompatAMODE0(t *testing.T) {
	g := NewGearbox()

	// QMul64 at AMODE=0 must produce identical results to the pre-M1 fast path.
	a := [4]float64{1, 0, 0, 0}
	b := [4]float64{0, 1, 0, 0}
	got := g.QMul64(a, b)
	want := [4]float64{0, 1, 0, 0} // 1·i = i
	if got != want {
		t.Fatalf("QMul64(1, i) at AMODE=0: got %v, want %v", got, want)
	}

	// QMul128 identity check.
	ai := [8]float64{1, 0, 0, 0, 0, 0, 0, 0}
	bi := [8]float64{0, 1, 0, 0, 0, 0, 0, 0}
	g128 := g.QMul128(ai, bi)
	if g128[0] != 0 || g128[1] != 1 {
		t.Fatalf("QMul128(1, i) W/X components: got %v", g128[:2])
	}
}

func TestCSR_ConcurrentRead(t *testing.T) {
	g := NewGearbox()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			_ = g.AMODE()
		}
		close(done)
	}()
	for i := 0; i < 1000; i++ {
		_ = g.AMODE()
	}
	<-done
}

func BenchmarkGearbox_QMul64_AMODE0(b *testing.B) {
	g := NewGearbox()
	a := [4]float64{1, 0, 0, 0}
	v := [4]float64{0, 1, 0, 0}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.QMul64(a, v)
	}
}
