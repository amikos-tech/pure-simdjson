package purejson

import (
	"errors"
	"runtime"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

func TestBigIntClassification(t *testing.T) {
	if got := uint32(TypeBigInt); got != 9 {
		t.Fatalf("TypeBigInt = %d, want 9", got)
	}
	if got := uint32(ffi.ValueKindBigInt); got != 9 {
		t.Fatalf("ffi.ValueKindBigInt = %d, want 9", got)
	}

	testCases := []struct {
		name string
		json string
		want ElementType
	}{
		{name: "negative overflow", json: "-9223372036854775809", want: TypeBigInt},
		{name: "positive overflow", json: "18446744073709551616", want: TypeBigInt},
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

func TestBigIntBoundaries(t *testing.T) {
	testCases := []struct {
		name string
		json string
		want ElementType
	}{
		{name: "min int64", json: "-9223372036854775808", want: TypeInt64},
		{name: "below min int64", json: "-9223372036854775809", want: TypeBigInt},
		{name: "max uint64", json: "18446744073709551615", want: TypeUint64},
		{name: "above max uint64", json: "18446744073709551616", want: TypeBigInt},
		{name: "decimal syntax", json: "1.0", want: TypeFloat64},
		{name: "exponent syntax", json: "1e20", want: TypeFloat64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)
			root := doc.Root()

			if got := root.Type(); got != tc.want {
				t.Fatalf("Type() = %v, want %v", got, tc.want)
			}
			if tc.want == TypeBigInt {
				got, err := root.GetBigInt()
				if err != nil {
					t.Fatalf("GetBigInt() error = %v", err)
				}
				if got != tc.json {
					t.Fatalf("GetBigInt() = %q, want %q", got, tc.json)
				}
			}
		})
	}
}

func TestBigIntGetter(t *testing.T) {
	testCases := []struct {
		name string
		json string
	}{
		{name: "positive", json: "100000000000000000000"},
		{name: "negative", json: "-100000000000000000000"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)

			got, err := doc.Root().GetBigInt()
			if err != nil {
				t.Fatalf("GetBigInt() error = %v", err)
			}
			if got != tc.json {
				t.Fatalf("GetBigInt() = %q, want %q", got, tc.json)
			}
		})
	}
}

func TestBigIntWrongType(t *testing.T) {
	testCases := []struct {
		name string
		json string
	}{
		{name: "null", json: "null"},
		{name: "bool", json: "true"},
		{name: "int64", json: "-1"},
		{name: "uint64", json: "18446744073709551615"},
		{name: "float64", json: "1.0"},
		{name: "string", json: `"1"`},
		{name: "array", json: "[]"},
		{name: "object", json: "{}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, doc := mustParseDoc(t, tc.json)

			if _, err := doc.Root().GetBigInt(); !errors.Is(err, ErrWrongType) {
				t.Fatalf("GetBigInt() error = %v, want ErrWrongType", err)
			}
		})
	}

	_, doc := mustParseDoc(t, "18446744073709551616")
	root := doc.Root()
	if _, err := root.GetInt64(); !errors.Is(err, ErrWrongType) {
		t.Fatalf("GetInt64() error = %v, want ErrWrongType", err)
	}
	if _, err := root.GetUint64(); !errors.Is(err, ErrWrongType) {
		t.Fatalf("GetUint64() error = %v, want ErrWrongType", err)
	}
	if _, err := root.GetFloat64(); !errors.Is(err, ErrWrongType) {
		t.Fatalf("GetFloat64() error = %v, want ErrWrongType", err)
	}
}

func TestBigIntCopyLifetime(t *testing.T) {
	const want = "-100000000000000000000000000000000000000"

	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	doc, err := parser.Parse([]byte(want))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	got, err := doc.Root().GetBigInt()
	if err != nil {
		t.Fatalf("GetBigInt() error = %v", err)
	}
	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}

	runtime.GC()
	if got != want {
		t.Fatalf("GetBigInt() copied value after Close/GC = %q, want %q", got, want)
	}
}

func TestBigIntTraversal(t *testing.T) {
	const (
		positive = "18446744073709551616"
		negative = "-9223372036854775809"
	)

	t.Run("root", func(t *testing.T) {
		_, doc := mustParseDoc(t, positive)

		got, err := doc.Root().GetBigInt()
		if err != nil {
			t.Fatalf("GetBigInt() error = %v", err)
		}
		if got != positive {
			t.Fatalf("GetBigInt() = %q, want %q", got, positive)
		}
	})

	t.Run("object field and array iteration", func(t *testing.T) {
		_, doc := mustParseDoc(t, `{"positive":18446744073709551616,"values":[-9223372036854775809,18446744073709551616]}`)
		object, err := doc.Root().AsObject()
		if err != nil {
			t.Fatalf("AsObject() error = %v", err)
		}

		field, err := object.GetField("positive")
		if err != nil {
			t.Fatalf("GetField(\"positive\") error = %v", err)
		}
		got, err := field.GetBigInt()
		if err != nil {
			t.Fatalf("GetField(\"positive\").GetBigInt() error = %v", err)
		}
		if got != positive {
			t.Fatalf("GetField(\"positive\").GetBigInt() = %q, want %q", got, positive)
		}

		valuesField, err := object.GetField("values")
		if err != nil {
			t.Fatalf("GetField(\"values\") error = %v", err)
		}
		values, err := valuesField.AsArray()
		if err != nil {
			t.Fatalf("GetField(\"values\").AsArray() error = %v", err)
		}

		wantValues := []string{negative, positive}
		iter := values.Iter()
		var index int
		for iter.Next() {
			if index >= len(wantValues) {
				t.Fatalf("array iterator returned too many values: %d", index+1)
			}
			got, err := iter.Value().GetBigInt()
			if err != nil {
				t.Fatalf("array value %d GetBigInt() error = %v", index, err)
			}
			if got != wantValues[index] {
				t.Fatalf("array value %d = %q, want %q", index, got, wantValues[index])
			}
			index++
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("array iter.Err() = %v", err)
		}
		if index != len(wantValues) {
			t.Fatalf("array iterator returned %d values, want %d", index, len(wantValues))
		}
	})

	t.Run("object iteration", func(t *testing.T) {
		_, doc := mustParseDoc(t, `{"positive":18446744073709551616,"negative":-9223372036854775809}`)
		object, err := doc.Root().AsObject()
		if err != nil {
			t.Fatalf("AsObject() error = %v", err)
		}

		wantValues := map[string]string{
			"positive": positive,
			"negative": negative,
		}
		iter := object.Iter()
		seen := make(map[string]bool, len(wantValues))
		for iter.Next() {
			key := iter.Key()
			want, ok := wantValues[key]
			if !ok {
				t.Fatalf("object iterator returned unexpected key %q", key)
			}
			got, err := iter.Value().GetBigInt()
			if err != nil {
				t.Fatalf("object value %q GetBigInt() error = %v", key, err)
			}
			if got != want {
				t.Fatalf("object value %q = %q, want %q", key, got, want)
			}
			seen[key] = true
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("object iter.Err() = %v", err)
		}
		if len(seen) != len(wantValues) {
			t.Fatalf("object iterator saw %d values, want %d", len(seen), len(wantValues))
		}
	})
}
