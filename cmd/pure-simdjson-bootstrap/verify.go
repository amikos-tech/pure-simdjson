package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
	"github.com/spf13/cobra"
)

var (
	resolveChecksumTimeout = 30 * time.Second
	resolveChecksumFn      = bootstrap.ResolveChecksum
)

// newVerifyCmd wires the `verify` subcommand (D-25, M4). With no flags it
// hashes the current-platform artifact in the default cache. With
// --all-platforms it iterates bootstrap.SupportedPlatforms; --dest redirects
// the cache base to a caller-supplied directory so downloaded bundles can be
// re-verified against a local SHA256SUMS or published release metadata.
func newVerifyCmd() *cobra.Command {
	var (
		allPlatforms bool
		dest         string
	)
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Re-verify SHA-256 of cached artifacts against release metadata",
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
		bootstrap.PlatformLibraryName(goos),
	)
}

func checksumFromLocalSums(dest, goos, goarch string) (string, bool, error) {
	if dest == "" {
		return "", false, nil
	}
	sumsPath := filepath.Join(dest, "v"+bootstrap.Version, "SHA256SUMS")
	f, err := os.Open(sumsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("open %s: %w", sumsPath, err)
	}
	defer f.Close()

	key := bootstrap.ChecksumKey(bootstrap.Version, goos, goarch)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || fields[1] != key {
			continue
		}
		digest := strings.ToLower(fields[0])
		if !bootstrap.LooksLikeSHA256Hex(digest) {
			return "", false, fmt.Errorf("invalid SHA256SUMS digest for %s in %s: %q", key, sumsPath, fields[0])
		}
		return digest, true, nil
	}
	if err := scanner.Err(); err != nil {
		return "", false, fmt.Errorf("scan %s: %w", sumsPath, err)
	}
	return "", false, nil
}

func expectedChecksum(dest, goos, goarch string) (string, error) {
	if digest, ok, err := checksumFromLocalSums(dest, goos, goarch); err != nil {
		return "", err
	} else if ok {
		return digest, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), resolveChecksumTimeout)
	defer cancel()
	return resolveChecksumFn(
		ctx,
		bootstrap.WithVersion(bootstrap.Version),
		bootstrap.WithTarget(goos, goarch),
	)
}

// runVerify dispatches between the current-platform and --all-platforms paths.
// Any mismatch or missing file is an error; the first error encountered wins
// but all platforms are still reported for diagnostics.
func runVerify(allPlatforms bool, dest string, outW, errW io.Writer) error {
	if !allPlatforms {
		goos, goarch := runtime.GOOS, runtime.GOARCH
		expected, err := expectedChecksum(dest, goos, goarch)
		if err != nil {
			return err
		}
		return verifyOne(artifactPath(dest, goos, goarch), expected, outW, errW)
	}

	// M4: iterate all 5 platforms. Any mismatch or missing file is a failure.
	var firstErr error
	for _, p := range bootstrap.SupportedPlatforms {
		goos, goarch := p[0], p[1]
		expected, err := expectedChecksum(dest, goos, goarch)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			fmt.Fprintf(errW, "MISS %s/%s: %v\n", goos, goarch, err)
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
