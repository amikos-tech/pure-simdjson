package bootstrap_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
)

func TestCacheDirPerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits not meaningful on windows")
	}
	dir := filepath.Join(t.TempDir(), "subdir-needing-create")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// Mask out type bits; compare only the permission bits (ignoring sticky, etc.).
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Fatalf("expected 0700 perms, got %#o", perm)
	}
}

func TestArtifactCachePath(t *testing.T) {
	// Exported via export_test.go? No — artifactCachePath is intentionally unexported.
	// Exercise the layout through CachePath + a custom cache dir via env override.
	custom := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", custom)

	got := bootstrap.CachePath("linux", "amd64")
	want := filepath.Join(custom, "v"+bootstrap.Version, "linux-amd64", "libpure_simdjson.so")
	if got != want {
		t.Fatalf("CachePath(linux,amd64) = %q, want %q", got, want)
	}
}

func TestCachePathCurrent(t *testing.T) {
	// Sanity: with the real os.UserCacheDir path, CachePath ends with the expected
	// per-version/per-platform suffix. We don't assert the exact prefix (varies by
	// OS) — just the tail.
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", "")

	for _, p := range bootstrap.SupportedPlatforms {
		goos, goarch := p[0], p[1]
		got := bootstrap.CachePath(goos, goarch)
		if !filepath.IsAbs(got) {
			t.Errorf("CachePath(%s,%s) = %q, expected absolute path", goos, goarch, got)
		}
		wantSuffix := filepath.Join("v"+bootstrap.Version, goos+"-"+goarch)
		if !strings.Contains(got, wantSuffix) {
			t.Errorf("CachePath(%s,%s) = %q, missing suffix %q", goos, goarch, got, wantSuffix)
		}
	}
}

func TestCacheDirEnvOverride(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", custom)

	got := bootstrap.DefaultCacheDir()
	if got != custom {
		t.Fatalf("DefaultCacheDir() = %q, want env override %q", got, custom)
	}
}

func TestCacheDirEnvOverrideEmpty(t *testing.T) {
	// Empty env var should fall through to os.UserCacheDir-based path.
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", "")
	got := bootstrap.DefaultCacheDir()
	if got == "" {
		t.Fatalf("DefaultCacheDir() returned empty with unset env")
	}
}

func TestCacheDirTempDirFallbackPerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("UserCacheDir practically never fails on windows")
	}
	// Force os.UserCacheDir to fail by clearing HOME + XDG_CACHE_HOME.
	// Per os.UserCacheDir docs, on non-darwin unix it consults $XDG_CACHE_HOME or
	// falls back to $HOME/.cache; with both empty it returns an error.
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	if runtime.GOOS == "darwin" {
		// darwin uses $HOME/Library/Caches; clearing HOME is enough.
	}

	got := bootstrap.DefaultCacheDir()
	if got == "" {
		t.Fatalf("DefaultCacheDir returned empty; expected TempDir fallback")
	}
	// Fallback path must be under os.TempDir and carry the UID-scoped name.
	if !strings.HasPrefix(got, os.TempDir()) {
		t.Fatalf("fallback path %q is not under os.TempDir() %q", got, os.TempDir())
	}
	if !strings.Contains(filepath.Base(got), "pure-simdjson-") {
		t.Fatalf("fallback path %q missing UID-scoped suffix 'pure-simdjson-<uid>'", got)
	}
	// Directory should exist with 0700 perms.
	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("fallback dir not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Fatalf("fallback dir %q has perm %#o, want 0700", got, perm)
	}
}

func TestWithProcessFileLockBasic(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".lock")
	called := false
	err := bootstrap.WithProcessFileLockForTest(context.Background(), lockPath, func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("withProcessFileLock: %v", err)
	}
	if !called {
		t.Fatalf("callback not invoked")
	}
}

func TestAtomicInstall(t *testing.T) {
	cacheDir := t.TempDir()
	// tmp file must live in the same dir as the final path for os.Rename to be atomic.
	tmp, err := os.CreateTemp(cacheDir, "pure-simdjson-*.tmp")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	body := []byte("hello world")
	if _, err := tmp.Write(body); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	final := filepath.Join(cacheDir, "artifact.bin")
	if err := bootstrap.AtomicInstallForTest(tmp.Name(), final); err != nil {
		t.Fatalf("atomicInstall: %v", err)
	}

	// tmp file should be gone
	if _, err := os.Stat(tmp.Name()); !os.IsNotExist(err) {
		t.Fatalf("tmp file still present: err=%v", err)
	}
	// final file should contain the body
	got, err := os.ReadFile(final)
	if err != nil {
		t.Fatalf("read final: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("final contents = %q, want %q", got, body)
	}
}
