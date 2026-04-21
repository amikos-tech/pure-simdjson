#!/usr/bin/env python3

from __future__ import annotations

import argparse
import os
import pathlib
import re
import sys

import update_bootstrap_release_state as contract


VERSION_RE = re.compile(r'^const Version = "([^"]+)"$', re.M)
CHECKSUM_MAP_RE = re.compile(
    r"var Checksums = map\[string\]string\{\n(?P<body>.*?)\n\}",
    re.S,
)
CHECKSUM_ENTRY_RE = re.compile(r'^\s*"([^"]+)":\s*"([^"]+)",\s*$')


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Validate committed bootstrap release state against the Phase 6 release "
            "manifest contract."
        )
    )
    parser.add_argument(
        "--manifest",
        help="Path to a release manifest.json to compare against committed source state.",
    )
    parser.add_argument(
        "--version",
        required=True,
        help="Release version without the leading v.",
    )
    parser.add_argument(
        "--check-source",
        action="store_true",
        help=(
            "Validate only committed internal/bootstrap/version.go plus the five "
            "required checksum keys."
        ),
    )
    args = parser.parse_args()
    if args.check_source and args.manifest:
        parser.error("--check-source cannot be combined with --manifest")
    if not args.check_source and not args.manifest:
        parser.error("--manifest is required unless --check-source is set")
    return args


def repo_root() -> pathlib.Path:
    return pathlib.Path(__file__).resolve().parents[2]


def load_version(version_path: pathlib.Path) -> str:
    text = version_path.read_text(encoding="utf-8")
    match = VERSION_RE.search(text)
    if not match:
        raise SystemExit(f"failed to parse Version constant from {version_path}")
    return match.group(1)


def load_checksums(checksums_path: pathlib.Path) -> dict[str, str]:
    text = checksums_path.read_text(encoding="utf-8")
    match = CHECKSUM_MAP_RE.search(text)
    if not match:
        raise SystemExit(f"failed to locate Checksums map in {checksums_path}")

    parsed: dict[str, str] = {}
    for raw_line in match.group("body").splitlines():
        stripped = raw_line.strip()
        if not stripped or stripped.startswith("//"):
            continue
        entry_match = CHECKSUM_ENTRY_RE.match(raw_line)
        if not entry_match:
            raise SystemExit(
                f"failed to parse checksum map entry in {checksums_path}: {stripped}"
            )
        key, digest = entry_match.groups()
        if key in parsed:
            raise SystemExit(f"duplicate checksum key in {checksums_path}: {key}")
        parsed[key] = digest
    return parsed


def required_checksum_keys(version: str) -> list[str]:
    return [
        contract.checksum_key(version, goos, goarch)
        for goos, goarch, _rust_target in contract.EXPECTED_TARGETS
    ]


def append_mismatch_summary(
    version: str,
    source_version: str,
    mismatches: list[tuple[str, str, str]],
) -> None:
    summary_path = os.environ.get("GITHUB_STEP_SUMMARY")
    if not summary_path:
        return

    path = pathlib.Path(summary_path)
    with path.open("a", encoding="utf-8") as fh:
        fh.write("## Prepared state check\n\n")
        if source_version != version:
            fh.write(
                f"- Version mismatch: expected `{version}`, found `{source_version}` in "
                "`internal/bootstrap/version.go`\n\n"
            )
        if not mismatches:
            return
        fh.write("| r2_key | expected | actual |\n")
        fh.write("| ------ | -------- | ------ |\n")
        for r2_key, expected, actual in mismatches:
            fh.write(f"| `{r2_key}` | `{expected}` | `{actual}` |\n")
        fh.write("\n")


def validate_source_state(
    version: str,
    checksums: dict[str, str],
    source_version: str,
) -> list[str]:
    errors: list[str] = []
    if source_version != version:
        errors.append(
            f"internal/bootstrap/version.go version {source_version!r} does not match {version!r}"
        )

    for key in required_checksum_keys(version):
        digest = checksums.get(key)
        if digest is None:
            errors.append(f"missing checksum key in internal/bootstrap/checksums.go: {key}")
            continue
        if not contract.SHA256_RE.fullmatch(digest):
            errors.append(
                f"checksum value for {key} is invalid {digest!r}; expected 64 lowercase hex characters"
            )
    return errors


def validate_manifest_against_source(
    version: str,
    manifest_path: pathlib.Path,
    checksums: dict[str, str],
    source_version: str,
) -> list[str]:
    manifest = contract.load_manifest(manifest_path)
    entries = contract.normalize_entries(version, manifest)

    errors = validate_source_state(version, checksums, source_version)
    mismatches: list[tuple[str, str, str]] = []

    for entry in entries:
        r2_key = entry["r2_key"]
        expected = checksums.get(r2_key)
        actual = entry["sha256"]
        if expected is None:
            mismatches.append((r2_key, "<missing>", actual))
            continue
        if expected != actual:
            mismatches.append((r2_key, expected, actual))

    append_mismatch_summary(version, source_version, mismatches)

    if mismatches:
        errors.extend(
            f"manifest digest drift for {r2_key}: expected {expected}, found {actual}"
            for r2_key, expected, actual in mismatches
        )
    return errors


def main() -> int:
    args = parse_args()
    root = repo_root()
    version_path = root / "internal" / "bootstrap" / "version.go"
    checksums_path = root / "internal" / "bootstrap" / "checksums.go"

    source_version = load_version(version_path)
    checksums = load_checksums(checksums_path)

    if args.check_source:
        errors = validate_source_state(args.version, checksums, source_version)
    else:
        errors = validate_manifest_against_source(
            args.version,
            pathlib.Path(args.manifest).resolve(),
            checksums,
            source_version,
        )

    if errors:
        raise SystemExit("\n".join(errors))

    if args.check_source:
        print(
            f"committed bootstrap source state matches version {args.version} and the five required checksum keys"
        )
    else:
        print(
            f"committed bootstrap source state matches manifest {pathlib.Path(args.manifest).resolve()}"
        )
    return 0


if __name__ == "__main__":
    sys.exit(main())
