#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import subprocess
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "release" / "check_macos_downloaded_dylib.sh"


class CheckMacOSDownloadedDylibScriptTests(unittest.TestCase):
    def run_script(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["bash", str(SCRIPT_PATH), *args],
            cwd=REPO_ROOT,
            check=False,
            capture_output=True,
            text=True,
        )

    def test_script_exists(self) -> None:
        self.assertTrue(
            SCRIPT_PATH.is_file(),
            msg=f"expected script at {SCRIPT_PATH}",
        )

    def test_help_prints_usage(self) -> None:
        result = self.run_script("--help")
        self.assertEqual(result.returncode, 0, msg=result.stderr)
        self.assertIn("usage: check_macos_downloaded_dylib.sh", result.stdout)
        self.assertIn("--artifact <path>", result.stdout)
        self.assertIn("--build-local", result.stdout)

    def test_requires_exactly_one_artifact_source(self) -> None:
        result = self.run_script()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("exactly one of --artifact or --build-local is required", result.stderr)

        result = self.run_script("--artifact", "fake.dylib", "--build-local")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("exactly one of --artifact or --build-local is required", result.stderr)

    def test_keep_temp_requires_artifact_source(self) -> None:
        result = self.run_script("--keep-temp")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("exactly one of --artifact or --build-local is required", result.stderr)


if __name__ == "__main__":
    unittest.main()
