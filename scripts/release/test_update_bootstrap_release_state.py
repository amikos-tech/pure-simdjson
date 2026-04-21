#!/usr/bin/env python3

from __future__ import annotations

import json
import pathlib
import shutil
import subprocess
import tempfile
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "release" / "update_bootstrap_release_state.py"
VERSION_PATH = REPO_ROOT / "internal" / "bootstrap" / "version.go"
CHECKSUMS_PATH = REPO_ROOT / "internal" / "bootstrap" / "checksums.go"

TARGET_ORDER = [
    ("linux", "amd64", "x86_64-unknown-linux-gnu"),
    ("linux", "arm64", "aarch64-unknown-linux-gnu"),
    ("darwin", "amd64", "x86_64-apple-darwin"),
    ("darwin", "arm64", "aarch64-apple-darwin"),
    ("windows", "amd64", "x86_64-pc-windows-msvc"),
]

KNOWN_SHA256S = {
    ("linux", "amd64"): "1" * 64,
    ("linux", "arm64"): "2" * 64,
    ("darwin", "amd64"): "3" * 64,
    ("darwin", "arm64"): "4" * 64,
    ("windows", "amd64"): "5" * 64,
}


def platform_library_name(goos: str) -> str:
    if goos == "linux":
        return "libpure_simdjson.so"
    if goos == "darwin":
        return "libpure_simdjson.dylib"
    if goos == "windows":
        return "pure_simdjson-msvc.dll"
    raise ValueError(f"unsupported goos: {goos}")


def github_asset_name(goos: str, goarch: str) -> str:
    if goos == "linux":
        return f"libpure_simdjson-{goos}-{goarch}.so"
    if goos == "darwin":
        return f"libpure_simdjson-{goos}-{goarch}.dylib"
    if goos == "windows":
        return f"pure_simdjson-{goos}-{goarch}-msvc.dll"
    raise ValueError(f"unsupported goos: {goos}")


def make_manifest(version: str) -> dict[str, object]:
    entries = []
    for goos, goarch, rust_target in reversed(TARGET_ORDER):
        key = f"v{version}/{goos}-{goarch}/{platform_library_name(goos)}"
        entries.append(
            {
                "goos": goos,
                "goarch": goarch,
                "rust_target": rust_target,
                "r2_key": key,
                "github_asset_name": github_asset_name(goos, goarch),
                "local_path": f"/tmp/staged/{goos}-{goarch}/{platform_library_name(goos)}",
                "sha256": KNOWN_SHA256S[(goos, goarch)],
            }
        )
    return {"version": version, "entries": entries}


