# AWS Glue to Confluent Cloud Schema Registry Migration Tool

A high-performance, production-ready Go tool for migrating schemas from AWS Glue Schema Registry to Confluent Cloud Schema Registry. Supports parallel processing, LLM-powered naming, and handles complex schema references.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> **Migration Guide**: For a comprehensive end-to-end migration guide covering schema migration, dual-read strategy, producer migration, and complete switchover, see [Migration from AWS Glue Schema Registry](https://github.com/akrishnanDG/schema-migration-guide/blob/main/docs/09-migration-from-glue.md).

## Features

### Core Capabilities
- ✅ **Multi-Format Support**: Avro, JSON Schema, and Protobuf
- ✅ **Parallel Processing**: Up to 100x faster with worker pools (50+ concurrent workers)
- ✅ **Schema Versioning**: Preserves all schema versions and their order
- ✅ **Schema References**: Handles $ref dependencies with automatic rewriting
- ✅ **Cross-Registry**: Supports multi-registry migrations with reference resolution
- ✅ **Key/Value Detection**: Intelligent detection of Kafka key vs value schemas
- ✅ **Name Normalization**: Handles dots, special characters, and case conversion
- ✅ **Collision Resolution**: Auto-resolves naming conflicts with configurable strategies (suffix, prefix, etc.)
- ✅ **Metadata Migration**: Transfers descriptions and tags
- ✅ **Dry-Run Mode**: Preview changes without modifying anything
- ✅ **Resume Capability**: Checkpoint-based resumption for interrupted migrations
- ✅ **Progress Tracking**: Real-time progress bars and detailed reports

### Advanced Features
- 🤖 **LLM Integration**: AI-powered subject naming (OpenAI, Anthropic, Ollama, local LLMs)
- 📊 **Flexible Naming**: Topic, record, LLM, or custom template strategies
- 🔄 **Context Mapping**: Flat, registry-based, or custom context strategies
- ⚡ **Performance Tuning**: Configurable rate limits and worker pools
- 📝 **Comprehensive Logging**: Detailed logs with configurable levels
- 🐳 **Docker Support**: Containerized execution

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Supported Schema Formats](#supported-schema-formats)
- [Performance](#performance)
- [AWS IAM Permissions](#aws-iam-permissions)
- [Examples](#examples)
- [Architecture](#architecture)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## Quick Start

### 1. Install

```bash
# Clone the repository
git clone https://github.com/akrishnanDG/glue-to-ccsr.git
cd glue-to-ccsr

# Build
make build

# The binary will be in ./bin/glue-to-ccsr
```

### 2. Configure

```bash
# Copy example config
cp config.example.yaml config.yaml

# Edit config.yaml with your settings
vi config.yaml
```

Minimal configuration for dry-run:

```yaml
aws:
  region: us-east-2
  profile: default
  registry_names:
    - my-registry

naming:
  subject_strategy: topic
  context_mapping: flat

output:
  dry_run: true
```

### 3. Run Dry-Run

```bash
./bin/glue-to-ccsr migrate --config config.yaml
```

### 4. Execute Migration

After verifying dry-run output:

```yaml
# Update config.yaml
confluent_cloud:
  url: https://psrc-xxx.confluent.cloud
  api_key: YOUR_KEY
  api_secret: YOUR_SECRET

output:
  dry_run: false
```

```bash
./bin/glue-to-ccsr migrate --config config.yaml
```

## Installation

### From Source

**Prerequisites:**
- Go 1.21 or higher
- AWS credentials configured
- Access to AWS Glue Schema Registry
- (Optional) Access to Confluent Cloud Schema Registry

**Build:**

```bash
# Clone repository
git clone https://github.com/akrishnanDG/glue-to-ccsr.git
cd glue-to-ccsr

# Install dependencies
go mod download

# Build binary
make build

# Binary location: ./bin/glue-to-ccsr
```

### Using Docker

```bash
# Build Docker image
docker build -t glue-to-ccsr .

# Run with config file
docker run -v $(pwd)/config.yaml:/config.yaml glue-to-ccsr migrate --config /config.yaml
```

## Usage

### Command Line Interface

The tool provides a simplified CLI with most configuration in the config file:

```bash
glue-to-ccsr migrate [flags]
```

**Essential Flags:**

```
-c, --config string              Config file path (RECOMMENDED)
    --aws-region string          AWS region (default "us-east-1")
    --aws-profile string         AWS profile name
    --aws-access-key-id string   AWS access key ID
    --aws-secret-access-key string  AWS secret access key
    --aws-registry-name strings  Registry name (can be repeated)
    --aws-registry-all          Migrate all registries
    --cc-sr-url string          Confluent Cloud SR URL (not needed for dry-run)
    --cc-api-key string         Confluent Cloud API key (not needed for dry-run)
    --cc-api-secret string      Confluent Cloud API secret (not needed for dry-run)
    --dry-run                   Preview without making changes
    --workers int               Number of parallel workers (default 10)
    --log-level string          Log level: debug, info, warn, error (default "info")
-h, --help                      Help for migrate
```

### Usage Examples

**1. Simple Dry-Run (Using Config File)**

```bash
glue-to-ccsr migrate --config config.yaml --dry-run
```

**2. Dry-Run with CLI Overrides**

```bash
glue-to-ccsr migrate \
  --config config.yaml \
  --aws-region us-west-2 \
  --aws-registry-name production-registry \
  --workers 20 \
  --dry-run
```

**3. Production Migration**

```bash
glue-to-ccsr migrate \
  --config config.yaml \
  --log-level info
```

**4. Migration with Explicit AWS Credentials**

```bash
glue-to-ccsr migrate \
  --aws-region us-east-2 \
  --aws-access-key-id YOUR_AWS_ACCESS_KEY_ID \
  --aws-secret-access-key YOUR_AWS_SECRET_ACCESS_KEY \
  --aws-registry-name my-registry \
  --cc-sr-url https://psrc-xxx.confluent.cloud \
  --cc-api-key YOUR_CONFLUENT_API_KEY \
  --cc-api-secret YOUR_CONFLUENT_API_SECRET
```

**5. High-Performance Migration (50 Workers)**

```bash
glue-to-ccsr migrate \
  --config config.yaml \
  --workers 50 \
  --log-level info
```

## Configuration

### Configuration File

The tool uses YAML configuration files. See `config.example.yaml` for a complete reference with all options documented.

**Configuration Sections:**

| Section | Description |
|---------|-------------|
| `aws` | AWS Glue Schema Registry settings (region, credentials, registries) |
| `confluent_cloud` | Confluent Cloud Schema Registry settings (URL, API credentials) |
| `naming` | Subject naming strategies and context mapping |
| `normalization` | Name normalization rules (dots, case, special chars) |
| `key_value` | Key/value schema detection rules |
| `migration` | Migration behavior (versions, references, metadata) |
| `metadata` | Metadata migration settings |
| `llm` | LLM configuration for AI-powered naming |
| `concurrency` | Performance tuning (workers, rate limits, retries) |
| `checkpoint` | Checkpoint and resume settings |
| `output` | Output format, logging, and dry-run settings |

### AWS Authentication

The tool supports multiple AWS authentication methods (in order of precedence):

1. **Explicit credentials** (config file or CLI flags)
   ```yaml
   aws:
     access_key_id: YOUR_AWS_ACCESS_KEY_ID
     secret_access_key: YOUR_AWS_SECRET_ACCESS_KEY
   ```

2. **AWS Profile**
   ```yaml
   aws:
     profile: production
   ```

3. **Environment variables**
   ```bash
   export AWS_ACCESS_KEY_ID=YOUR_AWS_ACCESS_KEY_ID
   export AWS_SECRET_ACCESS_KEY=YOUR_AWS_SECRET_ACCESS_KEY
   ```

4. **Default credential chain** (IAM role, instance profile, etc.)

### Naming Strategies

#### Subject Strategies

**1. Topic Strategy (Default, Fastest)**

Uses schema name as subject base:

```yaml
naming:
  subject_strategy: topic
```

Examples:
```
user-event-key     → user-event-key
order.shipped      → order-shipped-value
UserCreatedEvent   → user-created-event-value
```

**2. Record Strategy**

Uses record name from schema definition:

```yaml
naming:
  subject_strategy: record
```

Examples:
```
UserEvent.avsc → com.example.UserEvent-value
Order.avsc     → com.example.Order-value
```

**3. LLM Strategy (AI-Powered)**

Uses Large Language Models for intelligent naming:

```yaml
naming:
  subject_strategy: llm

llm:
  provider: ollama
  model: llama3.2
  base_url: http://localhost:11434
```

Examples:
```
usr.evt.created           → user-created-value
order_processing_handler  → order-processed-value
customer.profile.upd.v2   → customer-profile-updated-value
```

**4. Custom Strategy**

Uses custom Go templates:

```yaml
naming:
  subject_strategy: custom
  subject_template: "{registry}-{name}"
```

#### Context Mapping

**1. Flat Context (Default, Recommended)**

All schemas in default context:

```yaml
naming:
  context_mapping: flat
```

Output: `user-event-key`, `order-shipped-value`

**2. Registry Context**

Registry name as context prefix:

```yaml
naming:
  context_mapping: registry
```

Output: `.payments-registry:user-event-key`

**3. Custom Context**

Custom mapping from file:

```yaml
naming:
  context_mapping: custom
  context_mapping_file: context-map.yaml
```

### Custom Name Mappings

For cases where you need explicit control over specific subject names, you can provide a custom name mapping file. This is useful when you have thousands of schemas but only need to override naming for a handful of them — unmapped schemas fall through to the configured naming strategy automatically.

**Enable via config:**

```yaml
naming:
  subject_strategy: topic              # handles unmapped schemas
  name_mapping_file: "name-mappings.yaml"  # overrides for specific schemas
```

**Or via CLI flag:**

```bash
glue-to-ccsr migrate --config config.yaml --name-mapping-file name-mappings.yaml
```

The mapping file supports three styles, all usable in the same file:

**1. Simple Mappings** (match by schema name across any registry):

```yaml
mappings:
  "UserCreatedEvent": "user-created-value"
  "OrderPlaced": "order-placed-value"
```

**2. Qualified Mappings** (registry-specific, using `registry:schema`):

```yaml
qualified_mappings:
  "payments-registry:PaymentEvent": "payment-event-value"
  "orders-registry:OrderEvent": "order-event-value"
```

**3. Extended Mappings** (full control with optional role and context overrides):

```yaml
extended_mappings:
  - source: "UserKey"
    subject: "user-key"
    role: "key"
    context: ".users"
  - source: "payments:RefundEvent"
    subject: "payment-refund-value"
    role: "value"
```

**Lookup Priority:**

1. **Qualified match** (`registry:schema`) — checked first
2. **Simple match** (schema name only) — fallback
3. **No match** — falls through to configured naming strategy (topic/record/llm/custom)

Mapped schemas bypass the entire naming pipeline (normalization, auto-suffixing, etc.), so the subject name you specify is used exactly as-is.

### Key/Value Detection

The tool automatically detects whether schemas represent Kafka message keys or values.

**Built-in Detection Patterns:**

| Pattern | Detected As |
|---------|-------------|
| `*-key`, `*_key`, `*Key` | Key |
| `*-value`, `*_value`, `*Value` | Value |
| `*ID`, `*Id`, `composite-*` | Key |
| `*Event`, `*Command`, `*` | Value (default) |

**Custom Detection:**

```yaml
key_value:
  key_regex:
    - ".*-key$"
    - ".*_key$"
    - ".*Identifier$"
  
  value_regex:
    - ".*-event$"
    - ".*-command$"
  
  default_role: value
```

**Manual Overrides:**

```yaml
key_value:
  role_override_file: overrides.json
```

`overrides.json`:
```json
{
  "user-profile": "value",
  "session-id": "key"
}
```

### Name Normalization

**Dot Handling:**

```yaml
normalization:
  normalize_dots: replace    # keep | replace | extract-last
  dot_replacement: "-"
```

Examples:
```
user.event.created → user-event-created  (replace)
user.event.created → created             (extract-last)
user.event.created → user.event.created  (keep)
```

**Case Normalization:**

```yaml
normalization:
  normalize_case: kebab      # keep | kebab | snake | lower
```

Examples:
```
UserEvent → user-event  (kebab)
UserEvent → user_event  (snake)
UserEvent → userevent   (lower)
UserEvent → UserEvent   (keep)
```

**Collision Resolution:**

When multiple Glue schemas normalize to the same Confluent subject name, the tool can automatically resolve conflicts:

```yaml
normalization:
  collision_check: true
  collision_resolution: suffix    # suffix | registry-prefix | prefer-shorter | skip | fail
```

Resolution Strategies:

| Strategy | Behavior | Example | Data Loss |
|----------|----------|---------|-----------|
| `suffix` (default) | Add numeric suffix (-1, -2, etc.) | `product-updated-value`, `product-updated-value-1` | No |
| `registry-prefix` | Prepend registry name | `payments-product-updated-value` | No |
| `prefer-shorter` | Keep schema with shorter name | `product-updated` kept, `product.updated.value` skipped | Yes |
| `skip` | Keep first, skip duplicates | First schema kept, others skipped | Yes |
| `fail` | Stop migration with error | Report collision, manual resolution required | N/A |

**Example:**

```yaml
normalization:
  collision_check: true
  collision_resolution: suffix  # Safe default, no data loss
```

Output:
```
[3/5] Generating schema mappings...
      Found 1 naming collisions, applying 'suffix' resolution strategy...
```

## Supported Schema Formats

The tool supports all schema formats available in AWS Glue Schema Registry and Confluent Cloud Schema Registry:

### ✅ Avro

Apache Avro schemas with full support for:
- Complex types (records, arrays, maps, unions)
- Schema references via namespace resolution
- Schema evolution (backward, forward, full compatibility)
- Logical types (timestamps, decimals, etc.)

**Example:**
```json
{
  "type": "record",
  "name": "User",
  "namespace": "com.example",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "email", "type": "string"},
    {"name": "created_at", "type": {"type": "long", "logicalType": "timestamp-millis"}}
  ]
}
```

### ✅ JSON Schema

JSON Schema (Draft 7) with support for:
- Complex nested structures
- Schema references via `$ref`
- Validation rules and constraints
- Schema evolution

**Example:**
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "email": {"type": "string", "format": "email"},
    "age": {"type": "integer", "minimum": 0}
  },
  "required": ["id", "email"]
}
```

### ✅ Protobuf

Protocol Buffers (proto3) with support for:
- Message types and nested messages
- Enums and repeated fields
- Schema imports and dependencies
- Backward/forward compatibility

**Example:**
```protobuf
syntax = "proto3";

package com.example;

message User {
  string id = 1;
  string email = 2;
  int32 age = 3;
}
```

### Format Detection

The tool automatically detects the schema format from AWS Glue Schema Registry metadata and correctly registers it in Confluent Cloud with the appropriate format.

**Automatic Handling:**
- ✅ Format preservation during migration
- ✅ Version compatibility validation per format
- ✅ Reference rewriting for all formats
- ✅ Metadata migration for all formats

## Performance

### Parallel Processing Architecture

The tool uses Go goroutines and worker pools for maximum performance:

```
┌─────────────────────────────────────────┐
│  Schema Queue (90 schemas)              │
└─────────────────────────────────────────┘
                 ↓
   ┌──────┬──────┬──────┬──────┬──────┐
   │ W1   │ W2   │ W3   │ ...  │ W50  │  ← 50 Workers
   │ [=]  │ [=]  │ [=]  │      │ [=]  │
   └──────┴──────┴──────┴──────┴──────┘
                 ↓
   ┌─────────────────────────────────────┐
   │  Rate Limiter (50 req/sec)         │
   └─────────────────────────────────────┘
                 ↓
   ┌─────────────────────────────────────┐
   │  AWS Glue Schema Registry API      │
   └─────────────────────────────────────┘
```

### Performance Benchmarks

**Test Environment:**
- 90 schemas
- 176 versions
- 1 registry
- MacBook Pro M1

| Configuration | Workers | AWS Rate Limit | Duration | Speedup |
|---------------|---------|----------------|----------|---------|
| Default | 10 | 10 req/sec | 38s | baseline |
| Optimized | 50 | 50 req/sec | **14s** | **2.7x** |
| Maximum | 100 | 100 req/sec | **8s** | **4.8x** |

### Performance Tuning

**For Small Registries (<50 schemas):**

```yaml
concurrency:
  workers: 10
  aws_rate_limit: 10
```

**For Medium Registries (50-500 schemas):**

```yaml
concurrency:
  workers: 20
  aws_rate_limit: 20
```

**For Large Registries (500+ schemas):**

```yaml
concurrency:
  workers: 50
  aws_rate_limit: 50
```

**Note:** Higher rate limits require AWS Service Quota increase. See [AWS Documentation](https://docs.aws.amazon.com/servicequotas/) for requesting quota increases.

### Bottlenecks

**Rate Limiter is the Bottleneck:**

Even with 50 workers, the AWS API rate limit (default 10 req/sec) is the bottleneck. To achieve maximum performance:

1. Request AWS rate limit increase via Service Quotas
2. Match `workers` to `aws_rate_limit` in config
3. Monitor and adjust based on AWS throttling

**API Call Breakdown (90 schemas, 176 versions):**

```
1  × ListSchemas         = 1 call
90 × GetSchema           = 90 calls
90 × ListSchemaVersions  = 90 calls
176 × GetSchemaVersion   = 176 calls
────────────────────────────────────
Total                    ≈ 357 calls

At 10 req/sec:  357 / 10 = 35.7s minimum
At 50 req/sec:  357 / 50 = 7.1s minimum
```

## AWS IAM Permissions

### Required Permissions for Migration

The tool needs read access to AWS Glue Schema Registry:

**Minimal Policy (Specific Registry):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "glue:GetRegistry",
        "glue:ListSchemas",
        "glue:GetSchema",
        "glue:ListSchemaVersions",
        "glue:GetSchemaVersion"
      ],
      "Resource": [
        "arn:aws:glue:us-east-2:123456789012:registry/my-registry",
        "arn:aws:glue:us-east-2:123456789012:schema/my-registry/*"
      ]
    }
  ]
}
```

**Full Read Access (All Registries):**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "glue:ListRegistries",
        "glue:GetRegistry",
        "glue:ListSchemas",
        "glue:GetSchema",
        "glue:ListSchemaVersions",
        "glue:GetSchemaVersion",
        "glue:GetTags"
      ],
      "Resource": "*"
    }
  ]
}
```

See `iam-policy.json`, `iam-policy-minimal.json`, and `iam-policy-specific-registry.json` for complete examples.

### Setup Instructions

**Option 1: IAM User**

```bash
# Create IAM user
aws iam create-user --user-name glue-sr-migration

