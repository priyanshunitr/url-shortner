import os

base_dir = "/home/dipankar/Coding/pjts/url-shortener/backend/fooddash"

directories = [
    "cmd/server",
    "internal/auth", "internal/users", "internal/feed", "internal/reels",
    "internal/foods", "internal/cart", "internal/orders", "internal/restaurants",
    "internal/notifications", "internal/websocket",
    "pkg/config", "pkg/database", "pkg/middleware", "pkg/jwt", "pkg/logger",
    "pkg/response", "pkg/validator", "pkg/storage", "pkg/websocket",
    "migrations", "docs/swagger", "scripts", "deployments"
]

files = {
    "go.mod": """module fooddash

go 1.25.0

require (
\tgithub.com/gin-gonic/gin v1.9.1
\tgithub.com/golang-jwt/jwt/v5 v5.2.0
\tgithub.com/jackc/pgx/v5 v5.5.5
\tgithub.com/redis/go-redis/v9 v9.5.1
\tgithub.com/joho/godotenv v1.5.1
\tgo.uber.org/zap v1.27.0
)
""",
    "Makefile": """run:
\tgo run cmd/server/main.go

air:
\tair -c .air.toml

migrate:
\t./scripts/migrate.sh

swagger:
\tswag init -g cmd/server/main.go -o docs/swagger
""",
    ".env.example": """PORT=8080
GIN_MODE=debug
LOG_LEVEL=debug
DATABASE_URL=postgres://user:pass@localhost:5432/fooddash?sslmode=disable
REDIS_URL=redis://localhost:6379/0
JWT_SECRET=supersecret
""",
    ".air.toml": """root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/main ./cmd/server/main.go"
bin = "./tmp/main"
include_ext = ["go", "tpl", "tmpl", "html"]
exclude_dir = ["assets", "tmp", "vendor", "docs"]

[color]
main = "magenta"
watcher = "cyan"
build = "yellow"
runner = "green"
""",
    "README.md": """# FoodDash Backend

Modular Monolith Go backend for FoodDash. Built for scalability with Gin, PostgreSQL, Redis, and a clear separation of concerns (Service & Repository pattern) to allow easy future extraction into microservices.
""",
    "deployments/Dockerfile": """FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
COPY .env.example .env
EXPOSE 8080
CMD ["./server"]
""",
    "deployments/docker-compose.yml": """version: '3.8'
services:
  app:
    build:
      context: ..
      dockerfile: deployments/Dockerfile
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
    environment:
      - DATABASE_URL=postgres://postgres:postgres@postgres:5432/fooddash?sslmode=disable
      - REDIS_URL=redis://redis:6379/0

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
      - POSTGRES_DB=fooddash
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  pgdata:
""",
    "cmd/server/main.go": """package main

import (
\t"context"
\t"fmt"
\t"log"
\t"net/http"
\t"os"
\t"os/signal"
\t"syscall"
\t"time"

\t"fooddash/pkg/config"
\t"fooddash/pkg/database"
\t"fooddash/pkg/logger"
\t"fooddash/pkg/middleware"
\t"fooddash/pkg/response"

\t"fooddash/internal/auth"
\t"fooddash/internal/cart"
\t"fooddash/internal/feed"
\t"fooddash/internal/foods"
\t"fooddash/internal/notifications"
\t"fooddash/internal/orders"
\t"fooddash/internal/reels"
\t"fooddash/internal/restaurants"
\t"fooddash/internal/users"

\t"github.com/gin-gonic/gin"
)

// @title FoodDash API
// @version 1.0
// @description Production-grade Go backend for FoodDash
func main() {
\t// 1. Load Configuration
\tcfg := config.Load()
\tlogr := logger.New(cfg.LogLevel)

\t// 2. Connect to Database
\tdb, err := database.NewPostgres(cfg.DatabaseURL)
\tif err != nil {
\t\tlog.Fatalf("DB error: %v", err)
\t}

\t// 3. Connect to Redis
\tredisClient, err := database.NewRedis(cfg.RedisURL)
\tif err != nil {
\t\tlog.Fatalf("Redis error: %v", err)
\t}

\t// 4. Initialize Router
\tgin.SetMode(cfg.GinMode)
\trouter := gin.New()

\t// 5. Global Middlewares
\trouter.Use(middleware.Logger(logr))
\trouter.Use(middleware.Recovery(logr))
\trouter.Use(middleware.CORS())

\t// Health Check
\trouter.GET("/health", func(c *gin.Context) {
\t\tresponse.Success(c, http.StatusOK, "Healthy", nil)
\t})

\t// 6. API Versioning
\tapi := router.Group("/api/v1")

\t// 7. Dependency Injection & Route Registration (Modular Monolith Setup)
\tauth.RegisterRoutes(api, db, redisClient, logr)
\tusers.RegisterRoutes(api, db, logr)
\tfeed.RegisterRoutes(api, db, logr)
\tfoods.RegisterRoutes(api, db, logr)
\tnotifications.RegisterRoutes(api, db, logr)
\torders.RegisterRoutes(api, db, logr)
\treels.RegisterRoutes(api, db, logr)
\trestaurants.RegisterRoutes(api, db, logr)
\tcart.RegisterRoutes(api, db, redisClient, logr)

\t// 8. Server Setup with Graceful Shutdown
\tsrv := &http.Server{
\t\tAddr:    fmt.Sprintf(":%s", cfg.Port),
\t\tHandler: router,
\t}

\tgo func() {
\t\tif err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
\t\t\tlog.Fatalf("listen: %s\\n", err)
\t\t}
\t}()

\t// Wait for interrupt signal to gracefully shutdown the server
\tquit := make(chan os.Signal, 1)
\tsignal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
\t<-quit

\tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
\tdefer cancel()
\tif err := srv.Shutdown(ctx); err != nil {
\t\tlog.Fatal("Server Shutdown:", err)
\t}
}
""",
    "pkg/config/config.go": """package config

import (
\t"os"

\t"github.com/joho/godotenv"
)

// Config holds all environment variables
type Config struct {
\tPort        string
\tGinMode     string
\tLogLevel    string
\tDatabaseURL string
\tRedisURL    string
\tJWTSecret   string
}

// Load loads environment variables from .env file or system env
func Load() *Config {
\t_ = godotenv.Load() // Ignore error, fallback to env vars if .env doesn't exist
\treturn &Config{
\t\tPort:        getEnv("PORT", "8080"),
\t\tGinMode:     getEnv("GIN_MODE", "debug"),
\t\tLogLevel:    getEnv("LOG_LEVEL", "info"),
\t\tDatabaseURL: getEnv("DATABASE_URL", ""),
\t\tRedisURL:    getEnv("REDIS_URL", ""),
\t\tJWTSecret:   getEnv("JWT_SECRET", "secret"),
\t}
}

func getEnv(key, fallback string) string {
\tif value, ok := os.LookupEnv(key); ok {
\t\treturn value
\t}
\treturn fallback
}
""",
    "pkg/database/postgres.go": """package database

// Postgres represents the database connection
type Postgres struct{}

// NewPostgres creates a new PostgreSQL connection pool
func NewPostgres(url string) (*Postgres, error) {
\t// Placeholder for pgxpool connection
\treturn &Postgres{}, nil
}
""",
    "pkg/database/redis.go": """package database

// Redis represents the cache connection
type Redis struct{}

// NewRedis creates a new Redis client connection
func NewRedis(url string) (*Redis, error) {
\t// Placeholder for redis client setup
\treturn &Redis{}, nil
}
""",
    "pkg/logger/logger.go": """package logger

import "log"

// Logger interface for Dependency Injection
type Logger interface {
\tInfo(msg string, fields ...any)
\tError(msg string, fields ...any)
}

type zapLogger struct{}

// New initializes the structured logger
func New(level string) Logger {
\t// Placeholder for zap logger initialization
\treturn &zapLogger{}
}

func (l *zapLogger) Info(msg string, fields ...any) { log.Println("INFO:", msg) }
func (l *zapLogger) Error(msg string, fields ...any) { log.Println("ERROR:", msg) }
""",
    "pkg/response/response.go": """package response

import "github.com/gin-gonic/gin"

// Success sends a standard successful API response
func Success(c *gin.Context, code int, message string, data any) {
\tc.JSON(code, gin.H{
\t\t"status":  "success",
\t\t"message": message,
\t\t"data":    data,
\t})
}

// Error sends a standard error API response
func Error(c *gin.Context, code int, message string) {
\tc.JSON(code, gin.H{
\t\t"status":  "error",
\t\t"message": message,
\t})
}
""",
    "pkg/middleware/auth.go": """package middleware

import "github.com/gin-gonic/gin"

// Auth middleware for JWT validation
func Auth() gin.HandlerFunc {
\treturn func(c *gin.Context) {
\t\t// TODO: validate JWT token from headers
\t\tc.Next()
\t}
}
""",
    "pkg/middleware/logger.go": """package middleware

import (
\t"fooddash/pkg/logger"

\t"github.com/gin-gonic/gin"
)

// Logger middleware for logging HTTP requests
func Logger(log logger.Logger) gin.HandlerFunc {
\treturn func(c *gin.Context) {
\t\t// TODO: log request details (method, path, latency)
\t\tc.Next()
\t}
}
""",
    "pkg/middleware/cors.go": """package middleware

import "github.com/gin-gonic/gin"

// CORS middleware
func CORS() gin.HandlerFunc {
\treturn func(c *gin.Context) {
\t\t// TODO: configure strict CORS policies
\t\tc.Next()
\t}
}
""",
    "pkg/middleware/recovery.go": """package middleware

import (
\t"fooddash/pkg/logger"

\t"github.com/gin-gonic/gin"
)

// Recovery middleware to gracefully recover from panics
func Recovery(log logger.Logger) gin.HandlerFunc {
\treturn gin.Recovery()
}
""",
    "pkg/jwt/jwt.go": """package jwt

// Manager handles JWT token creation and parsing
type Manager struct{}

// NewManager initializes the JWT manager
func NewManager(secret string) *Manager { return &Manager{} }
""",
    "pkg/storage/minio.go": """package storage

// MinIO handles object storage (images, videos, reels)
type MinIO struct{}
""",
    "pkg/validator/validator.go": """package validator

// Validator handles request payload validation
type Validator struct{}
""",
    "pkg/websocket/manager.go": """package websocket

// Manager handles global websocket connections
type Manager struct{}
""",
    "scripts/migrate.sh": """#!/bin/bash
echo "Running database migrations..."
# Placeholder for goose or golang-migrate commands
""",
    "scripts/seed.sh": """#!/bin/bash
echo "Seeding database with initial data..."
"""
}

