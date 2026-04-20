package main

import (
	"fmt"
	"os"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
	"github.com/spf13/cobra"
)

// newPlatformsCmd wires the `platforms` subcommand (D-26). It lists the five
// supported targets and indicates whether each platform's artifact is
// currently present in the local cache.
func newPlatformsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "platforms",
		Short: "List supported platforms and local cache presence",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlatforms()
		},
	}
}

func runPlatforms() error {
	for _, p := range bootstrap.SupportedPlatforms {
		goos, goarch := p[0], p[1]
		cachePath := bootstrap.CachePath(goos, goarch)
		indicator := "missing"
		if _, err := os.Stat(cachePath); err == nil {
			indicator = "cached"
		}
		fmt.Printf("%-20s %s\n", goos+"/"+goarch, indicator)
	}
	return nil
}
