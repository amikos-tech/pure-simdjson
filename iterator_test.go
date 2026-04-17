package purejson

import (
	"errors"
	"slices"
	"testing"
	"unicode/utf8"
)

func TestArrayIterOrder(t *testing.T) {
	_, doc := mustParseDoc(t, `[1,"two",null,true]`)

	array, err := doc.Root().AsArray()
	if err != nil {
		t.Fatalf("AsArray() error = %v", err)
	}

	iter := array.Iter()
	index := 0
	for iter.Next() {
		value := iter.Value()
		switch index {
		case 0:
			got, err := value.GetInt64()
			if err != nil {
				t.Fatalf("Value().GetInt64() error = %v", err)
			}
			if got != 1 {
				t.Fatalf("Value().GetInt64() = %d, want 1", got)
			}
		case 1:
			got, err := value.GetString()
			if err != nil {
				t.Fatalf("Value().GetString() error = %v", err)
			}
			if got != "two" {
				t.Fatalf("Value().GetString() = %q, want %q", got, "two")
			}
			if !utf8.ValidString(got) {
				t.Fatalf("Value().GetString() = %q, want valid UTF-8", got)
			}
		case 2:
			if !value.IsNull() {
				t.Fatal("Value().IsNull() = false, want true")
			}
		case 3:
			got, err := value.GetBool()
			if err != nil {
				t.Fatalf("Value().GetBool() error = %v", err)
			}
			if !got {
				t.Fatal("Value().GetBool() = false, want true")
			}
		default:
			t.Fatalf("iterated too many values: index=%d", index)
		}
		index++
	}

	if err := iter.Err(); err != nil {
		t.Fatalf("iter.Err() = %v, want nil", err)
	}
	if index != 4 {
		t.Fatalf("iterated %d values, want 4", index)
	}
}

func TestArrayIterEmpty(t *testing.T) {
	_, doc := mustParseDoc(t, `[]`)

	array, err := doc.Root().AsArray()
	if err != nil {
		t.Fatalf("AsArray() error = %v", err)
	}

	iter := array.Iter()
	if iter.Next() {
		t.Fatal("iter.Next() = true, want false")
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("iter.Err() = %v, want nil", err)
	}
}

func TestObjectIterOrder(t *testing.T) {
	_, doc := mustParseDoc(t, `{"first":"alpha","second":2,"third":null}`)

	object, err := doc.Root().AsObject()
	if err != nil {
		t.Fatalf("AsObject() error = %v", err)
	}

	iter := object.Iter()
	var keys []string
	for iter.Next() {
		keys = append(keys, iter.Key())
		if !utf8.ValidString(iter.Key()) {
			t.Fatalf("iter.Key() = %q, want valid UTF-8", iter.Key())
		}

		value := iter.Value()
		switch len(keys) - 1 {
		case 0:
			got, err := value.GetString()
			if err != nil {
				t.Fatalf("Value().GetString() error = %v", err)
			}
			if got != "alpha" {
				t.Fatalf("Value().GetString() = %q, want %q", got, "alpha")
			}
			if !utf8.ValidString(got) {
				t.Fatalf("Value().GetString() = %q, want valid UTF-8", got)
			}
		case 1:
			got, err := value.GetInt64()
			if err != nil {
				t.Fatalf("Value().GetInt64() error = %v", err)
			}
			if got != 2 {
				t.Fatalf("Value().GetInt64() = %d, want 2", got)
			}
		case 2:
			if !value.IsNull() {
				t.Fatal("Value().IsNull() = false, want true")
			}
		default:
			t.Fatalf("iterated too many object entries: len=%d", len(keys))
		}
	}

	if err := iter.Err(); err != nil {
		t.Fatalf("iter.Err() = %v, want nil", err)
	}

	wantKeys := []string{"first", "second", "third"}
	if !slices.Equal(keys, wantKeys) {
		t.Fatalf("iterated keys = %v, want %v", keys, wantKeys)
	}

	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}
	if !slices.Equal(keys, wantKeys) {
		t.Fatalf("keys after doc.Close() = %v, want %v", keys, wantKeys)
	}
}

