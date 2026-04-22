#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: run_public_bootstrap_smoke.sh --repo-root <path> --version <semver-without-v> --goos <linux|darwin|windows> --goarch <amd64|arm64> --mode <r2|github-fallback> --cache-dir <path>
EOF
}

repo_root=""
version=""
goos=""
goarch=""
mode=""
cache_dir=""
requested_cache_dir=""

slashify_path() {
  printf '%s\n' "${1//\\//}"
}

trim_trailing_slash() {
  local value="$1"
  if [[ "$value" == "/" || "$value" =~ ^[A-Za-z]:/$ ]]; then
    printf '%s\n' "$value"
    return
  fi
  printf '%s\n' "${value%/}"
}

normalize_existing_dir() {
  local raw
  raw="$(slashify_path "$1")"
  if command -v cygpath >/dev/null 2>&1; then
    cygpath -am "$raw"
    return
  fi
  (cd "$raw" 2>/dev/null && pwd -P)
}

normalize_path() {
  local raw dir base

  raw="$(slashify_path "$1")"
  if [[ -z "$raw" ]]; then
    echo "path must not be empty" >&2
    return 1
  fi
  if command -v cygpath >/dev/null 2>&1; then
    cygpath -am "$raw"
    return
  fi
  if [[ "$raw" == "/" || "$raw" =~ ^[A-Za-z]:/$ ]]; then
    printf '%s\n' "$raw"
    return
  fi

  dir="$(dirname "$raw")"
  base="$(basename "$raw")"
  if [[ -z "$dir" || "$dir" == "." ]]; then
    dir="$PWD"
  fi
  dir="$(normalize_existing_dir "$dir")" || {
    echo "failed to resolve absolute path: $1" >&2
    return 1
  }
  dir="$(trim_trailing_slash "$dir")"
  if [[ "$dir" == "/" ]]; then
    printf '/%s\n' "$base"
    return
  fi
  printf '%s/%s\n' "$dir" "$base"
}

canonical_path() {
  trim_trailing_slash "$(normalize_path "$1")"
}

refuse_unsafe_cache_dir() {
  local raw="$1"
  local resolved="$2"
  local resolved_repo_root="$3"
  local resolved_home="$4"
  local raw_slash

  raw_slash="$(slashify_path "$raw")"
  case "$raw_slash" in
    ""|"."|".."|"/")
      echo "refusing unsafe cache dir: $raw" >&2
      return 1
      ;;
  esac

  if [[ "$raw_slash" =~ ^[A-Za-z]:/?$ || "$resolved" =~ ^[A-Za-z]:/$ || "$resolved" == "/" ]]; then
    echo "refusing unsafe cache dir: $raw" >&2
    return 1
  fi
  if [[ "$resolved" == "$resolved_repo_root" || "$resolved" == "$resolved_home" ]]; then
    echo "refusing unsafe cache dir: $raw" >&2
    return 1
  fi
}

stat_mode() {
  local target="$1"
  if stat -c '%a' "$target" >/dev/null 2>&1; then
    stat -c '%a' "$target"
    return
  fi
  if stat -f '%Lp' "$target" >/dev/null 2>&1; then
    stat -f '%Lp' "$target"
    return
  fi
  echo "stat_mode: neither GNU nor BSD stat is available" >&2
  return 1
}

libname_for_goos() {
  case "$1" in
    linux)
      printf '%s\n' "libpure_simdjson.so"
      ;;
    darwin)
      printf '%s\n' "libpure_simdjson.dylib"
      ;;
    windows)
      printf '%s\n' "pure_simdjson-msvc.dll"
      ;;
    *)
      echo "unsupported goos: $1" >&2
      return 1
      ;;
  esac
}

probe_http_code() {
  curl -sS -o /dev/null -w '%{http_code}' "$1"
}

append_step_summary() {
  if [[ -z "${GITHUB_STEP_SUMMARY:-}" ]]; then
    return
  fi

  {
    echo "## public bootstrap validation"
    echo
    echo "- mode: \`${mode}\`"
    echo "- target: \`${goos}/${goarch}\`"
    echo "- expected cache path: \`${expected_cache_path}\`"
    echo "- GitHub fallback enabled: \`${github_fallback_enabled}\`"
    if [[ "$mode" == "github-fallback" ]]; then
      echo "- broken-mirror checksum probe: \`${broken_checksums_status}\` (\`${broken_checksums_url}\`)"
      echo "- broken-mirror artifact probe: \`${broken_artifact_status}\` (\`${broken_artifact_url}\`)"
    fi
    if [[ "$goos" != "windows" ]]; then
      echo "- observed unix perms:"
      echo
      echo "| path | mode |"
      echo "| --- | --- |"
      for entry in "${observed_unix_perms[@]}"; do
        local perm_path="${entry%%=*}"
        local perm_mode="${entry#*=}"
        echo "| \`${perm_path}\` | \`${perm_mode}\` |"
      done
    fi
    echo
  } >>"$GITHUB_STEP_SUMMARY"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo-root)
      repo_root="${2:-}"
      shift 2
      ;;
    --version)
      version="${2:-}"
      shift 2
      ;;
    --goos)
      goos="${2:-}"
      shift 2
      ;;
    --goarch)
      goarch="${2:-}"
      shift 2
      ;;
    --mode)
      mode="${2:-}"
      shift 2
      ;;
    --cache-dir)
      cache_dir="${2:-}"
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

