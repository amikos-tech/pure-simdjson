package purejson

import (
	"bytes"
	"errors"
	"testing"
)

func TestMinify(t *testing.T) {
	testCases := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{
			name:  "object whitespace",
			input: []byte(`{"a":  1,   "b": 2}`),
			want:  []byte(`{"a":1,"b":2}`),
		},
		{
			name:  "empty",
			input: []byte{},
			want:  []byte{},
		},
		{
			name:  "structural whitespace",
			input: []byte(" \n [ 1 , { \"message\" : \"hello world\" } ] \t"),
			want:  []byte(`[1,{"message":"hello world"}]`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := append([]byte(nil), tc.input...)
			original := append([]byte(nil), input...)

			got, err := Minify(input)
			if err != nil {
				t.Fatalf("Minify() error = %v", err)
			}
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("Minify() = %q, want %q", got, tc.want)
			}
			if !bytes.Equal(input, original) {
				t.Fatalf("Minify() mutated input: got %q, want %q", input, original)
			}
		})
	}
}

func TestMinifyInto_Overlap(t *testing.T) {
	input := []byte(` { "message" : "hello world" } `)
	want, err := Minify(append([]byte(nil), input...))
	if err != nil {
		t.Fatalf("Minify() reference error = %v", err)
	}

	buf := append([]byte(nil), input...)
	written, err := MinifyInto(buf, buf)
	if err != nil {
		t.Fatalf("MinifyInto() in-place error = %v", err)
	}
	if got := buf[:written]; !bytes.Equal(got, want) {
		t.Fatalf("MinifyInto() in-place = %q, want %q", got, want)
	}
}

func TestMinifyInto_Disjoint(t *testing.T) {
	src := []byte(` { "a" : [ 1, 2 ] } `)
	srcBefore := append([]byte(nil), src...)
	dst := make([]byte, len(src))

	written, err := MinifyInto(dst, src)
	if err != nil {
		t.Fatalf("MinifyInto() disjoint error = %v", err)
	}
	if want := []byte(`{"a":[1,2]}`); !bytes.Equal(dst[:written], want) {
		t.Fatalf("MinifyInto() disjoint = %q, want %q", dst[:written], want)
	}
	if !bytes.Equal(src, srcBefore) {
		t.Fatalf("MinifyInto() mutated disjoint src: got %q, want %q", src, srcBefore)
	}
}

func TestMinifyInto_UndersizedDst(t *testing.T) {
	src := []byte(`{"a": 1}`)
	dst := bytes.Repeat([]byte{0xa5}, len(src)-1)
	dstBefore := append([]byte(nil), dst...)

	written, err := MinifyInto(dst, src)
	if written != 0 {
		t.Fatalf("MinifyInto() written = %d, want 0", written)
	}
	if !errors.Is(err, ErrBufferTooSmall) {
		t.Fatalf("MinifyInto() error = %v, want ErrBufferTooSmall", err)
	}
	if !bytes.Equal(dst, dstBefore) {
		t.Fatalf("MinifyInto() changed undersized dst: got %v, want %v", dst, dstBefore)
	}
}

func TestMinifyInto_PartialOverlap(t *testing.T) {
	input := []byte(`{"a": 1}`)
	testCases := []struct {
		name      string
		dstOffset int
		srcOffset int
	}{
		{name: "dst starts inside src", dstOffset: 1, srcOffset: 0},
		{name: "src starts inside dst", dstOffset: 0, srcOffset: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			backing := bytes.Repeat([]byte{0x7f}, len(input)+1)
			copy(backing[tc.srcOffset:], input)
			dst := backing[tc.dstOffset : tc.dstOffset+len(input)]
			src := backing[tc.srcOffset : tc.srcOffset+len(input)]
			before := append([]byte(nil), backing...)

			written, err := MinifyInto(dst, src)
			if written != 0 {
				t.Fatalf("MinifyInto() written = %d, want 0", written)
			}
			if !errors.Is(err, ErrInvalidOption) {
				t.Fatalf("MinifyInto() error = %v, want ErrInvalidOption", err)
			}
			if !bytes.Equal(backing, before) {
				t.Fatalf("MinifyInto() changed overlapping storage: got %v, want %v", backing, before)
			}
		})
	}
}

func TestMinifyInto_Empty(t *testing.T) {
	written, err := MinifyInto(nil, nil)
	if err != nil {
		t.Fatalf("MinifyInto(nil, nil) error = %v", err)
	}
	if written != 0 {
		t.Fatalf("MinifyInto(nil, nil) written = %d, want 0", written)
	}
}

func TestMinify_UnclosedString(t *testing.T) {
	if _, err := Minify([]byte(`{"message":"unterminated}`)); err == nil {
		t.Fatal("Minify() unclosed string error = nil, want non-nil")
	}
}

func TestMinifyAutomaticFallbackRejected(t *testing.T) {
	runKernelScenario(t, "minify-automatic-fallback", func(t *testing.T) {
		t.Setenv("PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION", "fallback")

		if _, err := Minify([]byte(`{}`)); !errors.Is(err, ErrCPUUnsupported) {
			t.Fatalf("Minify() error = %v, want ErrCPUUnsupported", err)
		}
		if err := SetKernel(Kernel()); err != nil {
			t.Fatalf("SetKernel() after ErrCPUUnsupported error = %v", err)
		}
	})
}

func TestMinifyPreflightFailuresDoNotLockKernelSelection(t *testing.T) {
	runKernelScenario(t, "minify-preflight", func(t *testing.T) {
		src := []byte(`{"a": 1}`)
		if _, err := MinifyInto(make([]byte, len(src)-1), src); !errors.Is(err, ErrBufferTooSmall) {
			t.Fatalf("short MinifyInto() error = %v, want ErrBufferTooSmall", err)
		}

		backing := append(append([]byte(nil), src...), 0)
		if _, err := MinifyInto(backing[1:], backing[:len(src)]); !errors.Is(err, ErrInvalidOption) {
			t.Fatalf("overlapping MinifyInto() error = %v, want ErrInvalidOption", err)
		}
		if library := cachedLibraryForKernelTest(); library != nil {
			t.Fatalf("preflight failures installed cached library %#v", library)
		}
		if err := SetKernel(Kernel()); err != nil {
			t.Fatalf("SetKernel() after preflight failures error = %v", err)
		}
	})
}

func TestMinifyLocksKernelSelection(t *testing.T) {
	runKernelScenario(t, "minify-success-lock", func(t *testing.T) {
		if _, err := Minify([]byte(`{}`)); err != nil {
			t.Fatalf("Minify() error = %v", err)
		}
		if err := SetKernel(""); !errors.Is(err, ErrKernelLocked) {
			t.Fatalf("SetKernel() after Minify() error = %v, want ErrKernelLocked", err)
		}
	})
}

func TestMinifyInvalidJSONLocksKernelSelection(t *testing.T) {
	runKernelScenario(t, "minify-invalid-json-lock", func(t *testing.T) {
		if _, err := Minify([]byte(`{"message":"unterminated}`)); !errors.Is(err, ErrInvalidJSON) {
			t.Fatalf("Minify() error = %v, want ErrInvalidJSON", err)
		}
		if err := SetKernel(""); !errors.Is(err, ErrKernelLocked) {
			t.Fatalf("SetKernel() after invalid Minify() error = %v, want ErrKernelLocked", err)
		}
	})
}
