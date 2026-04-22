package purejson

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	gojson "github.com/goccy/go-json"
)

const (
	benchmarkFixtureTwitter = "twitter.json"
	benchmarkFixtureCITM    = "citm_catalog.json"
	benchmarkFixtureCanada  = "canada.json"

	benchmarkComparatorPureSimdjson   = "pure-simdjson"
	benchmarkComparatorEncodingAny    = "encoding-json-any"
	benchmarkComparatorEncodingStruct = "encoding-json-struct"
	benchmarkComparatorMinioSimdjson  = "minio-simdjson-go"
	benchmarkComparatorBytedanceSonic = "bytedance-sonic"
	benchmarkComparatorGoccyGoJSON    = "goccy-go-json"
)

var benchmarkCanonicalComparatorKeys = []string{
	benchmarkComparatorPureSimdjson,
	benchmarkComparatorEncodingAny,
	benchmarkComparatorEncodingStruct,
	benchmarkComparatorMinioSimdjson,
	benchmarkComparatorBytedanceSonic,
	benchmarkComparatorGoccyGoJSON,
}

type benchmarkComparator struct {
	key            string
	materialize    func(fixtureName string, data []byte) (any, error)
	omissionReason string
}

func (c benchmarkComparator) available() bool {
	return c.materialize != nil && c.omissionReason == ""
}

type benchmarkComparatorRegistry struct {
	mu          sync.RWMutex
	comparators map[string]benchmarkComparator
}

var benchmarkComparators = &benchmarkComparatorRegistry{
	comparators: make(map[string]benchmarkComparator, len(benchmarkCanonicalComparatorKeys)),
}

func init() {
	registerBenchmarkComparator(benchmarkComparator{
		key:         benchmarkComparatorPureSimdjson,
		materialize: benchmarkMaterializePureSimdjson,
	})
	registerBenchmarkComparator(benchmarkComparator{
		key:         benchmarkComparatorEncodingAny,
		materialize: benchmarkMaterializeEncodingJSONAny,
	})
	registerBenchmarkComparator(benchmarkComparator{
		key:         benchmarkComparatorEncodingStruct,
		materialize: benchmarkMaterializeEncodingJSONStruct,
	})
	registerBenchmarkComparator(benchmarkComparator{
		key:         benchmarkComparatorGoccyGoJSON,
		materialize: benchmarkMaterializeGoccyGoJSON,
	})
}

func registerBenchmarkComparator(comparator benchmarkComparator) {
	if comparator.key == "" {
		panic("benchmark comparator key must not be empty")
	}
	if comparator.materialize == nil {
		panic(fmt.Sprintf("benchmark comparator %q must provide materialize", comparator.key))
	}
	if comparator.omissionReason != "" {
		panic(fmt.Sprintf("benchmark comparator %q cannot be both available and omitted", comparator.key))
	}

	benchmarkComparators.mu.Lock()
	defer benchmarkComparators.mu.Unlock()
	benchmarkComparators.comparators[comparator.key] = comparator
}

func registerOmittedBenchmarkComparator(key, reason string) {
	if key == "" {
		panic("benchmark comparator key must not be empty")
	}
	if reason == "" {
		panic(fmt.Sprintf("benchmark comparator %q omission reason must not be empty", key))
	}

	benchmarkComparators.mu.Lock()
	defer benchmarkComparators.mu.Unlock()
	benchmarkComparators.comparators[key] = benchmarkComparator{
		key:            key,
		omissionReason: reason,
	}
}

func allBenchmarkComparators(tb testing.TB) []benchmarkComparator {
	tb.Helper()

	benchmarkComparators.mu.RLock()
	defer benchmarkComparators.mu.RUnlock()

	comparators := make([]benchmarkComparator, 0, len(benchmarkCanonicalComparatorKeys))
	for _, key := range benchmarkCanonicalComparatorKeys {
		comparator, ok := benchmarkComparators.comparators[key]
		if !ok {
			tb.Fatalf("benchmark comparator %q is not registered", key)
		}
		comparators = append(comparators, comparator)
	}

	return comparators
}

