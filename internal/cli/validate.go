package cli

import (
	"fmt"

	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
	"github.com/spf13/cobra"
)

// NewValidateCmd creates the validate command
func NewValidateCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration without running migration",
		Long: `Validate the configuration file or command-line arguments without
actually performing the migration.

This is useful for checking your configuration before running a migration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg *config.Config
			var err error

			if configFile != "" {
				cfg, err = config.LoadFromFile(configFile)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
			} else {
				cfg = config.NewDefaultConfig()
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("validation failed:\n%w", err)
			}

			fmt.Println("✓ Configuration is valid")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")

	return cmd
}
