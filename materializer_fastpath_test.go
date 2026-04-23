package purejson

import (
	"errors"
	"reflect"
	"runtime"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

func TestFastMaterializerParity(t *testing.T) {
	_, doc := mustParseDoc(t, `{"outer":[1,18446744073709551615,3.5,true,null,"value",{"nested":"yes"}]}`)

	baseline := materializeViaAccessorsForTest(t, doc.Root())
	object, ok := baseline.(map[string]any)
	if !ok {
		t.Fatalf("accessor baseline root = %T, want map[string]any", baseline)
	}
	outer, ok := object["outer"].([]any)
	if !ok {
		t.Fatalf("accessor baseline outer = %T, want []any", object["outer"])
	}
	if len(outer) != 7 {
		t.Fatalf("accessor baseline outer length = %d, want 7", len(outer))
	}
	if got, ok := outer[0].(int64); !ok || got != 1 {
		t.Fatalf("accessor baseline outer[0] = %v (%T), want int64(1)", outer[0], outer[0])
	}
	if got, ok := outer[1].(uint64); !ok || got != ^uint64(0) {
		t.Fatalf("accessor baseline outer[1] = %v (%T), want max uint64", outer[1], outer[1])
	}
	if got, ok := outer[2].(float64); !ok || got != 3.5 {
		t.Fatalf("accessor baseline outer[2] = %v (%T), want float64(3.5)", outer[2], outer[2])
	}

	got, err := fastMaterializeElement(doc.Root())
	if err != nil {
		t.Fatalf("fastMaterializeElement() error = %v", err)
	}
	if !reflect.DeepEqual(got, baseline) {
		t.Fatalf("fastMaterializeElement() = %#v, want %#v", got, baseline)
	}
}

func TestAccessorMaterializerParity(t *testing.T) {
	_, doc := mustParseDoc(t, `{"outer":[1,18446744073709551615,3.5,true,null,"value",{"nested":"yes"}]}`)

	want := materializeViaAccessorsForTest(t, doc.Root())
	got, err := materializeElementViaAccessors(doc.Root())
	if err != nil {
		t.Fatalf("materializeElementViaAccessors() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("materializeElementViaAccessors() = %#v, want %#v", got, want)
	}
}

func TestFastMaterializerNumericSemantics(t *testing.T) {
	_, doc := mustParseDoc(t, `[9223372036854775807,18446744073709551615,1.25]`)

	baseline := materializeViaAccessorsForTest(t, doc.Root())
	values, ok := baseline.([]any)
	if !ok {
		t.Fatalf("accessor baseline root = %T, want []any", baseline)
	}
	if got, ok := values[0].(int64); !ok || got != 9223372036854775807 {
		t.Fatalf("accessor baseline values[0] = %v (%T), want max int64", values[0], values[0])
	}
	if got, ok := values[1].(uint64); !ok || got != ^uint64(0) {
		t.Fatalf("accessor baseline values[1] = %v (%T), want max uint64", values[1], values[1])
	}
	if got, ok := values[2].(float64); !ok || got != 1.25 {
		t.Fatalf("accessor baseline values[2] = %v (%T), want float64(1.25)", values[2], values[2])
	}

	got, err := fastMaterializeElement(doc.Root())
	if err != nil {
		t.Fatalf("fastMaterializeElement() error = %v", err)
	}
	if !reflect.DeepEqual(got, baseline) {
		t.Fatalf("fastMaterializeElement() = %#v, want %#v", got, baseline)
	}
}

func TestFastMaterializerOversizedLiteralParseRejected(t *testing.T) {
	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	doc, err := parser.Parse([]byte(`{"ok":1,"big":99999999999999999999999}`))
	if doc != nil {
		t.Fatal("Parse() oversized literal unexpectedly returned a document")
	}
	if !errors.Is(err, ErrInvalidJSON) {
		t.Fatalf("Parse() oversized literal error = %v, want ErrInvalidJSON", err)
	}
}

func TestFastMaterializerDuplicateKeySemantics(t *testing.T) {
	_, doc := mustParseDoc(t, `{"dup":1,"dup":2}`)

	baseline := materializeViaAccessorsForTest(t, doc.Root())
	object, ok := baseline.(map[string]any)
	if !ok {
		t.Fatalf("accessor baseline root = %T, want map[string]any", baseline)
	}
	if got, ok := object["dup"].(int64); !ok || got != 2 {
		t.Fatalf("accessor baseline dup = %v (%T), want last duplicate int64(2)", object["dup"], object["dup"])
	}

	domObject, err := doc.Root().AsObject()
	if err != nil {
		t.Fatalf("AsObject() error = %v", err)
	}
	firstField, err := domObject.GetField("dup")
	if err != nil {
		t.Fatalf("GetField(\"dup\") error = %v", err)
	}
	if got, err := firstField.GetInt64(); err != nil || got != 1 {
		t.Fatalf("GetField(\"dup\").GetInt64() = %d, %v; want first duplicate int64(1)", got, err)
	}

	got, err := fastMaterializeElement(doc.Root())
	if err != nil {
		t.Fatalf("fastMaterializeElement() error = %v", err)
	}
	object, ok = got.(map[string]any)
	if !ok {
		t.Fatalf("fast materialized root = %T, want map[string]any", got)
	}
	if got, ok := object["dup"].(int64); !ok || got != 2 {
		t.Fatalf("fast materialized dup = %v (%T), want last duplicate int64(2)", object["dup"], object["dup"])
	}
}

func TestFastMaterializerStringOwnershipAfterCloseAndGC(t *testing.T) {
	parser := mustNewParser(t)
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	doc, err := parser.Parse([]byte(`{"value":"survives"}`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	got, err := fastMaterializeElement(doc.Root())
	if err != nil {
		t.Fatalf("fastMaterializeElement() error = %v", err)
	}
	object, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("fast materialized root = %T, want map[string]any", got)
	}
	value, ok := object["value"].(string)
	if !ok || value != "survives" {
		t.Fatalf("fast materialized value = %v (%T), want %q", object["value"], object["value"], "survives")
	}
	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}
	runtime.GC()
	if value != "survives" {
		t.Fatalf("materialized string after Close+GC = %q, want %q", value, "survives")
	}
}

func TestFastMaterializerClosedDoc(t *testing.T) {
	_, doc := mustParseDoc(t, `{"value":1}`)
	root := doc.Root()
	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}

	if _, err := root.TypeErr(); !errors.Is(err, ErrClosed) {
		t.Fatalf("TypeErr() after Close error = %v, want ErrClosed", err)
	}
	if _, err := fastMaterializeElement(root); !errors.Is(err, ErrClosed) {
		t.Fatalf("fastMaterializeElement() after Close error = %v, want ErrClosed", err)
	}
}

func TestFastMaterializerSubtree(t *testing.T) {
	_, doc := mustParseDoc(t, `{"outer":{"inner":[1,"two"]},"skip":false}`)

	rootObject, err := doc.Root().AsObject()
	if err != nil {
		t.Fatalf("AsObject() error = %v", err)
	}
	outerField, err := rootObject.GetField("outer")
	if err != nil {
		t.Fatalf("GetField(\"outer\") error = %v", err)
	}

	baseline := materializeViaAccessorsForTest(t, outerField)
	object, ok := baseline.(map[string]any)
	if !ok {
		t.Fatalf("accessor baseline subtree = %T, want map[string]any", baseline)
	}
	inner, ok := object["inner"].([]any)
	if !ok {
		t.Fatalf("accessor baseline inner = %T, want []any", object["inner"])
	}
	if len(inner) != 2 {
		t.Fatalf("accessor baseline inner length = %d, want 2", len(inner))
	}
	if got, ok := inner[1].(string); !ok || got != "two" {
		t.Fatalf("accessor baseline inner[1] = %v (%T), want %q", inner[1], inner[1], "two")
	}

	got, err := fastMaterializeElement(outerField)
	if err != nil {
		t.Fatalf("fastMaterializeElement() error = %v", err)
	}
	if !reflect.DeepEqual(got, baseline) {
		t.Fatalf("fastMaterializeElement() = %#v, want %#v", got, baseline)
	}
}

func TestFastMaterializerConcurrentCloseGuard(t *testing.T) {
	_, doc := mustParseDoc(t, `{"value":1}`)
	root := doc.Root()

	doc.mu.Lock()
	if _, err := fastMaterializeElement(root); !errors.Is(err, ErrParserBusy) {
		t.Fatalf("fastMaterializeElement() with doc.mu held error = %v, want ErrParserBusy", err)
	}
	doc.mu.Unlock()

	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}
}

