#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: assemble_staged_release_tree.sh \
  --version <semver-without-v> \
  --manifest-dir <dir> \
  --artifact-dir <dir> \
  --staged-root <dir>
EOF
}

version=""
manifest_dir=""
artifact_dir=""
staged_root=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      version="${2:-}"
      shift 2
      ;;
    --manifest-dir)
      manifest_dir="${2:-}"
      shift 2
      ;;
    --artifact-dir)
      artifact_dir="${2:-}"
      shift 2
      ;;
    --staged-root)
      staged_root="${2:-}"
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

for required in version manifest_dir artifact_dir staged_root; do
  if [[ -z "${!required}" ]]; then
    echo "missing required argument: $required" >&2
    usage >&2
    exit 1
  fi
done

if [[ ! -d "$manifest_dir" ]]; then
  echo "manifest directory not found: $manifest_dir" >&2
  exit 1
fi
if [[ ! -d "$artifact_dir" ]]; then
  echo "artifact directory not found: $artifact_dir" >&2
  exit 1
fi

rm -rf "$staged_root"
mkdir -p "$staged_root"

python3 - "$version" "$manifest_dir" "$artifact_dir" "$staged_root" <<'PY'
import json
import pathlib
import shutil
import sys

version, manifest_dir, artifact_dir, staged_root = sys.argv[1:]
manifest_root = pathlib.Path(manifest_dir).resolve()
artifact_root = pathlib.Path(artifact_dir).resolve()
staged_root_path = pathlib.Path(staged_root).resolve()

expected = {
    ("linux", "amd64", "x86_64-unknown-linux-gnu"): f"v{version}/linux-amd64/libpure_simdjson.so",
    ("linux", "arm64", "aarch64-unknown-linux-gnu"): f"v{version}/linux-arm64/libpure_simdjson.so",
    ("darwin", "amd64", "x86_64-apple-darwin"): f"v{version}/darwin-amd64/libpure_simdjson.dylib",
    ("darwin", "arm64", "aarch64-apple-darwin"): f"v{version}/darwin-arm64/libpure_simdjson.dylib",
    ("windows", "amd64", "x86_64-pc-windows-msvc"): f"v{version}/windows-amd64/pure_simdjson-msvc.dll",
}
required_keys = {
    "goos",
    "goarch",
    "rust_target",
    "r2_key",
    "github_asset_name",
    "local_path",
    "sha256",
}

manifest_files = sorted(manifest_root.rglob("manifest-*.json"))
if not manifest_files:
    manifest_files = sorted(manifest_root.rglob("manifest.json"))
if len(manifest_files) != len(expected):
    raise SystemExit(
        f"expected exactly {len(expected)} manifest rows under {manifest_root}, found {len(manifest_files)}"
    )

seen = set()
copied = []

for manifest_path in manifest_files:
    try:
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise SystemExit(f"manifest {manifest_path} is not valid JSON: {exc}") from exc

    if set(manifest.keys()) != {"version", "entries"}:
        raise SystemExit(f"manifest {manifest_path} must contain only 'version' and 'entries'")
    if manifest["version"] != version:
        raise SystemExit(
            f"manifest {manifest_path} version {manifest['version']!r} does not match {version!r}"
        )

    entries = manifest["entries"]
    if not isinstance(entries, list) or len(entries) != 1:
        raise SystemExit(f"manifest {manifest_path} must contain exactly one manifest row")
    entry = entries[0]
    if not isinstance(entry, dict) or set(entry.keys()) != required_keys:
        raise SystemExit(
            f"manifest {manifest_path} entry must contain exactly {sorted(required_keys)}"
        )

    tuple_key = (entry["goos"], entry["goarch"], entry["rust_target"])
    if tuple_key not in expected:
        raise SystemExit(f"unexpected platform tuple in {manifest_path}: {tuple_key}")
    if tuple_key in seen:
        raise SystemExit(f"duplicate platform tuple in manifest rows: {tuple_key}")
    seen.add(tuple_key)

    expected_r2_key = expected[tuple_key]
    if entry["r2_key"] != expected_r2_key:
        raise SystemExit(
            f"manifest {manifest_path} r2_key {entry['r2_key']!r} does not match {expected_r2_key!r}"
        )

    candidates = [
        path
        for path in artifact_root.rglob(pathlib.Path(expected_r2_key).name)
        if path.as_posix().endswith("/" + expected_r2_key)
    ]
    if len(candidates) != 1:
        raise SystemExit(
            f"expected exactly one packaged artifact for {expected_r2_key}, found {len(candidates)}"
        )

    source = candidates[0]
    destination = staged_root_path / expected_r2_key
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, destination)
    copied.append(destination.as_posix())

missing = [tuple_key for tuple_key in expected if tuple_key not in seen]
if missing:
    raise SystemExit(f"missing manifest rows for supported tuples: {missing}")

for path in copied:
    print(path)
PY
