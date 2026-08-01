package purejson

import (
	"errors"
	"testing"
)

func TestValidateUTF8(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
		want bool
	}{
		{name: "ascii", data: []byte("plain text"), want: true},
		{name: "multibyte", data: []byte("Здравей"), want: true},
		{name: "invalid continuation", data: []byte{0x80}, want: false},
		{name: "empty", data: nil, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateUTF8(tc.data)
			if err != nil {
				t.Fatalf("ValidateUTF8() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("ValidateUTF8() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestValidateUTF8AutomaticFallbackRejected(t *testing.T) {
	runKernelScenario(t, "validate-utf8-automatic-fallback", func(t *testing.T) {
		t.Setenv("PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION", "fallback")

		if _, err := ValidateUTF8([]byte("ok")); !errors.Is(err, ErrCPUUnsupported) {
			t.Fatalf("ValidateUTF8() error = %v, want ErrCPUUnsupported", err)
		}
		if err := SetKernel(Kernel()); err != nil {
			t.Fatalf("SetKernel() after ErrCPUUnsupported error = %v", err)
		}
	})
}

func TestValidateUTF8LocksKernelSelection(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
		want bool
	}{
		{name: "valid", data: []byte("ok"), want: true},
		{name: "invalid", data: []byte{0x80}, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runKernelScenario(t, "validate-utf8-"+tc.name+"-lock", func(t *testing.T) {
				got, err := ValidateUTF8(tc.data)
				if err != nil {
					t.Fatalf("ValidateUTF8() error = %v", err)
				}
				if got != tc.want {
					t.Fatalf("ValidateUTF8() = %t, want %t", got, tc.want)
				}
				if err := SetKernel(""); !errors.Is(err, ErrKernelLocked) {
					t.Fatalf("SetKernel() after ValidateUTF8() error = %v, want ErrKernelLocked", err)
				}
			})
		})
	}
}

func TestParseRejectsInvalidUTF8(t *testing.T) {
	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	invalid := []byte{0x80}
	doc, err := parser.Parse(invalid)
	if doc != nil {
		_ = doc.Close()
		t.Fatal("Parse() invalid UTF-8 returned a document")
	}
	if !errors.Is(err, ErrInvalidJSON) {
		t.Fatalf("Parse() invalid UTF-8 error = %v, want ErrInvalidJSON", err)
	}
}
