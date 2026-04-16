package purejson

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
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

func TestLeakWarningTestBuild(t *testing.T) {
	if helperMode := os.Getenv("PUREJSON_HELPER_MODE"); helperMode != "" && helperMode != "single-parser-leak" {
		t.Skip("different helper mode")
	}
	if !testBuildFinalizersEnabled() {
		t.Skip("requires purejson_testbuild")
	}

	if os.Getenv("PUREJSON_HELPER_MODE") == "single-parser-leak" {
		runSingleParserLeakHelper(t)
		return
	}

	stdout, stderr := runLeakHelperProcess(t, "single-parser-leak")
	if !strings.Contains(stderr, "purejson leak: parser") {
		t.Fatalf("stderr = %q, want purejson leak prefix", stderr)
	}
	if count := parseFinalizerCount(t, stdout, "parser-finalizers"); count < 1 {
		t.Fatalf("parser finalizer count = %d, want >= 1", count)
	}
}

func TestLeakWarningSilentProd(t *testing.T) {
	if helperMode := os.Getenv("PUREJSON_HELPER_MODE"); helperMode != "" && helperMode != "single-parser-leak" {
		t.Skip("different helper mode")
	}
	if testBuildFinalizersEnabled() {
		t.Skip("production-only assertion")
	}

	if os.Getenv("PUREJSON_HELPER_MODE") == "single-parser-leak" {
		runSingleParserLeakHelper(t)
		return
	}

	stdout, stderr := runLeakHelperProcess(t, "single-parser-leak")
	if strings.Contains(stderr, "purejson leak:") {
		t.Fatalf("stderr = %q, want no leak warning", stderr)
	}
	if count := parseFinalizerCount(t, stdout, "parser-finalizers"); count < 1 {
		t.Fatalf("parser finalizer count = %d, want >= 1", count)
	}
}

func TestLeakWarningMassLeak10000(t *testing.T) {
	if helperMode := os.Getenv("PUREJSON_HELPER_MODE"); helperMode != "" && helperMode != "mass-parser-leak" {
		t.Skip("different helper mode")
	}
	if !testBuildFinalizersEnabled() {
		t.Skip("requires purejson_testbuild")
	}

	if os.Getenv("PUREJSON_HELPER_MODE") == "mass-parser-leak" {
		runMassParserLeakHelper(t, 10000)
		return
	}

	stdout, stderr := runLeakHelperProcess(t, "mass-parser-leak")
	if !strings.Contains(stderr, "purejson leak: parser") {
		t.Fatalf("stderr = %q, want purejson leak prefix", stderr)
	}
	if count := parseFinalizerCount(t, stdout, "parser-finalizers"); count < 10000 {
		t.Fatalf("parser finalizer count = %d, want >= 10000", count)
	}
}

func runLeakHelperProcess(t *testing.T, mode string) (string, string) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestLeakWarning")
	cmd.Env = append(os.Environ(), "PUREJSON_HELPER_MODE="+mode)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("helper process error = %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	return stdout.String(), stderr.String()
}

func runSingleParserLeakHelper(t *testing.T) {
	t.Helper()

	resetFinalizerCountsForTest()
	func() {
		parser := mustNewParser(t)
		_ = parser
	}()

	waitForFinalizers(t, func() bool {
		return parserFinalizerCountForTest() >= 1
	})

	fmt.Printf("parser-finalizers=%d\n", parserFinalizerCountForTest())
}

func runMassParserLeakHelper(t *testing.T, count int) {
	t.Helper()

	resetFinalizerCountsForTest()
	for i := 0; i < count; i++ {
		parser := mustNewParser(t)
		_ = parser
	}

	waitForFinalizers(t, func() bool {
		return parserFinalizerCountForTest() >= int64(count)
	})

	fmt.Printf("parser-finalizers=%d\n", parserFinalizerCountForTest())
}

func waitForFinalizers(t *testing.T, done func() bool) {
	t.Helper()

	for i := 0; i < 400; i++ {
		runtime.GC()
		runtime.Gosched()
		if done() {
			return
		}
	}

	t.Fatal("finalizer condition was not satisfied")
}

func parseFinalizerCount(t *testing.T, stdout string, key string) int {
	t.Helper()

	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, key+"=") {
			continue
		}

		value, err := strconv.Atoi(strings.TrimPrefix(line, key+"="))
		if err != nil {
			t.Fatalf("parse %s from %q: %v", key, line, err)
		}
		return value
	}

	t.Fatalf("missing %s in helper stdout %q", key, stdout)
	return 0
}
