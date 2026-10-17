# Spike: Multi-Dimensional QBP-CU (Xqbp v2): verdicts

**Branch:** `spike/mdcu-xqbp-v2` (local only, not pushed) · **Package:** `emulator/mdcu/` (frozen v0.1.0-rc1 API not touched, per ADR-002)
**Host:** AMD FX-8350, no AVX2, Go 1.24.2. Benchmarks: `-count=6 -benchtime=200ms`, medians reported. Load average at start was 2.8 on 8 cores; the benchmarks are single-threaded. Raw output: [`raw/mdcu-bench-2026-10-16.txt`](raw/mdcu-bench-2026-10-16.txt).

> MACs, ISA model cycles and host ns are three separate measurements and are never merged. The cycle numbers come from the spike's `Tariff` and P-tile model (`sched.go`). They are a model, not silicon.

## 1. Verdict table

| # | Claim (source) | Claimed | Measured | Verdict |
|---|---|---|---|---|
| A0 | CD-MAC gradient (6 OMUL) ≡ basis gradient (34 OMUL) | exact equivalence | max ‖Δg‖ ≤ 1e-14 on S¹⁴, 20k samples | **CONFIRMED** |
| A1 | Gradient speed-up (Case §2) | 2,176 → 384 MAC (5.67×) | MAC 5.67× (doc) · instrumented OMUL 34 → 6 · host 4572 → 842 ns (**5.43×**) | **CONFIRMED** |
| A2a | Three engines agree on #688 (200 seeds, t=8) | ≤ 1.72e-15 | 2.55e-15 max across engines; ⟨b0²⟩ = 0.13365313 (exact match) | **CONFIRMED** |
| A2b | Quench cost | 696.3M / 122.9M / 3.85M MAC (basis/CD-MAC/leaf: 5.7× / 180.7×) | doc MACs reproduced exactly. Host: 1591 / 335.6 / 23.4 ms → **4.74× / 68.1×** (leaf vs CD-MAC 14.4× vs 31.9× claimed) | **CONFIRMED-ALGORITHMIC-ONLY**: MAC ratios hold; host speed-up is roughly ⅓ of the MAC ratio for the leaf (renormalisation and sqrt overheads dominate a 5D state) |
| A2c | Leaf closed form b0²(∞) | b0²/(b0²+√((1−b0²)²−V0)) | max err 2.2e-6 at t=20 with V0 = flowlib `Potential` (the /4 reading fails at 0.45) | **CONFIRMED** (V0 = N([a,b]) unscaled) |
| A3 | Leaf break-even vs split/join overhead | amortised over many steps | split+join = 252 ns vs CD-MAC 0-step 74 ns; at **1 step** leaf 583 ns vs CD-MAC 4312 ns | **CONFIRMED**: break-even before the first step |
| A4 | Albert h₃(𝕆) product, 6 OMUL (Silicon §4) | 66 → 9 cycles | formula ≡ brute 3×3 product (5.3e-15). Model ops 63 → 24 (P=1); 9 cycles = brute at P=8, fast reaches 7 at P=4 / 4 at P=8. Host 10.1 µs → 1.35 µs (7.5×) | **PARTIAL**: algebra confirmed. The "66 → 9" figure mixes tile counts; same-P ratio is ≈ 2.2–2.6× in cycles |
| A5 | Pillar 2 Test A (bounce) | canon results.json | pure_gravity, g2_conserved, high_inertia **exact** to 5 digits (r_min, final r, ‖A‖², steps). fast_slip: plunge and step count (2374) match, r_min 0.05597 vs 0.05968, final 0.03689 vs 0.04288. QASSOC 201,360 vs 194,248 | **PARTIAL** (3/4 exact; source never shared, so the fast-slip boundary semantics are inferred) |
| A5 | Pillar 2 Test B (frame locking) | canon | initial / local_only / locked: local, global and SVD-3 match canon to all printed digits (metric = mean ‖A‖²) | **CONFIRMED** (`max_coordinate_fano_share` not reproduced: undefined in spec) |
| A5 | Pillar 2 Test C (law sieve) | canon | t0 exact (0.102467 / 0.618782 / 1.16032, weights 0.2452/0.2392/0.5156). t15 0.02834/0.2939/0.8500 vs 0.028105/0.291974/0.875411. t100 0.00076/0.0445/0.196 vs 0.000932/0.048362/0.195015, H 0.989 vs 0.9872. Ledgers 30.61/20.30 vs 31.59/19.34 | **PARTIAL**: qualitative gate passes (monotone S→O→H drain, H→~99%), exact match fails. Alternativity metric is `[x,y,y]` (calibrated at t0, where no dynamics are involved). The drain uses the prose `[x,x,y]`. Two other orderings were tried and also missed; I stopped there rather than tune |
| A6 | benchstat vs `doc/benchmarks` baseline | — | not run: the mdcu kernels have no counterpart in the frozen baseline suite | **NOT RUN** |

