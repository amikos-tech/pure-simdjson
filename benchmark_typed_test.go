package purejson

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"testing"

	gojson "github.com/goccy/go-json"
)

type benchmarkExtractionResult struct {
	int64Sum   int64
	float64Sum float64
	stringSize int64
	trueCount  int64
}

var benchmarkTier2TypedResult benchmarkExtractionResult

func BenchmarkTier2Typed_twitter_json(b *testing.B) {
	runTier2TypedBenchmark(b, benchmarkFixtureTwitter)
}

func BenchmarkTier2Typed_citm_catalog_json(b *testing.B) {
	runTier2TypedBenchmark(b, benchmarkFixtureCITM)
}

func BenchmarkTier2Typed_canada_json(b *testing.B) {
	runTier2TypedBenchmark(b, benchmarkFixtureCanada)
}

func runTier2TypedBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	for _, comparator := range availableTier2TypedComparators(b) {
		comparator := comparator
		b.Run(comparator.key, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))

			if comparator.key == benchmarkComparatorPureSimdjson {
				parser := benchmarkWarmPureParser(b, fixtureName, data)
				defer func() {
					if err := parser.Close(); err != nil {
						b.Fatalf("parser.Close(%s): %v", fixtureName, err)
					}
				}()

				benchmarkRunWithNativeAllocMetrics(b, true, func() {
					for i := 0; i < b.N; i++ {
						result, err := benchmarkTier2TypedExtractPureSimdjsonWithParser(parser, fixtureName, data)
						if err != nil {
							b.Fatalf("%s typed extract(%s): %v", comparator.key, fixtureName, err)
						}
						benchmarkTier2TypedResult = result
					}
				})
				return
			}

			benchmarkRunWithNativeAllocMetrics(b, false, func() {
				for i := 0; i < b.N; i++ {
					result, err := benchmarkTier2TypedExtract(comparator.key, fixtureName, data)
					if err != nil {
						b.Fatalf("%s typed extract(%s): %v", comparator.key, fixtureName, err)
					}
					benchmarkTier2TypedResult = result
				}
			})
		})
	}
}

func availableTier2TypedComparators(tb testing.TB) []benchmarkComparator {
	tb.Helper()

	var comparators []benchmarkComparator
	for _, comparator := range availableBenchmarkComparators(tb) {
		switch comparator.key {
		case benchmarkComparatorPureSimdjson,
			benchmarkComparatorEncodingStruct,
			benchmarkComparatorBytedanceSonic,
			benchmarkComparatorGoccyGoJSON:
			comparators = append(comparators, comparator)
		}
	}
	if len(comparators) == 0 {
		tb.Fatal("no Tier 2 typed benchmark comparators are available")
	}

	return comparators
}

func benchmarkTier2TypedExtract(comparatorKey, fixtureName string, data []byte) (benchmarkExtractionResult, error) {
	switch comparatorKey {
	case benchmarkComparatorPureSimdjson:
		return benchmarkTier2TypedExtractPureSimdjson(fixtureName, data)
	case benchmarkComparatorEncodingStruct:
		value, err := benchmarkDecodeSharedSchema(json.Unmarshal, fixtureName, data)
		if err != nil {
			return benchmarkExtractionResult{}, err
		}
		return benchmarkTier2TypedResultFromSharedSchema(value)
	case benchmarkComparatorGoccyGoJSON:
		value, err := benchmarkDecodeSharedSchema(gojson.Unmarshal, fixtureName, data)
		if err != nil {
			return benchmarkExtractionResult{}, err
		}
		return benchmarkTier2TypedResultFromSharedSchema(value)
	case benchmarkComparatorBytedanceSonic:
		value, err := benchmarkDecodeSharedSchemaBytedanceSonic(fixtureName, data)
		if err != nil {
			return benchmarkExtractionResult{}, err
		}
		return benchmarkTier2TypedResultFromSharedSchema(value)
	default:
		return benchmarkExtractionResult{}, fmt.Errorf("typed extraction unsupported for comparator %q", comparatorKey)
	}
}

func benchmarkTier2TypedExtractPureSimdjson(fixtureName string, data []byte) (result benchmarkExtractionResult, err error) {
	parser, err := NewParser()
	if err != nil {
		return result, err
	}
	defer func() {
		err = benchmarkCloseMaterializeResources(err, nil, parser)
	}()

	return benchmarkTier2TypedExtractPureSimdjsonWithParser(parser, fixtureName, data)
}

func benchmarkTier2TypedExtractPureSimdjsonWithParser(parser *Parser, fixtureName string, data []byte) (result benchmarkExtractionResult, err error) {
	var doc *Doc
	defer func() {
		err = benchmarkCloseMaterializeResources(err, doc, nil)
	}()

	doc, err = parser.Parse(data)
	if err != nil {
		return result, err
	}

	value, err := benchmarkDecodePureSimdjsonSharedSchema(fixtureName, doc.Root())
	if err != nil {
		return result, err
	}

	return benchmarkTier2TypedResultFromSharedSchema(value)
}

