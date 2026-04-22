package purejson

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"testing"

	gojson "github.com/goccy/go-json"
)

var benchmarkTier3SelectiveResult benchmarkExtractionResult

func BenchmarkTier3SelectivePlaceholder_twitter_json(b *testing.B) {
	runTier3SelectivePlaceholderBenchmark(b, benchmarkFixtureTwitter)
}

func BenchmarkTier3SelectivePlaceholder_citm_catalog_json(b *testing.B) {
	runTier3SelectivePlaceholderBenchmark(b, benchmarkFixtureCITM)
}

// Tier 3 remains a DOM-era placeholder benchmark. It measures selective reads
// on the current DOM API only and does not imply a new On-Demand or path-query
// surface in v0.1.
func runTier3SelectivePlaceholderBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	for _, comparator := range availableTier2TypedComparators(b) {
		comparator := comparator
		b.Run(comparator.key, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			benchmarkRunWithNativeAllocMetrics(b, func() {
				for i := 0; i < b.N; i++ {
					result, err := benchmarkTier3SelectivePlaceholderExtract(comparator.key, fixtureName, data)
					if err != nil {
						b.Fatalf("%s selective placeholder(%s): %v", comparator.key, fixtureName, err)
					}
					benchmarkTier3SelectiveResult = result
				}
			})
		})
	}
}

func benchmarkTier3SelectivePlaceholderExtract(comparatorKey, fixtureName string, data []byte) (benchmarkExtractionResult, error) {
	switch comparatorKey {
	case benchmarkComparatorPureSimdjson:
		return benchmarkTier3SelectivePlaceholderPureSimdjson(fixtureName, data)
	case benchmarkComparatorEncodingStruct:
		value, err := benchmarkDecodeSharedSchema(json.Unmarshal, fixtureName, data)
		if err != nil {
			return benchmarkExtractionResult{}, err
		}
		return benchmarkTier3SelectivePlaceholderResultFromSharedSchema(value)
	case benchmarkComparatorGoccyGoJSON:
		value, err := benchmarkDecodeSharedSchema(gojson.Unmarshal, fixtureName, data)
		if err != nil {
			return benchmarkExtractionResult{}, err
		}
		return benchmarkTier3SelectivePlaceholderResultFromSharedSchema(value)
	case benchmarkComparatorBytedanceSonic:
		value, err := benchmarkDecodeSharedSchemaBytedanceSonic(fixtureName, data)
		if err != nil {
			return benchmarkExtractionResult{}, err
		}
		return benchmarkTier3SelectivePlaceholderResultFromSharedSchema(value)
	default:
		return benchmarkExtractionResult{}, fmt.Errorf("selective placeholder unsupported for comparator %q", comparatorKey)
	}
}

func benchmarkTier3SelectivePlaceholderPureSimdjson(fixtureName string, data []byte) (result benchmarkExtractionResult, err error) {
	parser, err := NewParser()
	if err != nil {
		return result, err
	}

	var doc *Doc
	defer func() {
		err = benchmarkCloseMaterializeResources(err, doc, parser)
	}()

	doc, err = parser.Parse(data)
	if err != nil {
		return result, err
	}

	switch fixtureName {
	case benchmarkFixtureTwitter:
		return benchmarkTier3SelectivePlaceholderTwitterElement(doc.Root())
	case benchmarkFixtureCITM:
		return benchmarkTier3SelectivePlaceholderCITMElement(doc.Root())
	default:
		return result, fmt.Errorf("selective placeholder unsupported for fixture %q", fixtureName)
	}
}

func benchmarkTier3SelectivePlaceholderResultFromSharedSchema(value any) (benchmarkExtractionResult, error) {
	switch row := value.(type) {
	case benchTwitterRow:
		return benchmarkTier3SelectivePlaceholderTwitterRow(row), nil
	case benchCITMRow:
		return benchmarkTier3SelectivePlaceholderCITMRow(row)
	default:
		return benchmarkExtractionResult{}, fmt.Errorf("unsupported selective placeholder value %T", value)
	}
}

