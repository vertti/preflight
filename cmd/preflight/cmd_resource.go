package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/resourcecheck"
)

var (
	resourceMinDisk   string
	resourceMinMemory string
	resourceMinCPUs   int
	resourcePath      string
)

var resourceCmd = &cobra.Command{
	Use:   "resource",
	Short: "Check system resources (disk, memory, CPU)",
	Long: `Check system resources meet minimum requirements.

Examples:
  preflight resource --min-disk 10G                    # At least 10GB free disk
  preflight resource --min-disk 10G --path /var/lib/docker  # Check specific path
  preflight resource --min-memory 2G                   # At least 2GB memory
  preflight resource --min-cpus 4                      # At least 4 CPU cores
  preflight resource --min-disk 10G --min-memory 2G   # Combined checks`,
	RunE: runResourceCheck,
}

func init() {
	resourceCmd.Flags().StringVar(&resourceMinDisk, "min-disk", "",
		"minimum free disk space (e.g., 10G, 500M)")
	resourceCmd.Flags().StringVar(&resourceMinMemory, "min-memory", "",
		"minimum available memory (e.g., 2G, 512M)")
	resourceCmd.Flags().IntVar(&resourceMinCPUs, "min-cpus", 0,
		"minimum number of CPU cores")
	resourceCmd.Flags().StringVar(&resourcePath, "path", "",
		"path for disk space check (default: current directory)")
	rootCmd.AddCommand(resourceCmd)
}

func runResourceCheck(_ *cobra.Command, _ []string) error {
	// Require at least one check flag
	if resourceMinDisk == "" && resourceMinMemory == "" && resourceMinCPUs == 0 {
		return fmt.Errorf("at least one check flag required: --min-disk, --min-memory, or --min-cpus")
	}

	c := &resourcecheck.Check{
		Path:    resourcePath,
		MinCPUs: resourceMinCPUs,
		Checker: &resourcecheck.RealResourceChecker{},
	}

	// Parse disk size
	if resourceMinDisk != "" {
		size, err := resourcecheck.ParseSize(resourceMinDisk)
		if err != nil {
			return fmt.Errorf("invalid --min-disk value: %w", err)
		}
		c.MinDisk = size
	}

	// Parse memory size
	if resourceMinMemory != "" {
		size, err := resourcecheck.ParseSize(resourceMinMemory)
		if err != nil {
			return fmt.Errorf("invalid --min-memory value: %w", err)
		}
		c.MinMemory = size
	}

	return runCheck(c)
}
