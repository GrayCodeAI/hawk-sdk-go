#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

violations="$(
  grep -RInE --include='*.go' \
    'github\.com/GrayCodeAI/(eyrie|inspect|sight|tok|trace|yaad)(/|")|github\.com/GrayCodeAI/hawk/(internal/|shared/types)' \
    . || true
)"

if [[ -n "${violations}" ]]; then
  echo "forbidden Hawk consumer imports found:"
  echo "${violations}"
  echo
  echo "hawk-sdk-go must depend on Hawk public APIs/contracts only; do not import support engines, hawk/internal, or removed hawk/shared/types"
  exit 1
fi

echo "consumer boundary guard passed"
