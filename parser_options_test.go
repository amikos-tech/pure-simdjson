package purejson

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

func TestParserOptionDefaultsAndDuplicates(t *testing.T) {
	omitted, err := normalizeParserOptions()
	if err != nil {
		t.Fatalf("normalizeParserOptions() error = %v", err)
	}
	explicit, err := normalizeParserOptions(
		WithMaxCapacity(0),
		WithMaxDepth(0),
	)
	if err != nil {
		t.Fatalf("normalizeParserOptions(explicit defaults) error = %v", err)
	}
	if omitted != explicit {
		t.Fatalf("omitted config = %+v, explicit defaults = %+v; want equal", omitted, explicit)
	}
	if omitted != defaultParserConfig {
		t.Fatalf("default config = %+v, want %+v", omitted, defaultParserConfig)
	}
	if omitted.maxCapacity != 0xFFFFFFFF || omitted.maxDepth != 1024 {
		t.Fatalf("default config = %+v, want capacity=%d depth=%d", omitted, uint64(0xFFFFFFFF), uint32(1024))
	}

	duplicates, err := normalizeParserOptions(
		WithMaxCapacity(64),
		WithMaxCapacity(96),
		WithMaxDepth(4),
		WithMaxDepth(8),
	)
	if err != nil {
		t.Fatalf("normalizeParserOptions(duplicates) error = %v", err)
	}
	want := parserConfig{maxCapacity: 96, maxDepth: 8}
	if duplicates != want {
		t.Fatalf("duplicate config = %+v, want last-option-wins %+v", duplicates, want)
	}

	resetToDefaults, err := normalizeParserOptions(
		WithMaxCapacity(64),
		WithMaxDepth(4),
		WithMaxCapacity(0),
		WithMaxDepth(0),
	)
	if err != nil {
		t.Fatalf("normalizeParserOptions(reset defaults) error = %v", err)
	}
	if resetToDefaults != defaultParserConfig {
		t.Fatalf("reset config = %+v, want defaults %+v", resetToDefaults, defaultParserConfig)
	}
}

func TestParserOptionRejectsInvalidValues(t *testing.T) {
	testCases := []struct {
		name   string
		option ParserOption
	}{
		{name: "zero value", option: ParserOption{}},
		{name: "unknown kind", option: ParserOption{kind: parserOptionKind(255), value: 64}},
		{name: "negative capacity", option: WithMaxCapacity(-1)},
		{name: "capacity one", option: WithMaxCapacity(1)},
		{name: "capacity thirty one", option: WithMaxCapacity(31)},
		{name: "negative depth", option: WithMaxDepth(-1)},
	}
	if strconv.IntSize > 32 {
		tooLargeCapacity := uint64(0xFFFFFFFF) + 1
		tooLargeDepth := uint64(^uint32(0)) + 1
		testCases = append(testCases,
			struct {
				name   string
				option ParserOption
			}{name: "capacity ABI overflow", option: WithMaxCapacity(int(tooLargeCapacity))},
			struct {
				name   string
				option ParserOption
			}{name: "depth ABI overflow", option: WithMaxDepth(int(tooLargeDepth))},
		)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := normalizeParserOptions(tc.option)
			if !errors.Is(err, ErrInvalidOption) {
				t.Fatalf("normalizeParserOptions(%+v) error = %v, want ErrInvalidOption", tc.option, err)
			}
		})
	}
}

func TestParserOptionInvalidFailsBeforeLibraryResolution(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	t.Setenv(libraryEnvPath, filepath.Join(t.TempDir(), "missing-native-library"))
	parser, err := NewParser(ParserOption{})
	if parser != nil {
		_ = parser.Close()
		t.Fatal("NewParser(ParserOption{}) returned a parser")
	}
	if !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("NewParser(ParserOption{}) error = %v, want ErrInvalidOption before library resolution", err)
	}
	if errors.Is(err, errLoadLibrary) {
		t.Fatalf("NewParser(ParserOption{}) error = %v, unexpectedly resolved the library", err)
	}
}

func TestParserCapacityBoundary(t *testing.T) {
	parser, err := NewParser(WithMaxCapacity(32))
	if err != nil {
		t.Fatalf("NewParser(WithMaxCapacity(32)) error = %v", err)
	}
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})
	if parser.config != (parserConfig{maxCapacity: 32, maxDepth: 1024}) {
		t.Fatalf("parser config = %+v, want capacity=32 depth=1024", parser.config)
	}

	exact := []byte(`"` + strings.Repeat("x", 30) + `"`)
	doc, err := parser.Parse(exact)
	if err != nil {
		t.Fatalf("Parse(%d-byte exact-capacity input) error = %v", len(exact), err)
	}
	if err := doc.Close(); err != nil {
		t.Fatalf("exact-capacity doc.Close() error = %v", err)
	}

	oversized := []byte(`"` + strings.Repeat("x", 31) + `"`)
	doc, err = parser.Parse(oversized)
	if doc != nil {
		t.Fatal("Parse(capacity+1) unexpectedly returned a document")
	}
	if !errors.Is(err, ErrCapacityLimitExceeded) {
		t.Fatalf("Parse(%d-byte capacity+1 input) error = %v, want ErrCapacityLimitExceeded", len(oversized), err)
	}
	var nativeErr *Error
	if !errors.As(err, &nativeErr) {
		t.Fatalf("Parse(capacity+1) error = %v, want *Error", err)
	}
	if nativeErr.Code() != int32(ffi.ErrCapacityLimit) {
		t.Fatalf("Parse(capacity+1) code = %d, want %d", nativeErr.Code(), ffi.ErrCapacityLimit)
	}
	if nativeErr.Offset() != 0 || nativeErr.Message() != "" {
		t.Fatalf("Parse(capacity+1) details = offset %d, message %q; want no stale details", nativeErr.Offset(), nativeErr.Message())
	}
}

func TestParserDepthBoundaries(t *testing.T) {
	testCases := []struct {
		name     string
		option   ParserOption
		accepted int
		rejected int
	}{
		{name: "configured", option: WithMaxDepth(4), accepted: 3, rejected: 4},
		{name: "default", option: WithMaxDepth(0), accepted: 1023, rejected: 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser, err := NewParser(tc.option)
			if err != nil {
				t.Fatalf("NewParser() error = %v", err)
			}
			t.Cleanup(func() {
				if err := parser.Close(); err != nil {
					t.Fatalf("parser.Close() cleanup error = %v", err)
				}
			})

			doc, err := parser.Parse([]byte(nestedArrayJSON(tc.accepted)))
			if err != nil {
				t.Fatalf("Parse(depth %d) error = %v, want nil", tc.accepted, err)
			}
			if err := doc.Close(); err != nil {
				t.Fatalf("doc.Close() at depth %d error = %v", tc.accepted, err)
			}

			doc, err = parser.Parse([]byte(nestedArrayJSON(tc.rejected)))
			if doc != nil {
				t.Fatalf("Parse(depth %d) unexpectedly returned a document", tc.rejected)
			}
			if !errors.Is(err, ErrDepthLimitExceeded) {
				t.Fatalf("Parse(depth %d) error = %v, want ErrDepthLimitExceeded", tc.rejected, err)
			}
		})
	}
}
