# Pippaothy

A modern, lightweight web application built with Go, featuring session-based authentication, structured logging, and real-time log viewing capabilities.

## Features

- **Session-based Authentication**: Secure user registration and login system with CSRF protection
- **Real-time Logging**: Comprehensive structured JSON logging with geolocation data
- **Log Viewer**: Web-based interface for viewing and filtering application logs
- **Modern Frontend**: HTMX-powered dynamic UI with Tailwind CSS styling
- **Database Integration**: PostgreSQL with connection pooling and automatic schema setup
- **Security**: Password hashing with salts, CSRF tokens, and secure session management
- **Health Checks**: Built-in health and readiness endpoints for monitoring
- **Development Tools**: Hot-reload development environment with auto-rebuilding assets

## Architecture

### Backend (Go)
- **Framework**: Standard library `net/http` with custom routing
- **Database**: PostgreSQL with `sqlx` for enhanced SQL operations
- **Templating**: Type-safe HTML templating with [Templ](https://templ.guide/)
- **Logging**: Structured JSON logging with `slog` to both stdout and file
- **Authentication**: Session-based with PostgreSQL storage

### Frontend
- **CSS Framework**: Tailwind CSS with custom configuration
- **JavaScript**: HTMX for dynamic interactions and Server-Sent Events
- **Typography**: Comic Code Ligatures font for a unique aesthetic
- **Build System**: Custom Makefile with asset watching and hot-reload

### Database Schema
```sql
-- Users table with authentication data
users (user_id, first_name, last_name, email, password_hash, salt, last_login, created_at, updated_at)

-- Sessions table for authentication state
sessions (session_id, user_id, expires_at, flash_message)
```

## Quick Start

### Prerequisites
- Go 1.24+
- PostgreSQL database
- Node.js (for Tailwind CSS CLI)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/pippaothy.git
   cd pippaothy
   ```

2. **Install development tools**
   ```bash
   make install-tools
   ```

3. **Set up environment variables**
   ```bash
   export DB_HOST=localhost
   export DB_PORT=5432
   export DB_USER=your_username
   export DB_PASSWORD=your_password
   export DB_NAME=pippaothy
   ```
   Or create a `.env` file with these variables.

4. **Start development environment**
   ```bash
   make dev
   ```

The application will be available at `http://localhost:8080` with hot-reload enabled.

## Development Commands

### Building and Running
- `make build` - Full build process (CSS, templates, Go binary)
- `make dev` - Start development with hot-reloading
- `go build -o ./bin/pippaothy ./cmd/main.go` - Build Go binary directly

### Asset Generation
- `make tailwind-build` - Build Tailwind CSS
- `make tailwind-watch` - Watch and rebuild CSS on changes
- `make templ-generate` - Generate Go code from templates
- `make templ-watch` - Watch and regenerate templates

### Tool Installation
- `make install-tools` - Install Tailwind CSS, Templ, and Air

## Project Structure

```
pippaothy/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── auth/                   # Authentication logic
│   │   └── session.go
│   ├── database/               # Database connection and setup
│   │   └── init.go
│   ├── ipinfo/                 # IP geolocation service
│   │   └── geolocation.go
│   ├── logs/                   # Log processing utilities
│   │   └── logs.go
│   ├── server/                 # HTTP server implementation
│   │   ├── server.go           # Server setup and lifecycle
│   │   ├── routes.go           # Route handlers
│   │   └── middleware.go       # Authentication and logging middleware
│   ├── templates/              # Templ template files
│   │   ├── base.templ          # Base layout and home page
│   │   ├── auth.templ          # Login and registration forms
│   │   └── simple-logs.templ   # Log viewing interface
│   └── users/                  # User management
│       └── user.go
├── schema/
│   └── model.sql               # PostgreSQL database schema
├── static/
│   ├── css/
│   │   ├── input.css           # Tailwind CSS source
│   │   └── output.css          # Generated CSS (build artifact)
│   ├── fonts/                  # Custom fonts
│   ├── images/                 # Static images
│   └── js/                     # JavaScript libraries (HTMX)
├── k3s/                        # Kubernetes deployment manifests
├── logs/                       # Application log files
├── Dockerfile                  # Container build configuration
├── Makefile                    # Build and development commands
├── tailwind.config.js          # Tailwind CSS configuration
└── CLAUDE.md                   # AI assistant instructions
```

## Key Features Deep Dive

### Authentication System
- **Registration**: Email validation, password strength requirements, duplicate prevention
- **Login**: Secure credential verification with session creation
- **Session Management**: Database-stored sessions with automatic expiration
- **CSRF Protection**: Token-based protection for all state-changing operations
- **Flash Messages**: User feedback system integrated with sessions

### Logging & Monitoring
- **Structured Logging**: JSON format with standardized fields
- **Request Tracing**: Unique request IDs for correlation
- **IP Geolocation**: Automatic geographic data collection for requests
- **Performance Metrics**: Response times, status codes, and request sizes
- **Log Viewer**: Web interface with filtering and pagination
- **Health Endpoints**: `/health` and `/ready` for monitoring systems

### Security Features
- **Password Security**: Salted hashing with crypto/rand
- **Session Security**: Secure token generation and storage
- **CSRF Protection**: Mandatory for all POST requests
- **Input Validation**: Server-side validation with sanitization
- **SQL Injection Prevention**: Parameterized queries throughout

## Configuration

### Environment Variables
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - Database connection
- `DB_MAX_OPEN_CONNS` - Maximum database connections (default: 25)
- `DB_MAX_IDLE_CONNS` - Maximum idle connections (default: 5)
- `DB_CONN_MAX_LIFETIME` - Connection lifetime (default: 5m)
- `DB_CONN_MAX_IDLE_TIME` - Connection idle time (default: 1m)

### Logging Configuration
- Logs are written to both stdout and `./logs/app.log`
- JSON format with structured fields
- Debug level logging in development
- Automatic log directory creation

## Deployment

### Docker
```bash
docker build -t pippaothy .
docker run -p 8080:8080 --env-file .env pippaothy
```

### Kubernetes
Deployment manifests are provided in the `k3s/` directory:
- Service configuration with MetalLB load balancer
- Ingress with Let's Encrypt SSL
- Certificate management
- Resource limits and health checks

## Development Workflow

1. **Make changes** to Go code, templates, or CSS
2. **Templates auto-regenerate** via templ-watch
3. **CSS rebuilds** automatically via tailwind-watch  
4. **Go server restarts** automatically via Air
5. **Browser refreshes** to see changes

## API Endpoints

### Public Routes
- `GET /` - Home page (redirects to login if not authenticated)
- `GET /login` - Login form
- `POST /login` - Process login
- `GET /register` - Registration form
- `POST /register` - Process registration
- `GET /health` - Health check
- `GET /ready` - Readiness check
- `GET /static/*` - Static assets

### Protected Routes (require authentication)
- `POST /logout` - Process logout
- `GET /logs` - Log viewer interface

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes following the existing code style
4. Ensure all tests pass and the application builds
5. Submit a pull request

## Code Quality Notes

### Strengths
- **Clean Architecture**: Well-organized package structure with clear separation of concerns
- **Security-First**: Comprehensive security measures throughout the application
- **Type Safety**: Leverages Go's type system and Templ for compile-time template checking
- **Observability**: Excellent logging and monitoring capabilities
- **Modern Tooling**: Contemporary development workflow with hot-reload and asset watching
- **Production Ready**: Health checks, graceful shutdown, and container support

### Technical Excellence
- **Error Handling**: Proper error wrapping and contextual logging
- **Database Management**: Connection pooling and prepared statements
- **Middleware Pattern**: Clean request/response pipeline with composable middleware
- **Context Usage**: Proper context propagation for request tracing and cancellation
- **Resource Management**: Proper cleanup with defer statements and graceful shutdown

## License

This project is licensed under the MIT License - see the LICENSE file for details.