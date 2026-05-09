# Tracker Scrapper

A production-ready Private API that integrates with Mannaiah to retrieve order information and scrape Colombian courier tracking pages. Built with **Domain-Driven Design (DDD)** and **Hexagonal Architecture** (Ports and Adapters), organized by **Features/Bounded Contexts** for scalability, maintainability, and testability.

## 🚀 Features

### Core Functionality
- **Mannaiah Integration**: Fetch order details with email validation through M2M auth
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
    │   └── adapters/          # Mannaiah adapter
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

# Mannaiah Backend Integration
MANNAIAH_BACKEND_URL=https://api.flockstore.co
LOGTO_M2M_ENDPOINT=https://your-logto-tenant
# LOGTO_M2M_TOKEN_ENDPOINT=https://your-logto-tenant/oidc/token
LOGTO_M2M_APP_ID=your_m2m_app_id
LOGTO_M2M_APP_SECRET=your_m2m_app_secret
LOGTO_M2M_SCOPE=order:view contact:view shipping:quotations

# Courier Tracking URLs
COURIER_COORDINADORA_CO=https://coordinadora.com/rastreo/rastreo-de-guia/detalle-de-rastreo-de-guia/?guia=
COURIER_SERVIENTREGA_CO=https://mobile.servientrega.com/WebSitePortal/RastreoEnvioDetalle.html?Guia=
COURIER_INTERRAPIDISIMO_CO=https://www3.interrapidisimo.com/SiguetuEnvio/shipment

# Proxy Configuration (Optional - for non-Colombian servers)
# PROXY_HOSTNAME=geo.iproyal.com
# PROXY_PORT=12321
# PROXY_USERNAME=your_username
# PROXY_PASSWORD=your_password
# PROXY_SERVIENTREGA=true

# Redis Cache Configuration (REQUIRED)
CACHE_REDIS_URL=redis://localhost:6379
CACHE_ORDER_TTL=3600          # Order cache TTL in seconds (1 hour)
CACHE_TRACKING_TTL=1800       # Tracking cache TTL in seconds (30 minutes)
```

## 🌐 Proxy Configuration (Non-Colombian Servers)

When deploying outside Colombia (AWS, DigitalOcean, VPS, etc.), Colombian courier websites may block datacenter IP addresses. You'll need a **residential proxy** to access their tracking APIs.

### When Proxies Are Needed

- ✅ **Local development in Colombia**: No proxy needed
- ✅ **Docker/Server in Colombia**: Usually no proxy needed  
- ❌ **AWS, GCP, Azure, VPS outside Colombia**: Proxy required for some couriers
- ❌ **Servientrega specifically**: Often blocks non-Colombian IPs

### Symptoms of IP Blocking

```
dial tcp 190.145.160.56:443: i/o timeout
```

If you see timeout errors like this while other couriers work fine, you need a proxy.

### Recommended Proxy Providers

| Provider | Type | Cost | Notes |
|----------|------|------|-------|
| **IPRoyal** | Residential | ~$7/GB | Pay-as-you-go, Colombia IPs available |
| **Webshare** | Datacenter | $5.49/mo | Cheaper but may not work |
| **ProxyScrape** | Free | Free | Unreliable, for testing only |

**Cost estimate**: ~$12 USD for 2GB, which lasts for approximately 40,000 requests.

### Testing Your Proxy

Before configuring, **always test with curl** to verify the proxy works:

```bash
# Test proxy connectivity to Servientrega
curl -x http://user:pass@proxy:port --connect-timeout 15 -I \
  "https://mobile.servientrega.com/WebSitePortal/RastreoEnvioDetalle.html?Guia=2259200012"

# Expected: HTTP/1.1 200 OK
```

### Configuration

Set proxy credentials and enable per-courier in your `.env` file:

```env
# Shared proxy credentials
PROXY_HOSTNAME=geo.iproyal.com
PROXY_PORT=12321
PROXY_USERNAME=your_username
PROXY_PASSWORD=your_password_country-co

# Enable proxy per courier (true/false)
PROXY_SERVIENTREGA=true
PROXY_COORDINADORA=false
PROXY_INTERRAPIDISIMO=false
```

| Variable | Description |
|----------|-------------|
| `PROXY_HOSTNAME` | Proxy server hostname (e.g., `geo.iproyal.com`) |
| `PROXY_PORT` | Proxy server port (e.g., `12321`) |
| `PROXY_USERNAME` | Proxy authentication username |
| `PROXY_PASSWORD` | Proxy authentication password (include geo-targeting suffix like `_country-co`) |
| `PROXY_SERVIENTREGA` | Enable proxy for Servientrega (`true`/`false`) |
| `PROXY_COORDINADORA` | Enable proxy for Coordinadora (`true`/`false`) |
| `PROXY_INTERRAPIDISIMO` | Enable proxy for Interrapidisimo (`true`/`false`) |

> **⚠️ Important**: Use your own proxy and test with curl commands first. Take advantage of free trials from proxy providers before making any payment.



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

**Mannaiah Connection Failed or Forbidden:**
- Verify `MANNAIAH_BACKEND_URL`, `LOGTO_M2M_APP_ID`, and `LOGTO_M2M_APP_SECRET` are correct
- Ensure the M2M app has `order:view`, `contact:view`, and `shipping:quotations` scopes
- Check backend logs for `Mannaiah request returned non-success status`; 401/403 responses include a bounded upstream body excerpt

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
   - Connect to Mannaiah (health check)
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
