package ffi

import (
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

type Bindings struct {
	handle uintptr

	getABIVersion            func(*uint32) int32
	getImplementationNameLen func(*uintptr) int32
	copyImplementationName   func(*byte, uintptr, *uintptr) int32
	nativeAllocStatsReset    func() int32
	nativeAllocStatsSnapshot func(*NativeAllocStats) int32

	parserNew                func(*ParserHandle) int32
	parserFree               func(ParserHandle) int32
	parserParse              func(ParserHandle, *byte, uintptr, *DocHandle) int32
	parserGetLastErrorLen    func(ParserHandle, *uintptr) int32
	parserCopyLastError      func(ParserHandle, *byte, uintptr, *uintptr) int32
	parserGetLastErrorOffset func(ParserHandle, *uint64) int32

	docFree           func(DocHandle) int32
	docRoot           func(DocHandle, *ValueView) int32
	elementType       func(*ValueView, *uint32) int32
	elementGetInt64   func(*ValueView, *int64) int32
	elementGetUint64  func(*ValueView, *uint64) int32
	elementGetFloat64 func(*ValueView, *float64) int32
	elementGetString  func(*ValueView, **byte, *uintptr) int32
	bytesFree         func(*byte, uintptr) int32
	elementGetBool    func(*ValueView, *byte) int32
	elementIsNull     func(*ValueView, *byte) int32
	arrayIterNew      func(*ValueView, *ArrayIter) int32
	arrayIterNext     func(*ArrayIter, *ValueView, *byte) int32
	objectIterNew     func(*ValueView, *ObjectIter) int32
	objectIterNext    func(*ObjectIter, *ValueView, *ValueView, *byte) int32
	objectGetField    func(*ValueView, *byte, uintptr, *ValueView) int32
}

type SymbolLookup func(handle uintptr, name string) (uintptr, error)

func Bind(handle uintptr, lookup SymbolLookup) (*Bindings, error) {
	b := &Bindings{handle: handle}

	symbols := []struct {
		name   string
		target any
	}{
		{name: "pure_simdjson_get_abi_version", target: &b.getABIVersion},
		{name: "pure_simdjson_get_implementation_name_len", target: &b.getImplementationNameLen},
		{name: "pure_simdjson_copy_implementation_name", target: &b.copyImplementationName},
		{name: "pure_simdjson_native_alloc_stats_reset", target: &b.nativeAllocStatsReset},
		{name: "pure_simdjson_native_alloc_stats_snapshot", target: &b.nativeAllocStatsSnapshot},
		{name: "pure_simdjson_parser_new", target: &b.parserNew},
		{name: "pure_simdjson_parser_free", target: &b.parserFree},
		{name: "pure_simdjson_parser_parse", target: &b.parserParse},
		{name: "pure_simdjson_parser_get_last_error_len", target: &b.parserGetLastErrorLen},
		{name: "pure_simdjson_parser_copy_last_error", target: &b.parserCopyLastError},
		{name: "pure_simdjson_parser_get_last_error_offset", target: &b.parserGetLastErrorOffset},
		{name: "pure_simdjson_doc_free", target: &b.docFree},
		{name: "pure_simdjson_doc_root", target: &b.docRoot},
		{name: "pure_simdjson_element_type", target: &b.elementType},
		{name: "pure_simdjson_element_get_int64", target: &b.elementGetInt64},
		{name: "pure_simdjson_element_get_uint64", target: &b.elementGetUint64},
		{name: "pure_simdjson_element_get_float64", target: &b.elementGetFloat64},
		{name: "pure_simdjson_element_get_string", target: &b.elementGetString},
		{name: "pure_simdjson_bytes_free", target: &b.bytesFree},
		{name: "pure_simdjson_element_get_bool", target: &b.elementGetBool},
		{name: "pure_simdjson_element_is_null", target: &b.elementIsNull},
		{name: "pure_simdjson_array_iter_new", target: &b.arrayIterNew},
		{name: "pure_simdjson_array_iter_next", target: &b.arrayIterNext},
		{name: "pure_simdjson_object_iter_new", target: &b.objectIterNew},
		{name: "pure_simdjson_object_iter_next", target: &b.objectIterNext},
		{name: "pure_simdjson_object_get_field", target: &b.objectGetField},
	}

	for _, symbol := range symbols {
		if err := registerFunc(handle, lookup, symbol.name, symbol.target); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func registerFunc(handle uintptr, lookup SymbolLookup, name string, target any) (err error) {
	sym, err := lookup(handle, name)
	if err != nil {
		return fmt.Errorf("lookup %s: %w", name, err)
	}

	defer func() {
		if panicVal := recover(); panicVal != nil {
			err = fmt.Errorf("register %s: %v", name, panicVal)
		}
	}()

	purego.RegisterFunc(target, sym)
	return nil
}

func (b *Bindings) ABI() (uint32, int32) {
	var abi uint32
	rc := b.getABIVersion(&abi)
	runtime.KeepAlive(b)
	return abi, rc
}

func (b *Bindings) ImplementationName() (string, int32) {
	var length uintptr
	rc := b.getImplementationNameLen(&length)
	if rc != int32(OK) {
		runtime.KeepAlive(b)
		return "", rc
	}
	if length == 0 {
		runtime.KeepAlive(b)
		return "", int32(OK)
	}

	buffer := make([]byte, length)
	var written uintptr
	rc = b.copyImplementationName(unsafe.SliceData(buffer), uintptr(len(buffer)), &written)
	runtime.KeepAlive(buffer)
	runtime.KeepAlive(b)
	if rc != int32(OK) {
		return "", rc
	}

	if written > uintptr(len(buffer)) {
		written = uintptr(len(buffer))
	}
	return string(buffer[:written]), int32(OK)
}

func (b *Bindings) NativeAllocStatsReset() int32 {
	rc := b.nativeAllocStatsReset()
	runtime.KeepAlive(b)
	return rc
}

func (b *Bindings) NativeAllocStatsSnapshot() (NativeAllocStats, int32) {
	var stats NativeAllocStats
	rc := b.nativeAllocStatsSnapshot(&stats)
	runtime.KeepAlive(b)
	return stats, rc
}

func (b *Bindings) ParserNew() (ParserHandle, int32) {
	var parser ParserHandle
	rc := b.parserNew(&parser)
	runtime.KeepAlive(b)
	return parser, rc
}

func (b *Bindings) ParserFree(parser ParserHandle) int32 {
	rc := b.parserFree(parser)
	runtime.KeepAlive(b)
	return rc
}

func (b *Bindings) ParserParse(parser ParserHandle, data []byte) (DocHandle, int32) {
	var inputPtr *byte
	if len(data) > 0 {
		inputPtr = unsafe.SliceData(data)
	}

	var doc DocHandle
	rc := b.parserParse(parser, inputPtr, uintptr(len(data)), &doc)
	runtime.KeepAlive(data)
	runtime.KeepAlive(b)
	return doc, rc
}

func (b *Bindings) ParserLastError(parser ParserHandle) (string, int32) {
	var length uintptr
	rc := b.parserGetLastErrorLen(parser, &length)
	if rc != int32(OK) {
		runtime.KeepAlive(b)
		return "", rc
	}
	if length == 0 {
		runtime.KeepAlive(b)
		return "", int32(OK)
	}

	buffer := make([]byte, length)
	var written uintptr
	rc = b.parserCopyLastError(parser, unsafe.SliceData(buffer), uintptr(len(buffer)), &written)
	runtime.KeepAlive(buffer)
	runtime.KeepAlive(b)
	if rc != int32(OK) {
		return "", rc
	}

	if written > uintptr(len(buffer)) {
		written = uintptr(len(buffer))
	}
	return string(buffer[:written]), int32(OK)
}

func (b *Bindings) ParserLastErrorOffset(parser ParserHandle) (uint64, int32) {
	var offset uint64
	rc := b.parserGetLastErrorOffset(parser, &offset)
	runtime.KeepAlive(b)
	return offset, rc
}

func (b *Bindings) DocFree(doc DocHandle) int32 {
	rc := b.docFree(doc)
	runtime.KeepAlive(b)
	return rc
}

func (b *Bindings) DocRoot(doc DocHandle) (ValueView, int32) {
	var view ValueView
	rc := b.docRoot(doc, &view)
	runtime.KeepAlive(b)
	return view, rc
}

func (b *Bindings) ElementType(view *ValueView) (uint32, int32) {
	var kind uint32
	rc := b.elementType(view, &kind)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	return kind, rc
}

func (b *Bindings) ElementGetInt64(view *ValueView) (int64, int32) {
	var value int64
	rc := b.elementGetInt64(view, &value)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	return value, rc
}

func (b *Bindings) ElementGetUint64(view *ValueView) (uint64, int32) {
	var value uint64
	rc := b.elementGetUint64(view, &value)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	return value, rc
}

func (b *Bindings) ElementGetFloat64(view *ValueView) (float64, int32) {
	var value float64
	rc := b.elementGetFloat64(view, &value)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	return value, rc
}

func (b *Bindings) ElementGetString(view *ValueView) (string, int32) {
	var ptr *byte
	var length uintptr
	rc := b.elementGetString(view, &ptr, &length)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	if rc != int32(OK) {
		return "", rc
	}

	defer func() {
		if ptr == nil {
			return
		}
		if freeRC := b.BytesFree(ptr, length); freeRC != int32(OK) && leakWarningsEnabled() {
			fmt.Fprintf(os.Stderr, "purejson leak: bytes_free rc=%d len=%d\n", freeRC, length)
		}
	}()

	if ptr == nil && length == 0 {
		return "", int32(OK)
	}

	return string(unsafe.Slice(ptr, length)), int32(OK)
}

func (b *Bindings) BytesFree(ptr *byte, length uintptr) int32 {
	rc := b.bytesFree(ptr, length)
	runtime.KeepAlive(b)
	return rc
}

func (b *Bindings) ElementGetBool(view *ValueView) (bool, int32) {
	var value byte
	rc := b.elementGetBool(view, &value)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	return value != 0, rc
}

func (b *Bindings) ElementIsNull(view *ValueView) (bool, int32) {
	var value byte
	rc := b.elementIsNull(view, &value)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	return value != 0, rc
}

func (b *Bindings) ArrayIterNew(view *ValueView) (ArrayIter, int32) {
	var iter ArrayIter
	rc := b.arrayIterNew(view, &iter)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	return iter, rc
}

func (b *Bindings) ArrayIterNext(iter *ArrayIter) (ValueView, bool, int32) {
	var value ValueView
	var done byte
	rc := b.arrayIterNext(iter, &value, &done)
	runtime.KeepAlive(iter)
	runtime.KeepAlive(b)
	return value, done != 0, rc
}

func (b *Bindings) ObjectIterNew(view *ValueView) (ObjectIter, int32) {
	var iter ObjectIter
	rc := b.objectIterNew(view, &iter)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	return iter, rc
}

func (b *Bindings) ObjectIterNext(iter *ObjectIter) (ValueView, ValueView, bool, int32) {
	var key ValueView
	var value ValueView
	var done byte
	rc := b.objectIterNext(iter, &key, &value, &done)
	runtime.KeepAlive(iter)
	runtime.KeepAlive(b)
	return key, value, done != 0, rc
}

func (b *Bindings) ObjectGetField(view *ValueView, key string) (ValueView, int32) {
	var keyBytes []byte
	if key != "" {
		keyBytes = []byte(key)
	}
	var keyPtr *byte
	if len(keyBytes) != 0 {
		keyPtr = unsafe.SliceData(keyBytes)
	}

	var value ValueView
	rc := b.objectGetField(view, keyPtr, uintptr(len(keyBytes)), &value)
	runtime.KeepAlive(keyBytes)
	runtime.KeepAlive(view)
	runtime.KeepAlive(b)
	return value, rc
}
