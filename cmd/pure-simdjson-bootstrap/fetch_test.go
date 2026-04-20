package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
)

func fakeHex(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

// TestFetchCmd verifies that runFetch with allPlatforms=true downloads all 5
// artifacts to --dest. Uses httptest.Server as the mirror.
// This is the DIST-08 integration test.
func TestFetchCmd(t *testing.T) {
	fakeBody := []byte("fake-library-bytes-for-fetch-test")
	fakeSum := fakeHex(fakeBody)

	// Build a mux serving all 5 platform artifacts at the R2 path layout:
	//   /v<Version>/<goos>-<goarch>/<platformLibraryName>
	mux := http.NewServeMux()
	var hits atomic.Int32
	for _, p := range bootstrap.SupportedPlatforms {
		goos, goarch := p[0], p[1]
		urlPath := "/v" + bootstrap.Version + "/" + goos + "-" + goarch + "/" + platformLibraryNameForCLI(goos)
		mux.HandleFunc(urlPath, func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			_, _ = w.Write(fakeBody)
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	destDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")
	t.Setenv("PURE_SIMDJSON_BINARY_MIRROR", srv.URL)

	// Replace Checksums with fake values so verification passes.
	origChecksums := bootstrap.Checksums
	fakeChecksums := make(map[string]string)
	for _, p := range bootstrap.SupportedPlatforms {
		k := bootstrap.ChecksumKey(bootstrap.Version, p[0], p[1])
		fakeChecksums[k] = fakeSum
	}
	bootstrap.Checksums = fakeChecksums
	t.Cleanup(func() { bootstrap.Checksums = origChecksums })

	if err := runFetch(context.Background(), true, nil, destDir, "", srv.URL, io.Discard); err != nil {
		t.Fatalf("runFetch --all-platforms: %v", err)
	}
	if got := hits.Load(); got != 5 {
		t.Fatalf("expected 5 downloads (one per platform), got %d", got)
	}
	// Verify all 5 artifacts exist at destDir at their expected platform paths.
	for _, p := range bootstrap.SupportedPlatforms {
		want := filepath.Join(destDir, "v"+bootstrap.Version, p[0]+"-"+p[1], platformLibraryNameForCLI(p[0]))
		if _, err := os.Stat(want); err != nil {
			t.Errorf("artifact missing for %s/%s at %s: %v", p[0], p[1], want, err)
		}
	}
}

// TestFetchCmdSingleTarget verifies --target=linux/amd64 downloads exactly one
// platform.
func TestFetchCmdSingleTarget(t *testing.T) {
	fakeBody := []byte("single-target-bytes")
	fakeSum := fakeHex(fakeBody)

	mux := http.NewServeMux()
	var hits atomic.Int32
	urlPath := "/v" + bootstrap.Version + "/linux-amd64/libpure_simdjson.so"
	mux.HandleFunc(urlPath, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write(fakeBody)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	destDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")
	t.Setenv("PURE_SIMDJSON_BINARY_MIRROR", srv.URL)
	orig := bootstrap.Checksums
	bootstrap.Checksums = map[string]string{
		bootstrap.ChecksumKey(bootstrap.Version, "linux", "amd64"): fakeSum,
	}
	t.Cleanup(func() { bootstrap.Checksums = orig })

	if err := runFetch(context.Background(), false, []string{"linux/amd64"}, destDir, "", srv.URL, io.Discard); err != nil {
		t.Fatalf("runFetch --target linux/amd64: %v", err)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected 1 download, got %d", got)
	}
}
