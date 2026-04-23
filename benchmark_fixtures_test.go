package purejson

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

const (
	benchmarkFixturesDir = "testdata/bench"
	oracleManifestPath   = "testdata/jsontestsuite/expectations.tsv"
	oracleCasesDir       = "testdata/jsontestsuite/cases"

	oracleManifestHeader = "relative_path\texpect\tnote"
	oracleExpectAccept   = "accept"
	oracleExpectReject   = "reject"
)

type oracleCase struct {
	relativePath string
	expect       string
	note         string
}

var benchmarkFixtureCache struct {
	once  sync.Once
	files map[string][]byte
	err   error
}

var oracleManifestCache struct {
	once  sync.Once
	cases []oracleCase
	err   error
}

func loadBenchmarkFixture(tb testing.TB, name string) []byte {
	tb.Helper()

	if err := validateBenchmarkFixtureName(name); err != nil {
		tb.Fatalf("loadBenchmarkFixture(%q): %v", name, err)
	}

	benchmarkFixtureCache.once.Do(func() {
		benchmarkFixtureCache.files, benchmarkFixtureCache.err = readBenchmarkFixtures()
	})
	if benchmarkFixtureCache.err != nil {
		tb.Fatalf("loadBenchmarkFixture(%q): %v", name, benchmarkFixtureCache.err)
	}

	data, ok := benchmarkFixtureCache.files[name]
	if !ok {
		tb.Fatalf(
			"loadBenchmarkFixture(%q): missing fixture %q under %s (available: %s)",
			name,
			name,
			benchmarkFixturesDir,
			strings.Join(sortedFixtureNames(benchmarkFixtureCache.files), ", "),
		)
	}

	return data
}

func loadOracleManifest(tb testing.TB) []oracleCase {
	tb.Helper()

	oracleManifestCache.once.Do(func() {
		oracleManifestCache.cases, oracleManifestCache.err = readOracleManifest()
	})
	if oracleManifestCache.err != nil {
		tb.Fatalf("loadOracleManifest: %v", oracleManifestCache.err)
	}

	return oracleManifestCache.cases
}

func validateBenchmarkFixtureName(name string) error {
	if name == "" {
		return fmt.Errorf("fixture name must not be empty")
	}
	if filepath.Base(name) != name || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("fixture name must be a single file name, got %q", name)
	}

	return nil
}

func readBenchmarkFixtures() (map[string][]byte, error) {
	entries, err := os.ReadDir(benchmarkFixturesDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", benchmarkFixturesDir, err)
	}

	files := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(benchmarkFixturesDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", filePath, err)
		}
		files[entry.Name()] = data
	}

	return files, nil
}

func readOracleManifest() ([]oracleCase, error) {
	file, err := os.Open(oracleManifestPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", oracleManifestPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read %s: %w", oracleManifestPath, err)
		}
		return nil, fmt.Errorf("%s: empty manifest", oracleManifestPath)
	}

	if header := scanner.Text(); header != oracleManifestHeader {
		return nil, fmt.Errorf("%s: header = %q, want %q", oracleManifestPath, header, oracleManifestHeader)
	}

	seen := make(map[string]struct{})
	var cases []oracleCase
	for lineNum := 2; scanner.Scan(); lineNum++ {
		line := scanner.Text()
		if line == "" {
			return nil, fmt.Errorf("%s:%d: blank line", oracleManifestPath, lineNum)
		}

		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			return nil, fmt.Errorf("%s:%d: got %d fields, want 3", oracleManifestPath, lineNum, len(fields))
		}

		relativePath, err := cleanOracleRelativePath(fields[0])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", oracleManifestPath, lineNum, err)
		}
		if _, ok := seen[relativePath]; ok {
			return nil, fmt.Errorf("%s:%d: duplicate manifest row for %q", oracleManifestPath, lineNum, relativePath)
		}

		expect := fields[1]
		if expect != oracleExpectAccept && expect != oracleExpectReject {
			return nil, fmt.Errorf("%s:%d: expect = %q, want %q or %q", oracleManifestPath, lineNum, expect, oracleExpectAccept, oracleExpectReject)
		}

		cases = append(cases, oracleCase{
			relativePath: relativePath,
			expect:       expect,
			note:         fields[2],
		})
		seen[relativePath] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", oracleManifestPath, err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("%s: no oracle rows found", oracleManifestPath)
	}

	return cases, nil
}

func cleanOracleRelativePath(relativePath string) (string, error) {
	if relativePath == "" {
		return "", fmt.Errorf("relative_path must not be empty")
	}

	clean := path.Clean(relativePath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("relative_path %q escapes %s", relativePath, oracleCasesDir)
	}

	return clean, nil
}

func sortedFixtureNames(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
