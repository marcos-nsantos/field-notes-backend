.PHONY: build run test test-unit test-integration lint clean docker-up docker-down migrate-up migrate-down mocks

# Go parameters
BINARY_NAME=api
MAIN_PATH=./cmd/api

# Build the application
build:
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

# Run the application
run:
	go run $(MAIN_PATH)

# Run all tests
test:
	go test -v -race -cover ./...

# Run unit tests only (exclude integration tests)
test-unit:
	go test -v -race -cover -short ./...

# Run integration tests only
test-integration:
	go test -v -race -run Integration ./...

# Run tests with coverage report
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker commands
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api

docker-build:
	docker compose build

# Database migrations (requires golang-migrate)
migrate-up:
	migrate -path migrations -database "postgres://fieldnotes:fieldnotes@localhost:5432/fieldnotes?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://fieldnotes:fieldnotes@localhost:5432/fieldnotes?sslmode=disable" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install go.uber.org/mock/mockgen@latest

# Tidy dependencies
tidy:
	go mod tidy

# Generate mocks using uber-go/mock
mocks:
	mockgen -source=internal/adapter/repository/interfaces.go -destination=internal/mocks/repository_mocks.go -package=mocks
	mockgen -source=internal/adapter/storage/interfaces.go -destination=internal/mocks/storage_mocks.go -package=mocks
	mockgen -source=internal/adapter/handler/interfaces.go -destination=internal/mocks/handler_mocks.go -package=mocks

# Full check before commit
check: fmt lint test
