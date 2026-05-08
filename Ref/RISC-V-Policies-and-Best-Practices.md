# RISC-V — Policies and Best Practices

**Compiled:** 2026-05-03
**Compiled by:** Claude (Red Team) for QBP Compute Unit
**Purpose:** Authoritative reference for QBP ISA design and review work. All ISA proposals — Gemini's, Claude's, future contributors' — are to be measured against this document.

This is a working summary of RISC-V International's published rules, the Toolchain SIG's vendor policy, the Vector ("V") spec's configuration model, and the LLVM project's treatment of vendor extensions. It is descriptive, not aspirational: every claim below has a citable source in §10.

---

## 1. Specification Maturity Workflow

RISC-V International defines a four-stage maturity ladder. The defining quotes are from the official ratified-specifications page:

| Stage | RISC-V's defining text |
|-------|------------------------|
| **Draft / Development** | "Assume everything is subject to change… ideas, structures, and content are still evolving." |
| **Stable** | "Core structure settled; limited, carefully-considered modifications expected." |
| **Frozen** | "Changes are highly unlikely… modifications will only be made in response to critical issues." |
| **Ratified** | **"No changes are allowed… must be addressed through a follow-on extension."** |

**Implications for QBP:**

- "Approved for Hardware Emulator Integration" (the status currently on the QBP RISC-V ISA Spec v1.0) is closest to **Stable→Frozen**, *not* Ratified. New work *can* still happen, but it must be carefully scoped.
- Once a QBP ISA is Ratified, the conformant path for new functionality is a **follow-on extension**, never a major-version bump.
- Bumping v1.0 → v2.0 of an existing ratified document is **not a thing in RISC-V**. The procedural form is `Xqbp<base>` v1.0 + `Xqbp<newfeature>` v0.1 (a new extension document at draft maturity).

---

## 2. Specification Document Architecture

RISC-V's canonical document structure:

| Document | Contents |
|----------|----------|
| **Volume I — Unprivileged ISA** | Base instructions, encoding, programmer's model, registers (user-mode) |
| **Volume II — Privileged Architecture** | M/S/U mode, CSRs, traps, virtual memory, interrupts |
| **Per-extension specs** | One PDF per extension family (e.g., `riscv-v-spec.pdf`, `riscv-bitmanip.pdf`, `riscv-crypto-spec.pdf`) |
| **Profile specs** | RVA20, RVA22, RVA23 (application processors), RVI20 (microcontrollers) — bundles of mandatory extensions for a market segment |
| **Non-ISA specs** | ABI, debug, calling conventions, etc. |

**Key pattern:** *small focused documents*, not one monolithic spec. The combined manual that compiles all the above is generated; the canonical sources are the per-extension specs.

**Implications for QBP:**

