# SiFive — Documentation Patterns and Vendor-Extension Practice

**Compiled:** 2026-05-03
**Compiled by:** Claude (Red Team) for QBP Compute Unit
**Purpose:** Reference for QBP documentation organization. SiFive is the canonical RISC-V vendor; their published practice is the closest real-world analog to what QBP is building. Companion to `Ref/RISC-V-Policies-and-Best-Practices.md`.

---

## 1. Documentation Portal Architecture

The SiFive documentation portal organizes content into six top-level categories:

| Category | Contents |
|----------|----------|
| **SiFive Core IP** | Product briefs and technical specifications for the IP catalog |
| **Case Studies** | Real-world deployment narratives, indexed by core (e.g., "X160 Accelerator Control Unit Use Case Study") |
| **Chips** | Manuals + datasheets for SiFive silicon (Freedom E310, U-series) |
| **Boards** | Dev-platform guides, schematics, BOMs, MCU manuals, image-update procedures |
| **VCIX** | Vector Coprocessor Interface eXtension specifications |
| **Extensions & Standards** | ISA extension specs and TileLink protocol |

Localization: documents are tagged inline with locale codes — `[EN]`, `[ZH]`, `[FR]` etc. — at distinct URLs per locale. Localized variants are first-class citizens, not afterthoughts.

---

## 2. Document Type Ladder

SiFive maintains a clear separation of concerns across document types. Each type has a well-defined audience and scope:

| Doc type | Audience | Scope | Length characteristic |
|----------|----------|-------|------------------------|
| **Family Brief** | Sales, technical evaluators | Marketing-level overview of a product family (Performance, Intelligence, Essential, Automotive) | Short (8–20 pp) |
| **Product Brief** | Same | Marketing-level for one specific core (e.g., X280 Gen 2) | Short |
| **Datasheet** | Hardware engineers integrating the silicon | Electrical / timing / pinout parameters | Medium |
| **Manual** | Architects, firmware/OS implementors | Architectural details, register maps, interrupt model, AXI/TileLink interface | Long |
| **Errata** | Anyone shipping a product based on the IP | Known issues per silicon revision | Short, growing |
| **Application Note** | Software / firmware developers | Cross-cutting topics (e.g., "Optimizing RISC-V Software for Code Density") | Short to medium |
| **Vendor Extension Spec** | Toolchain vendors, kernel developers, application developers | One PDF per extension family. Small. Focused. | **Short — typically <50 pp** |

**Key pattern:** SiFive does *not* publish "the SiFive ISA spec." It publishes nine separate small extension specs, each a standalone PDF. This is a deliberate architectural choice that mirrors the RISC-V principle of small, focused, composable extensions.

**Implications for QBP:**
- The current monolithic `QBP-RISCV-ISA-Spec-v1.0.md` (143 lines mixing base algebra, QEC, memory, registers, risk register) does not match the vendor norm.
- Decompose into per-feature extension specs (see §3 of `RISC-V-Policies-and-Best-Practices.md`).
- A "QBP Architecture Overview" or family brief can stay as a single doc — but the *normative* spec content belongs in per-extension PDFs.

---

## 3. Product Family Structure

SiFive partitions its IP into four families. The families serve as a top-level taxonomy, with individual cores nested inside:

| Family | Series | Target |
|--------|--------|--------|
| **Essential** | E-series, S-series, U-series | Embedded and embedded-server-class cores |
| **Intelligence** | X100, X200, X300, XM-series | AI/ML-focused with vector acceleration |
| **Performance** | P400, P500, P600, P800-series | High-end datacenter processors |
| **Automotive** | E6-A, E7-A, S7-A | Safety-qualified variants for vehicle systems |

**Pattern:** family → product line → specific core → generation. E.g., Intelligence → X-series → X280 → Gen 2.

**Implications for QBP:**
- If QBP eventually offers more than one Run-phase silicon target, this is the precedent for naming. E.g., "QBP-A" (algebraic computing family) → QBP-A100 (Crawl FX-8350-equivalent) → QBP-A200 (Walk RX 9070 XT-equivalent) → QBP-A300 (Run ASIC).
- Premature for now, but worth keeping in mind as the program grows.

---

## 4. SiFive Vendor Extension Catalog (Xsf*)

SiFive's published extensions, with versions, mnemonic prefixes, and purposes — ordered by topical relevance to QBP:

### 4.1 The interface vs. accelerator split

| Extension | Version | Role | Mnemonic |
|-----------|---------|------|----------|
| **`XSfvcp`** | **v1.1.0** | **Vector Coprocessor Interface eXtension (VCIX)** — defines how the scalar core dispatches to a coprocessor | `sf.vc.x`, `sf.vc.v` |
| **`Xsfmm*`** | **v0.6** | Matrix extensions — the coprocessor-internal compute | `sf.mm.*` |

**This is the single most important pattern for QBP.** SiFive separates:
- **The dispatch interface** (`XSfvcp`, ratified at v1.1.0)
- **The accelerator's own instructions** (`Xsfmm*`, still v0.6)

