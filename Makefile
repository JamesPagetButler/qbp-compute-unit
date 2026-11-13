# QBP-Compute-Unit Makefile
# Root targets for the lean2rom pipeline and emulator tests.
#
# Usage:
#   make lean-build      — Build lean/ (toolchain-pinned); fails on compile error or any sorry
#   make sign-roms       — Regenerate roms/ from lean/QBP/Sedenion.lean
#   make verify-roms     — Verify ROM checksums match Lean source (CI gate)
#   make test            — Run all emulator tests
#   make bench           — Run all benchmarks
#   make baseline        — Capture a durable, env-stamped perf baseline (doc/benchmarks/)
#   make vet             — Run go vet

.PHONY: lean-build sign-roms verify-roms test bench baseline vet

# Build the Lean proofs under lean/ (toolchain pinned by lean/lean-toolchain;
# requires elan on PATH). `lake build` alone only fails on a compile error —
# `sorry` typechecks and warns, it doesn't fail — so also grep the build
# output for sorry/sorryAx and fail CI on either (qbp-compute-unit#82 AC-c:
# closes the "proven ≠ wired" gap where lean2rom silently fell back to its
# Go-native extraction whenever `lean` wasn't on PATH).
#
# make's recipe shell is /bin/sh (dash on Ubuntu/CI), which has no
# `pipefail` — piping `lake build | tee` would make the recipe's exit
# status tee's (always 0), so a real compile error would NOT fail this
# target (caught in review, qbp-oppenheimer, live-test seq 4695: the same
# "proven ≠ wired" gap one layer down). Capture lake build's own exit
# status explicitly instead of piping through tee.
lean-build:
	@cd lean && lake build > /tmp/lake-build.log 2>&1; rc=$$?; \
	cat /tmp/lake-build.log; \
	if [ $$rc -ne 0 ]; then echo "FAIL: lake build failed (exit $$rc)"; exit $$rc; fi; \
	if grep -qE "sorryAx|declaration uses 'sorry'" /tmp/lake-build.log; then \
		echo "FAIL: sorry/sorryAx detected in Lean build output"; exit 1; \
	fi; \
	echo "lean/ builds clean: no sorry, no sorryAx"

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

# Capture a durable, environment-stamped performance baseline into doc/benchmarks/.
# Refuses on a busy box (a contended capture mis-sets the record); count defaults to 6
# for benchstat validity. Compare: benchstat <old-raw> <new-raw>.
baseline:
	./scripts/qbp-cu-baseline.sh

# Run go vet across all packages.
vet:
	cd emulator && go vet ./...
	go vet ./cmd/lean2rom
