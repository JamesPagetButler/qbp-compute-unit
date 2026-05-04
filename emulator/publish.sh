#!/bin/bash
set -e

echo "Initializing git repository..."
git init -b main

echo "Adding files to the repository..."
git add .

echo "Creating the initial commit..."
git commit -m "Initial commit: AVX1+FMA3 SIMD implementations with CPUID dispatch and scalar fallbacks

- Added zero-dependency CPUID assembly stub (cpuid_amd64.s)
- Implemented AVX1-clean quaternion math (qmath_amd64.s)
- Implemented scalar fallbacks (qmath_scalar.go)
- Added CPU-feature dispatch wrappers
- Added test coverage for equivalence and fallback paths (isa_test.go, qmath_dispatch_amd64_test.go)"

echo "Publishing to GitHub..."
gh repo create JamesPagetButler/qbp-emulator --private --source=. --remote=origin --push

echo "Successfully published!"
