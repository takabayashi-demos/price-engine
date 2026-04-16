package main

import (
	"encoding/json"
	"net/http"
)

type bulkPricingRequest struct {
	Items []BulkLineItem `json:"items"`
}

type bulkPricingResponse struct {
	Results []BulkPriceResult `json:"results"`
	Count   int               `json:"count"`
}

// RegisterBulkPricingHandler mounts the POST /api/v1/price/bulk endpoint.
func RegisterBulkPricingHandler(mux *http.ServeMux, svc *BulkPricingService) {
	mux.HandleFunc("/api/v1/price/bulk", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req bulkPricingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		results, err := svc.Calculate(r.Context(), req.Items)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bulkPricingResponse{
			Results: results,
			Count:   len(results),
		})
	})
}
