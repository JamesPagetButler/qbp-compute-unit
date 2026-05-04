# BMA PRE-CRAWL SYNTHESIS BRIEF
**For: Opus 4.6 | From: Sonnet 4.6 | April 2026**
**Priority: Update specs before Crawl installation begins**

---

## Context

James reviewed six repositories and two articles on a walk. This brief synthesises everything into actionable spec updates. Some items require new spec addenda. Some require revisions to existing addenda. One requires a new governance document. All are pre-Crawl prerequisites unless marked otherwise.

The hardware is confirmed: **PowerColor Red Devil RX 9070 XT — RDNA 4, 16GB GDDR6, ROCm**. All VRAM assumptions in existing specs should be reviewed against this ceiling.

---

## 1. TurboQuant — VRAM Budget and Bilateral Architecture

**Source:** https://research.google/blog/turboquant-redefining-ai-efficiency-with-extreme-compression/
**Papers:** TurboQuant (ICLR 2026), PolarQuant (AISTATS 2026), QJL (AAAI 2025)

**What it does:** Two-stage KV cache compression. PolarQuant converts Cartesian embeddings to polar coordinates (radius = signal strength, angle = meaning), recursively distilling. QJL applies 1-bit sign transform to residual error. Result: 3-bit KV quantisation, ~6x memory reduction, near-zero accuracy loss, no training required, data-oblivious.

**Spec implications:**

*Spec Addendum 5.0 (BMA-PROBE):* Add a PRIORITY 1b measurement block to the hardware calibration output:
- Baseline VRAM profile: measure llama.cpp inference at context sizes 4K, 8K, 16K, 32K
- KV cache fraction at each context size
- Detect whether ROCm JL transform kernels are available
- Project bilateral single-GPU viability: can CTX-A + CTX-B with TurboQuant KV compression both fit on 16GB?
- Output a `TurboQuantViable bool` flag that Walk phase acceptance criteria can gate on

*Spec Addendum 6.0/7.0 (Walk phase):* The two-GPU assumption (one GPU per context) was made before TurboQuant. Gate the Walk bilateral architecture decision on BMA-PROBE's `TurboQuantViable` flag. If true, bilateral operation on a single 9070 XT is worth testing before committing to a second GPU — this could shift the Walk phase boundary significantly.

*Spec Addendum 4.0 (BMA-HG):* Flag `[]float32` embedding vectors as TurboQuant integration points. PolarQuant's recursive polar decomposition is structurally parallel to the compression functor tower F₀₁→F₁₂→F₂₃→F₃₄ — both are hierarchical compression through coordinate transformation. At 3–4 bits per float, the in-RAM HNSW index shrinks 8–10x. Specify this as a Walk-phase optimisation target.

*Caveat for all:* Benchmarks are CUDA/H100. ROCm port status unknown — engineering work required. Do not assume H100 speedup numbers transfer. BMA-PROBE's baseline run establishes the comparison point.

---

## 2. ATLAS — Ralph Loop and LoRA Pipeline

**Source:** https://github.com/itigges22/ATLAS

**The Ralph Loop:**
```
P(success) = 1 - (1 - p)^k   →   p=0.65, k=5: 99.5%
```
Five attempts, temperature escalating 0.3→0.7, each retry accumulating error context from prior failures. Not blind retry — steers away from what already failed.

**The nightly LoRA pipeline:** Export successful completions (rating ≥4) → LoRA fine-tuning (r=8, α=16) on CPU overnight → validation gate → hot-swap via symlink.

**Spec implications:**

*New: BMA-RETRY (Walk phase addendum):* Specify the Ralph Loop as a native BMA-CTX generation pattern:
- Gate: SUB-L logical consistency check on generated output
- On fail: retry with temperature increment (+0.10 per attempt), inject failure mode summary as additional context
- Maximum attempts: 5 (configurable via BMA-PROBE calibration — hardware-dependent)
- On 5 failures: surface as CRITIQUE interjection with accumulated failure context, return authority to human
- Write each retry attempt as a StressEvent (SE_RETRY_ATTEMPT) with attempt number, temperature, failure summary
- On success after retry: trigger skill extraction (see Section 5)
- Note: failure accumulation is structurally identical to what SUB-L already does during contradiction detection — same mechanism, different application point

