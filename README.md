# Pippaothy

A modern, full-featured recipe management and web scraping application built with Go. Features comprehensive user authentication, recipe database management, web scraping capabilities, and real-time log monitoring.

## Features

### Core Application
- **Recipe Management**: Create, edit, delete, and organize personal recipes with ingredients and instructions
- **Recipe Discovery**: Browse public recipes shared by other users
- **Recipe Import**: Advanced web scraping system to import recipes from external sources
- **Search & Filtering**: Find recipes by title, tags, cuisine, and difficulty level
- **User Profiles**: Personal recipe collections with privacy controls

### Authentication & Security
- **Session-based Authentication**: Secure user registration and login with CSRF protection
- **Password Security**: Salted password hashing with strength requirements
- **Session Management**: Database-stored sessions with automatic expiration and flash messages
- **Input Validation**: Comprehensive server-side validation and sanitization

### Technical Features
- **Real-time Logging**: Structured JSON logging with request tracing and geolocation
- **Log Viewer**: Web-based interface for viewing and filtering application logs
- **Modern Frontend**: HTMX-powered dynamic UI with Tailwind CSS and custom typography
- **Database Integration**: PostgreSQL with optimized connection pooling and migrations
- **Health Monitoring**: Built-in health and readiness endpoints
- **Development Tools**: Hot-reload environment with asset watching and live rebuilding

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

-- Recipes table with detailed recipe information
recipes (recipe_id, user_id, source_name, source_url, title, description, prep_time, 
         cook_time, servings, difficulty, cuisine, tags, image_url, is_public, 
         created_at, updated_at)

-- Recipe ingredients with order and parsing
recipe_ingredients (ingredient_id, recipe_id, ingredient_text, order_index, 
                   amount, unit, item, notes, created_at)

-- Recipe instructions with order and timing
recipe_instructions (instruction_id, recipe_id, instruction_text, order_index, 
                    estimated_time, created_at)
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

### Command Line Interface

Pippaothy includes a powerful CLI with multiple commands:

