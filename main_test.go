package main

import (
	"math"
	"testing"
)

func TestPriceRounding(t *testing.T) {
	tests := []struct {
		sku           string
		expectedPrice float64
	}{
		{"SKU-001", 509.99},  // 599.99 * 0.85 = 509.9915 → should round to 509.99
		{"SKU-003", 279.99},  // 349.99 * 0.80 = 279.992 → should round to 279.99
		{"SKU-004", 314.99},  // 349.99 * 0.90 = 314.991 → should round to 314.99
		{"SKU-006", 322.49},  // 429.99 * 0.75 = 322.4925 → should round to 322.49
		{"SKU-007", 62.99},   // 89.99 * 0.70 = 62.993 → should round to 62.99
	}

	for _, tt := range tests {
		t.Run(tt.sku, func(t *testing.T) {
			cacheMu.RLock()
			rule, exists := priceCache[tt.sku]
			cacheMu.RUnlock()

			if !exists {
				t.Fatalf("SKU %s not found in cache", tt.sku)
			}

			if rule.FinalPrice != tt.expectedPrice {
				t.Errorf("FinalPrice = %.2f, want %.2f", rule.FinalPrice, tt.expectedPrice)
			}

			// Verify no more than 2 decimal places
			rounded := math.Round(rule.FinalPrice*100) / 100
			if rule.FinalPrice != rounded {
				t.Errorf("FinalPrice has more than 2 decimal places: %v", rule.FinalPrice)
			}
		})
	}
}

func TestNoDiscountPrices(t *testing.T) {
	tests := []struct {
		sku           string
		expectedPrice float64
	}{
		{"SKU-002", 999.99},
		{"SKU-008", 159.99},
	}

	for _, tt := range tests {
		t.Run(tt.sku, func(t *testing.T) {
			cacheMu.RLock()
			rule, exists := priceCache[tt.sku]
			cacheMu.RUnlock()

			if !exists {
				t.Fatalf("SKU %s not found in cache", tt.sku)
			}

			if rule.FinalPrice != tt.expectedPrice {
				t.Errorf("FinalPrice = %.2f, want %.2f (base price)", rule.FinalPrice, tt.expectedPrice)
			}

			if rule.FinalPrice != rule.BasePrice {
				t.Errorf("Expected final price to equal base price for 0%% discount")
			}
		})
	}
}
