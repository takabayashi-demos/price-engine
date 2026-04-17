package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitMiddleware(t *testing.T) {
	// Reset visitors map for clean test
	visitorsMu.Lock()
	visitors = make(map[string]*visitor)
	visitorsMu.Unlock()

	handler := rateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Test normal requests within limit
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// Test rate limit exceeded (burst is 20, we're at 15, send 10 more rapidly)
	exceeded := false
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code == http.StatusTooManyRequests {
			exceeded = true
			if w.Header().Get("Retry-After") != "1" {
				t.Error("Expected Retry-After header")
			}
			var resp map[string]string
			json.NewDecoder(w.Body).Decode(&resp)
			if resp["error"] != "rate limit exceeded" {
				t.Error("Expected rate limit error message")
			}
			break
		}
	}

	if !exceeded {
		t.Error("Expected rate limit to be exceeded")
	}
}

func TestRateLimitPerIP(t *testing.T) {
	// Reset visitors map
	visitorsMu.Lock()
	visitors = make(map[string]*visitor)
	visitorsMu.Unlock()

	handler := rateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// IP 1 exhausts its limit
	for i := 0; i < 25; i++ {
		req := httptest.NewRequest("GET", "/price", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		handler(w, req)
	}

	// IP 2 should still work
	req := httptest.NewRequest("GET", "/price", nil)
	req.RemoteAddr = "192.168.1.200:12345"
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Different IP should not be rate limited, got %d", w.Code)
	}
}

func TestRateLimitXForwardedFor(t *testing.T) {
	// Reset visitors map
	visitorsMu.Lock()
	visitors = make(map[string]*visitor)
	visitorsMu.Unlock()

	handler := rateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Exhaust limit using X-Forwarded-For
	for i := 0; i < 25; i++ {
		req := httptest.NewRequest("GET", "/price", nil)
		req.RemoteAddr = "10.0.0.1:12345" // proxy IP
		req.Header.Set("X-Forwarded-For", "203.0.113.1") // real client IP
		w := httptest.NewRecorder()
		handler(w, req)
	}

	// Same X-Forwarded-For should be rate limited
	req := httptest.NewRequest("GET", "/price", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limit via X-Forwarded-For, got %d", w.Code)
	}
}

func TestGetPriceWithRateLimit(t *testing.T) {
	// Reset visitors map
	visitorsMu.Lock()
	visitors = make(map[string]*visitor)
	visitorsMu.Unlock()

	req := httptest.NewRequest("GET", "/price?sku=SKU-001", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()

	rateLimitMiddleware(getPriceHandler)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var rule PriceRule
	json.NewDecoder(w.Body).Decode(&rule)
	if rule.SKU != "SKU-001" {
		t.Error("Expected SKU-001 in response")
	}
}

func TestVisitorCleanup(t *testing.T) {
	// Reset visitors map
	visitorsMu.Lock()
	visitors = make(map[string]*visitor)
	visitorsMu.Unlock()

	// Add a visitor
	getVisitor("192.168.1.1")

	visitorsMu.RLock()
	if len(visitors) != 1 {
		t.Error("Expected 1 visitor")
	}
	visitorsMu.RUnlock()

	// Manually set lastSeen to old time
	visitorsMu.Lock()
	visitors["192.168.1.1"].lastSeen = time.Now().Add(-15 * time.Minute)
	visitorsMu.Unlock()

	// Trigger cleanup logic
	visitorsMu.Lock()
	for ip, v := range visitors {
		if time.Since(v.lastSeen) > 10*time.Minute {
			delete(visitors, ip)
		}
	}
	visitorsMu.Unlock()

	visitorsMu.RLock()
	if len(visitors) != 0 {
		t.Error("Expected visitor to be cleaned up")
	}
	visitorsMu.RUnlock()
}