func TestObjectIterEmpty(t *testing.T) {
	_, doc := mustParseDoc(t, `{}`)

	object, err := doc.Root().AsObject()
	if err != nil {
		t.Fatalf("AsObject() error = %v", err)
	}

	iter := object.Iter()
	if iter.Next() {
		t.Fatal("iter.Next() = true, want false")
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("iter.Err() = %v, want nil", err)
	}
}

func TestObjectGetFieldMissingVsNull(t *testing.T) {
	_, doc := mustParseDoc(t, `{"present":null,"name":"alice"}`)

	object, err := doc.Root().AsObject()
	if err != nil {
		t.Fatalf("AsObject() error = %v", err)
	}

	nullField, err := object.GetField("present")
	if err != nil {
		t.Fatalf("GetField(\"present\") error = %v", err)
	}
	if !nullField.IsNull() {
		t.Fatal("GetField(\"present\").IsNull() = false, want true")
	}

	nameField, err := object.GetField("name")
	if err != nil {
		t.Fatalf("GetField(\"name\") error = %v", err)
	}
	name, err := nameField.GetString()
	if err != nil {
		t.Fatalf("GetString() error = %v", err)
	}
	if name != "alice" {
		t.Fatalf("GetString() = %q, want %q", name, "alice")
	}

	if _, err := object.GetField("missing"); !errors.Is(err, ErrElementNotFound) {
		t.Fatalf("GetField(\"missing\") error = %v, want ErrElementNotFound", err)
	}
}

func TestGetStringField(t *testing.T) {
	_, doc := mustParseDoc(t, `{"name":"alice"}`)

	object, err := doc.Root().AsObject()
	if err != nil {
		t.Fatalf("AsObject() error = %v", err)
	}

	got, err := object.GetStringField("name")
	if err != nil {
		t.Fatalf("GetStringField(\"name\") error = %v", err)
	}
	if got != "alice" {
		t.Fatalf("GetStringField(\"name\") = %q, want %q", got, "alice")
	}

	if _, err := object.GetStringField("missing"); !errors.Is(err, ErrElementNotFound) {
		t.Fatalf("GetStringField(\"missing\") error = %v, want ErrElementNotFound", err)
	}
}

func TestGetStringFieldNullValue(t *testing.T) {
	_, doc := mustParseDoc(t, `{"name":null}`)

	object, err := doc.Root().AsObject()
	if err != nil {
		t.Fatalf("AsObject() error = %v", err)
	}

	if _, err := object.GetStringField("name"); !errors.Is(err, ErrWrongType) {
		t.Fatalf("GetStringField(\"name\") error = %v, want ErrWrongType", err)
	}
}

func TestObjectGetFieldEmptyKey(t *testing.T) {
	_, doc := mustParseDoc(t, `{"":1,"name":"alice"}`)

	object, err := doc.Root().AsObject()
	if err != nil {
		t.Fatalf("AsObject() error = %v", err)
	}

	field, err := object.GetField("")
	if err != nil {
		t.Fatalf("GetField(\"\") error = %v", err)
	}

	value, err := field.GetInt64()
	if err != nil {
		t.Fatalf("GetField(\"\").GetInt64() error = %v", err)
	}
	if value != 1 {
		t.Fatalf("GetField(\"\").GetInt64() = %d, want 1", value)
	}

	got, err := object.GetStringField("name")
	if err != nil {
		t.Fatalf("GetStringField(\"name\") error = %v", err)
	}
	if got != "alice" {
		t.Fatalf("GetStringField(\"name\") = %q, want %q", got, "alice")
	}
}

