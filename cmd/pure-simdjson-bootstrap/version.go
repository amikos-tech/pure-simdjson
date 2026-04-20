package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
	"github.com/spf13/cobra"
)

// newVersionCmd wires the `version` subcommand (D-27). It prints the library
// version, the Go runtime version, and — when available via
// runtime/debug.ReadBuildInfo — the module version of the CLI itself.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print library version, Go runtime version, and build info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion()
		},
	}
}

func runVersion() error {
	info, ok := debug.ReadBuildInfo()
	if ok {
		fmt.Fprintf(os.Stdout, "library:  %s\ngo:       %s\nmodule:   %s\n",
			bootstrap.Version, runtime.Version(), info.Main.Version)
	} else {
		fmt.Fprintf(os.Stdout, "library: %s\ngo:      %s\n",
			bootstrap.Version, runtime.Version())
	}
	return nil
}
