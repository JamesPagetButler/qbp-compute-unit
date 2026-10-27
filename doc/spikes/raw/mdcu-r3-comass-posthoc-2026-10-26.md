# R3-Q1 comass diagnostic — POST-HOC

**POST-HOC.** Not part of the frozen R3-Q1 analysis; no verdict, reading row or rule changes.
Per-seed comass quantities are recomputed from each seed's own sampler streams (the frozen samplers are
called unchanged); no ring runs. Associations are descriptive. `comass` here = max |f(u,v,w)| over orthonormal
triples (multistart ascent, so a lower bound); comass(φ) = 1, so the number is also the ratio to φ.

Input: `../doc/spikes/raw/mdcu-r3-2026-10-26.jsonl` (200 seeds).

**Calibration**: estimator on φ = 1.000000000 (exact: 1).
**Convergence check** (two independent restart sets per form): max relative difference 1.31e-15, median 2.15e-16.

## Comass of the ψ forms actually used (Frobenius-matched to φ)

| form | n | min | median | max |
|---|---|---|---|---|
| K-psi, definite seeds (S2) | 100 | 1.240 | 1.630 | 2.234 |
| K-psi-S1, definite seeds (S1-definite) | 100 | 1.546 | 1.928 | 2.286 |
| K-psi, split seeds (S1-split) | 100 | 1.754 | 1.983 | 2.250 |
| ‖g‖_op³ (S2 bound, pre-match) | 100 | 1.538 | 3.459 | 13.040 |

Descriptive: with −4ψ², the peak coupling is 4·comass², i.e. comass² times φ's. Median comass² on S2 = 2.66, on S1-definite = 3.72, on S1-split = 3.93.

Pooled K-psi forms (S2 on definite seeds + S1-split on split seeds, n=200): comass median 1.880 [1.240, 2.250], comass² median 3.536.

## Does comass grow with the deformation? (definite seeds, S2)

- Spearman(comass, d) = 0.972; Spearman(comass, s) = 0.879; Spearman(‖g‖_op³, d) = 0.952; Spearman(‖g‖_op³, comass) = 0.909 (n=100)

## Does comass track locking? (main cell, per stratum)

Caveat: S3 is saturated near 1 (188 of 200 K-psi runs have S3 ≥ 0.999), so rank correlations on S3 are driven by tiny differences;
the 3-lock rate by tercile below is the cleaner view.

- **definite** (n=100): Spearman(comass, S3 K-psi) = 0.661; Spearman(comass, S3 K-psigram) = 0.340

- **split** (n=100): Spearman(comass, S3 K-psi) = -0.004; Spearman(comass, S3 K-psigram) = -0.009

### K-psi / K-psigram by comass tercile — definite (S2) (n=100)

| comass tercile | comass range | K-psi 3-lock | K-psi median S3 | K-psigram 3-lock | K-psigram median S3 |
|---|---|---|---|---|---|
| 1 | 1.240 – 1.470 | 90.9% | 1.000 | 69.7% | 0.974 |
| 2 | 1.488 – 1.779 | 81.8% | 1.000 | 57.6% | 0.998 |
| 3 | 1.782 – 2.234 | 76.5% | 1.000 | 88.2% | 0.999 |

### K-psi-S1 by comass tercile — definite (S1-definite form) (n=100)

| comass tercile | comass range | K-psi-S1 3-lock | K-psi-S1 median S3 |
|---|---|---|---|
| 1 | 1.546 – 1.852 | 66.7% | 1.000 |
| 2 | 1.856 – 1.969 | 60.6% | 1.000 |
| 3 | 1.972 – 2.286 | 73.5% | 1.000 |

### K-psi / K-psigram by comass tercile — split (S1-split) (n=100)

| comass tercile | comass range | K-psi 3-lock | K-psi median S3 | K-psigram 3-lock | K-psigram median S3 |
|---|---|---|---|---|---|
| 1 | 1.754 – 1.932 | 72.7% | 1.000 | 81.8% | 1.000 |
| 2 | 1.934 – 2.020 | 72.7% | 1.000 | 66.7% | 0.998 |
| 3 | 2.023 – 2.250 | 79.4% | 1.000 | 79.4% | 0.999 |

## Dose-response, with and without comass (definite seeds)

Same quantity as the frozen analyzer, S3(K-phi) − S3(K-psi) vs d: n=100, ρ = -0.258 (the frozen analyzer reports −0.258).
- Spearman(gap, comass) = -0.240
- Partial Spearman(gap, d | comass) = -0.108

Caveats: (1) comass and d are rank-collinear on S2 (Spearman 0.972), so the partial is poorly determined and cannot separate them.
(2) Every ψ form has comass above φ's (φ = 1.000; smallest ψ = 1.240), so R3 has no ψ at φ's coupling scale: the comass
confound between arms cannot be tested within R3 data. Within-form comass variation is the only handle, and it is weak and non-monotone.
R4's rescaled-ψ arm (max|ψ| = 1) is the pre-registered test; this report only motivates it. No verdict is changed.
