# R3-Q1 analysis (frozen rules, prereg sha256 070aeab8…c1848)
input ../doc/spikes/raw/mdcu-r3-2026-10-26.jsonl · mode=grid · first_seed=3000 seeds=200 · run end: end: complete: 4900 runs

## Sampler record (reported alongside every verdict, arch seq 42)
- **definite** (100 seeds, S1 (sensitivity K-psi-S1; primary ψ is S2)): S1 draws min 3 · median 1100 · max 9496 · mean 1588 · misses 0
  histogram (draws ≥ bin): [2+]=1 [8+]=1 [32+]=1 [64+]=4 [128+]=9 [256+]=13 [512+]=18 [1024+]=28 [2048+]=17 [4096+]=7 [8192+]=1 
- **split** (100 seeds, S1 (primary)): S1 draws min 1 · median 1 · max 2 · mean 1 · misses 0
  histogram (draws ≥ bin): [1+]=99 [2+]=1 
- **S2** (definite seeds, n=100): s median 0.157 · distance d median 0.383 [0.132, 0.673] · metric condition median 5.6

## Main cell (N=24, σ=0.1, 250 steps) — all seeds
| arm | n | 3-lock | 2-collapse | diffuse | median S3 | median λ₃ share |
|---|---|---|---|---|---|---|
| K-full | 200 |  99.0% |   0.0% |   1.0% | 1.000 | 0.278 |
| K-gram | 200 |   3.0% |  75.5% |  21.5% | 1.000 | 0.000 |
| K-phi | 200 |   0.0% |   0.0% | 100.0% | 0.642 | 0.179 |
| K-psi | 200 |  79.0% |   0.0% |  21.0% | 1.000 | 0.200 |
| K-psi-S1 | 100 |  67.0% |   0.0% |  33.0% | 1.000 | 0.180 |
| K-psigram | 200 |  74.0% |   0.0% |  26.0% | 0.999 | 0.309 |
| K-rand3 | 200 |  33.5% |   0.0% |  66.5% | 0.914 | 0.211 |
| K-none | 200 |   0.0% |   0.0% | 100.0% | 0.601 | 0.168 |

### stratum: definite
| arm | n | 3-lock | 2-collapse | diffuse | median S3 | median λ₃ share |
|---|---|---|---|---|---|---|
| K-full | 100 |  99.0% |   0.0% |   1.0% | 1.000 | 0.280 |
| K-gram | 100 |   3.0% |  79.0% |  18.0% | 1.000 | 0.000 |
| K-phi | 100 |   0.0% |   0.0% | 100.0% | 0.641 | 0.175 |
| K-psi | 100 |  83.0% |   0.0% |  17.0% | 1.000 | 0.298 |
| K-psi-S1 | 100 |  67.0% |   0.0% |  33.0% | 1.000 | 0.180 |
| K-psigram | 100 |  72.0% |   0.0% |  28.0% | 0.999 | 0.317 |
| K-rand3 | 100 |  33.0% |   0.0% |  67.0% | 0.916 | 0.210 |
| K-none | 100 |   0.0% |   0.0% | 100.0% | 0.602 | 0.168 |

### stratum: split
| arm | n | 3-lock | 2-collapse | diffuse | median S3 | median λ₃ share |
|---|---|---|---|---|---|---|
| K-full | 100 |  99.0% |   0.0% |   1.0% | 1.000 | 0.275 |
| K-gram | 100 |   3.0% |  72.0% |  25.0% | 1.000 | 0.000 |
| K-phi | 100 |   0.0% |   0.0% | 100.0% | 0.642 | 0.179 |
| K-psi | 100 |  75.0% |   0.0% |  25.0% | 1.000 | 0.185 |
| K-psigram | 100 |  76.0% |   0.0% |  24.0% | 0.999 | 0.290 |
| K-rand3 | 100 |  34.0% |   0.0% |  66.0% | 0.914 | 0.211 |
| K-none | 100 |   0.0% |   0.0% | 100.0% | 0.601 | 0.168 |

## Decision rules

**Replication gate**: K-full 3-lock rate  99.0% (needs ≥ 90%) → **pass**

**Q1a** — Δ(K-full, K-gram): n=200 rates  99.0% vs   3.0%, Δ median -0.000, 95% CI [-0.000, -0.000]
→ **no**

