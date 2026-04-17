package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkGetPrice(b *testing.B) {
	req := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getPriceHandler(w, req)
	}
}

func BenchmarkBulkPrice_10Items(b *testing.B) {
	body := map[string]interface{}{
		"skus": []string{"SKU-001", "SKU-002", "SKU-003", "SKU-004", "SKU-005",
			"SKU-006", "SKU-007", "SKU-008", "SKU-001", "SKU-002"},
	}
	jsonBody, _ := json.Marshal(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/bulk-price", bytes.NewReader(jsonBody))
		w := httptest.NewRecorder()
		bulkPriceHandler(w, req)
	}
}

func BenchmarkBulkPrice_50Items(b *testing.B) {
	skus := make([]string, 50)
	for i := 0; i < 50; i++ {
		skus[i] = "SKU-001"
	}
	body := map[string]interface{}{"skus": skus}
	jsonBody, _ := json.Marshal(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/bulk-price", bytes.NewReader(jsonBody))
		w := httptest.NewRecorder()
		bulkPriceHandler(w, req)
	}
}

func BenchmarkBulkPrice_100Items(b *testing.B) {
	skus := make([]string, 100)
	for i := 0; i < 100; i++ {
		skus[i] = "SKU-001"
	}
	body := map[string]interface{}{"skus": skus}
	jsonBody, _ := json.Marshal(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/bulk-price", bytes.NewReader(jsonBody))
		w := httptest.NewRecorder()
		bulkPriceHandler(w, req)
	}
}

func BenchmarkApplyPromo(b *testing.B) {
	body := map[string]interface{}{
		"sku":        "SKU-001",
		"promo_code": "SAVE10",
	}
	jsonBody, _ := json.Marshal(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/apply-promo", bytes.NewReader(jsonBody))
		w := httptest.NewRecorder()
		applyPromoHandler(w, req)
	}
}

func TestBulkPriceReturnsCorrectCount(t *testing.T) {
	body := map[string]interface{}{
		"skus": []string{"SKU-001", "SKU-002", "SKU-999"},
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/bulk-price", bytes.NewReader(jsonBody))
	w := httptest.NewRecorder()

	bulkPriceHandler(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"].(float64) != 2 {
		t.Errorf("expected 2 results, got %v", resp["total"])
	}
}
