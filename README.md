# Scalable E-commerce Platform 🛒

A modern, high-performance e-commerce backend platform built with Go, designed for scalability, reliability, and maintainability. This platform provides a comprehensive API for managing users, products, shopping carts, orders, payments, and notifications.

## 🚀 Features

### Core Functionality
- **User Management**: Registration, authentication, and profile management
- **Product Catalog**: Full CRUD operations for product management
- **Shopping Cart**: Add, update, and manage cart items
- **Order Processing**: Complete order lifecycle management
- **Payment Integration**: Stripe payment processing with webhook support
- **Notification System**: Email notifications via SendGrid

### Technical Features
- **RESTful API**: Well-documented API endpoints with Swagger/OpenAPI
- **JWT Authentication**: Secure token-based authentication
- **Database Integration**: PostgreSQL with optimized queries
- **Redis Caching**: High-performance caching layer
- **Rate Limiting**: Built-in request rate limiting
- **Health Checks**: Comprehensive health monitoring endpoints
- **Observability**: OpenTelemetry integration for tracing and metrics
- **Docker Support**: Containerized deployment ready
- **Cross-Platform Builds**: Support for Linux, macOS, and Windows

## 📋 Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the Application](#running-the-application)
- [API Documentation](#api-documentation)
- [Development](#development)
- [Testing](#testing)
- [Deployment](#deployment)
- [Contributing](#contributing)
- [License](#license)

## 📦 Prerequisites

- **Go**: Version 1.22 or higher
- **PostgreSQL**: Version 12 or higher
- **Redis**: Version 6 or higher
- **Docker** (optional): For containerized deployment
- **Make**: For using the provided Makefile commands

## 🛠 Installation

### 1. Clone the Repository

```bash
git clone https://github.com/aaravmahajanofficial/Scalable-Ecommerce-Platform.git
cd Scalable-Ecommerce-Platform
```

### 2. Install Dependencies

```bash
make deps
```

### 3. Install Development Tools

```bash
make tools-install
```

## ⚙️ Configuration

### Environment Variables

Create a configuration file or set the following environment variables:

#### Required Variables
```bash
# Application Environment
ENV=development

# Database Configuration
PG_HOST=localhost
PG_PORT=5432
PG_USER=your_db_user
PG_PASSWORD=your_db_password
PG_DBNAME=ecommerce_db
PG_SSLMODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_USER=your_redis_user
REDIS_PASSWORD=your_redis_password

# Security
JWT_KEY=your_super_secret_jwt_key
JWT_EXPIRY_HOURS=24

# Stripe Configuration
STRIPE_API_KEY=sk_test_your_stripe_secret_key
STRIPE_WEBHOOK_SECRET=whsec_your_webhook_secret
STRIPE_SUPPORTED_CURRENCIES=inr,usd,eur

# SendGrid Configuration
API_KEY=your_sendgrid_api_key
FROM_EMAIL=noreply@yourcompany.com
FROM_NAME=Your Company Name
SMSENABLED=false
```

#### Optional Variables
```bash
# HTTP Server Configuration
HTTP_ADDRESS=:8085
HTTP_READ_TIMEOUT=30s
HTTP_WRITE_TIMEOUT=30s
HTTP_IDLE_TIMEOUT=120s

# OpenTelemetry Configuration
OTEL_SERVICE_NAME=scalable-ecommerce-platform
OTEL_EXPORTER_ENDPOINT=http://localhost:4318/v1/traces
OTEL_TRACES_SAMPLER_ARG=1.0

# Cache Configuration
CACHE_DEFAULT_TTL=5m

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m
```

### Configuration File

Alternatively, create a YAML configuration file (e.g., `config/local.yaml`):

```yaml
env: development
http_server:
  ADDRESS: ":8085"
  READ_TIMEOUT: 30s
  WRITE_TIMEOUT: 30s
  IDLE_TIMEOUT: 120s
  SHUTDOWN_TIMEOUT: 30s
  GRACEFUL_SHUTDOWN_TIMEOUT: 5s

database:
  PG_HOST: localhost
  PG_PORT: "5432"
  PG_USER: your_db_user
  PG_PASSWORD: your_db_password
  PG_DBNAME: ecommerce_db
  PG_SSLMODE: disable

redis:
  REDIS_HOST: localhost
  REDIS_PORT: "6379"
  REDIS_USER: your_redis_user
  REDIS_PASSWORD: your_redis_password

security:
  JWT_KEY: your_super_secret_jwt_key
  JWT_EXPIRY_HOURS: 24

stripe:
  API_KEY: sk_test_your_stripe_secret_key
  WEBHOOK_SECRET: whsec_your_webhook_secret
  SUPPORTED_CURRENCIES: ["inr", "usd", "eur"]

sendgrid:
  API_KEY: your_sendgrid_api_key
  FROM_EMAIL: noreply@yourcompany.com
  FROM_NAME: Your Company Name
  SMSENABLED: false

otel:
  SERVICE_NAME: scalable-ecommerce-platform
  EXPORTER_ENDPOINT: http://localhost:4318/v1/traces
  SAMPLER_RATIO: 1.0

cache:
  default_ttl: 5m
```

## 🏃‍♂️ Running the Application

### Development Mode

1. **Start dependencies** (PostgreSQL and Redis)
2. **Run the application**:

```bash
# Using Make
make run

# Or directly with Go
go run ./cmd/scalable-ecommerce-platform --config config/local.yaml
```

### Using Docker

```bash
# Build and run with Docker
make docker-build
make docker-run

# Or using docker-compose (if available)
docker-compose up
```

### Available Make Commands

```bash
# View all available commands
make help

# Common commands
make build          # Build the application
make test           # Run tests
make test-coverage  # Generate test coverage report
make lint           # Run linter
make fmt            # Format code
make docs           # Generate API documentation
make clean          # Clean build artifacts
```

## 📚 API Documentation

### Swagger/OpenAPI Documentation

Once the application is running, access the interactive API documentation at:

```
http://localhost:8085/swagger/index.html
```

### Health Check Endpoints

- **Liveness**: `GET /livez` - Check if the application is alive
- **Readiness**: `GET /readyz` - Check if the application is ready to serve traffic

### Core API Endpoints

#### Authentication
- `POST /api/v1/users/register` - Register a new user
- `POST /api/v1/users/login` - User login
- `GET /api/v1/users/profile` - Get user profile (authenticated)

#### Products
- `POST /api/v1/products` - Create a new product (authenticated)
- `GET /api/v1/products/{id}` - Get product by ID (authenticated)
- `GET /api/v1/products` - List products with pagination (authenticated)
- `PUT /api/v1/products/{id}` - Update product (authenticated)
- `DELETE /api/v1/products/{id}` - Delete product (authenticated)

#### Shopping Cart
- `POST /api/v1/carts` - Add item to cart (authenticated)
- `GET /api/v1/carts` - Get user's cart (authenticated)
- `PUT /api/v1/carts/{id}` - Update cart item (authenticated)
- `DELETE /api/v1/carts/{id}` - Remove item from cart (authenticated)

#### Orders
- `POST /api/v1/orders` - Create a new order (authenticated)
- `GET /api/v1/orders/{id}` - Get order by ID (authenticated)
- `GET /api/v1/orders` - List user's orders (authenticated)

#### Payments
- `POST /api/v1/payments/process` - Process payment (authenticated)
- `POST /api/v1/payments/webhook` - Stripe webhook endpoint

## 🔧 Development

### Project Structure

```
├── cmd/
│   └── scalable-ecommerce-platform/   # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/                  # HTTP handlers
│   │   └── middleware/                # HTTP middleware
│   ├── cache/                         # Caching layer
│   ├── config/                        # Configuration management
│   ├── errors/                        # Error definitions
│   ├── health/                        # Health check handlers
│   ├── metrics/                       # Metrics collection
│   ├── models/                        # Data models
│   ├── repositories/                  # Data access layer
│   ├── services/                      # Business logic
│   └── utils/                         # Utility functions
├── pkg/
│   ├── sendgrid/                      # SendGrid integration
│   └── stripe/                        # Stripe integration
├── docs/                              # API documentation
└── prometheus/                        # Prometheus configuration
```

### Code Generation

Generate mocks and other generated code:

```bash
make generate
make mock
```

### Linting and Formatting

```bash
# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet
```

### Database Migrations

Database schema is managed through the application. The database tables will be created automatically when the application starts if they don't exist.

## 🧪 Testing

### Run Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run benchmarks
make benchmark
```

### Test Coverage

The project maintains high test coverage across all modules:
- Handlers: ~97.6%
- Services: ~96.2%
- Cache: 100%
- Repositories: ~81.9%

## 🐳 Deployment

### Docker Deployment

1. **Build the Docker image**:
```bash
make docker-build
```

2. **Run the container**:
```bash
make docker-run
```

3. **Push to registry**:
```bash
make docker-push
```

### Production Configuration

For production deployments:

1. Use environment variables for sensitive configuration
2. Enable SSL/TLS termination
3. Configure proper logging levels
4. Set up monitoring and alerting
5. Use a reverse proxy (nginx, HAProxy)
6. Configure database connection pooling
7. Set up Redis clustering for high availability

### Environment-Specific Builds

```bash
# Build for different platforms
make build-linux      # Linux
make build-mac         # macOS
make build-windows     # Windows
make build-all         # All platforms
```

## 🔒 Security

### Security Features

- **JWT Authentication**: Secure token-based authentication
- **Input Validation**: Comprehensive input validation using go-playground/validator
- **SQL Injection Protection**: Parameterized queries with sqlx
- **XSS Protection**: HTML sanitization with bluemonday
- **Rate Limiting**: Built-in request rate limiting
- **CORS Support**: Configurable CORS policies

### Security Best Practices

1. Keep dependencies updated
2. Use HTTPS in production
3. Rotate JWT secrets regularly
4. Monitor for security vulnerabilities
5. Implement proper logging and monitoring
6. Use environment variables for secrets

## 📊 Monitoring and Observability

### Metrics

The application exposes Prometheus metrics at `/metrics` endpoint, including:
- HTTP request duration and count
- Database connection metrics
- Redis connection metrics
- Custom business metrics

### Tracing

OpenTelemetry integration provides distributed tracing:
- HTTP request tracing
- Database query tracing
- External service call tracing

### Logging

Structured logging using Go's `slog` package with different log levels and contextual information.

## 🤝 Contributing

We welcome contributions! Please follow these guidelines:

### Development Workflow

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/your-feature-name`
3. **Make your changes**
4. **Run tests**: `make test`
5. **Run linters**: `make lint`
6. **Commit your changes**: `git commit -am 'Add some feature'`
7. **Push to the branch**: `git push origin feature/your-feature-name`
8. **Create a Pull Request**

### Code Standards

- Follow Go conventions and best practices
- Write comprehensive tests for new features
- Update documentation as needed
- Use meaningful commit messages
- Ensure all CI checks pass

### Reporting Issues

Please use the GitHub issue tracker to report bugs or request features. Include:
- Detailed description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](http://www.apache.org/licenses/LICENSE-2.0.html) file for details.

## 👥 Contact

- **Author**: Aarav Mahajan
- **Email**: [aaravmahajan2003@gmail.com](mailto:aaravmahajan2003@gmail.com)
- **GitHub**: [@aaravmahajanofficial](https://github.com/aaravmahajanofficial)

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/)
- Uses [Stripe](https://stripe.com/) for payment processing
- Uses [SendGrid](https://sendgrid.com/) for email notifications
- API documentation with [Swagger](https://swagger.io/)
- Observability with [OpenTelemetry](https://opentelemetry.io/)

---

**Happy Coding!** 🚀