func benchmarkTier3SelectivePlaceholderTwitterRow(row benchTwitterRow) benchmarkExtractionResult {
	result := benchmarkExtractionResult{}

	limit := len(row.Statuses)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		status := row.Statuses[i]
		result.int64Sum += status.User.ID
		if status.Retweeted {
			result.trueCount++
		}
	}

	return result
}

func benchmarkTier3SelectivePlaceholderCITMRow(row benchCITMRow) (benchmarkExtractionResult, error) {
	result := benchmarkExtractionResult{
		int64Sum: int64(len(row.AreaNames)),
	}

	ids, err := benchmarkSortedCITMEventIDs(row.Events)
	if err != nil {
		return benchmarkExtractionResult{}, err
	}
	limit := len(ids)
	if limit > 20 {
		limit = 20
	}
	for i := 0; i < limit; i++ {
		result.int64Sum += ids[i]
	}

	return result, nil
}

func benchmarkTier3SelectivePlaceholderTwitterElement(root Element) (benchmarkExtractionResult, error) {
	rootObject, err := root.AsObject()
	if err != nil {
		return benchmarkExtractionResult{}, err
	}
	statusesArray, err := benchmarkObjectFieldAsArray(rootObject, "statuses")
	if err != nil {
		return benchmarkExtractionResult{}, err
	}

	result := benchmarkExtractionResult{}
	iter := statusesArray.Iter()
	for seen := 0; seen < 10 && iter.Next(); seen++ {
		statusObject, err := iter.Value().AsObject()
		if err != nil {
			return benchmarkExtractionResult{}, err
		}
		userObject, err := benchmarkObjectFieldAsObject(statusObject, "user")
		if err != nil {
			return benchmarkExtractionResult{}, err
		}

		userID, err := benchmarkObjectFieldInt64(userObject, "id")
		if err != nil {
			return benchmarkExtractionResult{}, err
		}
		retweeted, err := benchmarkObjectFieldBool(statusObject, "retweeted")
		if err != nil {
			return benchmarkExtractionResult{}, err
		}

		result.int64Sum += userID
		if retweeted {
			result.trueCount++
		}
	}
	if err := iter.Err(); err != nil {
		return benchmarkExtractionResult{}, err
	}

	return result, nil
}

func benchmarkTier3SelectivePlaceholderCITMElement(root Element) (benchmarkExtractionResult, error) {
	rootObject, err := root.AsObject()
	if err != nil {
		return benchmarkExtractionResult{}, err
	}

	areaNames, err := benchmarkObjectFieldAsObject(rootObject, "areaNames")
	if err != nil {
		return benchmarkExtractionResult{}, err
	}
	events, err := benchmarkObjectFieldAsObject(rootObject, "events")
	if err != nil {
		return benchmarkExtractionResult{}, err
	}

	result := benchmarkExtractionResult{}
	areaCount, err := benchmarkCountObjectEntries(areaNames)
	if err != nil {
		return benchmarkExtractionResult{}, err
	}
	result.int64Sum = int64(areaCount)

	eventIter := events.Iter()
	keys := make([]string, 0)
	for eventIter.Next() {
		keys = append(keys, eventIter.Key())
	}
	if err := eventIter.Err(); err != nil {
		return benchmarkExtractionResult{}, err
	}

	ids, err := benchmarkSortedCITMEventIDsFromKeys(keys)
	if err != nil {
		return benchmarkExtractionResult{}, err
	}
	limit := len(ids)
	if limit > 20 {
		limit = 20
	}
	for i := 0; i < limit; i++ {
		result.int64Sum += ids[i]
	}

	return result, nil
}

func benchmarkCountObjectEntries(object Object) (int, error) {
	count := 0
	iter := object.Iter()
	for iter.Next() {
		count++
	}
	if err := iter.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

func benchmarkSortedCITMEventIDsFromKeys(keys []string) ([]int64, error) {
	ids := make([]int64, 0, len(keys))
	for _, key := range keys {
		value, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse event id %q: %w", key, err)
		}
		ids = append(ids, value)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}
