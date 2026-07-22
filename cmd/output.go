package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func writeOutputPaths(cmd *cobra.Command, paths ...string) error {
	for _, path := range paths {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), path); err != nil {
			return fmt.Errorf("print output path: %w", err)
		}
	}
	return nil
}
