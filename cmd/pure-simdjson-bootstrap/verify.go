package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
	"github.com/spf13/cobra"
)

// newVerifyCmd wires the `verify` subcommand (D-25, M4). With no flags it
// hashes the current-platform artifact in the default cache. With
// --all-platforms it iterates bootstrap.SupportedPlatforms; --dest redirects
// the cache base to a caller-supplied directory so offline bundles produced by
// `fetch --all-platforms --dest` can be round-trip verified.
func newVerifyCmd() *cobra.Command {
	var (
		allPlatforms bool
		dest         string
	)
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Re-verify SHA-256 of cached artifacts against embedded checksums",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify(allPlatforms, dest, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().BoolVar(&allPlatforms, "all-platforms", false, "verify all 5 platforms (requires matching files to exist)")
	cmd.Flags().StringVar(&dest, "dest", "", "cache base directory (default: OS user cache); useful for offline bundles produced by 'fetch --dest'")
	return cmd
}

// verifyOne hashes path and compares against expected hex digest.
// Returns nil on match, bootstrap.ErrChecksumMismatch on mismatch, or an fs
// error if path is missing.
func verifyOne(path, expected string, outW, errW io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash %s: %w", path, err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expected {
		fmt.Fprintf(errW, "FAIL %s: expected %s, got %s\n", path, expected, got)
		return bootstrap.ErrChecksumMismatch
	}
	fmt.Fprintf(outW, "PASS %s\n", path)
	return nil
}

// artifactPath returns the absolute path to the artifact for the given platform,
// using either the OS user cache (dest == "") or the caller-supplied
// destination directory. The layout mirrors `fetch --dest`.
func artifactPath(dest, goos, goarch string) string {
	if dest == "" {
		return bootstrap.CachePath(goos, goarch)
	}
	return filepath.Join(dest,
		"v"+bootstrap.Version,
		goos+"-"+goarch,
		platformLibraryNameForCLI(goos),
	)
}

// platformLibraryNameForCLI duplicates the minimal on-disk filename logic from
// internal/bootstrap/url.go because platformLibraryName is unexported there
// (D-10 locks the name set). Keep in sync. This lives in verify.go because the
// only cmd/ callers that need it are verify and the integration tests.
func platformLibraryNameForCLI(goos string) string {
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

// runVerify dispatches between the current-platform and --all-platforms paths.
// Any mismatch or missing file is an error; the first error encountered wins
// but all platforms are still reported for diagnostics.
func runVerify(allPlatforms bool, dest string, outW, errW io.Writer) error {
	if !allPlatforms {
		goos, goarch := runtime.GOOS, runtime.GOARCH
		key := bootstrap.ChecksumKey(bootstrap.Version, goos, goarch)
		expected, ok := bootstrap.Checksums[key]
		if !ok {
			return fmt.Errorf("%w: %s", bootstrap.ErrNoChecksum, key)
		}
		return verifyOne(artifactPath(dest, goos, goarch), expected, outW, errW)
	}

	// M4: iterate all 5 platforms. Any mismatch or missing file is a failure.
	var firstErr error
	for _, p := range bootstrap.SupportedPlatforms {
		goos, goarch := p[0], p[1]
		key := bootstrap.ChecksumKey(bootstrap.Version, goos, goarch)
		expected, ok := bootstrap.Checksums[key]
		if !ok {
			if firstErr == nil {
				firstErr = fmt.Errorf("%w: %s", bootstrap.ErrNoChecksum, key)
			}
			fmt.Fprintf(errW, "MISS %s/%s: %v\n", goos, goarch, bootstrap.ErrNoChecksum)
			continue
		}
		if err := verifyOne(artifactPath(dest, goos, goarch), expected, outW, errW); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