func availableBenchmarkComparators(tb testing.TB) []benchmarkComparator {
	tb.Helper()

	all := allBenchmarkComparators(tb)
	available := make([]benchmarkComparator, 0, len(all))
	for _, comparator := range all {
		if comparator.available() {
			available = append(available, comparator)
		}
	}
	if len(available) == 0 {
		tb.Fatal("no benchmark comparators are available")
	}

	return available
}

func benchmarkOmittedComparators(tb testing.TB) map[string]string {
	tb.Helper()

	all := allBenchmarkComparators(tb)
	omitted := make(map[string]string)
	for _, comparator := range all {
		if comparator.omissionReason != "" {
			omitted[comparator.key] = comparator.omissionReason
		}
	}

	return omitted
}

type benchmarkUnmarshalFunc func([]byte, any) error

func benchmarkMaterializeEncodingJSONAny(_ string, data []byte) (any, error) {
	return benchmarkDecodeAny(json.Unmarshal, data)
}

func benchmarkMaterializeEncodingJSONStruct(fixtureName string, data []byte) (any, error) {
	return benchmarkDecodeSharedSchema(json.Unmarshal, fixtureName, data)
}

func benchmarkMaterializeGoccyGoJSON(_ string, data []byte) (any, error) {
	return benchmarkDecodeAny(gojson.Unmarshal, data)
}

func benchmarkDecodeAny(unmarshal benchmarkUnmarshalFunc, data []byte) (any, error) {
	var value any
	if err := unmarshal(data, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func benchmarkDecodeSharedSchema(unmarshal benchmarkUnmarshalFunc, fixtureName string, data []byte) (any, error) {
	target, err := benchmarkNewSchemaTarget(fixtureName)
	if err != nil {
		return nil, err
	}
	if err := unmarshal(data, target); err != nil {
		return nil, err
	}
	return benchmarkDerefSchemaTarget(target)
}

func benchmarkNewSchemaTarget(fixtureName string) (any, error) {
	switch fixtureName {
	case benchmarkFixtureTwitter:
		return &benchTwitterRow{}, nil
	case benchmarkFixtureCITM:
		return &benchCITMRow{}, nil
	case benchmarkFixtureCanada:
		return &benchCanadaRow{}, nil
	default:
		return nil, fmt.Errorf("no shared benchmark schema for fixture %q", fixtureName)
	}
}

func benchmarkDerefSchemaTarget(target any) (any, error) {
	switch value := target.(type) {
	case *benchTwitterRow:
		return *value, nil
	case *benchCITMRow:
		return *value, nil
	case *benchCanadaRow:
		return *value, nil
	default:
		return nil, fmt.Errorf("unsupported benchmark schema target %T", target)
	}
}

func benchmarkMaterializePureSimdjson(_ string, data []byte) (result any, err error) {
	parser, err := NewParser()
	if err != nil {
		return nil, err
	}

	var doc *Doc
	defer func() {
		err = benchmarkCloseMaterializeResources(err, doc, parser)
	}()

	doc, err = parser.Parse(data)
	if err != nil {
		return nil, err
	}

	result, err = benchmarkMaterializePureElement(doc.Root())
	return result, err
}

func benchmarkCloseMaterializeResources(currentErr error, doc *Doc, parser *Parser) error {
	if doc != nil {
		if err := doc.Close(); err != nil {
			currentErr = errors.Join(currentErr, err)
		}
	}
	if parser != nil {
		if err := parser.Close(); err != nil {
			currentErr = errors.Join(currentErr, err)
		}
	}
	return currentErr
}

func benchmarkMaterializePureElement(element Element) (any, error) {
	switch element.Type() {
	case TypeNull:
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

		values := make([]any, 0)
		iter := array.Iter()
		for iter.Next() {
			value, err := benchmarkMaterializePureElement(iter.Value())
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

		values := make(map[string]any)
		iter := object.Iter()
		for iter.Next() {
			value, err := benchmarkMaterializePureElement(iter.Value())
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
		return nil, fmt.Errorf("unsupported pure-simdjson element type %v", element.Type())
	}
}
