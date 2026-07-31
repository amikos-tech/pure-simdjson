#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import re
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
BUILD_SHARED_LIBRARY_ACTION = (
    REPO_ROOT / ".github" / "actions" / "build-shared-library" / "action.yml"
)
BUILD_RS = REPO_ROOT / "build.rs"
SETUP_RUST_ACTION = REPO_ROOT / ".github" / "actions" / "setup-rust" / "action.yml"
RELEASE_WORKFLOW = REPO_ROOT / ".github" / "workflows" / "release.yml"
PHASE2_RUST_SHIM_SMOKE = (
    REPO_ROOT / ".github" / "workflows" / "phase2-rust-shim-smoke.yml"
)
GO_BOOTSTRAP_SMOKE = REPO_ROOT / "tests" / "smoke" / "go_bootstrap_smoke.go"
RUN_GO_PACKAGED_SMOKE = REPO_ROOT / "scripts" / "release" / "run_go_packaged_smoke.sh"
RUN_ALPINE_SMOKE = REPO_ROOT / "scripts" / "release" / "run_alpine_smoke.sh"
RUN_NATIVE_SMOKE = REPO_ROOT / "scripts" / "release" / "run_native_smoke.sh"
VERIFY_GLIBC_FLOOR = REPO_ROOT / "scripts" / "release" / "verify_glibc_floor.sh"
MINIFY_BUFFER_SAFETY_PROBE = (
    REPO_ROOT / "tests" / "native" / "minify_buffer_safety_probe.cpp"
)
VERIFY_MINIFY_BUFFER_SAFETY = (
    REPO_ROOT / "scripts" / "ci" / "verify_minify_buffer_safety.sh"
)