func TestFastMaterializerFramesFullyConsumed(t *testing.T) {
	t.Run("single scalar succeeds", func(t *testing.T) {
		got, err := buildAnyFromFrames([]ffi.InternalFrame{
			{Kind: uint32(ffi.ValueKindInt64), Int64Value: 7},
		})
		if err != nil {
			t.Fatalf("buildAnyFromFrames() error = %v", err)
		}
		if got != int64(7) {
			t.Fatalf("buildAnyFromFrames() = %v (%T), want int64(7)", got, got)
		}
	})

	t.Run("trailing frame fails", func(t *testing.T) {
		_, err := buildAnyFromFrames([]ffi.InternalFrame{
			{Kind: uint32(ffi.ValueKindInt64), Int64Value: 7},
			{Kind: uint32(ffi.ValueKindNull)},
		})
		if !errors.Is(err, ErrInternal) {
			t.Fatalf("buildAnyFromFrames() error = %v, want ErrInternal", err)
		}
	})
}

func materializeViaAccessorsForTest(t *testing.T, element Element) any {
	t.Helper()

	kind := ElementType(element.view.KindHint)
	if kind == TypeInvalid {
		resolvedKind, err := element.TypeErr()
		if err != nil {
			t.Fatalf("TypeErr() error = %v", err)
		}
		kind = resolvedKind
	}

	switch kind {
	case TypeNull:
		isNull, err := element.IsNullErr()
		if err != nil {
			t.Fatalf("IsNullErr() error = %v", err)
		}
		if !isNull {
			t.Fatal("IsNullErr() = false, want true")
		}
		return nil
	case TypeBool:
		value, err := element.GetBool()
		if err != nil {
			t.Fatalf("GetBool() error = %v", err)
		}
		return value
	case TypeInt64:
		value, err := element.GetInt64()
		if err != nil {
			t.Fatalf("GetInt64() error = %v", err)
		}
		return value
	case TypeUint64:
		value, err := element.GetUint64()
		if err != nil {
			t.Fatalf("GetUint64() error = %v", err)
		}
		return value
	case TypeFloat64:
		value, err := element.GetFloat64()
		if err != nil {
			t.Fatalf("GetFloat64() error = %v", err)
		}
		return value
	case TypeString:
		value, err := element.GetString()
		if err != nil {
			t.Fatalf("GetString() error = %v", err)
		}
		return value
	case TypeArray:
		array, err := element.AsArray()
		if err != nil {
			t.Fatalf("AsArray() error = %v", err)
		}
		values := make([]any, 0)
		iter := array.Iter()
		for iter.Next() {
			values = append(values, materializeViaAccessorsForTest(t, iter.Value()))
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("array iter.Err() = %v", err)
		}
		return values
	case TypeObject:
		object, err := element.AsObject()
		if err != nil {
			t.Fatalf("AsObject() error = %v", err)
		}
		values := make(map[string]any)
		iter := object.Iter()
		for iter.Next() {
			values[iter.Key()] = materializeViaAccessorsForTest(t, iter.Value())
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("object iter.Err() = %v", err)
		}
		return values
	default:
		t.Fatalf("unsupported ElementType %v", kind)
		return nil
	}
}
