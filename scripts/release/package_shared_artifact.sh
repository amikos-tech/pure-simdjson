#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: package_shared_artifact.sh \
  --input <built-library-path> \
  --goos <goos> \
  --goarch <goarch> \
  --rust-target <triple> \
  --version <semver-without-v> \
  --out-dir <dir>
EOF
}

input=""
goos=""
goarch=""
rust_target=""
version=""
out_dir=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input)
      input="${2:-}"
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
    --rust-target)
      rust_target="${2:-}"
      shift 2
      ;;
    --version)
      version="${2:-}"
      shift 2
      ;;
    --out-dir)
      out_dir="${2:-}"
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

for required in input goos goarch rust_target version out_dir; do
  if [[ -z "${!required}" ]]; then
    echo "missing required argument: $required" >&2
    usage >&2
    exit 1
  fi
done

if [[ ! -f "$input" ]]; then
  echo "input library not found: $input" >&2
  exit 1
fi

# Keep this naming contract aligned with internal/bootstrap/url.go::PlatformLibraryName
# and the GitHub flat-namespace githubAssetName logic.
platform_library_name() {
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
      echo "unsupported goos for PlatformLibraryName: $1" >&2
      exit 1
      ;;
  esac
}

github_asset_name() {
  case "$1" in
    linux)
      printf 'libpure_simdjson-%s-%s.so\n' "$1" "$2"
      ;;
    darwin)
      printf 'libpure_simdjson-%s-%s.dylib\n' "$1" "$2"
      ;;
    windows)
      printf 'pure_simdjson-%s-%s-msvc.dll\n' "$1" "$2"
      ;;
    *)
      echo "unsupported goos for github_asset_name: $1" >&2
      exit 1
      ;;
  esac
}

sha256_file() {
  python3 - "$1" <<'PY'
import hashlib
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
print(hashlib.sha256(path.read_bytes()).hexdigest())
PY
}

abs_path() {
  python3 - "$1" <<'PY'
import pathlib
import sys

print(pathlib.Path(sys.argv[1]).resolve().as_posix())
PY
}

mkdir -p "$out_dir"

r2_filename="$(platform_library_name "$goos")"
github_filename="$(github_asset_name "$goos" "$goarch")"
r2_key="v${version}/${goos}-${goarch}/${r2_filename}"
r2_path="${out_dir}/${r2_key}"
github_path="${out_dir}/${github_filename}"

mkdir -p "$(dirname "$r2_path")"
cp "$input" "$r2_path"
cp "$r2_path" "$github_path"

sha256="$(sha256_file "$r2_path")"
r2_abs_path="$(abs_path "$r2_path")"
manifest_path="${out_dir}/manifest.json"

python3 - "$manifest_path" "$version" "$goos" "$goarch" "$rust_target" "$r2_key" "$github_filename" "$r2_abs_path" "$sha256" <<'PY'
import json
import pathlib
import sys

(
    manifest_path,
    version,
    goos,
    goarch,
    rust_target,
    r2_key,
    github_asset_name,
    local_path,
    sha256,
) = sys.argv[1:]

manifest = {
    "version": version,
    "entries": [
        {
            "goos": goos,
            "goarch": goarch,
            "rust_target": rust_target,
            "r2_key": r2_key,
            "github_asset_name": github_asset_name,
            "local_path": local_path,
            "sha256": sha256,
        }
    ],
}

path = pathlib.Path(manifest_path)
path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
PY

printf 'manifest_path=%s\n' "$manifest_path"
printf 'r2_key=%s\n' "$r2_key"
printf 'local_path=%s\n' "$r2_abs_path"
printf 'github_asset_name=%s\n' "$github_filename"
printf 'sha256=%s\n' "$sha256"
