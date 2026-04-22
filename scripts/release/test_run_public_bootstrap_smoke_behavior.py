#!/usr/bin/env python3

from __future__ import annotations

import hashlib
import os
import pathlib
import subprocess
import tempfile
import textwrap
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "release" / "run_public_bootstrap_smoke.sh"


class RunPublicBootstrapSmokeBehaviorTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmpdir = tempfile.TemporaryDirectory()
        self.root = pathlib.Path(self.tmpdir.name)
        self.repo_root = self.root / "repo"
        self.repo_root.mkdir()
        smoke_dir = self.repo_root / "tests" / "smoke"
        smoke_dir.mkdir(parents=True)
        (smoke_dir / "go_bootstrap_smoke.go").write_text("// stub smoke entrypoint\n", encoding="utf-8")

        self.bin_dir = self.root / "bin"
        self.bin_dir.mkdir()
        self.runner_temp = self.root / "runner-temp"
        self.runner_temp.mkdir()
        self.home = self.root / "home"
        self.home.mkdir()
        self.summary_path = self.root / "summary.md"

        self.write_fake_go()
        self.write_fake_curl()
        self.write_fake_sha256sum()

        self.base_env = os.environ.copy()
        # Ambient dev overrides would either abort the wrapper or mask the path under test.
        for leaked in (
            "PURE_SIMDJSON_LIB_PATH",
            "PURE_SIMDJSON_BINARY_MIRROR",
            "PURE_SIMDJSON_DISABLE_GH_FALLBACK",
            "PURE_SIMDJSON_CACHE_DIR",
        ):
            self.base_env.pop(leaked, None)
        self.base_env.update(
            {
                "HOME": str(self.home),
                "RUNNER_TEMP": str(self.runner_temp),
                "GITHUB_STEP_SUMMARY": str(self.summary_path),
                "PATH": os.pathsep.join([str(self.bin_dir), self.base_env.get("PATH", "")]),
                "TEST_VERSION": "1.2.3",
                "TEST_GOOS": "linux",
                "TEST_GOARCH": "amd64",
                "TEST_LIBNAME": "libpure_simdjson.so",
                "TEST_ARTIFACT_CONTENT": "bootstrap smoke artifact",
            }
        )

    def tearDown(self) -> None:
        self.tmpdir.cleanup()

    def write_executable(self, path: pathlib.Path, contents: str) -> None:
        path.write_text(contents, encoding="utf-8")
        path.chmod(0o755)

    def write_fake_go(self) -> None:
        self.write_executable(
            self.bin_dir / "go",
            textwrap.dedent(
                """\
                #!/usr/bin/env bash
                set -euo pipefail

                if [[ "${1:-}" != "run" || "${2:-}" != "./tests/smoke/go_bootstrap_smoke.go" ]]; then
                  echo "unexpected go invocation: $*" >&2
                  exit 1
                fi

                version="${TEST_VERSION:?}"
                goos="${TEST_GOOS:?}"
                goarch="${TEST_GOARCH:?}"
                libname="${TEST_LIBNAME:?}"
                artifact="${PURE_SIMDJSON_CACHE_DIR}/v${version}/${goos}-${goarch}/${libname}"

                mkdir -p "$(dirname "$artifact")"
                if [[ "${TEST_FAKE_GO_ZERO_BYTE:-0}" == "1" ]]; then
                  : > "$artifact"
                else
                  printf '%s' "${TEST_ARTIFACT_CONTENT:-bootstrap smoke artifact}" > "$artifact"
                fi

                chmod "${TEST_FAKE_GO_DIR_MODE:-700}" \
                  "${PURE_SIMDJSON_CACHE_DIR}" \
                  "${PURE_SIMDJSON_CACHE_DIR}/v${version}" \
                  "${PURE_SIMDJSON_CACHE_DIR}/v${version}/${goos}-${goarch}"
                """
            ),
        )

    def write_fake_curl(self) -> None:
        self.write_executable(
            self.bin_dir / "curl",
            textwrap.dedent(
                """\
                #!/usr/bin/env bash
                set -euo pipefail

                url="${@: -1}"
                if [[ "$*" == *"%{http_code}"* ]]; then
                  if [[ "$url" == *"pure-simdjson-does-not-exist"* ]]; then
                    printf '%s' "${TEST_BROKEN_MIRROR_STATUS:-404}"
                    exit 0
                  fi
                  printf '%s' "${TEST_HTTP_STATUS:-200}"
                  exit 0
                fi

                if [[ "$url" == *"/SHA256SUMS" ]]; then
                  printf '%s' "${TEST_SHA256SUMS_BODY:-}"
                  exit 0
                fi

                echo "unexpected curl invocation: $*" >&2
                exit 1
                """
            ),
        )

    def write_fake_sha256sum(self) -> None:
        self.write_executable(
            self.bin_dir / "sha256sum",
            textwrap.dedent(
                """\
                #!/usr/bin/env python3
                import hashlib
                import pathlib
                import sys

                for raw_path in sys.argv[1:]:
                    path = pathlib.Path(raw_path)
                    digest = hashlib.sha256(path.read_bytes()).hexdigest()
                    print(f"{digest}  {raw_path}")
                """
            ),
        )

    def run_script(self, *, mode: str, cache_dir: pathlib.Path, extra_env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
        env = self.base_env.copy()
        if extra_env:
            env.update(extra_env)

        return subprocess.run(
            [
                "bash",
                str(SCRIPT_PATH),
                "--repo-root",
                str(self.repo_root),
                "--version",
                env["TEST_VERSION"],
                "--goos",
                env["TEST_GOOS"],
                "--goarch",
                env["TEST_GOARCH"],
                "--mode",
                mode,
                "--cache-dir",
                str(cache_dir),
            ],
            cwd=REPO_ROOT,
            check=False,
            capture_output=True,
            text=True,
            env=env,
        )

    def test_r2_mode_validates_non_empty_artifact_and_checksum(self) -> None:
        cache_dir = self.runner_temp / "cache"
        artifact_digest = hashlib.sha256(b"bootstrap smoke artifact").hexdigest()
        checksums_body = f"{artifact_digest}  v1.2.3/linux-amd64/libpure_simdjson.so\n"

        result = self.run_script(
            mode="r2",
            cache_dir=cache_dir,
            extra_env={"TEST_SHA256SUMS_BODY": checksums_body},
        )

        self.assertEqual(result.returncode, 0, msg=result.stderr)
        self.assertIn("public bootstrap validation", self.summary_path.read_text(encoding="utf-8"))

    def test_github_fallback_mode_requires_explicit_404_proof(self) -> None:
        cache_dir = self.runner_temp / "fallback-cache"

        result = self.run_script(
            mode="github-fallback",
            cache_dir=cache_dir,
            extra_env={"TEST_BROKEN_MIRROR_STATUS": "200"},
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("requires explicit broken-mirror 404 proof", result.stderr)

    def test_refuses_existing_cache_dir_outside_runner_temp(self) -> None:
        cache_dir = self.root / "existing-cache"
        cache_dir.mkdir()

        result = self.run_script(
            mode="r2",
            cache_dir=cache_dir,
            extra_env={"TEST_SHA256SUMS_BODY": "deadbeef  v1.2.3/linux-amd64/libpure_simdjson.so\n"},
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("refusing existing cache dir outside RUNNER_TEMP", result.stderr)

    def test_rejects_zero_byte_artifact(self) -> None:
        cache_dir = self.runner_temp / "zero-byte-cache"

        result = self.run_script(
            mode="r2",
            cache_dir=cache_dir,
            extra_env={
                "TEST_FAKE_GO_ZERO_BYTE": "1",
                "TEST_SHA256SUMS_BODY": "deadbeef  v1.2.3/linux-amd64/libpure_simdjson.so\n",
            },
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("expected cache artifact missing or empty", result.stderr)

    def test_detects_sha256_mismatch_when_artifact_tampered(self) -> None:
        # SHA256SUMS advertises the digest of the *expected* bytes, but the fake `go`
        # writes different content — as a tampered mirror would. The wrapper's
        # independent re-hash must catch the mismatch.
        cache_dir = self.runner_temp / "tamper-cache"
        expected_digest = hashlib.sha256(b"bootstrap smoke artifact").hexdigest()
        checksums_body = f"{expected_digest}  v1.2.3/linux-amd64/libpure_simdjson.so\n"

        result = self.run_script(
            mode="r2",
            cache_dir=cache_dir,
            extra_env={
                "TEST_ARTIFACT_CONTENT": "tampered bytes",
                "TEST_SHA256SUMS_BODY": checksums_body,
            },
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("cached artifact digest mismatch", result.stderr)

    def test_rejects_missing_checksum_entry(self) -> None:
        cache_dir = self.runner_temp / "missing-entry-cache"
        artifact_digest = hashlib.sha256(b"bootstrap smoke artifact").hexdigest()
        checksums_body = f"{artifact_digest}  v1.2.3/linux-amd64/some-other-lib.so\n"

        result = self.run_script(
            mode="r2",
            cache_dir=cache_dir,
            extra_env={"TEST_SHA256SUMS_BODY": checksums_body},
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("checksum entry not found", result.stderr)

    def test_rejects_malformed_checksum_entry(self) -> None:
        cache_dir = self.runner_temp / "malformed-entry-cache"

        result = self.run_script(
            mode="r2",
            cache_dir=cache_dir,
            extra_env={
                "TEST_SHA256SUMS_BODY": "1234  v1.2.3/linux-amd64/libpure_simdjson.so\n",
            },
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("is not a lowercase SHA-256 digest", result.stderr)

    def test_refuses_unsafe_raw_cache_dir_inputs(self) -> None:
        for raw in (".", "..", "/"):
            with self.subTest(raw=raw):
                result = self.run_script(
                    mode="r2",
                    cache_dir=pathlib.Path(raw),
                    extra_env={"TEST_SHA256SUMS_BODY": "deadbeef  entry\n"},
                )

                self.assertNotEqual(result.returncode, 0)
                self.assertIn("refusing unsafe cache dir", result.stderr)

    def test_refuses_home_and_repo_root_as_cache_dir(self) -> None:
        for label, path in (("home", self.home), ("repo-root", self.repo_root)):
            with self.subTest(label=label):
                result = self.run_script(
                    mode="r2",
                    cache_dir=path,
                    extra_env={"TEST_SHA256SUMS_BODY": "deadbeef  entry\n"},
                )

                self.assertNotEqual(result.returncode, 0)
                self.assertIn("refusing unsafe cache dir", result.stderr)

    def test_rejects_incorrect_unix_permissions(self) -> None:
        cache_dir = self.runner_temp / "bad-perms-cache"
        artifact_digest = hashlib.sha256(b"bootstrap smoke artifact").hexdigest()
        checksums_body = f"{artifact_digest}  v1.2.3/linux-amd64/libpure_simdjson.so\n"

        result = self.run_script(
            mode="r2",
            cache_dir=cache_dir,
            extra_env={
                "TEST_FAKE_GO_DIR_MODE": "755",
                "TEST_SHA256SUMS_BODY": checksums_body,
            },
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("expected unix permission 700", result.stderr)


if __name__ == "__main__":
    unittest.main()
