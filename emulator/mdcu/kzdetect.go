package mdcu

// mdcu-kz Z2 holonomy detector (frozen mdcu-kz prereg §3). A pure function of the
// field: it never touches the dynamics.
//
// Method (a Z2 lattice-gauge style discretisation of w2 of the tautological rank-3
// bundle over each cell boundary):
//   1. Each lattice edge a->b carries the nearest rotation Q_ab ∈ SO(3) to the 3x3
//      overlap matrix M_ab[i][j] = f_a[i]·f_b[j] of the two sites' frames (polar
//      part; computed as Horn's optimal quaternion, which is directly an SU(2) lift
//      eps_ab). Q_ba = Q_ab^T, eps_ba = conj(eps_ab).
//   2. A cell boundary is triangulated into 12 triangles: each of the 6 faces is
//      split along the diagonal from its minimum corner A to the opposite corner C
//      (A,B,C and A,C,D). The diagonal rule is a function of the face, so the two
//      cells that share a face share both triangles.
//   3. A triangle's holonomy lift is L = eps_ab eps_bc eps_ca ∈ SU(2). Its sign
//      s_t = sign(L.w) is the lift "on the branch nearest the identity" relative to
//      the arbitrary edge-lift signs. The triangle is "ambiguous" when the rotation
//      angle of the holonomy is >= AmbigAngle (frozen: pi).
//   4. A cell is -1 iff the product of its 12 triangle signs is -1. Flipping any
//      edge lift's sign flips the two triangles of the cell that contain that edge,
//      so the cell sign does not depend on the edge-lift signs; it also does not
//      depend on the frame gauge. -1 marks a defect unless some triangle of the cell
//      is ambiguous (then the cell counts as ambiguous, never as a defect).
//
// Parity: every lattice triangle belongs to exactly two cells, so on a periodic
// lattice the product of all cell signs is Π s_t² = +1: the number of -1 cells is
// even, exactly, for any field. (The frozen invariant only requires it at zero
// ambiguity; it is checked as such at run time and in tests.)

import "math"

// Z2Params carries the one tunable of the detector. The frozen text sets
// AmbigAngle = pi; DefaultZ2Params returns that.
type Z2Params struct {
	AmbigAngle float64
}

func DefaultZ2Params() Z2Params { return Z2Params{AmbigAngle: math.Pi} }

// Z2Result is one detector pass over an L³ periodic lattice. Cell ids are the
// site index of the cell's minimum corner.
type Z2Result struct {
	L           int
	Cells       int
	Defects     int   // -1 cells with no ambiguous triangle
	Ambiguous   int   // cells with at least one ambiguous triangle
	NegCells    int   // cells whose product is -1, ambiguous or not
	DefectCells []int // ids of the defects
	AmbigCells  []int // ids of the ambiguous cells
	ParityOK    bool  // NegCells is even
}

// ---- quaternions (w, x, y, z) -----------------------------------------------

func QuatMul(a, b [4]float64) [4]float64 {
	return [4]float64{
		a[0]*b[0] - a[1]*b[1] - a[2]*b[2] - a[3]*b[3],
		a[0]*b[1] + a[1]*b[0] + a[2]*b[3] - a[3]*b[2],
		a[0]*b[2] - a[1]*b[3] + a[2]*b[0] + a[3]*b[1],
		a[0]*b[3] + a[1]*b[2] - a[2]*b[1] + a[3]*b[0],
	}
}

func QuatConj(a [4]float64) [4]float64 { return [4]float64{a[0], -a[1], -a[2], -a[3]} }

// QuatToMat returns the rotation matrix R(q) of a unit quaternion, with R(ab) = R(a)R(b).
func QuatToMat(q [4]float64) [3][3]float64 {
	w, x, y, z := q[0], q[1], q[2], q[3]
	return [3][3]float64{
		{w*w + x*x - y*y - z*z, 2 * (x*y - w*z), 2 * (x*z + w*y)},
		{2 * (x*y + w*z), w*w - x*x + y*y - z*z, 2 * (y*z - w*x)},
		{2 * (x*z - w*y), 2 * (y*z + w*x), w*w - x*x - y*y + z*z},
	}
}

