package mdcu

// P0 correctness gates T-C1..T-C5 (spike plan). These must all pass before any
// benchmark number is admissible.

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"math/big"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const romDir = "../../roms"

func rng() *rand.Rand { return rand.New(rand.NewPCG(2026, 688)) }

func randVec(r *rand.Rand, n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = r.NormFloat64()
	}
	return v
}

func norm(v []float64) float64 {
	var s float64
	for _, x := range v {
		s += x * x
	}
	return math.Sqrt(s)
}

// errVsBig returns the max-norm error of got vs the BigPrec reference, in units
// of eps64·‖x‖·‖y‖ (normwise backward-style scale; robust to cancellation).
func errVsBig(got []float64, ref []*big.Float, scale float64) float64 {
	var m float64
	for i := range got {
		r, _ := ref[i].Float64()
		e := math.Abs(got[i]-r) / (scale * 0x1p-52)
		m = math.Max(m, e)
	}
	return m
}

// ulpErr returns max component error in ulps of the reference component.
func ulpErr(got []float64, ref []*big.Float) float64 {
	var m float64
	for i := range got {
		r, _ := ref[i].Float64()
		if r == 0 {
			continue
		}
		u := math.Nextafter(math.Abs(r), math.Inf(1)) - math.Abs(r)
		m = math.Max(m, math.Abs(got[i]-r)/u)
	}
	return m
}

// --- T-C1: kernel agreement ------------------------------------------------

func TestC1_KernelAgreement(t *testing.T) {
	const N = 100_000
	r := rng()
	var maxRecS, maxMacS, maxRecO, maxMacO, maxDD, ulpMac float64
	var exactRecVsRef int
	for it := 0; it < N; it++ {
		xs, ys := randVec(r, 16), randVec(r, 16)
		var x, y [16]float64
		copy(x[:], xs)
		copy(y[:], ys)
		ref := MulBig(ToBig(xs), ToBig(ys))
		scale := norm(xs) * norm(ys)

		mac := SMulCDMAC(&x, &y)
		rec := SMulRec(&x, &y)
		maxMacS = math.Max(maxMacS, errVsBig(mac[:], ref, scale))
		maxRecS = math.Max(maxRecS, errVsBig(rec[:], ref, scale))
		ulpMac = math.Max(ulpMac, ulpErr(mac[:], ref))

		// flowlib-port reference must be bit-identical to fixed-array recursive CD
		fl := MulRef(xs, ys)
		same := true
		for i := range fl {
			if fl[i] != rec[i] {
				same = false
			}
		}
		if same {
			exactRecVsRef++
		}

		// octonion sub-tower
		var xo, yo [8]float64
		copy(xo[:], xs[:8])
		copy(yo[:], ys[:8])
		refO := MulBig(ToBig(xs[:8]), ToBig(ys[:8]))
		so := norm(xs[:8]) * norm(ys[:8])
		mo, ro := OMulCDMAC(&xo, &yo), OMulRec(&xo, &yo)
		maxMacO = math.Max(maxMacO, errVsBig(mo[:], refO, so))
		maxRecO = math.Max(maxRecO, errVsBig(ro[:], refO, so))

		// QW128 every 100th sample (big.Float compare is slow)
		if it%100 == 0 {
			var xd, yd [16]DD
			for i := 0; i < 16; i++ {
				xd[i], yd[i] = DD{xs[i], 0}, DD{ys[i], 0}
			}
			zd := SMulCDMAC128(&xd, &yd)
			for i := 0; i < 16; i++ {
				sum := new(big.Float).SetPrec(BigPrec).SetFloat64(zd[i].Hi)
				sum.Add(sum, new(big.Float).SetPrec(BigPrec).SetFloat64(zd[i].Lo))
				diff := new(big.Float).SetPrec(BigPrec).Sub(sum, ref[i])
				d, _ := diff.Float64()
				maxDD = math.Max(maxDD, math.Abs(d)/(scale*0x1p-104))
			}
		}
	}
	t.Logf("T-C1 (N=%d) normwise err [eps·‖x‖‖y‖]: S CD-MAC=%.3f  S Rec=%.3f  O CD-MAC=%.3f  O Rec=%.3f",
		N, maxMacS, maxRecS, maxMacO, maxRecO)
	t.Logf("T-C1 QW128 CD-MAC normwise err [2^-104·‖x‖‖y‖]=%.3f ; S CD-MAC max component ulp=%.1f (cancellation-sensitive, informational)",
		maxDD, ulpMac)
	t.Logf("T-C1 flowlib-port MulRef ≡ SMulRec bit-exact: %d/%d", exactRecVsRef, N)
	for name, v := range map[string]float64{"S CD-MAC": maxMacS, "S Rec": maxRecS, "O CD-MAC": maxMacO, "O Rec": maxRecO} {
		if v > 4 {
			t.Errorf("%s normwise error %.3f > 4 eps", name, v)
		}
	}
	if maxDD > 4 {
		t.Errorf("QW128 normwise error %.3f > 4·2^-104", maxDD)
	}
	if exactRecVsRef != N {
		t.Errorf("MulRef vs SMulRec not bit-exact on %d samples", N-exactRecVsRef)
	}
}

