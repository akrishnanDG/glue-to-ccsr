# AWS Glue Schema Registry - Write Permissions

## Minimal Permissions to Register Schemas

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "glue:CreateSchema",
        "glue:RegisterSchemaVersion",
        "glue:GetSchema",
        "glue:GetRegistry"
      ],
      "Resource": "*"
    }
  ]
}
```

## Full Write Permissions (with registry creation)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "glue:CreateRegistry",
        "glue:CreateSchema",
        "glue:RegisterSchemaVersion",
        "glue:UpdateSchema",
        "glue:PutSchemaVersionMetadata",
        "glue:TagResource"
      ],
      "Resource": "*"
    }
  ]
}
```

## Key Actions

- `glue:CreateSchema` - Create new schema
- `glue:RegisterSchemaVersion` - Register new version of existing schema
- `glue:CreateRegistry` - Create new registry (if needed)
- `glue:UpdateSchema` - Update schema metadata
- `glue:PutSchemaVersionMetadata` - Add metadata to versions
- `glue:TagResource` - Add tags