- **Server**: `./pippaothy serve` - Start the web server
- **Recipe Scraping**: `./pippaothy scraper [command]` - Recipe import utilities
  - `list` - Scrape recipe list from all pages
  - `list-page <page>` - Scrape specific page
  - `list-range <start> <end>` - Scrape page range
  - `detail <url>` - Scrape specific recipe URL
  - `import` - Import all scraped recipes to database
  - `import-range <start> <end>` - Import specific page range
  - `import-user <user-id>` - Import recipes for specific user
  - `import-all` - Import all categories from recipe index
  - `list-categories` - List available recipe categories

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
│   ├── main.go                 # Application entry point
│   ├── root.go                 # CLI root command setup
│   ├── serve.go                # Web server command
│   ├── scraper.go              # Recipe scraping commands
│   └── scraper/
│       └── main.go             # Scraper implementation
├── internal/
│   ├── auth/                   # Authentication and security
│   │   └── session.go          # Sessions, CSRF, cookies
│   ├── database/               # Database management
│   │   └── init.go             # Connection, pooling, migrations
│   ├── logs/                   # Logging utilities
│   │   └── logs.go
│   ├── recipes/                # Recipe business logic
│   │   └── recipes.go          # CRUD operations, validation
│   ├── server/                 # HTTP server implementation
│   │   ├── server.go           # Server setup and lifecycle
│   │   ├── middleware.go       # Auth, CSRF, logging middleware
│   │   ├── routes.go           # Route definitions (legacy)
│   │   └── recipes_handlers.go # Recipe HTTP handlers
│   ├── templates/              # Type-safe HTML templates
│   │   ├── base.templ          # Layout and navigation
│   │   ├── auth.templ          # Login/registration forms
│   │   ├── recipes.templ       # Recipe management UI
│   │   ├── simple-logs.templ   # Log viewing interface
│   │   └── *_templ.go          # Generated template code
│   └── users/                  # User management
│       └── user.go             # User CRUD, validation, auth
├── schema/
│   └── model.sql               # Complete PostgreSQL schema
├── static/
│   ├── css/
│   │   ├── input.css           # Tailwind source with custom styles
│   │   └── output.css          # Generated CSS (build artifact)
│   ├── fonts/                  # Comic Code Ligatures font
│   ├── images/                 # Static assets
│   └── js/                     # HTMX and extensions
├── k3s/                        # Kubernetes deployment configs
│   ├── pippaothy.yaml          # Deployment and service
│   ├── ingress.yaml            # Load balancer and SSL
│   ├── certificate.yaml        # Let's Encrypt certificate
│   ├── letsencrypt-issuer.yaml # Certificate issuer
│   └── metallb.yaml            # MetalLB configuration
├── logs/                       # Application log output
├── bin/                        # Built binaries
├── tmp/                        # Build artifacts and temp files
├── Dockerfile                  # Multi-stage container build
├── Makefile                    # Build, dev, and asset commands
├── tailwind.config.js          # Tailwind configuration
├── test_idempotent.sh          # Scraper testing script
├── deploy.sh                   # Deployment automation
└── CLAUDE.md                   # AI development assistant config
```

## Key Features Deep Dive

### Recipe Management System
- **Recipe Creation**: Rich recipe editor with ingredients, instructions, and metadata
- **Recipe Organization**: Categorization by cuisine, difficulty, prep/cook time, and custom tags
- **Privacy Controls**: Public/private recipe visibility settings
- **Recipe Validation**: Server-side validation for all recipe data
- **Bulk Operations**: Efficient loading and management of recipe collections
- **Source Attribution**: Support for both user-created and scraped recipes with source tracking

### Web Scraping Engine
- **Multi-source Support**: Designed to scrape recipes from various websites
- **Idempotent Operations**: Prevents duplicate imports with URL-based deduplication
- **Batch Processing**: Efficient import of large recipe collections
- **Category Support**: Automatic categorization and tagging of imported recipes
- **Error Handling**: Robust error recovery and logging for failed scrapes
- **Testing Framework**: Built-in testing utilities for scraper validation

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
- `GET /health` - Health check endpoint
- `GET /ready` - Readiness check endpoint
- `GET /static/*` - Static assets (CSS, JS, images, fonts)
- `GET /recipes/public` - Browse public recipes (with optional authentication)
- `GET /recipes/search` - Search public recipes (with optional authentication)
- `GET /recipes/{id}` - View recipe details (public recipes or own recipes)

### Protected Routes (require authentication)
- `POST /logout` - Process logout and clear session
- `GET /logs` - Log viewer interface with filtering
- `GET /recipes` - User's personal recipe collection
- `GET /recipes/new` - Create new recipe form
- `POST /recipes` - Submit new recipe
- `GET /recipes/{id}/edit` - Edit recipe form (own recipes only)
- `PUT /recipes/{id}` - Update recipe (own recipes only)
- `DELETE /recipes/{id}` - Delete recipe (own recipes only)

### API Design Notes
- **CSRF Protection**: All POST/PUT/DELETE requests require CSRF tokens
- **Content Types**: Forms use `application/x-www-form-urlencoded`, API responses are HTML (HTMX)
- **Error Handling**: Errors returned as HTML fragments for HTMX integration
- **Authentication**: Session-based with automatic redirects for unauthorized access

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes following the existing code style
4. Ensure all tests pass and the application builds
5. Submit a pull request

## Code Quality Analysis

### Architecture Strengths
- **Clean Architecture**: Well-organized package structure following Go conventions
- **Domain Separation**: Clear boundaries between auth, recipes, users, and server concerns
- **Type Safety**: Leverages Go's type system and Templ for compile-time template checking
- **Security-First Design**: Comprehensive security measures integrated throughout
- **Observability**: Excellent structured logging with request tracing and context
- **Modern Tooling**: Contemporary development workflow with hot-reload and asset watching

### Technical Excellence
- **Error Handling**: Consistent error wrapping with contextual information
- **Database Design**: Proper normalization, constraints, and relationship management
- **Connection Pooling**: Optimized database connection management with configurable limits
- **Middleware Architecture**: Clean, composable request/response pipeline
- **Context Propagation**: Proper context usage for request tracing and cancellation
- **Resource Management**: Careful cleanup with defer statements and graceful shutdown
- **Transaction Management**: Proper database transactions with rollback handling

### Security Implementation
- **Authentication**: Secure session-based authentication with proper token generation
- **Password Security**: Salted hashing using scrypt with secure random salt generation
- **CSRF Protection**: Comprehensive CSRF token validation for state-changing operations
- **Input Validation**: Server-side validation and sanitization for all user inputs
- **SQL Injection Prevention**: Consistent use of parameterized queries throughout
- **Session Security**: Secure cookie configuration with HttpOnly and SameSite attributes

### Development Quality
- **CLI Design**: Well-structured command hierarchy using Cobra framework
- **Build System**: Sophisticated Makefile with tool management and asset pipeline
- **Container Support**: Multi-stage Docker build with security best practices
- **Deployment Ready**: Kubernetes manifests with SSL, load balancing, and health checks
- **Development Experience**: Hot-reload environment with automatic asset rebuilding

### Areas for Improvement
- **Test Coverage**: No test files found - comprehensive testing needed
- **Error Consistency**: Mixed use of `errors.Join()` and `fmt.Errorf()` patterns
- **Scraper Implementation**: CLI commands exist but core scraping logic needs completion
- **Frontend Validation**: Client-side validation missing for better user experience
- **API Documentation**: Could benefit from OpenAPI/Swagger documentation

## License

This project is licensed under the MIT License - see the LICENSE file for details.