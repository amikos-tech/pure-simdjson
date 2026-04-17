//go:build purejson_testbuild

package purejson

import (
	"fmt"
	"os"
	"runtime"
)

func attachParserFinalizer(parser *Parser) {
	runtime.SetFinalizer(parser, func(leaked *Parser) {
		if !leaked.hasLeakedState() {
			return
		}
		fmt.Fprintln(os.Stderr, "purejson leak: parser")
		leaked.finalizeLeaked()
	})
}

func clearParserFinalizer(parser *Parser) {
	runtime.SetFinalizer(parser, nil)
}

func attachDocFinalizer(doc *Doc) {
	runtime.SetFinalizer(doc, func(leaked *Doc) {
		if !leaked.hasLeakedState() {
			return
		}
		fmt.Fprintln(os.Stderr, "purejson leak: doc")
		leaked.finalizeLeaked()
	})
}

func clearDocFinalizer(doc *Doc) {
	runtime.SetFinalizer(doc, nil)
}

func testBuildFinalizersEnabled() bool {
	return true
}

func leakWarningsEnabled() bool {
	return true
}