**Q1b (definite)** — Δ(K-phi, K-psi) definite: n=100 rates   0.0% vs  83.0%, Δ median -0.356, 95% CI [-0.367, -0.345]
→ **inconclusive (φ-only does not lock)**

**Q1b (split)** — Δ(K-phi, K-psi) split: n=100 rates   0.0% vs  75.0%, Δ median -0.358, 95% CI [-0.366, -0.343]
→ **inconclusive (φ-only does not lock)**

**Reading** (Q1a=no, Q1b definite=inconclusive (φ-only does not lock), Q1b split=inconclusive (φ-only does not lock)):
> Inconclusive, or a mixed result reported as-is. (Row 5)

## Secondaries (diagnostic only; never a decision input)

**Dose-response** (definite seeds; Spearman of paired S3(K-phi)−S3(K-psi) vs S2 distance d): n=100, ρ=-0.258
(Euclidean-vs-g_ψ mismatch is the Q1b question: ψ is measured in the Euclidean ring while its own metric g_ψ differs, per qbp-oppenheimer.)

**K-phi vs K-full / K-psigram vs K-full**
- Δ(K-phi, K-full) all: n=200 rates   0.0% vs  99.0%, Δ median -0.356, 95% CI [-0.362, -0.347]
- Δ(K-psigram, K-full) definite: n=100 rates  72.0% vs  99.0%, Δ median -0.001, 95% CI [-0.002, -0.000]
- Δ(K-psigram, K-full) split: n=100 rates  76.0% vs  99.0%, Δ median -0.000, 95% CI [-0.001, -0.000]

**Sampler sensitivity** (Q1b rule with K-psi-S1; not a decision input) — Δ(K-phi, K-psi-S1) definite: n=100 rates   0.0% vs  67.0%, Δ median -0.359, 95% CI [-0.368, -0.347]
→ inconclusive (φ-only does not lock)