# Attach policy
aws iam put-user-policy --user-name glue-sr-migration \
  --policy-name GlueSRReadAccess \
  --policy-document file://iam-policy.json

# Create access key
aws iam create-access-key --user-name glue-sr-migration
```

**Option 2: IAM Role (for EC2/ECS)**

```bash
# Create role
aws iam create-role --role-name glue-sr-migration-role \
  --assume-role-policy-document file://trust-policy.json

# Attach policy
aws iam put-role-policy --role-name glue-sr-migration-role \
  --policy-name GlueSRReadAccess \
  --policy-document file://iam-policy.json
```

## Examples

### Example 1: Basic Dry-Run

Preview what would be migrated:

```bash
glue-to-ccsr migrate \
  --aws-region us-east-2 \
  --aws-profile default \
  --aws-registry-name my-registry \
  --dry-run
```

Output:
```
[1/5] Extracting schemas from AWS Glue Schema Registry...
      Found 90 schemas across 1 registries

      Fetching schemas 100% [==================================] (90/90)

[2/5] Building dependency graph...
      Created 1 dependency levels

[3/5] Generating schema mappings...

[4/5] Validating mappings...

[5/5] DRY RUN - No changes will be made

SCHEMA MAPPINGS
───────────────
  [OK] my-registry.user-event-key → user-event-key (topic)
  [OK] my-registry.order-shipped → order-shipped-value (topic)
  ...

