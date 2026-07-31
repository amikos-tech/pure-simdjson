package purejson

import (
	"errors"
	"testing"
)

func TestElement_AtPointer(t *testing.T) {
	testCases := []struct {
		name    string
		json    string
		pointer string
		want    int64
		wantErr error
	}{
		{name: "nested value", json: `{"a":{"b":42}}`, pointer: "/a/b", want: 42},
		{name: "missing leading slash", json: `{"a":1}`, pointer: "a", wantErr: ErrInvalidPath},
		{name: "missing object key", json: `{"a":1}`, pointer: "/missing", wantErr: ErrElementNotFound},
		{name: "array index out of range", json: `[1,2,3]`, pointer: "/5", wantErr: ErrIndexOutOfRange},
		{name: "traversal type mismatch", json: `{"a":[10,20]}`, pointer: "/a/b", wantErr: ErrWrongType},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)

			got, err := doc.Root().AtPointer(tc.pointer)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("AtPointer(%q) error = %v, want %v", tc.pointer, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("AtPointer(%q) error = %v", tc.pointer, err)
			}
			value, err := got.GetInt64()
			if err != nil {
				t.Fatalf("GetInt64() error = %v", err)
			}
			if value != tc.want {
				t.Fatalf("GetInt64() = %d, want %d", value, tc.want)
			}
		})
	}
}

func TestElement_AtPath(t *testing.T) {
	testCases := []struct {
		name    string
		json    string
		path    string
		want    int64
		wantErr error
	}{
		{name: "nested value", json: `{"a":{"b":42}}`, path: ".a.b", want: 42},
		{name: "traversal type mismatch", json: `{"a":[10,20]}`, path: ".a.b", wantErr: ErrWrongType},
		{name: "missing leading separator", json: `{"name":1}`, path: "name", wantErr: ErrInvalidPath},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)

			got, err := doc.Root().AtPath(tc.path)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("AtPath(%q) error = %v, want %v", tc.path, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("AtPath(%q) error = %v", tc.path, err)
			}
			value, err := got.GetInt64()
			if err != nil {
				t.Fatalf("GetInt64() error = %v", err)
			}
			if value != tc.want {
				t.Fatalf("GetInt64() = %d, want %d", value, tc.want)
			}
		})
	}
}

func TestElement_AtPathAll(t *testing.T) {
	t.Run("wildcard required", func(t *testing.T) {
		_, doc := mustParseDoc(t, `{"a":{"b":1}}`)

		if _, err := doc.Root().AtPathAll(".a.b"); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("AtPathAll(%q) error = %v, want ErrInvalidPath", ".a.b", err)
		}
	})

	testCases := []struct {
		name string
		json string
		path string
		want []int64
	}{
		{
			name: "document order",
			json: `{"items":[{"id":1},{"id":2},{"id":3}]}`,
			path: ".items[*].id",
			want: []int64{1, 2, 3},
		},
		{
			name: "empty match",
			json: `{"items":[]}`,
			path: ".items[*].id",
			want: []int64{},
		},
		{
			name: "missing and non-container branches skipped",
			json: `{"items":[{"id":1},{"other":2},3]}`,
			path: ".items[*].id",
			want: []int64{1},
		},
		{
			name: "missing prefix",
			json: `{"items":[]}`,
			path: ".missing[*].id",
			want: []int64{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)

			got, err := doc.Root().AtPathAll(tc.path)
			if err != nil {
				t.Fatalf("AtPathAll(%q) error = %v", tc.path, err)
			}
			if got == nil {
				t.Fatalf("AtPathAll(%q) returned a nil slice", tc.path)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("AtPathAll(%q) len = %d, want %d", tc.path, len(got), len(tc.want))
			}
			for i, element := range got {
				value, err := element.GetInt64()
				if err != nil {
					t.Fatalf("result[%d].GetInt64() error = %v", i, err)
				}
				if value != tc.want[i] {
					t.Fatalf("result[%d] = %d, want %d", i, value, tc.want[i])
				}
			}
		})
	}
}