func benchmarkDecodePureSimdjsonSharedSchema(fixtureName string, root Element) (any, error) {
	switch fixtureName {
	case benchmarkFixtureTwitter:
		return benchmarkDecodePureSimdjsonTwitterRow(root)
	case benchmarkFixtureCITM:
		return benchmarkDecodePureSimdjsonCITMRow(root)
	case benchmarkFixtureCanada:
		return benchmarkDecodePureSimdjsonCanadaRow(root)
	default:
		return nil, fmt.Errorf("pure-simdjson shared-schema decode unsupported for fixture %q", fixtureName)
	}
}

func benchmarkTier2TypedResultFromSharedSchema(value any) (benchmarkExtractionResult, error) {
	switch row := value.(type) {
	case benchTwitterRow:
		return benchmarkTier2TypedResultFromTwitterRow(row), nil
	case benchCITMRow:
		return benchmarkTier2TypedResultFromCITMRow(row)
	case benchCanadaRow:
		return benchmarkTier2TypedResultFromCanadaRow(row), nil
	default:
		return benchmarkExtractionResult{}, fmt.Errorf("unsupported Tier 2 shared-schema value %T", value)
	}
}

func benchmarkTier2TypedResultFromTwitterRow(row benchTwitterRow) benchmarkExtractionResult {
	result := benchmarkExtractionResult{
		int64Sum: row.SearchMetadata.MaxID,
	}

	limit := len(row.Statuses)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		status := row.Statuses[i]
		result.int64Sum += status.ID + status.User.ID
		if status.Favorited {
			result.trueCount++
		}
		if status.Retweeted {
			result.trueCount++
		}
		result.stringSize += int64(len(status.User.Name))
	}

	return result
}

