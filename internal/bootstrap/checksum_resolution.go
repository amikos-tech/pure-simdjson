package bootstrap

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxChecksumsMetadataBytes int64 = 1 << 20 // 1 MiB

// ResolveChecksum returns the expected SHA-256 digest for the resolved target.
// Test and controlled local flows may inject values through the Checksums map.
// Production resolution falls back to published SHA256SUMS metadata.
func ResolveChecksum(ctx context.Context, opts ...BootstrapOption) (string, error) {
	if ctx == nil {
		return "", errors.New("bootstrap: nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cfg, err := resolveConfig(opts...)
	if err != nil {
		return "", err
	}
	return resolveExpectedChecksum(ctx, cfg)
}

func resolveExpectedChecksum(ctx context.Context, cfg bootstrapConfig) (string, error) {
	key := ChecksumKey(cfg.version, cfg.goos, cfg.goarch)
	if digest, ok := Checksums[key]; ok {
		digest = strings.ToLower(strings.TrimSpace(digest))
		if !looksLikeSHA256Hex(digest) {
			return "", fmt.Errorf("invalid checksum override for %s: %q", key, digest)
		}
		return digest, nil
	}
	return resolveChecksumFromPublishedSums(ctx, cfg, key)
}

func resolveChecksumFromPublishedSums(ctx context.Context, cfg bootstrapConfig, key string) (string, error) {
	r2Base := cfg.mirrorURL
	if r2Base == "" {
		r2Base = defaultR2BaseURL
	}

	urls := []string{r2ChecksumsURL(r2Base, cfg.version)}
	if !cfg.disableGH {
		urls = append(urls, githubChecksumsURL(cfg.githubBaseURL, cfg.version))
	}

	var lastErr error
	for _, rawURL := range urls {
		var serverHint time.Duration
		for attempt := 0; attempt < bootstrapRetryAttempt; attempt++ {
			if attempt > 0 {
				if err := sleepWithJitter(ctx, attempt, serverHint); err != nil {
					return "", err
				}
			}
			if err := ctx.Err(); err != nil {
				return "", err
			}

			digest, err := fetchChecksumFromURL(ctx, cfg, rawURL, key)
			if err == nil {
				return digest, nil
			}

			lastErr = fmt.Errorf("attempt %d/%d %s: %w", attempt+1, bootstrapRetryAttempt, rawURL, err)
			if isPermanentBootstrapError(err) {
				break
			}

			var hintErr *retryHintError
			if errors.As(err, &hintErr) {
				serverHint = hintErr.after
			} else {
				serverHint = 0
			}
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("%w: %s", ErrNoChecksum, key)
	}
	if errors.Is(lastErr, ErrNoChecksum) {
		return "", markPermanentBootstrapError(fmt.Errorf("%w: %s", ErrNoChecksum, key))
	}
	return "", lastErr
}

func fetchChecksumFromURL(ctx context.Context, cfg bootstrapConfig, rawURL, key string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", markPermanentBootstrapError(fmt.Errorf("create checksum request: %w", err))
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := cfg.httpClient.Do(req)
	if err != nil {
		if isBootstrapRedirectPolicyError(err) {
			return "", markPermanentBootstrapError(fmt.Errorf("redirect policy: %w", err))
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		statusErr := fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, rawURL, strings.TrimSpace(string(snippet)))
		if !isRetryable(resp.StatusCode, resp.Header, string(snippet)) {
			return "", markPermanentBootstrapError(statusErr)
		}
		if hint := parseRetryAfter(resp.Header); hint > 0 {
			return "", &retryHintError{err: statusErr, after: hint}
		}
		return "", statusErr
	}

	if resp.ContentLength > maxChecksumsMetadataBytes {
		return "", markPermanentBootstrapError(
			fmt.Errorf("advertised checksum metadata too large: %d bytes from %s (cap: %d)",
				resp.ContentLength, rawURL, maxChecksumsMetadataBytes))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxChecksumsMetadataBytes+1))
	if err != nil {
		return "", fmt.Errorf("read checksum metadata: %w", err)
	}
	if int64(len(body)) > maxChecksumsMetadataBytes {
		return "", markPermanentBootstrapError(
			fmt.Errorf("checksum metadata too large: %d bytes from %s", len(body), rawURL))
	}

	digest, err := parseChecksumFromSHA256SUMS(body, key)
	if err != nil {
		return "", markPermanentBootstrapError(err)
	}
	return digest, nil
}

func parseChecksumFromSHA256SUMS(body []byte, key string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return "", fmt.Errorf("invalid SHA256SUMS line: %q", line)
		}
		if fields[1] != key {
			continue
		}
		digest := strings.ToLower(strings.TrimSpace(fields[0]))
		if !looksLikeSHA256Hex(digest) {
			return "", fmt.Errorf("invalid SHA256SUMS digest for %s: %q", key, fields[0])
		}
		return digest, nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan SHA256SUMS: %w", err)
	}
	return "", fmt.Errorf("%w: %s", ErrNoChecksum, key)
}

func looksLikeSHA256Hex(raw string) bool {
	if len(raw) != 64 {
		return false
	}
	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}