modules = ["users", "feed", "reels", "foods", "orders", "restaurants", "notifications"]

for mod in modules:
    files[f"internal/{mod}/routes.go"] = f"""package {mod}

import (
\t"fooddash/pkg/database"
\t"fooddash/pkg/logger"

\t"github.com/gin-gonic/gin"
)

// RegisterRoutes sets up the {mod} module routes and dependencies.
// Being an independent module allows for easy extraction to a microservice later.
func RegisterRoutes(router *gin.RouterGroup, db *database.Postgres, log logger.Logger) {{
\trepo := NewRepository(db)
\tsvc := NewService(repo, log)
\thandler := NewHandler(svc)

\tgroup := router.Group("/{mod}")
\t{{
\t\tgroup.GET("/", handler.List)
\t}}
}}
"""
    files[f"internal/{mod}/handler.go"] = f"""package {mod}

import (
\t"net/http"

\t"fooddash/pkg/response"

\t"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for {mod} (Delivery Layer)
type Handler struct {{
\tservice Service
}}

// NewHandler creates a new {mod} handler
func NewHandler(s Service) *Handler {{ return &Handler{{service: s}} }}

// List is a placeholder for getting items
func (h *Handler) List(c *gin.Context) {{
\tresponse.Success(c, http.StatusOK, "{mod} list fetched", nil)
}}
"""
    files[f"internal/{mod}/service.go"] = f"""package {mod}

import "fooddash/pkg/logger"

// Service defines the business logic for {mod} (Domain Layer)
type Service interface {{
\tDoSomething() error
}}

type serviceImpl struct {{
\trepo Repository
\tlog  logger.Logger
}}

// NewService creates a new {mod} service
func NewService(r Repository, l logger.Logger) Service {{
\treturn &serviceImpl{{repo: r, log: l}}
}}

func (s *serviceImpl) DoSomething() error {{ return nil }}
"""
    files[f"internal/{mod}/repository.go"] = f"""package {mod}

import "fooddash/pkg/database"

// Repository defines data access for {mod} (Persistence Layer)
type Repository interface {{
\tFindAll() error
}}

type repositoryImpl struct {{
\tdb *database.Postgres
}}

// NewRepository creates a new {mod} repository
func NewRepository(db *database.Postgres) Repository {{
\treturn &repositoryImpl{{db: db}}
}}

func (r *repositoryImpl) FindAll() error {{ return nil }}
"""
    files[f"internal/{mod}/model.go"] = f"""package {mod}

// Model represents the database entity for {mod}
type Model struct {{
\tID int `json:"id"`
}}
"""
    files[f"internal/{mod}/dto.go"] = f"""package {mod}

// CreateRequest represents the creation payload for {mod}
type CreateRequest struct {{}}
"""

