package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	config := generateKeyValueSchemas()
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("keyvalue-config.json", data, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("✓ Generated keyvalue-config.json with key/value test schemas")
	fmt.Printf("  - Total schemas: %d\n", len(config.Schemas))
	
	totalVersions := 0
	for _, s := range config.Schemas {
		totalVersions += len(s.Versions)
	}
	fmt.Printf("  - Total versions: %d\n", totalVersions)
	
	// Count key vs value schemas
	keyCount := 0
	valueCount := 0
	ambiguousCount := 0
	
	for _, s := range config.Schemas {
		if containsSubstring(s.Name, []string{"-key", "_key", ".key", "Key"}) {
			keyCount++
		} else if containsSubstring(s.Name, []string{"-value", "_value", ".value", "Value"}) {
			valueCount++
		} else {
			ambiguousCount++
		}
	}
	
	fmt.Printf("  - Key schemas: %d\n", keyCount)
	fmt.Printf("  - Value schemas: %d\n", valueCount)
	fmt.Printf("  - Ambiguous: %d\n", ambiguousCount)
}

type SchemaConfig struct {
	Name          string   `json:"name"`
	Compatibility string   `json:"compatibility"`
	Versions      []string `json:"versions"`
}

type RegistrationConfig struct {
	RegistryName string         `json:"registry_name"`
	Region       string         `json:"region"`
	Schemas      []SchemaConfig `json:"schemas"`
}

func generateKeyValueSchemas() *RegistrationConfig {
	return &RegistrationConfig{
		RegistryName: "payments-regsitry",
		Region:       "us-east-2",
		Schemas: []SchemaConfig{
			// Standard naming: hyphen-key/value
			generateSchema("user-event-key", "BACKWARD", 3, "UserEventKey", "com.test.keys"),
			generateSchema("user-event-value", "BACKWARD", 3, "UserEventValue", "com.test.values"),
			
			// Underscore naming: snake_case_key/value
			generateSchema("order_created_key", "BACKWARD", 3, "OrderCreatedKey", "com.test.keys"),
			generateSchema("order_created_value", "BACKWARD", 4, "OrderCreatedValue", "com.test.values"),
			
			// Dot notation: dot.separated.key/value
			generateSchema("product.updated.key", "BACKWARD", 3, "ProductUpdatedKey", "com.test.keys"),
			generateSchema("product.updated.value", "BACKWARD", 3, "ProductUpdatedValue", "com.test.values"),
			
			// CamelCase variations
			generateSchema("UserRegisteredKey", "BACKWARD", 3, "UserRegisteredKey", "com.test.keys"),
			generateSchema("UserRegisteredValue", "BACKWARD", 4, "UserRegisteredValue", "com.test.values"),
			
			// Mixed case with suffixes
			generateSchema("accountStatusKey", "BACKWARD", 3, "AccountStatusKey", "com.test.keys"),
			generateSchema("accountStatusValue", "BACKWARD", 3, "AccountStatusValue", "com.test.values"),
			
			// ID suffix patterns (ambiguous - could be key)
			generateSchema("userId", "BACKWARD", 3, "UserId", "com.test.identifiers"),
			generateSchema("orderId", "BACKWARD", 3, "OrderId", "com.test.identifiers"),
			generateSchema("customerId", "BACKWARD", 3, "CustomerId", "com.test.identifiers"),
			
			// ID suffix uppercase
			generateSchema("productID", "BACKWARD", 3, "ProductID", "com.test.identifiers"),
			generateSchema("transactionID", "BACKWARD", 3, "TransactionID", "com.test.identifiers"),
			
			// Compound words without clear separator
			generateSchema("userevent_key", "BACKWARD", 3, "UserEventKey", "com.test.keys"),
			generateSchema("userevent_value", "BACKWARD", 3, "UserEventValue", "com.test.values"),
			
			// Event suffix (typically value schemas)
			generateSchema("UserCreatedEvent", "BACKWARD", 4, "UserCreatedEvent", "com.test.events"),
			generateSchema("OrderPlacedEvent", "BACKWARD", 4, "OrderPlacedEvent", "com.test.events"),
			generateSchema("PaymentProcessedEvent", "BACKWARD", 3, "PaymentProcessedEvent", "com.test.events"),
			
			// Identifier prefix (likely key schemas)
			generateSchema("identifier-user", "BACKWARD", 3, "IdentifierUser", "com.test.keys"),
			generateSchema("identifier-order", "BACKWARD", 3, "IdentifierOrder", "com.test.keys"),
			
			// Key prefix variations
			generateSchema("key-user-account", "BACKWARD", 3, "KeyUserAccount", "com.test.keys"),
			generateSchema("key.product.catalog", "BACKWARD", 3, "KeyProductCatalog", "com.test.keys"),
			
			// Complex multi-word patterns
			generateSchema("user-profile-update-key", "BACKWARD", 3, "UserProfileUpdateKey", "com.test.keys"),
			generateSchema("user-profile-update-value", "BACKWARD", 4, "UserProfileUpdateValue", "com.test.values"),
			
			// Ambiguous patterns (no clear indicator)
			generateSchema("notification", "BACKWARD", 3, "Notification", "com.test.ambiguous"),
			generateSchema("alert", "BACKWARD", 3, "Alert", "com.test.ambiguous"),
			generateSchema("message", "BACKWARD", 3, "Message", "com.test.ambiguous"),
			
			// Different compatibility modes
			generateSchema("session-key", "FORWARD", 3, "SessionKey", "com.test.keys"),
			generateSchema("session-value", "FORWARD", 4, "SessionValue", "com.test.values"),
			
			generateSchema("metric-key", "FORWARD", 3, "MetricKey", "com.test.keys"),
			generateSchema("metric-value", "FORWARD", 3, "MetricValue", "com.test.values"),
			
			// Edge cases with multiple separators
			generateSchema("user.account.profile.key", "BACKWARD", 3, "UserAccountProfileKey", "com.test.keys"),
			generateSchema("user.account.profile.value", "BACKWARD", 3, "UserAccountProfileValue", "com.test.values"),
			
			generateSchema("order_item_details_key", "BACKWARD", 3, "OrderItemDetailsKey", "com.test.keys"),
			generateSchema("order_item_details_value", "BACKWARD", 4, "OrderItemDetailsValue", "com.test.values"),
			
			// Keys with references (key schema references another key schema)
			generateSchemaWithRef("composite-user-key", "BACKWARD", 3, "CompositeUserKey", "com.test.keys", "userId"),
			generateSchemaWithRef("composite-order-key", "BACKWARD", 3, "CompositeOrderKey", "com.test.keys", "orderId"),
		},
	}
}

