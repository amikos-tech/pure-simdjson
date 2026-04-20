// Command pure-simdjson-bootstrap is a thin CLI wrapper around internal/bootstrap.
// It exposes four verbs — fetch, verify, platforms, version — and owns no
// domain logic of its own (D-22/D-23). All errors go to stderr, normal output
// to stdout, and a non-zero exit on failure (D-28).
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "pure-simdjson-bootstrap",
		Short:         "Bootstrap pure-simdjson shared library artifacts",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.AddCommand(
		newFetchCmd(),
		newVerifyCmd(),
		newPlatformsCmd(),
		newVersionCmd(),
	)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
