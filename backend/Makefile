.PHONY: build run test vet security swag migrate.up migrate.down docker.run docker.stop docker.test

build:
	go build -trimpath -o build/server ./cmd/server
	go build -trimpath -o build/migrate ./cmd/migrate

run:
	go run ./cmd/server

test:
	go test -count=1 ./...

vet:
	go vet ./...

# Optional: install gosec separately before invoking this target.
security:
	gosec -quiet ./...

swag:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go

migrate.up:
	go run ./cmd/migrate -direction up

# Deliberately roll back only the most recent migration.
migrate.down:
	go run ./cmd/migrate -direction down -steps 1

docker.run:
	docker compose up -d --build

docker.stop:
	docker compose down

docker.test:
	docker compose run --rm --build test
