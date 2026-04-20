package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
	"github.com/spf13/cobra"
)

// newFetchCmd wires the `fetch` subcommand (D-24). Flags map 1:1 to
// bootstrap.BootstrapOption setters; this file owns no download logic itself.
func newFetchCmd() *cobra.Command {
	var (
		allPlatforms bool
		targets      []string
		dest         string
		version      string
		mirror       string
	)
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Download artifacts to cache or --dest",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch(cmd.Context(), allPlatforms, targets, dest, version, mirror, cmd.ErrOrStderr())
		},
	}
	cmd.Flags().BoolVar(&allPlatforms, "all-platforms", false, "fetch for all 5 supported platforms")
	cmd.Flags().StringArrayVar(&targets, "target", nil, "fetch for specific os/arch, e.g. linux/amd64 (repeatable)")
	cmd.Flags().StringVar(&dest, "dest", "", "destination directory (default: OS user cache)")
	cmd.Flags().StringVar(&version, "version", "", "library version (default: embedded Version constant)")
	cmd.Flags().StringVar(&mirror, "mirror", "", "override R2 base URL (same as PURE_SIMDJSON_BINARY_MIRROR)")
	return cmd
}

// runFetch selects the target platforms and drives BootstrapSync per platform.
// Per-platform progress is written to errW so --all-platforms never looks
// silently hung (L4 from 05-REVIEWS.md).
func runFetch(ctx context.Context, allPlatforms bool, targets []string, dest, version, mirror string, errW io.Writer) error {
	var platforms [][2]string

	switch {
	case allPlatforms:
		platforms = bootstrap.SupportedPlatforms
	case len(targets) > 0:
		for _, t := range targets {
			parts := strings.SplitN(t, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return fmt.Errorf("invalid target %q: expected os/arch format", t)
			}
			platforms = append(platforms, [2]string{parts[0], parts[1]})
		}
	default:
		// Empty set → BootstrapSync uses runtime.GOOS/GOARCH defaults.
	}

	buildOpts := func(goos, goarch string) []bootstrap.BootstrapOption {
		var opts []bootstrap.BootstrapOption
		if dest != "" {
			opts = append(opts, bootstrap.WithDest(dest))
		}
		if version != "" {
			opts = append(opts, bootstrap.WithVersion(version))
		}
		if mirror != "" {
			opts = append(opts, bootstrap.WithMirror(mirror))
		}
		if goos != "" {
			opts = append(opts, bootstrap.WithTarget(goos, goarch))
		}
		return opts
	}

	if len(platforms) == 0 {
		fmt.Fprintln(errW, "fetching current platform artifact...")
		return bootstrap.BootstrapSync(ctx, buildOpts("", "")...)
	}

	for _, p := range platforms {
		goos, goarch := p[0], p[1]
		// L4: per-platform progress so users perceive forward motion.
		fmt.Fprintf(errW, "fetching %s/%s...\n", goos, goarch)
		if err := bootstrap.BootstrapSync(ctx, buildOpts(goos, goarch)...); err != nil {
			return fmt.Errorf("fetch %s/%s: %w", goos, goarch, err)
		}
		fmt.Fprintf(errW, "  ok %s/%s\n", goos, goarch)
	}
	return nil
}
