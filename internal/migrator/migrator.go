package migrator

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/akrishnanDG/glue-to-ccsr/internal/extractor"
	"github.com/akrishnanDG/glue-to-ccsr/internal/graph"
	"github.com/akrishnanDG/glue-to-ccsr/internal/keyvalue"
	"github.com/akrishnanDG/glue-to-ccsr/internal/llm"
	"github.com/akrishnanDG/glue-to-ccsr/internal/loader"
	"github.com/akrishnanDG/glue-to-ccsr/internal/mapper"
	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/internal/normalizer"
	"github.com/akrishnanDG/glue-to-ccsr/internal/validator"
	"github.com/akrishnanDG/glue-to-ccsr/internal/worker"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
	"github.com/schollz/progressbar/v3"
)

// Result represents the result of a migration
type Result struct {
	RegistriesProcessed int
	SchemasProcessed    int
	VersionsProcessed   int
	Successful          int
	Failed              int
	Skipped             int
	LLMCalls            int
	LLMCost             float64
	Errors              []error
	Report              *models.MigrationReport
}

// Migrator orchestrates the migration process
type Migrator struct {
	config      *config.Config
	extractor   *extractor.GlueExtractor
	loader      *loader.ConfluentLoader
	mapper      *mapper.NomenclatureMapper
	normalizer  *normalizer.Normalizer
	kvDetector  *keyvalue.Detector
	llmNamer    *llm.Namer
	validator   *validator.Validator
	workerPool  *worker.Pool
	checkpoint  *worker.CheckpointManager
}

// New creates a new Migrator
func New(cfg *config.Config) (*Migrator, error) {
	// Create extractor
	ext, err := extractor.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create extractor: %w", err)
	}

	// Create loader
	ldr, err := loader.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create loader: %w", err)
	}

	// Create normalizer
	norm := normalizer.New(cfg)

	// Create key/value detector
	kvDet, err := keyvalue.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create key/value detector: %w", err)
	}

	// Create LLM namer if using LLM strategy
	var llmNmr *llm.Namer
	if cfg.Naming.SubjectStrategy == "llm" {
		llmNmr, err = llm.NewNamer(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create LLM namer: %w", err)
		}
	}

	// Create mapper
	mpr := mapper.New(cfg, norm, kvDet, llmNmr)

	// Create validator
	val := validator.New(cfg)

	// Create worker pool
	pool := worker.NewPool(cfg)

	// Create checkpoint manager if checkpoint file specified
	var chkpt *worker.CheckpointManager
	if cfg.Checkpoint.File != "" {
		chkpt = worker.NewCheckpointManager(cfg.Checkpoint.File)
	}

	return &Migrator{
		config:     cfg,
		extractor:  ext,
		loader:     ldr,
		mapper:     mpr,
		normalizer: norm,
		kvDetector: kvDet,
		llmNamer:   llmNmr,
		validator:  val,
		workerPool: pool,
		checkpoint: chkpt,
	}, nil
}

