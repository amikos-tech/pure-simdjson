package purejson

import "testing"

// Benchmark-local staging model for the documented SIMDJSON_PADDING contract.
const benchmarkStageInputPaddingBytes = 64

var benchmarkStageInputSink byte

func BenchmarkTier1Diagnostics_twitter_json(b *testing.B) {
	runTier1DiagnosticsBenchmark(b, benchmarkFixtureTwitter)
}

func BenchmarkTier1Diagnostics_citm_catalog_json(b *testing.B) {
	runTier1DiagnosticsBenchmark(b, benchmarkFixtureCITM)
}

func BenchmarkTier1Diagnostics_canada_json(b *testing.B) {
	runTier1DiagnosticsBenchmark(b, benchmarkFixtureCanada)
}

// These rows intentionally isolate pieces of the steady-state Tier 1 path.
// They are diagnostic cuts, not additive accounting: materialize-only keeps one
// parsed document open across the loop so the DOM walk and string extraction
// path can be measured without parse/setup noise.
func runTier1DiagnosticsBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	b.Run(benchmarkComparatorPureSimdjson+"-full", func(b *testing.B) {
		benchmarkRunTier1DiagnosticsFullPureSimdjson(b, fixtureName, data)
	})
	b.Run(benchmarkComparatorPureSimdjson+"-parse-only", func(b *testing.B) {
		benchmarkRunTier1DiagnosticsParseOnly(b, fixtureName, data)
	})
	b.Run(benchmarkComparatorPureSimdjson+"-materialize-only", func(b *testing.B) {
		benchmarkRunTier1DiagnosticsMaterializeOnly(b, fixtureName, data)
	})
	b.Run(benchmarkComparatorPureSimdjson+"-stage-input-reuse-model", func(b *testing.B) {
		benchmarkRunTier1DiagnosticsStageInputReuseModel(b, data)
	})
	b.Run(benchmarkComparatorPureSimdjson+"-stage-input-alloc-model", func(b *testing.B) {
		benchmarkRunTier1DiagnosticsStageInputAllocModel(b, data)
	})
	b.Run(benchmarkComparatorEncodingAny+"-full", func(b *testing.B) {
		benchmarkRunTier1DiagnosticsEncodingJSONAnyFull(b, data)
	})
}

func benchmarkRunTier1DiagnosticsFullPureSimdjson(b *testing.B, fixtureName string, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

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
				b.Fatalf("%s full(%s): %v", benchmarkComparatorPureSimdjson, fixtureName, err)
			}
			benchmarkTier1Result = value
		}
	})
}

func benchmarkRunTier1DiagnosticsParseOnly(b *testing.B, fixtureName string, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	parser := benchmarkWarmPureParser(b, fixtureName, data)
	defer func() {
		if err := parser.Close(); err != nil {
			b.Fatalf("parser.Close(%s): %v", fixtureName, err)
		}
	}()

	benchmarkRunWithNativeAllocMetrics(b, true, func() {
		for i := 0; i < b.N; i++ {
			doc, err := parser.Parse(data)
			if err != nil {
				b.Fatalf("%s parse-only(%s): %v", benchmarkComparatorPureSimdjson, fixtureName, err)
			}

			benchmarkParserResult = doc.Root()

			if err := doc.Close(); err != nil {
				b.Fatalf("doc.Close(%s): %v", fixtureName, err)
			}
		}
	})
}

func benchmarkRunTier1DiagnosticsMaterializeOnly(b *testing.B, fixtureName string, data []byte) {
	parser := benchmarkWarmPureParser(b, fixtureName, data)
	defer func() {
		if err := parser.Close(); err != nil {
			b.Fatalf("parser.Close(%s): %v", fixtureName, err)
		}
	}()

	doc, err := parser.Parse(data)
	if err != nil {
		b.Fatalf("%s materialize-only Parse(%s): %v", benchmarkComparatorPureSimdjson, fixtureName, err)
	}
	defer func() {
		if err := doc.Close(); err != nil {
			b.Fatalf("doc.Close(%s): %v", fixtureName, err)
		}
	}()

	root := doc.Root()

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	benchmarkRunWithNativeAllocMetrics(b, false, func() {
		for i := 0; i < b.N; i++ {
			value, err := benchmarkMaterializePureElement(root)
			if err != nil {
				b.Fatalf("%s materialize-only(%s): %v", benchmarkComparatorPureSimdjson, fixtureName, err)
			}
			benchmarkTier1Result = value
		}
	})
}

func benchmarkRunTier1DiagnosticsStageInputReuseModel(b *testing.B, data []byte) {
	staged := make([]byte, len(data)+benchmarkStageInputPaddingBytes)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkStageInputSink = benchmarkStageInputModel(staged, data)
	}
}

func benchmarkRunTier1DiagnosticsStageInputAllocModel(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		staged := make([]byte, len(data)+benchmarkStageInputPaddingBytes)
		benchmarkStageInputSink = benchmarkStageInputModel(staged, data)
	}
}

func benchmarkRunTier1DiagnosticsEncodingJSONAnyFull(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	benchmarkRunWithNativeAllocMetrics(b, false, func() {
		for i := 0; i < b.N; i++ {
			value, err := benchmarkMaterializeEncodingJSONAny("", data)
			if err != nil {
				b.Fatalf("%s full: %v", benchmarkComparatorEncodingAny, err)
			}
			benchmarkTier1Result = value
		}
	})
}

func benchmarkStageInputModel(dst, src []byte) byte {
	copied := copy(dst, src)
	clear(dst[copied:])
	return dst[0] ^ dst[len(dst)-1]
}