## 2. Ladder capability matrix (question b)

| Rung | Op(s) | Correctness gate | Result | Cost (model cycles) | Host |
|---|---|---|---|---|---|
| Tier ℝ↔ℂ↔ℍ↔𝕆↔𝕊 | QLIFT / QSPLIT / QJOIN | T-L1 bit-exact round-trip | ✅ 10k/10k | 1 per rung | wiring |
| Fano frames (7) | QSPLIT/QJOIN f=1..7 | T-L2 bit-exact; G₂-torus invariance of the frame average | ✅ automorphism defect 1.5e-14, avg invariant 5e-15, single frame NOT invariant (as predicted) | 1; 7-frame avg 8 → 2 (P=8) | — |
| Lossy down-step | QPROJ + ledger | T-L3 ‖bulk‖²+ledger = 1 | ✅ 6.7e-16 | 1 | — |
| Leaf 𝕊 ↔ 5D | QSPLIT.LEAF / QJOIN.LEAF | T-L4 per-step ≤ 2e-15 | ✅ 2.2e-16 per step; 2.1e-15 at t=8 (doc 1.72e-15) | 2 + 2 | 169 ns round-trip |
| Leaf escape | LEAFCHK vs **CSR-locked** frame | T-L5 detects off-plane; no false trips | ✅ unperturbed r² ≤ 3.8e-30. A 1e-6 perturbation trips; a fresh re-extraction would miss it (asserted) | 2 | — |
| Degenerate leaf | fallback frame | T-L5b orthonormal + round-trip | ✅ 4/4 cases | — | — |
| Width QW64↔QW128 | QWIDEN / QNARROW | T-L6 widen exact, narrow ≤ 0.5 ulp, seam flag | ✅ exact; 0.000 ulp; seam fires | 1 | — |
| Diagonal | lift∘OMUL = SMUL∘lift | T-L7 bit-exact | ✅ 20k/20k | — | — |
| Soak | 10⁶ leaf steps | T-L8 drift, escape | ✅ ‖q‖²−1 = 0, escape r² 3.6e-33 | — | 0.27 s |
| ISA core | `CPU.Exec` program | split/join/assoc/leaf via Exec | ✅ 11 cycles, 7 ops | — | — |

**Answer to (b):** yes. Every rung of the ladder (tier, Fano frame, width, leaf) round-trips exactly or within its stated budget. Lossy steps are fully accounted for in the ledger, and leaf escape is detected against the locked frame. All rungs cost 1–2 model cycles.

## 3. Tile model (L9; model cycles)

| kernel | ops | P=1 | P=2 | P=4 | P=8 | P=16 |
|---|---|---|---|---|---|---|
| albert_fast | 24 | 24 | 12 | 7 | 4 | 3 |
| albert_brute | 63 | 63 | 32 | 17 | 9 | 5 |
| frame_avg7 | 8 | 8 | 5 | 3 | 2 | 2 |
| grad_basis | 35 | 35 | 18 | 10 | 6 | 4 |
| grad_cdmac | 7 | 7 | 4 | 3 | 2 | 2 |
| leaf_roundtrip | 2 | 4 | 4 | 4 | 4 | 4 |
| qassoc | 1 | 2 | 2 | 2 | 2 | 2 |

## 4. Kernel microbenchmarks (host ns, medians of 6)

| kernel | ns/op |
|---|---|
| OMUL CD-MAC / recursive | 102.5 / 105.1 |
| SMUL CD-MAC / recursive | 418.1 / 482.6 |
| Leaf split+join | 168.8 |

## 5. Caveats
- FX-8350, no AVX2: CD-MAC's 64/256-way parallelism is serialised on the host. The MAC and cycle columns are the architectural claims; ns only shows that the algorithmic win survives on scalar hardware.
- Pillar 2 was ported from a prose spec. The original `fix_test_a.py` / `run_fault_line_tests.py` were not available. Exact matches (Test A ×3, Test B) give strong evidence that the port is faithful. The remaining mismatches are reported unchanged.
- G₂ maximal torus used: fix e₁ and rotate (e₂,e₃) by θ₁, (e₄,e₅) by θ₂, (e₆,e₇) by θ₁+θ₂.
- Physics claims (KK, α, CTH anchors) are out of scope.
