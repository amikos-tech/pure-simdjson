package purejson

import (
	"bytes"
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
)

const benchmarkComparatorPureSimdjson = "pure-simdjson"

const (
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

type benchmarkJSONShape struct {
	objects       int
	objectFields  int
	arrays        int
	arrayElements int
	strings       int
	numbers       int
	bools         int
	nulls         int
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
	if existing, ok := benchmarkComparators.comparators[comparator.key]; ok {
		panic(fmt.Sprintf("benchmark comparator %q registered twice (existing omitted=%t)", comparator.key, existing.omissionReason != ""))
	}
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
	if existing, ok := benchmarkComparators.comparators[key]; ok {
		panic(fmt.Sprintf("benchmark comparator %q registered twice (existing omitted=%t)", key, existing.omissionReason != ""))
	}
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

func TestTierNComparatorsAgree(t *testing.T) {
	t.Run("Tier1FullParseShape", func(t *testing.T) {
		for _, fixtureName := range []string{benchmarkFixtureTwitter, benchmarkFixtureCITM, benchmarkFixtureCanada} {
			data := loadBenchmarkFixture(t, fixtureName)
			want, err := benchmarkJSONShapeFromBytes(data)
			if err != nil {
				t.Fatalf("reference shape(%s): %v", fixtureName, err)
			}

			for _, comparator := range availableBenchmarkComparators(t) {
				if comparator.key == benchmarkComparatorEncodingStruct {
					continue
				}
				result, err := comparator.materialize(fixtureName, data)
				if err != nil {
					t.Fatalf("%s materialize(%s): %v", comparator.key, fixtureName, err)
				}
				got, err := benchmarkJSONShapeFromMaterialized(result)
				if err != nil {
					t.Fatalf("%s shape(%s): %v", comparator.key, fixtureName, err)
				}
				if got != want {
					t.Fatalf("%s shape(%s) = %+v, want %+v", comparator.key, fixtureName, got, want)
				}
			}
		}
	})

	t.Run("Tier2TypedExtraction", func(t *testing.T) {
		for _, fixtureName := range []string{benchmarkFixtureTwitter, benchmarkFixtureCITM, benchmarkFixtureCanada} {
			data := loadBenchmarkFixture(t, fixtureName)
			want, err := benchmarkTier2TypedExtract(benchmarkComparatorPureSimdjson, fixtureName, data)
			if err != nil {
				t.Fatalf("%s typed extract(%s): %v", benchmarkComparatorPureSimdjson, fixtureName, err)
			}

			for _, comparator := range availableTier2TypedComparators(t) {
				got, err := benchmarkTier2TypedExtract(comparator.key, fixtureName, data)
				if err != nil {
					t.Fatalf("%s typed extract(%s): %v", comparator.key, fixtureName, err)
				}
				if got != want {
					t.Fatalf("%s typed extract(%s) = %+v, want %+v", comparator.key, fixtureName, got, want)
				}
			}
		}
	})

	t.Run("Tier3SelectiveExtraction", func(t *testing.T) {
		for _, fixtureName := range []string{benchmarkFixtureTwitter, benchmarkFixtureCITM} {
			data := loadBenchmarkFixture(t, fixtureName)
			want, err := benchmarkTier3SelectivePlaceholderExtract(benchmarkComparatorPureSimdjson, fixtureName, data)
			if err != nil {
				t.Fatalf("%s selective extract(%s): %v", benchmarkComparatorPureSimdjson, fixtureName, err)
			}

			for _, comparator := range availableTier2TypedComparators(t) {
				got, err := benchmarkTier3SelectivePlaceholderExtract(comparator.key, fixtureName, data)
				if err != nil {
					t.Fatalf("%s selective extract(%s): %v", comparator.key, fixtureName, err)
				}
				if got != want {
					t.Fatalf("%s selective extract(%s) = %+v, want %+v", comparator.key, fixtureName, got, want)
				}
			}
		}
	})
}

func benchmarkJSONShapeFromBytes(data []byte) (benchmarkJSONShape, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return benchmarkJSONShape{}, err
	}
	return benchmarkJSONShapeOf(value), nil
}

func benchmarkJSONShapeFromMaterialized(value any) (benchmarkJSONShape, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return benchmarkJSONShape{}, err
	}
	return benchmarkJSONShapeFromBytes(data)
}

func benchmarkJSONShapeOf(value any) benchmarkJSONShape {
	switch typed := value.(type) {
	case nil:
		return benchmarkJSONShape{nulls: 1}
	case bool:
		return benchmarkJSONShape{bools: 1}
	case string:
		return benchmarkJSONShape{strings: 1}
	case json.Number, float64:
		return benchmarkJSONShape{numbers: 1}
	case []any:
		shape := benchmarkJSONShape{arrays: 1, arrayElements: len(typed)}
		for _, item := range typed {
			shape.add(benchmarkJSONShapeOf(item))
		}
		return shape
	case map[string]any:
		shape := benchmarkJSONShape{objects: 1, objectFields: len(typed)}
		for _, item := range typed {
			shape.add(benchmarkJSONShapeOf(item))
		}
		return shape
	default:
		return benchmarkJSONShape{numbers: 1}
	}
}

func (s *benchmarkJSONShape) add(other benchmarkJSONShape) {
	s.objects += other.objects
	s.objectFields += other.objectFields
	s.arrays += other.arrays
	s.arrayElements += other.arrayElements
	s.strings += other.strings
	s.numbers += other.numbers
	s.bools += other.bools
	s.nulls += other.nulls
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
	defer func() {
		err = benchmarkCloseMaterializeResources(err, nil, parser)
	}()

	return benchmarkMaterializePureSimdjsonWithParser(parser, data)
}

func benchmarkMaterializePureSimdjsonWithParser(parser *Parser, data []byte) (result any, err error) {
	var doc *Doc
	defer func() {
		err = benchmarkCloseMaterializeResources(err, doc, nil)
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

func benchmarkWarmPureParser(tb testing.TB, fixtureName string, data []byte) *Parser {
	tb.Helper()

	parser, err := NewParser()
	if err != nil {
		tb.Fatalf("NewParser(%s): %v", fixtureName, err)
	}

	doc, err := parser.Parse(data)
	if err != nil {
		_ = parser.Close()
		tb.Fatalf("warm Parse(%s): %v", fixtureName, err)
	}
	if err := doc.Close(); err != nil {
		_ = parser.Close()
		tb.Fatalf("warm doc.Close(%s): %v", fixtureName, err)
	}

	return parser
}

func benchmarkMaterializePureElement(element Element) (any, error) {
	return fastMaterializeElement(element)
}
