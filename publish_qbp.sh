#!/bin/bash
set -e

echo "Initializing git repository for QBP-Compute-Unit..."
git init -b main

echo "Adding files to the repository..."
git add .

echo "Creating the initial commit..."
git commit -m "Initial commit: QBP Compute Unit

- Precision scaling architecture and BMA integration docs
- Core algebraic packages (qword, quat, octonion, fano)
- Early prototype packages (gap, persona)"

echo "Publishing to GitHub..."
gh repo create JamesPagetButler/qbp-compute-unit --private --source=. --remote=origin --push

echo "Successfully published QBP-Compute-Unit!"
