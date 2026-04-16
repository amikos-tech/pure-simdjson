//go:build !purejson_testbuild

package purejson

import "runtime"

func attachParserFinalizer(parser *Parser) {
	runtime.SetFinalizer(parser, func(leaked *Parser) {
		leaked.finalizeLeaked()
	})
}

func clearParserFinalizer(parser *Parser) {
	runtime.SetFinalizer(parser, nil)
}

func attachDocFinalizer(doc *Doc) {
	runtime.SetFinalizer(doc, func(leaked *Doc) {
		leaked.finalizeLeaked()
	})
}

func clearDocFinalizer(doc *Doc) {
	runtime.SetFinalizer(doc, nil)
}

func testBuildFinalizersEnabled() bool {
	return false
}
