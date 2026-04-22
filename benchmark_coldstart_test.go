package purejson

import "testing"

var benchmarkParserResult Element

func BenchmarkColdStart_twitter_json(b *testing.B) {
	runColdStartBenchmark(b, benchmarkFixtureTwitter)
}

func BenchmarkColdStart_citm_catalog_json(b *testing.B) {
	runColdStartBenchmark(b, benchmarkFixtureCITM)
}

func BenchmarkColdStart_canada_json(b *testing.B) {
	runColdStartBenchmark(b, benchmarkFixtureCanada)
}

func BenchmarkWarm_twitter_json(b *testing.B) {
	runWarmBenchmark(b, benchmarkFixtureTwitter)
}

func BenchmarkWarm_citm_catalog_json(b *testing.B) {
	runWarmBenchmark(b, benchmarkFixtureCITM)
}

func BenchmarkWarm_canada_json(b *testing.B) {
	runWarmBenchmark(b, benchmarkFixtureCanada)
}

// cold-start here means first Parse after NewParser inside an already loaded process.
// It intentionally excludes bootstrap or download time.
// Both cold and warm families report native-bytes/op, native-allocs/op, and
// native-live-bytes through benchmarkRunWithNativeAllocMetrics.
func runColdStartBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	benchmarkRunWithNativeAllocMetrics(b, func() {
		for i := 0; i < b.N; i++ {
			parser, err := NewParser()
			if err != nil {
				b.Fatalf("NewParser(%s): %v", fixtureName, err)
			}

			doc, err := parser.Parse(data)
			if err != nil {
				_ = parser.Close()
				b.Fatalf("Parse(%s): %v", fixtureName, err)
			}

			benchmarkParserResult = doc.Root()

			if err := doc.Close(); err != nil {
				_ = parser.Close()
				b.Fatalf("doc.Close(%s): %v", fixtureName, err)
			}
			if err := parser.Close(); err != nil {
				b.Fatalf("parser.Close(%s): %v", fixtureName, err)
			}
		}
	})
}

// Warm benchmarks do one warm-up parse before ResetTimer and then reuse the parser.
func runWarmBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	parser, err := NewParser()
	if err != nil {
		b.Fatalf("NewParser(%s): %v", fixtureName, err)
	}

	warmDoc, err := parser.Parse(data)
	if err != nil {
		_ = parser.Close()
		b.Fatalf("warm Parse(%s): %v", fixtureName, err)
	}
	if err := warmDoc.Close(); err != nil {
		_ = parser.Close()
		b.Fatalf("warm doc.Close(%s): %v", fixtureName, err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	benchmarkRunWithNativeAllocMetrics(b, func() {
		for i := 0; i < b.N; i++ {
			doc, err := parser.Parse(data)
			if err != nil {
				b.Fatalf("Parse(%s): %v", fixtureName, err)
			}

			benchmarkParserResult = doc.Root()

			if err := doc.Close(); err != nil {
				b.Fatalf("doc.Close(%s): %v", fixtureName, err)
			}
		}
	})
	if err := parser.Close(); err != nil {
		b.Fatalf("parser.Close(%s): %v", fixtureName, err)
	}
}