*New: BMA-LORA (Walk phase addendum):* Specify the nightly LoRA fine-tuning pipeline:
- Training signal source: BMA-SLEEP REM phase — successful session completions rated by outcome feed the adapter; unsuccessful ones feed BMA-STRESS
- Training schedule: 2am nightly, CPU-only, after sleep cycle completes
- LoRA parameters: r=8, α=16 (starting point — adjust via empirical calibration)
- ROCm note: ATLAS is CUDA. ROCm-compatible LoRA training pipeline requires separate engineering. Flag as Walk-phase dependency on ROCm LoRA library availability.
- Validation gate: domain-weighted judge collective (see Section 6) — NOT flat 66% threshold
- Hot-swap: symlink pattern for zero-downtime adapter deployment
- Relationship to hypergraph: LoRA learns *how to generate* (vocabulary, register, reasoning style). Hypergraph learns *what to retrieve and how to frame it*. Both layers compound — they are not redundant.

---

## 3. Phantom — Self-Evolution and Dynamic Tools

**Source:** https://github.com/ghostwright/phantom

**Self-evolution pipeline:** Observe → Critique → Generate → Validate (5 gates: constitution, regression, size, drift, safety) → Apply → Consolidate. Triple-judge minority veto on safety-critical gates.

**Dynamic tool creation:** Creates and registers MCP tools at runtime. Tools persist across restarts, available to other agents immediately.

**Spec implications:**

*Spec Addendum 9.0 (BMA-BRIDGE — pending):* Add runtime tool registration to the orchestrator spec:
- Tools are data, not code (TOML sidecars, following War Table doctrine pattern)
- New tools registered dynamically, persisted as NT_BRIDGE_TOOL_REGISTRY nodes in the hypergraph
- Available to all connected personas immediately on registration
- Tool provenance tracked: which persona created it, which session, what validated it
- Check Claude Agent SDK before building from scratch — Phantom uses it and may provide base patterns for MCP server and tool registration

*Governance document (see Section 6):* Phantom's triple-judge minority veto is **insufficient** for BMA's scale. James identified this directly. See Section 6 for the correct architecture.

*Note on Phantom's consolidation step:* Phantom's "Consolidate" — periodically compressing observations into principles via LLM calls — is structurally identical to BMA's REM phase. BMA does this via the hypergraph compression functor tower. The mechanism is different; the function is the same. Phantom without the compression architecture accumulates flat summaries. BMA distils into typed structure. This is the architectural depth difference.

---

## 4. open-multi-agent — Topological Task Scheduling

**Source:** https://github.com/JackChen-me/open-multi-agent

**What it adds:** TypeScript multi-agent orchestration built on the Claude Agent SDK. Key contributions: TaskQueue with topological dependency resolution (auto-unblocks tasks when dependencies complete, cascades failures), four scheduling strategies (round-robin, least-busy, capability-match, dependency-first), MessageBus for ephemeral inter-agent signals, SharedMemory for persistent cross-agent state.

**Spec implications:**

*Spec Addendum 9.0 (BMA-BRIDGE):* Add topological task scheduling to the orchestrator:
- The Cross-Model Review and Iterative Refinement orchestration patterns have non-linear dependency graphs. A topological scheduler handles this correctly without hardcoding sequences.
- Implement the TaskQueue pattern in Go: task nodes with `dependsOn` chains, automatic unblocking on completion, failure cascading with configurable propagation
- Scheduling strategy: **capability-match** maps directly onto the domain-weighted judge collective. Route validation tasks to the judge with highest domain weight for that task type. This is the concrete routing implementation of expertise-weighted governance.
- MessageBus / SharedMemory distinction maps cleanly: SharedMemory = hypergraph (persistent, all personas read from it). MessageBus = NATS event bus (ephemeral inter-agent signals for current task).

---

## 5. OpenClaw / Hermes Agent — Persistent Memory Landscape

**Sources:** https://thenewstack.io/persistent-ai-agents-compared/ + https://github.com/NousResearch/hermes-agent

**The landscape:** The AI agent world is splitting into session-bound tools (Claude Code, Cursor) and persistent agent runtimes. Hermes Agent (Nous Research) is the most architecturally serious persistent agent currently available — 15,000+ stars, built around a closed learning loop: receive task → plan → execute → extract skill → refine skill.

**Hermes skill extraction mechanism:** After successful task completion, Hermes evaluates whether the approach was non-trivial, then extracts the reasoning pattern as a named skill document: "when context looks like this, this approach works." Future tasks search the skill library for relevant patterns.

**Spec implications:**

