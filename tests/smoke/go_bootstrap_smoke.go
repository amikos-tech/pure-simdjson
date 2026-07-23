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
	parser, err := purejson.NewParser(
		purejson.WithMaxCapacity(64),
		purejson.WithMaxDepth(8),
	)
	if err != nil {
		failf("NewParser(configured): %v", err)
	}

	if kernel := purejson.Kernel(); kernel == "" {
		failf("Kernel() = empty, want active implementation")
	}

	doc, err := parser.Parse([]byte(`{"big":18446744073709551616,"small":42}`))
	if err != nil {
		failf("Parse(configured ABI 1.2 smoke): %v", err)
	}

	object, err := doc.Root().AsObject()
	if err != nil {
		failf("Root().AsObject(): %v", err)
	}

	big, err := object.GetField("big")
	if err != nil {
		failf("GetField(big): %v", err)
	}
	if got := big.Type(); got != purejson.TypeBigInt {
		failf("GetField(big).Type() = %v, want TypeBigInt", got)
	}
	if got, err := big.GetBigInt(); err != nil {
		failf("GetField(big).GetBigInt(): %v", err)
	} else if got != "18446744073709551616" {
		failf("GetField(big).GetBigInt() = %q, want exact text", got)
	}

	small, err := object.GetField("small")
	if err != nil {
		failf("GetField(small): %v", err)
	}
	if got, err := small.GetInt64(); err != nil {
		failf("GetField(small).GetInt64(): %v", err)
	} else if got != 42 {
		failf("GetField(small).GetInt64() = %d, want 42", got)
	}

	if err := doc.Close(); err != nil {
		failf("doc.Close(): %v", err)
	}
	if err := parser.Close(); err != nil {
		failf("parser.Close(): %v", err)
	}
}
