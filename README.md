# Price Engine Service

Dynamic pricing and promotions engine for Walmart's e-commerce platform.

## Overview

The Price Engine service manages product pricing rules, discount calculations, and promotional pricing. It provides real-time price lookups with support for bulk operations and dynamic promo code application.

**Team:** Pricing  
**Service:** `price-engine`  
**Version:** 1.4.2  
**Language:** Go 1.21+  
**Framework:** net/http (stdlib)

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker (optional, for containerized deployment)

### Local Development

```bash
# Clone the repository
git clone https://github.com/takabayashi-demos/price-engine.git
cd price-engine

# Run the service
go run main.go

# Service will start on port 8080
curl http://localhost:8080/health
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

### Docker

```bash
# Build image
docker build -t price-engine:latest .

# Run container
docker run -p 8080:8080 price-engine:latest
```

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

## Architecture

### Components

- **HTTP Server**: Standard library `net/http` server on port 8080
- **In-Memory Cache**: Thread-safe price rule cache with `sync.RWMutex`
- **Price Calculator**: Dynamic pricing engine with discount application

### Data Model

Price rules are stored in-memory and initialized at startup:

```go
type PriceRule struct {
    SKU        string  // Product identifier
    BasePrice  float64 // Original price
    Discount   float64 // Discount percentage
    FinalPrice float64 // Calculated price after discount
    PromoCode  string  // Associated promo code
    ExpiresAt  string  // Expiration timestamp
}
```

### Performance Characteristics

- **Cache Lookup**: O(1) average case
- **Single Price Query**: ~50ms average (includes simulated computation)
- **Bulk Query**: ~10ms per SKU
- **Concurrency**: Read-heavy workload optimized with `RWMutex`

## Known Issues

This service contains intentional issues for demonstration purposes:

1. **Floating-point rounding errors** in price calculations
2. **No rate limiting** on pricing API endpoints
3. **Cache stampede vulnerability** under high load
4. **Promo stacking bug** allows multiple discounts on same SKU
5. **No size limits** on bulk price requests

These issues are tracked and will be addressed in future releases.

## Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Example API test
curl -X POST http://localhost:8080/bulk-price \
  -H "Content-Type: application/json" \
  -d '{"skus": ["SKU-001", "SKU-003"]}'
```

## Deployment

The service is deployed to Kubernetes with the following resources:

- **Deployment**: 3 replicas for high availability
- **Service**: ClusterIP on port 8080
- **Health Probes**: `/health` (liveness) and `/ready` (readiness)
- **Resource Limits**: 256Mi memory, 200m CPU

## Support

For issues or questions, contact the Pricing team or file a ticket in the internal support portal.
