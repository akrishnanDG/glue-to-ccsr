package extractor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/glue/types"
	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/time/rate"
)

// GlueExtractor extracts schemas from AWS Glue Schema Registry
type GlueExtractor struct {
	client      *glue.Client
	config      *config.Config
	rateLimiter *rate.Limiter
}

// New creates a new GlueExtractor
func New(cfg *config.Config) (*GlueExtractor, error) {
	// Load AWS configuration
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWS.Region),
	}

	// Use explicit credentials if provided
	if cfg.AWS.AccessKeyID != "" && cfg.AWS.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     cfg.AWS.AccessKeyID,
					SecretAccessKey: cfg.AWS.SecretAccessKey,
				}, nil
			}),
		))
	} else if cfg.AWS.Profile != "" {
		// Use named profile
		opts = append(opts, awsconfig.WithSharedConfigProfile(cfg.AWS.Profile))
	}
	// Otherwise, use default credential chain (env vars, ~/.aws/credentials, etc.)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := glue.NewFromConfig(awsCfg)

	// Create rate limiter
	limiter := rate.NewLimiter(rate.Limit(cfg.Concurrency.AWSRateLimit), 1)

	return &GlueExtractor{
		client:      client,
		config:      cfg,
		rateLimiter: limiter,
	}, nil
}

// ExtractAll extracts all schemas from all specified registries
func (e *GlueExtractor) ExtractAll(ctx context.Context) ([]*models.GlueSchema, error) {
	// Get list of registries to process
	registries, err := e.getRegistries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get registries: %w", err)
	}

	var allSchemas []*models.GlueSchema

	for _, registry := range registries {
		schemas, err := e.extractRegistrySchemas(ctx, registry.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to extract schemas from registry %s: %w", registry.Name, err)
		}
		allSchemas = append(allSchemas, schemas...)
	}

	return allSchemas, nil
}

