package purejson

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

func TestGetInt64WrongType(t *testing.T) {
	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	doc, err := parser.Parse([]byte(`"hello"`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	t.Cleanup(func() {
		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() cleanup error = %v", err)
		}
	})

	if _, err := doc.Root().GetInt64(); !errors.Is(err, ErrWrongType) {
		t.Fatalf("GetInt64() on string error = %v, want ErrWrongType", err)
	}
}

func TestAsArrayRejectsScalar(t *testing.T) {
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

	if _, err := doc.Root().AsArray(); !errors.Is(err, ErrWrongType) {
		t.Fatalf("AsArray() on int error = %v, want ErrWrongType", err)
	}
	if _, err := doc.Root().AsObject(); !errors.Is(err, ErrWrongType) {
		t.Fatalf("AsObject() on int error = %v, want ErrWrongType", err)
	}
}

func TestAsArrayAfterDocClosed(t *testing.T) {
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

	if _, err := root.AsArray(); !errors.Is(err, ErrClosed) {
		t.Fatalf("AsArray() after Close error = %v, want ErrClosed", err)
	}
	if _, err := root.AsObject(); !errors.Is(err, ErrClosed) {
		t.Fatalf("AsObject() after Close error = %v, want ErrClosed", err)
	}
}

func TestErrorFormatBranches(t *testing.T) {
	sentinel := errors.New("test sentinel")

	testCases := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "code-message-offset",
			err:  &Error{code: 32, offset: 7, message: "bad token", err: sentinel},
			want: "test sentinel (code=32, offset=7): bad token",
		},
		{
			name: "code-message",
			err:  &Error{code: 32, message: "bad token", err: sentinel},
			want: "test sentinel (code=32): bad token",
		},
		{
			name: "code-offset",
			err:  &Error{code: 32, offset: 7, err: sentinel},
			want: "test sentinel (code=32, offset=7)",
		},
		{
			name: "code-only",
			err:  &Error{code: 32, err: sentinel},
			want: "test sentinel (code=32)",
		},
		{
			name: "message-only",
			err:  &Error{message: "load failure", err: sentinel},
			want: "test sentinel: load failure",
		},
		{
			name: "no-details",
			err:  &Error{err: sentinel},
			want: "test sentinel",
		},
		{
			name: "no-sentinel-uses-default-label",
			err:  &Error{code: 32, message: "raw"},
			want: "purejson error (code=32): raw",
		},
		{
			name: "offset-unknown-treated-as-no-offset",
			err:  &Error{code: 32, offset: ffi.LastErrorOffsetUnknown, err: sentinel},
			want: "test sentinel (code=32)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestErrorNilReceiverIsSafe(t *testing.T) {
	var nilErr *Error
	if got := nilErr.Error(); got != "<nil>" {
		t.Fatalf("nil.Error() = %q, want %q", got, "<nil>")
	}
	if code := nilErr.Code(); code != 0 {
		t.Fatalf("nil.Code() = %d, want 0", code)
	}
	if offset := nilErr.Offset(); offset != 0 {
		t.Fatalf("nil.Offset() = %d, want 0", offset)
	}
	if msg := nilErr.Message(); msg != "" {
		t.Fatalf("nil.Message() = %q, want empty", msg)
	}
	if unwrap := nilErr.Unwrap(); unwrap != nil {
		t.Fatalf("nil.Unwrap() = %v, want nil", unwrap)
	}
}

func TestParserZeroValueRejectsCalls(t *testing.T) {
	var p Parser

	if _, err := p.Parse([]byte("42")); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("zero-value Parse error = %v, want ErrInvalidHandle", err)
	}
	if err := p.Close(); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("zero-value Close error = %v, want ErrInvalidHandle", err)
	}
}

func TestLeakWarningDocTestBuild(t *testing.T) {
	if helperMode := os.Getenv("PUREJSON_HELPER_MODE"); helperMode != "" && helperMode != "single-doc-leak" {
		t.Skip("different helper mode")
	}
	if !testBuildFinalizersEnabled() {
		t.Skip("requires purejson_testbuild")
	}

	if os.Getenv("PUREJSON_HELPER_MODE") == "single-doc-leak" {
		runSingleDocLeakHelper(t)
		return
	}

	stdout, stderr := runDocLeakHelperProcess(t, "single-doc-leak")
	if !strings.Contains(stderr, "purejson leak: doc") {
		t.Fatalf("stderr = %q, want purejson leak: doc prefix", stderr)
	}
	if count := parseFinalizerCount(t, stdout, "doc-finalizers"); count < 1 {
		t.Fatalf("doc finalizer count = %d, want >= 1", count)
	}
}

func TestLeakWarningDocSilentProd(t *testing.T) {
	if helperMode := os.Getenv("PUREJSON_HELPER_MODE"); helperMode != "" && helperMode != "single-doc-leak" {
		t.Skip("different helper mode")
	}
	if testBuildFinalizersEnabled() {
		t.Skip("production-only assertion")
	}

	if os.Getenv("PUREJSON_HELPER_MODE") == "single-doc-leak" {
		runSingleDocLeakHelper(t)
		return
	}

	stdout, stderr := runDocLeakHelperProcess(t, "single-doc-leak")
	if strings.Contains(stderr, "purejson leak:") {
		t.Fatalf("stderr = %q, want no leak warning", stderr)
	}
	if count := parseFinalizerCount(t, stdout, "doc-finalizers"); count < 1 {
		t.Fatalf("doc finalizer count = %d, want >= 1", count)
	}
}

func TestLeakWarningDocProdWhenEnabled(t *testing.T) {
	if helperMode := os.Getenv("PUREJSON_HELPER_MODE"); helperMode != "" && helperMode != "single-doc-leak" {
		t.Skip("different helper mode")
	}
	if testBuildFinalizersEnabled() {
		t.Skip("production-only assertion")
	}

	if os.Getenv("PUREJSON_HELPER_MODE") == "single-doc-leak" {
		runSingleDocLeakHelper(t)
		return
	}

	stdout, stderr := runDocLeakHelperProcess(t, "single-doc-leak", "PURE_SIMDJSON_WARN_LEAKS=1")
	if !strings.Contains(stderr, "purejson leak: doc") {
		t.Fatalf("stderr = %q, want purejson leak: doc prefix", stderr)
	}
	if count := parseFinalizerCount(t, stdout, "doc-finalizers"); count < 1 {
		t.Fatalf("doc finalizer count = %d, want >= 1", count)
	}
}

func runDocLeakHelperProcess(t *testing.T, mode string, extraEnv ...string) (string, string) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestLeakWarningDoc")
	cmd.Env = append(os.Environ(), append([]string{"PUREJSON_HELPER_MODE=" + mode}, extraEnv...)...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("helper process error = %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	return stdout.String(), stderr.String()
}

func runSingleDocLeakHelper(t *testing.T) {
	t.Helper()

	resetFinalizerCountsForTest()
	parser := mustNewParser(t)
	defer func() {
		// Force-clear liveDoc so the parser cleanup does not block on the leaked doc.
		parser.mu.Lock()
		parser.liveDoc = 0
		parser.mu.Unlock()
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	}()

	func() {
		doc, err := parser.Parse([]byte("42"))
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		_ = doc
	}()

	waitForFinalizers(t, func() bool {
		return docFinalizerCountForTest() >= 1
	})

	fmt.Printf("doc-finalizers=%d\n", docFinalizerCountForTest())
}
