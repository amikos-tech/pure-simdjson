#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: verify_glibc_floor.sh <path-to-linux-shared-object>
EOF
}

fail() {
  echo "verify_glibc_floor.sh: $*" >&2
  exit 1
}

write_expected_exports() {
  local header_path="$1"
  local repo_root
  repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

  python3 - "$repo_root" "$header_path" <<'PY'
import importlib.util
import pathlib
import sys

repo_root = pathlib.Path(sys.argv[1])
header_path = pathlib.Path(sys.argv[2])
check_header_path = repo_root / "tests" / "abi" / "check_header.py"

spec = importlib.util.spec_from_file_location("check_header", check_header_path)
if spec is None or spec.loader is None:
    raise SystemExit(f"failed to load ABI header parser: {check_header_path}")

check_header = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check_header)

prototypes = check_header.parse_prototypes(header_path.read_text(encoding="utf-8"))
for name in sorted(name for name in prototypes if name.startswith("pure_simdjson_")):
    print(name)
PY
}

version_gt() {
  local left="$1"
  local right="$2"
  [[ "$(printf '%s\n%s\n' "$right" "$left" | sort -V | tail -n1)" == "$left" && "$left" != "$right" ]]
}

main() {
  if [[ $# -ne 1 ]]; then
    usage >&2
    exit 1
  fi

  local library_path="$1"
  [[ -f "$library_path" ]] || fail "library not found: $library_path"

  local tmp_dir
  tmp_dir="$(mktemp -d)"
  trap "rm -rf -- '$tmp_dir'" EXIT

  local objdump_output="$tmp_dir/objdump.txt"
  local observed_symbols="$tmp_dir/observed-symbols.txt"
  local pure_symbols="$tmp_dir/pure-symbols.txt"
  local expected_symbols="$tmp_dir/expected-symbols.txt"
  local allowed_symbols="$tmp_dir/allowed-symbols.txt"
  local missing_symbols="$tmp_dir/missing-symbols.txt"
  local unexpected_symbols="$tmp_dir/unexpected-symbols.txt"
  local header_path
  header_path="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/include/pure_simdjson.h"

  objdump -T "$library_path" >"$objdump_output"
  nm -D --defined-only "$library_path" | awk '{print $NF}' | sed 's/@.*$//' | sort -u >"$observed_symbols"
  grep '^pure_simdjson_' "$observed_symbols" >"$pure_symbols" || true
  write_expected_exports "$header_path" >"$expected_symbols"

  {
    cat "$expected_symbols"
    printf '%s\n' \
      psdj_internal_materialize_build \
      psdj_internal_test_hold_materialize_guard
  } | sort -u >"$allowed_symbols"

  comm -23 "$expected_symbols" "$pure_symbols" >"$missing_symbols" || true
  comm -23 "$observed_symbols" "$allowed_symbols" >"$unexpected_symbols" || true

  if [[ ! -s "$pure_symbols" ]]; then
    fail "nm -D --defined-only reported no pure_simdjson_ exports for $library_path"
  fi

  if [[ -s "$missing_symbols" ]]; then
    echo "missing exports for $library_path:" >&2
    cat "$missing_symbols" >&2
    fail "observed export surface is missing expected pure_simdjson_ symbols"
  fi

  if [[ -s "$unexpected_symbols" ]]; then
    echo "unexpected exports for $library_path:" >&2
    cat "$unexpected_symbols" >&2
    fail "observed export surface includes symbols outside the expected release ABI export set"
  fi

  mapfile -t glibc_versions < <(grep -oE 'GLIBC_[0-9]+(\.[0-9]+)*' "$objdump_output" | sed 's/^GLIBC_//' | sort -uV || true)

  local highest_glibc="none"
  if [[ ${#glibc_versions[@]} -gt 0 ]]; then
    highest_glibc="${glibc_versions[$((${#glibc_versions[@]} - 1))]}"
  fi

  echo "verify_glibc_floor.sh inspected: $library_path"
  echo "highest observed GLIBC symbol version: $highest_glibc"

  if [[ "$highest_glibc" != "none" ]] && version_gt "$highest_glibc" "2.17"; then
    fail "glibc floor exceeded for $library_path: observed GLIBC_${highest_glibc}, allowed up to GLIBC_2.17"
  fi
}

main "$@"
