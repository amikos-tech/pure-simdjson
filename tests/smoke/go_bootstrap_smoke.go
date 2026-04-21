//go:build ignore

package main

import (
	"fmt"
	"os"

	purejson "github.com/amikos-tech/pure-simdjson"
)

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func main() {
	parser, err := purejson.NewParser()
	if err != nil {
		failf("NewParser(): %v", err)
	}
	defer func() {
		if err := parser.Close(); err != nil {
			failf("parser.Close(): %v", err)
		}
	}()

	doc, err := parser.Parse([]byte("42"))
	if err != nil {
		failf("Parse(42): %v", err)
	}
	defer func() {
		if err := doc.Close(); err != nil {
			failf("doc.Close(): %v", err)
		}
	}()

	value, err := doc.Root().GetInt64()
	if err != nil {
		failf("Root().GetInt64(): %v", err)
	}
	if value != 42 {
		failf("Root().GetInt64() = %d, want 42", value)
	}
}
