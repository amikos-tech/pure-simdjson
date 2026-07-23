package purejson

func resetFinalizerCountsForTest() {
	parserFinalizerCount.Store(0)
	docFinalizerCount.Store(0)
}

func parserFinalizerCountForTest() int64 {
	return parserFinalizerCount.Load()
}

func docFinalizerCountForTest() int64 {
	return docFinalizerCount.Load()
}