for required in repo_root version goos goarch mode cache_dir; do
  if [[ -z "${!required}" ]]; then
    echo "missing required argument: $required" >&2
    usage >&2
    exit 1
  fi
done

case "$goarch" in
  amd64|arm64)
    ;;
  *)
    echo "unsupported goarch: $goarch" >&2
    exit 1
    ;;
esac

repo_root="$(trim_trailing_slash "$(normalize_existing_dir "$repo_root")")" || {
  echo "repo root not found: $repo_root" >&2
  exit 1
}
if [[ ! -d "$repo_root" ]]; then
  echo "repo root not found: $repo_root" >&2
  exit 1
fi
requested_cache_dir="$cache_dir"
cache_dir="$(canonical_path "$cache_dir")"
home_dir="$(trim_trailing_slash "$(normalize_existing_dir "${HOME:-/}")")" || {
  echo "failed to resolve HOME directory" >&2
  exit 1
}
libname="$(libname_for_goos "$goos")"

if [[ ! -f "$repo_root/tests/smoke/go_bootstrap_smoke.go" ]]; then
  echo "target smoke file not found under repo root: $repo_root/tests/smoke/go_bootstrap_smoke.go" >&2
  exit 1
fi
if [[ -n "${PURE_SIMDJSON_LIB_PATH:-}" ]]; then
  echo "PURE_SIMDJSON_LIB_PATH must stay unset for public bootstrap validation" >&2
  exit 1
fi

refuse_unsafe_cache_dir "$requested_cache_dir" "$cache_dir" "$repo_root" "$home_dir"
rm -rf -- "$cache_dir"

github_fallback_enabled="false"
broken_checksums_url=""
broken_artifact_url=""
broken_checksums_status="n/a"
broken_artifact_status="n/a"
observed_unix_perms=()

case "$mode" in
  r2)
    unset PURE_SIMDJSON_BINARY_MIRROR
    export PURE_SIMDJSON_DISABLE_GH_FALLBACK=1
    ;;
  github-fallback)
    export PURE_SIMDJSON_BINARY_MIRROR="https://releases.amikos.tech/pure-simdjson-does-not-exist"
    unset PURE_SIMDJSON_DISABLE_GH_FALLBACK
    github_fallback_enabled="true"
    broken_checksums_url="${PURE_SIMDJSON_BINARY_MIRROR}/v${version}/SHA256SUMS"
    broken_artifact_url="${PURE_SIMDJSON_BINARY_MIRROR}/v${version}/${goos}-${goarch}/${libname}"
    broken_checksums_status="$(probe_http_code "$broken_checksums_url")"
    broken_artifact_status="$(probe_http_code "$broken_artifact_url")"
    if [[ "$broken_checksums_status" != "404" || "$broken_artifact_status" != "404" ]]; then
      echo "github-fallback mode requires explicit broken-mirror 404 proof before smoke runs; got SHA256SUMS=${broken_checksums_status}, artifact=${broken_artifact_status}" >&2
      exit 1
    fi
    ;;
  *)
    echo "unsupported mode: $mode" >&2
    exit 1
    ;;
esac

export PURE_SIMDJSON_CACHE_DIR="$cache_dir"
(cd "$repo_root" && go run ./tests/smoke/go_bootstrap_smoke.go)

expected_cache_path="${cache_dir}/v${version}/${goos}-${goarch}/${libname}"
if [[ ! -f "$expected_cache_path" ]]; then
  echo "expected cache artifact not found: $expected_cache_path" >&2
  exit 1
fi

if [[ "$goos" != "windows" ]]; then
  for target in \
    "$cache_dir" \
    "$cache_dir/v${version}" \
    "$cache_dir/v${version}/${goos}-${goarch}"; do
    mode_bits="$(stat_mode "$target")"
    if [[ "$mode_bits" != "700" ]]; then
      echo "expected unix permission 700 at $target, got $mode_bits" >&2
      exit 1
    fi
    observed_unix_perms+=("${target}=${mode_bits}")
  done
fi

append_step_summary
