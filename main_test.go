package main

import (
	"math"
	"testing"
)

func TestPricePrecision(t *testing.T) {
	testCases := []struct {
		sku           string
		expectedPrice float64
	}{
		{"SKU-001", 509.99},  // 599.99 * 0.85 = 509.9915 → 509.99
		{"SKU-002", 999.99},  // No discount
		{"SKU-003", 279.99},  // 349.99 * 0.80 = 279.992 → 279.99
		{"SKU-004", 314.99},  // 349.99 * 0.90 = 314.991 → 314.99
		{"SKU-005", 712.49},  // 749.99 * 0.95 = 712.4905 → 712.49
		{"SKU-006", 322.49},  // 429.99 * 0.75 = 322.4925 → 322.49
		{"SKU-007", 62.99},   // 89.99 * 0.70 = 62.993 → 62.99
		{"SKU-008", 159.99},  // No discount
	}

	for _, tc := range testCases {
		t.Run(tc.sku, func(t *testing.T) {
			cacheMu.RLock()
			rule, exists := priceCache[tc.sku]
			cacheMu.RUnlock()

			if !exists {
				t.Fatalf("SKU %s not found in cache", tc.sku)
			}

			if rule.FinalPrice != tc.expectedPrice {
				t.Errorf("Price precision error for %s: got %.2f, want %.2f",
					tc.sku, rule.FinalPrice, tc.expectedPrice)
			}

			// Verify no more than 2 decimal places
			scaled := rule.FinalPrice * 100
			if scaled != math.Floor(scaled) {
				t.Errorf("Price has more than 2 decimal places: %.10f", rule.FinalPrice)
			}
		})
	}
}

func TestPriceCalculationConsistency(t *testing.T) {
	// Verify that manual calculation matches cached value
	cacheMu.RLock()
	rule := priceCache["SKU-001"]
	cacheMu.RUnlock()

	expected := math.Round(rule.BasePrice*(1-rule.Discount/100)*100) / 100
	if rule.FinalPrice != expected {
		t.Errorf("Inconsistent price calculation: cached=%.2f, calculated=%.2f",
			rule.FinalPrice, expected)
	}
}
