.PHONY: build test docker clean

# Variables
BINARY_NAME=server
DOCKER_IMAGE=openx-enrichment-service-template

# Build the application
build:
	go build -o bin/$(BINARY_NAME) cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Build docker image
docker:
	docker build -t $(DOCKER_IMAGE) .

# Run docker container
docker-run:
	docker run -p 8080:8080 $(DOCKER_IMAGE)

# Clean build artifacts
clean:
	rm -f bin/$(BINARY_NAME)
	go clean

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	go vet ./...

# Install dependencies
deps:
	go mod tidy

# All (format, lint, test, build)
all: fmt lint test build 