package purejson

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func TestPhase7BenchmarkFixtureContract(t *testing.T) {
	fixtures, err := readBenchmarkFixtures()
	if err != nil {
		t.Fatalf("readBenchmarkFixtures(): %v", err)
	}

	want := []string{
		"canada.json",
		"citm_catalog.json",
		"mesh.json",
		"numbers.json",
		"twitter.json",
	}
	got := sortedFixtureNames(fixtures)
	if !slices.Equal(got, want) {
		t.Fatalf("benchmark fixture set = %v, want %v", got, want)
	}

	for _, name := range want {
		if len(fixtures[name]) == 0 {
			t.Fatalf("fixture %q is empty", name)
		}
	}

	readme := mustReadPhase7ContractFile(t, "testdata/bench/README.md")
	requirePhase7ContractContainsAll(t, "testdata/bench/README.md", readme,
		"| filename | source | upstream_ref | sha256 | notes |",
		"19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448",
		"`twitter.json`",
		"`citm_catalog.json`",
		"`canada.json`",
		"`mesh.json`",
		"`numbers.json`",
	)
}

func TestPhase7BenchmarkComparatorContract(t *testing.T) {
	comparators := allBenchmarkComparators(t)
	if len(comparators) != len(benchmarkCanonicalComparatorKeys) {
		t.Fatalf("registered comparators = %d, want %d", len(comparators), len(benchmarkCanonicalComparatorKeys))
	}

	byKey := make(map[string]benchmarkComparator, len(comparators))
	for _, comparator := range comparators {
		byKey[comparator.key] = comparator
	}

	for _, key := range benchmarkCanonicalComparatorKeys {
		comparator, ok := byKey[key]
		if !ok {
			t.Fatalf("benchmark comparator %q is not registered", key)
		}
		if comparator.available() {
			if comparator.materialize == nil {
				t.Fatalf("benchmark comparator %q reports available without a materializer", key)
			}
			if comparator.omissionReason != "" {
				t.Fatalf("benchmark comparator %q is both available and omitted", key)
			}
			continue
		}
		if comparator.omissionReason == "" {
			t.Fatalf("benchmark comparator %q is unavailable without an omission reason", key)
		}
	}

	for _, key := range []string{
		benchmarkComparatorPureSimdjson,
		benchmarkComparatorEncodingAny,
		benchmarkComparatorEncodingStruct,
		benchmarkComparatorGoccyGoJSON,
	} {
		if !byKey[key].available() {
			t.Fatalf("benchmark comparator %q must be available on all supported Phase 7 benchmark targets", key)
		}
	}
}

func TestPhase7ReleaseArtifactContract(t *testing.T) {
	t.Run("README", func(t *testing.T) {
		readme := mustReadPhase7ContractFile(t, "README.md")
		requirePhase7ContractContainsAll(t, "README.md", readme,
			"# pure-simdjson",
			"## Installation",
			"## Quick Start",
			"## Supported Platforms",
			"## Benchmark Snapshot",
			"results-v0.1.2.md",
			"docs/bootstrap.md",
			"Tier 1",
			"Tier 2",
			"Tier 3",
			"NewParser",
			"Parse",
			"Close",
		)
	})

	t.Run("Methodology", func(t *testing.T) {
		doc := mustReadPhase7ContractFile(t, "docs/benchmarks.md")
		requirePhase7ContractContainsAll(t, "docs/benchmarks.md", doc,
			"Tier 1",
			"Tier 2",
			"Tier 3",
			"first Parse after NewParser",
			"native allocator",
			"materialization dominates parse",
			"run_benchstat.sh",
			"testdata/benchmark-results/v0.1.2",
		)
	})

	t.Run("ResultsSnapshot", func(t *testing.T) {
		results := mustReadPhase7ContractFile(t, "docs/benchmarks/results-v0.1.1.md")
		requirePhase7ContractContainsAll(t, "docs/benchmarks/results-v0.1.1.md", results,
			"BENCH-07 truthful-positioning: PASS",
			"Tier 1 headline on current DOM ABI: NOT SUPPORTED",
			"Tier 2/Tier 3 headline on current DOM ABI: SUPPORTED",
			"x86_64 minio parity on this snapshot: UNAVAILABLE",
			"encoding/json + any",
			"minio/simdjson-go",
			"phase7.bench.txt",
			"coldwarm.bench.txt",
			"tier1-diagnostics.bench.txt",
		)
	})

	t.Run("Changelog", func(t *testing.T) {
		changelog := mustReadPhase7ContractFile(t, "CHANGELOG.md")
		requirePhase7ContractContainsAll(t, "CHANGELOG.md", changelog,
			"## [Unreleased]",
			"Keep a Changelog",
			"Benchmark",
			"README",
			"NOTICE",
			"LICENSE",
			"results-v0.1.2",
			"Phase 8",
			"Phase 9",
		)
	})

	t.Run("Legal", func(t *testing.T) {
		license := mustReadPhase7ContractFile(t, "LICENSE")
		requirePhase7ContractContainsAll(t, "LICENSE", license,
			"MIT License",
			"Copyright (c) 2026 Amikos Tech",
		)

		notice := mustReadPhase7ContractFile(t, "NOTICE")
		requirePhase7ContractContainsAll(t, "NOTICE", notice,
			"simdjson",
			"Apache License",
			"LICENSE-MIT",
			"third_party/simdjson/LICENSE",
		)
	})

	t.Run("CloseoutRouting", func(t *testing.T) {
		summary := mustReadPhase7ContractFile(t, ".planning/phases/07-benchmarks-v0.1-release/07-06-SUMMARY.md")
		requirePhase7ContractContainsAll(t, ".planning/phases/07-benchmarks-v0.1-release/07-06-SUMMARY.md", summary,
			"Phase 8",
			"Phase 9",
			"results-v0.1.1.md",
		)

		state := mustReadPhase7ContractFile(t, ".planning/STATE.md")
		requirePhase7ContractContainsAll(t, ".planning/STATE.md", state,
			"Phase 08",
		)

		roadmap := mustReadPhase7ContractFile(t, ".planning/ROADMAP.md")
		requirePhase7ContractContainsAll(t, ".planning/ROADMAP.md", roadmap,
			"| 7. Benchmarks + Release-Facing Artifacts | 6/6 | Complete |",
		)
	})
}

func mustReadPhase7ContractFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}

	return string(data)
}

func requirePhase7ContractContainsAll(t *testing.T, path, content string, needles ...string) {
	t.Helper()

	for _, needle := range needles {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s does not contain %q", path, needle)
		}
	}
}
