DATABASE_URL := postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)?sslmode=$(DATABASE_SSL_MODE)

.PHONY: dev build sqlc migrate migration test test-integration

dev:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

test-integration:
	go test -tags integration ./...

sqlc:
	sqlc generate

migrate:
	migrate -path database/migrations -database "$(DATABASE_URL)" goto $(v)

migration:
	migrate create -dir database/migrations -ext sql -seq $(name)