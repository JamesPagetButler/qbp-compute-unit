package mdcu

import "math"

// ZDCHK — sedenion zero-divisor class match (Xqbp v2, Funct7 = 0x0A).
//
// The sedenion zero divisors of "assessor" type lie on the diagonals e_i ± e_j
// of the 42 planes span{e_i, e_j}, i ∈ 1..7, j ∈ 9..15, j ≠ i+8
// (QBP.zero_divisor_count_42; de Marrais "box-kites"). ZDCHK reports whether a
// sedenion is (within tol, relative to its norm) on one of those diagonals.

// Assessor is a zero-divisor plane span{e_I, e_J}.
type Assessor struct{ I, J int }

// Assessors returns the 42 canonical assessor planes.
func Assessors() []Assessor {
	out := make([]Assessor, 0, 42)
	for i := 1; i <= 7; i++ {
		for j := 9; j <= 15; j++ {
			if j != i+8 {
				out = append(out, Assessor{i, j})
			}
		}
	}
	return out
}

// isAssessor reports whether span{e_i,e_j} (i<j) is one of the 42 planes.
func isAssessor(i, j int) bool {
	if i > j {
		i, j = j, i
	}
	return i >= 1 && i <= 7 && j >= 9 && j <= 15 && j != i+8
}

// ZDCHK returns (true, plane, sign) if x lies on a zero-divisor diagonal
// e_I + sign·e_J of an assessor plane, to relative tolerance tol.
func ZDCHK(x *[16]float64, tol float64) (bool, Assessor, float64) {
	var n2 float64
	for _, v := range x {
		n2 += v * v
	}
	if n2 == 0 {
		return false, Assessor{}, 0
	}
	// two largest-magnitude components
	i1, i2 := -1, -1
	for k, v := range x {
		a := math.Abs(v)
		if i1 < 0 || a > math.Abs(x[i1]) {
			i2, i1 = i1, k
		} else if i2 < 0 || a > math.Abs(x[i2]) {
			i2 = k
		}
	}
	scale := math.Sqrt(n2)
	var rest float64
	for k, v := range x {
		if k != i1 && k != i2 {
			rest += v * v
		}
	}
	if math.Sqrt(rest) > tol*scale {
		return false, Assessor{}, 0
	}
	if math.Abs(math.Abs(x[i1])-math.Abs(x[i2])) > tol*scale {
		return false, Assessor{}, 0
	}
	i, j := i1, i2
	if i > j {
		i, j = j, i
	}
	if !isAssessor(i, j) {
		return false, Assessor{}, 0
	}
	return true, Assessor{i, j}, math.Copysign(1, x[i]*x[j])
}
