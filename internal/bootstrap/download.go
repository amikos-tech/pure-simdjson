package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// userAgent is stamped on every outbound HTTP request so R2/GitHub server-side
// telemetry can identify the library (L3 from 05-REVIEWS.md). The version
// suffix tracks releases 1:1 via the bootstrap package Version constant.
const userAgent = "pure-simdjson-go/v" + Version

// maxBootstrapArtifactBytes caps the downloaded body. A single .so/.dylib/.dll
// for pure-simdjson is well under this.
const maxBootstrapArtifactBytes int64 = 128 << 20 // 128 MiB

// HTTP client timing parameters. Values match pure-onnx for consistency; each
// request also carries the caller's ctx so cancellation trumps these.
const (
	httpDialTimeout       = 30 * time.Second
	httpTLSHandshake      = 10 * time.Second
	httpResponseHeader    = 30 * time.Second
	httpIdleConnTimeout   = 90 * time.Second
	httpOverallTimeout    = 2 * time.Minute
	bootstrapRetryBaseMS  = 500
	bootstrapRetryAttempt = 3
)

// redirectDowngradeError is returned by rejectHTTPSDowngrade and detected by
// isBootstrapRedirectPolicyError — used so the retry loop can tag the error
// permanent instead of retrying against the same hostile upstream.
var errHTTPSDowngrade = errors.New("redirect from HTTPS to HTTP rejected")

// newHTTPClient builds the download client. CheckRedirect rejects any hop from
// HTTPS to HTTP as a permanent error (T-05-04). The Transport's nested
// timeouts bound each phase of a single request; the outer Timeout bounds the
// whole request (ctx layered on top in downloadOnce).
func newHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: httpDialTimeout, KeepAlive: 30 * time.Second}
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       httpIdleConnTimeout,
		TLSHandshakeTimeout:   httpTLSHandshake,
		ResponseHeaderTimeout: httpResponseHeader,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport:     tr,
		Timeout:       httpOverallTimeout,
		CheckRedirect: rejectHTTPSDowngrade,
	}
}

// rejectHTTPSDowngrade fails fast on HTTPS → HTTP hops.
func rejectHTTPSDowngrade(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return nil
	}
	// The originating request's scheme drives the policy. If we started on
	// HTTPS, every hop must remain HTTPS.
	if strings.EqualFold(via[0].URL.Scheme, "https") && !strings.EqualFold(req.URL.Scheme, "https") {
		return errHTTPSDowngrade
	}
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	return nil
}

// isBootstrapRedirectPolicyError unwraps the http.Client wrapping to spot our
// downgrade sentinel.
func isBootstrapRedirectPolicyError(err error) bool {
	if err == nil {
		return false
	}
	// http.Client wraps CheckRedirect errors in *url.Error whose Err field is
	// the original error. errors.Is walks the chain and works for both the
	// direct sentinel and the *url.Error wrap.
	if errors.Is(err, errHTTPSDowngrade) {
		return true
	}
	var uerr *url.Error
	if errors.As(err, &uerr) {
		return errors.Is(uerr.Err, errHTTPSDowngrade)
	}
	return false
}

// sleepWithJitter implements D-13 Full-Jitter + D-14 ctx-aware select. The
// delay is capped at 8s so a stuck retry loop never dominates a NewParser call.
// math/rand/v2 is auto-seeded in Go 1.22+; no manual seeding needed.
func sleepWithJitter(ctx context.Context, attempt int) error {
	const (
		base = 500 * time.Millisecond
		cap  = 8 * time.Second
	)
	shift := uint(attempt)
	if shift > 10 {
		shift = 10
	}
	expBackoff := base << shift
	if expBackoff > cap || expBackoff < 0 {
		expBackoff = cap
	}
	jitter := time.Duration(mrand.Float64() * float64(base))
	d := expBackoff + jitter

	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// isRetryable classifies an HTTP response as worth retrying. The 403 body-sniff
// handles the common GitHub-release rate-limit case (Pattern 6); headers and
// body snippet are consulted together.
func isRetryable(statusCode int, headers http.Header, bodySnippet string) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	}
	if statusCode >= 500 {
		return true
	}
	if statusCode == http.StatusForbidden {
		if headers != nil {
			if headers.Get("Retry-After") != "" {
				return true
			}
			if strings.TrimSpace(headers.Get("X-RateLimit-Remaining")) == "0" {
				return true
			}
		}
		lower := strings.ToLower(bodySnippet)
		if strings.Contains(lower, "rate limit exceeded") ||
			strings.Contains(lower, "secondary rate limit") {
			return true
		}
	}
	return false
}

// downloadAndVerify resolves URLs, drives the retry loop, verifies the SHA-256,
// and atomically installs the artifact.
func downloadAndVerify(ctx context.Context, cfg bootstrapConfig, cachePath string) error {
	// Compute URLs. Empty mirror falls back to the default R2 base.
	r2Base := cfg.mirrorURL
	if r2Base == "" {
		r2Base = defaultR2BaseURL
	}
	primary := r2ArtifactURL(r2Base, cfg.version, cfg.goos, cfg.goarch)

	fallback := ""
	if !cfg.disableGH {
		fallback = githubArtifactURL(cfg.githubBaseURL, cfg.version, cfg.goos, cfg.goarch)
	}

	cacheDir := filepath.Dir(cachePath)
	tmpPath, digest, err := downloadWithRetry(ctx, cfg, primary, fallback, cacheDir)
	if err != nil {
		return err
	}

	// Verify SHA-256 against the embedded Checksums map.
	key := ChecksumKey(cfg.version, cfg.goos, cfg.goarch)
	expected, ok := Checksums[key]
	if !ok {
		_ = os.Remove(tmpPath)
		return markPermanentBootstrapError(
			fmt.Errorf("%w: %s", ErrNoChecksum, key))
	}
	if !strings.EqualFold(digest, expected) {
		_ = os.Remove(tmpPath)
		return markPermanentBootstrapError(
			fmt.Errorf("%w: expected %s, got %s for %s",
				ErrChecksumMismatch, expected, digest, key))
	}

	return atomicInstall(tmpPath, cachePath)
}

