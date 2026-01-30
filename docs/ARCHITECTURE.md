# EquiShare Global Trading - System Architecture

> Democratizing Global Markets for Kenya

## Vision

Enable every Kenyan - from a farmer in Kisumu with a feature phone to a tech worker in Nairobi with a smartphone - to invest in global markets (US, London, eventually NSE) through multiple accessible interfaces.

## Research & Competitive Analysis (2025-2026)

### Market Context

According to [TechPoint Africa's 2026 outlook](https://techpoint.africa/guide/african-fintech-outlook/):
- African startups raised **$3+ billion** in 2025 (33% YoY increase)
- Kenya secured **$638 million** in 2025 - highest in Africa (29% of continent's total)
- Africa now has **9 tech unicorns**, 8 of which are fintech companies
- "2026 will mark the transition from African fintech going global to becoming the globe itself"

### How Others Built It

| Platform | Key Insight | What We Learn |
|----------|-------------|---------------|
| [Bamboo](https://investbamboo.com/) (Nigeria) | YC-backed, fractional shares from ₦15,000, expanded to South Africa in 2025 | Fractional shares drive adoption; multi-market expansion |
| [Chipper Cash](https://www.chippercash.com/) | Built trust with P2P payments first, then added stocks | Start simple, prove reliability, then expand features |
| [Flutterwave](https://flutterwave.com/) | $3B valuation, Africa's most valuable fintech | Payment infrastructure is foundational |
| [OPay](https://www.opayweb.com/) | $2B valuation, payments → savings → loans | Super-app model works in Africa |
| [Alpaca](https://alpaca.markets/) | Nasdaq membership (2025), 24/5 trading, 99.99% uptime | Let Alpaca handle US regulatory complexity |

### Platform Capabilities (2025-2026)

**[Alpaca 2025 Review](https://alpaca.markets/blog/alpacas-2025-in-review/):**
- 24/5 trading for stocks and ETFs
- Achieved **Nasdaq exchange membership** in 2025
- **99.99% system uptime** since January 2025
- **1.5ms order processing** with OMS v2
- Fractional shares from $1
- 6.25% margin rate
- Tokenized equities with 24/7 API access

**[Daraja 3.0](https://techcabal.com/2025/11/25/safaricom-overhauls-m-pesa-api-platform/) (November 2025):**
- Cloud-native architecture with **12,000 TPS** capacity
- **Mini Apps** - lightweight apps inside M-Pesa Super App
- **Security APIs** - built-in KYC and fraud detection
- **Ratiba API** - recurring payments and standing orders
- AI-powered developer tools for real-time troubleshooting
- 66,000+ integrations, 105,000+ developers
- 25% of all M-Pesa transactions now via APIs

**[Africa's Talking](https://africastalking.com/):**
- 4+ billion messages/year
- 80+ telecom integrations
- Same USSD code across Safaricom, Airtel, Equitel, Telkom
- Developer sandbox (Yoda Platform) for testing

### Why This Will Work

1. **USSD Gap**: No one offers USSD stock trading in Kenya
2. **Alpaca 2025**: Nasdaq membership, 24/5 trading, fractional shares, handles SEC compliance
3. **Daraja 3.0**: Cloud-native M-Pesa with Security APIs for built-in KYC
4. **Market Timing**: Kenya digital payments CAGR 14.1% → $14.54B by 2028
5. **Regulatory Progress**: CMA actively modernizing for fintech, NIFC launched

---

## Repository Structure (Mono-repo)

```
equishare-global-trading/
├── .github/
│   ├── workflows/           # CI/CD pipelines
│   └── ISSUE_TEMPLATE/      # Issue templates
├── docs/
│   ├── ARCHITECTURE.md      # This document
│   ├── API.md               # API specifications
│   ├── ADR/                 # Architecture Decision Records
│   └── runbooks/            # Operational runbooks
├── proto/                   # Protobuf definitions (gRPC)
│   ├── user/
│   ├── trading/
│   ├── portfolio/
│   └── common/
├── pkg/                     # Shared Go packages
│   ├── config/              # Configuration management
│   ├── logger/              # Structured logging
│   ├── middleware/          # Common middleware
│   ├── events/              # Kafka event definitions
│   ├── database/            # Database utilities
│   └── errors/              # Error handling
├── services/
│   ├── api-gateway/         # Public API (REST/GraphQL)
│   ├── auth-service/        # Authentication & KYC
│   ├── user-service/        # User management
│   ├── trading-service/     # Order execution
│   ├── portfolio-service/   # Portfolio management
│   ├── market-data-service/ # Real-time market data
│   ├── payment-service/     # M-Pesa integration
│   ├── notification-service/# SMS, Push, Email
│   └── ussd-service/        # USSD interface
├── clients/
│   ├── cli/                 # CLI tool (Cobra)
│   ├── web/                 # React frontend
│   └── mobile/              # React Native app
├── infrastructure/
│   ├── docker/              # Dockerfiles
│   ├── k8s/                 # Kubernetes manifests
│   ├── terraform/           # Infrastructure as Code
│   └── scripts/             # Deployment scripts
├── migrations/              # Database migrations
├── tools/                   # Development tools
├── go.work                  # Go workspace
├── go.work.sum
├── Makefile                 # Build automation
└── README.md
```

---

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENTS                                         │
├────────────┬────────────┬────────────┬────────────┬────────────────────────┤
│    CLI     │    USSD    │    Web     │   Mobile   │   Webhooks/Callbacks   │
│  (Cobra)   │ (Africa's  │  (React)   │   (RN)     │   (Price Alerts)       │
│            │  Talking)  │            │            │                        │
└─────┬──────┴─────┬──────┴─────┬──────┴─────┬──────┴───────────┬────────────┘
      │            │            │            │                   │
      └────────────┴────────────┼────────────┴───────────────────┘
                                │
                    ┌───────────▼───────────┐
                    │   API Gateway (Go)    │
                    │   ─────────────────   │
                    │   • Rate Limiting     │
                    │   • Auth (JWT)        │
                    │   • Request Routing   │
                    │   • API Versioning    │
                    └───────────┬───────────┘
                                │
         ┌──────────────────────┼──────────────────────┐
         │                      │                      │
         ▼                      ▼                      ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  Auth Service   │   │ Trading Service │   │ Portfolio Svc   │
│  ─────────────  │   │  ─────────────  │   │  ─────────────  │
│  • JWT/Refresh  │   │  • Order Exec   │   │  • Holdings     │
│  • KYC (Smile)  │   │  • Order Status │   │  • P&L Calc     │
│  • 2FA (TOTP)   │   │  • Trade History│   │  • Dividends    │
└────────┬────────┘   └────────┬────────┘   └────────┬────────┘
         │                     │                      │
         └─────────────────────┼──────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │       Kafka         │
                    │  ─────────────────  │
                    │  • order.created    │
                    │  • order.filled     │
                    │  • price.update     │
                    │  • payment.received │
                    │  • kyc.verified     │
                    └──────────┬──────────┘
                               │
         ┌─────────────────────┼─────────────────────┐
         │                     │                     │
         ▼                     ▼                     ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  Payment Svc    │   │ Market Data Svc │   │ Notification Svc│
│  ─────────────  │   │  ─────────────  │   │  ─────────────  │
│  • M-Pesa STK   │   │  • Price Stream │   │  • SMS (AT)     │
│  • B2C Withdraw │   │  • Watchlists   │   │  • Push (FCM)   │
│  • Balance Mgmt │   │  • Historical   │   │  • Email        │
└────────┬────────┘   └────────┬────────┘   └────────┬────────┘
         │                     │                     │
         ▼                     ▼                     ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│   M-Pesa API    │   │   Alpaca API    │   │ Africa's Talking│
│   (Daraja)      │   │   (Markets)     │   │    (SMS/USSD)   │
└─────────────────┘   └─────────────────┘   └─────────────────┘
```

### Data Flow Example: Buy Order via USSD

```
User dials *XXX#
      │
      ▼
┌─────────────────┐
│ Africa's Talking│──► POST /ussd/callback
└─────────────────┘
      │
      ▼
┌─────────────────┐
│  USSD Service   │──► Parse menu state, validate input
└─────────────────┘
      │
      ▼
┌─────────────────┐
│  API Gateway    │──► Auth check, rate limit
└─────────────────┘
      │
      ▼
┌─────────────────┐
│ Trading Service │──► Validate balance, create order
└─────────────────┘
      │
      ├──► Kafka: order.created
      │
      ▼
┌─────────────────┐
│   Alpaca API    │──► Submit market order
└─────────────────┘
      │
      ▼
Webhook: order.filled
      │
      ▼
┌─────────────────┐
│ Kafka: order.filled │
└─────────────────┘
      │
      ├──► Portfolio Service (update holdings)
      ├──► Notification Service (SMS confirmation)
      └──► USSD Service (session response)
```

---

## Technology Stack

### Backend Services (Go)

Based on [2025-2026 Go microservices best practices](https://golang.elitedev.in/golang/building-event-driven-microservices-with-nats-go-and-kubernetes-complete-production-guide-5352213b/):

| Component | Technology | Rationale |
|-----------|------------|-----------|
| HTTP Framework | [Fiber](https://gofiber.io/) | Fastest Go framework, Express-like API |
| gRPC | [grpc-go](https://grpc.io/) + [connect-go](https://connectrpc.com/) | Inter-service communication |
| Event Streaming | [Watermill](https://watermill.io/) | Broker-agnostic (Kafka/NATS/RabbitMQ) with built-in retry, dead letter queues |
| Message Broker | [NATS JetStream](https://nats.io/) or Kafka | NATS for simplicity, Kafka for scale (can switch via Watermill) |
| Kafka Client | [segmentio/kafka-go](https://github.com/segmentio/kafka-go) | Pure Go, no CGO dependencies |
| Database | PostgreSQL + [pgx](https://github.com/jackc/pgx) | Best Go Postgres driver |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) | SQL-based migrations |
| Config | [Viper](https://github.com/spf13/viper) | 12-factor app config |
| Logging | [zerolog](https://github.com/rs/zerolog) | Structured, zero-allocation |
| Validation | [validator](https://github.com/go-playground/validator) | Struct validation |
| Observability | [OpenTelemetry](https://opentelemetry.io/) | Distributed tracing, metrics |
| Testing | [testify](https://github.com/stretchr/testify) + [testcontainers](https://testcontainers.com/) | Mocking + integration tests |

**Why Watermill?** It provides a consistent API whether you're using NATS, Kafka, RabbitMQ, or AWS SQS - switch message brokers with minimal code changes. Built-in middleware for retry logic, correlation IDs, poison queues, and circuit breakers.

### Infrastructure

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Container Runtime | Docker | Standard containerization |
| Orchestration | Kubernetes (GKE) | Alpaca uses GCP, minimize latency |
| Message Broker | Apache Kafka (Confluent Cloud) | Managed Kafka, less ops burden |
| Cache | Redis | Session storage, rate limiting |
| Database | Cloud SQL (PostgreSQL 15) | Managed, HA, automatic backups |
| Time-series | TimescaleDB | Price history, analytics |
| Secrets | Google Secret Manager | Secure credential storage |
| Monitoring | Prometheus + Grafana | Metrics and dashboards |
| Tracing | OpenTelemetry + Jaeger | Distributed tracing |
| CI/CD | GitHub Actions | Native integration |

### External APIs

| Service | Provider | Purpose |
|---------|----------|---------|
| Stock Trading | [Alpaca](https://alpaca.markets/) | US stocks, 24/5 trading, fractional shares, Nasdaq member |
| Payments | [Safaricom Daraja 3.0](https://developer.safaricom.co.ke/) | M-Pesa STK Push, B2C, Ratiba (recurring), Security APIs |
| USSD/SMS | [Africa's Talking](https://africastalking.com/) | USSD gateway (same code across all telcos), SMS, Voice |
| KYC | [Daraja Security APIs](https://developer.safaricom.co.ke/) + [Smile Identity](https://www.usesmileid.com/) | Built-in M-Pesa KYC + ID verification, liveness |
| Email | [Resend](https://resend.com/) | Modern transactional emails |
| Push Notifications | Firebase Cloud Messaging | Mobile push |

**Note:** Daraja 3.0's Security APIs can handle basic KYC (fraud detection, identity verification) - use Smile Identity for advanced liveness checks and document verification for higher KYC tiers.

---

## Database Design

### Core Entities

```sql
-- Users & Authentication
users
├── id (UUID, PK)
├── phone (VARCHAR, UNIQUE, indexed)  -- Primary identifier for Kenya
├── email (VARCHAR, UNIQUE, nullable)
├── password_hash (VARCHAR)
├── pin_hash (VARCHAR)  -- For USSD/quick auth
├── kyc_status (ENUM: pending, submitted, verified, rejected)
├── kyc_tier (ENUM: tier1, tier2, tier3)  -- Different limits
├── alpaca_account_id (VARCHAR, nullable)
├── created_at, updated_at

-- Wallets (KES balance before converting to USD)
wallets
├── id (UUID, PK)
├── user_id (UUID, FK)
├── currency (ENUM: KES, USD)
├── balance (DECIMAL)
├── locked_balance (DECIMAL)  -- Pending orders
├── created_at, updated_at

-- Transactions (all money movements)
transactions
├── id (UUID, PK)
├── user_id (UUID, FK)
├── wallet_id (UUID, FK)
├── type (ENUM: deposit, withdrawal, trade_buy, trade_sell, fee, dividend)
├── amount (DECIMAL)
├── currency (ENUM: KES, USD)
├── status (ENUM: pending, completed, failed, reversed)
├── reference (VARCHAR)  -- M-Pesa transaction ID, etc.
├── metadata (JSONB)
├── created_at

-- Orders
orders
├── id (UUID, PK)
├── user_id (UUID, FK)
├── alpaca_order_id (VARCHAR, indexed)
├── symbol (VARCHAR)  -- AAPL, TSLA, etc.
├── side (ENUM: buy, sell)
├── type (ENUM: market, limit, stop)
├── quantity (DECIMAL)
├── filled_quantity (DECIMAL)
├── price (DECIMAL, nullable)  -- For limit orders
├── filled_avg_price (DECIMAL, nullable)
├── status (ENUM: pending, accepted, filled, partially_filled, cancelled, rejected)
├── source (ENUM: web, mobile, ussd, cli)
├── created_at, updated_at

-- Holdings (current portfolio)
holdings
├── id (UUID, PK)
├── user_id (UUID, FK)
├── symbol (VARCHAR)
├── quantity (DECIMAL)
├── avg_cost_basis (DECIMAL)
├── updated_at

-- Watchlists
watchlists
├── id (UUID, PK)
├── user_id (UUID, FK)
├── name (VARCHAR)
├── symbols (VARCHAR[])
├── created_at

-- Price Alerts
price_alerts
├── id (UUID, PK)
├── user_id (UUID, FK)
├── symbol (VARCHAR)
├── condition (ENUM: above, below)
├── target_price (DECIMAL)
├── triggered (BOOLEAN)
├── created_at
```

### Kafka Topics

```
equishare.orders.created      -- New order submitted
equishare.orders.updated      -- Order status changed
equishare.orders.filled       -- Order fully executed
equishare.payments.initiated  -- M-Pesa STK Push sent
equishare.payments.completed  -- M-Pesa payment confirmed
equishare.payments.failed     -- Payment failed
equishare.kyc.submitted       -- KYC documents uploaded
equishare.kyc.verified        -- KYC approved
equishare.kyc.rejected        -- KYC rejected
equishare.prices.realtime     -- Real-time price updates
equishare.alerts.triggered    -- Price alert hit
equishare.notifications.send  -- Notification to be sent
```

---

## Security Architecture

### Authentication Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    Authentication Flow                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  USSD/CLI:  Phone + PIN (4-6 digits)                        │
│             └─► Rate limited (3 attempts, 15 min lockout)   │
│                                                              │
│  Web/Mobile: Phone + Password + Optional 2FA               │
│              └─► JWT (15 min) + Refresh Token (7 days)      │
│              └─► Refresh tokens stored in Redis             │
│                                                              │
│  Sensitive Actions (withdraw, large trades):                 │
│              └─► Re-authentication required                  │
│              └─► SMS OTP verification                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### KYC Tiers (Regulatory Compliance)

| Tier | Requirements | Daily Limit (KES) | Features |
|------|--------------|-------------------|----------|
| Tier 1 | Phone verification only | 10,000 | View prices, paper trading |
| Tier 2 | ID + Selfie verification | 100,000 | Trade US stocks |
| Tier 3 | Address proof + Income source | 1,000,000+ | Higher limits, margin |

### Security Measures

- **Data Encryption**: AES-256 at rest, TLS 1.3 in transit
- **PIN Storage**: Argon2id hashing
- **API Security**: Rate limiting, request signing for sensitive endpoints
- **Audit Logging**: All financial actions logged immutably
- **Secrets Management**: Google Secret Manager, never in code
- **Dependency Scanning**: Dependabot + Snyk
- **Penetration Testing**: Before launch (required by CMA)

---

## USSD Menu Design

```
*XXX# (Main Menu)
├── 1. Buy Shares
│   ├── 1. Search by Name
│   │   └── Enter company name: [input]
│   │       └── Select: 1.AAPL 2.AMZN 3.More
│   │           └── Enter amount (KES): [input]
│   │               └── Confirm: Buy KES 500 of AAPL? 1.Yes 2.No
│   └── 2. Popular Stocks
│       └── 1.Apple 2.Tesla 3.Amazon 4.Microsoft 5.More
│
├── 2. Sell Shares
│   └── Your Holdings:
│       └── 1.AAPL (5 shares) 2.TSLA (2 shares)
│           └── Sell how many? [input]
│               └── Confirm: Sell 2 AAPL shares? 1.Yes 2.No
│
├── 3. My Portfolio
│   └── Holdings: KES 25,430 (+5.2%)
│       └── 1.AAPL: KES 12,500 2.TSLA: KES 8,430 3.More
│
├── 4. Deposit (M-Pesa)
│   └── Enter amount (KES): [input]
│       └── STK Push sent to 0712XXXXXX
│
├── 5. Withdraw
│   └── Balance: KES 5,000
│       └── Enter amount: [input]
│           └── Enter PIN to confirm: [input]
│
├── 6. Price Alerts
│   └── 1.Set Alert 2.My Alerts
│
└── 0. Help
    └── Call 0800-XXX-XXX or SMS HELP to XXX
```

---

## API Design Principles

### REST API (Public)

```yaml
# Versioned API
/api/v1/

# Resources
/api/v1/auth/register
/api/v1/auth/login
/api/v1/auth/refresh
/api/v1/auth/verify-otp

/api/v1/users/me
/api/v1/users/me/kyc

/api/v1/wallet
/api/v1/wallet/deposit
/api/v1/wallet/withdraw
/api/v1/wallet/transactions

/api/v1/market/quotes/{symbol}
/api/v1/market/search?q=apple
/api/v1/market/popular

/api/v1/orders
/api/v1/orders/{id}
/api/v1/orders/{id}/cancel

/api/v1/portfolio
/api/v1/portfolio/holdings
/api/v1/portfolio/history

/api/v1/watchlist
/api/v1/alerts
```

### gRPC (Internal Services)

```protobuf
// trading.proto
service TradingService {
  rpc CreateOrder(CreateOrderRequest) returns (Order);
  rpc GetOrder(GetOrderRequest) returns (Order);
  rpc CancelOrder(CancelOrderRequest) returns (Order);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
}

// portfolio.proto
service PortfolioService {
  rpc GetHoldings(GetHoldingsRequest) returns (Holdings);
  rpc GetPortfolioValue(GetPortfolioValueRequest) returns (PortfolioValue);
  rpc GetPerformance(GetPerformanceRequest) returns (Performance);
}
```

---

## Development Phases

### Phase 1: Foundation (MVP)
- Core infrastructure setup
- User authentication (phone + PIN)
- Basic KYC (Tier 1)
- M-Pesa deposits
- Paper trading mode
- USSD basic flow

### Phase 2: Real Trading
- Alpaca integration
- Real order execution
- Portfolio tracking
- Full KYC (Tier 2)
- Web app

### Phase 3: Enhanced Experience
- Mobile app (React Native)
- Price alerts
- Watchlists
- CLI tool
- Advanced order types

### Phase 4: Scale & Expand
- NSE integration (Kenya stocks)
- London Stock Exchange
- Margin trading
- Social features (copy trading)

---

## Regulatory Considerations

### Kenya CMA Requirements (2025-2026)

Based on [CMA Kenya licensing guidelines](https://www.cma.or.ke/licensing/) and [Chambers Fintech 2024 Kenya Guide](https://practiceguides.chambers.com/practice-guides/fintech-2024/kenya):

1. **License Type**: Investment Adviser or Authorized Securities Dealer
2. **Capital Requirements**: Minimum paid-up capital (varies by license)
3. **Fit & Proper Test**: Directors must pass background checks
4. **Compliance Officer**: Must appoint a compliance officer
5. **Audit**: Annual audit by approved auditor
6. **Investor Protection**: Client money segregation
7. **Trading System Approval**: CMA must approve any trading system before implementation

**Recent Developments:**
- Capital Markets (Amendment) Bill 2023 shows intent to regulate digital assets
- Virtual Asset Service Providers Bill tabled in National Assembly (March 2024)
- Nairobi International Financial Centre (NIFC) launched - creating favorable environment
- Regulators intensifying efforts to promote digital stock trading

### Recommended Approach

1. **Start with paper trading** - No license required for simulation
2. **Partner with licensed stockbroker** - White-label their license initially (e.g., Standard Investment Bank, Genghis Capital)
3. **Apply for own license** - Once traction proven (application fee: KES 2,500)
4. **Alpaca handles US compliance** - They're SEC/FINRA registered, achieved Nasdaq membership in 2025
5. **Use Daraja 3.0 Security APIs** - Leverage M-Pesa's built-in KYC to reduce compliance burden

---

## Monitoring & Observability

### Key Metrics

```
# Business Metrics
equishare_users_registered_total
equishare_users_kyc_verified_total
equishare_orders_total{side="buy|sell", status="filled|cancelled"}
equishare_order_value_kes_total
equishare_deposits_total
equishare_withdrawals_total

# Technical Metrics
equishare_api_request_duration_seconds
equishare_api_requests_total{endpoint, status}
equishare_kafka_consumer_lag
equishare_alpaca_api_latency_seconds
equishare_mpesa_callback_duration_seconds
```

### Alerting Rules

- Order fill rate < 95% → P1
- M-Pesa callback latency > 5s → P2
- Kafka consumer lag > 1000 → P2
- API error rate > 1% → P2
- Database connection pool exhausted → P1

---

## Cost Estimation (Monthly)

| Component | Provider | Estimated Cost |
|-----------|----------|----------------|
| Kubernetes (GKE) | Google Cloud | $150-300 |
| Cloud SQL (PostgreSQL) | Google Cloud | $50-100 |
| Kafka | Confluent Cloud | $0 (free tier) → $200 |
| Redis | Redis Cloud | $0 (free tier) → $50 |
| Africa's Talking | AT | Pay per transaction |
| M-Pesa API | Safaricom | Transaction fees only |
| Alpaca | Alpaca | Free (they earn on spread) |
| **Total** | | **$200-650/month** |

---

## Next Steps

See GitHub Issues for detailed implementation tasks organized by epic.

---

## References

### Platform APIs
- [Alpaca API Documentation](https://docs.alpaca.markets/)
- [Alpaca 2025 Review](https://alpaca.markets/blog/alpacas-2025-in-review/)
- [Safaricom Daraja 3.0 Launch](https://techcabal.com/2025/11/25/safaricom-overhauls-m-pesa-api-platform/)
- [Safaricom Developer Portal](https://developer.safaricom.co.ke/)
- [Africa's Talking USSD](https://africastalking.com/ussd)

### Architecture & Best Practices
- [Event-Driven Microservices with NATS, Go, and Kubernetes](https://golang.elitedev.in/golang/building-event-driven-microservices-with-nats-go-and-kubernetes-complete-production-guide-5352213b/)
- [Watermill - Event-Driven Go](https://watermill.io/)
- [Outbox Pattern in Go with NATS](https://dev.to/zanzythebar/outbox-done-right-in-go-building-resilient-event-driven-systems-with-nats-and-sql-32gh)
- [Three Dots Labs - Go Event-Driven Training](https://threedots.tech/event-driven/)

### Market Research
- [African Fintech 2026 Outlook](https://techpoint.africa/guide/african-fintech-outlook/)
- [Kenya Fintech 2026 Landscape](https://sdk.finance/blog/fintech-kenya-2025-landscape-overview-growth-drivers-and-barriers/)
- [Investment Apps in Africa 2025](https://afritechbizhub.com/digital-transformation/investment-apps-in-africa/2025/11/25/)
- [Stock Trading Apps in Nigeria 2025](https://techcabal.com/2025/07/04/stock-investing-apps-nigerians-are-using-to-trade-in-2025/)

### Regulatory
- [Kenya CMA Licensing](https://www.cma.or.ke/licensing/)
- [Chambers Fintech 2024 Kenya](https://practiceguides.chambers.com/practice-guides/fintech-2024/kenya)
