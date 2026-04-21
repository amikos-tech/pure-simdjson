#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import pathlib
import re
import sys


EXPECTED_TARGETS: list[tuple[str, str, str]] = [
    ("linux", "amd64", "x86_64-unknown-linux-gnu"),
    ("linux", "arm64", "aarch64-unknown-linux-gnu"),
    ("darwin", "amd64", "x86_64-apple-darwin"),
    ("darwin", "arm64", "aarch64-apple-darwin"),
    ("windows", "amd64", "x86_64-pc-windows-msvc"),
]

EXPECTED_ENTRY_KEYS = {
    "goos",
    "goarch",
    "rust_target",
    "r2_key",
    "github_asset_name",
    "local_path",
    "sha256",
}

SHA256_RE = re.compile(r"^[0-9a-f]{64}$")


def platform_library_name(goos: str) -> str:
    if goos == "linux":
        return "libpure_simdjson.so"
    if goos == "darwin":
        return "libpure_simdjson.dylib"
    if goos == "windows":
        return "pure_simdjson-msvc.dll"
    raise SystemExit(f"unsupported goos in manifest: {goos}")


def github_asset_name(goos: str, goarch: str) -> str:
    if goos == "linux":
        return f"libpure_simdjson-{goos}-{goarch}.so"
    if goos == "darwin":
        return f"libpure_simdjson-{goos}-{goarch}.dylib"
    if goos == "windows":
        return f"pure_simdjson-{goos}-{goarch}-msvc.dll"
    raise SystemExit(f"unsupported goos in manifest: {goos}")


def checksum_key(version: str, goos: str, goarch: str) -> str:
    return f"v{version}/{goos}-{goarch}/{platform_library_name(goos)}"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Rewrite bootstrap version/checksum state from a release manifest."
    )
    parser.add_argument("--manifest", required=True, help="Path to manifest.json")
    parser.add_argument(
        "--version", required=True, help="Release version without the leading v"
    )
    return parser.parse_args()


def load_manifest(path: pathlib.Path) -> dict[str, object]:
    try:
        manifest = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise SystemExit(f"manifest not found: {path}") from exc
    except json.JSONDecodeError as exc:
        raise SystemExit(f"manifest is not valid JSON: {exc}") from exc

    if not isinstance(manifest, dict) or set(manifest.keys()) != {"version", "entries"}:
        raise SystemExit(
            "manifest must have exact shape {'version': <string>, 'entries': <list>}"
        )
    if not isinstance(manifest["version"], str) or not isinstance(
        manifest["entries"], list
    ):
        raise SystemExit(
            "manifest must have exact shape {'version': <string>, 'entries': <list>}"
        )
    return manifest


def normalize_entries(version: str, manifest: dict[str, object]) -> list[dict[str, str]]:
    if manifest["version"] != version:
        raise SystemExit(
            f"manifest version {manifest['version']!r} does not match --version {version!r}"
        )

    raw_entries = manifest["entries"]
    assert isinstance(raw_entries, list)
    if len(raw_entries) != len(EXPECTED_TARGETS):
        raise SystemExit(
            f"manifest must contain exactly {len(EXPECTED_TARGETS)} entries, got {len(raw_entries)}"
        )

    normalized: dict[tuple[str, str, str], dict[str, str]] = {}
    seen_checksum_keys: set[str] = set()

    for index, raw_entry in enumerate(raw_entries):
        if not isinstance(raw_entry, dict) or set(raw_entry.keys()) != EXPECTED_ENTRY_KEYS:
            raise SystemExit(
                f"manifest entry {index} must contain exactly {sorted(EXPECTED_ENTRY_KEYS)}"
            )

        entry = {}
        for key in EXPECTED_ENTRY_KEYS:
            value = raw_entry[key]
            if not isinstance(value, str) or not value:
                raise SystemExit(f"manifest entry {index} field {key!r} must be a non-empty string")
            entry[key] = value

        goos = entry["goos"]
        goarch = entry["goarch"]
        rust_target = entry["rust_target"]
        tuple_key = (goos, goarch, rust_target)
        if tuple_key not in EXPECTED_TARGETS:
            raise SystemExit(f"unexpected platform tuple in manifest: {tuple_key}")
        if tuple_key in normalized:
            raise SystemExit(f"duplicate platform tuple in manifest: {tuple_key}")

        expected_checksum_key = checksum_key(version, goos, goarch)
        if entry["r2_key"] != expected_checksum_key:
            raise SystemExit(
                f"entry {tuple_key} has r2_key {entry['r2_key']!r}, expected {expected_checksum_key!r}"
            )
        if entry["r2_key"] in seen_checksum_keys:
            raise SystemExit(f"duplicate checksum key in manifest: {entry['r2_key']}")
        seen_checksum_keys.add(entry["r2_key"])

        expected_asset = github_asset_name(goos, goarch)
        if entry["github_asset_name"] != expected_asset:
            raise SystemExit(
                f"entry {tuple_key} has github_asset_name {entry['github_asset_name']!r}, expected {expected_asset!r}"
            )

        sha256 = entry["sha256"]
        if not SHA256_RE.fullmatch(sha256):
            raise SystemExit(
                f"entry {tuple_key} has invalid sha256 {sha256!r}; expected 64 lowercase hex characters"
            )

        normalized[tuple_key] = entry

    missing = [target for target in EXPECTED_TARGETS if target not in normalized]
    if missing:
        raise SystemExit(f"manifest is missing supported platform tuples: {missing}")

    return [normalized[target] for target in EXPECTED_TARGETS]


def rewrite_version_file(path: pathlib.Path, version: str) -> None:
    text = path.read_text(encoding="utf-8")
    updated, replaced = re.subn(
        r'const Version = "[^"]+"',
        f'const Version = "{version}"',
        text,
        count=1,
    )
    if replaced != 1:
        raise SystemExit(f"failed to rewrite version constant in {path}")
    path.write_text(updated, encoding="utf-8")


def rewrite_checksums_file(path: pathlib.Path, version: str, entries: list[dict[str, str]]) -> None:
    lines = [
        f'\t"{checksum_key(version, entry["goos"], entry["goarch"])}": "{entry["sha256"]}",'
        for entry in entries
    ]
    replacement = "var Checksums = map[string]string{\n" + "\n".join(lines) + "\n}"

    text = path.read_text(encoding="utf-8")
    updated, replaced = re.subn(
        r"var Checksums = map\[string\]string\{\n.*?\n\}",
        replacement,
        text,
        count=1,
        flags=re.S,
    )
    if replaced != 1:
        raise SystemExit(f"failed to rewrite checksum map in {path}")
    path.write_text(updated, encoding="utf-8")


def main() -> int:
    args = parse_args()
    manifest_path = pathlib.Path(args.manifest).resolve()
    repo_root = pathlib.Path(__file__).resolve().parents[2]

    manifest = load_manifest(manifest_path)
    entries = normalize_entries(args.version, manifest)

    version_path = repo_root / "internal" / "bootstrap" / "version.go"
    checksums_path = repo_root / "internal" / "bootstrap" / "checksums.go"

    rewrite_version_file(version_path, args.version)
    rewrite_checksums_file(checksums_path, args.version, entries)

    print(f"updated {version_path}")
    print(f"updated {checksums_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