func TestZeroValueIteratorsReportInvalidHandle(t *testing.T) {
	var array Array
	arrayIter := array.Iter()
	if arrayIter == nil {
		t.Fatal("zero-value Array.Iter() = nil, want iterator")
	}
	if !errors.Is(arrayIter.Err(), ErrInvalidHandle) {
		t.Fatalf("zero-value Array.Iter().Err() = %v, want ErrInvalidHandle", arrayIter.Err())
	}
	if arrayIter.Next() {
		t.Fatal("zero-value Array.Iter().Next() = true, want false")
	}

	var object Object
	objectIter := object.Iter()
	if objectIter == nil {
		t.Fatal("zero-value Object.Iter() = nil, want iterator")
	}
	if !errors.Is(objectIter.Err(), ErrInvalidHandle) {
		t.Fatalf("zero-value Object.Iter().Err() = %v, want ErrInvalidHandle", objectIter.Err())
	}
	if objectIter.Next() {
		t.Fatal("zero-value Object.Iter().Next() = true, want false")
	}

	if _, err := object.GetField("name"); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("zero-value Object.GetField() error = %v, want ErrInvalidHandle", err)
	}
	if _, err := object.GetStringField("name"); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("zero-value Object.GetStringField() error = %v, want ErrInvalidHandle", err)
	}
}

func TestParseRejectsMalformedUTF8Objects(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{name: "invalid key", data: []byte{0x7b, 0x22, 0xff, 0x22, 0x3a, 0x22, 0x6f, 0x6b, 0x22, 0x7d}},
		{name: "invalid string value", data: []byte{0x7b, 0x22, 0x6b, 0x22, 0x3a, 0x22, 0xff, 0x22, 0x7d}},
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

func TestIteratorNextAfterDone(t *testing.T) {
	_, doc := mustParseDoc(t, `[1]`)

	array, err := doc.Root().AsArray()
	if err != nil {
		t.Fatalf("AsArray() error = %v", err)
	}

	iter := array.Iter()
	if !iter.Next() {
		t.Fatalf("first iter.Next() = false, want true (err=%v)", iter.Err())
	}

	got, err := iter.Value().GetInt64()
	if err != nil {
		t.Fatalf("Value().GetInt64() error = %v", err)
	}
	if got != 1 {
		t.Fatalf("Value().GetInt64() = %d, want 1", got)
	}

	if iter.Next() {
		t.Fatal("second iter.Next() = true, want false")
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("iter.Err() after done = %v, want nil", err)
	}

	if iter.Next() {
		t.Fatal("third iter.Next() = true, want false")
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("iter.Err() after repeated done = %v, want nil", err)
	}
}

func TestIteratorAfterDocClose(t *testing.T) {
	t.Run("before-first-next", func(t *testing.T) {
		_, doc := mustParseDoc(t, `[1,2]`)

		array, err := doc.Root().AsArray()
		if err != nil {
			t.Fatalf("AsArray() error = %v", err)
		}

		iter := array.Iter()
		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() error = %v", err)
		}
		if iter.Next() {
			t.Fatal("iter.Next() after doc.Close() = true, want false")
		}
		if !errors.Is(iter.Err(), ErrClosed) {
			t.Fatalf("iter.Err() after doc.Close() = %v, want ErrClosed", iter.Err())
		}
	})

	t.Run("mid-iteration", func(t *testing.T) {
		_, doc := mustParseDoc(t, `{"a":"first","b":"second"}`)

		object, err := doc.Root().AsObject()
		if err != nil {
			t.Fatalf("AsObject() error = %v", err)
		}

		iter := object.Iter()
		if !iter.Next() {
			t.Fatalf("first iter.Next() = false, want true (err=%v)", iter.Err())
		}
		if iter.Key() != "a" {
			t.Fatalf("iter.Key() = %q, want %q", iter.Key(), "a")
		}

		if err := doc.Close(); err != nil {
			t.Fatalf("doc.Close() error = %v", err)
		}
		if iter.Next() {
			t.Fatal("iter.Next() after doc.Close() = true, want false")
		}
		if !errors.Is(iter.Err(), ErrClosed) {
			t.Fatalf("iter.Err() after doc.Close() = %v, want ErrClosed", iter.Err())
		}
	})
}