// downloadWithRetry runs the primary-then-fallback attempt ladder (D-15): up to
// 3 attempts against the R2 URL, then up to 3 against the GitHub fallback.
// Backoff is Full-Jitter, capped at 8s, and interruptible via ctx.
//
// Permanent-error semantics on the outer loop:
//   - Ladder-fatal (checksum mismatch, redirect downgrade, oversized body,
//     request-construction failure) → abort the whole ladder. These do not get
//     "better" on a different URL.
//   - Per-URL fatal (non-retryable HTTP status, e.g. 404) → skip remaining
//     retries for this URL and move to the next one. Per Fault Injection Matrix
//     item 9 the GH fallback MUST fire when R2 returns 404.
func downloadWithRetry(ctx context.Context, cfg bootstrapConfig, primaryURL, fallbackURL, cacheDir string) (tmpPath, digest string, err error) {
	urls := []string{primaryURL}
	if fallbackURL != "" {
		urls = append(urls, fallbackURL)
	}

	var lastErr error
	for _, u := range urls {
		for attempt := 0; attempt < bootstrapRetryAttempt; attempt++ {
			if attempt > 0 {
				if sleepErr := sleepWithJitter(ctx, attempt); sleepErr != nil {
					return "", "", sleepErr
				}
			}
			if err := ctx.Err(); err != nil {
				return "", "", err
			}
			tmp, hexDigest, oneErr := downloadOnce(ctx, cfg, u, cacheDir)
			if oneErr == nil {
				return tmp, hexDigest, nil
			}
			lastErr = fmt.Errorf("attempt %d/%d %s: %w",
				attempt+1, bootstrapRetryAttempt, u, oneErr)
			if isLadderFatalError(oneErr) {
				return "", "", markPermanentBootstrapError(lastErr)
			}
			if isPermanentBootstrapError(oneErr) {
				// Per-URL fatal (e.g. 404) — stop retrying this URL, move on.
				break
			}
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no URLs attempted")
	}
	return "", "", fmt.Errorf("%w: %v (set PURE_SIMDJSON_LIB_PATH to bypass)",
		ErrAllSourcesFailed, lastErr)
}

// isLadderFatalError reports whether err is permanent across the whole URL
// ladder (not just the current URL). Checksum mismatch, missing-checksum, and
// HTTPS→HTTP redirect downgrade are ladder-fatal: trying the next URL cannot
// recover because either the embedded checksum is wrong or the upstream is
// actively hostile.
func isLadderFatalError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrChecksumMismatch) || errors.Is(err, ErrNoChecksum) {
		return true
	}
	if isBootstrapRedirectPolicyError(err) {
		return true
	}
	return false
}

// downloadOnce performs a single HTTP GET with User-Agent stamped, streams the
// body through io.MultiWriter(file, sha256) for one-pass hashing, and returns
// the temp-file path plus the hex digest.
func downloadOnce(ctx context.Context, cfg bootstrapConfig, rawURL, cacheDir string) (tmpPath, digest string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", markPermanentBootstrapError(fmt.Errorf("create request: %w", err))
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := cfg.httpClient.Do(req)
	if err != nil {
		if isBootstrapRedirectPolicyError(err) {
			return "", "", markPermanentBootstrapError(fmt.Errorf("redirect policy: %w", err))
		}
		// Context cancellations propagate as retryable-but-caller-cancelled; we
		// short-circuit in downloadWithRetry by checking ctx.Err() next iteration.
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		statusErr := fmt.Errorf("HTTP %d from %s: %s",
			resp.StatusCode, rawURL, strings.TrimSpace(string(snippet)))
		if !isRetryable(resp.StatusCode, resp.Header, string(snippet)) {
			return "", "", markPermanentBootstrapError(statusErr)
		}
		return "", "", statusErr
	}

	// os.CreateTemp(cacheDir, ...) ensures the temp file is on the same
	// filesystem as the final cache path — critical for os.Rename atomicity
	// (pitfall #3).
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", "", markPermanentBootstrapError(fmt.Errorf("create cache dir: %w", err))
	}
	f, err := os.CreateTemp(cacheDir, "pure-simdjson-*.tmp")
	if err != nil {
		return "", "", markPermanentBootstrapError(fmt.Errorf("create temp: %w", err))
	}
	// Capture the temp path in a local so the cleanup defer is not subject to
	// named-return-zeroing when an early `return "", "", err` fires below
	// (Plan 05-06 Rule 1 — fixes orphan *.tmp leak observed in
	// TestBootstrapSyncCancellation).
	createdTmp := f.Name()
	success := false
	defer func() {
		_ = f.Close()
		if !success {
			_ = os.Remove(createdTmp)
		}
	}()

	h := sha256.New()
	// LimitReader(maxBytes+1) lets us detect oversize responses (written > maxBytes).
	written, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, maxBootstrapArtifactBytes+1))
	if err != nil {
		return "", "", fmt.Errorf("write to temp: %w", err)
	}
	if written > maxBootstrapArtifactBytes {
		return "", "", markPermanentBootstrapError(
			fmt.Errorf("response too large: %d bytes from %s", written, rawURL))
	}
	if err := f.Sync(); err != nil {
		return "", "", fmt.Errorf("fsync temp: %w", err)
	}
	digest = hex.EncodeToString(h.Sum(nil))
	success = true
	return createdTmp, digest, nil
}