// Run executes the migration
func (m *Migrator) Run(ctx context.Context) (*Result, error) {
	startTime := time.Now()
	result := &Result{}

	// Step 1: Extract schemas from AWS Glue
	fmt.Println("[1/5] Extracting schemas from AWS Glue Schema Registry...")
	schemas, err := m.extractor.ExtractAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schemas: %w", err)
	}
	fmt.Printf("      Found %d schemas across %d registries\n", len(schemas), m.countRegistries(schemas))

	// Step 2: Build dependency graph
	fmt.Println("\n[2/5] Building dependency graph...")
	depGraph, err := graph.Build(schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}
	levels := depGraph.GetLevels()
	fmt.Printf("      Created %d dependency levels\n", len(levels))

	// Step 3: Generate mappings
	fmt.Println("\n[3/5] Generating schema mappings...")
	mappings, err := m.mapper.MapAll(ctx, schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to generate mappings: %w", err)
	}
	
	// Create a lookup map for the complete mappings
	mappingLookup := make(map[string]*models.SchemaMapping)
	for i := range mappings {
		key := fmt.Sprintf("%s:%s", mappings[i].SourceRegistry, mappings[i].SourceSchemaName)
		mappingLookup[key] = mappings[i]
	}
	
	// Replace the incomplete mappings in the levels with the complete ones from the mapper
	for i := range levels {
		for j := range levels[i].Schemas {
			key := fmt.Sprintf("%s:%s", levels[i].Schemas[j].SourceRegistry, levels[i].Schemas[j].SourceSchemaName)
			if completeMapping, found := mappingLookup[key]; found {
				// Copy the target fields from the complete mapping
				levels[i].Schemas[j].TargetContext = completeMapping.TargetContext
				levels[i].Schemas[j].TargetSubject = completeMapping.TargetSubject
				levels[i].Schemas[j].DetectedRole = completeMapping.DetectedRole
				levels[i].Schemas[j].NamingStrategy = completeMapping.NamingStrategy
				levels[i].Schemas[j].NamingReason = completeMapping.NamingReason
				levels[i].Schemas[j].Transformations = completeMapping.Transformations
				levels[i].Schemas[j].Status = completeMapping.Status
				levels[i].Schemas[j].Error = completeMapping.Error
			}
		}
	}

	// Step 3.5: Auto-resolve collisions if enabled
	if m.config.Normalization.CollisionCheck && m.config.Normalization.CollisionResolution != "" && m.config.Normalization.CollisionResolution != "fail" {
		collisions := m.normalizer.DetectCollisions(mappings)
		if len(collisions) > 0 {
			fmt.Printf("      Found %d naming collisions, applying '%s' resolution strategy...\n", len(collisions), m.config.Normalization.CollisionResolution)
			mappings = m.normalizer.ResolveCollisions(mappings)
			
			// Update the mappings in the levels after collision resolution
			for i := range mappings {
				key := fmt.Sprintf("%s:%s", mappings[i].SourceRegistry, mappings[i].SourceSchemaName)
				mappingLookup[key] = mappings[i]
			}
			for i := range levels {
				for j := range levels[i].Schemas {
					key := fmt.Sprintf("%s:%s", levels[i].Schemas[j].SourceRegistry, levels[i].Schemas[j].SourceSchemaName)
					if completeMapping, found := mappingLookup[key]; found {
						levels[i].Schemas[j].TargetContext = completeMapping.TargetContext
						levels[i].Schemas[j].TargetSubject = completeMapping.TargetSubject
						levels[i].Schemas[j].Transformations = completeMapping.Transformations
					}
				}
			}
		}
	}

	// Step 4: Validate mappings
	fmt.Println("[4/5] Validating mappings...")
	validationResult := m.validator.ValidateAll(mappings)
	if validationResult.HasErrors() {
		for _, e := range validationResult.Errors {
			fmt.Printf("      ERROR: %s: %s\n", e.Schema, e.Message)
		}
		if m.config.Output.DryRun {
			// In dry-run mode, continue to show the report
		} else {
			return nil, fmt.Errorf("validation failed with %d errors", len(validationResult.Errors))
		}
	}

	// Check for collisions
	if m.config.Normalization.CollisionCheck {
		collisions := m.normalizer.DetectCollisions(mappings)
		if len(collisions) > 0 {
			fmt.Println("      WARNING: Naming collisions detected:")
			for _, c := range collisions {
				fmt.Printf("         %v -> %s\n", c.SourceSchemas, c.NormalizedName)
			}
		}
	}

	// Step 5: Create migration plan
	plan := m.createPlan(schemas, mappings, levels)
	result.RegistriesProcessed = len(plan.SourceRegistries)
	result.SchemasProcessed = plan.TotalSchemas
	result.VersionsProcessed = plan.TotalVersions

	// If dry-run, print report and return
	if m.config.Output.DryRun {
		fmt.Println("\n[5/5] DRY RUN - No changes will be made\n")
		m.printDryRunReport(plan)
		result.Report = m.generateReport(plan, startTime, true)
		return result, nil
	}

	// Step 6: Execute migration
	fmt.Println("\n[5/5] Executing migration...")
	
	// Resume from checkpoint if specified
	var state *models.MigrationState
	if m.checkpoint != nil && m.config.Checkpoint.Resume {
		state, err = m.checkpoint.Load()
		if err != nil {
			fmt.Printf("      WARNING: Could not load checkpoint: %v, starting fresh\n", err)
			state = models.NewMigrationState("")
		} else {
			fmt.Printf("      Resuming from checkpoint (%d/%d completed)\n", state.CompletedCount, state.TotalSchemas)
		}
	} else {
		state = models.NewMigrationState("")
	}
	state.TotalSchemas = len(mappings)
	state.MigrationOrder = getMigrationOrder(levels)

	// Migrate level by level
	for _, level := range levels {
		fmt.Printf("   Processing dependency level %d (%d schemas)...\n", level.Level, len(level.Schemas))
		
		levelResult, err := m.migrateLevel(ctx, level, state)
		if err != nil {
			return nil, fmt.Errorf("failed at level %d: %w", level.Level, err)
		}

		result.Successful += levelResult.Successful
		result.Failed += levelResult.Failed
		result.Skipped += levelResult.Skipped
		result.Errors = append(result.Errors, levelResult.Errors...)

		// Save checkpoint after each level
		if m.checkpoint != nil {
			if err := m.checkpoint.Save(state); err != nil {
				fmt.Printf("      WARNING: Failed to save checkpoint: %v\n", err)
			}
		}
	}

	// Update LLM stats if used
	if m.llmNamer != nil {
		result.LLMCalls = m.llmNamer.GetCallCount()
		result.LLMCost = m.llmNamer.GetTotalCost()
	}

	result.Report = m.generateReport(plan, startTime, false)

	return result, nil
}