// --- T-C2: ROM authority ---------------------------------------------------

func readHexROM(t *testing.T, name string) []int {
	t.Helper()
	f, err := os.Open(filepath.Join(romDir, name))
	if err != nil {
		t.Fatalf("open ROM: %v", err)
	}
	defer f.Close()
	var out []int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "//") {
			continue
		}
		switch l {
		case "0":
			out = append(out, -1)
		case "1":
			out = append(out, +1)
		default:
			t.Fatalf("unexpected ROM token %q", l)
		}
	}
	return out
}

func TestC2_ROMAuthority(t *testing.T) {
	// checksum against the Lean-verified manifest
	manifest, err := os.ReadFile(filepath.Join(romDir, "CHECKSUMS.lean-verified"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sedenion_signs.hex", "octonion_signs.hex"} {
		raw, err := os.ReadFile(filepath.Join(romDir, name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if !strings.Contains(string(manifest), hex.EncodeToString(sum[:])) {
			t.Errorf("%s sha256 not in CHECKSUMS.lean-verified", name)
		}
	}
	// sign table (imaginary block, row-major i,j ∈ 1..n-1) must equal the
	// Schafer-convention table the CD-MAC uses.
	for _, c := range []struct {
		file string
		n    int
	}{{"sedenion_signs.hex", 16}, {"octonion_signs.hex", 8}} {
		rom := readHexROM(t, c.file)
		if len(rom) != (c.n-1)*(c.n-1) {
			t.Fatalf("%s: %d entries, want %d", c.file, len(rom), (c.n-1)*(c.n-1))
		}
		mism := 0
		for i := 1; i < c.n; i++ {
			for j := 1; j < c.n; j++ {
				if float64(rom[(i-1)*(c.n-1)+(j-1)]) != signTable[i][j] {
					mism++
				}
			}
		}
		t.Logf("T-C2 %s: %d/%d entries match CD-MAC sign table (Schafer)", c.file, (c.n-1)*(c.n-1)-mism, (c.n-1)*(c.n-1))
		if mism != 0 {
			t.Errorf("%s: %d sign mismatches vs CD-MAC table — convention drift", c.file, mism)
		}
	}
}

// --- T-C3: algebra laws per tier ------------------------------------------

func assoc(mul func(a, b []float64) []float64, x, y, z []float64) float64 {
	l := mul(mul(x, y), z)
	r := mul(x, mul(y, z))
	var s float64
	for i := range l {
		s += (l[i] - r[i]) * (l[i] - r[i])
	}
	return math.Sqrt(s)
}

func sub(a, b []float64) []float64 {
	c := make([]float64, len(a))
	for i := range a {
		c[i] = a[i] - b[i]
	}
	return c
}

func TestC3_AlgebraLaws(t *testing.T) {
	r := rng()
	const N = 20_000
	var hAssoc, oAlt, oMoufang, oNormMul, oAssocMin, sFlex, sPowAssoc, sNormMulMax float64
	oAssocMin = math.Inf(1)
	for it := 0; it < N; it++ {
		// ℍ: associative (exact zero is required of QASSOC at tier ≤ ℍ; here the
		// floating kernel must be within rounding)
		x4, y4, z4 := randVec(r, 4), randVec(r, 4), randVec(r, 4)
		hAssoc = math.Max(hAssoc, assoc(MulRef, x4, y4, z4)/(norm(x4)*norm(y4)*norm(z4)))

		// 𝕆
		x, y, z := randVec(r, 8), randVec(r, 8), randVec(r, 8)
		s3 := norm(x) * norm(y) * norm(z)
		oAlt = math.Max(oAlt, assoc(MulRef, x, x, y)/(norm(x)*norm(x)*norm(y)))
		// Moufang: z(x(zy)) = ((zx)z)y
		m1 := MulRef(z, MulRef(x, MulRef(z, y)))
		m2 := MulRef(MulRef(MulRef(z, x), z), y)
		oMoufang = math.Max(oMoufang, norm(sub(m1, m2))/(norm(z)*norm(z)*norm(x)*norm(y)))
		oNormMul = math.Max(oNormMul, math.Abs(norm(MulRef(x, y))-norm(x)*norm(y))/(norm(x)*norm(y)))
		oAssocMin = math.Min(oAssocMin, assoc(MulRef, x, y, z)/s3)

		// 𝕊
		xs, ys := randVec(r, 16), randVec(r, 16)
		f1 := MulRef(xs, MulRef(ys, xs))
		f2 := MulRef(MulRef(xs, ys), xs)
		sFlex = math.Max(sFlex, norm(sub(f1, f2))/(norm(xs)*norm(xs)*norm(ys)))
		x2 := MulRef(xs, xs)
		sPowAssoc = math.Max(sPowAssoc, norm(sub(MulRef(x2, xs), MulRef(xs, x2)))/math.Pow(norm(xs), 3))
		sNormMulMax = math.Max(sNormMulMax, math.Abs(norm(MulRef(xs, ys))-norm(xs)*norm(ys))/(norm(xs)*norm(ys)))
	}
	t.Logf("T-C3 H assoc=%.2e | O alt=%.2e moufang=%.2e normMul=%.2e min|assoc|=%.3e | S flex=%.2e powAssoc=%.2e max normMul defect=%.3e",
		hAssoc, oAlt, oMoufang, oNormMul, oAssocMin, sFlex, sPowAssoc, sNormMulMax)
	const tol = 1e-14
	if hAssoc > tol || oAlt > tol || oMoufang > tol || oNormMul > tol || sFlex > tol || sPowAssoc > tol {
		t.Error("a required identity failed beyond rounding")
	}
	if oAssocMin < 1e-6 {
		t.Error("octonion associator unexpectedly ~0 for generic triples")
	}
	if sNormMulMax < 1e-3 {
		t.Error("sedenions appear norm-multiplicative — they must not be")
	}
}

// --- T-C4: ZDCHK -----------------------------------------------------------

func diag(i, j int, s float64) [16]float64 {
	var v [16]float64
	v[i], v[j] = 1, s
	return v
}

func TestC4_ZeroDivisors(t *testing.T) {
	// Brute force over all 105 imaginary planes span{e_i,e_j}: a plane is a ZD
	// plane iff one of its diagonals annihilates some diagonal of another plane.
	zdPlanes := 0
	for i := 1; i < 16; i++ {
		for j := i + 1; j < 16; j++ {
			found := false
			for _, s := range []float64{1, -1} {
				x := diag(i, j, s)
				for k := 1; k < 16 && !found; k++ {
					for l := k + 1; l < 16 && !found; l++ {
						for _, u := range []float64{1, -1} {
							y := diag(k, l, u)
							p := SMulCDMAC(&x, &y)
							if norm(p[:]) == 0 {
								found = true
								break
							}
						}
					}
				}
			}
			if found {
				zdPlanes++
				if !isAssessor(i, j) {
					t.Errorf("plane (%d,%d) has zero divisors but is not a canonical assessor", i, j)
				}
			} else if isAssessor(i, j) {
				t.Errorf("canonical assessor (%d,%d) has no zero-divisor partner", i, j)
			}
		}
	}
	t.Logf("T-C4 brute-force ZD planes = %d (QBP.zero_divisor_count_42 expects 42)", zdPlanes)
	if zdPlanes != 42 || len(Assessors()) != 42 {
		t.Errorf("zero-divisor plane count %d, want 42", zdPlanes)
	}

	// ZDCHK flags every assessor diagonal (scaled, rotated sign)…
	for _, a := range Assessors() {
		for _, s := range []float64{1, -1} {
			v := diag(a.I, a.J, s)
			for k := range v {
				v[k] *= 3.7
			}
			if ok, p, _ := ZDCHK(&v, 1e-12); !ok || p != a {
				t.Errorf("ZDCHK missed assessor %v sign %v", a, s)
			}
		}
	}
	// …and never the (i,i+8) diagonals nor random sedenions.
	for i := 1; i <= 7; i++ {
		v := diag(i, i+8, 1)
		if ok, _, _ := ZDCHK(&v, 1e-12); ok {
			t.Errorf("ZDCHK false positive on (e%d + e%d)", i, i+8)
		}
	}
	r := rng()
	fp := 0
	const N = 1_000_000
	for it := 0; it < N; it++ {
		var v [16]float64
		copy(v[:], randVec(r, 16))
		if ok, _, _ := ZDCHK(&v, 1e-9); ok {
			fp++
		}
	}
	t.Logf("T-C4 ZDCHK random false positives: %d / %d", fp, N)
	if fp != 0 {
		t.Errorf("ZDCHK false positives on random sedenions: %d", fp)
	}
}

// --- T-C5: Theorem 3 (2-plane double-commutator identity) -------------------

// gradVExact is a port of flowlib.gradV_exact using the CD-MAC kernel:
// V(s) = N([a,b]), g_i = 2⟨c, ∂c/∂s_i⟩.
func gradVExact(s *[16]float64) (g [16]float64) {
	var a, b [8]float64
	copy(a[:], s[:8])
	copy(b[:], s[8:])
	ab, ba := OMulCDMAC(&a, &b), OMulCDMAC(&b, &a)
	var c [8]float64
	for i := range c {
		c[i] = ab[i] - ba[i]
	}
	for i := 0; i < 8; i++ {
		var e [8]float64
		e[i] = 1
		eb, be := OMulCDMAC(&e, &b), OMulCDMAC(&b, &e)
		ae, ea := OMulCDMAC(&a, &e), OMulCDMAC(&e, &a)
		var d1, d2 float64
		for k := 0; k < 8; k++ {
			d1 += c[k] * (eb[k] - be[k])
			d2 += c[k] * (ae[k] - ea[k])
		}
		g[i], g[8+i] = 2*d1, 2*d2
	}
	return
}

func TestC5_Theorem3_TwoPlane(t *testing.T) {
	r := rng()
	var maxErr float64
	for it := 0; it < 50_000; it++ {
		var s [16]float64
		copy(s[:], randVec(r, 16))
		g := gradVExact(&s)
		// a = Im(cdLo s), c = Im(cdHi s)
		var a, c [8]float64
		copy(a[1:], s[1:8])
		copy(c[1:], s[9:16])
		var Na, Nc, bil float64
		for k := 1; k < 8; k++ {
			Na += a[k] * a[k]
			Nc += c[k] * c[k]
			bil += a[k] * c[k]
		}
		scale := 8 * (Na + Nc) * math.Sqrt(Na+Nc)
		for k := 0; k < 8; k++ {
			lo := 8*Nc*a[k] - 8*bil*c[k]
			hi := 8*Na*c[k] - 8*bil*a[k]
			maxErr = math.Max(maxErr, math.Abs(g[k]-lo)/scale)
			maxErr = math.Max(maxErr, math.Abs(g[8+k]-hi)/scale)
		}
	}
	t.Logf("T-C5 Theorem 3 (cdLo/cdHi ∇V = 2-plane comb) max relative residual = %.3e", maxErr)
	if maxErr > 1e-14 {
		t.Errorf("Theorem 3 residual %.3e > 1e-14", maxErr)
	}
}
