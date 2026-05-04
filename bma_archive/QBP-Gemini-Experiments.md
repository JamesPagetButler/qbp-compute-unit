# QBP Compute Unit — Follow-Up Experiments for Gemini

**Context:** You've just completed Phase 1 (full), Phase 2 (partial), and Phase 4 (partial) of the QBP technology evaluation. While you still have the codebase and context loaded, these experiments would significantly inform the Walk-phase integration plan for BMA.

---

## Experiment A: QW8 Precision Boundary for FATHOM Operations

**Why:** The width parameterization finding (QW16 insufficient for physics, QW8 fine for hypergraph) raises the question: where exactly does QW8 break down for BMA-specific operations? FATHOM does embedding comparisons, cosine similarity, and pattern matching. These are more precision-sensitive than simple graph propagation.

**Test:**
1. Generate 1,000 synthetic embedding vectors (128-dimensional, float32 — typical embedding output)
2. Quantize them to QW8 (int8)
3. Compute cosine similarity between all pairs at float32 and at QW8
4. Measure: what is the rank correlation between float32 similarities and QW8 similarities?
5. Specifically: do the top-10 nearest neighbours change between float32 and QW8?
6. Repeat with 256-dimensional and 512-dimensional embeddings

**What we learn:** If top-10 NN retrieval is preserved at QW8, FATHOM's observation matching can run entirely at QW8 with no quality loss. If not, we know the precision floor for FATHOM and can route accordingly.

---

## Experiment B: Octonionic vs Scalar Edge Retrieval Quality

**Why:** The Walk evaluation lists octonionic edges as an experiment (Walk Step 6). You have the algebra infrastructure loaded right now. We can get early signal.

**Test:**
1. Build a small knowledge graph (1,000 nodes, 5,000 edges) with known semantic relationships — e.g., a subset of ConceptNet or a hand-crafted domain ontology
2. Assign scalar edge weights (float32 salience scores) — the Crawl representation
3. Assign octonionic edge weights (Quat8) — encode edge TYPE in the algebraic structure:
   - Quaternionic subalgebra for IS-A and HAS-PROPERTY (associative, deterministic)
   - Full octonionic for RELATED-TO and REMINDS-OF (non-associative, context-dependent)
4. Run retrieval queries: "given node X, find the most relevant nodes for context Y"
5. Compare: do octonionic edges produce better retrieval rankings than scalar edges?
6. Define "better" as: higher precision@10 against human-judged relevance labels

**What we learn:** Whether the algebraic structure actually helps retrieval, or whether it's elegant theory with no practical benefit. This is the make-or-break experiment for octonionic edges in BMA.

---

## Experiment C: Spreading Activation vs HNSW for Retrieval

**Why:** The Walk evaluation assumes spreading activation can match or exceed HNSW embedding search. This needs testing with the actual Quat8 edge propagation you've already benchmarked.

**Test:**
1. Using the 100K-node / 500K-edge graph from Phase 4
2. Insert 100 "query" nodes with known ground-truth nearest neighbours
3. Retrieval method 1: HNSW search on float32 embeddings (the standard approach)
4. Retrieval method 2: Spreading activation from the query node using Quat8 edges
5. Compare: precision@10, recall@10, and latency for both methods

**What we learn:** Whether spreading activation is a viable replacement for HNSW, a complement, or neither. If spreading activation retrieves different (but still relevant) results, it's a complement — use both. If it retrieves worse results, keep HNSW and use the compute unit only for inference.

---

## Experiment D: Sustained Quat8 Norm Drift Under Sleep Consolidation

**Why:** Phase 1 confirmed 2.0x norm conservation at 100M iterations for quaternions. But BMA's sleep consolidation doesn't just compose quaternions — it updates edge weights, merges nodes, and rewrites salience scores. Do Quat8 edges maintain stability through these destructive operations?

**Test:**
1. Build a 10K-node graph with Quat8 edges
2. Simulate 1,000 sleep cycles. Each cycle:
   - Select 10% of edges randomly, update their weights (simulating salience decay)
   - Merge 5 pairs of nodes (simulating compression functor L0→L1)
   - Add 20 new nodes with random edges (simulating new episodic content)
   - Delete 10 lowest-norm edges (simulating discard)
3. After each cycle: measure total graph norm, check for NaN/Inf, check for edge weight collapse (all edges converging to zero)
4. Run for the full 1,000 cycles

**What we learn:** Whether Quat8 edges are numerically stable over months of simulated sleep consolidation. If norm drifts beyond acceptable bounds after N cycles, that's the maintenance window — BMA needs to renormalize the graph every N sleep cycles. If stable to 1,000 cycles: effectively permanent.

---

## Experiment E: Memory Bandwidth Saturation on FX-8350 (DDR3)

**Why:** The benchmark showed the matmul inner loop is compute-bound at benchmark scale but memory-bandwidth-bound at production scale (400MB model weights). The graph traversal has a different memory access pattern (random vs sequential). What's the actual bandwidth utilization during Quat8 graph traversal?

**Test:**
1. Using the 100K-node / 500K-edge graph
2. Run 1,000 full spreading activation traversals
3. Measure: total bytes read from memory, total wall time
4. Calculate: effective memory bandwidth (bytes / time)
5. Compare against DDR3-1866 theoretical bandwidth (29.9 GB/s dual-channel, 14.9 GB/s single-channel)
6. If you can detect it: is the FX-8350 running single-channel or dual-channel?

**What we learn:** Whether graph traversal is bandwidth-limited on the Crawl hardware. If bandwidth utilization is low (<20%), the CPU cores are the bottleneck and Walk hardware (Zen, DDR5) won't help much. If utilization is high (>60%), Walk hardware will give a proportional speedup. This directly informs the Walk hardware purchasing decision.

---

## Experiment F: PrecisionWidth Field for BRIDGE TaskProfile

**Why:** The Walk evaluation recommends adding a PrecisionWidth field to BRIDGE's TaskProfile. We need concrete numbers for which operations need which width.

**Test:**
1. Run the following operations at QW8, QW16, and QW32:
   - Cosine similarity between embedding vectors (100 pairs)
   - Salience scoring (threshold comparison)
   - Contradiction detection (two nodes, check for semantic opposition)
   - Pattern matching (does a new observation match an existing pattern?)
2. For each operation at each width: measure accuracy relative to float64 reference
3. Define "sufficient accuracy" as: same binary decision (yes/no) as float64 for thresholded operations, rank correlation > 0.99 for similarity operations

**What we learn:** The exact precision requirements for each FATHOM and BMA operation. This becomes the lookup table that BRIDGE uses to select width when routing to the compute unit.

---

## Priority Order

If time is limited, run in this order:

1. **Experiment A** (QW8 precision for FATHOM) — directly gates Walk Step 4
2. **Experiment C** (spreading activation vs HNSW) — directly gates Walk Step 5
3. **Experiment D** (norm drift under sleep) — determines maintenance requirements
4. **Experiment F** (precision width table) — informs BRIDGE routing
5. **Experiment B** (octonionic vs scalar retrieval) — informs Walk Step 6
6. **Experiment E** (bandwidth saturation) — informs Walk hardware purchase

---

**These experiments use the codebase and graph infrastructure you've already loaded. Most are extensions of Phase 4's 100K-node graph. Results feed directly into BMA Spec Addendum 8.2 (the post-experiment revision).**
