# URL Shortener

A production-style URL shortener backend built with Go, Gin, PostgreSQL, and pgxpool.

## Project Structure

```text
.
├── backend
│   ├── app
│   │   ├── controllers
│   │   ├── models
│   │   └── queries
│   │
│   ├── pkg
│   │   ├── configs
│   │   ├── middleware
│   │   ├── routes
│   │   └── utils
│   │
│   ├── platform
│   │   ├── database
│   │   └── migrations
│   │
│   ├── .env
│   ├── go.mod
│   └── main.go
│
├── frontend
│
└── README.md
```

---

# Features

* URL shortening
* Custom short codes
* Redirect support
* Expiry support
* PostgreSQL persistence
* pgx connection pooling
* Graceful shutdown
* Layered architecture
* Request validation

---

# Tech Stack

## Backend

* Go
* Gin
* PostgreSQL
* pgxpool
* go-playground/validator

## Frontend

* (Add frontend stack here)

---

# API Endpoints

## Create Short URL

```http
POST /api/v1/shorten
```

### Request Body

```json
{
  "long_url": "https://google.com",
  "custom_code": "google123"
}
```

### Response

```json
{
  "short_url": "google123"
}
```

---

## Redirect URL

```http
GET /api/v1/:shortCode
```

Example:

```http
GET /api/v1/google123
```

Redirects to:

```text
https://google.com
```

---

# Environment Variables

Create `.env` inside `backend/`

```env
PORT=8080

DB_SERVER_URL=postgres://urluser:password@localhost:5432/url_shortener

DB_MAX_CONNECTIONS=20
DB_MIN_CONNECTIONS=5
DB_MAX_IDLE_TIME=10
DB_MAX_LIFETIME=1
```

---

# Running Backend

## Install dependencies

```bash
go mod tidy
```

## Run server

```bash
go run .
```

---

# PostgreSQL Setup

Create database:

```sql
CREATE DATABASE url_shortener;
```

Create table:

```sql
CREATE TABLE urls (
    id BIGSERIAL PRIMARY KEY,
    long_url TEXT NOT NULL,
    short_url TEXT UNIQUE NOT NULL,
    clicks BIGINT DEFAULT 0,
    expiry TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

# Example cURL

## Create URL

```bash
curl -X POST http://localhost:8080/api/v1/shorten \
-H "Content-Type: application/json" \
-d '{
  "long_url":"https://github.com",
  "custom_code":"gh2026"
}'
```

## Redirect

```bash
curl -I http://localhost:8080/api/v1/gh2026
```

---

# Future Improvements

* Authentication
* Analytics dashboard
* Rate limiting
* Redis caching
* QR code generation
* Docker support
* Swagger/OpenAPI docs

---

# License

MIT
