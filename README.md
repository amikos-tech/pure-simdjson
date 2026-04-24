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

This snippet is derived from [example_test.go](example_test.go).

## Supported Platforms

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`

## Benchmark Snapshot

The current benchmark evidence is published in [v0.1.2 results](docs/benchmarks/results-v0.1.2.md) with the [benchmark methodology](docs/benchmarks.md). Comparator tables omit unsupported libraries on a given target instead of showing synthetic `N/A` rows.

On the `linux/amd64` CI target, Tier 1 full `any` materialization now beats `encoding/json` + `any` by `3.15x` on `twitter.json`, `3.39x` on `citm_catalog.json`, and `2.47x` on `canada.json`. Tier 2 typed extraction beats `encoding/json` + struct by `12.49x` to `14.56x`, and Tier 3 selective traversal on the current DOM API beats `encoding/json` + struct by `15.18x` to `15.97x`. Headline numbers come from linux/amd64; other platforms may differ.

The `v0.1.2` benchmark snapshot is upcoming-release evidence; Phase 09.1 still owns bootstrap artifact and default-install alignment before a release tag. Bootstrap and install details remain in [docs/bootstrap.md](docs/bootstrap.md).
