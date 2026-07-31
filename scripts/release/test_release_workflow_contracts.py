#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import re
import unittest

try:
    from scripts.release.workflow_yaml import load_workflow_definition
except ModuleNotFoundError:  # pragma: no cover - direct script invocation
    from workflow_yaml import load_workflow_definition


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
PHASE3_GO_WRAPPER_SMOKE = (
    REPO_ROOT / ".github" / "workflows" / "phase3-go-wrapper-smoke.yml"
)
GO_BOOTSTRAP_SMOKE = REPO_ROOT / "tests" / "smoke" / "go_bootstrap_smoke.go"
FFI_EXPORT_SURFACE = REPO_ROOT / "tests" / "smoke" / "ffi_export_surface.c"
FFI_CONTRACT = REPO_ROOT / "docs" / "ffi-contract.md"
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


def read_workflow_contract(
    path: pathlib.Path,
) -> tuple[dict[str, tuple[str, ...]], tuple[tuple[str, str], ...]]:
    """Extract workflow events and runners through the shared YAML loader."""
    workflow = load_workflow_definition(path, REPO_ROOT)
    on = workflow.get("on")
    jobs_mapping = workflow.get("jobs")
    if not isinstance(on, dict) or not isinstance(jobs_mapping, dict):
        raise AssertionError(f"missing on/jobs mappings in {path}")

    events: dict[str, tuple[str, ...]] = {}
    for event, configuration in on.items():
        branches = configuration.get("branches", ()) if isinstance(configuration, dict) else ()
        if isinstance(branches, str):
            branches = (branches,)
        if not isinstance(branches, list | tuple) or not all(isinstance(branch, str) for branch in branches):
            raise AssertionError(f"invalid branches for {event!r} in {path}")
        events[str(event)] = tuple(branches)

    jobs: list[tuple[str, str]] = []
    for name, configuration in jobs_mapping.items():
        if not isinstance(configuration, dict):
            raise AssertionError(f"invalid job mapping for {name!r} in {path}")
        runner = configuration.get("runs-on")
        if not isinstance(runner, str):
            raise AssertionError(f"missing scalar runs-on for {name!r} in {path}")
        jobs.append((str(name), runner))
    return events, tuple(jobs)


