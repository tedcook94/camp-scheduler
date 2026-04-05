DATABASE_URL := postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)?sslmode=$(DATABASE_SSL_MODE)
DATABASE_URL_TEST := postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)_test?sslmode=$(DATABASE_SSL_MODE)

ifdef v
MIGRATE_CMD = goto $(v)
else
MIGRATE_CMD = up
endif

.PHONY: dev build sqlc migrate migrate-test migrate-all migration test test-integration

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
	migrate -path database/migrations -database "$(DATABASE_URL)" $(MIGRATE_CMD)

migrate-test:
	migrate -path database/migrations -database "$(DATABASE_URL_TEST)" $(MIGRATE_CMD)

migrate-all:
	migrate -path database/migrations -database "$(DATABASE_URL)" $(MIGRATE_CMD)
	migrate -path database/migrations -database "$(DATABASE_URL_TEST)" $(MIGRATE_CMD)

migration:
	migrate create -dir database/migrations -ext sql -seq $(name)