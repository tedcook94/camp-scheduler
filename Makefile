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
	migrate -path database/migrations -database "${DATABASE_URL}" goto ${VERSION}

migration:
	migrate create -dir database/migrations -ext sql -seq $(name)
