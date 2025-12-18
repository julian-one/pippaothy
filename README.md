# Pippaothy

A Go REST API with JWT authentication and Redis-backed token management.

## Auth Flow

```
┌─────────┐                    ┌─────────┐                    ┌───────┐
│  Client │                    │   API   │                    │ Redis │
└────┬────┘                    └────┬────┘                    └───┬───┘
     │                              │                             │
     │  POST /register or /login    │                             │
     │─────────────────────────────>│                             │
     │                              │                             │
     │                              │  Store refresh token        │
     │                              │────────────────────────────>│
     │                              │                             │
     │  { access_token (5min),      │                             │
     │    refresh_token (24hr) }    │                             │
     │<─────────────────────────────│                             │
     │                              │                             │
     │  GET /me                     │                             │
     │  Authorization: Bearer <at>  │                             │
     │─────────────────────────────>│                             │
     │                              │  Check blacklist            │
     │                              │────────────────────────────>│
     │                              │<────────────────────────────│
     │  { user_id, email, ... }     │                             │
     │<─────────────────────────────│                             │
     │                              │                             │
     │  POST /refresh               │                             │
     │  { refresh_token }           │                             │
     │─────────────────────────────>│                             │
     │                              │  Validate & rotate token    │
     │                              │────────────────────────────>│
     │  { new access_token,         │                             │
     │    new refresh_token }       │                             │
     │<─────────────────────────────│                             │
     │                              │                             │
     │  POST /logout                │                             │
     │  Authorization: Bearer <at>  │                             │
     │─────────────────────────────>│                             │
     │                              │  Blacklist AT, delete RTs   │
     │                              │────────────────────────────>│
     │  204 No Content              │                             │
     │<─────────────────────────────│                             │
```

## API Reference

### Register

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "email": "alice@example.com", "password": "secret123"}'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "550e8400-e29b-41d4-a716-446655440000",
  "token_type": "Bearer",
  "expires_in": 300,
  "user": {
    "user_id": 1,
    "username": "alice",
    "email": "alice@example.com",
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T10:00:00Z"
  }
}
```

### Login

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email": "alice@example.com", "password": "secret123"}'
```

### Refresh Token

```bash
curl -X POST http://localhost:8080/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "550e8400-e29b-41d4-a716-446655440000"}'
```

### Get Current User

```bash
curl http://localhost:8080/me \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

Response:
```json
{
  "user_id": 1,
  "email": "alice@example.com",
  "username": "alice"
}
```

### Logout

```bash
curl -X POST http://localhost:8080/logout \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

Returns `204 No Content`. Blacklists the access token and invalidates all refresh tokens for the user.

## Running

```bash
go build -o pippaothy .
./pippaothy serve
```
