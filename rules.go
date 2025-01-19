package main

import (
	"testing"
)

func TestPricingProcess(t *testing.T) {
	svc := NewPricingService()

	t.Run("processes valid request", func(t *testing.T) {
		req := map[string]interface{}{"key": "value"}
		result, err := svc.Process(nil, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result["status"] != "ok" {
			t.Errorf("expected ok, got %v", result["status"])
		}
	})
}

func BenchmarkPricing(b *testing.B) {
	svc := NewPricingService()
	req := map[string]interface{}{"key": "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Process(nil, req)
	}
}


// --- perf: optimize discount query performance ---
package main

import (
	"testing"
)

func TestCalculatorProcess(t *testing.T) {
	svc := NewCalculatorService()

	t.Run("processes valid request", func(t *testing.T) {
		req := map[string]interface{}{"key": "value"}
		result, err := svc.Process(nil, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
