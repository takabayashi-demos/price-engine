# price-engine

Dynamic pricing and promotions engine

## Tech Stack
- **Language**: go
- **Team**: pricing
- **Platform**: Walmart Global K8s

## Quick Start
```bash
docker build -t price-engine:latest .
docker run -p 8080:8080 price-engine:latest
curl http://localhost:8080/health
```

## API Endpoints
| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |
| GET | /ready | Readiness probe |
| GET | /metrics | Prometheus metrics |
# PR 1 - 2026-04-15T18:47:58
