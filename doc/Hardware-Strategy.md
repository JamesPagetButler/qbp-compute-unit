# QBP Compute Unit: Walk Phase Hardware Strategy

This document formally captures the strategic transition of hardware from the Crawl Phase (AMD FX-8350 software emulation) to the Walk Phase (AVX-512 / GPU acceleration).

## The Three Upgrade Paths

### 1. The High-Performance Path (Brute Force)
**Goal:** Maximum possible wall-clock throughput for QW128 physics simulation and massive BMA hypergraph spreading activation.
- **CPU:** AMD Ryzen Threadripper PRO 9000 WX-Series (Zen 5/6) for massive 8-channel DDR5 memory bandwidth and AVX-512 vector execution width.
- **GPU:** Dual AMD Radeon RX 9070 XTs (ROCm backend). One for SENSE emulation, one for ACT boundary visualization.
- **Best For:** Centralized Deep Compute / Physics processing where speed is the only metric.

### 2. The Cost-Efficient Beast (Sharp Butler Node)
**Goal:** A distributed "House Node" that runs the full BMA software stack with exceptional power efficiency and a highly accessible price point.
- **CPU:** AMD Ryzen 9 9900X (Zen 5). Retains AVX-512 but strictly caps TDP to ~65W-120W.
- **GPU:** Integrated graphics or a single low-profile GPU. Offloads all heavy lifting to the CPU's AVX kernel to preserve power.
- **Best For:** Distributed edge compute, 24/7 BMA inference running locally.

### 3. Highly Optimized Cost/Power Server Node (ASIC)
**Goal:** Achieve an astronomical Cost-to-Power ratio for massive datacenter deployment by ruthlessly tightening feature requirements.
- **Architecture:** Custom RISC-V ASIC (28nm or 130nm OpenMPW).
- **Features:** Strips away OS, branch prediction, and complex floating point. Implements strictly QW8 (int8) TMAC and Fano-plane routing.
- **Best For:** Cloud inference servers focused on operational expense (OpEx) energy reductions.

## The Hardware Evaluator (hw-eval)
Because component prices and power profiles change, this repository includes an automated Go CLI (`cmd/hw-eval`) to calculate optimal part lists and total cost/power based on these strategies. It allows developers to specify which components they already own (e.g., `-own-gpu=1`) to get an accurate upgrade cost.