// NearestRotationQuat returns the unit quaternion (w >= 0) of the rotation R that
// maximises tr(R^T M), i.e. the nearest rotation to M in the Frobenius norm among
// proper rotations (Horn's method: the top eigenvector of a symmetric 4x4 matrix).
func NearestRotationQuat(m [3][3]float64) [4]float64 {
	var n [4][4]float64
	n[0][0] = m[0][0] + m[1][1] + m[2][2]
	n[1][1] = m[0][0] - m[1][1] - m[2][2]
	n[2][2] = -m[0][0] + m[1][1] - m[2][2]
	n[3][3] = -m[0][0] - m[1][1] + m[2][2]
	n[0][1] = m[2][1] - m[1][2]
	n[0][2] = m[0][2] - m[2][0]
	n[0][3] = m[1][0] - m[0][1]
	n[1][2] = m[0][1] + m[1][0]
	n[1][3] = m[0][2] + m[2][0]
	n[2][3] = m[1][2] + m[2][1]
	for i := 0; i < 4; i++ {
		for j := i + 1; j < 4; j++ {
			n[j][i] = n[i][j]
		}
	}
	q := topEigenvector4(n)
	if q[0] < 0 {
		for i := range q {
			q[i] = -q[i]
		}
	}
	return q
}

// topEigenvector4 is a cyclic Jacobi eigen-solver for a symmetric 4x4 matrix; it
// returns the unit eigenvector of the largest eigenvalue.
func topEigenvector4(a [4][4]float64) [4]float64 {
	var v [4][4]float64
	for i := 0; i < 4; i++ {
		v[i][i] = 1
	}
	for sweep := 0; sweep < 30; sweep++ {
		off := 0.0
		for i := 0; i < 4; i++ {
			for j := i + 1; j < 4; j++ {
				off += a[i][j] * a[i][j]
			}
		}
		if off < 1e-30 {
			break
		}
		for p := 0; p < 3; p++ {
			for q := p + 1; q < 4; q++ {
				if math.Abs(a[p][q]) < 1e-300 {
					continue
				}
				theta := (a[q][q] - a[p][p]) / (2 * a[p][q])
				t := 1 / (math.Abs(theta) + math.Sqrt(theta*theta+1))
				if theta < 0 {
					t = -t
				}
				c := 1 / math.Sqrt(t*t+1)
				s := t * c
				for k := 0; k < 4; k++ { // rotate columns p, q
					akp, akq := a[k][p], a[k][q]
					a[k][p], a[k][q] = c*akp-s*akq, s*akp+c*akq
				}
				for k := 0; k < 4; k++ { // rotate rows p, q
					apk, aqk := a[p][k], a[q][k]
					a[p][k], a[q][k] = c*apk-s*aqk, s*apk+c*aqk
				}
				for k := 0; k < 4; k++ {
					vkp, vkq := v[k][p], v[k][q]
					v[k][p], v[k][q] = c*vkp-s*vkq, s*vkp+c*vkq
				}
			}
		}
	}
	best := 0
	for i := 1; i < 4; i++ {
		if a[i][i] > a[best][best] {
			best = i
		}
	}
	q := [4]float64{v[0][best], v[1][best], v[2][best], v[3][best]}
	nn := math.Sqrt(q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3])
	for i := range q {
		q[i] /= nn
	}
	return q
}

// TriangleZ2 returns the sign (neg = the lift is on the -1 side of the nearest-
// identity branch) and the ambiguity flag of one triangle with edge lifts
// eab, ebc, eca (a->b, b->c, c->a).
func TriangleZ2(eab, ebc, eca [4]float64, ambigAngle float64) (neg, ambiguous bool) {
	l := QuatMul(QuatMul(eab, ebc), eca)
	aw := math.Abs(l[0])
	if aw > 1 {
		aw = 1
	}
	theta := 2 * math.Acos(aw) // rotation angle of the holonomy, in [0, pi]
	return l[0] < 0, theta >= ambigAngle-1e-9
}

// ---- overlaps -----------------------------------------------------------------

func planeOverlap(fa, fb *[3][7]float64) (m [3][3]float64) {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			s := 0.0
			for k := 0; k < 7; k++ {
				s += fa[i][k] * fb[j][k]
			}
			m[i][j] = s
		}
	}
	return
}

func rotOverlap(ra, rb *[3][3]float64) (m [3][3]float64) { // R_a R_b^T
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			s := 0.0
			for k := 0; k < 3; k++ {
				s += ra[i][k] * rb[j][k]
			}
			m[i][j] = s
		}
	}
	return
}

// ---- single cell ------------------------------------------------------------------

// CellZ2Planes runs the detector on one cell from its 8 corner frames (corner id =
// x<<2 | y<<1 | z). It returns the cell sign (+1, -1) and whether the cell is ambiguous.
func CellZ2Planes(c [8][3][7]float64, p Z2Params) (sign int, ambiguous bool) {
	var lifts [8][8]*[4]float64
	lift := func(a, b int) [4]float64 {
		if a > b {
			return QuatConj(cellLift(&lifts, &c, b, a))
		}
		return cellLift(&lifts, &c, a, b)
	}
	sign = 1
	bit := [3]uint{2, 1, 0}
	for d := 0; d < 3; d++ {
		e1, e2 := (d+1)%3, (d+2)%3
		for side := 0; side < 2; side++ {
			A := side << bit[d]
			B := A | 1<<bit[e1]
			D := A | 1<<bit[e2]
			C := B | D
			for _, tri := range [2][3]int{{A, B, C}, {A, C, D}} {
				neg, amb := TriangleZ2(lift(tri[0], tri[1]), lift(tri[1], tri[2]), lift(tri[2], tri[0]), p.AmbigAngle)
				if neg {
					sign = -sign
				}
				if amb {
					ambiguous = true
				}
			}
		}
	}
	return
}

