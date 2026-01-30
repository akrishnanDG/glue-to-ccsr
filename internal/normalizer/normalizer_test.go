package normalizer

import (
	"testing"

	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		dotStrategy    string
		caseStrategy   string
		expected       string
	}{
		{
			name:         "simple name kebab",
			input:        "UserEvent",
			dotStrategy:  "keep",
			caseStrategy: "kebab",
			expected:     "user-event",
		},
		{
			name:         "dotted name replace",
			input:        "a.b.c.schema",
			dotStrategy:  "replace",
			caseStrategy: "kebab",
			expected:     "a-b-c-schema",
		},
		{
			name:         "dotted name extract-last",
			input:        "a.b.c.schema",
			dotStrategy:  "extract-last",
			caseStrategy: "kebab",
			expected:     "schema",
		},
		{
			name:         "dotted name keep",
			input:        "a.b.c.schema",
			dotStrategy:  "keep",
			caseStrategy: "keep",
			expected:     "a.b.c.schema",
		},
		{
			name:         "camelCase to kebab",
			input:        "PaymentTransactionEvent",
			dotStrategy:  "keep",
			caseStrategy: "kebab",
			expected:     "payment-transaction-event",
		},
		{
			name:         "snake_case to kebab",
			input:        "payment_transaction_event",
			dotStrategy:  "keep",
			caseStrategy: "kebab",
			expected:     "payment-transaction-event",
		},
		{
			name:         "camelCase to snake",
			input:        "PaymentTransactionEvent",
			dotStrategy:  "keep",
			caseStrategy: "snake",
			expected:     "payment_transaction_event",
		},
		{
			name:         "with namespace dotted",
			input:        "com.example.UserEvent",
			dotStrategy:  "replace",
			caseStrategy: "kebab",
			expected:     "com-example-user-event",
		},
		{
			name:         "keep case",
			input:        "UserEventData",
			dotStrategy:  "keep",
			caseStrategy: "keep",
			expected:     "UserEventData",
		},
		{
			name:         "lower case only",
			input:        "UserEventData",
			dotStrategy:  "keep",
			caseStrategy: "lower",
			expected:     "usereventdata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			cfg.Normalization.NormalizeDots = tt.dotStrategy
			cfg.Normalization.NormalizeCase = tt.caseStrategy
			cfg.Normalization.DotReplacement = "-"

			n := New(cfg)
			result, _ := n.Normalize(tt.input)

			if result != tt.expected {
				t.Errorf("Normalize(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReplaceInvalidChars(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user/events", "user-events"},
		{"payment:transaction", "payment-transaction"},
		{"user events", "user-events"},
		{"user\\events", "user-events"},
		{"valid-name", "valid-name"},
	}

	cfg := config.NewDefaultConfig()
	n := New(cfg)

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, _ := n.replaceInvalidChars(tt.input)
			if result != tt.expected {
				t.Errorf("replaceInvalidChars(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UserEvent", "user-event"},
		{"userEvent", "user-event"},
		{"user_event", "user-event"},
		{"USER_EVENT", "user-event"},
		{"user-event", "user-event"},
		{"PaymentTransactionEvent", "payment-transaction-event"},
		{"MSKPaymentTxn", "mskpayment-txn"}, // Consecutive uppercase not split
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toKebabCase(tt.input)
			if result != tt.expected {
				t.Errorf("toKebabCase(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStripKeySuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user-event-key", "user-event"},
		{"user_event_key", "user_event"},
		{"UserEventKey", "UserEvent"},
		{"user-event", "user-event"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := StripKeySuffix(tt.input)
			if result != tt.expected {
				t.Errorf("StripKeySuffix(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStripValueSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user-event-value", "user-event"},
		{"user_event_value", "user_event"},
		{"UserEventValue", "UserEvent"},
		{"user-event", "user-event"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := StripValueSuffix(tt.input)
			if result != tt.expected {
				t.Errorf("StripValueSuffix(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
