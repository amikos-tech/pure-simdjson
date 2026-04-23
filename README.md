# pure-simdjson

`pure-simdjson` is a cgo-free Go wrapper around simdjson with a DOM API, typed number accessors, cursor-style iteration, and bootstrap support for prebuilt native libraries.

## Installation

```sh
go get github.com/amikos-tech/pure-simdjson@latest
```

The first `NewParser()` call downloads the platform-native library if it is not already cached locally. Mirror, offline, and override details live in [docs/bootstrap.md](docs/bootstrap.md).

## Quick Start

```go
package main

import (
	"fmt"

	purejson "github.com/amikos-tech/pure-simdjson"
)

func main() {
	parser, err := purejson.NewParser()
	if err != nil {
		panic(err)
	}
	defer func() { _ = parser.Close() }()

	doc, err := parser.Parse([]byte(`{"name":"alice"}`))
	if err != nil {
		panic(err)
	}
	defer func() { _ = doc.Close() }()

	object, err := doc.Root().AsObject()
	if err != nil {
		panic(err)
	}

	name, err := object.GetStringField("name")
	if err != nil {
		panic(err)
	}

	fmt.Println(name)
}
```

This snippet is derived from [example_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/example_test.go:25).

## Supported Platforms

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`

## Benchmark Snapshot

The current benchmark evidence is published in [results-v0.1.1.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks/results-v0.1.1.md) with the methodology in [benchmarks.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks.md). Comparator tables omit unsupported libraries on a given target instead of showing synthetic `N/A` rows.

Tier 1 is a strict full `any` materialization benchmark, and on the current `darwin/arm64` DOM ABI it is still slower than `encoding/json` for the three published corpus files: `0.21x` on `twitter.json`, `0.20x` on `citm_catalog.json`, and `0.17x` on `canada.json`. The current strength story is Tier 2 typed extraction and Tier 3 selective traversal on the DOM API, where the same snapshot shows `10.08x` to `14.52x` wins over `encoding/json` struct decoding in Tier 2 and `15.19x` to `20.05x` wins in the Tier 3 placeholder rows.

Use Tier 1 as the worst-case “parse and build a full generic Go tree” reference point. Use Tier 2 and Tier 3 as the representative performance story for the current API. Bootstrap and install details remain in [docs/bootstrap.md](docs/bootstrap.md).
