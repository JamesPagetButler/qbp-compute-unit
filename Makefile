# QBP-Compute-Unit Makefile
# Root targets for the lean2rom pipeline and emulator tests.
#
# Usage:
#   make sign-roms       — Regenerate roms/ from lean/QBP/Sedenion.lean
#   make verify-roms     — Verify ROM checksums match Lean source (CI gate)
#   make test            — Run all emulator tests
#   make bench           — Run all benchmarks
#   make vet             — Run go vet

.PHONY: sign-roms verify-roms test bench vet

# Regenerate all four ROM hex files, Go constants, and asm constants.
# Runs lean2rom which either invokes Lean (if available) or uses the
# Go-native Cayley-Dickson derivation as fallback.
# Only needs to be re-run when lean/QBP/Sedenion.lean changes.
sign-roms:
	go run ./cmd/lean2rom -root .
	@echo "ROMs regenerated; commit roms/ and run cosim tests"

# Verify ROM checksums match the Lean source without regenerating.
# Exits non-zero if any checksum mismatches; blocks CI on drift.
verify-roms:
	go run ./cmd/lean2rom -root . -verify
	@echo "ROM checksums match Lean source"

# Run all emulator tests (includes TestSIMDConstantsMatchROM).
test:
	cd emulator && go test ./...

# Run benchmarks — all should report 0 B/op, 0 allocs/op.
bench:
	cd emulator && go test -bench=. -benchmem ./...

# Run go vet across all packages.
vet:
	cd emulator && go vet ./...
	go vet ./cmd/lean2rom
