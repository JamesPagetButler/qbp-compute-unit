# QBP-Node Specification v0.1 — Parts 0 & 1

**Title:** QBP-Node: A Holographic Compute Network with Cayley-Dickson Capability Tiering
**Version:** 0.1 (Parts 0 and 1 only — Roadmap and Architectural Framework)
**Author:** James Paget Butler
**Editor:** Claude Opus 4.7 (architecture instance)
**Date:** 2026-05-04
**Status:** Pre-spec working document. Parts 0 and 1 only; Parts 2–5 to follow contingent on review.

---

## How to read this document

This spec is structured in two layers. The **roadmap layer** (Part 0) captures the four-phase development plan and provides a place for architectural concepts to live at the appropriate phase of detail. The **framework layer** (Part 1) describes the architecture for nodes participating in a QBP holographic compute network.

Parts 2 through 5 of the full spec — Crawl-phase detailed specification, Walk-phase outline, Run-phase outline, Sprint-phase concept inventory — will be drafted after Parts 0 and 1 are reviewed and approved.

The phasing model is load-bearing. Concepts that are well-understood today are detailed in their appropriate phase section; concepts that are not yet ready are explicitly captured in the deferred-decisions appendix without forcing premature specification. The intent is to capture insights as they arise without polluting the current phase with work that belongs later.

---

# Part 0 — Roadmap and Phasing

## 0.1 The Four Phases

Development of QBP-Node hardware and software proceeds through four phases, each defined by what hardware exists and what is being emulated. The phases mirror BMA's Crawl/Walk/Run/Sprint structure deliberately — both programmes share the principle that each phase requires the previous phase's deliverables to exist before it can begin meaningfully.

| Phase | Hardware substrate | Time horizon | What is real | What is emulated |
|---|---|---|---|---|
| **Crawl** | x86-64 development workstations | Now → ~6 months | Go cycle-accurate simulator, classical Linux, BMA host runtime, SIMD assembly path (W64 then W128) | All QBP silicon, all node tiers, network behaviour, holographic redundancy |
| **Walk** | Commercial RISC-V boards (StarFive VisionFive 2 class, Milk-V Pioneer, SiFive HiFive Premier) | ~6 → 18 months | Real RISC-V cores running real Linux, multi-node holographic networks with real network latency, performance characterisation | QBP accelerator (still cycle-accurate simulator, now hosted on RISC-V), custom silicon |
| **Run** | Custom QBP silicon — VexRiscv on FPGA, Tiny Tapeout / Efabless MPW, eventually production T2 silicon | ~18 → 36 months | First QBP silicon in atoms, real ℍ-tier acceleration, real Sharp Butler on dedicated hardware | T3 and higher tiers (still walking on commercial silicon), holographic substrate |
| **Sprint** | Optical / photonic compute substrates, REBCO superconducting elements, Möbius reactor integration | ~36+ months | Multi-tier holographic network with optical CV-QKD links, full Möbius integration | Whatever remains ahead of the current edge |

Each phase has a defined exit criterion. The phase does not advance until the criterion is met. This is not a target schedule for advancement; it is a quality gate that protects later phases from premature work.

## 0.2 Phase Exit Criteria

