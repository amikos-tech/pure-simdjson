package bootstrap

import "testing"

func TestParseChecksumFromSHA256SUMSFindsMatchingEntry(t *testing.T) {
	const (
		key        = "v0.1.0/linux-amd64/libpure_simdjson.so"
		otherKey   = "v0.1.0/darwin-arm64/libpure_simdjson.dylib"
		otherSum   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		expected   = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
		wantDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)

	body := []byte("\n" + otherSum + "  " + otherKey + "\n" + expected + "  " + key + "\n")

	got, err := parseChecksumFromSHA256SUMS(body, key)
	if err != nil {
		t.Fatalf("parseChecksumFromSHA256SUMS: %v", err)
	}
	if expected == wantDigest {
		t.Fatal("test fixture should verify lowercase normalization")
	}
	if got != wantDigest {
		t.Fatalf("digest = %q, want normalized lowercase checksum", got)
	}
}
