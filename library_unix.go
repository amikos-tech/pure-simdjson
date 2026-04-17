//go:build !windows

package purejson

import (
	"fmt"

	"github.com/ebitengine/purego"
)

func loadLibrary(path string) (uintptr, error) {
	handle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return 0, fmt.Errorf("load %s: %w", path, err)
	}
	if handle == 0 {
		return 0, fmt.Errorf("load %s: nil handle", path)
	}
	return handle, nil
}

func lookupSymbol(handle uintptr, name string) (uintptr, error) {
	sym, err := purego.Dlsym(handle, name)
	if err != nil {
		return 0, fmt.Errorf("lookup %s: %w", name, err)
	}
	if sym == 0 {
		return 0, fmt.Errorf("lookup %s: nil symbol", name)
	}
	return sym, nil
}
