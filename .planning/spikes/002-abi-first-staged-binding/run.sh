#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(git -C "$script_dir" rev-parse --show-toplevel)
build_dir=$(mktemp -d "${TMPDIR:-/tmp}/pure-simdjson-spike-002.XXXXXX")
trap 'rm -rf "$build_dir"' EXIT

case "$(uname -s)" in
  Darwin)
    library_flag=(-dynamiclib)
    extension=".dylib"
    ;;
  Linux)
    library_flag=(-shared)
    extension=".so"
    ;;
  *)
    echo "unsupported platform: $(uname -s)" >&2
    exit 1
    ;;
esac

for fixture in abi11 abi12_complete abi12_missing; do
  cc "${library_flag[@]}" -fPIC -fvisibility=hidden -Wall -Wextra -Werror \
    "$script_dir/fixtures/$fixture.c" \
    -o "$build_dir/lib$fixture$extension"
done

cd "$repo_root"
CGO_ENABLED=0 go run "$script_dir/main.go" \
  "$build_dir/libabi11$extension" \
  "$build_dir/libabi12_complete$extension" \
  "$build_dir/libabi12_missing$extension"
