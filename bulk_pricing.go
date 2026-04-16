package main

import (
	"context"
	"fmt"
	"sort"
)

// TierRule defines a quantity threshold and the discount percentage applied
// when the ordered quantity meets or exceeds MinQty.
type TierRule struct {
	MinQty  int
	Percent float64 // e.g. 5.0 means 5% off
}

// BulkLineItem is the input for a single SKU in a bulk pricing request.
type BulkLineItem struct {
	SKU       string  `json:"sku"`
	BasePrice float64 `json:"base_price"`
	Quantity  int     `json:"quantity"`
}

// BulkPriceResult is the output for a single SKU after tier evaluation.
type BulkPriceResult struct {
	SKU            string  `json:"sku"`
	Quantity       int     `json:"quantity"`
	BasePrice      float64 `json:"base_price"`
	DiscountPct    float64 `json:"discount_pct"`
	UnitPrice      float64 `json:"unit_price"`
	TotalPrice     float64 `json:"total_price"`
	TierApplied    string  `json:"tier_applied"`
}

// BulkPricingService evaluates quantity-based tier discounts.
type BulkPricingService struct {
	tiers []TierRule
}

// DefaultTiers returns the standard Walmart quantity breakpoints.
func DefaultTiers() []TierRule {
	return []TierRule{
		{MinQty: 1, Percent: 0},
		{MinQty: 10, Percent: 2},
		{MinQty: 50, Percent: 5},
		{MinQty: 100, Percent: 8},
		{MinQty: 500, Percent: 12},
		{MinQty: 1000, Percent: 15},
	}
}

// NewBulkPricingService creates a service with the given tier rules.
// Tiers are sorted by MinQty ascending internally.
func NewBulkPricingService(tiers []TierRule) *BulkPricingService {
	sorted := make([]TierRule, len(tiers))
	copy(sorted, tiers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].MinQty < sorted[j].MinQty
	})
	return &BulkPricingService{tiers: sorted}
}

// Calculate evaluates the tier discount for a batch of line items.
func (s *BulkPricingService) Calculate(_ context.Context, items []BulkLineItem) ([]BulkPriceResult, error) {
	if len(items) == 0 {
		return []BulkPriceResult{}, nil
	}

	results := make([]BulkPriceResult, 0, len(items))
	for _, item := range items {
		if item.Quantity < 1 {
			return nil, fmt.Errorf("invalid quantity %d for SKU %s", item.Quantity, item.SKU)
		}
		if item.BasePrice < 0 {
			return nil, fmt.Errorf("negative base price for SKU %s", item.SKU)
		}

		tier := s.matchTier(item.Quantity)
		unitPrice := item.BasePrice * (1 - tier.Percent/100)
		totalPrice := unitPrice * float64(item.Quantity)

		results = append(results, BulkPriceResult{
			SKU:         item.SKU,
			Quantity:    item.Quantity,
			BasePrice:   item.BasePrice,
			DiscountPct: tier.Percent,
			UnitPrice:   unitPrice,
			TotalPrice:  totalPrice,
			TierApplied: fmt.Sprintf("%d+ units", tier.MinQty),
		})
	}
	return results, nil
}

// matchTier returns the highest tier whose MinQty the quantity meets.
func (s *BulkPricingService) matchTier(qty int) TierRule {
	matched := s.tiers[0]
	for _, t := range s.tiers {
		if qty >= t.MinQty {
			matched = t
		} else {
			break
		}
	}
	return matched
}