func cellLift(cache *[8][8]*[4]float64, c *[8][3][7]float64, a, b int) [4]float64 {
	if cache[a][b] == nil {
		q := NearestRotationQuat(planeOverlap(&c[a], &c[b]))
		cache[a][b] = &q
	}
	return *cache[a][b]
}

// ---- lattice -----------------------------------------------------------------------

// Z2DetectPlanes runs the detector over an L³ periodic lattice of 3-frames in R^7
// (site index (i*L + j)*L + k).
func Z2DetectPlanes(L int, f [][3][7]float64, p Z2Params) Z2Result {
	return z2Lattice(L, func(a, b int) [3][3]float64 { return planeOverlap(&f[a], &f[b]) }, p)
}

// Z2DetectS3 is the S³ null adapter (frozen §4 control 2): q -> Ad_q ∈ SO(3), the
// frame being the 3x3 rotation matrix, with the global SU(2) lift q. It reuses the
// same transport and lift code, so any -1 is a false positive of that code.
func Z2DetectS3(L int, q [][4]float64, p Z2Params) Z2Result {
	rm := make([][3][3]float64, len(q))
	for i := range q {
		rm[i] = QuatToMat(q[i])
	}
	return z2Lattice(L, func(a, b int) [3][3]float64 { return rotOverlap(&rm[a], &rm[b]) }, p)
}

func z2Lattice(L int, ov func(a, b int) [3][3]float64, p Z2Params) Z2Result {
	N := L * L * L
	// neighbour table nbr[s][d] = s + e_d (periodic)
	nbr := make([][3]int, N)
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				s := (i*L+j)*L + k
				nbr[s][0] = (((i+1)%L)*L+j)*L + k
				nbr[s][1] = (i*L+(j+1)%L)*L + k
				nbr[s][2] = (i*L+j)*L + (k+1)%L
			}
		}
	}
	// edge lifts: E[d][s] = s -> s+e_d ; Dg[d][s] = face-d diagonal A -> C
	var E, Dg [3][][4]float64
	for d := 0; d < 3; d++ {
		E[d] = make([][4]float64, N)
		Dg[d] = make([][4]float64, N)
	}
	for s := 0; s < N; s++ {
		for d := 0; d < 3; d++ {
			E[d][s] = NearestRotationQuat(ov(s, nbr[s][d]))
			e1, e2 := (d+1)%3, (d+2)%3
			C := nbr[nbr[s][e1]][e2]
			Dg[d][s] = NearestRotationQuat(ov(s, C))
		}
	}
	// per-face triangle signs: tn[d][s][h], ta[d][s][h]
	type triRes struct{ neg, amb [2]bool }
	var tri [3][]triRes
	for d := 0; d < 3; d++ {
		tri[d] = make([]triRes, N)
		e1, e2 := (d+1)%3, (d+2)%3
		for s := 0; s < N; s++ {
			A := s
			B := nbr[A][e1]
			Dd := nbr[A][e2]
			var r triRes
			// triangle (A,B,C): A->B, B->C, C->A
			r.neg[0], r.amb[0] = TriangleZ2(E[e1][A], E[e2][B], QuatConj(Dg[d][A]), p.AmbigAngle)
			// triangle (A,C,D): A->C, C->D, D->A
			r.neg[1], r.amb[1] = TriangleZ2(Dg[d][A], QuatConj(E[e1][Dd]), QuatConj(E[e2][A]), p.AmbigAngle)
			tri[d][s] = r
		}
	}
	res := Z2Result{L: L, Cells: N}
	for s := 0; s < N; s++ {
		neg, amb := false, false
		for d := 0; d < 3; d++ {
			for _, face := range [2]int{s, nbr[s][d]} {
				r := tri[d][face]
				for h := 0; h < 2; h++ {
					if r.neg[h] {
						neg = !neg
					}
					if r.amb[h] {
						amb = true
					}
				}
			}
		}
		if neg {
			res.NegCells++
		}
		switch {
		case amb:
			res.Ambiguous++
			res.AmbigCells = append(res.AmbigCells, s)
		case neg:
			res.Defects++
			res.DefectCells = append(res.DefectCells, s)
		}
	}
	res.ParityOK = res.NegCells%2 == 0
	return res
}
