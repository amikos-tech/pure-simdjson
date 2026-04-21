#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: run_native_smoke.sh <staged-artifact-path> <platform-tuple>
example: run_native_smoke.sh dist/v0.1.0/linux-amd64/libpure_simdjson.so linux-amd64
EOF
}

if [[ $# -ne 2 ]]; then
  usage >&2
  exit 1
fi

artifact_path="$1"
platform_tuple="$2"

if [[ ! -f "$artifact_path" ]]; then
  echo "staged artifact not found: $artifact_path" >&2
  exit 1
fi

artifact_dir="$(cd "$(dirname "$artifact_path")" && pwd)"

run_unix_smoke() {
  local artifact="$1"
  local tuple="$2"
  local smoke_dir
  smoke_dir="$(mktemp -d "${TMPDIR:-/tmp}/pure-simdjson-native-smoke.XXXXXX")"
  trap 'rm -rf "$smoke_dir"' RETURN

  local ffi_binary="$smoke_dir/ffi_export_surface"
  local parse_binary="$smoke_dir/minimal_parse"
  local extra_link_flags=()

  if [[ "$tuple" == linux-* ]]; then
    extra_link_flags=(-ldl)
  fi

  cc -std=c11 -Wall -Wextra -Iinclude tests/smoke/ffi_export_surface.c -o "$ffi_binary" "${extra_link_flags[@]}"
  "$ffi_binary" "$artifact"

  cc -std=c11 -Wall -Wextra -Iinclude tests/smoke/minimal_parse.c \
    -L"$artifact_dir" \
    -lpure_simdjson \
    -Wl,-rpath,"$artifact_dir" \
    -o "$parse_binary"
  "$parse_binary"
}

run_windows_smoke() {
  local artifact="$1"
  local artifact_dir="$2"
  local import_library="$artifact_dir/pure_simdjson.dll.lib"
  local ps1
  local artifact_windows_path
  local import_library_windows_path

  if [[ ! -f "$import_library" ]]; then
    echo "windows native smoke requires import library: $import_library" >&2
    exit 1
  fi

  ps1="$(mktemp "${TMPDIR:-/tmp}/pure-simdjson-native-smoke.XXXXXX.ps1")"
  artifact_windows_path="$(cygpath -w "$artifact")"
  import_library_windows_path="$(cygpath -w "$import_library")"
  trap 'rm -f "$ps1"' RETURN

  cat >"$ps1" <<PWSH
\$ErrorActionPreference = 'Stop'
\$artifactPath = '${artifact_windows_path}'
\$artifactDir = Split-Path -Parent \$artifactPath
\$importLibraryPath = '${import_library_windows_path}'
\$smokeDir = Join-Path \$env:RUNNER_TEMP 'pure-simdjson-native-smoke'
New-Item -ItemType Directory -Force -Path \$smokeDir | Out-Null

dumpbin /EXPORTS \$artifactPath | Out-Host

cl /nologo /TC /Iinclude tests\smoke\ffi_export_surface.c /link /OUT:"\$smokeDir\ffi_export_surface.exe"
if (\$LASTEXITCODE -ne 0) { exit \$LASTEXITCODE }
& "\$smokeDir\ffi_export_surface.exe" \$artifactPath
if (\$LASTEXITCODE -ne 0) { exit \$LASTEXITCODE }

cl /nologo /TC /Iinclude tests\smoke\minimal_parse.c /link /LIBPATH:\$artifactDir pure_simdjson.dll.lib /OUT:"\$smokeDir\minimal_parse.exe"
if (\$LASTEXITCODE -ne 0) { exit \$LASTEXITCODE }
\$env:PATH = "\$artifactDir;\$env:PATH"
& "\$smokeDir\minimal_parse.exe"
if (\$LASTEXITCODE -ne 0) { exit \$LASTEXITCODE }
PWSH

  powershell.exe -NoLogo -NoProfile -File "$(cygpath -w "$ps1")"
}

case "$platform_tuple" in
  linux-*)
    nm -D --defined-only "$artifact_path"
    objdump -T "$artifact_path"
    run_unix_smoke "$artifact_path" "$platform_tuple"
    ;;
  darwin-*)
    codesign --verify --verbose "$artifact_path"
    file "$artifact_path"
    run_unix_smoke "$artifact_path" "$platform_tuple"
    ;;
  windows-*)
    dumpbin /EXPORTS "$artifact_path"
    run_windows_smoke "$artifact_path" "$artifact_dir"
    ;;
  *)
    echo "unsupported platform tuple: $platform_tuple" >&2
    exit 1
    ;;
esac
