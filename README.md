# NMI Payment Integration Service

**Version:** 1.0

A production-ready Go microservice for integrating with the NMI (Network Merchants, Inc.) payment gateway. This service provides comprehensive payment processing capabilities, including tokenization, recurring payments, refunds, voids, and detailed transaction logging and monitoring.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Configuration](#configuration)
- [API Reference](#api-reference)
  - [Health Check](#1-health-check)
  - [Add a Plan](#2-add-a-plan)
  - [List All Plans](#3-list-all-plans)
  - [Tokenize a Credit Card](#4-tokenize-a-credit-card)
  - [Process a Sale](#5-process-a-sale)
  - [Create a Recurring Payment](#6-create-a-recurring-payment)
  - [Process a Refund](#7-process-a-refund)
  - [Void a Transaction](#8-void-a-transaction)
- [Migrating from Sandbox to Production](#migrating-from-sandbox-to-production)
- [Docker Deployment](#docker-deployment)
- [Monitoring and Logging](#monitoring-and-logging)
- [Troubleshooting](#troubleshooting)
- [License](#license)

---

## Features

### Core Payment Features
- **Tokenization**: Securely tokenize credit card details for future use.
- **Recurring Payments**: Set up recurring billing based on pre-defined plans.
- **Refunds**: Full or partial refunds for transactions.
- **Voids**: Cancel a transaction before settlement.
- **Plan Management**: Add, update, and list subscription plans.

### Security
- Validates credit card details using the Luhn algorithm.
- Ensures proper CVV, expiration date, and amount formatting.
- Supports idempotency keys to prevent duplicate transactions.

### Logging and Monitoring
- Logs all transactions in structured format.
- Exposes Prometheus metrics for real-time monitoring.
- Supports logging to files (CSV and JSON).

---

## Installation

### Prerequisites
- **Go 1.20** or higher.
- **Docker** (for containerized deployment).
- An **NMI API Key** (sandbox or production).

### Steps

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/nmi-payment-service.git
   cd nmi-payment-service
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build the application:
   ```bash
   go build -o payment-service ./cmd/main.go
   ```

4. Run the service locally:
   ```bash
   MODE=serve API_URL=https://secure.nmi.com/api/transact.php NMI_API_KEY=your_api_key ./payment-service
   ```

---

## Configuration

Set the following environment variables:

```env
NMI_API_KEY=your_nmi_api_key
LOG_FILE=transactions.log
CSV_FILE=transactions.csv
API_URL=https://secure.networkmerchants.com/api/transact.php  # Sandbox
# API_URL=https://secure.nmi.com/api/transact.php  # Production
DEBUG_MODE=true
```

---

## API Reference

### 1. Health Check

**Endpoint:** `GET /health`

Checks if the service is running.

**Response:**
```json
{
  "status": "OK",
  "timestamp": "2025-01-15T18:25:43Z"
}
```

### 2. Add a Plan

**Endpoint:** `POST /plans/add`

Adds a new subscription plan.

**Request Example:**
```json
{
  "event_id": "35f56738-4755-4cd9-9068-ea9e83d60d2e",
  "event_type": "recurring.plan.add",
  "event_body": {
    "merchant": {
      "id": "944025",
      "name": "Test merchant account"
    },
    "features": {
      "is_test_mode": true
    },
    "plan": {
      "id": "TestPlanId1",
      "name": "Test Plan",
      "amount": "10.00",
      "day_frequency": "",
      "payments": "Until canceled",
      "month_frequency": "1",
      "day_of_month": "15"
    }
  }
}
```

**Response Example:**
```json
{
  "plan": {
    "id": "TestPlanId1",
    "name": "Test Plan",
    "amount": "10.00",
    "day_frequency": "",
    "payments": "Until canceled",
    "month_frequency": "1",
    "day_of_month": "15"
  },
  "message": "Plan added successfully"
}
```

### 3. List All Plans

**Endpoint:** `GET /plans/list`

Retrieves all subscription plans.

**Response Example:**
```json
{
  "TestPlanId1": {
    "id": "TestPlanId1",
    "name": "Test Plan",
    "amount": "10.00",
    "day_frequency": "",
    "payments": "Until canceled",
    "month_frequency": "1",
    "day_of_month": "15"
  }
}
```

### 4. Tokenize a Credit Card

**Endpoint:** `POST /payments/tokenize`

Tokenizes a credit card for future transactions.

**Request Example:**
```json
{
  "credit_card": "4111111111111111",
  "exp_date": "1225",
  "cvv": "123",
  "amount": "1.00",
  "type": "sale",
  "billing": {
    "first_name": "John",
    "last_name": "Doe",
    "address1": "123 Test St",
    "city": "TestCity",
    "state": "TX",
    "zip": "12345",
    "country": "US",
    "email": "test@example.com",
    "phone": "1234567890"
  }
}
```

**Response Example:**
```json
{
  "customer_vault_id": "5508470413134828416",
  "token": "5508470413134828416",
  "masked_card": "************1111",
  "card_type": "VISA",
  "expiry_date": "1225",
  "success": true,
  "message": "SUCCESS"
}
```

### 5. Process a Sale

**Endpoint:** `POST /payments/sale`

Processes a sale transaction.

**Request Example:**
```json
{
  "customer_vault_id": "5508470413134828416",
  "amount": "10.00",
  "type": "sale",
  "billing": {
    "first_name": "John",
    "last_name": "Doe",
    "address1": "123 Test St",
    "city": "TestCity",
    "state": "TX",
    "zip": "12345",
    "country": "US",
    "email": "test@example.com",
    "phone": "1234567890"
  }
}
```

**Response Example:**
```json
{
  "transaction_id": "10317389463",
  "status": "success",
  "response": "SUCCESS"
}
```

### 6. Create a Recurring Payment

**Endpoint:** `POST /payments/recurring/create`

Sets up a recurring payment based on a plan.

**Request Example:**
```json
{
  "customer_vault_id": "5508470413134828416",
  "plan_id": "TestPlanId1",
  "billing": {
    "first_name": "John",
    "last_name": "Doe",
    "address1": "123 Test St",
    "city": "TestCity",
    "state": "TX",
    "zip": "12345",
    "country": "US",
    "email": "test@example.com",
    "phone": "1234567890"
  }
}
```

**Response Example:**
```json
{
  "subscription_id": "10317410976",
  "status": "success",
  "plan_id": "TestPlanId1",
  "customer_vault_id": "5508470413134828416"
}
```

### 7. Process a Refund

**Endpoint:** `POST /payments/refund`

Refunds a transaction (full or partial).

**Request Example:**
```json
{
  "transaction_id": "10317389463",
  "amount": "5.00"
}
```

**Response Example:**
```json
{
  "status": "success",
  "response": "SUCCESS",
  "transaction_id": "10317415000"
}
```

### 8. Void a Transaction

**Endpoint:** `POST /payments/void`

Voids a transaction before settlement.

**Request Example:**
```json
{
  "transaction_id": "10317389463"
}
```

**Response Example:**
```json
{
  "status": "success",
  "response": "SUCCESS",
  "transaction_id": "10317389463"
}
```

---

## Migrating from Sandbox to Production

### Update Environment Configuration
Set the following environment variables for production:

```env
API_URL=https://secure.nmi.com/api/transact.php
NMI_API_KEY=your_production_api_key
DEBUG_MODE=false
```

### Configure SSL
TLS settings should enforce modern security standards:

```go
// TLS configuration example
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
    CipherSuites: []uint16{
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
    },
}
```

### Set Up Proper Monitoring
Prometheus configuration example for monitoring:

```yaml
scrape_configs:
  - job_name: 'nmi-payment-service'
    scrape_interval: 15s
    static_configs:
      - targets: ['nmi-payment:8080']
```

---

## Docker Deployment

### Steps
1. Build the Docker image:
   ```bash
   docker build -t nmi-payment-service .
   ```

2. Run the Docker container:
   ```bash
   docker run -p 8080:8080 -e MODE=serve -e NMI_API_KEY=your_api_key nmi-payment-service
   ```

### Docker Compose
```yaml
version: '3.7'
services:
  nmi-payment:
    build: .
    ports:
      - "8080:8080"
    environment:
      - MODE=serve
      - NMI_API_KEY=your_api_key
```

---

## Monitoring and Logging

### Prometheus Metrics
Available at `http://localhost:8080/metrics`

Key Metrics:
- `http_requests_total`: Total HTTP requests.
- `http_request_duration_seconds`: Request duration histograms.
- `nmi_transactions_total`: Total processed transactions.

### Log Files
- `transactions.log`: Logs all transactions.
- `transactions.csv`: Logs transaction records in CSV format.

---

## Troubleshooting

### Common Docker Issues

1. **Docker Desktop Not Running:**
   ```
   error during connect: Get "http://%2F%2F.%2Fpipe%2FdockerDesktopLinuxEngine/v1.47/images/...": open //./pipe/dockerDesktopLinuxEngine: The system cannot find the file specified.
   ```
   **Solution:** Start Docker Desktop before running docker commands.

2. **Environment Variables Not Set:**
   ```
   time="2025-01-08T14:38:04-05:00" level=warning msg="The \"DEBUG_MODE\" variable is not set. Defaulting to a blank string."
   ```
   **Solution:** Ensure all environment variables are properly set in the `.env` file.

3. **Build Issues:**
   If you encounter build issues, try cleaning Docker cache:
   ```bash
   docker-compose down
   docker system prune -f
   docker-compose up --build -d
   ```

### Common Errors and Solutions

1. **Authentication Error (300):**
   ```json
   {
       "raw_response": "response=3&responsetext=Authentication Failed&response_code=300"
   }
   ```
   **Solution:** Verify API key and environment settings.

2. **Duplicate Transaction:**
   ```
   NMI Error duplicate_transaction: duplicate transaction detected
   ```
   **Solution:** Use a unique `idempotency_key` for each transaction.

3. **Invalid Card:**
   ```
   NMI Error invalid_card: invalid credit card number length
   ```
   **Solution:** Verify card number format and validation.

---

## License

This project is licensed under the MIT License.

## Author

Zachary Kleckner | zkleckner@gmail.com