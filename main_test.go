package main

import (
	"math"
	"testing"
)

func TestRoundCurrency(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"rounds up", 509.9915, 510.00},
		{"rounds down", 509.991, 509.99},
		{"exact value", 509.99, 509.99},
		{"zero decimals", 500.00, 500.00},
		{"small value", 0.126, 0.13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roundCurrency(tt.input)
			if result != tt.expected {
				t.Errorf("roundCurrency(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPriceCachePrecision(t *testing.T) {
	for sku, rule := range priceCache {
		// Verify price has exactly 2 decimal places
		rounded := math.Round(rule.FinalPrice*100) / 100
		if rule.FinalPrice != rounded {
			t.Errorf("SKU %s has imprecise price: %v (expected %v)", sku, rule.FinalPrice, rounded)
		}

		// Verify price matches expected calculation
		expectedPrice := roundCurrency(rule.BasePrice * (1 - rule.Discount/100))
		if rule.FinalPrice != expectedPrice {
			t.Errorf("SKU %s: FinalPrice %v != expected %v", sku, rule.FinalPrice, expectedPrice)
		}
	}
}

func TestSpecificPriceCalculations(t *testing.T) {
	tests := []struct {
		sku      string
		expected float64
	}{
		{"SKU-001", 509.99},  // 599.99 * 0.85 = 509.9915 → 509.99
		{"SKU-003", 279.99},  // 349.99 * 0.80 = 279.992 → 279.99
		{"SKU-007", 62.99},   // 89.99 * 0.70 = 62.993 → 62.99
	}

	for _, tt := range tests {
		t.Run(tt.sku, func(t *testing.T) {
			rule, exists := priceCache[tt.sku]
			if !exists {
				t.Fatalf("SKU %s not found in cache", tt.sku)
			}
			if rule.FinalPrice != tt.expected {
				t.Errorf("SKU %s: got price %v, want %v", tt.sku, rule.FinalPrice, tt.expected)
			}
		})
	}
}
