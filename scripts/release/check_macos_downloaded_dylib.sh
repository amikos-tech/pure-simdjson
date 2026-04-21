#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: check_macos_downloaded_dylib.sh (--artifact <path> | --build-local) [--keep-temp]

Validate that a macOS dylib artifact can still be loaded through the repo's
native and Go smoke paths after it is copied to a fresh temp location and after
synthetic quarantine metadata is applied and removed.

Options:
  --artifact <path>  Use an existing dylib artifact
  --build-local      Build target/release/libpure_simdjson.dylib first
  --keep-temp        Preserve the temp copy and logs for inspection
EOF
}

artifact_path=""
build_local=false
keep_temp=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --artifact)
      artifact_path="${2:-}"
      shift 2
      ;;
    --build-local)
      build_local=true
      shift
      ;;
    --keep-temp)
      keep_temp=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

artifact_sources=0
if [[ -n "$artifact_path" ]]; then
  artifact_sources=$((artifact_sources + 1))
fi
if [[ "$build_local" == true ]]; then
  artifact_sources=$((artifact_sources + 1))
fi
if [[ "$artifact_sources" -ne 1 ]]; then
  echo "exactly one of --artifact or --build-local is required" >&2
  usage >&2
  exit 1
fi

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "check_macos_downloaded_dylib.sh requires macOS" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

arch="$(uname -m)"
case "$arch" in
  arm64)
    platform_tuple="darwin-arm64"
    ;;
  x86_64)
    platform_tuple="darwin-amd64"
    ;;
  *)
    echo "unsupported macOS architecture: $arch" >&2
    exit 1
    ;;
esac

if [[ "$build_local" == true ]]; then
  cargo build --release
  artifact_path="$repo_root/target/release/libpure_simdjson.dylib"
fi

if [[ ! -f "$artifact_path" ]]; then
  echo "artifact not found: $artifact_path" >&2
  exit 1
fi

temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/pure-simdjson-macos-artifact-check.XXXXXX")"
artifact_copy="$temp_dir/libpure_simdjson.dylib"
cleanup() {
  if [[ "$keep_temp" != true ]]; then
    rm -rf "$temp_dir"
  fi
}
trap cleanup EXIT

cp "$artifact_path" "$artifact_copy"
codesign -s - --force --timestamp=none "$artifact_copy" >/dev/null

overall_status=0

run_required_check() {
  local label="$1"
  shift
  printf '\n== %s ==\n' "$label"
  if "$@"; then
    printf '%s=PASS\n' "$label"
  else
    overall_status=1
    printf '%s=FAIL\n' "$label"
  fi
}

run_informational_check() {
  local label="$1"
  shift
  printf '\n== %s ==\n' "$label"
  if "$@"; then
    printf '%s=PASS\n' "$label"
  else
    printf '%s=FAIL\n' "$label"
  fi
}

go_smoke() {
  PURE_SIMDJSON_LIB_PATH="$artifact_copy" go run ./tests/smoke/go_bootstrap_smoke.go
}

write_quarantine() {
  local quarantine_value
  quarantine_value="0081;$(date +%s);pure-simdjson;$(uuidgen)"
  xattr -w com.apple.quarantine "$quarantine_value" "$artifact_copy"
  xattr -p com.apple.quarantine "$artifact_copy" >/dev/null
}

clear_quarantine() {
  xattr -d com.apple.quarantine "$artifact_copy"
  ! xattr -p com.apple.quarantine "$artifact_copy" >/dev/null 2>&1
}

printf 'repo_root=%s\n' "$repo_root"
printf 'platform_tuple=%s\n' "$platform_tuple"
printf 'source_artifact=%s\n' "$artifact_path"
printf 'artifact_copy=%s\n' "$artifact_copy"
printf 'keep_temp=%s\n' "$keep_temp"

run_required_check "baseline_codesign" codesign --verify --verbose "$artifact_copy"
run_required_check "baseline_native_smoke" bash scripts/release/run_native_smoke.sh "$artifact_copy" "$platform_tuple"
run_required_check "baseline_go_smoke" go_smoke
run_required_check "write_synthetic_quarantine" write_quarantine

printf '\n== quarantine_xattrs ==\n'
xattr -l "$artifact_copy"

run_required_check "quarantined_native_smoke" bash scripts/release/run_native_smoke.sh "$artifact_copy" "$platform_tuple"
run_required_check "quarantined_go_smoke" go_smoke
run_informational_check "quarantined_spctl_assess" spctl --assess --type execute --verbose=4 "$artifact_copy"
run_required_check "clear_quarantine" clear_quarantine
run_required_check "post_clear_native_smoke" bash scripts/release/run_native_smoke.sh "$artifact_copy" "$platform_tuple"
run_required_check "post_clear_go_smoke" go_smoke

printf '\n== summary ==\n'
if [[ "$overall_status" -eq 0 ]]; then
  echo "required load checks passed"
else
  echo "one or more required load checks failed" >&2
fi
echo "note: spctl is informational here; real downloaded-artifact Gatekeeper behavior still needs human UAT"

exit "$overall_status"
