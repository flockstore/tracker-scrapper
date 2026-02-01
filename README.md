# Tracker Scrapper

This project is a Private API designed to obtain order information and scrape courier pages based on that data. It implements Domain-Driven Design (DDD) and Hexagonal Architecture (Ports and Adapters), organized by **Features/Bounded Contexts** to ensure scalability, isolation, and flexibility.

## Architecture

The project structure is split into two main areas:
1.  **Core Infrastructure (`/internal/core`)**: The foundational elements "outside the hexagon" that support the application (Configuration, Logger, Server, etc.).
2.  **Features (`/internal/features`)**: The business capabilities, each complying with Hexagonal Architecture.

### Directory Layout

```text
cmd/
└── api/                # Main entry point
internal/
├── core/               # Infrastructure & Shared Kernel
│   ├── config/         # Viper configuration with struct tags (defaults/required)
│   ├── logger/         # Zap logger setup
│   └── server/         # HTTP Server setup
└── features/           # Bounded Contexts
    ├── orders/
    │   ├── domain/     # Entities & Value Objects
    │   ├── ports/      # Primary & Secondary Ports
    │   └── adapters/   # implementations (HTTP handlers, Repositories)
    ├── tracking/
    │   ├── domain/
    │   ├── ports/
    │   └── adapters/
    ├── couriers/
    │   ├── domain/
    │   ├── ports/
    │   └── adapters/   # Scrappers (DHL, FedEx, etc.)
    └── customer/
        ├── domain/
        ├── ports/
        └── adapters/
```

## Features

### Infrastructure Core
-   **Configuration**: Powered by `viper`. Supports `.env` files and environment variables. Uses struct tags to enforce required values and define defaults.
-   **Logging**: Structured logging using `uber-go/zap` with pretty console output for development.

### Business Features
-   **Orders**: Management of order lifecycle.
-   **Tracking**: Core tracking logic connecting generic orders to specific courier updates.
-   **Couriers**: Adapter implementations for different shipping providers (Hot-swappable).
-   **Customers**: Customer data management.

## Getting Started

### Prerequisites
-   Go 1.20+

### Environment Setup
Create a `.env` file in the root directory:
```env
APP_ENV=development
LOG_LEVEL=debug
SERVER_PORT=8080

# WooCommerce Integration
WC_URL=https://your-woocommerce-site.com
WC_CONSUMER_KEY=ck_your_consumer_key
WC_CONSUMER_SECRET=cs_your_consumer_secret
```

### Generate Swagger Documentation
Swagger docs are generated locally and not committed to the repository.

1. Install swag CLI:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

2. Generate documentation:
```bash
~/go/bin/swag init -g cmd/api/main.go --output docs/swagger
```

3. Access Swagger UI at `http://localhost:8080/swagger/index.html` after starting the server.

### Running the Application
```bash
go run cmd/api/main.go
```

### Running Tests
```bash
go test ./...
```

## Code Standards

-   **Modularity & Reusability**: Code should be broken down into small, reusable components.
-   **Self-Explanatory**: Logic should be clear and readable without the need for inline comments.
-   **Documentation**: Use GoDocs for all exported functions, structs, interfaces, and fields. Avoid comments inside function bodies.
-   **Testing**: Maintain high test coverage with comprehensive unit tests for all components.
