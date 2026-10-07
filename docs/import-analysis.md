# URL Shortener

> Archived analysis from before the backend blueprint implementation. The repository's root README describes the current architecture, commands and verified behavior; findings and paths below reflect the imported project at that time.

A Go/Gin API with PostgreSQL persistence, Redis caching and rate limiting, plus a Next.js dashboard branded SnapLink.

Imported from `D:\Downloads\url-shortener-main` in staged commits. All 122 source files were verified byte-for-byte immediately after import. The original root README is preserved in [docs/imported-readme.md](docs/imported-readme.md); it describes an earlier backend and should be treated as historical documentation.

## Layout and architecture

- `backend/app/models`: domain types and API responses.
- `backend/app/repositories`: pgx SQL queries and Redis operations.
- `backend/app/services`: URL policies, authentication transactions and analytics authorization.
- `backend/app/controllers`: HTTP handlers.
- `backend/pkg`: routing, middleware, validation, token utilities and server configuration.
- `backend/platform`: database clients and SQL migrations.
- `backend/docs`: checked-in Swagger output, currently behind the implemented routes.
- `frontend/src/app`: landing, login, registration, settings, dashboard, link creation/listing and per-link analytics pages.
- `frontend/src/components`: shared UI primitives, navigation, tables and charts.
- `frontend/src/services`, `hooks`, `types`: frontend API client, React Query hooks and UI-facing types.
- `.github/workflows/ci.yml`: imported backend validation workflow.

The backend uses Go 1.26.2 or later, Gin, pgx/v5, PostgreSQL, Redis, bcrypt, JWTs and rotating refresh tokens. The frontend uses Next.js 16.2.6, React 19, TypeScript, Tailwind CSS 4, Base UI, React Query and Recharts. The Go module retains its source name so internal imports remain consistent.

## Preview the frontend

The live API integration is unfinished in the downloaded project. Use its existing mock mode to preview the UI:

```powershell
cd frontend
pnpm install --frozen-lockfile
Copy-Item .env.example .env.local
pnpm dev
```

Open http://localhost:3000. Mock sign-in/registration returns a demo token. Mock create/delete operations do not persist changes to the mock link list, and mock short links are examples rather than usable redirects.

## Run the backend

Start PostgreSQL and Redis. The imported Compose file starts PostgreSQL only; its Redis service is commented out. From `backend`, copy `.env.test.example` to `.env.test` on a fresh checkout, then configure real local values. The downloaded `.env.test` has already been copied into this working directory and is ignored by Git.

`main.go` loads `.env.test` before `.env`; the first file takes precedence over the second, while process environment variables take precedence over both. Never commit private environment files.

Use these variable names; some names in the imported examples are outdated:

```dotenv
DB_SERVER_URL=postgres://urluser:yourpassword@127.0.0.1:5432/url_shortener?sslmode=disable
DB_MAX_CONNECTIONS=20
DB_MIN_CONNECTIONS=5
DB_MAX_IDLETIME_CONNECTION_TIME=10
DB_MAX_LIFETIME_CONNECTION_TIME=1
REDIS_URL=redis://127.0.0.1:6379/0
PORT=8080
GIN_MODE=debug
APP_ENV=debug
JWT_SECRET=replace-with-a-long-random-local-secret
```

Connection lifetime is measured in hours and idle time in minutes. For authenticated Redis, put credentials/database selection in `REDIS_URL`: the current client does not read `REDIS_PASSWORD` or `REDIS_DB` separately.

Apply the SQL migrations before starting the API, for example with the `migrate` CLI:

```powershell
cd backend
migrate -path platform/migrations -database 'postgres://urluser:yourpassword@127.0.0.1:5432/url_shortener?sslmode=disable' up
go run .
```

The server defaults to http://localhost:8080, and Swagger is at `/swagger/index.html`. The imported Dockerfile/Makefile need updates before container deployment; see the analysis below.

## Implemented API

All paths below are relative to `/api/v1`. Authenticated requests require `Authorization: Bearer <token>`.

| Area | Routes and behavior |
| --- | --- |
| Authentication | Public `POST /auth/register`, `/auth/login`, `/auth/refresh`; authenticated logout, current-user and session endpoints. |
| URLs | Authenticated `POST /shorten`, `GET /urls`, `PUT /urls/:id`, `DELETE /urls/:id`, `DELETE /urls/bulk`. Responses use backend DTOs and snake_case fields. |
| Redirect | Public `GET /:shortCode` returns an HTTP redirect. |
| Analytics | Authenticated `GET /analytics/:id/{overview,daily,recent,browser,device}`; requires base, premium or admin role and URL ownership. |
| Administration | Admin-only user listing and per-user URL listing. |

Custom aliases require premium/admin; free accounts are limited to 10 URLs and base accounts to 1,000. Created URLs currently expire after 24 hours regardless of the request's expiry field.

