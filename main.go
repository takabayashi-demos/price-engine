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

const (
	// HTTP Status codes
	StatusOK               = 200
	StatusBadRequest       = 400
	StatusNotFound         = 404
	StatusMethodNotAllowed = 405

	// Service configuration
	ServiceName    = "price-engine"
	ServiceVersion = "1.4.2"

	// Timing constants
	MaxPriceComputeLatency = 50 * time.Millisecond
	BulkItemComputeLatency = 10 * time.Millisecond

	// Promo constants
	ExtraPromoDiscount = 10.0
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
	logger       *log.Logger
)

func init() {
	logger = log.New(os.Stdout, "[price-engine] ", log.LstdFlags|log.Lmsgprefix)

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
	logger.Printf("Loaded %d price rules into cache", len(priceCache))
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Printf("ERROR: failed to encode JSON response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, StatusOK, map[string]string{
		"status":  "UP",
		"service": ServiceName,
		"version": ServiceVersion,
	})
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, StatusOK, map[string]string{"status": "READY"})
}

func getPriceHandler(w http.ResponseWriter, r *http.Request) {
	sku := r.URL.Query().Get("sku")
	if sku == "" {
		logger.Printf("WARN: getPriceHandler called without SKU")
		respondError(w, StatusBadRequest, "sku required")
		return
	}

	// ❌ BUG: No rate limiting — can be abused for price scraping
	requestCount++

	cacheMu.RLock()
	rule, exists := priceCache[sku]
	cacheMu.RUnlock()

	if !exists {
		logger.Printf("WARN: SKU not found: %s", sku)
		respondError(w, StatusNotFound, "product not found")
		return
	}

	// Simulate pricing engine computation
	time.Sleep(time.Duration(rand.Intn(int(MaxPriceComputeLatency))))

	logger.Printf("INFO: Price retrieved for SKU=%s, price=%.2f", sku, rule.FinalPrice)
	respondJSON(w, StatusOK, rule)
}

func bulkPriceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		respondError(w, StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		SKUs []string `json:"skus"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Printf("ERROR: failed to decode bulk price request: %v", err)
		respondError(w, StatusBadRequest, "invalid request body")
		return
	}

	// ❌ LATENCY: No limit on bulk request size
	results := make([]*PriceRule, 0)
	cacheMu.RLock()
	for _, sku := range req.SKUs {
		if rule, ok := priceCache[sku]; ok {
			results = append(results, rule)
		}
		// Simulate per-item computation
		time.Sleep(BulkItemComputeLatency)
	}
	cacheMu.RUnlock()

	logger.Printf("INFO: Bulk price request processed: %d SKUs requested, %d found", len(req.SKUs), len(results))
	respondJSON(w, StatusOK, map[string]interface{}{
		"prices": results,
		"total":  len(results),
	})
}

func applyPromoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		respondError(w, StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		SKU   string `json:"sku"`
		Promo string `json:"promo_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Printf("ERROR: failed to decode promo request: %v", err)
		respondError(w, StatusBadRequest, "invalid request body")
		return
	}

	cacheMu.RLock()
	rule, exists := priceCache[req.SKU]
	cacheMu.RUnlock()

	if !exists {
		logger.Printf("WARN: Promo applied to non-existent SKU: %s", req.SKU)
		respondError(w, StatusNotFound, "product not found")
		return
	}

	// ❌ BUG: Promo stacking - doesn't check if promo already applied
	newPrice := rule.FinalPrice * (1 - ExtraPromoDiscount/100)

	logger.Printf("INFO: Promo applied: SKU=%s, promo=%s, new_price=%.2f", req.SKU, req.Promo, newPrice)
	respondJSON(w, StatusOK, map[string]interface{}{
		"sku":            req.SKU,
		"original_price": rule.BasePrice,
		"promo_price":    math.Round(newPrice*100) / 100,
		"savings":        math.Round((rule.BasePrice-newPrice)*100) / 100,
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)
	http.HandleFunc("/price", getPriceHandler)
	http.HandleFunc("/bulk-price", bulkPriceHandler)
	http.HandleFunc("/apply-promo", applyPromoHandler)

	addr := fmt.Sprintf(":%s", port)
	logger.Printf("Starting %s v%s on %s", ServiceName, ServiceVersion, addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}
