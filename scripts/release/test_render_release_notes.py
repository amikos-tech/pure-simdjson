#!/usr/bin/env python3

from __future__ import annotations

import importlib.util
import pathlib
import subprocess
import tempfile
import textwrap
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "release" / "render_release_notes.py"
SPEC = importlib.util.spec_from_file_location("render_release_notes", SCRIPT_PATH)
assert SPEC is not None
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(MODULE)

SAMPLE_CHANGELOG = textwrap.dedent(
    """\
    # Changelog

    ## [Unreleased]

    ### Changed
    - Future work.

    ## [1.2.0] - 2026-04-22

    ### Added
    - Brand new thing.

    ## [1.1.0] - 2026-04-21

    ### Fixed
    - Existing bug.
    """
)


class RenderReleaseNotesTests(unittest.TestCase):
    def write_text_file(self, tmpdir: str, name: str, contents: str) -> pathlib.Path:
        path = pathlib.Path(tmpdir) / name
        path.write_text(contents, encoding="utf-8")
        return path

    def run_cli(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["python3", str(SCRIPT_PATH), *args],
            cwd=REPO_ROOT,
            check=False,
            capture_output=True,
            text=True,
        )

    def test_normalize_version_accepts_tag_prefixes(self) -> None:
        self.assertEqual(MODULE.normalize_version("1.2.0"), "1.2.0")
        self.assertEqual(MODULE.normalize_version("v1.2.0"), "1.2.0")
        self.assertEqual(MODULE.normalize_version("refs/tags/v1.2.0"), "1.2.0")

    def test_extract_release_section_returns_exact_tagged_entry(self) -> None:
        rendered = MODULE.extract_release_section(SAMPLE_CHANGELOG, "1.2.0")

        self.assertEqual(
            rendered,
            "## [1.2.0] - 2026-04-22\n\n### Added\n- Brand new thing.\n",
        )

    def test_extract_release_section_rejects_duplicate_headings(self) -> None:
        duplicate = SAMPLE_CHANGELOG + textwrap.dedent(
            """\

            ## [1.2.0] - 2026-04-20

            ### Fixed
            - Duplicate heading.
            """
        )

        with self.assertRaisesRegex(ValueError, "duplicate changelog headings"):
            MODULE.extract_release_section(duplicate, "1.2.0")

    def test_extract_release_section_rejects_heading_without_body(self) -> None:
        changelog = textwrap.dedent(
            """\
            # Changelog

            ## [1.2.0] - 2026-04-22
            """
        )

        with self.assertRaisesRegex(ValueError, "has no body"):
            MODULE.extract_release_section(changelog, "1.2.0")

    def test_cli_writes_requested_changelog_entry(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            changelog_path = self.write_text_file(tmpdir, "CHANGELOG.md", SAMPLE_CHANGELOG)
            output_path = pathlib.Path(tmpdir) / "release-notes.md"

            result = self.run_cli(
                "--changelog",
                str(changelog_path),
                "--version",
                "refs/tags/v1.1.0",
                "--output",
                str(output_path),
            )

            self.assertEqual(result.returncode, 0, msg=result.stderr)
            self.assertEqual(
                output_path.read_text(encoding="utf-8"),
                "## [1.1.0] - 2026-04-21\n\n### Fixed\n- Existing bug.\n",
            )

    def test_cli_reports_missing_version(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            changelog_path = self.write_text_file(tmpdir, "CHANGELOG.md", SAMPLE_CHANGELOG)

            result = self.run_cli(
                "--changelog",
                str(changelog_path),
                "--version",
                "v9.9.9",
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("version '9.9.9' not found in changelog", result.stderr)

    def test_cli_reports_missing_changelog_file(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            missing_path = pathlib.Path(tmpdir) / "MISSING.md"

            result = self.run_cli("--changelog", str(missing_path), "--version", "1.2.0")

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("render_release_notes.py:", result.stderr)
            self.assertIn("MISSING.md", result.stderr)

    def test_cli_reports_malformed_heading_as_missing_version(self) -> None:
        malformed = textwrap.dedent(
            """\
            # Changelog

            ## [1.2.0 - 2026-04-22

            ### Added
            - Broken heading.
            """
        )

        with tempfile.TemporaryDirectory() as tmpdir:
            changelog_path = self.write_text_file(tmpdir, "CHANGELOG.md", malformed)

            result = self.run_cli("--changelog", str(changelog_path), "--version", "1.2.0")

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("version '1.2.0' not found in changelog", result.stderr)

    def test_cli_reports_non_utf8_changelog(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            changelog_path = pathlib.Path(tmpdir) / "CHANGELOG.md"
            changelog_path.write_bytes(b"\xff\xfe\xfd")

            result = self.run_cli("--changelog", str(changelog_path), "--version", "1.2.0")

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("render_release_notes.py:", result.stderr)

    def test_cli_reports_output_write_error(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            changelog_path = self.write_text_file(tmpdir, "CHANGELOG.md", SAMPLE_CHANGELOG)
            output_path = pathlib.Path(tmpdir) / "missing-dir" / "release-notes.md"

            result = self.run_cli(
                "--changelog",
                str(changelog_path),
                "--version",
                "1.2.0",
                "--output",
                str(output_path),
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertIn("render_release_notes.py:", result.stderr)
            self.assertIn("missing-dir", result.stderr)


if __name__ == "__main__":
    unittest.main()
