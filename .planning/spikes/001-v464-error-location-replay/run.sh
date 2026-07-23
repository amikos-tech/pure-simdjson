#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
repo_root="$(git -C "$script_dir" rev-parse --show-toplevel)"
simdjson_repo="$repo_root/third_party/simdjson"
expected_commit="1bcf71bd85059ab6574ea1159de9298dcc1212c5"

if ! actual_commit="$(git -C "$simdjson_repo" rev-parse 'v4.6.4^{commit}' 2>/dev/null)"; then
  echo "missing simdjson v4.6.4 object; run:" >&2
  echo "  git -C third_party/simdjson fetch --no-tags origin refs/tags/v4.6.4:refs/tags/v4.6.4" >&2
  exit 1
fi

if [[ "$actual_commit" != "$expected_commit" ]]; then
  echo "simdjson v4.6.4 resolved to $actual_commit, expected $expected_commit" >&2
  exit 1
fi

build_dir="$(mktemp -d "${TMPDIR:-/tmp}/pure-simdjson-spike-001.XXXXXX")"
trap 'rm -rf "$build_dir"' EXIT

git -C "$simdjson_repo" archive v4.6.4 \
  singleheader/simdjson.h singleheader/simdjson.cpp |
  tar -x -C "$build_dir"

"${CXX:-c++}" -std=c++17 -O2 -DNDEBUG \
  -I"$build_dir/singleheader" \
  "$script_dir/probe.cpp" \
  "$build_dir/singleheader/simdjson.cpp" \
  -o "$build_dir/probe"

"$build_dir/probe"
