package purejson

import (
	"bytes"
	"testing"
)

func TestMinify(t *testing.T) {
	input := []byte(`{"a":  1,   "b": 2}`)
	original := append([]byte(nil), input...)

	got, err := Minify(input)
	if err != nil {
		t.Fatalf("Minify() error = %v", err)
	}
	if want := []byte(`{"a":1,"b":2}`); !bytes.Equal(got, want) {
		t.Fatalf("Minify() = %q, want %q", got, want)
	}
	if !bytes.Equal(input, original) {
		t.Fatalf("Minify() mutated input: got %q, want %q", input, original)
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
