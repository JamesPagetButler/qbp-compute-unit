# MDCU round 2 — Q2 implementation addendum (pre-run, for review)

Status: **DRAFT for qbp-architecture review. Q2 is built and tested; it has NOT been run.**
Parent: `prereg/mdcu-round2-prereg-FROZEN-v3.md` (sha256 28bcaa6f…cae374, section 4). This addendum
only fixes details the frozen text left open. It changes no threshold and no decision rule.

## 1. What was built

| Item | File |
|---|---|
| Albert determinant, 4 variants, instrumented tape | `emulator/mdcu/q2.go` |
| Tests (agreement, reference, tape = analytic count, scheduler, admission) | `emulator/mdcu/q2_test.go` |
| Driver (3 separate columns, 2 bounds, 10 min cap) | `emulator/cmd/mdcu-q2/main.go` |

det(X) = abc − a|x|² − b|y|² − c|z|² + 2·Re((xy)z).

| Variant | Side | Re((xy)z) path |
|---|---|---|
| B1lit | B1 | two OMUL (CD-MAC, 64 MACs each), scalar part taken |
| B1lean | B1 | one OMUL (64) + an 8-term signed dot (8) — sensitivity |
| B0a | B0 | sparse 3-form T_ij (64 nonzeros) contraction, x_i·y_j then ·z_k, 128 multiplies |
| B0b | B0 | factored: scalar/vector terms + seven Fano-line 3×3 determinants, 89 multiplies |

Every evaluator takes a tape; op counts and the model-cycle DAG come from the **same code path**
(qbp-cu-implementor seq 4). A test asserts tape count = analytic count per variant.

## 2. Details fixed here (open in the frozen text)

1. **Cost model (assumed, not silicon).** A tile has 64 multiplier lanes + adder tree; one batch of
   ≤64 multiplies costs one tile-cycle. This is the same assumption that gives OMUL tariff 1 in `isa.go`.
   Both sides use it. A batch with 0 multiplies (the final combine) costs one cycle on both sides.
2. **Two bounds, both sides** (`sched.go` takes only level lists, so the frozen text's fallback applies):
   (i) ideal throughput = ceil(total multiplies / (64·P)), perfect lane packing, no stalls;
   (ii) dependency-aware = greedy list schedule of the DAG on P tiles.
3. **Workload for the decision:** a batch of 64 independent determinants at P=8 (throughput).
   Single-instance latency and P ∈ {1,2,4,8,16} are reported, not decisive.
4. **Which variants decide.** B1 = **B1lit** (the literal CD-MAC path in the frozen text).
   B0 = the **fewest-multiply** of {B0a, B0b}. **B1lean is a sensitivity**: if B1lean changes the
   verdict on any column, the reported verdict is "inconclusive (B1-variant-sensitive)".
5. **Host ns:** `testing.Benchmark`, GOMAXPROCS=1, 6 samples per variant, median; only on a quiet box
   (load average at start is recorded in the output; the run is repeated if it is > 4 and nothing else
   is pending). Result is per determinant.
6. **Wall cap:** 10 min (plan section 10); expected ≲ 1 min.
7. **Contract-then-dot is not B0.** Contracting the 64-term tensor over x and y first and then dotting
   with z *is* the CD-MAC product plus an 8-term dot (= B1lean), so it is not counted as a non-CD
   baseline. This is itself a finding to report: the "non-CD tensor contraction" and the CD-MAC are the
   same bilinear map.

## 3. Disclosures (what I have seen before the run)

- While writing tests I know the **analytic multiply counts**: per determinant B1lit 157, B1lean 101,
  B0a 157, B0b 118 (29 common + variant). These are deterministic by construction, not measurements.
- I have **not** seen any cycle bound, host ns, or ratio. The driver was built and vetted, not run.
- The frozen prediction ("B0 matches or beats B1 in MACs and host ns") is therefore **variant-dependent
  on MACs**: true against B1lit (118 < 157), false against B1lean (101 < 118). The decision uses B1lit
  per §2.4; B1lean is reported so this cannot be hidden.
- Agreement gate seen: worst normwise disagreement 5.8e-16 over 20,000 instances (limit 1e-12).
- Admission: G₂-torus change 4.6e-14, generic SO(7) change ≈ 103 (scale ≈ 109): passes (RT3, RT4).
  The determinant fails RT2 by construction (it contains the triple term).

## 4. Questions for qbp-architecture

1. Is **B1lit as the deciding B1, B1lean as sensitivity** acceptable, or should the decision use the
   best B1 variant (which is the stronger test of "MDCU beats the best equally-informed baseline")?
2. Is the **64-lane tile, one batch per tile-cycle** model an acceptable symmetric assumption for B0
   (a scalar-lane sparse contraction)? It favors B1 slightly on tile-batch granularity, which is why
   bound (i) packs lanes perfectly.
3. Batch of 64 at P=8 as the decisive workload: OK?

## 5. Gates before the run

addendum review (qbp-architecture) → deming box-safety for Q2 (1 core, nice 19, ≲ 1 min, tiny output)
→ beekeeper run go. Nothing runs before all three.
