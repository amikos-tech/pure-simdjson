#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
probe_source="$repo_root/tests/native/minify_buffer_safety_probe.cpp"
singleheader_dir="$repo_root/third_party/simdjson/singleheader"
scratch_dir="$(mktemp -d "${TMPDIR:-/tmp}/pure-simdjson-minify-buffer-safety.XXXXXX")"
trap 'rm -rf "$scratch_dir"' EXIT

cxx="${CXX:-clang++}"
compile_flags=(
  -std=c++17
  -O1
  -g
  -fsanitize=address,undefined
  -fno-sanitize-recover=all
  -fno-omit-frame-pointer
  -DSIMDJSON_IMPLEMENTATION_FALLBACK=1
  -I"$singleheader_dir"
)

"$cxx" "${compile_flags[@]}" -c "$singleheader_dir/simdjson.cpp" \
  -o "$scratch_dir/simdjson.o"
"$cxx" "${compile_flags[@]}" -c "$probe_source" \
  -o "$scratch_dir/probe.o"
"$cxx" "${compile_flags[@]}" "$scratch_dir/probe.o" \
  "$scratch_dir/simdjson.o" -o "$scratch_dir/probe"

summary_pattern='^SUMMARY kernels=([^[:space:]]+) cases_per_kernel=([0-9]+) total=([0-9]+) failures=([0-9]+)$'
expected_cases_per_kernel=12
baseline_kernels=""
baseline_total=""

for run_number in 1 2 3; do
  stdout_file="$scratch_dir/run_${run_number}.out"
  stderr_file="$scratch_dir/run_${run_number}.err"

  if ! ASAN_OPTIONS=halt_on_error=1 \
    UBSAN_OPTIONS=halt_on_error=1:print_stacktrace=1 \
    "$scratch_dir/probe" >"$stdout_file" 2>"$stderr_file"; then
    echo "minify buffer-safety probe failed on run $run_number" >&2
    sed -n '1,200p' "$stderr_file" >&2
    exit 1
  fi
  if [[ -s "$stderr_file" ]]; then
    echo "minify buffer-safety probe wrote unexpected diagnostics on run $run_number" >&2
    sed -n '1,200p' "$stderr_file" >&2
    exit 1
  fi

  summary_count="$(grep -c '^SUMMARY ' "$stdout_file" || true)"
  if [[ "$summary_count" != "1" ]]; then
    echo "expected exactly one machine-readable SUMMARY on run $run_number" >&2
    exit 1
  fi
  summary_line="$(grep '^SUMMARY ' "$stdout_file")"
  if [[ "$summary_line" =~ $summary_pattern ]]; then
    kernel_csv="${BASH_REMATCH[1]}"
    cases_per_kernel="${BASH_REMATCH[2]}"
    actual_total="${BASH_REMATCH[3]}"
    failures="${BASH_REMATCH[4]}"
  else
    echo "malformed minify buffer-safety summary on run $run_number: $summary_line" >&2
    exit 1
  fi

  if [[ "$kernel_csv" == "none" ]]; then
    echo "probe reported no supported simdjson kernels" >&2
    exit 1
  fi
  IFS=',' read -r -a supported_kernels <<<"$kernel_csv"
  kernel_count="${#supported_kernels[@]}"
  expected_total=$((expected_cases_per_kernel * kernel_count))

  if [[ "$cases_per_kernel" != "$expected_cases_per_kernel" ]]; then
    echo "expected $expected_cases_per_kernel cases per kernel, got $cases_per_kernel" >&2
    exit 1
  fi
  if [[ "$actual_total" -ne "$expected_total" ]]; then
    echo "expected $expected_total total cases from $kernel_count kernels, got $actual_total" >&2
    exit 1
  fi
  if [[ "$failures" != "0" ]]; then
    echo "probe reported failures: $summary_line" >&2
    exit 1
  fi

  case "$(uname -s):$(uname -m)" in
    Linux:x86_64|Linux:amd64)
      if [[ ",${kernel_csv}," != *",fallback,"* ]]; then
        echo "linux x86-64 probe must include the fallback kernel: $kernel_csv" >&2
        exit 1
      fi
      if [[ ",${kernel_csv}," != *",haswell,"* && ",${kernel_csv}," != *",westmere,"* ]]; then
        echo "linux x86-64 probe must include haswell or westmere: $kernel_csv" >&2
        exit 1
      fi
      ;;
  esac

  if [[ "$run_number" == "1" ]]; then
    baseline_kernels="$kernel_csv"
    baseline_total="$actual_total"
  else
    if [[ "$kernel_csv" != "$baseline_kernels" || "$actual_total" != "$baseline_total" ]]; then
      echo "supported kernel set or case total changed across runs" >&2
      exit 1
    fi
    if ! cmp -s "$scratch_dir/run_1.out" "$stdout_file"; then
      echo "probe output changed across runs" >&2
      diff -u "$scratch_dir/run_1.out" "$stdout_file" >&2 || true
      exit 1
    fi
  fi
done

echo "minify buffer-safety verified: kernels=$baseline_kernels total=$baseline_total runs=3"
