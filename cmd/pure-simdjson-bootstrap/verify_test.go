package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
)

// TestVerifyAllPlatformsDest stages a fake offline bundle on disk for all 5
// platforms under an ephemeral --dest dir, then invokes runVerify with
// allPlatforms=true, dest=<that dir>. Every artifact should pass.
// This is the M4 round-trip test: `fetch --all-platforms --dest X` then
// `verify --all-platforms --dest X` must succeed.
func TestVerifyAllPlatformsDest(t *testing.T) {
	destDir := t.TempDir()

	// Stage identical fake bytes for each of the 5 platforms; compute one hash.
	fakeBody := []byte("offline-bundle-payload")
	h := sha256.New()
	h.Write(fakeBody)
	sum := hex.EncodeToString(h.Sum(nil))

	fakeChecksums := make(map[string]string)
	for _, p := range bootstrap.SupportedPlatforms {
		goos, goarch := p[0], p[1]
		// Write the fake artifact at the expected <dest>/v<ver>/<os>-<arch>/<lib> path.
		dir := filepath.Join(destDir, "v"+bootstrap.Version, goos+"-"+goarch)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		libPath := filepath.Join(dir, platformLibraryNameForCLI(goos))
		if err := os.WriteFile(libPath, fakeBody, 0600); err != nil {
			t.Fatalf("write %s: %v", libPath, err)
		}
		fakeChecksums[bootstrap.ChecksumKey(bootstrap.Version, goos, goarch)] = sum
	}

	orig := bootstrap.Checksums
	bootstrap.Checksums = fakeChecksums
	t.Cleanup(func() { bootstrap.Checksums = orig })

	var outBuf, errBuf bytes.Buffer
	if err := runVerify(true, destDir, &outBuf, &errBuf); err != nil {
		t.Fatalf("runVerify --all-platforms --dest: %v\nstderr:\n%s", err, errBuf.String())
	}
	// Stdout should contain one PASS line per platform.
	passCount := bytes.Count(outBuf.Bytes(), []byte("PASS "))
	if passCount != 5 {
		t.Fatalf("expected 5 PASS lines, got %d\nstdout:\n%s", passCount, outBuf.String())
	}
}

// TestVerifyAllPlatformsDestMismatchFails ensures a single corrupted file in
// the offline bundle causes runVerify to return ErrChecksumMismatch.
func TestVerifyAllPlatformsDestMismatchFails(t *testing.T) {
	destDir := t.TempDir()
	fakeBody := []byte("ok-payload")
	badBody := []byte("corrupted-payload")
	h := sha256.New()
	h.Write(fakeBody)
	okSum := hex.EncodeToString(h.Sum(nil))

	fakeChecksums := make(map[string]string)
	for i, p := range bootstrap.SupportedPlatforms {
		goos, goarch := p[0], p[1]
		dir := filepath.Join(destDir, "v"+bootstrap.Version, goos+"-"+goarch)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		libPath := filepath.Join(dir, platformLibraryNameForCLI(goos))
		body := fakeBody
		if i == 2 { // flip one platform to bad bytes
			body = badBody
		}
		if err := os.WriteFile(libPath, body, 0600); err != nil {
			t.Fatalf("write %s: %v", libPath, err)
		}
		fakeChecksums[bootstrap.ChecksumKey(bootstrap.Version, goos, goarch)] = okSum
	}

	orig := bootstrap.Checksums
	bootstrap.Checksums = fakeChecksums
	t.Cleanup(func() { bootstrap.Checksums = orig })

	var outBuf, errBuf bytes.Buffer
	err := runVerify(true, destDir, &outBuf, &errBuf)
	if err == nil {
		t.Fatal("expected error from corrupted bundle, got nil")
	}
	if !errors.Is(err, bootstrap.ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
}

// TestVerifyCurrentPlatformDefault sanity-checks the no-flag path: current
// platform only, default OS cache dir (redirected via PURE_SIMDJSON_CACHE_DIR).
func TestVerifyCurrentPlatformDefault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", tmp)

	fakeBody := []byte("current-platform-payload")
	h := sha256.New()
	h.Write(fakeBody)
	sum := hex.EncodeToString(h.Sum(nil))

	dir := filepath.Join(tmp, "v"+bootstrap.Version, runtime.GOOS+"-"+runtime.GOARCH)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	libPath := filepath.Join(dir, platformLibraryNameForCLI(runtime.GOOS))
	if err := os.WriteFile(libPath, fakeBody, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	orig := bootstrap.Checksums
	bootstrap.Checksums = map[string]string{
		bootstrap.ChecksumKey(bootstrap.Version, runtime.GOOS, runtime.GOARCH): sum,
	}
	t.Cleanup(func() { bootstrap.Checksums = orig })

	var outBuf, errBuf bytes.Buffer
	if err := runVerify(false, "", &outBuf, &errBuf); err != nil {
		t.Fatalf("runVerify default: %v\nstderr:\n%s", err, errBuf.String())
	}
	if !bytes.Contains(outBuf.Bytes(), []byte("PASS ")) {
		t.Fatalf("expected PASS in stdout, got: %s", outBuf.String())
	}
}
