package purejson

import (
	"errors"
	"testing"
	"unicode/utf8"
)

func FuzzParseThenGetString(f *testing.F) {
	f.Add([]byte(`"hello"`))
	f.Add([]byte(`{"name":"alice","tags":["one","two"]}`))
	f.Add([]byte(`[1,"two",true,null]`))
	f.Add([]byte(`9223372036854775807`))
	f.Add([]byte(`-9223372036854775808`))
	f.Add([]byte(`9007199254740992`))
	f.Add([]byte(`9007199254740993`))
	f.Add([]byte(`-9007199254740992`))
	f.Add([]byte(`-9007199254740993`))
	f.Add([]byte(`18446744073709551615`))
	f.Add([]byte{0x22, 0xff, 0x22})
	f.Add([]byte{0x7b, 0x22, 0x6b, 0x22, 0x3a, 0x22, 0xff, 0x22, 0x7d})

	f.Fuzz(func(t *testing.T, data []byte) {
		parser := mustNewParser(t)
		t.Cleanup(func() {
			if err := parser.Close(); err != nil {
				t.Fatalf("parser.Close() cleanup error = %v", err)
			}
		})

		doc, err := parser.Parse(data)
		if err != nil {
			if !errors.Is(err, ErrInvalidJSON) && !errors.Is(err, ErrPrecisionLoss) {
				t.Fatalf("Parse(%q) error = %v, want ErrInvalidJSON or ErrPrecisionLoss", data, err)
			}
			return
		}
		t.Cleanup(func() {
			if err := doc.Close(); err != nil {
				t.Fatalf("doc.Close() cleanup error = %v", err)
			}
		})

		fuzzWalkElement(t, doc.Root())
	})
}

func fuzzWalkElement(t *testing.T, element Element) {
	t.Helper()

	switch element.Type() {
	case TypeInvalid:
		t.Fatal("Type() = TypeInvalid after successful Parse()")
	case TypeNull:
		if !element.IsNull() {
			t.Fatal("IsNull() = false, want true")
		}
	case TypeBool:
		if _, err := element.GetBool(); err != nil {
			t.Fatalf("GetBool() error = %v", err)
		}
	case TypeInt64:
		if _, err := element.GetInt64(); err != nil {
			t.Fatalf("GetInt64() error = %v", err)
		}
		if _, err := element.GetFloat64(); err != nil && !errors.Is(err, ErrPrecisionLoss) {
			t.Fatalf("GetFloat64() on TypeInt64 error = %v, want nil or ErrPrecisionLoss", err)
		}
	case TypeUint64:
		if _, err := element.GetUint64(); err != nil {
			t.Fatalf("GetUint64() error = %v", err)
		}
		if _, err := element.GetFloat64(); err != nil && !errors.Is(err, ErrPrecisionLoss) {
			t.Fatalf("GetFloat64() on TypeUint64 error = %v, want nil or ErrPrecisionLoss", err)
		}
	case TypeFloat64:
		if _, err := element.GetFloat64(); err != nil {
			t.Fatalf("GetFloat64() error = %v", err)
		}
	case TypeString:
		value, err := element.GetString()
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		if !utf8.ValidString(value) {
			t.Fatalf("GetString() = %q, want valid UTF-8", value)
		}
	case TypeArray:
		array, err := element.AsArray()
		if err != nil {
			t.Fatalf("AsArray() error = %v", err)
		}

		iter := array.Iter()
		for iter.Next() {
			fuzzWalkElement(t, iter.Value())
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("array iter.Err() = %v", err)
		}
	case TypeObject:
		object, err := element.AsObject()
		if err != nil {
			t.Fatalf("AsObject() error = %v", err)
		}

		iter := object.Iter()
		for iter.Next() {
			if !utf8.ValidString(iter.Key()) {
				t.Fatalf("iter.Key() = %q, want valid UTF-8", iter.Key())
			}
			fuzzWalkElement(t, iter.Value())
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("object iter.Err() = %v", err)
		}
	}
}
