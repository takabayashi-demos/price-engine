package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(60, 5)
	clientIP := "192.168.1.1"

	// Should allow burst requests
	for i := 0; i < 5; i++ {
		if !rl.allow(clientIP) {
			t.Errorf("Request %d should be allowed within burst", i+1)
		}
	}

	// Next request should be denied
	if rl.allow(clientIP) {
		t.Error("Request should be denied after burst exhausted")
	}

	// Wait for token refill (1 request per second at 60 req/min)
	time.Sleep(1100 * time.Millisecond)
	if !rl.allow(clientIP) {
		t.Error("Request should be allowed after token refill")
	}
}

func TestRateLimiterMultipleClients(t *testing.T) {
	rl := newRateLimiter(60, 2)

	// First client exhausts limit
	rl.allow("10.0.0.1")
	rl.allow("10.0.0.1")

	// Second client should still be allowed
	if !rl.allow("10.0.0.2") {
		t.Error("Different client should have independent rate limit")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	pricingLimiter = newRateLimiter(10, 2)

	handler := rateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d should succeed, got status %d", i+1, w.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}

	retryAfter := w.Header().Get("Retry-After")
	if retryAfter != "60" {
		t.Errorf("Expected Retry-After: 60, got %s", retryAfter)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remoteIP string
		expected string
	}{
		{
			name:     "X-Forwarded-For",
			headers:  map[string]string{"X-Forwarded-For": "203.0.113.1"},
			remoteIP: "192.168.1.1:12345",
			expected: "203.0.113.1",
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "198.51.100.1"},
			remoteIP: "192.168.1.1:12345",
			expected: "198.51.100.1",
		},
		{
			name:     "RemoteAddr",
			headers:  map[string]string{},
			remoteIP: "192.168.1.1:12345",
			expected: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteIP
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := getClientIP(req)
			if got != tt.expected {
				t.Errorf("getClientIP() = %v, want %v", got, tt.expected)
			}
		})
	}
}
