#!/usr/bin/env bash
# Spike 004: Minify buffer safety verifier.
#
# Builds probe.cpp against the untouched vendored simdjson singleheader in a
# mktemp scratch dir, compiled with ASan+UBSan and the same
# SIMDJSON_IMPLEMENTATION_FALLBACK=1 macro build.rs uses in production, runs
# it 3x, and fails if any run traps, fails, or the three runs are not
# byte-identical (determinism check per spike CONVENTIONS.md).
#
# Does not modify production source, the vendored gitlink, or any pinned
# version — only reads third_party/simdjson/singleheader/*.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SPIKE_DIR="$REPO_ROOT/.planning/spikes/004-minify-buffer-safety"
SINGLEHEADER="$REPO_ROOT/third_party/simdjson/singleheader"

SCRATCH="$(mktemp -d)"
trap 'rm -rf "$SCRATCH"' EXIT

cp "$SINGLEHEADER/simdjson.h" "$SINGLEHEADER/simdjson.cpp" "$SCRATCH/"
cp "$SPIKE_DIR/probe.cpp" "$SCRATCH/"

CXX="${CXX:-clang++}"
CXXFLAGS=(-std=c++17 -O1 -g -fsanitize=address,undefined -fno-omit-frame-pointer
  -DSIMDJSON_IMPLEMENTATION_FALLBACK=1 -I"$SCRATCH")

echo "==> Compiling vendored simdjson.cpp (ASan+UBSan, fallback kernel enabled)"
"$CXX" "${CXXFLAGS[@]}" -c "$SCRATCH/simdjson.cpp" -o "$SCRATCH/simdjson.o"

echo "==> Compiling probe.cpp"
"$CXX" "${CXXFLAGS[@]}" -c "$SCRATCH/probe.cpp" -o "$SCRATCH/probe.o"

echo "==> Linking"
"$CXX" "${CXXFLAGS[@]}" "$SCRATCH/probe.o" "$SCRATCH/simdjson.o" -o "$SCRATCH/probe"

echo "==> Running 3x for determinism"
declare -a digests=()
for i in 1 2 3; do
  if ! "$SCRATCH/probe" > "$SCRATCH/run_$i.out" 2> "$SCRATCH/run_$i.err"; then
    echo "FAIL: run $i exited non-zero (ASan/UBSan trap or logical mismatch)" >&2
    cat "$SCRATCH/run_$i.err" >&2
    exit 1
  fi
  digest="$(shasum -a 256 "$SCRATCH/run_$i.out" | awk '{print $1}')"
  digests+=("$digest")
  echo "  run $i: exit=0 sha256=$digest"
done

if [[ "${digests[0]}" != "${digests[1]}" || "${digests[1]}" != "${digests[2]}" ]]; then
  echo "FAIL: non-deterministic output across 3 runs" >&2
  exit 1
fi

if ! grep -q "^SUMMARY total=24 any_failure=0$" "$SCRATCH/run_1.out"; then
  echo "FAIL: expected SUMMARY total=24 any_failure=0, got:" >&2
  tail -1 "$SCRATCH/run_1.out" >&2
  exit 1
fi

echo "==> VERIFIED: 24/24 cases safe, correct, and deterministic across 3 runs"
