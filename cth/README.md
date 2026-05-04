# CTH: Confluent Trust Hypergraph Engine

**Status:** Alpha / Crawl Phase
**Reference:** [Confluent-Trust-Hypergraph-Theory-v0_2.md](../Confluent-Trust-Hypergraph-Theory-v0_2.md)

## Overview

The CTH engine is a Go library that implements the formal theory of *confluent trust hypergraphs*. It provides an information-theoretic framework for quantifying the epistemic health of scientific research programmes.

This engine unifies three formalisms:
1. **Shannon Information Theory:** Measuring entropy and channel capacity.
2. **Dempster-Shafer Belief Functions:** Representing ignorance and belief.
3. **Confluence Theory:** Detecting errors via independent derivation paths.

## Architecture

The engine is structured into five core packages:
- `model/`: Pure data types (Anchors, Chains, Confluences).
- `compute/`: Pure functions for entropy, fidelity, mutual information, and compression.
- `store/`: Storage interfaces (JSON, MuninnDB, SurrealDB).
- `report/`: Generation of health dashboards and river-map narratives.
- `cmd/cth/`: CLI tool for inventory analysis and programme merging.

## Development Status

- [ ] **Crawl Phase:** Core types, compute functions, JSON storage.
- [ ] **Walk Phase:** MuninnDB integration, NATS events, Hebbian decay.
- [ ] **Run Phase:** SurrealDB storage, BMA agent navigation flow field.

## License

Business Source License 1.1 (BSL 1.1)
Changes to Apache 2.0 on 2030-01-01.
See [LICENSE](LICENSE) for details.
