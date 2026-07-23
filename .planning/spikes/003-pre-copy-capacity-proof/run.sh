#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
build_dir=$(mktemp -d "${TMPDIR:-/tmp}/pure-simdjson-spike-003.XXXXXX")
trap 'rm -rf "$build_dir"' EXIT

rustc --edition=2021 -D warnings --test "$script_dir/probe.rs" \
  -o "$build_dir/probe-tests"
"$build_dir/probe-tests" --test-threads=1

rustc --edition=2021 -D warnings -O "$script_dir/probe.rs" \
  -o "$build_dir/probe"
"$build_dir/probe"
