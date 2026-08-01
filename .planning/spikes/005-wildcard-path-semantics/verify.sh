#!/usr/bin/env bash
# Spike 005: Wildcard path semantics verifier.
#
# Builds probe.cpp against the untouched vendored simdjson singleheader in a
# mktemp scratch dir with ASan+UBSan, runs it 3x, and fails if any run traps,
# the three runs are not byte-identical (determinism check per spike
# CONVENTIONS.md), or the output drifts from the pinned expected.txt.
#
# Path resolution lives in dom/*-inl.h and is kernel-independent, so unlike
# spike 004 this probe does not iterate implementations -- it records which
# implementation was active for the record and pins semantics once.
#
# Does not modify production source, the vendored gitlink, or any pinned
# version -- only reads third_party/simdjson/singleheader/*.
#
# Regenerate the golden file deliberately with: ./verify.sh --update

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SPIKE_DIR="$REPO_ROOT/.planning/spikes/005-wildcard-path-semantics"
SINGLEHEADER="$REPO_ROOT/third_party/simdjson/singleheader"
EXPECTED="$SPIKE_DIR/expected.txt"

UPDATE=0
if [[ "${1:-}" == "--update" ]]; then UPDATE=1; fi

SCRATCH="$(mktemp -d)"
trap 'rm -rf "$SCRATCH"' EXIT

cp "$SINGLEHEADER/simdjson.h" "$SINGLEHEADER/simdjson.cpp" "$SCRATCH/"
cp "$SPIKE_DIR/probe.cpp" "$SCRATCH/"

CXX="${CXX:-clang++}"
CXXFLAGS=(-std=c++17 -O1 -g -fsanitize=address,undefined -fno-omit-frame-pointer
  -I"$SCRATCH")

echo "==> Compiling vendored simdjson.cpp (ASan+UBSan)"
"$CXX" "${CXXFLAGS[@]}" -c "$SCRATCH/simdjson.cpp" -o "$SCRATCH/simdjson.o"

echo "==> Compiling probe.cpp"
"$CXX" "${CXXFLAGS[@]}" -c "$SCRATCH/probe.cpp" -o "$SCRATCH/probe.o"

echo "==> Linking"
"$CXX" "${CXXFLAGS[@]}" "$SCRATCH/probe.o" "$SCRATCH/simdjson.o" -o "$SCRATCH/probe"

echo "==> Running 3x for determinism"
declare -a digests=()
for i in 1 2 3; do
  if ! "$SCRATCH/probe" > "$SCRATCH/run_$i.out" 2> "$SCRATCH/run_$i.err"; then
    echo "FAIL: run $i exited non-zero (ASan/UBSan trap or fixture/parse failure)" >&2
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

if [[ "$UPDATE" == "1" ]]; then
  cp "$SCRATCH/run_1.out" "$EXPECTED"
  echo "==> UPDATED golden file: $EXPECTED"
  exit 0
fi

if [[ ! -f "$EXPECTED" ]]; then
  echo "FAIL: $EXPECTED missing. Generate it with: ./verify.sh --update" >&2
  exit 1
fi

# The active implementation varies by host and is recorded, not contracted.
if ! diff -u <(grep -v '^IMPL ' "$EXPECTED") <(grep -v '^IMPL ' "$SCRATCH/run_1.out"); then
  echo "FAIL: wildcard path semantics drifted from pinned expected.txt" >&2
  exit 1
fi

cases="$(grep -c '^CASE ' "$SCRATCH/run_1.out")"
echo "==> VERIFIED: $cases cases match pinned semantics, deterministic across 3 runs"
