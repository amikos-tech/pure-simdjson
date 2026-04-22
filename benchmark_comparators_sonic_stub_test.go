//go:build (!amd64 && !arm64) || go1.27 || !go1.17 || (arm64 && !go1.20)

package purejson

func init() {
	registerOmittedBenchmarkComparator(benchmarkComparatorBytedanceSonic, "unsupported on this toolchain")
}
