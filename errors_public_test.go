package purejson_test

import (
	"errors"
	"reflect"
	"testing"

	purejson "github.com/amikos-tech/pure-simdjson"
	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

func TestErrorHasNoExportedFields(t *testing.T) {
	t.Helper()

	errorType := reflect.TypeOf(purejson.Error{})
	for i := 0; i < errorType.NumField(); i++ {
		field := errorType.Field(i)
		if field.IsExported() {
			t.Fatalf("purejson.Error field %q is exported; want accessor-only API", field.Name)
		}
	}
}

func TestErrorAccessorsExposeNativeStatus(t *testing.T) {
	parser, err := purejson.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	t.Cleanup(func() {
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() cleanup error = %v", err)
		}
	})

	_, err = parser.Parse([]byte("{"))
	if !errors.Is(err, purejson.ErrInvalidJSON) {
		t.Fatalf("Parse() error = %v, want ErrInvalidJSON", err)
	}

	var nativeErr *purejson.Error
	if !errors.As(err, &nativeErr) {
		t.Fatalf("Parse() error = %v, want *purejson.Error", err)
	}
	if nativeErr.Code() != int32(ffi.ErrInvalidJSON) {
		t.Fatalf("native error code = %d, want %d", nativeErr.Code(), ffi.ErrInvalidJSON)
	}
	if nativeErr.Message() == "" {
		t.Fatal("native error message is empty")
	}
}
