package purejson

import (
	"runtime"
	"unsafe"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

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

	frames, rc := doc.parser.library.bindings.InternalMaterializeBuild(&element.view)
	if err := wrapStatus(rc); err != nil {
		return nil, err
	}
	// KeepAlive must run after the final borrowed frame read.
	defer runtime.KeepAlive(doc)

	return buildAnyFromFrames(frames)
}

func buildAnyFromFrames(frames []ffi.InternalFrame) (any, error) {
	if len(frames) == 0 {
		return nil, ErrInternal
	}

	value, consumed, err := buildAnyFromFrame(frames, 0)
	if err != nil {
		return nil, err
	}
	if consumed != len(frames) {
		return nil, ErrInternal
	}
	return value, nil
}

func buildAnyFromFrame(frames []ffi.InternalFrame, index int) (any, int, error) {
	if index >= len(frames) {
		return nil, index, ErrInternal
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
		value, err := copyFrameString(frame.StringPtr, frame.StringLen)
		if err != nil {
			return nil, index, err
		}
		return value, index + 1, nil
	case ffi.ValueKindArray:
		values := make([]any, 0, int(frame.ChildCount))
		nextIndex := index + 1
		for child := uint32(0); child < frame.ChildCount; child++ {
			if nextIndex >= len(frames) {
				return nil, nextIndex, ErrInternal
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
				return nil, nextIndex, ErrInternal
			}
			key, err := copyFrameString(frames[nextIndex].KeyPtr, frames[nextIndex].KeyLen)
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
		return nil, index, ErrInternal
	}
}

func copyFrameString(ptr uintptr, length uintptr) (string, error) {
	if length == 0 {
		return "", nil
	}
	if ptr == 0 {
		return "", ErrInternal
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(length))), nil
}
