package purejson

import (
	"encoding/binary"
	"errors"
	"testing"
)

var descendantViewTag = binary.LittleEndian.Uint64([]byte("PSDJDESC"))

func mustParseDoc(t *testing.T, json string) (*Parser, *Doc) {
	t.Helper()

	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	doc, err := parser.Parse([]byte(json))
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", json, err)
	}
	t.Cleanup(func() {
		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() cleanup error = %v", err)
		}
	})

	return parser, doc
}

func TestElementTypeClassification(t *testing.T) {
	testCases := []struct {
		name string
		json string
		want ElementType
	}{
		{name: "int64", json: "42", want: TypeInt64},
		{name: "uint64", json: "18446744073709551615", want: TypeUint64},
		{name: "float64", json: "3.25", want: TypeFloat64},
		{name: "string", json: `"hello"`, want: TypeString},
		{name: "bool", json: "true", want: TypeBool},
		{name: "null", json: "null", want: TypeNull},
		{name: "array", json: "[]", want: TypeArray},
		{name: "object", json: "{}", want: TypeObject},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)

			if got := doc.Root().Type(); got != tc.want {
				t.Fatalf("Type() = %v, want %v", got, tc.want)
			}
		})
	}

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

	if got := root.Type(); got != TypeInvalid {
		t.Fatalf("Type() after Close = %v, want %v", got, TypeInvalid)
	}
}

func TestTypeInvalidOnTamperedView(t *testing.T) {
	testCases := []struct {
		name   string
		tamper func(*Element)
	}{
		{
			name: "invalid descendant tag",
			tamper: func(element *Element) {
				element.view.State0 = 1
				element.view.State1 = descendantViewTag
			},
		},
		{
			name: "reserved bits",
			tamper: func(element *Element) {
				element.view.Reserved = 1
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, "42")
			root := doc.Root()
			tc.tamper(&root)

			if got := root.Type(); got != TypeInvalid {
				t.Fatalf("Type() = %v, want %v", got, TypeInvalid)
			}
		})
	}
}

func TestGetUint64(t *testing.T) {
	_, doc := mustParseDoc(t, "18446744073709551615")

	value, err := doc.Root().GetUint64()
	if err != nil {
		t.Fatalf("GetUint64() error = %v", err)
	}
	if value != ^uint64(0) {
		t.Fatalf("GetUint64() = %d, want %d", value, ^uint64(0))
	}

	t.Run("negative", func(t *testing.T) {
		_, doc := mustParseDoc(t, "-1")

		if _, err := doc.Root().GetUint64(); !errors.Is(err, ErrNumberOutOfRange) {
			t.Fatalf("GetUint64() error = %v, want ErrNumberOutOfRange", err)
		}
	})

	t.Run("oversized literal rejected at parse", func(t *testing.T) {
		parser := mustNewParser(t)
		t.Cleanup(func() {
			if err := parser.Close(); err != nil {
				t.Fatalf("parser.Close() cleanup error = %v", err)
			}
		})

		if _, err := parser.Parse([]byte("18446744073709551616")); !errors.Is(err, ErrInvalidJSON) {
			t.Fatalf("Parse() oversized uint64 error = %v, want ErrInvalidJSON", err)
		}
	})
}

func TestGetFloat64(t *testing.T) {
	_, doc := mustParseDoc(t, "1.5")

	value, err := doc.Root().GetFloat64()
	if err != nil {
		t.Fatalf("GetFloat64() error = %v", err)
	}
	if value != 1.5 {
		t.Fatalf("GetFloat64() = %v, want 1.5", value)
	}

	_, doc = mustParseDoc(t, "9007199254740993")
	if _, err := doc.Root().GetFloat64(); !errors.Is(err, ErrPrecisionLoss) {
		t.Fatalf("GetFloat64() precision-loss error = %v, want ErrPrecisionLoss", err)
	}
}

func TestGetString(t *testing.T) {
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

	value, err := doc.Root().GetString()
	if err != nil {
		t.Fatalf("GetString() error = %v", err)
	}
	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}
	if value != "hello" {
		t.Fatalf("GetString() copied value = %q, want %q", value, "hello")
	}
}

func TestGetBool(t *testing.T) {
	_, doc := mustParseDoc(t, "true")

	value, err := doc.Root().GetBool()
	if err != nil {
		t.Fatalf("GetBool() error = %v", err)
	}
	if !value {
		t.Fatal("GetBool() = false, want true")
	}

	_, doc = mustParseDoc(t, "1")
	if _, err := doc.Root().GetBool(); !errors.Is(err, ErrWrongType) {
		t.Fatalf("GetBool() wrong-type error = %v, want ErrWrongType", err)
	}
}

func TestIsNull(t *testing.T) {
	_, doc := mustParseDoc(t, "null")
	root := doc.Root()

	if !root.IsNull() {
		t.Fatal("IsNull() = false, want true")
	}

	_, doc = mustParseDoc(t, "false")
	if doc.Root().IsNull() {
		t.Fatal("IsNull() on bool = true, want false")
	}

	_, doc = mustParseDoc(t, "null")
	tampered := doc.Root()
	tampered.view.State0 = 1
	tampered.view.State1 = descendantViewTag
	if tampered.IsNull() {
		t.Fatal("IsNull() on tampered view = true, want false")
	}

	_, doc = mustParseDoc(t, "null")
	closed := doc.Root()
	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}
	if closed.IsNull() {
		t.Fatal("IsNull() after Close = true, want false")
	}
}