func (m *Migrator) countRegistries(schemas []*models.GlueSchema) int {
	registries := make(map[string]bool)
	for _, s := range schemas {
		registries[s.RegistryName] = true
	}
	return len(registries)
}

func (m *Migrator) createPlan(schemas []*models.GlueSchema, mappings []*models.SchemaMapping, levels []graph.Level) *models.MigrationPlan {
	plan := &models.MigrationPlan{
		SourceRegistries: m.getRegistryNames(schemas),
		TotalSchemas:     len(schemas),
	}

	// Count versions and references
	for _, s := range schemas {
		plan.TotalVersions += len(s.Versions)
	}

	for _, mapping := range mappings {
		plan.TotalReferences += len(mapping.References)
		plan.Mappings = append(plan.Mappings, *mapping)
	}

	// Convert levels
	for _, l := range levels {
		plan.Levels = append(plan.Levels, models.DependencyLevel{
			Level:   l.Level,
			Schemas: l.Schemas,
		})
	}

	// Calculate summary
	plan.Summary = m.calculateSummary(plan)

	return plan
}

func (m *Migrator) getRegistryNames(schemas []*models.GlueSchema) []string {
	registryMap := make(map[string]bool)
	for _, s := range schemas {
		registryMap[s.RegistryName] = true
	}
	
	var names []string
	for name := range registryMap {
		names = append(names, name)
	}
	return names
}

func (m *Migrator) calculateSummary(plan *models.MigrationPlan) models.MigrationSummary {
	summary := models.MigrationSummary{
		Registries: len(plan.SourceRegistries),
		Schemas:    plan.TotalSchemas,
		Versions:   plan.TotalVersions,
		References: plan.TotalReferences,
	}

	for _, mapping := range plan.Mappings {
		switch mapping.Status {
		case models.MappingStatusReady:
			summary.Ready++
		case models.MappingStatusWarning:
			summary.Warnings++
		case models.MappingStatusError:
			summary.Errors++
		}
	}

	summary.Collisions = len(plan.Collisions)

	return summary
}

type levelResult struct {
	Successful int
	Failed     int
	Skipped    int
	Errors     []error
}