class ReleaseWorkflowContractTests(unittest.TestCase):
    def test_release_accepts_only_final_semver_tags_and_checks_abi_state(self) -> None:
        workflow_text = RELEASE_WORKFLOW.read_text(encoding="utf-8")

        self.assertIn('^v[0-9]+(\\.[0-9]+){2}$', workflow_text)
        self.assertIn("scripts/release/check_bootstrap_abi_state.py", workflow_text)

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
            '"SUMMARY kernels=%s cases_per_kernel=%zu total=%zu failures=%zu\\n",',
            probe_text,
        )
        self.assertNotIn("sanitizer_clean", probe_text)
        self.assertNotIn("sanitizer_clean", verifier_text)

    def test_phase12_go_wrapper_smoke_keeps_five_platform_jobs(self) -> None:
        events, jobs = read_workflow_contract(PHASE3_GO_WRAPPER_SMOKE)
        self.assertEqual(events["pull_request"], ("main",))
        self.assertEqual(
            jobs,
            (
                ("linux-amd64-go-race", "ubuntu-latest"),
                ("linux-arm64-go-race", "ubuntu-24.04-arm"),
                ("darwin-amd64-go-race", "macos-15-intel"),
                ("darwin-arm64-go-race", "macos-15"),
                ("windows-amd64-go-race", "windows-latest"),
            ),
        )

    def test_phase2_rust_shim_smoke_runs_for_pull_requests_to_main(self) -> None:
        events, _ = read_workflow_contract(PHASE2_RUST_SHIM_SMOKE)
        self.assertEqual(events["pull_request"], ("main",))

    def test_phase12_ffi_smoke_invokes_all_abi_1_3_exports(self) -> None:
        smoke_text = FFI_EXPORT_SURFACE.read_text(encoding="utf-8")
        compact = re.sub(r"\s+", " ", smoke_text)
        exports = (
            (
                "element_at_pointer",
                "ELEMENT_AT_POINTER",
                "typedef pure_simdjson_error_code_t (*fn_element_at_pointer)(const pure_simdjson_value_view_t *, const uint8_t *, size_t, pure_simdjson_value_view_t *);",
            ),
            (
                "element_at_path",
                "ELEMENT_AT_PATH",
                "typedef pure_simdjson_error_code_t (*fn_element_at_path)(const pure_simdjson_value_view_t *, const uint8_t *, size_t, pure_simdjson_value_view_t *);",
            ),
            (
                "element_at_path_wildcard",
                "ELEMENT_AT_PATH_WILDCARD",
                "typedef pure_simdjson_error_code_t (*fn_element_at_path_wildcard)( const pure_simdjson_value_view_t *, const uint8_t *, size_t, pure_simdjson_value_view_t **, size_t *);",
            ),
            (
                "value_views_free",
                "VALUE_VIEWS_FREE",
                "typedef pure_simdjson_error_code_t (*fn_value_views_free)(pure_simdjson_value_view_t *, size_t);",
            ),
            (
                "array_at",
                "ARRAY_AT",
                "typedef pure_simdjson_error_code_t (*fn_array_at)(const pure_simdjson_value_view_t *, uint64_t, pure_simdjson_value_view_t *);",
            ),
            (
                "array_len",
                "ARRAY_LEN",
                "typedef pure_simdjson_error_code_t (*fn_array_len)(const pure_simdjson_value_view_t *, uint64_t *);",
            ),
            (
                "object_size",
                "OBJECT_SIZE",
                "typedef pure_simdjson_error_code_t (*fn_object_size)(const pure_simdjson_value_view_t *, uint64_t *);",
            ),
            (
                "minify",
                "MINIFY",
                "typedef pure_simdjson_error_code_t (*fn_minify)(const uint8_t *, size_t, uint8_t *, size_t, size_t *);",
            ),
            (
                "validate_utf8",
                "VALIDATE_UTF8",
                "typedef pure_simdjson_error_code_t (*fn_validate_utf8)(const uint8_t *, size_t, uint8_t *);",
            ),
        )

        for field, enum_suffix, typedef in exports:
            with self.subTest(export=field):
                self.assertIn(f'"pure_simdjson_{field}"', smoke_text)
                self.assertIn(typedef, compact)
                self.assertIn(f"fn_{field} {field};", compact)
                self.assertIn(
                    f"RESOLVE({field}, EXPORT_{enum_suffix}, fn_{field});",
                    compact,
                )
                self.assertIn(
                    f"mark_called(&exports, EXPORT_{enum_suffix});",
                    compact,
                )
                self.assertIn(f"exports.{field}(", compact)

        for semantic_check in (
            'static const uint8_t pointer_int[] = "/int";',
            'static const uint8_t path_obj_x[] = ".obj.x";',
            'static const uint8_t wildcard_arr[] = ".arr[*]";',
            "wildcard_count != 2",
            "array_len != 2",
            "object_size != 8",
            '"{\\"a\\":[1,2]}"',
            "static const uint8_t invalid_utf8[] = {0x80};",
            "utf8_valid != 1",
            "utf8_valid != 0",
        ):
            self.assertIn(semantic_check, smoke_text)

    def test_phase12_ffi_contract_requires_complete_abi_1_3_surface(self) -> None:
        contract_text = FFI_CONTRACT.read_text(encoding="utf-8")
        required_exports = (
            "pure_simdjson_element_at_pointer",
            "pure_simdjson_element_at_path",
            "pure_simdjson_element_at_path_wildcard",
            "pure_simdjson_value_views_free",
            "pure_simdjson_array_at",
            "pure_simdjson_array_len",
            "pure_simdjson_object_size",
            "pure_simdjson_minify",
            "pure_simdjson_validate_utf8",
        )

        self.assertIn(
            "normative FFI contract for `pure-simdjson` ABI 1.3 (`0x00010003`)",
            contract_text,
        )
        self.assertIn(
            "ABI 1.3 is the strict minimum for the current wrapper",
            contract_text,
        )
        self.assertIn("The ABI 1.3 build policy", contract_text)
        for symbol in required_exports:
            self.assertIn(f"- `{symbol}`", contract_text)

        for paragraph in contract_text.split("\n\n"):
            if "ABI 1.2" in paragraph or "0x00010002" in paragraph:
                classification = paragraph.lower()
                self.assertTrue(
                    "historical" in classification or "rejected" in classification,
                    f"ABI 1.2 reference is not historical/rejected: {paragraph}",
                )

        for contract in (
            "release the carrier array exactly once",
            "Exact same-start aliasing (`dst_ptr == src_ptr`) is supported",
            "must be fully disjoint; either partial-overlap direction",
            "minify is not JSON validation",
            "reject an automatically selected `fallback` implementation with status 64",
            "locks process-global selection before the empty-input return or SIMD scan",
            "Later additive ABI 1.x values may be accepted only after every wrapper-required symbol binds successfully",
            "## Wildcard copy and free",
        ):
            self.assertIn(contract, contract_text)

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