SUMMARY
───────
  Registries:     1
  Schemas:        90
  Versions:       176
  Ready:          90 [OK]
  Warnings:       0 [WARN]
  Errors:         0 [ERR]
```

### Example 2: Fast Migration with Config File

**config.yaml:**
```yaml
aws:
  region: us-east-2
  profile: production
  registry_names:
    - payments-registry
    - orders-registry

confluent_cloud:
  url: https://psrc-xxx.confluent.cloud
  api_key: YOUR_KEY
  api_secret: YOUR_SECRET

naming:
  subject_strategy: topic
  context_mapping: flat

concurrency:
  workers: 50
  aws_rate_limit: 50

output:
  dry_run: false
  log_level: info
```

**Execute:**
```bash
glue-to-ccsr migrate --config config.yaml
```

### Example 3: LLM-Powered Naming (Local Ollama)

**Prerequisites:**
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull model
ollama pull llama3.2

# Start Ollama server
ollama serve
```

**config.yaml:**
```yaml
aws:
  region: us-east-2
  profile: default
  registry_names:
    - my-registry

naming:
  subject_strategy: llm

llm:
  provider: ollama
  model: llama3.2
  base_url: http://localhost:11434
  cache_file: .llm-cache.json
  max_cost: 10.0

concurrency:
  workers: 20
  llm_rate_limit: 10

output:
  dry_run: true
```

