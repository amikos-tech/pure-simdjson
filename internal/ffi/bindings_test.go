package ffi

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestElementGetStringWarnsOnBytesFreeFailure(t *testing.T) {
	t.Setenv("PURE_SIMDJSON_WARN_LEAKS", "1")

	payload := []byte("hello")
	var freed bool
	b := &Bindings{
		elementGetString: func(_ *ValueView, outPtr **byte, outLen *uintptr) int32 {
			*outPtr = &payload[0]
			*outLen = uintptr(len(payload))
			return int32(OK)
		},
		bytesFree: func(_ *byte, _ uintptr) int32 {
			freed = true
			return int32(ErrInternal)
		},
	}

	stderr := captureStderr(t, func() {
		value, rc := b.ElementGetString(&ValueView{})
		if rc != int32(OK) {
			t.Fatalf("ElementGetString() rc = %d, want %d", rc, OK)
		}
		if value != "hello" {
			t.Fatalf("ElementGetString() value = %q, want %q", value, "hello")
		}
	})

	if !freed {
		t.Fatal("BytesFree() was not called")
	}
	if !strings.Contains(stderr, "purejson leak: bytes_free rc=") {
		t.Fatalf("stderr = %q, want bytes_free warning", stderr)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	defer r.Close()

	os.Stderr = w
	defer func() {
		os.Stderr = original
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("stderr writer close error = %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(stderr) error = %v", err)
	}
	return string(data)
}
