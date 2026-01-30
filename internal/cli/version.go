package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCmd creates the version command
func NewVersionCmd(version, buildTime string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("glue-to-ccsr %s\n", version)
			fmt.Printf("Build time: %s\n", buildTime)
		},
	}
}
