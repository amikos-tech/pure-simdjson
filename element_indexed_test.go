package purejson

import (
	"errors"
	"testing"
)

func TestArray_At(t *testing.T) {
	testCases := []struct {
		name       string
		json       string
		index      int
		forge      bool
		poisonView bool
		want       int64
		wantErr    error
	}{
		{name: "valid index", json: `[10,20,30]`, index: 1, want: 20},
		{name: "out of range", json: `[10,20,30]`, index: 5, wantErr: ErrIndexOutOfRange},
		{
			name:       "negative index rejected before native validation",
			json:       `[10,20,30]`,
			index:      -1,
			poisonView: true,
			wantErr:    ErrIndexOutOfRange,
		},
		{name: "forged wrong kind", json: `{"a":1}`, index: 0, forge: true, wantErr: ErrWrongType},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)
			root := doc.Root()

			var array Array
			if tc.forge {
				array = Array{element: root}
			} else {
				var err error
				array, err = root.AsArray()
				if err != nil {
					t.Fatalf("AsArray() error = %v", err)
				}
			}
			if tc.poisonView {
				// Native validation would reject this descendant state. Receiving the
				// range sentinel proves the negative-index guard ran before the FFI call.
				array.element.view.State0 = 1
				array.element.view.State1 = descendantViewTag
			}

			element, err := array.At(tc.index)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("At(%d) error = %v, want %v", tc.index, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("At(%d) error = %v", tc.index, err)
			}
			value, err := element.GetInt64()
			if err != nil {
				t.Fatalf("At(%d).GetInt64() error = %v", tc.index, err)
			}
			if value != tc.want {
				t.Fatalf("At(%d).GetInt64() = %d, want %d", tc.index, value, tc.want)
			}
		})
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

	t.Run("forged wrong kind", func(t *testing.T) {
		_, doc := mustParseDoc(t, `{}`)
		array := Array{element: doc.Root()}

		if got := array.Len(); got != 0 {
			t.Fatalf("Len() = %d, want 0", got)
		}
		got, err := array.LenErr()
		if got != 0 {
			t.Fatalf("LenErr() = %d, want 0", got)
		}
		if !errors.Is(err, ErrWrongType) {
			t.Fatalf("LenErr() error = %v, want ErrWrongType", err)
		}
	})

	t.Run("closed", func(t *testing.T) {
		_, doc := mustParseDoc(t, `[1]`)
		array, err := doc.Root().AsArray()
		if err != nil {
			t.Fatalf("AsArray() error = %v", err)
		}
		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() error = %v", err)
		}

		if got := array.Len(); got != 0 {
			t.Fatalf("Len() after Close = %d, want 0", got)
		}
		got, err := array.LenErr()
		if got != 0 {
			t.Fatalf("LenErr() after Close = %d, want 0", got)
		}
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("LenErr() after Close error = %v, want ErrClosed", err)
		}
	})

	t.Run("zero value", func(t *testing.T) {
		var array Array

		if got := array.Len(); got != 0 {
			t.Fatalf("zero-value Len() = %d, want 0", got)
		}
		got, err := array.LenErr()
		if got != 0 {
			t.Fatalf("zero-value LenErr() = %d, want 0", got)
		}
		if !errors.Is(err, ErrInvalidHandle) {
			t.Fatalf("zero-value LenErr() error = %v, want ErrInvalidHandle", err)
		}
	})
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

	t.Run("forged wrong kind", func(t *testing.T) {
		_, doc := mustParseDoc(t, `[]`)
		object := Object{element: doc.Root()}

		if got := object.Size(); got != 0 {
			t.Fatalf("Size() = %d, want 0", got)
		}
		got, err := object.SizeErr()
		if got != 0 {
			t.Fatalf("SizeErr() = %d, want 0", got)
		}
		if !errors.Is(err, ErrWrongType) {
			t.Fatalf("SizeErr() error = %v, want ErrWrongType", err)
		}
	})

	t.Run("closed", func(t *testing.T) {
		_, doc := mustParseDoc(t, `{"a":1}`)
		object, err := doc.Root().AsObject()
		if err != nil {
			t.Fatalf("AsObject() error = %v", err)
		}
		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() error = %v", err)
		}

		if got := object.Size(); got != 0 {
			t.Fatalf("Size() after Close = %d, want 0", got)
		}
		got, err := object.SizeErr()
		if got != 0 {
			t.Fatalf("SizeErr() after Close = %d, want 0", got)
		}
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("SizeErr() after Close error = %v, want ErrClosed", err)
		}
	})

	t.Run("zero value", func(t *testing.T) {
		var object Object

		if got := object.Size(); got != 0 {
			t.Fatalf("zero-value Size() = %d, want 0", got)
		}
		got, err := object.SizeErr()
		if got != 0 {
			t.Fatalf("zero-value SizeErr() = %d, want 0", got)
		}
		if !errors.Is(err, ErrInvalidHandle) {
			t.Fatalf("zero-value SizeErr() error = %v, want ErrInvalidHandle", err)
		}
	})
}
