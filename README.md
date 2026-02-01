# Tracker Scrapper

A production-ready Private API that integrates with WooCommerce to retrieve order information and scrape Colombian courier tracking pages. Built with **Domain-Driven Design (DDD)** and **Hexagonal Architecture** (Ports and Adapters), organized by **Features/Bounded Contexts** for scalability, maintainability, and testability.

## 🚀 Features

### Core Functionality
- **WooCommerce Integration**: Fetch order details with email validation
- **Multi-Courier Support**: Automated tracking scraping for:
  - 🚚 Coordinadora (JSON API scraping)
  - 📦 Servientrega (Browser automation with go-rod)
  - 🏃 Interrapidisimo (JSON API scraping)
- **Redis Caching**: Mandatory caching layer with configurable TTL
  - Order cache: `order_{id}_{email}` (default 1 hour)
  - Tracking cache: `ts_{courier}_{number}` (default 30 minutes)
- **Swagger/OpenAPI Documentation**: Interactive API documentation at `/swagger/index.html`

### Architecture Highlights
- **Hexagonal Architecture**: Clean separation of domain, ports, and adapters
- **Dependency Injection**: All services properly wired with dependencies
- **Comprehensive Testing**: 60% overall coverage with critical paths at 80-100%
- **Structured Logging**: Zap logger with request IDs and context tracking
- **Configuration Management**: Environment-based config with validation

## 📁 Project Structure

```text
cmd/
└── api/                        # Main entry point
internal/
├── core/                       # Infrastructure & Shared Kernel
│   ├── cache/                  # Cache port & Redis adapter
│   │   ├── ports.go           # Cache interface
│   │   └── redis_adapter.go   # Redis implementation
│   ├── config/                # Viper configuration with validation
│   ├── httpclient/            # HTTP client wrapper with logging
│   ├── logger/                # Zap logger setup
│   └── server/                # Fiber HTTP server
└── features/                  # Bounded Contexts
    ├── orders/
    │   ├── domain/            # Order entities & value objects
    │   ├── ports/             # Interfaces (OrderProvider)
    │   ├── service/           # Business logic with cache
    │   ├── handler/           # HTTP handlers
    │   └── adapters/          # WooCommerce adapter
    └── tracking/
        ├── domain/            # Tracking entities & status enums
        ├── ports/             # Interfaces (TrackingProvider)
        ├── service/           # Business logic with cache
        ├── handler/           # HTTP handlers
        └── adapters/          # Courier implementations
            ├── coordinadora_adapter.go
            ├── servientrega_adapter.go
            └── interrapidisimo_adapter.go
docs/
├── COORDINADORA.md            # Coordinadora implementation docs
├── SERVIENTREGA.md            # Servientrega implementation docs
└── INTERRAPIDISIMO.md         # Interrapidisimo implementation docs
```

## 🛠️ Getting Started

### Prerequisites
- Go 1.20+
- Redis 7+ (required for caching)
- Docker (optional, for running Redis)

### Environment Setup

Create a `.env` file in the root directory:

```env
# Application Settings
APP_ENV=development
LOG_LEVEL=debug
SERVER_PORT=8080

# WooCommerce Integration
WC_URL=https://your-woocommerce-site.com
WC_CONSUMER_KEY=ck_your_consumer_key_here
WC_CONSUMER_SECRET=cs_your_consumer_secret_here

# Courier Tracking URLs
COURIER_COORDINADORA_CO=https://coordinadora.com/rastreo/rastreo-de-guia/detalle-de-rastreo-de-guia/?guia=
COURIER_SERVIENTREGA_CO=https://mobile.servientrega.com/WebSitePortal/RastreoEnvioDetalle.html?Guia=
COURIER_INTERRAPIDISIMO_CO=https://www3.interrapidisimo.com/SiguetuEnvio/shipment

# Redis Cache Configuration (REQUIRED)
CACHE_REDIS_URL=redis://localhost:6379
CACHE_ORDER_TTL=3600          # Order cache TTL in seconds (1 hour)
CACHE_TRACKING_TTL=1800       # Tracking cache TTL in seconds (30 minutes)
```

### Installation & Running

1. **Install dependencies:**
   ```bash
   go mod download
   ```

2. **Start Redis:**
   ```bash
   # Using Docker
   docker run -d -p 6379:6379 redis:7-alpine
   
   # Or use your local Redis installation
   redis-server
   ```

