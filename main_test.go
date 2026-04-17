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
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "valid SKU",
			sku:            "SKU-001",
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "missing SKU parameter",
			sku:            "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:           "non-existent SKU",
			sku:            "SKU-999",
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
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

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if !tt.expectedError {
				var rule PriceRule
				err := json.NewDecoder(w.Body).Decode(&rule)
				if err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if rule.SKU != tt.sku {
					t.Errorf("expected SKU %s, got %s", tt.sku, rule.SKU)
				}
				if rule.BasePrice <= 0 {
					t.Errorf("expected positive base price, got %f", rule.BasePrice)
				}
			}
		})
	}
}

func TestBulkPriceHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		skus           []string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "valid bulk request",
			method:         "POST",
			skus:           []string{"SKU-001", "SKU-002", "SKU-003"},
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "empty SKU list",
			method:         "POST",
			skus:           []string{},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "mixed valid and invalid SKUs",
			method:         "POST",
			skus:           []string{"SKU-001", "SKU-999", "SKU-003"},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "wrong HTTP method",
			method:         "GET",
			skus:           []string{"SKU-001"},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string][]string{"skus": tt.skus})
			req := httptest.NewRequest(tt.method, "/bulk-price", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			bulkPriceHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.NewDecoder(w.Body).Decode(&resp)
				total := int(resp["total"].(float64))
				if total != tt.expectedCount {
					t.Errorf("expected %d results, got %d", tt.expectedCount, total)
				}
			}
		})
	}
}

func TestApplyPromoHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		sku            string
		promo          string
		expectedStatus int
	}{
		{
			name:           "valid promo application",
			method:         "POST",
			sku:            "SKU-001",
			promo:          "EXTRA10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-existent SKU",
			method:         "POST",
			sku:            "SKU-999",
			promo:          "EXTRA10",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "wrong HTTP method",
			method:         "GET",
			sku:            "SKU-001",
			promo:          "EXTRA10",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"sku":        tt.sku,
				"promo_code": tt.promo,
			})
			req := httptest.NewRequest(tt.method, "/apply-promo", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			applyPromoHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.NewDecoder(w.Body).Decode(&resp)
				if resp["sku"] != tt.sku {
					t.Errorf("expected SKU %s, got %v", tt.sku, resp["sku"])
				}
				if _, ok := resp["promo_price"]; !ok {
					t.Error("expected promo_price in response")
				}
				if _, ok := resp["savings"]; !ok {
					t.Error("expected savings in response")
				}
			}
		})
	}
}