# Auth requires Redis
files["internal/auth/routes.go"] = """package auth

import (
\t"fooddash/pkg/database"
\t"fooddash/pkg/logger"

\t"github.com/gin-gonic/gin"
)

// RegisterRoutes sets up the auth module routes
func RegisterRoutes(router *gin.RouterGroup, db *database.Postgres, redis *database.Redis, log logger.Logger) {
\trepo := NewRepository(db)
\tsvc := NewService(repo, log)
\thandler := NewHandler(svc)

\tgroup := router.Group("/auth")
\t{
\t\tgroup.POST("/login", handler.Login)
\t}
}
"""
files["internal/auth/handler.go"] = """package auth

import (
\t"net/http"

\t"fooddash/pkg/response"

\t"github.com/gin-gonic/gin"
)

type Handler struct { service Service }

func NewHandler(s Service) *Handler { return &Handler{service: s} }

func (h *Handler) Login(c *gin.Context) { response.Success(c, http.StatusOK, "login placeholder", nil) }
"""
files["internal/auth/service.go"] = files["internal/users/service.go"].replace("users", "auth")
files["internal/auth/repository.go"] = files["internal/users/repository.go"].replace("users", "auth")
files["internal/auth/model.go"] = files["internal/users/model.go"].replace("users", "auth")
files["internal/auth/dto.go"] = files["internal/users/dto.go"].replace("users", "auth")

