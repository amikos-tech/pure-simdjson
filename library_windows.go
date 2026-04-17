//go:build windows

package purejson

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func loadLibrary(path string) (uintptr, error) {
	handle, err := windows.LoadLibrary(path)
	if err != nil {
		return 0, fmt.Errorf("load %s: %w", path, err)
	}
	if handle == 0 {
		return 0, fmt.Errorf("load %s: nil handle", path)
	}
	return uintptr(handle), nil
}

func lookupSymbol(handle uintptr, name string) (uintptr, error) {
	sym, err := windows.GetProcAddress(windows.Handle(handle), name)
	if err != nil {
		return 0, fmt.Errorf("lookup %s: %w", name, err)
	}
	if sym == 0 {
		return 0, fmt.Errorf("lookup %s: nil symbol", name)
	}
	return uintptr(sym), nil
}
