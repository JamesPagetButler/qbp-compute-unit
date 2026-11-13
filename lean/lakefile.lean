import Lake
open Lake DSL

-- Minimal package for the repo's core-Lean-4 proof files (no Mathlib, no `require`).
-- Toolchain pinned via lean-toolchain (v4.30.0); see .github/workflows/verify-lean-roms.yml
-- for the CI wiring (qbp-compute-unit#82 AC-c).
package qbpcu where

@[default_target]
lean_lib QBP where
  globs := #[.submodules `QBP]

@[default_target]
lean_lib T10_CDLane where

@[default_target]
lean_lib T10Ties where
