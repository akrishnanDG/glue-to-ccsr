package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/glue/types"
)

type SchemaConfig struct {
	Name          string   `json:"name"`
	Compatibility string   `json:"compatibility"` // BACKWARD, FORWARD, FULL, NONE
	Versions      []string `json:"versions"`
}

type RegistrationConfig struct {
	RegistryName string         `json:"registry_name"`
	Region       string         `json:"region"`
	Schemas      []SchemaConfig `json:"schemas"`
}

func main() {
	var (
		configFile    = flag.String("config", "schema-config.json", "Schema configuration file")
		credFile      = flag.String("creds", "aws-creds.json", "AWS credentials file")
		dryRun        = flag.Bool("dry-run", false, "Preview without registering")
		registryName  = flag.String("registry", "payments-regsitry", "Registry name")
		region        = flag.String("region", "us-east-2", "AWS region")
	)
	flag.Parse()

	ctx := context.Background()

	// Load credentials
	creds, err := loadCredentials(*credFile)
	if err != nil {
		log.Fatalf("Failed to load credentials: %v", err)
	}

	// Create AWS config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(*region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			creds.AccessKeyID,
			creds.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	client := glue.NewFromConfig(cfg)

	// Load schema configuration
	schemaConfig, err := loadSchemaConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load schema config: %v", err)
	}

	if *dryRun {
		fmt.Println("🔍 DRY RUN MODE - No schemas will be registered")
		fmt.Println("=" + string(make([]byte, 60)))
	}

	fmt.Printf("Registry: %s\n", *registryName)
	fmt.Printf("Region: %s\n", *region)
	fmt.Printf("Total Schemas: %d\n\n", len(schemaConfig.Schemas))

	totalVersions := 0
	successCount := 0
	errorCount := 0

	for i, schema := range schemaConfig.Schemas {
		fmt.Printf("[%d/%d] Processing: %s (Compatibility: %s, Versions: %d)\n",
			i+1, len(schemaConfig.Schemas), schema.Name, schema.Compatibility, len(schema.Versions))

		if *dryRun {
			fmt.Printf("  ✓ Would create schema with %d versions\n", len(schema.Versions))
			totalVersions += len(schema.Versions)
			successCount++
			continue
		}

		// Register schema and versions
		err := registerSchema(ctx, client, *registryName, schema)
		if err != nil {
			fmt.Printf("  ✗ Error: %v\n", err)
			errorCount++
			continue
		}

		fmt.Printf("  ✓ Successfully registered %d versions\n", len(schema.Versions))
		totalVersions += len(schema.Versions)
		successCount++

		// Rate limit to avoid throttling
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println("\n" + string(make([]byte, 60)))
	fmt.Println("SUMMARY")
	fmt.Println(string(make([]byte, 60)))
	fmt.Printf("Total Schemas:    %d\n", len(schemaConfig.Schemas))
	fmt.Printf("Total Versions:   %d\n", totalVersions)
	fmt.Printf("Successful:       %d ✓\n", successCount)
	fmt.Printf("Failed:           %d ✗\n", errorCount)

	if *dryRun {
		fmt.Println("\nRun without --dry-run to register schemas")
	}
}

func registerSchema(ctx context.Context, client *glue.Client, registryName string, schema SchemaConfig) error {
	// First, try to get the schema (it might already exist)
	_, err := client.GetSchema(ctx, &glue.GetSchemaInput{
		SchemaId: &types.SchemaId{
			RegistryName: aws.String(registryName),
			SchemaName:   aws.String(schema.Name),
		},
	})

	// If schema doesn't exist, create it with first version
	if err != nil {
		_, err = client.CreateSchema(ctx, &glue.CreateSchemaInput{
			SchemaName: aws.String(schema.Name),
			RegistryId: &types.RegistryId{
				RegistryName: aws.String(registryName),
			},
			DataFormat:    types.DataFormatAvro,
			Compatibility: types.Compatibility(schema.Compatibility),
			SchemaDefinition: aws.String(schema.Versions[0]),
		})
		if err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}

		// Register remaining versions
		for i := 1; i < len(schema.Versions); i++ {
			_, err = client.RegisterSchemaVersion(ctx, &glue.RegisterSchemaVersionInput{
				SchemaId: &types.SchemaId{
					RegistryName: aws.String(registryName),
					SchemaName:   aws.String(schema.Name),
				},
				SchemaDefinition: aws.String(schema.Versions[i]),
			})
			if err != nil {
				return fmt.Errorf("failed to register version %d: %w", i+1, err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	} else {
		// Schema exists, just register new versions
		for i, version := range schema.Versions {
			_, err = client.RegisterSchemaVersion(ctx, &glue.RegisterSchemaVersionInput{
				SchemaId: &types.SchemaId{
					RegistryName: aws.String(registryName),
					SchemaName:   aws.String(schema.Name),
				},
				SchemaDefinition: aws.String(version),
			})
			if err != nil {
				// Version might already exist, log but continue
				fmt.Printf("  ⚠ Version %d: %v\n", i+1, err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

type AWSCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

func loadCredentials(path string) (*AWSCredentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var creds AWSCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

func loadSchemaConfig(path string) (*RegistrationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg RegistrationConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