func (m *Migrator) migrateLevel(ctx context.Context, level graph.Level, state *models.MigrationState) (*levelResult, error) {
	result := &levelResult{}

	// Filter schemas that need to be migrated
	var toMigrate []models.SchemaMapping
	for _, mapping := range level.Schemas {
		key := fmt.Sprintf("%s:%s", mapping.SourceRegistry, mapping.SourceSchemaName)
		if _, completed := state.CompletedSchemas[key]; completed {
			result.Skipped++
			continue
		}
		if mapping.Status == models.MappingStatusError {
			result.Skipped++
			continue
		}
		toMigrate = append(toMigrate, mapping)
	}

	if len(toMigrate) == 0 {
		return result, nil
	}

	// Create progress bar for schema registration
	bar := progressbar.NewOptions(len(toMigrate),
		progressbar.OptionSetDescription("      Registering schemas"),
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

	// Track progress with atomic counter
	var completed int64
	progressCallback := func() {
		atomic.AddInt64(&completed, 1)
		bar.Add(1)
	}

	// Execute migrations using worker pool with progress
	errors := m.workerPool.ExecuteWithProgress(ctx, toMigrate, func(ctx context.Context, mapping models.SchemaMapping) error {
		return m.migrateSchema(ctx, &mapping, state)
	}, progressCallback)

	bar.Finish()
	fmt.Println()

	// Collect results and print errors immediately
	for i, err := range errors {
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err)
			// Print error immediately so user sees what's failing
			if i < len(toMigrate) {
				schemaKey := fmt.Sprintf("%s.%s", toMigrate[i].SourceRegistry, toMigrate[i].SourceSchemaName)
				fmt.Printf("      [ERROR] %s: %v\n", schemaKey, err)
			}
		} else {
			result.Successful++
		}
	}
	if result.Failed > 0 {
		fmt.Println()
	}

	return result, nil
}

func (m *Migrator) migrateSchema(ctx context.Context, mapping *models.SchemaMapping, state *models.MigrationState) error {
	key := fmt.Sprintf("%s:%s", mapping.SourceRegistry, mapping.SourceSchemaName)

	// Get the full schema data
	schema, err := m.extractor.GetSchema(ctx, mapping.SourceRegistry, mapping.SourceSchemaName)
	if err != nil {
		state.FailedSchemas[key] = models.FailedSchema{
			SourceRegistry: mapping.SourceRegistry,
			SourceSchema:   mapping.SourceSchemaName,
			Error:          err.Error(),
			Attempts:       1,
			LastAttempt:    time.Now(),
		}
		return fmt.Errorf("failed to get schema %s: %w", key, err)
	}

	// Register each version in order
	versions := schema.Versions
	if m.config.Migration.VersionStrategy == "latest" {
		// Only migrate latest version
		if len(versions) > 0 {
			versions = versions[len(versions)-1:]
		}
	}

	for _, version := range versions {
		err := m.loader.RegisterSchema(ctx, mapping, &version)
		if err != nil {
			state.FailedSchemas[key] = models.FailedSchema{
				SourceRegistry: mapping.SourceRegistry,
				SourceSchema:   mapping.SourceSchemaName,
				Error:          err.Error(),
				Attempts:       1,
				LastAttempt:    time.Now(),
			}
			return fmt.Errorf("failed to register version %d of %s: %w", version.VersionNumber, key, err)
		}
	}

	// Mark as completed
	state.CompletedSchemas[key] = models.CompletedSchema{
		SourceRegistry: mapping.SourceRegistry,
		SourceSchema:   mapping.SourceSchemaName,
		TargetSubject:  mapping.TargetSubject,
		Versions:       len(versions),
		CompletedAt:    time.Now(),
	}
	state.CompletedCount++

	return nil
}

