//go:build (!amd64 && !arm64) || go1.27 || !go1.17 || (arm64 && !go1.20)

package purejson

import "fmt"

func init() {
	registerOmittedBenchmarkComparator(benchmarkComparatorBytedanceSonic, "unsupported on this toolchain")
}

func benchmarkDecodeSharedSchemaBytedanceSonic(_ string, _ []byte) (any, error) {
	return nil, fmt.Errorf("%s shared-schema decode unavailable on this toolchain", benchmarkComparatorBytedanceSonic)
}
