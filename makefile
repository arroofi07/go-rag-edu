.PHONY: dev run build swagger swagger-run clean tidy test

# Hot reload development (auto-restart on file changes)
dev:
	go run github.com/air-verse/air@latest

# Run without hot reload
run:
	go run cmd/api/main.go

# Build binary
build:
	go build -o tmp/main.exe cmd/api/main.go

# Generate swagger docs
swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -o docs

# Generate swagger docs then run with hot reload
swagger-run: swagger dev

# Tidy dependencies
tidy:
	go mod tidy

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf tmp/