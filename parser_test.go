package purejson

import (
	"errors"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

func mustNewParser(t *testing.T) *Parser {
	t.Helper()

	parser, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	return parser
}

func TestHappyPathGetInt64(t *testing.T) {
	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	doc, err := parser.Parse([]byte("42"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	t.Cleanup(func() {
		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() cleanup error = %v", err)
		}
	})

	value, err := doc.Root().GetInt64()
	if err != nil {
		t.Fatalf("GetInt64() error = %v", err)
	}
	if value != 42 {
		t.Fatalf("GetInt64() = %d, want 42", value)
	}
}

func TestABIMismatchAtNewParser(t *testing.T) {
	restore := setExpectedABIVersionForTest(0xDEADBEEF)
	t.Cleanup(restore)

	_, err := NewParser()
	if !errors.Is(err, ErrABIVersionMismatch) {
		t.Fatalf("NewParser() mismatch error = %v, want ErrABIVersionMismatch", err)
	}

	var nativeErr *Error
	if !errors.As(err, &nativeErr) {
		t.Fatalf("NewParser() mismatch error = %v, want *Error", err)
	}
	if nativeErr.Code != int32(ffi.ErrABIMismatch) {
		t.Fatalf("native error code = %d, want %d", nativeErr.Code, ffi.ErrABIMismatch)
	}
	if nativeErr.Message == "" {
		t.Fatal("native error message is empty")
	}
}

func TestParserDoubleClose(t *testing.T) {
	parser := mustNewParser(t)

	if err := parser.Close(); err != nil {
		t.Fatalf("first parser.Close() error = %v", err)
	}
	if err := parser.Close(); err != nil {
		t.Fatalf("second parser.Close() error = %v, want nil", err)
	}
}

func TestDocDoubleClose(t *testing.T) {
	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	doc, err := parser.Parse([]byte("42"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if err := doc.Close(); err != nil {
		t.Fatalf("first doc.Close() error = %v", err)
	}
	if err := doc.Close(); err != nil {
		t.Fatalf("second doc.Close() error = %v, want nil", err)
	}
}

func TestParserCloseWhileDocLive(t *testing.T) {
	parser := mustNewParser(t)
	doc, err := parser.Parse([]byte("42"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if err := parser.Close(); !errors.Is(err, ErrParserBusy) {
		t.Fatalf("parser.Close() with live doc error = %v, want ErrParserBusy", err)
	}

	value, err := doc.Root().GetInt64()
	if err != nil {
		t.Fatalf("GetInt64() after busy parser.Close() error = %v", err)
	}
	if value != 42 {
		t.Fatalf("GetInt64() after busy parser.Close() = %d, want 42", value)
	}

	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}
	if err := parser.Close(); err != nil {
		t.Fatalf("parser.Close() after doc.Close() error = %v", err)
	}
}

func TestParseAfterClose(t *testing.T) {
	parser := mustNewParser(t)
	if err := parser.Close(); err != nil {
		t.Fatalf("parser.Close() error = %v", err)
	}

	_, err := parser.Parse([]byte("42"))
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Parse() after Close error = %v, want ErrClosed", err)
	}
}

func TestAccessorAfterClose(t *testing.T) {
	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	doc, err := parser.Parse([]byte("42"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	root := doc.Root()
	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}

	_, err = root.GetInt64()
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("GetInt64() after doc.Close error = %v, want ErrClosed", err)
	}
}

func TestParserBusy(t *testing.T) {
	parser := mustNewParser(t)
	doc, err := parser.Parse([]byte("42"))
	if err != nil {
		t.Fatalf("first Parse() error = %v", err)
	}

	_, err = parser.Parse([]byte("43"))
	if !errors.Is(err, ErrParserBusy) {
		t.Fatalf("second Parse() error = %v, want ErrParserBusy", err)
	}

	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}
	if err := parser.Close(); err != nil {
		t.Fatalf("parser.Close() error = %v", err)
	}
}

func TestStructuredErrorDetails(t *testing.T) {
	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	_, err := parser.Parse([]byte("{"))
	if !errors.Is(err, ErrInvalidJSON) {
		t.Fatalf("Parse() invalid json error = %v, want ErrInvalidJSON", err)
	}

	var nativeErr *Error
	if !errors.As(err, &nativeErr) {
		t.Fatalf("Parse() invalid json error = %v, want *Error", err)
	}
	if nativeErr.Code != int32(ffi.ErrInvalidJSON) {
		t.Fatalf("native error code = %d, want %d", nativeErr.Code, ffi.ErrInvalidJSON)
	}
	if nativeErr.Message == "" {
		t.Fatal("native error message is empty")
	}

	doc, err := parser.Parse([]byte("42"))
	if err != nil {
		t.Fatalf("Parse() after invalid json error = %v", err)
	}
	defer func() {
		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() cleanup error = %v", err)
		}
	}()

	if _, err := doc.Root().GetInt64(); err != nil {
		t.Fatalf("GetInt64() after invalid json recovery error = %v", err)
	}
}
