package purejson

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"unsafe"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

var fastMaterializerFallbackWarningOnce sync.Once

func fastMaterializeElement(element Element) (any, error) {
	doc, err := element.usableDoc()
	if err != nil {
		return nil, err
	}
	if !doc.mu.TryLock() {
		return nil, ErrParserBusy
	}
	defer doc.mu.Unlock()

	if doc.closed {
		return nil, ErrClosed
	}

	bindings := doc.parser.library.bindings
	// KeepAlive must run after the final borrowed frame read. Registered
	// second on purpose: Go defers are LIFO, so this runs before the
	// doc.mu.Unlock() deferred above, keeping doc (and the C++-owned
	// materialize_frames span returned below) alive for the full
	// duration of buildAnyFromFrames, including all nested string
	// copies. Do not reorder with the Unlock defer.
	defer runtime.KeepAlive(doc)

	if !bindings.HasInternalMaterializeBuild() {
		emitFastMaterializerFallbackWarning()
		return materializeElementViaAccessors(element)
	}

	frames, rc := bindings.InternalMaterializeBuild(&element.view)
	if err := wrapStatus(rc); err != nil {
		return nil, err
	}

	return buildAnyFromFrames(frames)
}

func materializeElementViaAccessors(element Element) (any, error) {
	kind := ElementType(element.view.KindHint)
	if kind == TypeInvalid {
		resolvedKind, err := element.TypeErr()
		if err != nil {
			return nil, err
		}
		kind = resolvedKind
	}

	switch kind {
	case TypeNull:
		isNull, err := element.IsNullErr()
		if err != nil {
			return nil, err
		}
		if !isNull {
			return nil, ErrInternal
		}
		return nil, nil
	case TypeBool:
		return element.GetBool()
	case TypeInt64:
		return element.GetInt64()
	case TypeUint64:
		return element.GetUint64()
	case TypeFloat64:
		return element.GetFloat64()
	case TypeString:
		return element.GetString()
	case TypeArray:
		array, err := element.AsArray()
		if err != nil {
			return nil, err
		}
		iter := array.Iter()
		values := make([]any, 0)
		for iter.Next() {
			value, err := materializeElementViaAccessors(iter.Value())
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		if err := iter.Err(); err != nil {
			return nil, err
		}
		return values, nil
	case TypeObject:
		object, err := element.AsObject()
		if err != nil {
			return nil, err
		}
		iter := object.Iter()
		values := make(map[string]any)
		for iter.Next() {
			value, err := materializeElementViaAccessors(iter.Value())
			if err != nil {
				return nil, err
			}
			values[iter.Key()] = value
		}
		if err := iter.Err(); err != nil {
			return nil, err
		}
		return values, nil
	default:
		return nil, ErrInternal
	}
}

func buildAnyFromFrames(frames []ffi.InternalFrame) (any, error) {
	if len(frames) == 0 {
		return nil, frameProtocolError(0, 0, "empty frame span")
	}

	value, consumed, err := buildAnyFromFrame(frames, 0)
	if err != nil {
		return nil, err
	}
	if consumed != len(frames) {
		return nil, frameProtocolError(consumed, 0, fmt.Sprintf("trailing frames: consumed=%d len=%d", consumed, len(frames)))
	}
	return value, nil
}

func buildAnyFromFrame(frames []ffi.InternalFrame, index int) (any, int, error) {
	if index >= len(frames) {
		return nil, index, frameProtocolError(index, 0, "frame index out of span")
	}

	frame := frames[index]
	switch ffi.ValueKind(frame.Kind) {
	case ffi.ValueKindNull:
		return nil, index + 1, nil
	case ffi.ValueKindBool:
		return frame.Flags != 0, index + 1, nil
	case ffi.ValueKindInt64:
		return frame.Int64Value, index + 1, nil
	case ffi.ValueKindUint64:
		return frame.Uint64Value, index + 1, nil
	case ffi.ValueKindFloat64:
		return frame.Float64Value, index + 1, nil
	case ffi.ValueKindString:
		value, err := copyFrameString(frame.StringPtr, frame.StringLen, index, frame.Kind, "string")
		if err != nil {
			return nil, index, err
		}
		return value, index + 1, nil
	case ffi.ValueKindArray:
		values := make([]any, 0, int(frame.ChildCount))
		nextIndex := index + 1
		for child := uint32(0); child < frame.ChildCount; child++ {
			if nextIndex >= len(frames) {
				return nil, nextIndex, frameProtocolError(index, frame.Kind, fmt.Sprintf("array child_count exceeds frame span: child=%d child_count=%d len=%d", child, frame.ChildCount, len(frames)))
			}
			value, consumed, err := buildAnyFromFrame(frames, nextIndex)
			if err != nil {
				return nil, consumed, err
			}
			values = append(values, value)
			nextIndex = consumed
		}
		return values, nextIndex, nil
	case ffi.ValueKindObject:
		values := make(map[string]any, int(frame.ChildCount))
		nextIndex := index + 1
		for child := uint32(0); child < frame.ChildCount; child++ {
			if nextIndex >= len(frames) {
				return nil, nextIndex, frameProtocolError(index, frame.Kind, fmt.Sprintf("object child_count exceeds frame span: child=%d child_count=%d len=%d", child, frame.ChildCount, len(frames)))
			}
			key, err := copyFrameString(frames[nextIndex].KeyPtr, frames[nextIndex].KeyLen, nextIndex, frames[nextIndex].Kind, "key")
			if err != nil {
				return nil, nextIndex, err
			}
			value, consumed, err := buildAnyFromFrame(frames, nextIndex)
			if err != nil {
				return nil, consumed, err
			}
			values[key] = value
			nextIndex = consumed
		}
		return values, nextIndex, nil
	default:
		return nil, index, frameProtocolError(index, frame.Kind, "unknown frame kind")
	}
}

func copyFrameString(ptr uintptr, length uintptr, index int, kind uint32, label string) (string, error) {
	if length == 0 {
		return "", nil
	}
	if ptr == 0 {
		return "", frameProtocolError(index, kind, label+" span has nil pointer")
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))), nil
}

func frameProtocolError(index int, kind uint32, detail string) error {
	return newError(int32(ffi.ErrInternal), nativeDetails{
		message: fmt.Sprintf("materialize frame protocol violation index=%d kind=%d: %s", index, kind, detail),
	}, ErrInternal)
}

func emitFastMaterializerFallbackWarning() {
	if os.Getenv("PURE_SIMDJSON_DEBUG") != "1" {
		return
	}
	fastMaterializerFallbackWarningOnce.Do(func() {
		fmt.Fprintln(os.Stderr, "purejson debug: psdj_internal_materialize_build unavailable; using accessor materializer")
	})
}
