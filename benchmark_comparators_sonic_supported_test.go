//go:build (amd64 && go1.17 && !go1.27) || (arm64 && go1.20 && !go1.27)

package purejson

import "github.com/bytedance/sonic"

func init() {
	registerBenchmarkComparator(benchmarkComparator{
		key:         benchmarkComparatorBytedanceSonic,
		materialize: benchmarkMaterializeBytedanceSonic,
	})
}

func benchmarkMaterializeBytedanceSonic(_ string, data []byte) (any, error) {
	var value any
	if err := sonic.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return value, nil
}
