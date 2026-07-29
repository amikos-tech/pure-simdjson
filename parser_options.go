package purejson

import "fmt"

const (
	defaultMaxCapacity uint64 = 0xFFFFFFFF
	maxSupportedDepth  uint32 = 1024
	defaultMaxDepth           = maxSupportedDepth
	minMaxCapacity            = 32
)

type parserConfig struct {
	maxCapacity uint64
	maxDepth    uint32
}

var defaultParserConfig = parserConfig{
	maxCapacity: defaultMaxCapacity,
	maxDepth:    defaultMaxDepth,
}

type parserOptionKind uint8

const (
	parserOptionInvalid parserOptionKind = iota
	parserOptionMaxCapacity
	parserOptionMaxDepth
)

// ParserOption configures an immutable parser capacity or depth bound.
//
// Options are created with WithMaxCapacity and WithMaxDepth. The zero value is
// invalid.
type ParserOption struct {
	kind  parserOptionKind
	value int
}

// WithMaxCapacity sets the maximum accepted input size in bytes. Zero selects
// the default 0xFFFFFFFF-byte limit.
func WithMaxCapacity(bytes int) ParserOption {
	return ParserOption{kind: parserOptionMaxCapacity, value: bytes}
}

// WithMaxDepth sets the upstream parser's maximum nesting depth. Zero selects
// the default depth of 1024; nonzero values must be between 1 and 1024.
func WithMaxDepth(depth int) ParserOption {
	return ParserOption{kind: parserOptionMaxDepth, value: depth}
}

func normalizeParserOptions(opts ...ParserOption) (parserConfig, error) {
	config := defaultParserConfig
	for _, option := range opts {
		switch option.kind {
		case parserOptionMaxCapacity:
			switch {
			case option.value == 0:
				config.maxCapacity = defaultMaxCapacity
			case option.value < minMaxCapacity:
				return parserConfig{}, fmt.Errorf(
					"%w: maximum capacity must be zero or between %d and %d bytes",
					ErrInvalidOption,
					minMaxCapacity,
					defaultMaxCapacity,
				)
			case uint64(option.value) > defaultMaxCapacity:
				return parserConfig{}, fmt.Errorf(
					"%w: maximum capacity %d exceeds %d bytes",
					ErrInvalidOption,
					option.value,
					defaultMaxCapacity,
				)
			default:
				config.maxCapacity = uint64(option.value)
			}
		case parserOptionMaxDepth:
			switch {
			case option.value == 0:
				config.maxDepth = defaultMaxDepth
			case option.value < 0:
				return parserConfig{}, fmt.Errorf(
					"%w: maximum depth must not be negative",
					ErrInvalidOption,
				)
			case uint64(option.value) > uint64(maxSupportedDepth):
				return parserConfig{}, fmt.Errorf(
					"%w: maximum depth %d exceeds the supported maximum of %d",
					ErrInvalidOption,
					option.value,
					maxSupportedDepth,
				)
			default:
				config.maxDepth = uint32(option.value)
			}
		default:
			return parserConfig{}, fmt.Errorf(
				"%w: unrecognized parser option kind %d",
				ErrInvalidOption,
				option.kind,
			)
		}
	}
	return config, nil
}