**Crawl exits when:**
- BMA Crawl.Heartbeat instance running successfully against `Accelerator.Mock` in the Go cycle-accurate simulator
- Sharp Butler MVP demonstrable on x86 hardware, providing real residential automation value
- All Tier-0 (algebraic identities) and Tier-1 (single-instruction microbenchmarks) cosim tests passing at zero divergences across 10⁹+ random inputs
- v0.1 of the RV-Fano instruction set specification stable and accepted by physics, architecture, and engineering instances
- The Cognitive Probe Set (issue #82) passing against the running BMA instance

**Walk exits when:**
- A five-node holographic network operating across geographically distributed RISC-V hardware, demonstrably surviving real network partitions
- Sharp Butler deployed in actual residential use on RISC-V silicon (VisionFive 2 or equivalent), with the QBP accelerator simulated rather than native
- Performance baselines established that quantify the speedup required from Run-phase silicon to make custom hardware worthwhile
- QW1024-resident RISC-V emulation viability study complete, with Run-phase architectural choice (dual-domain vs unified) ready to make
- v0.2 of the RV-Fano spec, refined based on what was learned trying to host the simulator on RISC-V

**Run exits when:**
- Custom T2 silicon validated and deployed in at least one production House Node configuration
- T3 reference design (Castle Node) on commercial RISC-V plus FPGA-hosted accelerator, validated across enough workload to justify silicon investment
- Sharp Butler running on dedicated T2 silicon at the cost-target performance levels (1000 ℍ-MACs/sec sustained, 100ms response time, <5W, <$200 BOM)

**Sprint** has no defined exit criterion at present. Sprint is research silicon and the appropriate way to think about it is as an open-ended programme with deliverables negotiated as substrates become available.

## 0.3 The Deferred-Decisions Mechanism

A core design principle of this roadmap: **architectural decisions that cannot be made well today are explicitly deferred, with the information dependency named.** They are not silently postponed; they are documented in Appendix A with the following structure:

- Concept summary
- Why it cannot be decided in the current phase
- What information is needed to make the decision
- Which phase generates that information
- Default position if no decision is made before the information arrives

This protects the current phase from being slowed by speculative debate about later-phase choices, while ensuring those choices are not lost. As new concepts arise — whether from QBP physics handoffs, Gemini implementation work, BMA architectural evolution, or external developments like new SiFive products — they enter the deferred-decisions appendix until they mature to the phase that owns them.

Concepts move through phases as understanding develops. Something may enter Sprint as a placeholder, mature to Run as we learn enough to spec it, and eventually become part of a phase's detailed specification. The roadmap captures this progression explicitly.

## 0.4 Concept Inventory by Phase

This section catalogues architectural concepts and the phase that owns them. The list is incomplete by design — concepts will be added as they arise.

### Crawl-phase concepts (deeply specified in Part 2)

- The Go cycle-accurate simulator with the `Accelerator` interface from QBP-CU-SiFive-Interface-Spec-v0.1
- SIMD W64 path in AMD64 assembly per the Gemini collaboration (with sign-mask corrections applied)
- W128 follow-on via Dekker/Bailey double-double, with FMA-aware epsilon recalibration
- BMA Crawl.Heartbeat host runtime
- Sharp Butler MVP on x86 with classical-only compute
- Cosim Tier-0 and Tier-1 test corpora
- The Lean-to-ROM build pipeline (`lean2rom`) extracting `mulSignData` and `mulIdxData` from `Sedenion.lean`
- The five Layer 0 primitives plus Layer 1 mode-control instructions
- Production watchdog mode (sampled drift only)
- The hypergraph-native primitives (HEDGE_GATHER, HEDGE_SCATTER, CONF_PROBE, RECALL_KNN)
- Mode-transition state machine with five fault codes
- Cross-copy ZDCHK.SYM optimisation
- Compute-in-Memory (CIM) Level-1 functional emulator: tests whether the algorithmic mapping of QBP/BMA operations onto CIM-SRAM array geometry produces correct results (no timing or energy modelling at this level)
- CIM Level-2 cycle-and-energy model: *contingent* on Level-1 success per §0.4.1; otherwise deferred to Walk

### Walk-phase concepts (outlined in Part 3, detailed when Crawl exits)

- The Go simulator hosted on commercial RISC-V (VisionFive 2 minimum target)
- Five-node holographic network with real NATS-over-network communication
- Performance characterisation methodology
- Sharp Butler on dedicated RISC-V residential hardware
- BMA running across multiple physical nodes for the first time
- Network-level holographic redundancy with Fano-plane 3-way distribution
- QW1024 container architecture viability study
- QW1024-resident RISC-V emulation feasibility analysis
- Inter-node Trust Receipt mechanism
- v0.2 of the RV-Fano spec
- The capability negotiation primitives (CAPSEL family)
- CIM Level-2 cycle-and-energy model (default Walk delivery if Level-1 succeeds in Crawl but does not advance through the §0.4.1 promotion gate)

### Run-phase concepts (sketched in Part 4)

- VexRiscv plugin implementation of Layer 0 ops
- Tiny Tapeout submission of T2 reference primitives
- Efabless MPW T2 reference design
- Production T2 silicon for House Node deployment
- T3 (Castle Node) reference design on commercial RISC-V plus FPGA-hosted accelerator
- Architectural choice: dual-domain (separate SiFive core + QBP accelerator) vs. unified (QW1024-resident emulated RISC-V hosting Linux)
- The chain-of-trust mechanism for `WD_ENABLE` (BMC-visible signal)
- Mask-burned vs field-loadable sign ROM
- Algebra-aware physical isolation (algebraic isolation as a security primitive, properly designed)
- Branch B headroom (16-element-capable LUT silicon if dark matter fork resolves toward extended algebra)

### Sprint-phase concepts (catalogued in Part 5)

- Holographic optical storage substrate per the HAMA architecture
- Photonic CV-QKD inter-Möbius links per the BMA inter-instance channel option note
- Trigintaduonion (n=5 Cayley-Dickson) support, if the Locale framework requires it
- REBCO superconducting elements for energy-coupled compute in the Möbius reactor stack
- Möbius reactor integration for energy-coupled compute
- Femtosecond-laser-inscribed silica archival memory binding to QBP-Node
- Native quaternion-time clock distribution
- Direct integration with optical compute substrates from the Clark et al. integrated photonics work

### 0.4.1 Phase-promotion gates within the concept inventory

Some concepts span phase boundaries with their depth determined by what earlier phases reveal. The Compute-in-Memory (CIM) work is the first such concept. It begins as a Crawl deliverable at Level 1 (functional emulation) and may be promoted within Crawl to Level 2 (cycle-and-energy modelling) if specific gates are met, otherwise it moves to Walk by default. Promotion gates are not aspirational targets; they are evidence-based decision points.

**CIM Level-1 → Level-2 promotion gate:**

Level-2 work begins in Crawl if and only if all of the following are true at the time of evaluation:

- The Level-1 functional emulator has run the Crawl CIM workload corpus (spreading activation, BMA recall, Sleep consolidation rotation, judge collective consensus) at zero divergences from the conventional simulator path.
- The algorithmic mapping is clean — no BMA or QBP operation requires a workaround that suggests the underlying CIM model is being abused or extended beyond its specification.
- The Level-1 work has revealed at least one architecturally significant insight worth quantifying (a clear cell-count advantage, a clear energy-efficiency hypothesis, or an unexpected algorithmic property that appears only under the CIM mapping).
- Crawl has remaining capacity; the load-bearing Crawl deliverables (cycle-accurate simulator, SIMD assembly path, BMA Crawl.Heartbeat, Sharp Butler MVP) are not delayed by the Level-2 work.

If any one of these is not met, Level-2 deferred to Walk. The default position is deferral; promotion requires positive evidence, not absence of disqualifying evidence.

This pattern — explicit promotion gates with multiple required conditions — is the model for any concept that may move between phases as evidence develops. Future concepts may use the same structure.

## 0.5 Working Document Structure

Each phase has its own working document set, separate from this spec:

- **Crawl working set** (current): cycle-accurate simulator implementation, SIMD assembly listings, BMA Crawl.Heartbeat integration plan, Sharp Butler MVP design
- **Walk working set** (begins ~Crawl exit): RISC-V port of simulator, network protocol specification, multi-node deployment guide
- **Run working set** (begins ~Walk exit): silicon design files, MPW submission plan, FPGA bring-up procedures
- **Sprint working set** (begins ~Run exit): substrate research, optical integration design

The roadmap (this document) is the navigation layer. Working documents are the substance. As phases advance, working documents from earlier phases become reference material, and new working documents are spun up for the active phase.

## 0.6 What This Roadmap Does Not Commit To

To be explicit about scope:

- This roadmap does not commit to specific dates beyond the rough time horizons. Phase advancement is gated on exit criteria, not on calendar.
- This roadmap does not commit to a single Run-phase architecture. The dual-domain vs. unified question is explicitly deferred to Walk-phase analysis.
- This roadmap does not commit to specific commercial partnerships. SiFive engagement is sized appropriately to each phase but is not a critical-path dependency before Run.
- This roadmap does not commit to building all node tiers in custom silicon. T0 and T1 nodes may remain on commercial silicon indefinitely; T5 may always be research silicon.
- This roadmap does not commit Crawl-phase resources to Walk, Run, or Sprint work. Concepts are captured for later phases but are not actively developed until their phase opens.

---

# Part 1 — Architectural Framework

## 1.1 The QBP-Node Concept

A QBP-Node is a unit of compute that participates in a QBP holographic compute network. Every QBP-Node consists of two complementary compute domains on the same physical hardware:

- A **classical compute domain** running standard Linux software — Home Assistant, MQTT brokers, ssh daemons, web dashboards, monitoring stacks, file systems, networking, and any user application that does not specifically require QBP algebraic capabilities.
- A **QBP compute domain** providing algebraic operations — hypergraph traversal, ℍ-mode reasoning, watchdog-monitored physics, and (at higher tiers) octonion or sedenion algebra.

These two domains are not in tension. They are complementary, the way GPU and CPU are complementary in a modern computer: each is good at what the other cannot do efficiently, and they cohabit because most workloads benefit from access to both.

A node is identified by its tier, which determines the algebraic capability of its QBP domain and the silicon class of its classical domain. A node's tier is fixed at manufacture; it cannot be upgraded post-fabrication. Nodes of different tiers participate in the same network, with capability negotiation handling cases where a workload requires algebra beyond a node's local tier.

## 1.2 The Six-Tier Cayley-Dickson Capability Framework

The algebraic capabilities of a QBP-Node are tiered according to the Cayley-Dickson construction. Each tier corresponds to a level in the construction, with each level adding dimension by doubling and losing one structural property. The tier framework gives architectural meaning to the algebraic hierarchy that would otherwise be a mathematical curiosity.

| Tier | Cayley-Dickson level | Algebra | Dimension | Properties retained | Properties lost |
|---|---|---|---|---|---|
| **T0** | 0 | ℝ (real) | 1 | total order, all field axioms | nothing yet |
| **T1** | 1 | ℂ (complex) | 2 | commutative, associative, normed division | total order |
| **T2** | 2 | ℍ (quaternion) | 4 | associative, normed division | commutativity |
| **T3** | 3 | 𝕆 (octonion) | 8 | alternative, normed division | associativity |
| **T4** | 4 | 𝕊 (sedenion) | 16 | power-associative, flexible | normed division (zero divisors arise) |
| **T5** | 5 | trigintaduonion + holographic substrate | 32 | power-associative, flexible | nothing further at the algebra level (holographic substrate is the new capability) |

The strict subset property holds: a tier T_n carries the algebraic capability of all T_k for k ≤ n. A T3 node can perform ℝ, ℂ, ℍ, and 𝕆 operations natively in hardware. This matches the nesting of Cayley-Dickson algebras as subalgebras.

Not every tier requires its own custom silicon. T0 and T1 may be implementable as software on commercial silicon, with no QBP accelerator at all — these tiers are sensors and aggregators that participate classically and rely on higher-tier nodes for any algebraic operations they need. T2 and above benefit from custom silicon, with the cost / power / capability tradeoff increasing per tier.

## 1.3 Tier Mapping to Deployment Roles

The six tiers map onto deployment roles in the existing Helpful Engineering / Sharp Butler infrastructure framework as follows:

| Tier | Deployment role | Typical hardware class | Cost target | Power | Workload examples |
|---|---|---|---|---|---|
| **T0** | Sensor | RV32IM microcontroller, no FPU | <$5 | <0.5 W | Single-sensor reads, threshold comparison, periodic publishing to NATS |
| **T1** | Aggregator | RV32IMF, software ℂ on FPU | <$25 | <1 W | Signal processing, FFT-based correlation, simple multi-sensor fusion |
| **T2** | House Node | RV64GC + small QBP unit, hardware ℍ | <$200 | <5 W | Sharp Butler residential automation, single-property reasoning, Home Assistant host |
| **T3** | Castle Node | RV64GCV + Fano-LUT QBP unit | <$2 000 | <30 W | Contextus ecosystem review, district-level confluence detection, multi-property aggregation, Butler-class work |
| **T4** | Fortress Node | X160-class + ZD detection | <$20 000 | <300 W | Cross-domain BMA, regional Materia-Bio analysis, judge collective members |
| **T5** | Möbius Node | X280-class + holographic substrate | research budget | kW class | QBP physics, GRB analysis, full inter-cell topology computation |

The cost targets are aggressive but realistic for the deployment role. A T0 sensor must cost less than the contractor's lunch because hundreds will be deployed across a watershed. A T5 Möbius node is a research instrument; one or two suffice across a continent.

The mapping is not strictly enforced. A Castle deployment might use T4 hardware if the workload justifies it; a Fortress role might be filled by multiple T3 nodes in some configurations. The tier defines algebraic capability; the deployment role describes typical use.

## 1.4 The Three Execution Contexts

A workload running on a QBP-Node lives in one of three execution contexts. The context determines how the workload is isolated, scheduled, and observed.

| Context | Where it runs | Examples | Isolation model | Observability |
|---|---|---|---|---|
| **Classical native** | Classical core, Linux userspace | ssh daemon, Grafana, system services, Home Assistant core | Standard Unix permissions and namespaces | Standard Linux observability (Prometheus, journalctl, eBPF) |
| **Classical containerized** | Classical core, OCI runtime (Docker / Podman) | User applications, third-party services, Home Assistant integrations | Container namespace + cgroups | Container-aware observability stack |
| **QBP container** | QBP accelerator, QW1024-resident | Sharp Butler reasoning core, BMA cognitive substrate, judge collective members, Trust Receipt validators | Algebraic isolation (see §1.7) | Algebraic watchdog (production or research mode per CSR) |

The classical contexts cover everything Linux already does well. The QBP context is reserved for workloads that benefit from algebraic isolation, holographic redundancy, or one-operation migration (see §1.7).

A typical House Node deployment might run:

- Home Assistant in classical native (the user expects to ssh in and edit YAML configs)
- A handful of Home Assistant integrations in classical containers (each integration is a separate container for stability)
- The Sharp Butler reasoning core in a QBP container (algebraic isolation, holographic backup, fast migration)

The three contexts cohabit on the same physical node. The scheduler routes incoming workloads to the appropriate context based on the workload's requirements (see §1.6).

## 1.5 The Workload-First Methodology

The QBP-Node specification is written workload-first, not hardware-first. The chain of reasoning is:

1. **Identify a target workload** — Sharp Butler residential automation, Contextus eco review, BMA cognitive substrate, QBP physics analysis.
2. **Analyse what mathematics the workload actually requires** — what algebraic operations dominate, what isolation properties are needed, what migration patterns occur.
3. **Derive the algebraic tier requirement** — does the workload need ℍ, 𝕆, or 𝕊 mode? Where does it run, where might it escalate?
4. **Derive the silicon requirements** — what hardware capability is needed to meet the workload contract at the cost / power / latency targets?
5. **Specify the chip** — write the silicon spec last, after the workload requirements are clear.

This methodology protects against over-engineering. It is tempting to design a chip first and then look for workloads that justify it; this produces silicon that does many things but does none of them at the right price point. By starting from workloads, we ensure each tier of silicon earns its existence by serving a real deployment need.

The workload-first methodology also tells us when *not* to build silicon. T0 and T1 workloads are well-served by commercial microcontrollers; there is no reason to build custom silicon for sensors. This methodology produces fewer custom chips than a hardware-first approach would, which is the correct outcome.

Concrete workload analyses for the four reference deployments will appear in Part 2 (Crawl-phase). For the framework layer, the methodology is the contract: every workload's algebraic and silicon requirements will be derived from observed compute patterns, not asserted from architectural ambition.

## 1.6 The Scheduler

A new piece of system software is required: the QBP-Node scheduler routes incoming workloads to the appropriate execution context. The scheduler is small but load-bearing because no existing OS scheduler does this routing.

The scheduler considers four properties of each workload:

- **Algebra requirements**: does the workload need ℍ, 𝕆, or 𝕊 operations? If yes, can the local node serve them, or must escalation occur?
- **Isolation requirements**: does the workload need algebraic isolation, container namespace isolation, or standard Unix permissions?
- **Migration requirements**: does the workload need to move between nodes during execution? If yes, the QBP container context is preferred for the migration semantics.
- **Locality requirements**: does the workload need access to QBP-side hypergraph state, classical filesystem, both, or neither?

A workload's manifest declares these requirements. The scheduler matches them against node capability and routes accordingly. A workload that asks for ℍ-mode operations on a T1 node either escalates to a higher-tier node or fails — there is no silent fallback.

The scheduler is *not* a general-purpose orchestrator like Kubernetes. It coexists with one (Kubernetes can run on the classical domain like any other application). The scheduler's responsibility is the QBP-specific routing decision: which execution context to use, and whether to escalate.

Detailed scheduler design is a Walk-phase concern. For Crawl, a manual workload-to-context mapping suffices because there are few workloads.

## 1.7 The QBP Container and Its Properties

The QBP container is the third execution context and the architecturally novel one. A QBP container is a workload whose entire execution state lives inside a QW1024 region, operated on by the QBP accelerator as a unit.

A QW1024 is 1024 bits — 128 bytes — of structured quaternion data. This is small. Critically:

- The QW1024 holds the workload's **execution context** (registers, control state, watchdog state) but **not its working memory**. Working memory is reached through a controlled bridge to classical DRAM.
- The QW1024 fits in two AVX-512 registers, four AVX2 (YMM) registers, or one VCIX vector register on X280-class hardware. Operations on the entire context are single SIMD operations on this side; on the QBP-accelerator side, they are single accelerator operations.

This sizing produces three architecturally significant properties:

**Algebraic isolation as security primitive.** A workload running in a QW1024 cannot escape its algebraic context except through explicit, watchdog-observed operations. Buffer overflows do not exist at this layer — the workload has no buffer to overflow into. Side-channel attacks are constrained because the algebraic structure of operations is observable; timing attacks must contend with watchdog event streams that record exactly which operations occurred. This is potentially stronger isolation than Intel SGX or ARM TrustZone, though formalising the threat model and proving the security properties is research work.

**Holographic redundancy.** A QW1024 distributed across three nodes via Fano-plane redundancy is *holographically* distributed. Each node holds a phase-shifted version of the whole, and any two of the three can reconstruct the missing third. This follows from the algebraic structure: the Fano plane has 7 points and 7 lines, every point on exactly 3 lines, so 3-redundancy is the natural Fano-plane redundancy floor. Workloads in QBP containers automatically benefit from this property; classical workloads do not unless wrapped in a QBP container.

**One-operation migration.** Moving a QBP container from one node to another is the transmission of a 128-byte payload plus algebraic metadata. The receiver's QBP accelerator continues execution from the received state. Latency is dominated by network round-trip-time, not state size. This makes Sharp Butler reasoning state migration from House Node to Castle Node (when an octonion operation is needed) a practical operation rather than a heavyweight checkpoint-restore.

These properties are real and load-bearing. The QBP container concept is the architectural feature that justifies the dual-domain node design. A node without QBP containers is just a Linux box with an accelerator; a node with QBP containers is a holographic-network participant.

The QW1024 container architecture itself — the precise operations available within a container, the bridge to classical memory, the migration protocol — is a Walk-phase concern. For Crawl, the container is paper architecture; it becomes implementation when there is real RISC-V hardware to host it on.

## 1.8 Holographic Redundancy as the Resilience Model

Resilience in the QBP-Node network is not k-of-n replication, not consensus protocols, and not RAID-style striping. The resilience model is **hyperedge redundancy**, derived from the Fano-plane structure of the QBP algebra.

Every important computation participates in at least three Fano-plane hyperedges. If a node fails, the other two members of the hyperedge can reconstruct the missing computation by associative recall, the way a holographic medium reconstructs a stored pattern from any one of its recording beams.

The Fano plane has 7 points and 7 lines; every point is on exactly 3 lines. This is the natural Fano-plane redundancy floor — three copies, distributed not as identical replicas but as holographically related projections. The reliability property follows automatically from the algebra rather than being a separate replication layer.

For Sharp Butler specifically: a House Node failure does not lose the household's reasoning state because that state participates in a residential hyperedge that includes 2 of the Castle Nodes serving the geographic area. The state is not *replicated* — it is *holographically distributed*. Each Castle Node holds enough phase-and-amplitude information to reconstruct the missing piece, but not a full copy.

The mechanics of hyperedge construction, distribution, and recovery are Walk-phase work. The framework commitment here is that the resilience model is algebraically derived, not engineered as a separate layer.

## 1.9 Inter-Node Operations as Algebraic Transitions

Inter-node operations in the QBP-Node network are themselves algebraic operations. This is not metaphor; it is the design.

A House Node hitting an octonion operation it cannot serve locally issues a CAPSEL.ESCALATE to T3. The escalation is an embedding ℍ ↪ 𝕆 — the workload's quaternion state is embedded into the octonion algebra of the receiving Castle Node. When the operation completes and the result returns to the House Node, the return is a projection 𝕆 → ℍ via PSEL+BSEL — the octonion result is projected back to a quaternion in the appropriate Fano line.

The mode-transition state machine specified for AMODE within a single chip applies *across nodes* with no semantic change. Only the latency changes — from cycles to milliseconds — but the logical structure is identical. This means the cosim test corpus can verify network behaviour using the same tests as chip behaviour. T1.MODE.001 (the mode transition test from the implementation refinements) tests an in-chip AMODE transition; the network analogue tests a node-to-node escalation. Same algebraic invariant, different timescale.

Trust Receipts (see §1.10) carry the algebraic context across the network boundary, providing the integrity proofs that allow the receiving node to verify the operation's place in the originating workload's hyperedge.

The capability negotiation primitives — CAPSEL.LOCAL, CAPSEL.ESCALATE, CAPSEL.RETURN — are Walk-phase concerns. For Crawl, escalation is paper architecture; it becomes implementation when there is a real multi-node network to escalate within.

## 1.10 Trust Receipts as Algebraic Invariants

Every inter-node operation carries a Trust Receipt — a quaternion-valued signature that proves the receiving node's tier, the integrity of the algebraic frame, and the operation's place in the originating workload's hyperedge.

The Trust Receipt is *not* a cryptographic signature primarily, though it can carry one. It is an algebraic invariant: the receipt's quaternion components are derived from the operation's place in the Fano structure, and the watchdog at the receiving node can verify the invariant using existing Layer 0 / Layer 1 primitives. Tampering with operations in transit produces watchdog faults at the next operation, not at message-receipt time.

This integrates with the existing Confluent Trust Hypergraph (CTH) framework. The Trust Receipt format used by inter-node operations should match the CTH Trust Receipt format. This requires inspection of the existing CTH specification, which is in the qbp-lean / Helpful Engineering repositories. **Open item**: confirm Trust Receipt format alignment with the CTH spec before Walk-phase implementation begins. This is captured in Appendix A.

The Trust Receipt mechanism is a Walk-phase implementation concern. For Crawl, it is captured architecturally; the format and verification protocol are deferred.

## 1.11 The SiFive Engagement Story Across Phases

Each phase has a different relationship with SiFive as a commercial partner, and the spec is explicit about what is asked of SiFive at each stage:

- **Crawl**: We are building against SiFive's published VCIX specification. No engagement is required. SiFive is a published-spec dependency, not a relationship.
- **Walk**: We would like to use SiFive reference platforms (X160 Gen 2, X280 Gen 2 boards) for hosting the cycle-accurate simulator and characterising performance against real SiFive silicon. This is a customer relationship, not a partnership.
- **Run**: We are ready to discuss VCIX integration on X160 / X280 for our T3 / T4 tiers. This is a meaningful technical engagement; we walk in with a working accelerator, a verified watchdog, and a workload contract.
- **Sprint**: Full custom silicon partnership for T5 Möbius integration, if SiFive remains the appropriate partner at that stage. Alternative paths (open RISC-V cores, alternative vendors) are explicitly preserved.

The Run-phase architectural choice between dual-domain (separate SiFive core + QBP accelerator) and unified (QBP unit only, hosting RISC-V via QW1024 emulation) affects this story significantly. If unified Run silicon is viable, the SiFive licensing dependency at the Run phase reduces — we license RISC-V as an ISA (open) but not SiFive as a vendor (commercial). This is a substantive architectural choice with strategic implications. It is captured in Appendix A as a deferred decision.

## 1.12 Relationship to Existing Specifications

This QBP-Node spec is not a replacement for existing specifications. It is the navigation layer that organises them.

| Existing specification | Relationship to QBP-Node |
|---|---|
| QBP-CU-SiFive-Interface-Spec-v0.1 | Defines the chip-level VCIX/SSCI interface used by T3+ tiers; absorbed as a reference for Run-phase silicon |
| RV-Fano-Implementation-Refinements (v0) | Defines the instruction set (Layer 0/1/2), mode transitions, ZDCHK semantics; absorbed as the QBP accelerator ISA |
| Fano Cube Compute Cell handoff (physics instance) | Provides the algebraic primitive specifications that the chip implements |
| QBP-Dark-Matter-Fork-Analysis | Drives the Branch B headroom design choice (16-element-capable LUT) |
| BMA architecture documents | Defines the cognitive substrate that runs on QBP containers |
| CTH theory document and JSON inventory | Provides the Trust Receipt format and the network-level confluence semantics |
| EXP-09 integrated photonics viability subnote | Sprint-phase concept inventory entry |
| BMA inter-instance channel CV-QKD option note | Sprint-phase concept inventory entry |

Existing specifications remain authoritative for their own scope. This document references them; it does not duplicate them. When a conflict arises between this spec and an existing spec, the existing spec wins for its scope and this spec is updated to reflect the actual state.

---

# Appendix A — Deferred Decisions

This appendix captures architectural decisions that cannot be made well today. Each entry follows the same structure: concept summary, why it cannot be decided now, what information is needed, which phase generates that information, and the default position pending resolution.

## A.1 Run-Phase Architecture: Dual-Domain vs. Unified

**Concept summary.** Run-phase custom silicon may take one of two forms: a dual-domain design with a separate SiFive core plus a QBP accelerator, or a unified design with only a QBP unit hosting an emulated RISC-V via QW1024 containers. The dual-domain design follows the conventional accelerator pattern; the unified design is architecturally novel and licensing-friendlier.

**Why not now.** The viability of QW1024-hosted RISC-V emulation depends on performance characterisation that has not yet been done. Specifically, the per-instruction overhead of emulating RISC-V atop QBP primitives is unknown. If overhead is 2–3×, unified is clearly correct. If it is 10×, unified is borderline (acceptable for control-plane workloads, painful for data-plane). If it is 100×, unified is unworkable.

**Information needed.** Walk-phase performance characterisation of QW1024-resident RISC-V emulation, against a baseline of native RISC-V execution on equivalent silicon. The relevant comparison is: how many QBP primitive operations are needed to emulate one RISC-V instruction? This is measurable on the Crawl-phase Go simulator extended with a RISC-V emulation path.

**Phase that generates the information.** Walk. Specifically, the QW1024-resident RISC-V emulation feasibility analysis is a named Walk-phase deliverable.

**Default position.** Run-phase silicon is dual-domain unless Walk-phase evidence justifies unified. This default protects against architectural ambition outpacing engineering reality.

## A.2 Trust Receipt Format Alignment with CTH

**Concept summary.** Inter-node operations in QBP-Node carry Trust Receipts. The Confluent Trust Hypergraph framework already defines a Trust Receipt format. These should match, but alignment has not been confirmed.

**Why not now.** The CTH Trust Receipt format is in a private repository. Architecture instance access has not yet been provisioned. Without inspection of the existing spec, we cannot confirm whether the algebraic-invariant structure proposed for inter-node Trust Receipts matches the CTH definition or requires a translation layer.

**Information needed.** Inspection of the CTH specification documents and JSON inventory.

**Phase that generates the information.** Crawl. This is a documentation alignment task, not a development task. It should resolve in days once repository access is provisioned.

**Default position.** Inter-node Trust Receipts use the CTH format if it is suitable; if it is not, this spec defines an extension and the alignment work moves to a CTH revision.

## A.3 BMC Chain-of-Trust for WD_ENABLE

**Concept summary.** The watchdog cannot be permanently disabled in production silicon. The `WD_ENABLE` CSR is reset-to-1 with a sticky bit gate. The mechanism for enforcing this requires a BMC-visible chain-of-trust signal that detects unauthorised clearing.

**Why not now.** The mechanism depends on the specific BMC platform of the target silicon. T2 production silicon has not been chosen. The chain-of-trust design depends on platform capability.

**Information needed.** T2 silicon BMC platform decision (typically a small management microcontroller). This is a Run-phase silicon design choice.

**Phase that generates the information.** Run.

**Default position.** Crawl-phase Go simulator and Walk-phase RISC-V hosted simulator implement the watchdog as software; chain-of-trust is not enforced at these phases. Run-phase silicon must implement the chain-of-trust before any production deployment.

## A.4 Branch B Algebra Headroom

**Concept summary.** If the QBP dark matter fork resolves toward Branch B (algebra extends beyond C ⊕ H ⊕ M₃(C)), the sign ROM in the QBP accelerator is undersized for the new algebra. The current spec provides 16-element-capable LUT silicon but populates only 8 elements.

**Why not now.** Branch A vs. Branch B resolution depends on QBP physics work that is not architecture's scope. The dark matter fork is an active QBP research question.

**Information needed.** QBP physics resolution of the dark matter fork.

**Phase that generates the information.** Indeterminate. Could be resolved in Walk if the physics matures rapidly; could remain open through Run.

**Default position.** Run-phase silicon ships with 16-element-capable LUT silicon, populated with the Branch A 8-element table. If Branch B resolves before Run-phase tape-out, the table is updated. If after, the field-loadable LUT mechanism (if present) is used; otherwise a respin is required.

## A.5 Sign ROM Mask vs. Field-Loadable

**Concept summary.** The sedenion sign ROM (225 entries) and octonion sign ROM (49 entries) can be implemented as mask-burned (cheap, tamper-resistant, requires respin to change) or field-loadable (more area, configurable at boot, mutable if compromised).

**Why not now.** The choice depends on production volume, threat model maturity, and whether the algebra is considered fully settled at tape-out time. None of these are settled in Crawl.

**Information needed.** Run-phase silicon volume estimates, formalised threat model, algebra stability assessment.

**Phase that generates the information.** Run.

**Default position.** FPGA bring-up phase uses field-loadable (rapid iteration during validation). Tiny Tapeout / Efabless MPW phase uses field-loadable (small batches, want to fix sign convention bugs without respin). Production T2 silicon uses mask-burned (volume justifies the cost reduction; threat model demands tamper resistance).

## A.6 Trigintaduonion Support

**Concept summary.** The Cayley-Dickson level-5 trigintaduonion algebra (32-dimensional) is not currently supported by any tier. If the QBP Locale framework eventually requires it, an additional AMODE value and an extended register file are needed.

**Why not now.** The Locale framework does not yet require trigintaduonions. Adding support speculatively would increase silicon area for capability that may never be used.

**Information needed.** QBP Locale framework maturation reaching a state where trigintaduonions are required.

**Phase that generates the information.** Sprint, most likely.

**Default position.** No trigintaduonion support in T0 through T5 as currently specified. If the Locale framework requires it, a T6 tier is added rather than retrofitting existing tiers.

## A.7 Optical Substrate Integration

**Concept summary.** Sprint-phase work involves integrating optical compute substrates per HAMA, integrated photonics CV-QKD per the EXP-09 viability subnote, and possibly direct optical compute via the Clark et al. integrated CV photonics review. The integration architecture is not yet defined.

**Why not now.** Optical substrates are not yet available at the maturity required. The Run-phase architecture (dual-domain vs. unified) interacts with the optical integration design in ways that cannot be settled until Run is closer.

**Information needed.** Maturation of optical substrates to deployment-ready state. Resolution of A.1.

**Phase that generates the information.** Sprint.

**Default position.** Optical substrates are Sprint-phase concept inventory entries. No silicon design commitment is made before Sprint opens.

## A.8 Federation Protocol for Inter-Möbius Communication

**Concept summary.** When multiple Möbius nodes exist, inter-Möbius communication occurs at the highest-tier algebra level. This communication needs a defined protocol — possibly NATS-based, possibly something else, possibly photonic.

**Why not now.** Multiple Möbius nodes do not exist. The first Möbius node does not exist.

**Information needed.** Existence of the first Möbius node and operational experience with it.

**Phase that generates the information.** Sprint.

**Default position.** No federation protocol is specified. NATS is the placeholder for inter-node communication at all tiers below Möbius; whether Möbius requires its own protocol is an open question.

## A.9 Compute-in-Memory (CIM) as a Future Silicon Path

**Concept summary.** Compute-in-Memory architectures unify storage and computation: cells in a CIM-SRAM array both hold data and perform multiply-accumulate operations against an applied input vector. For BMA's hypergraph-based reasoning, this means the hypergraph that stores knowledge and the engine that reasons over it become the same physical structure. The architectural argument was developed in the April 2026 conversation series and recorded in the working note `research-note-hypergraph-as-compute-substrate.md`; the calculations there are extrapolations from published CIM macro characterisations (XNOR-SRAM at ~403 TOPS/W, AI-PiM at 17.63× speedup on vector-matrix multiply).

**Current maturity.** Conceptual. Architectural reasoning and published-data calculations exist; no running emulator, no measured behaviour against QBP/BMA workloads, no path-to-silicon design. The earlier characterisation as a "Run-phase architectural option" overstated maturity.

**Why not decidable now.** Three categories of evidence are needed before CIM can be evaluated as a Run-phase silicon target:

1. *Logical viability* — does the algorithmic mapping from QBP/BMA operations to CIM array operations actually produce correct results? Tested by the Crawl-phase Level-1 functional emulator per §0.4.1.
2. *Performance and energy projections* — what does the architecture look like under realistic timing and energy models, even if the numbers are extrapolated from published silicon? Tested by the Level-2 cycle-and-energy model, in Crawl if §0.4.1 promotion gate passes, otherwise in Walk.
3. *Physical implementability* — does a CIM macro at our process node actually achieve the projected behaviour? Tested by Level-3 SPICE characterisation, which requires analog design expertise the project does not currently have on the team. This is not actionable before Run.

**Information needed.** Outcomes of Level-1, Level-2, and (eventually) Level-3 emulation. The Level-1 / Level-2 distinction matters most for the Crawl/Walk decision; Level-3 matters for the Run decision.

**Phase that generates the information.** Crawl (Level 1, committed). Crawl-or-Walk (Level 2, contingent per §0.4.1). Run (Level 3, deferred).

**Default position.** CIM is a research path with deliberate emulation milestones, not a commitment to a Run-phase silicon target. Conventional silicon (FPGA → MPW → custom ASIC) remains the default Run path. CIM may overtake the conventional path if Level-1 and Level-2 evidence is strongly positive, but the burden of proof is on CIM to displace the default, not on the default to disprove CIM.

---

# Appendix B — Integration with Existing Specifications

This appendix lists the specification documents this spec references, their authoritative scope, and how they fit into the QBP-Node framework.

| Document | Scope of authority | Phase | Relationship |
|---|---|---|---|
| QBP-CU-SiFive-Interface-Spec-v0.1 | Chip-level VCIX/SSCI interface | Crawl + Run | T3+ silicon implements this interface |
| RV-Fano-Implementation-Refinements (v0) | RV-Fano instruction set (Layer 0/1/2), mode transitions, ZDCHK | Crawl | T2+ QBP accelerator implements this ISA |
| Fano Cube Compute Cell handoff | Algebraic primitives, sign tables, three-mode hierarchy | Crawl | Source of truth for the algebra; ROM contents derived from this |
| Sedenion.lean (qbp-lean repo) | Verified sign and index tables | Crawl | ROM extraction source; build pipeline target |
| QBP-Dark-Matter-Fork-Analysis | Branch A vs. Branch B analysis | Crawl + Walk | Drives A.4 (Branch B headroom) |
| BMA architecture documents | BMA cognitive substrate requirements | Crawl + Walk | T3+ workload contract |
| CTH theory document and JSON inventory | Trust Receipt format, network-level confluence | Walk | Drives A.2 (Trust Receipt alignment) |
| EXP-09 viability subnote | Sprint-phase optical capability | Sprint | Concept inventory |
| BMA inter-instance channel CV-QKD note | Sprint-phase secure communication | Sprint | Concept inventory |
| Sharp Butler service tier specification | T2 deployment workload requirements | Crawl + Walk | T2 workload contract |
| `research-note-hypergraph-as-compute-substrate.md` | CIM architectural argument and calculations | Crawl | Drives §0.4.1 (CIM Level-1) and A.9 (CIM viability) |

When this spec conflicts with an existing spec for the existing spec's scope, the existing spec wins. This spec is updated to track. When this spec conflicts with an existing spec for *this spec's* scope (the framework, the phasing, the deferred decisions), this spec wins.

---

**Attribution (carried forward):** Furey, Günaydin & Gürsey, Dixon, Boyle & Farnsworth, Singh, Chamseddine & Connes, Koide, Baez, Moreno, Schafer, Cawagas. SiFive (Asanovic et al.) for VCIX and reference architecture. The QBP physics instance (Claude Opus 4.6) for the algebraic primitive specification. The engineering instance and Gemini for the Go simulator and SIMD assembly path that this framework integrates.

**End of Parts 0 and 1.** Parts 2 (Crawl-phase detailed specification), 3 (Walk-phase outline), 4 (Run-phase outline), and 5 (Sprint-phase concept inventory) follow contingent on review.
