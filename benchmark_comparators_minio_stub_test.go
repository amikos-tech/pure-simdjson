//go:build !amd64

package purejson

func init() {
	registerOmittedBenchmarkComparator(benchmarkComparatorMinioSimdjson, "unsupported on this GOARCH")
}