These two concerns evolve at independent rates. The interface is stable; the accelerator's instruction set is still being shaped.

**Direct parallel for the Fano mesh:**
- The interface — how the scalar RISC-V hart dispatches to the mesh — should be `Xqbpvcp`, modeled on `XSfvcp`.
- The mesh-internal instructions — `QSETWLI`, mesh allocation, mesh-internal compute — should be `Xqbpmesh`, modeled on `Xsfmm*`.

Conflating them in one v2.0 spec (Gemini's plan) is exactly the mistake SiFive avoided.

### 4.2 The full Xsf* catalog (from LLVM)

| Extension | Version | Purpose |
|-----------|---------|---------|
| `Xsfmm*` | v0.6 | Matrix extensions |
| `XSfvcp` | v1.1.0 | Vector Coprocessor Interface (VCIX) |
| `Xsfvfexp16e` | v0.5 | Vector exponential (FP16) |
| `Xsfvfbfexp16e` | v0.5 | Vector exponential (BF16) |
| `Xsfvfexp32e` | v0.5 | Vector exponential (FP32) |
| `Xsfvfexpa` | v0.2 | Vector exponential approximation |
| `Xsfvfexpa64e` | v0.2 | Vector exponential approximation (FP64) |
| `XSfvqmaccdod` | v1.1.0 | Int8 matrix multiply (DOD layout) |
| `XSfvqmaccqoq` | v1.1.0 | Int8 matrix multiply (QOQ layout) |
| `Xsfvfnrclipxfqf` | v1.0.0 | FP32-to-int8 ranged clipping |
| `Xsfvfwmaccqqq` | v1.0.0 | Matrix multiply-accumulate |
| `XSiFivecdiscarddlone` | (n/a) | L1 cache discard |
| `XSiFivecflushdlone` | (n/a) | L1 cache flush |
| `XSfcease` | v1.0.0 | Cease instruction |

**Patterns visible from this table:**

1. **Granularity:** Each extension covers *one feature*. Exponential, matrix multiply, cache management — each is its own spec.
2. **Versioning:** `v0.x` means draft / pre-silicon / pre-toolchain-stable. `v1.0.0` is reached only after silicon validation and stable toolchain integration. `v1.1.0` is a backward-compatible refinement.
3. **Vendor prefix is universal.** Every mnemonic starts with `sf.`. No exceptions.
4. **Even SiFive's flagship matrix extension is at v0.6** — they have not yet declared it stable. This is the realistic cadence.

---

## 5. Versioning Maturity Pattern

The SiFive Xsf* catalog reveals a clear maturity ladder:

| Version range | Maturity | What it implies |
|---------------|----------|-----------------|
| **v0.1 – v0.4** | Early draft | Experimental; subject to breaking changes; not in production silicon |
| **v0.5 – v0.9** | Late draft / stable | Spec-stable enough for software prototyping; toolchain integration may be experimental-flagged |
| **v1.0.0** | Initial release | Silicon-validated; toolchain integrated; backward-compatibility starts here |
| **v1.x.0** | Backward-compatible refinements | Minor additions; existing code continues to work |
| **v2.0.0** | Backward-incompatible major revision | Rare. Requires a strong justification. |

**SiFive's flagship matrix extension `Xsfmm*` is at v0.6 today.** They have shipped silicon, but have not yet committed to the stability promise of v1.0.

**Implications for QBP — direct contrast with Gemini's plan:**

- The QBP RISC-V ISA Spec was approved at **v1.0** on 2026-05-04.
- Gemini's plan proposes **v2.0 thirteen days later** — a backward-incompatible major revision.
- This cadence has **no industry analog**. SiFive's path is v0.x → v0.x → v0.x → v1.0 (after silicon) → v1.1 → v1.2 → … → eventual v2.0 only when truly necessary.
- The QBP spec should consider whether v1.0 was premature labeling. A more honest current label would be **v0.6 to v0.9**, with v1.0 reserved for after Walk-phase ROCm/AVX validation.

---

## 6. The VCIX Pattern (XSfvcp) — Architectural Template for QBP Mesh Dispatch

VCIX is worth examining in detail because it solves the exact problem the Fano mesh poses: how does a scalar RISC-V core dispatch to a vector / matrix / mesh coprocessor without baking the coprocessor's microarchitecture into the scalar ISA?

### 6.1 What VCIX does

- Defines a small set of `sf.vc.*` instructions in the scalar core.
- These instructions package operands (general-purpose registers, vector registers, immediates) and an opaque "function code."
- The coprocessor decodes the function code and executes its own instruction set.

### 6.2 What VCIX does not do

- Does not specify the coprocessor's instruction set.
- Does not specify the coprocessor's microarchitecture (number of lanes, mesh topology, register file).
- Does not assume a specific coprocessor family.

### 6.3 Why this matters for QBP

The Fano mesh is a coprocessor. The plan to expose `QSETWLI` and mesh allocation in the scalar custom-0 opcode space couples the scalar ISA to the mesh microarchitecture. This is exactly what VCIX was designed to avoid.

**The conformant pattern for QBP:**

| Concern | Where it lives |
|---------|----------------|
| Scalar core dispatches operands to mesh | `Xqbpvcp` (modeled on `XSfvcp`) |
| Mesh allocation, width selection, watchdog config | `qmesh` CSR cluster (modeled on `vtype`/`vl`) |
| Mesh-internal instructions (e.g., the 28-lane QW8 packing) | `Xqbpmesh` |
| Quaternion algebra (when executed by scalar core or by mesh) | `Xqbpquat` |
| Octonion algebra | `Xqbpoct` |
| Quantum error correction | `Xqbpqec` |
| Wide memory ops | `Xqbpmem` |

This decomposition is **a strict superset of what Gemini wants to deliver**, but it does not require modifying the v1.0 spec. Each new extension can land at `v0.1` and mature independently.

---

## 7. Other Documentation Conventions Worth Noting

### 7.1 Hierarchical breadcrumbs

SiFive's portal uses umbrella → product → variant pattern: "SiFive Intelligence Gen 2 Family Brief" → "X160 Gen 2" → variant. This is mirrored in their case studies, which cross-reference product lines explicitly.

### 7.2 Versioned filenames

Document filenames carry versions inline: `HiFive Premier P550 Getting Started Guide V1p2`, `HiFive Unmatched Getting Started Guide v1p4`. The `p` separator matches the RISC-V ISA-string convention (§3.3 of the policies doc).

### 7.3 Separation of marketing from normative content

- **Briefs** are marketing-oriented summaries (allowed to make claims).
- **Manuals + Datasheets** are normative (binding to silicon).

QBP's current "QBP-Compute-Unit-Master-Record.docx" mixes both registers. The patterns above suggest splitting:
- A QBP family brief (marketing-level claims about advantages)
- A QBP architecture manual (normative, binding to the spec)
- A QBP roadmap (Crawl/Walk/Run/Glide/Fly progression)

### 7.4 Errata as first-class artifacts

SiFive publishes errata sheets per silicon revision. This is the standard hardware practice and would matter once QBP reaches Run-phase ASIC tape-out.

### 7.5 Application notes for cross-cutting topics

"Optimizing RISC-V Software for Code Density" is the kind of cross-cutting doc that doesn't belong in any spec. QBP's BMA integration content (currently in `doc/BMA-Emulator-Integration.md` and `doc/memo_to_bma_implementor.md`) is best characterized as application notes.

---

## 8. Documentation Conventions QBP Should Adopt

| # | Convention | Current QBP state | Recommended action |
|---|------------|-------------------|---------------------|
| D1 | Vendor mnemonic prefix on every instruction | None — uses bare `QMUL`, `QROT`, etc. | Add `qbp.` prefix in v1.1 |
| D2 | One PDF per extension family | One monolithic v1.0 doc | Decompose per §6 above |
| D3 | Version with `p` separator in ISA strings | Uses `v1.0` | Keep `v1.0` in titles; use `1p0` in any ISA-string output |
| D4 | Family Brief / Product Brief / Manual / Datasheet / Errata / App Note ladder | Mixed across `MANIFEST.md`, `Master-Record.docx`, etc. | Reorganize over Walk phase |
| D5 | Per-extension version cadence (v0.x → v1.0 only after silicon) | Spec at v1.0 with no silicon | Consider relabeling to v0.6 or v0.9 |
| D6 | Coprocessor-interface separated from coprocessor-instructions | Conflated | Adopt VCIX pattern: `Xqbpvcp` + `Xqbpmesh` |
| D7 | Errata document per silicon revision | None (no silicon yet) | Add when Run-phase ASIC ships |
| D8 | Localization tags on documents | None | Defer until traction warrants |
| D9 | Trap behavior section in every spec | Missing | Add to v1.1 |
| D10 | Revision history at end of each spec | Missing | Add to v1.1 |

---

## 9. Sources

| # | Source | URL |
|---|--------|-----|
| 1 | SiFive Documentation Portal | https://www.sifive.com/documentation |
| 2 | SiFive Vendor Extensions in Linux 6.16 (Phoronix) | https://www.phoronix.com/news/Linux-6.16-New-SiFive-RISC-ISA |
| 3 | SiFive RISC-V Vector Extension Intrinsic Support | https://www.sifive.com/blog/risc-v-vector-extension-intrinsic-support |
| 4 | SiFive RISC-V Core IP Portfolio | https://www.sifive.com/risc-v-core-ip |
| 5 | SiFive RiscvSpecFormal (Coq formal model) | https://github.com/sifive/RiscvSpecFormal |
| 6 | LLVM RISC-V Usage Guide (Xsf* catalog) | https://llvm.org/docs/RISCVUsage.html |
| 7 | Linux SiFive vendor extensions patch series | https://patchew.org/linux/20250418053239.4351-1-cyan.yang@sifive.com/20250418053239.4351-2-cyan.yang@sifive.com/ |

---

*End of document. Companion file: `Ref/RISC-V-Policies-and-Best-Practices.md`. Both should be cited from `architecture/peer-review-002-fano-mesh-isa-redteam.md`.*