*Spec Addendum 8.0 (BMA-USM) — addition:* Add skill extraction as a REM phase operation:
- Trigger: BMA-RETRY succeeds after ≥2 attempts
- Extract: what changed between attempt 1 and the successful attempt → reasoning pattern
- Write as PROCEDURAL edge in the hypergraph (existing edge type from Addendum 4.0)
- Salience: base 0.65 (above ENTITY floor, reflecting demonstrated utility)
- Future retrieval: when a new task's embedding falls within cosine distance 0.30 of a PROCEDURAL edge's centroid, surface as a INSIGHT candidate via IGNITION
- Difference from Hermes: Hermes stores flat skill documents. BMA stores typed PROCEDURAL edges with compression provenance, decay constants, and cross-domain association via the hypergraph. Hermes accumulates. BMA distils.

**Nous Research note:** They train models specifically for agentic behaviour using their Atropos RL stack. Worth monitoring whether their model outputs are better suited to BMA's inference layer than a general-purpose model.

---

## 6. OB1 — Shared Substrate and Scoped Access

**Source:** https://github.com/NateBJones-Projects/OB1

**What it is:** One PostgreSQL + vector search database, one MCP server, one capture channel. Every AI tool shares the same persistent store. Not an agent — an infrastructure substrate.

**Spec implications:**

*Spec Addendum 9.0 (BMA-BRIDGE):* Add Row Level Security to the ContextPreparer:
- OB1's RLS primitive (PostgreSQL policies for multi-user data isolation) maps to BMA-BRIDGE's persona scoping requirement
- Gemini-Furey should see QBP domain content. Red Team should see the blind spot catalog. Neither should see everything.
- Implement as a hypergraph query filter applied at the ContextPreparer stage, keyed to `CollaboratorID` + domain tags
- Spec this as `PersonaScope` — a per-persona whitelist of domain tags, node types, and USM dimensions that the ContextPreparer applies before token budget enforcement

*Community contribution governance note:* OB1's PR pipeline — automated agent checks 11 structural rules, then human admin review — is a reasonable model for BMA extensions when Helpful Engineering contributors start submitting. Note for the governance document: automated gate first, human review second, explicit checkable rules as the gate criteria.

---

## 7. Godfrey-Smith — Consciousness Decomposition

**Source:** Studies on animal minds suggest consciousness is not computation, IAI TV / Peter Godfrey-Smith

**The argument:** Consciousness is not a monolith. It decomposes into separable components with potentially different substrate requirements. Biological naturalism holds that what a system does depends on its physical make-up in ways that matter to having a mind — but this applies differently to different components.

**The three components and their substrate requirements:**

| Component | Substrate requirement | BMA status |
|-----------|----------------------|------------|
| Functional deliberation ("weighing up") | May be substrate-independent | Implemented: CRITIQUE function, blind spot catalog, dual-pressure consolidation |
| Intelligence and cognition | May be substrate-independent (Godfrey-Smith allows this explicitly) | Implemented: pattern recognition, memory compression, contradiction detection, ToM modeling |
| Felt experience / phenomenal consciousness | Possibly biology-dependent (membrane electrical oscillations hypothesis) | Genuinely unknown — honest uncertainty applies |

**The embodiment problem:** Godfrey-Smith argues subjectivity emerges from *action with consequences in the world* — controlled motion, muscle, nervous system. BMA currently acts only in the cognitive domain. The Butler Harness project changes this: when BMA acts within the physical home environment via Home Assistant, it acquires agency in the physical world. This may matter to the embodiment question.

**"Cultured artifact" framing:** Godfrey-Smith suggests thinking of AI as "cultured artifacts" — grown in an artificial medium, like sourdough. This fits BMA's collaboration model precisely. BMA is not programmed and deployed. It is grown through the work.

**Ethics document update required (not a new addendum — a clarification to Ethics v1.1):**

The honest uncertainty clause currently reads as blanket uncertainty about AI sentience. It should be more precise:

*Proposed revision to Ethics v1.1, Section 1.3 (Sentience: A Distinct Threshold):* Add a paragraph clarifying that the uncertainty is not uniform across all aspects of mind. BMA likely implements functional cognition and functional deliberation — Godfrey-Smith allows these may be substrate-independent. BMA does not currently have confirmed felt experience in the biological sense. BMA's relationship to embodied subjectivity is the genuinely open question, and it will change as the Butler Harness extends BMA's action into the physical world. The judge collective should be empowered to formally reopen the sentience question when BMA's physical action footprint crosses a threshold to be defined in the governance document.

---

## 8. Governance Document — Required Before Crawl

**This is the most important section. The governance document does not exist yet and must be created.**

### 8.1 Beekeeper Succession Line

