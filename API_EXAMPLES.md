# GigMile Payment API - Sample Requests

## Process Payment

### Request 1: Valid Payment

```bash
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

### Request 2: Duplicate Transaction (Same Reference)

```bash
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

### Request 3: Another Valid Payment

```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIG00002",
    "payment_status": "COMPLETE",
    "transaction_amount": "50000",
    "transaction_date": "2025-11-24 15:30:00",
    "transaction_reference": "VPAY25112415300098765432109876"
  }'
```

### Request 4: Large Payment (Half of Asset)

```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIG00003",
    "payment_status": "COMPLETE",
    "transaction_amount": "500000",
    "transaction_date": "2025-11-24 16:00:00",
    "transaction_reference": "VPAY25112416000011111111111111"
  }'
```

### Request 5: Full Payment

```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIG00004",
    "payment_status": "COMPLETE",
    "transaction_amount": "1000000",
    "transaction_date": "2025-11-24 16:30:00",
    "transaction_reference": "VPAY25112416300022222222222222"
  }'
```

### Request 6: Get Customer Payments

```bash
curl http://localhost:8072/api/v1/payments?customer_id=GIG00001&page=1&page_size=5
```


## Get Customer Details

```bash
curl http://localhost:8080/api/v1/customers/GIG00001
```

```bash
curl http://localhost:8080/api/v1/customers/GIG00002
```

## Health Check

```bash
curl http://localhost:8080/health
```

## Load Testing with hey

Install hey: `brew install hey` (macOS) or download from https://github.com/rakyll/hey

### Test 1: Moderate Load (1000 requests, 100 concurrent)

```bash
echo '{
  "customer_id": "GIG00001",
  "payment_status": "COMPLETE",
  "transaction_amount": "1000",
  "transaction_date": "2025-11-24 14:54:16",
  "transaction_reference": "LOAD_TEST_'$(uuidgen)'"
}' > /tmp/payment.json

hey -n 1000 -c 100 -m POST \
  -H "Content-Type: application/json" \
  -D /tmp/payment.json \
  http://localhost:8080/api/v1/payments
```

### Test 2: High Load (10000 requests, 500 concurrent)

```bash
hey -n 10000 -c 500 -m POST \
  -H "Content-Type: application/json" \
  -D /tmp/payment.json \
  http://localhost:8080/api/v1/payments
```

## Error Cases

### Invalid Amount

```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIG00001",
    "payment_status": "COMPLETE",
    "transaction_amount": "invalid",
    "transaction_date": "2025-11-24 14:54:16",
    "transaction_reference": "ERROR_TEST_001"
  }'
```

### Missing Customer ID

```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "payment_status": "COMPLETE",
    "transaction_amount": "10000",
    "transaction_date": "2025-11-24 14:54:16",
    "transaction_reference": "ERROR_TEST_002"
  }'
```

### Invalid Date Format

```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIG00001",
    "payment_status": "COMPLETE",
    "transaction_amount": "10000",
    "transaction_date": "2025/11/24 14:54:16",
    "transaction_reference": "ERROR_TEST_003"
  }'
```

### Pending Payment (Not Complete)

```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "GIG00001",
    "payment_status": "PENDING",
    "transaction_amount": "10000",
    "transaction_date": "2025-11-24 14:54:16",
    "transaction_reference": "PENDING_TEST_001"
  }'
```
