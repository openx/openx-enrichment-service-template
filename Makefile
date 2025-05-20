.PHONY: build test docker clean cert cert-local cert-remote stop-service

# Variables
BINARY_NAME=server
DOCKER_IMAGE=openx-enrichment-service-template
SERVICE_PORT?=8082
CERT_HOST?=localhost
CERT_PORT?=8080
USE_HTTPS?=false
CONTAINER_NAME=enrichment-service-test
SIMULATE_LATENCY?=false
LATENCY_MEAN_MS?=5
LATENCY_STDDEV_MS?=5
SIMULATE_CPU_LOAD?=false
CPU_LOAD_PERCENTAGE?=50

# Build the application
build:
	go build -o bin/$(BINARY_NAME) cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Build docker image
docker:
	docker build -t $(DOCKER_IMAGE) .

# Run docker container in background
docker-run:
	@docker run -d --name $(CONTAINER_NAME) \
		-p $(SERVICE_PORT):8080 \
		-e SIMULATE_LATENCY=$(SIMULATE_LATENCY) \
		-e LATENCY_MEAN_MS=$(LATENCY_MEAN_MS) \
		-e LATENCY_STDDEV_MS=$(LATENCY_STDDEV_MS) \
		-e SIMULATE_CPU_LOAD=$(SIMULATE_CPU_LOAD) \
		-e CPU_LOAD_PERCENTAGE=$(CPU_LOAD_PERCENTAGE) \
		$(DOCKER_IMAGE)
	@echo "Waiting for service to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:$(SERVICE_PORT)/healthz > /dev/null; then \
			echo "Service is ready!"; \
			exit 0; \
		fi; \
		echo "Waiting for service... ($$i/30)"; \
		sleep 1; \
	done; \
	echo "Service failed to become ready in time"; \
	exit 1

# Stop and remove the test container
stop-service:
	@docker stop $(CONTAINER_NAME) 2>/dev/null || true
	@docker rm $(CONTAINER_NAME) 2>/dev/null || true

# Clean build artifacts
clean: stop-service
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

# Run certification suite against local service
cert-local: stop-service docker-run
	@echo "Running certification tests..."
	@cd certification && TARGET_HOST=host.docker.internal TARGET_PORT=$(SERVICE_PORT) USE_HTTPS=$(USE_HTTPS) docker compose run --rm k6 run --out json=results.json --summary-export=summary.json --summary-time-unit=ms /scripts/openrtb.js
	@make stop-service

# Run certification suite against remote service
cert-remote:
	@echo "Running certification tests..."
	@cd certification && TARGET_HOST=$(CERT_HOST) TARGET_PORT=$(CERT_PORT) USE_HTTPS=$(USE_HTTPS) docker compose run --rm k6 run --out json=results.json --summary-export=summary.json --summary-time-unit=ms /scripts/openrtb.js

# Alias for cert-local
cert: cert-local 