- The current single-doc `QBP-RISCV-ISA-Spec-v1.0.md` (143 lines, mixing base algebra + QEC + memory + registers + risk register) is structurally non-conforming.
- The conformant decomposition for QBP is at minimum:
  - `Xqbpquat` — quaternion algebra (custom-0)
  - `Xqbpoct` — octonion algebra
  - `Xqbpqec` — quantum error correction (custom-1)
  - `Xqbpmem` — wide memory ops (custom-2)
  - `Xqbpmesh` — Fano mesh dispatch (new — Gemini's mesh refinements belong here)
  - `Xqbpvcp` — coprocessor-interface, modeled on SiFive's VCIX (`XSfvcp`)

---

## 3. Extension Naming Convention

From `riscv-isa-manual/src/naming.adoc` (the binding registry):

### 3.1 Single-letter extensions (base set)

| Letter | Extension |
|--------|-----------|
| **I** | Base integer ISA |
| **M** | Multiplication / division |
| **A** | Atomics |
| **F** | Single-precision FP |
| **D** | Double-precision FP |
| **Q** | Quad-precision FP |
| **C** | 16-bit compressed instructions |
| **V** | Vector extension |
| **B** | Bit manipulation (umbrella) |
| **K** | Cryptography (umbrella) |
| **P** | Packed SIMD |
| **H** | Hypervisor |

`G` is shorthand for `IMAFDZicsr_Zifencei` (the "general-purpose" baseline).

### 3.2 Multi-letter extensions

| Prefix | Use | Examples |
|--------|-----|----------|
| **Z\<name\>** | Standard unprivileged sub-extensions; **the first letter after Z indicates the most closely related single-letter category** | `Zicsr` (CSR access, Zi*), `Zifencei` (Zi*), `Zicbom` (cache-block management, Zic*), `Zfa` (additional FP, Zf*), `Zfh` (half-precision FP), `Zba`/`Zbb`/`Zbc`/`Zbs` (bit-manip subsets, Zb*), `Zbkb`/`Zbkx`/`Zknd` (crypto subsets, Zk*), `Zvfh`/`Zvbb` (vector subsets, Zv*) |
| **Sm\<name\>** | Machine-level supervisor extensions | `Smaia` (machine advanced interrupt arch.) |
| **Ss\<name\>** | Supervisor-level extensions | `Ssaia`, `Sscofpmf` |
| **Sv\<name\>** | Virtual-memory extensions | `Sv32`, `Sv39`, `Sv48`, `Sv57` |
| **Sh\<name\>** | Hypervisor-level extensions | (placeholder) |
| **Su\<name\>** | User-level extensions | (placeholder) |
| **X\<vendor\>\<name\>** | **Non-standard / vendor extensions** | `XSfvcp`, `XTHeadVdot`, `XVentanaCondOps`, `Xqci*`, `XCVbitmanip` |

### 3.3 Version numbering

- Major / minor separator is **`p`** (not `.`) when used in ISA strings: `rv32i2p2` = "version 2.2 of RV32I".
- **Major** changes are backward-incompatible.
- **Minor** changes are backward-compatible.
- Document titles may use `v1.0` / `v2.0` for human readability, but tools parse `2p0`.

### 3.4 Canonical ISA-string ordering

Base → single-letter (MAFDQCBVPH) → Z* → Su* → Ss* → Sv* → Sh* → Sm* → **X*** (always last).

X-extensions must follow all standard extensions. Underscores separate multi-letter extensions for readability.

**Implications for QBP:**

- The QBP extension family must commit to an `X<vendor>` umbrella name. Recommended: `Xqbp` as the vendor cluster, with feature suffixes (`Xqbpquat`, `Xqbpoct`, `Xqbpqec`, `Xqbpmem`, `Xqbpmesh`, `Xqbpvcp`).
- Without an X-name, the QBP spec cannot appear in any conformant ISA string.

---

## 4. Vendor Extension Policy (Toolchain SIG)

From `riscv-non-isa/riscv-toolchain-conventions/vendor-policy.adoc`, supplemented by the LLVM RISC-V usage guide.

### 4.1 Mandatory mnemonic prefix

Every vendor instruction's mnemonic **must** carry a vendor prefix that:
- Is **at least 2 characters** long.
- Is lowercase.
- Ends with a period.
- Corresponds to the vendor (not the feature).

### 4.2 Registered vendor prefixes

| Vendor | Mnemonic prefix | Example |
|--------|------------------|---------|
| SiFive | `sf.` | `sf.vc.x`, `sf.vfexp32.v` |
| T-HEAD (Alibaba) | `th.` | `th.mula`, `th.bb` |
| Ventana | `vt.` | `vt.maskc` |
| Qualcomm | `qc.` | `qc.csrr` |
| OpenHW Group (CORE-V) | `cv.` | `cv.elw`, `cv.macsn` |
| Andes Technology | `nds.` | `nds.bfoz` |
| SpacemiT | `smt.` | `smt.vdot` |
| MIPS | `mips.` | `mips.lsp` |
| Esperanto / AI Foundry | `aif.` | (family) |
| WCH (QingKe) | `wch.` | (compressed family) |
| Rivos | (experimental) | (Vizip family) |

### 4.3 Allocation procedure

- Open-source project maintainers may PR the conventions repo to **request** allocation of a vendor prefix.
- Presenting the extension spec when requesting a prefix is not mandatory but is "good faith."
- The Toolchain SIG **reserves the right to reclaim** a vendor prefix for misuse or insufficient upstreaming.

### 4.4 What a vendor extension spec must publish

Quoting the policy:

> "a link to the documentation that details the extension name, version number, instructions list, CSRs, and other components."

### 4.5 Toolchain integration

- LLVM treats unratified vendor extensions as `experimental-` and requires `-menable-experimental-extensions` to enable them.
- "There is explicitly no compatibility promised between versions of the toolchain" for experimental extensions.
- Inclusion in upstream LLVM is "case by case basis. All proposals should be brought to the bi-weekly RISC-V sync calls for discussion."
- **Mnemonic-prefix conformance is a hard gate**: no upstream toolchain will accept a vendor extension whose mnemonics lack a vendor prefix.

**Implications for QBP:**

- Every QBP instruction in the current Spec v1.0 (`QMUL.w`, `QROT.w`, `OMAC.w`, `FANO`, `QNORM.w`, etc.) **violates the mnemonic-prefix policy**.
- Conformant mnemonics: `qbp.qmul.w`, `qbp.qrot.w`, `qbp.omac.w`, `qbp.fano`, `qbp.qnorm.w`, etc.
- Until the spec adopts a vendor prefix and registers it, no upstream LLVM/GCC integration is possible. This is a **Walk-phase blocker** for the BMA integration path.

---

## 5. Custom Opcode Space Rules

| Opcode | Standard name | Stability promise |
|--------|---------------|-------------------|
| `0x0B` | `custom-0` | **Never to be ratified.** Vendors may reuse freely. No interoperability guarantee. |
| `0x2B` | `custom-1` | Same. |
| `0x5B` | `custom-2` | Same. RISC-V has signaled possible future allocation here. |
| `0x7B` | `custom-3` | Same. |

**Crucial property:** code using the custom opcode space is **not portable** between vendors. Two QBP-conformant implementations are guaranteed to be compatible *only if both follow the QBP X-extension spec at the same version*.

Standard extensions get RISC-V-allocated opcode bits. The price of interoperability is the ratification process. The price of the custom space is no portability.

**Implications for QBP:**

- The current allocation in QBP-RISCV-ISA-Spec-v1.0 — `custom-0` for continuous algebra, `custom-1` for QEC, `custom-2` for memory — is *technically permissible*, but it commits QBP to a closed ISA family unless it later seeks ratification.
- The spec must **explicitly state** the closed-ecosystem property. Current spec does not.

---

## 6. Vector Extension (V) — the Architectural Template for Configuration State

The V-spec is the closest in-tree analog to what QBP is doing (a compute-heavy extension with width/length configuration). Its configuration model is the gold-standard reference.

### 6.1 Configuration CSRs

| CSR | Field | Role |
|-----|-------|------|
| `vtype` | `vill, vma, vta, vsew, vlmul` | Element width, register-grouping multiplier, agnosticism flags, **illegality bit** |
| `vl` | (entire CSR) | Active vector length |
| `vlenb` | (read-only) | Implementation's max vector length in bytes |

`vsew` (3-bit, encodes 8/16/32/64/128/256/512/1024) maps directly onto QBP's funct3 width selector — but RISC-V puts it in a CSR, not in the instruction encoding.

`vlmul` — register-grouping multiplier. Treats N consecutive vector registers as one logical wider register. **This is the standard pattern for the 28-lane QW8 SIMD packing Gemini's R3 proposes.**

### 6.2 Configuration instructions

- `vsetvli rd, rs1, vtypei` — I-type. vtype encoded as immediate. AVL from rs1.
- `vsetivli rd, uimm, vtypei` — I-type. AVL is also immediate.
- `vsetvl rd, rs1, rs2` — R-type. vtype value comes from rs2 register.

These instructions **set the CSRs**. They are not data-path opcodes. The CSR-write is the architectural action; the instruction is the convenient encoding.

### 6.3 Illegal-config semantics

> If the `vtype` setting is not supported by the implementation, then the `vill` bit is set in `vtype`.

**Subsequent vector instructions trap with illegal-instruction exception.** They do **not** silently downgrade. RISC-V never allows hardware to autonomously change the precision or semantics of a computation.

### 6.4 Context-switch / preemption

`vtype` and `vl` are **per-hart CSRs**. The OS context-switch path saves/restores them as part of the standard CSR save-restore sequence. Vector register state is also per-hart. Multi-hart systems share no vector configuration state implicitly.

### 6.5 Determinism

The V-spec mandates that for identical inputs and identical `vtype`/`vl`, vector instructions produce identical outputs. There is no autonomous precision throttling. The architectural state is fully observable.

**Implications for QBP — Critical:**

- Mesh allocation and width selection should live in **a `qmesh` CSR** (or a small CSR cluster), not in custom-0 data-path opcodes.
- The instruction the user calls `QSETWLI` should be a **CSR-writing pseudo-op**, exactly mirroring `vsetvli`.
- Any "illegal mesh request" should set a `vill`-equivalent bit and **trap** subsequent mesh-using instructions. **Silent downgrade (Gemini's QYIELD) directly inverts this model.**
- Mesh state must be per-hart and OS-saveable to maintain context-switch correctness.

---

## 7. Profiles (RVA / RVI)

Profiles bundle extensions for a target market and define a portability target.

| Profile | Year | Target |
|---------|------|--------|
| **RVI20** | 2020 | Microcontrollers (RV32I/RV64I + Zicsr/Zifencei) |
| **RVA20** | 2020 | Application processors (G + C) |
| **RVA22** | 2022 | Application processors with B (bit-manip) and crypto |
| **RVA23** | 2023 | Adds V (vector) as mandatory; broader Zicb cluster |

Profiles are **mandatory for software portability**. A binary compiled for RVA22 will run on any RVA22-conformant processor. Vendor X-extensions are *not* part of any profile and require explicit dispatch.

**Implications for QBP:**

- QBP silicon at the Run-phase ASIC level is unlikely to claim a standard RVA profile because the FX-8350-class cost target precludes it. Realistic claim is "RV64IM + Xqbp\*".
- Software compiled with QBP X-extensions cannot run on non-QBP RISC-V chips. The spec should say so.

---

## 8. LLVM Toolchain Treatment of Vendor Extensions

From the LLVM RISC-V usage guide:

1. Vendor extensions are gated behind `-menable-experimental-extensions` until they are ratified.
2. **No version-to-version compatibility promise** for experimental extensions.
3. Each extension lists its specific spec version (e.g., `Xsfmm` v0.6, `XSfvcp` v1.1.0).
4. New vendor-extension proposals are evaluated case-by-case at the bi-weekly RISC-V sync call.
5. Mnemonic-prefix conformance is enforced.

**Notable observation:** LLVM tracks 70+ vendor extensions across 11+ vendors. None of them are in the QBP family naming pattern. QBP is currently invisible to upstream tooling.

---

## 9. The "Dispatch via Coprocessor Interface" Pattern

SiFive ratified a vendor extension called **`XSfvcp`** (v1.1.0) — the **Vector Coprocessor Interface eXtension** (VCIX). This is the standard pattern for "scalar core dispatches to a coprocessor mesh":

- The scalar core issues `sf.vc.*` instructions.
- These instructions package operands and route them to a coprocessor.
- The coprocessor is described in a *separate* extension document.
- The **interface** (VCIX) and the **coprocessor's instructions** (e.g., `Xsfmm`) are independent extensions. They can evolve at different rates.

**Implications for QBP — directly relevant to Gemini's mesh proposal:**

- The Fano mesh is an *accelerator*. The scalar core's interaction with it is an *interface*.
- These should be **two separate extensions**: `Xqbpvcp` (interface) and `Xqbpmesh` (mesh-internal instructions).
- Conflating them in one v2.0 spec (Gemini's plan) repeats a mistake the RISC-V ecosystem has already learned not to make.

---

## 10. Sources

| # | Source | URL |
|---|--------|-----|
| 1 | RISC-V Ratified Specifications | https://riscv.org/specifications/ratified/ |
| 2 | RISC-V Technical Specifications wiki | https://lf-riscv.atlassian.net/wiki/spaces/HOME/pages/16154769/RISC-V+Technical+Specifications |
| 3 | RISC-V Ratified Extensions wiki | https://lf-riscv.atlassian.net/wiki/spaces/HOME/pages/16154732/Ratified+Extensions |
| 4 | ISA Naming Convention (riscv-isa-manual) | https://github.com/riscv/riscv-isa-manual/blob/main/src/naming.adoc |
| 5 | Vendor Extension Policy (riscv-toolchain-conventions) | https://github.com/riscv-non-isa/riscv-toolchain-conventions/blob/main/vendor-policy.adoc |
| 6 | LLVM RISC-V Usage Guide (vendor extension catalog) | https://llvm.org/docs/RISCVUsage.html |
| 7 | RISC-V V Vector Extension v1.0 (lists) | https://lists.riscv.org/g/tech-vector-ext/attachment/686/0/riscv-v-spec.pdf |
| 8 | RISC-V V Vector Extension v1.0 (docs) | https://docs.riscv.org/reference/isa/extensions/vector/_attachments/riscv-v-spec.pdf |
| 9 | RISC-V Opcodes Repository | https://github.com/riscv/riscv-opcodes |
| 10 | Red Hat Research — RISC-V extensions overview | https://research.redhat.com/blog/article/risc-v-extensions-whats-available-and-how-to-find-it/ |
| 11 | Five EmbedDev — User-ISA naming reference | https://five-embeddev.com/riscv-user-isa-manual/Priv-v1.12/extensions.html |
| 12 | LLVM Vector Extension reference | https://llvm.org/docs/RISCV/RISCVVectorExtension.html |

---

*End of document. Companion file: `Ref/SiFive-Documentation-Patterns.md`.*
