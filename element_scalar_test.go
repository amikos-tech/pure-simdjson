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

func TestElementTypeErrClassification(t *testing.T) {
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

			got, err := doc.Root().TypeErr()
			if err != nil {
				t.Fatalf("TypeErr() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("TypeErr() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestElementTypeErrPreservesErrors(t *testing.T) {
	t.Run("after close", func(t *testing.T) {
		_, doc := mustParseDoc(t, "42")
		root := doc.Root()
		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() error = %v", err)
		}

		got, err := root.TypeErr()
		if got != TypeInvalid {
			t.Fatalf("TypeErr() type = %v, want %v", got, TypeInvalid)
		}
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("TypeErr() error = %v, want ErrClosed", err)
		}
	})

	t.Run("tampered view", func(t *testing.T) {
		_, doc := mustParseDoc(t, "42")
		root := doc.Root()
		root.view.State0 = 1
		root.view.State1 = descendantViewTag

		got, err := root.TypeErr()
		if got != TypeInvalid {
			t.Fatalf("TypeErr() type = %v, want %v", got, TypeInvalid)
		}
		if !errors.Is(err, ErrInvalidHandle) {
			t.Fatalf("TypeErr() error = %v, want ErrInvalidHandle", err)
		}
	})
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

func TestZeroValueDocRootSemantics(t *testing.T) {
	var doc Doc
	root := doc.Root()

	if got := root.Type(); got != TypeInvalid {
		t.Fatalf("zero-value Doc Root().Type() = %v, want %v", got, TypeInvalid)
	}
	if root.IsNull() {
		t.Fatal("zero-value Doc Root().IsNull() = true, want false")
	}
	if _, err := root.GetInt64(); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("zero-value Doc Root().GetInt64() error = %v, want ErrInvalidHandle", err)
	}
	if _, err := root.AsArray(); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("zero-value Doc Root().AsArray() error = %v, want ErrInvalidHandle", err)
	}
	if _, err := root.AsObject(); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("zero-value Doc Root().AsObject() error = %v, want ErrInvalidHandle", err)
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

func TestGetInt64BoundaryContract(t *testing.T) {
	t.Run("max int64", func(t *testing.T) {
		_, doc := mustParseDoc(t, "9223372036854775807")

		value, err := doc.Root().GetInt64()
		if err != nil {
			t.Fatalf("GetInt64() error = %v", err)
		}
		if value != 9223372036854775807 {
			t.Fatalf("GetInt64() = %d, want %d", value, int64(9223372036854775807))
		}
	})

	t.Run("uint64 above int64 max", func(t *testing.T) {
		_, doc := mustParseDoc(t, "9223372036854775808")

		if _, err := doc.Root().GetInt64(); !errors.Is(err, ErrNumberOutOfRange) {
			t.Fatalf("GetInt64() error = %v, want ErrNumberOutOfRange", err)
		}
	})

	t.Run("float-kind reports wrong type", func(t *testing.T) {
		_, doc := mustParseDoc(t, "1e20")

		if _, err := doc.Root().GetInt64(); !errors.Is(err, ErrWrongType) {
			t.Fatalf("GetInt64() error = %v, want ErrWrongType", err)
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

func TestGetFloat64PrecisionBoundaries(t *testing.T) {
	testCases := []struct {
		name    string
		json    string
		want    float64
		wantErr error
	}{
		{name: "positive exact boundary", json: "9007199254740992", want: 9007199254740992},
		{name: "positive precision loss", json: "9007199254740993", wantErr: ErrPrecisionLoss},
		{name: "negative exact boundary", json: "-9007199254740992", want: -9007199254740992},
		{name: "negative precision loss", json: "-9007199254740993", wantErr: ErrPrecisionLoss},
		{name: "max uint64 precision loss", json: "18446744073709551615", wantErr: ErrPrecisionLoss},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)

			got, err := doc.Root().GetFloat64()
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("GetFloat64() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetFloat64() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("GetFloat64() = %.0f, want %.0f", got, tc.want)
			}
		})
	}
}

func TestParseRejectsMalformedUTF8Scalars(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{name: "root string", data: []byte{0x22, 0xff, 0x22}},
		{name: "array string", data: []byte{0x5b, 0x22, 0xff, 0x22, 0x5d}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := mustNewParser(t)
			t.Cleanup(func() {
				if err := parser.Close(); err != nil {
					t.Fatalf("parser.Close() cleanup error = %v", err)
				}
			})

			doc, err := parser.Parse(tc.data)
			if doc != nil {
				t.Fatalf("Parse(%q) unexpectedly returned a document", tc.data)
			}
			if !errors.Is(err, ErrInvalidJSON) {
				t.Fatalf("Parse(%q) error = %v, want ErrInvalidJSON", tc.data, err)
			}
		})
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

func TestIsNullErr(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		_, doc := mustParseDoc(t, "null")

		got, err := doc.Root().IsNullErr()
		if err != nil {
			t.Fatalf("IsNullErr() error = %v", err)
		}
		if !got {
			t.Fatal("IsNullErr() = false, want true")
		}
	})

	t.Run("bool", func(t *testing.T) {
		_, doc := mustParseDoc(t, "false")

		got, err := doc.Root().IsNullErr()
		if err != nil {
			t.Fatalf("IsNullErr() error = %v", err)
		}
		if got {
			t.Fatal("IsNullErr() on bool = true, want false")
		}
	})

	t.Run("tampered view", func(t *testing.T) {
		_, doc := mustParseDoc(t, "null")
		tampered := doc.Root()
		tampered.view.State0 = 1
		tampered.view.State1 = descendantViewTag

		got, err := tampered.IsNullErr()
		if got {
			t.Fatal("IsNullErr() on tampered view = true, want false")
		}
		if !errors.Is(err, ErrInvalidHandle) {
			t.Fatalf("IsNullErr() error = %v, want ErrInvalidHandle", err)
		}
	})

	t.Run("after close", func(t *testing.T) {
		_, doc := mustParseDoc(t, "null")
		root := doc.Root()
		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() error = %v", err)
		}

		got, err := root.IsNullErr()
		if got {
			t.Fatal("IsNullErr() after Close = true, want false")
		}
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("IsNullErr() error = %v, want ErrClosed", err)
		}
	})
}
