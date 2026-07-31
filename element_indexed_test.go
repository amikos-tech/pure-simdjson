package purejson

import (
	"errors"
	"testing"
)

func TestArray_At(t *testing.T) {
	_, doc := mustParseDoc(t, `[10,20,30]`)
	array, err := doc.Root().AsArray()
	if err != nil {
		t.Fatalf("AsArray() error = %v", err)
	}

	element, err := array.At(1)
	if err != nil {
		t.Fatalf("At(1) error = %v", err)
	}
	value, err := element.GetInt64()
	if err != nil {
		t.Fatalf("At(1).GetInt64() error = %v", err)
	}
	if value != 20 {
		t.Fatalf("At(1).GetInt64() = %d, want 20", value)
	}

	if _, err := array.At(5); !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("At(5) error = %v, want ErrIndexOutOfRange", err)
	}
	if _, err := array.At(-1); !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("At(-1) error = %v, want ErrIndexOutOfRange", err)
	}
}

func TestArray_Len(t *testing.T) {
	testCases := []struct {
		name string
		json string
		want int
	}{
		{name: "populated", json: `[1,2,3,4]`, want: 4},
		{name: "empty", json: `[]`, want: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)
			array, err := doc.Root().AsArray()
			if err != nil {
				t.Fatalf("AsArray() error = %v", err)
			}

			if got := array.Len(); got != tc.want {
				t.Fatalf("Len() = %d, want %d", got, tc.want)
			}
			got, err := array.LenErr()
			if err != nil {
				t.Fatalf("LenErr() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("LenErr() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestObject_Size(t *testing.T) {
	testCases := []struct {
		name string
		json string
		want int
	}{
		{name: "populated", json: `{"a":1,"b":2}`, want: 2},
		{name: "empty", json: `{}`, want: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)
			object, err := doc.Root().AsObject()
			if err != nil {
				t.Fatalf("AsObject() error = %v", err)
			}

			if got := object.Size(); got != tc.want {
				t.Fatalf("Size() = %d, want %d", got, tc.want)
			}
			got, err := object.SizeErr()
			if err != nil {
				t.Fatalf("SizeErr() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("SizeErr() = %d, want %d", got, tc.want)
			}
		})
	}
}