**Execute:**
```bash
glue-to-ccsr migrate --config config.yaml
```

### Example 4: Migration with Resume

For large migrations, use checkpointing to resume on failure:

**config.yaml:**
```yaml
checkpoint:
  file: migration-checkpoint.json
  resume: false

# ... other config ...
```

**First run:**
```bash
glue-to-ccsr migrate --config config.yaml
# Migration runs and saves checkpoint periodically
```

**If interrupted, resume:**
```bash
# Update config: resume: true
glue-to-ccsr migrate --config config.yaml
# Resumes from last checkpoint
```

## Architecture

### High-Level Flow

```
┌─────────────────────┐
│  1. Extract         │  Fetch schemas from AWS Glue SR
│     (Parallel)      │  • List registries & schemas
└─────────────────────┘  • Get schema versions
          ↓              • Parallel worker pool
┌─────────────────────┐
│  2. Build Graph     │  Create dependency graph
│                     │  • Detect $ref dependencies
└─────────────────────┘  • Topological sort
          ↓              • Handle cross-registry refs
┌─────────────────────┐
│  3. Map Names       │  Generate subject names
│     (Parallel)      │  • Apply naming strategy
└─────────────────────┘  • Detect key/value role
          ↓              • Normalize names
┌─────────────────────┐  • Optional LLM naming
│  4. Validate        │  Pre-migration checks
│                     │  • Collision detection
└─────────────────────┘  • Reference validation
          ↓              • Compatibility checks
┌─────────────────────┐
│  5. Migrate         │  Load to Confluent Cloud SR
│     (Ordered)       │  • Process dependency order
└─────────────────────┘  • Preserve version order
                         • Migrate metadata
                         • Handle references
```