func generateSchema(name, compat string, versions int, recordName, namespace string) SchemaConfig {
	versionList := make([]string, versions)
	
	for i := 0; i < versions; i++ {
		fields := fmt.Sprintf(`[{"name":"id","type":"string"},{"name":"timestamp","type":"long"},{"name":"version","type":"int","default":%d}`, i+1)
		
		// Add more fields in later versions for backward compatibility
		if i >= 1 {
			fields += fmt.Sprintf(`,{"name":"metadata","type":["null","string"],"default":null}`)
		}
		if i >= 2 {
			fields += fmt.Sprintf(`,{"name":"source","type":"string","default":"system"}`)
		}
		
		fields += `]`
		
		versionList[i] = fmt.Sprintf(`{"type":"record","name":"%s","namespace":"%s","fields":%s}`, 
			recordName, namespace, fields)
	}
	
	return SchemaConfig{
		Name:          name,
		Compatibility: compat,
		Versions:      versionList,
	}
}

func generateSchemaWithRef(name, compat string, versions int, recordName, namespace, refSchema string) SchemaConfig {
	versionList := make([]string, versions)
	
	for i := 0; i < versions; i++ {
		// First version has a reference to another schema
		if i == 0 {
			versionList[i] = fmt.Sprintf(`{"type":"record","name":"%s","namespace":"%s","fields":[{"name":"id","type":"string"},{"name":"ref_id","type":"string"},{"name":"timestamp","type":"long"}]}`, 
				recordName, namespace)
		} else {
			fields := fmt.Sprintf(`[{"name":"id","type":"string"},{"name":"ref_id","type":"string"},{"name":"timestamp","type":"long"},{"name":"version","type":"int","default":%d}`, i+1)
			
			if i >= 2 {
				fields += fmt.Sprintf(`,{"name":"composite","type":"boolean","default":true}`)
			}
			
			fields += `]`
			
			versionList[i] = fmt.Sprintf(`{"type":"record","name":"%s","namespace":"%s","fields":%s}`, 
				recordName, namespace, fields)
		}
	}
	
	return SchemaConfig{
		Name:          name,
		Compatibility: compat,
		Versions:      versionList,
	}
}

func containsSubstring(s string, substrings []string) bool {
	for _, sub := range substrings {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}
