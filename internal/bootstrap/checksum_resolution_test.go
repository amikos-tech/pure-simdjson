package bootstrap_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
)

func TestResolveChecksumFromMirrorSHA256SUMS(t *testing.T) {
	clearBootstrapEnv(t)

	const (
		goos   = "linux"
		goarch = "amd64"
		digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	)
	key := bootstrap.ChecksumKey(bootstrap.Version, goos, goarch)
	path := "/v" + bootstrap.Version + "/SHA256SUMS"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, "%s  %s\n", digest, key)
	}))
	defer srv.Close()

	got, err := bootstrap.ResolveChecksum(
		context.Background(),
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(bootstrap.NewHTTPClientForTest()),
	)
	if err != nil {
		t.Fatalf("ResolveChecksum: %v", err)
	}
	if got != digest {
		t.Fatalf("digest = %q, want %q", got, digest)
	}
}

func TestResolveChecksumFallsBackToGitHubSHA256SUMS(t *testing.T) {
	clearBootstrapEnv(t)

	const (
		goos   = "darwin"
		goarch = "arm64"
		digest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)
	key := bootstrap.ChecksumKey(bootstrap.Version, goos, goarch)
	checksumsPath := "/v" + bootstrap.Version + "/SHA256SUMS"

	var mirrorHits atomic.Int32
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mirrorHits.Add(1)
		http.NotFound(w, r)
	}))
	defer mirror.Close()

	var githubHits atomic.Int32
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != checksumsPath {
			http.NotFound(w, r)
			return
		}
		githubHits.Add(1)
		fmt.Fprintf(w, "%s  %s\n", digest, key)
	}))
	defer github.Close()

	got, err := bootstrap.ResolveChecksum(
		context.Background(),
		bootstrap.WithMirror(mirror.URL),
		bootstrap.WithGitHubBaseURL(github.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(bootstrap.NewHTTPClientForTest()),
	)
	if err != nil {
		t.Fatalf("ResolveChecksum: %v", err)
	}
	if got != digest {
		t.Fatalf("digest = %q, want %q", got, digest)
	}
	if mirrorHits.Load() == 0 {
		t.Fatal("expected mirror checksum lookup before GitHub fallback")
	}
	if githubHits.Load() == 0 {
		t.Fatal("expected GitHub checksum fallback hit")
	}
}

func TestResolveChecksumOverrideSkipsNetwork(t *testing.T) {
	clearBootstrapEnv(t)

	const (
		goos     = "windows"
		goarch   = "amd64"
		override = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	)

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.NotFound(w, r)
	}))
	defer srv.Close()

	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, override)()

	got, err := bootstrap.ResolveChecksum(
		context.Background(),
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(bootstrap.NewHTTPClientForTest()),
	)
	if err != nil {
		t.Fatalf("ResolveChecksum: %v", err)
	}
	if got != override {
		t.Fatalf("digest = %q, want %q", got, override)
	}
	if hits.Load() != 0 {
		t.Fatalf("expected checksum override to avoid network, got %d requests", hits.Load())
	}
}

func TestResolveChecksumMissingEntryReturnsErrNoChecksum(t *testing.T) {
	clearBootstrapEnv(t)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	const (
		goos   = "linux"
		goarch = "arm64"
	)
	path := "/v" + bootstrap.Version + "/SHA256SUMS"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintln(w, "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd  v0.0.0/linux-amd64/libpure_simdjson.so")
	}))
	defer srv.Close()

	_, err := bootstrap.ResolveChecksum(
		context.Background(),
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(bootstrap.NewHTTPClientForTest()),
	)
	if err == nil {
		t.Fatal("expected missing checksum error")
	}
	if !errors.Is(err, bootstrap.ErrNoChecksum) {
		t.Fatalf("err = %v, want ErrNoChecksum", err)
	}
}