**Cross-calibration** (main cell; own-form calibration is tautological) — mean |φ₀| plane / |φ₀|≥0.95 triple fraction / |ψ₀| plane / |ψ₀|≥0.95 fraction
| arm | stratum | φ₀ plane | φ₀ frac | ψ₀ plane | ψ₀ frac |
|---|---|---|---|---|---|
| K-full | definite | 1.000 | 0.977 | 0.917 | 0.389 |
| K-full | split | 1.000 | 0.985 | 0.365 | 0.041 |
| K-gram | definite | 0.356 | 0.002 | 0.351 | 0.014 |
| K-gram | split | 0.343 | 0.006 | 0.400 | 0.025 |
| K-phi | definite | 0.699 | 0.330 | 0.628 | 0.244 |
| K-phi | split | 0.751 | 0.337 | 0.347 | 0.027 |
| K-psi | definite | 0.874 | 0.443 | 1.505 | 1.000 |
| K-psi | split | 0.369 | 0.000 | 1.980 | 1.000 |
| K-psi-S1 | definite | 0.360 | 0.000 | 1.906 | 1.000 |
| K-psigram | definite | 0.742 | 0.000 | 1.349 | 0.999 |
| K-psigram | split | 0.374 | 0.007 | 1.593 | 1.000 |
| K-rand3 | definite | 0.344 | 0.002 | 0.343 | 0.014 |
| K-rand3 | split | 0.353 | 0.005 | 0.342 | 0.022 |
| K-none | definite | 0.410 | 0.004 | 0.415 | 0.019 |
| K-none | split | 0.368 | 0.002 | 0.353 | 0.030 |
(|ψ₀| is not bounded by 1: ψ is Frobenius-matched to φ's norm but is not Euclidean-compatible.)

**Persistence** (secondary; same rules applied for information only)

### N12 (250 steps)
| arm | n | 3-lock | 2-collapse | diffuse | median S3 | median λ₃ share |
|---|---|---|---|---|---|---|
| K-full | 200 |  99.0% |   0.0% |   1.0% | 1.000 | 0.259 |
| K-gram | 200 |  10.5% |  77.5% |  12.0% | 1.000 | 0.000 |
| K-phi | 200 |   1.0% |   0.0% |  99.0% | 0.738 | 0.213 |
| K-psi | 200 |  61.0% |   0.5% |  38.5% | 1.000 | 0.163 |
| K-psi-S1 | 100 |  38.0% |   1.0% |  61.0% | 1.000 | 0.142 |
| K-psigram | 200 |  92.5% |   0.0% |   7.5% | 1.000 | 0.318 |
| K-rand3 | 200 |  75.5% |   0.0% |  24.5% | 0.998 | 0.218 |
| K-none | 200 |   0.0% |   0.0% | 100.0% | 0.680 | 0.175 |

- Q1a-rule: Δ(K-full, K-gram) N12: n=200 rates  99.0% vs  10.5%, Δ median +0.000, 95% CI [-0.000, +0.000] → no
- Q1b-rule definite: Δ(K-phi, K-psi) N12 definite: n=100 rates   0.0% vs  81.0%, Δ median -0.285, 95% CI [-0.310, -0.258] → inconclusive (φ-only does not lock)
- Q1b-rule split: Δ(K-phi, K-psi) N12 split: n=100 rates   2.0% vs  41.0%, Δ median -0.234, 95% CI [-0.279, -0.225] → inconclusive (φ-only does not lock)

### N48 (250 steps)
| arm | n | 3-lock | 2-collapse | diffuse | median S3 | median λ₃ share |
|---|---|---|---|---|---|---|
| K-full | 200 |  18.5% |   0.0% |  81.5% | 0.892 | 0.249 |
| K-gram | 200 |   9.0% |  49.5% |  41.5% | 0.996 | 0.043 |
| K-phi | 200 |   0.0% |   0.0% | 100.0% | 0.574 | 0.165 |
| K-psi | 200 |  90.5% |   0.0% |   9.5% | 1.000 | 0.235 |
| K-psi-S1 | 100 |  96.0% |   0.0% |   4.0% | 1.000 | 0.210 |
| K-psigram | 200 |  46.5% |   0.0% |  53.5% | 0.946 | 0.299 |
| K-rand3 | 200 |   0.5% |   0.0% |  99.5% | 0.786 | 0.189 |
| K-none | 200 |   0.0% |   0.0% | 100.0% | 0.554 | 0.160 |

- Q1a-rule: Δ(K-full, K-gram) N48: n=200 rates  18.5% vs   9.0%, Δ median -0.089, 95% CI [-0.100, -0.074] → no
- Q1b-rule definite: Δ(K-phi, K-psi) N48 definite: n=100 rates   0.0% vs  86.0%, Δ median -0.421, 95% CI [-0.426, -0.415] → inconclusive (φ-only does not lock)
- Q1b-rule split: Δ(K-phi, K-psi) N48 split: n=100 rates   0.0% vs  95.0%, Δ median -0.421, 95% CI [-0.432, -0.417] → inconclusive (φ-only does not lock)

**N=48 convergence (K-full, K-gram at 250 vs 1000 steps)**
| arm | steps | n | 3-lock | median S3 |
|---|---|---|---|---|
| K-full | 250 | 200 |  18.5% | 0.892 |
| K-full | 1000 | 200 |  75.5% | 0.989 |
| K-gram | 250 | 200 |   9.0% | 0.996 |
| K-gram | 1000 | 200 |   6.0% | 1.000 |

**Spontaneity** (round-2 definition: KS D of pairwise plane distances vs Haar-G₂ reference, MC p over 1000 reps; arms with ≥ 20 three-locking runs; spontaneous iff p > 0.05)
- K-full: n=198 locked planes, D=0.0080, p=0.545 → consistent with Haar-G₂ (spontaneous)
- K-psi: n=158 locked planes, D=0.0838, p=0.000 → not Haar-G₂ (non-spontaneous)
- K-psi-S1: n=67 locked planes, D=0.0941, p=0.000 → not Haar-G₂ (non-spontaneous)
- K-psigram: n=148 locked planes, D=0.0849, p=0.000 → not Haar-G₂ (non-spontaneous)
- K-rand3: n=67 locked planes, D=0.1140, p=0.000 → not Haar-G₂ (non-spontaneous)

**On-file prediction (r1 + addendum, unedited in the frozen file):** Q1a = no; Q1b = no in both strata; reading row 2. Score it against the verdicts above in the results document.
