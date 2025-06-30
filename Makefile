# Disk Health Exporter Makefile

BINARY_NAME=disk-health-exporter
DOCKER_IMAGE=disk-health-exporter
BUILD_PATH=./cmd/disk-health-exporter
TEMP_DIR = ./tmp

# Build variables
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_BY ?= $(shell whoami)

# Go build variables
LDFLAGS = -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildTime=$(BUILD_TIME) \
	-X main.buildBy=$(BUILD_BY)



# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	@test -z "$(shell git status --porcelain)"

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #


## tidy: tidy modfiles and format .go files
.PHONY: tidy
tidy:
	go mod tidy -v
	go fmt ./...

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags="$(LDFLAGS)" -o=${TEMP_DIR}/bin/${BINARY_NAME} ${BUILD_PATH}

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...
	
test/e2e:
	@echo "Running end-to-end tests..."
	./scripts/test.sh

test/coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf ${TEMP_DIR}/bin ${TEMP_DIR}/release
	docker rmi $(DOCKER_IMAGE) 2>/dev/null || true

# audit: run quality control checks
.PHONY: audit
audit: test
	go mod tidy -diff
	go mod verify
	test -z "$(shell gofmt -l .)" 
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...


# security/scan: run security vulnerability checks
.PHONY: security/scan
security/scan:
	@echo "Running security vulnerability checks..."
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	@echo "Security scan completed"

.PHONY: run/live
run/live:
	go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "make build" --build.bin "${TEMP_DIR}/bin/${BINARY_NAME}" --build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go, tpl, tmpl, config.json, sql" \
		--misc.clean_on_exit "true"

# Run the exporter locally
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Check Go dependencies
deps:
	@echo "Checking dependencies..."
	go mod tidy
	go mod verify

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# ==================================================================================== #
# DISTRIBUTION
# ==================================================================================== #

## production/build: build optimized production binaries for all platforms
.PHONY: production/build
production/build: audit 
	@echo "Building production binaries..."
	@mkdir -p ${TEMP_DIR}/release
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${RELEASE_FLAGS} -ldflags='${LDFLAGS}' -o=${TEMP_DIR}/release/${BINARY_NAME}-linux-amd64 ${BUILD_PATH}
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${RELEASE_FLAGS} -ldflags='${LDFLAGS}' -o=${TEMP_DIR}/release/${BINARY_NAME}-linux-arm64 ${BUILD_PATH}
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${RELEASE_FLAGS} -ldflags='${LDFLAGS}' -o=${TEMP_DIR}/release/${BINARY_NAME}-windows-amd64.exe ${BUILD_PATH}
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ${RELEASE_FLAGS} -ldflags='${LDFLAGS}' -o=${TEMP_DIR}/release/${BINARY_NAME}-darwin-arm64 ${BUILD_PATH}
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ${RELEASE_FLAGS} -ldflags='${LDFLAGS}' -o=${TEMP_DIR}/release/${BINARY_NAME}-darwin-amd64 ${BUILD_PATH}
	@echo "Verifying binary metadata..."
	@for binary in ${TEMP_DIR}/release/${BINARY_NAME}-*; do \
		echo "$$binary:"; \
		file "$$binary" 2>/dev/null || echo "  File info not available"; \
		ls -lh "$$binary"; \
	done

## production/compress: compress production binaries
.PHONY: production/compress
production/compress: production/build
	@echo "Compressing binaries..."
	@which upx >/dev/null 2>&1 || (echo "UPX not found. Install with: brew install upx" && exit 1)
	upx -9 ${TEMP_DIR}/release/${BINARY_NAME}-linux-amd64 2>/dev/null || true
	upx -9 ${TEMP_DIR}/release/${BINARY_NAME}-windows-amd64.exe 2>/dev/null || true
	upx -9 ${TEMP_DIR}/release/${BINARY_NAME}-darwin-arm64 --force-macos 2>/dev/null || true
	upx -9 ${TEMP_DIR}/release/${BINARY_NAME}-darwin-amd64 --force-macos 2>/dev/null || true

## production/compress: compress production binaries
.PHONY: production/release
production/release: production/build production/compress

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -f deployments/Dockerfile -t $(DOCKER_IMAGE):latest .
	docker build -f deployments/Dockerfile -t $(DOCKER_IMAGE):$(VERSION) .

# Build Docker image for production
.PHONY: production/docker
production/docker:
	@echo "Building production Docker image..."
	docker build -f deployments/Dockerfile -t $(DOCKER_IMAGE):$(VERSION) .
	docker build -f deployments/Dockerfile -t $(DOCKER_IMAGE):latest .

# Install the exporter (universal - detects OS automatically)
install:
	@echo "Installing $(BINARY_NAME)..."
	./scripts/install.sh



# Show available targets  
show-help:
	@echo "Available targets:"
	@echo "  build     - Build the binary"
	@echo "  test      - Run tests and integration tests"
	@echo "  clean     - Clean build artifacts"
	@echo "  docker    - Build Docker image"
	@echo "  install   - Install the exporter system-wide (auto-detects OS)"
	@echo "  run       - Build and run the exporter"
	@echo "  deps      - Check and tidy dependencies"
	@echo "  fmt       - Format code"
	@echo "  lint      - Lint code"
	@echo "  help      - Show this help"
