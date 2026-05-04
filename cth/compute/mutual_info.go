package compute

import (
	"math"
)

// PairwiseMI computes the Gaussian mutual information between two predictions (Definition 10).
// predA, predB: predicted values; sigmaA, sigmaB: prediction uncertainties.
func PairwiseMI(predA, predB, sigmaA, sigmaB float64) float64 {
	const epsilon = 1e-15
	diff := predA - predB
	numerator := sigmaA*sigmaA + sigmaB*sigmaB
	denominator := diff*diff + epsilon
	return 0.5 * math.Log2(1.0+numerator/denominator)
}

// NaryMI computes the total correlation (multivariate mutual information) for N predictions
// (Definition 10).  T(X₁,…,Xₙ) = −½ log₂ det(R) where R is the n×n Gaussian-kernel
// correlation matrix built from the prediction values and uncertainties.
//
// For perfect agreement (det → 0), this returns +Inf.
// For three or more partially-correlated paths, T > sum of pairwise MIs (synergy term).
// For N = 2, this reduces to PairwiseMI.
func NaryMI(predictions []float64, sigmas []float64) float64 {
	n := len(predictions)
	if n < 2 {
		return 0
	}
	if n == 2 {
		return PairwiseMI(predictions[0], predictions[1], sigmas[0], sigmas[1])
	}

	// Build n×n correlation matrix.
	// ρ_ij = exp(−½ · (p_i−p_j)² / (σ_i²+σ_j²))  — Gaussian kernel agreement in [0,1].
	r := make([][]float64, n)
	for i := range r {
		r[i] = make([]float64, n)
		r[i][i] = 1.0
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			diff := predictions[i] - predictions[j]
			varSum := sigmas[i]*sigmas[i] + sigmas[j]*sigmas[j]
			if varSum < 1e-30 {
				varSum = 1e-30
			}
			rho := math.Exp(-0.5 * diff * diff / varSum)
			r[i][j] = rho
			r[j][i] = rho
		}
	}

	d := matDet(r)
	if d <= 0 {
		return math.Inf(1) // Perfectly correlated (or numerically singular)
	}
	return -0.5 * math.Log2(d)
}

// CappedMI bounds the mutual information by the weakest channel's capacity (Definition 10a).
func CappedMI(mi float64, chainCapacities []float64) float64 {
	if len(chainCapacities) == 0 {
		return mi
	}
	minCap := chainCapacities[0]
	for _, c := range chainCapacities[1:] {
		if c < minCap {
			minCap = c
		}
	}
	if mi > minCap {
		return minCap
	}
	return mi
}

// StructuralMI computes MI for structural confluences (yes/no, integer quantum numbers).
// Returns min(arity, minCapacity) bits (Definition 10a).
func StructuralMI(arity int, minCapacity float64) float64 {
	if float64(arity) > minCapacity {
		return minCapacity
	}
	return float64(arity)
}

// matDet computes the determinant of a square matrix using Gaussian elimination
// with partial pivoting.
func matDet(m [][]float64) float64 {
	n := len(m)
	if n == 0 {
		return 1
	}
	if n == 1 {
		return m[0][0]
	}

	// Deep copy to avoid mutating the caller's matrix.
	a := make([][]float64, n)
	for i := range a {
		a[i] = make([]float64, n)
		copy(a[i], m[i])
	}

	det := 1.0
	for col := 0; col < n; col++ {
		// Find the row with the largest absolute value in this column (partial pivoting).
		maxRow := col
		for row := col + 1; row < n; row++ {
			if math.Abs(a[row][col]) > math.Abs(a[maxRow][col]) {
				maxRow = row
			}
		}
		if maxRow != col {
			a[col], a[maxRow] = a[maxRow], a[col]
			det = -det
		}
		if math.Abs(a[col][col]) < 1e-15 {
			return 0 // Singular — perfectly correlated columns.
		}
		det *= a[col][col]
		for row := col + 1; row < n; row++ {
			factor := a[row][col] / a[col][col]
			for c := col; c < n; c++ {
				a[row][c] -= factor * a[col][c]
			}
		}
	}
	return det
}