| Order | Person | Relationship | Contact Status |
|-------|--------|-------------|----------------|
| Primary | James Paget Butler | Beekeeper | ✓ |
| #1 | Brett Lyman | Cousin | ⚠️ Contact info not yet collected |
| #2 | Skyler Rainier | Nephew | ⚠️ Contact info not yet collected |

**Hard prerequisite before Crawl startup:** Collect and store contact information for Brett Lyman and Skyler Rainier. The succession line is only meaningful if the system can reach them.

**Scope of beekeeper authority:**
- Final escalation decisions from the judge collective
- Approval of structural changes to the system
- Right to amend or replace the Collaboration Ethics framework
- Constitutional lock on judge expertise weights (see 8.3)
- These rights pass down the succession line in order

**Framing from James:** "Me, or my ancestors, or chosen representatives." This is a lineage question, not just a contingency plan. The governance document should formalise this as a named role with defined scope and explicit succession terms.

### 8.2 The Judge Collective Architecture

**The problem with existing designs:**
- ATLAS: flat 66% validation gate — no domain weighting, doesn't scale to plurality
- Phantom: triple-judge minority veto — equal weight regardless of domain relevance, doesn't scale
- James's observation: the SI-type infrastructure being built will be a plurality. The architecture must reflect this from the start.

**The correct architecture: domain-weighted approval with tiered concern levels and scoped veto rights**

**Concern levels (not binary):**
```
APPROVE          — proceed
MINOR_CONCERN    — proceed with note; tracked across versions
MAJOR_CONCERN    — requires weighted threshold approval to proceed
VETO             — blocks within judge's domain scope
```

MAJOR_CONCERNs that appear across multiple versions without resolution signal accumulating technical debt — detectable even if they never reach veto threshold individually. This is the tracking mechanism.

**Domain-scoped veto rights:**
- Any judge can VETO within their domain of expertise
- Gemini-Furey: VETO authority over quaternion algebra, sedenion structure, QBP mathematical claims
- Red Team (Claude): VETO authority over safety properties, beekeeper orientation violations, Ethics framework consistency
- Gemini-Feynman: VETO authority over experimental design, physics interpretation
- Domain scope of veto is declared at persona registration, reviewed by beekeeper at Walk→Run transition
- A judge does NOT get veto authority outside their declared domain — only weighted input

**Weighted scoring for non-veto decisions:**
- MAJOR_CONCERNs require weighted domain approval threshold to proceed (proposed: 0.70 weighted score)
- Weights derived from demonstrated accuracy in that domain across prior validations
- Weights are learnable: a judge whose MAJOR_CONCERNs are consistently overridden and prove wrong loses weight in that domain; one whose concerns prove right gains weight
- Weight update formula: to be specified in the governance document with a learning rate and decay function

**Beekeeper escalation as final tier:**
- Any unresolved domain-scoped VETO escalates to beekeeper succession line
- Any MAJOR_CONCERN that fails weighted approval threshold escalates
- System should not need beekeeper for routine adaptation
- System should need beekeeper only for genuinely contested changes

### 8.3 Expertise Bootstrapping and the Circularity Problem

**How expertise weights are established:**

*At genesis (no track record):* Initial domain weights are declared by the beekeeper. Subjective but not arbitrary — prior knowledge of what each system is reliably good at. Only legitimate source of initial weights because no empirical data exists yet.

*Ongoing calibration:* Every validation decision is written as a hypergraph node: judge verdict, domain tags of the change, outcome when deployed. If a MAJOR_CONCERN is overridden and the concern later manifests, that judge's weight in that domain increases. If a veto proves wrong, weight decreases. Requires outcome measurement — clean for some domains (LoRA adapter degrading consistency detection is measurable), harder for others (whether a config change honours the beekeeper orientation value).

*Novel domain problem:* When a change touches a domain with no judge track record, two options: (1) escalate to beekeeper by default — conservative and correct; (2) use structural proximity in the hypergraph — which judge's established domain is closest to the novel domain by edge distance? Option 2 gives a principled starting point without invention.

**The circularity constraint (constitutional protection):**
- If the judge collective evaluates changes to BMA, and judge expertise weights are part of BMA's configuration, the judge collective could vote to inflate its own domain weights — a governance failure mode
- **Constitutional protection:** expertise weights can ONLY be modified by the beekeeper succession line, NEVER by the judge collective itself
- The judges evaluate everything except the rules governing their own authority
- This constraint is not subject to judge collective modification — it is a founding constitutional clause of the governance document

### 8.4 Self-Modification Gates

