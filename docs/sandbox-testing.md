# Sandbox Testing Guide

This guide explains how to run integration tests against mock provider servers without real API credentials.

## Overview

EquiShare provides mock servers for external providers:

| Provider | Mock Port | Purpose |
|----------|-----------|---------|
| M-Pesa | 8090 | Payment deposits/withdrawals |
| Africa's Talking | 8091 | SMS notifications |
| Alpaca | 8092 | Stock trading |

## Quick Start

### 1. Start Mock Servers

```bash
# Start all mock servers with Docker Compose
docker compose -f docker-compose.yml -f docker-compose.sandbox.yml --profile sandbox up -d

# Or build and run individually
cd tools/mockservers
go run ./mpesa &
go run ./africastalking &
go run ./alpaca &
```

### 2. Run Integration Tests

```bash
# Run all integration tests
go test ./tests/integration/... -v

# Run specific provider tests
go test ./tests/integration/... -run TestMPesa -v
go test ./tests/integration/... -run TestAlpaca -v

# Skip integration tests (for unit testing only)
go test ./... -short
```

### 3. Verify Mock State

Each mock server has admin endpoints for debugging:

```bash
# M-Pesa - List all requests
curl http://localhost:8090/admin/requests

# Africa's Talking - List sent messages
curl http://localhost:8091/admin/messages

# Alpaca - Get current state
curl http://localhost:8092/admin/state
```

## Mock Server Details

### M-Pesa Mock

**Endpoints:**
- `GET /oauth/v1/generate` - OAuth token generation
- `POST /mpesa/stkpush/v1/processrequest` - STK Push (Lipa Na M-Pesa)
- `POST /mpesa/stkpushquery/v1/query` - Query STK status
- `POST /mpesa/b2c/v1/paymentrequest` - B2C withdrawal

**Behavior:**
- Tokens expire after 1 hour
- STK Push triggers automatic callback after 2 seconds
- B2C triggers automatic callback after 2 seconds
- Callbacks simulate successful payments by default

**Admin Endpoints:**
- `GET /admin/requests` - List all STK/B2C requests
- `POST /admin/reset` - Clear all state
- `POST /admin/trigger-callback/:id?success=true|false` - Manually trigger callback

### Africa's Talking Mock

**Endpoints:**
- `POST /version1/messaging` - Send SMS

**Behavior:**
- Requires `apiKey` header
- Records all sent messages
- Returns success for all valid requests

**Admin Endpoints:**
- `GET /admin/messages` - List all sent messages
- `POST /admin/reset` - Clear message history
- `POST /admin/trigger-delivery/:id?status=Delivered|Failed` - Update delivery status

### Alpaca Mock

**Endpoints:**
- `GET /v2/account` - Account information
- `POST /v2/orders` - Create order
- `GET /v2/orders` - List orders
- `DELETE /v2/orders/:id` - Cancel order
- `GET /v2/positions` - List positions
- `DELETE /v2/positions/:symbol` - Close position
- `GET /v2/assets` - List tradable assets
- `GET /v2/stocks/:symbol/quotes/latest` - Get quote

**Behavior:**
- Requires `APCA-API-KEY-ID` and `APCA-API-SECRET-KEY` headers
- Market orders fill immediately
- Limit orders remain in "new" status
- Default cash balance: $100,000

**Admin Endpoints:**
- `POST /admin/reset` - Reset to initial state
- `POST /admin/set-cash` - Set account cash balance
- `GET /admin/state` - Get full state (account, orders, positions)

## Environment Variables

Configure mock server URLs:

```bash
export MPESA_MOCK_URL=http://localhost:8090
export AT_MOCK_URL=http://localhost:8091
export ALPACA_MOCK_URL=http://localhost:8092
```

## CI/CD Integration

### GitHub Actions

```yaml
jobs:
  integration-tests:
    runs-on: ubuntu-latest
    services:
      mpesa-mock:
        image: ghcr.io/rohianon/equishare-mpesa-mock:latest
        ports:
          - 8090:8090
      alpaca-mock:
        image: ghcr.io/rohianon/equishare-alpaca-mock:latest
        ports:
          - 8092:8092
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run integration tests
        run: go test ./tests/integration/... -v
        env:
          MPESA_MOCK_URL: http://localhost:8090
          ALPACA_MOCK_URL: http://localhost:8092
```

## Testing Patterns

### Test Setup/Teardown

```go
func TestFeature(t *testing.T) {
    h := integration.NewHarness(t)

    // Wait for mock servers
    if err := h.WaitForAll(30 * time.Second); err != nil {
        t.Skip("Mock servers not available")
    }

    // Reset state before test
    h.ResetAll()

    // ... test logic ...
}
```

### Simulating Failures

```go
// Trigger a failed STK callback
h.Do(Request{
    Method: "POST",
    URL:    h.Config().MPesaURL + "/admin/trigger-callback/" + checkoutReqID + "?success=false",
})
```

### Testing Idempotency

```go
// Same request should return same result
for i := 0; i < 3; i++ {
    resp, _ := h.Do(Request{
        Method: "POST",
        URL:    h.Config().AlpacaURL + "/v2/orders",
        Body: map[string]any{
            "client_order_id": "unique-id-123", // Same ID
            "symbol": "AAPL",
            // ...
        },
    })
    // Verify consistent behavior
}
```

## Troubleshooting

### Mock Server Not Responding

```bash
# Check if server is running
curl http://localhost:8090/health

# Check Docker logs
docker compose logs mpesa-mock
```

### Tests Flaky

1. Increase timeouts in `WaitFor*` calls
2. Ensure state is reset between tests with `h.ResetAll()`
3. Check for race conditions in concurrent tests

### Callbacks Not Received

1. Ensure callback URL is accessible from mock server
2. Check Docker network configuration
3. Review mock server logs for callback attempts
