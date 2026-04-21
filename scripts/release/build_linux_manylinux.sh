#!/usr/bin/env bash
set -euo pipefail

readonly LINUX_AMD64_PLATFORM="linux/amd64"
readonly LINUX_AMD64_TARGET="x86_64-unknown-linux-gnu"
readonly LINUX_AMD64_IMAGE="quay.io/pypa/manylinux2014_x86_64@sha256:96412a3110ba598851ba1cd9bfa66b74c2903bfec1af978f5e55def5f0f1912c"

readonly LINUX_ARM64_PLATFORM="linux/arm64"
readonly LINUX_ARM64_TARGET="aarch64-unknown-linux-gnu"
readonly LINUX_ARM64_IMAGE="quay.io/pypa/manylinux2014_aarch64@sha256:2cfb8a1feca0f640b26689e27aadff0a8ff367243d2672a31207d075318d26c7"

usage() {
  cat <<'EOF'
usage:
  build_linux_manylinux.sh prove-pagesize \
    --linux-platform linux/arm64 \
    --output linux-arm64-pagesize.txt

  build_linux_manylinux.sh build \
    --linux-platform <linux/amd64|linux/arm64> \
    --rust-target <target-triple> \
    --manylinux-image <image-ref> \
    --target-dir <dir>
EOF
}

fail() {
  echo "build_linux_manylinux.sh: $*" >&2
  exit 1
}

resolve_expected_tuple() {
  expected_target=""
  expected_image=""

  case "$1" in
    "$LINUX_AMD64_PLATFORM")
      expected_target="$LINUX_AMD64_TARGET"
      expected_image="$LINUX_AMD64_IMAGE"
      ;;
    "$LINUX_ARM64_PLATFORM")
      expected_target="$LINUX_ARM64_TARGET"
      expected_image="$LINUX_ARM64_IMAGE"
      ;;
    *)
      fail "unsupported linux platform tuple: $1"
      ;;
  esac
}

record_arm64_pagesize_proof() {
  local output_path="$1"

  PAGE_SIZE="$(getconf PAGE_SIZE 2>/dev/null || pagesize)"
  echo "observed linux/arm64 PAGE_SIZE=${PAGE_SIZE}"

  if [[ -n "$output_path" ]]; then
    mkdir -p "$(dirname "$output_path")"
    printf 'PAGE_SIZE=%s\n' "$PAGE_SIZE" >"$output_path"
  fi

  if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
    printf 'linux/arm64 PAGE_SIZE=%s\n' "$PAGE_SIZE" >>"$GITHUB_STEP_SUMMARY"
  fi

  if [[ "$PAGE_SIZE" != "4096" ]]; then
    fail "linux/arm64 release artifacts require a 4096-byte page size, observed ${PAGE_SIZE}"
  fi
}

prove_pagesize_main() {
  local linux_platform=""
  local output_path=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --linux-platform)
        linux_platform="${2:-}"
        shift 2
        ;;
      --output)
        output_path="${2:-}"
        shift 2
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        fail "unknown prove-pagesize argument: $1"
        ;;
    esac
  done

  [[ -n "$linux_platform" ]] || fail "missing --linux-platform"
  [[ -n "$output_path" ]] || fail "missing --output"
  [[ "$linux_platform" == "$LINUX_ARM64_PLATFORM" ]] || fail "prove-pagesize only supports ${LINUX_ARM64_PLATFORM}"

  record_arm64_pagesize_proof "$output_path"
}

build_main() {
  local linux_platform=""
  local rust_target=""
  local manylinux_image=""
  local target_dir=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --linux-platform)
        linux_platform="${2:-}"
        shift 2
        ;;
      --rust-target)
        rust_target="${2:-}"
        shift 2
        ;;
      --manylinux-image)
        manylinux_image="${2:-}"
        shift 2
        ;;
      --target-dir)
        target_dir="${2:-}"
        shift 2
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        fail "unknown build argument: $1"
        ;;
    esac
  done

  [[ -n "$linux_platform" ]] || fail "missing --linux-platform"
  [[ -n "$rust_target" ]] || fail "missing --rust-target"
  [[ -n "$manylinux_image" ]] || fail "missing --manylinux-image"
  [[ -n "$target_dir" ]] || fail "missing --target-dir"

  resolve_expected_tuple "$linux_platform"
  [[ "$rust_target" == "$expected_target" ]] || fail "expected rust target ${expected_target} for ${linux_platform}, got ${rust_target}"
  [[ "$manylinux_image" == "$expected_image" ]] || fail "expected manylinux image ${expected_image} for ${linux_platform}, got ${manylinux_image}"

  if [[ "$linux_platform" == "$LINUX_ARM64_PLATFORM" ]]; then
    record_arm64_pagesize_proof "${PURE_SIMDJSON_PAGESIZE_PROOF_OUTPUT:-}"
  fi

  command -v docker >/dev/null 2>&1 || fail "docker is required for manylinux builds"

  local workspace="${GITHUB_WORKSPACE:?GITHUB_WORKSPACE is required}"
  local home_dir="${HOME:?HOME is required}"
  local cargo_home="${CARGO_HOME:-$home_dir/.cargo}"
  local rustup_home="${RUSTUP_HOME:-$home_dir/.rustup}"

  [[ -n "${SOURCE_DATE_EPOCH:-}" ]] || fail "SOURCE_DATE_EPOCH is required"
  [[ -n "${RUSTFLAGS:-}" ]] || fail "RUSTFLAGS is required"

  mkdir -p "$cargo_home" "$rustup_home" "$target_dir"

  docker run --rm \
    --user "$(id -u):$(id -g)" \
    -v "$workspace:$workspace" \
    -v "$target_dir:$target_dir" \
    -v "$cargo_home:$cargo_home" \
    -v "$rustup_home:$rustup_home" \
    -w "$workspace" \
    -e HOME="$home_dir" \
    -e CARGO_HOME="$cargo_home" \
    -e RUSTUP_HOME="$rustup_home" \
    -e PATH="$cargo_home/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
    -e CARGO_INCREMENTAL="${CARGO_INCREMENTAL:-0}" \
    -e SOURCE_DATE_EPOCH="$SOURCE_DATE_EPOCH" \
    -e RUSTFLAGS="$RUSTFLAGS" \
    -e CARGO_TARGET_DIR="$target_dir" \
    -e PSDJ_RUST_TARGET="$rust_target" \
    "$manylinux_image" \
    bash -lc 'set -euo pipefail; rustup show active-toolchain >/dev/null; cargo build --release --locked --target "$PSDJ_RUST_TARGET"'
}

main() {
  local command="${1:-}"
  if [[ -z "$command" ]]; then
    usage >&2
    exit 1
  fi
  shift

  case "$command" in
    prove-pagesize)
      prove_pagesize_main "$@"
      ;;
    build)
      build_main "$@"
      ;;
    -h|--help)
      usage
      ;;
    *)
      fail "unknown command: ${command}"
      ;;
  esac
}

main "$@"
