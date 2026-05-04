// Package cth implements the Confluent Trust Hypergraph (CTH) engine.
//
// The engine provides information-theoretic metrics for assessing the
// epistemic health of scientific research programmes.
//
// Core concepts:
//
//   - Anchors: Claims with associated trust tiers (Axiom, Proof, Measurement, Prediction).
//   - Chains: Directed derivation paths from axioms to targets.
//   - Confluences: Points where independent chains meet, providing error detection.
//   - Metrics: Residual entropy (uncertainty), confirmatory information (evidence),
//     and net compression ratio (explanatory power).
//
// For theoretical background, see: Confluent-Trust-Hypergraph-Theory-v0_2.md
package cth
