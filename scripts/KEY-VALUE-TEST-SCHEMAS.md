# Key/Value Test Schemas - Registration Summary

## 🎉 Successfully Registered!

**Registry:** `payments-regsitry`  
**Region:** `us-east-2`

- **39 test schemas** ✓
- **124 total versions** ✓
- **13 key schemas** (explicit)
- **11 value schemas** (explicit)
- **15 ambiguous schemas** (for testing detection logic)

## 📋 Schema Patterns by Category

### 1. Standard Hyphen Notation (`-key`/`-value`)
Perfect for testing basic key/value detection:
- ✓ `user-event-key` (3 versions)
- ✓ `user-event-value` (3 versions)
- ✓ `user-profile-update-key` (3 versions)
- ✓ `user-profile-update-value` (4 versions)
- ✓ `session-key` (3 versions, FORWARD compat)
- ✓ `session-value` (4 versions, FORWARD compat)
- ✓ `metric-key` (3 versions, FORWARD compat)
- ✓ `metric-value` (3 versions, FORWARD compat)

### 2. Underscore/Snake Case (`_key`/`_value`)
Tests underscore separator handling:
- ✓ `order_created_key` (3 versions)
- ✓ `order_created_value` (4 versions)
- ✓ `userevent_key` (3 versions)
- ✓ `userevent_value` (3 versions)
- ✓ `order_item_details_key` (3 versions)
- ✓ `order_item_details_value` (4 versions)

### 3. Dot Notation (`.key`/`.value`)
Tests dot separator handling (important for name normalization):
- ✓ `product.updated.key` (3 versions)
- ✓ `product.updated.value` (3 versions)
- ✓ `user.account.profile.key` (3 versions)
- ✓ `user.account.profile.value` (3 versions)
- ✓ `key.product.catalog` (3 versions)

### 4. CamelCase with Key/Value Suffix
Tests capitalization variations:
- ✓ `UserRegisteredKey` (3 versions)
- ✓ `UserRegisteredValue` (4 versions)
- ✓ `accountStatusKey` (3 versions)
- ✓ `accountStatusValue` (3 versions)

### 5. ID Suffix Patterns (Ambiguous - likely keys)
Tests patterns like `userId`, `customerId`, etc.:
- ✓ `userId` (3 versions) - lowercase 'Id'
- ✓ `orderId` (3 versions)
- ✓ `customerId` (3 versions)
- ✓ `productID` (3 versions) - uppercase 'ID'
- ✓ `transactionID` (3 versions)

### 6. Event Suffix Patterns (Typically values)
Tests event naming conventions:
- ✓ `UserCreatedEvent` (4 versions)
- ✓ `OrderPlacedEvent` (4 versions)
- ✓ `PaymentProcessedEvent` (3 versions)

### 7. Identifier/Key Prefix Patterns
Tests prefix-based detection:
- ✓ `identifier-user` (3 versions)
- ✓ `identifier-order` (3 versions)
- ✓ `key-user-account` (3 versions)

### 8. Ambiguous Patterns (No clear key/value indicator)
Tests fallback logic when heuristics fail:
- ✓ `notification` (3 versions)
- ✓ `alert` (3 versions)
- ✓ `message` (3 versions)

### 9. Composite Key Patterns (With References)
Tests schemas that reference other schemas:
- ✓ `composite-user-key` (3 versions) - references `userId`
- ✓ `composite-order-key` (3 versions) - references `orderId`

## 🧪 Testing the Migration Tool

### Test Basic Key/Value Detection
```bash
./bin/glue-to-ccsr migrate \
  --aws-region us-east-2 \
  --glue-registries payments-regsitry \
  --cc-sr-url YOUR_SR_URL \
  --cc-api-key YOUR_KEY \
  --cc-api-secret YOUR_SECRET \
  --dry-run
```

Expected behavior:
- Schemas ending with `-key`, `_key`, `.key` → detected as key schemas
- Schemas ending with `-value`, `_value`, `.value` → detected as value schemas
- Schemas with `Key`/`Value` in name → detected accordingly
- Schemas ending with `Id`/`ID` → heuristic suggests key schemas
- Schemas ending with `Event` → typically value schemas
- Ambiguous schemas → use user-defined regex or manual override

### Test Custom Regex Patterns
```bash
./bin/glue-to-ccsr migrate \
  --aws-region us-east-2 \
  --glue-registries payments-regsitry \
  --key-regex '.*[Ii][Dd]$' \
  --value-regex '.*Event$' \
  --dry-run
```

This will:
- Match `userId`, `orderId`, `productID`, `transactionID` as key schemas
- Match `UserCreatedEvent`, `OrderPlacedEvent`, `PaymentProcessedEvent` as value schemas

### Test Name Normalization with Dots
```bash
./bin/glue-to-ccsr migrate \
  --aws-region us-east-2 \
  --glue-registries payments-regsitry \
  --name-normalization replace \
  --dot-replacement _ \
  --dry-run
```

Expected transformations:
- `product.updated.key` → `product_updated_key`
- `user.account.profile.key` → `user_account_profile_key`
- `key.product.catalog` → `key_product_catalog`

### Test Extract-Last Strategy
```bash
./bin/glue-to-ccsr migrate \
  --aws-region us-east-2 \
  --glue-registries payments-regsitry \
  --name-normalization extract-last \
  --dry-run
```

Expected transformations:
- `product.updated.key` → `key`
- `user.account.profile.value` → `value`
- `key.product.catalog` → `catalog`

## 📊 Detection Heuristics Validation

The migration tool should apply these rules in order:

1. **Explicit suffix match** (highest priority):
   - `-key`, `_key`, `.key` → KEY
   - `-value`, `_value`, `.value` → VALUE

2. **CamelCase suffix**:
   - `*Key` → KEY
   - `*Value` → VALUE

3. **User-defined regex** (if provided)

4. **Heuristic patterns**:
   - `*Id`, `*ID` → likely KEY
   - `*Event` → likely VALUE
   - `identifier-*`, `key-*` → likely KEY

5. **Fallback**: Default to VALUE (safer assumption)

## 🎯 Total Test Coverage

**Overall Registry Stats:**
- **Original schemas:** 50 subjects, 183 versions
- **Key/Value test schemas:** 39 subjects, 124 versions
- **Grand Total:** 89 schemas, 307 versions

**Test Coverage:**
- ✅ Hyphen separators (`-key`, `-value`)
- ✅ Underscore separators (`_key`, `_value`)
- ✅ Dot separators (`.key`, `.value`)
- ✅ CamelCase variations (`UserKey`, `accountKey`)
- ✅ ID patterns (`userId`, `productID`)
- ✅ Event patterns (`UserCreatedEvent`)
- ✅ Prefix patterns (`key-*`, `identifier-*`)
- ✅ Ambiguous cases (`notification`, `alert`)
- ✅ Composite keys with references
- ✅ Mixed compatibility modes (BACKWARD, FORWARD)
- ✅ Multiple versions per schema (3-4 versions each)

## 📝 Next Steps

1. **Run dry-run migration** to see detection results
2. **Validate key/value assignments** in the output
3. **Test custom regex patterns** for edge cases
4. **Test name normalization** strategies with dotted names
5. **Verify subject naming** in Confluent Cloud format

All schemas are ready for comprehensive migration testing! 🚀
