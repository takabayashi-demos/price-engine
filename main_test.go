package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiting(t *testing.T) {
	// Create multiple requests from the same IP
	req := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First 10 requests should succeed (burst allowance)
	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		handler := rateLimitMiddleware(getPriceHandler)
		handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i+1, rr.Code)
		}
	}

	// 11th request should be rate limited
	rr := httptest.NewRecorder()
	handler := rateLimitMiddleware(getPriceHandler)
	handler(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limit (429), got %d", rr.Code)
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["error"] != "rate limit exceeded" {
		t.Errorf("Expected rate limit error message, got %v", resp)
	}
}

func TestRateLimitingDifferentIPs(t *testing.T) {
	req1 := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
	req1.RemoteAddr = "192.168.1.1:12345"

	req2 := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
	req2.RemoteAddr = "192.168.1.2:12345"

	handler := rateLimitMiddleware(getPriceHandler)

	// Each IP should have its own rate limit
	for i := 0; i < 10; i++ {
		rr1 := httptest.NewRecorder()
		handler(rr1, req1)
		if rr1.Code != http.StatusOK {
			t.Errorf("IP1 Request %d failed: %d", i+1, rr1.Code)
		}

		rr2 := httptest.NewRecorder()
		handler(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Errorf("IP2 Request %d failed: %d", i+1, rr2.Code)
		}
	}
}

func TestXForwardedForParsing(t *testing.T) {
	tests := []struct {
		header   string
		expected string
	}{
		{"203.0.113.1", "203.0.113.1"},
		{"203.0.113.1, 70.41.3.18", "203.0.113.1"},
		{"  203.0.113.1  ,  70.41.3.18  ", "203.0.113.1"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
		req.Header.Set("X-Forwarded-For", tt.header)

		ip := getIPAddress(req)
		if ip != tt.expected {
			t.Errorf("X-Forwarded-For %q: expected %q, got %q", tt.header, tt.expected, ip)
		}
	}
}

func TestGetPriceBasic(t *testing.T) {
	req := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
	rr := httptest.NewRecorder()

	getPriceHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var rule PriceRule
	json.NewDecoder(rr.Body).Decode(&rule)

	if rule.SKU != "SKU-001" {
		t.Errorf("Expected SKU-001, got %s", rule.SKU)
	}
}

func TestGetPriceMissingSKU(t *testing.T) {
	req := httptest.NewRequest("GET", "/price", nil)
	rr := httptest.NewRecorder()

	getPriceHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}
