package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command
func NewRootCmd(version, buildTime string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "glue-to-ccsr",
		Short: "Migrate schemas from AWS Glue Schema Registry to Confluent Cloud Schema Registry",
		Long: `A high-performance, concurrent CLI tool for migrating schemas from 
AWS Glue Schema Registry to Confluent Cloud Schema Registry.

Features:
  - Multi-registry support with cross-registry reference handling
  - Concurrent processing with dependency-aware batching
  - LLM-powered subject naming (cloud and local models)
  - Schema preprocessing for efficient LLM token usage
  - Checkpointing and resume for large migrations
  - Comprehensive dry-run with detailed reports`,
		Version: version,
	}

	// Add subcommands
	rootCmd.AddCommand(NewMigrateCmd())
	rootCmd.AddCommand(NewValidateCmd())
	rootCmd.AddCommand(NewVersionCmd(version, buildTime))

	return rootCmd
}
