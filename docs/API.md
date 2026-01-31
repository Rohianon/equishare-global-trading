# EquiShare API Documentation

## Overview

The EquiShare API follows REST conventions and uses JSON for request/response bodies.
All API endpoints are versioned under `/api/v1/`.

## API Versioning Strategy

### URL Versioning

External REST APIs use URL-based versioning:

```
/api/v1/auth/login
/api/v1/trading/orders
/api/v1/payments/deposit
```

### Version Lifecycle

1. **v1 (Current)**: Active development, full support
2. **Deprecated**: 6-month warning before removal
3. **Sunset**: Endpoint returns 410 Gone

### Breaking vs Non-Breaking Changes

**Non-breaking changes** (added to existing version):
- Adding new optional fields to responses
- Adding new endpoints
- Adding new optional query parameters
- Adding new error codes

**Breaking changes** (require new version):
- Removing or renaming fields
- Changing field types
- Changing endpoint paths
- Removing endpoints
- Changing authentication mechanisms

## Response Envelope

All API responses follow a standard envelope format for consistency.

### Success Response

```json
{
  "data": {
    // Response payload (varies by endpoint)
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-31T12:00:00Z",
    "version": "v1"
  }
}
```

### Error Response

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input",
    "details": ["field 'email' is required", "field 'amount' must be positive"]
  },
  "meta": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-01-31T12:00:00Z"
  }
}
```

### Paginated Response

```json
{
  "data": {
    "items": [...],
    "pagination": {
      "page": 1,
      "per_page": 20,
      "total": 100,
      "total_pages": 5,
      "has_more": true
    }
  },
  "meta": {
    "request_id": "...",
    "timestamp": "..."
  }
}
```

## Error Codes

### HTTP Status Codes

| Status | Meaning |
|--------|---------|
| 200 | Success |
| 201 | Created |
| 202 | Accepted (async operation started) |
| 204 | No Content |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 429 | Too Many Requests |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

### Application Error Codes

#### Common Errors

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `BAD_REQUEST` | 400 | Generic bad request |
| `VALIDATION_ERROR` | 400 | Input validation failed |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Access denied |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource already exists |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Unexpected server error |
| `SERVICE_UNAVAILABLE` | 503 | External service down |

#### Authentication Errors

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `AUTH_INVALID_CREDENTIALS` | 401 | Wrong phone/PIN/password |
| `AUTH_INVALID_TOKEN` | 401 | Invalid JWT token |
| `AUTH_TOKEN_EXPIRED` | 401 | JWT token expired |
| `AUTH_SESSION_EXPIRED` | 401 | Session expired |
| `AUTH_INVALID_OTP` | 400 | Invalid OTP |
| `AUTH_OTP_EXPIRED` | 400 | OTP expired |
| `AUTH_2FA_REQUIRED` | 403 | 2FA verification needed |
| `AUTH_INVALID_2FA_CODE` | 400 | Wrong 2FA code |

#### User Errors

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `USER_INVALID_PHONE` | 400 | Invalid phone format |
| `USER_PHONE_EXISTS` | 409 | Phone already registered |
| `USER_EMAIL_EXISTS` | 409 | Email already registered |
| `USER_NOT_FOUND` | 404 | User not found |
| `USER_DEACTIVATED` | 403 | Account deactivated |
| `USER_KYC_REQUIRED` | 403 | KYC verification needed |
| `USER_KYC_PENDING` | 403 | KYC in progress |
| `USER_KYC_REJECTED` | 403 | KYC was rejected |

#### Payment Errors

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `PAYMENT_INSUFFICIENT_FUNDS` | 400 | Balance too low |
| `PAYMENT_WALLET_NOT_FOUND` | 404 | Wallet not found |
| `PAYMENT_FAILED` | 400 | Payment processing failed |
| `PAYMENT_PENDING` | 202 | Payment in progress |
| `PAYMENT_WITHDRAWAL_FAILED` | 400 | Withdrawal failed |
| `PAYMENT_MINIMUM_AMOUNT` | 400 | Below minimum |
| `PAYMENT_MAXIMUM_AMOUNT` | 400 | Above maximum |
| `PAYMENT_DAILY_LIMIT` | 400 | Daily limit exceeded |

#### Trading Errors

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `TRADING_MARKET_CLOSED` | 400 | Market is closed |
| `TRADING_ORDER_NOT_FOUND` | 404 | Order not found |
| `TRADING_ORDER_FILLED` | 400 | Order already filled |
| `TRADING_ORDER_CANCELLED` | 400 | Order already cancelled |
| `TRADING_ORDER_LIMIT` | 400 | Order limit exceeded |
| `TRADING_INVALID_SYMBOL` | 400 | Unknown symbol |
| `TRADING_INVALID_ORDER_TYPE` | 400 | Invalid order type |
| `TRADING_INVALID_ORDER_SIDE` | 400 | Invalid side |
| `TRADING_INVALID_QUANTITY` | 400 | Invalid quantity |
| `TRADING_SYMBOL_NOT_TRADEABLE` | 400 | Symbol not tradeable |
| `TRADING_POSITION_NOT_FOUND` | 404 | No position |

#### Provider Errors

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `PROVIDER_MPESA_UNAVAILABLE` | 503 | M-Pesa is down |
| `PROVIDER_ALPACA_UNAVAILABLE` | 503 | Trading service down |
| `PROVIDER_SMS_UNAVAILABLE` | 503 | SMS service down |

## OpenAPI Specifications

OpenAPI specs are located in `api/openapi/`:

- `common.yaml` - Shared schemas and components
- `auth.yaml` - Authentication service API
- `trading.yaml` - Trading service API
- `payment.yaml` - Payment service API
- `gateway.yaml` - Consolidated API gateway spec

### Viewing Documentation

In local development, Swagger UI is available at `/docs`:

```bash
# Start any service
go run services/api-gateway/main.go

