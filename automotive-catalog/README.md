# Automotive Catalog & Product Synchronization Platform

A scalable Go microservices platform for managing millions of automotive product and fitment records, with event-driven synchronization across enterprise data sources.

**Stack:** Go · Gin · PostgreSQL · Oracle · SQL Server · Apache Kafka · Redis · Docker · Kubernetes · AWS

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [API Reference](#api-reference)
- [Configuration](#configuration)
- [Database](#database)
- [Kafka Events](#kafka-events)
- [Background Workers](#background-workers)
- [Deployment](#deployment)

---

## Overview

This platform provides:

- **Catalog Management** — full lifecycle CRUD for automotive parts and fitment (ACES-standard) data
- **Vehicle Search** — find all compatible products by year/make/model
- **Event-Driven Sync** — Kafka producers and consumers keep product, fitment, and inventory data in sync across systems
- **ETL Pipelines** — concurrent fan-out workers for high-volume batch imports and data transformation
- **Multi-Database** — PostgreSQL as the primary store, with SQL Server and Oracle adapters for enterprise integrations
- **Caching** — Redis cache-aside layer for hot product and fitment lookups

---

## Architecture

```
                        ┌─────────────────────────────────────────┐
                        │              Kubernetes Cluster          │
                        │                                          │
  Client ──────────────►│  Ingress ──► catalog-api (3–20 pods)    │
                        │                    │                     │
                        │             catalog-worker (2 pods)      │
                        │                    │                     │
                        └────────────────────┼────────────────────┘
                                             │
               ┌─────────────────────────────┼──────────────────────────┐
               │                             │                          │
          PostgreSQL                       Redis                      Kafka
        (primary store)                  (cache)               (event streaming)
               │                                                       │
        SQL Server / Oracle                               Inventory · Products · Fitments
        (enterprise integrations)                              (topics)
```

### Services

| Service | Description | Replicas |
|---|---|---|
| `catalog-api` | RESTful HTTP API (Gin) | 3–20 (HPA) |
| `catalog-worker` | Background ETL + scheduled jobs | 2 |

---

## Project Structure

```
automotive-catalog/
├── cmd/
│   ├── api/main.go              # API server entry point
│   └── worker/main.go           # Background worker entry point
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   │   ├── catalog.go       # Product & fitment CRUD handlers
│   │   │   └── search.go        # Vehicle search & health check
│   │   ├── middleware/
│   │   │   ├── auth.go          # JWT Bearer auth + role-based access
│   │   │   └── logger.go        # Structured request logging
│   │   └── routes/routes.go     # Route registration
│   ├── catalog/
│   │   ├── model/
│   │   │   ├── product.go       # Product, Inventory, Category types
│   │   │   └── fitment.go       # Fitment (ACES), Vehicle types
│   │   ├── repository/
│   │   │   ├── product.go       # Postgres CRUD + bulk upsert via CopyFrom
│   │   │   └── fitment.go       # Upsert, list, vehicle search queries
│   │   └── service/
│   │       └── product.go       # Business logic + cache-aside + Kafka publish
│   ├── database/
│   │   ├── postgres.go          # pgxpool connection & health check
│   │   ├── sqlserver.go         # SQL Server connection
│   │   └── oracle.go            # Oracle connection
│   ├── cache/
│   │   └── redis.go             # Redis get/set/delete/pattern invalidation
│   ├── kafka/
│   │   ├── producer.go          # Multi-topic producer (keyed/hash partitioning)
│   │   └── consumer.go          # Per-topic concurrent consumers
│   └── etl/
│       ├── pipeline.go          # Fan-out worker pool for bulk imports
│       └── worker.go            # Scheduled background jobs (errgroup + tickers)
├── pkg/
│   ├── config/config.go         # Viper config (env vars + .env)
│   ├── logger/logger.go         # Zap structured logger
│   └── errors/errors.go         # Typed AppError with HTTP codes
├── migrations/
│   └── 001_init.sql             # Full schema with indexes and outbox table
├── deployments/
│   └── k8s/
│       ├── namespace.yaml
│       ├── configmap.yaml
│       ├── secrets.yaml         # Template only — use Vault/Sealed Secrets in prod
│       ├── api-deployment.yaml  # Deployment + Service + HPA
│       └── worker-deployment.yaml
├── Dockerfile                   # Multi-stage scratch image (api or worker via ARG)
├── docker-compose.yml           # Full local stack
└── .env.example
```

---

## Getting Started

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Docker + Docker Compose](https://docs.docker.com/get-docker/)

### Run locally

```bash
# 1. Clone and enter the project
cd automotive-catalog

# 2. Copy environment config
cp .env.example .env

# 3. Start all infrastructure + services
docker compose up --build

# 4. Verify the API is running
curl http://localhost:8080/health
```

The full stack starts:
- **API** at `http://localhost:8080`
- **PostgreSQL** at `localhost:5432`
- **Redis** at `localhost:6379`
- **Kafka** at `localhost:9092`

### Run without Docker

```bash
# Start infrastructure only
docker compose up postgres redis kafka -d

# Run API
go run ./cmd/api

# Run worker (separate terminal)
go run ./cmd/worker
```

### Build binaries

```bash
# API
go build -o bin/api ./cmd/api

# Worker
go build -o bin/worker ./cmd/worker
```

---

## API Reference

All endpoints are prefixed with `/api/v1` and require a `Bearer` token in the `Authorization` header.

### Products

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/products` | Create a product |
| `GET` | `/api/v1/products` | List products (paginated, filterable) |
| `GET` | `/api/v1/products/:id` | Get product by ID |
| `PUT` | `/api/v1/products/:id` | Update a product |
| `DELETE` | `/api/v1/products/:id` | Soft-delete a product *(admin role required)* |

### Fitments

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/products/:id/fitments` | Upsert a fitment record |
| `GET` | `/api/v1/products/:id/fitments` | List fitments for a product |

### Search

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/search/vehicle?year=&make=&model=` | Find compatible products for a vehicle |
| `GET` | `/health` | Health check (no auth required) |

### Query parameters for `GET /api/v1/products`

| Param | Type | Description |
|-------|------|-------------|
| `q` | string | Full-text search (name + description) |
| `brand` | string | Filter by brand |
| `category_id` | string | Filter by category |
| `status` | string | `active` \| `inactive` \| `discontinued` |
| `page` | int | Page number (default: 1) |
| `page_size` | int | Results per page (default: 50, max: 200) |
| `sort_by` | string | `name` \| `created_at` \| `part_number` \| `brand` \| `msrp` |
| `sort_order` | string | `asc` \| `desc` |

### Example requests

```bash
# Create product
curl -X POST http://localhost:8080/api/v1/products \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "part_number": "BRK-12345",
    "brand": "Bosch",
    "name": "Disc Brake Pad Set",
    "category_id": "cat-brakes-001",
    "msrp": 49.99,
    "cost": 22.50
  }'

# Vehicle search
curl "http://localhost:8080/api/v1/search/vehicle?year=2021&make=Toyota&model=Camry" \
  -H "Authorization: Bearer <token>"
```

---

## Configuration

All configuration is read from environment variables (with `.env` file support via Viper).

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `JWT_SECRET` | — | Secret key for JWT validation |
| `POSTGRES_DSN` | — | PostgreSQL connection string |
| `POSTGRES_MAX_OPEN_CONNS` | `25` | Max open DB connections |
| `MSSQL_DSN` | — | SQL Server connection string |
| `ORACLE_DSN` | — | Oracle connection string |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_TTL` | `15m` | Default cache TTL |
| `KAFKA_BROKERS` | — | Comma-separated broker list |
| `KAFKA_PRODUCT_TOPIC` | `automotive.products` | Product events topic |
| `KAFKA_INVENTORY_TOPIC` | `automotive.inventory` | Inventory events topic |
| `KAFKA_FITMENT_TOPIC` | `automotive.fitments` | Fitment events topic |
| `KAFKA_CONSUMER_GROUP` | `catalog-service` | Consumer group ID |
| `AWS_REGION` | `us-east-1` | AWS region |
| `AWS_S3_BUCKET` | — | S3 bucket for assets |

See `.env.example` for a full template.

---

## Database

### Schema overview

| Table | Description |
|-------|-------------|
| `products` | Core product catalog with JSONB attributes |
| `products_staging` | Temp table for high-speed bulk upserts |
| `categories` | Hierarchical product categories |
| `vehicles` | Unique year/make/model/engine combinations |
| `fitments` | Product-to-vehicle application records (ACES) |
| `inventory` | Per-warehouse stock levels |
| `outbox_events` | Transactional outbox for reliable Kafka publishing |

### Migrations

Migrations are plain SQL files in `migrations/`. They run automatically in Docker Compose via `docker-entrypoint-initdb.d`.

For production, integrate with [golang-migrate](https://github.com/golang-migrate/migrate):

```bash
migrate -path migrations -database "$POSTGRES_DSN" up
```

### Key indexes

- Full-text search on `products(name, description)` via `GIN` + `tsvector`
- GIN index on `products(attributes)` for JSONB queries
- Composite index on `fitments(year, make, model)` for vehicle lookups
- Partial indexes on all hot queries filtering `deleted_at IS NULL`

---

## Kafka Events

### Topics

| Topic | Key | Events |
|-------|-----|--------|
| `automotive.products` | `product_id` | `product.created`, `product.updated`, `product.deleted` |
| `automotive.fitments` | `product_id` | `fitment.upserted` |
| `automotive.inventory` | `product_id` | `inventory.sync` |

### Event envelope

```json
{
  "type": "product.updated",
  "timestamp": "2026-09-02T10:00:00Z",
  "payload": { ... }
}
```

### Consumer registration

```go
consumer.Register(kafka.EventProductUpdated, func(ctx context.Context, event kafka.Event) error {
    // handle event
    return nil
})
consumer.Start(ctx) // launches one goroutine per topic
```

---

## Background Workers

The worker binary runs scheduled ETL jobs via the `BackgroundWorker`. Each job runs on its own goroutine with an independent ticker.

| Job | Interval | Description |
|-----|----------|-------------|
| `inventory-sync` | 5 min | Pull from SQL Server, transform, push to Postgres + Kafka |
| `fitment-validation` | 1 hr | Validate fitment records against vehicle database |
| `cache-warmup` | 30 min | Pre-warm Redis for high-traffic product queries |

Register a new job:

```go
worker.Register(etl.ScheduledJob{
    Name:     "my-job",
    Interval: 10 * time.Minute,
    Run: func(ctx context.Context) error {
        // ETL logic here
        return nil
    },
})
```

### Bulk import

```go
result, err := pipeline.ImportProducts(ctx, products) // fan-out across 10 goroutines
fmt.Printf("processed=%d failed=%d duration=%s\n", result.Processed, result.Failed, result.Duration)
```

---

## Deployment

### Docker

```bash
# Build API image
docker build --build-arg TARGET=api -t automotive-catalog-api .

# Build Worker image
docker build --build-arg TARGET=worker -t automotive-catalog-worker .
```

### Kubernetes

```bash
# Apply all manifests
kubectl apply -f deployments/k8s/

# Check rollout
kubectl rollout status deployment/catalog-api -n automotive-catalog

# Scale manually
kubectl scale deployment catalog-api --replicas=5 -n automotive-catalog
```

The API deployment includes a **HorizontalPodAutoscaler** that scales from 3 to 20 replicas based on CPU (70%) and memory (80%) utilization.

> **Note:** Replace `your-ecr-repo/...` in the deployment manifests with your actual container registry path. Never commit real secret values — use AWS Secrets Manager, HashiCorp Vault, or Kubernetes Sealed Secrets in production.
