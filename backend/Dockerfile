FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM builder AS test
RUN apk add --no-cache gcc musl-dev
ENV CGO_ENABLED=1
CMD ["go", "test", "-race", "-count=1", "./..."]

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates tzdata && addgroup -S api && adduser -S -G api api
WORKDIR /app
COPY --from=builder /out/server /out/migrate ./
USER api
EXPOSE 8080
CMD ["/app/server"]