func (m *Migrator) printDryRunReport(plan *models.MigrationPlan) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    GLUE TO CONFLUENT CLOUD SR - DRY RUN REPORT               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Registry summary
	fmt.Println("REGISTRY SUMMARY")
	fmt.Println("────────────────")
	for _, reg := range plan.SourceRegistries {
		count := 0
		for _, m := range plan.Mappings {
			if m.SourceRegistry == reg {
				count++
			}
		}
		fmt.Printf("  %s: %d schemas\n", reg, count)
	}
	fmt.Println()

	// Schema mappings
	fmt.Println("SCHEMA MAPPINGS")
	fmt.Println("───────────────")
	for _, mapping := range plan.Mappings {
		status := "[OK]"
		if mapping.Status == models.MappingStatusWarning {
			status = "[WARN]"
		} else if mapping.Status == models.MappingStatusError {
			status = "[ERR]"
		}
		
		// Format target subject with context (only add prefix if context is not empty)
		targetSubject := mapping.TargetSubject
		if mapping.TargetContext != "" {
			targetSubject = mapping.TargetContext + ":" + mapping.TargetSubject
		}
		
		fmt.Printf("  %s %s.%s → %s (%s)\n",
			status,
			mapping.SourceRegistry,
			mapping.SourceSchemaName,
			targetSubject,
			mapping.NamingStrategy,
		)
	}
	fmt.Println()

	// Summary
	fmt.Println("SUMMARY")
	fmt.Println("───────")
	fmt.Printf("  Registries:     %d\n", plan.Summary.Registries)
	fmt.Printf("  Schemas:        %d\n", plan.Summary.Schemas)
	fmt.Printf("  Versions:       %d\n", plan.Summary.Versions)
	fmt.Printf("  References:     %d\n", plan.Summary.References)
	fmt.Printf("  Ready:          %d [OK]\n", plan.Summary.Ready)
	fmt.Printf("  Warnings:       %d [WARN]\n", plan.Summary.Warnings)
	fmt.Printf("  Errors:         %d [ERR]\n", plan.Summary.Errors)
	fmt.Println()
	fmt.Println("Run without --dry-run to execute migration.")
}

func (m *Migrator) generateReport(plan *models.MigrationPlan, startTime time.Time, dryRun bool) *models.MigrationReport {
	endTime := time.Now()
	
	report := &models.MigrationReport{
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime).String(),
		DryRun:    dryRun,
		Source: models.SourceReport{
			Type:       "aws_glue",
			Region:     m.config.AWS.Region,
			Registries: plan.SourceRegistries,
		},
		Target: models.TargetReport{
			Type: "confluent_cloud",
			URL:  m.config.ConfluentCloud.URL,
		},
		Config: models.ConfigReport{
			SubjectStrategy:   m.config.Naming.SubjectStrategy,
			ContextMapping:    m.config.Naming.ContextMapping,
			VersionStrategy:   m.config.Migration.VersionStrategy,
			ReferenceStrategy: m.config.Migration.ReferenceStrategy,
			NormalizeDots:     m.config.Normalization.NormalizeDots,
			NormalizeCase:     m.config.Normalization.NormalizeCase,
			LLMProvider:       m.config.LLM.Provider,
			LLMModel:          m.config.LLM.Model,
		},
		Results: models.ResultsReport{
			RegistriesProcessed: plan.Summary.Registries,
			SchemasProcessed:    plan.Summary.Schemas,
			VersionsProcessed:   plan.Summary.Versions,
			Successful:          plan.Summary.Ready,
		},
	}

	// Add schema details
	for _, mapping := range plan.Mappings {
		schemaReport := models.SchemaReport{
			SourceRegistry:   mapping.SourceRegistry,
			SourceSchema:     mapping.SourceSchemaName,
			TargetContext:    mapping.TargetContext,
			TargetSubject:    mapping.TargetSubject,
			DetectedRole:     mapping.DetectedRole,
			RoleReason:       mapping.NamingReason,
			NamingStrategy:   mapping.NamingStrategy,
			Transformations:  mapping.Transformations,
			References:       mapping.References,
			Status:           string(mapping.Status),
			Warning:          mapping.Warning,
			Error:            mapping.Error,
		}
		report.Schemas = append(report.Schemas, schemaReport)
	}

	return report
}

func getMigrationOrder(levels []graph.Level) []string {
	var order []string
	for _, level := range levels {
		for _, schema := range level.Schemas {
			order = append(order, fmt.Sprintf("%s:%s", schema.SourceRegistry, schema.SourceSchemaName))
		}
	}
	return order
}
