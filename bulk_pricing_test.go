package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBulkPricingCalculate(t *testing.T) {
	svc := NewBulkPricingService(DefaultTiers())

	t.Run("single item no discount", func(t *testing.T) {
		items := []BulkLineItem{{SKU: "WMT-001", BasePrice: 10.00, Quantity: 5}}
		results, err := svc.Calculate(context.Background(), items)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].DiscountPct != 0 {
			t.Errorf("expected 0%% discount for qty 5, got %.1f%%", results[0].DiscountPct)
		}
		if results[0].UnitPrice != 10.00 {
			t.Errorf("expected unit price 10.00, got %.2f", results[0].UnitPrice)
		}
	})

	t.Run("tier boundary at 50 units", func(t *testing.T) {
		items := []BulkLineItem{{SKU: "WMT-002", BasePrice: 20.00, Quantity: 50}}
		results, err := svc.Calculate(context.Background(), items)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results[0].DiscountPct != 5.0 {
			t.Errorf("expected 5%% discount at qty 50, got %.1f%%", results[0].DiscountPct)
		}
		expectedUnit := 20.00 * 0.95
		if results[0].UnitPrice != expectedUnit {
			t.Errorf("expected unit price %.2f, got %.2f", expectedUnit, results[0].UnitPrice)
		}
	})

	t.Run("highest tier at 1000 units", func(t *testing.T) {
		items := []BulkLineItem{{SKU: "WMT-003", BasePrice: 5.00, Quantity: 1500}}
		results, err := svc.Calculate(context.Background(), items)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results[0].DiscountPct != 15.0 {
			t.Errorf("expected 15%% discount at qty 1500, got %.1f%%", results[0].DiscountPct)
		}
	})

	t.Run("multiple items", func(t *testing.T) {
		items := []BulkLineItem{
			{SKU: "WMT-A", BasePrice: 10.00, Quantity: 5},
			{SKU: "WMT-B", BasePrice: 10.00, Quantity: 100},
		}
		results, err := svc.Calculate(context.Background(), items)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0].DiscountPct != 0 {
			t.Errorf("item A: expected 0%% discount, got %.1f%%", results[0].DiscountPct)
		}
		if results[1].DiscountPct != 8.0 {
			t.Errorf("item B: expected 8%% discount, got %.1f%%", results[1].DiscountPct)
		}
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		results, err := svc.Calculate(context.Background(), []BulkLineItem{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected empty results, got %d", len(results))
		}
	})

	t.Run("invalid quantity returns error", func(t *testing.T) {
		items := []BulkLineItem{{SKU: "BAD", BasePrice: 10.00, Quantity: 0}}
		_, err := svc.Calculate(context.Background(), items)
		if err == nil {
			t.Fatal("expected error for zero quantity")
		}
	})

	t.Run("negative price returns error", func(t *testing.T) {
		items := []BulkLineItem{{SKU: "BAD", BasePrice: -1.00, Quantity: 10}}
		_, err := svc.Calculate(context.Background(), items)
		if err == nil {
			t.Fatal("expected error for negative price")
		}
	})
}

func TestBulkPricingHandler(t *testing.T) {
	svc := NewBulkPricingService(DefaultTiers())
	mux := http.NewServeMux()
	RegisterBulkPricingHandler(mux, svc)

	t.Run("POST returns tiered prices", func(t *testing.T) {
		body := `{"items":[{"sku":"WMT-001","base_price":9.99,"quantity":150}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/price/bulk", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp bulkPricingResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Count != 1 {
			t.Errorf("expected count 1, got %d", resp.Count)
		}
		if resp.Results[0].DiscountPct != 8.0 {
			t.Errorf("expected 8%% discount for qty 150, got %.1f%%", resp.Results[0].DiscountPct)
		}
	})

	t.Run("GET returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/price/bulk", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("malformed JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/price/bulk", bytes.NewBufferString("{bad"))
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})
}

func BenchmarkBulkPricing(b *testing.B) {
	svc := NewBulkPricingService(DefaultTiers())
	items := []BulkLineItem{
		{SKU: "WMT-001", BasePrice: 9.99, Quantity: 150},
		{SKU: "WMT-002", BasePrice: 24.99, Quantity: 500},
		{SKU: "WMT-003", BasePrice: 4.99, Quantity: 25},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Calculate(context.Background(), items)
	}
}
