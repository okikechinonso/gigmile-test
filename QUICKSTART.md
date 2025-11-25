# Quick Start Guide

Get the GigMile Payment Service running in 5 minutes!

## Prerequisites

- Go 1.21 or higher
- Redis (via Docker or locally installed)
- curl (for testing)

## Option 1: Docker Compose (Recommended)

### Step 1: Start All Services

```bash
docker-compose up -d
```

Wait about 10 seconds for services to initialize.

### Step 2: Seed Sample Data

```bash
docker-compose exec api go run scripts/seed.go
```

### Step 3: Test the API

```bash
# Health check
curl http://localhost:8080/health

# Process a payment
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIG00001",
    "payment_status": "COMPLETE",
    "transaction_amount": "10000",
    "transaction_date": "2025-11-24 14:54:16",
    "transaction_reference": "VPAY25112414541112345678901234"
  }'

# Get customer details
curl http://localhost:8080/api/v1/customers/GIG00001
```

### Step 4: View Logs

```bash
docker-compose logs -f api
```

### Stop Services

```bash
docker-compose down
```

---

## Option 2: Local Development

### Step 1: Install Dependencies

```bash
go mod download
```

### Step 2: Start Redis

```bash
# Using Docker
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Or using Homebrew (macOS)
brew install redis
brew services start redis

# Or using apt (Ubuntu/Debian)
sudo apt-get install redis-server
sudo systemctl start redis
```

### Step 3: Seed Sample Data

```bash
go run scripts/seed.go
```

### Step 4: Run the API

```bash
go run cmd/api/main.go
```

### Step 5: Test the API

Open another terminal and run:

```bash
# Health check
curl http://localhost:8080/health

# Process a payment
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIG00001",
    "payment_status": "COMPLETE",
    "transaction_amount": "10000",
    "transaction_date": "2025-11-24 14:54:16",
    "transaction_reference": "VPAY25112414541112345678901234"
  }'
```

Expected response:

```json
{
  "success": true,
  "message": "payment processed successfully",
  "customer_id": "GIG00001",
  "outstanding_balance": 99000000,
  "total_paid": 1000000,
  "payment_progress": 1.0,
  "is_fully_paid": false
}
```

---

## Option 3: Using Makefile

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

### Run with Coverage

```bash
make coverage
```

### Seed Data

```bash
make seed
```

