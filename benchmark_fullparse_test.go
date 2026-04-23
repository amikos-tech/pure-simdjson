package purejson

import "testing"

var benchmarkTier1Result any

func BenchmarkTier1FullParse_twitter_json(b *testing.B) {
	runTier1FullParseBenchmark(b, benchmarkFixtureTwitter)
}

func BenchmarkTier1FullParse_citm_catalog_json(b *testing.B) {
	runTier1FullParseBenchmark(b, benchmarkFixtureCITM)
}

func BenchmarkTier1FullParse_canada_json(b *testing.B) {
	runTier1FullParseBenchmark(b, benchmarkFixtureCanada)
}

// Tier 1 reports native-bytes/op, native-allocs/op, and native-live-bytes via
// benchmarkRunWithNativeAllocMetrics so the published rows include native and
// Go allocation signals together. Cold-start parser construction lives in the
// dedicated BenchmarkColdStart_* family, so the steady-state pure-simdjson path
// reuses a warmed parser here.
func runTier1FullParseBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	for _, comparator := range availableBenchmarkComparators(b) {
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
						value, err := benchmarkMaterializePureSimdjsonWithParser(parser, data)
						if err != nil {
							b.Fatalf("%s materialize(%s): %v", comparator.key, fixtureName, err)
						}
						benchmarkTier1Result = value
					}
				})
				return
			}

			benchmarkRunWithNativeAllocMetrics(b, false, func() {
				for i := 0; i < b.N; i++ {
					value, err := comparator.materialize(fixtureName, data)
					if err != nil {
						b.Fatalf("%s materialize(%s): %v", comparator.key, fixtureName, err)
					}
					benchmarkTier1Result = value
				}
			})
		})
	}
}