# Open browser
open http://localhost:8080/docs
```

### Generating Clients

Use OpenAPI Generator to create client SDKs:

```bash
# TypeScript/JavaScript
openapi-generator generate -i api/openapi/gateway.yaml -g typescript-fetch -o clients/typescript

# Python
openapi-generator generate -i api/openapi/gateway.yaml -g python -o clients/python

# Go
openapi-generator generate -i api/openapi/gateway.yaml -g go -o clients/go
```

## Using the Response Package

Services should use `pkg/response` for consistent responses:

```go
import "github.com/Rohianon/equishare-global-trading/pkg/response"

// Success response
func GetUser(c *fiber.Ctx) error {
    user := getUserFromDB(c.Params("id"))
    return response.Success(c, user)
}

// Created response (201)
func CreateUser(c *fiber.Ctx) error {
    user := createUser(...)
    return response.Created(c, user)
}

// Paginated response
func ListOrders(c *fiber.Ctx) error {
    orders, total := getOrders(page, perPage)
    return response.Paginated(c, orders, page, perPage, total)
}

// Error response
func DoSomething(c *fiber.Ctx) error {
    return response.Error(c, 400, "VALIDATION_ERROR", "Invalid input", "field 'x' required")
}
```

### Using AppErrors

For domain-specific errors, use `pkg/errors`:

```go
import apperrors "github.com/Rohianon/equishare-global-trading/pkg/errors"

func PlaceOrder(c *fiber.Ctx) error {
    if balance < amount {
        return apperrors.ErrInsufficientFunds
    }
    if !isValidSymbol(symbol) {
        return apperrors.ErrInvalidSymbol.WithDetails("Symbol not found: " + symbol)
    }
    // ...
}
```

### Error Handler Middleware

Configure Fiber to use the standard error handler:

```go
import "github.com/Rohianon/equishare-global-trading/pkg/response"

app := fiber.New(fiber.Config{
    ErrorHandler: response.ErrorHandler,
})
```

## Rate Limiting

| Endpoint Type | Limit |
|--------------|-------|
| Authenticated | 100 req/min |
| Unauthenticated | 20 req/min |
| Login/Register | 5 req/min per IP |
| OTP requests | 3 per hour per phone |

Rate limit headers are included in responses:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

## Request Tracing

All requests receive a unique `request_id` for tracing:

1. If client provides `X-Request-ID` header, it's used
2. Otherwise, a UUID is generated
3. The ID is returned in `meta.request_id`
4. Use this ID when contacting support

## CORS

The API supports CORS for browser-based clients:

- Allowed origins: Configured per environment
- Allowed methods: GET, POST, PUT, DELETE, OPTIONS
- Allowed headers: Authorization, Content-Type, X-Request-ID
- Credentials: Supported

## Webhooks (Internal)

Services communicate via Kafka events. See `pkg/events` for:

- Event envelope format
- Topic conventions
- Payload definitions
