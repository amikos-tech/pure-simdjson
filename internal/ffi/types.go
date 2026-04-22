package ffi

const (
	// ABIVersion encodes the expected native ABI as 0xMMMMmmmm (16-bit major,
	// 16-bit minor). It must match PURE_SIMDJSON_ABI_VERSION exported by the
	// Rust shim; bumping major signals a breaking C-ABI change.
	ABIVersion             uint32 = 0x00010000
	LastErrorOffsetUnknown uint64 = ^uint64(0)
)

type ErrorCode int32

const (
	OK                 ErrorCode = 0
	ErrInvalidArg      ErrorCode = 1
	ErrInvalidHandle   ErrorCode = 2
	ErrParserBusy      ErrorCode = 3
	ErrWrongType       ErrorCode = 4
	ErrElementNotFound ErrorCode = 5
	ErrBufferTooSmall  ErrorCode = 6

	ErrInvalidJSON      ErrorCode = 32
	ErrNumberOutOfRange ErrorCode = 33
	ErrPrecisionLoss    ErrorCode = 34

	ErrCPUUnsupported ErrorCode = 64
	ErrABIMismatch    ErrorCode = 65

	ErrPanic        ErrorCode = 96
	ErrCPPException ErrorCode = 97
	ErrInternal     ErrorCode = 127
)

type ValueKind uint32

const (
	ValueKindInvalid ValueKind = 0
	ValueKindNull    ValueKind = 1
	ValueKindBool    ValueKind = 2
	ValueKindInt64   ValueKind = 3
	ValueKindUint64  ValueKind = 4
	ValueKindFloat64 ValueKind = 5
	ValueKindString  ValueKind = 6
	ValueKindArray   ValueKind = 7
	ValueKindObject  ValueKind = 8
)

type ParserHandle uint64
type DocHandle uint64

type ValueView struct {
	Doc      DocHandle
	State0   uint64
	State1   uint64
	KindHint uint32
	Reserved uint32
}

type ArrayIter struct {
	Doc      DocHandle
	State0   uint64
	State1   uint64
	Index    uint32
	Tag      uint16
	Reserved uint16
}

type ObjectIter struct {
	Doc      DocHandle
	State0   uint64
	State1   uint64
	Index    uint32
	Tag      uint16
	Reserved uint16
}

type NativeAllocStats struct {
	LiveBytes       uint64
	TotalAllocBytes uint64
	AllocCount      uint64
	FreeCount       uint64
}
