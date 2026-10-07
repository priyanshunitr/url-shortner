# AI Agent Instructions for URL Shortener Backend

Welcome, AI Agent! This file contains the architectural decisions, constraints, and context for this Go backend project. Please read these instructions carefully before making any code modifications.

## 1. Architecture Overview
This project strictly follows a **Layered Architecture** (Horizontal Slicing).
* **Controllers (`app/controllers`)**: Strictly handles HTTP transport logic. Responsible for JSON parsing, reading cookies, formatting responses, and invoking services. **NO BUSINESS LOGIC HERE.**
* **Services (`app/services`)**: Contains all core business logic, orchestrates database transactions, hashes passwords, generates tokens, and enforces constraints (e.g., maximum sessions).
* **Repositories (`app/repositories`)**: Exclusively handles database interaction (`SQL`). Methods should accept `context.Context` and often `pgx.Tx` for transactional integrity.
* **Models (`app/models`)**: Contains Domain models and API DTOs (Data Transfer Objects).
* **Routes (`pkg/routes`)**: Maps endpoints to controllers.

## 2. Authentication Architecture
* **Access Tokens**: Short-lived (15 min) JWTs containing minimal claims (`UserID`, `Role`, `SessionID`). Returned in the JSON response body.
* **Refresh Tokens**: Long-lived (7 days) secure random strings hashed via SHA-256 before storing in the database. Returned via an `HttpOnly`, `Secure`, `SameSite=Lax` Cookie.
* **Token Rotation**: We use **In-Place Rotation** for refresh tokens (`RotateRefreshTokenTx`). Instead of revoking and inserting, we update the existing row in `refresh_tokens`. This naturally maps 1 row to 1 active session.
* **Rate Limiting**: Configured in routes using the Redis-based rate limiter middleware.

## 3. Database Conventions
* **Driver**: We use `pgx/v5` (specifically `pgxpool`). We DO NOT use `database/sql` directly or an ORM like GORM.
* **Transactions**: Operations that span multiple queries MUST be wrapped in a transaction. The `Service` layer is responsible for beginning the transaction (`tx, err := s.Repo.DB.Begin(ctx)`), deferring the rollback, and committing. The `Repository` layer methods should end with the `Tx` suffix (e.g., `CreateUserTx`) and accept `tx pgx.Tx` as a parameter.

## 4. Coding Conventions & Best Practices
1. **Error Handling**: Bubble up errors from Repositories -> Services -> Controllers. Controllers should map specific errors to appropriate HTTP status codes (e.g., `401 Unauthorized`, `500 Internal Server Error`).
2. **Swagger Docs**: We use `swaggo/swag`. Whenever adding or modifying an API endpoint in `app/controllers`, update the `// @Summary`, `// @Param`, and `// @Success` annotations above the controller method.
3. **Building**: Always run `go build` after making modifications to ensure no syntax or typing errors were introduced.

## 5. Active Task List / Future Improvements
* Ensure any new controllers strictly adhere to the Service Pattern.
* Implement comprehensive Unit Testing for the `AuthService`.
* Complete the frontend integration and deploy on AWS.
