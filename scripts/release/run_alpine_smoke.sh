#!/usr/bin/env bash
set -euo pipefail

readonly EXPECTED_IMAGE_REF="alpine:latest@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11"

usage() {
  cat <<'EOF'
usage: run_alpine_smoke.sh --image-ref <alpine:latest@sha256:<resolved-digest>>
EOF
}

image_ref=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --image-ref)
      image_ref="${2:-}"
      shift 2
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

if [[ -z "$image_ref" ]]; then
  echo "missing required argument: image_ref" >&2
  usage >&2
  exit 1
fi

if [[ "$image_ref" != "$EXPECTED_IMAGE_REF" ]]; then
  echo "alpine smoke requires exact image ref ${EXPECTED_IMAGE_REF}, got ${image_ref}" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for Alpine smoke" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

docker pull "$image_ref" >/dev/null
docker run --rm \
  -v "$repo_root:/repo" \
  -w /repo \
  "$image_ref" \
  sh -lc '
    set -euo pipefail
    export HOME=/tmp/pure-simdjson-home
    mkdir -p "$HOME"
    apk add --no-cache bash build-base clang go cargo rust
    cargo build --release
    export PURE_SIMDJSON_LIB_PATH=/repo/target/release/libpure_simdjson.so
    go run ./tests/smoke/go_bootstrap_smoke.go
  '