3. **Generate Swagger Documentation (optional):**
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ~/go/bin/swag init -g cmd/api/main.go -o docs/swagger
   ```

4. **Run the application:**
   ```bash
   go run cmd/api/main.go
   ```

5. **Access the API:**
   - API Base: `http://localhost:8080`
   - Swagger UI: `http://localhost:8080/swagger/index.html`
   - Swagger JSON: `http://localhost:8080/swagger/doc.json`

## 📡 API Endpoints

### Orders
- `GET /orders/:id?email=user@example.com`
  - Retrieve order by ID with email validation
  - Returns order details with tracking information
  - Cached for 1 hour (configurable)

### Tracking
- `GET /tracking/:number?courier=coordinadora_co`
  - Get tracking history for a tracking number
  - Supported couriers: `coordinadora_co`, `servientrega_co`, `interrapidisimo_co`
  - Cached for 30 minutes (configurable)

## 🧪 Testing

### Run All Tests
```bash
go test ./... -v
```

### Run Tests with Coverage
```bash
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out  # View HTML coverage report
```

### Current Test Coverage
- **Overall Coverage**: ~60%
- **Core Infrastructure**: 83-100% (cache, config, server, httpclient)
- **Tracking Handler**: 84.6%
- **Tracking Service**: 83.3%
- **Order Adapters**: 79.2%

## 🔧 Technology Stack

- **Framework**: [Fiber v2](https://gofiber.io/) - Fast HTTP framework
- **Cache**: [go-redis/v9](https://github.com/redis/go-redis) - Redis client
- **Browser Automation**: [go-rod](https://github.com/go-rod/rod) - For Servientrega scraping
- **Logging**: [zap](https://github.com/uber-go/zap) - Structured logging
- **Configuration**: [Viper](https://github.com/spf13/viper) - Config management
- **API Docs**: [swaggo/swag](https://github.com/swaggo/swag) - Swagger generation
- **Testing**: [testify](https://github.com/stretchr/testify) - Testing assertions

## 📋 Code Standards

### Architecture Principles
- **Hexagonal Architecture**: Domain logic isolated from infrastructure
- **Dependency Injection**: All dependencies injected through constructors
- **Interface-Based Design**: All external dependencies defined as ports
- **Single Responsibility**: Each package has one clear purpose

### Code Quality
- **Modularity & Reusability**: Code broken into small, reusable components
- **Self-Explanatory**: Clear naming and logical structure
- **Documentation**: GoDocs for all exported types and functions
- **Testing**: Comprehensive test coverage with unit and integration tests
- **Error Handling**: Descriptive errors with proper wrapping

## 🔍 Debugging & Troubleshooting

### Common Issues

**Redis Connection Failed:**
```bash
# Ensure Redis is running
docker ps | grep redis
# Or check logs
docker logs <redis-container-id>
```

**WooCommerce Connection Failed:**
- Verify `WC_URL`, `WC_CONSUMER_KEY`, and `WC_CONSUMER_SECRET` are correct
- Check that the WooCommerce REST API is enabled
- Ensure your IP is not blocked by the WooCommerce site

**Tracking Scraping Timeout:**
- Servientrega uses browser automation (slower, ~3-4 seconds)
- Coordinadora and Interrapidisimo use direct API calls (faster, <1 second)
- Check courier website availability

## 📚 Documentation

Detailed implementation documentation for each courier adapter:
- [Coordinadora](docs/COORDINADORA.md) - JSON API scraping details
- [Servientrega](docs/SERVIENTREGA.md) - Browser automation workflow
- [Interrapidisimo](docs/INTERRAPIDISIMO.md) - API integration guide

## 🚦 Application Lifecycle

1. **Startup**:
   - Load configuration from `.env` and environment variables
   - Validate required fields
   - Initialize Zap logger
   - Connect to WooCommerce (health check)
   - Connect to Redis (health check, fails if unavailable)
   - Wire up services with cache dependencies
   - Start Fiber HTTP server

2. **Request Flow**:
   - Request received → Request ID middleware → Logger middleware
   - Handler validates input → Service checks cache
   - Cache hit: Return cached data
   - Cache miss: Call provider → Cache result → Return data

3. **Graceful Shutdown**:
   - Close Redis connection
   - Flush logger buffers

## 📄 License

This is a private API project.