class ReleaseWorkflowContractTests(unittest.TestCase):
    def test_phase12_native_smoke_gate_tracks_durable_minify_probe(self) -> None:
        workflow_text = PHASE2_RUST_SHIM_SMOKE.read_text(encoding="utf-8")
        verifier_text = VERIFY_MINIFY_BUFFER_SAFETY.read_text(encoding="utf-8")
        probe_text = MINIFY_BUFFER_SAFETY_PROBE.read_text(encoding="utf-8")

        self.assertIn("- scripts/ci/verify_minify_buffer_safety.sh", workflow_text)
        self.assertIn("- tests/native/**", workflow_text)
        linux_smoke = workflow_text.split("  linux-smoke:", 1)[1]
        linux_smoke = linux_smoke.split("  windows-smoke:", 1)[0]
        self.assertIn(
            "bash scripts/ci/verify_minify_buffer_safety.sh",
            linux_smoke,
        )

        for production_text in (workflow_text, verifier_text, probe_text):
            self.assertNotIn(".planning/spikes", production_text)
            self.assertNotIn("total=24", production_text)

        self.assertIn("for run_number in 1 2 3", verifier_text)
        self.assertIn(
            "expected_total=$((expected_cases_per_kernel * kernel_count))",
            verifier_text,
        )
        self.assertIn("Linux:x86_64|Linux:amd64", verifier_text)
        for kernel_name in ("fallback", "haswell", "westmere"):
            self.assertIn(kernel_name, verifier_text)
        self.assertIn("-fsanitize=address,undefined", verifier_text)

        self.assertIn("constexpr size_t kCasesPerKernel = 12;", probe_text)
        self.assertIn("std::sort(supported.begin(), supported.end()", probe_text)
        self.assertIn(
            '"SUMMARY kernels=%s cases_per_kernel=%zu total=%zu failures=%zu "',
            probe_text,
        )
        self.assertIn('"sanitizer_clean=1\\n"', probe_text)

    def test_release_matrix_keeps_five_targets_and_native_smoke(self) -> None:
        workflow_text = RELEASE_WORKFLOW.read_text(encoding="utf-8")

        self.assertEqual(
            tuple(
                re.findall(
                    r"^\s+- platform_id:\s+([a-z0-9-]+)\s*$",
                    workflow_text,
                    re.MULTILINE,
                )
            ),
            (
                "linux-amd64",
                "linux-arm64",
                "darwin-amd64",
                "darwin-arm64",
                "windows-amd64",
            ),
        )

        for job_name, next_job_name in (
            ("linux-build", "darwin-build"),
            ("darwin-build", "windows-build"),
            ("windows-build", "alpine-smoke"),
        ):
            job_text = workflow_text.split(f"  {job_name}:", 1)[1]
            job_text = job_text.split(f"  {next_job_name}:", 1)[0]
            self.assertIn("bash scripts/release/run_native_smoke.sh", job_text)

    def test_packaged_smoke_reaches_abi_1_2_go_contract(self) -> None:
        workflow_text = RELEASE_WORKFLOW.read_text(encoding="utf-8")
        harness_text = RUN_GO_PACKAGED_SMOKE.read_text(encoding="utf-8")
        smoke_text = GO_BOOTSTRAP_SMOKE.read_text(encoding="utf-8")

        self.assertIn(
            "bash scripts/release/run_go_packaged_smoke.sh",
            workflow_text,
        )
        self.assertIn(
            "go run ./tests/smoke/go_bootstrap_smoke.go",
            harness_text,
        )
        self.assertIn(
            "PURE_SIMDJSON_LIB_PATH must stay unset for packaged-artifact smoke",
            harness_text,
        )
        for snippet in (
            "WithMaxCapacity",
            "WithMaxDepth",
            "TypeBigInt",
            "GetBigInt",
            "GetInt64",
            "Kernel()",
        ):
            self.assertIn(snippet, smoke_text)

    def test_alpine_smoke_installs_git_for_audited_patch_verification(self) -> None:
        script_text = RUN_ALPINE_SMOKE.read_text(encoding="utf-8")

        self.assertRegex(script_text, r"apk add --no-cache [^\n]*\bgit\b")
        self.assertIn(
            "git config --global --add safe.directory /repo\n",
            script_text,
        )
        self.assertIn(
            "git config --global --add safe.directory /repo/third_party/simdjson",
            script_text,
        )
        preflight_idx = script_text.index(
            "git -C /repo/third_party/simdjson rev-parse --verify HEAD"
        )
        build_idx = script_text.index("cargo build --release")
        self.assertLess(preflight_idx, build_idx)
        self.assertIn("cargo build --release", script_text)

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

    def test_linux_cdylib_hides_static_native_archive_symbols(self) -> None:
        build_rs = BUILD_RS.read_text(encoding="utf-8")

        self.assertIn("-Wl,--exclude-libs,ALL", build_rs)

    def test_linux_export_audit_allows_only_public_and_optional_internal_symbols(self) -> None:
        script_text = VERIFY_GLIBC_FLOOR.read_text(encoding="utf-8")

        self.assertIn("write_expected_exports", script_text)
        self.assertIn("psdj_internal_materialize_build", script_text)
        self.assertIn("psdj_internal_test_hold_materialize_guard", script_text)
        self.assertIn("expected release ABI export set", script_text)
        self.assertNotIn("_Znwm", script_text)
        self.assertNotIn("_ZdlPv", script_text)

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

    def test_release_publish_uses_rendered_changelog_body_without_auto_generated_notes(self) -> None:
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
        self.assertIn("if: startsWith(github.ref, 'refs/tags/')", render_section)
        self.assertIn("python3 scripts/release/render_release_notes.py", render_section)
        self.assertIn('--version "${{ github.ref_name }}"', render_section)
        self.assertIn('--output "${{ github.workspace }}/release-notes.md"', render_section)

        publish_section = workflow_text.split("- name: Publish GitHub release", 1)[1]
        self.assertIn("if: startsWith(github.ref, 'refs/tags/')", publish_section)
        self.assertIn("body_path: ${{ github.workspace }}/release-notes.md", publish_section)
        self.assertNotIn("generate_release_notes:", publish_section)


if __name__ == "__main__":
    unittest.main()
