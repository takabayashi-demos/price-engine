# Price Engine Service

Dynamic pricing and promotions engine for Walmart's e-commerce platform.

## Overview

The Price Engine service manages product pricing rules, discount calculations, and promotional pricing. It provides real-time price lookups with support for bulk operations and dynamic promo code application.

**Team:** Pricing  
**Service:** `price-engine`  
**Version:** 1.4.2  
**Language:** Go 1.21+  
**Framework:** net/http (stdlib)

## API Endpoints

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "UP",
  "service": "price-engine",
  "version": "1.4.2"
}
```

### Readiness Check

```http
GET /ready
```

**Response:**
```json
{
  "status": "READY"
}
```

### Get Price

Retrieve pricing information for a single SKU.

```http
GET /price?sku=SKU-001
```

**Query Parameters:**
- `sku` (required): Product SKU identifier

**Response:**
```json
{
  "sku": "SKU-001",
  "base_price": 599.99,
  "discount_pct": 15,
  "final_price": 509.99,
  "promo_code": "TV15OFF",
  "expires_at": ""
}
```

**Error Responses:**
- `400` - Missing SKU parameter
- `404` - Product not found

### Bulk Price Lookup

Retrieve pricing for multiple SKUs in a single request.

```http
POST /bulk-price
Content-Type: application/json

{
  "skus": ["SKU-001", "SKU-002", "SKU-003"]
}
```

**Response:**
```json
{
  "prices": [
    {
      "sku": "SKU-001",
      "base_price": 599.99,
      "discount_pct": 15,
      "final_price": 509.99,
      "promo_code": "TV15OFF"
    }
  ],
  "total": 1
}
```

### Apply Promo Code

Calculate price with additional promotional discount.

```http
POST /apply-promo
Content-Type: application/json

{
  "sku": "SKU-001",
  "promo_code": "EXTRA10"
}
```

**Response:**
```json
{
  "sku": "SKU-001",
  "original_price": 599.99,
  "promo_price": 458.99,
  "savings": 141.0
}
```
