// Price Engine - Walmart Platform
// Dynamic pricing and promotions with intentional issues.
//
// INTENTIONAL ISSUES (for demo):
// - Floating point rounding errors (bug)
// - Cache stampede vulnerability (bug)
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type PriceRule struct {
	SKU        string  `json:"sku"`
	BasePrice  float64 `json:"base_price"`
	Discount   float64 `json:"discount_pct"`
	FinalPrice float64 `json:"final_price"`
	PromoCode  string  `json:"promo_code,omitempty"`
	ExpiresAt  string  `json:"expires_at,omitempty"`
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	priceCache   = make(map[string]*PriceRule)
	cacheMu      sync.RWMutex
	requestCount int64
	visitors     = make(map[string]*visitor)
	visitorsMu   sync.RWMutex
)

func init() {
	rules := []PriceRule{
		{SKU: "SKU-001", BasePrice: 599.99, Discount: 15, PromoCode: "TV15OFF"},
		{SKU: "SKU-002", BasePrice: 999.99, Discount: 0},
		{SKU: "SKU-003", BasePrice: 349.99, Discount: 20, PromoCode: "AUDIO20"},
		{SKU: "SKU-004", BasePrice: 349.99, Discount: 10, PromoCode: "GAME10"},
		{SKU: "SKU-005", BasePrice: 749.99, Discount: 5},
		{SKU: "SKU-006", BasePrice: 429.99, Discount: 25, PromoCode: "KITCHEN25"},
		{SKU: "SKU-007", BasePrice: 89.99, Discount: 30, PromoCode: "FLASH30"},
		{SKU: "SKU-008", BasePrice: 159.99, Discount: 0},
	}
	for i := range rules {
		r := rules[i]
		// ❌ BUG: Floating point rounding error
		r.FinalPrice = r.BasePrice * (1 - r.Discount/100)
		// This produces values like 509.9915 instead of 509.99
		priceCache[r.SKU] = &r
	}

	// Start cleanup goroutine for stale visitors
	go cleanupVisitors()
}

func getVisitor(ip string) *rate.Limiter {
	visitorsMu.Lock()
	defer visitorsMu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		// 10 requests per second with burst of 20
		limiter := rate.NewLimiter(10, 20)
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func cleanupVisitors() {
	for {
		time.Sleep(5 * time.Minute)
		visitorsMu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(visitors, ip)
			}
		}
		visitorsMu.Unlock()
	}
}

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		// Check X-Forwarded-For for proxied requests
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}

		limiter := getVisitor(ip)
		if !limiter.Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status": "UP", "service": "price-engine", "version": "1.4.2",
	})
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "READY"})
}

func getPriceHandler(w http.ResponseWriter, r *http.Request) {
	sku := r.URL.Query().Get("sku")
	if sku == "" {
		http.Error(w, `{"error":"sku required"}`, 400)
		return
	}

	requestCount++

	cacheMu.RLock()
	rule, exists := priceCache[sku]
	cacheMu.RUnlock()

	if !exists {
		http.Error(w, `{"error":"product not found"}`, 404)
		return
	}

	// Simulate pricing engine computation
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

func bulkPriceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		SKUs []string `json:"skus"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// ❌ LATENCY: No limit on bulk request size
	results := make([]*PriceRule, 0)
	cacheMu.RLock()
	for _, sku := range req.SKUs {
		if rule, ok := priceCache[sku]; ok {
			results = append(results, rule)
		}
		// Simulate per-item computation
		time.Sleep(10 * time.Millisecond)
	}
	cacheMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"prices": results,
		"total":  len(results),
	})
}

func applyPromoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		SKU   string `json:"sku"`
		Promo string `json:"promo_code"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	cacheMu.RLock()
	rule, exists := priceCache[req.SKU]
	cacheMu.RUnlock()

	if !exists {
		http.Error(w, `{"error":"product not found"}`, 404)
		return
	}

	// ❌ BUG: Promo stacking - doesn't check if promo already applied
	extraDiscount := 10.0
	newPrice := rule.FinalPrice * (1 - extraDiscount/100)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sku":            req.SKU,
		"original_price": rule.BasePrice,
		"promo_price":    math.Round(newPrice*100) / 100,
		"savings":        math.Round((rule.BasePrice-newPrice)*100) / 100,
		"promo_code":     req.Promo,
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests_total": requestCount,
		"cache_size":     len(priceCache),
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)
	http.HandleFunc("/price", rateLimitMiddleware(getPriceHandler))
	http.HandleFunc("/bulk-price", rateLimitMiddleware(bulkPriceHandler))
	http.HandleFunc("/apply-promo", applyPromoHandler)
	http.HandleFunc("/metrics", metricsHandler)

	log.Printf("Price Engine starting on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
