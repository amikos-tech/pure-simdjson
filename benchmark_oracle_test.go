package purejson

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minOracleCaseCount = 300

func TestJSONTestSuiteOracle(t *testing.T) {
	manifest := loadOracleManifest(t)
	caseFiles := loadOracleCaseFiles(t)

	if len(manifest) < minOracleCaseCount {
		t.Fatalf("oracle manifest rows = %d, want at least %d", len(manifest), minOracleCaseCount)
	}
	if len(caseFiles) < minOracleCaseCount {
		t.Fatalf("oracle case files = %d, want at least %d", len(caseFiles), minOracleCaseCount)
	}

	manifestPaths := make(map[string]struct{}, len(manifest))
	for _, oracleCase := range manifest {
		if _, ok := caseFiles[oracleCase.relativePath]; !ok {
			t.Fatalf("oracle manifest references missing case %q", oracleCase.relativePath)
		}
		manifestPaths[oracleCase.relativePath] = struct{}{}
	}

	for relativePath := range caseFiles {
		if _, ok := manifestPaths[relativePath]; !ok {
			t.Fatalf("oracle cases directory contains unlisted file %q", relativePath)
		}
	}

	accepted := 0
	rejected := 0
	for _, oracleCase := range manifest {
		if strings.HasPrefix(filepath.Base(oracleCase.relativePath), "i_") && strings.TrimSpace(oracleCase.note) == "" {
			t.Fatalf("oracle manifest i_* case %q must document its implementation-defined expectation", oracleCase.relativePath)
		}

		data, err := os.ReadFile(caseFiles[oracleCase.relativePath])
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", oracleCase.relativePath, err)
		}

		parser, err := NewParser()
		if err != nil {
			t.Fatalf("NewParser() for %q: %v", oracleCase.relativePath, err)
		}

		doc, parseErr := parser.Parse(data)
		closeOracleResources(t, oracleCase.relativePath, doc, parser)

		switch oracleCase.expect {
		case oracleExpectAccept:
			if parseErr != nil {
				t.Fatalf("Parse(%q) error = %v, want success (%s)", oracleCase.relativePath, parseErr, oracleCase.note)
			}
			accepted++
		case oracleExpectReject:
			if parseErr == nil {
				t.Fatalf("Parse(%q) unexpectedly succeeded, want reject (%s)", oracleCase.relativePath, oracleCase.note)
			}
			rejected++
		default:
			t.Fatalf("oracle manifest expect = %q for %q, want %q or %q", oracleCase.expect, oracleCase.relativePath, oracleExpectAccept, oracleExpectReject)
		}
	}

	t.Logf("accepted=%d rejected=%d", accepted, rejected)
}

func loadOracleCaseFiles(tb testing.TB) map[string]string {
	tb.Helper()

	files := make(map[string]string)
	err := filepath.WalkDir(oracleCasesDir, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(oracleCasesDir, filePath)
		if err != nil {
			return err
		}

		manifestPath := filepath.ToSlash(relativePath)
		files[manifestPath] = filePath
		return nil
	})
	if err != nil {
		tb.Fatalf("loadOracleCaseFiles: %v", err)
	}

	return files
}

func closeOracleResources(tb testing.TB, relativePath string, doc *Doc, parser *Parser) {
	tb.Helper()

	if doc != nil {
		if err := doc.Close(); err != nil {
			tb.Fatalf("doc.Close() for %q: %v", relativePath, err)
		}
	}
	if parser != nil {
		if err := parser.Close(); err != nil {
			tb.Fatalf("parser.Close() for %q: %v", relativePath, err)
		}
	}
}