class UpdateBootstrapReleaseStateTests(unittest.TestCase):
    def setUp(self) -> None:
        self.real_version_text = VERSION_PATH.read_text(encoding="utf-8")
        self.real_checksums_text = CHECKSUMS_PATH.read_text(encoding="utf-8")

    def tearDown(self) -> None:
        self.assertEqual(self.real_version_text, VERSION_PATH.read_text(encoding="utf-8"))
        self.assertEqual(
            self.real_checksums_text,
            CHECKSUMS_PATH.read_text(encoding="utf-8"),
        )

    def make_workspace(self) -> tuple[tempfile.TemporaryDirectory[str], pathlib.Path]:
        tempdir = tempfile.TemporaryDirectory()
        root = pathlib.Path(tempdir.name)

        script_copy = root / "scripts" / "release" / "update_bootstrap_release_state.py"
        script_copy.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(SCRIPT_PATH, script_copy)

        version_copy = root / "internal" / "bootstrap" / "version.go"
        version_copy.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(VERSION_PATH, version_copy)
        shutil.copy2(CHECKSUMS_PATH, root / "internal" / "bootstrap" / "checksums.go")

        return tempdir, root

    def write_manifest(
        self, workspace_root: pathlib.Path, version: str, manifest: dict[str, object]
    ) -> pathlib.Path:
        manifest_path = workspace_root / "manifest.json"
        manifest_path.write_text(
            json.dumps(manifest, indent=2) + "\n",
            encoding="utf-8",
        )
        return manifest_path

    def run_script(
        self, workspace_root: pathlib.Path, manifest_path: pathlib.Path, version: str
    ) -> subprocess.CompletedProcess[str]:
        script_copy = workspace_root / "scripts" / "release" / "update_bootstrap_release_state.py"
        return subprocess.run(
            [
                "python3",
                str(script_copy),
                "--manifest",
                str(manifest_path),
                "--version",
                version,
            ],
            cwd=workspace_root,
            check=False,
            capture_output=True,
            text=True,
        )

    def test_rewrites_version_and_checksums_from_manifest(self) -> None:
        tempdir, workspace_root = self.make_workspace()
        self.addCleanup(tempdir.cleanup)

        version = "0.1.7"
        manifest_path = self.write_manifest(workspace_root, version, make_manifest(version))

        result = self.run_script(workspace_root, manifest_path, version)
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        version_text = (
            workspace_root / "internal" / "bootstrap" / "version.go"
        ).read_text(encoding="utf-8")
        self.assertIn('const Version = "0.1.7"', version_text)
        self.assertIn("// Version is the library version pinned at compile time.", version_text)

        checksums_text = (
            workspace_root / "internal" / "bootstrap" / "checksums.go"
        ).read_text(encoding="utf-8")
        positions: list[int] = []
        for goos, goarch, _ in TARGET_ORDER:
            key = f'v{version}/{goos}-{goarch}/{platform_library_name(goos)}'
            digest = KNOWN_SHA256S[(goos, goarch)]
            self.assertIn(key, checksums_text)
            self.assertIn(digest, checksums_text)
            positions.append(checksums_text.index(key))

        self.assertEqual(sorted(positions), positions)

    def test_rewrite_is_idempotent(self) -> None:
        tempdir, workspace_root = self.make_workspace()
        self.addCleanup(tempdir.cleanup)

        version = "0.1.8"
        manifest_path = self.write_manifest(workspace_root, version, make_manifest(version))

        first_result = self.run_script(workspace_root, manifest_path, version)
        self.assertEqual(first_result.returncode, 0, msg=first_result.stderr)

        version_copy = workspace_root / "internal" / "bootstrap" / "version.go"
        checksums_copy = workspace_root / "internal" / "bootstrap" / "checksums.go"
        first_version_text = version_copy.read_text(encoding="utf-8")
        first_checksums_text = checksums_copy.read_text(encoding="utf-8")

        second_result = self.run_script(workspace_root, manifest_path, version)
        self.assertEqual(second_result.returncode, 0, msg=second_result.stderr)
        self.assertEqual(first_version_text, version_copy.read_text(encoding="utf-8"))
        self.assertEqual(first_checksums_text, checksums_copy.read_text(encoding="utf-8"))

    def test_rejects_missing_sha256(self) -> None:
        tempdir, workspace_root = self.make_workspace()
        self.addCleanup(tempdir.cleanup)

        version = "0.1.9"
        manifest = make_manifest(version)
        manifest["entries"][0].pop("sha256")
        manifest_path = self.write_manifest(workspace_root, version, manifest)

        result = self.run_script(workspace_root, manifest_path, version)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("sha256", result.stderr)

    def test_rejects_malformed_sha256(self) -> None:
        tempdir, workspace_root = self.make_workspace()
        self.addCleanup(tempdir.cleanup)

        version = "0.2.0"
        manifest = make_manifest(version)
        manifest["entries"][0]["sha256"] = "abc123"
        manifest_path = self.write_manifest(workspace_root, version, manifest)

        result = self.run_script(workspace_root, manifest_path, version)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("sha256", result.stderr)


if __name__ == "__main__":
    unittest.main()
