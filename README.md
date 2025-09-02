# GoSlack - Slack-like Backend Application

[![Tests](https://github.com/rahulmishra/goslack/actions/workflows/test.yml/badge.svg)](https://github.com/rahulmishra/goslack/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rahulmishra/goslack)](https://goreportcard.com/report/github.com/rahulmishra/goslack)

A Slack-like backend application built with Go, following clean architecture principles and inspired by the [simplebank](https://github.com/techschool/simplebank) project structure.

## Tech Stack

- **Language**: Go 1.24+
- **Web Framework**: Gin
- **Database**: PostgreSQL
- **Query Management**: SQLC
- **Testing**: Go's testing package + Gomock for mocks
- **Authentication**: PASETO tokens
- **Containerization**: Docker

## Project Structure

```
├── api/              # HTTP handlers and routes
├── db/
│   ├── migration/    # SQL migration files
│   ├── query/        # SQL queries for sqlc
│   ├── sqlc/         # Generated Go code from sqlc
│   └── mock/         # Generated mocks for testing
├── service/          # Business logic layer
├── token/            # JWT/PASETO token management
├── util/             # Utility functions
├── main.go           # Application entry point
├── Makefile          # Build and development commands
└── sqlc.yaml         # SQLC configuration
```

## Features

### Phase 1: Organization & User Management ✅

- Organization creation and management
- User registration and authentication
- Password hashing with bcrypt
- PASETO token-based authentication
- Role-based access control foundations

## Getting Started

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- PostgreSQL
- SQLC
- golang-migrate

### Installation

1. **Clone the repository**

   ```bash
   git clone <repository-url>
   cd goslack
   ```

2. **Install dependencies**

   ```bash
   go mod tidy
   ```

3. **Start the infrastructure**

   ```bash
   # Create Docker network
   make network

   # Start PostgreSQL
   make postgres

   # Create database
   make createdb
   ```

4. **Run database migrations**

   ```bash
   make migrateup
   ```

5. **Generate code**

   ```bash
   # Generate SQLC code
   make sqlc

   # Generate mocks for testing
   make mock
   ```

6. **Start the server**
   ```bash
   make server
   ```

The server will start on `localhost:8080`.

## API Endpoints

### Public Endpoints

- `POST /organizations` - Create a new organization
- `GET /organizations` - List organizations
- `GET /organizations/:id` - Get organization by ID
- `POST /users` - Create a new user
- `POST /users/login` - User login

### Protected Endpoints (Requires Authentication)

- `GET /users/:id` - Get user by ID
- `GET /users` - List users in organization
- `PUT /users/:id/profile` - Update user profile
- `PUT /users/:id/password` - Change user password
- `PUT /organizations/:id` - Update organization
- `DELETE /organizations/:id` - Delete organization

### Authentication

Include the authorization header in your requests:

```
Authorization: Bearer <your-token>
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run database-specific tests only  
make dbtest

# Run tests with coverage
go test -v -cover ./...
```

### Database Operations

```bash
# Create a new migration
make new_migration name=add_new_table

# Production database
make migrateup      # Migrate up
make migratedown    # Migrate down
make migrateup1     # Migrate up one version
make migratedown1   # Migrate down one version

# The tests now run against the same database as development
# No separate test database needed - tests use transactions for isolation
```

### Code Generation

```bash
# Generate SQLC code after modifying queries
make sqlc

# Generate mocks after interface changes
make mock
```

## Configuration

Copy `app.env` and modify as needed:

```env
DB_DRIVER=postgres
DB_SOURCE=postgresql://root:secret@localhost:5432/goslack?sslmode=disable
HTTP_SERVER_ADDRESS=0.0.0.0:8080
TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012
ACCESS_TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=24h
```

## Example Usage

### 1. Create an Organization

```bash
curl -X POST http://localhost:8080/organizations \
  -H "Content-Type: application/json" \
  -d '{"name": "Tech Corp"}'
```

### 2. Create a User

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{
    "organization_id": 1,
    "email": "john@techcorp.com",
    "first_name": "John",
    "last_name": "Doe",
    "password": "secret123"
  }'
```

### 3. Login

```bash
curl -X POST http://localhost:8080/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@techcorp.com",
    "password": "secret123"
  }'
```

### 4. Get User (with authentication)

```bash
curl -X GET http://localhost:8080/users/1 \
  -H "Authorization: Bearer <your-token>"
```

## Testing

The project includes comprehensive unit tests with mocked dependencies:

- **Password utility tests**: Test password hashing and verification
- **Token tests**: Test PASETO token creation and verification
- **API tests**: Test HTTP handlers with mocked database layer
- **Database tests**: Test SQLC-generated queries against real PostgreSQL database

### Test Types

1. **Unit Tests** (with mocks):

   ```bash
   make test
   ```

2. **Database Integration Tests** (with real PostgreSQL):

   ```bash
   # First time setup
   make createtestdb
   make migratetestup

   # Run database tests
   make dbtest
   ```

### Test Coverage

- API Layer: 32% coverage
- Database Layer: 74.7% coverage  
- Token Layer: 84% coverage
- Utility Layer: 36.7% coverage

### Continuous Integration

The project uses GitHub Actions for automated testing:

- **Triggers**: Push to `main` branch and pull requests
- **Environment**: Ubuntu latest with PostgreSQL 17
- **Steps**: Code generation, migrations, mock generation, and full test suite
- **Status**: ![Tests](https://github.com/rahulmishra/goslack/actions/workflows/test.yml/badge.svg)

## Architecture

This project follows clean architecture principles:

1. **API Layer** (`api/`): HTTP handlers, request/response handling
2. **Service Layer** (`service/`): Business logic, validation
3. **Data Layer** (`db/`): Database queries, models
4. **Utility Layer** (`util/`): Shared utilities, configuration

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`make test`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Roadmap

- [x] **Phase 1**: Organization & User Management
- [ ] **Phase 2**: Workspaces, Channels & RBAC
- [ ] **Phase 3**: Direct & Channel Messaging
- [ ] **Phase 4**: File Upload and Management
- [ ] **Phase 5**: Real-Time WebSocket Communication
