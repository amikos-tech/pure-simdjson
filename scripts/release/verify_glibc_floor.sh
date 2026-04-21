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
  trap 'rm -rf "$tmp_dir"' EXIT

  local objdump_output="$tmp_dir/objdump.txt"
  local observed_symbols="$tmp_dir/observed-symbols.txt"
  local pure_symbols="$tmp_dir/pure-symbols.txt"
  local expected_symbols="$tmp_dir/expected-symbols.txt"
  local missing_symbols="$tmp_dir/missing-symbols.txt"
  local unexpected_symbols="$tmp_dir/unexpected-symbols.txt"

  objdump -T "$library_path" >"$objdump_output"
  nm -D --defined-only "$library_path" | awk '{print $NF}' | sed 's/@.*$//' | sort -u >"$observed_symbols"
  grep '^pure_simdjson_' "$observed_symbols" >"$pure_symbols" || true
  grep -oE 'pure_simdjson_[A-Za-z0-9_]+' include/pure_simdjson.h | sort -u >"$expected_symbols"

  grep -v '^pure_simdjson_' "$observed_symbols" >"$unexpected_symbols" || true
  comm -23 "$expected_symbols" "$pure_symbols" >"$missing_symbols" || true
  comm -13 "$expected_symbols" "$pure_symbols" >>"$unexpected_symbols" || true

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
    fail "observed export surface includes symbols outside the expected pure_simdjson_ ABI"
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
