#!/usr/bin/env python3

from __future__ import annotations

import argparse
import pathlib
import re
import sys


def normalize_version(raw: str) -> str:
    version = raw.strip()
    if version.startswith("refs/tags/"):
        version = version.rsplit("/", 1)[-1]
    if version.startswith("v"):
        version = version[1:]
    if not version:
        raise ValueError("version must not be empty")
    return version


def extract_release_section(changelog_text: str, version: str) -> str:
    heading_pattern = re.compile(
        rf"^## \[{re.escape(version)}\](?:\s+-\s+.+)?$",
        re.MULTILINE,
    )
    heading_matches = list(heading_pattern.finditer(changelog_text))
    if not heading_matches:
        raise ValueError(f"version {version!r} not found in changelog")
    if len(heading_matches) > 1:
        raise ValueError(f"duplicate changelog headings found for version {version!r}")

    heading_match = heading_matches[0]

    next_heading_pattern = re.compile(r"^## \[", re.MULTILINE)
    next_heading_match = next_heading_pattern.search(changelog_text, heading_match.end())
    section_end = next_heading_match.start() if next_heading_match is not None else len(changelog_text)

    section = changelog_text[heading_match.start():section_end].strip()
    if not section:
        raise ValueError(f"changelog entry for version {version!r} is empty")
    if section == heading_match.group(0):
        raise ValueError(f"changelog entry for version {version!r} has no body")

    return f"{section}\n"


def render_release_notes(changelog_path: pathlib.Path, version: str) -> str:
    changelog_text = changelog_path.read_text(encoding="utf-8")
    normalized_version = normalize_version(version)
    return extract_release_section(changelog_text, normalized_version)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Render the tagged CHANGELOG.md entry used to seed GitHub release notes."
    )
    parser.add_argument(
        "--version",
        required=True,
        help="Release version or tag (for example: v0.1.0 or 0.1.0).",
    )
    parser.add_argument(
        "--changelog",
        default="CHANGELOG.md",
        help="Path to the changelog file. Defaults to CHANGELOG.md.",
    )
    parser.add_argument(
        "--output",
        default="-",
        help="Write rendered notes to this path, or '-' for stdout. Defaults to stdout.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    changelog_path = pathlib.Path(args.changelog)

    try:
        rendered = render_release_notes(changelog_path, args.version)
        if args.output == "-":
            sys.stdout.write(rendered)
            return 0

        output_path = pathlib.Path(args.output)
        output_path.write_text(rendered, encoding="utf-8")
    except (OSError, UnicodeError, ValueError) as exc:
        print(f"render_release_notes.py: {exc}", file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
