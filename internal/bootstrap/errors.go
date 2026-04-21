// Package bootstrap implements artifact download, checksum verification, and
// cache management for the pure-simdjson shared library. Error sentinels
// defined here are canonical — the root purejson package re-exports them via
// pointer alias. Never call errors.New for these sentinels anywhere else.
package bootstrap

import "errors"

var (
	// ErrChecksumMismatch reports that a downloaded artifact's SHA-256 digest did
	// not match the authoritative expected value. Permanent: no retry on mismatch
	// (D-17, D-31).
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrAllSourcesFailed reports that all download sources (R2 + GitHub fallback)
	// were exhausted. The outer wrap at the library_loading.go boundary adds a
	// hint referencing PURE_SIMDJSON_LIB_PATH (D-21).
	ErrAllSourcesFailed = errors.New("all sources failed")

	// ErrNoChecksum reports that no authoritative digest could be resolved for the
	// requested platform/version from checksum overrides or published SHA256SUMS.
	ErrNoChecksum = errors.New("no checksum for platform")
)