## Analysis of the imported project

These are findings from the imported source, not claims that integration or deployment has been completed. The migration preserves application behavior; a separate cleanup commit removes the frontend's unused imports and catch bindings reported by lint.

| Priority | Finding | Source and next step |
| --- | --- | --- |
| High | Frontend and backend API contracts differ. The frontend defaults to `/api`, calls `/links`, expects camelCase link arrays and a combined analytics response, and expects a `user` alongside login tokens. The backend serves `/api/v1`, `/urls`, `/shorten`, paginated snake_case DTOs, token-only login and separate per-link analytics endpoints. | `frontend/src/services/api.ts`, `hooks/useAuth.ts`, backend routes/controllers. Add an adapter and fetch `/auth/me`; a base-URL change alone is insufficient. |
| High | Redis cache hits return a URL with no database ID or expiry. Redirect tracking then uses ID 0, so cached clicks lose counters/events. Update/delete queries also do not invalidate the cached redirect. | `backend/app/repositories/url_query.go`. Cache the required metadata and invalidate cache entries on mutations; add regression coverage. |
| High | Authentication Redis features are silently skipped because `OpenDBConnection` supplies no `RDB` to `UserQuery`. Token blacklisting and the service's IP/email login limiter therefore do not run; route-level limits still exist. | `backend/platform/database/open_db_connection.go`, `app/repositories/user_queries.go`. Wire the Redis client into the user repository and verify logout/limits. |
| High | Revoking all other sessions asserts a string from `sessionID`, although middleware stores an int64. The handler can panic. Revocation also does not immediately invalidate issued access tokens because middleware does not check session state. | `backend/app/controllers/auth_controller.go`, `pkg/middleware/auth.go`. Align the context type and enforce the intended revocation semantics. |
| High | Refresh rotation checks the token before opening the transaction, then updates by session ID alone. Concurrent uses of one refresh token can both pass the check. | `backend/app/services/auth_service.go`, `app/repositories/user_queries.go`. Lock and validate the session within the transaction or condition the update on the previous hash/state. |
| Medium | Credentialed CORS uses wildcard origin, frontend fetches omit `credentials: 'include'`, and the refresh cookie is scoped to `/auth/refresh`, so logout cannot receive it at `/auth/logout`. | Backend CORS/auth controller and frontend API client. Configure an allowed frontend origin and align credential/cookie handling. |
| Medium | Startup logs the full database URL, JWT configuration is not validated, `.env.test` wins over `.env`, and pool example names differ from the code. | Backend database/JWT/startup files. Remove credential logging, validate required config and make environment selection explicit. |
| Medium | Container/local tooling is stale: Dockerfile starts from Go 1.21, copies a required `.env` into the image and exposes 5000; Compose omits Redis; Makefile Docker settings do not match Compose. The first three migrations have no down scripts. | Backend Dockerfile, Makefile, Compose and migrations. Align versions/ports and externalize secrets; complete reversible migrations before deployment. |
| Medium | Several controller annotations are placeholders, while checked-in Swagger is incomplete. Account deletion has a controller but no registered route. | Backend controllers, routes and `docs`. Complete annotations and regenerate Swagger after route decisions. |
| Medium | UI-only placeholders remain: settings only shows success toasts, edit/QR actions are incomplete, and dashboard/settings routes have no authentication guard. | Frontend settings, create-link, link-table and layout files. Integrate the intended flows before treating them as working features. |
| Medium | Existing tests cover only short-code and refresh-token utilities. CI validates only the backend. | `backend/pkg/utils/*_test.go`, workflow. Add auth/cache/ownership integration coverage and frontend checks during follow-up development. |

## Validation

```powershell
cd backend
go build ./...
go test ./...
go vet ./...
go mod verify
go mod tidy -diff
gofmt -l .

cd ../frontend
pnpm lint
pnpm exec tsc --noEmit
pnpm build
```

Verified locally on 7 October 2026:

- All 122 source files matched byte-for-byte immediately after the staged import. The root README was subsequently expanded here, with its original content archived in `docs/imported-readme.md`.
- Backend build, existing tests, vet, module verification, module-tidiness check and formatting check passed with Go 1.26.5.
- Frontend frozen-lockfile install and production build passed. After removing 11 unused-symbol warnings, lint and standalone TypeScript checks passed with no errors or warnings.
- `go test -race -count=1 ./...` could not run because this local Go environment has cgo disabled and no C compiler available. Race checks remain configured in the imported Linux CI workflow.
- Docker Desktop's engine was not running. PostgreSQL/Redis API integration, migrations against a running database and container builds were not executed.

Private `.env.test`, installed dependencies and generated build files remain local and ignored. The commits are local to this repository; no push or deployment was performed.
