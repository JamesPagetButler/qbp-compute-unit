package mdcu

// Albert algebra h₃(𝕆) Jordan product (Silicon doc §4, A4).
//
// X = [[ξ1, x3, x̄2], [x̄3, ξ2, x1], [x2, x̄1, ξ3]] (Hermitian, Freudenthal
// convention). The fast path follows web-Gemini's formula (6 OMUL, 3 dots,
// 6 scalings); AlbertBrute forms XY+YX entry-by-entry and is the referee.

type Albert struct {
	D [3]float64
	X [3][8]float64
}

func oc(a [8]float64) [8]float64 { return oconj(a) }

func om(a, b [8]float64) [8]float64 { return OMulCDMAC(&a, &b) }

func oadd(a, b [8]float64, s float64) (z [8]float64) {
	for i := range z {
		z[i] = a[i] + s*b[i]
	}
	return
}

// matrix entry (r,c) as an octonion.
func (A *Albert) at(r, c int) (z [8]float64) {
	switch {
	case r == c:
		z[0] = A.D[r]
	case r == 0 && c == 1:
		z = A.X[2]
	case r == 1 && c == 0:
		z = oc(A.X[2])
	case r == 1 && c == 2:
		z = A.X[0]
	case r == 2 && c == 1:
		z = oc(A.X[0])
	case r == 2 && c == 0:
		z = A.X[1]
	case r == 0 && c == 2:
		z = oc(A.X[1])
	}
	return
}

// AlbertBrute computes ½(XY+YX) by 3×3 octonionic matrix products (54 OMUL).
func AlbertBrute(X, Y *Albert) (Z Albert) {
	var M [3][3][8]float64
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			for k := 0; k < 3; k++ {
				M[r][c] = oadd(M[r][c], om(X.at(r, k), Y.at(k, c)), 0.5)
				M[r][c] = oadd(M[r][c], om(Y.at(r, k), X.at(k, c)), 0.5)
			}
		}
	}
	for i := 0; i < 3; i++ {
		Z.D[i] = M[i][i][0]
	}
	Z.X[2], Z.X[0], Z.X[1] = M[0][1], M[1][2], M[2][0]
	return
}

// AlbertFast is the 6-OMUL closed form.
func AlbertFast(X, Y *Albert, ctr *Counter) (Z Albert) {
	for i := 0; i < 3; i++ {
		j, k := (i+1)%3, (i+2)%3
		Z.D[i] = X.D[i]*Y.D[i] + dot(X.X[j][:], Y.X[j][:]) + dot(X.X[k][:], Y.X[k][:])
		var z [8]float64
		z = oadd(z, Y.X[i], 0.5*(X.D[j]+X.D[k]))
		z = oadd(z, X.X[i], 0.5*(Y.D[j]+Y.D[k]))
		z = oadd(z, om(oc(X.X[k]), oc(Y.X[j])), 0.5)
		z = oadd(z, om(oc(Y.X[k]), oc(X.X[j])), 0.5)
		ctr.omul(2)
		Z.X[i] = z
	}
	return
}
