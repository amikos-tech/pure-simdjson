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

func runTier1FullParseBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	for _, comparator := range availableBenchmarkComparators(b) {
		comparator := comparator
		b.Run(comparator.key, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				value, err := comparator.materialize(fixtureName, data)
				if err != nil {
					b.Fatalf("%s materialize(%s): %v", comparator.key, fixtureName, err)
				}
				benchmarkTier1Result = value
			}
		})
	}
}
