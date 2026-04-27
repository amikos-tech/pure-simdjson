#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import subprocess
import tempfile
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
CHECKER = REPO_ROOT / "scripts" / "release" / "check_bootstrap_abi_state.py"


VERSION_GO = """package bootstrap

// Version is the library version pinned at compile time.
const Version = "{version}"
"""

TYPES_GO = """package ffi

const (
\tABIVersion uint32 = {abi}
)
"""

LIB_RS = """pub const PURE_SIMDJSON_ABI_VERSION: u32 = {abi};
"""


class BootstrapABIStateTest(unittest.TestCase):
    def run_checker(
        self,
        *,
        requested_version: str = "0.1.2",
        bootstrap_version: str = "0.1.2",
        go_abi: str = "0x00010001",
        rust_abi: str = "0x0001_0001",
    ) -> subprocess.CompletedProcess[str]:
        with tempfile.TemporaryDirectory() as tmp:
            repo_root = pathlib.Path(tmp)
            (repo_root / "internal" / "bootstrap").mkdir(parents=True)
            (repo_root / "internal" / "ffi").mkdir(parents=True)
            (repo_root / "src").mkdir()

            (repo_root / "internal" / "bootstrap" / "version.go").write_text(
                VERSION_GO.format(version=bootstrap_version),
                encoding="utf-8",
            )
            (repo_root / "internal" / "ffi" / "types.go").write_text(
                TYPES_GO.format(abi=go_abi),
                encoding="utf-8",
            )
            (repo_root / "src" / "lib.rs").write_text(
                LIB_RS.format(abi=rust_abi),
                encoding="utf-8",
            )

            return subprocess.run(
                [
                    "python3",
                    str(CHECKER),
                    "--version",
                    requested_version,
                    "--repo-root",
                    str(repo_root),
                ],
                check=False,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
            )

    def assert_failed_with(self, result: subprocess.CompletedProcess[str], diagnostic: str) -> None:
        self.assertNotEqual(result.returncode, 0, result)
        self.assertIn(diagnostic, result.stderr)

    def test_accepts_valid_0_1_2_state(self) -> None:
        result = self.run_checker()

        self.assertEqual(result.returncode, 0, result)
        self.assertIn(
            "bootstrap ABI state ok: version 0.1.2, abi 0x00010001",
            result.stdout,
        )

    def test_rejects_stale_bootstrap_version_for_current_abi(self) -> None:
        result = self.run_checker(bootstrap_version="0.1.0")

        self.assert_failed_with(result, "stale bootstrap.Version")

    def test_rejects_0_1_1_as_stale_for_current_abi(self) -> None:
        result = self.run_checker(
            requested_version="0.1.1", bootstrap_version="0.1.1"
        )

        self.assert_failed_with(result, "stale bootstrap.Version")

    def test_rejects_go_rust_abi_mismatch(self) -> None:
        result = self.run_checker(rust_abi="0x00010000")

        self.assert_failed_with(result, "Go/Rust ABI mismatch")

    def test_rejects_unknown_abi_policy(self) -> None:
        result = self.run_checker(go_abi="0x00010002", rust_abi="0x0001_0002")

        self.assert_failed_with(result, "unknown ABI policy")

    def test_rejects_requested_version_mismatch(self) -> None:
        result = self.run_checker(requested_version="0.1.1")

        self.assert_failed_with(result, "requested version")


if __name__ == "__main__":
    unittest.main()
