# BMA Architecture Review: 16GB VRAM + PCIe 2.0 Constraints

**For: Gemini (architecture review)**
**From: James Paget Butler**
**Attached: BMA-Spec-Consolidated.docx, BMA-Spec-GPU-Storage-Update.docx, report-2026-04-06.json, BMA-Crawl-Environment.docx**

---

## Context

You have worked on BMA's theoretical foundations — the empathy synthesis,
the dual-pressure consolidation, the evolutionary framework for Theory
of Mind. This review asks you to apply the same rigor to the engineering
constraints.

BMA is a Go-based persistent AI memory system with three cognitive layers
(Autonomic, Subconscious bilateral L/R, Conscious bilateral A/B), a typed
hypergraph, biologically-grounded sleep consolidation, and inter-model
communication (BRIDGE). We are in the Crawl phase, running in a Podman
rootless container on Pop!_OS with enforced resource limits (20GB RAM,
6 CPU cores).

A hardware probe (attached: report-2026-04-06.json) has confirmed our
actual hardware diverges significantly from original spec assumptions:

| Parameter | Original Spec | Confirmed Reality |
|-----------|--------------|-------------------|
| GPU | RX 7900 XTX, 24GB VRAM | RX 9070 XT (gfx1201), 16.3GB VRAM |
| ROCm | Unspecified, HSA_OVERRIDE needed | ROCm 7.2.1, native gfx1201 support |
| PCIe | Assumed 3.0+ | PCIe 2.0 x16 (~8 GB/s) |
| CPU | Assumed modern | FX-8350 Piledriver, 8 cores, no PCIe atomics |
| RAM | 32GB assumed | 31.9GB confirmed, 21.8GB available |
| Disk | Unlimited assumed | 229GB total, 87GB free, SATA SSD (Samsung 840) |
| VRAM idle | Not measured | 1.5GB consumed by desktop/driver |

The attached GPU/Storage Update (BMA-Spec-GPU-Storage-Update.docx) proposes
specific revisions. The full spec (BMA-Spec-Consolidated.docx) provides the
complete component architecture through Addendum 6.0.

---

## Review Areas

### 1. The 16GB VRAM Budget

The proposed budget for single-context Crawl:

| Component | VRAM |
|-----------|------|
| Desktop/driver idle | ~1.5 GB (measured) |
| Model weights (7-8B Q4_K_M) | ~4-5 GB |
| KV cache (8K context, fp16) | ~4-6 GB |
| Embedding model | ~0.5-1 GB |
| Headroom | ~2-4 GB |

Questions for your review:
- Is this budget realistic? What margins would you recommend?
- The spec proposes deferring bilateral (two contexts) to Walk, gated
  on a TurboQuant viability flag from BMA-PROBE. At 3-bit KV compression,
  two 8K contexts would need ~0.7-1.4 GB instead of ~8-12 GB. Is
  TurboQuant a sound architectural dependency, or should we design a
  "Unilateral-Active / Compressed-Shadow" alternative that doesn't
  depend on KV compression maturity on ROCm?
- The embedding model shares the GPU with inference. Should they share
  a single model (inference model generates embeddings) or run separate
  small embedding models? The tradeoff is VRAM versus quality.

### 2. PCIe 2.0 Transfer Discipline

At ~8 GB/s, transferring a full 8K KV cache (4-6 GB at fp16) takes
500-750ms. The spec targets <100ms for phase switching (context A/B
handoff).

Questions for your review:
- Is the <100ms target achievable on PCIe 2.0 via incremental transfer
  (delta encoding of KV cache changes) rather than full transfer?
- The context management system uses a "cognitive cache controller"
  pattern: CPU prefetches predicted content into RAM, then transfers
  to GPU in a single batch. On PCIe 2.0, is batching sufficient or
  do we need a streaming pipeline that overlaps transfer with computation?
- For sleep consolidation, batch embedding operations move content
  between CPU and GPU. What batch size minimizes PCIe round-trip
  overhead while staying within the VRAM headroom budget?

### 3. Autonomic Setpoints

The revised VRAM setpoints are Target 70%, Warn 80%, Critical 90%.
With 1.5GB already consumed at idle, the effective VRAM available to
BMA is ~14.7GB. At 70% target, BMA aims for ~10.3GB usage — leaving
~4.4GB as buffer.

Questions for your review:
- With model weights (~5GB) + KV cache (~5GB) = ~10GB baseline,
  the 70% target means BMA is already AT the target before any
  embedding or background GPU work. Should the target be even lower
  (65%), or should the setpoint be calculated against available VRAM
  (total minus idle) rather than total VRAM?
- The 10Hz polling rate for AUTO-S: is this fast enough to catch
  VRAM spikes from batch embedding operations? Or should GPU memory
  allocation events trigger interrupt-style notifications rather
  than polling?

### 4. Storage Survival on 87GB Free

The proposed escalation: Warn at 75% used (~57GB used, ~172GB),
Critical at 85%, Emergency at 95%.

Questions for your review:
- 87GB free out of 229GB means we're already at 62% used. The Warn
  threshold triggers at 75% — that's only ~30GB of BMA growth before
  the first warning. Is 30GB enough runway, or should the thresholds
  be adjusted for a partition that's already 62% full?
- The Samsung 840 has limited write endurance. Sleep consolidation
  does significant writes (inner ring processes every session's
  content, middle ring does cross-session comparisons, outer ring
  does random walks). Should sleep consolidation have a
  write-budget-per-cycle limit tied to SMART wear monitoring?
- The primary defense under storage pressure is BRIDGE transcript
  compression. Is this sufficient, or should aggressive N3 discard
  (removing low-salience content permanently rather than just
  compressing it) be the primary defense?

### 5. GPU Contention with External Workloads

James uses this machine for gaming (Beyond All Reason) and desktop
work. The GPU is shared via OS scheduling, not partitioned.

Questions for your review:
- The spec proposes three GPU modes: Inference, Idle, External.
  When James is gaming, BMA should defer all GPU operations. Is
  detection via GPU utilization polling (10Hz) sufficient, or should
  BMA register for GPU memory pressure notifications from the driver?
- During extended gaming sessions (hours), BMA accumulates unprocessed
  session content. When the GPU becomes available, should BMA
  prioritize inference readiness (warm the KV cache) or sleep
  consolidation (process accumulated content)? The biological analog
  would suggest consolidation first — you sleep when the threat
  passes, not when it arrives.

---

## Deliverables Requested

1. **Risk Assessment**: Ranked list of architectural risks specific to
   the 16GB/PCIe 2.0/87GB baseline. Which risks are acceptable for
   Crawl, and which require design changes before proceeding?

2. **Specification Refinements**: Concrete changes to the proposed
   setpoints, budgets, or protocols based on your analysis.

3. **Critical Questions**: 3-5 questions that must be answered
   empirically (by running code on this hardware) before the next
   implementation sprint.

4. **Dual-Pressure Relevance**: You designed the dual-pressure
   consolidation model (selfish-herd + inclusive-fitness). How does
   the storage constraint interact with dual-pressure? When disk space
   is scarce, does the system become more "selfish" (aggressive
   discard of collaborative content to preserve self-operation) or
   should collaborative content be protected even at the cost of
   system performance?

---

**Read the attached documents before responding. The spec is the
authority. The probe report is the ground truth. Your review should
be grounded in both.**
