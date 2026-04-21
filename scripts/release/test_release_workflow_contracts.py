#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import re
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
BUILD_SHARED_LIBRARY_ACTION = (
    REPO_ROOT / ".github" / "actions" / "build-shared-library" / "action.yml"
)
SETUP_RUST_ACTION = REPO_ROOT / ".github" / "actions" / "setup-rust" / "action.yml"
RELEASE_WORKFLOW = REPO_ROOT / ".github" / "workflows" / "release.yml"
RUN_NATIVE_SMOKE = REPO_ROOT / "scripts" / "release" / "run_native_smoke.sh"


class ReleaseWorkflowContractTests(unittest.TestCase):
    def test_build_shared_library_forwards_toolchain_file_input(self) -> None:
        action_text = BUILD_SHARED_LIBRARY_ACTION.read_text(encoding="utf-8")

        self.assertRegex(
            action_text,
            re.compile(
                r"uses:\s+\./\.github/actions/setup-rust\s+with:\s+"
                r"toolchain-file:\s+\${{\s*inputs\.toolchain-file\s*}}",
                re.MULTILINE,
            ),
        )

    def test_setup_rust_does_not_require_tomli(self) -> None:
        action_text = SETUP_RUST_ACTION.read_text(encoding="utf-8")

        self.assertNotIn("import tomli", action_text)
        self.assertIn("grep '^channel'", action_text)

    def test_windows_packaging_uses_workspace_absolute_out_dir(self) -> None:
        workflow_text = RELEASE_WORKFLOW.read_text(encoding="utf-8")
        windows_section = workflow_text.split("- name: Package windows shared library", 1)[1]

        self.assertIn(
            "out-dir: ${{ github.workspace }}/dist/${{ matrix.platform_id }}",
            windows_section,
        )
        self.assertNotIn("out-dir: dist/${{ matrix.platform_id }}", windows_section)

    def test_windows_import_library_is_preserved_next_to_staged_dll(self) -> None:
        workflow_text = RELEASE_WORKFLOW.read_text(encoding="utf-8")
        windows_section = workflow_text.split("- name: Preserve windows import library", 1)[1]

        self.assertIn(
            'r2_dir="$(dirname "${{ steps.package.outputs.r2-path }}")"',
            windows_section,
        )
        self.assertIn(
            'cp "$import_lib_path" "$r2_dir/${{ matrix.import_library_name }}"',
            windows_section,
        )

    def test_windows_native_smoke_restores_canonical_dll_name_for_import_lib(self) -> None:
        script_text = RUN_NATIVE_SMOKE.read_text(encoding="utf-8")

        self.assertIn(
            r"\$runtimeDllPath = Join-Path \$smokeDir 'pure_simdjson.dll'",
            script_text,
        )
        self.assertIn(
            r"Copy-Item -Force \$artifactPath \$runtimeDllPath",
            script_text,
        )
        self.assertIn(
            r"Copy-Item -Force \$importLibraryPath \$runtimeImportLibraryPath",
            script_text,
        )
        self.assertIn(
            r'cl /nologo /TC /Iinclude tests\smoke\minimal_parse.c /link /LIBPATH:\$smokeDir pure_simdjson.dll.lib /OUT:"\$smokeDir\minimal_parse.exe"',
            script_text,
        )
        self.assertIn(
            r'\$env:PATH = "\$smokeDir;\$env:PATH"',
            script_text,
        )


if __name__ == "__main__":
    unittest.main()
