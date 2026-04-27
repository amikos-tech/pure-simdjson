#!/usr/bin/env python3

from __future__ import annotations

import argparse
import pathlib
import re
import sys


# Must stay in sync with internal/bootstrap/abi_assertion.go.
ABI_MINIMUM_VERSION = {
    "0x00010000": "0.1.0",
    "0x00010001": "0.1.2",
}

VERSION_RE = re.compile(r'const\s+Version\s*=\s*"([^"]+)"')
GO_ABI_RE = re.compile(r"ABIVersion\s+(?:uint32\s+)?=\s*(0x[0-9A-Fa-f_]+|[0-9]+)")
RUST_ABI_RE = re.compile(
    r"PURE_SIMDJSON_ABI_VERSION\s*:\s*u32\s*=\s*(0x[0-9A-Fa-f_]+|[0-9]+)"
)
SEMVER_RE = re.compile(r"^(\d+)\.(\d+)\.(\d+)(?:[-.][0-9A-Za-z.-]+)?$")


class BootstrapABIStateError(ValueError):
    pass


def strip_line_comments(text: str) -> str:
    return "\n".join(line.split("//", 1)[0] for line in text.splitlines())


def read_text(path: pathlib.Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except OSError as error:
        raise BootstrapABIStateError(f"read {path}: {error}") from error


def extract_required(pattern: re.Pattern[str], path: pathlib.Path, label: str) -> str:
    match = pattern.search(strip_line_comments(read_text(path)))
    if match is None:
        raise BootstrapABIStateError(f"failed to parse {label} from {path}")
    return match.group(1)


def normalize_abi_literal(raw: str) -> str:
    literal = raw.replace("_", "").lower()
    try:
        if literal.startswith("0x"):
            value = int(literal, 16)
        else:
            value = int(literal, 10)
    except ValueError as error:
        raise BootstrapABIStateError(f"invalid ABI literal {raw!r}") from error
    if value < 0 or value > 0xFFFFFFFF:
        raise BootstrapABIStateError(f"ABI literal out of uint32 range: {raw!r}")
    return f"0x{value:08x}"


def semver_tuple(version: str) -> tuple[int, int, int]:
    match = SEMVER_RE.match(version)
    if match is None:
        raise BootstrapABIStateError(f"invalid semantic version: {version!r}")
    return int(match.group(1)), int(match.group(2)), int(match.group(3))


def check_state(repo_root: pathlib.Path, requested_version: str) -> tuple[str, str]:
    bootstrap_version = extract_required(
        VERSION_RE,
        repo_root / "internal" / "bootstrap" / "version.go",
        "bootstrap.Version",
    )
    go_abi = normalize_abi_literal(
        extract_required(GO_ABI_RE, repo_root / "internal" / "ffi" / "types.go", "Go ABI")
    )
    rust_abi = normalize_abi_literal(
        extract_required(RUST_ABI_RE, repo_root / "src" / "lib.rs", "Rust ABI")
    )

    if go_abi != rust_abi:
        raise BootstrapABIStateError(f"Go/Rust ABI mismatch: Go {go_abi}, Rust {rust_abi}")

    minimum_version = ABI_MINIMUM_VERSION.get(go_abi)
    if minimum_version is None:
        raise BootstrapABIStateError(f"unknown ABI policy: {go_abi}")

    if semver_tuple(bootstrap_version) < semver_tuple(minimum_version):
        raise BootstrapABIStateError(
            f"stale bootstrap.Version: {bootstrap_version} is below minimum "
            f"{minimum_version} for ABI {go_abi}"
        )

    if bootstrap_version != requested_version:
        raise BootstrapABIStateError(
            f"requested version {requested_version} does not match "
            f"bootstrap.Version {bootstrap_version}"
        )

    return bootstrap_version, go_abi


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Validate bootstrap.Version and Go/Rust ABI source state before tagging."
    )
    parser.add_argument("--version", required=True, help="Semver without leading v.")
    parser.add_argument(
        "--repo-root",
        type=pathlib.Path,
        default=pathlib.Path(__file__).resolve().parents[2],
        help="Repository root. Defaults to the root inferred from this script path.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        version, abi = check_state(args.repo_root, args.version)
    except BootstrapABIStateError as error:
        print(f"check_bootstrap_abi_state.py: {error}", file=sys.stderr)
        return 1

    print(f"bootstrap ABI state ok: version {version}, abi {abi}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
