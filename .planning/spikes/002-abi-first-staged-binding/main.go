package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ebitengine/purego"
)

const expectedABI = uint32(0x00010002)

var mandatorySymbols = []string{
	"pure_simdjson_parser_new_configured",
	"pure_simdjson_parser_get_last_error_has_offset",
	"pure_simdjson_element_get_bigint",
	"pure_simdjson_set_implementation",
	"pure_simdjson_lock_implementation_selection",
}

type result struct {
	Fixture          string   `json:"fixture"`
	Strategy         string   `json:"strategy"`
	ABI              string   `json:"abi,omitempty"`
	Outcome          string   `json:"outcome"`
	MandatoryLookups int      `json:"mandatory_lookups"`
	Missing          string   `json:"missing,omitempty"`
	Trace            []string `json:"trace"`
}

func probeABI(handle uintptr, trace *[]string) (uint32, error) {
	const name = "pure_simdjson_get_abi_version"
	*trace = append(*trace, "lookup:"+name)
	symbol, err := purego.Dlsym(handle, name)
	if err != nil {
		return 0, fmt.Errorf("lookup ABI probe: %w", err)
	}

	var getABI func(*uint32) int32
	purego.RegisterFunc(&getABI, symbol)

	var abi uint32
	if rc := getABI(&abi); rc != 0 {
		return 0, fmt.Errorf("ABI probe returned %d", rc)
	}
	*trace = append(*trace, fmt.Sprintf("call:abi=0x%08x", abi))
	return abi, nil
}

func resolveMandatory(handle uintptr, row *result) (string, error) {
	for _, name := range mandatorySymbols {
		row.MandatoryLookups++
		row.Trace = append(row.Trace, "lookup:"+name)
		symbol, err := purego.Dlsym(handle, name)
		if err != nil {
			return name, err
		}

		var fn func() int32
		purego.RegisterFunc(&fn, symbol)
	}
	return "", nil
}

func classifyNaive(fixture string, handle uintptr) result {
	row := result{Fixture: fixture, Strategy: "naive", Trace: []string{}}
	if missing, err := resolveMandatory(handle, &row); err != nil {
		row.Outcome = "missing_symbol"
		row.Missing = missing
		return row
	}

	abi, err := probeABI(handle, &row.Trace)
	if err != nil {
		row.Outcome = "abi_probe_failed"
		return row
	}
	row.ABI = fmt.Sprintf("0x%08x", abi)
	if abi != expectedABI {
		row.Outcome = "abi_mismatch"
		return row
	}
	row.Outcome = "ok"
	return row
}

func classifyStaged(fixture string, handle uintptr) result {
	row := result{Fixture: fixture, Strategy: "staged", Trace: []string{}}
	abi, err := probeABI(handle, &row.Trace)
	if err != nil {
		row.Outcome = "abi_probe_failed"
		return row
	}
	row.ABI = fmt.Sprintf("0x%08x", abi)
	if abi != expectedABI {
		row.Outcome = "abi_mismatch"
		return row
	}

	if missing, err := resolveMandatory(handle, &row); err != nil {
		row.Outcome = "corrupt_abi12"
		row.Missing = missing
		return row
	}
	row.Outcome = "ok"
	return row
}

func runFixture(encoder *json.Encoder, fixture, path string) error {
	handle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return fmt.Errorf("open %s: %w", fixture, err)
	}
	defer purego.Dlclose(handle)

	for _, row := range []result{
		classifyNaive(fixture, handle),
		classifyStaged(fixture, handle),
	} {
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("encode %s/%s: %w", fixture, row.Strategy, err)
		}
	}
	return nil
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: %s ABI11 ABI12_COMPLETE ABI12_MISSING\n", os.Args[0])
		os.Exit(2)
	}

	fixtures := []struct {
		name string
		path string
	}{
		{name: "abi11", path: os.Args[1]},
		{name: "abi12_complete", path: os.Args[2]},
		{name: "abi12_missing", path: os.Args[3]},
	}

	encoder := json.NewEncoder(os.Stdout)
	for _, fixture := range fixtures {
		if err := runFixture(encoder, fixture.name, fixture.path); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
