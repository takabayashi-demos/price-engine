package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "UP" {
		t.Errorf("expected status UP, got %s", resp["status"])
	}
	if resp["service"] != "price-engine" {
		t.Errorf("expected service price-engine, got %s", resp["service"])
	}
}

func TestReadyHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	readyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["status"] != "READY" {
		t.Errorf("expected status READY, got %s", resp["status"])
	}
}

func TestGetPriceHandler(t *testing.T) {
	tests := []struct {
		name           string
		sku            string
		expectedCode   int
		expectedSKU    string
		expectError    bool
	}{
		{
			name:         "valid SKU-001",
			sku:          "SKU-001",
			expectedCode: http.StatusOK,
			expectedSKU:  "SKU-001",
			expectError:  false,
		},
		{
			name:         "valid SKU-003",
			sku:          "SKU-003",
			expectedCode: http.StatusOK,
			expectedSKU:  "SKU-003",
			expectError:  false,
		},
		{
			name:         "invalid SKU",
			sku:          "SKU-999",
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		{
			name:         "missing SKU parameter",
			sku:          "",
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/price"
			if tt.sku != "" {
				url += "?sku=" + tt.sku
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			getPriceHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if !tt.expectError {
				var rule PriceRule
				json.NewDecoder(w.Body).Decode(&rule)
				if rule.SKU != tt.expectedSKU {
					t.Errorf("expected SKU %s, got %s", tt.expectedSKU, rule.SKU)
				}
				if rule.BasePrice <= 0 {
					t.Error("expected positive base price")
				}
			}
		})
	}
}

func TestBulkPriceHandler(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		skus          []string
		expectedCode  int
		expectedCount int
	}{
		{
			name:          "valid multiple SKUs",
			method:        "POST",
			skus:          []string{"SKU-001", "SKU-002", "SKU-003"},
			expectedCode:  http.StatusOK,
			expectedCount: 3,
		},
		{
			name:          "mixed valid and invalid SKUs",
			method:        "POST",
			skus:          []string{"SKU-001", "SKU-999", "SKU-003"},
			expectedCode:  http.StatusOK,
			expectedCount: 2,
		},
		{
			name:          "empty SKU list",
			method:        "POST",
			skus:          []string{},
			expectedCode:  http.StatusOK,
			expectedCount: 0,
		},
		{
			name:         "invalid method GET",
			method:       "GET",
			skus:         []string{"SKU-001"},
			expectedCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]interface{}{"skus": tt.skus})
			req := httptest.NewRequest(tt.method, "/bulk-price", bytes.NewReader(body))
			w := httptest.NewRecorder()

			bulkPriceHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectedCode == http.StatusOK {
				var resp map[string]interface{}
				json.NewDecoder(w.Body).Decode(&resp)
				total := int(resp["total"].(float64))
				if total != tt.expectedCount {
					t.Errorf("expected count %d, got %d", tt.expectedCount, total)
				}
			}
		})
	}
}

func TestApplyPromoHandler(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		sku          string
		promoCode    string
		expectedCode int
	}{
		{
			name:         "valid promo application",
			method:       "POST",
			sku:          "SKU-001",
			promoCode:    "EXTRA10",
			expectedCode: http.StatusOK,
		},
		{
			name:         "invalid SKU",
			method:       "POST",
			sku:          "SKU-999",
			promoCode:    "EXTRA10",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "invalid method GET",
			method:       "GET",
			sku:          "SKU-001",
			promoCode:    "EXTRA10",
			expectedCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"sku":        tt.sku,
				"promo_code": tt.promoCode,
			})
			req := httptest.NewRequest(tt.method, "/apply-promo", bytes.NewReader(body))
			w := httptest.NewRecorder()

			applyPromoHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectedCode == http.StatusOK {
				var resp map[string]interface{}
				json.NewDecoder(w.Body).Decode(&resp)
				if resp["sku"] != tt.sku {
					t.Errorf("expected SKU %s, got %s", tt.sku, resp["sku"])
				}
				if _, ok := resp["promo_price"]; !ok {
					t.Error("expected promo_price in response")
				}
			}
		})
	}
}
