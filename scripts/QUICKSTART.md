# 🚀 Quick Start Guide

## Register 50 Schemas to AWS Glue SR in 3 Steps

### 1️⃣ Setup Credentials (1 minute)

```bash
cd scripts
cp aws-creds.example.json aws-creds.json
```

Edit `aws-creds.json`:
```json
{
  "access_key_id": "YOUR_AWS_ACCESS_KEY_ID",
  "secret_access_key": "YOUR_AWS_SECRET_ACCESS_KEY"
}
```

### 2️⃣ Generate Schema Config (1 minute)

```bash
go run generate-schemas.go
```

Output:
```
✓ Generated schema-config.json with 50 subjects
  - Total schemas: 50
  - Total versions: 185
```

### 3️⃣ Register to Glue SR (5-10 minutes)

**Dry run first (recommended):**
```bash
go run register-schemas.go --dry-run
```

**Register for real:**
```bash
go run register-schemas.go
```

That's it! 🎉

## What Gets Created

- **50 schemas** across 6 domains:
  - 15 E-commerce (order-placed, product-created, etc.)
  - 10 Customer (customer-registered, loyalty-points-earned, etc.)
  - 10 Transaction (transaction-authorized, invoice-generated, etc.)
  - 5 Analytics (page-view-tracked, search-performed, etc.)
  - 5 Marketing (email-campaign-sent, promotion-applied, etc.)
  - 5 Inventory (warehouse-transfer, supplier-order-placed, etc.)

- **185 total versions** (3-5 per schema)
- **30 BACKWARD compatible** schemas
- **20 FORWARD compatible** schemas

## Verify

Check AWS Console or use CLI:

```bash
aws glue list-schemas \
  --registry-id RegistryName=payments-regsitry \
  --region us-east-2
```

## Need Help?

See `README.md` for detailed documentation.