// GetSchema gets a single schema with all its versions
func (e *GlueExtractor) GetSchema(ctx context.Context, registryName, schemaName string) (*models.GlueSchema, error) {
	// Wait for rate limiter
	if err := e.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Get schema metadata
	schemaInput := &glue.GetSchemaInput{
		SchemaId: &types.SchemaId{
			RegistryName: aws.String(registryName),
			SchemaName:   aws.String(schemaName),
		},
	}

	schemaResp, err := e.client.GetSchema(ctx, schemaInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	schema := &models.GlueSchema{
		Name:          aws.ToString(schemaResp.SchemaName),
		RegistryName:  registryName,
		ARN:           aws.ToString(schemaResp.SchemaArn),
		Description:   aws.ToString(schemaResp.Description),
		DataFormat:    models.SchemaType(schemaResp.DataFormat),
		Compatibility: string(schemaResp.Compatibility),
		LatestVersion: aws.ToInt64(schemaResp.LatestSchemaVersion),
	}

	if schemaResp.CreatedTime != nil {
		schema.CreatedTime = parseTimestamp(aws.ToString(schemaResp.CreatedTime))
	}
	if schemaResp.UpdatedTime != nil {
		schema.UpdatedTime = parseTimestamp(aws.ToString(schemaResp.UpdatedTime))
	}

	// Get all versions
	versions, err := e.getSchemaVersions(ctx, registryName, schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema versions: %w", err)
	}
	schema.Versions = versions

	return schema, nil
}

func (e *GlueExtractor) getRegistries(ctx context.Context) ([]*models.GlueRegistry, error) {
	var registries []*models.GlueRegistry

	if e.config.AWS.RegistryAll {
		// List all registries
		registries, err := e.listAllRegistries(ctx)
		if err != nil {
			return nil, err
		}

		// Filter out excluded registries
		var filtered []*models.GlueRegistry
		for _, reg := range registries {
			if !e.isExcluded(reg.Name) {
				filtered = append(filtered, reg)
			}
		}
		return filtered, nil
	}

	// Use specified registry names
	for _, name := range e.config.AWS.RegistryNames {
		reg, err := e.getRegistry(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get registry %s: %w", name, err)
		}
		registries = append(registries, reg)
	}

	return registries, nil
}

func (e *GlueExtractor) listAllRegistries(ctx context.Context) ([]*models.GlueRegistry, error) {
	var registries []*models.GlueRegistry
	var nextToken *string

	for {
		if err := e.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		input := &glue.ListRegistriesInput{
			NextToken: nextToken,
		}

		resp, err := e.client.ListRegistries(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list registries: %w", err)
		}

		for _, reg := range resp.Registries {
			registries = append(registries, &models.GlueRegistry{
				Name:        aws.ToString(reg.RegistryName),
				ARN:         aws.ToString(reg.RegistryArn),
				Description: aws.ToString(reg.Description),
			})
		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	return registries, nil
}

func (e *GlueExtractor) getRegistry(ctx context.Context, name string) (*models.GlueRegistry, error) {
	if err := e.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	input := &glue.GetRegistryInput{
		RegistryId: &types.RegistryId{
			RegistryName: aws.String(name),
		},
	}

	resp, err := e.client.GetRegistry(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}

	registry := &models.GlueRegistry{
		Name:        aws.ToString(resp.RegistryName),
		ARN:         aws.ToString(resp.RegistryArn),
		Description: aws.ToString(resp.Description),
	}

	if resp.CreatedTime != nil {
		registry.CreatedTime = parseTimestamp(aws.ToString(resp.CreatedTime))
	}
	if resp.UpdatedTime != nil {
		registry.UpdatedTime = parseTimestamp(aws.ToString(resp.UpdatedTime))
	}

	return registry, nil
}

func (e *GlueExtractor) extractRegistrySchemas(ctx context.Context, registryName string) ([]*models.GlueSchema, error) {
	// First, collect all schema names
	var schemaNames []string
	var nextToken *string

	for {
		if err := e.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		input := &glue.ListSchemasInput{
			RegistryId: &types.RegistryId{
				RegistryName: aws.String(registryName),
			},
			NextToken: nextToken,
		}

		resp, err := e.client.ListSchemas(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list schemas: %w", err)
		}

		for _, s := range resp.Schemas {
			schemaName := aws.ToString(s.SchemaName)

			// Apply schema filter if specified
			if e.config.AWS.SchemaFilter != "" {
				matched, err := filepath.Match(e.config.AWS.SchemaFilter, schemaName)
				if err != nil {
					return nil, fmt.Errorf("invalid schema filter pattern: %w", err)
				}
				if !matched {
					continue
				}
			}

			schemaNames = append(schemaNames, schemaName)
		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	// Create progress bar for schema extraction
	bar := progressbar.NewOptions(len(schemaNames),
		progressbar.OptionSetDescription("      Fetching schemas"),
		progressbar.OptionSetWidth(50),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	// Now fetch all schemas in parallel using worker pool
	schemas, err := e.fetchSchemasParallel(ctx, registryName, schemaNames, bar)
	bar.Finish()
	fmt.Println()
	return schemas, err
}

func (e *GlueExtractor) getSchemaVersions(ctx context.Context, registryName, schemaName string) ([]models.GlueSchemaVersion, error) {
	// First, collect all version numbers
	var versionNumbers []int64
	var nextToken *string

	for {
		if err := e.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		input := &glue.ListSchemaVersionsInput{
			SchemaId: &types.SchemaId{
				RegistryName: aws.String(registryName),
				SchemaName:   aws.String(schemaName),
			},
			NextToken: nextToken,
		}

		resp, err := e.client.ListSchemaVersions(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list schema versions: %w", err)
		}

		for _, v := range resp.Schemas {
			versionNumbers = append(versionNumbers, aws.ToInt64(v.VersionNumber))
		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	// Fetch all versions in parallel
	versions, err := e.fetchVersionsParallel(ctx, registryName, schemaName, versionNumbers)
	if err != nil {
		return nil, err
	}

	// Sort versions by version number
	sortVersions(versions)

	return versions, nil
}

// fetchSchemasParallel fetches multiple schemas in parallel using worker pool
func (e *GlueExtractor) fetchSchemasParallel(ctx context.Context, registryName string, schemaNames []string, bar *progressbar.ProgressBar) ([]*models.GlueSchema, error) {
	numWorkers := e.config.Concurrency.Workers
	if numWorkers <= 0 {
		numWorkers = 10
	}

	// Channels for work distribution
	jobs := make(chan string, len(schemaNames))
	results := make(chan *models.GlueSchema, len(schemaNames))
	errors := make(chan error, len(schemaNames))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for schemaName := range jobs {
				schema, err := e.GetSchema(ctx, registryName, schemaName)
				if err != nil {
					errors <- fmt.Errorf("failed to get schema %s: %w", schemaName, err)
					return
				}
				results <- schema
			}
		}()
	}

	// Send jobs
	for _, schemaName := range schemaNames {
		jobs <- schemaName
	}
	close(jobs)

	// Wait for completion
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	var schemas []*models.GlueSchema
	for i := 0; i < len(schemaNames); i++ {
		select {
		case err := <-errors:
			if err != nil {
				return nil, err
			}
		case schema := <-results:
			schemas = append(schemas, schema)
			bar.Add(1)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return schemas, nil
}

// fetchVersionsParallel fetches multiple schema versions in parallel
func (e *GlueExtractor) fetchVersionsParallel(ctx context.Context, registryName, schemaName string, versionNumbers []int64) ([]models.GlueSchemaVersion, error) {
	numWorkers := e.config.Concurrency.Workers
	if numWorkers <= 0 {
		numWorkers = 10
	}

	// Channels for work distribution
	jobs := make(chan int64, len(versionNumbers))
	results := make(chan models.GlueSchemaVersion, len(versionNumbers))
	errors := make(chan error, len(versionNumbers))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for versionNumber := range jobs {
				// Rate limit
				if err := e.rateLimiter.Wait(ctx); err != nil {
					errors <- err
					return
				}

				versionInput := &glue.GetSchemaVersionInput{
					SchemaId: &types.SchemaId{
						RegistryName: aws.String(registryName),
						SchemaName:   aws.String(schemaName),
					},
					SchemaVersionNumber: &types.SchemaVersionNumber{
						VersionNumber: aws.Int64(versionNumber),
					},
				}

				versionResp, err := e.client.GetSchemaVersion(ctx, versionInput)
				if err != nil {
					errors <- fmt.Errorf("failed to get schema version %d: %w", versionNumber, err)
					return
				}

				version := models.GlueSchemaVersion{
					VersionNumber:   versionNumber,
					SchemaVersionID: aws.ToString(versionResp.SchemaVersionId),
					Definition:      aws.ToString(versionResp.SchemaDefinition),
					Status:          string(versionResp.Status),
				}

				if versionResp.CreatedTime != nil {
					version.CreatedTime = parseTimestamp(aws.ToString(versionResp.CreatedTime))
				}

				results <- version
			}
		}()
	}

	// Send jobs
	for _, versionNumber := range versionNumbers {
		jobs <- versionNumber
	}
	close(jobs)

	// Wait for completion
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	var versions []models.GlueSchemaVersion
	for i := 0; i < len(versionNumbers); i++ {
		select {
		case err := <-errors:
			if err != nil {
				return nil, err
			}
		case version := <-results:
			versions = append(versions, version)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return versions, nil
}

func (e *GlueExtractor) isExcluded(name string) bool {
	for _, pattern := range e.config.AWS.RegistryExclude {
		// Support glob patterns
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
		// Also support simple prefix matching for patterns like "test-*"
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(name, prefix) {
				return true
			}
		}
	}
	return false
}

func sortVersions(versions []models.GlueSchemaVersion) {
	// Simple bubble sort - versions are typically small arrays
	for i := 0; i < len(versions); i++ {
		for j := i + 1; j < len(versions); j++ {
			if versions[i].VersionNumber > versions[j].VersionNumber {
				versions[i], versions[j] = versions[j], versions[i]
			}
		}
	}
}

// parseTimestamp parses AWS timestamp string to time.Time
func parseTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}
	// AWS timestamps are typically in RFC3339 or ISO8601 format
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		// Try parsing as Unix timestamp in string form
		return time.Time{}
	}
	return t
}
