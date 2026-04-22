#!/usr/bin/env bash
set -euo pipefail

readonly SEMVER_PATTERN='^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$'
readonly DEFAULT_R2_BASE_URL="https://releases.amikos.tech/pure-simdjson"
readonly DEFAULT_GITHUB_BASE_URL="https://github.com/amikos-tech/pure-simdjson/releases/download"

repo_root=""
version=""
goos=""
goarch=""
mode=""
cache_dir=""
requested_cache_dir=""
home_dir=""
runner_temp_dir=""
libname=""
expected_cache_path=""
github_fallback_enabled="false"
broken_checksums_url=""
broken_artifact_url=""
broken_checksums_status="n/a"
broken_artifact_status="n/a"
checksums_url=""
observed_unix_perms=()

usage() {
  cat <<'EOF'
usage: run_public_bootstrap_smoke.sh --repo-root <path> --version <semver-without-v> --goos <linux|darwin|windows> --goarch <amd64|arm64> --mode <r2|github-fallback> --cache-dir <path>
EOF
}

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
  (cd "$raw" && pwd -P)
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

validate_semver_version() {
  local candidate="$1"
  if [[ ! "$candidate" =~ $SEMVER_PATTERN ]]; then
    echo "version must match <major>.<minor>.<patch>[-suffix], got: $candidate" >&2
    return 1
  fi
}

path_is_within() {
  local child parent

  child="$(trim_trailing_slash "$(slashify_path "$1")")"
  parent="$(trim_trailing_slash "$(slashify_path "$2")")"

  if [[ "$child" == "$parent" ]]; then
    return 0
  fi
  if [[ "$parent" == "/" ]]; then
    return 0
  fi
  [[ "$child" == "$parent/"* ]]
}

refuse_unsafe_cache_dir() {
  local raw="$1"
  local resolved="$2"
  local resolved_repo_root="$3"
  local resolved_home="$4"
  local resolved_runner_temp="$5"
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

  if [[ -e "$resolved" ]]; then
    if [[ -z "$resolved_runner_temp" ]]; then
      echo "refusing existing cache dir outside RUNNER_TEMP: $raw" >&2
      return 1
    fi
    if ! path_is_within "$resolved" "$resolved_runner_temp"; then
      echo "refusing existing cache dir outside RUNNER_TEMP: $raw" >&2
      return 1
    fi
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

fetch_url() {
  curl -fsSL --max-time 15 --connect-timeout 5 --retry 2 --retry-connrefused "$1"
}

probe_http_code() {
  local code
  code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 15 --connect-timeout 5 --retry 2 --retry-connrefused "$1")" || return $?
  # curl -sS can exit 0 and emit 000 on DNS/TLS failures; reject anything that is not a real HTTP status.
  if [[ ! "$code" =~ ^[1-5][0-9]{2}$ ]]; then
    echo "probe_http_code: invalid HTTP status '$code' for $1" >&2
    return 1
  fi
  printf '%s' "$code"
}

sha256_file() {
  local target="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$target" | awk '{print tolower($1)}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$target" | awk '{print tolower($1)}'
    return
  fi
  if command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$target" | awk '{print tolower($NF)}'
    return
  fi
  echo "unable to compute sha256 digest: no sha256sum, shasum, or openssl available" >&2
  return 1
}

resolve_checksums_url() {
  case "$mode" in
    r2)
      printf '%s/v%s/SHA256SUMS\n' "$DEFAULT_R2_BASE_URL" "$version"
      ;;
    github-fallback)
      printf '%s/v%s/SHA256SUMS\n' "$DEFAULT_GITHUB_BASE_URL" "$version"
      ;;
    *)
      echo "unsupported mode: $mode" >&2
      return 1
      ;;
  esac
}

validate_cached_artifact_checksum() {
  local checksums_body checksum_key expected_digest observed_digest

  checksums_url="$(resolve_checksums_url)"
  checksums_body="$(fetch_url "$checksums_url")"
  checksum_key="v${version}/${goos}-${goarch}/${libname}"
  expected_digest="$(printf '%s\n' "$checksums_body" | awk -v key="$checksum_key" '$2 == key { print tolower($1); exit }')"
  if [[ -z "$expected_digest" ]]; then
    echo "checksum entry not found for $checksum_key in $checksums_url" >&2
    return 1
  fi
  if [[ ! "$expected_digest" =~ ^[0-9a-f]{64}$ ]]; then
    echo "checksum entry for $checksum_key is not a lowercase SHA-256 digest: $expected_digest" >&2
    return 1
  fi

  observed_digest="$(sha256_file "$expected_cache_path")"
  if [[ "$observed_digest" != "$expected_digest" ]]; then
    echo "cached artifact digest mismatch for $expected_cache_path: got $observed_digest, want $expected_digest" >&2
    return 1
  fi
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
    echo "- checksums URL: \`${checksums_url}\`"
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

main() {
  local normalized_repo_root normalized_home_dir normalized_runner_temp_dir

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

  validate_semver_version "$version"

  case "$goarch" in
    amd64|arm64)
      ;;
    *)
      echo "unsupported goarch: $goarch" >&2
      exit 1
      ;;
  esac

  normalized_repo_root="$(normalize_existing_dir "$repo_root")" || {
    echo "repo root not found: $repo_root" >&2
    exit 1
  }
  repo_root="$(trim_trailing_slash "$normalized_repo_root")"
  if [[ ! -d "$repo_root" ]]; then
    echo "repo root not found: $repo_root" >&2
    exit 1
  fi

  requested_cache_dir="$cache_dir"
  cache_dir="$(canonical_path "$cache_dir")"

  normalized_home_dir="$(normalize_existing_dir "${HOME:-/}")" || {
    echo "failed to resolve HOME directory" >&2
    exit 1
  }
  home_dir="$(trim_trailing_slash "$normalized_home_dir")"

  if [[ -n "${RUNNER_TEMP:-}" ]]; then
    normalized_runner_temp_dir="$(normalize_existing_dir "$RUNNER_TEMP")" || {
      echo "RUNNER_TEMP directory not found: $RUNNER_TEMP" >&2
      exit 1
    }
    runner_temp_dir="$(trim_trailing_slash "$normalized_runner_temp_dir")"
  fi

  libname="$(libname_for_goos "$goos")"

  if [[ ! -f "$repo_root/tests/smoke/go_bootstrap_smoke.go" ]]; then
    echo "target smoke file not found under repo root: $repo_root/tests/smoke/go_bootstrap_smoke.go" >&2
    exit 1
  fi
  # Ensure ambient dev overrides do not bypass the public bootstrap path being smoked.
  if [[ -n "${PURE_SIMDJSON_LIB_PATH:-}" ]]; then
    echo "PURE_SIMDJSON_LIB_PATH must stay unset for public bootstrap validation" >&2
    exit 1
  fi

  refuse_unsafe_cache_dir "$requested_cache_dir" "$cache_dir" "$repo_root" "$home_dir" "$runner_temp_dir"
  rm -rf -- "$cache_dir"

  case "$mode" in
    r2)
      unset PURE_SIMDJSON_BINARY_MIRROR
      export PURE_SIMDJSON_DISABLE_GH_FALLBACK=1
      github_fallback_enabled="false"
      ;;
    github-fallback)
      # Use a guaranteed 404 mirror path so the wrapper proves GitHub fallback actually ran.
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
  if [[ ! -s "$expected_cache_path" ]]; then
    echo "expected cache artifact missing or empty: $expected_cache_path" >&2
    exit 1
  fi

  validate_cached_artifact_checksum

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
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
