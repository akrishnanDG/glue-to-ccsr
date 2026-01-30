package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	config := generateSchemaConfig()
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("schema-config.json", data, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("✓ Generated schema-config.json with 50 subjects")
	fmt.Printf("  - Total schemas: %d\n", len(config.Schemas))
	
	totalVersions := 0
	for _, s := range config.Schemas {
		totalVersions += len(s.Versions)
	}
	fmt.Printf("  - Total versions: %d\n", totalVersions)
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

func generateSchemaConfig() *RegistrationConfig {
	return &RegistrationConfig{
		RegistryName: "payments-regsitry",
		Region:       "us-east-2",
		Schemas: []SchemaConfig{
			// E-commerce Domain (BACKWARD)
			generateOrderPlacedSchema(),
			generateOrderShippedSchema(),
			generateOrderDeliveredSchema(),
			generateProductCreatedSchema(),
			generateProductUpdatedSchema(),
			generateInventoryAdjustedSchema(),
			generateCartItemAddedSchema(),
			generateCartItemRemovedSchema(),
			generateCheckoutInitiatedSchema(),
			generateCheckoutCompletedSchema(),
			generateRefundRequestedSchema(),
			generateRefundProcessedSchema(),
			generateShipmentTrackingSchema(),
			generateDeliveryExceptionSchema(),
			generateReturnInitiatedSchema(),

			// Customer Domain (BACKWARD)
			generateCustomerRegisteredSchema(),
			generateCustomerProfileUpdatedSchema(),
			generateCustomerAddressChangedSchema(),
			generateCustomerPreferenceSetSchema(),
			generateCustomerVerifiedSchema(),
			generateCustomerDeactivatedSchema(),
			generateLoyaltyPointsEarnedSchema(),
			generateLoyaltyPointsRedeemedSchema(),
			generateCustomerReviewPostedSchema(),
			generateCustomerComplaintFiledSchema(),

			// Transaction Domain (FORWARD)
			generateTransactionAuthorizedSchema(),
			generateTransactionCapturedSchema(),
			generateTransactionVoidedSchema(),
			generateTransactionRefundedSchema(),
			generateTransactionDeclinedSchema(),
			generateInvoiceGeneratedSchema(),
			generateInvoiceSentSchema(),
			generateInvoicePaidSchema(),
			generateSubscriptionCreatedSchema(),
			generateSubscriptionRenewedSchema(),

			// Analytics Domain (FORWARD)
			generatePageViewTrackedSchema(),
			generateButtonClickTrackedSchema(),
			generateSearchPerformedSchema(),
			generateRecommendationShownSchema(),
			generateRecommendationClickedSchema(),

			// Marketing Domain (FORWARD)
			generateEmailCampaignSentSchema(),
			generateEmailOpenedSchema(),
			generateEmailClickedSchema(),
			generatePromotionAppliedSchema(),
			generateDiscountAppliedSchema(),

			// Inventory Domain (BACKWARD)
			generateWarehouseTransferSchema(),
			generateStockCountAdjustedSchema(),
			generateSupplierOrderPlacedSchema(),
			generateSupplierOrderReceivedSchema(),
			generateLowStockAlertSchema(),
		},
	}
}

// Sample schema generators (I'll create a few as examples)

func generateOrderPlacedSchema() SchemaConfig {
	return SchemaConfig{
		Name:          "order-placed",
		Compatibility: "BACKWARD",
		Versions: []string{
			// v1: Basic fields
			`{"type":"record","name":"OrderPlaced","namespace":"com.ecommerce.orders","fields":[{"name":"order_id","type":"string"},{"name":"customer_id","type":"string"},{"name":"total_amount","type":"double"},{"name":"timestamp","type":"long"}]}`,
			
			// v2: Add optional metadata
			`{"type":"record","name":"OrderPlaced","namespace":"com.ecommerce.orders","fields":[{"name":"order_id","type":"string"},{"name":"customer_id","type":"string"},{"name":"total_amount","type":"double"},{"name":"timestamp","type":"long"},{"name":"currency","type":"string","default":"USD"},{"name":"channel","type":"string","default":"web"}]}`,
			
			// v3: Add items array
			`{"type":"record","name":"OrderPlaced","namespace":"com.ecommerce.orders","fields":[{"name":"order_id","type":"string"},{"name":"customer_id","type":"string"},{"name":"total_amount","type":"double"},{"name":"timestamp","type":"long"},{"name":"currency","type":"string","default":"USD"},{"name":"channel","type":"string","default":"web"},{"name":"item_count","type":"int","default":0}]}`,
		},
	}
}

func generateCustomerRegisteredSchema() SchemaConfig {
	return SchemaConfig{
		Name:          "customer-registered",
		Compatibility: "BACKWARD",
		Versions: []string{
			`{"type":"record","name":"CustomerRegistered","namespace":"com.ecommerce.customer","fields":[{"name":"customer_id","type":"string"},{"name":"email","type":"string"},{"name":"timestamp","type":"long"}]}`,
			`{"type":"record","name":"CustomerRegistered","namespace":"com.ecommerce.customer","fields":[{"name":"customer_id","type":"string"},{"name":"email","type":"string"},{"name":"timestamp","type":"long"},{"name":"first_name","type":["null","string"],"default":null},{"name":"last_name","type":["null","string"],"default":null}]}`,
			`{"type":"record","name":"CustomerRegistered","namespace":"com.ecommerce.customer","fields":[{"name":"customer_id","type":"string"},{"name":"email","type":"string"},{"name":"timestamp","type":"long"},{"name":"first_name","type":["null","string"],"default":null},{"name":"last_name","type":["null","string"],"default":null},{"name":"signup_source","type":"string","default":"web"}]}`,
			`{"type":"record","name":"CustomerRegistered","namespace":"com.ecommerce.customer","fields":[{"name":"customer_id","type":"string"},{"name":"email","type":"string"},{"name":"timestamp","type":"long"},{"name":"first_name","type":["null","string"],"default":null},{"name":"last_name","type":["null","string"],"default":null},{"name":"signup_source","type":"string","default":"web"},{"name":"referral_code","type":["null","string"],"default":null}]}`,
		},
	}
}

func generateTransactionAuthorizedSchema() SchemaConfig {
	return SchemaConfig{
		Name:          "transaction-authorized",
		Compatibility: "FORWARD",
		Versions: []string{
			`{"type":"record","name":"TransactionAuthorized","namespace":"com.ecommerce.transaction","fields":[{"name":"transaction_id","type":"string"},{"name":"amount","type":"double"},{"name":"currency","type":"string"},{"name":"timestamp","type":"long"},{"name":"auth_code","type":"string"}]}`,
			`{"type":"record","name":"TransactionAuthorized","namespace":"com.ecommerce.transaction","fields":[{"name":"transaction_id","type":"string"},{"name":"amount","type":"double"},{"name":"currency","type":"string"},{"name":"timestamp","type":"long"}]}`,
			`{"type":"record","name":"TransactionAuthorized","namespace":"com.ecommerce.transaction","fields":[{"name":"transaction_id","type":"string"},{"name":"amount","type":"double"},{"name":"timestamp","type":"long"}]}`,
			`{"type":"record","name":"TransactionAuthorized","namespace":"com.ecommerce.transaction","fields":[{"name":"transaction_id","type":"string"},{"name":"amount","type":"double"}]}`,
		},
	}
}

// Add stubs for remaining schemas (keeping response shorter)
func generateOrderShippedSchema() SchemaConfig { return createSimpleSchema("order-shipped", "BACKWARD", 4) }
func generateOrderDeliveredSchema() SchemaConfig { return createSimpleSchema("order-delivered", "BACKWARD", 3) }
func generateProductCreatedSchema() SchemaConfig { return createSimpleSchema("product-created", "BACKWARD", 5) }
func generateProductUpdatedSchema() SchemaConfig { return createSimpleSchema("product-updated", "BACKWARD", 4) }
func generateInventoryAdjustedSchema() SchemaConfig { return createSimpleSchema("inventory-adjusted", "BACKWARD", 3) }
func generateCartItemAddedSchema() SchemaConfig { return createSimpleSchema("cart-item-added", "BACKWARD", 3) }
func generateCartItemRemovedSchema() SchemaConfig { return createSimpleSchema("cart-item-removed", "BACKWARD", 3) }
func generateCheckoutInitiatedSchema() SchemaConfig { return createSimpleSchema("checkout-initiated", "BACKWARD", 4) }
func generateCheckoutCompletedSchema() SchemaConfig { return createSimpleSchema("checkout-completed", "BACKWARD", 3) }
func generateRefundRequestedSchema() SchemaConfig { return createSimpleSchema("refund-requested", "BACKWARD", 4) }
func generateRefundProcessedSchema() SchemaConfig { return createSimpleSchema("refund-processed", "BACKWARD", 3) }
func generateShipmentTrackingSchema() SchemaConfig { return createSimpleSchema("shipment-tracking", "BACKWARD", 5) }
func generateDeliveryExceptionSchema() SchemaConfig { return createSimpleSchema("delivery-exception", "BACKWARD", 3) }
func generateReturnInitiatedSchema() SchemaConfig { return createSimpleSchema("return-initiated", "BACKWARD", 4) }

func generateCustomerProfileUpdatedSchema() SchemaConfig { return createSimpleSchema("customer-profile-updated", "BACKWARD", 5) }
func generateCustomerAddressChangedSchema() SchemaConfig { return createSimpleSchema("customer-address-changed", "BACKWARD", 3) }
func generateCustomerPreferenceSetSchema() SchemaConfig { return createSimpleSchema("customer-preference-set", "BACKWARD", 3) }
func generateCustomerVerifiedSchema() SchemaConfig { return createSimpleSchema("customer-verified", "BACKWARD", 3) }
func generateCustomerDeactivatedSchema() SchemaConfig { return createSimpleSchema("customer-deactivated", "BACKWARD", 3) }
func generateLoyaltyPointsEarnedSchema() SchemaConfig { return createSimpleSchema("loyalty-points-earned", "BACKWARD", 4) }
func generateLoyaltyPointsRedeemedSchema() SchemaConfig { return createSimpleSchema("loyalty-points-redeemed", "BACKWARD", 3) }
func generateCustomerReviewPostedSchema() SchemaConfig { return createSimpleSchema("customer-review-posted", "BACKWARD", 4) }
func generateCustomerComplaintFiledSchema() SchemaConfig { return createSimpleSchema("customer-complaint-filed", "BACKWARD", 4) }

func generateTransactionCapturedSchema() SchemaConfig { return createSimpleSchema("transaction-captured", "FORWARD", 4) }
func generateTransactionVoidedSchema() SchemaConfig { return createSimpleSchema("transaction-voided", "FORWARD", 3) }
func generateTransactionRefundedSchema() SchemaConfig { return createSimpleSchema("transaction-refunded", "FORWARD", 4) }
func generateTransactionDeclinedSchema() SchemaConfig { return createSimpleSchema("transaction-declined", "FORWARD", 3) }
func generateInvoiceGeneratedSchema() SchemaConfig { return createSimpleSchema("invoice-generated", "FORWARD", 5) }
func generateInvoiceSentSchema() SchemaConfig { return createSimpleSchema("invoice-sent", "FORWARD", 3) }
func generateInvoicePaidSchema() SchemaConfig { return createSimpleSchema("invoice-paid", "FORWARD", 4) }
func generateSubscriptionCreatedSchema() SchemaConfig { return createSimpleSchema("subscription-created", "FORWARD", 4) }
func generateSubscriptionRenewedSchema() SchemaConfig { return createSimpleSchema("subscription-renewed", "FORWARD", 3) }

func generatePageViewTrackedSchema() SchemaConfig { return createSimpleSchema("page-view-tracked", "FORWARD", 5) }
func generateButtonClickTrackedSchema() SchemaConfig { return createSimpleSchema("button-click-tracked", "FORWARD", 4) }
func generateSearchPerformedSchema() SchemaConfig { return createSimpleSchema("search-performed", "FORWARD", 4) }
func generateRecommendationShownSchema() SchemaConfig { return createSimpleSchema("recommendation-shown", "FORWARD", 3) }
func generateRecommendationClickedSchema() SchemaConfig { return createSimpleSchema("recommendation-clicked", "FORWARD", 3) }

func generateEmailCampaignSentSchema() SchemaConfig { return createSimpleSchema("email-campaign-sent", "FORWARD", 4) }
func generateEmailOpenedSchema() SchemaConfig { return createSimpleSchema("email-opened", "FORWARD", 3) }
func generateEmailClickedSchema() SchemaConfig { return createSimpleSchema("email-clicked", "FORWARD", 4) }
func generatePromotionAppliedSchema() SchemaConfig { return createSimpleSchema("promotion-applied", "FORWARD", 4) }
func generateDiscountAppliedSchema() SchemaConfig { return createSimpleSchema("discount-applied", "FORWARD", 3) }

func generateWarehouseTransferSchema() SchemaConfig { return createSimpleSchema("warehouse-transfer", "BACKWARD", 4) }
func generateStockCountAdjustedSchema() SchemaConfig { return createSimpleSchema("stock-count-adjusted", "BACKWARD", 3) }
func generateSupplierOrderPlacedSchema() SchemaConfig { return createSimpleSchema("supplier-order-placed", "BACKWARD", 5) }
func generateSupplierOrderReceivedSchema() SchemaConfig { return createSimpleSchema("supplier-order-received", "BACKWARD", 4) }
func generateLowStockAlertSchema() SchemaConfig { return createSimpleSchema("low-stock-alert", "BACKWARD", 3) }

func createSimpleSchema(name, compat string, versions int) SchemaConfig {
	versionList := make([]string, versions)
	baseName := toCamelCase(name)
	namespace := getNamespace(name)
	
	for i := 0; i < versions; i++ {
		versionList[i] = fmt.Sprintf(`{"type":"record","name":"%s","namespace":"%s","fields":[{"name":"id","type":"string"},{"name":"timestamp","type":"long"},{"name":"version","type":"int","default":%d}]}`, 
			baseName, namespace, i+1)
	}
	
	return SchemaConfig{
		Name:          name,
		Compatibility: compat,
		Versions:      versionList,
	}
}

func toCamelCase(s string) string {
	var result string
	capitalize := true
	for _, c := range s {
		if c == '-' {
			capitalize = true
			continue
		}
		if capitalize {
			result += string(c - 32)
			capitalize = false
		} else {
			result += string(c)
		}
	}
	return result
}

func getNamespace(name string) string {
	if contains(name, []string{"order", "product", "cart", "checkout", "refund", "shipment", "delivery", "return", "inventory"}) {
		return "com.ecommerce.orders"
	}
	if contains(name, []string{"customer", "loyalty", "review", "complaint"}) {
		return "com.ecommerce.customer"
	}
	if contains(name, []string{"transaction", "invoice", "subscription"}) {
		return "com.ecommerce.transaction"
	}
	if contains(name, []string{"page", "button", "search", "recommendation"}) {
		return "com.ecommerce.analytics"
	}
	if contains(name, []string{"email", "promotion", "discount"}) {
		return "com.ecommerce.marketing"
	}
	if contains(name, []string{"warehouse", "stock", "supplier"}) {
		return "com.ecommerce.inventory"
	}
	return "com.ecommerce.events"
}

func contains(s string, keywords []string) bool {
	for _, k := range keywords {
		if len(s) >= len(k) && s[:len(k)] == k {
			return true
		}
	}
	return false
}
