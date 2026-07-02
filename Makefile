.PHONY: help build clean test run install lint fmt vet docker-build docker-run

# Variables
BINARY_NAME=mailserver
GO=go
GOFLAGS=-v
DOCKER_IMAGE=mailserver:latest
DOCKER_CONTAINER=mailserver-container

# Default target
help:
	@echo "Golang IMAP/SMTP Mail Server - Makefile targets"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build          - Build the mail server binary"
	@echo "  clean          - Remove build artifacts and temporary files"
	@echo "  test           - Run unit tests"
	@echo "  run            - Run the mail server (requires password file)"
	@echo "  install        - Download and install dependencies"
	@echo "  fmt            - Format Go code"
	@echo "  lint           - Run golangci-lint (if installed)"
	@echo "  vet            - Run go vet"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-clean   - Remove Docker image and container"
	@echo "  all            - Build, format, vet, and test"
	@echo ""

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: ./$(BINARY_NAME)"

# Clean build artifacts and temporary files
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f /tmp/last_email_id
	$(GO) clean
	$(GO) clean -testcache
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@if [ ! -f "test_mail.sh" ]; then \
		echo "Error: test_mail.sh not found"; \
		exit 1; \
	fi
	@chmod +x test_mail.sh
	@./test_mail.sh

# Run the mail server
run: build
	@echo "Running mail server..."
	@if [ -z "$$PASSWORD_FILE" ]; then \
		echo "Error: PASSWORD_FILE environment variable not set"; \
		echo "Usage: PASSWORD_FILE=/path/to/passwords.tsv make run"; \
		exit 1; \
	fi
	@if [ ! -f "$$PASSWORD_FILE" ]; then \
		echo "Error: Password file not found: $$PASSWORD_FILE"; \
		exit 1; \
	fi
	./$(BINARY_NAME) \
		--password-file "$$PASSWORD_FILE" \
		--s3-bucket "$$S3_BUCKET" \
		--s3-region "$$S3_REGION" \
		--smtp-port "$$SMTP_PORT" \
		--imap-port "$$IMAP_PORT"

# Install dependencies
install:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Formatting complete"

# Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "go vet complete"

# Run linter (if installed)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...
	@echo "Linter complete"

# Build and run all checks
all: fmt vet install build test
	@echo "All checks passed!"

# Docker targets
docker-build:
	@echo "Building Docker image: $(DOCKER_IMAGE)"
	docker build -t $(DOCKER_IMAGE) .
	@echo "Docker image built: $(DOCKER_IMAGE)"

docker-run: docker-build
	@echo "Running Docker container..."
	@if [ -z "$$PASSWORD_FILE" ]; then \
		echo "Error: PASSWORD_FILE environment variable not set"; \
		exit 1; \
	fi
	@if [ -z "$$S3_BUCKET" ]; then \
		echo "Error: S3_BUCKET environment variable not set"; \
		exit 1; \
	fi
	docker run -d \
		--name $(DOCKER_CONTAINER) \
		-p 25:25 \
		-p 143:143 \
		-v "$$PASSWORD_FILE:/etc/mailserver/passwords.tsv:ro" \
		-e S3_BUCKET="$$S3_BUCKET" \
		-e S3_REGION="$$S3_REGION" \
		$(DOCKER_IMAGE)
	@echo "Container running: $(DOCKER_CONTAINER)"

docker-clean:
	@echo "Cleaning Docker..."
	docker rm -f $(DOCKER_CONTAINER) 2>/dev/null || true
	docker rmi $(DOCKER_IMAGE) 2>/dev/null || true
	@echo "Docker cleanup complete"

# Development target with watch (requires entr or similar)
dev-watch:
	@echo "Watching for changes..."
	@which entr > /dev/null || (echo "entr not installed. Install with: brew install entr (macOS) or apt-get install entr (Linux)" && exit 1)
	find . -name "*.go" | entr make build test

# Help for environment variables
env-help:
	@echo "Environment Variables:"
	@echo ""
	@echo "Required for 'make run':"
	@echo "  PASSWORD_FILE    - Path to TSV file with username and password"
	@echo "  S3_BUCKET        - AWS S3 bucket name"
	@echo ""
	@echo "Optional for 'make run':"
	@echo "  S3_REGION        - AWS region (default: us-east-1)"
	@echo "  SMTP_PORT        - SMTP port (default: :25)"
	@echo "  IMAP_PORT        - IMAP port (default: :143)"
	@echo ""
	@echo "Example:"
	@echo "  PASSWORD_FILE=/etc/mailserver/passwords.tsv S3_BUCKET=my-bucket make run"
	@echo ""