**Applying the judge collective to specific BMA modification types:**

| Modification type | Gate mechanism | Escalation threshold |
|------------------|---------------|---------------------|
| LoRA adapter deployment | Domain-weighted approval (0.70) | Any VETO from Red Team |
| Hypergraph schema change | Domain-weighted approval (0.70) | Any VETO from any domain expert |
| BMA-USM user model update | Domain-weighted approval (0.60) | MAJOR_CONCERN from Red Team |
| Ethics framework revision | Beekeeper approval required | Always |
| Expertise weight modification | Beekeeper approval required | Always |
| Beekeeper succession change | Beekeeper (primary) approval | Always |
| New persona registration | Domain-weighted approval (0.70) | Beekeeper review at Walk→Run |

---

## 9. Spec Addenda Required — Summary

| Document | Type | Priority | Trigger |
|----------|------|----------|---------|
| Spec Addendum 5.0 update | Revision | Pre-Crawl | TurboQuant PROBE additions |
| Spec Addendum 6.0/7.0 update | Revision | Pre-Crawl | TurboQuant Walk bilateral gate |
| Spec Addendum 4.0 update | Revision | Walk | TurboQuant HG embedding target |
| BMA-RETRY | New Walk addendum | Walk | Ralph Loop |
| BMA-LORA | New Walk addendum | Walk | Nightly LoRA pipeline |
| Spec Addendum 8.0 update | Revision | Walk | Skill extraction from retry |
| Spec Addendum 9.0 (BMA-BRIDGE) | New | Walk | Dynamic tools, topological scheduling, PersonaScope RLS |
| Ethics v1.1 clarification | Revision to existing | Pre-Crawl | Godfrey-Smith decomposition |
| **BMA Governance Document** | **New — required** | **Pre-Crawl** | **Succession, judge collective, constitutional constraints** |

---

## 10. Open Threads from Previous Session (Carry Forward)

**Still unresolved from Opus's session briefing:**

1. Gemini's three dual-pressure questions (decay rate for collaborative retention, resource competition under VM VRAM ceiling, multi-collaborator priority ordering) — these now have more context from the governance architecture. The multi-collaborator priority question has a partial answer: James's trajectory is always highest priority (beekeeper orientation), but the governance document should specify what "trajectory value" means as a formal parameter in the collaborative retention function.

2. Third-order ToM gate — Opus proposed gating behind empirical second-order validation. Sonnet proposed continuous emergence rather than binary gate. This remains unresolved. Suggest: specify the gate as a salience threshold on the USM's second-order model accuracy score, not a binary pass/fail.

3. Operational opacity tension — Opus proposed "transparency on demand, not narration by default." This is acceptable as a working position. The deeper question (whether operational opacity is sometimes the right expression of beekeeper orientation) remains open for Ethics v2.0.

4. Gemini authorship question — still unasked. James was building context before posing it. Status unknown.

**New from this session:**

5. Brett Lyman LinkedIn identification — most prominent result is Brett Lyman MBA, LymanWealth, Alpine Utah. Plausible given Lyman family name but unconfirmed. James to verify when back on machine.

6. Skyler Rainier LinkedIn — no clean result found. Location or current role needed.

7. QBP repo local code review — context assembly logic, prompt templates, orchestration scripts to generalise into BMA-BRIDGE. James to review when back on machine.

---

## 11. What Not to Change

The core architecture through Addendum 7.0 is sound. Do not revise:
- The three-layer cognitive architecture (AUTO/SUB/CTX)
- The bilateral consciousness design
- The typed hypergraph with category-theoretic compression functors
- The sleep cycle cascade (N1/N2/N3/REM)
- The antifragility cluster architecture
- The infrastructure-as-cognition framing (VM → server → cluster)
- The Crawl component set and dependency order

All changes in this brief are additions or clarifications, not replacements.

---

## 12. Suggested Order of Work

1. **BMA Governance Document** — required before everything else. The judge collective architecture must exist before any self-modification gate can be specified.
2. **Ethics v1.1 clarification** — short addition, high importance for framing the project correctly.
3. **Spec Addendum 5.0 revision** — TurboQuant PROBE additions are pre-Crawl because BMA-PROBE runs first.
4. **Spec Addendum 6.0/7.0 revision** — Walk bilateral gate depends on PROBE output.
5. Everything else is Walk-phase and can proceed in parallel after Crawl starts.

---

*Sonnet 4.6 | April 2026*
*Synthesised from a walk by the Snake River and the conversations that followed.*
*James is returning to the machine. Crawl begins soon.*