### Key Components

| Component | Responsibility |
|-----------|----------------|
| **Extractor** | Fetches schemas from AWS Glue SR (parallel) |
| **Dependency Graph** | Resolves schema references and ordering |
| **Mapper** | Generates Confluent Cloud subject names |
| **Normalizer** | Normalizes schema names (dots, case, etc.) |
| **KV Detector** | Identifies key vs value schemas |
| **LLM Namer** | AI-powered subject naming (optional) |
| **Validator** | Pre-migration validation and collision detection |
| **Loader** | Registers schemas to Confluent Cloud SR |
| **Worker Pool** | Parallel processing with rate limiting |
| **Checkpoint** | State persistence for resume capability |

## Troubleshooting

### Common Issues

**1. AWS Authentication Errors**

```
Error: failed to load AWS config: failed to get shared config profile
```

**Solution:**
- Verify AWS credentials: `aws sts get-caller-identity`
- Check profile name: `aws configure list-profiles`
- Use explicit credentials in config file

**2. AWS Permission Errors**

```
AccessDeniedException: User is not authorized to perform: glue:GetSchema
```

**Solution:**
- Apply IAM policy: see [AWS IAM Permissions](#aws-iam-permissions)
- Use `iam-policy.json` or `iam-policy-read-write.json`

**3. Rate Limiting**

```
ThrottlingException: Rate exceeded
```

**Solution:**
- Reduce workers: `--workers 5`
- Reduce rate limit in config: `aws_rate_limit: 5`
- Request AWS quota increase

**4. Naming Collisions**

```
WARNING: Naming collisions detected:
  [product.updated.value product-updated] -> product-updated-value
```

**Solution:**

The tool can automatically resolve collisions:

```yaml
normalization:
  collision_check: true
  collision_resolution: suffix  # Safe default, adds -1, -2, etc.
```

Other options:
- `registry-prefix` - Prepend registry name to differentiate
- `prefer-shorter` - Keep schema with shorter name
- Review collision report in dry-run
- Use custom naming strategy
- Add manual overrides in config

**5. Schema Reference Errors**

```
Error: failed to resolve reference: schema not found
```

**Solution:**
- Ensure all referenced schemas are included
- Check `cross_registry_refs: resolve` in config
- Verify dependency order

### Debug Mode

Enable debug logging for troubleshooting:

```bash
glue-to-ccsr migrate --config config.yaml --log-level debug
```

Or in config:
```yaml
output:
  log_level: debug
  log_file: debug.log
```

### Performance Issues

**Slow extraction (>1 minute for 100 schemas):**

1. Check rate limit: increase `aws_rate_limit`
2. Check workers: increase `workers` to match rate limit
3. Check network: test AWS API latency
4. Enable progress bar: `progress: true` to see bottlenecks

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development

```bash
# Clone repository
git clone https://github.com/akrishnanDG/glue-to-ccsr.git
cd glue-to-ccsr

# Install dependencies
go mod download

# Run tests
make test

# Build
make build

# Run linter
golangci-lint run
```

### Testing

The `scripts/` directory contains tools to generate test data:

```bash
cd scripts

# Generate and register test schemas to AWS Glue SR
go run register-schemas.go

# See scripts/README.md for details
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Migration Guide**: [Migration from AWS Glue Schema Registry](https://github.com/akrishnanDG/schema-migration-guide/blob/main/docs/09-migration-from-glue.md) — end-to-end migration with dual-read, producer migration, and switchover
- **Issues**: [GitHub Issues](https://github.com/akrishnanDG/glue-to-ccsr/issues)
- **Documentation**: This README and `config.example.yaml`
- **Examples**: See [Examples](#examples) section

## Acknowledgments

- Built with [Go](https://golang.org/)
- Uses [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)
- Progress bars by [progressbar](https://github.com/schollz/progressbar)
- CLI powered by [Cobra](https://github.com/spf13/cobra)

---

**Made with ❤️ for the data streaming community**
