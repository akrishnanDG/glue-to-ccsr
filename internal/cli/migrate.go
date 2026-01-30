package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akrishnanDG/glue-to-ccsr/internal/migrator"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
	"github.com/spf13/cobra"
)

// NewMigrateCmd creates the migrate command
func NewMigrateCmd() *cobra.Command {
	cfg := config.NewDefaultConfig()
	var configFile string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate schemas from AWS Glue SR to Confluent Cloud SR",
		Long: `Migrate schemas from AWS Glue Schema Registry to Confluent Cloud Schema Registry.

For detailed configuration, use a config file (recommended):
  glue-to-ccsr migrate --config config.yaml --dry-run

Quick dry-run with CLI flags:
  glue-to-ccsr migrate \
    --aws-region us-east-2 \
    --aws-registry-name my-registry \
    --dry-run

Full migration with config file:
  glue-to-ccsr migrate --config config.yaml

See config.example.yaml for all available options.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config file if specified
			if configFile != "" {
				loadedCfg, err := config.LoadFromFile(configFile)
				if err != nil {
					return fmt.Errorf("failed to load config file: %w", err)
				}
				// Merge loaded config with CLI flags (CLI flags take precedence)
				cfg = mergeConfigs(loadedCfg, cfg, cmd)
			}
			return runMigrate(cmd.Context(), cfg)
		},
	}

	// Essential CLI flags only - use config file for detailed configuration
	flags := cmd.Flags()
	
	// Config file (primary way to configure)
	flags.StringVarP(&configFile, "config", "c", "", "Config file path (recommended)")
	
	// AWS Source
	flags.StringVar(&cfg.AWS.Region, "aws-region", cfg.AWS.Region, "AWS region")
	flags.StringVar(&cfg.AWS.Profile, "aws-profile", "", "AWS profile name")
	flags.StringVar(&cfg.AWS.AccessKeyID, "aws-access-key-id", "", "AWS access key ID")
	flags.StringVar(&cfg.AWS.SecretAccessKey, "aws-secret-access-key", "", "AWS secret access key")
	flags.StringSliceVar(&cfg.AWS.RegistryNames, "aws-registry-name", nil, "AWS Glue registry name (can be repeated)")
	flags.BoolVar(&cfg.AWS.RegistryAll, "aws-registry-all", false, "Migrate all registries")
	
	// Confluent Cloud Target
	flags.StringVar(&cfg.ConfluentCloud.URL, "cc-sr-url", "", "Confluent Cloud Schema Registry URL")
	flags.StringVar(&cfg.ConfluentCloud.APIKey, "cc-api-key", "", "Confluent Cloud API key")
	flags.StringVar(&cfg.ConfluentCloud.APISecret, "cc-api-secret", "", "Confluent Cloud API secret")
	
	// Common Options
	flags.BoolVar(&cfg.Output.DryRun, "dry-run", false, "Preview without making changes")
	flags.IntVar(&cfg.Concurrency.Workers, "workers", cfg.Concurrency.Workers, "Number of parallel workers")
	flags.StringVar(&cfg.Output.LogLevel, "log-level", cfg.Output.LogLevel, "Log level: debug, info, warn, error")

	// Note: Confluent Cloud flags are not marked as required here because they're optional for --dry-run
	// Validation happens in the config.Validate() method based on dry-run mode

	return cmd
}

// mergeConfigs merges loaded config with CLI flags, giving precedence to CLI flags
func mergeConfigs(fileConfig, cliConfig *config.Config, cmd *cobra.Command) *config.Config {
	merged := fileConfig
	
	// Override with CLI flags if they were explicitly set
	flags := cmd.Flags()
	
	// AWS config
	if flags.Changed("aws-region") {
		merged.AWS.Region = cliConfig.AWS.Region
	}
	if flags.Changed("aws-profile") {
		merged.AWS.Profile = cliConfig.AWS.Profile
	}
	if flags.Changed("aws-access-key-id") {
		merged.AWS.AccessKeyID = cliConfig.AWS.AccessKeyID
	}
	if flags.Changed("aws-secret-access-key") {
		merged.AWS.SecretAccessKey = cliConfig.AWS.SecretAccessKey
	}
	if flags.Changed("aws-registry-name") {
		merged.AWS.RegistryNames = cliConfig.AWS.RegistryNames
	}
	if flags.Changed("aws-registry-all") {
		merged.AWS.RegistryAll = cliConfig.AWS.RegistryAll
	}
	
	// Confluent Cloud config
	if flags.Changed("cc-sr-url") {
		merged.ConfluentCloud.URL = cliConfig.ConfluentCloud.URL
	}
	if flags.Changed("cc-api-key") {
		merged.ConfluentCloud.APIKey = cliConfig.ConfluentCloud.APIKey
	}
	if flags.Changed("cc-api-secret") {
		merged.ConfluentCloud.APISecret = cliConfig.ConfluentCloud.APISecret
	}
	
	// Common options
	if flags.Changed("workers") {
		merged.Concurrency.Workers = cliConfig.Concurrency.Workers
	}
	if flags.Changed("dry-run") {
		merged.Output.DryRun = cliConfig.Output.DryRun
	}
	if flags.Changed("log-level") {
		merged.Output.LogLevel = cliConfig.Output.LogLevel
	}
	
	return merged
}

func runMigrate(ctx context.Context, cfg *config.Config) error {
	// Load API keys from environment if not provided
	if cfg.ConfluentCloud.APIKey == "" {
		cfg.ConfluentCloud.APIKey = os.Getenv("CC_API_KEY")
	}
	if cfg.ConfluentCloud.APISecret == "" {
		cfg.ConfluentCloud.APISecret = os.Getenv("CC_API_SECRET")
	}
	if cfg.LLM.APIKey == "" {
		switch cfg.LLM.Provider {
		case "openai":
			cfg.LLM.APIKey = os.Getenv("OPENAI_API_KEY")
		case "anthropic":
			cfg.LLM.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Create and run migrator
	m, err := migrator.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	startTime := time.Now()
	result, err := m.Run(ctx)
	duration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Print summary
	printMigrationSummary(result, duration, cfg.Output.DryRun)

	// Return error if any schemas failed (unless dry-run)
	if !cfg.Output.DryRun && result.Failed > 0 {
		return fmt.Errorf("migration completed with %d failures", result.Failed)
	}

	return nil
}

func printMigrationSummary(result *migrator.Result, duration time.Duration, dryRun bool) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	if dryRun {
		fmt.Println("                     DRY RUN COMPLETE")
	} else {
		if result.Failed > 0 {
			fmt.Println("              MIGRATION COMPLETED WITH ERRORS")
		} else {
			fmt.Println("                MIGRATION COMPLETED SUCCESSFULLY")
		}
	}
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("  Duration:        %s\n", duration.Round(time.Second))
	fmt.Printf("  Registries:      %d\n", result.RegistriesProcessed)
	fmt.Printf("  Schemas:         %d\n", result.SchemasProcessed)
	fmt.Printf("  Versions:        %d\n", result.VersionsProcessed)
	fmt.Printf("  Successful:      %d\n", result.Successful)
	if result.Failed > 0 {
		fmt.Printf("  Failed:          %d [ERROR]\n", result.Failed)
	} else {
		fmt.Printf("  Failed:          %d\n", result.Failed)
	}
	fmt.Printf("  Skipped:         %d\n", result.Skipped)
	if result.LLMCalls > 0 {
		fmt.Printf("  LLM Calls:       %d (cost: $%.2f)\n", result.LLMCalls, result.LLMCost)
	}
	fmt.Println("═══════════════════════════════════════════════════════════════")
	
	// Print errors if any
	if result.Failed > 0 && len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println("ERRORS:")
		fmt.Println("───────")
		for i, err := range result.Errors {
			if err != nil {
				fmt.Printf("  %d. %v\n", i+1, err)
			}
		}
		fmt.Println()
	}
}
