#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: check_readiness.sh [--strict] [--version <semver-without-v>]

Checks repository-local release readiness before creating a tag.

Basic mode:
  Verifies the release workflows and docs/releases.md exist.

Strict mode:
  Also validates committed bootstrap source state, committed Cargo.lock state,
  and that the current commit is anchored on origin/main.
EOF
}

strict=false
version=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --strict)
      strict=true
      shift
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

if [[ "$strict" == true && -z "$version" ]]; then
  echo "--version is required with --strict" >&2
  exit 1
fi

if [[ -n "$version" ]] && [[ ! "$version" =~ ^[0-9]+(\.[0-9]+){2}([-.][0-9A-Za-z.-]+)?$ ]]; then
  echo "expected --version without a leading v, got: $version" >&2
  exit 1
fi

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    echo "missing required file: $path" >&2
    exit 1
  fi
}

require_tracked_file() {
  local path="$1"
  require_file "$path"
  if ! git ls-files --error-unmatch "$path" >/dev/null 2>&1; then
    echo "required file is not tracked by git: $path" >&2
    exit 1
  fi
}

require_file ".github/workflows/release-prepare.yml"
require_file ".github/workflows/release.yml"
require_file "docs/releases.md"
require_tracked_file "Cargo.lock"

if [[ "$strict" != true ]]; then
  echo "basic release readiness checks passed"
  exit 0
fi

python3 scripts/release/assert_prepared_state.py --check-source --version "$version"
cargo metadata --format-version 1 --locked >/dev/null
git fetch origin main --depth=1
git merge-base --is-ancestor HEAD origin/main

echo "strict release readiness checks passed for version $version"
