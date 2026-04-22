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

    def test_release_publish_generates_checksums_before_packaged_smoke(self) -> None:
        workflow_text = RELEASE_WORKFLOW.read_text(encoding="utf-8")

        generate_idx = workflow_text.index("- name: Generate SHA256SUMS from the rebuilt manifest")
        smoke_idx = workflow_text.index(
            "- name: Run Go packaged-artifact smoke gate (PURE_SIMDJSON_BINARY_MIRROR + PURE_SIMDJSON_DISABLE_GH_FALLBACK)"
        )

        self.assertLess(
            generate_idx,
            smoke_idx,
            "release publish must generate SHA256SUMS before bootstrap smoke consumes the staged mirror",
        )

    def test_release_publish_sign_and_verify_target_resolution_avoid_heredocs(self) -> None:
        workflow_text = RELEASE_WORKFLOW.read_text(encoding="utf-8")
        sign_section = workflow_text.split("- name: Sign raw shared-library assets and SHA256SUMS", 1)[1]
        sign_section = sign_section.split("- name: Verify cosign signatures before upload", 1)[0]
        verify_section = workflow_text.split("- name: Verify cosign signatures before upload", 1)[1]
        verify_section = verify_section.split("- name: Prepare flat GitHub Release assets", 1)[0]

        self.assertIn("mapfile -t sign_targets < <(python3 -c", sign_section)
        self.assertNotIn("<<'PY'", sign_section)
        self.assertIn("mapfile -t verify_targets < <(python3 -c", verify_section)
        self.assertNotIn("<<'PY'", verify_section)

    def test_release_publish_prepends_changelog_body_to_generated_notes(self) -> None:
        workflow_text = RELEASE_WORKFLOW.read_text(encoding="utf-8")

        render_idx = workflow_text.index("- name: Render release notes from CHANGELOG.md")
        publish_idx = workflow_text.index("- name: Publish GitHub release")
        self.assertLess(
            render_idx,
            publish_idx,
            "release notes must be rendered before the GitHub release step",
        )

        render_section = workflow_text.split("- name: Render release notes from CHANGELOG.md", 1)[1]
        render_section = render_section.split("- name: Publish GitHub release", 1)[0]
        self.assertIn("python3 scripts/release/render_release_notes.py", render_section)
        self.assertIn('--version "${{ github.ref_name }}"', render_section)
        self.assertIn('--output "${{ github.workspace }}/release-notes.md"', render_section)

        publish_section = workflow_text.split("- name: Publish GitHub release", 1)[1]
        self.assertIn("body_path: ${{ github.workspace }}/release-notes.md", publish_section)
        self.assertIn("generate_release_notes: true", publish_section)


if __name__ == "__main__":
    unittest.main()
