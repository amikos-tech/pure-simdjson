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
		want    any
		wantErr error
	}{
		{name: "nested value", json: `{"a":{"b":42}}`, pointer: "/a/b", want: int64(42)},
		{name: "missing leading slash", json: `{"a":1}`, pointer: "a", wantErr: ErrInvalidPath},
		{name: "missing object key", json: `{"a":1}`, pointer: "/missing", wantErr: ErrElementNotFound},
		{name: "array index out of range", json: `[1,2,3]`, pointer: "/5", wantErr: ErrIndexOutOfRange},
		{name: "empty pointer returns root", json: `42`, pointer: "", want: int64(42)},
		{name: "traversal type mismatch", json: `{"a":[10,20]}`, pointer: "/a/b", wantErr: ErrWrongType},
		{name: "trailing separator selects empty key", json: `{"a":{"":"x"}}`, pointer: "/a/", want: "x"},
		{name: "trailing separator missing empty key", json: `{"a":1}`, pointer: "/a/", wantErr: ErrElementNotFound},
		{name: "tilde escape", json: `{"a~b":1}`, pointer: "/a~0b", want: int64(1)},
		{name: "slash escape", json: `{"a/b":2}`, pointer: "/a~1b", want: int64(2)},
		{name: "invalid escape", json: `{"a~2b":1}`, pointer: "/a~2b", wantErr: ErrInvalidPath},
		{name: "leading zero array index", json: `[10,20]`, pointer: "/01", wantErr: ErrInvalidPath},
		{name: "array dash token", json: `[10,20]`, pointer: "/-", wantErr: ErrIndexOutOfRange},
		{name: "leading zero object key", json: `{"01":"ok"}`, pointer: "/01", want: "ok"},
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
			switch want := tc.want.(type) {
			case int64:
				value, err := got.GetInt64()
				if err != nil {
					t.Fatalf("GetInt64() error = %v", err)
				}
				if value != want {
					t.Fatalf("GetInt64() = %d, want %d", value, want)
				}
			case string:
				value, err := got.GetString()
				if err != nil {
					t.Fatalf("GetString() error = %v", err)
				}
				if value != want {
					t.Fatalf("GetString() = %q, want %q", value, want)
				}
			default:
				t.Fatalf("unsupported expected value type %T", tc.want)
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
		{name: "empty path", json: `{"name":1}`, path: "", wantErr: ErrInvalidPath},
		{name: "dollar prefix", json: `{"a":{"b":42}}`, path: "$.a.b", want: 42},
		{
			name: "quoted bracket key stays quoted",
			json: `{"obj":{"'foo'":1,"foo":2}}`,
			path: ".obj['foo']",
			want: 1,
		},
		{
			name: "unquoted bracket key",
			json: `{"obj":{"'foo'":1,"foo":2}}`,
			path: ".obj[foo]",
			want: 2,
		},
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
			name: "partial heterogeneous branches",
			json: `{"items":[{"id":1},{"other":2},3,{"id":4}]}`,
			path: ".items[*].id",
			want: []int64{1, 4},
		},
		{
			name: "missing prefix",
			json: `{"items":[]}`,
			path: ".missing[*].id",
			want: []int64{},
		},
		{
			name: "out of range index",
			json: `{"items":[{"id":1}]}`,
			path: ".items[5].*",
			want: []int64{},
		},
		{
			name: "scalar receiver",
			json: `42`,
			path: ".*",
			want: []int64{},
		},
		{
			name: "non-container branches",
			json: `{"items":[1,2]}`,
			path: ".items[*].id",
			want: []int64{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)

			got, err := doc.Root().AtPathAll(tc.path)
			for _, branchErr := range []error{ErrElementNotFound, ErrIndexOutOfRange, ErrWrongType} {
				if errors.Is(err, branchErr) {
					t.Fatalf("AtPathAll(%q) leaked branch error %v", tc.path, branchErr)
				}
			}
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

	t.Run("malformed wildcard path", func(t *testing.T) {
		_, doc := mustParseDoc(t, `{"a":1}`)

		for _, path := range []string{"*", ".items[*].", ".items[*][", "['*'][*]", ".items[*]junk[0]", ".a[*]b[0]", ".rows[01][*]"} {
			if _, err := doc.Root().AtPathAll(path); !errors.Is(err, ErrInvalidPath) {
				t.Fatalf("AtPathAll(%q) error = %v, want ErrInvalidPath", path, err)
			}
		}
	})

	t.Run("quoted bracket keys preserve AtPath semantics around wildcard", func(t *testing.T) {
		_, doc := mustParseDoc(t, `{"obj":{"'foo'":[1,2],"foo":[3],"'foo.bar'":[1,2],"foo.bar":[3]},"items":[{"'foo'":4,"foo":5}],"arr":[[1,2]]}`)

		for _, tc := range []struct {
			path string
			want []int64
		}{
			{path: ".obj['foo'][*]", want: []int64{1, 2}},
			{path: ".items[*]['foo']", want: []int64{4}},
			{path: ".obj['foo.bar'][*]", want: []int64{1, 2}},
			{path: ".arr[0][*]", want: []int64{1, 2}},
		} {
			got, err := doc.Root().AtPathAll(tc.path)
			if err != nil {
				t.Fatalf("AtPathAll(%q) error = %v", tc.path, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("AtPathAll(%q) len = %d, want %d", tc.path, len(got), len(tc.want))
			}
			for i, want := range tc.want {
				value, err := got[i].GetInt64()
				if err != nil || value != want {
					t.Fatalf("AtPathAll(%q)[%d] = (%d, %v), want (%d, nil)", tc.path, i, value, err, want)
				}
			}
		}
	})

	t.Run("leading quoted bracket key is a valid literal prefix", func(t *testing.T) {
		_, doc := mustParseDoc(t, `{"'obj'":{"first":1,"second":2}}`)
		got, err := doc.Root().AtPathAll("['obj'].*")
		if err != nil {
			t.Fatalf("AtPathAll() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("AtPathAll() len = %d, want 2", len(got))
		}
		for index, want := range []int64{1, 2} {
			value, err := got[index].GetInt64()
			if err != nil || value != want {
				t.Fatalf("AtPathAll()[%d] = (%d, %v), want (%d, nil)", index, value, err, want)
			}
		}
	})
}

func TestElement_NavigationAfterClose(t *testing.T) {
	_, doc := mustParseDoc(t, `{"a":{"b":42},"items":[{"id":1}]}`)
	root := doc.Root()

	pointerResult, err := root.AtPointer("/a")
	if err != nil {
		t.Fatalf("AtPointer() error = %v", err)
	}
	pathResult, err := root.AtPath(".a")
	if err != nil {
		t.Fatalf("AtPath() error = %v", err)
	}
	wildcardResults, err := root.AtPathAll(".items[*]")
	if err != nil {
		t.Fatalf("AtPathAll() error = %v", err)
	}
	if len(wildcardResults) != 1 {
		t.Fatalf("AtPathAll() len = %d, want 1", len(wildcardResults))
	}

	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}

	testCases := []struct {
		name     string
		navigate func() error
	}{
		{
			name: "pointer result",
			navigate: func() error {
				_, err := pointerResult.AtPointer("/b")
				return err
			},
		},
		{
			name: "path result",
			navigate: func() error {
				_, err := pathResult.AtPath(".b")
				return err
			},
		},
		{
			name: "wildcard result",
			navigate: func() error {
				_, err := wildcardResults[0].AtPointer("/id")
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.navigate(); !errors.Is(err, ErrClosed) {
				t.Fatalf("navigation error = %v, want ErrClosed", err)
			}
		})
	}
}
