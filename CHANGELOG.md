# Changelog

## [1.0.0] - 2026-01-30

### Major Features
- ✅ **Parallel Schema Extraction**: Worker pool architecture with up to 100 concurrent workers
- ✅ **LLM Integration**: Support for OpenAI, Anthropic, Ollama, and local LLMs for intelligent naming
- ✅ **Progress Bars**: Real-time progress tracking during extraction and registration
- ✅ **Emoji-Free Output**: Clean, terminal-compatible output
- ✅ **Simplified CLI**: Moved most configuration to YAML file for better maintainability
- ✅ **Performance Optimization**: 2.7x - 4.8x faster extraction with configurable rate limits
- ✅ **Collision Resolution**: Auto-resolves naming conflicts with 5 strategies (suffix, registry-prefix, prefer-shorter, skip, fail)
- ✅ **Dry-Run Mode**: Preview migrations without making changes
- ✅ **URL Encoding**: Proper encoding of subject names in Confluent Cloud API calls

### Configuration
- 📝 **Comprehensive Config File**: `config.example.yaml` with all parameters documented
- 📝 **AWS Authentication**: Multiple methods (profile, explicit credentials, environment variables)
- 📝 **Flexible Naming**: Topic, record, LLM, or custom template strategies
- 📝 **Context Mapping**: Flat (default), registry-based, or custom contexts

### Performance
- ⚡ **Baseline (10 workers)**: 38 seconds for 90 schemas
- ⚡ **Optimized (50 workers)**: 14 seconds for 90 schemas (2.7x faster)
- ⚡ **Maximum (100 workers)**: 8 seconds for 90 schemas (4.8x faster)

### Documentation
- 📚 **Comprehensive README**: Complete usage guide with examples
- 📚 **IAM Policies**: Ready-to-use AWS IAM policies for Glue SR access
- 📚 **Troubleshooting**: Common issues and solutions

### Cleanup
- 🧹 Removed interim documentation files
- 🧹 Consolidated all documentation into README.md
- 🧹 Removed temporary test config files

### Testing
- ✅ Test data generation scripts in `scripts/` directory
- ✅ 90 test schemas with 176 versions registered
- ✅ Key/value detection test cases

## Performance Benchmarks

### Test Environment
- **Schemas**: 90 schemas, 176 versions
- **Registry**: payments-regsitry (AWS us-east-2)
- **Hardware**: MacBook Pro M1

### Results

| Configuration | Workers | Rate Limit | Duration | Speedup |
|---------------|---------|------------|----------|---------|
| Default | 10 | 10 req/sec | 38s | 1.0x |
| Optimized | 50 | 50 req/sec | 14s | 2.7x |
| Maximum | 100 | 100 req/sec | 8s | 4.8x |

### Scalability Estimates

| Registry Size | Default (10w) | Optimized (50w) |
|---------------|---------------|-----------------|
| Small (50 schemas) | ~20s | ~8s |
| Medium (500 schemas) | ~3min | ~1min |
| Large (5000 schemas) | ~30min | ~10min |

## Breaking Changes

### v1.0.0
- **CLI Flags**: Reduced from 50+ flags to 13 essential flags
- **Configuration**: Most settings now in config file (not CLI)
- **Default Context**: Changed from `registry` to `flat` (no context prefix)
- **Output Format**: Removed emojis, using `[OK]`, `[WARN]`, `[ERR]` instead

### Migration from Pre-1.0

**Old CLI:**
```bash
glue-to-ccsr migrate \
  --aws-region us-east-2 \
  --aws-profile default \
  --aws-registry-name my-registry \
  --subject-strategy topic \
  --context-mapping flat \
  --normalize-dots replace \
  --normalize-case kebab \
  --collision-check \
  --workers 10 \
  --dry-run
```

**New CLI:**
```bash
# Create config.yaml with all settings
glue-to-ccsr migrate --config config.yaml --dry-run
```

## Features by Category

### Core Migration
- [x] Schema extraction from AWS Glue SR
- [x] Schema registration to Confluent Cloud SR
- [x] Version preservation
- [x] Schema references ($ref) handling
- [x] Cross-registry reference resolution
- [x] Metadata migration (descriptions, tags)
- [x] Dependency graph construction
- [x] Topological sorting

### Naming & Normalization
- [x] Topic naming strategy
- [x] Record naming strategy
- [x] LLM naming strategy
- [x] Custom template strategy
- [x] Dot normalization (keep/replace/extract-last)
- [x] Case normalization (keep/kebab/snake/lower)
- [x] Key/value detection
- [x] Collision detection and resolution
- [x] Name validation

### Performance & Reliability
- [x] Parallel processing (worker pools)
- [x] Configurable rate limiting
- [x] Automatic retry with exponential backoff
- [x] Checkpoint and resume
- [x] Progress tracking
- [x] Batch processing

### Configuration & Authentication
- [x] YAML configuration file
- [x] AWS profile authentication
- [x] Explicit credentials
- [x] Environment variable credentials
- [x] Default credential chain
- [x] CLI flag overrides

### Output & Reporting
- [x] Dry-run mode
- [x] Real-time progress bars
- [x] Detailed migration reports
- [x] JSON/Table/CSV output formats
- [x] Structured logging
- [x] Debug mode

### Developer Experience
- [x] Comprehensive documentation
- [x] Example configurations
- [x] Test data generation scripts
- [x] IAM policy templates
- [x] Docker support
- [x] Makefile for common tasks

## Known Limitations

1. **AWS Rate Limits**: Default AWS Glue SR rate limit is 10 req/sec. Higher performance requires quota increase.
2. **LLM Costs**: LLM naming strategy incurs API costs (unless using local Ollama).
3. **Large Schemas**: Very large schema definitions (>1MB) may be slow to process.
4. **Schema Evolution**: Only handles backward/forward compatibility, not full/transitive.

## Roadmap

### v1.1.0 (Planned)
- [ ] Confluent Cloud authentication via OAuth
- [ ] Schema validation before migration
- [ ] Parallel loading to Confluent Cloud
- [ ] Web UI for migration management
- [ ] Prometheus metrics export

### v1.2.0 (Planned)
- [ ] Bidirectional migration (CC SR → Glue SR)
- [ ] Schema diff comparison
- [ ] Rollback capability
- [ ] Migration scheduling
- [ ] Slack/Teams notifications

### Future
- [ ] Schema transformation rules
- [ ] Custom validation plugins
- [ ] Multi-cloud support (Azure, GCP)
- [ ] GraphQL API

## Contributors

- Initial implementation: akrishnanDG

## License

MIT License - see LICENSE file for details
