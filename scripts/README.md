# AWS Glue Schema Registry - Schema Registration Scripts

This directory contains scripts to register 50+ test schemas to AWS Glue Schema Registry with multiple versions and different compatibility modes.

## 📋 Overview

- **50 schemas** across 6 business domains
- **3-5 versions per schema** (185 total versions)
- **30 BACKWARD compatible** schemas
- **20 FORWARD compatible** schemas
- No "payment" keyword used in schema names

## 🚀 Quick Start

### Step 1: Setup Credentials

Create `aws-creds.json` from the example:

```bash
cd scripts
cp aws-creds.example.json aws-creds.json
```

Edit `aws-creds.json` with your AWS credentials:

```json
{
  "access_key_id": "YOUR_AWS_ACCESS_KEY_ID",
  "secret_access_key": "YOUR_AWS_SECRET_ACCESS_KEY"
}
```

**⚠️ IMPORTANT:** Never commit `aws-creds.json` to git!

### Step 2: Generate Schema Configuration

```bash
go run generate-schemas.go
```

This creates `schema-config.json` with all 50 schemas and their versions.

### Step 3: Preview (Dry Run)

```bash
go run register-schemas.go --dry-run
```

This shows what will be registered without making any changes.

### Step 4: Register Schemas

```bash
go run register-schemas.go \
  --registry payments-regsitry \
  --region us-east-2 \
  --config schema-config.json \
  --creds aws-creds.json
```

## 📊 Schema Domains

### E-commerce (15 schemas - BACKWARD)
- Order lifecycle: placed, shipped, delivered
- Product management: created, updated
- Cart operations: item added, removed
- Checkout flow: initiated, completed
- Refunds and returns

### Customer (10 schemas - BACKWARD)
- Registration and profile management
- Address and preference updates
- Loyalty program
- Reviews and complaints

### Transaction (10 schemas - FORWARD)
- Transaction lifecycle: authorized, captured, voided
- Invoicing: generated, sent, paid
- Subscriptions: created, renewed

### Analytics (5 schemas - FORWARD)
- Page views and click tracking
- Search events
- Recommendation tracking

### Marketing (5 schemas - FORWARD)
- Email campaigns
- Promotion and discount tracking

### Inventory (5 schemas - BACKWARD)
- Warehouse transfers
- Stock adjustments
- Supplier orders

## 🔧 CLI Options

```bash
Flags:
  --config string      Schema configuration file (default "schema-config.json")
  --creds string       AWS credentials file (default "aws-creds.json")
  --dry-run           Preview without registering
  --registry string   Registry name (default "payments-regsitry")
  --region string     AWS region (default "us-east-2")
```

## 📝 Schema Evolution Examples

### Backward Compatible (can read old data with new schema)

**Version 1:**
```json
{
  "type": "record",
  "name": "OrderPlaced",
  "fields": [
    {"name": "order_id", "type": "string"},
    {"name": "total_amount", "type": "double"}
  ]
}
```

**Version 2:** (adds optional field)
```json
{
  "type": "record",
  "name": "OrderPlaced",
  "fields": [
    {"name": "order_id", "type": "string"},
    {"name": "total_amount", "type": "double"},
    {"name": "currency", "type": "string", "default": "USD"}
  ]
}
```

### Forward Compatible (can read new data with old schema)

**Version 1:**
```json
{
  "type": "record",
  "name": "TransactionAuthorized",
  "fields": [
    {"name": "transaction_id", "type": "string"},
    {"name": "amount", "type": "double"},
    {"name": "auth_code", "type": "string"}
  ]
}
```

**Version 2:** (removes optional field)
```json
{
  "type": "record",
  "name": "TransactionAuthorized",
  "fields": [
    {"name": "transaction_id", "type": "string"},
    {"name": "amount", "type": "double"}
  ]
}
```

## 🔒 Security Notes

1. **Never commit credentials:**
   - `aws-creds.json` is in `.gitignore`
   - Use AWS Secrets Manager in production

2. **IAM Permissions Required:**
   - `glue:CreateSchema`
   - `glue:RegisterSchemaVersion`
   - `glue:GetSchema`
   - `glue:GetRegistry`

   See `GLUE-WRITE-PERMISSIONS.md` for full policy.

3. **Rate Limiting:**
   - Script includes 200ms delay between schemas
   - 100ms delay between versions
   - Prevents AWS throttling

## 📈 Expected Output

```
Registry: payments-regsitry
Region: us-east-2
Total Schemas: 50

[1/50] Processing: order-placed (Compatibility: BACKWARD, Versions: 3)
  ✓ Successfully registered 3 versions
[2/50] Processing: order-shipped (Compatibility: BACKWARD, Versions: 4)
  ✓ Successfully registered 4 versions
...

============================================================
SUMMARY
============================================================
Total Schemas:    50
Total Versions:   185
Successful:       50 ✓
Failed:           0 ✗
```

## 🧪 Testing

To verify schemas were registered:

```bash
# List schemas in registry
aws glue list-schemas \
  --registry-id RegistryName=payments-regsitry \
  --region us-east-2

# Get specific schema versions
aws glue list-schema-versions \
  --schema-id SchemaName=order-placed,RegistryName=payments-regsitry \
  --region us-east-2
```

## 🐛 Troubleshooting

### Error: "Registry not found"
- Verify registry name: `payments-regsitry` (note the typo is intentional per user's request)
- Check region: `us-east-2`

### Error: "Access denied"
- Verify AWS credentials in `aws-creds.json`
- Check IAM permissions (see `GLUE-WRITE-PERMISSIONS.md`)

### Error: "Schema already exists"
- Script will skip existing versions
- Use `--dry-run` to preview first

### Rate limiting errors
- Script includes delays
- Increase delays in code if needed
