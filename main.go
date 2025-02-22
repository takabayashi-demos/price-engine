// Price Engine - Walmart Platform
// Dynamic pricing and promotions with intentional issues.
//
// INTENTIONAL ISSUES (for demo):
// - Floating point rounding errors (bug)
// - Cache stampede vulnerability (bug)
// - No rate limiting on pricing API (vulnerability)
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type PriceRule struct {
	SKU        string  `json:"sku"`
	BasePrice  float64 `json:"base_price"`
	Discount   float64 `json:"discount_pct"`
	FinalPrice float64 `json:"final_price"`
	PromoCode  string  `json:"promo_code,omitempty"`
	ExpiresAt  string  `json:"expires_at,omitempty"`
}

var (
	priceCache   = make(map[string]*PriceRule)
	cacheMu      sync.RWMutex
	requestCount int64
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

	// ❌ BUG: No rate limiting — can be abused for price scraping
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
		"promo_applied":  req.Promo,
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `# HELP price_requests_total Total pricing requests
# TYPE price_requests_total counter
price_requests_total %d
# HELP price_cache_size Number of cached price rules
# TYPE price_cache_size gauge
price_cache_size %d
# HELP price_service_up Service health
# TYPE price_service_up gauge
price_service_up 1
`, requestCount, len(priceCache))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)
	http.HandleFunc("/api/v1/price", getPriceHandler)
	http.HandleFunc("/api/v1/price/bulk", bulkPriceHandler)
	http.HandleFunc("/api/v1/price/promo", applyPromoHandler)
	http.HandleFunc("/metrics", metricsHandler)

	log.Printf("price-engine starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
// Decimal precision
// Flash sale engine
// Price match logic
// Cache tiers
// Floor price check
// Analytics export
// Rate limiter
