package bootstrap

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

const (
	defaultR2BaseURL     = "https://releases.amikos.tech/pure-simdjson"
	defaultGitHubBaseURL = "https://github.com/amikos-tech/pure-simdjson/releases/download"
)

// SupportedPlatforms lists the five release targets (DIST-01).
var SupportedPlatforms = [][2]string{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"windows", "amd64"},
}

// PlatformLibraryName returns the on-disk library filename for the given GOOS (D-10).
// This is the name the file has in the CACHE, not on the GitHub release.
// Exported so the CLI (cmd/pure-simdjson-bootstrap) can construct cache paths
// without redeclaring the name set.
func PlatformLibraryName(goos string) string {
	switch goos {
	case "darwin":
		return "libpure_simdjson.dylib"
	case "linux":
		return "libpure_simdjson.so"
	case "windows":
		return "pure_simdjson-msvc.dll"
	default:
		return "libpure_simdjson"
	}
}

// githubAssetName returns the GitHub release asset filename for the given platform.
// GitHub release assets live in a flat namespace — platform must be encoded in the
// filename itself to avoid collision between e.g. linux/amd64 and linux/arm64 (H1).
// Examples:
//
//	linux/amd64   -> libpure_simdjson-linux-amd64.so
//	linux/arm64   -> libpure_simdjson-linux-arm64.so
//	darwin/amd64  -> libpure_simdjson-darwin-amd64.dylib
//	darwin/arm64  -> libpure_simdjson-darwin-arm64.dylib
//	windows/amd64 -> pure_simdjson-windows-amd64-msvc.dll
func githubAssetName(goos, goarch string) string {
	switch goos {
	case "darwin":
		return fmt.Sprintf("libpure_simdjson-%s-%s.dylib", goos, goarch)
	case "linux":
		return fmt.Sprintf("libpure_simdjson-%s-%s.so", goos, goarch)
	case "windows":
		// Windows CI builds pure_simdjson-msvc.dll locally; at release upload time
		// the release workflow renames it to pure_simdjson-<goos>-<goarch>-msvc.dll for the flat
		// GH asset namespace.
		return fmt.Sprintf("pure_simdjson-%s-%s-msvc.dll", goos, goarch)
	default:
		return fmt.Sprintf("libpure_simdjson-%s-%s", goos, goarch)
	}
}

// r2ArtifactURL constructs the R2 primary download URL (DIST-01).
// Layout: <baseURL>/v<version>/<os>-<arch>/<PlatformLibraryName>
// The <os>-<arch>/ directory provides namespacing; the file segment can reuse
// PlatformLibraryName because directories prevent collision.
func r2ArtifactURL(baseURL, version, goos, goarch string) string {
	osArch := goos + "-" + goarch
	lib := PlatformLibraryName(goos)
	return fmt.Sprintf("%s/v%s/%s/%s",
		strings.TrimRight(baseURL, "/"), version, osArch, lib)
}

// githubArtifactURL constructs the GitHub Releases fallback URL (DIST-02).
// GitHub release assets are flat — asset name must be platform-tagged via
// githubAssetName to avoid collision (H1 fix).
// Layout: <baseURL>/v<version>/<githubAssetName(goos, goarch)>
func githubArtifactURL(baseURL, version, goos, goarch string) string {
	base := baseURL
	if base == "" {
		base = defaultGitHubBaseURL
	}
	asset := githubAssetName(goos, goarch)
	return fmt.Sprintf("%s/v%s/%s",
		strings.TrimRight(base, "/"), version, asset)
}

// r2ChecksumsURL constructs the published SHA256SUMS URL under the raw R2 tree.
func r2ChecksumsURL(baseURL, version string) string {
	return fmt.Sprintf("%s/v%s/SHA256SUMS", strings.TrimRight(baseURL, "/"), version)
}

// githubChecksumsURL constructs the GitHub Releases SHA256SUMS URL.
func githubChecksumsURL(baseURL, version string) string {
	base := baseURL
	if base == "" {
		base = defaultGitHubBaseURL
	}
	return fmt.Sprintf("%s/v%s/SHA256SUMS", strings.TrimRight(base, "/"), version)
}

// ChecksumKey returns the map key used in Checksums for a given platform (D-08).
// Format: "v<version>/<goos>-<goarch>/<PlatformLibraryName>"
// EXPORTED (uppercase) because the CLI in cmd/pure-simdjson-bootstrap (a
// separate package) needs it for the `verify` subcommand.
func ChecksumKey(version, goos, goarch string) string {
	return fmt.Sprintf("v%s/%s-%s/%s", version, goos, goarch, PlatformLibraryName(goos))
}

// validateBaseURL rejects non-HTTPS URLs except for loopback hosts (tests).
// Source: pure-onnx@v0.0.1/ort/bootstrap.go validateBootstrapBaseURL pattern.
func validateBaseURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid mirror URL %q: %w", rawURL, err)
	}
	if strings.EqualFold(u.Scheme, "https") {
		return nil
	}
	if strings.EqualFold(u.Scheme, "http") && isLoopbackHost(u.Hostname()) {
		return nil // allow http://localhost for tests
	}
	return fmt.Errorf("mirror URL must use HTTPS for non-loopback hosts: %s", rawURL)
}

// isLoopbackHost reports whether host is a loopback address or hostname. The
// hostname fast path handles "localhost"; everything else falls through to
// net.IP.IsLoopback which covers the full 127.0.0.0/8 range and ::1 per
// RFC 5735 / RFC 4291.
func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
