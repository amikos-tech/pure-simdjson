package purejson

func setExpectedABIVersionForTest(version uint32) func() {
	previousSet := abiVersionOverrideSet.Load()
	previousValue := abiVersionOverride.Load()

	abiVersionOverride.Store(version)
	abiVersionOverrideSet.Store(true)

	return func() {
		if previousSet {
			abiVersionOverride.Store(previousValue)
			abiVersionOverrideSet.Store(true)
			return
		}

		abiVersionOverride.Store(0)
		abiVersionOverrideSet.Store(false)
	}
}

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