# Cart requires Redis
files["internal/cart/routes.go"] = """package cart

import (
\t"fooddash/pkg/database"
\t"fooddash/pkg/logger"

\t"github.com/gin-gonic/gin"
)

// RegisterRoutes sets up the cart module routes
func RegisterRoutes(router *gin.RouterGroup, db *database.Postgres, redis *database.Redis, log logger.Logger) {
\trepo := NewRepository(db)
\tsvc := NewService(repo, log)
\thandler := NewHandler(svc)

\tgroup := router.Group("/cart")
\t{
\t\tgroup.GET("/", handler.List)
\t}
}
"""
files["internal/cart/handler.go"] = files["internal/users/handler.go"].replace("users", "cart")
files["internal/cart/service.go"] = files["internal/users/service.go"].replace("users", "cart")
files["internal/cart/repository.go"] = files["internal/users/repository.go"].replace("users", "cart")
files["internal/cart/model.go"] = files["internal/users/model.go"].replace("users", "cart")
files["internal/cart/dto.go"] = files["internal/users/dto.go"].replace("users", "cart")

# Websocket has different files
files["internal/websocket/hub.go"] = """package websocket

// Hub manages websocket clients and broadcasting
type Hub struct{}
"""
files["internal/websocket/client.go"] = """package websocket

// Client represents a single connected websocket
type Client struct{}
"""
files["internal/websocket/handler.go"] = """package websocket

// Handler handles HTTP to websocket upgrades
type Handler struct{}
"""

for d in directories:
    os.makedirs(os.path.join(base_dir, d), exist_ok=True)

for path, content in files.items():
    with open(os.path.join(base_dir, path), "w") as f:
        f.write(content)

for i, table in enumerate(["users", "restaurants", "food_items", "reels", "cart", "orders"]):
    with open(os.path.join(base_dir, f"migrations/{i+1:03d}_{table}.sql"), "w") as f:
        f.write(f"-- Migration for {table}\\n")

os.system(f"chmod +x {base_dir}/scripts/migrate.sh {base_dir}/scripts/seed.sh")

print("Scaffolding complete.")
