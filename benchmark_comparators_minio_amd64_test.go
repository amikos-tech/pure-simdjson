//go:build amd64

package purejson

import miniosimdjson "github.com/minio/simdjson-go"

func init() {
	registerBenchmarkComparator(benchmarkComparator{
		key:         benchmarkComparatorMinioSimdjson,
		materialize: benchmarkMaterializeMinioSimdjson,
	})
}

func benchmarkMaterializeMinioSimdjson(_ string, data []byte) (any, error) {
	parsed, err := miniosimdjson.Parse(data, nil)
	if err != nil {
		return nil, err
	}

	var result any
	if err := parsed.ForEach(func(iter miniosimdjson.Iter) error {
		value, err := iter.Interface()
		if err != nil {
			return err
		}
		result = value
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}
