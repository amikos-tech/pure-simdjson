#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: run_go_packaged_smoke.sh --staged-root <dir> --version <semver-without-v>
EOF
}

staged_root=""
version=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --staged-root)
      staged_root="${2:-}"
      shift 2
      ;;
    --version)
      version="${2:-}"
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

for required in staged_root version; do
  if [[ -z "${!required}" ]]; then
    echo "missing required argument: $required" >&2
    usage >&2
    exit 1
  fi
done

if [[ ! -d "$staged_root" ]]; then
  echo "staged root not found: $staged_root" >&2
  exit 1
fi
if [[ ! -d "$staged_root/v${version}" ]]; then
  echo "staged root is missing v${version}: $staged_root" >&2
  exit 1
fi

grep -q 'PURE_SIMDJSON_BINARY_MIRROR' internal/bootstrap/bootstrap.go
grep -q 'PURE_SIMDJSON_DISABLE_GH_FALLBACK' internal/bootstrap/bootstrap.go

server_root="$(mktemp -d "${TMPDIR:-/tmp}/pure-simdjson-bootstrap-server.XXXXXX")"
cache_dir="$(mktemp -d "${TMPDIR:-/tmp}/pure-simdjson-bootstrap-cache.XXXXXX")"
http_log="$(mktemp "${TMPDIR:-/tmp}/pure-simdjson-bootstrap-http.XXXXXX.log")"
ln -s "$staged_root" "$server_root/pure-simdjson"

cleanup() {
  if [[ -n "${server_pid:-}" ]]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$server_root" "$cache_dir"
  rm -f "$http_log"
}
trap cleanup EXIT

python3 -u -m http.server 0 --bind 127.0.0.1 --directory "$server_root" >"$http_log" 2>&1 &
server_pid=$!

port=""
for _ in $(seq 1 50); do
  port="$(sed -n 's|^Serving HTTP on 127\.0\.0\.1 port \([0-9][0-9]*\).*|\1|p' "$http_log" | head -n1)"
  [[ -n "$port" ]] && break
  sleep 0.2
done

if [[ -z "$port" ]]; then
  echo "failed to detect http.server port; log follows:" >&2
  cat "$http_log" >&2
  exit 1
fi

for _ in $(seq 1 50); do
  if python3 - "$port" <<'PY'
import sys
import urllib.request

port = sys.argv[1]
with urllib.request.urlopen(f"http://127.0.0.1:{port}/pure-simdjson/") as response:
    if response.status != 200:
        raise SystemExit(1)
PY
  then
    break
  fi
  sleep 0.2
done

if [[ -n "${PURE_SIMDJSON_LIB_PATH:-}" ]]; then
  echo "PURE_SIMDJSON_LIB_PATH must stay unset for packaged-artifact smoke" >&2
  exit 1
fi

export PURE_SIMDJSON_BINARY_MIRROR="http://127.0.0.1:${port}/pure-simdjson"
export PURE_SIMDJSON_DISABLE_GH_FALLBACK=1
export PURE_SIMDJSON_CACHE_DIR="$cache_dir"

go run ./tests/smoke/go_bootstrap_smoke.go