func benchmarkTier2TypedResultFromCITMRow(row benchCITMRow) (benchmarkExtractionResult, error) {
	result := benchmarkExtractionResult{
		int64Sum: int64(len(row.AreaNames) + len(row.TopicNames)),
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

func benchmarkTier2TypedResultFromCanadaRow(row benchCanadaRow) benchmarkExtractionResult {
	result := benchmarkExtractionResult{
		stringSize: int64(len(row.Type)),
	}

	valuesSeen := 0
	for _, feature := range row.Features {
		for _, polygon := range feature.Geometry.Coordinates {
			for _, point := range polygon {
				for _, value := range point {
					result.float64Sum += value
					valuesSeen++
					if valuesSeen == 256 {
						return result
					}
				}
			}
		}
	}

	return result
}

func benchmarkDecodePureSimdjsonTwitterRow(root Element) (benchTwitterRow, error) {
	rootObject, err := root.AsObject()
	if err != nil {
		return benchTwitterRow{}, err
	}

	searchMetadata, err := benchmarkObjectFieldAsObject(rootObject, "search_metadata")
	if err != nil {
		return benchTwitterRow{}, err
	}
	maxID, err := benchmarkObjectFieldInt64(searchMetadata, "max_id")
	if err != nil {
		return benchTwitterRow{}, err
	}

	statusesArray, err := benchmarkObjectFieldAsArray(rootObject, "statuses")
	if err != nil {
		return benchTwitterRow{}, err
	}

	row := benchTwitterRow{
		SearchMetadata: benchTwitterSearchMetadata{MaxID: maxID},
		Statuses:       make([]benchTwitterStatus, 0, 10),
	}

	iter := statusesArray.Iter()
	for len(row.Statuses) < 10 && iter.Next() {
		statusObject, err := iter.Value().AsObject()
		if err != nil {
			return benchTwitterRow{}, err
		}

		userObject, err := benchmarkObjectFieldAsObject(statusObject, "user")
		if err != nil {
			return benchTwitterRow{}, err
		}

		statusID, err := benchmarkObjectFieldInt64(statusObject, "id")
		if err != nil {
			return benchTwitterRow{}, err
		}
		favorited, err := benchmarkObjectFieldBool(statusObject, "favorited")
		if err != nil {
			return benchTwitterRow{}, err
		}
		retweeted, err := benchmarkObjectFieldBool(statusObject, "retweeted")
		if err != nil {
			return benchTwitterRow{}, err
		}
		userID, err := benchmarkObjectFieldInt64(userObject, "id")
		if err != nil {
			return benchTwitterRow{}, err
		}
		userName, err := benchmarkObjectFieldString(userObject, "name")
		if err != nil {
			return benchTwitterRow{}, err
		}

		row.Statuses = append(row.Statuses, benchTwitterStatus{
			Favorited: favorited,
			ID:        statusID,
			Retweeted: retweeted,
			User: benchTwitterUser{
				ID:   userID,
				Name: userName,
			},
		})
	}
	if err := iter.Err(); err != nil {
		return benchTwitterRow{}, err
	}

	return row, nil
}

func benchmarkDecodePureSimdjsonCITMRow(root Element) (benchCITMRow, error) {
	rootObject, err := root.AsObject()
	if err != nil {
		return benchCITMRow{}, err
	}

	areaNames, err := benchmarkObjectFieldAsObject(rootObject, "areaNames")
	if err != nil {
		return benchCITMRow{}, err
	}
	topicNames, err := benchmarkObjectFieldAsObject(rootObject, "topicNames")
	if err != nil {
		return benchCITMRow{}, err
	}
	events, err := benchmarkObjectFieldAsObject(rootObject, "events")
	if err != nil {
		return benchCITMRow{}, err
	}

	row := benchCITMRow{
		AreaNames:  make(map[string]string),
		TopicNames: make(map[string]string),
		Events:     make(map[string]benchCITMEvent),
	}

	areaIter := areaNames.Iter()
	for areaIter.Next() {
		row.AreaNames[areaIter.Key()] = ""
	}
	if err := areaIter.Err(); err != nil {
		return benchCITMRow{}, err
	}

	topicIter := topicNames.Iter()
	for topicIter.Next() {
		row.TopicNames[topicIter.Key()] = ""
	}
	if err := topicIter.Err(); err != nil {
		return benchCITMRow{}, err
	}

	eventIter := events.Iter()
	for eventIter.Next() {
		row.Events[eventIter.Key()] = benchCITMEvent{}
	}
	if err := eventIter.Err(); err != nil {
		return benchCITMRow{}, err
	}

	return row, nil
}

func benchmarkDecodePureSimdjsonCanadaRow(root Element) (benchCanadaRow, error) {
	rootObject, err := root.AsObject()
	if err != nil {
		return benchCanadaRow{}, err
	}

	typeName, err := benchmarkObjectFieldString(rootObject, "type")
	if err != nil {
		return benchCanadaRow{}, err
	}
	featuresArray, err := benchmarkObjectFieldAsArray(rootObject, "features")
	if err != nil {
		return benchCanadaRow{}, err
	}

	row := benchCanadaRow{
		Type:     typeName,
		Features: make([]benchCanadaFeature, 0, 1),
	}

	valuesSeen := 0
	featureIter := featuresArray.Iter()
	for featureIter.Next() && valuesSeen < 256 {
		featureObject, err := featureIter.Value().AsObject()
		if err != nil {
			return benchCanadaRow{}, err
		}

		geometryObject, err := benchmarkObjectFieldAsObject(featureObject, "geometry")
		if err != nil {
			return benchCanadaRow{}, err
		}
		coordinatesArray, err := benchmarkObjectFieldAsArray(geometryObject, "coordinates")
		if err != nil {
			return benchCanadaRow{}, err
		}

		polygons := make([][][]float64, 0)
		polygonIter := coordinatesArray.Iter()
		for polygonIter.Next() && valuesSeen < 256 {
			polygonArray, err := polygonIter.Value().AsArray()
			if err != nil {
				return benchCanadaRow{}, err
			}

			points := make([][]float64, 0)
			pointIter := polygonArray.Iter()
			for pointIter.Next() && valuesSeen < 256 {
				pointArray, err := pointIter.Value().AsArray()
				if err != nil {
					return benchCanadaRow{}, err
				}

				coords := make([]float64, 0)
				coordIter := pointArray.Iter()
				for coordIter.Next() && valuesSeen < 256 {
					value, err := coordIter.Value().GetFloat64()
					if err != nil {
						return benchCanadaRow{}, err
					}
					coords = append(coords, value)
					valuesSeen++
				}
				if err := coordIter.Err(); err != nil {
					return benchCanadaRow{}, err
				}
				if len(coords) > 0 {
					points = append(points, coords)
				}
			}
			if err := pointIter.Err(); err != nil {
				return benchCanadaRow{}, err
			}
			if len(points) > 0 {
				polygons = append(polygons, points)
			}
		}
		if err := polygonIter.Err(); err != nil {
			return benchCanadaRow{}, err
		}
		if len(polygons) > 0 {
			row.Features = append(row.Features, benchCanadaFeature{
				Geometry: benchCanadaGeometry{Coordinates: polygons},
			})
		}
	}
	if err := featureIter.Err(); err != nil {
		return benchCanadaRow{}, err
	}

	return row, nil
}

func benchmarkSortedCITMEventIDs(events map[string]benchCITMEvent) ([]int64, error) {
	ids := make([]int64, 0, len(events))
	for eventID := range events {
		parsed, err := strconv.ParseInt(eventID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse event id %q: %w", eventID, err)
		}
		ids = append(ids, parsed)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func benchmarkObjectFieldAsObject(object Object, key string) (Object, error) {
	field, err := object.GetField(key)
	if err != nil {
		return Object{}, err
	}
	return field.AsObject()
}

func benchmarkObjectFieldAsArray(object Object, key string) (Array, error) {
	field, err := object.GetField(key)
	if err != nil {
		return Array{}, err
	}
	return field.AsArray()
}

func benchmarkObjectFieldInt64(object Object, key string) (int64, error) {
	field, err := object.GetField(key)
	if err != nil {
		return 0, err
	}
	return field.GetInt64()
}

func benchmarkObjectFieldBool(object Object, key string) (bool, error) {
	field, err := object.GetField(key)
	if err != nil {
		return false, err
	}
	return field.GetBool()
}

func benchmarkObjectFieldString(object Object, key string) (string, error) {
	field, err := object.GetField(key)
	if err != nil {
		return "", err
	}
	return field.GetString()
}
